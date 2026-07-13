package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleListApplications lists all ArgoCD applications with optional search and pagination.
func HandleListApplications(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	search := GetArgString(request, "search")
	limit := GetArgInt(request, "limit", 0)
	offset := GetArgInt(request, "offset", 0)

	query := url.Values{}
	if search != "" {
		query.Set("name", "*"+search+"*")
	}

	data, err := DoWithAuth(ctx, argocdBaseUrl, "GET", "/api/v1/applications", query, nil)
	if err != nil {
		return ErrResult(err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	items, _ := resp["items"].([]interface{})
	if items == nil {
		items = []interface{}{}
	}

	stripped := StripApplicationFields(items)
	total := len(stripped)
	stripped = ApplyPagination(stripped, limit, offset)

	result := map[string]interface{}{
		"items": stripped,
		"total": total,
	}

	return JsonResult(result)
}

// HandleGetApplication gets detailed information about a specific ArgoCD application.
func HandleGetApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	appNamespace := GetArgString(request, "applicationNamespace")

	query := url.Values{}
	if appNamespace != "" {
		query.Set("appNamespace", appNamespace)
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName)
	data, err := DoWithAuth(ctx, argocdBaseUrl, "GET", path, query, nil)
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	return JsonResult(resp)
}

// HandleCreateApplication creates a new ArgoCD application.
func HandleCreateApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	application := GetArgObject(request, "application")
	if application == nil {
		return ErrResult(fmt.Errorf("required argument \"application\" not found"))
	}

	body, err := json.Marshal(application)
	if err != nil {
		return ErrResult(fmt.Errorf("failed to marshal application: %w", err))
	}

	data, err := DoWithAuth(ctx, argocdBaseUrl, "POST", "/api/v1/applications", nil, strings.NewReader(string(body)))
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	return JsonResult(resp)
}

// HandleUpdateApplication updates an existing ArgoCD application.
func HandleUpdateApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	application := GetArgObject(request, "application")
	if application == nil {
		return ErrResult(fmt.Errorf("required argument \"application\" not found"))
	}

	body, err := json.Marshal(application)
	if err != nil {
		return ErrResult(fmt.Errorf("failed to marshal application: %w", err))
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName)
	data, err := DoWithAuth(ctx, argocdBaseUrl, "PUT", path, nil, strings.NewReader(string(body)))
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	return JsonResult(resp)
}

// HandleDeleteApplication deletes an ArgoCD application.
func HandleDeleteApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	appNamespace := GetArgString(request, "applicationNamespace")
	cascade := GetArgBool(request, "cascade", true)
	propagationPolicy := GetArgString(request, "propagationPolicy")

	query := url.Values{}
	if appNamespace != "" {
		query.Set("appNamespace", appNamespace)
	}
	if !cascade {
		query.Set("cascade", "false")
	}
	if propagationPolicy != "" {
		query.Set("propagationPolicy", propagationPolicy)
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName)
	data, err := DoWithAuth(ctx, argocdBaseUrl, "DELETE", path, query, nil)
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	return JsonResult(resp)
}

// HandleSyncApplication syncs an ArgoCD application to its target state.
func HandleSyncApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	appNamespace := GetArgString(request, "applicationNamespace")
	dryRun := GetArgBool(request, "dryRun", false)
	prune := GetArgBool(request, "prune", false)
	revision := GetArgString(request, "revision")
	syncOptions := GetArgStringSlice(request, "syncOptions")

	query := url.Values{}
	if appNamespace != "" {
		query.Set("appNamespace", appNamespace)
	}

	syncBody := map[string]interface{}{}
	if dryRun {
		syncBody["dryRun"] = true
	}
	if prune {
		syncBody["prune"] = true
	}
	if revision != "" {
		syncBody["revision"] = revision
	}
	if len(syncOptions) > 0 {
		syncBody["syncOptions"] = map[string]interface{}{
			"items": syncOptions,
		}
	}

	body, err := json.Marshal(syncBody)
	if err != nil {
		return ErrResult(fmt.Errorf("failed to marshal sync body: %w", err))
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/sync"
	data, err := DoWithAuth(ctx, argocdBaseUrl, "POST", path, query, strings.NewReader(string(body)))
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	return JsonResult(resp)
}

// HandleRollbackApplication rolls back an ArgoCD application to a previous revision.
func HandleRollbackApplication(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	appNamespace := GetArgString(request, "applicationNamespace")
	revisionID := GetArgInt(request, "id", 0)
	if revisionID == 0 {
		return ErrResult(fmt.Errorf("required argument \"id\" not found or is zero"))
	}

	query := url.Values{}
	if appNamespace != "" {
		query.Set("appNamespace", appNamespace)
	}

	rollbackBody := map[string]interface{}{
		"id": int64(revisionID),
	}
	body, err := json.Marshal(rollbackBody)
	if err != nil {
		return ErrResult(fmt.Errorf("failed to marshal rollback body: %w", err))
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/rollback"
	data, err := DoWithAuth(ctx, argocdBaseUrl, "POST", path, query, strings.NewReader(string(body)))
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(fmt.Errorf("%w: %w", client.ErrParseResponse, err))
	}

	return JsonResult(resp)
}
