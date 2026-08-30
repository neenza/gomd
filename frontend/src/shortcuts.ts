// Keyboard Shortcuts Handler

export interface ShortcutHandlers {
  onNew: () => void;
  onOpen: () => void;
  onSave: () => void;
  onSaveAs: () => void;
  onExportHTML: () => void;
  onToggleEditorMode: () => void;
  onToggleSplitMode: () => void;
  onTogglePreviewMode: () => void;
  onToggleZenMode: () => void;
  onFind: () => void;
  onFormatBold: () => void;
  onFormatItalic: () => void;
  onFormatStrike: () => void;
  onFormatCode: () => void;
  onFormatLink: () => void;
  onShowShortcuts: () => void;
  onCloseTab: () => void;
  onNextTab: () => void;
  onPrevTab: () => void;
  onSelectTabByIndex: (index: number) => void;
}

export class ShortcutManager {
  private handlers: ShortcutHandlers;

  constructor(handlers: ShortcutHandlers) {
    this.handlers = handlers;
    this.bind();
  }

  private bind(): void {
    window.addEventListener('keydown', (e: KeyboardEvent) => {
      const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
      const mod = isMac ? e.metaKey : e.ctrlKey;

      // Handle F11 Zen mode
      if (e.key === 'F11') {
        e.preventDefault();
        this.handlers.onToggleZenMode();
        return;
      }

      // Alt+1 .. Alt+9 for tab switching
      if (e.altKey && !mod && !e.shiftKey) {
        const num = parseInt(e.key, 10);
        if (num >= 1 && num <= 9) {
          e.preventDefault();
          this.handlers.onSelectTabByIndex(num - 1);
          return;
        }
      }

      // Handle shortcuts with Ctrl/Cmd modifier
      if (mod) {
        const key = e.key.toLowerCase();

        // Ctrl+Tab / Ctrl+Shift+Tab
        if (e.key === 'Tab') {
          e.preventDefault();
          if (e.shiftKey) {
            this.handlers.onPrevTab();
          } else {
            this.handlers.onNextTab();
          }
          return;
        }

        // Ctrl+PageUp / Ctrl+PageDown
        if (e.key === 'PageUp') {
          e.preventDefault();
          this.handlers.onPrevTab();
          return;
        }
        if (e.key === 'PageDown') {
          e.preventDefault();
          this.handlers.onNextTab();
          return;
        }

        // Ctrl+1 .. Ctrl+9 for tab switching
        const num = parseInt(key, 10);
        if (num >= 1 && num <= 9) {
          e.preventDefault();
          this.handlers.onSelectTabByIndex(num - 1);
          return;
        }

        if (e.shiftKey) {
          switch (key) {
            case 's':
              e.preventDefault();
              this.handlers.onSaveAs();
              return;
            case 'f':
              e.preventDefault();
              this.handlers.onToggleZenMode();
              return;
            case 'h':
              e.preventDefault();
              this.handlers.onExportHTML();
              return;
            case 'c':
              e.preventDefault();
              this.handlers.onFormatCode();
              return;
            case 'x':
              e.preventDefault();
              this.handlers.onFormatStrike();
              return;
          }
        }

        switch (key) {
          case 's':
            e.preventDefault();
            this.handlers.onSave();
            return;
          case 'w':
            e.preventDefault();
            this.handlers.onCloseTab();
            return;
          case 't':
            e.preventDefault();
            this.handlers.onNew();
            return;
          case 'o':
            e.preventDefault();
            this.handlers.onOpen();
            return;
          case 'n':
            e.preventDefault();
            this.handlers.onNew();
            return;
          case 'e':
            e.preventDefault();
            this.handlers.onToggleEditorMode();
            return;
          case 'p':
            e.preventDefault();
            this.handlers.onTogglePreviewMode();
            return;
          case '\\':
            e.preventDefault();
            this.handlers.onToggleSplitMode();
            return;
          case 'f':
            e.preventDefault();
            this.handlers.onFind();
            return;
          case 'b':
            e.preventDefault();
            this.handlers.onFormatBold();
            return;
          case 'i':
            e.preventDefault();
            this.handlers.onFormatItalic();
            return;
          case 'k':
            e.preventDefault();
            this.handlers.onFormatLink();
            return;
          case '/':
            e.preventDefault();
            this.handlers.onShowShortcuts();
            return;
        }
      }
    });
  }
}
