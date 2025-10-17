package historyController

import (
	"context"
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

type HistoryController struct {
	historyRepo         repositories.HistoryRepository
	transactionService  *services.TransactionService
	db                  database.DB
	Config              config.Config
	log                 logger.Logger
}

type LogPlayRequest struct {
	ReleaseID    int64      `json:"releaseId"                validate:"required"`
	UserStylusID *uuid.UUID `json:"userStylusId,omitempty"`
	PlayedAt     *string    `json:"playedAt,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
}

type LogCleaningRequest struct {
	ReleaseID   int64   `json:"releaseId"              validate:"required"`
	CleanedAt   *string `json:"cleanedAt,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	IsDeepClean *bool   `json:"isDeepClean,omitempty"`
}

type HistoryControllerInterface interface {
	// Play History
	LogPlay(ctx context.Context, user *User, request *LogPlayRequest) (*PlayHistory, error)
	DeletePlayHistory(ctx context.Context, user *User, playHistoryID uuid.UUID) error
	
	// Cleaning History
	LogCleaning(ctx context.Context, user *User, request *LogCleaningRequest) (*CleaningHistory, error)
	DeleteCleaningHistory(ctx context.Context, user *User, cleaningHistoryID uuid.UUID) error
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
	db database.DB,
) HistoryControllerInterface {
	return &HistoryController{
		historyRepo:         repos.History,
		transactionService:  services.Transaction,
		db:                  db,
		Config:              config,
		log:                 logger.New("historyController"),
	}
}

func parseDateTime(dateStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}
	
	// Try ISO 8601 format first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return &t, nil
	}
	
	// Try date-only format
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return &t, nil
	}
	
	// Try datetime format
	if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
		return &t, nil
	}
	
	return nil, logger.New("parseDateTime").Error("invalid date format", "dateStr", dateStr)
}

func (c *HistoryController) LogPlay(
	ctx context.Context,
	user *User,
	request *LogPlayRequest,
) (*PlayHistory, error) {
	log := c.log.Function("LogPlay")

	if request.ReleaseID == 0 {
		return nil, log.ErrMsg("releaseId is required")
	}

	var playedAt time.Time

	if request.PlayedAt != nil && *request.PlayedAt != "" {
		parsed, err := parseDateTime(*request.PlayedAt)
		if err != nil {
			return nil, log.Err("invalid playedAt format", err)
		}
		if parsed != nil {
			playedAt = *parsed
		}
	} else {
		playedAt = time.Now()
	}

	// Verify user owns the stylus if provided
	if request.UserStylusID != nil {
		var userStylus UserStylus
		if err := c.db.SQL.WithContext(ctx).
			Where("id = ? AND user_id = ?", *request.UserStylusID, user.ID).
			First(&userStylus).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, log.ErrMsg("user stylus not found or not owned by user")
			}
			return nil, log.Err("failed to verify user stylus ownership", err)
		}
	}

	playHistory := &PlayHistory{
		UserID:       user.ID,
		ReleaseID:    request.ReleaseID,
		UserStylusID: request.UserStylusID,
		PlayedAt:     playedAt,
		Notes:        "",
	}

	if request.Notes != nil {
		playHistory.Notes = *request.Notes
	}

	if err := c.historyRepo.CreatePlayHistory(ctx, c.db.SQL, playHistory); err != nil {
		return nil, log.Err("failed to create play history", err, "userID", user.ID, "releaseID", request.ReleaseID)
	}

	log.Info("Play history created successfully", "userID", user.ID, "releaseID", request.ReleaseID, "playHistoryID", playHistory.ID)

	return playHistory, nil
}

func (c *HistoryController) DeletePlayHistory(
	ctx context.Context,
	user *User,
	playHistoryID uuid.UUID,
) error {
	log := c.log.Function("DeletePlayHistory")

	if playHistoryID == uuid.Nil {
		return log.ErrMsg("playHistoryId is required")
	}

	if err := c.historyRepo.DeletePlayHistory(ctx, c.db.SQL, user.ID, playHistoryID); err != nil {
		return log.Err("failed to delete play history", err, "userID", user.ID, "playHistoryID", playHistoryID)
	}

	log.Info("Play history deleted successfully", "userID", user.ID, "playHistoryID", playHistoryID)

	return nil
}

func (c *HistoryController) LogCleaning(
	ctx context.Context,
	user *User,
	request *LogCleaningRequest,
) (*CleaningHistory, error) {
	log := c.log.Function("LogCleaning")

	if request.ReleaseID == 0 {
		return nil, log.ErrMsg("releaseId is required")
	}

	var cleanedAt time.Time

	if request.CleanedAt != nil && *request.CleanedAt != "" {
		parsed, err := parseDateTime(*request.CleanedAt)
		if err != nil {
			return nil, log.Err("invalid cleanedAt format", err)
		}
		if parsed != nil {
			cleanedAt = *parsed
		}
	} else {
		cleanedAt = time.Now()
	}

	cleaningHistory := &CleaningHistory{
		UserID:      user.ID,
		ReleaseID:   request.ReleaseID,
		CleanedAt:   cleanedAt,
		Notes:       "",
		IsDeepClean: false,
	}

	if request.Notes != nil {
		cleaningHistory.Notes = *request.Notes
	}

	if request.IsDeepClean != nil {
		cleaningHistory.IsDeepClean = *request.IsDeepClean
	}

	if err := c.historyRepo.CreateCleaningHistory(ctx, c.db.SQL, cleaningHistory); err != nil {
		return nil, log.Err("failed to create cleaning history", err, "userID", user.ID, "releaseID", request.ReleaseID)
	}

	log.Info("Cleaning history created successfully", "userID", user.ID, "releaseID", request.ReleaseID, "cleaningHistoryID", cleaningHistory.ID)

	return cleaningHistory, nil
}

func (c *HistoryController) DeleteCleaningHistory(
	ctx context.Context,
	user *User,
	cleaningHistoryID uuid.UUID,
) error {
	log := c.log.Function("DeleteCleaningHistory")

	if cleaningHistoryID == uuid.Nil {
		return log.ErrMsg("cleaningHistoryId is required")
	}

	if err := c.historyRepo.DeleteCleaningHistory(ctx, c.db.SQL, user.ID, cleaningHistoryID); err != nil {
		return log.Err("failed to delete cleaning history", err, "userID", user.ID, "cleaningHistoryID", cleaningHistoryID)
	}

	log.Info("Cleaning history deleted successfully", "userID", user.ID, "cleaningHistoryID", cleaningHistoryID)

	return nil
}