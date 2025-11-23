import { ROUTES } from "@constants/api.constants";
import { logger } from "@services/logger.service";
import {
  type User as OidcUser,
  UserManager,
  type UserManagerSettings,
  WebStorageStateStore,
} from "oidc-client-ts";
import type { AuthConfig } from "src/types/User";

/**
 * Event callback types for OIDC service events
 */
export interface OIDCEventCallbacks {
  onTokenExpired?: () => void;
  onSilentRenewError?: (error: Error) => void;
  onUserSignedOut?: () => void;
}

/**
 * OIDC Service using oidc-client-ts for secure authentication
 * Addresses security concerns:
 * - Uses secure in-memory token storage by default
 * - Implements automatic token refresh
 * - Provides proper CSRF protection with state validation
 * - Handles PKCE flow securely
 * - Proper token expiry handling with automatic logout
 */
export class OIDCService {
  private userManager: UserManager | null = null;
  private config: AuthConfig | null = null;
  private eventCallbacks: OIDCEventCallbacks = {};

  /**
   * Discover OIDC configuration from the provider with caching
   */
  private async discoverOIDCConfiguration(instanceUrl: string): Promise<Record<string, unknown>> {
    logger.debug("Attempting OIDC discovery", { component: "OIDC", instanceUrl });

    // Check cache first
    const cacheKey = `oidc_discovery_${instanceUrl}`;
    const cached = sessionStorage.getItem(cacheKey);
    if (cached) {
      try {
        const metadata = JSON.parse(cached);
        logger.debug("Using cached OIDC discovery metadata", { component: "OIDC" });
        return metadata;
      } catch (e) {
        logger.warn("Failed to parse cached OIDC metadata", {
          component: "OIDC",
          error: { message: e instanceof Error ? e.message : String(e) },
        });
      }
    }

    // Try multiple discovery URL patterns for Zitadel
    const discoveryUrls = [
      `${instanceUrl}/.well-known/openid-configuration`, // Zitadel standard (hyphen)
      `${instanceUrl}/.well-known/openid_configuration`, // Generic OIDC standard (underscore)
      `${instanceUrl}/oidc/v1/.well-known/openid-configuration`, // Zitadel with path prefix
    ];

    for (const discoveryUrl of discoveryUrls) {
      try {
        logger.debug("Trying discovery URL", { component: "OIDC", discoveryUrl });
        const response = await fetch(discoveryUrl);

        if (response.ok) {
          const metadata = await response.json();
          logger.debug("OIDC discovery successful", {
            component: "OIDC",
            discoveryUrl,
            issuer: metadata.issuer,
            endpoints: {
              authorization: metadata.authorization_endpoint,
              token: metadata.token_endpoint,
              userinfo: metadata.userinfo_endpoint,
              end_session: metadata.end_session_endpoint,
            },
          });

          // Cache the discovery metadata for the session
          sessionStorage.setItem(cacheKey, JSON.stringify(metadata));

          return metadata;
        } else {
          logger.debug("Discovery URL failed", {
            component: "OIDC",
            discoveryUrl,
            status: response.status,
            statusText: response.statusText,
          });
        }
      } catch (error) {
        logger.debug("Discovery URL error", {
          component: "OIDC",
          discoveryUrl,
          error: { message: error instanceof Error ? error.message : String(error) },
        });
      }
    }

    logger.warn("All OIDC discovery URLs failed, falling back to Zitadel endpoints", {
      component: "OIDC",
    });

    // Fallback to hardcoded Zitadel endpoints
    const fallbackMetadata = {
      issuer: instanceUrl,
      authorization_endpoint: `${instanceUrl}/oauth/v2/authorize`,
      token_endpoint: `${instanceUrl}/oauth/v2/token`,
      userinfo_endpoint: `${instanceUrl}/oidc/v1/userinfo`,
      end_session_endpoint: `${instanceUrl}/oidc/v1/end_session`,
      jwks_uri: `${instanceUrl}/oauth/v2/keys`,
    };

    // Cache the fallback too
    sessionStorage.setItem(cacheKey, JSON.stringify(fallbackMetadata));

    return fallbackMetadata;
  }

  /**
   * Initialize the OIDC service with auth configuration
   */
  async initialize(config: AuthConfig): Promise<void> {
    if (!config.configured || !config.clientId || !config.instanceUrl) {
      throw new Error("OIDC configuration is incomplete");
    }

    // Validate URLs to prevent "Invalid URL" constructor errors
    try {
      new URL(config.instanceUrl);
      new URL(`${window.location.origin}${ROUTES.CALLBACK}`);
      new URL(`${window.location.origin}${ROUTES.SILENT_CALLBACK}`);
      new URL(`${window.location.origin}${ROUTES.LOGIN}`);
    } catch (error) {
      throw new Error(
        `Invalid URL in OIDC configuration: ${error instanceof Error ? error.message : "Unknown URL error"}`,
      );
    }

    this.config = config;

    // Discover OIDC endpoints dynamically
    const discoveredMetadata = await this.discoverOIDCConfiguration(config.instanceUrl);

    const settings: UserManagerSettings = {
      // Core OIDC settings
      authority: config.instanceUrl,
      client_id: config.clientId,
      redirect_uri: `${window.location.origin}${ROUTES.CALLBACK}`,
      post_logout_redirect_uri: `${window.location.origin}${ROUTES.LOGIN}`,
      response_type: "code",
      scope: "openid profile email offline_access",

      // Security settings
      automaticSilentRenew: true, // Automatic token refresh
      includeIdTokenInSilentRenew: true,
      silent_redirect_uri: `${window.location.origin}${ROUTES.SILENT_CALLBACK}`,

      // PKCE (Proof Key for Code Exchange) for enhanced security
      response_mode: "query",
      loadUserInfo: false, // We'll get user info from our backend

      // Secure token storage - use sessionStorage for better security
      // This prevents tokens from persisting after browser restart
      userStore: new WebStorageStateStore({ store: window.sessionStorage }),

      // Additional security settings
      filterProtocolClaims: true,
      staleStateAgeInSeconds: 900, // 15 minutes

      // Use discovered metadata instead of hardcoded endpoints
      metadata: discoveredMetadata,
    };

    try {
      logger.debug("Creating UserManager with settings", {
        component: "OIDC",
        authority: settings.authority,
        client_id: settings.client_id,
        redirect_uri: settings.redirect_uri,
        post_logout_redirect_uri: settings.post_logout_redirect_uri,
      });

      this.userManager = new UserManager(settings);

      // Set up event handlers
      this.setupEventHandlers();

      logger.debug("OIDC UserManager created successfully", { component: "OIDC" });
    } catch (error) {
      this.userManager = null;
      throw new Error(
        `Failed to create OIDC UserManager: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
    }
  }

  /**
   * Set up event handlers for token management and errors
   */
  private setupEventHandlers(): void {
    if (!this.userManager) return;

    // Token renewed successfully
    this.userManager.events.addAccessTokenExpiring(() => {
      logger.debug("Access token expiring, attempting renewal", { component: "OIDC" });
    });

    // Token expired - trigger logout if automatic renewal fails
    this.userManager.events.addAccessTokenExpired(() => {
      logger.warn("Access token expired", { component: "OIDC", action: "token_expired" });
      if (this.eventCallbacks.onTokenExpired) {
        this.eventCallbacks.onTokenExpired();
      }
    });

    // Silent renewal failed - trigger logout
    this.userManager.events.addSilentRenewError((error) => {
      logger.error("Silent token renewal failed", {
        component: "OIDC",
        action: "silent_renew_error",
        error: { message: error.message || "Silent renewal failed" },
      });
      if (this.eventCallbacks.onSilentRenewError) {
        this.eventCallbacks.onSilentRenewError(new Error(error.message || "Silent renewal failed"));
      }
    });

    // User loaded successfully
    this.userManager.events.addUserLoaded((user) => {
      logger.debug("User loaded from OIDC", {
        component: "OIDC",
        sub: user.profile.sub,
        exp: user.expires_at,
        scopes: user.scopes,
      });
    });

    // User session terminated
    this.userManager.events.addUserUnloaded(() => {
      logger.debug("User session terminated", { component: "OIDC" });
    });

    // User signed out
    this.userManager.events.addUserSignedOut(() => {
      logger.debug("User signed out", { component: "OIDC", action: "user_signed_out" });
      if (this.eventCallbacks.onUserSignedOut) {
        this.eventCallbacks.onUserSignedOut();
      }
    });
  }

  /**
   * Get the current user from OIDC client
   */
  async getUser(): Promise<OidcUser | null> {
    if (!this.userManager) {
      logger.warn("OIDC service not initialized - getUser() called too early", {
        component: "OIDC",
      });
      return null;
    }

    try {
      const user = await this.userManager.getUser();
      return user;
    } catch (error) {
      logger.error("Failed to get user from OIDC", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      return null;
    }
  }

  /**
   * Get current access token
   */
  async getAccessToken(): Promise<string | null> {
    const user = await this.getUser();
    return user?.access_token || null;
  }

  /**
   * Get current ID token (JWT)
   */
  async getIDToken(): Promise<string | null> {
    const user = await this.getUser();
    return user?.id_token || null;
  }

  /**
   * Check if user is currently authenticated
   */
  async isAuthenticated(): Promise<boolean> {
    if (!this.userManager) {
      logger.warn("OIDC service not initialized - isAuthenticated() called too early", {
        component: "OIDC",
      });
      return false;
    }

    const user = await this.getUser();
    return user !== null && !user.expired;
  }

  /**
   * Start the authentication flow
   */
  async signInRedirect(additionalState?: Record<string, unknown>): Promise<void> {
    if (!this.userManager) {
      throw new Error("OIDC service not initialized");
    }

    try {
      await this.userManager.signinRedirect({
        // Additional state can be passed here
        state: { timestamp: Date.now(), ...additionalState },
      });
    } catch (error) {
      logger.error("Failed to start sign-in redirect", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      throw new Error("Failed to initiate authentication");
    }
  }

  /**
   * Handle the callback after authentication
   */
  async signInRedirectCallback(): Promise<OidcUser> {
    if (!this.userManager) {
      throw new Error("OIDC service not initialized");
    }

    try {
      const user = await this.userManager.signinRedirectCallback();

      if (!user) {
        throw new Error("No user returned from callback");
      }

      // Validate that we have required tokens
      if (!user.access_token) {
        throw new Error("No access token received");
      }

      logger.debug("Authentication callback successful", {
        component: "OIDC",
        sub: user.profile.sub,
        exp: user.expires_at,
        scopes: user.scopes,
      });

      return user;
    } catch (error) {
      logger.error("OIDC callback failed", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      throw new Error("Authentication callback failed");
    }
  }

  /**
   * Sign out the user
   */
  async signOut(): Promise<void> {
    if (!this.userManager) {
      throw new Error("OIDC service not initialized");
    }

    try {
      // This will redirect to the OIDC provider's logout endpoint
      await this.userManager.signoutRedirect({
        state: { timestamp: Date.now() },
      });
    } catch (error) {
      logger.error("Failed to sign out", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      // Fallback: clear local session even if remote logout fails
      await this.userManager.removeUser();
      throw error;
    }
  }

  /**
   * Remove user session locally (for emergency logout)
   */
  async clearUserSession(): Promise<void> {
    if (!this.userManager) {
      throw new Error("OIDC service not initialized");
    }

    try {
      await this.userManager.removeUser();
    } catch (error) {
      logger.error("Failed to clear user session", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      throw error;
    }
  }

  /**
   * Handle silent renewal (for iframe-based token refresh)
   */
  async signInSilentCallback(): Promise<void> {
    if (!this.userManager) {
      throw new Error("OIDC service not initialized");
    }

    try {
      await this.userManager.signinSilentCallback();
    } catch (error) {
      logger.error("Silent renewal callback failed", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      throw error;
    }
  }

  /**
   * Manually trigger token renewal
   */
  async renewToken(): Promise<OidcUser | null> {
    if (!this.userManager) {
      throw new Error("OIDC service not initialized");
    }

    try {
      const user = await this.userManager.signinSilent();
      return user;
    } catch (error) {
      logger.error("Manual token renewal failed", {
        component: "OIDC",
        error: { message: error instanceof Error ? error.message : String(error) },
      });
      return null;
    }
  }

  /**
   * Get configuration info
   */
  getConfig(): AuthConfig | null {
    return this.config;
  }

  /**
   * Check if service is initialized
   */
  isInitialized(): boolean {
    return this.userManager !== null;
  }

  /**
   * Set event callbacks for OIDC events
   */
  setEventCallbacks(callbacks: OIDCEventCallbacks): void {
    this.eventCallbacks = { ...this.eventCallbacks, ...callbacks };
  }

  /**
   * Check if the current user's token is expired or about to expire
   */
  async isTokenExpired(): Promise<boolean> {
    const user = await this.getUser();
    if (!user) return true;

    // Check if token is already expired
    if (user.expired) return true;

    // Check if token expires within the next 5 minutes (configurable buffer)
    const now = Math.floor(Date.now() / 1000);
    const expiresAt = user.expires_at || 0;
    const bufferSeconds = 5 * 60; // 5 minutes

    return expiresAt - now <= bufferSeconds;
  }

  /**
   * Get token expiry information
   */
  async getTokenExpiry(): Promise<{
    expiresAt: number;
    expiresInSeconds: number;
  } | null> {
    const user = await this.getUser();
    if (!user || !user.expires_at) return null;

    const now = Math.floor(Date.now() / 1000);
    const expiresInSeconds = user.expires_at - now;

    return {
      expiresAt: user.expires_at,
      expiresInSeconds: Math.max(0, expiresInSeconds),
    };
  }
}

// Export singleton instance
export const oidcService = new OIDCService();
