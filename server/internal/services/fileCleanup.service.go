package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	YearMonth    string    `json:"year_month"`
	SizeBytes    int64     `json:"size_bytes"`
	ModifiedTime time.Time `json:"modified_time"`
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

		yearMonth := filepath.Dir(relPath)
		if yearMonth == "." {
			yearMonth = ""
		}

		files = append(files, StoredFile{
			FileName:     info.Name(),
			FilePath:     path,
			YearMonth:    yearMonth,
			SizeBytes:    info.Size(),
			ModifiedTime: info.ModTime(),
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

	if err := os.RemoveAll(DiscogsDataDir); err != nil {
		return log.Err("failed to remove download directory", err, "directory", DiscogsDataDir)
	}

	log.Info("Successfully cleaned up all files", "directory", DiscogsDataDir)
	return nil
}

func (fcs *FileCleanupService) CleanupYearMonth(ctx context.Context, yearMonth string) error {
	log := fcs.log.Function("CleanupYearMonth")

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

func (fcs *FileCleanupService) IsLastDayOfMonth() bool {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	return tomorrow.Month() != now.Month()
}

func (fcs *FileCleanupService) ScheduledMonthlyCleanup(ctx context.Context) error {
	log := fcs.log.Function("ScheduledMonthlyCleanup")

	if !fcs.IsLastDayOfMonth() {
		log.Info("Not last day of month, skipping cleanup")
		return nil
	}

	log.Info("Last day of month detected, running cleanup")
	return fcs.CleanupAllFiles(ctx)
}

func isValidYearMonth(yearMonth string) error {
	_, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return fmt.Errorf("invalid year-month format, expected YYYY-MM: %w", err)
	}
	return nil
}
