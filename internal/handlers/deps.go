package handlers

import "github.com/jbcjorge/mcp-argo/internal/client"

// Client is the API client used by all handlers for HTTP requests.
var Client client.APIClient

// Resolver is the token resolver used by all handlers for authentication.
var Resolver client.TokenResolver
