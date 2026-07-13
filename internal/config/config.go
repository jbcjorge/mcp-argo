package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
)

// TokenRegistryEntry represents an entry in the token registry JSON file.
type TokenRegistryEntry struct {
	BaseURL string `json:"baseUrl"`
	Token   string `json:"token"`
}

// Config holds the server configuration from environment variables.
type Config struct {
	DefaultBaseURL string
	DefaultToken   string
	TokenRegistry  []TokenRegistryEntry
	TokenCommand   string // external command to obtain tokens dynamically
	ReadOnly       bool
	Insecure       bool
}

// Cfg is the package-level configuration set by LoadConfig.
var Cfg *Config

// LoadConfig reads environment variables and populates the global Cfg.
func LoadConfig() *Config {
	c := &Config{}
	c.DefaultBaseURL = strings.TrimRight(os.Getenv("ARGOCD_BASE_URL"), "/")
	c.DefaultToken = os.Getenv("ARGOCD_API_TOKEN")
	c.ReadOnly = strings.EqualFold(os.Getenv("MCP_READ_ONLY"), "true")
	c.Insecure = strings.EqualFold(os.Getenv("ARGOCD_INSECURE"), "true")
	c.TokenCommand = os.Getenv("ARGOCD_TOKEN_COMMAND")

	registryPath := os.Getenv("ARGOCD_TOKEN_REGISTRY_PATH")
	if registryPath != "" {
		data, err := os.ReadFile(registryPath) // #nosec G304 G703 -- path from ARGOCD_TOKEN_REGISTRY_PATH env var, not user input
		if err == nil {
			_ = json.Unmarshal(data, &c.TokenRegistry)
		}
	}

	Cfg = c
	return c
}

// InitLogging sets up slog with the resolved log level.
// Precedence: defaultLevel (build-time) -> envVar env -> flagLevel flag.
func InitLogging(defaultLevel, envVar, flagLevel string) {
	resolved := defaultLevel
	if env := os.Getenv(envVar); env != "" {
		resolved = env
	}
	if flagLevel != "" {
		resolved = flagLevel
	}

	level := new(slog.LevelVar)
	switch strings.ToLower(resolved) {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn", "warning":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}
