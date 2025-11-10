// frontend/src/app/pages/dashboard-page/dashboard-page.component.ts
import { Component, ChangeDetectionStrategy } from '@angular/core';

@Component({
  selector: 'app-dashboard-page',
  templateUrl: './dashboard-page.component.html',
  styleUrls: ['./dashboard-page.component.css'],
  standalone: false,
  changeDetection: ChangeDetectionStrategy.OnPush, // Add OnPush for performance
})
export class DashboardPageComponent {
  
  /**
   * Tracks the collapsed state of the sidebar.
   * false = expanded, true = collapsed.
   */
  public isSidebarCollapsed: boolean = false;

  constructor() {}

  /**
   * Toggles the collapsed state of the sidebar.
   */
  toggleSidebar(): void {
    this.isSidebarCollapsed = !this.isSidebarCollapsed;
  }
}

