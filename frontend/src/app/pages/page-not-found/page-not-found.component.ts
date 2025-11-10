// frontend/src/app/pages/page-not-found/page-not-found.component.ts

import { Component } from '@angular/core';

@Component({
  selector: 'app-page-not-found',
  template: `
    <div style="text-align: center; padding-top: 5rem; font-family: Arial, sans-serif;">
      <h1>404 - Not Found</h1>
      <p>The page you are looking for does not exist.</p>
      <a routerLink="/">Go to Dashboard</a>
    </div>
  `,
  standalone: false
})
export class PageNotFoundComponent {}