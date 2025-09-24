package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReleaseArtistAssociation represents a specific release-artist association pair
type ReleaseArtistAssociation struct {
	ReleaseID int64
	ArtistID  int64
}

type ReleaseRepository interface {
	GetByID(ctx context.Context, tx *gorm.DB, id string) (*Release, error)
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Release, error)
	Create(ctx context.Context, tx *gorm.DB, release *Release) (*Release, error)
	Update(ctx context.Context, tx *gorm.DB, release *Release) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error
	UpsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	GetBatchByDiscogsIDs(ctx context.Context, tx *gorm.DB, discogsIDs []int64) (map[int64]*Release, error)
	InsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	// Association methods
	CreateReleaseArtistAssociations(ctx context.Context, tx *gorm.DB, associations []ReleaseArtistAssociation) error
	AssociateArtists(ctx context.Context, tx *gorm.DB, release *Release, artists []*Artist) error
	AssociateLabels(ctx context.Context, tx *gorm.DB, release *Release, labels []*Label) error
	AssociateGenres(ctx context.Context, tx *gorm.DB, release *Release, genres []*Genre) error
}

type releaseRepository struct {
	log logger.Logger
}

func NewReleaseRepository() ReleaseRepository {
	return &releaseRepository{
		log: logger.New("releaseRepository"),
	}
}

func (r *releaseRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (*Release, error) {
	log := r.log.Function("GetByID")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse release ID", err, "id", id)
	}

	var release Release
	if err := tx.WithContext(ctx).Preload("Labels").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", releaseID).Error; err != nil {
		return nil, log.Err("failed to get release by ID", err, "id", id)
	}

	return &release, nil
}

func (r *releaseRepository) GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Release, error) {
	log := r.log.Function("GetByDiscogsID")

	var release Release
	if err := tx.WithContext(ctx).Preload("Labels").Preload("Master").Preload("Artists").Preload("Genres").First(&release, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get release by Discogs ID", err, "discogsID", discogsID)
	}

	return &release, nil
}

func (r *releaseRepository) Create(ctx context.Context, tx *gorm.DB, release *Release) (*Release, error) {
	log := r.log.Function("Create")

	if err := tx.WithContext(ctx).Create(release).Error; err != nil {
		return nil, log.Err("failed to create release", err, "release", release)
	}

	return release, nil
}

func (r *releaseRepository) Update(ctx context.Context, tx *gorm.DB, release *Release) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(release).Error; err != nil {
		return log.Err("failed to update release", err, "release", release)
	}

	return nil
}

func (r *releaseRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	log := r.log.Function("Delete")

	releaseID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse release ID", err, "id", id)
	}

	if err := tx.WithContext(ctx).Delete(&Release{}, "id = ?", releaseID).Error; err != nil {
		return log.Err("failed to delete release", err, "id", id)
	}

	return nil
}

func (r *releaseRepository) UpsertBatch(
	ctx context.Context,
	tx *gorm.DB,
	releases []*Release,
) error {
	log := r.log.Function("UpsertBatch")

	if len(releases) == 0 {
		return nil
	}

	log.Info("Upserting releases", "count", len(releases))

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "updated_at"}),
	}).Create(&releases).Error; err != nil {
		return log.Err("failed to upsert release batch", err, "count", len(releases))
	}

	log.Info("Successfully upserted releases", "count", len(releases))
	return nil
}

func (r *releaseRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]*Release, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Release), nil
	}

	var releases []*Release
	if err := tx.WithContext(ctx).Where("id IN ?", discogsIDs).Find(&releases).Error; err != nil {
		return nil, log.Err("failed to get releases by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Release, len(releases))
	for _, release := range releases {
		result[release.ID] = release
	}

	log.Info(
		"Retrieved releases by Discogs IDs",
		"requested",
		len(discogsIDs),
		"found",
		len(result),
	)
	return result, nil
}


func (r *releaseRepository) InsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error {
	log := r.log.Function("InsertBatch")

	if len(releases) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(&releases).Error; err != nil {
		return log.Err("failed to insert release batch", err, "count", len(releases))
	}

	return nil
}

func (r *releaseRepository) UpdateBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error {
	log := r.log.Function("UpdateBatch")

	if len(releases) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Save(&releases).Error; err != nil {
		return log.Err("failed to update release batch", err, "count", len(releases))
	}

	return nil
}

func (r *releaseRepository) AssociateArtists(ctx context.Context, tx *gorm.DB, release *Release, artists []*Artist) error {
	log := r.log.Function("AssociateArtists")

	if len(artists) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Model(release).Association("Artists").Append(artists); err != nil {
		return log.Err("failed to associate artists with release", err,
			"releaseID", release.ID,
			"artistCount", len(artists))
	}

	return nil
}

func (r *releaseRepository) AssociateLabels(ctx context.Context, tx *gorm.DB, release *Release, labels []*Label) error {
	log := r.log.Function("AssociateLabels")

	if len(labels) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Model(release).Association("Labels").Append(labels); err != nil {
		return log.Err("failed to associate labels with release", err,
			"releaseID", release.ID,
			"labelCount", len(labels))
	}

	return nil
}

func (r *releaseRepository) AssociateGenres(ctx context.Context, tx *gorm.DB, release *Release, genres []*Genre) error {
	log := r.log.Function("AssociateGenres")

	if len(genres) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Model(release).Association("Genres").Append(genres); err != nil {
		return log.Err("failed to associate genres with release", err,
			"releaseID", release.ID,
			"genreCount", len(genres))
	}

	return nil
}

// CreateReleaseArtistAssociations creates specific release-artist association pairs
func (r *releaseRepository) CreateReleaseArtistAssociations(
	ctx context.Context,
	tx *gorm.DB,
	associations []ReleaseArtistAssociation,
) error {
	log := r.log.Function("CreateReleaseArtistAssociations")

	if len(associations) == 0 {
		return nil
	}

	// Prepare association pairs for bulk insert with ordered processing to prevent deadlocks
	releaseIDs := make([]int64, len(associations))
	artistIDs := make([]int64, len(associations))

	for i, assoc := range associations {
		releaseIDs[i] = assoc.ReleaseID
		artistIDs[i] = assoc.ArtistID
	}

	// Insert exact association pairs with ordering to prevent deadlocks
	query := `
		INSERT INTO release_artists (release_id, artist_id)
		SELECT release_id, artist_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(release_id, artist_id)
		ORDER BY release_id, artist_id
		ON CONFLICT (release_id, artist_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, releaseIDs, artistIDs)
	if result.Error != nil {
		return log.Err("failed to create release-artist associations", result.Error,
			"associationCount", len(associations))
	}

	log.Info("Created release-artist associations",
		"associationCount", len(associations),
		"rowsAffected", result.RowsAffected)

	return nil
}
