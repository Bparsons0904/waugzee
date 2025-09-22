package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GenreRepository interface {
	GetAll(ctx context.Context, tx *gorm.DB) ([]*Genre, error)
	GetByID(ctx context.Context, tx *gorm.DB, id string) (*Genre, error)
	GetByName(ctx context.Context, tx *gorm.DB, name string) (*Genre, error)
	Create(ctx context.Context, tx *gorm.DB, genre *Genre) (*Genre, error)
	Update(ctx context.Context, tx *gorm.DB, genre *Genre) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error
	FindOrCreate(ctx context.Context, tx *gorm.DB, name string) (*Genre, error)
	UpsertBatch(ctx context.Context, tx *gorm.DB, genres []*Genre) error
	GetBatchByNames(ctx context.Context, tx *gorm.DB, names []string) (map[string]*Genre, error)
	GetHashesByNames(ctx context.Context, tx *gorm.DB, names []string) (map[string]string, error)
	InsertBatch(ctx context.Context, tx *gorm.DB, genres []*Genre) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, genres []*Genre) error
}

type genreRepository struct {
	log logger.Logger
}

func NewGenreRepository() GenreRepository {
	return &genreRepository{
		log: logger.New("genreRepository"),
	}
}

func (r *genreRepository) GetAll(ctx context.Context, tx *gorm.DB) ([]*Genre, error) {
	log := r.log.Function("GetAll")

	var genres []*Genre
	if err := tx.WithContext(ctx).Find(&genres).Error; err != nil {
		return nil, log.Err("failed to get all genres", err)
	}

	return genres, nil
}

func (r *genreRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (*Genre, error) {
	log := r.log.Function("GetByID")

	genreID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse genre ID", err, "id", id)
	}

	var genre Genre
	if err := tx.WithContext(ctx).First(&genre, "id = ?", genreID).Error; err != nil {
		return nil, log.Err("failed to get genre by ID", err, "id", id)
	}

	return &genre, nil
}

func (r *genreRepository) GetByName(ctx context.Context, tx *gorm.DB, name string) (*Genre, error) {
	log := r.log.Function("GetByName")

	var genre Genre
	if err := tx.WithContext(ctx).First(&genre, "name = ?", name).Error; err != nil {
		return nil, log.Err("failed to get genre by name", err, "name", name)
	}

	return &genre, nil
}

func (r *genreRepository) Create(ctx context.Context, tx *gorm.DB, genre *Genre) (*Genre, error) {
	log := r.log.Function("Create")

	if err := tx.WithContext(ctx).Create(genre).Error; err != nil {
		return nil, log.Err("failed to create genre", err, "name", genre.Name)
	}

	return genre, nil
}

func (r *genreRepository) Update(ctx context.Context, tx *gorm.DB, genre *Genre) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(genre).Error; err != nil {
		return log.Err("failed to update genre", err, "id", genre.ID, "name", genre.Name)
	}

	return nil
}

func (r *genreRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	log := r.log.Function("Delete")

	genreID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse genre ID", err, "id", id)
	}

	if err := tx.WithContext(ctx).Delete(&Genre{}, "id = ?", genreID).Error; err != nil {
		return log.Err("failed to delete genre", err, "id", id)
	}

	return nil
}

func (r *genreRepository) FindOrCreate(ctx context.Context, tx *gorm.DB, name string) (*Genre, error) {
	log := r.log.Function("FindOrCreate")

	if name == "" {
		return nil, log.Err("genre name cannot be empty", nil)
	}

	// First, try to find existing genre
	genre, err := r.GetByName(ctx, tx, name)
	if err == nil {
		return genre, nil
	}

	// If not found, create new genre
	newGenre := &Genre{
		Name: name,
	}

	createdGenre, err := r.Create(ctx, tx, newGenre)
	if err != nil {
		return nil, log.Err("failed to create new genre", err, "name", name)
	}

	log.Info("Created new genre", "name", name, "id", createdGenre.ID)
	return createdGenre, nil
}

func (r *genreRepository) UpsertBatch(ctx context.Context, tx *gorm.DB, genres []*Genre) error {
	log := r.log.Function("UpsertBatch")

	if len(genres) == 0 {
		return nil
	}

	names := make([]string, len(genres))
	for i, genre := range genres {
		names[i] = genre.Name
	}

	existingHashes, err := r.GetHashesByNames(ctx, tx, names)
	if err != nil {
		return log.Err("failed to get existing hashes", err, "count", len(names))
	}

	hashableRecords := make([]utils.NameHashable, len(genres))
	for i, genre := range genres {
		hashableRecords[i] = genre
	}

	categories := utils.CategorizeRecordsByNameHash(hashableRecords, existingHashes)

	if len(categories.Insert) > 0 {
		insertGenres := make([]*Genre, len(categories.Insert))
		for i, record := range categories.Insert {
			insertGenres[i] = record.(*Genre)
		}
		err = r.InsertBatch(ctx, tx, insertGenres)
		if err != nil {
			return log.Err("failed to insert genre batch", err, "count", len(insertGenres))
		}
	}

	if len(categories.Update) > 0 {
		updateGenres := make([]*Genre, len(categories.Update))
		for i, record := range categories.Update {
			updateGenres[i] = record.(*Genre)
		}
		err = r.UpdateBatch(ctx, tx, updateGenres)
		if err != nil {
			return log.Err("failed to update genre batch", err, "count", len(updateGenres))
		}
	}

	return nil
}

func (r *genreRepository) GetBatchByNames(
	ctx context.Context,
	tx *gorm.DB,
	names []string,
) (map[string]*Genre, error) {
	log := r.log.Function("GetBatchByNames")

	if len(names) == 0 {
		return make(map[string]*Genre), nil
	}

	var genres []*Genre
	if err := tx.WithContext(ctx).Where("name IN ?", names).Find(&genres).Error; err != nil {
		return nil, log.Err("failed to get genres by names", err, "count", len(names))
	}

	// Convert to map for O(1) lookup
	result := make(map[string]*Genre, len(genres))
	for _, genre := range genres {
		result[genre.Name] = genre
	}

	return result, nil
}

func (r *genreRepository) GetHashesByNames(
	ctx context.Context,
	tx *gorm.DB,
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

	if err := tx.WithContext(ctx).
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

	return result, nil
}

func (r *genreRepository) InsertBatch(ctx context.Context, tx *gorm.DB, genres []*Genre) error {
	log := r.log.Function("InsertBatch")

	if len(genres) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(&genres).Error; err != nil {
		return log.Err("failed to insert genre batch", err, "count", len(genres))
	}

	return nil
}

func (r *genreRepository) UpdateBatch(ctx context.Context, tx *gorm.DB, genres []*Genre) error {
	log := r.log.Function("UpdateBatch")

	if len(genres) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Save(&genres).Error; err != nil {
		return log.Err("failed to update genre batch", err, "count", len(genres))
	}

	return nil
}
