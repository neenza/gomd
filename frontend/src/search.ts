// Search and Replace Controller

export class SearchController {
  private barElem: HTMLElement;
  private searchInput: HTMLInputElement;
  private replaceInput: HTMLInputElement;
  private countLabel: HTMLElement;
  private textarea: HTMLTextAreaElement;
  private onContentChange: () => void;

  private matches: number[] = [];
  private currentMatchIndex: number = -1;

  constructor(
    barElem: HTMLElement,
    searchInput: HTMLInputElement,
    replaceInput: HTMLInputElement,
    countLabel: HTMLElement,
    textarea: HTMLTextAreaElement,
    onContentChange: () => void
  ) {
    this.barElem = barElem;
    this.searchInput = searchInput;
    this.replaceInput = replaceInput;
    this.countLabel = countLabel;
    this.textarea = textarea;
    this.onContentChange = onContentChange;

    this.bindEvents();
  }

  public show(): void {
    this.barElem.classList.remove('hidden');
    // If text is selected in editor, populate search input with it
    const start = this.textarea.selectionStart;
    const end = this.textarea.selectionEnd;
    if (start !== end) {
      const selected = this.textarea.value.substring(start, end);
      if (!selected.includes('\n') && selected.length < 80) {
        this.searchInput.value = selected;
      }
    }
    this.searchInput.focus();
    this.searchInput.select();
    this.find();
  }

  public hide(): void {
    this.barElem.classList.add('hidden');
    this.textarea.focus();
  }

  public isVisible(): boolean {
    return !this.barElem.classList.contains('hidden');
  }

  private bindEvents(): void {
    this.searchInput.addEventListener('input', () => this.find());

    this.searchInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (e.shiftKey) {
          this.findPrev();
        } else {
          this.findNext();
        }
      } else if (e.key === 'Escape') {
        this.hide();
      }
    });

    this.replaceInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        this.replace();
      } else if (e.key === 'Escape') {
        this.hide();
      }
    });

    document.getElementById('btn-find-prev')?.addEventListener('click', () => this.findPrev());
    document.getElementById('btn-find-next')?.addEventListener('click', () => this.findNext());
    document.getElementById('btn-replace')?.addEventListener('click', () => this.replace());
    document.getElementById('btn-replace-all')?.addEventListener('click', () => this.replaceAll());
    document.getElementById('btn-close-search')?.addEventListener('click', () => this.hide());
  }

  public find(): void {
    const query = this.searchInput.value;
    this.matches = [];
    this.currentMatchIndex = -1;

    if (!query) {
      this.countLabel.textContent = '0 of 0';
      return;
    }

    const text = this.textarea.value.toLowerCase();
    const q = query.toLowerCase();
    let pos = text.indexOf(q);

    while (pos !== -1) {
      this.matches.push(pos);
      pos = text.indexOf(q, pos + 1);
    }

    if (this.matches.length > 0) {
      // Find the match closest to current cursor position
      const cursorPos = this.textarea.selectionStart;
      let closestIdx = 0;
      for (let i = 0; i < this.matches.length; i++) {
        if (this.matches[i] >= cursorPos) {
          closestIdx = i;
          break;
        }
      }
      this.currentMatchIndex = closestIdx;
      this.highlightMatch(this.currentMatchIndex);
    } else {
      this.countLabel.textContent = '0 of 0';
    }
  }

  public findNext(): void {
    if (this.matches.length === 0) return;
    this.currentMatchIndex = (this.currentMatchIndex + 1) % this.matches.length;
    this.highlightMatch(this.currentMatchIndex);
  }

  public findPrev(): void {
    if (this.matches.length === 0) return;
    this.currentMatchIndex = (this.currentMatchIndex - 1 + this.matches.length) % this.matches.length;
    this.highlightMatch(this.currentMatchIndex);
  }

  private highlightMatch(index: number): void {
    if (index < 0 || index >= this.matches.length) return;
    const start = this.matches[index];
    const query = this.searchInput.value;
    const end = start + query.length;

    this.textarea.focus();
    this.textarea.setSelectionRange(start, end);

    this.countLabel.textContent = `${index + 1} of ${this.matches.length}`;
  }

  public replace(): void {
    if (this.matches.length === 0 || this.currentMatchIndex < 0) return;
    const start = this.matches[this.currentMatchIndex];
    const query = this.searchInput.value;
    const replacement = this.replaceInput.value;
    const text = this.textarea.value;

    this.textarea.value = text.substring(0, start) + replacement + text.substring(start + query.length);
    this.textarea.setSelectionRange(start, start + replacement.length);
    this.onContentChange();
    this.find();
  }

  public replaceAll(): void {
    const query = this.searchInput.value;
    if (!query) return;
    const replacement = this.replaceInput.value;
    const text = this.textarea.value;

    const regex = new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
    this.textarea.value = text.replace(regex, replacement);
    this.onContentChange();
    this.find();
  }
}
