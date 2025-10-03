package userStylusController

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
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type UserStylusController struct {
	userStylusRepo     repositories.UserStylusRepository
	transactionService *services.TransactionService
	db                 database.DB
	Config             config.Config
	log                logger.Logger
}

type CreateUserStylusRequest struct {
	StylusID     uuid.UUID        `json:"stylusId"               validate:"required"`
	PurchaseDate *time.Time       `json:"purchaseDate,omitempty"`
	InstallDate  *time.Time       `json:"installDate,omitempty"`
	HoursUsed    *decimal.Decimal `json:"hoursUsed,omitempty"`
	Notes        *string          `json:"notes,omitempty"`
	IsActive     *bool            `json:"isActive,omitempty"`
}

type UpdateUserStylusRequest struct {
	PurchaseDate *time.Time       `json:"purchaseDate,omitempty"`
	InstallDate  *time.Time       `json:"installDate,omitempty"`
	HoursUsed    *decimal.Decimal `json:"hoursUsed,omitempty"`
	Notes        *string          `json:"notes,omitempty"`
	IsActive     *bool            `json:"isActive,omitempty"`
}

type UpdateUserStylusResponse struct {
	Success bool `json:"success"`
}

type UserStylusControllerInterface interface {
	GetUserStyluses(ctx context.Context, user *User) ([]*UserStylus, error)
	CreateUserStylus(
		ctx context.Context,
		user *User,
		request *CreateUserStylusRequest,
	) error
	UpdateUserStylus(
		ctx context.Context,
		user *User,
		stylusID uuid.UUID,
		request *UpdateUserStylusRequest,
	) error
	DeleteUserStylus(ctx context.Context, user *User, stylusID uuid.UUID) error
}

func New(
	repos repositories.Repository,
	services services.Service,
	config config.Config,
	db database.DB,
) UserStylusControllerInterface {
	return &UserStylusController{
		userStylusRepo:     repos.UserStylus,
		transactionService: services.Transaction,
		db:                 db,
		Config:             config,
		log:                logger.New("userStylusController"),
	}
}

func (c *UserStylusController) GetUserStyluses(
	ctx context.Context,
	user *User,
) ([]*UserStylus, error) {
	log := c.log.Function("GetUserStyluses")

	styluses, err := c.userStylusRepo.GetUserStyluses(ctx, c.db.SQL, user.ID)
	if err != nil {
		return nil, log.Err("failed to get user styluses", err, "userID", user.ID)
	}

	return styluses, nil
}

func (c *UserStylusController) CreateUserStylus(
	ctx context.Context,
	user *User,
	request *CreateUserStylusRequest,
) error {
	log := c.log.Function("CreateUserStylus")

	if request.StylusID == uuid.Nil {
		return log.ErrMsg("stylusId is required")
	}

	userStylus := &UserStylus{
		UserID:       user.ID,
		StylusID:     request.StylusID,
		PurchaseDate: request.PurchaseDate,
		InstallDate:  request.InstallDate,
		HoursUsed:    request.HoursUsed,
		Notes:        request.Notes,
		IsActive:     true,
	}

	if request.IsActive != nil {
		userStylus.IsActive = *request.IsActive
	}

	err := c.transactionService.Execute(ctx, func(ctx context.Context, tx *gorm.DB) error {
		if userStylus.IsActive {
			if err := c.userStylusRepo.UnsetAllPrimary(ctx, tx, user.ID); err != nil {
				return err
			}
		}

		return c.userStylusRepo.Create(ctx, tx, userStylus)
	})

	if err != nil {
		return log.Err("failed to create user stylus", err, "userID", user.ID)
	}

	log.Info("User stylus created successfully", "userID", user.ID, "stylusID", userStylus.ID)

	return nil
}

func (c *UserStylusController) UpdateUserStylus(
	ctx context.Context,
	user *User,
	stylusID uuid.UUID,
	request *UpdateUserStylusRequest,
) error {
	log := c.log.Function("UpdateUserStylus")

	updates := make(map[string]interface{})

	if request.PurchaseDate != nil {
		updates["purchase_date"] = request.PurchaseDate
	}
	if request.InstallDate != nil {
		updates["install_date"] = request.InstallDate
	}
	if request.HoursUsed != nil {
		updates["hours_used"] = request.HoursUsed
	}
	if request.Notes != nil {
		updates["notes"] = request.Notes
	}
	if request.IsActive != nil {
		updates["is_active"] = *request.IsActive
	}

	if len(updates) == 0 {
		return log.ErrMsg("no fields to update")
	}

	err := c.transactionService.Execute(ctx, func(ctx context.Context, tx *gorm.DB) error {
		if request.IsActive != nil && *request.IsActive {
			if err := c.userStylusRepo.UnsetAllPrimary(ctx, tx, user.ID); err != nil {
				return err
			}
		}

		return c.userStylusRepo.Update(ctx, tx, user.ID, stylusID, updates)
	})

	if err != nil {
		return log.Err("failed to update user stylus", err, "id", stylusID, "userID", user.ID)
	}

	log.Info("User stylus updated successfully", "userID", user.ID, "stylusID", stylusID)

	return nil
}

func (c *UserStylusController) DeleteUserStylus(
	ctx context.Context,
	user *User,
	stylusID uuid.UUID,
) error {
	log := c.log.Function("DeleteUserStylus")

	if err := c.userStylusRepo.Delete(ctx, c.db.SQL, user.ID, stylusID); err != nil {
		return log.Err("failed to delete user stylus", err, "userID", user.ID, "stylusID", stylusID)
	}

	log.Info("User stylus deleted successfully", "userID", user.ID, "stylusID", stylusID)

	return nil
}
