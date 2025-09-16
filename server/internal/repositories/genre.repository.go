package repositories

import (
	"context"
	"waugzee/internal/database"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	contextutil "waugzee/internal/context"

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

func (r *genreRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextutil.GetTransaction(ctx); ok {
		return tx
	}
	return r.db.SQLWithContext(ctx)
}

func (r *genreRepository) GetAll(ctx context.Context) ([]*Genre, error) {
	log := r.log.Function("GetAll")

	var genres []*Genre
	if err := r.getDB(ctx).Find(&genres).Error; err != nil {
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
	if err := r.getDB(ctx).First(&genre, "id = ?", genreID).Error; err != nil {
		return nil, log.Err("failed to get genre by ID", err, "id", id)
	}

	return &genre, nil
}

func (r *genreRepository) GetByName(ctx context.Context, name string) (*Genre, error) {
	log := r.log.Function("GetByName")

	var genre Genre
	if err := r.getDB(ctx).First(&genre, "name = ?", name).Error; err != nil {
		return nil, log.Err("failed to get genre by name", err, "name", name)
	}

	return &genre, nil
}

func (r *genreRepository) Create(ctx context.Context, genre *Genre) (*Genre, error) {
	log := r.log.Function("Create")

	if err := r.getDB(ctx).Create(genre).Error; err != nil {
		return nil, log.Err("failed to create genre", err, "name", genre.Name)
	}

	return genre, nil
}

func (r *genreRepository) Update(ctx context.Context, genre *Genre) error {
	log := r.log.Function("Update")

	if err := r.getDB(ctx).Save(genre).Error; err != nil {
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

	if err := r.getDB(ctx).Delete(&Genre{}, "id = ?", genreID).Error; err != nil {
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

	inserted := 0
	updated := 0

	for _, genre := range genres {
		existingGenre, err := r.GetByName(ctx, genre.Name)
		if err != nil {
			// Genre doesn't exist, create it
			_, err := r.Create(ctx, genre)
			if err != nil {
				return inserted, updated, log.Err("failed to create genre", err, "name", genre.Name)
			}
			inserted++
		} else {
			// Genre exists, update if needed
			if existingGenre.ParentGenreID != genre.ParentGenreID {
				existingGenre.ParentGenreID = genre.ParentGenreID

				if err := r.Update(ctx, existingGenre); err != nil {
					return inserted, updated, log.Err("failed to update genre", err, "name", genre.Name)
				}
				updated++
			}
		}
	}

	log.Info("Batch upsert completed", "inserted", inserted, "updated", updated)
	return inserted, updated, nil
}