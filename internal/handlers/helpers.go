package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	errors "github.com/jbcjorge/errors-library"
	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// DoWithAuth resolves the token for the given base URL, performs the request,
// and retries once with a fresh token if the server returns 401.
func DoWithAuth(ctx context.Context, argocdBaseUrl, method, path string, query url.Values, body io.Reader) ([]byte, error) {
	baseURL, token, err := Resolver.Resolve(argocdBaseUrl)
	if err != nil {
		return nil, err
	}

	data, err := Client.DoRequest(ctx, method, baseURL, path, token, query, body)
	if err != nil && isUnauthorized(err) {
		Resolver.Invalidate(baseURL)
		baseURL, token, err = Resolver.Resolve(argocdBaseUrl)
		if err != nil {
			return nil, err
		}
		data, err = Client.DoRequest(ctx, method, baseURL, path, token, query, body)
	}
	return data, err
}

// DoStreamWithAuth is like DoWithAuth but for streaming requests.
func DoStreamWithAuth(ctx context.Context, argocdBaseUrl, path string, query url.Values) ([]map[string]interface{}, error) {
	baseURL, token, err := Resolver.Resolve(argocdBaseUrl)
	if err != nil {
		return nil, err
	}

	data, err := Client.DoStreamingRequest(ctx, baseURL, path, token, query)
	if err != nil && isUnauthorized(err) {
		Resolver.Invalidate(baseURL)
		baseURL, token, err = Resolver.Resolve(argocdBaseUrl)
		if err != nil {
			return nil, err
		}
		data, err = Client.DoStreamingRequest(ctx, baseURL, path, token, query)
	}
	return data, err
}

func isUnauthorized(err error) bool {
	return errors.Is(err, client.ErrAPIError) && strings.Contains(err.Error(), "HTTP 401")
}

// ResourceRef represents a reference to a Kubernetes resource in ArgoCD.
type ResourceRef struct {
	UID       string `json:"uid"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Group     string `json:"group"`
}

// JsonResult marshals v to JSON and returns it as a tool result.
func JsonResult(v interface{}) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// ErrResult wraps an error into a tool error result.
func ErrResult(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}

// GetArgString extracts a string argument from the request.
func GetArgString(request mcp.CallToolRequest, key string) string {
	return request.GetString(key, "")
}

// GetArgInt extracts an integer argument from the request.
func GetArgInt(request mcp.CallToolRequest, key string, defaultVal int) int {
	return request.GetInt(key, defaultVal)
}

// GetArgBool extracts a boolean argument from the request.
func GetArgBool(request mcp.CallToolRequest, key string, defaultVal bool) bool {
	return request.GetBool(key, defaultVal)
}

// GetArgObject extracts a map argument from the request.
func GetArgObject(request mcp.CallToolRequest, key string) map[string]interface{} {
	args := request.GetArguments()
	if val, ok := args[key]; ok {
		if obj, ok := val.(map[string]interface{}); ok {
			return obj
		}
	}
	return nil
}

// GetArgStringSlice extracts a string slice argument from the request.
func GetArgStringSlice(request mcp.CallToolRequest, key string) []string {
	return request.GetStringSlice(key, nil)
}

// BuildResourceRefQuery extracts resource reference fields from a map and adds them to query params.
func BuildResourceRefQuery(resourceRef map[string]interface{}, query url.Values) {
	if ns, ok := resourceRef["namespace"].(string); ok && ns != "" {
		query.Set("namespace", ns)
	}
	if name, ok := resourceRef["name"].(string); ok && name != "" {
		query.Set("resourceName", name)
	}
	if group, ok := resourceRef["group"].(string); ok && group != "" {
		query.Set("group", group)
	}
	if kind, ok := resourceRef["kind"].(string); ok && kind != "" {
		query.Set("kind", kind)
	}
	if ver, ok := resourceRef["version"].(string); ok && ver != "" {
		query.Set("version", ver)
	}
}

// StripApplicationItem extracts only lightweight fields from a single application item.
func StripApplicationItem(app map[string]interface{}) map[string]interface{} {
	light := map[string]interface{}{}

	if meta, ok := app["metadata"].(map[string]interface{}); ok {
		lightMeta := map[string]interface{}{}
		for _, key := range []string{"name", "namespace", "labels", "creationTimestamp"} {
			if v, exists := meta[key]; exists {
				lightMeta[key] = v
			}
		}
		light["metadata"] = lightMeta
	}

	if spec, ok := app["spec"].(map[string]interface{}); ok {
		lightSpec := map[string]interface{}{}
		for _, key := range []string{"project", "source", "destination"} {
			if v, exists := spec[key]; exists {
				lightSpec[key] = v
			}
		}
		light["spec"] = lightSpec
	}

	if status, ok := app["status"].(map[string]interface{}); ok {
		lightStatus := map[string]interface{}{}
		for _, key := range []string{"sync", "health", "summary"} {
			if v, exists := status[key]; exists {
				lightStatus[key] = v
			}
		}
		light["status"] = lightStatus
	}

	return light
}

// StripApplicationFields processes a list of application items, keeping only lightweight fields.
func StripApplicationFields(items []interface{}) []map[string]interface{} {
	var stripped []map[string]interface{}
	for _, item := range items {
		app, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		stripped = append(stripped, StripApplicationItem(app))
	}
	return stripped
}

// ApplyPagination applies offset and limit to a slice of maps, returning the paginated subset.
func ApplyPagination(items []map[string]interface{}, limit, offset int) []map[string]interface{} {
	if offset > 0 {
		if offset >= len(items) {
			return []map[string]interface{}{}
		}
		items = items[offset:]
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}

// FetchResourceRefs fetches the resource tree for an application and returns the node refs.
func FetchResourceRefs(ctx context.Context, argocdBaseUrl, applicationName, appNamespace string) ([]ResourceRef, error) {
	query := url.Values{}
	query.Set("appNamespace", appNamespace)

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/resource-tree"
	data, err := DoWithAuth(ctx, argocdBaseUrl, "GET", path, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resource tree: %w", err)
	}

	var treeResp struct {
		Nodes []ResourceRef `json:"nodes"`
	}
	if err := json.Unmarshal(data, &treeResp); err != nil {
		return nil, fmt.Errorf("failed to parse resource tree: %w", err)
	}
	return treeResp.Nodes, nil
}

// FetchResourceManifest fetches a single resource manifest and returns it as an interface.
func FetchResourceManifest(ctx context.Context, argocdBaseUrl, applicationName, appNamespace string, ref ResourceRef) interface{} {
	query := url.Values{}
	query.Set("appNamespace", appNamespace)
	if ref.Namespace != "" {
		query.Set("namespace", ref.Namespace)
	}
	if ref.Name != "" {
		query.Set("resourceName", ref.Name)
	}
	if ref.Group != "" {
		query.Set("group", ref.Group)
	}
	if ref.Kind != "" {
		query.Set("kind", ref.Kind)
	}
	if ref.Version != "" {
		query.Set("version", ref.Version)
	}

	path := "/api/v1/applications/" + url.PathEscape(applicationName) + "/resource"
	data, err := DoWithAuth(ctx, argocdBaseUrl, "GET", path, query, nil)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
			"ref":   ref,
		}
	}

	var resourceResp struct {
		Manifest string `json:"manifest"`
	}
	if err := json.Unmarshal(data, &resourceResp); err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to parse resource response: %v", err),
			"ref":   ref,
		}
	}

	var manifest interface{}
	if err := json.Unmarshal([]byte(resourceResp.Manifest), &manifest); err != nil {
		return resourceResp.Manifest
	}
	return manifest
}
