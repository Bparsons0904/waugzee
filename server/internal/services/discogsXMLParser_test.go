package services

import (
	"testing"
	"waugzee/internal/models"
)

func TestProcessingStepTracking(t *testing.T) {
	// Create a processing record
	processing := &models.DiscogsDataProcessing{
		YearMonth: "2025-09",
		Status:    models.ProcessingStatusReadyForProcessing,
	}
	processing.InitializeProcessingSteps()

	// Test step completion tracking
	if processing.IsStepCompleted(models.StepLabelsProcessing) {
		t.Errorf("Step should not be completed initially")
	}

	// Mark step as completed
	recordsCount := int64(1000)
	duration := "5m30s"
	processing.MarkStepCompleted(models.StepLabelsProcessing, &recordsCount, &duration)

	// Verify step is marked as completed
	if !processing.IsStepCompleted(models.StepLabelsProcessing) {
		t.Errorf("Step should be marked as completed")
	}

	// Verify step status details
	stepStatus := processing.GetStepStatus(models.StepLabelsProcessing)
	if stepStatus == nil {
		t.Errorf("Step status should not be nil")
		return
	}
	if !stepStatus.Completed {
		t.Errorf("Step status should be completed")
	}
	if stepStatus.CompletedAt == nil {
		t.Errorf("Completed at should be set")
	}
	if stepStatus.RecordsCount == nil || *stepStatus.RecordsCount != recordsCount {
		t.Errorf("Records count should be %d", recordsCount)
	}
	if stepStatus.Duration == nil || *stepStatus.Duration != duration {
		t.Errorf("Duration should be %s", duration)
	}
	if stepStatus.ErrorMessage != nil {
		t.Errorf("Error message should be nil for successful completion")
	}
}

func TestProcessingStepFailure(t *testing.T) {
	// Create a processing record
	processing := &models.DiscogsDataProcessing{
		YearMonth: "2025-09",
		Status:    models.ProcessingStatusReadyForProcessing,
	}
	processing.InitializeProcessingSteps()

	// Test step failure tracking
	errorMessage := "test error occurred"
	processing.MarkStepFailed(models.StepLabelsProcessing, errorMessage)

	// Verify step is marked as failed (not completed)
	if processing.IsStepCompleted(models.StepLabelsProcessing) {
		t.Errorf("Failed step should not be marked as completed")
	}

	// Verify step status details
	stepStatus := processing.GetStepStatus(models.StepLabelsProcessing)
	if stepStatus == nil {
		t.Errorf("Step status should not be nil")
		return
	}
	if stepStatus.Completed {
		t.Errorf("Step status should not be completed")
	}
	if stepStatus.CompletedAt != nil {
		t.Errorf("Completed at should be nil for failed step")
	}
	if stepStatus.ErrorMessage == nil || *stepStatus.ErrorMessage != errorMessage {
		t.Errorf("Error message should be '%s'", errorMessage)
	}
}

func TestAllStepsCompleted(t *testing.T) {
	processing := &models.DiscogsDataProcessing{
		YearMonth: "2025-09",
		Status:    models.ProcessingStatusReadyForProcessing,
	}
	processing.InitializeProcessingSteps()

	// Initially, no steps completed
	if processing.AllStepsCompleted() {
		t.Errorf("No steps should be completed initially")
	}

	// Mark all steps as completed
	allSteps := []models.ProcessingStep{
		models.StepLabelsProcessing,
		models.StepArtistsProcessing,
		models.StepMastersProcessing,
		models.StepReleasesProcessing,
		models.StepMasterGenresCollection,
		models.StepMasterGenresUpsert,
		models.StepMasterGenreAssociations,
		models.StepReleaseGenresCollection,
		models.StepReleaseGenresUpsert,
		models.StepReleaseGenreAssociations,
		models.StepReleaseLabelAssociations,
		models.StepMasterArtistAssociations,
		models.StepReleaseArtistAssociations,
	}

	for _, step := range allSteps {
		processing.MarkStepCompleted(step, nil, nil)
	}

	// Now all steps should be completed
	if !processing.AllStepsCompleted() {
		t.Errorf("All steps should be completed")
	}
}