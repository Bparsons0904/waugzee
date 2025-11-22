import { AUTH_ENDPOINTS, ROUTES } from "@constants/api.constants";
import { api, setTokenGetter } from "@services/api";
import { logger } from "@services/logger.service";
import { oidcService } from "@services/oidc.service";
import { useNavigate } from "@solidjs/router";
import { createContext, createEffect, type JSX, onCleanup, Show, useContext } from "solid-js";
import { createStore } from "solid-js/store";
import type { AuthConfig, User } from "src/types/User";

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
  token: string | null;
  config: AuthConfig | null;
  configLoading: boolean;
  oidcInitialized: boolean;
  error: AuthError;
};

type AuthContextValue = {
  authState: AuthState;
  isAuthenticated: () => boolean;
  authToken: () => string | null;
  authConfig: () => AuthConfig | null;
  loginWithOIDC: (returnTo?: string) => Promise<void>;
  handleOIDCCallback: () => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue>({} as AuthContextValue);

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();

  const [authState, setAuthState] = createStore<AuthState>({
    status: "loading",
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
    logger.info("Performing local logout", { action: "local_logout" });

    // Clear all local state
    setAuthState({
      status: "unauthenticated",
      token: null,
      error: null,
    });

    navigate(ROUTES.LOGIN);
  };

  createEffect(async () => {
    if (!authState.oidcInitialized) return;

    try {
      const token = await oidcService.getIDToken();
      setAuthState("token", token);
    } catch (error) {
      logger.warn("Failed to get ID token", {
        error: { message: error instanceof Error ? error.message : String(error) },
      });
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

      // Try to load cached config first for faster initial render
      const cachedConfig = localStorage.getItem("auth_config");
      let config: AuthConfig;

      if (cachedConfig) {
        try {
          config = JSON.parse(cachedConfig);
          // Use cached config immediately for faster startup
          setAuthState({ config, configLoading: false });
        } catch (e) {
          logger.warn("Failed to parse cached auth config", {
            error: { message: e instanceof Error ? e.message : String(e) },
          });
        }
      }

      // Always fetch fresh config in background to ensure it's up to date
      config = await api.get<AuthConfig>(AUTH_ENDPOINTS.CONFIG);

      // Cache the fresh config for next time
      localStorage.setItem("auth_config", JSON.stringify(config));

      // Initialize OIDC service with the configuration
      if (config.configured) {
        await oidcService.initialize(config);

        // Set up OIDC event callbacks for token expiry and renewal failures
        oidcService.setEventCallbacks({
          onTokenExpired: () => {
            logger.warn("OIDC token expired", { action: "token_expired" });
            performLocalLogout();
          },
          onSilentRenewError: (error) => {
            logger.error("OIDC silent renewal failed", {
              action: "silent_renew_error",
              error: { message: error.message },
            });
            performLocalLogout();
          },
          onUserSignedOut: () => {
            logger.info("OIDC user signed out", { action: "user_signed_out" });
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
      logger.error("Auth config load failed", {
        action: "config_load_error",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      setAuthState({
        config: { configured: false },
        configLoading: false,
        oidcInitialized: false,
        error: {
          type: "config_error",
          message: error instanceof Error ? error.message : "Failed to load auth configuration",
        },
      });
    }
  });

  createEffect(() => {
    if (authState.configLoading || !authState.config?.configured || !authState.oidcInitialized) {
      return;
    }

    let cancelled = false;
    const controller = new AbortController();

    const checkAuthStatus = async () => {
      try {
        logger.debug("Checking authentication status");
        setAuthState("error", null);

        const isAuthenticated = await oidcService.isAuthenticated();
        const oidcUser = await oidcService.getUser();

        if (!isAuthenticated || !oidcUser || cancelled) {
          logger.debug("No valid OIDC session found");
          setAuthState({
            status: "unauthenticated",
            token: null,
            error: null,
          });
          return;
        }

        // Set authenticated status with ID token (JWT) for backend validation
        if (!cancelled) {
          logger.info("Authentication successful", { action: "auth_check_success" });

          setAuthState({
            status: "authenticated",
            token: oidcUser.id_token,
            error: null,
          });

          // Check if we're on a valid protected route after refresh
          const currentPath = window.location.pathname;
          const authPaths = [ROUTES.LOGIN, ROUTES.CALLBACK, ROUTES.SILENT_CALLBACK] as const;
          const isAuthPath = (authPaths as readonly string[]).includes(currentPath);

          // If user is authenticated and on an auth-related page, redirect home
          // Otherwise, stay on current page (handles refresh scenario)
          if (isAuthPath) navigate(ROUTES.HOME);
        }
      } catch (error) {
        if (!cancelled) {
          logger.warn("Authentication check failed", {
            error: { message: error instanceof Error ? error.message : String(error) },
          });
          setAuthState({
            status: "unauthenticated",
            token: null,
            error: {
              type: "network",
              message: error instanceof Error ? error.message : "Authentication check failed",
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

  // Initialize logger when user becomes authenticated
  createEffect(() => {
    if (authState.status === "authenticated") {
      logger.initialize();
      logger.info("User authenticated", { action: "auth_success" });
    }
  });

  // Cleanup logger on component unmount
  onCleanup(() => {
    logger.destroy();
  });

  const loginWithOIDC = async (returnTo?: string) => {
    try {
      if (!authState.config?.configured) {
        throw new Error("OIDC is not configured");
      }

      if (!authState.oidcInitialized) {
        throw new Error("OIDC service is not initialized yet");
      }

      const currentPath = returnTo || window.location.pathname;
      const authPaths = [ROUTES.LOGIN, ROUTES.CALLBACK, ROUTES.SILENT_CALLBACK];
      const shouldStoreReturn = !authPaths.includes(currentPath as (typeof authPaths)[number]);

      const returnPath = shouldStoreReturn ? currentPath : ROUTES.HOME;

      await oidcService.signInRedirect({
        returnTo: returnPath,
      });
    } catch (error) {
      logger.error("OIDC login failed", {
        action: "oidc_login_error",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
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
        throw new Error("Failed to complete OIDC authentication - missing tokens");
      }

      // Temporarily update the token in auth state so the API call can use it
      setAuthState("token", oidcUser.id_token);

      // Call our backend's callback endpoint with the ID token to create/update the user
      try {
        const callbackResponse = await api.post<CallbackResponse>(AUTH_ENDPOINTS.CALLBACK, {
          id_token: oidcUser.id_token,
          access_token: oidcUser.access_token,
          state:
            typeof oidcUser.state === "string" ? oidcUser.state : JSON.stringify(oidcUser.state),
        });

        logger.info("Backend callback successful", {
          action: "backend_callback_success",
          userId: callbackResponse?.user?.id,
          email: callbackResponse?.user?.email,
        });
      } catch (backendError) {
        logger.error("Backend callback failed", {
          action: "backend_callback_error",
          error: { message: backendError instanceof Error ? backendError.message : String(backendError) },
        });
        throw new Error("Failed to register user with backend");
      }

      // Set authenticated status after successful backend callback
      setAuthState({
        status: "authenticated",
        token: oidcUser.id_token,
        error: null,
      });

      // Extract returnTo from OIDC state
      let state: Record<string, unknown> = {};
      try {
        state =
          typeof oidcUser.state === "object"
            ? oidcUser.state
            : JSON.parse(typeof oidcUser.state === "string" ? oidcUser.state : "{}");
      } catch (error) {
        logger.warn("Failed to parse OIDC state, using default", {
          error: { message: error instanceof Error ? error.message : String(error) },
        });
      }

      const returnTo = typeof state.returnTo === "string" ? state.returnTo : ROUTES.HOME;

      // Navigate to original destination
      navigate(returnTo);
    } catch (error) {
      logger.error("OIDC callback failed", {
        action: "oidc_callback_error",
        error: { message: error instanceof Error ? error.message : String(error) },
      });

      setAuthState({
        status: "unauthenticated",
        token: null,
        error: {
          type: "auth_failed",
          message: error instanceof Error ? error.message : "Authentication callback failed",
        },
      });

      throw error;
    }
  };

  const logout = async () => {
    logger.info("Logout initiated", { action: "logout_start" });

    try {
      try {
        if (authState.token) {
          await api.post(AUTH_ENDPOINTS.LOGOUT, {
            access_token: authState.token,
          });
          logger.debug("Backend logout completed successfully");
        }
      } catch (backendError) {
        logger.warn("Backend logout failed, continuing with OIDC logout", {
          error: { message: backendError instanceof Error ? backendError.message : String(backendError) },
        });
      }

      if (!authState.oidcInitialized) {
        logger.warn("OIDC service not initialized, performing local logout only");
        performLocalLogout();
        return;
      }

      try {
        await oidcService.signOut();
        logger.debug("OIDC signOut completed successfully");
      } catch (oidcError) {
        logger.warn("OIDC signOut failed, forcing local cleanup", {
          error: { message: oidcError instanceof Error ? oidcError.message : String(oidcError) },
        });

        // Force clear OIDC session even if signOut fails
        try {
          await oidcService.clearUserSession();
          logger.debug("OIDC session cleared forcibly");
        } catch (clearError) {
          logger.error("Failed to clear OIDC session", {
            error: { message: clearError instanceof Error ? clearError.message : String(clearError) },
          });
        }
      }

      performLocalLogout();
    } catch (error) {
      logger.error("Logout process failed, performing emergency cleanup", {
        action: "logout_error",
        error: { message: error instanceof Error ? error.message : String(error) },
      });

      // Emergency cleanup - ensure we always clear local state
      performLocalLogout();
    }
  };

  return (
    <AuthContext.Provider
      value={{
        authState,
        isAuthenticated,
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
