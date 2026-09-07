package handlers

import (
	"context"
	"encoding/json"
	"net/url"

	errors "github.com/jbcjorge/errors-library"
	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleGetAppProject returns an ArgoCD AppProject (project) by its name.
func HandleGetAppProject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectName, err := request.RequireString("projectName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")

	path := "/api/v1/projects/" + url.PathEscape(projectName)
	data, err := DoWithAuth(ctx, argocdBaseUrl, "GET", path, nil, nil)
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ErrResult(client.ErrParseResponse.Parse(errors.WithError(err)))
	}

	return JsonResult(resp)
}
