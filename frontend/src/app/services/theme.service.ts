import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private activeTheme = new BehaviorSubject<'light' | 'dark'>('light');
  public theme$ = this.activeTheme.asObservable();

  constructor() {
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null;
    if (savedTheme) {
      this.setTheme(savedTheme);
    } else {
      this.setTheme('light');
    }
  }

  public get currentTheme(): 'light' | 'dark' {
    return this.activeTheme.value;
  }

  public toggleTheme(): void {
    const nextTheme = this.activeTheme.value === 'light' ? 'dark' : 'light';
    this.setTheme(nextTheme);
  }

  private setTheme(theme: 'light' | 'dark'): void {
    this.activeTheme.next(theme);
    localStorage.setItem('theme', theme);
    
    if (theme === 'light') {
      document.body.classList.add('light-theme');
    } else {
      document.body.classList.remove('light-theme');
    }
  }
}
