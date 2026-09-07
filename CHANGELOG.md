# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Release dates are not recorded here; derive them from the corresponding git tags.

## [Unreleased]

## [0.2.0]

### Added
- `argocd_get_appproject` - read AppProject (project) details: allowed sources, destinations, cluster/repository whitelists, and RBAC roles. Completes tool parity with the upstream argoproj-labs/mcp-for-argocd server (18 tools total: 12 read-only + 6 write).
- `make update-deps` target to update direct dependencies to latest, bump the Go toolchain, tidy, and verify with build + test.

### Changed
- Bump Go toolchain 1.26.5 -> 1.27.1.
- Bump github.com/mark3labs/mcp-go 0.57.0 -> 1.0.0.
- Test suite now exercises the real tool-registration functions (`registerReadTools`/`registerWriteTools`) and derives tool counts from a single source of truth instead of hardcoded numbers.

### Fixed
- Wrap the pointer-type errors-library sentinel via `ErrParseResponse.Parse(errors.WithError(err))` so the stricter `go vet` `%w` analyzer under Go 1.27 passes while preserving `errors.Is` matching.

## [0.1.3]

### Added
- THIRD_PARTY_NOTICES for dependency license compliance.

### Changed
- Bump actions/setup-go 6.5.0 -> 7.0.0.
- Bump SonarSource/sonarqube-scan-action 8.2.0 -> 8.2.1.
- Bump the CodeQL action group.

## [0.1.2]

### Changed
- Fix CodeQL version mismatch, add Codecov upload, and group Dependabot actions.

## [0.1.1]

### Added
- Tests for `requireFlagValue`, `resourceRefProperties`, and `parseArgs` combinations.

### Changed
- Bump all GitHub Actions to latest versions.

### Fixed
- Resolve SonarCloud issues (S1192 duplicate literals, S3776 cognitive complexity).

## [0.1.0]

Initial release: ArgoCD MCP server in Go with 17 tools (11 read-only + 6 write).

### Added
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
- errors-library adoption for structured error handling
- GitHub Actions pipeline, CodeQL, SonarCloud, Dependabot
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

[Unreleased]: https://github.com/jbcjorge/mcp-argo/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/jbcjorge/mcp-argo/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/jbcjorge/mcp-argo/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/jbcjorge/mcp-argo/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/jbcjorge/mcp-argo/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/jbcjorge/mcp-argo/releases/tag/v0.1.0
