package jobs

import (
	"testing"
	"time"
	"waugzee/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestDiscogsDownloadJob_Name tests the job name
func TestDiscogsDownloadJob_Name(t *testing.T) {
	job := &DiscogsDownloadJob{}
	assert.Equal(t, "DiscogsDailyDownloadCheck", job.Name())
}

// TestDiscogsDataProcessing_Model tests the model functionality
func TestDiscogsDataProcessing_Model(t *testing.T) {
	processing := &models.DiscogsDataProcessing{
		YearMonth: "2025-09",
		Status:    models.ProcessingStatusNotStarted,
	}

	// Test status update
	err := processing.UpdateStatus(models.ProcessingStatusDownloading)
	assert.NoError(t, err)
	assert.Equal(t, models.ProcessingStatusDownloading, processing.Status)

	// Test checksum functionality
	checksums := &models.FileChecksums{
		ArtistsDump: "abc123",
		LabelsDump:  "def456",
	}
	processing.FileChecksums = checksums
	processing.Status = models.ProcessingStatusReadyForProcessing
	now := time.Now()
	processing.DownloadCompletedAt = &now

	// Test ready for processing
	assert.True(t, processing.IsReadyForProcessing())

	// Test file checksum getter
	assert.Equal(t, "abc123", processing.GetFileChecksum("artists"))
	assert.Equal(t, "def456", processing.GetFileChecksum("labels"))
	assert.Equal(t, "", processing.GetFileChecksum("masters"))
	assert.Equal(t, "", processing.GetFileChecksum("unknown"))

	// Test available file types
	types := processing.GetAvailableFileTypes()
	assert.Contains(t, types, "artists")
	assert.Contains(t, types, "labels")
	assert.NotContains(t, types, "masters")
	assert.NotContains(t, types, "releases")
}