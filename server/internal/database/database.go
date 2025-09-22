package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"

	"github.com/valkey-io/valkey-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type CacheClient valkey.Client

type Cache struct {
	General  CacheClient
	Session  CacheClient
	User     CacheClient
	Events   CacheClient
	LoadTest CacheClient
}

type DB struct {
	SQL   *gorm.DB
	Cache Cache
	log   logger.Logger
}

func New(config config.Config) (DB, error) {
	log := logger.New("database").Function("New")

	log.Info("Initializing database")
	db := &DB{log: log}

	err := db.initializeDB(config)
	if err != nil {
		return DB{}, log.Err("failed to initialize database", err)
	}

	err = db.initializeCacheDB(config)
	if err != nil {
		return DB{}, log.Err("failed to initialize cache database", err)
	}

	return *db, nil
}

func TXDefer(tx *gorm.DB, log logger.Logger) {
	if tx.Error != nil {
		log.Er("failed to commit transaction", tx.Error)
		tx.Rollback()
	} else {
		err := tx.Commit().Error
		if err != nil {
			log.Er("failed to commit transaction", err)
		}
		// Removed success logging to reduce log noise during bulk operations
	}
}

func (s *DB) initializeDB(config config.Config) error {
	// Use Silent log level for bulk operations to prevent SQL query logging
	// This will completely disable GORM SQL logging to improve performance during data processing
	gormLogger := gormLogger.New(
		slog.NewLogLogger(slog.Default().Handler(), slog.LevelError), // Only show errors
		gormLogger.Config{
			SlowThreshold:             10 * time.Second,  // Only log extremely slow queries (10s+)
			LogLevel:                  gormLogger.Silent, // Silent mode - no SQL query logging
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      false,
			Colorful:                  true,
		},
	)

	gormConfig := &gorm.Config{
		Logger:                                   gormLogger,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: false,
		SkipDefaultTransaction:                   true,
	}

	return s.initializePostgresDB(gormConfig, config)
}

func (s *DB) initializePostgresDB(gormConfig *gorm.Config, config config.Config) error {
	log := s.log.Function("initializePostgresDB")

	if config.DatabaseHost == "" {
		return log.Error("database host is empty")
	}
	if config.DatabaseName == "" {
		return log.Error("database name is empty")
	}
	if config.DatabaseUser == "" {
		return log.Error("database user is empty")
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		config.DatabaseHost,
		config.DatabasePort,
		config.DatabaseUser,
		config.DatabasePassword,
		config.DatabaseName,
	)

	log.Info(
		"Connecting to PostgreSQL",
		"host",
		config.DatabaseHost,
		"port",
		config.DatabasePort,
		"database",
		config.DatabaseName,
	)
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return log.Err("failed to open PostgreSQL database with GORM", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return log.Err("failed to get database from GORM", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return log.Err("failed to ping PostgreSQL database through GORM", err)
	}

	log.Info("Successfully connected to PostgreSQL with GORM")
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	s.SQL = db

	return nil
}

func (s *DB) Close() (err error) {
	if s.SQL != nil {
		sqlDB, err := s.SQL.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				_ = s.log.Err("failed to close database", err)
			}
		}
	}

	if s.Cache.General != nil {
		s.Cache.General.Close()
	}

	if s.Cache.Session != nil {
		s.Cache.Session.Close()
	}

	if s.Cache.Events != nil {
		s.Cache.Events.Close()
	}

	if s.Cache.LoadTest != nil {
		s.Cache.LoadTest.Close()
	}

	return err
}

func (s *DB) SQLWithContext(ctx context.Context) *gorm.DB {
	return s.SQL.WithContext(ctx).Set("db_instance", *s)
}

func (s *DB) FlushAllCaches() error {
	log := s.log.Function("FlushAllCaches")
	log.Info("Flushing all cache databases")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cacheClients := []struct {
		client CacheClient
		name   string
	}{
		{s.Cache.General, "General"},
		{s.Cache.Session, "Session"},
		{s.Cache.User, "User"},
		{s.Cache.Events, "Events"},
		{s.Cache.LoadTest, "LoadTest"},
	}

	for _, cache := range cacheClients {
		if cache.client != nil {
			if err := cache.client.Do(ctx, cache.client.B().Flushdb().Build()).Error(); err != nil {
				log.Er("Failed to flush cache database", err, "cache", cache.name)
				return err
			}
			log.Info("Successfully flushed cache database", "cache", cache.name)
		}
	}

	log.Info("All cache databases flushed successfully")
	return nil
}
