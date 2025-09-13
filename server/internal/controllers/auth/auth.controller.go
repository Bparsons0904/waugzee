package authController

import (
	"context"
	"fmt"
	"strings"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
)

type AuthController struct {
	zitadelService *services.ZitadelService
	userRepo       repositories.UserRepository
	db             database.DB
	log            logger.Logger
}

type AuthControllerInterface interface {
	GetAuthConfig() (*AuthConfigResponse, error)
	HandleOIDCCallback(ctx context.Context, req OIDCCallbackRequest) (*TokenExchangeResult, error)
	LogoutUser(ctx context.Context, req LogoutRequest, authInfo *AuthInfo) (*LogoutResponse, error)
	IsConfigured() bool
}

type AuthConfigResponse struct {
	Configured  bool   `json:"configured"`
	Domain      string `json:"domain,omitempty"`
	InstanceURL string `json:"instanceUrl,omitempty"`
	ClientID    string `json:"clientId,omitempty"`
	Message     string `json:"message,omitempty"`
}


type TokenExchangeResult struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	IDToken      string `json:"id_token,omitempty"`
	State        string `json:"state,omitempty"`
	User         User   `json:"user"`
}

type OIDCCallbackRequest struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	State       string `json:"state,omitempty"`
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

func (ac *AuthController) IsConfigured() bool {
	return ac.zitadelService.IsConfigured()
}

func (ac *AuthController) GetAuthConfig() (*AuthConfigResponse, error) {
	log := ac.log.Function("GetAuthConfig")

	if !ac.zitadelService.IsConfigured() {
		log.Info("Zitadel not configured")
		return &AuthConfigResponse{
			Configured: false,
			Message:    "Authentication not configured",
		}, nil
	}

	config := ac.zitadelService.GetConfig()
	return &AuthConfigResponse{
		Configured:  true,
		Domain:      config.Domain,
		InstanceURL: config.InstanceURL,
		ClientID:    config.ClientID,
	}, nil
}




// getOrCreateOIDCUser finds or creates a user from OIDC token claims
func (ac *AuthController) getOrCreateOIDCUser(
	ctx context.Context,
	tokenInfo *services.TokenInfo,
) (*User, error) {
	log := ac.log.Function("getOrCreateOIDCUser")

	// Determine first and last names from various sources
	firstName := tokenInfo.GivenName
	lastName := tokenInfo.FamilyName

	// Fallback: parse the name field if given/family names aren't available
	if firstName == "" && lastName == "" && tokenInfo.Name != "" {
		names := strings.Fields(tokenInfo.Name)
		if len(names) > 0 {
			firstName = names[0]
		}
		if len(names) > 1 {
			lastName = strings.Join(names[1:], " ")
		}
	}

	// Use preferred name if available, otherwise build from first/last or use name
	displayName := tokenInfo.Name
	if displayName == "" && firstName != "" {
		displayName = firstName
		if lastName != "" {
			displayName += " " + lastName
		}
	}

	// Prepare email pointer (only if email is present and verified)
	var emailPtr *string
	if tokenInfo.Email != "" && tokenInfo.EmailVerified {
		emailPtr = &tokenInfo.Email
	}

	oidcReq := OIDCUserCreateRequest{
		OIDCUserID:      tokenInfo.UserID,
		Email:           emailPtr,
		Name:            &displayName,
		FirstName:       firstName,
		LastName:        lastName,
		OIDCProvider:    "zitadel",
		OIDCProjectID:   &tokenInfo.ProjectID,
		ProfileVerified: tokenInfo.EmailVerified,
	}

	user, err := ac.userRepo.FindOrCreateOIDCUser(ctx, oidcReq)
	if err != nil {
		log.Info(
			"failed to find or create OIDC user",
			"error",
			err.Error(),
			"oidcUserID",
			tokenInfo.UserID,
		)
		return nil, fmt.Errorf("failed to create user session")
	}

	return user, nil
}



// HandleOIDCCallback handles the OIDC callback - supports both code flow and token flow
func (ac *AuthController) HandleOIDCCallback(
	ctx context.Context,
	req OIDCCallbackRequest,
) (*TokenExchangeResult, error) {
	log := ac.log.Function("HandleOIDCCallback")

	if !ac.zitadelService.IsConfigured() {
		return nil, log.Error("authentication not configured")
	}

	tokenInfo, err := ac.zitadelService.ValidateIDToken(ctx, req.IDToken)
	if err != nil {
		log.Info("ID token validation failed", "error", err.Error())
		return nil, fmt.Errorf("authentication failed")
	}

	if !tokenInfo.Valid {
		log.Info("ID token is invalid")
		return nil, fmt.Errorf("authentication failed")
	}

	// Find or create user from OIDC claims
	user, err := ac.getOrCreateOIDCUser(ctx, tokenInfo)
	if err != nil {
		log.Info(
			"OIDC callback failed to create user",
			"error",
			err.Error(),
			"oidcUserID",
			tokenInfo.UserID,
		)
		return nil, fmt.Errorf("authentication failed")
	}

	log.Info("OIDC token callback successful", "userID", user.ID, "email", user.Email)
	return &TokenExchangeResult{
		AccessToken: req.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // Default expiry, client will handle renewal
		State:       req.State,
		User:        *user,
	}, nil
}

func (ac *AuthController) LogoutUser(
	ctx context.Context,
	req LogoutRequest,
	authInfo *AuthInfo,
) (*LogoutResponse, error) {
	log := ac.log.Function("LogoutUser")

	var userID string
	if authInfo != nil {
		userID = authInfo.UserID
		log.Info("processing logout request", "userID", userID)
	}

	var revokedTokens []string

	// Revoke access token if present
	if req.AccessToken != "" && ac.zitadelService.IsConfigured() {
		if err := ac.zitadelService.RevokeToken(ctx, req.AccessToken, "access_token"); err != nil {
			log.Warn("failed to revoke access token", "error", err.Error())
		} else {
			revokedTokens = append(revokedTokens, "access_token")
			log.Info("access token revoked successfully")
		}
	}

	// Revoke refresh token if provided
	if req.RefreshToken != "" && ac.zitadelService.IsConfigured() {
		if err := ac.zitadelService.RevokeToken(ctx, req.RefreshToken, "refresh_token"); err != nil {
			log.Warn("failed to revoke refresh token", "error", err.Error())
		} else {
			revokedTokens = append(revokedTokens, "refresh_token")
			log.Info("refresh token revoked successfully")
		}
	}

	// Clear user cache data if we have auth info
	if authInfo != nil {
		if err := ac.clearUserCacheByOIDC(ctx, authInfo.UserID); err != nil {
			log.Warn(
				"failed to clear user cache",
				"error",
				err.Error(),
				"oidcUserID",
				authInfo.UserID,
			)
		} else {
			log.Info("user cache cleared successfully", "oidcUserID", authInfo.UserID)
		}
	}

	// Generate logout URL
	var logoutURL string
	if ac.zitadelService.IsConfigured() {
		url, err := ac.zitadelService.GetLogoutURL(
			ctx,
			req.IDToken,
			req.PostLogoutRedirectURI,
			req.State,
		)
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
func (ac *AuthController) clearUserCacheByOIDC(ctx context.Context, oidcUserID string) error {
	log := ac.log.Function("clearUserCacheByOIDC")

	// Get user from database to find UUID for cache cleanup
	user, err := ac.userRepo.GetByOIDCUserID(ctx, oidcUserID)
	if err != nil {
		log.Warn(
			"failed to get user for cache cleanup",
			"error",
			err.Error(),
			"oidcUserID",
			oidcUserID,
		)
		return err
	}

	// Clear user cache by UUID
	if err := database.NewCacheBuilder(ac.db.Cache.User, user.ID.String()).Delete(); err != nil {
		log.Warn("failed to remove user from cache", "userID", user.ID, "error", err)
		return err
	}

	// Clear OIDC mapping cache
	oidcCacheKey := "oidc:" + oidcUserID
	if err := database.NewCacheBuilder(ac.db.Cache.User, oidcCacheKey).Delete(); err != nil {
		log.Warn("failed to remove OIDC mapping from cache", "oidcUserID", oidcUserID, "error", err)
		return err
	}

	return nil
}

// GetAllUsers returns all users (admin only)
func (ac *AuthController) GetAllUsers(
	ctx context.Context,
	authInfo *AuthInfo,
) (*AllUsersResponse, error) {
	log := ac.log.Function("GetAllUsers")

	if authInfo != nil {
		log.Info("admin requesting all users", "adminID", authInfo.UserID)
	}

	// This is a placeholder - in a real implementation, you'd fetch users from Zitadel
	return &AllUsersResponse{
		Message: "Admin endpoint - get all users",
		Users:   []interface{}{},
	}, nil
}

