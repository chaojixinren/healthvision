package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
	Chat     ChatConfig
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
	JWTSecret           string
	JWTIssuer           string
	AccessTokenTTL      time.Duration // absolute maximum lifetime of an access token
	RefreshTokenTTL     time.Duration
	MaxSessionsPerUser  int
	AccessSlidingWindow time.Duration // token expires if unused for this long (e.g. 24h)
}

type ChatConfig struct {
	RetentionDays      int
	MaxMessagesPerUser int
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(getenv("ACCESS_TOKEN_TTL", "2160h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse ACCESS_TOKEN_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(getenv("REFRESH_TOKEN_TTL", "720h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REFRESH_TOKEN_TTL: %w", err)
	}
	slidingWindow, err := time.ParseDuration(getenv("ACCESS_SLIDING_WINDOW", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse ACCESS_SLIDING_WINDOW: %w", err)
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
	chatRetentionDays, err := getenvInt("CHAT_RETENTION_DAYS", 30)
	if err != nil {
		return Config{}, err
	}
	chatMaxMessagesPerUser, err := getenvInt("CHAT_MAX_MESSAGES_PER_USER", 2000)
	if err != nil {
		return Config{}, err
	}

	maxSessions, err := getenvInt("MAX_SESSIONS_PER_USER", 5)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Env:  env,
		Port: getenv("PORT", "8080"),
		Database: DatabaseConfig{
			Driver: dbDriver,
			DSN:    dbDSN,
		},
		Auth: AuthConfig{
			JWTSecret:           jwtSecret,
			JWTIssuer:           getenv("JWT_ISSUER", "healthvision"),
			AccessTokenTTL:      ttl,
			RefreshTokenTTL:     refreshTTL,
			MaxSessionsPerUser:  maxSessions,
			AccessSlidingWindow: slidingWindow,
		},
		LLM: LLMConfig{
			ModelName: getenv("LLM_MODEL", "gpt-4o-mini"),
			BaseURL:   getenv("LLM_BASE_URL", "https://api.openai.com/v1"),
			APIKey:    getenv("LLM_API_KEY", ""),
		},
		Agent: AgentConfig{
			RequireWriteToolConfirmation: getenvBool("AGENT_REQUIRE_WRITE_TOOL_CONFIRMATION", true),
		},
		Chat: ChatConfig{
			RetentionDays:      chatRetentionDays,
			MaxMessagesPerUser: chatMaxMessagesPerUser,
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

func getenvInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s must be >= 0", key)
	}
	return value, nil
}
