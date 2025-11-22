package services

import (
	"context"
	"fmt"
	"waugzee/internal/database"
	logger "github.com/Bparsons0904/goLogger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"
)

// FolderValidationService handles validation logic for folder operations.
// Extracted from FoldersService to reduce complexity and improve separation of concerns.
type FolderValidationService struct {
	log   logger.Logger
	repos repositories.Repository
	db    database.DB
}

// NewFolderValidationService creates a new folder validation service.
func NewFolderValidationService(
	repos repositories.Repository,
	db database.DB,
) *FolderValidationService {
	return &FolderValidationService{
		log:   logger.New("FolderValidationService"),
		repos: repos,
		db:    db,
	}
}

// ValidateReleases checks which releases exist and identifies missing ones.
// Returns existing release IDs and missing release IDs that need to be fetched.
func (fv *FolderValidationService) ValidateReleases(
	ctx context.Context,
	releaseIDs []int64,
) (existingReleases []int64, missingReleases []int64, err error) {
	log := fv.log.Function("ValidateReleases")

	if len(releaseIDs) == 0 {
		log.Debug("No releases to validate")
		return []int64{}, []int64{}, nil
	}

	// Check which releases exist in our database
	existingReleases, missingReleases, err = fv.repos.Release.CheckReleaseExistence(
		ctx,
		fv.db.SQLWithContext(ctx),
		releaseIDs,
	)
	if err != nil {
		return nil, nil, log.Err("failed to check release existence", err)
	}

	log.Info("Release validation completed",
		"totalReleases", len(releaseIDs),
		"existing", len(existingReleases),
		"missing", len(missingReleases))

	return existingReleases, missingReleases, nil
}

// ExtractFolderID safely extracts the folder ID from API response data.
// Includes proper validation and fallback mechanisms.
func (fv *FolderValidationService) ExtractFolderID(
	responseData map[string]any,
	response *DiscogsFolderReleasesResponse,
) (int, error) {
	if responseData == nil {
		return 0, fmt.Errorf("responseData cannot be nil")
	}

	// Try to get folderID from request data first
	folderIDRaw, exists := responseData["folderID"]
	if !exists {
		// Fallback: extract folder_id from first release in the response
		if response != nil && len(response.Data.Releases) > 0 {
			folderIDRaw = response.Data.Releases[0].FolderID
		} else {
			return 0, fmt.Errorf("missing folderID in response data and no releases to extract from")
		}
	}

	folderID, ok := folderIDRaw.(int)
	if !ok {
		return 0, fmt.Errorf("folderID is not an integer")
	}

	return folderID, nil
}

// ValidateUserForFolderOperation validates that a user can perform folder operations.
// Checks for required configuration like Discogs token and username.
func (fv *FolderValidationService) ValidateUserForFolderOperation(user *User) error {
	log := fv.log.Function("ValidateUserForFolderOperation")

	if user == nil {
		return log.ErrMsg("user cannot be nil")
	}

	if user.Configuration == nil || user.Configuration.DiscogsToken == nil ||
		*user.Configuration.DiscogsToken == "" {
		return log.ErrMsg("user does not have a Discogs token")
	}

	if user.Configuration.DiscogsUsername == nil || *user.Configuration.DiscogsUsername == "" {
		return log.ErrMsg("user does not have a Discogs username")
	}

	return nil
}

// ValidatePaginationParams validates folder pagination parameters.
// Ensures page numbers are within reasonable bounds.
func (fv *FolderValidationService) ValidatePaginationParams(folderID, page int) (int, error) {
	if folderID < 0 {
		return 0, fmt.Errorf("folderID must be non-negative")
	}

	if page < 1 {
		page = 1
	}

	if page > MaxPageNumber {
		return 0, fmt.Errorf("page number too large (max: %d)", MaxPageNumber)
	}

	return page, nil
}

// UpdateReleaseImages updates image URLs for existing releases using folder release data.
// This optimizes image data without requiring full release API calls.
func (fv *FolderValidationService) UpdateReleaseImages(
	ctx context.Context,
	originalReleases map[int]DiscogsFolderReleaseItem,
	existingReleaseIDs []int64,
) error {
	log := fv.log.Function("UpdateReleaseImages")

	if len(existingReleaseIDs) == 0 || len(originalReleases) == 0 {
		log.Debug("No releases or release data to update images for")
		return nil
	}

	// Build image updates from original release data
	imageUpdates := make([]repositories.ReleaseImageUpdate, 0)
	existingMap := make(map[int64]bool)
	for _, id := range existingReleaseIDs {
		existingMap[id] = true
	}

	for _, originalRelease := range originalReleases {
		releaseID := originalRelease.ID
		if releaseID == 0 && originalRelease.BasicInformation.ID != 0 {
			releaseID = originalRelease.BasicInformation.ID
		}

		// Only update images for existing releases
		if existingMap[releaseID] {
			update := repositories.ReleaseImageUpdate{
				ReleaseID: releaseID,
			}

			// Set thumb if available
			if originalRelease.BasicInformation.Thumb != "" {
				update.Thumb = &originalRelease.BasicInformation.Thumb
			}

			// Set cover image if available
			if originalRelease.BasicInformation.CoverImage != "" {
				update.CoverImage = &originalRelease.BasicInformation.CoverImage
			}

			// Only add update if we have at least one image to update
			if update.Thumb != nil || update.CoverImage != nil {
				imageUpdates = append(imageUpdates, update)
			}
		}
	}

	// Execute image updates
	if len(imageUpdates) > 0 {
		err := fv.repos.Release.UpdateReleaseImages(
			ctx,
			fv.db.SQLWithContext(ctx),
			imageUpdates,
		)
		if err != nil {
			return log.Err("failed to update release images", err)
		}

		log.Debug("Updated release images", "count", len(imageUpdates))
	}

	return nil
}