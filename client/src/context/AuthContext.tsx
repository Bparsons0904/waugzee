import { initializeTokenInterceptor } from "@services/api/api.service";
import { useNavigate } from "@solidjs/router";
import {
  createContext,
  useContext,
  createSignal,
  JSX,
  Accessor,
  createEffect,
  Show,
} from "solid-js";
import { createStore } from "solid-js/store";
import {
  User,
  AuthConfig,
  // TokenExchangeRequest,
  // TokenExchangeResponse,
  // LoginURLResponse
} from "src/types/User";
import { apiClient } from "@services/api/client";
import { env } from "@services/env.service";

type AuthContextValue = {
  isAuthenticated: Accessor<boolean | null>;
  user: User | null;
  authToken: Accessor<string | null>;
  authConfig: Accessor<AuthConfig | null>;
  loginWithOIDC: () => Promise<void>;
  handleOIDCCallback: (
    code: string,
    state: string,
    redirectUri: string,
  ) => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue>({} as AuthContextValue);

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();
  const [user, setUser] = createStore<User | null>(null);
  const [isAuthenticated, setIsAuthenticated] = createSignal<boolean | null>(
    null,
  );
  const [authToken, setAuthToken] = createSignal<string | null>(null);
  const [authConfig, setAuthConfig] = createSignal<AuthConfig | null>(null);

  initializeTokenInterceptor(setAuthToken);

  // Initialize auth configuration
  const configQuery = apiClient.auth.config();
  createEffect(() => {
    if (configQuery.isSuccess && configQuery.data) {
      setAuthConfig(configQuery.data);
    } else if (configQuery.isError) {
      // Set a default config if the config endpoint fails
      setAuthConfig({ configured: false });
    }
  });

  // Check current user if we have a token - but don't run if we don't have basic config yet
  const meQuery = apiClient.auth.me();
  createEffect(() => {
    if (meQuery.isSuccess && meQuery.data?.user) {
      setUser(meQuery.data.user);
      setIsAuthenticated(true);
    } else if (meQuery.isError) {
      // Only set to false if we actually got an error (like 401)
      // Don't set false if query just hasn't run yet
      setIsAuthenticated(false);
      setUser(null);
    } else if (meQuery.isSuccess && !meQuery.data?.user) {
      setIsAuthenticated(false);
      setUser(null);
    }

    // If config is loaded but queries are still pending, set a reasonable default
    if (configQuery.isSuccess && meQuery.isPending) {
      // Still loading, keep auth state null to show loading
      return;
    }

    // If both config and me query failed, assume not authenticated
    if (configQuery.isError && meQuery.isError) {
      setIsAuthenticated(false);
      setUser(null);
    }
  });

  const loginWithOIDC = async () => {
    try {
      const config = authConfig();
      if (!config?.configured) {
        throw new Error("OIDC is not configured");
      }

      // Generate state for CSRF protection
      const state = crypto.randomUUID();
      const redirectUri = `${window.location.origin}/auth/callback`;

      // Store state in localStorage for validation
      localStorage.setItem("oidc_state", state);
      localStorage.setItem("oidc_redirect_uri", redirectUri);

      // Get authorization URL from backend
      const response = await fetch(
        `${env.apiUrl}/api/auth/login-url?state=${encodeURIComponent(state)}&redirect_uri=${encodeURIComponent(redirectUri)}`,
      );
      const data = await response.json();

      if (response.ok && data.authorizationUrl) {
        // Redirect to Zitadel for authentication
        window.location.href = data.authorizationUrl;
      } else {
        throw new Error(data.error || "Failed to get authorization URL");
      }
    } catch (error) {
      console.error("OIDC login failed:", error);
      throw error;
    }
  };

  const handleOIDCCallback = async (
    code: string,
    state: string,
    redirectUri: string,
  ) => {
    try {
      // Verify state to prevent CSRF attacks
      const storedState = localStorage.getItem("oidc_state");
      const storedRedirectUri = localStorage.getItem("oidc_redirect_uri");

      if (state !== storedState) {
        throw new Error("Invalid state parameter");
      }

      if (redirectUri !== storedRedirectUri) {
        throw new Error("Invalid redirect URI");
      }

      // Exchange authorization code for access token
      const response = await fetch(`${env.apiUrl}/api/auth/token-exchange`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          code,
          redirect_uri: redirectUri,
          state,
        }),
      });

      const data = await response.json();

      if (response.ok && data.access_token && data.user) {
        setAuthToken(data.access_token);
        setUser(data.user);
        setIsAuthenticated(true);

        // Clean up stored state
        localStorage.removeItem("oidc_state");
        localStorage.removeItem("oidc_redirect_uri");

        navigate("/");
      } else {
        throw new Error(data.error || "Token exchange failed");
      }
    } catch (error) {
      console.error("OIDC callback failed:", error);
      setIsAuthenticated(false);
      setUser(null);
      setAuthToken(null);

      // Clean up stored state
      localStorage.removeItem("oidc_state");
      localStorage.removeItem("oidc_redirect_uri");

      throw error;
    }
  };

  const logout = async () => {
    try {
      const logoutMutation = apiClient.auth.logout()();
      await logoutMutation.mutateAsync();
      setUser(null);
      setIsAuthenticated(false);
      setAuthToken(null);
      navigate("/login");
    } catch {
      // Still clear local state even if server logout fails
      setUser(null);
      setIsAuthenticated(false);
      setAuthToken(null);
      navigate("/login");
    }
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        user,
        authToken,
        authConfig,
        loginWithOIDC,
        handleOIDCCallback,
        logout,
      }}
    >
      <Show when={isAuthenticated() !== null || configQuery.isError}>
        {props.children}
      </Show>
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
