package config

import (
	"strings"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_DSN", "user:pass@tcp(localhost:3306)/healthvision?charset=utf8mb4&parseTime=True&loc=Local")
	t.Setenv("ACCESS_TOKEN_TTL", "24h")
	t.Setenv("REFRESH_TOKEN_TTL", "720h")
}

func TestLoadChatConfigDefaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("CHAT_RETENTION_DAYS", "")
	t.Setenv("CHAT_MAX_MESSAGES_PER_USER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Chat.RetentionDays != 30 {
		t.Fatalf("RetentionDays = %d, want 30", cfg.Chat.RetentionDays)
	}
	if cfg.Chat.MaxMessagesPerUser != 2000 {
		t.Fatalf("MaxMessagesPerUser = %d, want 2000", cfg.Chat.MaxMessagesPerUser)
	}
}

func TestLoadChatConfigFromEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("CHAT_RETENTION_DAYS", "45")
	t.Setenv("CHAT_MAX_MESSAGES_PER_USER", "500")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Chat.RetentionDays != 45 {
		t.Fatalf("RetentionDays = %d, want 45", cfg.Chat.RetentionDays)
	}
	if cfg.Chat.MaxMessagesPerUser != 500 {
		t.Fatalf("MaxMessagesPerUser = %d, want 500", cfg.Chat.MaxMessagesPerUser)
	}
}

func TestLoadRejectsNegativeChatConfig(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("CHAT_RETENTION_DAYS", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "CHAT_RETENTION_DAYS") {
		t.Fatalf("Load() error = %q, want CHAT_RETENTION_DAYS", err.Error())
	}
}
