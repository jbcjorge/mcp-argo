package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jbcjorge/mcp-argo/internal/client"
	"github.com/jbcjorge/mcp-argo/internal/config"
	"github.com/jbcjorge/mcp-argo/internal/handlers"
	internalserver "github.com/jbcjorge/mcp-argo/internal/server"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// version is set at build time via -ldflags
var version = "dev"

// defaultLogLevel can be overridden at build time via -ldflags
var defaultLogLevel = "info"

const (
	defaultHTTPPort = 8080
	defaultHost     = "127.0.0.1"
)

func main() {
	transport, host, port, stateless, logLevelArg := parseArgs()
	config.InitLogging(defaultLogLevel, "LOG_LEVEL", logLevelArg)
	config.LoadConfig()

	if config.Cfg.Insecure {
		slog.Warn("TLS certificate verification disabled via ARGOCD_INSECURE")
	}

	client.InitHTTPClient(config.Cfg.Insecure)

	// Wire interface-based dependencies for handlers
	handlers.Client = client.NewHTTPClient(config.Cfg.Insecure)
	handlers.Resolver = client.NewTokenResolver(config.Cfg)

	s := server.NewMCPServer(
		"mcp-argo",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	registerReadTools(s)
	if !config.Cfg.ReadOnly {
		registerWriteTools(s)
	}

	switch transport {
	case "stdio":
		if stateless {
			fmt.Fprintf(os.Stderr, "Error: --stateless is only supported with the http command\n")
			os.Exit(1)
		}
		slog.Info("server starting", "transport", "stdio", "version", version)
		if err := server.ServeStdio(s); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	case "sse":
		if stateless {
			fmt.Fprintf(os.Stderr, "Error: --stateless is only supported with the http command\n")
			os.Exit(1)
		}
		addr := host + ":" + strconv.Itoa(port)
		sseServer := server.NewSSEServer(s)
		slog.Info("server starting", "transport", "sse", "addr", addr, "version", version)
		mux := http.NewServeMux()
		mux.HandleFunc("/health", internalserver.HealthHandler(version))
		mux.Handle("/", sseServer)
		httpSrv := &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: internalserver.ReadHeaderTimeout,
			ReadTimeout:       internalserver.ReadTimeout,
			WriteTimeout:      internalserver.WriteTimeout,
			IdleTimeout:       internalserver.IdleTimeout,
		}
		go func() {
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("server error", "transport", "sse", "err", err)
				os.Exit(1)
			}
		}()
		internalserver.WaitForShutdown(httpSrv)
	case "http":
		addr := host + ":" + strconv.Itoa(port)
		var httpOpts []server.StreamableHTTPOption
		if stateless {
			httpOpts = append(httpOpts, server.WithStateLess(true))
		}
		streamableServer := server.NewStreamableHTTPServer(s, httpOpts...)
		slog.Info("server starting", "transport", "http", "addr", addr, "stateless", stateless, "version", version)
		mux := http.NewServeMux()
		mux.HandleFunc("/health", internalserver.HealthHandler(version))
		mux.Handle("/", streamableServer)
		httpSrv := &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: internalserver.ReadHeaderTimeout,
			ReadTimeout:       internalserver.ReadTimeout,
			WriteTimeout:      internalserver.WriteTimeout,
			IdleTimeout:       internalserver.IdleTimeout,
		}
		go func() {
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("server error", "transport", "http", "err", err)
				os.Exit(1)
			}
		}()
		internalserver.WaitForShutdown(httpSrv)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", transport)
		printUsage()
		os.Exit(1)
	}
}

func registerReadTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("argocd_list_applications",
		mcp.WithDescription("List all ArgoCD applications with optional search and pagination"),
		mcp.WithString("search", mcp.Description("Filter applications by name (partial match)")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of applications to return")),
		mcp.WithNumber("offset", mcp.Description("Number of applications to skip for pagination")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleListApplications)

	s.AddTool(mcp.NewTool("argocd_list_clusters",
		mcp.WithDescription("List all clusters registered in ArgoCD"),
		mcp.WithString("server", mcp.Description("Filter by cluster server URL")),
		mcp.WithString("name", mcp.Description("Filter by cluster name")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleListClusters)

	s.AddTool(mcp.NewTool("argocd_get_application",
		mcp.WithDescription("Get detailed information about a specific ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application (for multi-namespace mode)")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetApplication)

	s.AddTool(mcp.NewTool("argocd_get_application_resource_tree",
		mcp.WithDescription("Get the resource tree of an ArgoCD application showing all managed resources and their relationships"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application (for multi-namespace mode)")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetApplicationResourceTree)

	s.AddTool(mcp.NewTool("argocd_get_application_managed_resources",
		mcp.WithDescription("Get managed resources of an ArgoCD application with optional filtering"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("kind", mcp.Description("Filter by resource kind")),
		mcp.WithString("namespace", mcp.Description("Filter by resource namespace")),
		mcp.WithString("name", mcp.Description("Filter by resource name")),
		mcp.WithString("version", mcp.Description("Filter by resource API version")),
		mcp.WithString("group", mcp.Description("Filter by resource API group")),
		mcp.WithString("appNamespace", mcp.Description("Application namespace")),
		mcp.WithString("project", mcp.Description("Filter by project")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetApplicationManagedResources)

	s.AddTool(mcp.NewTool("argocd_get_application_workload_logs",
		mcp.WithDescription("Get logs from a workload managed by an ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Required(), mcp.Description("Namespace of the application")),
		mcp.WithObject("resourceRef", mcp.Required(), mcp.Description("Resource reference object with uid, kind, namespace, name, version, group"),
			mcp.Properties(map[string]any{
				"uid":       map[string]any{"type": "string", "description": "UID of the resource"},
				"kind":      map[string]any{"type": "string", "description": "Kind of the resource"},
				"namespace": map[string]any{"type": "string", "description": "Namespace of the resource"},
				"name":      map[string]any{"type": "string", "description": "Name of the resource"},
				"version":   map[string]any{"type": "string", "description": "API version of the resource"},
				"group":     map[string]any{"type": "string", "description": "API group of the resource"},
			}),
		),
		mcp.WithString("container", mcp.Required(), mcp.Description("Container name to get logs from")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetApplicationWorkloadLogs)

	s.AddTool(mcp.NewTool("argocd_get_application_events",
		mcp.WithDescription("Get Kubernetes events for an ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application (for multi-namespace mode)")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetApplicationEvents)

	s.AddTool(mcp.NewTool("argocd_get_resource_events",
		mcp.WithDescription("Get Kubernetes events for a specific resource managed by an ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Required(), mcp.Description("Namespace of the application")),
		mcp.WithString("resourceUID", mcp.Required(), mcp.Description("UID of the resource")),
		mcp.WithString("resourceNamespace", mcp.Required(), mcp.Description("Namespace of the resource")),
		mcp.WithString("resourceName", mcp.Required(), mcp.Description("Name of the resource")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetResourceEvents)

	s.AddTool(mcp.NewTool("argocd_get_resource_actions",
		mcp.WithDescription("Get available actions for a resource managed by an ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Required(), mcp.Description("Namespace of the application")),
		mcp.WithObject("resourceRef", mcp.Required(), mcp.Description("Resource reference object"),
			mcp.Properties(map[string]any{
				"uid":       map[string]any{"type": "string", "description": "UID of the resource"},
				"kind":      map[string]any{"type": "string", "description": "Kind of the resource"},
				"namespace": map[string]any{"type": "string", "description": "Namespace of the resource"},
				"name":      map[string]any{"type": "string", "description": "Name of the resource"},
				"version":   map[string]any{"type": "string", "description": "API version of the resource"},
				"group":     map[string]any{"type": "string", "description": "API group of the resource"},
			}),
		),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetResourceActions)

	s.AddTool(mcp.NewTool("argocd_get_resources",
		mcp.WithDescription("Get full resource manifests for resources managed by an ArgoCD application. If resourceRefs is not provided, fetches all resources from the application's resource tree."),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Required(), mcp.Description("Namespace of the application")),
		mcp.WithString("resourceRefs", mcp.Description("JSON array of resource references [{uid, kind, namespace, name, version, group}]. If empty, fetches all resources from the resource tree.")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetResources)

	s.AddTool(mcp.NewTool("argocd_get_application_sync_windows",
		mcp.WithDescription("Get sync window status for an ArgoCD application (active/inactive windows, whether syncs are currently blocked)"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application (for multi-namespace mode)")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleGetApplicationSyncWindows)
}

func registerWriteTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("argocd_create_application",
		mcp.WithDescription("Create a new ArgoCD application"),
		mcp.WithObject("application", mcp.Required(), mcp.Description("Application object with metadata.name, metadata.namespace, spec.project, spec.source, spec.destination, spec.syncPolicy")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleCreateApplication)

	s.AddTool(mcp.NewTool("argocd_update_application",
		mcp.WithDescription("Update an existing ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application to update")),
		mcp.WithObject("application", mcp.Required(), mcp.Description("Updated application object")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleUpdateApplication)

	s.AddTool(mcp.NewTool("argocd_delete_application",
		mcp.WithDescription("Delete an ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application to delete")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application")),
		mcp.WithBoolean("cascade", mcp.Description("Cascade deletion to application resources (default: true)")),
		mcp.WithString("propagationPolicy", mcp.Description("Resource propagation policy (foreground, background, orphan)")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleDeleteApplication)

	s.AddTool(mcp.NewTool("argocd_sync_application",
		mcp.WithDescription("Sync an ArgoCD application to its target state"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application to sync")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application")),
		mcp.WithBoolean("dryRun", mcp.Description("Perform a dry run without making changes")),
		mcp.WithBoolean("prune", mcp.Description("Allow pruning of resources not in git")),
		mcp.WithString("revision", mcp.Description("Specific revision/commit to sync to")),
		mcp.WithArray("syncOptions", mcp.Description("Sync options (e.g., Replace=true, Force=true)"), mcp.WithStringItems()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleSyncApplication)

	s.AddTool(mcp.NewTool("argocd_run_resource_action",
		mcp.WithDescription("Run an action on a resource managed by an ArgoCD application"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("applicationNamespace", mcp.Required(), mcp.Description("Namespace of the application")),
		mcp.WithObject("resourceRef", mcp.Required(), mcp.Description("Resource reference object"),
			mcp.Properties(map[string]any{
				"uid":       map[string]any{"type": "string", "description": "UID of the resource"},
				"kind":      map[string]any{"type": "string", "description": "Kind of the resource"},
				"namespace": map[string]any{"type": "string", "description": "Namespace of the resource"},
				"name":      map[string]any{"type": "string", "description": "Name of the resource"},
				"version":   map[string]any{"type": "string", "description": "API version of the resource"},
				"group":     map[string]any{"type": "string", "description": "API group of the resource"},
			}),
		),
		mcp.WithString("action", mcp.Required(), mcp.Description("Name of the action to run")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleRunResourceAction)

	s.AddTool(mcp.NewTool("argocd_rollback_application",
		mcp.WithDescription("Rollback an ArgoCD application to a previous revision"),
		mcp.WithString("applicationName", mcp.Required(), mcp.Description("Name of the application to rollback")),
		mcp.WithString("applicationNamespace", mcp.Description("Namespace of the application")),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Revision ID to rollback to")),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL (uses default if not provided)")),
	), handlers.HandleRollbackApplication)
}

func parseArgs() (transport, host string, port int, stateless bool, logLevel string) {
	transport = "stdio"
	host = defaultHost
	port = defaultHTTPPort
	stateless = false
	logLevel = ""

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		case "--version", "-v":
			fmt.Println(version)
			os.Exit(0)
		case "--port":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: --port requires a value\n")
				os.Exit(1)
			}
			p, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid port value: %s\n", args[i])
				os.Exit(1)
			}
			port = p
		case "--host":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: --host requires a value\n")
				os.Exit(1)
			}
			host = args[i]
		case "--log-level":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: --log-level requires a value\n")
				os.Exit(1)
			}
			logLevel = args[i]
		case "--stateless":
			stateless = true
		default:
			if strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(os.Stderr, "Error: unknown flag: %s\n", args[i])
				printUsage()
				os.Exit(1)
			}
			// First non-flag argument is the command
			if transport == "stdio" {
				transport = args[i]
			}
		}
	}

	return transport, host, port, stateless, logLevel
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: mcp-argo [command] [flags]

Commands:
  stdio              Start with stdio transport (default if no command given)
  sse                Start with SSE transport
  http               Start with streamable HTTP transport

Flags:
  --port int         Port for SSE/HTTP transports (default 8080)
  --host string      Host to bind for SSE/HTTP transports (default "127.0.0.1")
  --stateless        Disable session management (http only). No session ID is
                     assigned; any replica can handle any request. Credentials
                     must be supplied on every request via env vars or headers.
  --log-level string Log level: debug, info, warn, error (default from
                     LOG_LEVEL env or "info"). CLI flag takes precedence.
  --help, -h         Show this help message
  --version, -v      Show version
`)
}
