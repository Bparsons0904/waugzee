package services

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"
)

type FileCleanupService struct {
	config config.Config
	log    logger.Logger
}

func NewFileCleanupService(config config.Config) *FileCleanupService {
	return &FileCleanupService{
		config: config,
		log:    logger.New("fileCleanupService"),
	}
}

type StoredFile struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	IsXML      bool      `json:"is_xml"`
	IsGZ       bool      `json:"is_gz"`
}

func (fcs *FileCleanupService) ListStoredFiles(ctx context.Context) ([]StoredFile, error) {
	log := fcs.log.Function("ListStoredFiles")

	if _, err := os.Stat(DiscogsDataDir); os.IsNotExist(err) {
		log.Info("Download directory does not exist", "directory", DiscogsDataDir)
		return []StoredFile{}, nil
	}

	var files []StoredFile

	err := filepath.Walk(DiscogsDataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(DiscogsDataDir, path)
		if err != nil {
			return err
		}

		fileName := info.Name()

		files = append(files, StoredFile{
			Path:       relPath,
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			IsXML:      strings.HasSuffix(fileName, ".xml") || strings.Contains(fileName, ".xml."),
			IsGZ:       strings.HasSuffix(fileName, ".gz"),
		})

		return nil
	})

	if err != nil {
		return nil, log.Err("failed to walk directory", err, "directory", DiscogsDataDir)
	}

	log.Info("Listed stored files", "count", len(files))
	return files, nil
}

func (fcs *FileCleanupService) CleanupAllFiles(ctx context.Context) error {
	log := fcs.log.Function("CleanupAllFiles")

	if _, err := os.Stat(DiscogsDataDir); os.IsNotExist(err) {
		log.Info("Download directory does not exist, nothing to cleanup", "directory", DiscogsDataDir)
		return nil
	}

	entries, err := os.ReadDir(DiscogsDataDir)
	if err != nil {
		return log.Err("failed to read download directory", err, "directory", DiscogsDataDir)
	}

	var errors []error
	for _, entry := range entries {
		entryPath := filepath.Join(DiscogsDataDir, entry.Name())
		if err := os.RemoveAll(entryPath); err != nil {
			errors = append(errors, err)
			log.Er("failed to remove entry", err, "path", entryPath)
		}
	}

	if len(errors) > 0 {
		return log.Err("failed to cleanup some files", errors[0], "errorCount", len(errors))
	}

	log.Info("Successfully cleaned up all files", "directory", DiscogsDataDir, "itemsRemoved", len(entries))
	return nil
}

func (fcs *FileCleanupService) CleanupYearMonth(ctx context.Context, yearMonth string) error {
	log := fcs.log.Function("CleanupYearMonth")

	if !regexp.MustCompile(`^\d{4}-\d{2}$`).MatchString(yearMonth) {
		return log.Err("invalid yearMonth format", nil, "yearMonth", yearMonth, "expected", "YYYY-MM")
	}

	downloadDir := filepath.Join(DiscogsDataDir, yearMonth)

	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		log.Info("Download directory does not exist, nothing to cleanup", "directory", downloadDir)
		return nil
	}

	if err := os.RemoveAll(downloadDir); err != nil {
		return log.Err("failed to remove download directory", err, "directory", downloadDir)
	}

	log.Info("Successfully cleaned up year-month directory", "directory", downloadDir)
	return nil
}

func (fcs *FileCleanupService) ScheduledMonthlyCleanup(ctx context.Context) error {
	log := fcs.log.Function("ScheduledMonthlyCleanup")

	log.Info("Running scheduled monthly cleanup (last day of month)")
	return fcs.CleanupAllFiles(ctx)
}
