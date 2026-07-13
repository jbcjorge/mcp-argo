package config

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

func TestLoadConfig_ReadsEnvVars(t *testing.T) {
	t.Setenv("ARGOCD_BASE_URL", "https://argocd.example.com/")
	t.Setenv("ARGOCD_API_TOKEN", "test-token-123")
	t.Setenv("MCP_READ_ONLY", "true")
	t.Setenv("ARGOCD_INSECURE", "true")
	t.Setenv("ARGOCD_TOKEN_REGISTRY_PATH", "")

	c := LoadConfig()

	if c.DefaultBaseURL != "https://argocd.example.com" {
		t.Errorf("DefaultBaseURL = %q, want %q (trailing slash stripped)", c.DefaultBaseURL, "https://argocd.example.com")
	}
	if c.DefaultToken != "test-token-123" {
		t.Errorf("DefaultToken = %q, want %q", c.DefaultToken, "test-token-123")
	}
	if !c.ReadOnly {
		t.Error("ReadOnly should be true")
	}
	if !c.Insecure {
		t.Error("Insecure should be true")
	}
}

func TestLoadConfig_TokenRegistryFromFile(t *testing.T) {
	registry := []TokenRegistryEntry{
		{BaseURL: "https://argo1.example.com", Token: "token-1"},
		{BaseURL: "https://argo2.example.com", Token: "token-2"},
	}
	data, _ := json.Marshal(registry)

	tmpFile, err := os.CreateTemp(t.TempDir(), "registry-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Write(data)
	tmpFile.Close()

	t.Setenv("ARGOCD_BASE_URL", "https://default.example.com")
	t.Setenv("ARGOCD_API_TOKEN", "default-token")
	t.Setenv("MCP_READ_ONLY", "false")
	t.Setenv("ARGOCD_INSECURE", "false")
	t.Setenv("ARGOCD_TOKEN_REGISTRY_PATH", tmpFile.Name())

	c := LoadConfig()

	if len(c.TokenRegistry) != 2 {
		t.Fatalf("TokenRegistry length = %d, want 2", len(c.TokenRegistry))
	}
	if c.TokenRegistry[0].Token != "token-1" {
		t.Errorf("TokenRegistry[0].Token = %q, want %q", c.TokenRegistry[0].Token, "token-1")
	}
	if c.TokenRegistry[1].BaseURL != "https://argo2.example.com" {
		t.Errorf("TokenRegistry[1].BaseURL = %q, want %q", c.TokenRegistry[1].BaseURL, "https://argo2.example.com")
	}
}

func TestLoadConfig_EmptyEnvVars(t *testing.T) {
	t.Setenv("ARGOCD_BASE_URL", "")
	t.Setenv("ARGOCD_API_TOKEN", "")
	t.Setenv("MCP_READ_ONLY", "")
	t.Setenv("ARGOCD_INSECURE", "")
	t.Setenv("ARGOCD_TOKEN_REGISTRY_PATH", "")

	c := LoadConfig()

	if c.DefaultBaseURL != "" {
		t.Errorf("DefaultBaseURL = %q, want empty", c.DefaultBaseURL)
	}
	if c.DefaultToken != "" {
		t.Errorf("DefaultToken = %q, want empty", c.DefaultToken)
	}
	if c.ReadOnly {
		t.Error("ReadOnly should be false for empty env")
	}
	if c.Insecure {
		t.Error("Insecure should be false for empty env")
	}
	if c.TokenRegistry != nil {
		t.Errorf("TokenRegistry = %v, want nil", c.TokenRegistry)
	}
}

func TestInitLogging_DebugLevel(t *testing.T) {
	InitLogging("info", "LOG_LEVEL", "debug")
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected debug level to be enabled")
	}
}

func TestInitLogging_WarnLevel(t *testing.T) {
	InitLogging("info", "LOG_LEVEL", "warn")
	if slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected info level to be disabled at warn")
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected warn level to be enabled")
	}
}

func TestInitLogging_ErrorLevel(t *testing.T) {
	InitLogging("info", "LOG_LEVEL", "error")
	if slog.Default().Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected warn level to be disabled at error")
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelError) {
		t.Error("expected error level to be enabled")
	}
}

func TestInitLogging_DefaultIsInfo(t *testing.T) {
	InitLogging("info", "LOG_LEVEL", "")
	if !slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected info level to be enabled by default")
	}
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected debug level to be disabled by default")
	}
}

func TestInitLogging_EnvOverride(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	InitLogging("info", "LOG_LEVEL", "")
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected LOG_LEVEL=debug to enable debug")
	}
}

func TestInitLogging_FlagOverridesEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	InitLogging("info", "LOG_LEVEL", "error")
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected flag=error to override env=debug")
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelError) {
		t.Error("expected error level to be enabled")
	}
}
