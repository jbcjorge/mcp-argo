package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	errors "github.com/jbcjorge/errors-library"
	"github.com/jbcjorge/mcp-argo/internal/config"
)

// MaxResponseSize is the maximum response body size (50MB).
const MaxResponseSize = 50 * 1024 * 1024

const (
	defaultMaxIdleConns        = 20
	defaultMaxIdleConnsPerHost = 10
	defaultHTTPTimeout         = 60 * time.Second
)

// Sentinel errors for common failure modes.
var (
	ErrNoBaseURL     = errors.New("no ArgoCD base URL configured; set ARGOCD_BASE_URL or pass argocdBaseUrl")
	ErrNoToken       = errors.New("no API token configured; set ARGOCD_API_TOKEN")
	ErrTokenNotFound = errors.New("no token found for ArgoCD instance %s; add it to the token registry")
	ErrRequestFailed = errors.New("request failed: %s")
	ErrParseResponse = errors.New("failed to parse response: %s")
	ErrAPIError      = errors.New("ArgoCD API error (HTTP %d): %s")
)

// HTTPClient implements APIClient using net/http.
type HTTPClient struct {
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTPClient with optional TLS skip.
func NewHTTPClient(insecure bool) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        defaultMaxIdleConns,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
	}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 -- operator opt-in via ARGOCD_INSECURE env var
	}
	return &HTTPClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   defaultHTTPTimeout,
		},
	}
}

// DoRequest performs an HTTP request to the ArgoCD API and returns the response body.
func (c *HTTPClient) DoRequest(ctx context.Context, method, baseURL, path, token string, query url.Values, body io.Reader) ([]byte, error) {
	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("argocd api request", "method", method, "url", u)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrRequestFailed.Parse(errors.WithParsedMessage(err.Error()), errors.WithError(err))
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, ErrAPIError.Parse(errors.WithParsedMessage(resp.StatusCode, string(data)))
	}

	return data, nil
}

// DoStreamingRequest performs a streaming GET request and collects NDJSON results.
func (c *HTTPClient) DoStreamingRequest(ctx context.Context, baseURL, path, token string, query url.Values) ([]map[string]interface{}, error) {
	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrRequestFailed.Parse(errors.WithParsedMessage(err.Error()), errors.WithError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
		return nil, ErrAPIError.Parse(errors.WithParsedMessage(resp.StatusCode, string(data)))
	}

	var results []map[string]interface{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if result, ok := entry["result"]; ok {
			if resultMap, ok := result.(map[string]interface{}); ok {
				results = append(results, resultMap)
			}
		}
	}

	return results, nil
}

// RegistryTokenResolver implements TokenResolver using config-based token registry
// with optional fallback to an external token command for dynamic resolution.
type RegistryTokenResolver struct {
	DefaultBaseURL string
	DefaultToken   string
	Registry       []config.TokenRegistryEntry
	TokenCommand   string            // external command to get a token for a URL (receives URL as arg, prints token to stdout)
	cache          map[string]string // cache tokens obtained via command
}

// NewTokenResolver creates a RegistryTokenResolver from the given config.
func NewTokenResolver(cfg *config.Config) *RegistryTokenResolver {
	return &RegistryTokenResolver{
		DefaultBaseURL: cfg.DefaultBaseURL,
		DefaultToken:   cfg.DefaultToken,
		Registry:       cfg.TokenRegistry,
		TokenCommand:   cfg.TokenCommand,
		cache:          make(map[string]string),
	}
}

// Resolve returns the base URL and token for the given argocdBaseUrl argument.
// Security: the default token is NEVER sent to a non-default base URL.
// Resolution order: default token → registry → token command → error.
func (r *RegistryTokenResolver) Resolve(argocdBaseUrl string) (string, string, error) {
	baseURL := r.DefaultBaseURL
	if argocdBaseUrl != "" {
		baseURL = strings.TrimRight(argocdBaseUrl, "/")
	}

	if baseURL == "" {
		return "", "", ErrNoBaseURL
	}

	// If using the default base URL, return the default token
	if baseURL == r.DefaultBaseURL {
		if r.DefaultToken == "" {
			return "", "", ErrNoToken
		}
		return baseURL, r.DefaultToken, nil
	}

	// Non-default URL: look up in registry (never send default token)
	for _, entry := range r.Registry {
		entryURL := strings.TrimRight(entry.BaseURL, "/")
		if entryURL == baseURL {
			return baseURL, entry.Token, nil
		}
	}

	// Check cache from previous command invocations
	if token, ok := r.cache[baseURL]; ok {
		return baseURL, token, nil
	}

	// Fallback: call external token command if configured
	if r.TokenCommand != "" {
		token, err := r.execTokenCommand(baseURL)
		if err != nil {
			slog.Warn("token command failed", "url", baseURL, "err", err)
			return "", "", ErrTokenNotFound.Parse(errors.WithParsedMessage(baseURL))
		}
		if token != "" {
			r.cache[baseURL] = token
			slog.Debug("token obtained via command", "url", baseURL)
			return baseURL, token, nil
		}
	}

	return "", "", ErrTokenNotFound.Parse(errors.WithParsedMessage(baseURL))
}

// Invalidate removes a cached token for the given base URL.
func (r *RegistryTokenResolver) Invalidate(baseURL string) {
	normalized := strings.TrimRight(baseURL, "/")
	delete(r.cache, normalized)
	slog.Debug("token cache invalidated", "url", normalized)
}

// execTokenCommand runs the configured token command with the base URL as argument.
func (r *RegistryTokenResolver) execTokenCommand(baseURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.TokenCommand, baseURL) // #nosec G204 -- command from ARGOCD_TOKEN_COMMAND env var, not user input
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("token command %q failed: %w", r.TokenCommand, err)
	}

	token := strings.TrimSpace(string(out))
	return token, nil
}

// ---------------------------------------------------------------------------
// Legacy package-level functions (delegate to shared instance for backward compat)
// ---------------------------------------------------------------------------

var defaultClient *HTTPClient

// InitHTTPClient creates the shared HTTP client with optional TLS skip.
func InitHTTPClient(insecure bool) {
	defaultClient = NewHTTPClient(insecure)
}

// ResolveBaseURLAndToken resolves the base URL and token for a request using config.Cfg.
// Security: the default token is NEVER sent to a non-default base URL.
func ResolveBaseURLAndToken(argocdBaseUrl string) (string, string, error) {
	resolver := &RegistryTokenResolver{
		DefaultBaseURL: config.Cfg.DefaultBaseURL,
		DefaultToken:   config.Cfg.DefaultToken,
		Registry:       config.Cfg.TokenRegistry,
	}
	return resolver.Resolve(argocdBaseUrl)
}

// DoRequest performs an HTTP request to the ArgoCD API and returns the response body.
func DoRequest(ctx context.Context, method, baseURL, path, token string, query url.Values, body io.Reader) ([]byte, error) {
	return defaultClient.DoRequest(ctx, method, baseURL, path, token, query, body)
}

// DoStreamingRequest performs a streaming GET request and collects NDJSON results.
func DoStreamingRequest(ctx context.Context, baseURL, path, token string, query url.Values) ([]map[string]interface{}, error) {
	return defaultClient.DoStreamingRequest(ctx, baseURL, path, token, query)
}
