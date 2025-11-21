package historyController

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid RFC3339 datetime",
			dateStr:     "2024-01-15T14:30:00Z",
			expectError: false,
		},
		{
			name:        "Valid RFC3339 with timezone",
			dateStr:     "2024-01-15T14:30:00-05:00",
			expectError: false,
		},
		{
			name:        "Empty string",
			dateStr:     "",
			expectError: true,
			errorMsg:    "datetime is required",
		},
		{
			name:        "Invalid format",
			dateStr:     "2024-01-15 14:30:00",
			expectError: true,
			errorMsg:    "invalid datetime format, expected RFC3339",
		},
		{
			name:        "Invalid date",
			dateStr:     "not-a-date",
			expectError: true,
			errorMsg:    "invalid datetime format, expected RFC3339",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDateTime(tt.dateStr)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.True(t, result.IsZero())
			} else {
				assert.NoError(t, err)
				assert.False(t, result.IsZero())

				// Verify the parsed time is reasonable
				assert.True(t, result.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
			}
		})
	}
}

func TestMaxNotesLength(t *testing.T) {
	assert.Equal(t, 1000, MaxNotesLength, "MaxNotesLength should be 1000 characters")
}
