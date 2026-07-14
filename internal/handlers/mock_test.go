package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"testing"

	errors "github.com/jbcjorge/errors-library"
	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// mockAPIClient implements client.APIClient for testing.
type mockAPIClient struct {
	doRequestResponse []byte
	doRequestErr      error
	streamingResponse []map[string]interface{}
	streamingErr      error
	capturedMethod    string
	capturedBaseURL   string
	capturedPath      string
	capturedToken     string
	capturedQuery     url.Values
}

func (m *mockAPIClient) DoRequest(ctx context.Context, method, baseURL, path, token string, query url.Values, body io.Reader) ([]byte, error) {
	m.capturedMethod = method
	m.capturedBaseURL = baseURL
	m.capturedPath = path
	m.capturedToken = token
	m.capturedQuery = query
	return m.doRequestResponse, m.doRequestErr
}

func (m *mockAPIClient) DoStreamingRequest(ctx context.Context, baseURL, path, token string, query url.Values) ([]map[string]interface{}, error) {
	m.capturedBaseURL = baseURL
	m.capturedPath = path
	m.capturedToken = token
	m.capturedQuery = query
	return m.streamingResponse, m.streamingErr
}

// mockTokenResolver implements client.TokenResolver for testing.
type mockTokenResolver struct {
	baseURL string
	token   string
	err     error
}

func (m *mockTokenResolver) Resolve(argocdBaseUrl string) (string, string, error) {
	return m.baseURL, m.token, m.err
}

func (m *mockTokenResolver) Invalidate(baseURL string) {
	// no-op for tests
}

func TestMock_ListApplications(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mockResp := `{"items": [{"metadata":{"name":"app1"},"spec":{"project":"default"},"status":{"health":{"status":"Healthy"}}}]}`
	mc := &mockAPIClient{doRequestResponse: []byte(mockResp)}
	mr := &mockTokenResolver{baseURL: "https://argocd.test", token: "mock-token"}

	Client = mc
	Resolver = mr

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	// Verify the mock captured correct values
	if mc.capturedMethod != "GET" {
		t.Errorf("method = %q, want GET", mc.capturedMethod)
	}
	if mc.capturedBaseURL != "https://argocd.test" {
		t.Errorf("baseURL = %q, want https://argocd.test", mc.capturedBaseURL)
	}
	if mc.capturedPath != "/api/v1/applications" {
		t.Errorf("path = %q, want /api/v1/applications", mc.capturedPath)
	}
	if mc.capturedToken != "mock-token" {
		t.Errorf("token = %q, want mock-token", mc.capturedToken)
	}

	// Verify response parsing
	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	if parsed["total"].(float64) != 1 {
		t.Errorf("total = %v, want 1", parsed["total"])
	}
}

func TestMock_ResolverError(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mc := &mockAPIClient{}
	mr := &mockTokenResolver{err: client.ErrNoBaseURL}

	Client = mc
	Resolver = mr

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when resolver returns error")
	}
}

func TestMock_ClientError(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mc := &mockAPIClient{doRequestErr: client.ErrAPIError.Parse(errors.WithParsedMessage(500, "internal error"))}
	mr := &mockTokenResolver{baseURL: "https://argocd.test", token: "mock-token"}

	Client = mc
	Resolver = mr

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when client returns error")
	}
}

func TestMock_StreamingRequest(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mc := &mockAPIClient{
		streamingResponse: []map[string]interface{}{
			{"content": "log line 1", "podName": "pod-1"},
			{"content": "log line 2", "podName": "pod-1"},
		},
	}
	mr := &mockTokenResolver{baseURL: "https://argocd.test", token: "mock-token"}

	Client = mc
	Resolver = mr

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"container":            "main",
		"resourceRef":          map[string]interface{}{"kind": "Pod", "name": "web"},
	}

	result, err := HandleGetApplicationWorkloadLogs(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	// Verify mock captured the streaming call correctly
	if mc.capturedPath != "/api/v1/applications/my-app/logs" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/logs", mc.capturedPath)
	}
	if mc.capturedQuery.Get("container") != "main" {
		t.Errorf("query container = %q, want main", mc.capturedQuery.Get("container"))
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed []map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)
	if len(parsed) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(parsed))
	}
}

func TestMock_InterfaceComplianceHTTPClient(t *testing.T) {
	// Compile-time interface compliance check
	var _ client.APIClient = (*client.HTTPClient)(nil)
}

func TestMock_InterfaceComplianceTokenResolver(t *testing.T) {
	// Compile-time interface compliance check
	var _ client.TokenResolver = (*client.RegistryTokenResolver)(nil)
}

// mockAPIClientWithRetry simulates a 401 on first call, success on second.
type mockAPIClientWith401 struct {
	callCount       int
	successResponse []byte
}

func (m *mockAPIClientWith401) DoRequest(ctx context.Context, method, baseURL, path, token string, query url.Values, body io.Reader) ([]byte, error) {
	m.callCount++
	if m.callCount == 1 {
		return nil, client.ErrAPIError.Parse(errors.WithParsedMessage(401, "invalid session"))
	}
	return m.successResponse, nil
}

func (m *mockAPIClientWith401) DoStreamingRequest(ctx context.Context, baseURL, path, token string, query url.Values) ([]map[string]interface{}, error) {
	m.callCount++
	if m.callCount == 1 {
		return nil, client.ErrAPIError.Parse(errors.WithParsedMessage(401, "invalid session"))
	}
	return []map[string]interface{}{{"content": "log"}}, nil
}

// mockTokenResolverWithInvalidate tracks invalidation calls.
type mockTokenResolverWithInvalidate struct {
	baseURL      string
	token        string
	invalidated  bool
	resolveCount int
}

func (m *mockTokenResolverWithInvalidate) Resolve(argocdBaseUrl string) (string, string, error) {
	m.resolveCount++
	return m.baseURL, m.token, nil
}

func (m *mockTokenResolverWithInvalidate) Invalidate(baseURL string) {
	m.invalidated = true
}

func TestDoWithAuth_RetryOn401(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mc := &mockAPIClientWith401{successResponse: []byte(`{"items":[]}`)}
	mr := &mockTokenResolverWithInvalidate{baseURL: "https://argocd.test", token: "token"}

	Client = mc
	Resolver = mr

	data, err := DoWithAuth(context.Background(), "", "GET", "/api/v1/applications", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"items":[]}` {
		t.Errorf("data = %q, want {\"items\":[]}", string(data))
	}
	if mc.callCount != 2 {
		t.Errorf("callCount = %d, want 2 (first 401, then retry)", mc.callCount)
	}
	if !mr.invalidated {
		t.Error("expected Invalidate to be called on 401")
	}
	if mr.resolveCount != 2 {
		t.Errorf("resolveCount = %d, want 2 (initial + retry)", mr.resolveCount)
	}
}

func TestDoStreamWithAuth_RetryOn401(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mc := &mockAPIClientWith401{}
	mr := &mockTokenResolverWithInvalidate{baseURL: "https://argocd.test", token: "token"}

	Client = mc
	Resolver = mr

	results, err := DoStreamWithAuth(context.Background(), "", "/api/v1/applications/app/logs", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("results len = %d, want 1", len(results))
	}
	if mc.callCount != 2 {
		t.Errorf("callCount = %d, want 2", mc.callCount)
	}
	if !mr.invalidated {
		t.Error("expected Invalidate to be called on 401")
	}
}

func TestDoWithAuth_NoRetryOnNon401(t *testing.T) {
	origClient := Client
	origResolver := Resolver
	t.Cleanup(func() {
		Client = origClient
		Resolver = origResolver
	})

	mc := &mockAPIClient{doRequestErr: client.ErrAPIError.Parse(errors.WithParsedMessage(500, "internal error"))}
	mr := &mockTokenResolver{baseURL: "https://argocd.test", token: "token"}

	Client = mc
	Resolver = mr

	_, err := DoWithAuth(context.Background(), "", "GET", "/api/v1/applications", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, client.ErrAPIError) {
		t.Errorf("expected ErrAPIError, got: %v", err)
	}
}
