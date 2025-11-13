package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecommendationService struct {
	userReleaseRepo    repositories.UserReleaseRepository
	historyRepo        repositories.HistoryRepository
	recommendationRepo repositories.DailyRecommendationRepository
	userRepo           repositories.UserRepository
	folderRepo         repositories.FolderRepository
	cache              database.CacheClient
	db                 *gorm.DB
	log                logger.Logger
}

func NewRecommendationService(
	repos repositories.Repository,
	db *gorm.DB,
	cache database.CacheClient,
) *RecommendationService {
	return &RecommendationService{
		userReleaseRepo:    repos.UserRelease,
		historyRepo:        repos.History,
		recommendationRepo: repos.DailyRecommendation,
		userRepo:           repos.User,
		folderRepo:         repos.Folder,
		cache:              cache,
		db:                 db,
		log:                logger.New("recommendationService"),
	}
}

type releaseWeight struct {
	UserRelease *UserRelease
	Weight      int
}

func (s *RecommendationService) GenerateSmartRecommendation(
	ctx context.Context,
	userID uuid.UUID,
	folderID int,
) error {
	log := s.log.Function("GenerateSmartRecommendation")

	releases, err := s.userReleaseRepo.GetUserReleasesByFolderID(ctx, s.db, userID, folderID)
	if err != nil {
		return log.Err(
			"failed to get user releases",
			err,
			"userID",
			userID,
			"folderID",
			folderID,
		)
	}

	if len(releases) == 0 {
		return log.Error("no releases found for user", "userID", userID, "folderID", folderID)
	}

	playHistory, err := s.historyRepo.GetUserPlayHistory(ctx, s.db, userID, 1000)
	if err != nil {
		log.Warn(
			"failed to get play history, falling back to random",
			"userID",
			userID,
			"error",
			err,
		)
		return s.generateRandomRecommendation(ctx, userID, releases)
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
		log.Warn("no weighted releases available, falling back to random", "userID", userID)
		return s.generateRandomRecommendation(ctx, userID, releases)
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
		log.Warn("no release selected by weight, falling back to random", "userID", userID)
		return s.generateRandomRecommendation(ctx, userID, releases)
	}

	today := time.Now().Truncate(24 * time.Hour)
	recommendation := &DailyRecommendation{
		UserID:        userID,
		UserReleaseID: selectedRelease.ID,
		Date:          today,
		Algorithm:     "smart",
	}

	err = s.recommendationRepo.CreateRecommendation(ctx, s.db, recommendation)
	if err != nil {
		return log.Err("failed to create recommendation", err, "userID", userID)
	}

	log.Info(
		"generated smart recommendation",
		"userID",
		userID,
		"releaseID",
		selectedRelease.ReleaseID,
		"weight",
		maxWeight,
	)

	return nil
}

func (s *RecommendationService) generateRandomRecommendation(
	ctx context.Context,
	userID uuid.UUID,
	releases []*UserRelease,
) error {
	log := s.log.Function("generateRandomRecommendation")

	if len(releases) == 0 {
		return log.Error("no releases available for random recommendation", "userID", userID)
	}

	randomIndex := rand.Intn(len(releases))
	selectedRelease := releases[randomIndex]

	today := time.Now().Truncate(24 * time.Hour)
	recommendation := &DailyRecommendation{
		UserID:        userID,
		UserReleaseID: selectedRelease.ID,
		Date:          today,
		Algorithm:     "random",
	}

	err := s.recommendationRepo.CreateRecommendation(ctx, s.db, recommendation)
	if err != nil {
		return log.Err("failed to create random recommendation", err, "userID", userID)
	}

	log.Info(
		"generated random recommendation",
		"userID",
		userID,
		"releaseID",
		selectedRelease.ReleaseID,
	)

	return nil
}

func (s *RecommendationService) GenerateDailyRecommendationsForAllUsers(ctx context.Context) error {
	log := s.log.Function("GenerateDailyRecommendationsForAllUsers")

	users, err := s.userRepo.GetAllUsers(ctx, s.db)
	if err != nil {
		return log.Err("failed to get all users", err)
	}

	successCount := 0
	failureCount := 0

	for _, user := range users {
		var folders []*Folder
		folders, err = s.folderRepo.GetUserFolders(ctx, s.db, user.ID)
		if err != nil {
			log.Warn("failed to get user folders, skipping", "userID", user.ID, "error", err)
			failureCount++
			continue
		}

		if len(folders) == 0 {
			log.Warn("user has no folders, skipping", "userID", user.ID)
			failureCount++
			continue
		}

		var defaultFolder *Folder
		for _, folder := range folders {
			if folder.ID != nil && *folder.ID == 0 {
				defaultFolder = folder
				break
			}
		}

		if defaultFolder == nil {
			defaultFolder = folders[0]
		}

		err = s.GenerateSmartRecommendation(ctx, user.ID, *defaultFolder.ID)
		if err != nil {
			log.Warn("failed to generate recommendation for user", "userID", user.ID, "error", err)
			failureCount++
			continue
		}

		successCount++
	}

	err = s.ClearAllUserCaches(ctx)
	if err != nil {
		log.Warn("failed to clear all user caches", "error", err)
	}

	log.Info(
		"completed daily recommendation generation",
		"totalUsers",
		len(users),
		"successful",
		successCount,
		"failed",
		failureCount,
	)

	if failureCount > 0 {
		return fmt.Errorf(
			"failed to generate recommendations for %d/%d users",
			failureCount,
			len(users),
		)
	}

	return nil
}

func (s *RecommendationService) ClearAllUserCaches(ctx context.Context) error {
	log := s.log.Function("ClearAllUserCaches")

	users, err := s.userRepo.GetAllUsers(ctx, s.db)
	if err != nil {
		return log.Err("failed to get all users for cache clearing", err)
	}

	for _, user := range users {
		if err := s.historyRepo.ClearUserHistoryCache(ctx, user.ID); err != nil {
			log.Warn("failed to clear history cache", "userID", user.ID, "error", err)
		}
		if err := s.recommendationRepo.ClearUserRecommendationCache(ctx, user.ID); err != nil {
			log.Warn("failed to clear recommendation cache", "userID", user.ID, "error", err)
		}
		if err := s.folderRepo.ClearUserFoldersCache(ctx, user.ID); err != nil {
			log.Warn("failed to clear folder cache", "userID", user.ID, "error", err)
		}
	}

	log.Info("cleared caches for all users", "userCount", len(users))
	return nil
}
