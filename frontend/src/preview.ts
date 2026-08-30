// Markdown Preview Component & Synchronized Scrolling

import { RenderMarkdown } from '../wailsjs/go/main/App';

export interface PreviewOptions {
  container: HTMLElement;
  contentElem: HTMLElement;
  onToggleTask?: (lineIndex: number) => void;
}

export class Preview {
  private container: HTMLElement;
  private contentElem: HTMLElement;
  private onToggleTask?: (lineIndex: number) => void;
  private currentMarkdown: string = '';
  private currentTheme: string = 'dark';
  private debounceTimer: number | null = null;
  private isRendering: boolean = false;
  private queuedMarkdown: string | null = null;

  constructor(options: PreviewOptions) {
    this.container = options.container;
    this.contentElem = options.contentElem;
    this.onToggleTask = options.onToggleTask;
  }

  public update(markdown: string, theme: string, immediate: boolean = false): void {
    this.currentMarkdown = markdown;
    this.currentTheme = theme;

    if (this.debounceTimer !== null) {
      window.clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }

    if (immediate) {
      this.executeRender();
    } else {
      this.debounceTimer = window.setTimeout(() => {
        this.executeRender();
      }, 50);
    }
  }

  public syncScroll(percentage: number): void {
    const maxScroll = this.container.scrollHeight - this.container.clientHeight;
    if (maxScroll > 0) {
      this.container.scrollTop = percentage * maxScroll;
    }
  }

  private async executeRender(): Promise<void> {
    if (this.isRendering) {
      this.queuedMarkdown = this.currentMarkdown;
      return;
    }

    this.isRendering = true;
    try {
      if (!this.currentMarkdown.trim()) {
        this.contentElem.innerHTML = '<p class="preview-placeholder">Preview will appear here...</p>';
      } else {
        const html = await RenderMarkdown(this.currentMarkdown, this.currentTheme);
        this.contentElem.innerHTML = html;
        this.enhanceInteractiveElements();
      }
    } catch (err) {
      console.error('Failed to render markdown:', err);
    } finally {
      this.isRendering = false;
      if (this.queuedMarkdown !== null) {
        const next = this.queuedMarkdown;
        this.queuedMarkdown = null;
        this.update(next, this.currentTheme, true);
      }
    }
  }

  private enhanceInteractiveElements(): void {
    // Intercept task-list checkboxes
    const checkboxes = this.contentElem.querySelectorAll('input[type="checkbox"]');
    checkboxes.forEach((cb, index) => {
      // Enable checkbox for interaction
      (cb as HTMLInputElement).removeAttribute('disabled');

      cb.addEventListener('change', (e) => {
        e.preventDefault();
        this.handleCheckboxClick(index);
      });
    });

    // Handle internal link anchor scrolls
    const links = this.contentElem.querySelectorAll('a[href^="#"]');
    links.forEach((a) => {
      a.addEventListener('click', (e) => {
        e.preventDefault();
        const href = a.getAttribute('href');
        if (href && href.length > 1) {
          const target = this.contentElem.querySelector(href);
          if (target) {
            target.scrollIntoView({ behavior: 'smooth', block: 'start' });
          }
        }
      });
    });
  }

  private handleCheckboxClick(taskIndex: number): void {
    if (!this.onToggleTask) return;

    // Find corresponding line in markdown text
    const lines = this.currentMarkdown.split('\n');
    let currentTaskCount = 0;

    for (let i = 0; i < lines.length; i++) {
      if (lines[i].match(/^\s*-\s\[[ x]\]/)) {
        if (currentTaskCount === taskIndex) {
          this.onToggleTask(i);
          return;
        }
        currentTaskCount++;
      }
    }
  }
}
