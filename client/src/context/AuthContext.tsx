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

// Token storage utilities
const TOKEN_KEY = 'waugzee_auth_token';

const getStoredToken = (): string | null => {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
};

const setStoredToken = (token: string | null) => {
  try {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token);
    } else {
      localStorage.removeItem(TOKEN_KEY);
    }
  } catch {
    // Silently fail if localStorage is not available
  }
};

// Validate token format (basic JWT structure check)
const isValidTokenFormat = (token: string): boolean => {
  if (!token) return false;
  const parts = token.split('.');
  return parts.length === 3 && parts.every(part => part.length > 0);
};

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();
  const [user, setUser] = createStore<User | null>(null);
  const [isAuthenticated, setIsAuthenticated] = createSignal<boolean | null>(
    null,
  );
  const [authToken, setAuthToken] = createSignal<string | null>(getStoredToken());
  const [authConfig, setAuthConfig] = createSignal<AuthConfig | null>(null);

  // Initialize token interceptor with persistence
  const updateApiToken = initializeTokenInterceptor((token: string) => {
    setAuthToken(token);
    setStoredToken(token);
  });

  // Set initial token if we have one stored
  createEffect(() => {
    const token = authToken();
    if (token) {
      updateApiToken(token);
    }
  });

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

  // Check if we should attempt auth/me based on stored token
  const hasValidToken = () => {
    const token = authToken();
    return token && isValidTokenFormat(token);
  };

  // Only check current user if we have a valid token - otherwise immediately set as unauthenticated
  createEffect(() => {
    if (!hasValidToken()) {
      setIsAuthenticated(false);
      setUser(null);
      return;
    }

    // We have a token, try to validate it with /me endpoint
    const checkAuthStatus = async () => {
      try {
        const response = await fetch(`${env.apiUrl}/api/auth/me`, {
          headers: {
            'Authorization': `Bearer ${authToken()}`,
            'Content-Type': 'application/json',
          },
        });

        if (response.ok) {
          const data = await response.json();
          if (data.user) {
            setUser(data.user);
            setIsAuthenticated(true);
          } else {
            setIsAuthenticated(false);
            setUser(null);
          }
        } else {
          // Token is invalid, clear it
          setAuthToken(null);
          setStoredToken(null);
          updateApiToken(null);
          setIsAuthenticated(false);
          setUser(null);
        }
      } catch (error) {
        console.warn('Auth check failed:', error);
        setIsAuthenticated(false);
        setUser(null);
      }
    };

    checkAuthStatus();
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
        setStoredToken(data.access_token);
        updateApiToken(data.access_token);
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
      setStoredToken(null);
      updateApiToken(null);

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
      setStoredToken(null);
      updateApiToken(null);
      navigate("/login");
    } catch {
      // Still clear local state even if server logout fails
      setUser(null);
      setIsAuthenticated(false);
      setAuthToken(null);
      setStoredToken(null);
      updateApiToken(null);
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
      <Show when={isAuthenticated() !== null || configQuery.isError || (!hasValidToken() && configQuery.isSuccess)}>
        {props.children}
      </Show>
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
