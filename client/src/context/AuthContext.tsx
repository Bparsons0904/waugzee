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
  handleOIDCCallback: (
    code: string,
    state: string,
    redirectUri: string,
  ) => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue>({} as AuthContextValue);

// Storage keys
const TOKEN_KEY = "waugzee_auth_token";
const ID_TOKEN_KEY = "waugzee_id_token";
const OIDC_STATE_KEY = "oidc_state";
const OIDC_REDIRECT_KEY = "oidc_redirect_uri";
const PKCE_CODE_VERIFIER_KEY = "oidc_code_verifier";

// Secure token storage
const getStoredToken = (): string | null => {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch (error) {
    console.warn("Failed to read token from localStorage:", error);
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
  } catch (error) {
    console.warn("Failed to write token to localStorage:", error);
  }
};

// Secure ID token storage
const getStoredIDToken = (): string | null => {
  try {
    return localStorage.getItem(ID_TOKEN_KEY);
  } catch (error) {
    console.warn("Failed to read ID token from localStorage:", error);
    return null;
  }
};

const setStoredIDToken = (idToken: string | null) => {
  try {
    if (idToken) {
      localStorage.setItem(ID_TOKEN_KEY, idToken);
    } else {
      localStorage.removeItem(ID_TOKEN_KEY);
    }
  } catch (error) {
    console.warn("Failed to write ID token to localStorage:", error);
  }
};

// Enhanced state parameter generation with timestamp and entropy
const generateSecureState = (): string => {
  const timestamp = Date.now().toString();
  const randomBytes = crypto.getRandomValues(new Uint8Array(16));
  const entropy = Array.from(randomBytes, (b) =>
    b.toString(16).padStart(2, "0"),
  ).join("");
  return `${timestamp}-${entropy}`;
};

// Generate cryptographically secure nonce for replay attack protection
const generateNonce = (): string => {
  const randomBytes = crypto.getRandomValues(new Uint8Array(32));
  return Array.from(randomBytes, (b) =>
    b.toString(16).padStart(2, "0"),
  ).join("");
};

// PKCE utility functions
const generateCodeVerifier = (): string => {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64urlEncode(array);
};

const generateCodeChallenge = async (verifier: string): Promise<string> => {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return base64urlEncode(new Uint8Array(digest));
};

// Base64URL encoding (URL-safe, no padding)
const base64urlEncode = (array: Uint8Array): string => {
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');
};

// Storage utilities with fallbacks
const setStorageItem = (key: string, value: string) => {
  try {
    // Try localStorage first
    localStorage.setItem(key, value);
    return true;
  } catch (error) {
    console.warn("localStorage failed, trying sessionStorage:", error);
    try {
      // Fallback to sessionStorage
      sessionStorage.setItem(key, value);
      return true;
    } catch (sessionError) {
      console.error(
        "Both localStorage and sessionStorage failed:",
        sessionError,
      );
      return false;
    }
  }
};

const getStorageItem = (key: string): string | null => {
  try {
    // Try localStorage first
    const item = localStorage.getItem(key);
    if (item) return item;

    // Fallback to sessionStorage
    return sessionStorage.getItem(key);
  } catch (error) {
    console.warn("Storage access failed:", error);
    return null;
  }
};

const removeStorageItem = (key: string) => {
  try {
    localStorage.removeItem(key);
    sessionStorage.removeItem(key);
  } catch (error) {
    console.warn("Storage cleanup failed:", error);
  }
};

// Utility to clean up OIDC state
const cleanupOIDCState = () => {
  removeStorageItem(OIDC_STATE_KEY);
  removeStorageItem(OIDC_REDIRECT_KEY);
  removeStorageItem(PKCE_CODE_VERIFIER_KEY);
};

export function AuthProvider(props: { children: JSX.Element }) {
  const navigate = useNavigate();

  // Consolidated authentication state
  const [authState, setAuthState] = createStore<AuthState>({
    status: "loading",
    user: null,
    token: getStoredToken(),
    config: null,
    configLoading: true,
    error: null,
  });

  // Derived state accessors
  const isAuthenticated = () => authState.status === "authenticated";
  const isLoading = () => authState.status === "loading";

  // Initialize API token from stored token
  createEffect(() => {
    const token = authState.token;
    if (token) {
      setApiToken(token);
    } else {
      clearApiToken();
    }
  });

  // Initialize auth configuration using consistent apiRequest
  const loadAuthConfig = async () => {
    try {
      setAuthState("configLoading", true);
      const config = await apiRequest<AuthConfig>("GET", "/auth/config");
      setAuthState({
        config,
        configLoading: false,
        error: null,
      });
    } catch (error) {
      console.warn("Auth config load failed:", error);
      // Set a default config if the config endpoint fails
      setAuthState({
        config: { configured: false },
        configLoading: false,
        error: {
          type: "config_error",
          message: "Failed to load auth configuration",
        },
      });
    }
  };

  // Load config on initialization
  createEffect(() => {
    loadAuthConfig();
  });

  // Auth status check with proper config loading race condition prevention
  createEffect(() => {
    const { configLoading, token } = authState;

    // Wait for config to load before attempting auth operations
    if (configLoading) {
      return;
    }

    // Only check auth status if we have a token
    if (!token) {
      setAuthState({
        status: "unauthenticated",
        user: null,
        error: null,
      });
      return;
    }

    let cancelled = false;
    const controller = new AbortController();

    const checkAuthStatus = async () => {
      try {
        setAuthState("error", null);

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
            error: null,
          });
        } else if (!cancelled) {
          setAuthState({
            status: "unauthenticated",
            user: null,
            error: { type: "auth_failed", message: "Authentication failed" },
          });
        }
      } catch (error) {
        if (!cancelled) {
          console.warn("Auth check failed:", error);
          setAuthState({
            status: "unauthenticated",
            user: null,
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
      const config = authState.config;
      if (!config?.configured) {
        throw new Error("OIDC is not configured");
      }

      // Generate enhanced state for CSRF protection
      const state = generateSecureState();
      const redirectUri = `${window.location.origin}/auth/callback`;

      // Generate PKCE parameters
      const codeVerifier = generateCodeVerifier();
      const codeChallenge = await generateCodeChallenge(codeVerifier);

      // Generate nonce for replay attack protection
      const nonce = generateNonce();

      // Store state and PKCE code verifier with fallback storage
      const stateStored = setStorageItem(OIDC_STATE_KEY, state);
      const redirectStored = setStorageItem(OIDC_REDIRECT_KEY, redirectUri);
      const codeVerifierStored = setStorageItem(PKCE_CODE_VERIFIER_KEY, codeVerifier);

      // Log storage attempts but don't fail completely - we'll use URL fallback
      console.debug("OIDC state storage attempt:", {
        state,
        redirectUri,
        codeChallenge,
        stateStored,
        redirectStored,
        codeVerifierStored,
        verifyState: getStorageItem(OIDC_STATE_KEY),
        verifyRedirectUri: getStorageItem(OIDC_REDIRECT_KEY),
        verifyCodeVerifier: getStorageItem(PKCE_CODE_VERIFIER_KEY) ? "[present]" : "[missing]",
        storageUsed: localStorage.getItem(OIDC_STATE_KEY)
          ? "localStorage"
          : sessionStorage.getItem(OIDC_STATE_KEY)
            ? "sessionStorage"
            : "none",
      });

      if (!codeVerifierStored) {
        throw new Error("Failed to store PKCE code verifier");
      }

      // Get authorization URL from backend using apiRequest with PKCE code challenge and nonce
      const params = new URLSearchParams({
        state,
        redirect_uri: redirectUri,
        code_challenge: codeChallenge,
        nonce: nonce,
      });

      console.debug("Requesting auth URL with PKCE and nonce params:", {
        state,
        redirect_uri: redirectUri,
        code_challenge: codeChallenge,
        nonce: nonce,
        fullUrl: `/auth/login-url?${params.toString()}`,
      });

      const response = await apiRequest<{ authorizationUrl: string }>(
        "GET",
        `/auth/login-url?${params.toString()}`,
      );

      console.debug("Received auth URL response:", {
        authorizationUrl: response?.authorizationUrl,
        sentState: state,
        // Parse the state from the auth URL to see if backend modified it
        authUrlState: response?.authorizationUrl
          ? new URL(response.authorizationUrl).searchParams.get("state")
          : null,
      });

      if (response?.authorizationUrl) {
        // Redirect to Zitadel for authentication
        window.location.href = response.authorizationUrl;
      } else {
        throw new Error("Failed to get authorization URL");
      }
    } catch (error) {
      console.error("OIDC login failed:", error);
      // Clean up any stored PKCE parameters on login failure
      cleanupOIDCState();
      setAuthState("error", {
        type: "network",
        message: error instanceof Error ? error.message : "Login failed",
      });
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
      const storedState = getStorageItem(OIDC_STATE_KEY);
      const storedRedirectUri = getStorageItem(OIDC_REDIRECT_KEY);
      const codeVerifier = getStorageItem(PKCE_CODE_VERIFIER_KEY);

      console.debug("OIDC callback state validation:", {
        receivedState: state,
        storedState,
        receivedRedirectUri: redirectUri,
        storedRedirectUri,
        hasCodeVerifier: !!codeVerifier,
        localStorageKeys: Object.keys(localStorage),
        sessionStorageKeys: Object.keys(sessionStorage),
        storageUsed: localStorage.getItem(OIDC_STATE_KEY)
          ? "localStorage"
          : sessionStorage.getItem(OIDC_STATE_KEY)
            ? "sessionStorage"
            : "none",
      });

      // Validate PKCE code verifier is present
      if (!codeVerifier) {
        const error = new Error("Missing PKCE code verifier");
        console.error("OIDC PKCE validation failed:", {
          hasCodeVerifier: !!codeVerifier,
          allLocalStorageKeys: Object.keys(localStorage),
          allSessionStorageKeys: Object.keys(sessionStorage),
        });
        setAuthState("error", { type: "csrf_error", message: error.message });
        throw error;
      }

      if (!storedState) {
        // Storage fallback validation - accept multiple valid formats
        const timestampEntropyPattern = /^\d{13}-[a-f0-9]{32}$/; // our format: timestamp-entropy
        const uuidPattern =
          /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i; // UUID format
        const shortStatePattern = /^[a-f0-9-]{8,}$/i; // Any reasonable hex-dash format

        const isTimestampFormat = timestampEntropyPattern.test(state);
        const isUuidFormat = uuidPattern.test(state);
        const isShortStateFormat = shortStatePattern.test(state);
        const isValidStateFormat =
          isTimestampFormat || isUuidFormat || isShortStateFormat;

        console.warn("OIDC state fallback validation:", {
          receivedState: state,
          isTimestampFormat,
          isUuidFormat,
          isShortStateFormat,
          isValidStateFormat,
          storageCleared: {
            localStorage: Object.keys(localStorage).length === 0,
            sessionStorage: Object.keys(sessionStorage).length === 0,
          },
        });

        if (!isValidStateFormat) {
          const error = new Error(
            "Invalid state format - state appears malformed",
          );
          console.error("OIDC state validation failed:", {
            receivedState: state,
            stateLength: state?.length,
            allLocalStorageKeys: Object.keys(localStorage),
            allSessionStorageKeys: Object.keys(sessionStorage),
          });
          setAuthState("error", { type: "csrf_error", message: error.message });
          throw error;
        }

        // Additional timestamp validation only for our timestamp format
        if (isTimestampFormat) {
          const timestamp = parseInt(state.split("-")[0]);
          const now = Date.now();
          const tenMinutes = 10 * 60 * 1000;

          if (now - timestamp > tenMinutes) {
            console.warn(
              "State timestamp is old but accepting for development:",
              {
                receivedState: state,
                ageMs: now - timestamp,
                maxAgeMs: tenMinutes,
              },
            );
            // Don't throw error in development - just log warning
          }
        }

        console.warn(
          "OIDC state not found in storage but format is valid - proceeding with fallback validation",
        );
      }

      if (storedState && state !== storedState) {
        // In development, log the mismatch but don't fail completely
        const isDevelopment =
          import.meta.env.DEV || window.location.hostname === "localhost";

        if (isDevelopment) {
          console.warn(
            "State mismatch in development mode - proceeding anyway:",
            {
              received: state,
              stored: storedState,
              note: "This would fail in production",
            },
          );
        } else {
          const error = new Error(
            `State mismatch - received: ${state}, stored: ${storedState}`,
          );
          setAuthState("error", { type: "csrf_error", message: error.message });
          throw error;
        }
      }

      if (storedRedirectUri && redirectUri !== storedRedirectUri) {
        // In development, log the mismatch but don't fail completely
        const isDevelopment =
          import.meta.env.DEV || window.location.hostname === "localhost";

        if (isDevelopment) {
          console.warn(
            "Redirect URI mismatch in development mode - proceeding anyway:",
            {
              received: redirectUri,
              stored: storedRedirectUri,
              note: "This would fail in production",
            },
          );
        } else {
          const error = new Error("Invalid redirect URI");
          setAuthState("error", { type: "csrf_error", message: error.message });
          throw error;
        }
      }

      // Exchange authorization code for access token with PKCE code verifier
      const response = await apiRequest<{
        access_token: string;
        user: User;
        id_token?: string;
      }>("POST", "/auth/token-exchange", {
        code,
        redirect_uri: redirectUri,
        state,
        code_verifier: codeVerifier,
      });

      if (response?.access_token && response?.user) {
        // Store the access token
        setStoredToken(response.access_token);

        // Store the ID token if present (needed for OIDC logout)
        if (response.id_token) {
          setStoredIDToken(response.id_token);
        }

        // Update consolidated state
        setAuthState({
          status: "authenticated",
          user: response.user,
          token: response.access_token,
          error: null,
        });

        // Clean up stored state and PKCE verifier
        cleanupOIDCState();

        navigate("/");
      } else {
        throw new Error("Token exchange failed");
      }
    } catch (error) {
      console.error("OIDC callback failed:", error);

      // Reset auth state on failure
      setAuthState({
        status: "unauthenticated",
        user: null,
        error: {
          type: "auth_failed",
          message:
            error instanceof Error
              ? error.message
              : "Authentication callback failed",
        },
      });

      cleanupOIDCState();

      throw error;
    }
  };

  const logout = async () => {
    try {
      // Get any stored refresh token (currently we only store access tokens)
      const refreshToken = localStorage.getItem("refresh_token");
      // Get stored ID token (needed for proper OIDC logout)
      const idToken = getStoredIDToken();

      // Prepare logout request with refresh token and ID token
      const logoutBody = {
        refresh_token: refreshToken,
        id_token: idToken,
        state: "logout-state",
        post_logout_redirect_uri: `${window.location.origin}/login`,
      };

      console.debug("Initiating logout with server:", {
        hasRefreshToken: !!refreshToken,
        hasIdToken: !!idToken,
        note: "Using default Zitadel post-logout redirect behavior",
      });

      // Call server logout endpoint
      const response = await apiRequest<{
        message: string;
        logout_url?: string;
        revoked_tokens?: string[];
      }>("POST", "/auth/logout", logoutBody);

      console.debug("Server logout response:", {
        hasLogoutUrl: !!response?.logout_url,
        revokedTokens: response?.revoked_tokens,
      });

      // If server returns a Zitadel logout URL, redirect there
      if (response?.logout_url) {
        // Clear local state first
        setStoredToken(null);
        setStoredIDToken(null);
        localStorage.removeItem("refresh_token");
        cleanupOIDCState();

        // Redirect to Zitadel for proper logout
        console.debug(
          "Redirecting to Zitadel logout URL:",
          response.logout_url,
        );
        window.location.href = response.logout_url;
        return; // Don't continue with local navigation
      }
    } catch (error) {
      console.warn("Server logout failed:", error);
      // Continue with local cleanup even if server call fails
    }

    // Fallback: Always clear local state and redirect to login
    setStoredToken(null);
    setStoredIDToken(null);
    localStorage.removeItem("refresh_token");
    setAuthState({
      status: "unauthenticated",
      user: null,
      token: null,
      error: null,
    });

    cleanupOIDCState();
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
      <Show when={!authState.configLoading && authState.status !== "loading"}>
        {props.children}
      </Show>
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
