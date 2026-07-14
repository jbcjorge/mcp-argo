# mcp-argo

A Model Context Protocol (MCP) server for [Argo CD](https://argo-cd.readthedocs.io/), written in Go. Reimplementation of [argoproj-labs/mcp-for-argocd](https://github.com/argoproj-labs/mcp-for-argocd) (TypeScript) as a single binary with lower resource usage.

Single binary, no runtime dependencies, ~18 MB RSS at runtime.

## Features

- **17 ArgoCD tools** (11 read-only + 6 write)
- **Three transport modes**: stdio, SSE, streamable HTTP
- **Multi-instance support**: token registry for targeting multiple ArgoCD clusters
- **Read-only mode**: disable write operations via environment variable
- **Stateless HTTP mode**: for Kubernetes deployments without sticky sessions
- **Security**: default token is never sent to a non-default base URL

## Installation

### From source (recommended)

```bash
go install github.com/jbcjorge/mcp-argo@latest
```

### Build locally

```bash
git clone https://github.com/jbcjorge/mcp-argo.git
cd mcp-argo
make install  # builds, codesigns (macOS), and copies to ~/.local/bin/mcp-argo
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/jbcjorge/mcp-argo/releases).

On macOS, downloaded binaries are quarantined by Gatekeeper. Remove the quarantine attribute before running:

```bash
xattr -d com.apple.quarantine mcp-argo_darwin_arm64
chmod +x mcp-argo_darwin_arm64
mv mcp-argo_darwin_arm64 ~/.local/bin/mcp-argo
```

> This is standard for Go CLI tools distributed via GitHub releases. Building from source (`go install`) avoids this entirely.

## Usage

### stdio (default)

```bash
export ARGOCD_BASE_URL="https://argocd.example.com"
export ARGOCD_API_TOKEN="<your-token>"
export ARGOCD_INSECURE=true  # skip TLS verify for self-signed certs

mcp-argo
```

### SSE transport

```bash
mcp-argo sse --port 8080
```

Endpoints:
- `GET /sse` - event stream
- `POST /message` - send messages
- `GET /health` - health check

### Streamable HTTP transport

```bash
mcp-argo http --port 8080
```

Endpoints:
- `POST /mcp` - MCP requests
- `GET /health` - health check

### Stateless HTTP (for Kubernetes HPA)

```bash
mcp-argo http --port 8080 --stateless
```

No session ID required; any replica can handle any request.

## Configuration

### Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ARGOCD_BASE_URL` | Default ArgoCD instance URL | (none) |
| `ARGOCD_API_TOKEN` | API token for the default instance | (none) |
| `ARGOCD_INSECURE` | Skip TLS certificate verification | `false` |
| `MCP_READ_ONLY` | Disable write tools (create, update, delete, sync, rollback, run_resource_action) | `false` |
| `ARGOCD_TOKEN_REGISTRY_PATH` | Path to token registry JSON file for multi-instance | (none) |

### CLI flags

```
mcp-argo [command] [flags]

Commands:
  stdio              Start with stdio transport (default)
  sse                Start with SSE transport
  http               Start with streamable HTTP transport

Flags:
  --port int         Port for SSE/HTTP transports (default 8080)
  --host string      Host to bind for SSE/HTTP transports (default "127.0.0.1")
  --stateless        Disable session management (http only)
  --help, -h         Show help message
```

### Multi-instance (token registry)

Create a JSON file with tokens for each ArgoCD instance:

```json
[
  {"baseUrl": "https://argocd.app-a.example.com", "token": "<token-a>"},
  {"baseUrl": "https://argocd.app-b.example.com", "token": "<token-b>"}
]
```

Set `ARGOCD_TOKEN_REGISTRY_PATH=/path/to/tokens.json`. Each tool call can then target a specific instance via the `argocdBaseUrl` argument without including the token.

### Per-call base URL override

Every tool accepts an optional `argocdBaseUrl` argument to target a specific ArgoCD instance. The token is resolved from:
1. The default token (if URL matches the default base URL)
2. The token registry (for any other URL)

The default token is **never** sent to a non-default URL (prevents token exfiltration).

## Available tools

### Read-only (always available)

| Tool | Description |
|------|-------------|
| `argocd_list_applications` | List applications with optional search, limit, offset |
| `argocd_list_clusters` | List registered clusters |
| `argocd_get_application` | Get detailed application info |
| `argocd_get_application_resource_tree` | Get resource tree (pods, deployments, etc.) |
| `argocd_get_application_managed_resources` | Get managed resources with filters |
| `argocd_get_application_workload_logs` | Get workload logs (pods, deployments) |
| `argocd_get_application_events` | Get application events |
| `argocd_get_application_sync_windows` | Check if syncs are allowed/blocked |
| `argocd_get_resource_events` | Get events for a specific resource |
| `argocd_get_resource_actions` | Get available actions for a resource |
| `argocd_get_resources` | Get actual Kubernetes manifests |

### Write (disabled in read-only mode)

| Tool | Description |
|------|-------------|
| `argocd_create_application` | Create a new application |
| `argocd_update_application` | Update an existing application |
| `argocd_delete_application` | Delete an application |
| `argocd_sync_application` | Trigger a sync operation |
| `argocd_rollback_application` | Rollback to a previous revision |
| `argocd_run_resource_action` | Run an action on a resource |

## MCP client configuration examples

### Cursor / VSCode

```json
{
  "mcpServers": {
    "argocd": {
      "command": "mcp-argo",
      "env": {
        "ARGOCD_BASE_URL": "https://argocd.example.com",
        "ARGOCD_API_TOKEN": "<token>",
        "ARGOCD_INSECURE": "true",
        "MCP_READ_ONLY": "true"
      }
    }
  }
}
```

### Claude Desktop

```json
{
  "mcpServers": {
    "argocd": {
      "command": "/path/to/mcp-argo",
      "env": {
        "ARGOCD_BASE_URL": "https://argocd.example.com",
        "ARGOCD_API_TOKEN": "<token>",
        "ARGOCD_INSECURE": "true"
      }
    }
  }
}
```

### Behind an MCP gateway (stdio proxy)

```json
{
  "argocd": {
    "command": ["bash", "/path/to/mcp-argo-wrapper.sh"]
  }
}
```

Where the wrapper script obtains the token (from keychain, SSO, vault, etc.) and exec's the binary.

## Obtaining an ArgoCD API token

### Option A: Generate from a local account

```bash
# Create a local account in ArgoCD (requires admin)
# Then generate a token:
argocd account generate-token --account <account-name>
```

### Option B: Use the argocd CLI session token

```bash
argocd login argocd.example.com --sso
# Token stored in ~/.config/argocd/config
```

### Option C: Extract from SSO session cookie

ArgoCD sets an `argocd.token` cookie after SSO login. This JWT can be used as the API token.

## Development

```bash
make build    # compile
make test     # run tests
make install  # build + copy to ~/.local/bin
make clean    # remove binary
```

## License

Apache License 2.0
