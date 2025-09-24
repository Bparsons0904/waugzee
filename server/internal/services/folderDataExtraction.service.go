package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"waugzee/internal/logger"
	. "waugzee/internal/models"
	"waugzee/internal/repositories"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type FolderDataExtractionService struct {
	log   logger.Logger
	repos repositories.Repository
}

func NewFolderDataExtractionService(repos repositories.Repository) *FolderDataExtractionService {
	return &FolderDataExtractionService{
		log:   logger.New("FolderDataExtractionService"),
		repos: repos,
	}
}

// ExtractBasicInformation processes basic_information from folder releases and creates/updates
// minimal Artist, Master, Release, Label, and Genre records
func (f *FolderDataExtractionService) ExtractBasicInformation(
	ctx context.Context,
	tx *gorm.DB,
	folderReleases []DiscogsFolderReleaseItem,
) error {
	log := f.log.Function("ExtractBasicInformation")

	if len(folderReleases) == 0 {
		return nil
	}

	// Collect all entities to process
	artists := make([]*Artist, 0)
	releases := make([]*Release, 0)
	labels := make([]*Label, 0)
	genres := make([]*Genre, 0)
	masters := make([]*Master, 0)

	// Maps to track processed entities by ID/name to avoid duplicates
	processedArtists := make(map[int64]bool)
	processedReleases := make(map[int64]bool)
	processedLabels := make(map[int64]bool)
	processedGenres := make(map[string]bool)
	processedMasters := make(map[int64]bool)

	// Extract data from each folder release
	for _, folderRelease := range folderReleases {
		basicInfo := folderRelease.BasicInformation

		// Extract Release
		if basicInfo.ID > 0 && !processedReleases[basicInfo.ID] {
			release := &Release{
				BaseDiscogModel: BaseDiscogModel{
					ID: basicInfo.ID,
				},
				Title:      basicInfo.Title,
				Format:     FormatVinyl, // Default to vinyl for folder releases
				LastSynced: nil,         // Set to null for newly created records
			}

			// Set year if available
			if basicInfo.Year > 0 {
				release.Year = &basicInfo.Year
			}

			// Set thumbnail and cover image if available
			if basicInfo.Thumb != "" {
				release.Thumb = &basicInfo.Thumb
			}
			if basicInfo.CoverImage != "" {
				release.CoverImage = &basicInfo.CoverImage
			}

			// Set ResourceURL
			if basicInfo.ResourceURL != "" {
				release.ResourceURL = &basicInfo.ResourceURL
			}

			// Set Master ID if available
			// if basicInfo.MasterID > 0 {
			// 	release.MasterID = &basicInfo.MasterID
			// }

			// Store genres and styles in Data column using the new Data struct
			if len(basicInfo.Genres) > 0 || len(basicInfo.Styles) > 0 {
				dataStruct := Data{
					Genres: basicInfo.Genres,
					Styles: basicInfo.Styles,
				}
				if dataJSON, err := json.Marshal(dataStruct); err == nil {
					release.Data = datatypes.JSON(dataJSON)
				}
			}

			releases = append(releases, release)
			processedReleases[basicInfo.ID] = true
		}

		// Extract Master
		if basicInfo.MasterID > 0 && !processedMasters[basicInfo.MasterID] {
			master := &Master{
				BaseDiscogModel: BaseDiscogModel{
					ID: basicInfo.MasterID,
				},
				Title:      basicInfo.Title, // Use release title as master title
				LastSynced: nil,             // Set to null for newly created records
			}

			// Set year if available
			if basicInfo.Year > 0 {
				master.Year = &basicInfo.Year
			}

			// Set master URL if available
			if basicInfo.MasterURL != "" {
				master.MainReleaseResourceURL = &basicInfo.MasterURL
			}

			masters = append(masters, master)
			processedMasters[basicInfo.MasterID] = true
		}

		// Extract Artists
		for _, artist := range basicInfo.Artists {
			if artist.ID > 0 && !processedArtists[artist.ID] {
				artistRecord := &Artist{
					BaseDiscogModel: BaseDiscogModel{
						ID: artist.ID,
					},
					Name:    artist.Name,
					Profile: "", // Empty profile for basic information
				}

				// Set ResourceURL if available
				if artist.ResourceURL != "" {
					artistRecord.ResourceURL = artist.ResourceURL
				}

				artists = append(artists, artistRecord)
				processedArtists[artist.ID] = true
			}
		}

		// Extract Labels
		for _, label := range basicInfo.Labels {
			if label.ID > 0 && !processedLabels[label.ID] {
				labelRecord := &Label{
					BaseDiscogModel: BaseDiscogModel{
						ID: label.ID,
					},
					Name: label.Name,
				}

				// Set ResourceURL if available
				if label.ResourceURL != "" {
					labelRecord.ResourceURL = label.ResourceURL
				}

				labels = append(labels, labelRecord)
				processedLabels[label.ID] = true
			}
		}

		// Extract Genres
		for _, genreName := range basicInfo.Genres {
			if genreName != "" && !processedGenres[genreName] {
				genre := &Genre{
					Name: genreName,
				}
				genres = append(genres, genre)
				processedGenres[genreName] = true
			}
		}

		// Note: Styles are not extracted as separate genres for now
		// They could be processed as sub-genres in the future if needed
	}

	// Use batch operations for efficient database operations
	if len(artists) > 0 {
		if err := f.repos.Artist.UpsertBatch(ctx, tx, artists); err != nil {
			return log.Err("failed to upsert artists", err)
		}
	}

	if len(labels) > 0 {
		if err := f.repos.Label.UpsertBatch(ctx, tx, labels); err != nil {
			return log.Err("failed to upsert labels", err)
		}
	}

	if len(genres) > 0 {
		if err := f.repos.Genre.UpsertBatch(ctx, tx, genres); err != nil {
			return log.Err("failed to upsert genres", err)
		}
	}

	if len(masters) > 0 {
		if err := f.repos.Master.UpsertBatch(ctx, tx, masters); err != nil {
			return log.Err("failed to upsert masters", err)
		}
	}

	if len(releases) > 0 {
		if err := f.repos.Release.UpsertBatch(ctx, tx, releases); err != nil {
			return log.Err("failed to upsert releases", err)
		}
	}

	// Handle many-to-many relationships
	if err := f.createReleaseAssociations(ctx, tx, folderReleases); err != nil {
		return log.Err("failed to create release associations", err)
	}

	// Handle master associations (artists and genres)
	if err := f.createMasterAssociations(ctx, tx, folderReleases); err != nil {
		return log.Err("failed to create master associations", err)
	}

	return nil
}

// createReleaseAssociations handles the many-to-many relationships between releases and other entities
func (f *FolderDataExtractionService) createReleaseAssociations(
	ctx context.Context,
	tx *gorm.DB,
	folderReleases []DiscogsFolderReleaseItem,
) error {
	log := f.log.Function("createReleaseAssociations")

	for _, folderRelease := range folderReleases {
		basicInfo := folderRelease.BasicInformation
		if basicInfo.ID == 0 {
			continue
		}

		// Get the release record
		release, err := f.repos.Release.GetByDiscogsID(ctx, tx, basicInfo.ID)
		if err != nil || release == nil {
			log.Warn("Release not found for associations", "releaseID", basicInfo.ID)
			continue
		}

		// Associate artists
		artists := make([]*Artist, 0)
		for _, artistInfo := range basicInfo.Artists {
			if artistInfo.ID > 0 {
				artist, err := f.repos.Artist.GetByDiscogsID(ctx, tx, artistInfo.ID)
				if err != nil || artist == nil {
					log.Warn("Artist not found for association", "artistID", artistInfo.ID)
					continue
				}
				artists = append(artists, artist)
			}
		}

		if len(artists) > 0 {
			if err := f.repos.Release.AssociateArtists(ctx, tx, release, artists); err != nil {
				log.Warn("Failed to associate artists with release",
					"releaseID", release.ID,
					"artistCount", len(artists),
					"error", err)
			}
		}

		// Associate labels
		labels := make([]*Label, 0)
		for _, labelInfo := range basicInfo.Labels {
			if labelInfo.ID > 0 {
				label, err := f.repos.Label.GetByDiscogsID(ctx, tx, labelInfo.ID)
				if err != nil || label == nil {
					log.Warn("Label not found for association", "labelID", labelInfo.ID)
					continue
				}
				labels = append(labels, label)
			}
		}

		if len(labels) > 0 {
			if err := f.repos.Release.AssociateLabels(ctx, tx, release, labels); err != nil {
				log.Warn("Failed to associate labels with release",
					"releaseID", release.ID,
					"labelCount", len(labels),
					"error", err)
			}
		}

		// Associate genres
		genres := make([]*Genre, 0)
		for _, genreName := range basicInfo.Genres {
			if genreName != "" {
				genre, err := f.repos.Genre.GetByName(ctx, tx, genreName)
				if err != nil || genre == nil {
					log.Warn("Genre not found for association", "genreName", genreName)
					continue
				}
				genres = append(genres, genre)
			}
		}

		if len(genres) > 0 {
			if err := f.repos.Release.AssociateGenres(ctx, tx, release, genres); err != nil {
				log.Warn("Failed to associate genres with release",
					"releaseID", release.ID,
					"genreCount", len(genres),
					"error", err)
			}
		}
	}

	return nil
}

// GetRecordsNeedingFullData identifies records that need full data population (LastSynced is null)
func (f *FolderDataExtractionService) GetRecordsNeedingFullData(
	ctx context.Context,
	tx *gorm.DB,
	releaseIDs []int64,
) ([]int64, error) {
	log := f.log.Function("GetRecordsNeedingFullData")

	if len(releaseIDs) == 0 {
		return []int64{}, nil
	}

	var releases []struct {
		ID         int64      `json:"discogsId"`
		LastSynced *time.Time `json:"lastSynced"`
	}

	if err := tx.WithContext(ctx).
		Model(&Release{}).
		Select("id, last_synced").
		Where("id IN ? AND last_synced IS NULL", releaseIDs).
		Find(&releases).Error; err != nil {
		return nil, log.Err("failed to get releases needing full data", err)
	}

	needingFullData := make([]int64, len(releases))
	for i, release := range releases {
		needingFullData[i] = release.ID
	}

	return needingFullData, nil
}

// createMasterAssociations handles the many-to-many relationships between masters and other entities
func (f *FolderDataExtractionService) createMasterAssociations(
	ctx context.Context,
	tx *gorm.DB,
	folderReleases []DiscogsFolderReleaseItem,
) error {
	log := f.log.Function("createMasterAssociations")

	processedMasterArtists := make(map[string]bool) // key: "masterID:artistID"
	processedMasterGenres := make(map[string]bool)  // key: "masterID:genreName"

	for _, folderRelease := range folderReleases {
		basicInfo := folderRelease.BasicInformation

		// Only process if we have a master ID
		if basicInfo.MasterID <= 0 {
			continue
		}

		// Get the master to ensure it exists
		master, err := f.repos.Master.GetByDiscogsID(ctx, tx, basicInfo.MasterID)
		if err != nil {
			log.Warn("Master not found for associations", "masterID", basicInfo.MasterID)
			continue
		}
		if master == nil {
			log.Warn("Master not found for associations", "masterID", basicInfo.MasterID)
			continue
		}

		// Associate artists with master
		masterArtists := make([]*Artist, 0)
		for _, artistInfo := range basicInfo.Artists {
			if artistInfo.ID <= 0 {
				continue
			}

			associationKey := fmt.Sprintf("%d:%d", basicInfo.MasterID, artistInfo.ID)
			if processedMasterArtists[associationKey] {
				continue // Skip if already processed
			}

			// Get the artist to ensure it exists
			artist, err := f.repos.Artist.GetByDiscogsID(ctx, tx, artistInfo.ID)
			if err != nil {
				log.Warn("Artist not found for master association", "artistID", artistInfo.ID)
				continue
			}
			if artist == nil {
				log.Warn("Artist not found for master association", "artistID", artistInfo.ID)
				continue
			}

			masterArtists = append(masterArtists, artist)
			processedMasterArtists[associationKey] = true
		}

		if len(masterArtists) > 0 {
			if err := f.repos.Master.AssociateArtists(ctx, tx, master, masterArtists); err != nil {
				log.Warn("Failed to associate artists with master",
					"masterID", basicInfo.MasterID,
					"artistCount", len(masterArtists),
					"error", err)
			}
		}

		// Associate genres with master
		masterGenres := make([]*Genre, 0)
		for _, genreName := range basicInfo.Genres {
			if genreName == "" {
				continue
			}

			associationKey := fmt.Sprintf("%d:%s", basicInfo.MasterID, genreName)
			if processedMasterGenres[associationKey] {
				continue // Skip if already processed
			}

			// Get the genre to ensure it exists
			genre, err := f.repos.Genre.GetByName(ctx, tx, genreName)
			if err != nil {
				log.Warn("Genre not found for master association", "genreName", genreName)
				continue
			}
			if genre == nil {
				log.Warn("Genre not found for master association", "genreName", genreName)
				continue
			}

			masterGenres = append(masterGenres, genre)
			processedMasterGenres[associationKey] = true
		}

		if len(masterGenres) > 0 {
			if err := f.repos.Master.AssociateGenres(ctx, tx, master, masterGenres); err != nil {
				log.Warn("Failed to associate genres with master",
					"masterID", basicInfo.MasterID,
					"genreCount", len(masterGenres),
					"error", err)
			}
		}
	}

	return nil
}
