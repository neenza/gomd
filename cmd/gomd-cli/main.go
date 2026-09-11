package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"gomd/services"
)

const version = "1.0.0"

func printUsage() {
	fmt.Println(`gomd-convert — Fast, standalone Markdown to PDF / HTML converter
Built with pure Go. Requires ZERO external dependencies, WebKit, GTK, or Cgo.

Usage:
  gomd-convert [flags] <file.md>...
  gomd-convert [flags] -o <output> <file.md>
  cat document.md | gomd-convert [flags] -o <output>

Arguments:
  <file.md>...                 One or more markdown files to convert (use '-' for stdin)

Flags:
  -o, --output <file>          Output file path (default: stdout for stdin, or <input>.<ext>)
  -t, --to, --format <format>  Target format: "pdf" or "html" (inferred from -o extension if omitted)
      --theme <name>           Theme for syntax highlighting / HTML (dark, light, nord, solarized, oled)
      --title <string>         Document title (default: first heading or filename)
      --page-size <size>       PDF page size: A4, Letter, Legal, A3, A5 (default: A4)
      --orientation <orient>   PDF orientation: portrait, landscape (default: portrait)
      --stats                  Display document statistics (lines, words, chars, reading time)
  -s, --serve                  Start live-reloading browser preview server (pure Go, no GUI)
  -p, --port <port>            Port for preview server (default: 8080)
      --open                   Automatically open browser in serve mode
  -v, --version                Print version information
  -h, --help                   Show this help message

Examples:
  # Convert Markdown to PDF
  gomd-convert README.md -o README.pdf
  gomd-convert README.md --to pdf

  # Convert Markdown to standalone HTML (with embedded CSS & Chroma syntax highlighting)
  gomd-convert README.md -o README.html
  gomd-convert README.md --theme light

  # Stream through stdin and stdout
  cat doc.md | gomd-convert --to pdf > doc.pdf
  cat doc.md | gomd-convert --to html > doc.html

  # Batch convert multiple markdown files to PDF
  gomd-convert --to pdf ch1.md ch2.md ch3.md

  # Inspect document statistics
  gomd-convert --stats README.md

  # Headless browser preview with live sync (zero WebKit needed)
  gomd-convert --serve README.md --port 3000 --open`)
}

type options struct {
	outputFiles []string
	outputPath  string
	toFormat    string
	theme       string
	title       string
	pageSize    string
	orientation string
	statsOnly   bool
	serveMode   bool
	autoOpen    bool
	port        int
	inputFiles  []string
	showHelp    bool
	showVersion bool
}

func parseArgs(args []string) (*options, error) {
	opts := &options{
		theme:       "dark",
		pageSize:    "A4",
		orientation: "portrait",
		port:        8080,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			opts.showHelp = true
			return opts, nil
		case arg == "-v" || arg == "--version":
			opts.showVersion = true
			return opts, nil
		case arg == "--stats":
			opts.statsOnly = true
		case arg == "-s" || arg == "--serve" || arg == "serve" || arg == "-v" || arg == "--view":
			opts.serveMode = true
		case arg == "--open":
			opts.autoOpen = true
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				i++
				opts.outputPath = args[i]
			} else {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
		case strings.HasPrefix(arg, "-o="):
			opts.outputPath = strings.TrimPrefix(arg, "-o=")
		case strings.HasPrefix(arg, "--output="):
			opts.outputPath = strings.TrimPrefix(arg, "--output=")
		case arg == "-t" || arg == "--to" || arg == "--format":
			if i+1 < len(args) {
				i++
				opts.toFormat = strings.ToLower(args[i])
			} else {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
		case strings.HasPrefix(arg, "--to="):
			opts.toFormat = strings.ToLower(strings.TrimPrefix(arg, "--to="))
		case strings.HasPrefix(arg, "--format="):
			opts.toFormat = strings.ToLower(strings.TrimPrefix(arg, "--format="))
		case strings.HasPrefix(arg, "-t="):
			opts.toFormat = strings.ToLower(strings.TrimPrefix(arg, "-t="))
		case arg == "--theme":
			if i+1 < len(args) {
				i++
				opts.theme = strings.ToLower(args[i])
			} else {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
		case strings.HasPrefix(arg, "--theme="):
			opts.theme = strings.ToLower(strings.TrimPrefix(arg, "--theme="))
		case arg == "--title":
			if i+1 < len(args) {
				i++
				opts.title = args[i]
			} else {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
		case strings.HasPrefix(arg, "--title="):
			opts.title = strings.TrimPrefix(arg, "--title=")
		case arg == "--page-size":
			if i+1 < len(args) {
				i++
				opts.pageSize = args[i]
			} else {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
		case strings.HasPrefix(arg, "--page-size="):
			opts.pageSize = strings.TrimPrefix(arg, "--page-size=")
		case arg == "--orientation":
			if i+1 < len(args) {
				i++
				opts.orientation = strings.ToLower(args[i])
			} else {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
		case strings.HasPrefix(arg, "--orientation="):
			opts.orientation = strings.ToLower(strings.TrimPrefix(arg, "--orientation="))
		case arg == "-p" || arg == "--port":
			if i+1 < len(args) {
				i++
				p, err := strconv.Atoi(args[i])
				if err == nil && p > 0 {
					opts.port = p
				}
			}
		case strings.HasPrefix(arg, "--port="):
			p, err := strconv.Atoi(strings.TrimPrefix(arg, "--port="))
			if err == nil && p > 0 {
				opts.port = p
			}
		case strings.HasPrefix(arg, "-p="):
			p, err := strconv.Atoi(strings.TrimPrefix(arg, "-p="))
			if err == nil && p > 0 {
				opts.port = p
			}
		case strings.HasPrefix(arg, "-"):
			return nil, fmt.Errorf("unknown flag: %s", arg)
		default:
			opts.inputFiles = append(opts.inputFiles, arg)
		}
	}

	return opts, nil
}

func isPipeInput() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func extractTitle(content string, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return fallback
}

func inferFormat(outputPath string, defaultFormat string) string {
	if defaultFormat != "" {
		return defaultFormat
	}
	if outputPath != "" && outputPath != "-" {
		ext := strings.ToLower(filepath.Ext(outputPath))
		if ext == ".pdf" {
			return "pdf"
		}
		if ext == ".html" || ext == ".htm" {
			return "html"
		}
	}
	return "html"
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\nRun 'gomd-convert --help' for usage.\n", err)
		os.Exit(1)
	}

	if opts.showHelp {
		printUsage()
		return
	}

	if opts.showVersion {
		fmt.Printf("gomd-convert version %s (pure Go, zero WebKit dependencies)\n", version)
		return
	}

	mdService := services.NewMarkdownService()

	// 1. Headless Browser Preview Mode
	if opts.serveMode {
		runServe(opts, mdService)
		return
	}

	// 2. STDIN Pipeline Mode
	if len(opts.inputFiles) == 0 || (len(opts.inputFiles) == 1 && opts.inputFiles[0] == "-") {
		if !isPipeInput() && len(opts.inputFiles) == 0 {
			printUsage()
			return
		}

		inputBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}

		content := string(inputBytes)
		if opts.statsOnly {
			printStats("STDIN", mdService.GetStats(content))
			return
		}

		targetFormat := inferFormat(opts.outputPath, opts.toFormat)
		title := opts.title
		if title == "" {
			title = extractTitle(content, "Document")
		}

		if err := convertAndWrite(mdService, content, opts.outputPath, targetFormat, title, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 3. Document Statistics Only Mode
	if opts.statsOnly {
		for _, f := range opts.inputFiles {
			data, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", f, err)
				continue
			}
			printStats(f, mdService.GetStats(string(data)))
		}
		return
	}

	// 4. Single File with explicit -o
	if len(opts.inputFiles) == 1 && opts.outputPath != "" && opts.outputPath != "-" {
		inputFile := opts.inputFiles[0]
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputFile, err)
			os.Exit(1)
		}

		content := string(data)
		targetFormat := inferFormat(opts.outputPath, opts.toFormat)
		title := opts.title
		if title == "" {
			title = extractTitle(content, strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)))
		}

		baseDir := filepath.Dir(inputFile)
		opts.outputPath = filepath.Clean(opts.outputPath)
		if err := convertToFile(mdService, content, opts.outputPath, targetFormat, title, baseDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Conversion failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✔ Successfully converted %s -> %s\n", inputFile, opts.outputPath)
		return
	}

	// 5. Batch Conversion or Single File default output
	targetFormat := opts.toFormat
	if targetFormat == "" {
		targetFormat = "html"
	}

	for _, inputFile := range opts.inputFiles {
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", inputFile, err)
			os.Exit(1)
		}

		content := string(data)
		title := opts.title
		if title == "" {
			title = extractTitle(content, strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile)))
		}

		var outPath string
		if opts.outputPath == "-" {
			outPath = "-"
		} else {
			ext := ".html"
			if targetFormat == "pdf" {
				ext = ".pdf"
			}
			outPath = strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ext
		}

		baseDir := filepath.Dir(inputFile)
		if outPath == "-" {
			if err := convertAndWrite(mdService, content, "-", targetFormat, title, opts); err != nil {
				fmt.Fprintf(os.Stderr, "Conversion failed for %s: %v\n", inputFile, err)
				os.Exit(1)
			}
		} else {
			if err := convertToFile(mdService, content, outPath, targetFormat, title, baseDir, opts); err != nil {
				fmt.Fprintf(os.Stderr, "Conversion failed for %s: %v\n", inputFile, err)
				os.Exit(1)
			}
			fmt.Printf("✔ Successfully converted %s -> %s\n", inputFile, outPath)
		}
	}
}

func convertToFile(s *services.MarkdownService, content string, outPath string, format string, title string, baseDir string, opts *options) error {
	switch format {
	case "pdf":
		pdfOpts := services.PDFOptions{
			Title:       title,
			Orientation: opts.orientation,
			PageSize:    opts.pageSize,
			Theme:       opts.theme,
			BaseDir:     baseDir,
		}
		return s.ExportPDF(content, outPath, pdfOpts)
	case "html", "htm":
		return s.ExportHTMLFile(content, outPath, title, opts.theme)
	default:
		return fmt.Errorf("unsupported format %q (supported formats: pdf, html)", format)
	}
}

func convertAndWrite(s *services.MarkdownService, content string, outPath string, format string, title string, opts *options) error {
	if outPath != "" && outPath != "-" {
		return convertToFile(s, content, outPath, format, title, ".", opts)
	}

	// Write to os.Stdout
	switch format {
	case "pdf":
		pdfOpts := services.PDFOptions{
			Title:       title,
			Orientation: opts.orientation,
			PageSize:    opts.pageSize,
			Theme:       opts.theme,
		}
		pdfBytes, err := s.RenderPDF(content, pdfOpts)
		if err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, bytes.NewReader(pdfBytes))
		return err
	case "html", "htm":
		htmlDoc, err := s.ExportHTML(title, content, opts.theme)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, htmlDoc)
		return err
	default:
		return fmt.Errorf("unsupported format %q (supported formats: pdf, html)", format)
	}
}

func printStats(filename string, stats services.TextStats) {
	fmt.Printf("\n  Document Statistics: %s\n", filename)
	fmt.Println("  ──────────────────────────────────────────")
	fmt.Printf("    Lines:        %d\n", stats.Lines)
	fmt.Printf("    Words:        %d\n", stats.Words)
	fmt.Printf("    Characters:   %d\n", stats.Characters)
	fmt.Printf("    Reading Time: %s\n", stats.ReadingTime)
	fmt.Println("  ──────────────────────────────────────────\n")
}

func runServe(opts *options, mdService *services.MarkdownService) {
	var targetFile string
	if len(opts.inputFiles) > 0 {
		targetFile = opts.inputFiles[0]
	} else {
		for _, name := range []string{"README.md", "readme.md", "index.md", "doc.md"} {
			if _, err := os.Stat(name); err == nil {
				targetFile = name
				break
			}
		}
		if targetFile == "" {
			fmt.Println("Error: No markdown file specified to preview.")
			fmt.Println("Usage: gomd-convert --serve <file.md> [--port 8080] [--open]")
			os.Exit(1)
		}
	}

	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		fmt.Printf("Error: Invalid file path '%s': %v\n", targetFile, err)
		os.Exit(1)
	}

	serverService := services.NewServerService(mdService)
	serverURL, err := serverService.Start(absPath, opts.port, opts.autoOpen)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("  ╭────────────────────────────────────────────────────────╮")
	fmt.Println("  │  gomd-convert — Browser Live Preview                   │")
	fmt.Printf("  │  Document: %-44s│\n", truncateString(filepath.Base(absPath), 44))
	fmt.Printf("  │  URL:      %-44s│\n", serverURL)
	fmt.Println("  │  LiveSync: Active (auto-refreshes on file save)        │")
	fmt.Println("  │  WebKit:   None (Pure Go server, uses your browser)    │")
	fmt.Println("  │  Press Ctrl+C to stop                                  │")
	fmt.Println("  ╰────────────────────────────────────────────────────────╯")
	fmt.Println()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping server...")
	_ = serverService.Stop()
	fmt.Println("Server stopped.")
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
