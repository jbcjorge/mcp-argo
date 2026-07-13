package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := HealthHandler("1.2.3")
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("failed to parse health response: %v", err)
	}
	if parsed["status"] != "ok" {
		t.Errorf("status = %v, want ok", parsed["status"])
	}
	if parsed["version"] != "1.2.3" {
		t.Errorf("version = %v, want 1.2.3", parsed["version"])
	}
}

func TestWaitForShutdown_GracefulStop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	srv := &http.Server{Addr: "127.0.0.1:0", Handler: mux}

	// Start the server in background
	go func() {
		_ = srv.ListenAndServe()
	}()

	// Give server a moment to start
	time.Sleep(10 * time.Millisecond)

	// WaitForShutdown blocks until signal, so send one in background
	done := make(chan struct{})
	go func() {
		WaitForShutdown(srv)
		close(done)
	}()

	// Send SIGTERM to self
	time.Sleep(10 * time.Millisecond)
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	select {
	case <-done:
		// Success - shutdown completed
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForShutdown did not return after SIGTERM")
	}
}
