package repositories

import (
	"context"
	"time"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReleaseArtistAssociation struct {
	ReleaseID int64
	ArtistID  int64
}

type ReleaseLabelAssociation struct {
	ReleaseID int64
	LabelID   int64
}

type ReleaseGenreAssociation struct {
	ReleaseID int64
	GenreID   int64
}

type ReleaseImageUpdate struct {
	ReleaseID  int64
	Thumb      *string
	CoverImage *string
}

type ReleaseRepository interface {
	GetByDiscogsID(ctx context.Context, tx *gorm.DB, discogsID int64) (*Release, error)
	UpsertBatch(ctx context.Context, tx *gorm.DB, releases []*Release) error
	CheckReleaseExistence(
		ctx context.Context,
		tx *gorm.DB,
		releaseIDs []int64,
	) (existing []int64, missing []int64, err error)
	UpdateReleaseImages(ctx context.Context, tx *gorm.DB, updates []ReleaseImageUpdate) error
	CreateReleaseArtistAssociations(
		ctx context.Context,
		tx *gorm.DB,
		associations []ReleaseArtistAssociation,
	) error
	UpsertReleaseArtistAssociationsBatch(
		ctx context.Context,
		tx *gorm.DB,
		associations []*[]ReleaseArtistAssociation,
	) error
	CreateReleaseLabelAssociations(
		ctx context.Context,
		tx *gorm.DB,
		associations []ReleaseLabelAssociation,
	) error
	UpsertReleaseLabelAssociationsBatch(
		ctx context.Context,
		tx *gorm.DB,
		associations []*[]ReleaseLabelAssociation,
	) error
	CreateReleaseGenreAssociations(
		ctx context.Context,
		tx *gorm.DB,
		associations []ReleaseGenreAssociation,
	) error
	UpsertReleaseGenreAssociationsBatch(
		ctx context.Context,
		tx *gorm.DB,
		associations []*[]ReleaseGenreAssociation,
	) error
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

func (r *releaseRepository) GetByDiscogsID(
	ctx context.Context,
	tx *gorm.DB,
	discogsID int64,
) (*Release, error) {
	var release Release
	if err := tx.WithContext(ctx).
		Preload("Labels").
		Preload("Master").
		Preload("Artists").
		Preload("Genres").
		First(&release,
			&BaseDiscogModel{
				ID: discogsID,
			}).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, r.log.Function("GetByDiscogsID").
			Err("failed to get release by Discogs ID", err, "discogsID", discogsID)
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
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"title",
			"tracks_json",
			"images_json",
			"videos_json",
			"format_details_json",
			"total_duration",
			"updated_at",
		}),
	}).Create(&releases).Error; err != nil {
		return log.Err("failed to upsert release batch", err, "count", len(releases))
	}

	log.Info("Successfully upserted releases", "count", len(releases))
	return nil
}

func (r *releaseRepository) AssociateArtists(
	ctx context.Context,
	tx *gorm.DB,
	release *Release,
	artists []*Artist,
) error {
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

func (r *releaseRepository) AssociateLabels(
	ctx context.Context,
	tx *gorm.DB,
	release *Release,
	labels []*Label,
) error {
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

func (r *releaseRepository) AssociateGenres(
	ctx context.Context,
	tx *gorm.DB,
	release *Release,
	genres []*Genre,
) error {
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

func (r *releaseRepository) CreateReleaseArtistAssociations(
	ctx context.Context,
	tx *gorm.DB,
	associations []ReleaseArtistAssociation,
) error {
	log := r.log.Function("CreateReleaseArtistAssociations")

	if len(associations) == 0 {
		return nil
	}

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

func (r *releaseRepository) CreateReleaseLabelAssociations(
	ctx context.Context,
	tx *gorm.DB,
	associations []ReleaseLabelAssociation,
) error {
	log := r.log.Function("CreateReleaseLabelAssociations")

	if len(associations) == 0 {
		return nil
	}

	releaseIDs := make([]int64, len(associations))
	labelIDs := make([]int64, len(associations))

	for i, assoc := range associations {
		releaseIDs[i] = assoc.ReleaseID
		labelIDs[i] = assoc.LabelID
	}

	// Insert exact association pairs with ordering to prevent deadlocks
	// Only insert associations where both release_id and label_id exist in their respective tables
	query := `
		INSERT INTO release_labels (release_id, label_id)
		SELECT t.release_id, t.label_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(release_id, label_id)
		INNER JOIN releases r ON r.id = t.release_id
		INNER JOIN labels l ON l.id = t.label_id
		ORDER BY t.release_id, t.label_id
		ON CONFLICT (release_id, label_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, releaseIDs, labelIDs)
	if result.Error != nil {
		return log.Err("failed to create release-label associations", result.Error,
			"associationCount", len(associations))
	}

	log.Info("Created release-label associations",
		"associationCount", len(associations),
		"rowsAffected", result.RowsAffected)

	return nil
}

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

	if err := r.CreateReleaseArtistAssociations(ctx, tx, allAssociations); err != nil {
		return log.Err("failed to create release-artist associations batch", err,
			"totalAssociations", len(allAssociations))
	}

	log.Info("Created release-artist associations batch",
		"totalAssociations", len(allAssociations),
		"batchCount", len(associationBatches))

	return nil
}

func (r *releaseRepository) UpsertReleaseLabelAssociationsBatch(
	ctx context.Context,
	tx *gorm.DB,
	associationBatches []*[]ReleaseLabelAssociation,
) error {
	log := r.log.Function("UpsertReleaseLabelAssociationsBatch")

	if len(associationBatches) == 0 {
		return nil
	}

	var allAssociations []ReleaseLabelAssociation
	for _, batch := range associationBatches {
		if batch != nil && len(*batch) > 0 {
			allAssociations = append(allAssociations, *batch...)
		}
	}

	if len(allAssociations) == 0 {
		return nil
	}

	if err := r.CreateReleaseLabelAssociations(ctx, tx, allAssociations); err != nil {
		return log.Err("failed to create release-label associations batch", err,
			"totalAssociations", len(allAssociations))
	}

	log.Info("Created release-label associations batch",
		"totalAssociations", len(allAssociations),
		"batchCount", len(associationBatches))

	return nil
}

func (r *releaseRepository) CreateReleaseGenreAssociations(
	ctx context.Context,
	tx *gorm.DB,
	associations []ReleaseGenreAssociation,
) error {
	log := r.log.Function("CreateReleaseGenreAssociations")

	if len(associations) == 0 {
		return nil
	}

	releaseIDs := make([]int64, len(associations))
	genreIDs := make([]int64, len(associations))

	for i, assoc := range associations {
		releaseIDs[i] = assoc.ReleaseID
		genreIDs[i] = assoc.GenreID
	}

	// Insert exact association pairs with ordering to prevent deadlocks
	// Only insert associations where both release_id and genre_id exist in their respective tables
	query := `
		INSERT INTO release_genres (release_id, genre_id)
		SELECT t.release_id, t.genre_id
		FROM unnest($1::bigint[], $2::bigint[]) AS t(release_id, genre_id)
		INNER JOIN releases r ON r.id = t.release_id
		INNER JOIN genres g ON g.id = t.genre_id
		ORDER BY t.release_id, t.genre_id
		ON CONFLICT (release_id, genre_id) DO NOTHING
	`

	result := tx.WithContext(ctx).Exec(query, releaseIDs, genreIDs)
	if result.Error != nil {
		return log.Err("failed to create release-genre associations", result.Error,
			"associationCount", len(associations))
	}

	log.Info("Created release-genre associations",
		"associationCount", len(associations),
		"rowsAffected", result.RowsAffected)

	return nil
}

func (r *releaseRepository) UpsertReleaseGenreAssociationsBatch(
	ctx context.Context,
	tx *gorm.DB,
	associationBatches []*[]ReleaseGenreAssociation,
) error {
	log := r.log.Function("UpsertReleaseGenreAssociationsBatch")

	if len(associationBatches) == 0 {
		return nil
	}

	var allAssociations []ReleaseGenreAssociation
	for _, batch := range associationBatches {
		if batch != nil && len(*batch) > 0 {
			allAssociations = append(allAssociations, *batch...)
		}
	}

	if len(allAssociations) == 0 {
		return nil
	}

	if err := r.CreateReleaseGenreAssociations(ctx, tx, allAssociations); err != nil {
		return log.Err("failed to create release-genre associations batch", err,
			"totalAssociations", len(allAssociations))
	}

	log.Info("Created release-genre associations batch",
		"totalAssociations", len(allAssociations),
		"batchCount", len(associationBatches))

	return nil
}

func (r *releaseRepository) CheckReleaseExistence(
	ctx context.Context,
	tx *gorm.DB,
	releaseIDs []int64,
) (existing []int64, missing []int64, err error) {
	log := r.log.Function("CheckReleaseExistence")

	if len(releaseIDs) == 0 {
		return []int64{}, []int64{}, nil
	}

	var existingReleases []int64
	if err := tx.WithContext(ctx).Model(&Release{}).
		Where("id IN ?", releaseIDs).
		Pluck("id", &existingReleases).Error; err != nil {
		return nil, nil, log.Err(
			"failed to check release existence",
			err,
			"releaseCount",
			len(releaseIDs),
		)
	}

	existingMap := make(map[int64]bool, len(existingReleases))
	for _, id := range existingReleases {
		existingMap[id] = true
	}

	existing = make([]int64, 0, len(existingReleases))
	missing = make([]int64, 0)

	for _, id := range releaseIDs {
		if existingMap[id] {
			existing = append(existing, id)
		} else {
			missing = append(missing, id)
		}
	}

	log.Info("Checked release existence",
		"totalReleases", len(releaseIDs),
		"existing", len(existing),
		"missing", len(missing))

	return existing, missing, nil
}

func (r *releaseRepository) UpdateReleaseImages(
	ctx context.Context,
	tx *gorm.DB,
	updates []ReleaseImageUpdate,
) error {
	log := r.log.Function("UpdateReleaseImages")

	if len(updates) == 0 {
		return nil
	}

	var totalRowsAffected int64
	now := time.Now()

	for _, update := range updates {
		updateData := map[string]any{
			"updated_at": now,
		}

		if update.Thumb != nil {
			updateData["thumb"] = *update.Thumb
		}

		if update.CoverImage != nil {
			updateData["cover_image"] = *update.CoverImage
		}

		if len(updateData) > 1 {
			result := tx.WithContext(ctx).
				Model(&Release{}).
				Where("id = ?", update.ReleaseID).
				Updates(updateData)

			if result.Error != nil {
				return log.Err("failed to update release images", result.Error,
					"releaseID", update.ReleaseID, "updateCount", len(updates))
			}

			totalRowsAffected += result.RowsAffected
		}
	}

	log.Info("Updated release images",
		"updateCount", len(updates),
		"rowsAffected", totalRowsAffected)

	return nil
}
