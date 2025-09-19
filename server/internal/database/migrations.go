package database

import (
	"waugzee/internal/logger"
	"waugzee/internal/models"
)

// MigrateModels runs GORM AutoMigrate for all models
func (db *DB) MigrateModels() error {
	log := logger.New("database").Function("MigrateModels")
	log.Info("Starting database migration")

	// Define all models that need to be migrated
	modelsToMigrate := []interface{}{
		// Existing models
		&models.User{},
		&models.DiscogsSync{},
		&models.UserPreferences{},
		&models.Artist{},
		&models.Label{},
		&models.Master{},
		&models.Release{},
		&models.Genre{},
		&models.Image{},
		&models.UserCollection{},
		&models.Turntable{},
		&models.Cartridge{},
		&models.Stylus{},
		&models.PlaySession{},
		&models.MaintenanceRecord{},
		&models.DiscogsDataProcessing{},

		// New Discogs API Proxy models
		&models.DiscogsApiRequest{},
		&models.DiscogsCollectionSync{},
	}

	// Run migration for each model
	for _, model := range modelsToMigrate {
		if err := db.SQL.AutoMigrate(model); err != nil {
			log.Error("Failed to migrate model", "model", model, "error", err)
			return err
		}
	}

	log.Info("Database migration completed successfully")
	return nil
}

// CreateIndexes creates additional indexes that GORM doesn't create automatically
func (db *DB) CreateIndexes() error {
	log := logger.New("database").Function("CreateIndexes")
	log.Info("Creating additional database indexes")

	// Discogs API Request indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_discogs_api_requests_user_status ON discogs_api_requests(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_discogs_api_requests_sync_status ON discogs_api_requests(sync_session_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_discogs_collection_syncs_user_status ON discogs_collection_syncs(user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_discogs_collection_syncs_created_at ON discogs_collection_syncs(created_at DESC)",
	}

	for _, indexSQL := range indexes {
		if err := db.SQL.Exec(indexSQL).Error; err != nil {
			log.Warn("Failed to create index", "sql", indexSQL, "error", err)
			// Continue with other indexes even if one fails
		}
	}

	log.Info("Additional database indexes created")
	return nil
}