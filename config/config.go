package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	OpenRouter OpenRouterConfig `yaml:"openrouter"`
	Database   DatabaseConfig   `yaml:"database"`
	Services   []ServiceConfig  `yaml:"services"`
}

type ServerConfig struct {
	Port            int             `yaml:"port"`
	Env             string          `yaml:"env"`
	ShutdownTimeout int             `yaml:"shutdown_timeout"`
	AdminAPIKey     string          `yaml:"admin_api_key"`
	RateLimit       RateLimitConfig `yaml:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
}

type OpenRouterConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type DatabaseConfig struct {
	ServicesDir    string `yaml:"services_dir"`
	MaxConnections int    `yaml:"max_connections"`
}

type ServiceConfig struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Env == "" {
		cfg.Server.Env = "development"
	}
	if cfg.OpenRouter.BaseURL == "" {
		cfg.OpenRouter.BaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.Database.ServicesDir == "" {
		cfg.Database.ServicesDir = "./data/services"
	}
	if cfg.Database.MaxConnections == 0 {
		cfg.Database.MaxConnections = 10
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 15
	}
	if cfg.Server.RateLimit.RequestsPerSecond == 0 {
		cfg.Server.RateLimit.RequestsPerSecond = 10
	}
}

func validate(cfg *Config) error {
	if cfg.OpenRouter.APIKey == "" {
		return fmt.Errorf("openrouter.api_key is required (set OPENROUTER_API_KEY environment variable)")
	}
	return nil
}
