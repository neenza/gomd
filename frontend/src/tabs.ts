// Tab and Multi-File Manager

export interface Tab {
  id: string;
  filePath: string;
  fileName: string;
  content: string;
  isDirty: boolean;
  cursorLine: number;
  cursorCol: number;
  selectionStart: number;
  selectionEnd: number;
  scrollTop: number;
}

export interface TabManagerCallbacks {
  onTabSwitched: (currentTab: Tab, prevTab?: Tab) => void;
  onTabClosed: (closedTab: Tab, nextActiveTab?: Tab) => void;
  onNewTabRequested: () => void;
  onTabDirtyChanged: (tab: Tab) => void;
}

export class TabManager {
  private tabs: Tab[] = [];
  private activeTabId: string = '';
  private tabCounter: number = 0;
  private containerElem: HTMLElement;
  private callbacks: TabManagerCallbacks;

  constructor(containerElem: HTMLElement, callbacks: TabManagerCallbacks) {
    this.containerElem = containerElem;
    this.callbacks = callbacks;
  }

  public getTabs(): Tab[] {
    return this.tabs;
  }

  public getActiveTab(): Tab | undefined {
    return this.tabs.find((t) => t.id === this.activeTabId);
  }

  public createTab(fileName: string = 'Untitled.md', content: string = '', filePath: string = ''): Tab {
    this.tabCounter++;
    const id = `tab-${Date.now()}-${this.tabCounter}`;
    const newTab: Tab = {
      id,
      filePath,
      fileName: fileName || `Untitled-${this.tabCounter}.md`,
      content,
      isDirty: false,
      cursorLine: 1,
      cursorCol: 1,
      selectionStart: 0,
      selectionEnd: 0,
      scrollTop: 0,
    };

    this.tabs.push(newTab);
    this.render();
    this.switchTab(id);
    return newTab;
  }

  public switchTab(tabId: string): void {
    if (this.activeTabId === tabId && this.getActiveTab()) {
      return;
    }

    const prevTab = this.getActiveTab();
    const nextTab = this.tabs.find((t) => t.id === tabId);
    if (!nextTab) return;

    this.activeTabId = tabId;
    this.render();
    this.callbacks.onTabSwitched(nextTab, prevTab);
  }

  public closeTab(tabId: string): void {
    const tabIndex = this.tabs.findIndex((t) => t.id === tabId);
    if (tabIndex === -1) return;

    const tabToClose = this.tabs[tabIndex];

    if (tabToClose.isDirty) {
      const confirmClose = window.confirm(`"${tabToClose.fileName}" has unsaved changes. Discard and close?`);
      if (!confirmClose) return;
    }

    this.tabs.splice(tabIndex, 1);

    // If we closed the active tab, pick a new active tab
    if (this.activeTabId === tabId) {
      if (this.tabs.length > 0) {
        const nextIndex = Math.min(tabIndex, this.tabs.length - 1);
        const nextTab = this.tabs[nextIndex];
        this.activeTabId = nextTab.id;
        this.render();
        this.callbacks.onTabClosed(tabToClose, nextTab);
      } else {
        // No tabs left: create a fresh Untitled tab
        const freshTab = this.createTab('Untitled.md', '', '');
        this.callbacks.onTabClosed(tabToClose, freshTab);
      }
    } else {
      this.render();
      this.callbacks.onTabClosed(tabToClose, this.getActiveTab());
    }
  }

  public closeActiveTab(): void {
    if (this.activeTabId) {
      this.closeTab(this.activeTabId);
    }
  }

  public nextTab(): void {
    if (this.tabs.length <= 1) return;
    const currentIndex = this.tabs.findIndex((t) => t.id === this.activeTabId);
    const nextIndex = (currentIndex + 1) % this.tabs.length;
    this.switchTab(this.tabs[nextIndex].id);
  }

  public prevTab(): void {
    if (this.tabs.length <= 1) return;
    const currentIndex = this.tabs.findIndex((t) => t.id === this.activeTabId);
    const prevIndex = (currentIndex - 1 + this.tabs.length) % this.tabs.length;
    this.switchTab(this.tabs[prevIndex].id);
  }

  public selectTabByIndex(index: number): void {
    if (index >= 0 && index < this.tabs.length) {
      this.switchTab(this.tabs[index].id);
    }
  }

  public findTabByPath(filePath: string): Tab | undefined {
    if (!filePath) return undefined;
    return this.tabs.find((t) => t.filePath === filePath);
  }

  public updateActiveTabState(
    content: string,
    isDirty: boolean,
    cursorLine: number,
    cursorCol: number,
    selectionStart: number,
    selectionEnd: number,
    scrollTop: number
  ): void {
    const tab = this.getActiveTab();
    if (!tab) return;

    const dirtyChanged = tab.isDirty !== isDirty;
    tab.content = content;
    tab.isDirty = isDirty;
    tab.cursorLine = cursorLine;
    tab.cursorCol = cursorCol;
    tab.selectionStart = selectionStart;
    tab.selectionEnd = selectionEnd;
    tab.scrollTop = scrollTop;

    if (dirtyChanged) {
      this.render();
      this.callbacks.onTabDirtyChanged(tab);
    }
  }

  public updateActiveFileInfo(filePath: string, fileName: string): void {
    const tab = this.getActiveTab();
    if (!tab) return;

    tab.filePath = filePath;
    tab.fileName = fileName;
    tab.isDirty = false;
    this.render();
    this.callbacks.onTabDirtyChanged(tab);
  }

  public render(): void {
    this.containerElem.innerHTML = '';

    const tabListElem = document.createElement('div');
    tabListElem.className = 'tab-strip-list';

    this.tabs.forEach((tab) => {
      const tabElem = document.createElement('div');
      tabElem.className = `tab-item ${tab.id === this.activeTabId ? 'active' : ''} ${tab.isDirty ? 'is-dirty' : ''}`;
      tabElem.title = tab.filePath || tab.fileName;

      // File icon
      const icon = document.createElement('span');
      icon.className = 'tab-icon';
      icon.innerHTML = `<svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`;
      tabElem.appendChild(icon);

      // Title
      const titleSpan = document.createElement('span');
      titleSpan.className = 'tab-title';
      titleSpan.textContent = tab.fileName;
      tabElem.appendChild(titleSpan);

      // Dirty dot
      const dirtySpan = document.createElement('span');
      dirtySpan.className = 'tab-dirty-indicator';
      tabElem.appendChild(dirtySpan);

      // Close button
      const closeBtn = document.createElement('button');
      closeBtn.className = 'tab-close-btn';
      closeBtn.innerHTML = '✕';
      closeBtn.title = 'Close Tab (Ctrl+W)';
      closeBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        this.closeTab(tab.id);
      });
      tabElem.appendChild(closeBtn);

      // Click to switch tab
      tabElem.addEventListener('click', () => {
        this.switchTab(tab.id);
      });

      // Middle click to close tab
      tabElem.addEventListener('auxclick', (e) => {
        if (e.button === 1) { // Middle click
          e.preventDefault();
          this.closeTab(tab.id);
        }
      });

      tabListElem.appendChild(tabElem);
    });

    this.containerElem.appendChild(tabListElem);

    // New Tab (+) button
    const newTabBtn = document.createElement('button');
    newTabBtn.className = 'tab-new-btn';
    newTabBtn.title = 'New Tab (Ctrl+N / Ctrl+T)';
    newTabBtn.innerHTML = `<svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`;
    newTabBtn.addEventListener('click', () => {
      this.callbacks.onNewTabRequested();
    });
    this.containerElem.appendChild(newTabBtn);
  }
}
