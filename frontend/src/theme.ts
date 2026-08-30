// Theme Management Module

export type ThemeName = 'dark' | 'light' | 'oled' | 'nord' | 'solarized';

const THEME_KEY = 'gomd_theme';

export class ThemeManager {
  private currentTheme: ThemeName = 'dark';
  private onChangeCallbacks: Array<(theme: ThemeName) => void> = [];

  constructor() {
    const saved = localStorage.getItem(THEME_KEY) as ThemeName | null;
    if (saved && ['dark', 'light', 'oled', 'nord', 'solarized'].includes(saved)) {
      this.currentTheme = saved;
    } else {
      // Check system preference
      const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
      this.currentTheme = prefersDark ? 'dark' : 'light';
    }
    this.apply();
  }

  public getTheme(): ThemeName {
    return this.currentTheme;
  }

  public setTheme(theme: ThemeName): void {
    this.currentTheme = theme;
    localStorage.setItem(THEME_KEY, theme);
    this.apply();
    this.onChangeCallbacks.forEach(cb => cb(theme));
  }

  public onThemeChange(cb: (theme: ThemeName) => void): void {
    this.onChangeCallbacks.push(cb);
  }

  private apply(): void {
    document.documentElement.setAttribute('data-theme', this.currentTheme);
    const select = document.getElementById('theme-select') as HTMLSelectElement | null;
    if (select) {
      select.value = this.currentTheme;
    }
  }
}
