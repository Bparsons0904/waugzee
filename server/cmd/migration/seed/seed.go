package seed

import (
	"waugzee/config"
	"waugzee/internal/logger"
	. "waugzee/internal/models"

	"gorm.io/gorm"
)

func intPtr(i int) *int {
	return &i
}

func cartridgeTypePtr(ct CartridgeType) *CartridgeType {
	return &ct
}

func Seed(db *gorm.DB, config config.Config, log logger.Logger) error {
	log = log.Function("seed")
	log.Info("Seeding development data")

	if err := seedStyluses(db, log); err != nil {
		return log.Err("failed to seed styluses", err)
	}

	return nil
}

func seedStyluses(db *gorm.DB, log logger.Logger) error {
	log.Info("Seeding styluses")

	styluses := getStylusesData()

	for _, stylus := range styluses {
		var existingStylus Stylus
		if err := db.First(&existingStylus, "brand = ? AND model = ?", stylus.Brand, stylus.Model).Error; err == nil {
			log.Info("Stylus already exists", "brand", stylus.Brand, "model", stylus.Model)
			continue
		}
		log.Info("Seeding stylus", "brand", stylus.Brand, "model", stylus.Model)
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

	log.Info("Styluses seeded successfully", "count", len(styluses))
	return nil
}
