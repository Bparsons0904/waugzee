package authController

import (
	"context"
	"fmt"
	"strings"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

// AuthController handles authentication business logic
type AuthController struct {
	zitadelService *services.ZitadelService
	userRepo       repositories.UserRepository
	db             database.DB
	log            logger.Logger
}

// AuthControllerInterface defines the contract for auth business logic
type AuthControllerInterface interface {
	// Authentication flow methods
	GetAuthConfig() (*AuthConfigResponse, error)
	GenerateAuthURL(state, redirectURI, codeChallenge, nonce string) (*AuthURLResponse, error)
	ValidateAndExchangeToken(ctx context.Context, req services.TokenExchangeRequest) (*TokenExchangeResult, error)
	HandleOIDCCallback(ctx context.Context, req OIDCCallbackRequest) (*TokenExchangeResult, error)
	
	// User session methods
	GetCurrentUserInfo(ctx context.Context, authInfo *AuthInfo) (*UserProfileResponse, error)
	LogoutUser(ctx context.Context, req LogoutRequest, authInfo *AuthInfo) (*LogoutResponse, error)
	
	// Admin methods
	GetAllUsers(ctx context.Context, authInfo *AuthInfo) (*AllUsersResponse, error)
	
	// Utility methods
	IsConfigured() bool
}

// Request/Response types
type AuthConfigResponse struct {
	Configured  bool   `json:"configured"`
	Domain      string `json:"domain,omitempty"`
	InstanceURL string `json:"instanceUrl,omitempty"`
	ClientID    string `json:"clientId,omitempty"`
	Message     string `json:"message,omitempty"`
}

type AuthURLResponse struct {
	AuthorizationURL string `json:"authorizationUrl"`
	State            string `json:"state"`
}

type UserProfileResponse struct {
	User interface{} `json:"user"`
}

type TokenExchangeResult struct {
	AccessToken  string      `json:"access_token"`
	TokenType    string      `json:"token_type"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	ExpiresIn    int64       `json:"expires_in"`
	IDToken      string      `json:"id_token,omitempty"`
	State        string      `json:"state,omitempty"`
	User         interface{} `json:"user"`
}

type OIDCCallbackRequest struct {
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	State        string `json:"state,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
}

type LogoutRequest struct {
	RefreshToken          string `json:"refresh_token,omitempty"`
	IDToken               string `json:"id_token,omitempty"`
	PostLogoutRedirectURI string `json:"post_logout_redirect_uri,omitempty"`
	State                 string `json:"state,omitempty"`
	AccessToken           string `json:"access_token,omitempty"`
}

type LogoutResponse struct {
	Message       string   `json:"message"`
	LogoutURL     string   `json:"logout_url,omitempty"`
	RevokedTokens []string `json:"revoked_tokens,omitempty"`
}

type AllUsersResponse struct {
	Message string        `json:"message"`
	Users   []interface{} `json:"users"`
}

type AuthInfo struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	ProjectID string   `json:"project_id"`
}

// New creates a new AuthController instance
func New(
	zitadelService *services.ZitadelService,
	userRepo repositories.UserRepository,
	db database.DB,
) AuthControllerInterface {
	return &AuthController{
		zitadelService: zitadelService,
		userRepo:       userRepo,
		db:             db,
		log:            logger.New("authController"),
	}
}

// IsConfigured checks if Zitadel service is properly configured
func (c *AuthController) IsConfigured() bool {
	return c.zitadelService.IsConfigured()
}

// GetAuthConfig returns authentication configuration for the client
func (c *AuthController) GetAuthConfig() (*AuthConfigResponse, error) {
	log := c.log.Function("GetAuthConfig")

	if !c.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return &AuthConfigResponse{
			Configured: false,
			Message:    "Authentication not configured",
		}, nil
	}

	config := c.zitadelService.GetConfig()
	return &AuthConfigResponse{
		Configured:  true,
		Domain:      config.Domain,
		InstanceURL: config.InstanceURL,
		ClientID:    config.ClientID,
	}, nil
}

// GenerateAuthURL creates an authorization URL for OIDC login flow
func (c *AuthController) GenerateAuthURL(state, redirectURI, codeChallenge, nonce string) (*AuthURLResponse, error) {
	log := c.log.Function("GenerateAuthURL")

	if !c.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return nil, fmt.Errorf("authentication not configured")
	}

	if redirectURI == "" {
		log.Info("missing redirect_uri parameter")
		return nil, fmt.Errorf("redirect_uri parameter is required")
	}

	// Store nonce in cache if provided
	if nonce != "" {
		if err := c.storeNonce(context.Background(), nonce); err != nil {
			log.Info("failed to store nonce", "error", err.Error())
			// Continue without failing - nonce validation will be optional
		}
	}

	authURL := c.zitadelService.GetAuthorizationURL(state, redirectURI, codeChallenge, nonce)
	if authURL == "" {
		log.Info("failed to generate authorization URL")
		return nil, fmt.Errorf("failed to generate authorization URL")
	}

	log.Info("generated authorization URL", "state", state, "redirectURI", redirectURI, "hasPKCE", codeChallenge != "")
	return &AuthURLResponse{
		AuthorizationURL: authURL,
		State:            state,
	}, nil
}

// ValidateAndExchangeToken handles OIDC token exchange and user creation
func (c *AuthController) ValidateAndExchangeToken(ctx context.Context, req services.TokenExchangeRequest) (*TokenExchangeResult, error) {
	log := c.log.Function("ValidateAndExchangeToken")

	if !c.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return nil, fmt.Errorf("authentication not configured")
	}

	if req.Code == "" || req.RedirectURI == "" {
		log.Info("missing required fields", "code", req.Code != "", "redirectURI", req.RedirectURI != "", "hasCodeVerifier", req.CodeVerifier != "")
		return nil, fmt.Errorf("code and redirect_uri are required")
	}

	// Exchange code for token
	tokenResp, err := c.zitadelService.ExchangeCodeForToken(ctx, req)
	if err != nil {
		log.Info("token exchange failed", "error", err.Error())
		return nil, fmt.Errorf("token exchange failed")
	}

	// Validate the token and get user info
	tokenInfo, err := c.validateToken(ctx, tokenResp)
	if err != nil {
		return nil, err
	}

	// Validate nonce if present
	if tokenInfo.Nonce != "" {
		if err := c.validateAndCleanupNonce(ctx, tokenInfo.Nonce); err != nil {
			log.Info("nonce validation failed", "error", err.Error(), "nonce", tokenInfo.Nonce)
			return nil, fmt.Errorf("nonce validation failed")
		}
		log.Info("nonce validation successful", "nonce", tokenInfo.Nonce)
	}

	// Find or create user from OIDC claims
	user, err := c.getOrCreateOIDCUser(ctx, tokenInfo)
	if err != nil {
		return nil, err
	}

	log.Info("token exchange successful", "userID", user.ID, "email", user.Email)

	// Build response
	result := &TokenExchangeResult{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		User:        user.ToProfile(),
	}

	if tokenResp.IDToken != "" {
		result.IDToken = tokenResp.IDToken
	}

	return result, nil
}

// validateToken validates either ID token or access token
func (c *AuthController) validateToken(ctx context.Context, tokenResp *services.TokenExchangeResponse) (*services.TokenInfo, error) {
	log := c.log.Function("validateToken")

	var tokenInfo *services.TokenInfo
	var err error

	// Use ID token for public OIDC clients, fallback to access token
	if tokenResp.IDToken != "" {
		tokenInfo, err = c.zitadelService.ValidateIDToken(ctx, tokenResp.IDToken)
	} else {
		tokenInfo, err = c.zitadelService.ValidateToken(ctx, tokenResp.AccessToken)
	}

	if err != nil || !tokenInfo.Valid {
		log.Info("token validation failed", "error", err, "hasIDToken", tokenResp.IDToken != "")
		return nil, fmt.Errorf("invalid token received")
	}

	return tokenInfo, nil
}

// getOrCreateOIDCUser finds or creates a user from OIDC token claims
func (c *AuthController) getOrCreateOIDCUser(ctx context.Context, tokenInfo *services.TokenInfo) (*models.User, error) {
	log := c.log.Function("getOrCreateOIDCUser")

	oidcReq := models.OIDCUserCreateRequest{
		OIDCUserID:      tokenInfo.UserID,
		Email:           &tokenInfo.Email,
		Name:            &tokenInfo.Name,
		FirstName:       tokenInfo.Name,
		LastName:        "",
		OIDCProvider:    "zitadel",
		OIDCProjectID:   &tokenInfo.ProjectID,
		ProfileVerified: true,
	}

	// Split name into first/last if possible
	if tokenInfo.Name != "" {
		names := strings.Fields(tokenInfo.Name)
		if len(names) > 0 {
			oidcReq.FirstName = names[0]
		}
		if len(names) > 1 {
			oidcReq.LastName = strings.Join(names[1:], " ")
		}
	}

	user, err := c.userRepo.FindOrCreateOIDCUser(ctx, oidcReq)
	if err != nil {
		log.Info("failed to find or create OIDC user", "error", err.Error(), "oidcUserID", tokenInfo.UserID)
		return nil, fmt.Errorf("failed to create user session")
	}

	return user, nil
}

// storeNonce stores a nonce value in the cache with TTL for later validation
func (c *AuthController) storeNonce(ctx context.Context, nonce string) error {
	log := c.log.Function("storeNonce")

	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	err := database.NewCacheBuilder(c.db.Cache.Session, nonce).
		WithHashPattern("nonce:%s").
		WithValue("1").
		WithTTL(10 * time.Minute).
		WithContext(ctx).
		Set()

	if err != nil {
		return log.Err("failed to store nonce in cache", err, "nonce", nonce)
	}

	log.Info("nonce stored successfully", "nonce", nonce)
	return nil
}

// validateAndCleanupNonce validates a nonce from ID token and removes it from cache
func (c *AuthController) validateAndCleanupNonce(ctx context.Context, nonce string) error {
	log := c.log.Function("validateAndCleanupNonce")

	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	// Check if nonce exists in cache
	var result string
	found, err := database.NewCacheBuilder(c.db.Cache.Session, nonce).
		WithHashPattern("nonce:%s").
		WithContext(ctx).
		Get(&result)

	if err != nil {
		return log.Err("failed to validate nonce", err, "nonce", nonce)
	}

	if !found {
		return log.Err("nonce not found or expired", fmt.Errorf("nonce validation failed"), "nonce", nonce)
	}

	// Remove nonce from cache (one-time use)
	err = database.NewCacheBuilder(c.db.Cache.Session, nonce).
		WithHashPattern("nonce:%s").
		WithContext(ctx).
		Delete()

	if err != nil {
		log.Warn("failed to cleanup nonce from cache", "error", err.Error(), "nonce", nonce)
		// Don't fail validation if cleanup fails
	}

	log.Info("nonce validated and cleaned up successfully", "nonce", nonce)
	return nil
}

// HandleOIDCCallback handles the OIDC callback from Zitadel
func (c *AuthController) HandleOIDCCallback(ctx context.Context, req OIDCCallbackRequest) (*TokenExchangeResult, error) {
	log := c.log.Function("HandleOIDCCallback")

	if !c.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return nil, fmt.Errorf("authentication not configured")
	}

	if req.Code == "" || req.RedirectURI == "" {
		log.Info("missing required parameters", "code", req.Code != "", "redirectURI", req.RedirectURI != "")
		return nil, fmt.Errorf("code and redirect_uri are required")
	}

	// Exchange code for token
	tokenReq := services.TokenExchangeRequest{
		Code:         req.Code,
		RedirectURI:  req.RedirectURI,
		State:        req.State,
		CodeVerifier: req.CodeVerifier,
	}

	tokenResp, err := c.zitadelService.ExchangeCodeForToken(ctx, tokenReq)
	if err != nil {
		log.Info("OIDC callback token exchange failed", "error", err.Error())
		return nil, fmt.Errorf("authentication failed")
	}

	// Validate the token and get user info
	tokenInfo, err := c.validateToken(ctx, tokenResp)
	if err != nil {
		log.Info("OIDC callback token validation failed", "error", err)
		return nil, fmt.Errorf("authentication failed")
	}

	// Validate nonce if present
	if tokenInfo.Nonce != "" {
		if err := c.validateAndCleanupNonce(ctx, tokenInfo.Nonce); err != nil {
			log.Info("OIDC callback nonce validation failed", "error", err.Error(), "nonce", tokenInfo.Nonce)
			return nil, fmt.Errorf("authentication failed")
		}
		log.Info("OIDC callback nonce validation successful", "nonce", tokenInfo.Nonce)
	}

	// Find or create user from OIDC claims
	user, err := c.getOrCreateOIDCUser(ctx, tokenInfo)
	if err != nil {
		log.Info("OIDC callback failed to create user", "error", err.Error(), "oidcUserID", tokenInfo.UserID)
		return nil, fmt.Errorf("authentication failed")
	}

	log.Info("OIDC callback successful", "userID", user.ID, "email", user.Email)
	return &TokenExchangeResult{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresIn:   tokenResp.ExpiresIn,
		State:       req.State,
		User:        user.ToProfile(),
	}, nil
}

// GetCurrentUserInfo returns information about the currently authenticated user
func (c *AuthController) GetCurrentUserInfo(ctx context.Context, authInfo *AuthInfo) (*UserProfileResponse, error) {
	log := c.log.Function("GetCurrentUserInfo")

	if authInfo == nil {
		log.Info("no auth info provided")
		return nil, fmt.Errorf("authentication required")
	}

	// Fetch user from our local database using OIDC user ID
	user, err := c.userRepo.GetByOIDCUserID(ctx, authInfo.UserID)
	if err != nil {
		log.Info("failed to fetch user from database", "error", err.Error(), "oidcUserID", authInfo.UserID)
		// Fallback to basic info from token if database fetch fails
		return &UserProfileResponse{
			User: map[string]interface{}{
				"id":        authInfo.UserID,
				"email":     authInfo.Email,
				"name":      authInfo.Name,
				"roles":     authInfo.Roles,
				"projectId": authInfo.ProjectID,
			},
		}, nil
	}

	// Return user profile from our database
	userProfile := user.ToProfile()
	userProfile.ID = authInfo.UserID // Use OIDC ID for consistency

	log.Info("user profile retrieved from database", "userID", user.ID, "oidcUserID", authInfo.UserID)
	return &UserProfileResponse{
		User: userProfile,
	}, nil
}

// LogoutUser handles user logout with proper OIDC token revocation and cache cleanup
func (c *AuthController) LogoutUser(ctx context.Context, req LogoutRequest, authInfo *AuthInfo) (*LogoutResponse, error) {
	log := c.log.Function("LogoutUser")

	var userID string
	if authInfo != nil {
		userID = authInfo.UserID
		log.Info("processing logout request", "userID", userID)
	}

	var revokedTokens []string

	// Revoke access token if present
	if req.AccessToken != "" && c.zitadelService.IsConfigured() {
		if err := c.zitadelService.RevokeToken(ctx, req.AccessToken, "access_token"); err != nil {
			log.Warn("failed to revoke access token", "error", err.Error())
		} else {
			revokedTokens = append(revokedTokens, "access_token")
			log.Info("access token revoked successfully")
		}
	}

	// Revoke refresh token if provided
	if req.RefreshToken != "" && c.zitadelService.IsConfigured() {
		if err := c.zitadelService.RevokeToken(ctx, req.RefreshToken, "refresh_token"); err != nil {
			log.Warn("failed to revoke refresh token", "error", err.Error())
		} else {
			revokedTokens = append(revokedTokens, "refresh_token")
			log.Info("refresh token revoked successfully")
		}
	}

	// Clear user cache data if we have auth info
	if authInfo != nil {
		if err := c.clearUserCacheByOIDC(ctx, authInfo.UserID); err != nil {
			log.Warn("failed to clear user cache", "error", err.Error(), "oidcUserID", authInfo.UserID)
		} else {
			log.Info("user cache cleared successfully", "oidcUserID", authInfo.UserID)
		}
	}

	// Generate logout URL
	var logoutURL string
	if c.zitadelService.IsConfigured() {
		url, err := c.zitadelService.GetLogoutURL(ctx, req.IDToken, req.PostLogoutRedirectURI, req.State)
		if err != nil {
			log.Warn("failed to generate logout URL", "error", err.Error())
		} else {
			logoutURL = url
			log.Info("logout URL generated successfully")
		}
	}

	if userID != "" {
		log.Info("user logout completed", "userID", userID, "revokedTokens", len(revokedTokens))
	}

	response := &LogoutResponse{
		Message: "Logout successful",
	}

	if logoutURL != "" {
		response.LogoutURL = logoutURL
	}

	if len(revokedTokens) > 0 {
		response.RevokedTokens = revokedTokens
	}

	return response, nil
}

// clearUserCacheByOIDC clears user cache by OIDC user ID
func (c *AuthController) clearUserCacheByOIDC(ctx context.Context, oidcUserID string) error {
	log := c.log.Function("clearUserCacheByOIDC")

	// Get user from database to find UUID for cache cleanup
	user, err := c.userRepo.GetByOIDCUserID(ctx, oidcUserID)
	if err != nil {
		log.Warn("failed to get user for cache cleanup", "error", err.Error(), "oidcUserID", oidcUserID)
		return err
	}

	// Clear user cache by UUID
	if err := database.NewCacheBuilder(c.db.Cache.User, user.ID.String()).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "userID", user.ID, "error", err)
		return err
	}

	// Clear OIDC mapping cache
	oidcCacheKey := "oidc:" + oidcUserID
	if err := database.NewCacheBuilder(c.db.Cache.User, oidcCacheKey).Delete(); err != nil {
		log.Warn("failed to remove OIDC mapping from cache", "oidcUserID", oidcUserID, "error", err)
		return err
	}

	return nil
}

// GetAllUsers returns all users (admin only)
func (c *AuthController) GetAllUsers(ctx context.Context, authInfo *AuthInfo) (*AllUsersResponse, error) {
	log := c.log.Function("GetAllUsers")

	if authInfo != nil {
		log.Info("admin requesting all users", "adminID", authInfo.UserID)
	}

	// This is a placeholder - in a real implementation, you'd fetch users from Zitadel
	return &AllUsersResponse{
		Message: "Admin endpoint - get all users",
		Users:   []interface{}{},
	}, nil
}