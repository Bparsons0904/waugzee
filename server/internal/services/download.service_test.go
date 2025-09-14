package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"waugzee/config"
	"waugzee/internal/models"
)

func TestNewDownloadService(t *testing.T) {
	cfg := config.Config{
		DiscogsTimeoutSec: 30,
		GeneralVersion:    "1.0.0",
	}

	service := NewDownloadService(cfg)

	if service == nil {
		t.Fatal("expected service to be created, got nil")
	}

	if service.config.DiscogsTimeoutSec != 30 {
		t.Errorf("expected timeout 30, got %d", service.config.DiscogsTimeoutSec)
	}

	if service.httpClient == nil {
		t.Fatal("expected HTTP client to be created, got nil")
	}

	expectedTimeout := 30 * time.Second
	if service.httpClient.Timeout != expectedTimeout {
		t.Errorf("expected HTTP client timeout %v, got %v", expectedTimeout, service.httpClient.Timeout)
	}
}

func TestNewDownloadService_DefaultTimeout(t *testing.T) {
	cfg := config.Config{
		DiscogsTimeoutSec: 0, // Should use default
		GeneralVersion:    "1.0.0",
	}

	service := NewDownloadService(cfg)

	expectedTimeout := 30 * time.Second
	if service.httpClient.Timeout != expectedTimeout {
		t.Errorf("expected default HTTP client timeout %v, got %v", expectedTimeout, service.httpClient.Timeout)
	}
}

func TestParseChecksumFile(t *testing.T) {
	// Create a temporary checksum file for testing
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "CHECKSUM.txt")

	checksumContent := `# Discogs data dumps checksums
abc123def456  discogs_20240101_artists.xml.gz
def456ghi789  discogs_20240101_labels.xml.gz
ghi789jkl012  discogs_20240101_masters.xml.gz
jkl012mno345  discogs_20240101_releases.xml.gz
# Another comment line
xyz789uvw456  some_other_file.txt
`

	err := os.WriteFile(checksumFile, []byte(checksumContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test checksum file: %v", err)
	}

	cfg := config.Config{GeneralVersion: "1.0.0"}
	service := NewDownloadService(cfg)

	checksums, err := service.ParseChecksumFile(checksumFile)
	if err != nil {
		t.Fatalf("failed to parse checksum file: %v", err)
	}

	if checksums == nil {
		t.Fatal("expected checksums to be returned, got nil")
	}

	// Verify checksums were parsed correctly
	expectedChecksums := &models.FileChecksums{
		ArtistsDump:  "abc123def456",
		LabelsDump:   "def456ghi789",
		MastersDump:  "ghi789jkl012",
		ReleasesDump: "jkl012mno345",
	}

	if checksums.ArtistsDump != expectedChecksums.ArtistsDump {
		t.Errorf("expected ArtistsDump %s, got %s", expectedChecksums.ArtistsDump, checksums.ArtistsDump)
	}

	if checksums.LabelsDump != expectedChecksums.LabelsDump {
		t.Errorf("expected LabelsDump %s, got %s", expectedChecksums.LabelsDump, checksums.LabelsDump)
	}

	if checksums.MastersDump != expectedChecksums.MastersDump {
		t.Errorf("expected MastersDump %s, got %s", expectedChecksums.MastersDump, checksums.MastersDump)
	}

	if checksums.ReleasesDump != expectedChecksums.ReleasesDump {
		t.Errorf("expected ReleasesDump %s, got %s", expectedChecksums.ReleasesDump, checksums.ReleasesDump)
	}
}

func TestParseChecksumFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	checksumFile := filepath.Join(tmpDir, "empty.txt")

	// Create empty file
	err := os.WriteFile(checksumFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create test checksum file: %v", err)
	}

	cfg := config.Config{GeneralVersion: "1.0.0"}
	service := NewDownloadService(cfg)

	_, err = service.ParseChecksumFile(checksumFile)
	if err == nil {
		t.Fatal("expected error for empty checksum file, got nil")
	}

	if !strings.Contains(err.Error(), "empty or invalid checksum file") {
		t.Errorf("expected 'empty or invalid checksum file' error, got: %v", err)
	}
}

func TestParseChecksumFile_NonExistentFile(t *testing.T) {
	cfg := config.Config{GeneralVersion: "1.0.0"}
	service := NewDownloadService(cfg)

	_, err := service.ParseChecksumFile("/nonexistent/path/checksum.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("expected 'no such file or directory' error, got: %v", err)
	}
}

func TestIsValidYearMonth(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"2024-01", true},
		{"2024-12", true},
		{"1999-06", true},
		{"2024-1", false},   // month should be 2 digits
		{"24-01", false},    // year should be 4 digits
		{"2024/01", false},  // wrong separator
		{"2024-13", true},   // validation doesn't check month range (by design)
		{"2024", false},     // missing month
		{"", false},         // empty string
		{"2024-01-01", false}, // too many parts
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := isValidYearMonth(tc.input)
			if result != tc.expected {
				t.Errorf("isValidYearMonth(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestDownloadService_EnsureDirectory(t *testing.T) {
	cfg := config.Config{GeneralVersion: "1.0.0"}
	service := NewDownloadService(cfg)

	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test", "nested", "dir")

	// Directory should not exist initially
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatal("test directory should not exist initially")
	}

	// Create directory
	err := service.ensureDirectory(testDir)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Directory should exist now
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatal("directory should exist after creation")
	}

	// Calling again should not fail
	err = service.ensureDirectory(testDir)
	if err != nil {
		t.Fatalf("second call to ensureDirectory should not fail: %v", err)
	}
}

func TestDownloadChecksum_InvalidYearMonth(t *testing.T) {
	cfg := config.Config{GeneralVersion: "1.0.0"}
	service := NewDownloadService(cfg)

	ctx := context.Background()

	err := service.DownloadChecksum(ctx, "invalid-format")
	if err == nil {
		t.Fatal("expected error for invalid yearMonth format, got nil")
	}

	if !strings.Contains(err.Error(), "expected YYYY-MM format") {
		t.Errorf("expected 'expected YYYY-MM format' error, got: %v", err)
	}
}