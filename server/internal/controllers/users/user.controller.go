package userController

import (
	"context"
	"fmt"
	"waugzee/config"
	"waugzee/internal/logger"
	"waugzee/internal/repositories"
)

type UserController struct {
	userRepo repositories.UserRepository
	Config   config.Config
	log      logger.Logger
}

func New(
	userRepo repositories.UserRepository,
	config config.Config,
) *UserController {
	return &UserController{
		userRepo: userRepo,
		Config:   config,
		log:      logger.New("userController"),
	}
}

// AuthInfo represents authentication information from middleware
type AuthInfo struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	ProjectID string   `json:"project_id"`
}

// UserProfileResponse represents the current user profile response
type UserProfileResponse struct {
	User interface{} `json:"user"`
}

// GetCurrentUser returns information about the currently authenticated user
func (c *UserController) GetCurrentUser(ctx context.Context, authInfo *AuthInfo) (*UserProfileResponse, error) {
	log := c.log.Function("GetCurrentUser")

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
