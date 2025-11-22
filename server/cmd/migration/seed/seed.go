package seed

import (
	"waugzee/config"
	logger "github.com/Bparsons0904/goLogger"

	"gorm.io/gorm"
)

func Seed(db *gorm.DB, config config.Config, log logger.Logger) error {
	log = log.Function("seed")
	log.Info("Seeding development-only test data")

	log.Info("No development seed data configured")
	return nil
}
