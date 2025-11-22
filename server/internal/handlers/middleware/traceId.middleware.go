package middleware

import (
	logger "github.com/Bparsons0904/goLogger"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// TraceIDHeader is the HTTP header for trace ID
	TraceIDHeader = "X-Trace-ID"

	// TraceIDLocalKey is the Fiber locals key for trace ID
	TraceIDLocalKey = "traceID"
)

// TraceID middleware extracts or generates a trace ID for request tracking
func (m *Middleware) TraceID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check for incoming trace ID from header
		traceID := c.Get(TraceIDHeader)

		// Generate a new trace ID if not provided
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Add trace ID to response header for client visibility
		c.Set(TraceIDHeader, traceID)

		// Store in Fiber locals for easy access in handlers
		c.Locals(TraceIDLocalKey, traceID)

		// Add to Go context for use with logger.TraceFromContext()
		ctx := logger.ContextWithTraceID(c.Context(), traceID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// GetTraceID retrieves the trace ID from Fiber context
func GetTraceID(c *fiber.Ctx) string {
	if traceID, ok := c.Locals(TraceIDLocalKey).(string); ok {
		return traceID
	}
	return ""
}
