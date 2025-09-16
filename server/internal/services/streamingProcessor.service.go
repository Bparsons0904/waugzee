package services

import (
	"context"
	"sync"
	"time"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

const DefaultBatchSize = 2000

// StreamingProcessor manages concurrent channels for different entity types
type StreamingProcessor struct {
	// Channels for each entity type
	labelChan   chan *models.Label
	artistChan  chan *models.Artist
	genreChan   chan *models.Genre
	masterChan  chan *models.Master
	releaseChan chan *models.Release
	trackChan   chan *models.Track

	// Configuration
	batchSize int
	log       logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Batch storage
	labelBatch   []*models.Label
	artistBatch  []*models.Artist
	genreBatch   []*models.Genre
	masterBatch  []*models.Master
	releaseBatch []*models.Release
	trackBatch   []*models.Track

	// Batch mutexes for thread safety
	labelMux   sync.Mutex
	artistMux  sync.Mutex
	genreMux   sync.Mutex
	masterMux  sync.Mutex
	releaseMux sync.Mutex
	trackMux   sync.Mutex

	// Statistics
	stats StreamingStats
}

// StreamingStats tracks processing statistics
type StreamingStats struct {
	TotalLabels   int
	TotalArtists  int
	TotalGenres   int
	TotalMasters  int
	TotalReleases int
	TotalTracks   int
	BatchesProcessed map[string]int
	StartTime        time.Time
	LastUpdate       time.Time
	mu               sync.RWMutex
}

// NewStreamingProcessor creates a new streaming processor
func NewStreamingProcessor() *StreamingProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &StreamingProcessor{
		// Initialize channels with buffer size equal to batch size for smooth operation
		labelChan:   make(chan *models.Label, DefaultBatchSize),
		artistChan:  make(chan *models.Artist, DefaultBatchSize),
		genreChan:   make(chan *models.Genre, DefaultBatchSize),
		masterChan:  make(chan *models.Master, DefaultBatchSize),
		releaseChan: make(chan *models.Release, DefaultBatchSize),
		trackChan:   make(chan *models.Track, DefaultBatchSize),

		batchSize: DefaultBatchSize,
		log:       logger.New("streamingProcessor"),
		ctx:       ctx,
		cancel:    cancel,

		// Initialize batch slices
		labelBatch:   make([]*models.Label, 0, DefaultBatchSize),
		artistBatch:  make([]*models.Artist, 0, DefaultBatchSize),
		genreBatch:   make([]*models.Genre, 0, DefaultBatchSize),
		masterBatch:  make([]*models.Master, 0, DefaultBatchSize),
		releaseBatch: make([]*models.Release, 0, DefaultBatchSize),
		trackBatch:   make([]*models.Track, 0, DefaultBatchSize),

		stats: StreamingStats{
			BatchesProcessed: make(map[string]int),
			StartTime:        time.Now(),
			LastUpdate:       time.Now(),
		},
	}

	return processor
}

// Start begins processing all channels concurrently
func (s *StreamingProcessor) Start() {
	s.log.Info("Starting streaming processor",
		"batchSize", s.batchSize,
		"channelBufferSize", DefaultBatchSize)

	// Start goroutines for each entity type
	s.wg.Add(6)
	go s.processLabels()
	go s.processArtists()
	go s.processGenres()
	go s.processMasters()
	go s.processReleases()
	go s.processTracks()

	s.log.Info("All channel processors started")
}

// Stop gracefully shuts down the processor
func (s *StreamingProcessor) Stop() {
	s.log.Info("Stopping streaming processor...")
	
	// Close all channels to signal completion
	close(s.labelChan)
	close(s.artistChan)
	close(s.genreChan)
	close(s.masterChan)
	close(s.releaseChan)
	close(s.trackChan)

	// Cancel context and wait for all goroutines to finish
	s.cancel()
	s.wg.Wait()

	// Process any remaining items in batches
	s.flushAllBatches()

	s.logFinalStats()
	s.log.Info("Streaming processor stopped")
}

// SendLabel sends a label to the processing channel
func (s *StreamingProcessor) SendLabel(label *models.Label) {
	select {
	case s.labelChan <- label:
		// Successfully sent
	case <-s.ctx.Done():
		s.log.Warn("Cannot send label, processor is shutting down")
	}
}

// SendArtist sends an artist to the processing channel
func (s *StreamingProcessor) SendArtist(artist *models.Artist) {
	select {
	case s.artistChan <- artist:
		// Successfully sent
	case <-s.ctx.Done():
		s.log.Warn("Cannot send artist, processor is shutting down")
	}
}

// SendGenre sends a genre to the processing channel
func (s *StreamingProcessor) SendGenre(genre *models.Genre) {
	select {
	case s.genreChan <- genre:
		// Successfully sent
	case <-s.ctx.Done():
		s.log.Warn("Cannot send genre, processor is shutting down")
	}
}

// SendMaster sends a master to the processing channel
func (s *StreamingProcessor) SendMaster(master *models.Master) {
	select {
	case s.masterChan <- master:
		// Successfully sent
	case <-s.ctx.Done():
		s.log.Warn("Cannot send master, processor is shutting down")
	}
}

// SendRelease sends a release to the processing channel
func (s *StreamingProcessor) SendRelease(release *models.Release) {
	select {
	case s.releaseChan <- release:
		// Successfully sent
	case <-s.ctx.Done():
		s.log.Warn("Cannot send release, processor is shutting down")
	}
}

// SendTrack sends a track to the processing channel
func (s *StreamingProcessor) SendTrack(track *models.Track) {
	select {
	case s.trackChan <- track:
		// Successfully sent
	case <-s.ctx.Done():
		s.log.Warn("Cannot send track, processor is shutting down")
	}
}

// Channel processing goroutines

func (s *StreamingProcessor) processLabels() {
	defer s.wg.Done()
	
	for {
		select {
		case label, ok := <-s.labelChan:
			if !ok {
				return // Channel closed
			}
			s.addLabelToBatch(label)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *StreamingProcessor) processArtists() {
	defer s.wg.Done()
	
	for {
		select {
		case artist, ok := <-s.artistChan:
			if !ok {
				return // Channel closed
			}
			s.addArtistToBatch(artist)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *StreamingProcessor) processGenres() {
	defer s.wg.Done()
	
	for {
		select {
		case genre, ok := <-s.genreChan:
			if !ok {
				return // Channel closed
			}
			s.addGenreToBatch(genre)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *StreamingProcessor) processMasters() {
	defer s.wg.Done()
	
	for {
		select {
		case master, ok := <-s.masterChan:
			if !ok {
				return // Channel closed
			}
			s.addMasterToBatch(master)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *StreamingProcessor) processReleases() {
	defer s.wg.Done()
	
	for {
		select {
		case release, ok := <-s.releaseChan:
			if !ok {
				return // Channel closed
			}
			s.addReleaseToBatch(release)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *StreamingProcessor) processTracks() {
	defer s.wg.Done()
	
	for {
		select {
		case track, ok := <-s.trackChan:
			if !ok {
				return // Channel closed
			}
			s.addTrackToBatch(track)
		case <-s.ctx.Done():
			return
		}
	}
}

// Batch management methods

func (s *StreamingProcessor) addLabelToBatch(label *models.Label) {
	s.labelMux.Lock()
	defer s.labelMux.Unlock()

	s.labelBatch = append(s.labelBatch, label)
	s.updateStats("labels", 1)

	if len(s.labelBatch) >= s.batchSize {
		s.processLabelBatch(s.labelBatch)
		s.labelBatch = make([]*models.Label, 0, s.batchSize) // Reset batch
	}
}

func (s *StreamingProcessor) addArtistToBatch(artist *models.Artist) {
	s.artistMux.Lock()
	defer s.artistMux.Unlock()

	s.artistBatch = append(s.artistBatch, artist)
	s.updateStats("artists", 1)

	if len(s.artistBatch) >= s.batchSize {
		s.processArtistBatch(s.artistBatch)
		s.artistBatch = make([]*models.Artist, 0, s.batchSize) // Reset batch
	}
}

func (s *StreamingProcessor) addGenreToBatch(genre *models.Genre) {
	s.genreMux.Lock()
	defer s.genreMux.Unlock()

	s.genreBatch = append(s.genreBatch, genre)
	s.updateStats("genres", 1)

	if len(s.genreBatch) >= s.batchSize {
		s.processGenreBatch(s.genreBatch)
		s.genreBatch = make([]*models.Genre, 0, s.batchSize) // Reset batch
	}
}

func (s *StreamingProcessor) addMasterToBatch(master *models.Master) {
	s.masterMux.Lock()
	defer s.masterMux.Unlock()

	s.masterBatch = append(s.masterBatch, master)
	s.updateStats("masters", 1)

	if len(s.masterBatch) >= s.batchSize {
		s.processMasterBatch(s.masterBatch)
		s.masterBatch = make([]*models.Master, 0, s.batchSize) // Reset batch
	}
}

func (s *StreamingProcessor) addReleaseToBatch(release *models.Release) {
	s.releaseMux.Lock()
	defer s.releaseMux.Unlock()

	s.releaseBatch = append(s.releaseBatch, release)
	s.updateStats("releases", 1)

	if len(s.releaseBatch) >= s.batchSize {
		s.processReleaseBatch(s.releaseBatch)
		s.releaseBatch = make([]*models.Release, 0, s.batchSize) // Reset batch
	}
}

func (s *StreamingProcessor) addTrackToBatch(track *models.Track) {
	s.trackMux.Lock()
	defer s.trackMux.Unlock()

	s.trackBatch = append(s.trackBatch, track)
	s.updateStats("tracks", 1)

	if len(s.trackBatch) >= s.batchSize {
		s.processTrackBatch(s.trackBatch)
		s.trackBatch = make([]*models.Track, 0, s.batchSize) // Reset batch
	}
}

// Batch processing methods (currently just logging)

func (s *StreamingProcessor) processLabelBatch(labels []*models.Label) {
	s.incrementBatchCount("labels")
	
	firstID := 0
	lastID := 0
	if len(labels) > 0 {
		firstID = labels[0].ID
		lastID = labels[len(labels)-1].ID
	}

	s.log.Info("Processing label batch",
		"batchSize", len(labels),
		"firstID", firstID,
		"lastID", lastID,
		"totalProcessed", s.stats.TotalLabels,
		"batchNumber", s.stats.BatchesProcessed["labels"])
	
	// TODO: Add database operations here when ready
}

func (s *StreamingProcessor) processArtistBatch(artists []*models.Artist) {
	s.incrementBatchCount("artists")
	
	firstID := 0
	lastID := 0
	if len(artists) > 0 {
		firstID = artists[0].ID
		lastID = artists[len(artists)-1].ID
	}

	s.log.Info("Processing artist batch",
		"batchSize", len(artists),
		"firstID", firstID,
		"lastID", lastID,
		"totalProcessed", s.stats.TotalArtists,
		"batchNumber", s.stats.BatchesProcessed["artists"])
	
	// TODO: Add database operations here when ready
}

func (s *StreamingProcessor) processGenreBatch(genres []*models.Genre) {
	s.incrementBatchCount("genres")
	
	firstID := 0
	lastID := 0
	if len(genres) > 0 {
		firstID = genres[0].ID
		lastID = genres[len(genres)-1].ID
	}

	s.log.Info("Processing genre batch",
		"batchSize", len(genres),
		"firstID", firstID,
		"lastID", lastID,
		"totalProcessed", s.stats.TotalGenres,
		"batchNumber", s.stats.BatchesProcessed["genres"])
	
	// TODO: Add database operations here when ready
}

func (s *StreamingProcessor) processMasterBatch(masters []*models.Master) {
	s.incrementBatchCount("masters")
	
	firstID := 0
	lastID := 0
	if len(masters) > 0 {
		firstID = masters[0].ID
		lastID = masters[len(masters)-1].ID
	}

	s.log.Info("Processing master batch",
		"batchSize", len(masters),
		"firstID", firstID,
		"lastID", lastID,
		"totalProcessed", s.stats.TotalMasters,
		"batchNumber", s.stats.BatchesProcessed["masters"])
	
	// TODO: Add database operations here when ready
}

func (s *StreamingProcessor) processReleaseBatch(releases []*models.Release) {
	s.incrementBatchCount("releases")
	
	firstID := 0
	lastID := 0
	if len(releases) > 0 {
		firstID = releases[0].ID
		lastID = releases[len(releases)-1].ID
	}

	s.log.Info("Processing release batch",
		"batchSize", len(releases),
		"firstID", firstID,
		"lastID", lastID,
		"totalProcessed", s.stats.TotalReleases,
		"batchNumber", s.stats.BatchesProcessed["releases"])
	
	// TODO: Add database operations here when ready
}

func (s *StreamingProcessor) processTrackBatch(tracks []*models.Track) {
	s.incrementBatchCount("tracks")
	
	firstID := 0
	lastID := 0
	if len(tracks) > 0 {
		firstID = tracks[0].ID
		lastID = tracks[len(tracks)-1].ID
	}

	s.log.Info("Processing track batch",
		"batchSize", len(tracks),
		"firstID", firstID,
		"lastID", lastID,
		"totalProcessed", s.stats.TotalTracks,
		"batchNumber", s.stats.BatchesProcessed["tracks"])
	
	// TODO: Add database operations here when ready
}

// Statistics and utility methods

func (s *StreamingProcessor) updateStats(entityType string, count int) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	switch entityType {
	case "labels":
		s.stats.TotalLabels += count
	case "artists":
		s.stats.TotalArtists += count
	case "genres":
		s.stats.TotalGenres += count
	case "masters":
		s.stats.TotalMasters += count
	case "releases":
		s.stats.TotalReleases += count
	case "tracks":
		s.stats.TotalTracks += count
	}
	
	s.stats.LastUpdate = time.Now()
}

func (s *StreamingProcessor) incrementBatchCount(entityType string) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()
	s.stats.BatchesProcessed[entityType]++
}

func (s *StreamingProcessor) flushAllBatches() {
	s.log.Info("Flushing remaining batches...")

	// Process any remaining items in batches
	s.labelMux.Lock()
	if len(s.labelBatch) > 0 {
		s.processLabelBatch(s.labelBatch)
	}
	s.labelMux.Unlock()

	s.artistMux.Lock()
	if len(s.artistBatch) > 0 {
		s.processArtistBatch(s.artistBatch)
	}
	s.artistMux.Unlock()

	s.genreMux.Lock()
	if len(s.genreBatch) > 0 {
		s.processGenreBatch(s.genreBatch)
	}
	s.genreMux.Unlock()

	s.masterMux.Lock()
	if len(s.masterBatch) > 0 {
		s.processMasterBatch(s.masterBatch)
	}
	s.masterMux.Unlock()

	s.releaseMux.Lock()
	if len(s.releaseBatch) > 0 {
		s.processReleaseBatch(s.releaseBatch)
	}
	s.releaseMux.Unlock()

	s.trackMux.Lock()
	if len(s.trackBatch) > 0 {
		s.processTrackBatch(s.trackBatch)
	}
	s.trackMux.Unlock()
}

func (s *StreamingProcessor) logFinalStats() {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	totalDuration := time.Since(s.stats.StartTime)
	totalEntities := s.stats.TotalLabels + s.stats.TotalArtists + s.stats.TotalGenres + 
		s.stats.TotalMasters + s.stats.TotalReleases + s.stats.TotalTracks

	s.log.Info("Final streaming processor statistics",
		"totalDurationMs", totalDuration.Milliseconds(),
		"totalEntities", totalEntities,
		"entitiesPerSecond", float64(totalEntities)/totalDuration.Seconds(),
		"labels", s.stats.TotalLabels,
		"artists", s.stats.TotalArtists,
		"genres", s.stats.TotalGenres,
		"masters", s.stats.TotalMasters,
		"releases", s.stats.TotalReleases,
		"tracks", s.stats.TotalTracks,
		"batchesProcessed", s.stats.BatchesProcessed)
}

// GetStats returns a copy of current statistics
func (s *StreamingProcessor) GetStats() StreamingStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	statsCopy := StreamingStats{
		TotalLabels:      s.stats.TotalLabels,
		TotalArtists:     s.stats.TotalArtists,
		TotalGenres:      s.stats.TotalGenres,
		TotalMasters:     s.stats.TotalMasters,
		TotalReleases:    s.stats.TotalReleases,
		TotalTracks:      s.stats.TotalTracks,
		BatchesProcessed: make(map[string]int),
		StartTime:        s.stats.StartTime,
		LastUpdate:       s.stats.LastUpdate,
	}
	
	for k, v := range s.stats.BatchesProcessed {
		statsCopy.BatchesProcessed[k] = v
	}
	
	return statsCopy
}