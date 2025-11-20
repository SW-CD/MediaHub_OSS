// frontend/src/app/interceptors/jwt.interceptor.ts

import { Injectable } from '@angular/core';
import {
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpInterceptor,
  HttpErrorResponse
} from '@angular/common/http';
import { Observable, throwError, BehaviorSubject } from 'rxjs';
import { catchError, filter, switchMap, take } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';
import { TokenResponse } from '../models/api.models';

@Injectable()
export class JwtInterceptor implements HttpInterceptor {
  private isRefreshing = false;
  private refreshTokenSubject: BehaviorSubject<string | null> = new BehaviorSubject<string | null>(null);

  constructor(private authService: AuthService) {}

  intercept(request: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    // 1. Add Bearer token if available
    let authReq = request;
    const token = this.authService.getAccessToken();
    
    // Skip adding headers for the /token endpoints to avoid circular logic or overwriting Basic Auth
    if (request.url.includes('/api/token')) {
        return next.handle(request);
    }

    if (token) {
      authReq = this.addTokenHeader(request, token);
    }

    // 2. Handle the request and catch 401s
    return next.handle(authReq).pipe(
      catchError((error) => {
        if (error instanceof HttpErrorResponse && error.status === 401) {
          // If 401, try to refresh the token
          return this.handle401Error(authReq, next);
        }
        return throwError(() => error);
      })
    );
  }

  private handle401Error(request: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    if (!this.isRefreshing) {
      this.isRefreshing = true;
      this.refreshTokenSubject.next(null);

      return this.authService.refreshToken().pipe(
        switchMap((tokenResponse: TokenResponse) => {
          this.isRefreshing = false;
          this.refreshTokenSubject.next(tokenResponse.access_token);
          // Retry the original request with the new token
          return next.handle(this.addTokenHeader(request, tokenResponse.access_token));
        }),
        catchError((err) => {
          this.isRefreshing = false;
          this.authService.logout(false); // Logout locally without hitting API (since tokens are dead)
          return throwError(() => err);
        })
      );
    } else {
      // If already refreshing, wait until the new token is ready
      return this.refreshTokenSubject.pipe(
        filter((token) => token != null),
        take(1),
        switchMap((token) => {
          return next.handle(this.addTokenHeader(request, token!));
        })
      );
    }
  }

  private addTokenHeader(request: HttpRequest<any>, token: string): HttpRequest<any> {
    return request.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }
}