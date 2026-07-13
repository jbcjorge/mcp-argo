package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jbcjorge/mcp-argo/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestIntegration_ListClusters(t *testing.T) {
	mockResp := `{"items": [{"server": "https://kubernetes.default.svc", "name": "in-cluster"}, {"server": "https://remote.example.com", "name": "remote-cluster"}]}`

	var capturedPath string
	var capturedQuery url.Values
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"name": "remote"}

	result, err := HandleListClusters(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/clusters" {
		t.Errorf("path = %q, want /api/v1/clusters", capturedPath)
	}
	if capturedQuery.Get("name") != "remote" {
		t.Errorf("query name = %q, want remote", capturedQuery.Get("name"))
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)
	items := parsed["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(items))
	}
}

func TestIntegration_GetApplicationWorkloadLogs(t *testing.T) {
	ndjson := "{\"result\": {\"content\": \"line 1\\n\", \"podName\": \"web-abc123\"}}\n{\"result\": {\"content\": \"line 2\\n\", \"podName\": \"web-abc123\"}}\n{\"result\": {\"content\": \"line 3\\n\", \"podName\": \"web-abc123\"}}\n"

	var capturedPath string
	var capturedQuery url.Values
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(ndjson))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"container":            "main",
		"resourceRef": map[string]interface{}{
			"kind": "Pod", "name": "web-abc123", "namespace": "default", "version": "v1",
		},
	}

	result, err := HandleGetApplicationWorkloadLogs(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/applications/my-app/logs" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/logs", capturedPath)
	}
	if capturedQuery.Get("container") != "main" {
		t.Errorf("query container = %q, want main", capturedQuery.Get("container"))
	}
	if capturedQuery.Get("appNamespace") != "argocd" {
		t.Errorf("query appNamespace = %q, want argocd", capturedQuery.Get("appNamespace"))
	}
	if capturedQuery.Get("kind") != "Pod" {
		t.Errorf("query kind = %q, want Pod", capturedQuery.Get("kind"))
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed []map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)
	if len(parsed) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(parsed))
	}
}

func TestIntegration_GetApplicationEvents(t *testing.T) {
	mockResp := `{"items": [{"type": "Normal", "reason": "Synced"}, {"type": "Warning", "reason": "SyncError"}]}`

	var capturedPath string
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
	}

	result, err := HandleGetApplicationEvents(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/applications/my-app/events" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/events", capturedPath)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)
	items := parsed["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 events, got %d", len(items))
	}
}

func TestIntegration_GetApplicationSyncWindows(t *testing.T) {
	mockResp := `{"assignedWindows": [{"kind": "allow", "schedule": "0 0 * * *"}], "canSync": true}`

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/syncwindows") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}

	result, err := HandleGetApplicationSyncWindows(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)
	if parsed["canSync"] != true {
		t.Error("expected canSync=true")
	}
}

// =============================================================================
// No base URL error paths
// =============================================================================

func TestIntegration_ListClusters_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleListClusters(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetApplicationEvents_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplicationEvents(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetApplicationSyncWindows_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplicationSyncWindows(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetApplicationWorkloadLogs_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app", "applicationNamespace": "argocd",
		"container": "main", "resourceRef": map[string]interface{}{"kind": "Pod", "name": "web"},
	}
	result, _ := HandleGetApplicationWorkloadLogs(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetResourceEvents_NoBaseURL(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app", "applicationNamespace": "argocd",
		"resourceUID": "uid-1", "resourceNamespace": "default", "resourceName": "nginx",
	}
	result, _ := HandleGetResourceEvents(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true")
	}
}

func TestIntegration_GetApplicationWorkloadLogs_MissingResourceRef(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app", "applicationNamespace": "argocd", "container": "main",
	}
	result, _ := HandleGetApplicationWorkloadLogs(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing resourceRef")
	}
}

// =============================================================================
// Invalid JSON response tests
// =============================================================================

func TestIntegration_ListClusters_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleListClusters(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetApplication_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{broken`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetApplicationResourceTree_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{broken`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplicationResourceTree(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetApplicationManagedResources_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplicationManagedResources(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetApplicationEvents_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplicationEvents(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetApplicationSyncWindows_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{invalid`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleGetApplicationSyncWindows(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetResourceEvents_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app", "applicationNamespace": "argocd",
		"resourceUID": "uid-123", "resourceNamespace": "default", "resourceName": "nginx",
	}
	result, _ := HandleGetResourceEvents(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_GetResourceActions_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app", "applicationNamespace": "argocd",
		"resourceRef": map[string]interface{}{"kind": "Deployment", "name": "nginx"},
	}
	result, _ := HandleGetResourceActions(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON")
	}
}

func TestIntegration_CreateApplication_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"application": map[string]interface{}{"metadata": map[string]interface{}{"name": "app"}},
	}
	result, _ := HandleCreateApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON response")
	}
}

func TestIntegration_UpdateApplication_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
		"application":     map[string]interface{}{"metadata": map[string]interface{}{"name": "my-app"}},
	}
	result, _ := HandleUpdateApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON response")
	}
}

func TestIntegration_DeleteApplication_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleDeleteApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON response")
	}
}

func TestIntegration_SyncApplication_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleSyncApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON response")
	}
}

func TestIntegration_RollbackApplication_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app", "id": 3}
	result, _ := HandleRollbackApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON response")
	}
}

func TestIntegration_ListApplications_InvalidJSON(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not-json`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleListApplications(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for invalid JSON response")
	}
}

func TestIntegration_ListApplications_APIError(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"internal server error"}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleListApplications(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for 500 response")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "internal server error") {
		t.Errorf("error text = %q, want to contain 'internal server error'", text)
	}
}

func TestIntegration_HandleError_InvalidToken(t *testing.T) {
	setupCfg(t)
	config.Cfg.DefaultBaseURL = ""
	config.Cfg.DefaultToken = ""

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleListApplications(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true when no base URL configured")
	}
}

func TestIntegration_GetApplication_MissingName(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleGetApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing applicationName")
	}
}

func TestIntegration_GetApplicationResourceTree_MissingName(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleGetApplicationResourceTree(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing applicationName")
	}
}

func TestIntegration_SyncApplication_MissingName(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleSyncApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing applicationName")
	}
}

func TestIntegration_CreateApplication_MissingApplication(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleCreateApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing application arg")
	}
}

func TestIntegration_UpdateApplication_MissingApplication(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"applicationName": "my-app"}
	result, _ := HandleUpdateApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing application arg")
	}
}

func TestIntegration_DeleteApplication_MissingName(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}
	result, _ := HandleDeleteApplication(context.Background(), request)
	if !result.IsError {
		t.Error("expected IsError=true for missing applicationName")
	}
}
