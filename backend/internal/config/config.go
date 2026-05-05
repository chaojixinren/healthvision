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
	LLM      LLMConfig
	Agent    AgentConfig
}

type LLMConfig struct {
	ModelName string
	BaseURL   string
	APIKey    string
}

// AgentConfig holds runtime knobs for the LLM agent. The defaults are chosen
// to be safe in production; only flip them via env vars when intentionally
// testing in development.
type AgentConfig struct {
	// RequireWriteToolConfirmation controls whether the write-side tools
	// (create/update/delete medicine and reminder) require Human-in-the-Loop
	// confirmation. Defaults to true. Set
	// AGENT_REQUIRE_WRITE_TOOL_CONFIRMATION=false in development to allow
	// the agent to mutate data without an explicit user approval — useful
	// for end-to-end smoke tests, dangerous in production.
	RequireWriteToolConfirmation bool
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

	dbDriver := strings.ToLower(getenv("DB_DRIVER", "mysql"))
	dbDSN := getenv("DB_DSN", "")
	if dbDSN == "" {
		return Config{}, errors.New("DB_DSN is required")
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
		LLM: LLMConfig{
			ModelName: getenv("LLM_MODEL", "gpt-4o-mini"),
			BaseURL:   getenv("LLM_BASE_URL", "https://api.openai.com/v1"),
			APIKey:    getenv("LLM_API_KEY", ""),
		},
		Agent: AgentConfig{
			RequireWriteToolConfirmation: getenvBool("AGENT_REQUIRE_WRITE_TOOL_CONFIRMATION", true),
		},
	}, nil
}

// getenvBool reads a boolean env var. Recognises 1/0, true/false, yes/no,
// on/off (case-insensitive). Returns fallback for empty or unrecognised
// values.
func getenvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getenv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
