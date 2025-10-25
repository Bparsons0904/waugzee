package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

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

	genres, err := gorm.G[*Genre](tx).Find(ctx)
	if err != nil {
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

	genre, err := gorm.G[*Genre](tx).Where("id = ?", genreID).First(ctx)
	if err != nil {
		return nil, log.Err("failed to get genre by ID", err, "id", id)
	}

	return genre, nil
}

func (r *genreRepository) GetByName(ctx context.Context, tx *gorm.DB, name string) (*Genre, error) {
	log := r.log.Function("GetByName")

	genre, err := gorm.G[*Genre](tx).Where("name = ?", name).First(ctx)
	if err != nil {
		return nil, log.Err("failed to get genre by name", err, "name", name)
	}

	return genre, nil
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

	rowsAffected, err := gorm.G[*Genre](tx).Where("id = ?", genreID).Delete(ctx)
	if err != nil {
		return log.Err("failed to delete genre", err, "id", id)
	}

	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *genreRepository) FindOrCreate(
	ctx context.Context,
	tx *gorm.DB,
	name string,
) (*Genre, error) {
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

	existingGenres, err := r.GetBatchByNames(ctx, tx, names)
	if err != nil {
		return log.Err("failed to get existing genres", err, "count", len(names))
	}

	var toInsert []*Genre
	var toUpdate []*Genre

	for _, genre := range genres {
		if existing, exists := existingGenres[genre.Name]; exists {
			// Update existing genre
			genre.ID = existing.ID
			genre.CreatedAt = existing.CreatedAt
			toUpdate = append(toUpdate, genre)
		} else {
			// Insert new genre
			toInsert = append(toInsert, genre)
		}
	}

	if len(toInsert) > 0 {
		if err := r.InsertBatch(ctx, tx, toInsert); err != nil {
			return log.Err("failed to insert genre batch", err, "count", len(toInsert))
		}
	}

	if len(toUpdate) > 0 {
		if err := r.UpdateBatch(ctx, tx, toUpdate); err != nil {
			return log.Err("failed to update genre batch", err, "count", len(toUpdate))
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

	genres, err := gorm.G[*Genre](tx).Where("name IN ?", names).Find(ctx)
	if err != nil {
		return nil, log.Err("failed to get genres by names", err, "count", len(names))
	}

	// Convert to map for O(1) lookup
	result := make(map[string]*Genre, len(genres))
	for _, genre := range genres {
		result[genre.Name] = genre
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
