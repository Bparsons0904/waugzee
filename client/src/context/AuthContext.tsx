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
import { User, AuthConfig } from "src/types/User";
import { apiRequest, setTokenGetter } from "@services/api/api.service";
import { oidcService } from "@services/oidc.service";
import { AUTH_ENDPOINTS, FRONTEND_ROUTES } from "@constants/api.constants";
import { retryWithExponentialBackoff, authRetryConfig } from "@utils/retry.utils";
type AuthStatus = "loading" | "authenticated" | "unauthenticated";

type AuthError =
  | { type: "network"; message: string }
  | { type: "auth_failed"; message: string }
  | { type: "config_error"; message: string }
  | { type: "csrf_error"; message: string }
  | null;

type AuthState = {
  status: AuthStatus;
  user: User | null;
  token: string | null;
  config: AuthConfig | null;
  configLoading: boolean;
  oidcInitialized: boolean;
  error: AuthError;
};

type AuthContextValue = {
  authState: AuthState;
  isAuthenticated: () => boolean;
  user: User | null;
  authToken: () => string | null;
  authConfig: () => AuthConfig | null;
  loginWithOIDC: () => Promise<void>;
  handleOIDCCallback: () => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue>({} as AuthContextValue);

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();

  const [authState, setAuthState] = createStore<AuthState>({
    status: "loading",
    user: null,
    token: null,
    config: null,
    configLoading: true,
    oidcInitialized: false,
    error: null,
  });

  const isAuthenticated = () => authState.status === "authenticated";

  setTokenGetter(() => authState.token);

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

      const config = await retryWithExponentialBackoff(
        () => apiRequest<AuthConfig>("GET", AUTH_ENDPOINTS.CONFIG),
        authRetryConfig
      );

      // Initialize OIDC service with the configuration
      if (config.configured) {
        await oidcService.initialize(config);

        // Set up OIDC event callbacks for token expiry and renewal failures
        oidcService.setEventCallbacks({
          onTokenExpired: () => {
            console.warn('OIDC token expired - performing logout');
            performLocalLogout();
          },
          onSilentRenewError: (error) => {
            console.error('OIDC silent renewal failed - performing logout:', error);
            performLocalLogout();
          },
          onUserSignedOut: () => {
            console.info('OIDC user signed out - performing logout');
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
    // loadAuthConfig();
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
        setAuthState("error", null);

        const isAuthenticated = await oidcService.isAuthenticated();
        const oidcUser = await oidcService.getUser();

        if (!isAuthenticated || !oidcUser || cancelled) {
          setAuthState({
            status: "unauthenticated",
            user: null,
            token: null,
            error: null,
          });
          return;
        }

        const response = await apiRequest<{ user: User }>(
          "GET",
          AUTH_ENDPOINTS.ME,
          undefined,
          {
            signal: controller.signal,
          },
        );

        if (!cancelled && response?.user) {
          setAuthState({
            status: "authenticated",
            user: response.user,
            token: oidcUser.access_token,
            error: null,
          });
        } else if (!cancelled) {
          setAuthState({
            status: "unauthenticated",
            user: null,
            token: null,
            error: { type: "auth_failed", message: "Failed to get user info" },
          });
        }
      } catch (error) {
        if (!cancelled) {
          console.warn("Auth check failed:", error);
          setAuthState({
            status: "unauthenticated",
            user: null,
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

      const oidcUser = await oidcService.signInRedirectCallback();

      if (!oidcUser?.access_token) {
        throw new Error("Failed to complete authentication");
      }

      const response = await apiRequest<{ user: User }>("GET", AUTH_ENDPOINTS.ME);

      if (response?.user) {
        setAuthState({
          status: "authenticated",
          user: response.user,
          token: oidcUser.access_token,
          error: null,
        });

        navigate(FRONTEND_ROUTES.HOME);
      } else {
        throw new Error("Failed to get user info from backend");
      }
    } catch (error) {
      console.error("OIDC callback failed:", error);

      setAuthState({
        status: "unauthenticated",
        user: null,
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

  const logout = async () => {
    try {
      if (!authState.oidcInitialized) {
        console.warn(
          "OIDC service not initialized, performing local logout only",
        );
        performLocalLogout();
        return;
      }

      await oidcService.signOut();
      performLocalLogout();
    } catch (error) {
      console.warn("OIDC logout failed, performing local logout:", error);
      // Fallback: Clear local state and navigate to login
      performLocalLogout();
    }
  };

  const performLocalLogout = () => {
    // Clear all local state
    setAuthState({
      status: "unauthenticated",
      user: null,
      token: null,
      error: null,
    });

    navigate(FRONTEND_ROUTES.LOGIN);
  };

  return (
    <AuthContext.Provider
      value={{
        authState,
        isAuthenticated,
        user: authState.user,
        authToken: () => authState.token,
        authConfig: () => authState.config,
        loginWithOIDC,
        handleOIDCCallback,
        logout,
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
