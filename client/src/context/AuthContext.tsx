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
import {
  apiRequest,
  setApiToken,
  clearApiToken,
} from "@services/api/api.service";
import { oidcService } from "@services/oidc.service";
import { User as OidcUser } from "oidc-client-ts";

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
  isLoading: () => boolean;
  user: User | null;
  authToken: () => string | null;
  authConfig: () => AuthConfig | null;
  loginWithOIDC: () => Promise<void>;
  handleOIDCCallback: () => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue>({} as AuthContextValue);

// In-memory token storage - more secure than localStorage
// Tokens are cleared when the browser tab is closed
let currentOidcUser: OidcUser | null = null;

/**
 * Get the current OIDC user from secure in-memory storage
 * Currently unused but kept for potential future use
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const getCurrentOidcUser = (): OidcUser | null => {
  return currentOidcUser;
};

/**
 * Set the current OIDC user in secure in-memory storage
 */
const setCurrentOidcUser = (user: OidcUser | null): void => {
  currentOidcUser = user;
};

/**
 * Clear the current OIDC user from in-memory storage
 */
const clearCurrentOidcUser = (): void => {
  currentOidcUser = null;
};

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();

  // Consolidated authentication state
  const [authState, setAuthState] = createStore<AuthState>({
    status: "loading",
    user: null,
    token: null, // Will be managed by oidc-client-ts
    config: null,
    configLoading: true,
    oidcInitialized: false,
    error: null,
  });

  // Derived state accessors
  const isAuthenticated = () => authState.status === "authenticated";
  const isLoading = () => authState.status === "loading";

  // Initialize API token from OIDC user - only after OIDC is initialized
  createEffect(async () => {
    if (!authState.oidcInitialized) {
      return;
    }

    try {
      const token = await oidcService.getAccessToken();
      if (token) {
        setApiToken(token);
        setAuthState("token", token);
      } else {
        clearApiToken();
        setAuthState("token", null);
      }
    } catch (error) {
      console.warn("Failed to get access token:", error);
      clearApiToken();
      setAuthState("token", null);
    }
  });

  // Initialize auth configuration and OIDC service
  const loadAuthConfig = async () => {
    try {
      setAuthState({
        configLoading: true,
        oidcInitialized: false,
        error: null,
      });
      console.debug("Loading auth configuration...");

      const config = await apiRequest<AuthConfig>("GET", "/auth/config");
      console.debug("Auth config loaded:", { configured: config.configured });

      // Initialize OIDC service with the configuration
      if (config.configured) {
        console.debug("Initializing OIDC service with config:", {
          instanceUrl: config.instanceUrl,
          clientId: config.clientId,
        });

        await oidcService.initialize(config);
        console.debug("OIDC service initialized successfully");

        setAuthState({
          config,
          configLoading: false,
          oidcInitialized: true,
          error: null,
        });
      } else {
        console.debug("OIDC not configured, skipping initialization");
        setAuthState({
          config,
          configLoading: false,
          oidcInitialized: false,
          error: null,
        });
      }
    } catch (error) {
      console.error("Auth config load failed:", error);
      // Set a default config if the config endpoint fails
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
  };

  // Load config on initialization
  createEffect(() => {
    loadAuthConfig();
  });

  // Auth status check using OIDC service
  createEffect(() => {
    const { configLoading, oidcInitialized, config } = authState;

    // Wait for config to load and OIDC to be initialized before attempting auth operations
    if (configLoading || !config?.configured || !oidcInitialized) {
      console.debug("Skipping auth check - waiting for initialization", {
        configLoading,
        configured: config?.configured,
        oidcInitialized,
      });
      return;
    }

    let cancelled = false;
    const controller = new AbortController();

    const checkAuthStatus = async () => {
      try {
        setAuthState("error", null);

        // Check if user is authenticated via OIDC
        const isAuthenticated = await oidcService.isAuthenticated();
        const oidcUser = await oidcService.getUser();

        if (!isAuthenticated || !oidcUser || cancelled) {
          setAuthState({
            status: "unauthenticated",
            user: null,
            token: null,
            error: null,
          });
          clearCurrentOidcUser();
          return;
        }

        // Store OIDC user in memory
        setCurrentOidcUser(oidcUser);

        // Ensure the access token is set in the API client before making requests
        setApiToken(oidcUser.access_token);

        // Get user info from our backend (which has additional user data)
        const response = await apiRequest<{ user: User }>(
          "GET",
          "/auth/me",
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
          clearCurrentOidcUser();
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
      const { config, oidcInitialized } = authState;

      if (!config?.configured) {
        throw new Error("OIDC is not configured");
      }

      if (!oidcInitialized) {
        throw new Error("OIDC service is not initialized yet");
      }

      console.debug("Starting OIDC authentication flow");

      // Use oidc-client-ts for secure authentication flow
      // This handles PKCE, state generation, and CSRF protection automatically
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
      const { oidcInitialized } = authState;

      if (!oidcInitialized) {
        throw new Error("OIDC service is not initialized yet");
      }

      console.debug("Handling OIDC callback");

      // Use oidc-client-ts to handle the callback
      // This automatically validates state, PKCE, and exchanges the code for tokens
      const oidcUser = await oidcService.signInRedirectCallback();

      if (!oidcUser || !oidcUser.access_token) {
        throw new Error("Failed to complete authentication");
      }

      // Store OIDC user in memory
      setCurrentOidcUser(oidcUser);

      // CRITICAL FIX: Set the access token in the API client BEFORE making requests
      setApiToken(oidcUser.access_token);
      console.debug("Access token set in API client");

      // Now get user info from our backend (which has additional user data)
      const response = await apiRequest<{ user: User }>("GET", "/auth/me");

      if (response?.user) {
        // Update consolidated state
        setAuthState({
          status: "authenticated",
          user: response.user,
          token: oidcUser.access_token,
          error: null,
        });

        console.debug("Authentication completed successfully");
        navigate("/");
      } else {
        throw new Error("Failed to get user info from backend");
      }
    } catch (error) {
      console.error("OIDC callback failed:", error);

      // Reset auth state on failure
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

      // Clear any stored user data and API token
      clearCurrentOidcUser();
      clearApiToken();

      throw error;
    }
  };

  const logout = async () => {
    try {
      const { oidcInitialized } = authState;

      if (!oidcInitialized) {
        console.warn(
          "OIDC service not initialized, performing local logout only",
        );
        // Fallback to local cleanup
        performLocalLogout();
        return;
      }

      console.debug("Initiating secure OIDC logout");

      // Use oidc-client-ts for secure logout
      // This will redirect to the OIDC provider's logout endpoint
      await oidcService.signOut();

      // The logout will redirect to the provider, so we shouldn't reach this point
      // But if we do, clear local state as a fallback
      performLocalLogout();
    } catch (error) {
      console.warn("OIDC logout failed, performing local logout:", error);
      // Fallback: Clear local state and navigate to login
      performLocalLogout();
    }
  };

  const performLocalLogout = () => {
    // Clear all local state
    clearCurrentOidcUser();
    setAuthState({
      status: "unauthenticated",
      user: null,
      token: null,
      error: null,
    });

    // Clear API token
    clearApiToken();

    navigate("/login");
  };

  return (
    <AuthContext.Provider
      value={{
        authState,
        isAuthenticated,
        isLoading,
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

