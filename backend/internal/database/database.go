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
	if err := db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.Medicine{}, &models.Reminder{}, &models.Conversation{}, &models.ChatMessage{}, &models.Binding{}, &models.Confirmation{}); err != nil {
		return err
	}
	// Backfill default values for new reminder recurrence columns added 2026-05.
	return db.Exec(`UPDATE reminders SET repeat_type = 'daily' WHERE repeat_type IS NULL OR repeat_type = ''`).Error
}
