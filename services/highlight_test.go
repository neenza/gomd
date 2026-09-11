package services

import (
	"fmt"
	"strings"
	"testing"
)

func TestCodeHighlightOutput(t *testing.T) {
	ms := NewMarkdownService()
	codeBlock := "```go\npackage main\n\nfunc main() {}\n```"
	html, err := ms.Render(codeBlock, "dark")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	fmt.Printf("Rendered HTML:\n%s\n", html)
	if !strings.Contains(html, "color:") || !strings.Contains(html, "style=") {
		t.Errorf("Expected syntax coloring with inline styles, got:\n%s", html)
	}
}

func TestMermaidBlockOutput(t *testing.T) {
	ms := NewMarkdownService()
	mermaidBlock := "```mermaid\ngraph TD\n    A[Client] --> B[Server]\n```"
	html, err := ms.Render(mermaidBlock, "dark")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	fmt.Printf("Rendered Mermaid HTML:\n%s\n", html)
}
