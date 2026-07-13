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

// HandleGetApplicationResourceTree gets the resource tree of an ArgoCD application.
func HandleGetApplicationResourceTree(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/resource-tree"
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

// HandleGetApplicationManagedResources gets managed resources of an ArgoCD application.
func HandleGetApplicationManagedResources(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")

	query := url.Values{}
	for _, key := range []string{"kind", "namespace", "name", "version", "group", "appNamespace", "project"} {
		if v := GetArgString(request, key); v != "" {
			query.Set(key, v)
		}
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/managed-resources"
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

// HandleGetResources gets full resource manifests for resources managed by an ArgoCD application.
func HandleGetResources(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	appNamespace, err := request.RequireString("applicationNamespace")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")
	resourceRefsJSON := GetArgString(request, "resourceRefs")

	var refs []ResourceRef

	if resourceRefsJSON != "" {
		if unmarshalErr := json.Unmarshal([]byte(resourceRefsJSON), &refs); unmarshalErr != nil {
			return ErrResult(fmt.Errorf("failed to parse resourceRefs JSON: %w", unmarshalErr))
		}
	}

	if len(refs) == 0 {
		refs, err = FetchResourceRefs(ctx, argocdBaseUrl, applicationName, appNamespace)
		if err != nil {
			return ErrResult(err)
		}
	}

	var manifests []interface{}
	for _, ref := range refs {
		manifests = append(manifests, FetchResourceManifest(ctx, argocdBaseUrl, applicationName, appNamespace, ref))
	}

	return JsonResult(manifests)
}

// HandleGetResourceActions gets available actions for a resource managed by an ArgoCD application.
func HandleGetResourceActions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	appNamespace, err := request.RequireString("applicationNamespace")
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
	BuildResourceRefQuery(resourceRef, query)

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/resource/actions"
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

// HandleGetResourceEvents gets Kubernetes events for a specific resource.
func HandleGetResourceEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	appNamespace, err := request.RequireString("applicationNamespace")
	if err != nil {
		return ErrResult(err)
	}
	resourceUID, err := request.RequireString("resourceUID")
	if err != nil {
		return ErrResult(err)
	}
	resourceNamespace, err := request.RequireString("resourceNamespace")
	if err != nil {
		return ErrResult(err)
	}
	resourceName, err := request.RequireString("resourceName")
	if err != nil {
		return ErrResult(err)
	}
	argocdBaseUrl := GetArgString(request, "argocdBaseUrl")

	query := url.Values{}
	query.Set("appNamespace", appNamespace)
	query.Set("resourceNamespace", resourceNamespace)
	query.Set("resourceUID", resourceUID)
	query.Set("resourceName", resourceName)

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

// HandleRunResourceAction runs an action on a resource managed by an ArgoCD application.
func HandleRunResourceAction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	applicationName, err := request.RequireString("applicationName")
	if err != nil {
		return ErrResult(err)
	}
	appNamespace, err := request.RequireString("applicationNamespace")
	if err != nil {
		return ErrResult(err)
	}
	action, err := request.RequireString("action")
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
	BuildResourceRefQuery(resourceRef, query)

	body, err := json.Marshal(action)
	if err != nil {
		return ErrResult(fmt.Errorf("failed to marshal action: %w", err))
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/resource/actions"
	data, err := DoWithAuth(ctx, argocdBaseUrl, "POST", path, query, strings.NewReader(string(body)))
	if err != nil {
		return ErrResult(err)
	}

	var resp interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		// Some actions return empty body on success
		return mcp.NewToolResultText("Action executed successfully"), nil
	}

	return JsonResult(resp)
}
