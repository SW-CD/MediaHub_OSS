import { Injectable } from '@angular/core';
import { CanActivate, UrlTree, Router } from '@angular/router';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';

@Injectable({
  providedIn: 'root',
})
export class AuthGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  canActivate(): Observable<boolean | UrlTree> | boolean | UrlTree {
    // 1. Check if user is already in memory
    if (this.authService.getCurrentUser()) {
      return true;
    }

    // 2. Check if a token exists in storage.
    // Note: Update 'getAuthTokenFromStorage()' to whatever method you use to get the access token
    const token = this.authService.getAccessToken(); 
    if (!token) {
      return this.router.createUrlTree(['/login']);
    }

    // 3. Token exists, but no user in memory (Page Refresh). Fetch the user.
    return this.authService.fetchCurrentUser().pipe(
      map(user => {
        if (user) {
          return true; // Valid session restored
        }
        return this.router.createUrlTree(['/login']); // Invalid/expired session
      })
    );
  }
}