// Package logger provides backward compatibility by re-exporting from pkg/logger
// Deprecated: Use waugzee/pkg/logger instead for new code
package logger

import (
	"log/slog"

	pkglogger "waugzee/pkg/logger"
)

// Re-export types from pkg/logger
type (
	Logger = pkglogger.Logger
	Config = pkglogger.Config
	Format = pkglogger.Format
)

// Re-export constants from pkg/logger
const (
	DefaultTraceIDKey = pkglogger.DefaultTraceIDKey
	FormatJSON        = pkglogger.FormatJSON
	FormatText        = pkglogger.FormatText
)

// Re-export functions from pkg/logger
var (
	New                    = pkglogger.New
	NewWithConfig          = pkglogger.NewWithConfig
	NewWithContext         = pkglogger.NewWithContext
	ContextWithTraceID     = pkglogger.ContextWithTraceID
	ContextWithTraceIDName = pkglogger.ContextWithTraceIDName
	TraceIDFromContext     = pkglogger.TraceIDFromContext
	TraceIDFromContextName = pkglogger.TraceIDFromContextName
)

// Deprecated: Use pkg/logger.Config instead
type LegacyConfig struct {
	Name      string
	Format    Format
	Level     slog.Level
	AddSource bool
}
