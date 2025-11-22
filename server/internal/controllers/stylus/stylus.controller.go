package stylusController

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

type StylusController struct {
	stylusRepo         repositories.StylusRepository
	transactionService *services.TransactionService
	db                 database.DB
	Config             config.Config
	log                logger.Logger
}

type CreateCustomStylusRequest struct {
	Brand                   string `json:"brand"`
	Model                   string `json:"model"`
	Type                    string `json:"type,omitempty"`
	CartridgeType           string `json:"cartridgeType,omitempty"`
	RecommendedReplaceHours *int   `json:"recommendedReplaceHours,omitempty"`
}

type CreateUserStylusRequest struct {
	StylusID     uuid.UUID        `json:"stylusId"`
	PurchaseDate *string          `json:"purchaseDate,omitempty"`
	InstallDate  *string          `json:"installDate,omitempty"`
	HoursUsed    *decimal.Decimal `json:"hoursUsed,omitempty"`
	Notes        *string          `json:"notes,omitempty"`
	IsActive     *bool            `json:"isActive,omitempty"`
	IsPrimary    *bool            `json:"isPrimary,omitempty"`
}

type UpdateUserStylusRequest struct {
	PurchaseDate string          `json:"purchaseDate"`
	InstallDate  string          `json:"installDate"`
	HoursUsed    decimal.Decimal `json:"hoursUsed"`
	Notes        *string         `json:"notes"`
	IsActive     bool            `json:"isActive"`
	IsPrimary    bool            `json:"isPrimary"`
}

type UpdateUserStylusResponse struct {
	Success bool `json:"success"`
}

type StylusControllerInterface interface {
	GetAvailableStyluses(ctx context.Context, user *User) ([]*Stylus, error)
	GetUserStyluses(ctx context.Context, user *User) ([]*UserStylus, error)
	CreateCustomStylus(
		ctx context.Context,
		user *User,
		request *CreateCustomStylusRequest,
	) (*Stylus, error)
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
) StylusControllerInterface {
	return &StylusController{
		stylusRepo:         repos.Stylus,
		transactionService: services.Transaction,
		db:                 db,
		Config:             config,
		log:                logger.New("stylusController"),
	}
}

func parseDate(dateStr string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *StylusController) GetAvailableStyluses(
	ctx context.Context,
	user *User,
) ([]*Stylus, error) {
	styluses, err := c.stylusRepo.GetAllStyluses(ctx, c.db.SQL, &user.ID)
	if err != nil {
		return nil, c.log.Function("GetAvailableStyluses").
			Err("failed to get available styluses", err, "userID", user.ID)
	}

	return styluses, nil
}

func (c *StylusController) GetUserStyluses(
	ctx context.Context,
	user *User,
) ([]*UserStylus, error) {
	log := c.log.Function("GetUserStyluses")

	styluses, err := c.stylusRepo.GetUserStyluses(ctx, c.db.SQL, user.ID)
	if err != nil {
		return nil, log.Err("failed to get user styluses", err, "userID", user.ID)
	}

	usageHours, err := c.stylusRepo.GetStylusUsageHours(ctx, c.db.SQL, user.ID)
	if err != nil {
		log.Warn("failed to get stylus usage hours, using stored values", "userID", user.ID, "error", err)
	} else {
		for _, stylus := range styluses {
			if hours, ok := usageHours[stylus.ID]; ok {
				hoursDecimal := decimal.NewFromFloat(hours)
				stylus.HoursUsed = &hoursDecimal
			}
		}
	}

	return styluses, nil
}

func (c *StylusController) CreateCustomStylus(
	ctx context.Context,
	user *User,
	request *CreateCustomStylusRequest,
) (*Stylus, error) {
	log := c.log.Function("CreateCustomStylus")

	if request.Brand == "" {
		return nil, log.ErrMsg("brand is required")
	}
	if request.Model == "" {
		return nil, log.ErrMsg("model is required")
	}

	stylusType := StylusTypeElliptical
	if request.Type != "" {
		stylusType = StylusType(request.Type)
	}

	var cartridgeType *CartridgeType
	if request.CartridgeType != "" {
		ct := CartridgeType(request.CartridgeType)
		cartridgeType = &ct
	}

	stylus := &Stylus{
		Brand:                   request.Brand,
		Model:                   request.Model,
		Type:                    stylusType,
		CartridgeType:           cartridgeType,
		RecommendedReplaceHours: request.RecommendedReplaceHours,
		UserGeneratedID:         &user.ID,
		IsVerified:              false,
	}

	err := c.stylusRepo.CreateCustomStylus(ctx, c.db.SQL, stylus)
	if err != nil {
		return nil, log.Err("failed to create custom stylus", err, "userID", user.ID)
	}

	return stylus, nil
}

func (c *StylusController) CreateUserStylus(
	ctx context.Context,
	user *User,
	request *CreateUserStylusRequest,
) error {
	log := c.log.Function("CreateUserStylus")

	if request.StylusID == uuid.Nil {
		return log.ErrMsg("stylusId is required")
	}

	var purchaseDate, installDate *time.Time
	var err error

	if request.PurchaseDate != nil && *request.PurchaseDate != "" {
		purchaseDate, err = parseDate(*request.PurchaseDate)
		if err != nil {
			return log.Err("invalid purchase date format", err)
		}
	}

	if request.InstallDate != nil && *request.InstallDate != "" {
		installDate, err = parseDate(*request.InstallDate)
		if err != nil {
			return log.Err("invalid install date format", err)
		}
	}

	userStylus := &UserStylus{
		UserID:       user.ID,
		StylusID:     request.StylusID,
		PurchaseDate: purchaseDate,
		InstallDate:  installDate,
		HoursUsed:    request.HoursUsed,
		Notes:        request.Notes,
		IsActive:     true,
		IsPrimary:    false,
	}

	if request.IsActive != nil {
		userStylus.IsActive = *request.IsActive
	}
	if request.IsPrimary != nil {
		userStylus.IsPrimary = *request.IsPrimary
	}

	err = c.transactionService.Execute(ctx, func(ctx context.Context, tx *gorm.DB) error {
		if userStylus.IsPrimary {
			if err = c.stylusRepo.UnsetAllPrimary(ctx, tx, user.ID); err != nil {
				return err
			}
		}

		return c.stylusRepo.Create(ctx, tx, userStylus)
	})
	if err != nil {
		return log.Err("failed to create user stylus", err, "userID", user.ID)
	}

	log.Info("User stylus created successfully", "userID", user.ID, "stylusID", userStylus.ID)

	return nil
}

func (c *StylusController) UpdateUserStylus(
	ctx context.Context,
	user *User,
	stylusID uuid.UUID,
	request *UpdateUserStylusRequest,
) error {
	log := c.log.Function("UpdateUserStylus")

	var purchaseDate, installDate *time.Time
	var err error
	if request.PurchaseDate != "" {
		purchaseDate, err = parseDate(request.PurchaseDate)
		if err != nil {
			return log.Err("invalid purchase date format", err)
		}
	}

	if request.InstallDate != "" {
		installDate, err = parseDate(request.InstallDate)
		if err != nil {
			return log.Err("invalid install date format", err)
		}
	}

	updatedStylus := &UserStylus{
		UserID:       user.ID,
		PurchaseDate: purchaseDate,
		InstallDate:  installDate,
		HoursUsed:    &request.HoursUsed,
		Notes:        request.Notes,
		IsActive:     request.IsActive,
		IsPrimary:    request.IsPrimary,
	}

	err = c.transactionService.Execute(ctx, func(ctx context.Context, tx *gorm.DB) error {
		if request.IsPrimary {
			if err = c.stylusRepo.UnsetAllPrimary(ctx, tx, user.ID); err != nil {
				return err
			}
		}

		return c.stylusRepo.Update(ctx, tx, user.ID, stylusID, updatedStylus)
	})
	if err != nil {
		return log.Err("failed to update user stylus", err, "id", stylusID, "userID", user.ID)
	}

	log.Info("User stylus updated successfully", "userID", user.ID, "stylusID", stylusID)

	return nil
}

func (c *StylusController) DeleteUserStylus(
	ctx context.Context,
	user *User,
	stylusID uuid.UUID,
) error {
	log := c.log.Function("DeleteUserStylus")

	if err := c.stylusRepo.Delete(ctx, c.db.SQL, user.ID, stylusID); err != nil {
		return log.Err("failed to delete user stylus", err, "userID", user.ID, "stylusID", stylusID)
	}

	log.Info("User stylus deleted successfully", "userID", user.ID, "stylusID", stylusID)

	return nil
}
