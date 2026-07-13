package client

import (
	"context"
	"io"
	"net/url"
)

// APIClient defines the HTTP transport interface for ArgoCD API calls.
type APIClient interface {
	// DoRequest performs an HTTP request and returns the response body.
	DoRequest(ctx context.Context, method, baseURL, path, token string, query url.Values, body io.Reader) ([]byte, error)
	// DoStreamingRequest performs a streaming GET and collects NDJSON results.
	DoStreamingRequest(ctx context.Context, baseURL, path, token string, query url.Values) ([]map[string]interface{}, error)
}

// TokenResolver resolves ArgoCD base URL and authentication token.
type TokenResolver interface {
	// Resolve returns the base URL and token for the given argocdBaseUrl argument.
	// If argocdBaseUrl is empty, it uses the default.
	Resolve(argocdBaseUrl string) (baseURL, token string, err error)
	// Invalidate removes a cached token for the given base URL, forcing
	// re-resolution on the next Resolve call (e.g., after a 401 response).
	Invalidate(baseURL string)
}
