package handlers

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleGetApplicationWorkloadLogs gets logs from a workload managed by an ArgoCD application.
func HandleGetApplicationWorkloadLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	appNamespace, err := request.RequireString("applicationNamespace")
	if err != nil {
		return ErrResult(err)
	}
	container, err := request.RequireString("container")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")

	resourceRef := GetArgObject(request, "resourceRef")
	if resourceRef == nil {
		return ErrResult(fmt.Errorf("required argument \"resourceRef\" not found"))
	}

	query := url.Values{}
	query.Set("appNamespace", appNamespace)
	query.Set("container", container)
	query.Set("follow", "false")
	query.Set("tailLines", "100")
	BuildResourceRefQuery(resourceRef, query)

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/logs"
	results, err := DoStreamWithAuth(ctx, argocdBaseUrl, path, query)
	if err != nil {
		return ErrResult(err)
	}

	return JsonResult(results)
}
