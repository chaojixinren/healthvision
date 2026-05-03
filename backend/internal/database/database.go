package database

import (
	"fmt"

	"healthvision/backend/internal/config"
	"healthvision/backend/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	if cfg.Driver != "mysql" {
		return nil, fmt.Errorf("unsupported DB_DRIVER %q, only mysql is supported", cfg.Driver)
	}
	return gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.Medicine{}, &models.Reminder{}, &models.Conversation{}, &models.ChatMessage{}, &models.Binding{})
}
