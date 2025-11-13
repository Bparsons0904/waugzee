package recommendationController

import (
	"context"
	"errors"
	"time"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("recommendation not found")

type RecommendationController struct {
	recommendationRepo repositories.DailyRecommendationRepository
	historyRepo        repositories.HistoryRepository
	stylusRepo         repositories.StylusRepository
	userReleaseRepo    repositories.UserReleaseRepository
	db                 *gorm.DB
	log                logger.Logger
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
	db *gorm.DB,
) RecommendationControllerInterface {
	return &RecommendationController{
		recommendationRepo: repos.DailyRecommendation,
		historyRepo:        repos.History,
		stylusRepo:         repos.Stylus,
		userReleaseRepo:    repos.UserRelease,
		db:                 db,
		log:                logger.New("recommendationController"),
	}
}

func (c *RecommendationController) MarkAsListened(
	ctx context.Context,
	user *User,
	recommendationID uuid.UUID,
) error {
	log := c.log.Function("MarkAsListened")

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

	if err := c.historyRepo.CreatePlayHistory(ctx, c.db, playHistory); err != nil {
		return log.Err("failed to create play history", err, "userReleaseID", recommendation.UserReleaseID)
	}

	if err := c.recommendationRepo.MarkAsListened(ctx, c.db, recommendationID, user.ID); err != nil {
		return log.Err("failed to mark as listened", err, "recommendationID", recommendationID)
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
