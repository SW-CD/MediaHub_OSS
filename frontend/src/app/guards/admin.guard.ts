import { Injectable } from '@angular/core';
import { CanActivate, UrlTree, Router } from '@angular/router';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';
import { NotificationService } from '../services/notification.service';

@Injectable({
  providedIn: 'root',
})
export class AdminGuard implements CanActivate {
  constructor(
    private authService: AuthService, 
    private router: Router,
    private notificationService: NotificationService
  ) {}

  canActivate(): Observable<boolean | UrlTree> | boolean | UrlTree {
    return this.authService.currentUser$.pipe(
      map(user => {
        // Global Role: IsAdmin bypasses all permission checks
        if (user && user.is_admin) {
          return true;
        }
        
        // Show an error before redirecting
        this.notificationService.showError('Access Denied: Administrator privileges are required.');
        return this.router.createUrlTree(['/dashboard']);
      })
    );
  }
}