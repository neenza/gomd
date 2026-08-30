// Lightweight Textarea Markdown Editor Engine

export interface EditorCallbacks {
  onChange: (content: string) => void;
  onCursorMove: (line: number, col: number) => void;
  onScroll: (scrollPercentage: number) => void;
}

export class Editor {
  private textarea: HTMLTextAreaElement;
  private lineNumbers: HTMLElement;
  private callbacks: EditorCallbacks;
  private isComposing: boolean = false;

  constructor(
    textarea: HTMLTextAreaElement,
    lineNumbers: HTMLElement,
    callbacks: EditorCallbacks
  ) {
    this.textarea = textarea;
    this.lineNumbers = lineNumbers;
    this.callbacks = callbacks;

    this.bindEvents();
    this.updateLineNumbers();
  }

  public getValue(): string {
    return this.textarea.value;
  }

  public setValue(content: string): void {
    this.textarea.value = content;
    this.updateLineNumbers();
    this.notifyCursorMove();
    this.callbacks.onChange(content);
  }

  public setRawValue(content: string): void {
    this.textarea.value = content;
    this.updateLineNumbers();
    this.notifyCursorMove();
  }

  public getSelectionState(): { start: number; end: number; scrollTop: number } {
    return {
      start: this.textarea.selectionStart,
      end: this.textarea.selectionEnd,
      scrollTop: this.textarea.scrollTop,
    };
  }

  public setSelectionState(start: number, end: number, scrollTop: number): void {
    this.textarea.setSelectionRange(start, end);
    this.textarea.scrollTop = scrollTop;
    this.lineNumbers.scrollTop = scrollTop;
    this.notifyCursorMove();
  }

  public focus(): void {
    this.textarea.focus();
  }

  public getCursorPosition(): { line: number; col: number } {
    const text = this.textarea.value.substring(0, this.textarea.selectionStart);
    const lines = text.split('\n');
    const line = lines.length;
    const col = lines[lines.length - 1].length + 1;
    return { line, col };
  }

  public getScrollPercentage(): number {
    const maxScroll = this.textarea.scrollHeight - this.textarea.clientHeight;
    if (maxScroll <= 0) return 0;
    return this.textarea.scrollTop / maxScroll;
  }

  private bindEvents(): void {
    this.textarea.addEventListener('input', () => {
      this.updateLineNumbers();
      this.notifyCursorMove();
      this.callbacks.onChange(this.textarea.value);
    });

    this.textarea.addEventListener('scroll', () => {
      // Sync line numbers scroll
      this.lineNumbers.scrollTop = this.textarea.scrollTop;
      this.callbacks.onScroll(this.getScrollPercentage());
    });

    this.textarea.addEventListener('click', () => this.notifyCursorMove());
    this.textarea.addEventListener('keyup', (e) => {
      if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown'].includes(e.key)) {
        this.notifyCursorMove();
      }
    });

    this.textarea.addEventListener('compositionstart', () => { this.isComposing = true; });
    this.textarea.addEventListener('compositionend', () => { this.isComposing = false; });

    this.textarea.addEventListener('keydown', (e) => this.handleKeyDown(e));
  }

  private notifyCursorMove(): void {
    const { line, col } = this.getCursorPosition();
    this.callbacks.onCursorMove(line, col);
  }

  public updateLineNumbers(): void {
    const lineCount = (this.textarea.value.match(/\n/g) || []).length + 1;
    const numbersArray = [];
    for (let i = 1; i <= lineCount; i++) {
      numbersArray.push(i);
    }
    this.lineNumbers.textContent = numbersArray.join('\n');
  }

  private handleKeyDown(e: KeyboardEvent): void {
    if (this.isComposing) return;

    // Handle Tab and Shift+Tab
    if (e.key === 'Tab') {
      e.preventDefault();
      this.handleTab(e.shiftKey);
      return;
    }

    // Handle Enter (Auto-indent & list continuation)
    if (e.key === 'Enter') {
      if (this.handleEnter()) {
        e.preventDefault();
      }
      return;
    }

    // Handle auto-closing brackets and quotes
    const pairChars: Record<string, string> = {
      '(': ')',
      '[': ']',
      '{': '}',
      '"': '"',
      "'": "'",
      '`': '`',
    };

    if (pairChars[e.key]) {
      const open = e.key;
      const close = pairChars[open];
      const start = this.textarea.selectionStart;
      const end = this.textarea.selectionEnd;
      const text = this.textarea.value;

      // If text selected, wrap it
      if (start !== end) {
        e.preventDefault();
        const selected = text.substring(start, end);
        this.replaceSelection(`${open}${selected}${close}`, start + 1, end + 1);
        return;
      }
    }

    // Auto skip over closing character
    const closingChars = [')', ']', '}', '"', "'", '`'];
    if (closingChars.includes(e.key)) {
      const start = this.textarea.selectionStart;
      if (this.textarea.selectionEnd === start && this.textarea.value[start] === e.key) {
        e.preventDefault();
        this.textarea.setSelectionRange(start + 1, start + 1);
        return;
      }
    }

    // Backspace: delete matching open/close pair if empty inside
    if (e.key === 'Backspace') {
      const start = this.textarea.selectionStart;
      const end = this.textarea.selectionEnd;
      if (start === end && start > 0) {
        const prev = this.textarea.value[start - 1];
        const next = this.textarea.value[start];
        if (
          (prev === '(' && next === ')') ||
          (prev === '[' && next === ']') ||
          (prev === '{' && next === '}') ||
          (prev === '"' && next === '"') ||
          (prev === "'" && next === "'") ||
          (prev === '`' && next === '`')
        ) {
          e.preventDefault();
          const val = this.textarea.value;
          this.textarea.value = val.substring(0, start - 1) + val.substring(start + 1);
          this.textarea.setSelectionRange(start - 1, start - 1);
          this.updateLineNumbers();
          this.callbacks.onChange(this.textarea.value);
        }
      }
    }
  }

  private handleTab(shiftKey: boolean): void {
    const start = this.textarea.selectionStart;
    const end = this.textarea.selectionEnd;
    const text = this.textarea.value;
    const tabString = '  '; // 2 spaces

    if (start === end) {
      if (!shiftKey) {
        // Insert 2 spaces
        this.replaceSelection(tabString, start + tabString.length, start + tabString.length);
      }
    } else {
      // Multi-line selection indent/dedent
      const lineStart = text.lastIndexOf('\n', start - 1) + 1;
      let lineEnd = text.indexOf('\n', end);
      if (lineEnd === -1) lineEnd = text.length;

      const lines = text.substring(lineStart, lineEnd).split('\n');
      let modifiedLines: string[];

      if (shiftKey) {
        // Dedent
        modifiedLines = lines.map(line => {
          if (line.startsWith(tabString)) return line.substring(tabString.length);
          if (line.startsWith('\t') || line.startsWith(' ')) return line.substring(1);
          return line;
        });
      } else {
        // Indent
        modifiedLines = lines.map(line => tabString + line);
      }

      const replacement = modifiedLines.join('\n');
      this.textarea.value = text.substring(0, lineStart) + replacement + text.substring(lineEnd);
      this.textarea.setSelectionRange(lineStart, lineStart + replacement.length);
      this.updateLineNumbers();
      this.callbacks.onChange(this.textarea.value);
    }
  }

  private handleEnter(): boolean {
    const start = this.textarea.selectionStart;
    const text = this.textarea.value;
    const lineStart = text.lastIndexOf('\n', start - 1) + 1;
    const currentLine = text.substring(lineStart, start);

    // Check for empty list item (user wants to break out of list)
    const emptyListMatch = currentLine.match(/^(\s*)([-*+]|\d+\.|- \[[ x]\])\s*$/);
    if (emptyListMatch) {
      // Remove the list marker on current line
      const indent = emptyListMatch[1];
      this.textarea.value = text.substring(0, lineStart) + indent + text.substring(start);
      this.textarea.setSelectionRange(lineStart + indent.length, lineStart + indent.length);
      this.updateLineNumbers();
      this.callbacks.onChange(this.textarea.value);
      return true;
    }

    // Check for task list: "- [ ] " or "- [x] "
    const taskMatch = currentLine.match(/^(\s*)-\s\[([ x])\]\s+(.*)$/);
    if (taskMatch) {
      const indent = taskMatch[1];
      const prefix = `\n${indent}- [ ] `;
      this.insertAtCursor(prefix);
      return true;
    }

    // Check for unordered list: "- ", "* ", "+ "
    const bulletMatch = currentLine.match(/^(\s*)([-*+])\s+(.*)$/);
    if (bulletMatch) {
      const indent = bulletMatch[1];
      const bullet = bulletMatch[2];
      const prefix = `\n${indent}${bullet} `;
      this.insertAtCursor(prefix);
      return true;
    }

    // Check for ordered list: "1. ", "2. "
    const orderedMatch = currentLine.match(/^(\s*)(\d+)\.\s+(.*)$/);
    if (orderedMatch) {
      const indent = orderedMatch[1];
      const num = parseInt(orderedMatch[2], 10) + 1;
      const prefix = `\n${indent}${num}. `;
      this.insertAtCursor(prefix);
      return true;
    }

    // Standard auto-indent (inherit leading spaces)
    const indentMatch = currentLine.match(/^(\s+)/);
    if (indentMatch) {
      const indent = indentMatch[1];
      this.insertAtCursor(`\n${indent}`);
      return true;
    }

    return false;
  }

  public wrapSelection(before: string, after: string, defaultText: string = ''): void {
    const start = this.textarea.selectionStart;
    const end = this.textarea.selectionEnd;
    const text = this.textarea.value;
    const selected = text.substring(start, end) || defaultText;

    const replacement = `${before}${selected}${after}`;
    const newCursorPos = start + before.length + selected.length;

    this.textarea.value = text.substring(0, start) + replacement + text.substring(end);
    this.textarea.focus();
    if (start === end && defaultText === '') {
      this.textarea.setSelectionRange(start + before.length, start + before.length);
    } else {
      this.textarea.setSelectionRange(start + before.length, newCursorPos);
    }

    this.updateLineNumbers();
    this.notifyCursorMove();
    this.callbacks.onChange(this.textarea.value);
  }

  public insertAtCursor(text: string): void {
    const start = this.textarea.selectionStart;
    const end = this.textarea.selectionEnd;
    const fullText = this.textarea.value;

    this.textarea.value = fullText.substring(0, start) + text + fullText.substring(end);
    const newPos = start + text.length;
    this.textarea.setSelectionRange(newPos, newPos);
    this.textarea.focus();

    this.updateLineNumbers();
    this.notifyCursorMove();
    this.callbacks.onChange(this.textarea.value);
  }

  private replaceSelection(replacement: string, selectStart?: number, selectEnd?: number): void {
    const start = this.textarea.selectionStart;
    const end = this.textarea.selectionEnd;
    const text = this.textarea.value;

    this.textarea.value = text.substring(0, start) + replacement + text.substring(end);
    const targetStart = selectStart !== undefined ? selectStart : start + replacement.length;
    const targetEnd = selectEnd !== undefined ? selectEnd : targetStart;

    this.textarea.setSelectionRange(targetStart, targetEnd);
    this.updateLineNumbers();
    this.notifyCursorMove();
    this.callbacks.onChange(this.textarea.value);
  }

  public insertTable(rows: number = 3, cols: number = 3): void {
    let header = '|';
    let separator = '|';
    for (let c = 1; c <= cols; c++) {
      header += ` Header ${c} |`;
      separator += ' --- |';
    }
    header += '\n' + separator + '\n';

    let body = '';
    for (let r = 1; r <= rows; r++) {
      let row = '|';
      for (let c = 1; c <= cols; c++) {
        row += ` Cell ${r},${c} |`;
      }
      body += row + '\n';
    }

    this.insertAtCursor('\n' + header + body + '\n');
  }

  public toggleTaskItem(lineIndex: number): void {
    const lines = this.textarea.value.split('\n');
    if (lineIndex < 0 || lineIndex >= lines.length) return;

    const line = lines[lineIndex];
    if (line.includes('- [ ]')) {
      lines[lineIndex] = line.replace('- [ ]', '- [x]');
    } else if (line.includes('- [x]')) {
      lines[lineIndex] = line.replace('- [x]', '- [ ]');
    }

    const currentPos = this.textarea.selectionStart;
    this.textarea.value = lines.join('\n');
    this.textarea.setSelectionRange(currentPos, currentPos);
    this.callbacks.onChange(this.textarea.value);
  }
}
