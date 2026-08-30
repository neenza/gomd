package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownRender(t *testing.T) {
	ms := NewMarkdownService()

	// Test basic formatting
	input := "# Hello World\n\nThis is **bold** and *italic*.\n\n- [x] Task 1\n- [ ] Task 2"
	html, err := ms.Render(input, "dark")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "<h1") || !strings.Contains(html, "Hello World") {
		t.Errorf("Expected h1 heading, got: %s", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("Expected bold text, got: %s", html)
	}
	if !strings.Contains(html, "<em>italic</em>") {
		t.Errorf("Expected italic text, got: %s", html)
	}
	if !strings.Contains(html, "type=\"checkbox\"") {
		t.Errorf("Expected checkbox input for task list, got: %s", html)
	}

	// Test code syntax highlighting
	codeBlock := "```go\npackage main\n\nfunc main() {}\n```"
	codeHTML, err := ms.Render(codeBlock, "dark")
	if err != nil {
		t.Fatalf("Render code block failed: %v", err)
	}
	if !strings.Contains(codeHTML, "chroma") && !strings.Contains(codeHTML, "func") {
		t.Errorf("Expected highlighted code, got: %s", codeHTML)
	}

	// Test GFM table
	tableInput := "| Name | Age |\n| --- | --- |\n| Alice | 30 |"
	tableHTML, err := ms.Render(tableInput, "dark")
	if err != nil {
		t.Fatalf("Render table failed: %v", err)
	}
	if !strings.Contains(tableHTML, "<table>") || !strings.Contains(tableHTML, "Alice") {
		t.Errorf("Expected HTML table, got: %s", tableHTML)
	}
}

func TestTextStats(t *testing.T) {
	ms := NewMarkdownService()
	text := "The quick brown fox jumps over the lazy dog.\nSecond line here."
	stats := ms.GetStats(text)

	if stats.Lines != 2 {
		t.Errorf("Expected 2 lines, got %d", stats.Lines)
	}
	if stats.Words != 12 {
		t.Errorf("Expected 12 words, got %d", stats.Words)
	}
	if stats.Characters != len([]rune(text)) {
		t.Errorf("Expected %d characters, got %d", len([]rune(text)), stats.Characters)
	}
}

func TestExportHTML(t *testing.T) {
	ms := NewMarkdownService()
	htmlDoc, err := ms.ExportHTML("My Notes", "# Heading\nContent", "dark")
	if err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}
	if !strings.Contains(htmlDoc, "<!DOCTYPE html>") || !strings.Contains(htmlDoc, "<title>My Notes</title>") {
		t.Errorf("Expected complete HTML document, got: %s", htmlDoc)
	}
}

func TestFileServiceAtomicSaveAndRead(t *testing.T) {
	fs := NewFileService()
	tmpDir, err := os.MkdirTemp("", "gomd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	targetPath := filepath.Join(tmpDir, "test.md")
	content := "# Test Document\n\nAutomated test content."

	savedInfo, err := fs.SaveFile(targetPath, content)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	if savedInfo.Name != "test.md" {
		t.Errorf("Expected Name test.md, got %s", savedInfo.Name)
	}
	if savedInfo.Content != content {
		t.Errorf("Expected Content match, got %s", savedInfo.Content)
	}

	readInfo, err := fs.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if readInfo.Content != content {
		t.Errorf("Expected read content %q, got %q", content, readInfo.Content)
	}
}
