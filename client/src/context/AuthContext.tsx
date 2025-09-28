import { useNavigate } from "@solidjs/router";
import {
  createContext,
  useContext,
  JSX,
  createEffect,
  Show,
  onCleanup,
} from "solid-js";
import { createStore } from "solid-js/store";
import {
  User,
  AuthConfig,
  Folder,
  UserWithFoldersResponse,
} from "src/types/User";
import { api, setTokenGetter } from "@services/api";
import { oidcService } from "@services/oidc.service";
import {
  AUTH_ENDPOINTS,
  USER_ENDPOINTS,
  ROUTES,
} from "@constants/api.constants";
type AuthStatus = "loading" | "authenticated" | "unauthenticated";

type AuthError =
  | { type: "network"; message: string }
  | { type: "auth_failed"; message: string }
  | { type: "config_error"; message: string }
  | { type: "csrf_error"; message: string }
  | null;

interface CallbackResponse {
  access_token: string;
  token_type: string;
  refresh_token?: string;
  expires_in: number;
  id_token?: string;
  state?: string;
  user: User;
}

type AuthState = {
  status: AuthStatus;
  user: User | null;
  folders: Folder[];
  token: string | null;
  config: AuthConfig | null;
  configLoading: boolean;
  oidcInitialized: boolean;
  error: AuthError;
};

type AuthContextValue = {
  authState: AuthState;
  isAuthenticated: () => boolean;
  user: () => User | null;
  folders: () => Folder[];
  authToken: () => string | null;
  authConfig: () => AuthConfig | null;
  loginWithOIDC: () => Promise<void>;
  handleOIDCCallback: () => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<void>;
  updateUser: (user: User) => void;
};

const AuthContext = createContext<AuthContextValue>({} as AuthContextValue);

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();

  const [authState, setAuthState] = createStore<AuthState>({
    status: "loading",
    user: null,
    folders: [],
    token: null,
    config: null,
    configLoading: true,
    oidcInitialized: false,
    error: null,
  });

  const isAuthenticated = () => authState.status === "authenticated";

  setTokenGetter(() => authState.token);

  // Define performLocalLogout early so it can be used in callbacks
  const performLocalLogout = () => {
    console.info("Performing local logout - clearing all auth state");

    // Clear all local state
    setAuthState({
      status: "unauthenticated",
      user: null,
      folders: [],
      token: null,
      error: null,
    });

    navigate(ROUTES.LOGIN);
  };

  createEffect(async () => {
    if (!authState.oidcInitialized) return;

    try {
      const token = await oidcService.getAccessToken();
      setAuthState("token", token);
    } catch (error) {
      console.warn("Failed to get access token:", error);
      setAuthState("token", null);
    }
  });

  createEffect(async () => {
    try {
      setAuthState({
        configLoading: true,
        oidcInitialized: false,
        error: null,
      });

      const config = await api.get<AuthConfig>(AUTH_ENDPOINTS.CONFIG);

      // Initialize OIDC service with the configuration
      if (config.configured) {
        await oidcService.initialize(config);

        // Set up OIDC event callbacks for token expiry and renewal failures
        oidcService.setEventCallbacks({
          onTokenExpired: () => {
            console.warn("OIDC token expired - performing logout");
            performLocalLogout();
          },
          onSilentRenewError: (error) => {
            console.error(
              "OIDC silent renewal failed - performing logout:",
              error,
            );
            performLocalLogout();
          },
          onUserSignedOut: () => {
            console.info("OIDC user signed out - performing logout");
            performLocalLogout();
          },
        });

        setAuthState({
          config,
          configLoading: false,
          oidcInitialized: true,
          error: null,
        });
      } else {
        setAuthState({
          config,
          configLoading: false,
          oidcInitialized: false,
          error: null,
        });
      }
    } catch (error) {
      console.error("Auth config load failed:", error);
      setAuthState({
        config: { configured: false },
        configLoading: false,
        oidcInitialized: false,
        error: {
          type: "config_error",
          message:
            error instanceof Error
              ? error.message
              : "Failed to load auth configuration",
        },
      });
    }
  });

  createEffect(() => {
    if (
      authState.configLoading ||
      !authState.config?.configured ||
      !authState.oidcInitialized
    ) {
      return;
    }

    let cancelled = false;
    const controller = new AbortController();

    const checkAuthStatus = async () => {
      try {
        console.debug("Checking authentication status...");
        setAuthState("error", null);

        const isAuthenticated = await oidcService.isAuthenticated();
        const oidcUser = await oidcService.getUser();

        if (!isAuthenticated || !oidcUser || cancelled) {
          console.debug("No valid OIDC session found");
          setAuthState({
            status: "unauthenticated",
            user: null,
            token: null,
            error: null,
          });
          return;
        }

        // Get user info from backend
        const response = await api.get<UserWithFoldersResponse>(
          USER_ENDPOINTS.ME,
          {
            signal: controller.signal,
          },
        );

        if (!cancelled && response?.user) {
          console.info("Authentication successful", {
            userId: response.user.id,
            email: response.user.email,
          });

          setAuthState({
            status: "authenticated",
            user: response.user,
            folders: response.folders || [],
            token: oidcUser.access_token,
            error: null,
          });
        } else if (!cancelled) {
          console.warn("Backend user info not found, clearing session");
          await oidcService.clearUserSession();
          setAuthState({
            status: "unauthenticated",
            user: null,
            folders: [],
            token: null,
            error: { type: "auth_failed", message: "Failed to get user info" },
          });
        }
      } catch (error) {
        if (!cancelled) {
          console.warn("Authentication check failed:", error);
          setAuthState({
            status: "unauthenticated",
            user: null,
            folders: [],
            token: null,
            error: {
              type: "network",
              message:
                error instanceof Error
                  ? error.message
                  : "Authentication check failed",
            },
          });
        }
      }
    };

    checkAuthStatus();

    onCleanup(() => {
      cancelled = true;
      controller.abort();
    });
  });

  const loginWithOIDC = async () => {
    try {
      if (!authState.config?.configured) {
        throw new Error("OIDC is not configured");
      }

      if (!authState.oidcInitialized) {
        throw new Error("OIDC service is not initialized yet");
      }

      await oidcService.signInRedirect();
    } catch (error) {
      console.error("OIDC login failed:", error);
      setAuthState("error", {
        type: "auth_failed",
        message: error instanceof Error ? error.message : "Login failed",
      });
      throw error;
    }
  };

  const handleOIDCCallback = async () => {
    try {
      if (!authState.oidcInitialized) {
        throw new Error("OIDC service is not initialized yet");
      }

      // Complete the OIDC client-side callback to get tokens first
      const oidcUser = await oidcService.signInRedirectCallback();

      if (!oidcUser?.access_token || !oidcUser?.id_token) {
        throw new Error(
          "Failed to complete OIDC authentication - missing tokens",
        );
      }

      // Temporarily update the token in auth state so the API call can use it
      setAuthState("token", oidcUser.access_token);

      // Call our backend's callback endpoint with the ID token to create/update the user
      try {
        const callbackResponse = await api.post<CallbackResponse>(
          AUTH_ENDPOINTS.CALLBACK,
          {
            id_token: oidcUser.id_token,
            access_token: oidcUser.access_token,
            state:
              typeof oidcUser.state === "string"
                ? oidcUser.state
                : JSON.stringify(oidcUser.state),
          },
        );

        console.info("Backend callback successful:", {
          userId: callbackResponse?.user?.id,
          email: callbackResponse?.user?.email,
        });
      } catch (backendError) {
        console.error("Backend callback failed:", backendError);
        throw new Error("Failed to register user with backend");
      }

      // Finally, get the user info from our backend (which should now exist)
      const response = await api.get<UserWithFoldersResponse>(
        USER_ENDPOINTS.ME,
      );

      if (response?.user) {
        setAuthState({
          status: "authenticated",
          user: response.user,
          folders: response.folders || [],
          token: oidcUser.access_token,
          error: null,
        });

        navigate(ROUTES.HOME);
      } else {
        throw new Error("Failed to get user info from backend after callback");
      }
    } catch (error) {
      console.error("OIDC callback failed:", error);

      setAuthState({
        status: "unauthenticated",
        user: null,
        folders: [],
        token: null,
        error: {
          type: "auth_failed",
          message:
            error instanceof Error
              ? error.message
              : "Authentication callback failed",
        },
      });

      throw error;
    }
  };

  const updateUser = (user: User) => {
    if (!isAuthenticated()) {
      console.warn("Cannot update user - not authenticated");
      return;
    }
    setAuthState("user", user);
    console.debug("User state updated directly");
  };

  const refreshUser = async () => {
    if (!isAuthenticated()) {
      console.warn("Cannot refresh user - not authenticated");
      return;
    }

    try {
      const response = await api.get<UserWithFoldersResponse>(
        USER_ENDPOINTS.ME,
      );

      if (response?.user) {
        setAuthState("user", response.user);
        setAuthState("folders", response.folders || []);
        console.debug("User refreshed successfully");
      } else {
        console.warn("Failed to refresh user - no user data returned");
      }
    } catch (error) {
      console.error("Failed to refresh user:", error);
    }
  };

  const logout = async () => {
    console.info("Logout initiated");

    try {
      try {
        if (authState.token) {
          await api.post(AUTH_ENDPOINTS.LOGOUT, {
            access_token: authState.token,
          });
          console.debug("Backend logout completed successfully");
        }
      } catch (backendError) {
        console.warn(
          "Backend logout failed, continuing with OIDC logout:",
          backendError,
        );
      }

      if (!authState.oidcInitialized) {
        console.warn(
          "OIDC service not initialized, performing local logout only",
        );
        performLocalLogout();
        return;
      }

      try {
        await oidcService.signOut();
        console.debug("OIDC signOut completed successfully");
      } catch (oidcError) {
        console.warn("OIDC signOut failed, forcing local cleanup:", oidcError);

        // Force clear OIDC session even if signOut fails
        try {
          await oidcService.clearUserSession();
          console.debug("OIDC session cleared forcibly");
        } catch (clearError) {
          console.error("Failed to clear OIDC session:", clearError);
        }
      }

      performLocalLogout();
    } catch (error) {
      console.error(
        "Logout process failed, performing emergency cleanup:",
        error,
      );

      // Emergency cleanup - ensure we always clear local state
      performLocalLogout();
    }
  };

  return (
    <AuthContext.Provider
      value={{
        authState,
        isAuthenticated,
        user: () => authState.user,
        folders: () => authState.folders,
        authToken: () => authState.token,
        authConfig: () => authState.config,
        loginWithOIDC,
        handleOIDCCallback,
        logout,
        refreshUser,
        updateUser,
      }}
    >
      <Show
        when={
          !authState.configLoading &&
          (authState.config?.configured === false || authState.oidcInitialized)
        }
      >
        {props.children}
      </Show>
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
