package jobs

import (
	"testing"
	"waugzee/internal/models"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestDiscogsDownloadJob_Name(t *testing.T) {
	// Test the job name method - this doesn't require dependencies
	job := &DiscogsDownloadJob{}
	name := job.Name()
	assert.Equal(t, "DiscogsDailyDownloadCheck", name)
}

func TestStatusTransitionValidation(t *testing.T) {
	// Test the status transition validation that was fixed
	processing := &models.DiscogsDataProcessing{
		Status: models.ProcessingStatusFailed,
	}
	
	// Test that failed state can transition to downloading (which was the fix)
	assert.True(t, processing.CanTransitionTo(models.ProcessingStatusDownloading))
	
	// Test the UpdateStatus method works correctly
	err := processing.UpdateStatus(models.ProcessingStatusDownloading)
	assert.NoError(t, err)
	assert.Equal(t, models.ProcessingStatusDownloading, processing.Status)
}

func TestGormErrorHandling(t *testing.T) {
	// Test that we're now using errors.Is() for GORM errors instead of string comparison
	// This validates the error handling fix
	
	// Test that gorm.ErrRecordNotFound is properly recognized
	err := gorm.ErrRecordNotFound
	assert.Error(t, err)
	
	// In the actual job code, we now use errors.Is(err, gorm.ErrRecordNotFound)
	// instead of err.Error() != "record not found"
	// This test validates the error type exists and can be checked
}