package services

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServerService(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gomd-server-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "readme.md")
	content := "# Heading Server\n\nTesting web server preview mode."
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ms := NewMarkdownService()
	srv := NewServerService(ms)

	// Start on random available port (0)
	url, err := srv.Start(filePath, 0, false)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Stop()

	if !srv.IsRunning() {
		t.Fatalf("Server should be running")
	}

	time.Sleep(50 * time.Millisecond)

	// Test GET /
	resp, err := http.Get(url + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if !strings.Contains(string(body), "Heading Server") || !strings.Contains(string(body), "Live Sync") {
		t.Errorf("Expected page to contain heading and Live Sync, got:\n%s", string(body))
	}

	// Test GET /content
	respContent, err := http.Get(url + "/content?theme=dark")
	if err != nil {
		t.Fatalf("GET /content failed: %v", err)
	}
	defer respContent.Body.Close()

	bodyContent, err := io.ReadAll(respContent.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if !strings.Contains(string(bodyContent), "Heading Server") {
		t.Errorf("Expected content to contain heading, got:\n%s", string(bodyContent))
	}
}
