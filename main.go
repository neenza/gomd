package main

import (
	"embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"gomd/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	args := os.Args[1:]

	var serveMode bool
	var autoOpen bool
	var port int = 0
	var initialFilePaths []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--serve" || arg == "-s" || arg == "serve" || arg == "--view" || arg == "-v":
			serveMode = true
		case arg == "--open" || arg == "-o":
			autoOpen = true
		case arg == "--port" || arg == "-p":
			if i+1 < len(args) {
				i++
				p, err := strconv.Atoi(args[i])
				if err == nil && p > 0 {
					port = p
					serveMode = true
				}
			}
		case strings.HasPrefix(arg, "--port="):
			p, err := strconv.Atoi(strings.TrimPrefix(arg, "--port="))
			if err == nil && p > 0 {
				port = p
				serveMode = true
			}
		case strings.HasPrefix(arg, "-p="):
			p, err := strconv.Atoi(strings.TrimPrefix(arg, "-p="))
			if err == nil && p > 0 {
				port = p
				serveMode = true
			}
		case strings.HasPrefix(arg, "-"):
			// Other flags
		default:
			initialFilePaths = append(initialFilePaths, arg)
		}
	}

	// 1. Headless Browser Preview Mode
	if serveMode {
		runHeadlessServer(initialFilePaths, port, autoOpen)
		return
	}

	// 2. Desktop GUI Mode
	app := NewApp(initialFilePaths)

	err := wails.Run(&options.App{
		Title:     "gomd",
		Width:     1120,
		Height:    760,
		MinWidth:  640,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 20, B: 24, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func runHeadlessServer(filePaths []string, port int, autoOpen bool) {
	if port <= 0 {
		port = 8080
	}

	var targetFile string
	if len(filePaths) > 0 {
		targetFile = filePaths[0]
	} else {
		// Look for common markdown files in current directory
		for _, name := range []string{"README.md", "readme.md", "index.md", "doc.md"} {
			if _, err := os.Stat(name); err == nil {
				targetFile = name
				break
			}
		}
		if targetFile == "" {
			fmt.Println("Error: No markdown file specified to preview.")
			fmt.Println("Usage: gomd --serve <file.md> [--port 8080] [--open]")
			os.Exit(1)
		}
	}

	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		fmt.Printf("Error: Invalid file path '%s': %v\n", targetFile, err)
		os.Exit(1)
	}

	serverService := services.NewServerService(nil)
	serverURL, err := serverService.Start(absPath, port, autoOpen)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("  ╭────────────────────────────────────────────────────────╮")
	fmt.Println("  │  gomd — Browser Preview Mode                           │")
	fmt.Printf("  │  Document: %-44s│\n", truncateString(filepath.Base(absPath), 44))
	fmt.Printf("  │  URL:      %-44s│\n", serverURL)
	fmt.Println("  │  LiveSync: Active (auto-refreshes on file changes)     │")
	fmt.Println("  │  Press Ctrl+C to stop                                  │")
	fmt.Println("  ╰────────────────────────────────────────────────────────╯")
	fmt.Println()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping gomd server...")
	_ = serverService.Stop()
	fmt.Println("Server stopped.")
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
