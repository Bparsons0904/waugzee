package types

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogContext contains optional context for a log entry
type LogContext struct {
	Action      string                 `json:"action,omitempty"`
	ReleaseID   string                 `json:"releaseId,omitempty"`
	StylusID    string                 `json:"stylusId,omitempty"`
	Component   string                 `json:"component,omitempty"`
	Duration    *int64                 `json:"duration,omitempty"`
	TraceID     string                 `json:"traceId,omitempty"`
	Error       *LogErrorContext       `json:"error,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// LogErrorContext contains error details
type LogErrorContext struct {
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
	Code    string `json:"code,omitempty"`
}

// LogMetadata contains browser/client metadata
type LogMetadata struct {
	UserAgent string       `json:"userAgent"`
	URL       string       `json:"url"`
	Referrer  string       `json:"referrer"`
	Viewport  ViewportSize `json:"viewport"`
}

// ViewportSize represents browser viewport dimensions
type ViewportSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// LogEntry represents a single log entry from the client
type LogEntry struct {
	Timestamp string       `json:"timestamp"`
	Level     LogLevel     `json:"level"`
	Message   string       `json:"message"`
	Context   *LogContext  `json:"context,omitempty"`
	Metadata  LogMetadata  `json:"metadata"`
}

// LogBatchRequest is the request body for batch log submission
type LogBatchRequest struct {
	Logs      []LogEntry `json:"logs"`
	SessionID string     `json:"sessionId"`
}

// LogBatchResponse is the response for batch log submission
type LogBatchResponse struct {
	Success   bool `json:"success"`
	Processed int  `json:"processed"`
}

// VictoriaLogsEntry is the format expected by VictoriaLogs JSON line ingestion
type VictoriaLogsEntry struct {
	Time          string `json:"_time"`
	Msg           string `json:"_msg"`
	StreamFields  string `json:"_stream_fields"`
	Source        string `json:"source"`
	App           string `json:"app"`
	Level         string `json:"level"`
	UserID        string `json:"userId,omitempty"`
	SessionID     string `json:"sessionId,omitempty"`
	TraceID       string `json:"traceId,omitempty"`
	Action        string `json:"action,omitempty"`
	Component     string `json:"component,omitempty"`
	URL           string `json:"url,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	ErrorStack    string `json:"errorStack,omitempty"`
}
