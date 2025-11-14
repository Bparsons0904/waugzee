package userController

import (
	"context"
	"math/rand"
	"sort"
	"time"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	UserStreakCachePrefix = "user_streak"
	UserStreakCacheTTL    = 24 * time.Hour
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

type StreakData struct {
	CurrentStreak int `json:"currentStreak"`
	LongestStreak int `json:"longestStreak"`
}

type GetUserResponse struct {
	Folders             []*Folder            `json:"folders"`
	Releases            []*UserRelease       `json:"releases"`
	Styluses            []*UserStylus        `json:"styluses"`
	PlayHistory         []*PlayHistory       `json:"playHistory"`
	DailyRecommendation *DailyRecommendation `json:"dailyRecommendation"`
	Streak              *StreakData          `json:"streak"`
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

	dailyRecommendation, err := uc.getOrCreateRecommendation(ctx, user)
	if err != nil {
		log.Warn("failed to get or create recommendation", "userID", user.ID, "error", err)
		dailyRecommendation = nil
	}

	var streak *StreakData
	streak, err = uc.calculateUserStreak(ctx, user.ID)
	if err != nil {
		log.Warn("failed to calculate user streak", "userID", user.ID, "error", err)
		streak = nil
	}

	return &GetUserResponse{
		Folders:             folders,
		Releases:            releases,
		Styluses:            styluses,
		PlayHistory:         playHistory,
		DailyRecommendation: dailyRecommendation,
		Streak:              streak,
	}, nil
}

func (uc *UserController) getOrCreateRecommendation(
	ctx context.Context,
	user *User,
) (*DailyRecommendation, error) {
	log := uc.log.Function("getOrCreateRecommendation")

	mostRecent, err := uc.recommendationRepo.GetMostRecentRecommendation(ctx, uc.db.SQL, user.ID)

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		return uc.generateNewRecommendation(ctx, user)
	}

	if err != nil {
		return nil, log.Err("failed to get most recent recommendation", err, "userID", user.ID)
	}

	if mostRecent.ListenedAt == nil {
		createdHoursAgo := now.Sub(mostRecent.CreatedAt).Hours()

		if createdHoursAgo < 24 {
			log.Info(
				"returning existing unlistened recommendation",
				"userID",
				user.ID,
				"recommendationID",
				mostRecent.ID,
				"createdHoursAgo",
				createdHoursAgo,
			)
			return mostRecent, nil
		}

		log.Info(
			"existing recommendation is old, generating new one",
			"userID",
			user.ID,
			"recommendationID",
			mostRecent.ID,
			"createdHoursAgo",
			createdHoursAgo,
		)
		return uc.generateNewRecommendation(ctx, user)
	}

	listenedHoursAgo := now.Sub(*mostRecent.ListenedAt).Hours()

	if listenedHoursAgo < 18 {
		log.Info(
			"user completed recommendation recently, generating new one",
			"userID",
			user.ID,
			"recommendationID",
			mostRecent.ID,
			"listenedHoursAgo",
			listenedHoursAgo,
		)
		return uc.generateNewRecommendation(ctx, user)
	}

	log.Info(
		"returning existing listened recommendation",
		"userID",
		user.ID,
		"recommendationID",
		mostRecent.ID,
		"listenedHoursAgo",
		listenedHoursAgo,
	)
	return mostRecent, nil
}

func (uc *UserController) generateNewRecommendation(
	ctx context.Context,
	user *User,
) (*DailyRecommendation, error) {
	log := uc.log.Function("generateNewRecommendation")

	folderID := 0
	if user.Configuration != nil && user.Configuration.SelectedFolderID != nil {
		folderID = *user.Configuration.SelectedFolderID
	}

	releases, err := uc.userReleaseRepo.GetUserReleasesByFolderID(ctx, uc.db.SQL, user.ID, folderID)
	if err != nil {
		return nil, log.Err(
			"failed to get user releases",
			err,
			"userID",
			user.ID,
			"folderID",
			folderID,
		)
	}

	if len(releases) == 0 {
		return nil, log.Error("no releases found for user", "userID", user.ID, "folderID", folderID)
	}

	playHistory, err := uc.historyRepo.GetUserPlayHistory(ctx, uc.db.SQL, user.ID, 1000)
	if err != nil {
		log.Warn(
			"failed to get play history, falling back to random",
			"userID",
			user.ID,
			"error",
			err,
		)
		return uc.generateRandomRecommendation(ctx, user.ID, releases)
	}

	playCountMap := make(map[uuid.UUID]int)
	lastPlayedMap := make(map[uuid.UUID]time.Time)
	for _, play := range playHistory {
		playCountMap[play.UserReleaseID]++
		if lastPlayed, exists := lastPlayedMap[play.UserReleaseID]; !exists ||
			play.PlayedAt.After(lastPlayed) {
			lastPlayedMap[play.UserReleaseID] = play.PlayedAt
		}
	}

	type releaseWeight struct {
		UserRelease *UserRelease
		Weight      int
	}

	weights := make([]releaseWeight, 0, len(releases))
	now := time.Now()

	for _, release := range releases {
		baseWeight := 100

		playCount := playCountMap[release.ID]
		playPenalty := min(playCount*10, 95)

		lastPlayed, exists := lastPlayedMap[release.ID]
		recentPenalty := 0
		if exists {
			daysSincePlay := int(now.Sub(lastPlayed).Hours() / 24)
			if daysSincePlay < 30 {
				recentPenalty = 20
			}
		}

		randomBonus := rand.Intn(11)

		finalWeight := max(baseWeight-playPenalty-recentPenalty+randomBonus, 0)

		weights = append(weights, releaseWeight{
			UserRelease: release,
			Weight:      finalWeight,
		})
	}

	if len(weights) == 0 {
		log.Warn("no weighted releases available, falling back to random", "userID", user.ID)
		return uc.generateRandomRecommendation(ctx, user.ID, releases)
	}

	maxWeight := 0
	var selectedRelease *UserRelease
	for _, w := range weights {
		if w.Weight > maxWeight {
			maxWeight = w.Weight
			selectedRelease = w.UserRelease
		}
	}

	if selectedRelease == nil {
		log.Warn("no release selected by weight, falling back to random", "userID", user.ID)
		return uc.generateRandomRecommendation(ctx, user.ID, releases)
	}

	today := time.Now().Truncate(24 * time.Hour)
	recommendation := &DailyRecommendation{
		UserID:        user.ID,
		UserReleaseID: selectedRelease.ID,
		Date:          today,
		Algorithm:     "smart",
	}

	err = uc.recommendationRepo.CreateRecommendation(ctx, uc.db.SQL, recommendation)
	if err != nil {
		return nil, log.Err("failed to create recommendation", err, "userID", user.ID)
	}

	recommendation, err = uc.recommendationRepo.GetMostRecentRecommendation(
		ctx,
		uc.db.SQL,
		user.ID,
	)
	if err != nil {
		return nil, log.Err("failed to get newly created recommendation", err, "userID", user.ID)
	}

	log.Info(
		"generated smart recommendation",
		"userID",
		user.ID,
		"releaseID",
		selectedRelease.ReleaseID,
		"weight",
		maxWeight,
	)

	return recommendation, nil
}

func (uc *UserController) generateRandomRecommendation(
	ctx context.Context,
	userID uuid.UUID,
	releases []*UserRelease,
) (*DailyRecommendation, error) {
	log := uc.log.Function("generateRandomRecommendation")

	selectedRelease := releases[rand.Intn(len(releases))]

	today := time.Now().Truncate(24 * time.Hour)
	recommendation := &DailyRecommendation{
		UserID:        userID,
		UserReleaseID: selectedRelease.ID,
		Date:          today,
		Algorithm:     "random",
	}

	err := uc.recommendationRepo.CreateRecommendation(ctx, uc.db.SQL, recommendation)
	if err != nil {
		return nil, log.Err("failed to create random recommendation", err, "userID", userID)
	}

	recommendation, err = uc.recommendationRepo.GetMostRecentRecommendation(
		ctx,
		uc.db.SQL,
		userID,
	)
	if err != nil {
		return nil, log.Err("failed to get newly created recommendation", err, "userID", userID)
	}

	log.Info(
		"generated random recommendation",
		"userID",
		userID,
		"releaseID",
		selectedRelease.ReleaseID,
	)

	return recommendation, nil
}

func (uc *UserController) UpdateSelectedFolder(
	ctx context.Context,
	user *User,
	folderID int,
) (*User, error) {
	log := uc.log.Function("UpdateSelectedFolder")

	if user.Configuration == nil {
		return nil, log.ErrMsg(
			"user configuration not found, please set up Discogs integration first",
		)
	}

	user.Configuration.SelectedFolderID = &folderID

	if err := uc.userConfigRepo.Update(ctx, uc.db.SQL, user.Configuration, uc.userRepo); err != nil {
		return nil, log.Err("failed to update user configuration with selected folder", err)
	}

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

// Claude couldn't we do this with a sql query? Rather than returning every single record and parsing through them?
func (uc *UserController) calculateUserStreak(
	ctx context.Context,
	userID uuid.UUID,
) (*StreakData, error) {
	log := uc.log.Function("calculateUserStreak")

	var cachedStreak *StreakData
	found, err := database.NewCacheBuilder(uc.db.Cache.ClientAPI, userID.String()).
		WithContext(ctx).
		WithHash(UserStreakCachePrefix).
		Get(&cachedStreak)
	if err != nil {
		log.Warn("failed to get streak from cache", "userID", userID, "error", err)
	}

	if found {
		log.Info("streak retrieved from cache", "userID", userID)
		return cachedStreak, nil
	}

	recommendations, err := uc.recommendationRepo.GetAllUserRecommendations(ctx, uc.db.SQL, userID)
	if err != nil {
		return nil, log.Err("failed to get user recommendations", err, "userID", userID)
	}

	if len(recommendations) == 0 {
		streakData := &StreakData{
			CurrentStreak: 0,
			LongestStreak: 0,
		}
		uc.cacheStreak(ctx, userID, streakData)
		return streakData, nil
	}

	currentStreak := uc.calculateCurrentStreak(recommendations)
	longestStreak := uc.calculateLongestStreak(recommendations)

	streakData := &StreakData{
		CurrentStreak: currentStreak,
		LongestStreak: longestStreak,
	}

	err = database.NewCacheBuilder(uc.db.Cache.ClientAPI, userID.String()).
		WithContext(ctx).
		WithHash(UserStreakCachePrefix).
		WithStruct(streakData).
		WithTTL(UserStreakCacheTTL).
		Set()
	if err != nil {
		log.Warn("failed to cache streak data", "userID", userID, "error", err)
	}

	log.Info(
		"calculated user streak",
		"userID",
		userID,
		"currentStreak",
		currentStreak,
		"longestStreak",
		longestStreak,
	)

	return streakData, nil
}

func (uc *UserController) calculateCurrentStreak(recommendations []*DailyRecommendation) int {
	if len(recommendations) == 0 {
		return 0
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].CreatedAt.After(recommendations[j].CreatedAt)
	})

	streak := 0
	for _, rec := range recommendations {
		if rec.ListenedAt == nil {
			break
		}
		streak++
	}
	return streak
}

func (uc *UserController) calculateLongestStreak(recommendations []*DailyRecommendation) int {
	if len(recommendations) == 0 {
		return 0
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].CreatedAt.After(recommendations[j].CreatedAt)
	})

	longestStreak := 0
	currentStreak := 0

	for _, rec := range recommendations {
		if rec.ListenedAt == nil {
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
			currentStreak = 0
		} else {
			currentStreak++
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
		}
	}

	return longestStreak
}

func (uc *UserController) cacheStreak(
	ctx context.Context,
	userID uuid.UUID,
	streakData *StreakData,
) {
	err := database.NewCacheBuilder(uc.db.Cache.ClientAPI, userID.String()).
		WithContext(ctx).
		WithHash(UserStreakCachePrefix).
		WithStruct(streakData).
		WithTTL(UserStreakCacheTTL).
		Set()
	if err != nil {
		uc.log.Warn("failed to cache empty streak data", "userID", userID, "error", err)
	}
}
