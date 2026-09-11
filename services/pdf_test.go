package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownServicePDFAndHTMLExport(t *testing.T) {
	mdService := NewMarkdownService()

	md := `# Test Document

This is a test paragraph with **bold**, *italic*, and ` + "`inline code`" + `.

## Code Section
` + "```go\nfunc add(a, b int) int {\n    return a + b\n}\n```\n" + `

## Table
| Option | Description |
| :--- | :--- |
| HTML | Standalone file with CSS |
| PDF | Zero WebKit dependencies |
`

	// 1. Test RenderPDF
	pdfBytes, err := mdService.RenderPDF(md, PDFOptions{
		Title:       "Test Doc",
		Orientation: "portrait",
		PageSize:    "A4",
		Theme:       "github",
	})
	if err != nil {
		t.Fatalf("RenderPDF failed: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("RenderPDF returned 0 bytes")
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("PDF bytes missing standard header: %s", string(pdfBytes[:15]))
	}

	// 2. Test ExportPDF
	tmpDir := t.TempDir()
	pdfPath := filepath.Join(tmpDir, "sub", "test.pdf")
	if err := mdService.ExportPDF(md, pdfPath, PDFOptions{Title: "Test"}); err != nil {
		t.Fatalf("ExportPDF failed: %v", err)
	}
	info, err := os.Stat(pdfPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("ExportPDF output file invalid: %v", err)
	}

	// 3. Test ExportHTMLFile
	htmlPath := filepath.Join(tmpDir, "sub", "test.html")
	if err := mdService.ExportHTMLFile(md, htmlPath, "Test HTML", "dark"); err != nil {
		t.Fatalf("ExportHTMLFile failed: %v", err)
	}
	htmlData, err := os.ReadFile(htmlPath)
	if err != nil || len(htmlData) == 0 {
		t.Fatalf("ExportHTMLFile output invalid: %v", err)
	}
	if !bytes.Contains(htmlData, []byte("<!DOCTYPE html>")) || !bytes.Contains(htmlData, []byte("Test Document")) {
		t.Fatalf("ExportHTMLFile content mismatch: %s", string(htmlData[:200]))
	}
}

func TestMarkdownServicePDFWithQuotesAndStrings(t *testing.T) {
	mdService := NewMarkdownService()

	// Markdown with quotes, dashes, ellipses, and raw strings that previously triggered:
	// "panic: interface conversion: ast.Node is *ast.String, not *ast.Text"
	md := `# Quotes & Strings Regression Test

This is a test with "double quotes", 'single quotes', dashes -- like this --- and ellipses...

Quotes in lists:
- "Option A" - works
- "Option B" - works

Quotes in table:
| Key | Value |
| --- | --- |
| "name" | "test" |
| 'foo' | 'bar' |

` + "```bash\necho \"hello from bash\"\n```\n"

	pdfBytes, err := mdService.RenderPDF(md, PDFOptions{
		Title: "Quotes Test",
	})
	if err != nil {
		t.Fatalf("RenderPDF failed on quotes/strings: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("RenderPDF returned 0 bytes")
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("Generated invalid PDF header")
	}
}

