package services

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/browser"
)

// ServerService manages the headless browser preview HTTP server
type ServerService struct {
	mdService   *MarkdownService
	server      *http.Server
	filePath    string
	port        int
	mu          sync.RWMutex
	clients     map[chan bool]bool
	stopChan    chan struct{}
	lastModTime time.Time
	running     bool
}

// NewServerService creates a new ServerService
func NewServerService(mdService *MarkdownService) *ServerService {
	if mdService == nil {
		mdService = NewMarkdownService()
	}
	return &ServerService{
		mdService: mdService,
		clients:   make(map[chan bool]bool),
	}
}

// Start launches the HTTP preview server on the specified port
func (s *ServerService) Start(targetFilePath string, port int, autoOpen bool) (string, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return "", fmt.Errorf("server is already running")
	}

	absPath, err := filepath.Abs(targetFilePath)
	if err != nil {
		s.mu.Unlock()
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	stat, err := os.Stat(absPath)
	if err != nil {
		s.mu.Unlock()
		return "", fmt.Errorf("could not access file: %w", err)
	}
	if stat.IsDir() {
		s.mu.Unlock()
		return "", fmt.Errorf("specified path is a directory")
	}

	s.filePath = absPath
	s.port = port
	s.lastModTime = stat.ModTime()
	s.stopChan = make(chan struct{})
	s.running = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/content", s.handleContent)
	mux.HandleFunc("/events", s.handleEvents)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return "", fmt.Errorf("failed to bind to port %d: %w", port, err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	s.port = actualPort
	s.server = &http.Server{
		Handler: mux,
	}

	// Start HTTP server in background
	go func() {
		_ = s.server.Serve(listener)
	}()

	// Start file modification watcher
	go s.watchFile()

	serverURL := fmt.Sprintf("http://localhost:%d", actualPort)

	if autoOpen {
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = browser.OpenURL(serverURL)
		}()
	}

	return serverURL, nil
}

// Stop shuts down the preview server
func (s *ServerService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.stopChan)
	s.running = false

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// IsRunning returns whether the server is active
func (s *ServerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetURL returns the active server URL
func (s *ServerService) GetURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.running {
		return ""
	}
	return fmt.Sprintf("http://localhost:%d", s.port)
}

func (s *ServerService) watchFile() {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.RLock()
			path := s.filePath
			lastMod := s.lastModTime
			s.mu.RUnlock()

			stat, err := os.Stat(path)
			if err == nil && stat.ModTime().After(lastMod) {
				s.mu.Lock()
				s.lastModTime = stat.ModTime()
				s.mu.Unlock()

				// Notify all connected SSE clients
				s.notifyClients()
			}
		}
	}
}

func (s *ServerService) notifyClients() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for clientChan := range s.clients {
		select {
		case clientChan <- true:
		default:
		}
	}
}

func (s *ServerService) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	s.mu.RLock()
	filePath := s.filePath
	s.mu.RUnlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	docTitle := filepath.Base(filePath)
	rendered, err := s.mdService.Render(string(data), "dark")
	if err != nil {
		rendered = fmt.Sprintf("<p class='error'>Error rendering markdown: %s</p>", html.EscapeString(err.Error()))
	}

	pageHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s - gomd Web Preview</title>
<style>
  :root, html[data-theme="dark"] {
    --bg-app: #0d1117;
    --bg-surface: #161b22;
    --border-color: #30363d;
    --text-main: #e6edf3;
    --text-muted: #8b949e;
    --accent: #58a6ff;
    --code-bg: #161b22;
    --success: #3fb950;
  }
  html[data-theme="light"] {
    --bg-app: #ffffff;
    --bg-surface: #f6f8fa;
    --border-color: #d0d7de;
    --text-main: #1f2328;
    --text-muted: #656d76;
    --accent: #0969da;
    --code-bg: #f6f8fa;
    --success: #1a7f37;
  }
  html[data-theme="oled"] {
    --bg-app: #000000;
    --bg-surface: #0a0a0a;
    --border-color: #222222;
    --text-main: #f5f5f5;
    --text-muted: #888888;
    --accent: #00a8ff;
    --code-bg: #0d0d0d;
    --success: #2ed573;
  }
  html[data-theme="nord"] {
    --bg-app: #2e3440;
    --bg-surface: #3b4252;
    --border-color: #4c566a;
    --text-main: #eceff4;
    --text-muted: #d8dee9;
    --accent: #88c0d0;
    --code-bg: #242933;
    --success: #a3be8c;
  }
  html[data-theme="solarized"] {
    --bg-app: #002b36;
    --bg-surface: #073642;
    --border-color: #073642;
    --text-main: #93a1a1;
    --text-muted: #657b83;
    --accent: #268bd2;
    --code-bg: #073642;
    --success: #859900;
  }

  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    background-color: var(--bg-app);
    color: var(--text-main);
    line-height: 1.65;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .nav-header {
    height: 44px;
    background-color: var(--bg-surface);
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 1.5rem;
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .nav-left, .nav-right { display: flex; align-items: center; gap: 10px; }
  .brand { font-weight: 700; font-size: 13px; color: var(--accent); display: flex; align-items: center; gap: 6px; }
  .doc-title { font-size: 13px; font-weight: 600; color: var(--text-main); }
  .live-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 11px;
    font-weight: 500;
    color: var(--success);
    background: rgba(63, 185, 80, 0.1);
    padding: 2px 8px;
    border-radius: 12px;
  }
  .live-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%%;
    background-color: var(--success);
    animation: pulse 2s infinite;
  }

  .theme-select {
    background-color: var(--bg-app);
    color: var(--text-main);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 3px 8px;
    font-size: 12px;
    outline: none;
    cursor: pointer;
  }

  .btn {
    background-color: var(--bg-app);
    color: var(--text-main);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 3px 10px;
    font-size: 12px;
    cursor: pointer;
  }
  .btn:hover { border-color: var(--accent); }

  .content-wrapper {
    max-width: 860px;
    width: 100%%;
    margin: 0 auto;
    padding: 2.5rem 1.5rem;
    flex: 1;
  }

  .markdown-body h1, .markdown-body h2, .markdown-body h3,
  .markdown-body h4, .markdown-body h5, .markdown-body h6 {
    margin-top: 1.5em; margin-bottom: 0.5em; font-weight: 600; line-height: 1.3;
  }
  .markdown-body h1 { font-size: 2em; border-bottom: 1px solid var(--border-color); padding-bottom: 0.3em; }
  .markdown-body h2 { font-size: 1.5em; border-bottom: 1px solid var(--border-color); padding-bottom: 0.25em; }
  .markdown-body p, .markdown-body ul, .markdown-body ol,
  .markdown-body blockquote, .markdown-body table, .markdown-body pre {
    margin-top: 0; margin-bottom: 1rem;
  }
  .markdown-body a { color: var(--accent); text-decoration: none; }
  .markdown-body a:hover { text-decoration: underline; }
  .markdown-body code {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 0.88em;
    background: var(--code-bg);
    border: 1px solid var(--border-color);
    padding: 0.2em 0.4em;
    border-radius: 4px;
  }
  .markdown-body pre {
    padding: 1rem;
    overflow-x: auto;
    background: var(--code-bg);
    border: 1px solid var(--border-color);
    border-radius: 6px;
  }
  .markdown-body pre code { background: transparent; border: none; padding: 0; }
  .markdown-body blockquote {
    padding: 0 1rem;
    color: var(--text-muted);
    border-left: 4px solid var(--accent);
    margin-left: 0;
  }
  .markdown-body table { border-collapse: collapse; width: 100%%; margin: 1rem 0; }
  .markdown-body table th, .markdown-body table td { border: 1px solid var(--border-color); padding: 8px 12px; }
  .markdown-body table th { background: var(--bg-surface); }
  .markdown-body table tr:nth-child(2n) { background: var(--bg-surface); }
  .markdown-body hr { height: 2px; border: none; background: var(--border-color); margin: 2rem 0; }
  .markdown-body img { max-width: 100%%; height: auto; border-radius: 4px; }
  .markdown-body ul.task-list { list-style: none; padding-left: 0; }
  .markdown-body .task-list-item { display: flex; align-items: center; gap: 8px; }
  .markdown-body .task-list-item input[type="checkbox"] { accent-color: var(--accent); }

  @keyframes pulse { 0%% { opacity: 1; } 50%% { opacity: 0.4; } 100%% { opacity: 1; } }

  @media print {
    .nav-header { display: none !important; }
    body { background: #fff !important; color: #000 !important; }
    .content-wrapper { max-width: 100%%; padding: 0; }
  }
</style>
</head>
<body>
  <header class="nav-header">
    <div class="nav-left">
      <span class="brand">gomd</span>
      <span class="doc-title">%s</span>
      <span class="live-badge"><span class="live-dot"></span> Live Sync</span>
    </div>
    <div class="nav-right">
      <select id="theme-select" class="theme-select">
        <option value="dark">Dark</option>
        <option value="light">Light</option>
        <option value="oled">OLED</option>
        <option value="nord">Nord</option>
        <option value="solarized">Solarized</option>
      </select>
      <button class="btn" onclick="window.print()">Print / PDF</button>
    </div>
  </header>

  <div class="content-wrapper">
    <main class="markdown-body" id="content">
      %s
    </main>
  </div>

  <script>
    let currentTheme = localStorage.getItem('gomd_web_theme') || 'dark';
    document.documentElement.setAttribute('data-theme', currentTheme);
    const select = document.getElementById('theme-select');
    select.value = currentTheme;

    select.addEventListener('change', () => {
      currentTheme = select.value;
      localStorage.setItem('gomd_web_theme', currentTheme);
      document.documentElement.setAttribute('data-theme', currentTheme);
      reloadContent();
    });

    async function reloadContent() {
      try {
        const res = await fetch('/content?theme=' + currentTheme);
        if (res.ok) {
          const html = await res.text();
          document.getElementById('content').innerHTML = html;
        }
      } catch (err) {
        console.error('Failed to reload content:', err);
      }
    }

    // Connect to Server-Sent Events for Live-Reload
    const evtSource = new EventSource('/events');
    evtSource.onmessage = (e) => {
      if (e.data === 'reload') {
        reloadContent();
      }
    };
  </script>
</body>
</html>`,
		html.EscapeString(docTitle),
		html.EscapeString(docTitle),
		rendered,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(pageHTML))
}

func (s *ServerService) handleContent(w http.ResponseWriter, r *http.Request) {
	theme := r.URL.Query().Get("theme")
	if theme == "" {
		theme = "dark"
	}

	s.mu.RLock()
	filePath := s.filePath
	s.mu.RUnlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
		return
	}

	rendered, err := s.mdService.Render(string(data), theme)
	if err != nil {
		rendered = fmt.Sprintf("<p class='error'>Error rendering markdown: %s</p>", html.EscapeString(err.Error()))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(rendered))
}

func (s *ServerService) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan bool, 1)

	s.mu.Lock()
	s.clients[clientChan] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientChan)
		close(clientChan)
		s.mu.Unlock()
	}()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case <-clientChan:
			fmt.Fprintf(w, "data: reload\n\n")
			flusher.Flush()
		}
	}
}
