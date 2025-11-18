package logger_test

import (
	"context"
	"log/slog"
	"os"

	"waugzee/pkg/logger"
)

// Example of basic logger usage
func ExampleNew() {
	log := logger.New("my-service")

	log.Info("Application started")
	log.Debug("Configuration loaded", "config", "production")
	log.Warn("Cache miss", "key", "user:123")
}

// Example of logger with custom configuration
func ExampleNewWithConfig() {
	config := logger.Config{
		Name:      "api-service",
		Format:    logger.FormatJSON,
		Level:     slog.LevelDebug,
		Writer:    os.Stdout,
		AddSource: false,
	}

	log := logger.NewWithConfig(config)
	log.Info("Service initialized with custom config")
}

// Example of using traceID explicitly
func ExampleSlogLogger_WithTraceID() {
	log := logger.New("user-service")

	tracedLog := log.WithTraceID("req-abc-123")

	tracedLog.Info("Processing user request")
	tracedLog.Debug("Fetching from database")
	tracedLog.Info("Request completed")
}

// Example of extracting traceID from context
func ExampleSlogLogger_TraceFromContext() {
	ctx := context.Background()

	// Add traceID to context (typically done in middleware)
	ctx = logger.ContextWithTraceID(ctx, "req-xyz-789")

	// Create logger with traceID from context
	log := logger.New("order-service").TraceFromContext(ctx)

	log.Info("Creating order", "orderID", 12345)
	log.Info("Payment processed")
}

// Example of using custom context key for traceID
func ExampleSlogLogger_TraceFromContextName() {
	ctx := context.Background()

	// Add traceID with custom key name
	ctx = logger.ContextWithTraceIDName(ctx, "x-request-id", "custom-trace-999")

	// Extract using the same custom key
	log := logger.New("payment-service").TraceFromContextName(ctx, "x-request-id")

	log.Info("Payment initiated", "amount", 100.50)
}

// Example of method chaining
func ExampleSlogLogger_chaining() {
	ctx := logger.ContextWithTraceID(context.Background(), "req-123")

	log := logger.New("user-service").
		TraceFromContext(ctx).
		File("user.handler.go").
		Function("CreateUser")

	log.Info("Creating new user", "email", "user@example.com")
	log.Info("User created successfully", "userID", 42)
}

// Example of using timer for performance tracking
func ExampleSlogLogger_Timer() {
	log := logger.New("database-service")

	done := log.Timer("user migration")

	// Simulate work
	// Migration logic here...

	done() // Logs the duration
}

// Example of error handling
func ExampleSlogLogger_errorHandling() {
	log := logger.New("auth-service")

	// Log and return error
	if err := log.Error("Authentication failed", "reason", "invalid token"); err != nil {
		// Handle error
	}

	// Log existing error
	someErr := logger.New("db").ErrMsg("Connection timeout")
	log.Err("Database operation failed", someErr)

	// Log error without returning
	log.Er("Background task failed", someErr, "task", "cleanup")
}

// Example middleware pattern for Fiber
func Example_fiberMiddleware() {
	// This example shows the pattern - actual implementation would be in middleware package
	/*
		package middleware

		import (
			"github.com/gofiber/fiber/v2"
			"github.com/google/uuid"
			"waugzee/pkg/logger"
		)

		func TraceID() fiber.Handler {
			return func(c *fiber.Ctx) error {
				// Extract or generate traceID
				traceID := c.Get("X-Trace-ID")
				if traceID == "" {
					traceID = uuid.New().String()
				}

				// Add to response headers
				c.Set("X-Trace-ID", traceID)

				// Add to context for downstream handlers
				ctx := logger.ContextWithTraceID(c.Context(), traceID)
				c.SetUserContext(ctx)

				return c.Next()
			}
		}

		// Usage in handler
		func HandleCreateUser(c *fiber.Ctx) error {
			log := logger.New("user-handler").TraceFromContext(c.Context())

			log.Info("Create user request received", "path", c.Path())

			// All logs will automatically include the traceID
			log.Debug("Validating request")
			log.Info("User created successfully")

			return c.JSON(fiber.Map{"status": "created"})
		}
	*/
}

// Example of context helpers
func Example_contextHelpers() {
	ctx := context.Background()

	// Add traceID with default key
	ctx = logger.ContextWithTraceID(ctx, "trace-123")

	// Extract traceID
	traceID := logger.TraceIDFromContext(ctx)
	println(traceID) // "trace-123"

	// Add traceID with custom key
	ctx = logger.ContextWithTraceIDName(ctx, "request_id", "req-456")

	// Extract with custom key
	requestID := logger.TraceIDFromContextName(ctx, "request_id")
	println(requestID) // "req-456"
}
