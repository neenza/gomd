package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gomd-app-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sampleFile := filepath.Join(tmpDir, "sample.md")
	content := "# My Integration Test\n\n- [ ] Task A\n- [x] Task B\n\n```go\nfunc hello() {}\n```"
	if err := os.WriteFile(sampleFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	app := NewApp(sampleFile)

	// Test Initial File
	initial, err := app.GetInitialFile()
	if err != nil {
		t.Fatalf("GetInitialFile failed: %v", err)
	}
	if initial == nil || initial.Name != "sample.md" {
		t.Fatalf("Expected sample.md initial file, got: %v", initial)
	}
	if initial.Content != content {
		t.Fatalf("Content mismatch in initial file")
	}

	// Test Markdown Rendering
	html, err := app.RenderMarkdown(content, "dark")
	if err != nil {
		t.Fatalf("RenderMarkdown failed: %v", err)
	}
	if html == "" {
		t.Fatalf("Expected non-empty HTML")
	}

	// Test Stats
	stats := app.GetStats(content)
	if stats.Lines < 5 {
		t.Errorf("Expected at least 5 lines, got %d", stats.Lines)
	}

	// Test Save File
	newContent := "# Updated Content"
	saved, err := app.SaveFile(sampleFile, newContent)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	if saved.Content != newContent {
		t.Errorf("Expected updated content, got %s", saved.Content)
	}

	// Test Recent Files
	recent := app.GetRecentFiles()
	if len(recent) == 0 {
		t.Errorf("Expected recent files to contain saved file")
	}
}
