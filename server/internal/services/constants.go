// Package services contains shared constants used across orchestration, folder, and release sync services.
// These constants define cache patterns, API configuration, and operational limits to prevent
// memory leaks and system abuse in the client-as-proxy vinyl collection sync workflow.
package services

import "time"

// Cache hash patterns used for organizing different types of cached data.
// These patterns are used with the CacheBuilder to create namespaced cache keys.
const (
	API_HASH             = "api_request"     // Stores metadata for pending API requests
	COLLECTION_SYNC_HASH = "collection_sync" // Stores collection synchronization state
	RELEASE_QUEUE_HASH   = "release_queue"   // Stores queued releases for processing
)

// API configuration for external Discogs API integration.
// Used in the client-as-proxy pattern where clients make API calls with their own tokens.
const (
	APIRequestTTL     = 10 * time.Minute       // TTL for API request metadata in cache
	DiscogsAPIBaseURL = "https://api.discogs.com" // Base URL for all Discogs API calls
)

// Collection sync limits to prevent memory leaks and system abuse.
// These limits ensure that large collections don't overwhelm the system during sync operations.
const (
	MaxReleasesPerSync    = 50000              // Maximum releases processed in a single sync operation
	MaxPendingAPIRequests = 1000               // Maximum concurrent API requests to prevent rate limiting
	MaxCollectionSyncTTL  = 2 * time.Hour     // Maximum time allowed for a complete sync operation
	SyncStateTTL          = 30 * time.Minute   // TTL for sync state cache entries during processing
)

// Circuit breaker configuration for external API resilience.
// These settings help prevent cascade failures when external APIs are unavailable.
const (
	MaxConsecutiveFailures = 5                 // Maximum failures before circuit opens
	CircuitBreakerTimeout  = 1 * time.Minute   // Time before circuit attempts to close
	APIHealthCheckInterval = 30 * time.Second  // Interval for health check attempts
)

// Default folder IDs as defined by Discogs API standard.
// These IDs are consistent across all Discogs users and used for filtering sync operations.
const (
	AllFolderID           = 0     // Discogs "All" folder (read-only, contains all items)
	UncategorizedFolderID = 1     // Discogs "Uncategorized" folder (default for new items)
	MaxPageNumber         = 10000 // Maximum page number allowed to prevent API abuse
)