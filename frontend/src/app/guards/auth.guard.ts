import { Injectable } from '@angular/core';
import { CanActivate, UrlTree, Router } from '@angular/router';
import { Observable } from 'rxjs';
import { map, take } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';

@Injectable({
  providedIn: 'root',
})
export class AuthGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  canActivate(): Observable<boolean | UrlTree> | boolean | UrlTree {
    return this.authService.ensureCurrentUser().pipe(
      take(1),
      map(user => {
        if (user) {
          return true; // Valid session restored
        }
        return this.router.createUrlTree(['/login']); // Invalid/expired session
      })
    );
  }
}