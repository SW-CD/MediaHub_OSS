import { Injectable } from '@angular/core';
import { Permission } from '../models';
import { CanActivate, ActivatedRouteSnapshot, UrlTree, Router } from '@angular/router';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';
import { NotificationService } from '../services/notification.service';

@Injectable({
  providedIn: 'root',
})
export class DatabaseGuard implements CanActivate {
  constructor(
    private authService: AuthService, 
    private router: Router,
    private notificationService: NotificationService
  ) {}

  canActivate(route: ActivatedRouteSnapshot): Observable<boolean | UrlTree> | boolean | UrlTree {
    // Get the database name from the URL path (e.g., /database/:name)
    const dbName = route.paramMap.get('name');

    if (!dbName) {
      return this.router.createUrlTree(['/dashboard']);
    }

    return this.authService.currentUser$.pipe(
      map(user => {
        if (!user) return this.router.createUrlTree(['/login']);
        
        // Admins can view everything
        if (user.is_admin) return true;

        // Check if the user has explicitly been granted CanView for this specific database
        const hasViewPermission = user.permissions?.some(
          (p: Permission) => p.database_name === dbName && p.can_view
        );

        if (hasViewPermission) {
          return true;
        }

        this.notificationService.showError(`Access Denied: You do not have permission to view '${dbName}'.`);
        return this.router.createUrlTree(['/dashboard']);
      })
    );
  }
}