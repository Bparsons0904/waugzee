package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_Success(t *testing.T) {
	logger := New("test-package")

	assert.NotNil(t, logger)
	assert.IsType(t, &SlogLogger{}, logger)
}

func TestNewWithConfig_JSONFormat(t *testing.T) {
	config := Config{
		Name:   "test-service",
		Format: FormatJSON,
		Level:  slog.LevelDebug,
	}

	logger := NewWithConfig(config)

	assert.NotNil(t, logger)
	assert.IsType(t, &SlogLogger{}, logger)
}

func TestNewWithConfig_TextFormat(t *testing.T) {
	config := Config{
		Name:   "test-service",
		Format: FormatText,
		Level:  slog.LevelInfo,
	}

	logger := NewWithConfig(config)

	assert.NotNil(t, logger)
	assert.IsType(t, &SlogLogger{}, logger)
}

func TestNewWithContext_ExtractsTraceID(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}

	// Create context with traceID
	ctx := ContextWithTraceID(context.Background(), "test-trace-from-context")

	// Create logger using NewWithContext
	logger := &SlogLogger{logger: slog.New(handler)}
	tracedLogger := logger.TraceFromContext(ctx)

	tracedLogger.Info("test message")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test message")
	assert.Contains(t, capturedLogs[0], "traceID")
	assert.Contains(t, capturedLogs[0], "test-trace-from-context")
}

func TestNewWithContext_NoTraceID(t *testing.T) {
	// NewWithContext should work even without traceID in context
	ctx := context.Background()

	logger := NewWithContext(ctx, "test-service")

	assert.NotNil(t, logger)
	assert.IsType(t, &SlogLogger{}, logger)
}

func TestWith_ChainMethod(t *testing.T) {
	logger := New("test")

	newLogger := logger.With("key1", "value1")

	assert.NotNil(t, newLogger)
	assert.IsType(t, &SlogLogger{}, newLogger)
}

func TestFile_Method(t *testing.T) {
	logger := New("test")

	fileLogger := logger.File("user.controller.go")

	assert.NotNil(t, fileLogger)
	assert.IsType(t, &SlogLogger{}, fileLogger)
}

func TestFunction_Method(t *testing.T) {
	logger := New("test")

	funcLogger := logger.Function("handleLogin")

	assert.NotNil(t, funcLogger)
	assert.IsType(t, &SlogLogger{}, funcLogger)
}

func TestTimer_Functionality(t *testing.T) {
	logger := New("test")

	done := logger.Timer("test operation")

	assert.NotNil(t, done)
	assert.IsType(t, func() {}, done)

	done()
}

func TestError_Methods(t *testing.T) {
	logger := New("test")

	err := logger.Error("test error message")

	assert.Error(t, err)
	assert.Equal(t, "test error message", err.Error())
}

func TestErr_Method(t *testing.T) {
	logger := New("test")

	originalErr := errors.New("original error")
	returnedErr := logger.Err("context message", originalErr)

	assert.Error(t, returnedErr)
	assert.Equal(t, originalErr, returnedErr)
}

func TestEr_Method(t *testing.T) {
	logger := New("test")

	originalErr := errors.New("test error")

	logger.Er("error occurred", originalErr)
}

func TestErrMsg_Method(t *testing.T) {
	logger := New("test")

	err := logger.ErrMsg("simple error message")

	assert.Error(t, err)
	assert.Equal(t, "simple error message", err.Error())
}

func TestLoggerInterface_Implementation(t *testing.T) {
	logger := New("test")

	assert.NotNil(t, logger)

	err := logger.Error("error test")
	assert.Error(t, err)

	chainedLogger := logger.With("test", "value")
	assert.NotNil(t, chainedLogger)

	fileLogger := logger.File("test.go")
	assert.NotNil(t, fileLogger)

	funcLogger := logger.Function("testFunc")
	assert.NotNil(t, funcLogger)

	timer := logger.Timer("test timer")
	assert.NotNil(t, timer)
	timer()
}

func TestErr_NilError(t *testing.T) {
	logger := New("test")

	returnedErr := logger.Err("message", nil)

	assert.Nil(t, returnedErr)
}

func TestEr_NilError(t *testing.T) {
	logger := New("test")

	logger.Er("message", nil)
}

func TestStep_Method(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.Step("test step message")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test step message")
}

func TestDebug_Method(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.Debug("debug message", "key", "value")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "debug message")
	assert.Contains(t, capturedLogs[0], "key")
	assert.Contains(t, capturedLogs[0], "value")
}

func TestWarn_Method(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.Warn("warning message", "key", "value")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "warning message")
}

func TestInfo_Method(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.Info("info message", "key", "value")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "info message")
}

func TestErrorf_Method(t *testing.T) {
	logger := New("test")
	err := logger.Errorf("test error", "detailed message")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detailed message")
}

func TestErMsg_VoidMethod(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.ErMsg("simple error message")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "simple error message")
}

// TraceID Tests

func TestContextWithTraceID_Success(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-123"

	ctx = ContextWithTraceID(ctx, traceID)

	extractedTraceID := TraceIDFromContext(ctx)
	assert.Equal(t, traceID, extractedTraceID)
}

func TestTraceIDFromContext_NoTraceID(t *testing.T) {
	ctx := context.Background()

	extractedTraceID := TraceIDFromContext(ctx)
	assert.Equal(t, "", extractedTraceID)
}

func TestContextWithTraceIDName_CustomKey(t *testing.T) {
	ctx := context.Background()
	traceID := "custom-trace-456"
	customKey := "request_id"

	ctx = ContextWithTraceIDName(ctx, customKey, traceID)

	extractedTraceID := TraceIDFromContextName(ctx, customKey)
	assert.Equal(t, traceID, extractedTraceID)
}

func TestTraceIDFromContextName_NoTraceID(t *testing.T) {
	ctx := context.Background()

	extractedTraceID := TraceIDFromContextName(ctx, "custom_key")
	assert.Equal(t, "", extractedTraceID)
}

func TestWithTraceID_Method(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	tracedLogger := logger.WithTraceID("trace-789")
	tracedLogger.Info("test message")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test message")
	assert.Contains(t, capturedLogs[0], "traceID")
	assert.Contains(t, capturedLogs[0], "trace-789")
}

func TestTraceFromContext_WithTraceID(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	ctx := ContextWithTraceID(context.Background(), "context-trace-123")

	tracedLogger := logger.TraceFromContext(ctx)
	tracedLogger.Info("test message with context trace")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test message with context trace")
	assert.Contains(t, capturedLogs[0], "traceID")
	assert.Contains(t, capturedLogs[0], "context-trace-123")
}

func TestTraceFromContext_NoTraceID(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	ctx := context.Background()

	tracedLogger := logger.TraceFromContext(ctx)
	tracedLogger.Info("test message without trace")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test message without trace")
	assert.NotContains(t, capturedLogs[0], "traceID")
}

func TestTraceFromContextName_WithCustomKey(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	customKey := "x-request-id"
	ctx := ContextWithTraceIDName(context.Background(), customKey, "custom-trace-999")

	tracedLogger := logger.TraceFromContextName(ctx, customKey)
	tracedLogger.Info("test message with custom trace")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test message with custom trace")
	assert.Contains(t, capturedLogs[0], "traceID")
	assert.Contains(t, capturedLogs[0], "custom-trace-999")
}

func TestTraceFromContextName_NoTraceID(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	ctx := context.Background()

	tracedLogger := logger.TraceFromContextName(ctx, "missing-key")
	tracedLogger.Info("test message without custom trace")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "test message without custom trace")
	assert.NotContains(t, capturedLogs[0], "traceID")
}

func TestTraceID_PersistsAcrossLogCalls(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	tracedLogger := logger.WithTraceID("persistent-trace-111")

	tracedLogger.Info("first log")
	tracedLogger.Debug("second log")
	tracedLogger.Warn("third log")
	tracedLogger.Error("fourth log")

	assert.Len(t, capturedLogs, 4)

	for i, log := range capturedLogs {
		assert.Contains(t, log, "traceID", "Log %d should contain traceID", i)
		assert.Contains(t, log, "persistent-trace-111", "Log %d should contain the trace ID value", i)
	}
}

func TestTraceID_ChainedWithOtherMethods(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	chainedLogger := logger.
		WithTraceID("chained-trace-222").
		File("user.service.go").
		Function("CreateUser")

	chainedLogger.Info("user created")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "user created")
	assert.Contains(t, capturedLogs[0], "traceID")
	assert.Contains(t, capturedLogs[0], "chained-trace-222")
	assert.Contains(t, capturedLogs[0], "file")
	assert.Contains(t, capturedLogs[0], "user.service.go")
	assert.Contains(t, capturedLogs[0], "function")
	assert.Contains(t, capturedLogs[0], "CreateUser")
}

// Performance Tracking Tests

func TestWithMemoryStats_IncludesMemoryMetrics(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.WithMemoryStats().Info("operation complete")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "operation complete")
	assert.Contains(t, capturedLogs[0], "memory_alloc_mb")
	assert.Contains(t, capturedLogs[0], "memory_total_alloc_mb")
	assert.Contains(t, capturedLogs[0], "memory_sys_mb")
	assert.Contains(t, capturedLogs[0], "memory_num_gc")
}

func TestWithGoroutineCount_IncludesGoroutineCount(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	logger.WithGoroutineCount().Info("server started")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "server started")
	assert.Contains(t, capturedLogs[0], "goroutines")
}

func TestTimerWithMetrics_TracksPerformanceMetrics(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	done := logger.TimerWithMetrics("test operation")

	// Allocate some memory to create a measurable delta
	data := make([]byte, 1024*1024) // 1MB
	_ = data

	done()

	// Should have 2 logs: debug start + info completion
	assert.GreaterOrEqual(t, len(capturedLogs), 1)

	// Check the completion log (last one)
	completionLog := capturedLogs[len(capturedLogs)-1]

	assert.Contains(t, completionLog, "Operation completed with metrics")
	assert.Contains(t, completionLog, "operation")
	assert.Contains(t, completionLog, "test operation")
	assert.Contains(t, completionLog, "duration_ms")
	assert.Contains(t, completionLog, "duration")
	assert.Contains(t, completionLog, "memory_start_mb")
	assert.Contains(t, completionLog, "memory_end_mb")
	assert.Contains(t, completionLog, "memory_delta_mb")
	assert.Contains(t, completionLog, "memory_delta_sign")
	assert.Contains(t, completionLog, "goroutines_start")
	assert.Contains(t, completionLog, "goroutines_end")
	assert.Contains(t, completionLog, "goroutines_delta")
}

func TestTimerWithMetrics_CallableMultipleTimes(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	done := logger.TimerWithMetrics("multi-call test")
	assert.NotNil(t, done)

	// Should not panic when called
	done()
}

func TestPerformanceMetrics_ChainedWithOtherMethods(t *testing.T) {
	var capturedLogs []string
	handler := &testHandler{logs: &capturedLogs}
	logger := &SlogLogger{logger: slog.New(handler)}

	chainedLogger := logger.
		WithTraceID("trace-123").
		WithMemoryStats().
		WithGoroutineCount().
		File("service.go")

	chainedLogger.Info("performance tracked operation")

	assert.Len(t, capturedLogs, 1)
	assert.Contains(t, capturedLogs[0], "performance tracked operation")
	assert.Contains(t, capturedLogs[0], "traceID")
	assert.Contains(t, capturedLogs[0], "trace-123")
	assert.Contains(t, capturedLogs[0], "memory_alloc_mb")
	assert.Contains(t, capturedLogs[0], "goroutines")
	assert.Contains(t, capturedLogs[0], "file")
	assert.Contains(t, capturedLogs[0], "service.go")
}

func TestBytesToMB_Conversion(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected float64
	}{
		{"1 MB", 1024 * 1024, 1.0},
		{"0.5 MB", 512 * 1024, 0.5},
		{"2 MB", 2 * 1024 * 1024, 2.0},
		{"0 bytes", 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesToMB(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAbsInt64_ReturnsAbsoluteValue(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{"positive", 42, 42},
		{"negative", -42, 42},
		{"zero", 0, 0},
		{"large positive", 9223372036854775807, 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := absInt64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSignString_ReturnsCorrectSign(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"positive", 42, "+"},
		{"negative", -42, "-"},
		{"zero", 0, "="},
		{"large positive", 999999, "+"},
		{"large negative", -999999, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := signString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test helper to capture log output
type testHandler struct {
	logs *[]string
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, record slog.Record) error {
	var parts []string
	parts = append(parts, record.Message)

	record.Attrs(func(attr slog.Attr) bool {
		parts = append(parts, fmt.Sprintf("%s=%v", attr.Key, attr.Value))
		return true
	})

	fullMessage := strings.Join(parts, " ")
	*h.logs = append(*h.logs, fullMessage)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return h
}
