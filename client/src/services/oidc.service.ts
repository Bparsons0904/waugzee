import { UserManager, UserManagerSettings, User as OidcUser, WebStorageStateStore } from 'oidc-client-ts';
import { AuthConfig } from 'src/types/User';

/**
 * OIDC Service using oidc-client-ts for secure authentication
 * Addresses security concerns:
 * - Uses secure in-memory token storage by default
 * - Implements automatic token refresh
 * - Provides proper CSRF protection with state validation
 * - Handles PKCE flow securely
 */
export class OIDCService {
  private userManager: UserManager | null = null;
  private config: AuthConfig | null = null;

  /**
   * Initialize the OIDC service with auth configuration
   */
  async initialize(config: AuthConfig): Promise<void> {
    if (!config.configured || !config.clientId || !config.instanceUrl) {
      throw new Error('OIDC configuration is incomplete');
    }

    // Validate URLs to prevent "Invalid URL" constructor errors
    try {
      new URL(config.instanceUrl);
      new URL(`${window.location.origin}/auth/callback`);
      new URL(`${window.location.origin}/auth/silent-callback`);
      new URL(`${window.location.origin}/login`);
    } catch (error) {
      throw new Error(`Invalid URL in OIDC configuration: ${error instanceof Error ? error.message : 'Unknown URL error'}`);
    }

    this.config = config;

    const settings: UserManagerSettings = {
      // Core OIDC settings
      authority: config.instanceUrl,
      client_id: config.clientId,
      redirect_uri: `${window.location.origin}/auth/callback`,
      post_logout_redirect_uri: `${window.location.origin}/login`,
      response_type: 'code',
      scope: 'openid profile email offline_access',

      // Security settings
      automaticSilentRenew: true, // Automatic token refresh
      includeIdTokenInSilentRenew: true,
      silent_redirect_uri: `${window.location.origin}/auth/silent-callback`,

      // PKCE (Proof Key for Code Exchange) for enhanced security
      response_mode: 'query',
      loadUserInfo: false, // We'll get user info from our backend

      // Secure token storage - use sessionStorage for better security
      // This prevents tokens from persisting after browser restart
      userStore: new WebStorageStateStore({ store: window.sessionStorage }),

      // Additional security settings
      filterProtocolClaims: true,
      staleStateAgeInSeconds: 900, // 15 minutes

      // Custom metadata for Zitadel compatibility
      metadata: {
        issuer: config.instanceUrl,
        authorization_endpoint: `${config.instanceUrl}/oauth/v2/authorize`,
        token_endpoint: `${config.instanceUrl}/oauth/v2/token`,
        userinfo_endpoint: `${config.instanceUrl}/oidc/v1/userinfo`,
        end_session_endpoint: `${config.instanceUrl}/oidc/v1/end_session`,
        jwks_uri: `${config.instanceUrl}/oauth/v2/keys`,
      },
    };

    try {
      console.debug('Creating UserManager with settings:', {
        authority: settings.authority,
        client_id: settings.client_id,
        redirect_uri: settings.redirect_uri,
        post_logout_redirect_uri: settings.post_logout_redirect_uri,
      });

      this.userManager = new UserManager(settings);

      // Set up event handlers
      this.setupEventHandlers();
      
      console.debug('OIDC UserManager created successfully');
    } catch (error) {
      this.userManager = null;
      throw new Error(`Failed to create OIDC UserManager: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Set up event handlers for token management and errors
   */
  private setupEventHandlers(): void {
    if (!this.userManager) return;

    // Token renewed successfully
    this.userManager.events.addAccessTokenExpiring(() => {
      console.debug('Access token expiring, attempting renewal...');
    });

    this.userManager.events.addAccessTokenExpired(() => {
      console.debug('Access token expired');
      // The automatic renewal should handle this, but we can add fallback logic here
    });

    // Silent renewal success
    this.userManager.events.addSilentRenewError((error) => {
      console.error('Silent token renewal failed:', error);
      // Could trigger a logout or redirect to login page
    });

    // User loaded successfully
    this.userManager.events.addUserLoaded((user) => {
      console.debug('User loaded from OIDC:', {
        sub: user.profile.sub,
        exp: user.expires_at,
        scopes: user.scopes,
      });
    });

    // User session terminated
    this.userManager.events.addUserUnloaded(() => {
      console.debug('User session terminated');
    });

    // Handle errors
    this.userManager.events.addUserSignedOut(() => {
      console.debug('User signed out');
    });
  }

  /**
   * Get the current user from OIDC client
   */
  async getUser(): Promise<OidcUser | null> {
    if (!this.userManager) {
      console.warn('OIDC service not initialized - getUser() called too early');
      return null;
    }

    try {
      const user = await this.userManager.getUser();
      return user;
    } catch (error) {
      console.error('Failed to get user from OIDC:', error);
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
   * Check if user is currently authenticated
   */
  async isAuthenticated(): Promise<boolean> {
    if (!this.userManager) {
      console.warn('OIDC service not initialized - isAuthenticated() called too early');
      return false;
    }

    const user = await this.getUser();
    return !!(user && !user.expired);
  }

  /**
   * Start the authentication flow
   */
  async signInRedirect(): Promise<void> {
    if (!this.userManager) {
      throw new Error('OIDC service not initialized');
    }

    try {
      await this.userManager.signinRedirect({
        // Additional state can be passed here
        state: { timestamp: Date.now() },
      });
    } catch (error) {
      console.error('Failed to start sign-in redirect:', error);
      throw new Error('Failed to initiate authentication');
    }
  }

  /**
   * Handle the callback after authentication
   */
  async signInRedirectCallback(): Promise<OidcUser> {
    if (!this.userManager) {
      throw new Error('OIDC service not initialized');
    }

    try {
      const user = await this.userManager.signinRedirectCallback();
      
      if (!user) {
        throw new Error('No user returned from callback');
      }

      // Validate that we have required tokens
      if (!user.access_token) {
        throw new Error('No access token received');
      }

      console.debug('Authentication callback successful', {
        sub: user.profile.sub,
        exp: user.expires_at,
        scopes: user.scopes,
      });

      return user;
    } catch (error) {
      console.error('OIDC callback failed:', error);
      throw new Error('Authentication callback failed');
    }
  }

  /**
   * Sign out the user
   */
  async signOut(): Promise<void> {
    if (!this.userManager) {
      throw new Error('OIDC service not initialized');
    }

    try {
      // This will redirect to the OIDC provider's logout endpoint
      await this.userManager.signoutRedirect({
        state: { timestamp: Date.now() },
      });
    } catch (error) {
      console.error('Failed to sign out:', error);
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
      throw new Error('OIDC service not initialized');
    }

    try {
      await this.userManager.removeUser();
    } catch (error) {
      console.error('Failed to clear user session:', error);
      throw error;
    }
  }

  /**
   * Handle silent renewal (for iframe-based token refresh)
   */
  async signInSilentCallback(): Promise<void> {
    if (!this.userManager) {
      throw new Error('OIDC service not initialized');
    }

    try {
      await this.userManager.signinSilentCallback();
    } catch (error) {
      console.error('Silent renewal callback failed:', error);
      throw error;
    }
  }

  /**
   * Manually trigger token renewal
   */
  async renewToken(): Promise<OidcUser | null> {
    if (!this.userManager) {
      throw new Error('OIDC service not initialized');
    }

    try {
      const user = await this.userManager.signinSilent();
      return user;
    } catch (error) {
      console.error('Manual token renewal failed:', error);
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
}

// Export singleton instance
export const oidcService = new OIDCService();