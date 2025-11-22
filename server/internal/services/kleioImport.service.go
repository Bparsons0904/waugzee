package services

// TODO: REMOVE_AFTER_MIGRATION
// This entire file is for one-time Kleio data migration and should be deleted after import is complete.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TODO: REMOVE_AFTER_MIGRATION
type KleioImportService struct {
	db                  database.DB
	stylusRepo          repositories.StylusRepository
	userReleaseRepo     repositories.UserReleaseRepository
	historyRepo         repositories.HistoryRepository
	transactionService  *TransactionService
	log                 logger.Logger
}

func NewKleioImportService(
	db database.DB,
	stylusRepo repositories.StylusRepository,
	userReleaseRepo repositories.UserReleaseRepository,
	historyRepo repositories.HistoryRepository,
	transactionService *TransactionService,
) *KleioImportService {
	return &KleioImportService{
		db:                 db,
		stylusRepo:         stylusRepo,
		userReleaseRepo:    userReleaseRepo,
		historyRepo:        historyRepo,
		transactionService: transactionService,
		log:                logger.New("kleioImportService"),
	}
}

type KleioExport struct {
	ExportDate      string                 `json:"exportDate"`
	PlayHistory     []KleioPlayHistory     `json:"playHistory"`
	CleaningHistory []KleioCleaningHistory `json:"cleaningHistory"`
	Styluses        []KleioStylus          `json:"styluses"`
}

type KleioPlayHistory struct {
	ReleaseID int        `json:"releaseId"`
	StylusID  *int       `json:"stylusId"`
	PlayedAt  time.Time  `json:"playedAt"`
	Notes     string     `json:"notes"`
}

type KleioCleaningHistory struct {
	ReleaseID int       `json:"releaseId"`
	CleanedAt time.Time `json:"cleanedAt"`
	Notes     string    `json:"notes"`
}

type KleioStylus struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	Manufacturer     string     `json:"manufacturer"`
	ExpectedLifespan int        `json:"expected_lifespan_hours"`
	PurchaseDate     *time.Time `json:"purchase_date"`
	Active           bool       `json:"active"`
	Primary          bool       `json:"primary_stylus"`
}

type ImportSummary struct {
	StylusesCreated     int      `json:"stylusesCreated"`
	PlayHistoryImported int      `json:"playHistoryImported"`
	CleaningImported    int      `json:"cleaningImported"`
	DeepCleansDetected  int      `json:"deepCleansDetected"`
	FailedMappings      []string `json:"failedMappings"`
}

func (s *KleioImportService) validateKleioExport(export *KleioExport) error {
	if export.ExportDate == "" {
		return fmt.Errorf("missing export date")
	}

	if _, err := time.Parse(time.RFC3339Nano, export.ExportDate); err != nil {
		if _, err := time.Parse(time.RFC3339, export.ExportDate); err != nil {
			if _, err := time.Parse("2006-01-02", export.ExportDate); err != nil {
				return fmt.Errorf("invalid export date format: %w", err)
			}
		}
	}

	const maxNoteLength = 1000
	now := time.Now()

	for i, play := range export.PlayHistory {
		if play.ReleaseID <= 0 {
			return fmt.Errorf("play history[%d]: invalid release ID: %d", i, play.ReleaseID)
		}
		if play.PlayedAt.After(now) {
			return fmt.Errorf("play history[%d]: play date cannot be in the future", i)
		}
		if len(play.Notes) > maxNoteLength {
			return fmt.Errorf("play history[%d]: notes exceed maximum length of %d characters", i, maxNoteLength)
		}
	}

	for i, cleaning := range export.CleaningHistory {
		if cleaning.ReleaseID <= 0 {
			return fmt.Errorf("cleaning history[%d]: invalid release ID: %d", i, cleaning.ReleaseID)
		}
		if cleaning.CleanedAt.After(now) {
			return fmt.Errorf("cleaning history[%d]: cleaning date cannot be in the future", i)
		}
		if len(cleaning.Notes) > maxNoteLength {
			return fmt.Errorf("cleaning history[%d]: notes exceed maximum length of %d characters", i, maxNoteLength)
		}
	}

	for i, stylus := range export.Styluses {
		if stylus.Name == "" {
			return fmt.Errorf("stylus[%d]: missing name", i)
		}
		if stylus.Manufacturer == "" {
			return fmt.Errorf("stylus[%d]: missing manufacturer", i)
		}
		if stylus.ExpectedLifespan < 0 {
			return fmt.Errorf("stylus[%d]: invalid expected lifespan", i)
		}
	}

	return nil
}

func (s *KleioImportService) ImportKleioData(
	ctx context.Context,
	userID uuid.UUID,
	jsonData []byte,
) (*ImportSummary, error) {
	log := s.log.Function("ImportKleioData")

	var kleioExport KleioExport
	if err := json.Unmarshal(jsonData, &kleioExport); err != nil {
		return nil, log.Err("failed to parse kleio export json", err)
	}

	if err := s.validateKleioExport(&kleioExport); err != nil {
		return nil, log.Err("invalid kleio export data", err)
	}

	summary := &ImportSummary{
		FailedMappings: []string{},
	}

	err := s.transactionService.ExecuteInTransaction(ctx, func(ctx context.Context, tx *gorm.DB) error {
		stylusMapping, err := s.importStyluses(ctx, tx, userID, kleioExport.Styluses, summary)
		if err != nil {
			return log.Err("failed to import styluses", err)
		}

		releaseMapping, err := s.buildReleaseMapping(ctx, tx, userID)
		if err != nil {
			return log.Err("failed to build release mapping", err)
		}

		if err := s.importPlayHistory(ctx, tx, userID, kleioExport.PlayHistory, stylusMapping, releaseMapping, summary); err != nil {
			return log.Err("failed to import play history", err)
		}

		if err := s.importCleaningHistory(ctx, tx, userID, kleioExport.CleaningHistory, releaseMapping, summary); err != nil {
			return log.Err("failed to import cleaning history", err)
		}

		if err := s.stylusRepo.ClearUserStylusCache(ctx, userID); err != nil {
			log.Warn("failed to clear user stylus cache", "error", err)
		}

		if err := s.historyRepo.ClearUserHistoryCache(ctx, userID); err != nil {
			log.Warn("failed to clear user history cache", "error", err)
		}

		return nil
	})

	if err != nil {
		return nil, log.Err("transaction failed", err)
	}

	log.Info("Kleio import completed successfully",
		"stylusesCreated", summary.StylusesCreated,
		"playHistoryImported", summary.PlayHistoryImported,
		"cleaningImported", summary.CleaningImported,
		"deepCleansDetected", summary.DeepCleansDetected,
		"failedMappings", len(summary.FailedMappings))

	return summary, nil
}

func (s *KleioImportService) importStyluses(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	kleioStyluses []KleioStylus,
	summary *ImportSummary,
) (map[int]uuid.UUID, error) {
	log := s.log.Function("importStyluses")

	stylusMapping := make(map[int]uuid.UUID)

	existingStyluses, err := s.stylusRepo.GetAllStyluses(ctx, tx, nil)
	if err != nil {
		return nil, log.Err("failed to get existing styluses", err)
	}

	for _, kleioStylus := range kleioStyluses {
		var catalogStylusID uuid.UUID
		found := false
		for _, existing := range existingStyluses {
			if strings.EqualFold(existing.Brand, kleioStylus.Manufacturer) &&
				strings.EqualFold(existing.Model, kleioStylus.Name) {
				catalogStylusID = existing.ID
				found = true
				break
			}
		}

		if !found {
			recommendedHours := kleioStylus.ExpectedLifespan
			if recommendedHours == 0 {
				recommendedHours = 1000
			}

			catalogStylus := &Stylus{
				Brand:                   kleioStylus.Manufacturer,
				Model:                   kleioStylus.Name,
				Type:                    StylusTypeElliptical,
				RecommendedReplaceHours: &recommendedHours,
				IsVerified:              false,
			}

			if err := s.stylusRepo.CreateCustomStylus(ctx, tx, catalogStylus); err != nil {
				return nil, log.Err("failed to create catalog stylus", err)
			}

			catalogStylusID = catalogStylus.ID
			summary.StylusesCreated++
		}

		hoursUsed := decimal.NewFromInt(0)
		userStylus := &UserStylus{
			UserID:       userID,
			StylusID:     catalogStylusID,
			PurchaseDate: kleioStylus.PurchaseDate,
			HoursUsed:    &hoursUsed,
			IsActive:     kleioStylus.Active,
			IsPrimary:    kleioStylus.Primary,
		}

		if err := s.stylusRepo.Create(ctx, tx, userStylus); err != nil {
			return nil, log.Err("failed to create user stylus", err)
		}

		stylusMapping[kleioStylus.ID] = userStylus.ID
	}

	log.Info("Styluses imported successfully", "count", len(stylusMapping))
	return stylusMapping, nil
}

func (s *KleioImportService) buildReleaseMapping(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (map[int]uuid.UUID, error) {
	log := s.log.Function("buildReleaseMapping")

	existingReleases, err := s.userReleaseRepo.GetExistingByUser(ctx, tx, userID)
	if err != nil {
		return nil, log.Err("failed to get existing user releases", err)
	}

	releaseMapping := make(map[int]uuid.UUID)

	for instanceID, userRelease := range existingReleases {
		releaseMapping[int(userRelease.ReleaseID)] = userRelease.ID
		log.Info("Mapped release",
			"releaseID", userRelease.ReleaseID,
			"userReleaseID", userRelease.ID,
			"instanceID", instanceID)
	}

	log.Info("Release mapping built", "totalMapped", len(releaseMapping))
	return releaseMapping, nil
}

func (s *KleioImportService) importPlayHistory(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	kleioPlays []KleioPlayHistory,
	stylusMapping map[int]uuid.UUID,
	releaseMapping map[int]uuid.UUID,
	summary *ImportSummary,
) error {
	log := s.log.Function("importPlayHistory")

	for _, kleioPlay := range kleioPlays {
		userReleaseID, found := releaseMapping[kleioPlay.ReleaseID]
		if !found {
			warning := fmt.Sprintf("Play history: Release ID %d not found in user's collection", kleioPlay.ReleaseID)
			summary.FailedMappings = append(summary.FailedMappings, warning)
			log.Warn(warning)
			continue
		}

		var userStylusID *uuid.UUID
		if kleioPlay.StylusID != nil {
			if mappedID, ok := stylusMapping[*kleioPlay.StylusID]; ok {
				userStylusID = &mappedID
			} else {
				warning := fmt.Sprintf("Play history: Stylus ID %d not found in mapping for release %d", *kleioPlay.StylusID, kleioPlay.ReleaseID)
				summary.FailedMappings = append(summary.FailedMappings, warning)
				log.Warn(warning)
			}
		}

		playHistory := &PlayHistory{
			UserID:        userID,
			UserReleaseID: userReleaseID,
			UserStylusID:  userStylusID,
			PlayedAt:      kleioPlay.PlayedAt,
			Notes:         kleioPlay.Notes,
		}

		if err := s.historyRepo.CreatePlayHistory(ctx, tx, playHistory); err != nil {
			return log.Err("failed to create play history", err)
		}

		summary.PlayHistoryImported++
	}

	log.Info("Play history imported", "count", summary.PlayHistoryImported)
	return nil
}

func (s *KleioImportService) importCleaningHistory(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	kleioCleaning []KleioCleaningHistory,
	releaseMapping map[int]uuid.UUID,
	summary *ImportSummary,
) error {
	log := s.log.Function("importCleaningHistory")

	for _, kleioClean := range kleioCleaning {
		userReleaseID, found := releaseMapping[kleioClean.ReleaseID]
		if !found {
			warning := fmt.Sprintf("Cleaning history: Release ID %d not found in user's collection", kleioClean.ReleaseID)
			summary.FailedMappings = append(summary.FailedMappings, warning)
			log.Warn(warning)
			continue
		}

		isDeepClean := strings.Contains(strings.ToLower(kleioClean.Notes), "deep clean")
		if isDeepClean {
			summary.DeepCleansDetected++
		}

		cleaningHistory := &CleaningHistory{
			UserID:        userID,
			UserReleaseID: userReleaseID,
			CleanedAt:     kleioClean.CleanedAt,
			Notes:         kleioClean.Notes,
			IsDeepClean:   isDeepClean,
		}

		if err := s.historyRepo.CreateCleaningHistory(ctx, tx, cleaningHistory); err != nil {
			return log.Err("failed to create cleaning history", err)
		}

		summary.CleaningImported++
	}

	log.Info("Cleaning history imported",
		"count", summary.CleaningImported,
		"deepCleans", summary.DeepCleansDetected)
	return nil
}
