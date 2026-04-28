package database

import (
	"fmt"
	"os"
	"path/filepath"

	"healthvision/backend/internal/config"
	"healthvision/backend/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	switch cfg.Driver {
	case "sqlite":
		if err := ensureSQLiteDir(cfg.DSN); err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{})
	case "mysql":
		return gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", cfg.Driver)
	}
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{})
}

func ensureSQLiteDir(dsn string) error {
	if dsn == "" || dsn == ":memory:" || filepath.IsAbs(dsn) {
		return nil
	}

	return os.MkdirAll(filepath.Dir(dsn), 0o755)
}
