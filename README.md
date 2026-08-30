# gomd

> A lightning-fast, ultra-low memory minimalist Markdown editor built with **Go** and **Wails v2**.

---

## Highlights

- **⚡ Ultra-Low Memory Footprint**: Uses ~25MB - 35MB RAM compared to 300MB - 600MB+ for Electron-based editors.
- **🚀 Native Go Markdown Processing**: Powered by `goldmark` with GitHub Flavored Markdown (tables, task lists, strikethrough, autolinks) and Chroma code syntax highlighting.
- **🎯 Distraction-Free Modes**:
  - **Split View** (`Ctrl+\`): Side-by-side editing with synchronized live preview.
  - **Edit Only** (`Ctrl+E`): Clean focused editing interface.
  - **Preview Only** (`Ctrl+P`): Reader view for documents.
  - **Zen Mode** (`F11` / `Ctrl+Shift+F`): Fullscreen distraction-free writing environment.
- **💾 Robust File I/O**: Atomic writes, unsaved changes tracking, recent files history, and native file dialogs.
- **🎨 Beautiful Modern Themes**: Dark, Light, OLED Black, Nord, and Solarized.
- **📤 Standalone HTML Export** (`Ctrl+Shift+H`): Export beautifully styled, self-contained HTML documents (with print styling for PDF).
- **🖥️ CLI Integration**: Launch with `gomd notes.md` to open files directly.

---

## Keyboard Shortcuts

| Shortcut | Action |
| :--- | :--- |
| `Ctrl+N` / `Cmd+N` | New Document |
| `Ctrl+O` / `Cmd+O` | Open File |
| `Ctrl+S` / `Cmd+S` | Save File |
| `Ctrl+Shift+S` | Save As |
| `Ctrl+Shift+H` | Export to Standalone HTML |
| `Ctrl+E` | Editor Only Mode |
| `Ctrl+\` | Split View Mode |
| `Ctrl+P` | Preview Only Mode |
| `F11` / `Ctrl+Shift+F` | Zen / Fullscreen Mode |
| `Ctrl+F` | Find & Replace |
| `Ctrl+B` | Bold (`**text**`) |
| `Ctrl+I` | Italic (`*text*`) |
| `Ctrl+K` | Insert Link (`[title](url)`) |
| `Ctrl+Shift+C` | Fenced Code Block |
| `Ctrl+Shift+X` | Strikethrough (`~~text~~`) |
| `Tab` / `Shift+Tab` | Indent / Dedent selection |
| `Ctrl+/` | Keyboard Shortcuts Cheatsheet |

---

## Architecture & Design

- **Go Backend (`main.go`, `app.go`, `services/`)**:
  - [`services.MarkdownService`](file:///home/neel/Projects/gomd/services/markdown_service.go): GFM parser, Chroma syntax highlighter, HTML sanitizer (`bluemonday`), and text metrics calculator.
  - [`services.FileService`](file:///home/neel/Projects/gomd/services/file_service.go): Atomic file writes via temporary files, native dialogs, and recent files persistence in `~/.config/gomd/`.
- **Frontend (`frontend/src/`)**:
  - Pure Vanilla TypeScript with Vite (Zero runtime framework overhead, JS bundle < 26KB).
  - Enhanced lightweight `<textarea>` engine with auto-indentation, bracket pairing, and list continuation.
  - Synchronized scrolling and interactive task list checkmarks.

---

## Building and Running

### Prerequisites
- Go 1.20+
- Node.js 18+ & npm
- GTK3 and WebKitGTK (`webkit2gtk-4.1` on Linux)
- Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Build Production Binary
```bash
wails build -tags webkit2_41
```
The compiled standalone binary will be generated at `./build/bin/gomd`.

### Run Development Mode (with Hot Reload)
```bash
wails dev -tags webkit2_41
```

### Run Tests
```bash
go test ./... -v
```
