// frontend/src/app/app-routing.module.ts

import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { LoginPageComponent } from './pages/login-page/login-page.component';
import { DashboardPageComponent } from './pages/dashboard-page/dashboard-page.component';
import { PageNotFoundComponent } from './pages/page-not-found/page-not-found.component';
import { AuthGuard } from './guards/auth.guard';
import { AdminGuard } from './guards/admin.guard';
// UPDATED: Import renamed EntryListComponent
import { EntryListComponent } from './components/entry-list/entry-list.component';
import { DatabaseSettingsComponent } from './components/database-settings/database-settings.component';
import { AdminUserListComponent } from './components/admin-user-list/admin-user-list.component';

/**
 * Defines the application's routes.
 */
const routes: Routes = [
  {
    path: 'login',
    component: LoginPageComponent,
  },
  {
    path: 'dashboard',
    component: DashboardPageComponent,
    canActivate: [AuthGuard], // Protect this route and its children
    children: [
      // Child routes for the dashboard's <router-outlet>
      {
        path: 'db/:name', // e.g., /dashboard/db/MyDatabase
        // UPDATED: Use renamed EntryListComponent
        component: EntryListComponent,
      },
      {
        path: 'settings/:name', // e.g., /dashboard/settings/MyDatabase
        component: DatabaseSettingsComponent,
      },
      {
        path: 'admin/users',
        component: AdminUserListComponent,
        canActivate: [AdminGuard],
      },
      {
        path: '',
        // UPDATED: Load EntryListComponent directly
        component: EntryListComponent,
        pathMatch: 'full',
      },
    ],
  },
  {
    path: '',
    redirectTo: '/dashboard', // Default route redirects to dashboard
    pathMatch: 'full',
  },
  {
    path: '**', // Wildcard route for 404
    component: PageNotFoundComponent,
  },
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule],
})
export class AppRoutingModule {}
