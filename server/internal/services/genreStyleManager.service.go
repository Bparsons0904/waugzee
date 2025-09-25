package services

import (
	"context"
	"strings"
	"waugzee/internal/logger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"

	"gorm.io/gorm"
)

// GenreStyleManager handles in-memory genre/style processing for XML parsing
type GenreStyleManager struct {
	log       logger.Logger
	genreRepo repositories.GenreRepository
	nameToID  map[string]int64    // Maps "name|type" to ID for fast lookups
	lowNames  map[string]struct{} // Tracks collected "name|type" pairs to avoid duplicates
}

// NewGenreStyleManager creates a new genre/style manager
func NewGenreStyleManager(genreRepo repositories.GenreRepository) *GenreStyleManager {
	return &GenreStyleManager{
		log:       logger.New("genreStyleManager"),
		genreRepo: genreRepo,
		nameToID:  make(map[string]int64),
		lowNames:  make(map[string]struct{}),
	}
}

// Reset clears the in-memory state for processing a new entity type
func (gsm *GenreStyleManager) Reset() {
	gsm.nameToID = make(map[string]int64)
	gsm.lowNames = make(map[string]struct{})
}

// CollectNames collects unique genre and style names from the provided lists
func (gsm *GenreStyleManager) CollectNames(genres []string, styles []string) {
	// Collect genres
	for _, genre := range genres {
		if genre = strings.TrimSpace(genre); genre != "" {
			key := strings.ToLower(genre) + "|genre"
			gsm.lowNames[key] = struct{}{}
		}
	}

	// Collect styles as separate type
	for _, style := range styles {
		if style = strings.TrimSpace(style); style != "" {
			key := strings.ToLower(style) + "|style"
			gsm.lowNames[key] = struct{}{}
		}
	}
}

// BatchUpsertMissingGenres ensures all collected genre/style names exist in the database
func (gsm *GenreStyleManager) BatchUpsertMissingGenres(ctx context.Context, tx *gorm.DB) error {
	log := gsm.log.Function("BatchUpsertMissingGenres")

	if len(gsm.lowNames) == 0 {
		return nil
	}

	// Parse collected keys into name/type pairs
	type GenreEntry struct {
		Name string
		Type string
		Key  string
	}

	entries := make([]GenreEntry, 0, len(gsm.lowNames))
	for key := range gsm.lowNames {
		parts := strings.Split(key, "|")
		if len(parts) == 2 {
			entries = append(entries, GenreEntry{
				Name: parts[0],
				Type: parts[1],
				Key:  key,
			})
		}
	}

	log.Info("Processing collected genre/style entries", "count", len(entries))

	// Get existing genres by name and type
	existingGenres := make([]*models.Genre, 0)
	if err := tx.WithContext(ctx).Find(&existingGenres).Error; err != nil {
		return log.Err("failed to fetch existing genres", err)
	}

	// Build map of existing genres using name|type key
	existingMap := make(map[string]*models.Genre)
	for _, genre := range existingGenres {
		key := genre.NameLower + "|" + genre.Type
		existingMap[key] = genre
	}

	// Find missing genres that need to be created
	var newGenres []*models.Genre
	for _, entry := range entries {
		if _, exists := existingMap[entry.Key]; !exists {
			// Create proper case name (capitalize first letter of each word)
			properName := gsm.toProperCase(entry.Name)
			newGenres = append(newGenres, &models.Genre{
				Name: properName,
				Type: entry.Type,
			})
		}
	}

	// Batch insert new genres
	if len(newGenres) > 0 {
		log.Info("Creating new genres/styles", "count", len(newGenres))
		if err := gsm.genreRepo.InsertBatch(ctx, tx, newGenres); err != nil {
			return log.Err("failed to insert new genres/styles", err, "count", len(newGenres))
		}

		// Add newly created genres to existing map
		for _, genre := range newGenres {
			key := genre.NameLower + "|" + genre.Type
			existingMap[key] = genre
		}
	}

	// Build name|type-to-ID lookup map for fast association processing
	gsm.nameToID = make(map[string]int64, len(existingMap))
	for key, genre := range existingMap {
		gsm.nameToID[key] = genre.ID
	}

	log.Info("Prepared genre lookup map", "totalGenres", len(gsm.nameToID))
	return nil
}

// GetGenreIDsByNames returns genre IDs for the given genre and style names
func (gsm *GenreStyleManager) GetGenreIDsByNames(genres []string, styles []string) []int64 {
	var genreIDs []int64
	keysSeen := make(map[string]struct{})

	// Process genres
	for _, genre := range genres {
		if genre = strings.TrimSpace(genre); genre != "" {
			key := strings.ToLower(genre) + "|genre"
			if _, seen := keysSeen[key]; !seen {
				keysSeen[key] = struct{}{}
				if id, exists := gsm.nameToID[key]; exists {
					genreIDs = append(genreIDs, id)
				}
			}
		}
	}

	// Process styles as separate type
	for _, style := range styles {
		if style = strings.TrimSpace(style); style != "" {
			key := strings.ToLower(style) + "|style"
			if _, seen := keysSeen[key]; !seen {
				keysSeen[key] = struct{}{}
				if id, exists := gsm.nameToID[key]; exists {
					genreIDs = append(genreIDs, id)
				}
			}
		}
	}

	return genreIDs
}

// GetGenresByNames returns genre models for the given genre and style names
func (gsm *GenreStyleManager) GetGenresByNames(
	ctx context.Context,
	tx *gorm.DB,
	genres []string,
	styles []string,
) ([]*models.Genre, error) {
	genreIDs := gsm.GetGenreIDsByNames(genres, styles)

	if len(genreIDs) == 0 {
		return []*models.Genre{}, nil
	}

	var genreModels []*models.Genre
	if err := tx.WithContext(ctx).Where("id IN ?", genreIDs).Find(&genreModels).Error; err != nil {
		return nil, gsm.log.Err("failed to fetch genres by IDs", err, "count", len(genreIDs))
	}

	return genreModels, nil
}

// toProperCase converts a lowercase string to proper case (capitalizes first letter of each word)
func (gsm *GenreStyleManager) toProperCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// GetStats returns statistics about the current manager state
func (gsm *GenreStyleManager) GetStats() map[string]any {
	return map[string]any{
		"collectedNames": len(gsm.lowNames),
		"genreMappings":  len(gsm.nameToID),
	}
}

