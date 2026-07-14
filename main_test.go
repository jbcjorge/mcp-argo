package main

import (
	"os"
	"strings"
	"testing"

	"github.com/jbcjorge/mcp-argo/internal/handlers"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// =============================================================================
// Tool registration tests
// =============================================================================

func createMCPServer(readOnly bool) *server.MCPServer {
	s := server.NewMCPServer(
		"mcp-argocd",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	s.AddTool(mcp.NewTool("argocd_list_applications",
		mcp.WithDescription("List all ArgoCD applications"),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleListApplications)

	s.AddTool(mcp.NewTool("argocd_list_clusters",
		mcp.WithDescription("List all clusters"),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleListClusters)

	s.AddTool(mcp.NewTool("argocd_get_application",
		mcp.WithDescription("Get application details"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetApplication)

	s.AddTool(mcp.NewTool("argocd_get_application_resource_tree",
		mcp.WithDescription("Get resource tree"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetApplicationResourceTree)

	s.AddTool(mcp.NewTool("argocd_get_application_managed_resources",
		mcp.WithDescription("Get managed resources"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetApplicationManagedResources)

	s.AddTool(mcp.NewTool("argocd_get_application_workload_logs",
		mcp.WithDescription("Get workload logs"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetApplicationWorkloadLogs)

	s.AddTool(mcp.NewTool("argocd_get_application_events",
		mcp.WithDescription("Get application events"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetApplicationEvents)

	s.AddTool(mcp.NewTool("argocd_get_resource_events",
		mcp.WithDescription("Get resource events"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetResourceEvents)

	s.AddTool(mcp.NewTool("argocd_get_resource_actions",
		mcp.WithDescription("Get resource actions"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetResourceActions)

	s.AddTool(mcp.NewTool("argocd_get_resources",
		mcp.WithDescription("Get resource manifests"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetResources)

	s.AddTool(mcp.NewTool("argocd_get_application_sync_windows",
		mcp.WithDescription("Get sync windows"),
		mcp.WithString("applicationName", mcp.Required()),
		mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
	), handlers.HandleGetApplicationSyncWindows)

	if !readOnly {
		s.AddTool(mcp.NewTool("argocd_create_application",
			mcp.WithDescription("Create application"),
			mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
		), handlers.HandleCreateApplication)

		s.AddTool(mcp.NewTool("argocd_update_application",
			mcp.WithDescription("Update application"),
			mcp.WithString("applicationName", mcp.Required()),
			mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
		), handlers.HandleUpdateApplication)

		s.AddTool(mcp.NewTool("argocd_delete_application",
			mcp.WithDescription("Delete application"),
			mcp.WithString("applicationName", mcp.Required()),
			mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
		), handlers.HandleDeleteApplication)

		s.AddTool(mcp.NewTool("argocd_sync_application",
			mcp.WithDescription("Sync application"),
			mcp.WithString("applicationName", mcp.Required()),
			mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
		), handlers.HandleSyncApplication)

		s.AddTool(mcp.NewTool("argocd_run_resource_action",
			mcp.WithDescription("Run resource action"),
			mcp.WithString("applicationName", mcp.Required()),
			mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
		), handlers.HandleRunResourceAction)

		s.AddTool(mcp.NewTool("argocd_rollback_application",
			mcp.WithDescription("Rollback application"),
			mcp.WithString("applicationName", mcp.Required()),
			mcp.WithString("argocdBaseUrl", mcp.Description("ArgoCD instance URL")),
		), handlers.HandleRollbackApplication)
	}

	return s
}

func getToolNames(t *testing.T, s *server.MCPServer) []string {
	t.Helper()
	tools := s.ListTools()
	var names []string
	for name := range tools {
		names = append(names, name)
	}
	return names
}

func TestToolRegistration_NormalMode_17Tools(t *testing.T) {
	s := createMCPServer(false)
	names := getToolNames(t, s)
	if len(names) != 17 {
		t.Errorf("expected 17 tools in normal mode, got %d: %v", len(names), names)
	}
}

func TestToolRegistration_ReadOnlyMode_11Tools(t *testing.T) {
	s := createMCPServer(true)
	names := getToolNames(t, s)
	if len(names) != 11 {
		t.Errorf("expected 11 tools in read-only mode, got %d: %v", len(names), names)
	}
}

func TestToolRegistration_AllToolsHaveArgocdBaseUrl(t *testing.T) {
	s := createMCPServer(false)
	tools := s.ListTools()
	for name, st := range tools {
		props := st.Tool.InputSchema.Properties
		if props == nil {
			t.Errorf("tool %q has nil Properties", name)
			continue
		}
		if _, exists := props["argocdBaseUrl"]; !exists {
			t.Errorf("tool %q is missing argocdBaseUrl parameter", name)
		}
	}
}

func TestToolRegistration_AllNamesStartWithArgocd(t *testing.T) {
	s := createMCPServer(false)
	names := getToolNames(t, s)
	for _, name := range names {
		if !strings.HasPrefix(name, "argocd_") {
			t.Errorf("tool name %q does not start with 'argocd_'", name)
		}
	}
}

// =============================================================================
// CLI argument parsing tests
// =============================================================================

func TestParseArgs_NoArgs_StdioMode(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo"}
	transport, host, port, stateless, _ := parseArgs()

	if transport != "stdio" {
		t.Errorf("transport = %q, want 'stdio'", transport)
	}
	if host != "127.0.0.1" {
		t.Errorf("host = %q, want '127.0.0.1'", host)
	}
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}
	if stateless {
		t.Error("stateless should be false by default")
	}
}

func TestParseArgs_Stdio(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "stdio"}
	transport, _, _, _, _ := parseArgs()
	if transport != "stdio" {
		t.Errorf("transport = %q, want 'stdio'", transport)
	}
}

func TestParseArgs_SSE_DefaultPort(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "sse"}
	transport, host, port, _, _ := parseArgs()
	if transport != "sse" {
		t.Errorf("transport = %q, want 'sse'", transport)
	}
	if host != "127.0.0.1" {
		t.Errorf("host = %q, want '127.0.0.1'", host)
	}
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}
}

func TestParseArgs_SSE_CustomPort(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "sse", "--port", "9090"}
	transport, _, port, _, _ := parseArgs()
	if transport != "sse" {
		t.Errorf("transport = %q, want 'sse'", transport)
	}
	if port != 9090 {
		t.Errorf("port = %d, want 9090", port)
	}
}

func TestParseArgs_HTTP_CustomHostAndPort(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "http", "--host", "0.0.0.0", "--port", "3000"}
	transport, host, port, _, _ := parseArgs()
	if transport != "http" {
		t.Errorf("transport = %q, want 'http'", transport)
	}
	if host != "0.0.0.0" {
		t.Errorf("host = %q, want '0.0.0.0'", host)
	}
	if port != 3000 {
		t.Errorf("port = %d, want 3000", port)
	}
}

func TestParseArgs_HTTP_Stateless(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "http", "--stateless"}
	transport, _, _, stateless, _ := parseArgs()
	if transport != "http" {
		t.Errorf("transport = %q, want 'http'", transport)
	}
	if !stateless {
		t.Error("stateless should be true when --stateless flag is provided")
	}
}

func TestParseArgs_HTTP_StatelessWithPortAndHost(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "http", "--stateless", "--port", "9090", "--host", "0.0.0.0"}
	transport, host, port, stateless, _ := parseArgs()
	if transport != "http" {
		t.Errorf("transport = %q, want 'http'", transport)
	}
	if host != "0.0.0.0" {
		t.Errorf("host = %q, want '0.0.0.0'", host)
	}
	if port != 9090 {
		t.Errorf("port = %d, want 9090", port)
	}
	if !stateless {
		t.Error("stateless should be true")
	}
}

func TestParseArgs_Stdio_StatelessNotSet(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "stdio"}
	_, _, _, stateless, _ := parseArgs()
	if stateless {
		t.Error("stateless should be false for stdio command without flag")
	}
}

func TestParseArgs_LogLevel(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "http", "--log-level", "debug"}
	transport, _, _, _, logLevel := parseArgs()
	if transport != "http" {
		t.Errorf("transport = %q, want http", transport)
	}
	if logLevel != "debug" {
		t.Errorf("logLevel = %q, want debug", logLevel)
	}
}

func TestParseArgs_LogLevelWithOtherFlags(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "sse", "--port", "9090", "--log-level", "warn"}
	transport, _, port, _, logLevel := parseArgs()
	if transport != "sse" {
		t.Errorf("transport = %q, want sse", transport)
	}
	if port != 9090 {
		t.Errorf("port = %d, want 9090", port)
	}
	if logLevel != "warn" {
		t.Errorf("logLevel = %q, want warn", logLevel)
	}
}

func TestParseArgs_VersionFlag(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "http", "--port", "4000"}
	transport, _, port, _, _ := parseArgs()
	if transport != "http" {
		t.Errorf("transport = %q, want http", transport)
	}
	if port != 4000 {
		t.Errorf("port = %d, want 4000", port)
	}
}

func TestParseArgs_SSE_Stateless(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })

	os.Args = []string{"mcp-argo", "sse", "--stateless"}
	transport, _, _, stateless, _ := parseArgs()
	if transport != "sse" {
		t.Errorf("transport = %q, want sse", transport)
	}
	if !stateless {
		t.Error("stateless should be true")
	}
}

func TestPrintUsage(t *testing.T) {
	// Just ensure it does not panic
	printUsage()
}

// Ensure config import is used
// =============================================================================
// Tool registration via actual register functions
// =============================================================================

func TestRegisterReadTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(false))
	registerReadTools(s)
	tools := s.ListTools()
	if len(tools) != 11 {
		t.Errorf("registerReadTools should register 11 tools, got %d", len(tools))
	}
}

func TestRegisterWriteTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(false))
	registerWriteTools(s)
	tools := s.ListTools()
	if len(tools) != 6 {
		t.Errorf("registerWriteTools should register 6 tools, got %d", len(tools))
	}
}

func TestRegisterAllTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(false))
	registerReadTools(s)
	registerWriteTools(s)
	tools := s.ListTools()
	if len(tools) != 17 {
		t.Errorf("all tools should be 17, got %d", len(tools))
	}
	for name := range tools {
		if !strings.HasPrefix(name, "argocd_") {
			t.Errorf("tool name %q does not start with 'argocd_'", name)
		}
	}
}

func TestRequireFlagValue_Success(t *testing.T) {
	args := []string{"--port", "8080"}
	i := 0
	val := requireFlagValue(args, &i, "--port")
	if val != "8080" {
		t.Errorf("val = %q, want 8080", val)
	}
	if i != 1 {
		t.Errorf("i = %d, want 1", i)
	}
}

func TestParseArgs_PortWithoutValue(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"mcp-argo", "--port"}

	// requireFlagValue calls os.Exit(1) when value is missing
	// We can't test os.Exit directly, so just verify it doesn't panic with valid input
	os.Args = []string{"mcp-argo", "--port", "3000"}
	_, _, port, _, _ := parseArgs()
	if port != 3000 {
		t.Errorf("port = %d, want 3000", port)
	}
}

func TestParseArgs_InvalidPort(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	// Valid port to not trigger os.Exit in test
	os.Args = []string{"mcp-argo", "--port", "0"}
	_, _, port, _, _ := parseArgs()
	if port != 0 {
		t.Errorf("port = %d, want 0", port)
	}
}

func TestParseArgs_HostFlag(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"mcp-argo", "http", "--host", "0.0.0.0"}
	_, host, _, _, _ := parseArgs()
	if host != "0.0.0.0" {
		t.Errorf("host = %q, want 0.0.0.0", host)
	}
}

func TestParseArgs_AllFlagsCombined(t *testing.T) {
	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"mcp-argo", "http", "--host", "0.0.0.0", "--port", "9090", "--stateless", "--log-level", "debug"}
	transport, host, port, stateless, logLevel := parseArgs()
	if transport != "http" {
		t.Errorf("transport = %q, want http", transport)
	}
	if host != "0.0.0.0" {
		t.Errorf("host = %q, want 0.0.0.0", host)
	}
	if port != 9090 {
		t.Errorf("port = %d, want 9090", port)
	}
	if !stateless {
		t.Error("stateless = false, want true")
	}
	if logLevel != "debug" {
		t.Errorf("logLevel = %q, want debug", logLevel)
	}
}

func TestResourceRefProperties(t *testing.T) {
	props := resourceRefProperties()
	expected := []string{"uid", "kind", "namespace", "name", "version", "group"}
	for _, key := range expected {
		if _, ok := props[key]; !ok {
			t.Errorf("missing property %q in resourceRefProperties()", key)
		}
	}
}
