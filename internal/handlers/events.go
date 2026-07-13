package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleGetApplicationEvents gets Kubernetes events for an ArgoCD application.
func HandleGetApplicationEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/events"
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

// HandleGetApplicationSyncWindows gets sync window status for an ArgoCD application.
func HandleGetApplicationSyncWindows(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/syncwindows"
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
