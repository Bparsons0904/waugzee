package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	logger "github.com/Bparsons0904/goLogger"

	"waugzee/internal/types"
)

type LoggingService struct {
	victoriaLogsURL string
	httpClient      *http.Client
	log             logger.Logger
	enabled         bool
}

func NewLoggingService(victoriaLogsURL string) *LoggingService {
	enabled := victoriaLogsURL != ""

	log := logger.New("loggingService")
	if !enabled {
		log.Warn("VictoriaLogs URL not configured, client logging disabled")
	} else {
		log.Info("Logging service initialized", "url", victoriaLogsURL)
	}

	return &LoggingService{
		victoriaLogsURL: victoriaLogsURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log:     log,
		enabled: enabled,
	}
}

// ProcessLogBatch receives client logs, enriches them, and sends to VictoriaLogs
func (s *LoggingService) ProcessLogBatch(
	ctx context.Context,
	batch types.LogBatchRequest,
	userID string,
) (*types.LogBatchResponse, error) {
	log := s.log.Function("ProcessLogBatch")

	if !s.enabled {
		log.Debug("Logging disabled, skipping batch", "count", len(batch.Logs))
		return &types.LogBatchResponse{
			Success:   true,
			Processed: 0,
		}, nil
	}

	if len(batch.Logs) == 0 {
		return &types.LogBatchResponse{
			Success:   true,
			Processed: 0,
		}, nil
	}

	// Convert to VictoriaLogs format
	var entries []types.VictoriaLogsEntry
	for _, logEntry := range batch.Logs {
		entry := s.convertToVictoriaLogsEntry(logEntry, batch.SessionID, userID)
		entries = append(entries, entry)
	}

	// Send to VictoriaLogs
	if err := s.sendToVictoriaLogs(ctx, entries); err != nil {
		log.Er("Failed to send logs to VictoriaLogs", err,
			"count", len(entries),
			"userID", userID,
			"sessionID", batch.SessionID)
		return nil, fmt.Errorf("failed to send logs: %w", err)
	}

	log.Debug("Successfully processed log batch",
		"count", len(entries),
		"userID", userID,
		"sessionID", batch.SessionID)

	return &types.LogBatchResponse{
		Success:   true,
		Processed: len(entries),
	}, nil
}

// convertToVictoriaLogsEntry converts a client log entry to VictoriaLogs format
func (s *LoggingService) convertToVictoriaLogsEntry(
	entry types.LogEntry,
	sessionID string,
	userID string,
) types.VictoriaLogsEntry {
	vlEntry := types.VictoriaLogsEntry{
		Time:         entry.Timestamp,
		Msg:          entry.Message,
		StreamFields: "source,app,level",
		Source:       "client",
		App:          "waugzee",
		Level:        string(entry.Level),
		UserID:       userID,
		SessionID:    sessionID,
		URL:          entry.Metadata.URL,
		UserAgent:    entry.Metadata.UserAgent,
	}

	if entry.Context != nil {
		vlEntry.Action = entry.Context.Action
		vlEntry.Component = entry.Context.Component
		vlEntry.TraceID = entry.Context.TraceID

		if entry.Context.Error != nil {
			vlEntry.ErrorMessage = entry.Context.Error.Message
			vlEntry.ErrorStack = entry.Context.Error.Stack
		}
	}

	return vlEntry
}

// sendToVictoriaLogs sends log entries to VictoriaLogs via HTTP
func (s *LoggingService) sendToVictoriaLogs(
	ctx context.Context,
	entries []types.VictoriaLogsEntry,
) error {
	// Build JSON lines payload
	var lines []string
	for _, entry := range entries {
		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			s.log.Er("Failed to marshal log entry", err)
			continue
		}
		lines = append(lines, string(jsonBytes))
	}

	if len(lines) == 0 {
		return nil
	}

	payload := strings.Join(lines, "\n")

	// Compress with gzip
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write([]byte(payload)); err != nil {
		return fmt.Errorf("failed to compress payload: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/insert/jsonline", s.victoriaLogsURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("Content-Encoding", "gzip")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("VictoriaLogs returned status %d", resp.StatusCode)
	}

	return nil
}

// IsEnabled returns whether the logging service is enabled
func (s *LoggingService) IsEnabled() bool {
	return s.enabled
}
