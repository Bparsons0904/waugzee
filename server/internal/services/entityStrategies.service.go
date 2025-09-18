package services

import (
	"strconv"
	"waugzee/internal/imports"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

type EntityProcessor struct {
	parserService *DiscogsParserService
	log           logger.Logger
}

func NewEntityProcessor(parserService *DiscogsParserService) *EntityProcessor {
	return &EntityProcessor{
		parserService: parserService,
		log:           logger.New("entityProcessor"),
	}
}

// processLabel extracts Label data only - no related entities
func (ep *EntityProcessor) ProcessLabel(
	rawLabel *imports.Label,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawLabel == nil || rawLabel.ID <= 0 {
		return ep.log.Err("invalid label data", nil, "processingID", processingID)
	}

	// Convert to model and send to label buffer
	if convertedLabel := ep.parserService.convertDiscogsLabel(rawLabel); convertedLabel != nil {
		buffers.Labels.Channel <- convertedLabel
	}

	return nil
}

// processArtist extracts Artist data + Images[] and sends Images to image buffer
func (ep *EntityProcessor) ProcessArtist(
	rawArtist *imports.Artist,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawArtist == nil || rawArtist.ID <= 0 {
		return ep.log.Err("invalid artist data", nil, "processingID", processingID)
	}

	// Extract and send Images to image buffer with context
	for i := range rawArtist.Images {
		contextualImage := &ContextualDiscogsImage{
			DiscogsImage:  &rawArtist.Images[i],
			ImageableID:   strconv.FormatInt(int64(rawArtist.ID), 10),
			ImageableType: models.ImageableTypeArtist,
		}
		buffers.Images.Channel <- contextualImage
	}

	return nil
}

// processMaster extracts Master data + Images[] + Genres[] + Artists[] and sends to appropriate buffers
func (ep *EntityProcessor) ProcessMaster(
	rawMaster *imports.Master,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawMaster == nil || rawMaster.ID <= 0 {
		return ep.log.Err("invalid master data", nil, "processingID", processingID)
	}

	// Extract and send Images to image buffer with context
	for i := range rawMaster.Images {
		contextualImage := &ContextualDiscogsImage{
			DiscogsImage:  &rawMaster.Images[i],
			ImageableID:   strconv.FormatInt(int64(rawMaster.ID), 10),
			ImageableType: models.ImageableTypeMaster,
		}
		buffers.Images.Channel <- contextualImage
	}

	// Extract and send Genres to genre buffer
	for _, genre := range rawMaster.Genres {
		if genre != "" {
			buffers.Genres.Channel <- genre
		}
	}

	// Extract and send Artists to artist buffer
	for i := range rawMaster.Artists {
		artist := &rawMaster.Artists[i]
		buffers.Artists.Channel <- artist
	}

	// Extract and send Master-Genre associations
	for _, genre := range rawMaster.Genres {
		if genre != "" {
			association := &MasterGenreAssociation{
				MasterDiscogsID: int64(rawMaster.ID),
				GenreName:       genre,
			}
			buffers.MasterGenres.Channel <- association
		}
	}

	// Extract and send Master-Artist associations
	for _, artist := range rawMaster.Artists {
		if artist.ID > 0 {
			association := &MasterArtistAssociation{
				MasterDiscogsID: int64(rawMaster.ID),
				ArtistDiscogsID: int64(artist.ID),
			}
			buffers.MasterArtists.Channel <- association
		}
	}

	// Convert to model and send to master buffer
	if convertedMaster := ep.parserService.convertDiscogsMaster(rawMaster); convertedMaster != nil {
		buffers.Masters.Channel <- convertedMaster
	}

	return nil
}

// processRelease extracts Release data + Artists[] + TrackList[] + Genres[] + Images[]
// and sends all related entities to appropriate buffers
func (ep *EntityProcessor) ProcessRelease(
	rawRelease *imports.Release,
	processingID string,
	buffers *ProcessingBuffers,
) error {
	if rawRelease == nil || rawRelease.ID <= 0 {
		return ep.log.Err("invalid release data", nil, "processingID", processingID)
	}

	// Extract and send Artists to artist buffer
	for i := range rawRelease.Artists {
		artist := &rawRelease.Artists[i]
		buffers.Artists.Channel <- artist
	}

	// Extract and send Genres to genre buffer
	for _, genre := range rawRelease.Genres {
		if genre != "" {
			buffers.Genres.Channel <- genre
		}
	}

	// Extract and send Images to image buffer with context
	for i := range rawRelease.Images {
		contextualImage := &ContextualDiscogsImage{
			DiscogsImage:  &rawRelease.Images[i],
			ImageableID:   strconv.FormatInt(int64(rawRelease.ID), 10),
			ImageableType: models.ImageableTypeRelease,
		}
		buffers.Images.Channel <- contextualImage
	}

	// Convert to model and send to release buffer
	if convertedRelease := ep.parserService.convertDiscogsRelease(rawRelease); convertedRelease != nil {
		buffers.Releases.Channel <- convertedRelease
	}

	return nil
}