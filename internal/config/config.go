package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Port     int    `envconfig:"PORT" default:"8272"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Authentication - comma-separated list of API keys
	APIKeysRaw string `envconfig:"API_KEYS" required:"true"`
	APIKeys    map[string]bool

	// Rate Limiting
	RateLimitPerMinute int `envconfig:"RATE_LIMIT_PER_MINUTE" default:"60"`

	// Redis
	RedisAddr      string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword  string `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB        int    `envconfig:"REDIS_DB" default:"0"`
	RedisKeyPrefix string `envconfig:"REDIS_KEY_PREFIX" default:"pns"`

	// Worker
	WorkerConcurrency int `envconfig:"WORKER_CONCURRENCY" default:"10"`
	MaxRetries        int `envconfig:"MAX_RETRIES" default:"5"`

	// Telegram
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`
	TelegramChatID   string `envconfig:"TELEGRAM_CHAT_ID" required:"true"`

	// Graceful Shutdown
	ShutdownTimeoutSeconds int `envconfig:"SHUTDOWN_TIMEOUT_SECONDS" default:"30"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	// Parse comma-separated API keys into a map for O(1) lookup
	cfg.APIKeys = make(map[string]bool)
	for _, key := range strings.Split(cfg.APIKeysRaw, ",") {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			cfg.APIKeys[trimmed] = true
		}
	}

	if len(cfg.APIKeys) == 0 {
		return nil, fmt.Errorf("at least one API key must be configured")
	}

	return &cfg, nil
}

// ValidateAPIKey checks if the provided API key is valid
func (c *Config) ValidateAPIKey(key string) bool {
	return c.APIKeys[key]
}
