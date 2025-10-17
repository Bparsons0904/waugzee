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

const (
	MaxNotesLength = 1000
)

type HistoryController struct {
	historyRepo         repositories.HistoryRepository
	stylusRepo          repositories.StylusRepository
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
		stylusRepo:          repos.Stylus,
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

	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, nil
}

func (c *HistoryController) LogPlay(
	ctx context.Context,
	user *User,
	request *LogPlayRequest,
) (*PlayHistory, error) {
	log := c.log.Function("LogPlay")

	if request.ReleaseID == 0 {
		return nil, log.Error("releaseId is required")
	}

	if request.Notes != nil && len(*request.Notes) > MaxNotesLength {
		return nil, log.Error("notes exceed maximum length", "length", len(*request.Notes), "max", MaxNotesLength)
	}

	var playedAt time.Time

	if request.PlayedAt != nil && *request.PlayedAt != "" {
		parsed, err := parseDateTime(*request.PlayedAt)
		if err != nil {
			return nil, err
		}
		if parsed == nil {
			return nil, log.Error("invalid date format", "playedAt", *request.PlayedAt)
		}
		playedAt = *parsed
	} else {
		playedAt = time.Now()
	}

	if playedAt.After(time.Now()) {
		return nil, log.Error("playedAt cannot be in the future")
	}

	if request.UserStylusID != nil {
		if err := c.stylusRepo.VerifyUserOwnership(ctx, c.db.SQL, *request.UserStylusID, user.ID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, log.Error("user stylus not found or not owned by user")
			}
			return nil, log.Error("failed to verify user stylus ownership", "error", err)
		}
	}

	playHistory := &PlayHistory{
		UserID:       user.ID,
		ReleaseID:    request.ReleaseID,
		UserStylusID: request.UserStylusID,
		PlayedAt:     playedAt,
	}

	if request.Notes != nil {
		playHistory.Notes = *request.Notes
	}

	if err := c.historyRepo.CreatePlayHistory(ctx, c.db.SQL, playHistory); err != nil {
		return nil, log.Error("failed to create play history", "error", err, "userID", user.ID, "releaseID", request.ReleaseID)
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
		return log.Error("playHistoryId is required")
	}

	if err := c.historyRepo.DeletePlayHistory(ctx, c.db.SQL, user.ID, playHistoryID); err != nil {
		return err
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
		return nil, log.Error("releaseId is required")
	}

	if request.Notes != nil && len(*request.Notes) > MaxNotesLength {
		return nil, log.Error("notes exceed maximum length", "length", len(*request.Notes), "max", MaxNotesLength)
	}

	var cleanedAt time.Time

	if request.CleanedAt != nil && *request.CleanedAt != "" {
		parsed, err := parseDateTime(*request.CleanedAt)
		if err != nil {
			return nil, err
		}
		if parsed == nil {
			return nil, log.Error("invalid date format", "cleanedAt", *request.CleanedAt)
		}
		cleanedAt = *parsed
	} else {
		cleanedAt = time.Now()
	}

	if cleanedAt.After(time.Now()) {
		return nil, log.Error("cleanedAt cannot be in the future")
	}

	cleaningHistory := &CleaningHistory{
		UserID:    user.ID,
		ReleaseID: request.ReleaseID,
		CleanedAt: cleanedAt,
	}

	if request.Notes != nil {
		cleaningHistory.Notes = *request.Notes
	}

	if request.IsDeepClean != nil {
		cleaningHistory.IsDeepClean = *request.IsDeepClean
	}

	if err := c.historyRepo.CreateCleaningHistory(ctx, c.db.SQL, cleaningHistory); err != nil {
		return nil, log.Error("failed to create cleaning history", "error", err, "userID", user.ID, "releaseID", request.ReleaseID)
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
		return log.Error("cleaningHistoryId is required")
	}

	if err := c.historyRepo.DeleteCleaningHistory(ctx, c.db.SQL, user.ID, cleaningHistoryID); err != nil {
		return err
	}

	log.Info("Cleaning history deleted successfully", "userID", user.ID, "cleaningHistoryID", cleaningHistoryID)

	return nil
}