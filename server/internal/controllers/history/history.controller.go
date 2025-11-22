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
}

type LogPlayRequest struct {
	UserReleaseID uuid.UUID  `json:"userReleaseId"`
	UserStylusID  *uuid.UUID `json:"userStylusId,omitempty"`
	PlayedAt      string     `json:"playedAt"`
	Notes         string     `json:"notes,omitempty"`
}

type LogCleaningRequest struct {
	UserReleaseID uuid.UUID `json:"userReleaseId"`
	CleanedAt     string    `json:"cleanedAt"`
	Notes         string    `json:"notes,omitempty"`
	IsDeepClean   bool      `json:"isDeepClean,omitempty"`
}

type LogBothRequest struct {
	UserReleaseID uuid.UUID  `json:"userReleaseId"`
	UserStylusID  *uuid.UUID `json:"userStylusId,omitempty"`
	Timestamp     string     `json:"timestamp"`
	Notes         string     `json:"notes,omitempty"`
	IsDeepClean   bool       `json:"isDeepClean,omitempty"`
}

type LogBothResponse struct {
	PlayHistory     *PlayHistory     `json:"playHistory"`
	CleaningHistory *CleaningHistory `json:"cleaningHistory"`
}

type UpdatePlayHistoryRequest struct {
	PlayedAt     *string    `json:"playedAt,omitempty"`
	UserStylusID *uuid.UUID `json:"userStylusId,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
}

type UpdateCleaningHistoryRequest struct {
	CleanedAt   *string `json:"cleanedAt,omitempty"`
	IsDeepClean *bool   `json:"isDeepClean,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type HistoryControllerInterface interface {
	LogPlay(ctx context.Context, user *User, request *LogPlayRequest) (*PlayHistory, error)
	UpdatePlayHistory(
		ctx context.Context,
		user *User,
		playHistoryID uuid.UUID,
		request *UpdatePlayHistoryRequest,
	) (*PlayHistory, error)
	DeletePlayHistory(ctx context.Context, user *User, playHistoryID uuid.UUID) error
	LogCleaning(
		ctx context.Context,
		user *User,
		request *LogCleaningRequest,
	) (*CleaningHistory, error)
	UpdateCleaningHistory(
		ctx context.Context,
		user *User,
		cleaningHistoryID uuid.UUID,
		request *UpdateCleaningHistoryRequest,
	) (*CleaningHistory, error)
	DeleteCleaningHistory(ctx context.Context, user *User, cleaningHistoryID uuid.UUID) error
	LogBoth(ctx context.Context, user *User, request *LogBothRequest) (*LogBothResponse, error)
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
	log := logger.NewWithContext(ctx, "historyController").Function("LogPlay")

	if request.UserReleaseID == uuid.Nil {
		return nil, log.ErrorWithType(ErrValidation, "userReleaseId is required")
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
				return nil, log.ErrorWithType(
					ErrNotFound,
					"user stylus not found or not owned by user",
				)
			}
			return nil, log.Error("failed to verify user stylus ownership", "error", err)
		}
	}

	playHistory := &PlayHistory{
		UserID:        user.ID,
		UserReleaseID: request.UserReleaseID,
		UserStylusID:  request.UserStylusID,
		PlayedAt:      playedAt,
		Notes:         request.Notes,
	}

	if err := c.historyRepo.CreatePlayHistory(ctx, c.db.SQL, playHistory); err != nil {
		return nil, log.Error(
			"failed to create play history",
			"error",
			err,
			"userID",
			user.ID,
			"userReleaseID",
			request.UserReleaseID,
		)
	}

	log.Info(
		"Play history created successfully",
		"userID",
		user.ID,
		"userReleaseID",
		request.UserReleaseID,
		"playHistoryID",
		playHistory.ID,
	)

	return playHistory, nil
}

func (c *HistoryController) UpdatePlayHistory(
	ctx context.Context,
	user *User,
	playHistoryID uuid.UUID,
	request *UpdatePlayHistoryRequest,
) (*PlayHistory, error) {
	log := logger.NewWithContext(ctx, "historyController").Function("UpdatePlayHistory")

	if playHistoryID == uuid.Nil {
		return nil, log.ErrorWithType(ErrValidation, "playHistoryId is required")
	}

	var existingPlayHistory PlayHistory
	err := c.db.SQL.Where("id = ?", playHistoryID).First(&existingPlayHistory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, log.ErrorWithType(ErrNotFound, "play history not found")
		}
		return nil, log.Error("failed to retrieve play history", "error", err)
	}

	if existingPlayHistory.UserID != user.ID {
		return nil, log.ErrorWithType(ErrValidation, "play history does not belong to user")
	}

	updates := make(map[string]any)

	if request.PlayedAt != nil {
		var playedAt time.Time
		playedAt, err = parseDateTime(*request.PlayedAt)
		if err != nil {
			return nil, log.ErrorWithType(ErrValidation, "invalid playedAt", "error", err)
		}

		if playedAt.After(time.Now()) {
			return nil, log.ErrorWithType(ErrValidation, "playedAt cannot be in the future")
		}

		updates["played_at"] = playedAt
	}

	if request.UserStylusID != nil {
		if err = c.stylusRepo.VerifyUserOwnership(ctx, c.db.SQL, *request.UserStylusID, user.ID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, log.ErrorWithType(
					ErrNotFound,
					"user stylus not found or not owned by user",
				)
			}
			return nil, log.Error("failed to verify user stylus ownership", "error", err)
		}
		updates["user_stylus_id"] = request.UserStylusID
	}

	if request.Notes != nil {
		if len(*request.Notes) > MaxNotesLength {
			return nil, log.ErrorWithType(
				ErrValidation,
				"notes exceed maximum length",
				"length",
				len(*request.Notes),
				"max",
				MaxNotesLength,
			)
		}
		updates["notes"] = *request.Notes
	}

	if len(updates) == 0 {
		return nil, log.ErrorWithType(ErrValidation, "no fields to update")
	}

	playHistory, err := c.historyRepo.UpdatePlayHistory(ctx, c.db.SQL, playHistoryID, updates)
	if err != nil {
		return nil, log.Error(
			"failed to update play history",
			"error",
			err,
			"playHistoryID",
			playHistoryID,
		)
	}

	log.Info(
		"Play history updated successfully",
		"userID",
		user.ID,
		"playHistoryID",
		playHistoryID,
	)

	return playHistory, nil
}

func (c *HistoryController) DeletePlayHistory(
	ctx context.Context,
	user *User,
	playHistoryID uuid.UUID,
) error {
	log := logger.NewWithContext(ctx, "historyController").Function("DeletePlayHistory")

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
	log := logger.NewWithContext(ctx, "historyController").Function("LogCleaning")

	if request.UserReleaseID == uuid.Nil {
		return nil, log.ErrorWithType(ErrValidation, "userReleaseId is required")
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
		UserID:        user.ID,
		UserReleaseID: request.UserReleaseID,
		CleanedAt:     cleanedAt,
		Notes:         request.Notes,
		IsDeepClean:   request.IsDeepClean,
	}

	if err := c.historyRepo.CreateCleaningHistory(ctx, c.db.SQL, cleaningHistory); err != nil {
		return nil, log.Error(
			"failed to create cleaning history",
			"error",
			err,
			"userID",
			user.ID,
			"userReleaseID",
			request.UserReleaseID,
		)
	}

	log.Info(
		"Cleaning history created successfully",
		"userID",
		user.ID,
		"userReleaseID",
		request.UserReleaseID,
		"cleaningHistoryID",
		cleaningHistory.ID,
	)

	return cleaningHistory, nil
}

func (c *HistoryController) UpdateCleaningHistory(
	ctx context.Context,
	user *User,
	cleaningHistoryID uuid.UUID,
	request *UpdateCleaningHistoryRequest,
) (*CleaningHistory, error) {
	log := logger.NewWithContext(ctx, "historyController").Function("UpdateCleaningHistory")

	if cleaningHistoryID == uuid.Nil {
		return nil, log.ErrorWithType(ErrValidation, "cleaningHistoryId is required")
	}

	var existingCleaningHistory CleaningHistory
	err := c.db.SQL.Where("id = ?", cleaningHistoryID).First(&existingCleaningHistory).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, log.ErrorWithType(ErrNotFound, "cleaning history not found")
		}
		return nil, log.Error("failed to retrieve cleaning history", "error", err)
	}

	if existingCleaningHistory.UserID != user.ID {
		return nil, log.ErrorWithType(ErrValidation, "cleaning history does not belong to user")
	}

	updates := make(map[string]any)

	if request.CleanedAt != nil {
		var cleanedAt time.Time
		cleanedAt, err = parseDateTime(*request.CleanedAt)
		if err != nil {
			return nil, log.ErrorWithType(ErrValidation, "invalid cleanedAt", "error", err)
		}

		if cleanedAt.After(time.Now()) {
			return nil, log.ErrorWithType(ErrValidation, "cleanedAt cannot be in the future")
		}

		updates["cleaned_at"] = cleanedAt
	}

	if request.IsDeepClean != nil {
		updates["is_deep_clean"] = *request.IsDeepClean
	}

	if request.Notes != nil {
		if len(*request.Notes) > MaxNotesLength {
			return nil, log.ErrorWithType(
				ErrValidation,
				"notes exceed maximum length",
				"length",
				len(*request.Notes),
				"max",
				MaxNotesLength,
			)
		}
		updates["notes"] = *request.Notes
	}

	if len(updates) == 0 {
		return nil, log.ErrorWithType(ErrValidation, "no fields to update")
	}

	cleaningHistory, err := c.historyRepo.UpdateCleaningHistory(
		ctx,
		c.db.SQL,
		cleaningHistoryID,
		updates,
	)
	if err != nil {
		return nil, log.Error(
			"failed to update cleaning history",
			"error",
			err,
			"cleaningHistoryID",
			cleaningHistoryID,
		)
	}

	log.Info(
		"Cleaning history updated successfully",
		"userID",
		user.ID,
		"cleaningHistoryID",
		cleaningHistoryID,
	)

	return cleaningHistory, nil
}

func (c *HistoryController) DeleteCleaningHistory(
	ctx context.Context,
	user *User,
	cleaningHistoryID uuid.UUID,
) error {
	log := logger.NewWithContext(ctx, "historyController").Function("DeleteCleaningHistory")

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

func (c *HistoryController) LogBoth(
	ctx context.Context,
	user *User,
	request *LogBothRequest,
) (*LogBothResponse, error) {
	log := logger.NewWithContext(ctx, "historyController").Function("LogBoth")

	if request.UserReleaseID == uuid.Nil {
		return nil, log.ErrorWithType(ErrValidation, "userReleaseId is required")
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

	timestamp, err := parseDateTime(request.Timestamp)
	if err != nil {
		return nil, log.ErrorWithType(ErrValidation, "invalid timestamp", "error", err)
	}

	if timestamp.After(time.Now()) {
		return nil, log.ErrorWithType(ErrValidation, "timestamp cannot be in the future")
	}

	if request.UserStylusID != nil {
		if err = c.stylusRepo.VerifyUserOwnership(ctx, c.db.SQL, *request.UserStylusID, user.ID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, log.ErrorWithType(
					ErrNotFound,
					"user stylus not found or not owned by user",
				)
			}
			return nil, log.Error("failed to verify user stylus ownership", "error", err)
		}
	}

	var playHistory *PlayHistory
	var cleaningHistory *CleaningHistory

	err = c.transactionService.Execute(ctx, func(ctx context.Context, tx *gorm.DB) error {
		playHistory = &PlayHistory{
			UserID:        user.ID,
			UserReleaseID: request.UserReleaseID,
			UserStylusID:  request.UserStylusID,
			PlayedAt:      timestamp,
			Notes:         request.Notes,
		}

		if err = c.historyRepo.CreatePlayHistory(ctx, tx, playHistory); err != nil {
			return log.Error(
				"failed to create play history in transaction",
				"error",
				err,
				"userID",
				user.ID,
				"userReleaseID",
				request.UserReleaseID,
			)
		}

		cleaningHistory = &CleaningHistory{
			UserID:        user.ID,
			UserReleaseID: request.UserReleaseID,
			CleanedAt:     timestamp,
			Notes:         request.Notes,
			IsDeepClean:   request.IsDeepClean,
		}

		if err = c.historyRepo.CreateCleaningHistory(ctx, tx, cleaningHistory); err != nil {
			return log.Error(
				"failed to create cleaning history in transaction",
				"error",
				err,
				"userID",
				user.ID,
				"userReleaseID",
				request.UserReleaseID,
			)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Info(
		"Play and cleaning history created successfully",
		"userID",
		user.ID,
		"userReleaseID",
		request.UserReleaseID,
		"playHistoryID",
		playHistory.ID,
		"cleaningHistoryID",
		cleaningHistory.ID,
	)

	return &LogBothResponse{
		PlayHistory:     playHistory,
		CleaningHistory: cleaningHistory,
	}, nil
}
