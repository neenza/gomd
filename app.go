package main

import (
	"context"
	"fmt"
	"path/filepath"

	"gomd/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	fileService     *services.FileService
	markdownService *services.MarkdownService
	initialFilePath string
}

// NewApp creates a new App application struct
func NewApp(initialFilePath string) *App {
	return &App{
		fileService:     services.NewFileService(),
		markdownService: services.NewMarkdownService(),
		initialFilePath: initialFilePath,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// OpenFile opens a specific file from disk
func (a *App) OpenFile(filePath string) (*services.FileInfo, error) {
	return a.fileService.ReadFile(filePath)
}

// SaveFile saves content to a specific file path
func (a *App) SaveFile(filePath string, content string) (*services.FileInfo, error) {
	return a.fileService.SaveFile(filePath, content)
}

// OpenFileDialog opens the OS file picker dialog
func (a *App) OpenFileDialog() (*services.FileInfo, error) {
	return a.fileService.OpenFileDialog(a.ctx)
}

// SaveFileDialog opens the OS save file dialog
func (a *App) SaveFileDialog(defaultName string, content string) (*services.FileInfo, error) {
	return a.fileService.SaveFileDialog(a.ctx, defaultName, content)
}

// RenderMarkdown parses markdown text to HTML with the selected theme
func (a *App) RenderMarkdown(source string, theme string) (string, error) {
	return a.markdownService.Render(source, theme)
}

// ExportHTMLDialog prompts user for destination and writes standalone HTML
func (a *App) ExportHTMLDialog(defaultName string, source string, theme string) (string, error) {
	htmlDoc, err := a.markdownService.ExportHTML(defaultName, source, theme)
	if err != nil {
		return "", err
	}
	return a.fileService.ExportHTMLDialog(a.ctx, defaultName, htmlDoc)
}

// GetStats returns document line, word, character metrics and reading time
func (a *App) GetStats(source string) services.TextStats {
	return a.markdownService.GetStats(source)
}

// GetRecentFiles returns a list of existing recent files
func (a *App) GetRecentFiles() []string {
	return a.fileService.GetRecentFiles()
}

// ClearRecentFiles clears recent files history
func (a *App) ClearRecentFiles() {
	a.fileService.ClearRecentFiles()
}

// GetInitialFile returns FileInfo if a file was provided as CLI argument
func (a *App) GetInitialFile() (*services.FileInfo, error) {
	if a.initialFilePath == "" {
		return nil, nil
	}
	return a.fileService.ReadFile(a.initialFilePath)
}

// SetWindowTitle updates the native window title
func (a *App) SetWindowTitle(title string) {
	if a.ctx != nil {
		if title == "" {
			title = "gomd"
		} else {
			title = fmt.Sprintf("%s - gomd", filepath.Base(title))
		}
		runtime.WindowSetTitle(a.ctx, title)
	}
}
