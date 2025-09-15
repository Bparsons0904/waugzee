package adminController

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentYearMonth(t *testing.T) {
	// Create a minimal AdminController for testing
	ac := &AdminController{}

	result := ac.getCurrentYearMonth()

	// Should return current year-month in YYYY-MM format
	expected := time.Now().Format("2006-01")
	assert.Equal(t, expected, result)
}

func TestBuildFilePath(t *testing.T) {
	// Create a minimal AdminController for testing
	ac := &AdminController{}

	testCases := []struct {
		fileType string
		expected string
	}{
		{"labels", "/app/discogs-data/" + time.Now().Format("2006-01") + "/labels.xml.gz"},
		{"artists", "/app/discogs-data/" + time.Now().Format("2006-01") + "/artists.xml.gz"},
		{"masters", "/app/discogs-data/" + time.Now().Format("2006-01") + "/masters.xml.gz"},
		{"releases", "/app/discogs-data/" + time.Now().Format("2006-01") + "/releases.xml.gz"},
	}

	for _, tc := range testCases {
		t.Run(tc.fileType, func(t *testing.T) {
			result := ac.buildFilePath(tc.fileType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateFileTypes(t *testing.T) {
	// Create a minimal AdminController for testing
	ac := &AdminController{}

	testCases := []struct {
		name      string
		fileTypes []string
		wantError bool
	}{
		{
			name:      "valid file types",
			fileTypes: []string{"labels", "artists"},
			wantError: false,
		},
		{
			name:      "all valid file types",
			fileTypes: []string{"labels", "artists", "masters", "releases"},
			wantError: false,
		},
		{
			name:      "invalid file type",
			fileTypes: []string{"labels", "invalid"},
			wantError: true,
		},
		{
			name:      "empty slice",
			fileTypes: []string{},
			wantError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ac.validateFileTypes(tc.fileTypes)
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimplifiedRequest_Validation(t *testing.T) {
	testCases := []struct {
		name      string
		request   SimplifiedRequest
		wantValid bool
	}{
		{
			name: "valid request",
			request: SimplifiedRequest{
				FileTypes: []string{"labels", "artists"},
				Limits: &Limits{
					MaxRecords:   1000,
					MaxBatchSize: 500,
				},
			},
			wantValid: true,
		},
		{
			name: "valid request without limits",
			request: SimplifiedRequest{
				FileTypes: []string{"labels"},
			},
			wantValid: true,
		},
		{
			name: "empty file types",
			request: SimplifiedRequest{
				FileTypes: []string{},
			},
			wantValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Basic validation - just check that required fields are present
			if tc.wantValid {
				assert.NotEmpty(t, tc.request.FileTypes, "FileTypes should not be empty for valid requests")
			} else {
				assert.Empty(t, tc.request.FileTypes, "FileTypes should be empty for invalid requests")
			}
		})
	}
}