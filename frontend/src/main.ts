// gomd - Main Application Entry Point

import { ThemeManager } from './theme';
import { Editor } from './editor';
import { Preview } from './preview';
import { SearchController } from './search';
import { ShortcutManager } from './shortcuts';

import {
  OpenFile,
  SaveFile,
  OpenFileDialog,
  SaveFileDialog,
  ExportHTMLDialog,
  GetStats,
  GetRecentFiles,
  ClearRecentFiles,
  GetInitialFile,
  SetWindowTitle
} from '../wailsjs/go/main/App';

export type ViewMode = 'editor' | 'split' | 'preview';

class GomdApp {
  private themeManager: ThemeManager;
  private editor: Editor;
  private preview: Preview;
  private searchController: SearchController;
  public shortcutManager: ShortcutManager;
  public currentMode: ViewMode = 'split';
  private currentFilePath: string = '';
  private currentFileName: string = 'Untitled.md';
  private isDirty: boolean = false;
  private isZen: boolean = false;

  // DOM Elements
  private appContainer: HTMLElement;
  private fileNameLabel: HTMLElement;
  private dirtyDot: HTMLElement;
  private statPosition: HTMLElement;
  private statLines: HTMLElement;
  private statWords: HTMLElement;
  private statChars: HTMLElement;
  private statReadTime: HTMLElement;
  private statPath: HTMLElement;
  private statSaveState: HTMLElement;

  private btnModeEditor: HTMLButtonElement;
  private btnModeSplit: HTMLButtonElement;
  private btnModePreview: HTMLButtonElement;

  private recentModal: HTMLElement;
  private shortcutsModal: HTMLElement;
  private recentFilesList: HTMLElement;

  constructor() {
    this.appContainer = document.getElementById('app')!;
    this.fileNameLabel = document.getElementById('file-name-label')!;
    this.dirtyDot = document.getElementById('dirty-dot')!;
    this.statPosition = document.getElementById('stat-position')!;
    this.statLines = document.getElementById('stat-lines')!;
    this.statWords = document.getElementById('stat-words')!;
    this.statChars = document.getElementById('stat-chars')!;
    this.statReadTime = document.getElementById('stat-readtime')!;
    this.statPath = document.getElementById('stat-path')!;
    this.statSaveState = document.getElementById('stat-save-state')!;

    this.btnModeEditor = document.getElementById('btn-mode-editor') as HTMLButtonElement;
    this.btnModeSplit = document.getElementById('btn-mode-split') as HTMLButtonElement;
    this.btnModePreview = document.getElementById('btn-mode-preview') as HTMLButtonElement;

    this.recentModal = document.getElementById('recent-modal')!;
    this.shortcutsModal = document.getElementById('shortcuts-modal')!;
    this.recentFilesList = document.getElementById('recent-files-list')!;

    // 1. Initialize Theme Manager
    this.themeManager = new ThemeManager();
    this.themeManager.onThemeChange((theme) => {
      this.preview.update(this.editor.getValue(), theme, true);
    });

    const themeSelect = document.getElementById('theme-select') as HTMLSelectElement;
    themeSelect.addEventListener('change', () => {
      this.themeManager.setTheme(themeSelect.value as any);
    });

    // 2. Initialize Preview
    const previewContainer = document.getElementById('preview-scroll-container')!;
    const previewContent = document.getElementById('preview-content')!;
    this.preview = new Preview({
      container: previewContainer,
      contentElem: previewContent,
      onToggleTask: (lineIndex) => {
        this.editor.toggleTaskItem(lineIndex);
      },
    });

    // 3. Initialize Editor
    const textarea = document.getElementById('editor') as HTMLTextAreaElement;
    const lineNumbers = document.getElementById('line-numbers')!;
    this.editor = new Editor(textarea, lineNumbers, {
      onChange: (content) => this.handleContentChange(content),
      onCursorMove: (line, col) => this.handleCursorMove(line, col),
      onScroll: (pct) => this.preview.syncScroll(pct),
    });

    // 4. Initialize Search
    const searchBar = document.getElementById('search-bar')!;
    const searchInput = document.getElementById('search-input') as HTMLInputElement;
    const replaceInput = document.getElementById('replace-input') as HTMLInputElement;
    const searchCount = document.getElementById('search-count')!;
    this.searchController = new SearchController(
      searchBar,
      searchInput,
      replaceInput,
      searchCount,
      textarea,
      () => this.handleContentChange(textarea.value)
    );

    // 5. Initialize Shortcuts
    this.shortcutManager = new ShortcutManager({
      onNew: () => this.newDocument(),
      onOpen: () => this.openFile(),
      onSave: () => this.saveFile(),
      onSaveAs: () => this.saveFileAs(),
      onExportHTML: () => this.exportHTML(),
      onToggleEditorMode: () => this.setViewMode('editor'),
      onToggleSplitMode: () => this.setViewMode('split'),
      onTogglePreviewMode: () => this.setViewMode('preview'),
      onToggleZenMode: () => this.toggleZenMode(),
      onFind: () => this.searchController.show(),
      onFormatBold: () => this.editor.wrapSelection('**', '**', 'bold text'),
      onFormatItalic: () => this.editor.wrapSelection('*', '*', 'italic text'),
      onFormatStrike: () => this.editor.wrapSelection('~~', '~~', 'strikethrough text'),
      onFormatCode: () => this.editor.wrapSelection('```\n', '\n```', 'code here'),
      onFormatLink: () => this.editor.wrapSelection('[', '](https://example.com)', 'link title'),
      onShowShortcuts: () => this.showShortcutsModal(),
    });

    this.bindUI();
    this.initStartup();
  }

  private bindUI(): void {
    // Mode Buttons
    this.btnModeEditor.addEventListener('click', () => this.setViewMode('editor'));
    this.btnModeSplit.addEventListener('click', () => this.setViewMode('split'));
    this.btnModePreview.addEventListener('click', () => this.setViewMode('preview'));

    // File Action Buttons
    document.getElementById('btn-new')?.addEventListener('click', () => this.newDocument());
    document.getElementById('btn-open')?.addEventListener('click', () => this.openFile());
    document.getElementById('btn-save')?.addEventListener('click', () => this.saveFile());
    document.getElementById('btn-recent')?.addEventListener('click', () => this.showRecentModal());
    document.getElementById('btn-export-html')?.addEventListener('click', () => this.exportHTML());
    document.getElementById('btn-zen')?.addEventListener('click', () => this.toggleZenMode());
    document.getElementById('btn-shortcuts')?.addEventListener('click', () => this.showShortcutsModal());

    // Formatting Toolbar Buttons
    document.getElementById('btn-fmt-bold')?.addEventListener('click', () => this.editor.wrapSelection('**', '**', 'bold text'));
    document.getElementById('btn-fmt-italic')?.addEventListener('click', () => this.editor.wrapSelection('*', '*', 'italic text'));
    document.getElementById('btn-fmt-strike')?.addEventListener('click', () => this.editor.wrapSelection('~~', '~~', 'strikethrough text'));
    document.getElementById('btn-fmt-code')?.addEventListener('click', () => this.editor.wrapSelection('```\n', '\n```', 'code here'));
    document.getElementById('btn-fmt-link')?.addEventListener('click', () => this.editor.wrapSelection('[', '](https://)', 'link title'));
    document.getElementById('btn-fmt-quote')?.addEventListener('click', () => this.editor.wrapSelection('> ', '', 'quoted text'));
    document.getElementById('btn-fmt-table')?.addEventListener('click', () => this.editor.insertTable(3, 3));

    // Modals Close
    document.getElementById('btn-close-recent')?.addEventListener('click', () => this.hideRecentModal());
    document.getElementById('btn-clear-recent')?.addEventListener('click', () => this.handleClearRecent());
    document.getElementById('btn-close-shortcuts')?.addEventListener('click', () => this.hideShortcutsModal());

    this.recentModal.addEventListener('click', (e) => {
      if (e.target === this.recentModal) this.hideRecentModal();
    });
    this.shortcutsModal.addEventListener('click', (e) => {
      if (e.target === this.shortcutsModal) this.hideShortcutsModal();
    });

    // Resizer Dragging
    const resizer = document.getElementById('pane-resizer');
    const editorPane = document.getElementById('editor-pane');
    if (resizer && editorPane) {
      let isDragging = false;

      resizer.addEventListener('mousedown', (e) => {
        isDragging = true;
        resizer.classList.add('dragging');
        document.body.style.cursor = 'col-resize';
        e.preventDefault();
      });

      window.addEventListener('mousemove', (e) => {
        if (!isDragging) return;
        const workspaceRect = this.appContainer.getBoundingClientRect();
        const newWidth = e.clientX - workspaceRect.left;
        const percentage = (newWidth / workspaceRect.width) * 100;
        if (percentage >= 15 && percentage <= 85) {
          editorPane.style.flex = `0 0 ${percentage}%`;
        }
      });

      window.addEventListener('mouseup', () => {
        if (isDragging) {
          isDragging = false;
          resizer.classList.remove('dragging');
          document.body.style.cursor = '';
        }
      });
    }

    // Drag & drop file support
    window.addEventListener('dragover', (e) => e.preventDefault());
    window.addEventListener('drop', async (e) => {
      e.preventDefault();
      if (e.dataTransfer && e.dataTransfer.files.length > 0) {
        const file = e.dataTransfer.files[0];
        const filePath = (file as any).path;
        if (filePath) {
          await this.loadFilePath(filePath);
        } else {
          const reader = new FileReader();
          reader.onload = () => {
            if (typeof reader.result === 'string') {
              this.currentFilePath = '';
              this.currentFileName = file.name || 'Untitled.md';
              this.editor.setValue(reader.result);
              this.markDirty(false);
              this.updateTitle();
            }
          };
          reader.readAsText(file);
        }
      }
    });
  }

  private async initStartup(): Promise<void> {
    try {
      const initial = await GetInitialFile();
      if (initial && initial.content !== undefined) {
        this.currentFilePath = initial.path;
        this.currentFileName = initial.name;
        this.editor.setValue(initial.content);
        this.markDirty(false);
        this.updateTitle();
        return;
      }
    } catch (err) {
      console.warn('No initial file found:', err);
    }

    // Default welcome template
    const sampleMarkdown = `# Welcome to gomd

A lightning-fast, distraction-free minimalist Markdown editor built with **Go** and **Wails**.

---

### Key Features
- **Ultra-low memory footprint** (~30MB RAM)
- **Live Markdown preview** with GitHub-flavored syntax
- **Split**, **Editor only**, and **Preview only** modes
- **Zen / Distraction-free mode** (\`F11\` or \`Ctrl+Shift+F\`)
- **Fast shortcuts** for quick formatting
- **Standalone HTML Export** (\`Ctrl+Shift+H\`)

### Task List
- [x] Create document
- [ ] Write great notes
- [ ] Export to HTML

### Code Syntax Highlighting
\`\`\`go
package main

import "fmt"

func main() {
    fmt.Println("Hello, minimal Markdown!")
}
\`\`\`

> *"Simplicity is the ultimate sophistication."* — Leonardo da Vinci
`;

    this.editor.setValue(sampleMarkdown);
    this.markDirty(false);
    this.updateTitle();
  }

  private handleContentChange(content: string): void {
    this.markDirty(true);
    this.preview.update(content, this.themeManager.getTheme());
    this.updateStats(content);
  }

  private handleCursorMove(line: number, col: number): void {
    this.statPosition.textContent = `Ln ${line}, Col ${col}`;
  }

  private async updateStats(content: string): Promise<void> {
    try {
      const stats = await GetStats(content);
      this.statLines.textContent = `${stats.lines} ${stats.lines === 1 ? 'line' : 'lines'}`;
      this.statWords.textContent = `${stats.words} ${stats.words === 1 ? 'word' : 'words'}`;
      this.statChars.textContent = `${stats.characters} ${stats.characters === 1 ? 'char' : 'chars'}`;
      this.statReadTime.textContent = `${stats.readingTime} read`;
    } catch {
      // Fallback local calc
      const lines = (content.match(/\n/g) || []).length + 1;
      this.statLines.textContent = `${lines} lines`;
    }
  }

  private markDirty(dirty: boolean): void {
    this.isDirty = dirty;
    if (dirty) {
      this.dirtyDot.classList.add('is-dirty');
      this.statSaveState.textContent = 'Unsaved changes';
      this.statSaveState.style.color = 'var(--warning)';
    } else {
      this.dirtyDot.classList.remove('is-dirty');
      this.statSaveState.textContent = 'Saved';
      this.statSaveState.style.color = 'var(--success)';
    }
    this.updateTitle();
  }

  private updateTitle(): void {
    this.fileNameLabel.textContent = this.currentFileName;
    this.statPath.textContent = this.currentFilePath || 'Untitled';
    const displayTitle = (this.isDirty ? '● ' : '') + this.currentFileName;
    SetWindowTitle(displayTitle);
  }

  public setViewMode(mode: ViewMode): void {
    this.currentMode = mode;
    this.appContainer.classList.remove('editor-mode', 'split-mode', 'preview-mode');
    this.btnModeEditor.classList.remove('active');
    this.btnModeSplit.classList.remove('active');
    this.btnModePreview.classList.remove('active');

    if (mode === 'editor') {
      this.appContainer.classList.add('editor-mode');
      this.btnModeEditor.classList.add('active');
      this.editor.focus();
    } else if (mode === 'preview') {
      this.appContainer.classList.add('preview-mode');
      this.btnModePreview.classList.add('active');
    } else {
      this.appContainer.classList.add('split-mode');
      this.btnModeSplit.classList.add('active');
    }
  }

  public toggleZenMode(): void {
    this.isZen = !this.isZen;
    if (this.isZen) {
      this.appContainer.classList.add('zen-mode');
    } else {
      this.appContainer.classList.remove('zen-mode');
    }
    this.editor.focus();
  }

  public newDocument(): void {
    if (this.isDirty && !confirm('You have unsaved changes. Discard and create new file?')) {
      return;
    }
    this.currentFilePath = '';
    this.currentFileName = 'Untitled.md';
    this.editor.setValue('');
    this.markDirty(false);
    this.updateTitle();
    this.editor.focus();
  }

  public async openFile(): Promise<void> {
    try {
      const fileInfo = await OpenFileDialog();
      if (fileInfo && fileInfo.path) {
        this.currentFilePath = fileInfo.path;
        this.currentFileName = fileInfo.name;
        this.editor.setValue(fileInfo.content);
        this.markDirty(false);
        this.updateTitle();
      }
    } catch (err) {
      console.error('Failed to open file:', err);
    }
  }

  public async loadFilePath(path: string): Promise<void> {
    try {
      const fileInfo = await OpenFile(path);
      if (fileInfo && fileInfo.path) {
        this.currentFilePath = fileInfo.path;
        this.currentFileName = fileInfo.name;
        this.editor.setValue(fileInfo.content);
        this.markDirty(false);
        this.updateTitle();
      }
    } catch (err) {
      alert(`Could not open file: ${err}`);
    }
  }

  public async saveFile(): Promise<void> {
    if (!this.currentFilePath) {
      return this.saveFileAs();
    }

    try {
      const saved = await SaveFile(this.currentFilePath, this.editor.getValue());
      if (saved) {
        this.markDirty(false);
        this.statSaveState.textContent = 'Saved';
      }
    } catch (err) {
      alert(`Error saving file: ${err}`);
    }
  }

  public async saveFileAs(): Promise<void> {
    try {
      const saved = await SaveFileDialog(this.currentFileName, this.editor.getValue());
      if (saved && saved.path) {
        this.currentFilePath = saved.path;
        this.currentFileName = saved.name;
        this.markDirty(false);
        this.updateTitle();
      }
    } catch (err) {
      alert(`Error saving file: ${err}`);
    }
  }

  public async exportHTML(): Promise<void> {
    try {
      const defaultName = this.currentFileName.replace(/\.md$/, '.html');
      const exportedPath = await ExportHTMLDialog(
        defaultName,
        this.editor.getValue(),
        this.themeManager.getTheme()
      );
      if (exportedPath) {
        alert(`Successfully exported HTML to:\n${exportedPath}`);
      }
    } catch (err) {
      alert(`Error exporting HTML: ${err}`);
    }
  }

  public async showRecentModal(): Promise<void> {
    this.recentModal.classList.remove('hidden');
    try {
      const files = await GetRecentFiles();
      this.recentFilesList.innerHTML = '';

      if (!files || files.length === 0) {
        this.recentFilesList.innerHTML = '<li class="empty-list">No recent files</li>';
        return;
      }

      files.forEach((path) => {
        const li = document.createElement('li');
        li.className = 'recent-item';
        const name = path.split('/').pop() || path;
        li.innerHTML = `
          <span class="recent-name">${name}</span>
          <span class="recent-path">${path}</span>
        `;
        li.addEventListener('click', async () => {
          this.hideRecentModal();
          await this.loadFilePath(path);
        });
        this.recentFilesList.appendChild(li);
      });
    } catch (err) {
      console.error('Failed to load recent files:', err);
    }
  }

  public hideRecentModal(): void {
    this.recentModal.classList.add('hidden');
  }

  public async handleClearRecent(): Promise<void> {
    await ClearRecentFiles();
    this.showRecentModal();
  }

  public showShortcutsModal(): void {
    this.shortcutsModal.classList.remove('hidden');
  }

  public hideShortcutsModal(): void {
    this.shortcutsModal.classList.add('hidden');
  }
}

// Initialize Application once DOM is ready
window.addEventListener('DOMContentLoaded', () => {
  new GomdApp();
});
