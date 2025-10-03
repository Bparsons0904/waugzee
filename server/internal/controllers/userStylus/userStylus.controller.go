package userStylusController

import (
	"context"
	"time"
	"waugzee/config"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserStylusController struct {
	userStylusRepo repositories.UserStylusRepository
	db             database.DB
	Config         config.Config
	log            logger.Logger
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

type UserStylusControllerInterface interface {
	GetUserStyluses(ctx context.Context, user *User) ([]*UserStylus, error)
	CreateUserStylus(
		ctx context.Context,
		user *User,
		request *CreateUserStylusRequest,
	) (*UserStylus, error)
	UpdateUserStylus(
		ctx context.Context,
		user *User,
		stylusID uuid.UUID,
		request *UpdateUserStylusRequest,
	) (*UserStylus, error)
	DeleteUserStylus(ctx context.Context, user *User, stylusID uuid.UUID) error
}

func New(
	repos repositories.Repository,
	_ any,
	config config.Config,
	db database.DB,
) UserStylusControllerInterface {
	return &UserStylusController{
		userStylusRepo: repos.UserStylus,
		db:             db,
		Config:         config,
		log:            logger.New("userStylusController"),
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
) (*UserStylus, error) {
	log := c.log.Function("CreateUserStylus")

	if request.StylusID == uuid.Nil {
		return nil, log.ErrMsg("stylusId is required")
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

	if userStylus.IsActive {
		if err := c.userStylusRepo.UnsetAllPrimary(ctx, c.db.SQL, user.ID); err != nil {
			return nil, log.Err("failed to unset all primary styluses", err, "userID", user.ID)
		}
	}

	if err := c.userStylusRepo.Create(ctx, c.db.SQL, userStylus); err != nil {
		return nil, log.Err("failed to create user stylus", err, "userID", user.ID)
	}

	createdStylus, err := c.userStylusRepo.GetByID(ctx, c.db.SQL, user.ID, userStylus.ID)
	if err != nil {
		return nil, log.Err("failed to get created user stylus", err, "id", userStylus.ID)
	}

	log.Info("User stylus created successfully", "userID", user.ID, "stylusID", userStylus.ID)

	return createdStylus, nil
}

func (c *UserStylusController) UpdateUserStylus(
	ctx context.Context,
	user *User,
	stylusID uuid.UUID,
	request *UpdateUserStylusRequest,
) (*UserStylus, error) {
	log := c.log.Function("UpdateUserStylus")

	userStylus, err := c.userStylusRepo.GetByID(ctx, c.db.SQL, user.ID, stylusID)
	if err != nil {
		return nil, log.Err("user stylus not found or not owned by user", err)
	}

	if request.IsActive != nil && *request.IsActive {
		if err = c.userStylusRepo.UnsetAllPrimary(ctx, c.db.SQL, user.ID); err != nil {
			return nil, log.Err("failed to unset all primary styluses", err, "userID", user.ID)
		}
	}

	if request.PurchaseDate != nil {
		userStylus.PurchaseDate = request.PurchaseDate
	}
	if request.InstallDate != nil {
		userStylus.InstallDate = request.InstallDate
	}
	if request.HoursUsed != nil {
		userStylus.HoursUsed = request.HoursUsed
	}
	if request.Notes != nil {
		userStylus.Notes = request.Notes
	}
	if request.IsActive != nil {
		userStylus.IsActive = *request.IsActive
	}

	if err = c.userStylusRepo.Update(ctx, c.db.SQL, userStylus); err != nil {
		return nil, log.Err("failed to update user stylus", err, "id", stylusID)
	}

	updatedStylus, err := c.userStylusRepo.GetByID(ctx, c.db.SQL, user.ID, stylusID)
	if err != nil {
		return nil, log.Err("failed to get updated user stylus", err, "id", stylusID)
	}

	log.Info("User stylus updated successfully", "userID", user.ID, "stylusID", stylusID)

	return updatedStylus, nil
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
