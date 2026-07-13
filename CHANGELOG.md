# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release with 17 ArgoCD tools (11 read-only + 6 write)
- Three transport modes: stdio, SSE, streamable HTTP
- Stateless HTTP mode for Kubernetes deployments without sticky sessions
- Multi-instance token registry support (ARGOCD_TOKEN_REGISTRY_PATH)
- Per-call argocdBaseUrl override for targeting different instances
- Read-only mode via MCP_READ_ONLY=true
- Graceful shutdown on SIGINT/SIGTERM for HTTP/SSE modes
- Structured logging with log/slog (configurable: build-time, env, CLI flag)
- Security: default token bound to default URL only (prevents token exfiltration)
- TLS skip verify via ARGOCD_INSECURE=true
- HTTP client with 60s timeout, 50MB response limit, connection pooling
- Context propagation from handlers to HTTP requests
- Sentinel errors for programmatic error handling
- Health endpoint with version info (GET /health)
- Version flag (--version)
- goreleaser configuration for automated releases

### Tools (read-only)
- `argocd_list_applications` - list/search with pagination
- `argocd_list_clusters` - list registered clusters
- `argocd_get_application` - full application details
- `argocd_get_application_resource_tree` - resource hierarchy
- `argocd_get_application_managed_resources` - managed resources with filters
- `argocd_get_application_workload_logs` - pod/workload logs
- `argocd_get_application_events` - application events
- `argocd_get_application_sync_windows` - sync window status
- `argocd_get_resource_events` - resource-level events
- `argocd_get_resource_actions` - available resource actions
- `argocd_get_resources` - fetch Kubernetes manifests

### Tools (write)
- `argocd_create_application` - create new application
- `argocd_update_application` - update existing application
- `argocd_delete_application` - delete application
- `argocd_sync_application` - trigger sync
- `argocd_rollback_application` - rollback to revision
- `argocd_run_resource_action` - execute resource action
