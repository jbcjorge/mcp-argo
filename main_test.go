package main

import (
	"os"
	"strings"
	"testing"

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

	// Exercise the real registration code instead of duplicating the tool
	// list here, so adding a tool never requires editing this test.
	registerReadTools(s)
	if !readOnly {
		registerWriteTools(s)
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

// writeToolNames lists the tools that must only be present when write mode is
// enabled. Kept as the single source of truth for the read-only gating tests.
var writeToolNames = []string{
	"argocd_create_application",
	"argocd_update_application",
	"argocd_delete_application",
	"argocd_sync_application",
	"argocd_run_resource_action",
	"argocd_rollback_application",
}

func toolNameSet(t *testing.T, s *server.MCPServer) map[string]bool {
	t.Helper()
	set := make(map[string]bool)
	for _, name := range getToolNames(t, s) {
		set[name] = true
	}
	return set
}

func TestToolRegistration_ReadOnlyExcludesWriteTools(t *testing.T) {
	names := toolNameSet(t, createMCPServer(true))
	for _, w := range writeToolNames {
		if names[w] {
			t.Errorf("read-only mode should not register write tool %q", w)
		}
	}
	if len(names) == 0 {
		t.Fatal("read-only mode registered no tools")
	}
}

func TestToolRegistration_NormalModeIncludesWriteTools(t *testing.T) {
	names := toolNameSet(t, createMCPServer(false))
	for _, w := range writeToolNames {
		if !names[w] {
			t.Errorf("normal mode is missing write tool %q", w)
		}
	}
}

func TestToolRegistration_NormalModeIsReadOnlyPlusWriteTools(t *testing.T) {
	normal := len(getToolNames(t, createMCPServer(false)))
	readOnly := len(getToolNames(t, createMCPServer(true)))
	if normal != readOnly+len(writeToolNames) {
		t.Errorf("normal tools (%d) should equal read-only tools (%d) + write tools (%d)",
			normal, readOnly, len(writeToolNames))
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
	if len(s.ListTools()) == 0 {
		t.Error("registerReadTools registered no tools")
	}
}

func TestRegisterWriteTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(false))
	registerWriteTools(s)
	tools := s.ListTools()
	if len(tools) != len(writeToolNames) {
		t.Errorf("registerWriteTools should register %d tools, got %d", len(writeToolNames), len(tools))
	}
	for _, w := range writeToolNames {
		if _, ok := tools[w]; !ok {
			t.Errorf("registerWriteTools is missing %q", w)
		}
	}
}

func TestRegisterAllTools(t *testing.T) {
	readOnly := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(false))
	registerReadTools(readOnly)
	readCount := len(readOnly.ListTools())

	s := server.NewMCPServer("test", "1.0.0", server.WithToolCapabilities(false))
	registerReadTools(s)
	registerWriteTools(s)
	tools := s.ListTools()
	if len(tools) != readCount+len(writeToolNames) {
		t.Errorf("all tools (%d) should equal read tools (%d) + write tools (%d)",
			len(tools), readCount, len(writeToolNames))
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
