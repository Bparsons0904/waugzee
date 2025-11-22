package authController

import (
	"context"
	"strings"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"
	"waugzee/internal/types"
)

type AuthController struct {
	zitadelService *services.ZitadelService
	userRepo       repositories.UserRepository
	historyRepo    repositories.HistoryRepository
	stylusRepo     repositories.StylusRepository
	db             database.DB
}

type AuthControllerInterface interface {
	GetAuthConfig() (*AuthConfigResponse, error)
	HandleOIDCCallback(ctx context.Context, req OIDCCallbackRequest) (*TokenExchangeResult, error)
	LogoutUser(ctx context.Context, req LogoutRequest, user *User) (*LogoutResponse, error)
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

func New(
	services services.Service,
	repos repositories.Repository,
	db database.DB,
) AuthControllerInterface {
	return &AuthController{
		zitadelService: services.Zitadel,
		userRepo:       repos.User,
		historyRepo:    repos.History,
		stylusRepo:     repos.Stylus,
		db:             db,
	}
}

func (ac *AuthController) GetAuthConfig() (*AuthConfigResponse, error) {
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
	tokenInfo *types.TokenInfo,
) (*User, error) {
	log := logger.NewWithContext(ctx, "authController").Function("getOrCreateOIDCUser")

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

	// Create User struct directly
	user := &User{
		FirstName:       firstName,
		LastName:        lastName,
		FullName:        displayName,
		DisplayName:     displayName,
		IsAdmin:         false,
		IsActive:        true,
		OIDCUserID:      tokenInfo.UserID,
		ProfileVerified: tokenInfo.EmailVerified,
	}

	// Set email only if present and verified
	if tokenInfo.Email != "" && tokenInfo.EmailVerified {
		user.Email = &tokenInfo.Email
	}

	// Set OIDC provider
	provider := "zitadel"
	user.OIDCProvider = &provider

	// Set project ID if available
	if tokenInfo.ProjectID != "" {
		user.OIDCProjectID = &tokenInfo.ProjectID
	}

	user, err := ac.userRepo.FindOrCreateOIDCUser(ctx, ac.db.SQL, user)
	if err != nil {
		log.Info(
			"failed to find or create OIDC user",
			"error",
			err.Error(),
			"oidcUserID",
			tokenInfo.UserID,
		)
		return nil, log.ErrMsg("failed to create user session")
	}

	return user, nil
}

// HandleOIDCCallback handles the OIDC callback - supports both code flow and token flow
func (ac *AuthController) HandleOIDCCallback(
	ctx context.Context,
	req OIDCCallbackRequest,
) (*TokenExchangeResult, error) {
	log := logger.NewWithContext(ctx, "authController").Function("HandleOIDCCallback")

	tokenInfo, err := ac.zitadelService.ValidateIDToken(ctx, req.IDToken)
	if err != nil {
		log.Info("ID token validation failed", "error", err.Error())
		return nil, log.ErrMsg("authentication failed")
	}

	if !tokenInfo.Valid {
		log.Info("ID token is invalid")
		return nil, log.ErrMsg("authentication failed")
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
		return nil, log.ErrMsg("authentication failed")
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
	user *User,
) (*LogoutResponse, error) {
	log := logger.NewWithContext(ctx, "authController").Function("LogoutUser")

	var oidcUserID string
	if user != nil {
		log.Info("processing logout request", "dbUserID", user.ID)
	}

	// Get OIDC User ID from the ID token since cached user may not have it
	if req.AccessToken != "" {
		tokenInfo, err := ac.zitadelService.ValidateIDToken(ctx, req.AccessToken)
		if err != nil {
			log.Warn("failed to validate ID token during logout", "error", err.Error())
		} else {
			oidcUserID = tokenInfo.UserID
			log.Info("extracted OIDC user ID from token", "oidcUserID", oidcUserID)
		}
	}

	var revokedTokens []string

	// Revoke access token if present
	if req.AccessToken != "" {
		if err := ac.zitadelService.RevokeToken(ctx, req.AccessToken, "access_token"); err != nil {
			log.Warn("failed to revoke access token", "error", err.Error())
		} else {
			revokedTokens = append(revokedTokens, "access_token")
			log.Info("access token revoked successfully")
		}
	}

	// Revoke refresh token if provided
	if req.RefreshToken != "" {
		if err := ac.zitadelService.RevokeToken(ctx, req.RefreshToken, "refresh_token"); err != nil {
			log.Warn("failed to revoke refresh token", "error", err.Error())
		} else {
			revokedTokens = append(revokedTokens, "refresh_token")
			log.Info("refresh token revoked successfully")
		}
	}

	// Clear user cache data using OIDC User ID from token
	if oidcUserID != "" {
		if err := ac.userRepo.ClearUserCacheByOIDC(ctx, oidcUserID); err != nil {
			log.Warn(
				"failed to clear user cache",
				"error",
				err.Error(),
				"oidcUserID",
				oidcUserID,
			)
		} else {
			log.Info("user cache cleared successfully", "oidcUserID", oidcUserID)
		}
	} else {
		log.Warn("no OIDC user ID available for cache clearing")
	}

	if user != nil {
		if err := ac.historyRepo.ClearUserHistoryCache(ctx, user.ID); err != nil {
			log.Warn(
				"failed to clear user history cache",
				"error",
				err.Error(),
				"userID",
				user.ID,
			)
		} else {
			log.Info("user history cache cleared successfully", "userID", user.ID)
		}

		if err := ac.stylusRepo.ClearUserStylusCache(ctx, user.ID); err != nil {
			log.Warn(
				"failed to clear user stylus cache",
				"error",
				err.Error(),
				"userID",
				user.ID,
			)
		} else {
			log.Info("user stylus cache cleared successfully", "userID", user.ID)
		}
	}

	// Generate logout URL
	var logoutURL string
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

	if oidcUserID != "" {
		log.Info(
			"user logout completed",
			"oidcUserID",
			oidcUserID,
			"revokedTokens",
			len(revokedTokens),
		)
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
