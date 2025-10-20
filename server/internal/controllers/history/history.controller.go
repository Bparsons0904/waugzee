package historyController

import (
	"context"
	"errors"
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

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type HistoryController struct {
	historyRepo        repositories.HistoryRepository
	stylusRepo         repositories.StylusRepository
	transactionService *services.TransactionService
	db                 database.DB
	Config             config.Config
	log                logger.Logger
}

type LogPlayRequest struct {
	ReleaseID    int64      `json:"releaseId"`
	UserStylusID *uuid.UUID `json:"userStylusId,omitempty"`
	PlayedAt     string     `json:"playedAt"`
	Notes        string     `json:"notes,omitempty"`
}

type LogCleaningRequest struct {
	ReleaseID   int64  `json:"releaseId"`
	CleanedAt   string `json:"cleanedAt"`
	Notes       string `json:"notes,omitempty"`
	IsDeepClean bool   `json:"isDeepClean,omitempty"`
}

type HistoryControllerInterface interface {
	LogPlay(ctx context.Context, user *User, request *LogPlayRequest) (*PlayHistory, error)
	DeletePlayHistory(ctx context.Context, user *User, playHistoryID uuid.UUID) error
	LogCleaning(
		ctx context.Context,
		user *User,
		request *LogCleaningRequest,
	) (*CleaningHistory, error)
	DeleteCleaningHistory(ctx context.Context, user *User, cleaningHistoryID uuid.UUID) error
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
	db database.DB,
) HistoryControllerInterface {
	return &HistoryController{
		historyRepo:        repos.History,
		stylusRepo:         repos.Stylus,
		transactionService: services.Transaction,
		db:                 db,
		Config:             config,
		log:                logger.New("historyController"),
	}
}

func parseDateTime(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("datetime is required")
	}

	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}, errors.New("invalid datetime format, expected RFC3339")
	}

	return t, nil
}

func (c *HistoryController) LogPlay(
	ctx context.Context,
	user *User,
	request *LogPlayRequest,
) (*PlayHistory, error) {
	log := c.log.Function("LogPlay")

	if request.ReleaseID == 0 {
		return nil, log.ErrorWithType(ErrValidation, "releaseId is required")
	}

	if len(request.Notes) > MaxNotesLength {
		return nil, log.ErrorWithType(
			ErrValidation,
			"notes exceed maximum length",
			"length",
			len(request.Notes),
			"max",
			MaxNotesLength,
		)
	}

	playedAt, err := parseDateTime(request.PlayedAt)
	if err != nil {
		return nil, log.ErrorWithType(ErrValidation, "invalid playedAt", "error", err)
	}

	if playedAt.After(time.Now()) {
		return nil, log.ErrorWithType(ErrValidation, "playedAt cannot be in the future")
	}

	if request.UserStylusID != nil {
		if err := c.stylusRepo.VerifyUserOwnership(ctx, c.db.SQL, *request.UserStylusID, user.ID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, log.ErrorWithType(ErrNotFound, "user stylus not found or not owned by user")
			}
			return nil, log.Error("failed to verify user stylus ownership", "error", err)
		}
	}

	playHistory := &PlayHistory{
		UserID:       user.ID,
		ReleaseID:    request.ReleaseID,
		UserStylusID: request.UserStylusID,
		PlayedAt:     playedAt,
		Notes:        request.Notes,
	}

	if err := c.historyRepo.CreatePlayHistory(ctx, c.db.SQL, playHistory); err != nil {
		return nil, log.Error(
			"failed to create play history",
			"error",
			err,
			"userID",
			user.ID,
			"releaseID",
			request.ReleaseID,
		)
	}

	log.Info(
		"Play history created successfully",
		"userID",
		user.ID,
		"releaseID",
		request.ReleaseID,
		"playHistoryID",
		playHistory.ID,
	)

	return playHistory, nil
}

func (c *HistoryController) DeletePlayHistory(
	ctx context.Context,
	user *User,
	playHistoryID uuid.UUID,
) error {
	log := c.log.Function("DeletePlayHistory")

	if playHistoryID == uuid.Nil {
		return log.ErrorWithType(ErrValidation, "playHistoryId is required")
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
		return nil, log.ErrorWithType(ErrValidation, "releaseId is required")
	}

	if len(request.Notes) > MaxNotesLength {
		return nil, log.ErrorWithType(
			ErrValidation,
			"notes exceed maximum length",
			"length",
			len(request.Notes),
			"max",
			MaxNotesLength,
		)
	}

	cleanedAt, err := parseDateTime(request.CleanedAt)
	if err != nil {
		return nil, log.ErrorWithType(ErrValidation, "invalid cleanedAt", "error", err)
	}

	if cleanedAt.After(time.Now()) {
		return nil, log.ErrorWithType(ErrValidation, "cleanedAt cannot be in the future")
	}

	cleaningHistory := &CleaningHistory{
		UserID:      user.ID,
		ReleaseID:   request.ReleaseID,
		CleanedAt:   cleanedAt,
		Notes:       request.Notes,
		IsDeepClean: request.IsDeepClean,
	}

	if err := c.historyRepo.CreateCleaningHistory(ctx, c.db.SQL, cleaningHistory); err != nil {
		return nil, log.Error(
			"failed to create cleaning history",
			"error",
			err,
			"userID",
			user.ID,
			"releaseID",
			request.ReleaseID,
		)
	}

	log.Info(
		"Cleaning history created successfully",
		"userID",
		user.ID,
		"releaseID",
		request.ReleaseID,
		"cleaningHistoryID",
		cleaningHistory.ID,
	)

	return cleaningHistory, nil
}

func (c *HistoryController) DeleteCleaningHistory(
	ctx context.Context,
	user *User,
	cleaningHistoryID uuid.UUID,
) error {
	log := c.log.Function("DeleteCleaningHistory")

	if cleaningHistoryID == uuid.Nil {
		return log.ErrorWithType(ErrValidation, "cleaningHistoryId is required")
	}

	if err := c.historyRepo.DeleteCleaningHistory(ctx, c.db.SQL, user.ID, cleaningHistoryID); err != nil {
		return err
	}

	log.Info(
		"Cleaning history deleted successfully",
		"userID",
		user.ID,
		"cleaningHistoryID",
		cleaningHistoryID,
	)

	return nil
}
