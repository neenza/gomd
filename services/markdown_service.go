package services

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/microcosm-cc/bluemonday"
	pdf "github.com/stephenafamo/goldmark-pdf"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	classRegex = regexp.MustCompile(`^(chroma|language-[a-zA-Z0-9_\-]+|task-list-item|task-list-item-checkbox|heading-anchor|[a-zA-Z0-9_\-\s]+)$`)
	idRegex    = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

// TextStats holds metrics for document content
type TextStats struct {
	Lines       int    `json:"lines"`
	Words       int    `json:"words"`
	Characters  int    `json:"characters"`
	ReadingTime string `json:"readingTime"`
}

// MarkdownService handles markdown parsing and HTML generation
type MarkdownService struct {
	sanitizer *bluemonday.Policy
}

// NewMarkdownService creates a new MarkdownService instance
func NewMarkdownService() *MarkdownService {
	// bluemonday UGCPolicy with safe extensions for syntax highlighting and task lists
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(classRegex).Globally()
	policy.AllowAttrs("type", "checked", "disabled").OnElements("input")
	policy.AllowAttrs("id").Matching(idRegex).Globally()
	policy.AllowAttrs("aria-hidden").OnElements("a")
	policy.AllowAttrs("style").Matching(regexp.MustCompile(`.*`)).OnElements("span", "pre", "code")
	policy.AllowStyles("color", "background-color", "font-weight", "font-style", "text-decoration").Globally()

	return &MarkdownService{
		sanitizer: policy,
	}
}

// createGoldmarkEngine builds a Goldmark instance configured with GFM and Chroma highlighting
func (s *MarkdownService) createGoldmarkEngine(theme string) goldmark.Markdown {
	chromaTheme := "github-dark"
	if theme == "light" || theme == "solarized-light" {
		chromaTheme = "github"
	} else if theme == "monochrome" || theme == "oled" {
		chromaTheme = "monokai"
	} else if theme == "nord" {
		chromaTheme = "nord"
	} else if theme == "solarized" {
		chromaTheme = "solarized-dark"
	}

	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.DefinitionList,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle(chromaTheme),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(false),
					chromahtml.TabWidth(4),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			goldmarkhtml.WithHardWraps(),
			goldmarkhtml.WithXHTML(),
			renderer.WithNodeRenderers(),
		),
	)
}

// Render converts markdown text to sanitized HTML
func (s *MarkdownService) Render(source string, theme string) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", nil
	}

	gm := s.createGoldmarkEngine(theme)
	var buf bytes.Buffer
	if err := gm.Convert([]byte(source), &buf); err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	sanitized := s.sanitizer.Sanitize(buf.String())
	return sanitized, nil
}

// GetStats computes word, character, line count and reading time estimate
func (s *MarkdownService) GetStats(source string) TextStats {
	if len(source) == 0 {
		return TextStats{Lines: 0, Words: 0, Characters: 0, ReadingTime: "0 min"}
	}

	lines := strings.Count(source, "\n") + 1
	chars := len([]rune(source))

	words := 0
	inWord := false
	for _, r := range source {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}

	// Calculate reading time assuming average reading speed of 200 words/min
	minutes := float64(words) / 200.0
	var readingTime string
	if minutes < 0.8 {
		seconds := int(math.Ceil(minutes * 60))
		if seconds <= 0 {
			seconds = 1
		}
		readingTime = fmt.Sprintf("%d sec", seconds)
	} else {
		readingTime = fmt.Sprintf("%d min", int(math.Ceil(minutes)))
	}

	return TextStats{
		Lines:       lines,
		Words:       words,
		Characters:  chars,
		ReadingTime: readingTime,
	}
}

// ExportHTML produces a standalone HTML document with embedded CSS
func (s *MarkdownService) ExportHTML(docTitle string, source string, theme string) (string, error) {
	rendered, err := s.Render(source, theme)
	if err != nil {
		return "", err
	}

	if docTitle == "" {
		docTitle = "Document"
	}

	bgColor := "#0f141c"
	textColor := "#e6edf3"
	if theme == "light" {
		bgColor = "#ffffff"
		textColor = "#1f2328"
	}

	doc := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
  :root {
    --bg-color: %s;
    --text-color: %s;
    --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    --font-mono: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  }
  body {
    margin: 0;
    padding: 2.5rem 1.5rem;
    background-color: var(--bg-color);
    color: var(--text-color);
    font-family: var(--font-family);
    line-height: 1.65;
    word-wrap: break-word;
  }
  .markdown-body {
    max-width: 860px;
    margin: 0 auto;
  }
  h1, h2, h3, h4, h5, h6 { margin-top: 1.5em; margin-bottom: 0.5em; font-weight: 600; line-height: 1.25; }
  h1 { font-size: 2em; border-bottom: 1px solid rgba(128,128,128,0.2); padding-bottom: 0.3em; }
  h2 { font-size: 1.5em; border-bottom: 1px solid rgba(128,128,128,0.2); padding-bottom: 0.3em; }
  p, ul, ol, blockquote, table, pre { margin-top: 0; margin-bottom: 1rem; }
  a { color: #58a6ff; text-decoration: none; }
  a:hover { text-decoration: underline; }
  code { font-family: var(--font-mono); font-size: 0.88em; padding: 0.2em 0.4em; background: rgba(128,128,128,0.15); border-radius: 4px; }
  pre { padding: 1rem; overflow-x: auto; background: rgba(128,128,128,0.1); border-radius: 6px; }
  pre code { padding: 0; background: transparent; }
  blockquote { padding: 0 1rem; color: #8b949e; border-left: 4px solid #30363d; margin-left: 0; }
  table { border-collapse: collapse; width: 100%%; margin: 1rem 0; }
  table th, table td { border: 1px solid rgba(128,128,128,0.2); padding: 6px 13px; }
  table tr:nth-child(2n) { background-color: rgba(128,128,128,0.05); }
  hr { height: 2px; padding: 0; margin: 24px 0; background-color: rgba(128,128,128,0.2); border: 0; }
  img { max-width: 100%%; height: auto; border-radius: 4px; }
  ul.task-list { list-style: none; padding-left: 0; }
  .task-list-item { display: flex; align-items: center; gap: 0.5rem; }
  @media print {
    body { background: #fff !important; color: #000 !important; padding: 0; }
    .markdown-body { max-width: 100%%; }
  }
</style>
</head>
<body>
<main class="markdown-body">
%s
</main>
</body>
</html>`, html.EscapeString(docTitle), bgColor, textColor, rendered)

	return doc, nil
}

// PDFOptions configures PDF export settings
type PDFOptions struct {
	Title       string
	Orientation string // "portrait" (default) or "landscape"
	PageSize    string // "A4" (default), "Letter", "Legal", "A3", "A5"
	Theme       string // syntax highlighting theme ("github", "github-dark", "monokai", "nord", "solarized-dark")
	BaseDir     string // base directory for resolving relative image paths
}

// RenderPDF converts markdown text to PDF document bytes
func (s *MarkdownService) RenderPDF(source string, opts PDFOptions) ([]byte, error) {
	if strings.TrimSpace(source) == "" {
		return nil, fmt.Errorf("source markdown is empty")
	}

	orientation := opts.Orientation
	if orientation == "" {
		orientation = "portrait"
	}

	pageSize := opts.PageSize
	if pageSize == "" {
		pageSize = "A4"
	}

	chromaTheme := "github"
	if opts.Theme == "dark" || opts.Theme == "github-dark" {
		chromaTheme = "github-dark"
	} else if opts.Theme == "monokai" || opts.Theme == "oled" || opts.Theme == "monochrome" {
		chromaTheme = "monokai"
	} else if opts.Theme == "nord" {
		chromaTheme = "nord"
	} else if opts.Theme == "solarized" || opts.Theme == "solarized-dark" {
		chromaTheme = "solarized-dark"
	}

	title := opts.Title
	if title == "" {
		title = "Document"
	}

	ctx := context.Background()
	cfg := pdf.FpdfConfig{
		Title:       title,
		Orientation: orientation,
		PaperSize:   pageSize,
	}

	rendererOpts := []pdf.Option{
		pdf.WithFpdf(ctx, cfg),
		pdf.WithHeadingFont(pdf.FontHelvetica),
		pdf.WithBodyFont(pdf.FontHelvetica),
		pdf.WithCodeFont(pdf.FontCourier),
		pdf.WithCodeBlockTheme(styles.Get(chromaTheme)),
	}

	if opts.BaseDir != "" {
		rendererOpts = append(rendererOpts, pdf.WithImageFS(http.Dir(opts.BaseDir)))
	}

	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.DefinitionList,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRenderer(
			pdf.New(rendererOpts...),
		),
	)

	var buf bytes.Buffer
	if err := gm.Convert([]byte(source), &buf); err != nil {
		return nil, fmt.Errorf("failed to render PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportPDF converts markdown text and writes the resulting PDF to outputPath
func (s *MarkdownService) ExportPDF(source string, outputPath string, opts PDFOptions) error {
	pdfBytes, err := s.RenderPDF(source, opts)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}

	if err := os.WriteFile(outputPath, pdfBytes, 0644); err != nil {
		return fmt.Errorf("failed to write PDF to %q: %w", outputPath, err)
	}

	return nil
}

// ExportHTMLFile converts markdown text and writes standalone HTML with CSS to outputPath
func (s *MarkdownService) ExportHTMLFile(source string, outputPath string, docTitle string, theme string) error {
	htmlDoc, err := s.ExportHTML(docTitle, source, theme)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}
	}

	if err := os.WriteFile(outputPath, []byte(htmlDoc), 0644); err != nil {
		return fmt.Errorf("failed to write HTML to %q: %w", outputPath, err)
	}

	return nil
}
