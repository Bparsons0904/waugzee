// Package logger_test tests backward compatibility of re-exports from pkg/logger
package logger

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that basic logger creation works via re-export
func TestNew_BackwardCompatibility(t *testing.T) {
	logger := New("test-package")

	assert.NotNil(t, logger)
}

// Test that traceID context functions work via re-export
func TestContextWithTraceID_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-123"

	ctx = ContextWithTraceID(ctx, traceID)
	extractedTraceID := TraceIDFromContext(ctx)

	assert.Equal(t, traceID, extractedTraceID)
}

// Test that custom key traceID functions work via re-export
func TestContextWithTraceIDName_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	traceID := "custom-trace-456"
	customKey := "request_id"

	ctx = ContextWithTraceIDName(ctx, customKey, traceID)
	extractedTraceID := TraceIDFromContextName(ctx, customKey)

	assert.Equal(t, traceID, extractedTraceID)
}

// Test basic logging methods work
func TestLogger_BasicMethods(t *testing.T) {
	logger := New("test")

	// Test error methods
	err := logger.Error("test error")
	assert.Error(t, err)

	// Test error with existing error
	originalErr := errors.New("original")
	returnedErr := logger.Err("context", originalErr)
	assert.Equal(t, originalErr, returnedErr)

	// Test method chaining
	chainedLogger := logger.With("key", "value")
	assert.NotNil(t, chainedLogger)

	fileLogger := logger.File("test.go")
	assert.NotNil(t, fileLogger)

	funcLogger := logger.Function("testFunc")
	assert.NotNil(t, funcLogger)
}

// Test traceID methods work via re-export
func TestLogger_TraceIDMethods(t *testing.T) {
	logger := New("test")

	// Test explicit traceID
	tracedLogger := logger.WithTraceID("trace-123")
	assert.NotNil(t, tracedLogger)

	// Test context-based traceID
	ctx := ContextWithTraceID(context.Background(), "context-trace")
	contextLogger := logger.TraceFromContext(ctx)
	assert.NotNil(t, contextLogger)

	// Test custom key context traceID
	customCtx := ContextWithTraceIDName(context.Background(), "custom-key", "custom-trace")
	customLogger := logger.TraceFromContextName(customCtx, "custom-key")
	assert.NotNil(t, customLogger)
}

// Test that timer functionality works
func TestLogger_Timer(t *testing.T) {
	logger := New("test")

	done := logger.Timer("test operation")
	assert.NotNil(t, done)

	// Should not panic when called
	done()
}
