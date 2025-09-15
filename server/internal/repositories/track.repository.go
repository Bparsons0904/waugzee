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

const (
	TRACK_BATCH_SIZE = 1000
)

type TrackRepository interface {
	GetByID(ctx context.Context, id string) (*Track, error)
	GetByReleaseID(ctx context.Context, releaseID string) ([]*Track, error)
	Create(ctx context.Context, track *Track) (*Track, error)
	Update(ctx context.Context, track *Track) error
	Delete(ctx context.Context, id string) error
	CreateBatch(ctx context.Context, tracks []*Track) error
	DeleteByReleaseID(ctx context.Context, releaseID string) error
}

type trackRepository struct {
	db  database.DB
	log logger.Logger
}

func NewTrackRepository(db database.DB) TrackRepository {
	return &trackRepository{
		db:  db,
		log: logger.New("trackRepository"),
	}
}

func (t *trackRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := contextutil.GetTransaction(ctx); ok {
		return tx
	}
	return t.db.SQLWithContext(ctx)
}

func (t *trackRepository) GetByID(ctx context.Context, id string) (*Track, error) {
	log := t.log.Function("GetByID")

	trackID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse track ID", err, "id", id)
	}

	var track Track
	if err := t.getDB(ctx).Preload("Release").Preload("Artist").First(&track, "id = ?", trackID).Error; err != nil {
		return nil, log.Err("failed to get track by ID", err, "id", id)
	}

	return &track, nil
}

func (t *trackRepository) GetByReleaseID(ctx context.Context, releaseID string) ([]*Track, error) {
	log := t.log.Function("GetByReleaseID")

	releaseUUID, err := uuid.Parse(releaseID)
	if err != nil {
		return nil, log.Err("failed to parse release ID", err, "releaseID", releaseID)
	}

	var tracks []*Track
	if err := t.getDB(ctx).Where("release_id = ?", releaseUUID).Order("position").Find(&tracks).Error; err != nil {
		return nil, log.Err("failed to get tracks by release ID", err, "releaseID", releaseID)
	}

	return tracks, nil
}

func (t *trackRepository) Create(ctx context.Context, track *Track) (*Track, error) {
	log := t.log.Function("Create")

	if err := t.getDB(ctx).Create(track).Error; err != nil {
		return nil, log.Err("failed to create track", err, "track", track)
	}

	log.Info("Track created successfully", "trackID", track.ID, "title", track.Title)
	return track, nil
}

func (t *trackRepository) Update(ctx context.Context, track *Track) error {
	log := t.log.Function("Update")

	if err := t.getDB(ctx).Save(track).Error; err != nil {
		return log.Err("failed to update track", err, "trackID", track.ID)
	}

	log.Info("Track updated successfully", "trackID", track.ID)
	return nil
}

func (t *trackRepository) Delete(ctx context.Context, id string) error {
	log := t.log.Function("Delete")

	trackID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse track ID", err, "id", id)
	}

	if err := t.getDB(ctx).Delete(&Track{}, "id = ?", trackID).Error; err != nil {
		return log.Err("failed to delete track", err, "id", id)
	}

	log.Info("Track deleted successfully", "trackID", trackID)
	return nil
}

func (t *trackRepository) CreateBatch(ctx context.Context, tracks []*Track) error {
	log := t.log.Function("CreateBatch")

	if len(tracks) == 0 {
		return nil
	}

	// Process in batches to avoid memory issues
	for i := 0; i < len(tracks); i += TRACK_BATCH_SIZE {
		end := i + TRACK_BATCH_SIZE
		if end > len(tracks) {
			end = len(tracks)
		}

		batch := tracks[i:end]
		if err := t.getDB(ctx).CreateInBatches(batch, TRACK_BATCH_SIZE).Error; err != nil {
			return log.Err("failed to create track batch", err, "batchStart", i, "batchSize", len(batch))
		}

		log.Info("Track batch created", "batchStart", i, "batchSize", len(batch))
	}

	log.Info("All track batches created successfully", "totalTracks", len(tracks))
	return nil
}

func (t *trackRepository) DeleteByReleaseID(ctx context.Context, releaseID string) error {
	log := t.log.Function("DeleteByReleaseID")

	releaseUUID, err := uuid.Parse(releaseID)
	if err != nil {
		return log.Err("failed to parse release ID", err, "releaseID", releaseID)
	}

	if err := t.getDB(ctx).Where("release_id = ?", releaseUUID).Delete(&Track{}).Error; err != nil {
		return log.Err("failed to delete tracks by release ID", err, "releaseID", releaseID)
	}

	log.Info("Tracks deleted successfully", "releaseID", releaseID)
	return nil
}