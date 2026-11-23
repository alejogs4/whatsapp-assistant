package main

import (
	"errors"
	"os"
	"strings"
	"time"
)

var ErrMissingGeminiAPIKey = errors.New("daemon: GEMINI_API_KEY is required")

// Config holds runtime configuration for the WhatsApp AI Learning daemon.
type Config struct {
	SQLiteDSN         string
	WhatsAppSessionDB string
	GeminiAPIKey      string
	TickInterval      time.Duration
}

// LoadConfigFromEnv loads and validates daemon configuration from environment variables.
func LoadConfigFromEnv(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}

	cfg := Config{
		SQLiteDSN:         strings.TrimSpace(getenv("SQLITE_DSN")),
		WhatsAppSessionDB: strings.TrimSpace(getenv("WHATSAPP_SESSION_DB")),
		GeminiAPIKey:      strings.TrimSpace(getenv("GEMINI_API_KEY")),
		TickInterval:      time.Minute,
	}

	if cfg.SQLiteDSN == "" {
		cfg.SQLiteDSN = "file:assistant.db"
	}
	if cfg.WhatsAppSessionDB == "" {
		cfg.WhatsAppSessionDB = "file:whatsapp_session.db"
	}
	if cfg.GeminiAPIKey == "" {
		return Config{}, ErrMissingGeminiAPIKey
	}

	return cfg, nil
}
