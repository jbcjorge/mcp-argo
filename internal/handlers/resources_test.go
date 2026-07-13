package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jbcjorge/mcp-argo/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestIntegration_GetApplicationResourceTree(t *testing.T) {
	mockResp := `{
		"nodes": [
			{"kind": "Deployment", "name": "nginx", "namespace": "default", "version": "v1", "group": "apps"},
			{"kind": "Service", "name": "nginx-svc", "namespace": "default", "version": "v1", "group": ""}
		]
	}`

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/resource-tree") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
	}

	result, err := HandleGetApplicationResourceTree(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	nodes := parsed["nodes"].([]interface{})
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestIntegration_GetApplicationManagedResources(t *testing.T) {
	mockResp := `{
		"items": [
			{"kind": "Deployment", "name": "web", "namespace": "default"},
			{"kind": "Service", "name": "web-svc", "namespace": "default"}
		]
	}`

	var capturedPath string
	var capturedQuery url.Values
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
		"kind":            "Deployment",
		"namespace":       "default",
	}

	result, err := HandleGetApplicationManagedResources(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/applications/my-app/managed-resources" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/managed-resources", capturedPath)
	}
	if capturedQuery.Get("kind") != "Deployment" {
		t.Errorf("query kind = %q, want Deployment", capturedQuery.Get("kind"))
	}
	if capturedQuery.Get("namespace") != "default" {
		t.Errorf("query namespace = %q, want default", capturedQuery.Get("namespace"))
	}
}

func TestIntegration_GetResources(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/resource") {
			w.Write([]byte(`{"manifest": "{\"kind\":\"Deployment\",\"metadata\":{\"name\":\"nginx\"}}"}`))
		} else if strings.HasSuffix(r.URL.Path, "/resource-tree") {
			w.Write([]byte(`{"nodes": [{"kind": "Deployment", "name": "nginx", "namespace": "default", "version": "v1", "group": "apps"}]}`))
		}
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
	}

	result, err := HandleGetResources(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed []interface{}
	json.Unmarshal([]byte(text), &parsed)

	if len(parsed) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(parsed))
	}

	manifest := parsed[0].(map[string]interface{})
	if manifest["kind"] != "Deployment" {
		t.Errorf("kind = %v, want Deployment", manifest["kind"])
	}
}

func TestIntegration_GetResources_WithExplicitRefs(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/resource") {
			w.Write([]byte(`{"manifest": "{\"kind\":\"Service\",\"metadata\":{\"name\":\"web\"}}"}`))
		}
	})

	refs := `[{"kind":"Service","name":"web","namespace":"default","version":"v1","group":""}]`
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"resourceRefs":         refs,
	}

	result, err := HandleGetResources(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed []interface{}
	json.Unmarshal([]byte(text), &parsed)

	if len(parsed) != 1 {
		t.Errorf("expected 1 manifest, got %d", len(parsed))
	}
}

func TestIntegration_GetResourceActions(t *testing.T) {
	mockResp := `{"actions": [{"name": "restart", "disabled": false}, {"name": "pause", "disabled": true}]}`

	var capturedPath string
	var capturedQuery url.Values
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"resourceRef": map[string]interface{}{
			"kind": "Deployment", "name": "nginx", "namespace": "default", "group": "apps", "version": "v1",
		},
	}

	result, err := HandleGetResourceActions(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/applications/my-app/resource/actions" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/resource/actions", capturedPath)
	}
	if capturedQuery.Get("kind") != "Deployment" {
		t.Errorf("query kind = %q, want Deployment", capturedQuery.Get("kind"))
	}
	if capturedQuery.Get("resourceName") != "nginx" {
		t.Errorf("query resourceName = %q, want nginx", capturedQuery.Get("resourceName"))
	}
}

func TestIntegration_RunResourceAction(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedQuery url.Values
	var capturedBody string
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"action":               "restart",
		"resourceRef": map[string]interface{}{
			"kind": "Deployment", "name": "nginx", "namespace": "default", "group": "apps", "version": "v1",
		},
	}

	result, err := HandleRunResourceAction(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}
	if capturedPath != "/api/v1/applications/my-app/resource/actions" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/resource/actions", capturedPath)
	}
	if capturedQuery.Get("kind") != "Deployment" {
		t.Errorf("query kind = %q, want Deployment", capturedQuery.Get("kind"))
	}
	if !strings.Contains(capturedBody, "restart") {
		t.Errorf("body = %q, want to contain 'restart'", capturedBody)
	}
}

func TestIntegration_GetResourceEvents(t *testing.T) {
	mockResp := `{"items": [{"type": "Normal", "reason": "Pulled", "message": "Successfully pulled image"}]}`

	var capturedPath string
	var capturedQuery url.Values
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"resourceUID":          "uid-12345",
		"resourceNamespace":    "default",
		"resourceName":         "nginx-pod",
	}

	result, err := HandleGetResourceEvents(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/applications/my-app/events" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/events", capturedPath)
	}
	if capturedQuery.Get("resourceUID") != "uid-12345" {
		t.Errorf("query resourceUID = %q, want uid-12345", capturedQuery.Get("resourceUID"))
	}
}

// Error path tests for resources

func TestIntegration_GetApplicationResourceTree_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}

	result, _ := HandleGetApplicationResourceTree(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetApplicationManagedResources_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}

	result, _ := HandleGetApplicationManagedResources(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetResources_MissingNamespace(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}

	result, _ := HandleGetResources(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing applicationNamespace")
	}
}

func TestIntegration_GetResources_InvalidResourceRefs(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"resourceRefs":         "not-valid-json",
	}

	result, _ := HandleGetResources(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid resourceRefs JSON")
	}
}

func TestIntegration_GetResourceActions_MissingResourceRef(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
	}

	result, _ := HandleGetResourceActions(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing resourceRef")
	}
}

func TestIntegration_RunResourceAction_MissingAction(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"resourceRef":          map[string]interface{}{"kind": "Deployment", "name": "nginx"},
	}

	result, _ := HandleRunResourceAction(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing action")
	}
}

func TestIntegration_RunResourceAction_MissingResourceRef(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"action":               "restart",
	}

	result, _ := HandleRunResourceAction(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing resourceRef")
	}
}

func TestIntegration_GetResourceEvents_MissingFields(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}

	result, _ := HandleGetResourceEvents(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing required fields")
	}
}
