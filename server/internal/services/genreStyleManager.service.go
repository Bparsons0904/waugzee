package services

import (
	"context"
	"strings"
	"unicode/utf8"
	logger "github.com/Bparsons0904/goLogger"
	"waugzee/internal/models"
	"waugzee/internal/repositories"

	"gorm.io/gorm"
)

type GenreStyleManager struct {
	log       logger.Logger
	genreRepo repositories.GenreRepository
	nameToID  map[string]int64    // Maps "name|type" to ID for fast lookups
	lowNames  map[string]struct{} // Tracks collected "name|type" pairs to avoid duplicates
}

func NewGenreStyleManager(genreRepo repositories.GenreRepository) *GenreStyleManager {
	return &GenreStyleManager{
		log:       logger.New("genreStyleManager"),
		genreRepo: genreRepo,
		nameToID:  make(map[string]int64),
		lowNames:  make(map[string]struct{}),
	}
}

func (gsm *GenreStyleManager) Reset() {
	gsm.nameToID = make(map[string]int64)
	gsm.lowNames = make(map[string]struct{})
}

func (gsm *GenreStyleManager) CollectNames(genres []string, styles []string) {
	for _, genre := range genres {
		if genre = strings.TrimSpace(genre); genre != "" {
			key := strings.ToLower(genre) + "|genre"
			gsm.lowNames[key] = struct{}{}
		}
	}

	for _, style := range styles {
		if style = strings.TrimSpace(style); style != "" {
			key := strings.ToLower(style) + "|style"
			gsm.lowNames[key] = struct{}{}
		}
	}
}

func (gsm *GenreStyleManager) BatchUpsertMissingGenres(ctx context.Context, tx *gorm.DB) error {
	log := gsm.log.Function("BatchUpsertMissingGenres")

	if len(gsm.lowNames) == 0 {
		return nil
	}

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


	existingGenres := make([]*models.Genre, 0)
	if err := tx.WithContext(ctx).Find(&existingGenres).Error; err != nil {
		return log.Err("failed to fetch existing genres", err)
	}

	existingMap := make(map[string]*models.Genre)
	for _, genre := range existingGenres {
		key := genre.NameLower + "|" + genre.Type
		existingMap[key] = genre
	}


	var newGenres []*models.Genre
	for _, entry := range entries {
		if _, exists := existingMap[entry.Key]; !exists {
			properName := gsm.toProperCase(entry.Name)

			cleanName := gsm.cleanUTF8String(properName)
			cleanNameLower := strings.ToLower(cleanName)

			cleanKey := cleanNameLower + "|" + entry.Type
			if _, exists := existingMap[cleanKey]; !exists {
				newGenres = append(newGenres, &models.Genre{
					Name: cleanName,
					Type: entry.Type,
				})
			}
		}
	}

	if len(newGenres) > 0 {
		if err := gsm.genreRepo.InsertBatch(ctx, tx, newGenres); err != nil {
			return log.Err("failed to insert new genres/styles", err, "count", len(newGenres))
		}

		for _, genre := range newGenres {
			key := genre.NameLower + "|" + genre.Type
			existingMap[key] = genre
		}
	}

	gsm.nameToID = make(map[string]int64, len(existingMap))
	for key, genre := range existingMap {
		gsm.nameToID[key] = genre.ID
	}

	return nil
}

func (gsm *GenreStyleManager) GetGenreIDsByNames(genres []string, styles []string) []int64 {
	var genreIDs []int64
	keysSeen := make(map[string]struct{})

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

func (gsm *GenreStyleManager) toProperCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func (gsm *GenreStyleManager) cleanUTF8String(s string) string {
	if !utf8.ValidString(s) || strings.Contains(s, "\x00") {
		return strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", ""), "")
	}
	return s
}

func (gsm *GenreStyleManager) GetStats() map[string]any {
	return map[string]any{
		"collectedNames": len(gsm.lowNames),
		"genreMappings":  len(gsm.nameToID),
	}
}

