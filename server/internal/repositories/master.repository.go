package repositories

import (
	"context"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MasterArtistAssociation represents a specific master-artist association pair
type MasterArtistAssociation struct {
	MasterID int64
	ArtistID int64
}

type MasterRepository interface {
	GetByID(ctx context.Context, tx *gorm.DB, id string) (*Master, error)
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Master, error)
	Create(ctx context.Context, tx *gorm.DB, master *Master) (*Master, error)
	Update(ctx context.Context, tx *gorm.DB, master *Master) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error
	UpsertBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error
	GetBatchByDiscogsIDs(ctx context.Context, tx *gorm.DB, discogsIDs []int64) (map[int64]*Master, error)
	GetHashesByDiscogsIDs(ctx context.Context, tx *gorm.DB, discogsIDs []int64) (map[int64]string, error)
	InsertBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error
	UpdateBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error
	// Association methods
	CreateMasterArtistAssociations(
		ctx context.Context,
		tx *gorm.DB,
		associations []MasterArtistAssociation,
	) error
	CreateMasterGenreAssociations(
		ctx context.Context,
		tx *gorm.DB,
		masterIDs []int64,
		genreNames []string,
	) error
}

type masterRepository struct {
	log logger.Logger
}

func NewMasterRepository() MasterRepository {
	return &masterRepository{
		log: logger.New("masterRepository"),
	}
}

func (r *masterRepository) GetByID(ctx context.Context, tx *gorm.DB, id string) (*Master, error) {
	log := r.log.Function("GetByID")

	masterID, err := uuid.Parse(id)
	if err != nil {
		return nil, log.Err("failed to parse master ID", err, "id", id)
	}

	var master Master
	if err := tx.WithContext(ctx).First(&master, "id = ?", masterID).Error; err != nil {
		return nil, log.Err("failed to get master by ID", err, "id", id)
	}

	return &master, nil
}

func (r *masterRepository) GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Master, error) {
	log := r.log.Function("GetByDiscogsID")

	var master Master
	if err := tx.WithContext(ctx).First(&master, "id = ?", discogsID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, log.Err("failed to get master by Discogs ID", err, "discogsID", discogsID)
	}

	return &master, nil
}

func (r *masterRepository) Create(ctx context.Context, tx *gorm.DB, master *Master) (*Master, error) {
	log := r.log.Function("Create")

	if err := tx.WithContext(ctx).Create(master).Error; err != nil {
		return nil, log.Err("failed to create master", err, "master", master)
	}

	return master, nil
}

func (r *masterRepository) Update(ctx context.Context, tx *gorm.DB, master *Master) error {
	log := r.log.Function("Update")

	if err := tx.WithContext(ctx).Save(master).Error; err != nil {
		return log.Err("failed to update master", err, "master", master)
	}

	return nil
}

func (r *masterRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	log := r.log.Function("Delete")

	masterID, err := uuid.Parse(id)
	if err != nil {
		return log.Err("failed to parse master ID", err, "id", id)
	}

	if err := tx.WithContext(ctx).Delete(&Master{}, "id = ?", masterID).Error; err != nil {
		return log.Err("failed to delete master", err, "id", id)
	}

	return nil
}

func (r *masterRepository) UpsertBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error {
	log := r.log.Function("UpsertBatch")

	if len(masters) == 0 {
		return nil
	}

	discogsIDs := make([]int64, len(masters))
	for i, master := range masters {
		discogsIDs[i] = master.ID
	}

	existingHashes, err := r.GetHashesByDiscogsIDs(ctx, tx, discogsIDs)
	if err != nil {
		return log.Err("failed to get existing hashes", err, "count", len(discogsIDs))
	}

	hashableRecords := make([]utils.DiscogsHashable, len(masters))
	for i, master := range masters {
		hashableRecords[i] = master
	}

	categories := utils.CategorizeRecordsByHash(hashableRecords, existingHashes)
	if len(categories.Insert) > 0 {
		insertMasters := make([]*Master, len(categories.Insert))
		for i, record := range categories.Insert {
			insertMasters[i] = record.(*Master)
		}
		err = r.InsertBatch(ctx, tx, insertMasters)
		if err != nil {
			return log.Err("failed to insert master batch", err, "count", len(insertMasters))
		}
	}

	if len(categories.Update) > 0 {
		updateMasters := make([]*Master, len(categories.Update))
		for i, record := range categories.Update {
			updateMasters[i] = record.(*Master)
		}
		err = r.UpdateBatch(ctx, tx, updateMasters)
		if err != nil {
			return log.Err(
				"failed to update master batch",
				err,
				"count",
				len(updateMasters),
			)
		}
	}

	return nil
}

func (r *masterRepository) GetBatchByDiscogsIDs(
	ctx context.Context,
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]*Master, error) {
	log := r.log.Function("GetBatchByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]*Master), nil
	}

	var masters []*Master
	if err := tx.WithContext(ctx).Where("id IN ?", discogsIDs).Find(&masters).Error; err != nil {
		return nil, log.Err("failed to get masters by Discogs IDs", err, "count", len(discogsIDs))
	}

	// Convert to map for O(1) lookup
	result := make(map[int64]*Master, len(masters))
	for _, master := range masters {
		result[master.ID] = master
	}

	return result, nil
}

func (r *masterRepository) GetHashesByDiscogsIDs(
	ctx context.Context,
	tx *gorm.DB,
	discogsIDs []int64,
) (map[int64]string, error) {
	log := r.log.Function("GetHashesByDiscogsIDs")

	if len(discogsIDs) == 0 {
		return make(map[int64]string), nil
	}

	var masters []struct {
		ID   int64  `json:"discogsId"`
		ContentHash string `json:"contentHash"`
	}

	if err := tx.WithContext(ctx).
		Model(&Master{}).
		Select("id, content_hash").
		Where("id IN ?", discogsIDs).
		Find(&masters).Error; err != nil {
		return nil, log.Err(
			"failed to get master hashes by Discogs IDs",
			err,
			"count",
			len(discogsIDs),
		)
	}

	result := make(map[int64]string, len(masters))
	for _, master := range masters {
		result[master.ID] = master.ContentHash
	}

	return result, nil
}

func (r *masterRepository) InsertBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error {
	log := r.log.Function("InsertBatch")

	if len(masters) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Create(&masters).Error; err != nil {
		return log.Err("failed to insert master batch", err, "count", len(masters))
	}

	return nil
}

func (r *masterRepository) UpdateBatch(ctx context.Context, tx *gorm.DB, masters []*Master) error {
	log := r.log.Function("UpdateBatch")

	if len(masters) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Save(&masters).Error; err != nil {
		return log.Err("failed to update master batch", err, "count", len(masters))
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
	query := `
		INSERT INTO master_artists (master_id, artist_id)
		SELECT master_id, artist_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(master_id, artist_id)
		ORDER BY master_id, artist_id
		ON CONFLICT (master_id, artist_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, masterIDs, artistIDs)
	if result.Error != nil {
		return log.Err("failed to create master-artist associations", result.Error,
			"associationCount", len(associations))
	}

	return nil
}

// CreateMasterGenreAssociations creates many-to-many associations between masters and genres
func (r *masterRepository) CreateMasterGenreAssociations(
	ctx context.Context,
	tx *gorm.DB,
	masterIDs []int64,
	genreNames []string,
) error {
	log := r.log.Function("CreateMasterGenreAssociations")

	if len(masterIDs) == 0 || len(genreNames) == 0 {
		return nil
	}

	// Build cross-product associations using genre names with ON CONFLICT DO NOTHING
	query := `
		INSERT INTO master_genres (master_id, genre_id)
		SELECT m.id, g.id
		FROM unnest($1::bigint[]) AS m(id)
		CROSS JOIN genres g
		WHERE g.name = ANY($2::text[])
		ON CONFLICT (master_id, genre_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, masterIDs, genreNames)
	if result.Error != nil {
		return log.Err("failed to create master-genre associations", result.Error,
			"masterCount", len(masterIDs), "genreCount", len(genreNames))
	}

	return nil
}
