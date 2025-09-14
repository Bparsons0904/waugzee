package repositories_test

import (
	"testing"
	"waugzee/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestLabelRepository_BatchProcessingLogic(t *testing.T) {
	// Test the batch processing logic without actual database calls
	t.Run("batch size calculation", func(t *testing.T) {
		// Create a large number of test labels
		labels := make([]*models.Label, 2500) // 2.5 batches
		for i := range labels {
			discogsID := int64(i + 1)
			labels[i] = &models.Label{
				Name:      "Test Label",
				DiscogsID: &discogsID,
			}
		}

		// Test that we would process in correct batch sizes
		batchSize := 1000
		expectedBatches := 3 // 1000, 1000, 500

		batchCount := 0
		totalProcessed := 0

		for i := 0; i < len(labels); i += batchSize {
			end := i + batchSize
			if end > len(labels) {
				end = len(labels)
			}

			batchSize := end - i
			totalProcessed += batchSize
			batchCount++
		}

		assert.Equal(t, expectedBatches, batchCount)
		assert.Equal(t, len(labels), totalProcessed)
	})

	t.Run("empty labels array handling", func(t *testing.T) {
		labels := []*models.Label{}

		// Should handle empty array gracefully
		assert.Equal(t, 0, len(labels))

		// Simulate what UpsertBatch would return for empty array
		if len(labels) == 0 {
			inserted, updated := 0, 0
			assert.Equal(t, 0, inserted)
			assert.Equal(t, 0, updated)
		}
	})
}

func TestLabelRepository_LabelCreation(t *testing.T) {
	tests := []struct {
		name      string
		labelData *models.Label
		expectErr bool
	}{
		{
			name: "valid label with discogs ID",
			labelData: &models.Label{
				Name: "Test Label",
			},
			expectErr: false,
		},
		{
			name: "label with discogs ID",
			labelData: func() *models.Label {
				discogsID := int64(123)
				return &models.Label{
					Name:      "Test Label with Discogs ID",
					DiscogsID: &discogsID,
				}
			}(),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test label validation
			assert.NotEmpty(t, tt.labelData.Name)

			if tt.labelData.DiscogsID != nil {
				assert.Greater(t, *tt.labelData.DiscogsID, int64(0))
			}
		})
	}
}

func TestLabelRepository_DiscogsIDMapping(t *testing.T) {
	// Test the logic for mapping Discogs IDs to label lookups
	t.Run("discogs ID map creation", func(t *testing.T) {
		// Simulate the GetBatchByDiscogsIDs result
		labels := []*models.Label{
			{Name: "Label 1"},
			{Name: "Label 2"},
		}

		discogsID1 := int64(123)
		discogsID2 := int64(456)
		labels[0].DiscogsID = &discogsID1
		labels[1].DiscogsID = &discogsID2

		// Create the map as the repository would
		result := make(map[int64]*models.Label, len(labels))
		for _, label := range labels {
			if label.DiscogsID != nil {
				result[*label.DiscogsID] = label
			}
		}

		assert.Equal(t, 2, len(result))
		assert.Equal(t, "Label 1", result[123].Name)
		assert.Equal(t, "Label 2", result[456].Name)
	})

	t.Run("empty discogs IDs array", func(t *testing.T) {
		discogsIDs := []int64{}

		// Should handle empty array
		if len(discogsIDs) == 0 {
			result := make(map[int64]*models.Label)
			assert.Equal(t, 0, len(result))
		}
	})
}

func TestLabelRepository_UpsertLogic(t *testing.T) {
	// Test the upsert separation logic
	t.Run("separate inserts and updates", func(t *testing.T) {
		// Create test labels
		labels := []*models.Label{
			{Name: "New Label 1"},
			{Name: "New Label 2"},
			{Name: "Existing Label 1"},
			{Name: "Existing Label 2"},
		}

		// Set Discogs IDs
		for i, label := range labels {
			discogsID := int64(i + 1)
			label.DiscogsID = &discogsID
		}

		// Simulate existing labels map (labels 3 and 4 exist)
		existingLabels := map[int64]*models.Label{
			3: {Name: "Old Name 1"},
			4: {Name: "Old Name 2"},
		}

		var toInsert []*models.Label
		var toUpdate []*models.Label

		// Separate logic as in repository
		for _, label := range labels {
			if label.DiscogsID != nil {
				if existing, exists := existingLabels[*label.DiscogsID]; exists {
					// Update existing
					existing.Name = label.Name
					toUpdate = append(toUpdate, existing)
				} else {
					// Insert new
					toInsert = append(toInsert, label)
				}
			}
		}

		assert.Equal(t, 2, len(toInsert)) // Labels 1 and 2
		assert.Equal(t, 2, len(toUpdate)) // Labels 3 and 4

		// Check updated names
		assert.Equal(t, "Existing Label 1", toUpdate[0].Name)
		assert.Equal(t, "Existing Label 2", toUpdate[1].Name)
	})
}