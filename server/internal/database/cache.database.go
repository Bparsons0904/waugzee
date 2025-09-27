package database

import (
	"context"
	"fmt"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"

	"github.com/valkey-io/valkey-go"
)

const (
	GENERAL_CACHE_INDEX = iota
	SESSION_CACHE_INDEX // This can be repurposed to something else
	USER_CACHE_INDEX
	EVENTS_CACHE_INDEX
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
