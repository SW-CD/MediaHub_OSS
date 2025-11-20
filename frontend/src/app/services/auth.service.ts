// frontend/src/app/services/auth.service.ts

import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpErrorResponse } from '@angular/common/http';
import { BehaviorSubject, Observable, of, throwError } from 'rxjs';
import { catchError, map, switchMap, tap } from 'rxjs/operators';
import { User, TokenResponse } from '../models/api.models';
import { Router } from '@angular/router';

/**
 * Manages user authentication using JWT (JSON Web Tokens).
 * Handles Login (Token Fetch), Logout (Token Revocation), and Token Refresh.
 */
@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly apiUrl = '/api';
  private readonly ACCESS_TOKEN_KEY = 'access_token';
  private readonly REFRESH_TOKEN_KEY = 'refresh_token';

  private currentUserSubject = new BehaviorSubject<User | null>(null);
  public currentUser$ = this.currentUserSubject.asObservable();

  // Helper to check if we have a user loaded
  public isAuthenticated$ = this.currentUser$.pipe(map((user) => !!user));

  constructor(private http: HttpClient, private router: Router) {}

  /**
   * Logs the user in by exchanging credentials for JWT tokens,
   * then fetching the user's profile.
   */
  login(username: string, password: string): Observable<User> {
    // 1. Prepare Basic Auth header for the token endpoint
    const basicAuth = 'Basic ' + btoa(`${username}:${password}`);
    const headers = new HttpHeaders({ Authorization: basicAuth });

    // 2. Call POST /api/token to get the tokens
    return this.http.post<TokenResponse>(`${this.apiUrl}/token`, {}, { headers }).pipe(
      tap((tokens) => {
        this.storeTokens(tokens);
      }),
      // 3. Once we have tokens, fetch the user profile (Interceptor will inject Bearer token)
      switchMap(() => this.fetchCurrentUser()),
      map(user => {
        if (!user) throw new Error('Failed to fetch user details after login');
        return user;
      }),
      catchError((err: HttpErrorResponse) => {
        this.logout(false); // Clean up if anything fails
        return throwError(() => err);
      })
    );
  }

  /**
   * Logs the user out.
   * Attempts to revoke the refresh token on the server, then clears local state.
   * @param notifyServer Whether to call the API to revoke the token (default: true)
   */
  logout(notifyServer: boolean = true): void {
    const refreshToken = this.getRefreshToken();
    
    if (notifyServer && refreshToken) {
      // Call the server to revoke the token
      // We use a simple object for the body as per the spec
      this.http.post(`${this.apiUrl}/logout`, { refresh_token: refreshToken }).subscribe({
        next: () => console.log('Logout successful on server'),
        error: (err) => console.warn('Logout failed on server (token might be expired)', err),
        complete: () => this.clearSessionAndRedirect()
      });
    } else {
      this.clearSessionAndRedirect();
    }
  }

  private clearSessionAndRedirect(): void {
    this.clearTokens();
    this.currentUserSubject.next(null);
    this.router.navigate(['/login']);
  }

  /**
   * Refreshes the access token using the refresh token.
   * This is typically called by the JwtInterceptor.
   */
  refreshToken(): Observable<TokenResponse> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) {
      return throwError(() => new Error('No refresh token available'));
    }

    return this.http.post<TokenResponse>(`${this.apiUrl}/token/refresh`, { 
      refresh_token: refreshToken 
    }).pipe(
      tap((tokens) => {
        this.storeTokens(tokens);
      })
    );
  }

  /**
   * Fetches the current user's data using the stored access token.
   */
  public fetchCurrentUser(): Observable<User | null> {
    if (!this.getAccessToken()) {
      return of(null);
    }
    
    return this.http.get<User>(`${this.apiUrl}/me`).pipe(
      tap((user) => {
        this.currentUserSubject.next(user);
      }),
      catchError(() => {
        // If fetching user fails (e.g., 401 even after refresh attempts), log out locally
        this.clearSessionAndRedirect();
        return of(null);
      })
    );
  }

  // --- Token Management Helpers ---

  private storeTokens(tokens: TokenResponse): void {
    localStorage.setItem(this.ACCESS_TOKEN_KEY, tokens.access_token);
    localStorage.setItem(this.REFRESH_TOKEN_KEY, tokens.refresh_token);
  }

  private clearTokens(): void {
    localStorage.removeItem(this.ACCESS_TOKEN_KEY);
    localStorage.removeItem(this.REFRESH_TOKEN_KEY);
  }

  public getAccessToken(): string | null {
    return localStorage.getItem(this.ACCESS_TOKEN_KEY);
  }

  public getRefreshToken(): string | null {
    return localStorage.getItem(this.REFRESH_TOKEN_KEY);
  }

  /**
   * @deprecated Use getAccessToken() or let the Interceptor handle it.
   * Kept for compatibility if other components check storage existence.
   */
  public getAuthTokenFromStorage(): string | null {
    return this.getAccessToken();
  }

  public getCurrentUser(): User | null {
    return this.currentUserSubject.value;
  }

  // --- User Management Methods (unchanged) ---

  public hasRole(role: keyof User): boolean {
    const user = this.getCurrentUser();
    return user ? !!user[role] : false;
  }

  changeOwnPassword(newPassword: string): Observable<any> {
    return this.http.patch(`${this.apiUrl}/me`, { password: newPassword });
  }

  getUsers(): Observable<User[]> {
    return this.http.get<User[]>(`${this.apiUrl}/users`);
  }

  createUser(userData: any): Observable<User> {
    return this.http.post<User>(`${this.apiUrl}/user`, userData);
  }

  updateUser(userId: number, updates: any): Observable<User> {
    return this.http.patch<User>(`${this.apiUrl}/user?id=${userId}`, updates);
  }

  deleteUser(userId: number): Observable<{ message: string }> {
    return this.http.delete<{ message: string }>(`${this.apiUrl}/user?id=${userId}`);
  }
}