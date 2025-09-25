package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReleaseArtistAssociation represents a specific release-artist association pair
type ReleaseArtistAssociation struct {
	ReleaseID int64
	ArtistID  int64
}

type ReleaseRepository interface {
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Release, error)
	UpsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	// Association methods
	CreateReleaseArtistAssociations(ctx context.Context, tx *gorm.DB, associations []ReleaseArtistAssociation) error
	UpsertReleaseArtistAssociationsBatch(ctx context.Context, tx *gorm.DB, associations []*[]ReleaseArtistAssociation) error
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
	// Only insert associations where both release_id and artist_id exist in their respective tables
	query := `
		INSERT INTO release_artists (release_id, artist_id)
		SELECT t.release_id, t.artist_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(release_id, artist_id)
		INNER JOIN releases r ON r.id = t.release_id
		INNER JOIN artists a ON a.id = t.artist_id
		ORDER BY t.release_id, t.artist_id
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

// UpsertReleaseArtistAssociationsBatch processes batches of association arrays for the EntityProcessor pattern
func (r *releaseRepository) UpsertReleaseArtistAssociationsBatch(
	ctx context.Context,
	tx *gorm.DB,
	associationBatches []*[]ReleaseArtistAssociation,
) error {
	log := r.log.Function("UpsertReleaseArtistAssociationsBatch")

	if len(associationBatches) == 0 {
		return nil
	}

	// Flatten all association batches into a single slice
	var allAssociations []ReleaseArtistAssociation
	for _, batch := range associationBatches {
		if batch != nil && len(*batch) > 0 {
			allAssociations = append(allAssociations, *batch...)
		}
	}

	if len(allAssociations) == 0 {
		return nil
	}

	// Use the existing CreateReleaseArtistAssociations method which has foreign key validation
	if err := r.CreateReleaseArtistAssociations(ctx, tx, allAssociations); err != nil {
		return log.Err("failed to create release-artist associations batch", err,
			"totalAssociations", len(allAssociations))
	}

	log.Info("Created release-artist associations batch",
		"totalAssociations", len(allAssociations),
		"batchCount", len(associationBatches))

	return nil
}
