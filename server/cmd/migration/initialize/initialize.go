package initialize

import (
	"waugzee/config"
	logger "github.com/Bparsons0904/goLogger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
)

func InitializeTables(db *gorm.DB, config config.Config, log logger.Logger) error {
	log = log.Function("InitializeTables")
	log.Info("Initializing essential production data")

	if err := initializeStyluses(db, log); err != nil {
		return log.Err("failed to initialize styluses", err)
	}

	log.Info("Table initialization complete")
	return nil
}

func initializeStyluses(db *gorm.DB, log logger.Logger) error {
	log.Info("Initializing stylus reference data")

	styluses := getStylusesData()

	for _, stylus := range styluses {
		var existingStylus Stylus
		if err := db.First(&existingStylus, "brand = ? AND model = ?", stylus.Brand, stylus.Model).Error; err == nil {
			log.Debug("Stylus already exists", "brand", stylus.Brand, "model", stylus.Model)
			continue
		}
		log.Info("Initializing stylus", "brand", stylus.Brand, "model", stylus.Model)
		if err := db.Create(&stylus).Error; err != nil {
			return log.Err(
				"failed to create stylus",
				err,
				"brand",
				stylus.Brand,
				"model",
				stylus.Model,
			)
		}
	}

	log.Info("Stylus reference data initialized", "count", len(styluses))
	return nil
}

