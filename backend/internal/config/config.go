package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const defaultJWTSecret = "change-me-in-production"

type Config struct {
	Env      string
	Port     string
	Database DatabaseConfig
	Auth     AuthConfig
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type AuthConfig struct {
	JWTSecret      string
	JWTIssuer      string
	AccessTokenTTL time.Duration
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(getenv("ACCESS_TOKEN_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse ACCESS_TOKEN_TTL: %w", err)
	}

	env := getenv("APP_ENV", "development")
	jwtSecret := getenv("JWT_SECRET", defaultJWTSecret)
	if strings.EqualFold(env, "production") && jwtSecret == defaultJWTSecret {
		return Config{}, errors.New("JWT_SECRET must be set in production")
	}

	dbDriver := strings.ToLower(getenv("DB_DRIVER", "sqlite"))
	dbDSN := getenv("DB_DSN", "data/healthvision.db")
	if dbDriver == "mysql" && dbDSN == "" {
		return Config{}, errors.New("DB_DSN is required when DB_DRIVER=mysql")
	}

	return Config{
		Env:  env,
		Port: getenv("PORT", "8080"),
		Database: DatabaseConfig{
			Driver: dbDriver,
			DSN:    dbDSN,
		},
		Auth: AuthConfig{
			JWTSecret:      jwtSecret,
			JWTIssuer:      getenv("JWT_ISSUER", "healthvision"),
			AccessTokenTTL: ttl,
		},
	}, nil
}

func getenv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
