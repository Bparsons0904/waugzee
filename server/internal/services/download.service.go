package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

type DownloadService struct {
	config     config.Config
	httpClient *http.Client
	log        logger.Logger
}

// Exponential backoff schedule (immediate, 5min, 25min, 75min, 375min)
var retrySchedule = []time.Duration{
	0 * time.Second,      // Immediate
	5 * time.Minute,      // 5 minutes
	25 * time.Minute,     // 25 minutes
	75 * time.Minute,     // 75 minutes
	375 * time.Minute,    // 375 minutes (6.25 hours)
}

const maxRetries = 5

func NewDownloadService(cfg config.Config) *DownloadService {
	log := logger.New("downloadService")

	// Create HTTP client with configurable timeout
	timeout := time.Duration(cfg.DiscogsTimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			MaxConnsPerHost:     10,
		},
	}

	return &DownloadService{
		config:     cfg,
		httpClient: httpClient,
		log:        log,
	}
}

// DownloadChecksum downloads the CHECKSUM.txt file from Discogs S3 for the current year-month
func (ds *DownloadService) DownloadChecksum(ctx context.Context, yearMonth string) error {
	log := ds.log.Function("DownloadChecksum")

	// Validate yearMonth format
	if !isValidYearMonth(yearMonth) {
		return log.Err("invalid yearMonth format", fmt.Errorf("expected YYYY-MM format, got: %s", yearMonth))
	}

	// Use current year-month if not provided (always use current for URL construction)
	currentYearMonth := time.Now().UTC().Format("2006-01")
	
	// Extract year for URL construction
	year := strings.Split(currentYearMonth, "-")[0]
	
	// Build S3 URL for CHECKSUM.txt
	checksumURL := fmt.Sprintf(
		"https://discogs-data-dumps.s3-us-west-2.amazonaws.com/data/%s/discogs_%s01_CHECKSUM.txt",
		year,
		strings.ReplaceAll(currentYearMonth, "-", ""),
	)

	// Create download directory
	downloadDir := fmt.Sprintf("/tmp/discogs-%s", yearMonth)
	if err := ds.ensureDirectory(downloadDir); err != nil {
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
		return nil, log.Err("no valid checksums found in file", fmt.Errorf("empty or invalid checksum file"), "filePath", filePath)
	}

	log.Info("Successfully parsed checksum file",
		"filePath", filePath,
		"foundArtists", checksums.ArtistsDump != "",
		"foundLabels", checksums.LabelsDump != "",
		"foundMasters", checksums.MastersDump != "",
		"foundReleases", checksums.ReleasesDump != "")

	return checksums, nil
}

// downloadFileWithRetry downloads a file with exponential backoff retry logic
func (ds *DownloadService) downloadFileWithRetry(ctx context.Context, url, targetFile string) error {
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

// downloadFile downloads a single file from URL to targetFile
func (ds *DownloadService) downloadFile(ctx context.Context, url, targetFile string) error {
	log := ds.log.Function("downloadFile")

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return log.Err("failed to create HTTP request", err, "url", url)
	}

	// Set User-Agent header for Discogs S3
	req.Header.Set("User-Agent", "Waugzee/"+ds.config.GeneralVersion+" (Discogs Data Sync)")

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

	// Track download progress
	contentLength := resp.ContentLength
	downloaded := int64(0)

	// Copy response body to file with progress tracking
	buffer := make([]byte, 32*1024) // 32KB buffer
	lastLogTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return log.Err("download cancelled", ctx.Err())
		default:
		}

		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := outFile.Write(buffer[:n]); writeErr != nil {
				return log.Err("failed to write to file", writeErr, "targetFile", targetFile)
			}
			downloaded += int64(n)

			// Log progress every 30 seconds or at completion
			now := time.Now()
			if now.Sub(lastLogTime) >= 30*time.Second {
				ds.logDownloadProgress(contentLength, downloaded, url)
				lastLogTime = now
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return log.Err("failed to read response body", readErr, "url", url)
		}
	}

	// Final progress log
	ds.logDownloadProgress(contentLength, downloaded, url)

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

// ensureDirectory creates directory if it doesn't exist
func (ds *DownloadService) ensureDirectory(dir string) error {
	log := ds.log.Function("ensureDirectory")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return log.Err("failed to create directory", err, "directory", dir)
		}
		log.Info("Created download directory", "directory", dir)
	}
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