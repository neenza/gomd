package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileInfo holds metadata and content of a file
type FileInfo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Size       int64  `json:"size"`
	ModTime    string `json:"modTime"`
	IsReadOnly bool   `json:"isReadOnly"`
}

// FileService handles file I/O, dialogs, and recent files
type FileService struct {
	mu          sync.RWMutex
	recentFiles []string
	configDir   string
}

// NewFileService creates a new FileService
func NewFileService() *FileService {
	homeDir, err := os.UserHomeDir()
	configDir := ""
	if err == nil {
		configDir = filepath.Join(homeDir, ".config", "gomd")
		_ = os.MkdirAll(configDir, 0755)
	}

	fs := &FileService{
		configDir:   configDir,
		recentFiles: make([]string, 0),
	}
	fs.loadRecentFiles()
	return fs
}

// ReadFile reads the full contents of a file and returns its FileInfo
func (fs *FileService) ReadFile(filePath string) (*FileInfo, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", absPath)
	}

	// Limit reading large non-text files or huge binaries
	if info.Size() > 50*1024*1024 { // 50MB safeguard
		return nil, fmt.Errorf("file too large (> 50MB)")
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check if read-only
	isReadOnly := (info.Mode().Perm() & 0200) == 0

	fileInfo := &FileInfo{
		Path:       absPath,
		Name:       filepath.Base(absPath),
		Content:    string(data),
		Size:       info.Size(),
		ModTime:    info.ModTime().Format(time.RFC3339),
		IsReadOnly: isReadOnly,
	}

	fs.AddRecentFile(absPath)
	return fileInfo, nil
}

// SaveFile atomically saves content to the specified file path
func (fs *FileService) SaveFile(filePath string, content string) (*FileInfo, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file in same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, ".gomd-tmp-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	// Clean up temp file on failure
	var writeSuccess bool
	defer func() {
		if !writeSuccess {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.WriteString(tmpFile, content); err != nil {
		return nil, fmt.Errorf("failed to write content: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomically replace target file
	if err := os.Rename(tmpName, absPath); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	writeSuccess = true

	stat, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat saved file: %w", err)
	}

	fs.AddRecentFile(absPath)

	return &FileInfo{
		Path:       absPath,
		Name:       filepath.Base(absPath),
		Content:    content,
		Size:       stat.Size(),
		ModTime:    stat.ModTime().Format(time.RFC3339),
		IsReadOnly: false,
	}, nil
}

// OpenFileDialog opens the native file picker for selecting a markdown file
func (fs *FileService) OpenFileDialog(ctx context.Context) (*FileInfo, error) {
	selectedPath, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "Open Markdown File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown Files (*.md, *.markdown, *.mdown)", Pattern: "*.md;*.markdown;*.mdown;*.mkd"},
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if selectedPath == "" {
		return nil, nil // User cancelled
	}

	return fs.ReadFile(selectedPath)
}

// SaveFileDialog opens the native save dialog to select location and save content
func (fs *FileService) SaveFileDialog(ctx context.Context, defaultName string, content string) (*FileInfo, error) {
	if defaultName == "" {
		defaultName = "Untitled.md"
	}

	selectedPath, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Save Markdown File",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown Files (*.md)", Pattern: "*.md"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if selectedPath == "" {
		return nil, nil // User cancelled
	}

	return fs.SaveFile(selectedPath, content)
}

// ExportHTMLDialog opens a save dialog for exporting HTML
func (fs *FileService) ExportHTMLDialog(ctx context.Context, defaultName string, htmlContent string) (string, error) {
	if defaultName == "" {
		defaultName = "document.html"
	}
	if !filepath.IsAbs(defaultName) && filepath.Ext(defaultName) != ".html" {
		defaultName += ".html"
	}

	selectedPath, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           "Export as HTML",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "HTML Files (*.html)", Pattern: "*.html"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if selectedPath == "" {
		return "", nil // User cancelled
	}

	if err := os.WriteFile(selectedPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write HTML export: %w", err)
	}

	return selectedPath, nil
}

// AddRecentFile adds a file path to the list of recent files
func (fs *FileService) AddRecentFile(path string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Remove duplicate if already present
	updated := make([]string, 0, 10)
	updated = append(updated, path)
	for _, p := range fs.recentFiles {
		if p != path && len(updated) < 10 {
			updated = append(updated, p)
		}
	}
	fs.recentFiles = updated
	fs.saveRecentFiles()
}

// GetRecentFiles returns the list of recent files that still exist on disk
func (fs *FileService) GetRecentFiles() []string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	result := make([]string, 0, len(fs.recentFiles))
	for _, p := range fs.recentFiles {
		if _, err := os.Stat(p); err == nil {
			result = append(result, p)
		}
	}
	return result
}

// ClearRecentFiles clears the recent files list
func (fs *FileService) ClearRecentFiles() {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.recentFiles = make([]string, 0)
	fs.saveRecentFiles()
}

func (fs *FileService) loadRecentFiles() {
	if fs.configDir == "" {
		return
	}
	recentPath := filepath.Join(fs.configDir, "recent.json")
	data, err := os.ReadFile(recentPath)
	if err != nil {
		return
	}

	var files []string
	if err := json.Unmarshal(data, &files); err == nil {
		fs.recentFiles = files
	}
}

func (fs *FileService) saveRecentFiles() {
	if fs.configDir == "" {
		return
	}
	recentPath := filepath.Join(fs.configDir, "recent.json")
	data, err := json.MarshalIndent(fs.recentFiles, "", "  ")
	if err == nil {
		_ = os.WriteFile(recentPath, data, 0644)
	}
}
