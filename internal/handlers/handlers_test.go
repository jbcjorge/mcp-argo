package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/jbcjorge/mcp-argo/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

func init() {
	config.Cfg = &config.Config{}
	client.InitHTTPClient(false)
	Client = client.NewHTTPClient(false)
	Resolver = client.NewTokenResolver(config.Cfg)
}

func setupCfg(t *testing.T) {
	t.Helper()
	original := config.Cfg
	origResolver := Resolver
	config.Cfg = &config.Config{}
	Resolver = client.NewTokenResolver(config.Cfg)
	t.Cleanup(func() {
		config.Cfg = original
		Resolver = origResolver
	})
}

func setupMockArgoCD(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	setupCfg(t)
	config.Cfg.DefaultBaseURL = ts.URL
	config.Cfg.DefaultToken = "test-token"
	Resolver = client.NewTokenResolver(config.Cfg)

	return ts
}

// =============================================================================
// Unit tests for helper functions
// =============================================================================

func TestErrResult(t *testing.T) {
	testErr := fmt.Errorf("something went wrong")
	result, err := ErrResult(testErr)
	if err != nil {
		t.Fatalf("ErrResult returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true from ErrResult")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if text != "something went wrong" {
		t.Errorf("error text = %q, want %q", text, "something went wrong")
	}
}

func TestGetArgObject_Present(t *testing.T) {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"resourceRef": map[string]interface{}{
			"kind":      "Deployment",
			"name":      "nginx",
			"namespace": "default",
		},
	}

	obj := GetArgObject(request, "resourceRef")
	if obj == nil {
		t.Fatal("expected non-nil object")
	}
	if obj["kind"] != "Deployment" {
		t.Errorf("kind = %v, want Deployment", obj["kind"])
	}
	if obj["name"] != "nginx" {
		t.Errorf("name = %v, want nginx", obj["name"])
	}
}

func TestGetArgObject_Missing(t *testing.T) {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}

	obj := GetArgObject(request, "resourceRef")
	if obj != nil {
		t.Errorf("expected nil for missing key, got %v", obj)
	}
}

func TestGetArgObject_WrongType(t *testing.T) {
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"resourceRef": "not-an-object",
	}

	obj := GetArgObject(request, "resourceRef")
	if obj != nil {
		t.Errorf("expected nil for non-object value, got %v", obj)
	}
}

func TestBuildResourceRefQuery(t *testing.T) {
	resourceRef := map[string]interface{}{
		"namespace": "my-ns",
		"name":      "my-deploy",
		"group":     "apps",
		"kind":      "Deployment",
		"version":   "v1",
	}

	query := url.Values{}
	BuildResourceRefQuery(resourceRef, query)

	if query.Get("namespace") != "my-ns" {
		t.Errorf("namespace = %q, want my-ns", query.Get("namespace"))
	}
	if query.Get("resourceName") != "my-deploy" {
		t.Errorf("resourceName = %q, want my-deploy", query.Get("resourceName"))
	}
	if query.Get("group") != "apps" {
		t.Errorf("group = %q, want apps", query.Get("group"))
	}
	if query.Get("kind") != "Deployment" {
		t.Errorf("kind = %q, want Deployment", query.Get("kind"))
	}
	if query.Get("version") != "v1" {
		t.Errorf("version = %q, want v1", query.Get("version"))
	}
}

func TestBuildResourceRefQuery_EmptyFields(t *testing.T) {
	resourceRef := map[string]interface{}{
		"namespace": "",
		"name":      "",
	}

	query := url.Values{}
	BuildResourceRefQuery(resourceRef, query)

	if query.Get("namespace") != "" {
		t.Errorf("namespace should be empty, got %q", query.Get("namespace"))
	}
	if query.Get("resourceName") != "" {
		t.Errorf("resourceName should be empty, got %q", query.Get("resourceName"))
	}
}

func TestBuildResourceRefQuery_PartialFields(t *testing.T) {
	resourceRef := map[string]interface{}{
		"kind": "Service",
		"name": "my-svc",
	}

	query := url.Values{}
	BuildResourceRefQuery(resourceRef, query)

	if query.Get("kind") != "Service" {
		t.Errorf("kind = %q, want Service", query.Get("kind"))
	}
	if query.Get("resourceName") != "my-svc" {
		t.Errorf("resourceName = %q, want my-svc", query.Get("resourceName"))
	}
	if query.Has("namespace") {
		t.Error("namespace should not be set")
	}
}
