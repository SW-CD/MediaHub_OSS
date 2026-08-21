import { Injectable } from '@angular/core';
import {
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpInterceptor,
  HttpErrorResponse
} from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError, switchMap, shareReplay, finalize } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';
import { TokenResponse } from '../models';

@Injectable()
export class JwtInterceptor implements HttpInterceptor {
  private refreshToken$: Observable<TokenResponse> | null = null;

  constructor(private authService: AuthService) {}

  intercept(request: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    let authReq = request;
    const token = this.authService.getAccessToken();
    
    // UPDATED: Check for 'api/token' without the leading slash to match our relative paths!
    // This ensures both 'api/token' and 'api/token/refresh' are skipped
    if (request.url.includes('api/token')) {
        return next.handle(request);
    }

    // 1. Add Bearer token if available
    if (token) {
      authReq = this.addTokenHeader(request, token);
    }

    // 2. Handle the request and catch 401s
    return next.handle(authReq).pipe(
      catchError((error) => {
        if (error instanceof HttpErrorResponse && error.status === 401) {
          // Skip token refresh for password update requests to avoid logouts on incorrect credentials
          if (request.url.includes('api/me') && request.method === 'PATCH') {
            return throwError(() => error);
          }
          // If 401, try to refresh the token
          return this.handle401Error(authReq, next);
        }
        return throwError(() => error);
      })
    );
  }

  private handle401Error(request: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    if (!this.refreshToken$) {
      this.refreshToken$ = this.authService.refreshToken().pipe(
        shareReplay(1),
        finalize(() => {
          this.refreshToken$ = null;
        })
      );
    }

    return this.refreshToken$.pipe(
      switchMap((tokenResponse: TokenResponse) => {
        return next.handle(this.addTokenHeader(request, tokenResponse.access_token));
      }),
      catchError((err) => {
        this.authService.logout(false); // Logout locally without hitting API (since tokens are dead)
        return throwError(() => err);
      })
    );
  }

  private addTokenHeader(request: HttpRequest<any>, token: string): HttpRequest<any> {
    return request.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }
}