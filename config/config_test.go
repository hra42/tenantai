package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	path := writeTestConfig(t, `
server:
  port: 9090
  env: production
openrouter:
  api_key: "test-key"
  base_url: "https://example.com/api"
database:
  services_dir: "/tmp/services"
  max_connections: 5
services:
  - id: svc1
    name: "Service One"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.Env != "production" {
		t.Errorf("env = %q, want production", cfg.Server.Env)
	}
	if cfg.OpenRouter.APIKey != "test-key" {
		t.Errorf("api_key = %q, want test-key", cfg.OpenRouter.APIKey)
	}
	if cfg.Database.ServicesDir != "/tmp/services" {
		t.Errorf("services_dir = %q, want /tmp/services", cfg.Database.ServicesDir)
	}
	if cfg.Database.MaxConnections != 5 {
		t.Errorf("max_connections = %d, want 5", cfg.Database.MaxConnections)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].ID != "svc1" {
		t.Errorf("services = %+v, want [{svc1 Service One}]", cfg.Services)
	}
}

func TestLoad_EnvSubstitution(t *testing.T) {
	t.Setenv("TEST_API_KEY", "from-env")

	path := writeTestConfig(t, `
openrouter:
  api_key: "${TEST_API_KEY}"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OpenRouter.APIKey != "from-env" {
		t.Errorf("api_key = %q, want from-env", cfg.OpenRouter.APIKey)
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	path := writeTestConfig(t, `
server:
  port: 8080
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	path := writeTestConfig(t, `
openrouter:
  api_key: "test-key"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Env != "development" {
		t.Errorf("default env = %q, want development", cfg.Server.Env)
	}
	if cfg.OpenRouter.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("default base_url = %q", cfg.OpenRouter.BaseURL)
	}
	if cfg.Database.ServicesDir != "./data/services" {
		t.Errorf("default services_dir = %q", cfg.Database.ServicesDir)
	}
	if cfg.Database.MaxConnections != 10 {
		t.Errorf("default max_connections = %d, want 10", cfg.Database.MaxConnections)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
