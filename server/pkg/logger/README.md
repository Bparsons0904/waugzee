# Logger Package

A flexible, structured logging package built on Go's `slog` with built-in support for trace IDs and request tracking.

## Features

- **Structured Logging**: Built on Go's standard `log/slog` package
- **TraceID Support**: First-class support for request tracing across your application
- **Flexible Configuration**: Configure format, level, output destination, and more
- **Method Chaining**: Fluent API for adding context to your logs
- **Multiple Output Formats**: JSON or text format
- **Context Integration**: Extract trace IDs directly from context
- **Zero Dependencies**: Only uses Go standard library (except for testing)

## Installation

```bash
go get your-module/pkg/logger
```

## Quick Start

### Basic Usage

```go
package main

import "your-module/pkg/logger"

func main() {
    log := logger.New("my-service")

    log.Info("Application started")
    log.Debug("Debug information", "key", "value")
    log.Warn("Warning message", "reason", "low memory")
    log.Error("Error occurred", "error", err)
}
```

### With Configuration

```go
import (
    "log/slog"
    "os"
    "your-module/pkg/logger"
)

config := logger.Config{
    Name:      "my-service",
    Format:    logger.FormatJSON,
    Level:     slog.LevelDebug,
    Writer:    os.Stdout,
    AddSource: true,
}

log := logger.NewWithConfig(config)
log.Info("Configured logger ready")
```

## TraceID Support

### Method 1: Explicit TraceID

```go
log := logger.New("user-service")

// Add traceID to all subsequent logs
tracedLog := log.WithTraceID("req-123-abc")

tracedLog.Info("Processing request")  // Includes traceID=req-123-abc
tracedLog.Debug("Cache hit")          // Includes traceID=req-123-abc
tracedLog.Error("Database error")     // Includes traceID=req-123-abc
```

### Method 2: From Context (Recommended)

```go
// In middleware: add traceID to context
ctx := logger.ContextWithTraceID(ctx, "req-123-abc")

// In handler/service: create logger from context
log := logger.New("user-service").TraceFromContext(ctx)

log.Info("User logged in")  // Automatically includes traceID=req-123-abc
```

### Method 3: Custom Context Key

```go
// Add traceID with custom key
ctx := logger.ContextWithTraceIDName(ctx, "x-request-id", "req-456-def")

// Extract using custom key
log := logger.New("api-service").TraceFromContextName(ctx, "x-request-id")

log.Info("API call processed")  // Includes traceID=req-456-def
```

## HTTP Middleware Example

### Fiber Framework

```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "your-module/pkg/logger"
)

func TraceID() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Get or generate trace ID
        traceID := c.Get("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.New().String()
        }

        // Add to response header
        c.Set("X-Trace-ID", traceID)

        // Add to context
        ctx := logger.ContextWithTraceID(c.Context(), traceID)
        c.SetUserContext(ctx)

        return c.Next()
    }
}

// Usage in handler
func HandleRequest(c *fiber.Ctx) error {
    log := logger.New("api").TraceFromContext(c.Context())

    log.Info("Request received", "path", c.Path())
    // All logs will include the traceID

    return c.JSON(fiber.Map{"status": "ok"})
}
```

### Standard net/http

```go
package middleware

import (
    "context"
    "net/http"
    "github.com/google/uuid"
    "your-module/pkg/logger"
)

func TraceIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get or generate trace ID
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.New().String()
        }

        // Add to response
        w.Header().Set("X-Trace-ID", traceID)

        // Add to context
        ctx := logger.ContextWithTraceID(r.Context(), traceID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Usage in handler
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    log := logger.New("api").TraceFromContext(r.Context())

    log.Info("Request received", "path", r.URL.Path)
    // All logs will include the traceID
}
```

## Method Chaining

The logger supports method chaining for adding contextual information:

```go
log := logger.New("user-service").
    TraceFromContext(ctx).
    File("user.handler.go").
    Function("CreateUser")

log.Info("Creating new user", "email", email)
// Output includes: package=user-service, traceID=xxx, file=user.handler.go, function=CreateUser
```

## Available Methods

### Log Levels

```go
log.Debug("Debug message", "key", "value")
log.Info("Info message", "key", "value")
log.Warn("Warning message", "key", "value")
log.Error("Error message", "key", "value")
```

### Error Methods

```go
// Log and return error
err := log.Error("Something failed")

// Log with existing error
err = log.Err("Database connection failed", dbErr)

// Log without returning
log.Er("Operation failed", err)

// Simple error message
err = log.ErrMsg("Invalid input")
```

### Context Methods

```go
// Add arbitrary key-value pairs
log.With("userID", 123, "role", "admin")

// Add file context
log.File("user.service.go")

// Add function context
log.Function("ProcessPayment")

// Add traceID
log.WithTraceID("trace-123")
log.TraceFromContext(ctx)
log.TraceFromContextName(ctx, "custom-key")
```

### Utility Methods

```go
// Step logging (info level)
log.Step("Processing batch 1 of 10")

// Timer
done := log.Timer("database migration")
// ... do work ...
done()  // Logs duration
```

## Configuration Options

```go
type Config struct {
    // Name is the logger identifier (e.g., package or service name)
    Name string

    // Format specifies the output format (FormatJSON or FormatText)
    Format Format

    // Level specifies the minimum log level (slog.LevelDebug, LevelInfo, LevelWarn, LevelError)
    Level slog.Level

    // Writer is the output destination (defaults to os.Stderr if nil)
    Writer io.Writer

    // AddSource adds source code position to log output
    AddSource bool
}
```

### Format Options

- `logger.FormatJSON` - JSON structured logs (default)
- `logger.FormatText` - Human-readable text format

### Log Levels

- `slog.LevelDebug` - Detailed diagnostic information
- `slog.LevelInfo` - Informational messages (default)
- `slog.LevelWarn` - Warning messages
- `slog.LevelError` - Error messages

## Example Output

### JSON Format

```json
{
  "time": "2025-01-18T10:30:00Z",
  "level": "INFO",
  "msg": "User logged in",
  "package": "user-service",
  "traceID": "req-123-abc",
  "file": "user.handler.go",
  "function": "HandleLogin",
  "userID": 42,
  "email": "user@example.com"
}
```

### Text Format

```
2025-01-18T10:30:00Z INFO User logged in package=user-service traceID=req-123-abc file=user.handler.go function=HandleLogin userID=42 email=user@example.com
```

## Best Practices

1. **Create logger instances at initialization**: Create one logger per service/package and reuse it

```go
type UserService struct {
    log logger.Logger
}

func NewUserService() *UserService {
    return &UserService{
        log: logger.New("user-service"),
    }
}
```

2. **Use TraceFromContext in handlers**: Extract trace ID from context in your HTTP handlers

```go
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) error {
    log := s.log.TraceFromContext(ctx)
    log.Info("Creating user", "email", req.Email)
    // ...
}
```

3. **Chain context early**: Add file/function context early in method chains

```go
log := s.log.
    TraceFromContext(ctx).
    File("user.service.go").
    Function("CreateUser")
```

4. **Use structured logging**: Always use key-value pairs instead of string concatenation

```go
// Good
log.Info("User created", "userID", user.ID, "email", user.Email)

// Avoid
log.Info(fmt.Sprintf("User created: %s (%s)", user.ID, user.Email))
```

5. **Leverage timers**: Use the built-in timer for performance tracking

```go
done := log.Timer("database migration")
defer done()
// ... perform migration ...
```

## Testing

The logger automatically detects test mode and discards output during tests. You can also create test loggers:

```go
func TestMyFunction(t *testing.T) {
    log := logger.NewWithConfig(logger.Config{
        Name:   "test",
        Format: logger.FormatText,
        Writer: io.Discard,  // Or use a buffer to capture logs
    })

    // Your test code
}
```

## Migration from internal/logger

If you're migrating from `internal/logger` to `pkg/logger`, the API is 100% backward compatible. Simply update your imports:

```go
// Old
import "your-module/internal/logger"

// New
import "your-module/pkg/logger"
```

The `New()` function maintains backward compatibility with environment variable configuration.

## License

This package is part of your project and follows the same license.
