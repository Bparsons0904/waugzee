package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MasterArtistAssociation represents a specific master-artist association pair
type MasterArtistAssociation struct {
	MasterID int64
	ArtistID int64
}

// MasterGenreAssociation represents a specific master-genre association pair
type MasterGenreAssociation struct {
	MasterID int64
	GenreID  int64
}

type MasterRepository interface {
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Master, error)
	UpsertBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error
	// Association methods
	CreateMasterArtistAssociations(
		ctx context.Context,
		tx *gorm.DB,
		associations []MasterArtistAssociation,
	) error
	UpsertMasterArtistAssociationsBatch(
		ctx context.Context,
		tx *gorm.DB,
		associations []*[]MasterArtistAssociation,
	) error
	CreateMasterGenreAssociations(
		ctx context.Context,
		tx *gorm.DB,
		associations []MasterGenreAssociation,
	) error
	UpsertMasterGenreAssociationsBatch(
		ctx context.Context,
		tx *gorm.DB,
		associations []*[]MasterGenreAssociation,
	) error
	// Individual association methods
	AssociateArtists(ctx context.Context, tx *gorm.DB, master *Master, artists []*Artist) error
	AssociateGenres(ctx context.Context, tx *gorm.DB, master *Master, genres []*Genre) error
}

type masterRepository struct {
	log logger.Logger
}

func NewMasterRepository() MasterRepository {
	return &masterRepository{
		log: logger.New("masterRepository"),
	}
}


func (r *masterRepository) GetByDiscogsID(
	ctx context.Context,
	tx *gorm.DB,
	discogsID int64,
) (*Master, error) {
	log := r.log.Function("GetByDiscogsID")

	master, err := gorm.G[*Master](tx).Where(BaseDiscogModel{ID: discogsID}).First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get master by Discogs ID", err, "discogsID", discogsID)
	}

	return master, nil
}



func (r *masterRepository) UpsertBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error {
	log := r.log.Function("UpsertBatch")

	if len(masters) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "updated_at"}),
	}).Create(&masters).Error; err != nil {
		return log.Err("failed to upsert master batch", err, "count", len(masters))
	}

	return nil
}



// CreateMasterArtistAssociations creates specific master-artist association pairs
func (r *masterRepository) CreateMasterArtistAssociations(
	ctx context.Context,
	tx *gorm.DB,
	associations []MasterArtistAssociation,
) error {
	log := r.log.Function("CreateMasterArtistAssociations")

	if len(associations) == 0 {
		return nil
	}

	// Prepare association pairs for bulk insert with ordered processing to prevent deadlocks
	masterIDs := make([]int64, len(associations))
	artistIDs := make([]int64, len(associations))

	for i, assoc := range associations {
		masterIDs[i] = assoc.MasterID
		artistIDs[i] = assoc.ArtistID
	}

	// Insert exact association pairs with ordering to prevent deadlocks
	// Only insert associations where both master_id and artist_id exist in their respective tables
	query := `
		INSERT INTO master_artists (master_id, artist_id)
		SELECT t.master_id, t.artist_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(master_id, artist_id)
		INNER JOIN masters m ON m.id = t.master_id
		INNER JOIN artists a ON a.id = t.artist_id
		ORDER BY t.master_id, t.artist_id
		ON CONFLICT (master_id, artist_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, masterIDs, artistIDs)
	if result.Error != nil {
		return log.Err("failed to create master-artist associations", result.Error,
			"associationCount", len(associations))
	}

	return nil
}

// UpsertMasterArtistAssociationsBatch processes batches of association arrays for the EntityProcessor pattern
func (r *masterRepository) UpsertMasterArtistAssociationsBatch(
	ctx context.Context,
	tx *gorm.DB,
	associationBatches []*[]MasterArtistAssociation,
) error {
	log := r.log.Function("UpsertMasterArtistAssociationsBatch")

	if len(associationBatches) == 0 {
		return nil
	}

	// Flatten all association batches into a single slice
	var allAssociations []MasterArtistAssociation
	for _, batch := range associationBatches {
		if batch != nil && len(*batch) > 0 {
			allAssociations = append(allAssociations, *batch...)
		}
	}

	if len(allAssociations) == 0 {
		return nil
	}

	// Use the existing CreateMasterArtistAssociations method which has foreign key validation
	if err := r.CreateMasterArtistAssociations(ctx, tx, allAssociations); err != nil {
		return log.Err("failed to create master-artist associations batch", err,
			"totalAssociations", len(allAssociations))
	}

	log.Info("Created master-artist associations batch",
		"totalAssociations", len(allAssociations),
		"batchCount", len(associationBatches))

	return nil
}


func (r *masterRepository) AssociateArtists(
	ctx context.Context,
	tx *gorm.DB,
	master *Master,
	artists []*Artist,
) error {
	log := r.log.Function("AssociateArtists")

	if len(artists) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Model(master).Association("Artists").Append(artists); err != nil {
		return log.Err("failed to associate artists with master", err,
			"masterID", master.ID,
			"artistCount", len(artists))
	}

	return nil
}

func (r *masterRepository) AssociateGenres(
	ctx context.Context,
	tx *gorm.DB,
	master *Master,
	genres []*Genre,
) error {
	log := r.log.Function("AssociateGenres")

	if len(genres) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Model(master).Association("Genres").Append(genres); err != nil {
		return log.Err("failed to associate genres with master", err,
			"masterID", master.ID,
			"genreCount", len(genres))
	}

	return nil
}

// CreateMasterGenreAssociations creates specific master-genre association pairs
func (r *masterRepository) CreateMasterGenreAssociations(
	ctx context.Context,
	tx *gorm.DB,
	associations []MasterGenreAssociation,
) error {
	log := r.log.Function("CreateMasterGenreAssociations")

	if len(associations) == 0 {
		return nil
	}

	// Prepare association pairs for bulk insert with ordered processing to prevent deadlocks
	masterIDs := make([]int64, len(associations))
	genreIDs := make([]int64, len(associations))

	for i, assoc := range associations {
		masterIDs[i] = assoc.MasterID
		genreIDs[i] = assoc.GenreID
	}

	// Insert exact association pairs with ordering to prevent deadlocks
	// Only insert associations where both master_id and genre_id exist in their respective tables
	query := `
		INSERT INTO master_genres (master_id, genre_id)
		SELECT t.master_id, t.genre_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(master_id, genre_id)
		INNER JOIN masters m ON m.id = t.master_id
		INNER JOIN genres g ON g.id = t.genre_id
		ORDER BY t.master_id, t.genre_id
		ON CONFLICT (master_id, genre_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, masterIDs, genreIDs)
	if result.Error != nil {
		return log.Err("failed to create master-genre associations", result.Error,
			"associationCount", len(associations))
	}

	log.Info("Created master-genre associations",
		"associationCount", len(associations),
		"rowsAffected", result.RowsAffected)

	return nil
}

// UpsertMasterGenreAssociationsBatch processes batches of association arrays for the EntityProcessor pattern
func (r *masterRepository) UpsertMasterGenreAssociationsBatch(
	ctx context.Context,
	tx *gorm.DB,
	associationBatches []*[]MasterGenreAssociation,
) error {
	log := r.log.Function("UpsertMasterGenreAssociationsBatch")

	if len(associationBatches) == 0 {
		return nil
	}

	// Flatten all association batches into a single slice
	var allAssociations []MasterGenreAssociation
	for _, batch := range associationBatches {
		if batch != nil && len(*batch) > 0 {
			allAssociations = append(allAssociations, *batch...)
		}
	}

	if len(allAssociations) == 0 {
		return nil
	}

	// Use the existing CreateMasterGenreAssociations method which has foreign key validation
	if err := r.CreateMasterGenreAssociations(ctx, tx, allAssociations); err != nil {
		return log.Err("failed to create master-genre associations batch", err,
			"totalAssociations", len(allAssociations))
	}

	log.Info("Created master-genre associations batch",
		"totalAssociations", len(allAssociations),
		"batchCount", len(associationBatches))

	return nil
}
