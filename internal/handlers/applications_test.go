package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestIntegration_ListApplications(t *testing.T) {
	mockResp := `{
		"items": [
			{
				"metadata": {"name": "app1", "namespace": "argocd", "labels": {"team": "platform"}, "creationTimestamp": "2024-01-01T00:00:00Z"},
				"spec": {"project": "default", "source": {"repoURL": "git@example.com:repo.git"}, "destination": {"server": "https://kubernetes.default.svc"}},
				"status": {"sync": {"status": "Synced"}, "health": {"status": "Healthy"}, "summary": {"images": ["nginx:1.25"]}}
			},
			{
				"metadata": {"name": "app2", "namespace": "argocd", "labels": {}, "creationTimestamp": "2024-02-01T00:00:00Z"},
				"spec": {"project": "infra", "source": {"repoURL": "git@example.com:infra.git"}, "destination": {"server": "https://kubernetes.default.svc"}},
				"status": {"sync": {"status": "OutOfSync"}, "health": {"status": "Degraded"}, "summary": {}}
			}
		]
	}`

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Name = "argocd_list_applications"
	request.Params.Arguments = map[string]interface{}{}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	items := parsed["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	total := int(parsed["total"].(float64))
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
}

func TestIntegration_GetApplication(t *testing.T) {
	mockResp := `{
		"metadata": {"name": "my-app", "namespace": "argocd"},
		"spec": {"project": "default", "source": {"repoURL": "git@example.com:repo.git", "path": "manifests", "targetRevision": "main"}},
		"status": {"sync": {"status": "Synced"}, "health": {"status": "Healthy"}}
	}`

	var capturedPath string
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Name = "argocd_get_application"
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
	}

	result, err := HandleGetApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedPath != "/api/v1/applications/my-app" {
		t.Errorf("path = %q, want /api/v1/applications/my-app", capturedPath)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	meta := parsed["metadata"].(map[string]interface{})
	if meta["name"] != "my-app" {
		t.Errorf("metadata.name = %v, want 'my-app'", meta["name"])
	}
}

func TestIntegration_SyncApplication(t *testing.T) {
	var capturedBody string
	var capturedMethod string

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "syncing"}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Name = "argocd_sync_application"
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
		"dryRun":          true,
		"prune":           true,
		"revision":        "abc123",
	}

	result, err := HandleSyncApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}

	var syncBody map[string]interface{}
	json.Unmarshal([]byte(capturedBody), &syncBody)

	if syncBody["dryRun"] != true {
		t.Error("expected dryRun=true in request body")
	}
	if syncBody["prune"] != true {
		t.Error("expected prune=true in request body")
	}
	if syncBody["revision"] != "abc123" {
		t.Errorf("revision = %v, want 'abc123'", syncBody["revision"])
	}
}

func TestIntegration_CreateApplication(t *testing.T) {
	mockResp := `{"metadata": {"name": "new-app"}, "spec": {"project": "default"}}`

	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]interface{}
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"application": map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "new-app",
				"namespace": "argocd",
			},
			"spec": map[string]interface{}{
				"project": "default",
			},
		},
	}

	result, err := HandleCreateApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}
	if capturedPath != "/api/v1/applications" {
		t.Errorf("path = %q, want /api/v1/applications", capturedPath)
	}
	meta := capturedBody["metadata"].(map[string]interface{})
	if meta["name"] != "new-app" {
		t.Errorf("body metadata.name = %v, want new-app", meta["name"])
	}
}

func TestIntegration_UpdateApplication(t *testing.T) {
	mockResp := `{"metadata": {"name": "my-app"}, "spec": {"project": "updated"}}`

	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]interface{}
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
		"application": map[string]interface{}{
			"metadata": map[string]interface{}{"name": "my-app"},
			"spec":     map[string]interface{}{"project": "updated"},
		},
	}

	result, err := HandleUpdateApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "PUT" {
		t.Errorf("method = %q, want PUT", capturedMethod)
	}
	if capturedPath != "/api/v1/applications/my-app" {
		t.Errorf("path = %q, want /api/v1/applications/my-app", capturedPath)
	}
	spec := capturedBody["spec"].(map[string]interface{})
	if spec["project"] != "updated" {
		t.Errorf("body spec.project = %v, want updated", spec["project"])
	}
}

func TestIntegration_DeleteApplication(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedQuery url.Values
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
		"cascade":         false,
	}

	result, err := HandleDeleteApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", capturedMethod)
	}
	if capturedPath != "/api/v1/applications/my-app" {
		t.Errorf("path = %q, want /api/v1/applications/my-app", capturedPath)
	}
	if capturedQuery.Get("cascade") != "false" {
		t.Errorf("query cascade = %q, want false", capturedQuery.Get("cascade"))
	}
}

func TestIntegration_DeleteApplication_WithPropagationPolicy(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":   "my-app",
		"propagationPolicy": "foreground",
	}

	result, err := HandleDeleteApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}
}

func TestIntegration_RollbackApplication(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]interface{}
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "rolling back"}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName":      "my-app",
		"applicationNamespace": "argocd",
		"id":                   5,
	}

	result, err := HandleRollbackApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}
	if capturedPath != "/api/v1/applications/my-app/rollback" {
		t.Errorf("path = %q, want /api/v1/applications/my-app/rollback", capturedPath)
	}
	if capturedBody["id"] == nil {
		t.Error("expected 'id' in request body")
	}
	if int(capturedBody["id"].(float64)) != 5 {
		t.Errorf("body id = %v, want 5", capturedBody["id"])
	}
}

func TestIntegration_RollbackApplication_MissingID(t *testing.T) {
	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{
		"applicationName": "my-app",
	}

	result, err := HandleRollbackApplication(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing id")
	}
}

func TestListApplications_StripsHeavyFields(t *testing.T) {
	mockResp := `{
		"items": [{
			"metadata": {"name": "app1", "namespace": "argocd", "labels": {"team": "platform"}, "creationTimestamp": "2024-01-01T00:00:00Z", "uid": "stripped", "resourceVersion": "stripped", "managedFields": [{}], "annotations": {"x": "y"}},
			"spec": {"project": "default", "source": {"repoURL": "git@example.com:repo.git"}, "destination": {"server": "https://kubernetes.default.svc"}, "syncPolicy": {}, "ignoreDifferences": []},
			"status": {"sync": {"status": "Synced"}, "health": {"status": "Healthy"}, "summary": {}, "operationState": {}, "history": [], "resources": [], "conditions": []},
			"operation": {"sync": {}}
		}]
	}`

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResp))
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	items := parsed["items"].([]interface{})
	app := items[0].(map[string]interface{})

	meta := app["metadata"].(map[string]interface{})
	allowedMeta := map[string]bool{"name": true, "namespace": true, "labels": true, "creationTimestamp": true}
	for key := range meta {
		if !allowedMeta[key] {
			t.Errorf("metadata should not contain %q", key)
		}
	}

	spec := app["spec"].(map[string]interface{})
	allowedSpec := map[string]bool{"project": true, "source": true, "destination": true}
	for key := range spec {
		if !allowedSpec[key] {
			t.Errorf("spec should not contain %q", key)
		}
	}

	status := app["status"].(map[string]interface{})
	allowedStatus := map[string]bool{"sync": true, "health": true, "summary": true}
	for key := range status {
		if !allowedStatus[key] {
			t.Errorf("status should not contain %q", key)
		}
	}

	if _, exists := app["operation"]; exists {
		t.Error("top-level 'operation' field should be stripped")
	}
}

func TestListApplications_Pagination(t *testing.T) {
	items := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		items[i] = map[string]interface{}{
			"metadata": map[string]interface{}{"name": fmt.Sprintf("app%d", i), "namespace": "argocd", "labels": map[string]interface{}{}, "creationTimestamp": "2024-01-01T00:00:00Z"},
			"spec":     map[string]interface{}{"project": "default", "source": map[string]interface{}{"repoURL": "git@example.com:repo.git"}, "destination": map[string]interface{}{"server": "https://kubernetes.default.svc"}},
			"status":   map[string]interface{}{"sync": map[string]interface{}{"status": "Synced"}, "health": map[string]interface{}{"status": "Healthy"}},
		}
	}
	mockResp, _ := json.Marshal(map[string]interface{}{"items": items})

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(mockResp)
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"offset": 2, "limit": 2}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	total := int(parsed["total"].(float64))
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}

	returnedItems := parsed["items"].([]interface{})
	if len(returnedItems) != 2 {
		t.Fatalf("expected 2 items with limit=2, got %d", len(returnedItems))
	}

	firstName := returnedItems[0].(map[string]interface{})["metadata"].(map[string]interface{})["name"]
	if firstName != "app2" {
		t.Errorf("first item name = %v, want 'app2' (offset=2)", firstName)
	}
}

func TestListApplications_PaginationOffsetBeyondTotal(t *testing.T) {
	items := []map[string]interface{}{
		{"metadata": map[string]interface{}{"name": "app0", "namespace": "argocd", "labels": map[string]interface{}{}, "creationTimestamp": "2024-01-01T00:00:00Z"}, "spec": map[string]interface{}{"project": "default", "source": map[string]interface{}{}, "destination": map[string]interface{}{}}, "status": map[string]interface{}{"sync": map[string]interface{}{}, "health": map[string]interface{}{}}},
	}
	mockResp, _ := json.Marshal(map[string]interface{}{"items": items})

	setupMockArgoCD(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(mockResp)
	})

	request := mcp.CallToolRequest{}
	request.Params.Arguments = map[string]interface{}{"offset": 100}

	result, err := HandleListApplications(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var parsed map[string]interface{}
	json.Unmarshal([]byte(text), &parsed)

	returnedItems := parsed["items"].([]interface{})
	if len(returnedItems) != 0 {
		t.Errorf("expected 0 items for offset beyond total, got %d", len(returnedItems))
	}
}
