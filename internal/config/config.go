package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port                   int    `yaml:"port"`
	LogLevel               string `yaml:"log_level"`
	ShutdownTimeoutSeconds int    `yaml:"shutdown_timeout_seconds"`
}

type RedisConfig struct {
	Addr      string `yaml:"addr"`
	Password  string `yaml:"password"`
	DB        int    `yaml:"db"`
	KeyPrefix string `yaml:"key_prefix"`
}

type WorkerConfig struct {
	Concurrency int `yaml:"concurrency"`
	MaxRetries  int `yaml:"max_retries"`
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type WebhookTarget struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Secret string `yaml:"secret"`
}

// Config holds all application configuration
type Config struct {
	Server             ServerConfig   `yaml:"server"`
	APIKeys            []string       `yaml:"api_keys"`
	RateLimitPerMinute int            `yaml:"rate_limit_per_minute"`
	Redis              RedisConfig    `yaml:"redis"`
	Worker             WorkerConfig   `yaml:"worker"`
	Telegram           TelegramConfig `yaml:"telegram"`
	Webhooks           []WebhookTarget `yaml:"webhooks"`
	apiKeysMap         map[string]bool
}

// Load reads configuration from a YAML file.
// Path is taken from PNS_CONFIG env var, defaulting to config.yaml.
func Load() (*Config, error) {
	path := os.Getenv("PNS_CONFIG")
	if path == "" {
		path = "config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:                   8272,
			LogLevel:               "info",
			ShutdownTimeoutSeconds: 30,
		},
		RateLimitPerMinute: 60,
		Redis: RedisConfig{
			Addr:      "localhost:6379",
			DB:        0,
			KeyPrefix: "pns",
		},
		Worker: WorkerConfig{
			Concurrency: 10,
			MaxRetries:  5,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	if len(cfg.APIKeys) == 0 {
		return nil, fmt.Errorf("at least one api_key must be configured")
	}

	if cfg.Telegram.BotToken == "" {
		return nil, fmt.Errorf("telegram.bot_token is required")
	}

	if cfg.Telegram.ChatID == "" {
		return nil, fmt.Errorf("telegram.chat_id is required")
	}

	cfg.apiKeysMap = make(map[string]bool, len(cfg.APIKeys))
	for _, key := range cfg.APIKeys {
		if key != "" {
			cfg.apiKeysMap[key] = true
		}
	}

	return cfg, nil
}

// ValidateAPIKey checks if the provided API key is valid
func (c *Config) ValidateAPIKey(key string) bool {
	return c.apiKeysMap[key]
}
