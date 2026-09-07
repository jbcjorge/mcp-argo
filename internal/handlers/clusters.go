package handlers

import (
	"context"
	"encoding/json"
	"net/url"

	errors "github.com/jbcjorge/errors-library"
	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleListClusters lists all clusters registered in ArgoCD.
func HandleListClusters(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	serverFilter := GetArgString(request, "server")
	nameFilter := GetArgString(request, "name")

	query := url.Values{}
	if serverFilter != "" {
		query.Set("server", serverFilter)
	}
	if nameFilter != "" {
		query.Set("name", nameFilter)
	}

	data, err := DoWithAuth(ctx, argocdBaseUrl, "GET", "/api/v1/clusters", query, nil)
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(client.ErrParseResponse.Parse(errors.WithError(err)))
	}

	return JsonResult(resp)
}
