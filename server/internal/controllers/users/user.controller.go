package userController

import (
	"context"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"github.com/google/uuid"
)

type UserController struct {
	userRepo           repositories.UserRepository
	userConfigRepo     repositories.UserConfigurationRepository
	folderRepo         repositories.FolderRepository
	userReleaseRepo    repositories.UserReleaseRepository
	userStylusRepo     repositories.StylusRepository
	historyRepo        repositories.HistoryRepository
	recommendationRepo repositories.DailyRecommendationRepository
	discogsService     *services.DiscogsService
	db                 database.DB
	Config             config.Config
	log                logger.Logger
}

type GetUserResponse struct {
	Folders             []*Folder            `json:"folders"`
	Releases            []*UserRelease       `json:"releases"`
	Styluses            []*UserStylus        `json:"styluses"`
	PlayHistory         []*PlayHistory       `json:"playHistory"`
	DailyRecommendation *DailyRecommendation `json:"dailyRecommendation"`
}

type UpdateUserPreferencesRequest struct {
	RecentlyPlayedThresholdDays   *int `json:"recentlyPlayedThresholdDays"`
	CleaningFrequencyPlays        *int `json:"cleaningFrequencyPlays"`
	NeglectedRecordsThresholdDays *int `json:"neglectedRecordsThresholdDays"`
}

type UserControllerInterface interface {
	UpdateDiscogsToken(
		ctx context.Context,
		user *User,
		token string,
	) (*User, error)
	GetUser(ctx context.Context, user *User) (*GetUserResponse, error)
	UpdateSelectedFolder(
		ctx context.Context,
		user *User,
		folderID int,
	) (*User, error)
	UpdateUserPreferences(
		ctx context.Context,
		user *User,
		preferences *UpdateUserPreferencesRequest,
	) (*User, error)
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
	db database.DB,
) UserControllerInterface {
	return &UserController{
		userRepo:           repos.User,
		userConfigRepo:     repos.UserConfiguration,
		folderRepo:         repos.Folder,
		userReleaseRepo:    repos.UserRelease,
		userStylusRepo:     repos.Stylus,
		historyRepo:        repos.History,
		recommendationRepo: repos.DailyRecommendation,
		discogsService:     services.Discogs,
		db:                 db,
		Config:             config,
		log:                logger.New("userController"),
	}
}

func (uc *UserController) UpdateDiscogsToken(
	ctx context.Context,
	user *User,
	token string,
) (*User, error) {
	log := uc.log.Function("UpdateDiscogsToken")

	if token == "" {
		return nil, log.ErrMsg("token is required")
	}

	identity, err := uc.discogsService.GetUserIdentity(token)
	if err != nil {
		log.Warn("Invalid Discogs token provided", "userID", user.ID, "error", err)
		return nil, log.Err("invalid discogs token", err)
	}

	config := &UserConfiguration{
		UserID:          user.ID,
		DiscogsToken:    &token,
		DiscogsUsername: &identity.Username,
	}

	if err := uc.userConfigRepo.CreateOrUpdate(ctx, uc.db.SQL, config, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with discogs credentials", err)
	}

	user.Configuration = config

	log.Info(
		"Discogs credentials updated successfully",
		"userID",
		user.ID,
		"username",
		identity.Username,
	)

	return user, nil
}

func (uc *UserController) GetUser(
	ctx context.Context,
	user *User,
) (*GetUserResponse, error) {
	log := uc.log.Function("GetUser")

	// Get user folders
	folders, err := uc.folderRepo.GetUserFolders(ctx, uc.db.SQL, user.ID)
	if err != nil {
		return nil, log.Err("failed to get user folders", err, "userID", user.ID)
	}

	// Get user releases for selected folder
	var releases []*UserRelease

	log.Info(
		"selected folder ID",
		"userID",
		user.ID,
		"folderID",
		user.Configuration,
	)
	if user.Configuration != nil && user.Configuration.SelectedFolderID != nil {
		// Get the folder using the composite key lookup
		var selectedFolder *Folder
		selectedFolder, err = uc.folderRepo.GetFolderByID(
			ctx,
			uc.db.SQL,
			user.ID,
			*user.Configuration.SelectedFolderID,
		)
		if err != nil {
			log.Warn(
				"selected folder not found, returning empty releases (likely needs sync)",
				"userID",
				user.ID,
				"folderID",
				*user.Configuration.SelectedFolderID,
			)
			log.Info(
				"selected folder not found",
				"userID",
				user.ID,
				"folderID",
				*user.Configuration.SelectedFolderID,
			)
			releases = []*UserRelease{}
		} else {
			// Use the folder's ID to query user releases

			releases, err = uc.userReleaseRepo.GetUserReleasesByFolderID(
				ctx,
				uc.db.SQL,
				user.ID,
				*selectedFolder.ID,
			)
			log.Info("selected folder found", "userID", user.ID, "releaseCount", len(releases))
			if err != nil {
				return nil, log.Err(
					"failed to get user releases",
					err,
					"userID",
					user.ID,
					"folderID",
					*user.Configuration.SelectedFolderID,
					"folderID",
					*selectedFolder.ID,
				)
			}
		}
	}

	log.Info("user releases found", "userID", user.ID, "releaseCount", len(releases))
	// Get user styluses
	styluses, err := uc.userStylusRepo.GetUserStyluses(ctx, uc.db.SQL, user.ID)
	if err != nil {
		return nil, log.Err("failed to get user styluses", err, "userID", user.ID)
	}

	// Get user play history (limit to 1000 most recent)
	playHistory, err := uc.historyRepo.GetUserPlayHistory(ctx, uc.db.SQL, user.ID, 1000)
	if err != nil {
		return nil, log.Err("failed to get user play history", err, "userID", user.ID)
	}

	// Get today's daily recommendation (if one exists)
	var dailyRecommendation *DailyRecommendation
	dailyRecommendation, err = uc.recommendationRepo.GetTodayRecommendation(ctx, uc.db.SQL, user.ID)
	if err != nil {
		log.Info("no daily recommendation found for today", "userID", user.ID)
		dailyRecommendation = nil
	}

	return &GetUserResponse{
		Folders:             folders,
		Releases:            releases,
		Styluses:            styluses,
		PlayHistory:         playHistory,
		DailyRecommendation: dailyRecommendation,
	}, nil
}

func (uc *UserController) UpdateSelectedFolder(
	ctx context.Context,
	user *User,
	folderID int,
) (*User, error) {
	log := uc.log.Function("UpdateSelectedFolder")

	// Validate that the folder exists and belongs to the user
	_, err := uc.folderRepo.GetFolderByID(ctx, uc.db.SQL, user.ID, folderID)
	if err != nil {
		return nil, log.Err("folder not found or not owned by user", err)
	}

	if user.Configuration == nil {
		return nil, log.ErrMsg(
			"user configuration not found, please set up Discogs integration first",
		)
	}

	user.Configuration.SelectedFolderID = &folderID

	if err := uc.userConfigRepo.Update(ctx, uc.db.SQL, user.Configuration, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with selected folder", err)
	}

	// Clear the user folders cache since the configuration changed
	if err := uc.clearUserFoldersCache(ctx, user.ID); err != nil {
		log.Warn("failed to clear user folders cache", "userID", user.ID, "error", err)
	}

	log.Info(
		"Selected folder updated successfully",
		"userID",
		user.ID,
		"folderID",
		folderID,
	)

	return user, nil
}

func (uc *UserController) clearUserFoldersCache(ctx context.Context, userID uuid.UUID) error {
	return uc.folderRepo.ClearUserFoldersCache(ctx, userID)
}

func (uc *UserController) UpdateUserPreferences(
	ctx context.Context,
	user *User,
	preferences *UpdateUserPreferencesRequest,
) (*User, error) {
	log := uc.log.Function("UpdateUserPreferences")

	if user.Configuration == nil {
		return nil, log.ErrMsg(
			"user configuration not found, please set up Discogs integration first",
		)
	}

	if preferences.RecentlyPlayedThresholdDays != nil {
		if *preferences.RecentlyPlayedThresholdDays < 1 ||
			*preferences.RecentlyPlayedThresholdDays > 365 {
			return nil, log.ErrMsg("recentlyPlayedThresholdDays must be between 1 and 365")
		}
		user.Configuration.RecentlyPlayedThresholdDays = preferences.RecentlyPlayedThresholdDays
	}

	if preferences.CleaningFrequencyPlays != nil {
		if *preferences.CleaningFrequencyPlays < 1 || *preferences.CleaningFrequencyPlays > 50 {
			return nil, log.ErrMsg("cleaningFrequencyPlays must be between 1 and 50")
		}
		user.Configuration.CleaningFrequencyPlays = preferences.CleaningFrequencyPlays
	}

	if preferences.NeglectedRecordsThresholdDays != nil {
		if *preferences.NeglectedRecordsThresholdDays < 1 ||
			*preferences.NeglectedRecordsThresholdDays > 730 {
			return nil, log.ErrMsg("neglectedRecordsThresholdDays must be between 1 and 730")
		}
		user.Configuration.NeglectedRecordsThresholdDays = preferences.NeglectedRecordsThresholdDays
	}

	if err := uc.userConfigRepo.Update(ctx, uc.db.SQL, user.Configuration, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user preferences", err)
	}

	log.Info(
		"User preferences updated successfully",
		"userID",
		user.ID,
		"recentlyPlayedThresholdDays",
		user.Configuration.RecentlyPlayedThresholdDays,
		"cleaningFrequencyPlays",
		user.Configuration.CleaningFrequencyPlays,
		"neglectedRecordsThresholdDays",
		user.Configuration.NeglectedRecordsThresholdDays,
	)

	return user, nil
}
