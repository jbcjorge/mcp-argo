package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestIntegration_GetAppProject(t *testing.T) {
	mockResp := `{
		"metadata": {"name": "my-project", "namespace": "argocd"},
		"spec": {
			"sourceRepos": ["*"],
			"destinations": [{"server": "https://kubernetes.default.svc", "namespace": "*"}],
			"clusterResourceWhitelist": [{"group": "*", "kind": "*"}]
		}
	}`

	var capturedPath string
	var capturedMethod string
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Name = "argocd_get_appproject"
	request.Params.Arguments = map[string]interface{}{
		"projectName": "my-project",
	}

	result, err := HandleGetAppProject(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "GET" {
		t.Errorf("method = %q, want GET", capturedMethod)
	}
	if capturedPath != "/api/v1/projects/my-project" {
		t.Errorf("path = %q, want /api/v1/projects/my-project", capturedPath)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	meta := parsed["metadata"].(map[string]interface{})
	if meta["name"] != "my-project" {
		t.Errorf("metadata.name = %v, want 'my-project'", meta["name"])
	}
}

func TestIntegration_GetAppProject_MissingName(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Name = "argocd_get_appproject"
	request.Params.Arguments = map[string]interface{}{}

	result, err := HandleGetAppProject(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing projectName")
	}
}
