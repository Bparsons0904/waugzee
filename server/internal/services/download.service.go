package services

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"waugzee/config"
	"waugzee/internal/events"
	logger "github.com/Bparsons0904/goLogger"
	"waugzee/internal/models"

	"github.com/google/uuid"
)

// Discogs S3 configuration constants
const (
	DiscogsS3BaseURL       = "https://discogs-data-dumps.s3-us-west-2.amazonaws.com/data"
	DiscogsDataDir         = "/app/discogs-data"
	DiscogsTimeoutSec      = 3600 // 1 hour HTTP client timeout (safety net)
	DiscogsStallTimeoutSec = 300  // 5 minutes stall timeout (no progress detection)
	DiscogsUserAgent       = "Waugzee/1.0 (Discogs Data Sync)"
	DiscogsMaxRetries      = 5
)

type DownloadService struct {
	config     config.Config
	httpClient *http.Client
	log        logger.Logger
	eventBus   *events.EventBus
}

// Exponential backoff schedule (immediate, 5min, 25min, 75min, 375min)
var retrySchedule = []time.Duration{
	0 * time.Second,   // Immediate
	5 * time.Minute,   // 5 minutes
	25 * time.Minute,  // 25 minutes
	75 * time.Minute,  // 75 minutes
	375 * time.Minute, // 375 minutes (6.25 hours)
}

const maxRetries = DiscogsMaxRetries

func NewDownloadService(cfg config.Config, eventBus *events.EventBus) *DownloadService {
	log := logger.New("downloadService")

	// Create HTTP client with constant timeout
	timeout := time.Duration(DiscogsTimeoutSec) * time.Second

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
			MaxConnsPerHost:    10,
		},
	}

	return &DownloadService{
		config:     cfg,
		httpClient: httpClient,
		log:        log,
		eventBus:   eventBus,
	}
}

func (ds *DownloadService) BroadcastProgress(yearMonth, status, fileType, stage string, downloaded, total int64, err error) {
	if ds.eventBus == nil {
		return
	}

	var percentage float64
	if total > 0 {
		percentage = float64(downloaded) / float64(total) * 100
	}

	var errMsg *string
	if err != nil {
		msg := err.Error()
		errMsg = &msg
	}

	progressEvent := map[string]any{
		"yearMonth":    yearMonth,
		"status":       status,
		"fileType":     fileType,
		"stage":        stage,
		"downloaded":   downloaded,
		"total":        total,
		"percentage":   percentage,
		"errorMessage": errMsg,
	}

	message := events.Message{
		ID:        uuid.New().String(),
		Service:   events.SYSTEM,
		Event:     string(events.ADMIN_DOWNLOAD_PROGRESS),
		Payload:   progressEvent,
		Timestamp: time.Now(),
	}

	if err := ds.eventBus.Publish(events.WEBSOCKET, "admin", message); err != nil {
		ds.log.Warn("Failed to publish admin download progress", "error", err)
	}
}

// DownloadChecksum downloads the CHECKSUM.txt file from Discogs S3 for the current year-month
func (ds *DownloadService) DownloadChecksum(ctx context.Context, yearMonth string) error {
	log := ds.log.Function("DownloadChecksum")

	// Validate yearMonth format
	if !isValidYearMonth(yearMonth) {
		return log.Err(
			"invalid yearMonth format",
			fmt.Errorf("expected YYYY-MM format, got: %s", yearMonth),
		)
	}

	// Use current year-month if not provided (always use current for URL construction)
	currentYearMonth := time.Now().UTC().Format("2006-01")

	// Extract year for URL construction
	year := strings.Split(currentYearMonth, "-")[0]

	// Build S3 URL for CHECKSUM.txt
	checksumURL := fmt.Sprintf(
		"%s/%s/discogs_%s01_CHECKSUM.txt",
		DiscogsS3BaseURL,
		year,
		strings.ReplaceAll(currentYearMonth, "-", ""),
	)

	// Create download directory
	downloadDir := fmt.Sprintf("%s/%s", DiscogsDataDir, yearMonth)
	if err := ensureDirectory(downloadDir, log); err != nil {
		return log.Err("failed to create download directory", err, "directory", downloadDir)
	}

	// Target file path
	targetFile := filepath.Join(downloadDir, "CHECKSUM.txt")

	log.Info("Starting checksum download",
		"url", checksumURL,
		"targetFile", targetFile,
		"yearMonth", yearMonth)

	// Download with retry logic
	return ds.downloadFileWithRetry(ctx, checksumURL, targetFile)
}

// ParseChecksumFile parses the CHECKSUM.txt file and returns FileChecksums struct
func (ds *DownloadService) ParseChecksumFile(filePath string) (*models.FileChecksums, error) {
	log := ds.log.Function("ParseChecksumFile")

	file, err := os.Open(filePath)
	if err != nil {
		return nil, log.Err("failed to open checksum file", err, "filePath", filePath)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Warn("failed to close checksum file", "error", closeErr, "filePath", filePath)
		}
	}()

	checksums := &models.FileChecksums{}
	scanner := bufio.NewScanner(file)

	log.Debug("Parsing checksum file", "filePath", filePath)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Expected format: "checksum  filename"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			log.Warn("skipping malformed line in checksum file", "line", line)
			continue
		}

		checksum := parts[0]
		filename := parts[1]

		// Map filenames to checksum fields
		switch {
		case strings.Contains(strings.ToLower(filename), "artists"):
			checksums.ArtistsDump = checksum
			log.Debug("Found artists checksum", "checksum", checksum, "filename", filename)
		case strings.Contains(strings.ToLower(filename), "labels"):
			checksums.LabelsDump = checksum
			log.Debug("Found labels checksum", "checksum", checksum, "filename", filename)
		case strings.Contains(strings.ToLower(filename), "masters"):
			checksums.MastersDump = checksum
			log.Debug("Found masters checksum", "checksum", checksum, "filename", filename)
		case strings.Contains(strings.ToLower(filename), "releases"):
			checksums.ReleasesDump = checksum
			log.Debug("Found releases checksum", "checksum", checksum, "filename", filename)
		default:
			log.Debug("Skipping unrecognized file in checksum", "filename", filename)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, log.Err("error reading checksum file", err, "filePath", filePath)
	}

	// Validate that we found at least some checksums
	if checksums.ArtistsDump == "" && checksums.LabelsDump == "" &&
		checksums.MastersDump == "" && checksums.ReleasesDump == "" {
		return nil, log.Err(
			"no valid checksums found in file",
			fmt.Errorf("empty or invalid checksum file"),
			"filePath",
			filePath,
		)
	}

	log.Info("Successfully parsed checksum file",
		"filePath", filePath,
		"foundArtists", checksums.ArtistsDump != "",
		"foundLabels", checksums.LabelsDump != "",
		"foundMasters", checksums.MastersDump != "",
		"foundReleases", checksums.ReleasesDump != "")

	return checksums, nil
}

// DownloadXMLFile downloads a specific XML file (artists.xml.gz or labels.xml.gz) from Discogs S3
func (ds *DownloadService) DownloadXMLFile(ctx context.Context, yearMonth, fileType string) error {
	log := ds.log.Function("DownloadXMLFile")

	// Validate inputs
	if !isValidYearMonth(yearMonth) {
		return log.Err(
			"invalid yearMonth format",
			fmt.Errorf("expected YYYY-MM format, got: %s", yearMonth),
		)
	}

	// Validate file type
	validFileTypes := []string{"artists", "labels", "masters", "releases"}
	isValid := slices.Contains(validFileTypes, fileType)
	if !isValid {
		return log.Err(
			"invalid file type",
			fmt.Errorf("expected one of %v, got: %s", validFileTypes, fileType),
		)
	}

	// Use current year-month for URL construction (always download current month data)
	currentYearMonth := time.Now().UTC().Format("2006-01")
	year := strings.Split(currentYearMonth, "-")[0]

	// Build S3 URL for XML file
	xmlURL := fmt.Sprintf(
		"%s/%s/discogs_%s01_%s.xml.gz",
		DiscogsS3BaseURL,
		year,
		strings.ReplaceAll(currentYearMonth, "-", ""),
		fileType,
	)

	// Create download directory
	downloadDir := fmt.Sprintf("%s/%s", DiscogsDataDir, yearMonth)
	if err := ensureDirectory(downloadDir, log); err != nil {
		return log.Err("failed to create download directory", err, "directory", downloadDir)
	}

	// Target file path
	targetFile := filepath.Join(downloadDir, fmt.Sprintf("%s.xml.gz", fileType))

	log.Info("Starting XML file download",
		"fileType", fileType,
		"url", xmlURL,
		"targetFile", targetFile,
		"yearMonth", yearMonth)

	// Download with retry logic
	return ds.downloadFileWithRetry(ctx, xmlURL, targetFile)
}

// ValidateFileChecksum validates an existing file against its expected SHA256 checksum
func (ds *DownloadService) ValidateFileChecksum(filePath, expectedChecksum string) error {
	log := ds.log.Function("ValidateFileChecksum")

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return log.Err("failed to open file for checksum validation", err, "filePath", filePath)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Warn(
				"failed to close file after checksum validation",
				"error",
				closeErr,
				"filePath",
				filePath,
			)
		}
	}()

	// Create SHA256 hash
	hash := sha256.New()

	// Copy file content to hash with buffered reading for large files
	buffer := make([]byte, 32*1024) // 32KB buffer
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			if _, writeErr := hash.Write(buffer[:n]); writeErr != nil {
				return log.Err("failed to write to SHA256 hash", writeErr, "filePath", filePath)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return log.Err("failed to read file for checksum", readErr, "filePath", filePath)
		}
	}

	// Get computed checksum
	computedChecksum := hex.EncodeToString(hash.Sum(nil))

	// Compare checksums (case insensitive)
	if !strings.EqualFold(computedChecksum, expectedChecksum) {
		return log.Err("checksum validation failed",
			fmt.Errorf("computed: %s, expected: %s", computedChecksum, expectedChecksum),
			"filePath", filePath,
			"computed", computedChecksum,
			"expected", expectedChecksum)
	}

	log.Info("Checksum validation successful",
		"filePath", filePath,
		"checksum", computedChecksum)

	return nil
}

// downloadFileWithRetry downloads a file with exponential backoff retry logic
func (ds *DownloadService) downloadFileWithRetry(
	ctx context.Context,
	url, targetFile string,
) error {
	log := ds.log.Function("downloadFileWithRetry")

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Wait for retry delay (except first attempt)
		if attempt > 0 {
			delay := retrySchedule[attempt-1]
			log.Info("Retrying download after delay",
				"attempt", attempt+1,
				"maxRetries", maxRetries,
				"delay", delay,
				"url", url)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return log.Err("download cancelled during retry delay", ctx.Err())
			}
		} else {
			log.Info("Starting download attempt",
				"attempt", attempt+1,
				"maxRetries", maxRetries,
				"url", url)
		}

		// Attempt download
		err := ds.downloadFile(ctx, url, targetFile)
		if err == nil {
			log.Info("Download completed successfully",
				"attempt", attempt+1,
				"url", url,
				"targetFile", targetFile)
			return nil
		}

		lastErr = err
		log.Warn("Download attempt failed",
			"attempt", attempt+1,
			"error", err,
			"url", url)

		// Check if we should retry (context cancellation should not be retried)
		if ctx.Err() != nil {
			return log.Err("download cancelled", ctx.Err())
		}
	}

	return log.Err("download failed after all retry attempts",
		lastErr,
		"maxRetries", maxRetries,
		"url", url)
}

// downloadFile downloads a single file from URL to targetFile with progress-based timeout
func (ds *DownloadService) downloadFile(ctx context.Context, url, targetFile string) error {
	log := ds.log.Function("downloadFile")

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return log.Err("failed to create HTTP request", err, "url", url)
	}

	// Set User-Agent header for Discogs S3
	req.Header.Set("User-Agent", DiscogsUserAgent)

	// Make HTTP request
	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return log.Err("HTTP request failed", err, "url", url)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Warn("failed to close response body", "error", closeErr)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return log.Err("HTTP request failed",
			fmt.Errorf("status code: %d", resp.StatusCode),
			"url", url,
			"statusCode", resp.StatusCode)
	}

	// Create target file
	outFile, err := os.Create(targetFile)
	if err != nil {
		return log.Err("failed to create target file", err, "targetFile", targetFile)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			log.Warn("failed to close target file", "error", closeErr, "targetFile", targetFile)
		}
	}()

	// Track download progress with stall detection
	contentLength := resp.ContentLength
	downloaded := int64(0)
	lastProgressTime := time.Now()
	lastLogTime := time.Now()
	stallTimeout := time.Duration(DiscogsStallTimeoutSec) * time.Second

	// Extract metadata for WebSocket broadcasts
	yearMonth := ds.extractYearMonthFromPath(targetFile)
	fileType := ds.extractFileTypeFromURL(url)

	log.Info("Starting download with progress-based timeout",
		"url", url,
		"stallTimeoutSec", DiscogsStallTimeoutSec,
		"contentLength", contentLength)

	// Broadcast download started
	ds.BroadcastProgress(yearMonth, "downloading", fileType, "in_progress", 0, contentLength, nil)

	// Copy response body to file with progress tracking and stall detection
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		select {
		case <-ctx.Done():
			return log.Err("download cancelled", ctx.Err())
		default:
		}

		// Check for stall - if no progress for stallTimeout, abort
		if time.Since(lastProgressTime) > stallTimeout {
			return log.Err("download stalled - no progress detected",
				fmt.Errorf("no progress for %v seconds", DiscogsStallTimeoutSec),
				"url", url,
				"downloaded", downloaded,
				"stallTimeoutSec", DiscogsStallTimeoutSec)
		}

		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := outFile.Write(buffer[:n]); writeErr != nil {
				return log.Err("failed to write to file", writeErr, "targetFile", targetFile)
			}

			// Progress made - reset stall timer
			downloaded += int64(n)
			lastProgressTime = time.Now()

			// Log and broadcast progress every 30 seconds
			now := time.Now()
			if now.Sub(lastLogTime) >= 30*time.Second {
				ds.logDownloadProgress(contentLength, downloaded, url)
				ds.BroadcastProgress(yearMonth, "downloading", fileType, "in_progress", downloaded, contentLength, nil)
				lastLogTime = now
				log.Debug("Progress detected, stall timer reset",
					"downloaded", downloaded,
					"stallTimeoutSec", DiscogsStallTimeoutSec)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return log.Err("failed to read response body", readErr, "url", url)
		}
	}

	// Final progress log and broadcast
	ds.logDownloadProgress(contentLength, downloaded, url)
	ds.BroadcastProgress(yearMonth, "completed", fileType, "completed", downloaded, contentLength, nil)

	log.Info("File download completed",
		"url", url,
		"targetFile", targetFile,
		"size", downloaded)

	return nil
}

// logDownloadProgress logs download progress information
func (ds *DownloadService) logDownloadProgress(contentLength, downloaded int64, url string) {
	if contentLength > 0 {
		percentage := float64(downloaded) / float64(contentLength) * 100
		ds.log.Info("Download progress",
			"url", url,
			"downloaded", downloaded,
			"total", contentLength,
			"percentage", fmt.Sprintf("%.1f%%", percentage))
	} else {
		ds.log.Info("Download progress",
			"url", url,
			"downloaded", downloaded,
			"total", "unknown")
	}
}

// extractFileTypeFromURL extracts file type from Discogs S3 URL
func (ds *DownloadService) extractFileTypeFromURL(url string) string {
	if strings.Contains(url, "artists") {
		return "artists"
	} else if strings.Contains(url, "labels") {
		return "labels"
	} else if strings.Contains(url, "masters") {
		return "masters"
	} else if strings.Contains(url, "releases") {
		return "releases"
	} else if strings.Contains(url, "CHECKSUM") {
		return "checksum"
	}
	return "unknown"
}

// extractYearMonthFromURL extracts yearMonth from target file path
func (ds *DownloadService) extractYearMonthFromPath(targetFile string) string {
	parts := strings.Split(targetFile, "/")
	for _, part := range parts {
		if len(part) == 7 && strings.Count(part, "-") == 1 {
			return part
		}
	}
	return ""
}

// ensureDirectory creates directory if it doesn't exist
func ensureDirectory(dir string, logger logger.Logger) error {
	log := logger.Function("ensureDirectory")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return log.Err("failed to create directory", err, "directory", dir)
		}
		log.Info("Created download directory", "directory", dir)
	}
	return nil
}

// CheckExistingFile checks if a file exists and optionally validates its checksum
func (ds *DownloadService) CheckExistingFile(
	filePath string,
	expectedChecksum string,
	validateChecksum bool,
) (exists bool, valid bool, size int64, err error) {
	log := ds.log.Function("CheckExistingFile")

	// Check if file exists
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, false, 0, nil
	}
	if err != nil {
		return false, false, 0, log.Err("failed to stat file", err, "filePath", filePath)
	}

	exists = true
	size = info.Size()

	// If checksum validation is not requested, return early
	if !validateChecksum || expectedChecksum == "" {
		return exists, true, size, nil // Assume valid if not validating checksum
	}

	// Validate checksum if requested
	if err := ds.ValidateFileChecksum(filePath, expectedChecksum); err != nil {
		log.Warn("Existing file failed checksum validation",
			"filePath", filePath,
			"expectedChecksum", expectedChecksum,
			"error", err)
		return exists, false, size, nil
	}

	log.Info("Existing file validated successfully",
		"filePath", filePath,
		"size", size)

	return exists, true, size, nil
}

// GetFileStatus returns the current status of a file based on existence and checksum validation
func (ds *DownloadService) GetFileStatus(
	filePath string,
	expectedChecksum string,
) (*models.FileDownloadInfo, error) {
	log := ds.log.Function("GetFileStatus")

	exists, valid, size, err := ds.CheckExistingFile(
		filePath,
		expectedChecksum,
		expectedChecksum != "",
	)
	if err != nil {
		return nil, log.Err("failed to check existing file", err, "filePath", filePath)
	}

	info := &models.FileDownloadInfo{
		Size: size,
	}

	if !exists {
		info.Status = models.FileDownloadStatusNotStarted
		info.Downloaded = false
		info.Validated = false
	} else if !valid {
		info.Status = models.FileDownloadStatusFailed
		info.Downloaded = true
		info.Validated = false
		errorMsg := "checksum validation failed"
		info.ErrorMessage = &errorMsg
	} else {
		info.Status = models.FileDownloadStatusValidated
		info.Downloaded = true
		info.Validated = true
		now := time.Now().UTC()
		info.DownloadedAt = &now
		info.ValidatedAt = &now
	}

	return info, nil
}

// CleanupDownloadDirectory removes all downloaded files for a specific year-month
func (ds *DownloadService) CleanupDownloadDirectory(ctx context.Context, yearMonth string) error {
	log := ds.log.Function("CleanupDownloadDirectory")

	if !isValidYearMonth(yearMonth) {
		return log.Err(
			"invalid yearMonth format",
			fmt.Errorf("expected YYYY-MM format, got: %s", yearMonth),
		)
	}

	downloadDir := fmt.Sprintf("%s/%s", DiscogsDataDir, yearMonth)

	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		log.Info("Download directory does not exist, nothing to cleanup", "directory", downloadDir)
		return nil
	}

	if err := os.RemoveAll(downloadDir); err != nil {
		return log.Err("failed to remove download directory", err, "directory", downloadDir)
	}

	log.Info("Successfully cleaned up download directory", "directory", downloadDir)
	return nil
}

// isValidYearMonth validates YYYY-MM format
func isValidYearMonth(yearMonth string) bool {
	parts := strings.Split(yearMonth, "-")
	if len(parts) != 2 {
		return false
	}

	year, month := parts[0], parts[1]
	return len(year) == 4 && len(month) == 2
}
