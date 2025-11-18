package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type contextKey string

const (
	// DefaultTraceIDKey is the default context key for trace IDs
	DefaultTraceIDKey contextKey = "traceID"
)

// Format represents the logging output format
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Config holds logger configuration options
type Config struct {
	// Name is the logger identifier (e.g., package or service name)
	Name string

	// Format specifies the output format (json or text)
	Format Format

	// Level specifies the minimum log level
	Level slog.Level

	// Writer is the output destination (defaults to os.Stderr if nil)
	Writer io.Writer

	// AddSource adds source code position to log output
	AddSource bool
}

// Logger defines the logging interface
type Logger interface {
	Errorf(msg string, errMessage string) error
	Error(msg string, args ...any) error
	ErrorWithType(errType error, msg string, args ...any) error
	Err(msg string, err error, args ...any) error
	ErrMsg(msg string) error
	ErMsg(msg string)
	Er(msg string, err error, args ...any)
	Step(msg string)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Info(msg string, args ...any)
	With(args ...any) Logger
	File(name string) Logger
	Function(name string) Logger
	Timer(msg string) func()

	// TraceID methods
	WithTraceID(traceID string) Logger
	TraceFromContext(ctx context.Context) Logger
	TraceFromContextName(ctx context.Context, key string) Logger

	// Performance tracking methods
	WithMemoryStats() Logger
	WithGoroutineCount() Logger
	TimerWithMetrics(msg string) func()
}

// SlogLogger implements the Logger interface using slog
type SlogLogger struct {
	logger *slog.Logger
}

// New creates a new logger with the provided name using default configuration
// For backward compatibility - uses environment variables for configuration
func New(name string) Logger {
	var handler slog.Handler

	if isTestMode() {
		handler = slog.NewTextHandler(io.Discard, nil)
	} else {
		logFormat := os.Getenv("LOG_FORMAT")
		if logFormat == "text" {
			handler = slog.Default().Handler()
		} else {
			handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
		}
	}

	return &SlogLogger{
		logger: slog.New(handler).With("package", name),
	}
}

// NewWithConfig creates a new logger with the provided configuration
func NewWithConfig(config Config) Logger {
	var handler slog.Handler

	writer := config.Writer
	if writer == nil {
		writer = os.Stderr
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     config.Level,
		AddSource: config.AddSource,
	}

	switch config.Format {
	case FormatText:
		handler = slog.NewTextHandler(writer, handlerOpts)
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, handlerOpts)
	default:
		handler = slog.NewJSONHandler(writer, handlerOpts)
	}

	return &SlogLogger{
		logger: slog.New(handler).With("package", config.Name),
	}
}

func isTestMode() bool {
	for _, arg := range os.Args {
		if arg == "-test.v" || arg == "-test.run" || arg == "-test.bench" {
			return true
		}
	}
	return false
}

// ContextWithTraceID adds a trace ID to the context using the default key
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, DefaultTraceIDKey, traceID)
}

// ContextWithTraceIDName adds a trace ID to the context using a custom key
func ContextWithTraceIDName(ctx context.Context, key string, traceID string) context.Context {
	return context.WithValue(ctx, contextKey(key), traceID)
}

// TraceIDFromContext extracts the trace ID from context using the default key
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(DefaultTraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// TraceIDFromContextName extracts the trace ID from context using a custom key
func TraceIDFromContextName(ctx context.Context, key string) string {
	if traceID, ok := ctx.Value(contextKey(key)).(string); ok {
		return traceID
	}
	return ""
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}

func (l *SlogLogger) Error(msg string, args ...any) error {
	l.logger.Error(msg, args...)
	return fmt.Errorf("%s", msg)
}

func (l *SlogLogger) ErrorWithType(errType error, msg string, args ...any) error {
	l.logger.Error(msg, args...)
	return fmt.Errorf("%w: %s", errType, msg)
}

func (l *SlogLogger) File(name string) Logger {
	return l.With("file", name)
}

func (l *SlogLogger) Function(name string) Logger {
	return l.With("function", name)
}

func (l *SlogLogger) Timer(msg string) func() {
	start := time.Now()
	l.logger.Debug("Starting", "operation", msg)

	return func() {
		duration := time.Since(start)
		l.logger.Info("Timer Completed",
			"operation", msg,
			"duration_ms", duration.Milliseconds(),
			"duration", duration.String(),
		)
	}
}

func (l *SlogLogger) Errorf(msg string, errMessage string) error {
	err := fmt.Errorf("error: %s", errMessage)
	l.logger.Error(msg, "error", err)
	return err
}

func (l *SlogLogger) Er(msg string, err error, args ...any) {
	logArgs := append([]any{"error", err}, args...)
	l.logger.Error(msg, logArgs...)
}

func (l *SlogLogger) Err(msg string, err error, args ...any) error {
	logArgs := append([]any{"error", err}, args...)
	l.logger.Error(msg, logArgs...)
	return err
}

func (l *SlogLogger) ErMsg(msg string) {
	l.logger.Error(msg)
}

func (l *SlogLogger) ErrMsg(msg string) error {
	l.logger.Error(msg)
	return fmt.Errorf("%s", msg)
}

func (l *SlogLogger) Step(msg string) {
	l.logger.Info(msg)
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// WithTraceID adds a trace ID to all subsequent log calls
func (l *SlogLogger) WithTraceID(traceID string) Logger {
	return l.With("traceID", traceID)
}

// TraceFromContext extracts trace ID from context using the default key and adds it to the logger
func (l *SlogLogger) TraceFromContext(ctx context.Context) Logger {
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		return l
	}
	return l.WithTraceID(traceID)
}

// TraceFromContextName extracts trace ID from context using a custom key and adds it to the logger
func (l *SlogLogger) TraceFromContextName(ctx context.Context, key string) Logger {
	traceID := TraceIDFromContextName(ctx, key)
	if traceID == "" {
		return l
	}
	return l.WithTraceID(traceID)
}

// Performance tracking methods

// WithMemoryStats adds current memory statistics to the logger
func (l *SlogLogger) WithMemoryStats() Logger {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return l.With(
		"memory_alloc_mb", bytesToMB(m.Alloc),
		"memory_total_alloc_mb", bytesToMB(m.TotalAlloc),
		"memory_sys_mb", bytesToMB(m.Sys),
		"memory_num_gc", m.NumGC,
	)
}

// WithGoroutineCount adds current goroutine count to the logger
func (l *SlogLogger) WithGoroutineCount() Logger {
	return l.With("goroutines", runtime.NumGoroutine())
}

// TimerWithMetrics returns a function that logs operation duration along with memory and goroutine metrics
func (l *SlogLogger) TimerWithMetrics(msg string) func() {
	start := time.Now()

	// Capture starting metrics
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)
	startGoroutines := runtime.NumGoroutine()

	l.logger.Debug("Starting operation with metrics",
		"operation", msg,
		"memory_start_mb", bytesToMB(startMem.Alloc),
		"goroutines_start", startGoroutines,
	)

	return func() {
		// Capture ending metrics
		var endMem runtime.MemStats
		runtime.ReadMemStats(&endMem)
		endGoroutines := runtime.NumGoroutine()

		duration := time.Since(start)
		memDelta := int64(endMem.Alloc) - int64(startMem.Alloc)
		goroutineDelta := endGoroutines - startGoroutines

		l.logger.Info("Operation completed with metrics",
			"operation", msg,
			"duration_ms", duration.Milliseconds(),
			"duration", duration.String(),
			"memory_start_mb", bytesToMB(startMem.Alloc),
			"memory_end_mb", bytesToMB(endMem.Alloc),
			"memory_delta_mb", bytesToMB(uint64(absInt64(memDelta))),
			"memory_delta_sign", signString(memDelta),
			"goroutines_start", startGoroutines,
			"goroutines_end", endGoroutines,
			"goroutines_delta", goroutineDelta,
		)
	}
}

// Helper functions

// bytesToMB converts bytes to megabytes
func bytesToMB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}

// absInt64 returns the absolute value of an int64
func absInt64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// signString returns "+" for positive, "-" for negative, "=" for zero
func signString(n int64) string {
	if n > 0 {
		return "+"
	} else if n < 0 {
		return "-"
	}
	return "="
}
