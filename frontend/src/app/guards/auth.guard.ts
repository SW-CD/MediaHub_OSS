// frontend/src/app/guards/auth.guard.ts

import { Injectable } from '@angular/core';
import {
  CanActivate,
  ActivatedRouteSnapshot,
  RouterStateSnapshot,
  UrlTree,
  Router,
} from '@angular/router';
import { Observable, of } from 'rxjs';
import { map, take } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';

/**
 * A route guard that prevents unauthenticated users from
 * accessing protected routes (like the dashboard).
 */
@Injectable({
  providedIn: 'root',
})
export class AuthGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  /**
   * Checks if the user is authenticated.
   * If not, it redirects them to the /login page.
   * FIX: This logic is updated to handle the page refresh race condition.
   */
  canActivate(
    route: ActivatedRouteSnapshot,
    state: RouterStateSnapshot
  ):
    | Observable<boolean | UrlTree>
    | Promise<boolean | UrlTree>
    | boolean
    | UrlTree {
    
    // 1. Check if user is already in memory (logged in and navigating).
    if (this.authService.getCurrentUser()) {
      return true;
    }

    // 2. Check if a token exists in storage. If not, they are not logged in.
    const token = this.authService.getAuthTokenFromStorage();
    if (!token) {
      return this.router.createUrlTree(['/login']);
    }

    // 3. Token exists, but no user in memory. This is the page refresh case.
    // We must call fetchCurrentUser() and wait for the async result before
    // allowing or denying access to the route.
    return this.authService.fetchCurrentUser().pipe(
      map(user => {
        if (user) {
          return true; // The token was valid, user is fetched, allow access.
        }
        // The token was invalid, fetchCurrentUser failed and called logout().
        // Redirect to login as a fallback.
        return this.router.createUrlTree(['/login']);
      })
    );
  }
}
