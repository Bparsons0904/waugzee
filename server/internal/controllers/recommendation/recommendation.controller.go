package recommendationController

import (
	"context"
	"errors"
	"time"
	"waugzee/internal/database"
	logger "github.com/Bparsons0904/goLogger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
	"waugzee/internal/services"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("recommendation not found")

type RecommendationController struct {
	recommendationRepo repositories.DailyRecommendationRepository
	historyRepo        repositories.HistoryRepository
	stylusRepo         repositories.StylusRepository
	userReleaseRepo    repositories.UserReleaseRepository
	transactionService *services.TransactionService
	cache              database.CacheClient
	db                 *gorm.DB
}

type RecommendationControllerInterface interface {
	MarkAsListened(
		ctx context.Context,
		user *User,
		recommendationID uuid.UUID,
	) error
}

func New(
	repos repositories.Repository,
	services *services.Service,
	db *gorm.DB,
	cache database.CacheClient,
) RecommendationControllerInterface {
	return &RecommendationController{
		recommendationRepo: repos.DailyRecommendation,
		historyRepo:        repos.History,
		stylusRepo:         repos.Stylus,
		userReleaseRepo:    repos.UserRelease,
		transactionService: services.Transaction,
		cache:              cache,
		db:                 db,
	}
}

func (c *RecommendationController) MarkAsListened(
	ctx context.Context,
	user *User,
	recommendationID uuid.UUID,
) error {
	log := logger.New("recommendationController").TraceFromContext(ctx).Function("MarkAsListened")

	recommendation, err := c.recommendationRepo.GetByID(ctx, c.db, recommendationID, user.ID)
	if err != nil {
		return log.Err("failed to get recommendation", err, "recommendationID", recommendationID)
	}

	var userStylusID *uuid.UUID
	primaryStylus, err := c.stylusRepo.GetPrimaryUserStylus(ctx, c.db, user.ID)
	if err != nil {
		log.Warn("failed to get primary stylus", "userID", user.ID, "error", err)
	}

	if primaryStylus != nil {
		userStylusID = &primaryStylus.ID
	}

	playHistory := &PlayHistory{
		UserID:        user.ID,
		UserReleaseID: recommendation.UserReleaseID,
		UserStylusID:  userStylusID,
		PlayedAt:      time.Now(),
		Notes:         "",
	}

	err = c.transactionService.Execute(ctx, func(txCtx context.Context, tx *gorm.DB) error {
		if err := c.historyRepo.CreatePlayHistory(txCtx, tx, playHistory); err != nil {
			return log.Err(
				"failed to create play history",
				err,
				"userReleaseID",
				recommendation.UserReleaseID,
			)
		}

		if err := c.recommendationRepo.MarkAsListened(txCtx, tx, recommendationID, user.ID); err != nil {
			return log.Err("failed to mark as listened", err, "recommendationID", recommendationID)
		}

		return nil
	})

	if err != nil {
		return err
	}

	if err = c.recommendationRepo.ClearUserStreakCache(ctx, user.ID); err != nil {
		log.Warn("failed to invalidate user streak cache", "userID", user.ID, "error", err)
	}

	log.Info(
		"marked recommendation as listened and created play history",
		"recommendationID",
		recommendationID,
		"userID",
		user.ID,
		"playHistoryID",
		playHistory.ID,
	)
	return nil
}
