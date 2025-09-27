package services

import "time"

// Cache hash patterns
const (
	API_HASH             = "api_request"
	COLLECTION_SYNC_HASH = "collection_sync"
	RELEASE_QUEUE_HASH   = "release_queue"
)

// API configuration
const (
	APIRequestTTL     = 10 * time.Minute
	DiscogsAPIBaseURL = "https://api.discogs.com"
)