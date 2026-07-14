package handlers

// Tool description constants for repeated strings (SonarCloud S1192).
const (
	DescArgocdBaseURL        = "ArgoCD instance URL (uses default if not provided)"
	DescApplicationName      = "Name of the application"
	DescApplicationNamespace = "Namespace of the application (for multi-namespace mode)"
	DescAppNamespace         = "Namespace of the application"
	DescResourceUID          = "UID of the resource"
	DescResourceKind         = "Kind of the resource"
	DescResourceNamespace    = "Namespace of the resource"
	DescResourceName         = "Name of the resource"
	DescResourceVersion      = "API version of the resource"
	DescResourceGroup        = "API group of the resource"
)

// apiApplicationsPath is the common API path prefix for ArgoCD application endpoints.
const apiApplicationsPath = "/api/v1/applications/"
