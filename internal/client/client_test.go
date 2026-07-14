package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	errors "github.com/jbcjorge/errors-library"
	"github.com/jbcjorge/mcp-argo/internal/config"
)

func setupCfg(t *testing.T) {
	t.Helper()
	original := config.Cfg
	config.Cfg = &config.Config{}
	t.Cleanup(func() { config.Cfg = original })
}

func TestMain(m *testing.M) {
	config.Cfg = &config.Config{}
	InitHTTPClient(false)
	m.Run()
}

// =============================================================================
// Token resolution tests
// =============================================================================

func TestResolveBaseURLAndToken_DefaultURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = "https://argocd.example.com"
	config.Cfg.DefaultToken = "my-token"

	baseURL, token, err := ResolveBaseURLAndToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if baseURL != "https://argocd.example.com" {
		t.Errorf("baseURL = %q, want %q", baseURL, "https://argocd.example.com")
	}
	if token != "my-token" {
		t.Errorf("token = %q, want %q", token, "my-token")
	}
}

func TestResolveBaseURLAndToken_EmptyBaseURLNoDefault(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	config.Cfg.DefaultToken = "some-token"

	_, _, err := ResolveBaseURLAndToken("")
	if err == nil {
		t.Fatal("expected error for empty base URL with no default, got nil")
	}
	if !errors.Is(err, ErrNoBaseURL) {
		t.Errorf("error = %v, want ErrNoBaseURL", err)
	}
}

func TestResolveBaseURLAndToken_NonDefaultURLInRegistry(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = "https://default.example.com"
	config.Cfg.DefaultToken = "default-token"
	config.Cfg.TokenRegistry = []config.TokenRegistryEntry{
		{BaseURL: "https://other.example.com", Token: "other-token"},
	}

	baseURL, token, err := ResolveBaseURLAndToken("https://other.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if baseURL != "https://other.example.com" {
		t.Errorf("baseURL = %q, want %q", baseURL, "https://other.example.com")
	}
	if token != "other-token" {
		t.Errorf("token = %q, want %q", token, "other-token")
	}
}

func TestResolveBaseURLAndToken_NonDefaultURLNotInRegistry(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = "https://default.example.com"
	config.Cfg.DefaultToken = "default-token"
	config.Cfg.TokenRegistry = []config.TokenRegistryEntry{
		{BaseURL: "https://known.example.com", Token: "known-token"},
	}

	_, _, err := ResolveBaseURLAndToken("https://unknown.example.com")
	if err == nil {
		t.Fatal("expected error for non-default URL not in registry, got nil")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("error = %v, want ErrTokenNotFound", err)
	}
}

func TestResolveBaseURLAndToken_DefaultTokenNeverReturnedForNonDefaultURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = "https://default.example.com"
	config.Cfg.DefaultToken = "default-token"
	config.Cfg.TokenRegistry = []config.TokenRegistryEntry{}

	_, token, err := ResolveBaseURLAndToken("https://attacker.example.com")
	if err == nil {
		t.Fatal("expected error, got nil - default token would leak to non-default URL")
	}
	if token == "default-token" {
		t.Error("SECURITY: default token was returned for a non-default URL")
	}
}

func TestResolveBaseURLAndToken_URLNormalizationTrailingSlash(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = "https://argocd.example.com"
	config.Cfg.DefaultToken = "my-token"
	config.Cfg.TokenRegistry = []config.TokenRegistryEntry{
		{BaseURL: "https://other.example.com/", Token: "other-token"},
	}

	baseURL, token, err := ResolveBaseURLAndToken("https://other.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if baseURL != "https://other.example.com" {
		t.Errorf("baseURL = %q, want trailing slash stripped", baseURL)
	}
	if token != "other-token" {
		t.Errorf("token = %q, want %q", token, "other-token")
	}
}

// =============================================================================
// HTTP client tests
// =============================================================================

func TestDoRequest_BuildsCorrectURL(t *testing.T) {
	var capturedURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	setupCfg(t)

	query := url.Values{}
	query.Set("name", "*myapp*")
	query.Set("project", "default")

	_, err := DoRequest(context.Background(), "GET", ts.URL, "/api/v1/applications", "token", query, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedURL, "/api/v1/applications") {
		t.Errorf("URL path not correct: %q", capturedURL)
	}
	if !strings.Contains(capturedURL, "name=%2Amyapp%2A") {
		t.Errorf("URL query param 'name' not found or not encoded: %q", capturedURL)
	}
	if !strings.Contains(capturedURL, "project=default") {
		t.Errorf("URL query param 'project' not found: %q", capturedURL)
	}
}

func TestDoRequest_SetsAuthorizationBearerHeader(t *testing.T) {
	var capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	setupCfg(t)

	_, err := DoRequest(context.Background(), "GET", ts.URL, "/test", "secret-token-xyz", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Bearer secret-token-xyz"
	if capturedAuth != expected {
		t.Errorf("Authorization header = %q, want %q", capturedAuth, expected)
	}
}

func TestDoRequest_HandlesNon2xxAsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer ts.Close()

	setupCfg(t)

	_, err := DoRequest(context.Background(), "GET", ts.URL, "/forbidden", "token", nil, nil)
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}
	if !errors.Is(err, ErrAPIError) {
		t.Errorf("error = %v, want ErrAPIError", err)
	}
	if !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("error = %q, want it to include response body", err.Error())
	}
}

func TestDoRequest_TLSSkipVerifyWhenInsecure(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	setupCfg(t)

	// Without insecure, the self-signed cert should cause an error
	InitHTTPClient(false)
	_, err := DoRequest(context.Background(), "GET", ts.URL, "/test", "token", nil, nil)
	if err == nil {
		t.Fatal("expected TLS error without insecure flag")
	}

	// With insecure, should succeed
	InitHTTPClient(true)
	data, err := DoRequest(context.Background(), "GET", ts.URL, "/test", "token", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error with insecure=true: %v", err)
	}
	if !strings.Contains(string(data), "ok") {
		t.Errorf("response = %q, want it to contain 'ok'", string(data))
	}

	// Reset to non-insecure for other tests
	InitHTTPClient(false)
}

// =============================================================================
// Streaming request tests
// =============================================================================

func TestDoStreamingRequest_Success(t *testing.T) {
	ndjson := `{"result": {"content": "hello world", "podName": "pod-1"}}
{"result": {"content": "second line", "podName": "pod-1"}}

{"result": {"content": "third line", "podName": "pod-2"}}
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(ndjson))
	}))
	defer ts.Close()

	setupCfg(t)

	results, err := DoStreamingRequest(context.Background(), ts.URL, "/api/v1/applications/app/logs", "token", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if results[0]["content"] != "hello world" {
		t.Errorf("result[0].content = %v, want 'hello world'", results[0]["content"])
	}
	if results[2]["podName"] != "pod-2" {
		t.Errorf("result[2].podName = %v, want pod-2", results[2]["podName"])
	}
}

func TestDoStreamingRequest_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer ts.Close()

	setupCfg(t)

	_, err := DoStreamingRequest(context.Background(), ts.URL, "/api/v1/applications/app/logs", "token", nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !errors.Is(err, ErrAPIError) {
		t.Errorf("error = %v, want ErrAPIError", err)
	}
}

func TestDoStreamingRequest_InvalidJSON(t *testing.T) {
	ndjson := `not json at all
{"result": {"content": "valid line"}}
{broken json
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(ndjson))
	}))
	defer ts.Close()

	setupCfg(t)

	results, err := DoStreamingRequest(context.Background(), ts.URL, "/test", "token", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 valid result (invalid lines skipped), got %d", len(results))
	}
}

func TestDoStreamingRequest_NoResultField(t *testing.T) {
	ndjson := `{"other": "data"}
{"result": {"content": "valid"}}
{"info": "ignored"}
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(ndjson))
	}))
	defer ts.Close()

	setupCfg(t)

	results, err := DoStreamingRequest(context.Background(), ts.URL, "/test", "token", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result (entries without 'result' skipped), got %d", len(results))
	}
}

func TestDoStreamingRequest_WithQueryParams(t *testing.T) {
	var capturedQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": {"content": "ok"}}`))
	}))
	defer ts.Close()

	setupCfg(t)

	query := url.Values{}
	query.Set("container", "main")
	query.Set("follow", "false")

	_, err := DoStreamingRequest(context.Background(), ts.URL, "/test", "token", query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedQuery.Get("container") != "main" {
		t.Errorf("query container = %q, want main", capturedQuery.Get("container"))
	}
	if capturedQuery.Get("follow") != "false" {
		t.Errorf("query follow = %q, want false", capturedQuery.Get("follow"))
	}
}

func TestNewTokenResolver(t *testing.T) {
	cfg := &config.Config{
		DefaultBaseURL: "https://default.test",
		DefaultToken:   "default-token",
		TokenCommand:   "echo",
		TokenRegistry: []config.TokenRegistryEntry{
			{BaseURL: "https://other.test", Token: "other-token"},
		},
	}
	r := NewTokenResolver(cfg)
	if r.DefaultBaseURL != "https://default.test" {
		t.Errorf("DefaultBaseURL = %q", r.DefaultBaseURL)
	}
	if r.DefaultToken != "default-token" {
		t.Errorf("DefaultToken = %q", r.DefaultToken)
	}
	if r.TokenCommand != "echo" {
		t.Errorf("TokenCommand = %q", r.TokenCommand)
	}
	if len(r.Registry) != 1 {
		t.Errorf("Registry len = %d", len(r.Registry))
	}
}

func TestRegistryTokenResolver_Invalidate(t *testing.T) {
	r := &RegistryTokenResolver{
		DefaultBaseURL: "https://default.test",
		DefaultToken:   "token",
		cache:          map[string]string{"https://cached.test": "cached-token"},
	}

	// Before invalidate
	if _, ok := r.cache["https://cached.test"]; !ok {
		t.Fatal("expected cache entry to exist before invalidate")
	}

	r.Invalidate("https://cached.test")

	// After invalidate
	if _, ok := r.cache["https://cached.test"]; ok {
		t.Fatal("expected cache entry to be removed after invalidate")
	}
}

func TestRegistryTokenResolver_ExecTokenCommand(t *testing.T) {
	r := &RegistryTokenResolver{
		DefaultBaseURL: "https://default.test",
		DefaultToken:   "token",
		TokenCommand:   "echo",
		cache:          make(map[string]string),
	}

	// "echo" will print the baseURL argument + newline
	baseURL, token, err := r.Resolve("https://dynamic.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if baseURL != "https://dynamic.test" {
		t.Errorf("baseURL = %q", baseURL)
	}
	// echo prints the URL as the token (trimmed)
	if token != "https://dynamic.test" {
		t.Errorf("token = %q, want the URL echoed back", token)
	}

	// Second call should use cache
	r.TokenCommand = "false" // would fail if called
	_, token2, err := r.Resolve("https://dynamic.test")
	if err != nil {
		t.Fatalf("unexpected error on cached resolve: %v", err)
	}
	if token2 != "https://dynamic.test" {
		t.Errorf("cached token = %q", token2)
	}
}

func TestRegistryTokenResolver_ExecTokenCommand_Failure(t *testing.T) {
	r := &RegistryTokenResolver{
		DefaultBaseURL: "https://default.test",
		DefaultToken:   "token",
		TokenCommand:   "false", // always exits 1
		cache:          make(map[string]string),
	}

	_, _, err := r.Resolve("https://unknown.test")
	if err == nil {
		t.Fatal("expected error when token command fails")
	}
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got: %v", err)
	}
}
