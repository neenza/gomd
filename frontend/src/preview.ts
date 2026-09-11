// Markdown Preview Component, Synchronized Scrolling & On-Demand Mermaid Diagram Rendering

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
  private diagramCounter: number = 0;

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
        await this.enhanceInteractiveElements();
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

  private async enhanceInteractiveElements(): Promise<void> {
    // 1. Render Mermaid diagrams (lazily loaded on demand)
    await this.renderMermaidDiagrams();

    // 2. Intercept task-list checkboxes
    const checkboxes = this.contentElem.querySelectorAll('input[type="checkbox"]');
    checkboxes.forEach((cb, index) => {
      (cb as HTMLInputElement).removeAttribute('disabled');
      cb.addEventListener('change', (e) => {
        e.preventDefault();
        this.handleCheckboxClick(index);
      });
    });

    // 3. Handle internal link anchor scrolls
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

  private async renderMermaidDiagrams(): Promise<void> {
    const mermaidNodes = this.contentElem.querySelectorAll('pre code.language-mermaid, code.language-mermaid');
    if (mermaidNodes.length === 0) return;

    try {
      const mermaidModule = await import('mermaid');
      const mermaid = mermaidModule.default || mermaidModule;

      let mTheme: 'dark' | 'default' | 'neutral' | 'base' = 'dark';
      if (this.currentTheme === 'light') {
        mTheme = 'default';
      } else if (this.currentTheme === 'oled') {
        mTheme = 'dark';
      } else if (this.currentTheme === 'nord') {
        mTheme = 'base';
      } else if (this.currentTheme === 'solarized') {
        mTheme = 'neutral';
      }

      mermaid.initialize({
        startOnLoad: false,
        theme: mTheme,
        securityLevel: 'loose',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
        suppressErrorRendering: true,
      });

      for (let i = 0; i < mermaidNodes.length; i++) {
        const codeElem = mermaidNodes[i];
        const preElem = codeElem.closest('pre');
        const diagramCode = codeElem.textContent || '';
        if (!diagramCode.trim()) continue;

        this.diagramCounter++;
        const id = `mermaid-svg-${Date.now()}-${this.diagramCounter}`;

        try {
          const { svg } = await mermaid.render(id, diagramCode);
          const container = document.createElement('div');
          container.className = 'mermaid-container';
          container.innerHTML = svg;

          if (preElem && preElem.parentNode) {
            preElem.parentNode.replaceChild(container, preElem);
          } else if (codeElem.parentNode) {
            codeElem.parentNode.replaceChild(container, codeElem);
          }
        } catch (err) {
          // While typing incomplete diagram syntax, keep original code block
          console.debug('Mermaid rendering in progress:', err);
        }
      }
    } catch (err) {
      console.warn('Could not load mermaid module:', err);
    }
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
