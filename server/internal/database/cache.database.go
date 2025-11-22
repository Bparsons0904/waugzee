package database

import (
	"context"
	"fmt"
	"time"
	"waugzee/config"
	logger "github.com/Bparsons0904/goLogger"

	"github.com/valkey-io/valkey-go"
)

// Valkey Database Index Organization
// Each database index provides logical separation for different cache categories
const (
	// GENERAL_CACHE_INDEX (DB 0) - General purpose caching
	// Used for miscellaneous cache operations that don't fit into specific categories
	GENERAL_CACHE_INDEX = iota

	// SESSION_CACHE_INDEX (DB 1) - Session management
	// Used for user sessions and authentication-related temporary data
	SESSION_CACHE_INDEX

	// USER_CACHE_INDEX (DB 2) - All user-related data
	// Consolidates all user-specific caches including:
	// - User profiles and OIDC mappings
	// - User folders and releases
	// - Play and cleaning history
	// - Stylus tracking
	// - Daily recommendations and streaks
	USER_CACHE_INDEX

	// EVENTS_CACHE_INDEX (DB 3) - Event-driven data
	// Used for event sourcing, notifications, and real-time updates
	EVENTS_CACHE_INDEX

	// CLIENT_API_CACHE_INDEX (DB 4) - External API responses
	// Reserved for caching responses from external services (Discogs, etc.)
	CLIENT_API_CACHE_INDEX
)

func (s *DB) initializeCacheDB(config config.Config) error {
	log := s.log.Function("initializeCacheDB")
	log.Info("initializing cache database")

	address := config.DatabaseCacheAddress
	port := config.DatabaseCachePort
	if address == "" || port == 0 {
		return log.Errorf("failed to initialize cache database", "address or port is empty")
	}

	var cacheDB Cache

	var err error
	cacheDB.General, err = valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", address, port)},
			SelectDB:    GENERAL_CACHE_INDEX,
		},
	)
	if err != nil {
		return log.Err("failed to create general valkey client", err)
	}

	cacheDB.Session, err = valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", address, port)},
			SelectDB:    SESSION_CACHE_INDEX,
		},
	)
	if err != nil {
		return log.Err("failed to create session valkey client", err)
	}

	cacheDB.User, err = valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", address, port)},
			SelectDB:    USER_CACHE_INDEX,
		},
	)
	if err != nil {
		return log.Err("failed to create user valkey client", err)
	}

	cacheDB.Events, err = valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", address, port)},
			SelectDB:    EVENTS_CACHE_INDEX,
		},
	)
	if err != nil {
		return log.Err("failed to create events valkey client", err)
	}

	cacheDB.ClientAPI, err = valkey.NewClient(
		valkey.ClientOption{
			InitAddress: []string{fmt.Sprintf("%s:%d", address, port)},
			SelectDB:    CLIENT_API_CACHE_INDEX,
		},
	)
	if err != nil {
		return log.Err("failed to create client api valkey client", err)
	}

	s.Cache = cacheDB

	if config.DatabaseCacheReset != -1 {
		go clearCacheDB(config.DatabaseCacheReset, cacheDB)
	}

	return nil
}

func clearCacheDB(index int, cacheDB Cache) {
	log := logger.New("database").File("cache.database").Function("clearCacheDB")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var client CacheClient
	var dbName string

	switch index {
	case GENERAL_CACHE_INDEX:
		client = cacheDB.General
		dbName = "General"
	case SESSION_CACHE_INDEX:
		client = cacheDB.Session
		dbName = "Session"
	case USER_CACHE_INDEX:
		client = cacheDB.User
		dbName = "User"
	case EVENTS_CACHE_INDEX:
		client = cacheDB.Events
		dbName = "Events"
	case CLIENT_API_CACHE_INDEX:
		client = cacheDB.ClientAPI
		dbName = "ClientAPI"
	default:
		log.Warn("Invalid cache database index", "index", index)
		return
	}

	if err := client.Do(ctx, client.B().Flushdb().Build()).Error(); err != nil {
		log.Er("Failed to clear cache database", err, "index", index, "dbName", dbName)
		return
	}

	log.Info("Successfully cleared cache database", "index", index, "dbName", dbName)
}
