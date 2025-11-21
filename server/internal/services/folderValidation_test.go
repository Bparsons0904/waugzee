package services

import (
	"testing"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestFolderValidationService_ExtractFolderID(t *testing.T) {
	fv := &FolderValidationService{
		log: logger.New("test"),
	}

	t.Run("Extracts folderID from responseData", func(t *testing.T) {
		responseData := map[string]any{
			"folderID": 123,
		}
		folderID, err := fv.ExtractFolderID(responseData, nil)
		assert.NoError(t, err)
		assert.Equal(t, 123, folderID)
	})

	t.Run("Falls back to response.Data.Releases when folderID missing", func(t *testing.T) {
		responseData := map[string]any{}
		response := &DiscogsFolderReleasesResponse{
			Data: DiscogsFolderReleasesData{
				Releases: []DiscogsFolderReleaseItem{
					{FolderID: 456},
				},
			},
		}
		folderID, err := fv.ExtractFolderID(responseData, response)
		assert.NoError(t, err)
		assert.Equal(t, 456, folderID)
	})

	t.Run("Returns error when responseData is nil", func(t *testing.T) {
		folderID, err := fv.ExtractFolderID(nil, nil)
		assert.Error(t, err)
		assert.Equal(t, 0, folderID)
		assert.Contains(t, err.Error(), "responseData cannot be nil")
	})

	t.Run("Returns error when folderID missing and no releases", func(t *testing.T) {
		responseData := map[string]any{}
		folderID, err := fv.ExtractFolderID(responseData, nil)
		assert.Error(t, err)
		assert.Equal(t, 0, folderID)
		assert.Contains(t, err.Error(), "missing folderID")
	})

	t.Run("Returns error when folderID is not an integer", func(t *testing.T) {
		responseData := map[string]any{
			"folderID": "not-an-int",
		}
		folderID, err := fv.ExtractFolderID(responseData, nil)
		assert.Error(t, err)
		assert.Equal(t, 0, folderID)
		assert.Contains(t, err.Error(), "not an integer")
	})
}

func TestFolderValidationService_ValidateUserForFolderOperation(t *testing.T) {
	fv := &FolderValidationService{
		log: logger.New("test"),
	}

	t.Run("Valid user with token and username", func(t *testing.T) {
		token := "test-token"
		username := "test-user"
		user := &User{
			Configuration: &UserConfiguration{
				DiscogsToken:    &token,
				DiscogsUsername: &username,
			},
		}
		err := fv.ValidateUserForFolderOperation(user)
		assert.NoError(t, err)
	})

	t.Run("Returns error when user is nil", func(t *testing.T) {
		err := fv.ValidateUserForFolderOperation(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user cannot be nil")
	})

	t.Run("Returns error when configuration is nil", func(t *testing.T) {
		user := &User{Configuration: nil}
		err := fv.ValidateUserForFolderOperation(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Discogs token")
	})

	t.Run("Returns error when token is nil", func(t *testing.T) {
		user := &User{
			Configuration: &UserConfiguration{
				DiscogsToken: nil,
			},
		}
		err := fv.ValidateUserForFolderOperation(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Discogs token")
	})

	t.Run("Returns error when token is empty", func(t *testing.T) {
		emptyToken := ""
		user := &User{
			Configuration: &UserConfiguration{
				DiscogsToken: &emptyToken,
			},
		}
		err := fv.ValidateUserForFolderOperation(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Discogs token")
	})

	t.Run("Returns error when username is nil", func(t *testing.T) {
		token := "test-token"
		user := &User{
			Configuration: &UserConfiguration{
				DiscogsToken:    &token,
				DiscogsUsername: nil,
			},
		}
		err := fv.ValidateUserForFolderOperation(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Discogs username")
	})

	t.Run("Returns error when username is empty", func(t *testing.T) {
		token := "test-token"
		emptyUsername := ""
		user := &User{
			Configuration: &UserConfiguration{
				DiscogsToken:    &token,
				DiscogsUsername: &emptyUsername,
			},
		}
		err := fv.ValidateUserForFolderOperation(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Discogs username")
	})
}

func TestFolderValidationService_ValidatePaginationParams(t *testing.T) {
	fv := &FolderValidationService{
		log: logger.New("test"),
	}

	t.Run("Valid folderID and page", func(t *testing.T) {
		page, err := fv.ValidatePaginationParams(1, 5)
		assert.NoError(t, err)
		assert.Equal(t, 5, page)
	})

	t.Run("Defaults page to 1 when less than 1", func(t *testing.T) {
		page, err := fv.ValidatePaginationParams(1, 0)
		assert.NoError(t, err)
		assert.Equal(t, 1, page)

		page, err = fv.ValidatePaginationParams(1, -5)
		assert.NoError(t, err)
		assert.Equal(t, 1, page)
	})

	t.Run("Returns error when folderID is negative", func(t *testing.T) {
		page, err := fv.ValidatePaginationParams(-1, 1)
		assert.Error(t, err)
		assert.Equal(t, 0, page)
		assert.Contains(t, err.Error(), "folderID must be non-negative")
	})

	t.Run("Returns error when page exceeds max", func(t *testing.T) {
		// MaxPageNumber is defined in constants.go of services package
		page, err := fv.ValidatePaginationParams(1, MaxPageNumber+1)
		assert.Error(t, err)
		assert.Equal(t, 0, page)
		assert.Contains(t, err.Error(), "page number too large")
	})

	t.Run("Accepts folderID of 0", func(t *testing.T) {
		page, err := fv.ValidatePaginationParams(0, 1)
		assert.NoError(t, err)
		assert.Equal(t, 1, page)
	})
}
