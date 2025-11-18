# Migration Guide: internal/logger → pkg/logger

This guide will help you migrate from `internal/logger` to the new standalone `pkg/logger` package with traceID support.

## What's New

### 1. TraceID Support

The biggest addition is first-class support for trace IDs:

```go
// Extract from context
log := logger.New("service").TraceFromContext(ctx)

// Explicit traceID
log := logger.New("service").WithTraceID("trace-123")

// Custom context key
log := logger.New("service").TraceFromContextName(ctx, "request_id")
```

### 2. Configuration-Based Initialization

You can now create loggers without relying on environment variables:

```go
config := logger.Config{
    Name:      "my-service",
    Format:    logger.FormatJSON,
    Level:     slog.LevelInfo,
    Writer:    os.Stdout,
    AddSource: true,
}
log := logger.NewWithConfig(config)
```

### 3. Standalone Package

The logger is now in `pkg/logger` making it:
- Reusable across multiple projects
- Independently testable
- Free from internal dependencies

## Migration Steps

### Step 1: Update Imports

**Before:**
```go
import "waugzee/internal/logger"
```

**After:**
```go
import "waugzee/pkg/logger"
```

### Step 2: Update Logger Creation (Optional)

The existing `logger.New()` API is **100% backward compatible**. No changes required unless you want to use the new configuration API.

**Current (Still Works):**
```go
log := logger.New("user-service")
```

**New Configuration API (Optional):**
```go
log := logger.NewWithConfig(logger.Config{
    Name:   "user-service",
    Format: logger.FormatJSON,
    Level:  slog.LevelInfo,
})
```

### Step 3: Add TraceID Support

#### Option A: Add Middleware (Recommended)

Create or update your middleware to inject traceIDs:

```go
// server/internal/handlers/middleware/traceId.go
package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"waugzee/pkg/logger"
)

func (m *MiddlewareImpl) TraceID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract from header or generate
		traceID := c.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Add to response
		c.Set("X-Trace-ID", traceID)

		// Add to context
		ctx := logger.ContextWithTraceID(c.Context(), traceID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}
```

#### Option B: Update Handlers

In your handlers, extract the traceID from context:

**Before:**
```go
func (h *Handler) HandleRequest(c *fiber.Ctx) error {
    log := logger.New("user-handler")
    log.Info("Processing request")
    // ...
}
```

**After:**
```go
func (h *Handler) HandleRequest(c *fiber.Ctx) error {
    log := logger.New("user-handler").TraceFromContext(c.Context())
    log.Info("Processing request")  // Now includes traceID
    // ...
}
```

#### Option C: Update Controllers

If your controllers store logger instances:

**Before:**
```go
type UserController struct {
    log logger.Logger
}

func (c *UserController) CreateUser(ctx context.Context, req Request) error {
    c.log.Info("Creating user")
    // ...
}
```

**After:**
```go
type UserController struct {
    log logger.Logger
}

func (c *UserController) CreateUser(ctx context.Context, req Request) error {
    log := c.log.TraceFromContext(ctx)
    log.Info("Creating user")  // Now includes traceID
    // ...
}
```

## Waugzee-Specific Migration

### 1. Update Middleware Registration

In `server/internal/handlers/router.go`:

```go
// Add TraceID middleware
app.Use(m.TraceID())
```

### 2. Update Controllers

Controllers already receive context in most methods. Just add `.TraceFromContext(ctx)`:

**Example - Auth Controller:**

```go
// Before
func (c *AuthController) HandleOIDCCallback(ctx context.Context, req OIDCCallbackRequest) (*TokenExchangeResult, error) {
    c.log.Info("Processing OIDC callback")
    // ...
}

// After
func (c *AuthController) HandleOIDCCallback(ctx context.Context, req OIDCCallbackRequest) (*TokenExchangeResult, error) {
    log := c.log.TraceFromContext(ctx)
    log.Info("Processing OIDC callback")
    // ...
}
```

### 3. Update Services

Services should also accept context and extract traceID:

```go
// Before
func (s *UserService) GetUser(userID string) (*User, error) {
    s.log.Info("Fetching user", "userID", userID)
    // ...
}

// After
func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    log := s.log.TraceFromContext(ctx)
    log.Info("Fetching user", "userID", userID)
    // ...
}
```

## Automated Migration Script

You can use this bash script to update all imports:

```bash
#!/bin/bash

# Update all Go files to use pkg/logger
find server -type f -name "*.go" -exec sed -i 's|"waugzee/internal/logger"|"waugzee/pkg/logger"|g' {} +

echo "Migration complete! Please review changes and run tests."
```

## Testing

After migration, run tests to ensure everything works:

```bash
# Run all tests
go test ./...

# Run logger tests specifically
go test ./pkg/logger/...

# Run with coverage
go test -cover ./...
```

## Rollback Plan

If you need to rollback:

1. The old `internal/logger` package is still available
2. Simply revert import changes
3. No breaking changes were made to the API

## Benefits of Migration

### Immediate Benefits

1. **Distributed Tracing**: Track requests across your entire application
2. **Improved Debugging**: Correlate logs from different services/layers
3. **Better Observability**: Trace IDs make it easy to filter logs in production
4. **Standalone Package**: Reusable in other projects

### Example Log Output

**Before:**
```json
{
  "time": "2025-01-18T10:30:00Z",
  "level": "INFO",
  "msg": "User logged in",
  "package": "auth-controller",
  "userID": 42
}
```

**After (with traceID):**
```json
{
  "time": "2025-01-18T10:30:00Z",
  "level": "INFO",
  "msg": "User logged in",
  "package": "auth-controller",
  "traceID": "req-7f8a9b2c-4d5e-6f7g-8h9i-0j1k2l3m4n5o",
  "userID": 42
}
```

Now you can search for `traceID=req-7f8a9b2c...` and see ALL logs related to that specific request across all services!

## Gradual Migration

You don't have to migrate everything at once:

### Phase 1: Update Imports
- Update all imports from `internal/logger` to `pkg/logger`
- No code changes needed
- Run tests to ensure nothing broke

### Phase 2: Add Middleware
- Create TraceID middleware
- Add to router
- TraceIDs now available in context

### Phase 3: Update Critical Paths
- Update authentication flows
- Update API endpoints
- Update background jobs

### Phase 4: Update Everything
- Update all controllers
- Update all services
- Update all repositories

## Common Patterns

### Pattern 1: Handler with TraceID

```go
func (h *Handler) HandleRequest(c *fiber.Ctx) error {
    log := logger.New("handler").TraceFromContext(c.Context())

    log.Info("Request received", "path", c.Path())

    result, err := h.service.Process(c.Context(), req)
    if err != nil {
        return log.Err("Processing failed", err)
    }

    log.Info("Request completed")
    return c.JSON(result)
}
```

### Pattern 2: Service with Context Propagation

```go
func (s *Service) Process(ctx context.Context, data Data) error {
    log := s.log.TraceFromContext(ctx).Function("Process")

    log.Debug("Starting processing")

    // Pass context to repository
    if err := s.repo.Save(ctx, data); err != nil {
        return log.Err("Save failed", err)
    }

    log.Info("Processing complete")
    return nil
}
```

### Pattern 3: Repository with TraceID

```go
func (r *Repository) Save(ctx context.Context, data Data) error {
    log := r.log.TraceFromContext(ctx).Function("Save")

    log.Debug("Saving to database")

    result := r.db.Create(&data)
    if result.Error != nil {
        return log.Err("Database error", result.Error)
    }

    log.Info("Saved successfully", "id", data.ID)
    return nil
}
```

## Need Help?

- Check the [README.md](./README.md) for full API documentation
- Review [examples_test.go](./examples_test.go) for usage examples
- Existing tests in [logger_test.go](./logger_test.go) show all features

## Breaking Changes

**None!** The migration is 100% backward compatible. The only change is the import path.
