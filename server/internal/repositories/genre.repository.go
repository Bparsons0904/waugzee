package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)



type GenreRepository interface {
	GetAll(ctx context.Context) ([]*Genre, error)
	GetByID(ctx context.Context, id string) (*Genre, error)
	GetByName(ctx context.Context, name string) (*Genre, error)
	Create(ctx context.Context, genre *Genre) (*Genre, error)
	Update(ctx context.Context, genre *Genre) error
	Delete(ctx context.Context, id string) error
	FindOrCreate(ctx context.Context, name string) (*Genre, error)
	UpsertBatch(ctx context.Context, genres []*Genre) (int, int, error)
	GetBatchByNames(ctx context.Context, names []string) (map[string]*Genre, error)
	GetHashesByNames(ctx context.Context, names []string) (map[string]string, error)
	InsertBatch(ctx context.Context, genres []*Genre) (int, error)
	UpdateBatch(ctx context.Context, genres []*Genre) (int, error)
}

type genreRepository struct {
	db  database.DB
	log logger.Logger
}

func NewGenreRepository(db database.DB) GenreRepository {
	return &genreRepository{
		db:  db,
		log: logger.New("genreRepository"),
	}
}

func (r *genreRepository) GetAll(ctx context.Context) ([]*Genre, error) {
	log := r.log.Function("GetAll")

	var genres []*Genre
	if err := r.db.SQLWithContext(ctx).Find(&genres).Error; err != nil {
		return nil, log.Err("failed to get all genres", err)
	}

	return genres, nil
}

func (r *genreRepository) GetByID(ctx context.Context, id string) (*Genre, error) {
	log := r.log.Function("GetByID")

	genreID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse genre ID", err, "id", id)
	}

	var genre Genre
	if err := r.db.SQLWithContext(ctx).First(&genre, "id = ?", genreID).Error; err != nil {
		return nil, log.Err("failed to get genre by ID", err, "id", id)
	}

	return &genre, nil
}

func (r *genreRepository) GetByName(ctx context.Context, name string) (*Genre, error) {
	log := r.log.Function("GetByName")

	var genre Genre
	if err := r.db.SQLWithContext(ctx).First(&genre, "name = ?", name).Error; err != nil {
		return nil, log.Err("failed to get genre by name", err, "name", name)
	}

	return &genre, nil
}

func (r *genreRepository) Create(ctx context.Context, genre *Genre) (*Genre, error) {
	log := r.log.Function("Create")

	if err := r.db.SQLWithContext(ctx).Create(genre).Error; err != nil {
		return nil, log.Err("failed to create genre", err, "name", genre.Name)
	}

	return genre, nil
}

func (r *genreRepository) Update(ctx context.Context, genre *Genre) error {
	log := r.log.Function("Update")

	if err := r.db.SQLWithContext(ctx).Save(genre).Error; err != nil {
		return log.Err("failed to update genre", err, "id", genre.ID, "name", genre.Name)
	}

	return nil
}

func (r *genreRepository) Delete(ctx context.Context, id string) error {
	log := r.log.Function("Delete")

	genreID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse genre ID", err, "id", id)
	}

	if err := r.db.SQLWithContext(ctx).Delete(&Genre{}, "id = ?", genreID).Error; err != nil {
		return log.Err("failed to delete genre", err, "id", id)
	}

	return nil
}

func (r *genreRepository) FindOrCreate(ctx context.Context, name string) (*Genre, error) {
	log := r.log.Function("FindOrCreate")

	if name == "" {
		return nil, log.Err("genre name cannot be empty", nil)
	}

	// First, try to find existing genre
	genre, err := r.GetByName(ctx, name)
	if err == nil {
		return genre, nil
	}

	// If not found, create new genre
	newGenre := &Genre{
		Name: name,
	}

	createdGenre, err := r.Create(ctx, newGenre)
	if err != nil {
		return nil, log.Err("failed to create new genre", err, "name", name)
	}

	log.Info("Created new genre", "name", name, "id", createdGenre.ID)
	return createdGenre, nil
}

func (r *genreRepository) UpsertBatch(ctx context.Context, genres []*Genre) (int, int, error) {
	log := r.log.Function("UpsertBatch")

	if len(genres) == 0 {
		return 0, 0, nil
	}

	// 1. Extract names from incoming genres
	names := make([]string, len(genres))
	for i, genre := range genres {
		names[i] = genre.Name
	}

	// 2. Get existing hashes for these names
	existingHashes, err := r.GetHashesByNames(ctx, names)
	if err != nil {
		return 0, 0, log.Err("failed to get existing hashes", err, "count", len(names))
	}

	// 3. Convert genres to NameHashable interface
	hashableRecords := make([]utils.NameHashable, len(genres))
	for i, genre := range genres {
		hashableRecords[i] = genre
	}

	// 4. Categorize records by hash comparison
	categories := utils.CategorizeRecordsByNameHash(hashableRecords, existingHashes)

	var insertedCount, updatedCount int

	// 5. Execute insert batch for new records
	if len(categories.Insert) > 0 {
		insertGenres := make([]*Genre, len(categories.Insert))
		for i, record := range categories.Insert {
			insertGenres[i] = record.(*Genre)
		}
		insertedCount, err = r.InsertBatch(ctx, insertGenres)
		if err != nil {
			return 0, 0, log.Err("failed to insert genre batch", err, "count", len(insertGenres))
		}
	}

	// 6. Execute update batch for changed records
	if len(categories.Update) > 0 {
		updateGenres := make([]*Genre, len(categories.Update))
		for i, record := range categories.Update {
			updateGenres[i] = record.(*Genre)
		}
		updatedCount, err = r.UpdateBatch(ctx, updateGenres)
		if err != nil {
			return insertedCount, 0, log.Err("failed to update genre batch", err, "count", len(updateGenres))
		}
	}

	log.Info("Hash-based upsert completed",
		"total", len(genres),
		"inserted", insertedCount,
		"updated", updatedCount,
		"skipped", len(categories.Skip))

	return insertedCount, updatedCount, nil
}

func (r *genreRepository) GetBatchByNames(
	ctx context.Context,
	names []string,
) (map[string]*Genre, error) {
	log := r.log.Function("GetBatchByNames")

	if len(names) == 0 {
		return make(map[string]*Genre), nil
	}

	var genres []*Genre
	if err := r.db.SQLWithContext(ctx).Where("name IN ?", names).Find(&genres).Error; err != nil {
		return nil, log.Err("failed to get genres by names", err, "count", len(names))
	}

	// Convert to map for O(1) lookup
	result := make(map[string]*Genre, len(genres))
	for _, genre := range genres {
		result[genre.Name] = genre
	}

	log.Info("Retrieved genres by names", "requested", len(names), "found", len(result))
	return result, nil
}

func (r *genreRepository) GetHashesByNames(
	ctx context.Context,
	names []string,
) (map[string]string, error) {
	log := r.log.Function("GetHashesByNames")

	if len(names) == 0 {
		return make(map[string]string), nil
	}

	var genres []struct {
		Name        string `json:"name"`
		ContentHash string `json:"contentHash"`
	}

	if err := r.db.SQLWithContext(ctx).
		Model(&Genre{}).
		Select("name, content_hash").
		Where("name IN ?", names).
		Find(&genres).Error; err != nil {
		return nil, log.Err("failed to get genre hashes by names", err, "count", len(names))
	}

	result := make(map[string]string, len(genres))
	for _, genre := range genres {
		result[genre.Name] = genre.ContentHash
	}

	log.Info("Retrieved genre hashes by names", "requested", len(names), "found", len(result))
	return result, nil
}

func (r *genreRepository) InsertBatch(ctx context.Context, genres []*Genre) (int, error) {
	log := r.log.Function("InsertBatch")

	if len(genres) == 0 {
		return 0, nil
	}

	if err := r.db.SQLWithContext(ctx).Create(&genres).Error; err != nil {
		return 0, log.Err("failed to insert genre batch", err, "count", len(genres))
	}

	log.Info("Inserted genres", "count", len(genres))
	return len(genres), nil
}

func (r *genreRepository) UpdateBatch(ctx context.Context, genres []*Genre) (int, error) {
	log := r.log.Function("UpdateBatch")

	if len(genres) == 0 {
		return 0, nil
	}

	updatedCount := 0
	for _, genre := range genres {
		// Get existing record first to ensure we have the complete model
		existingGenre := &Genre{}
		err := r.db.SQLWithContext(ctx).Where("name = ?", genre.Name).First(existingGenre).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// Skip if record doesn't exist (should not happen in our flow)
				log.Warn("Genre not found for update", "name", genre.Name)
				continue
			}
			return updatedCount, log.Err("failed to get existing genre", err, "name", genre.Name)
		}

		// Update only the specific fields we want to change
		existingGenre.ParentGenreID = genre.ParentGenreID
		existingGenre.ContentHash = genre.ContentHash

		// Use Save() which handles all GORM hooks properly
		result := r.db.SQLWithContext(ctx).Save(existingGenre)
		if result.Error != nil {
			return updatedCount, log.Err("failed to save genre", result.Error, "name", genre.Name)
		}

		if result.RowsAffected > 0 {
			updatedCount++
		}
	}

	log.Info("Updated genres", "count", updatedCount)
	return updatedCount, nil
}
