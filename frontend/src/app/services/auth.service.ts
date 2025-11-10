// frontend/src/app/services/auth.service.ts

import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpErrorResponse } from '@angular/common/http';
import { BehaviorSubject, Observable, of, throwError } from 'rxjs';
import { catchError, map, switchMap, tap } from 'rxjs/operators';
import { User } from '../models/api.models';
import { Router } from '@angular/router';

/**
 * Manages user authentication, including login, logout,
 * and storing the current user's data and permissions
 * fetched from /api/me.
 */
@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly apiUrl = '/api';

  /**
   * Holds the currently authenticated user's data.
   * BehaviorSubject allows components to subscribe and get the
   * current value immediately.
   */
  private currentUserSubject = new BehaviorSubject<User | null>(null);

  /**
   * Exposes the current user as an observable.
   * Components can subscribe to this to react to login/logout events.
   */
  public currentUser$ = this.currentUserSubject.asObservable();

  /**
   * A simple boolean observable to quickly check if the user is authenticated.
   */
  public isAuthenticated$ = this.currentUser$.pipe(map((user) => !!user));

  /**
   * Stores the Basic Auth credentials in base64 format (e.g., "Basic dXNlcjpwYXNz")
   */
  private currentAuthToken: string | null = null;

  constructor(private http: HttpClient, private router: Router) {
    // FIX: On service initialization, ONLY set the token from storage.
    // The AuthGuard will now be responsible for triggering the user fetch
    // to prevent a race condition on page refresh.
    this.currentAuthToken = this.getAuthTokenFromStorage();
  }

  /**
   * Attempts to log the user in with a username and password.
   * @param username The user's username.
   * @param password The user's password.
   * @returns An observable of the User object.
   */
  login(username: string, password: string): Observable<User> {
    // Create the Basic Auth token
    const token = 'Basic ' + btoa(`${username}:${password}`);
    const headers = new HttpHeaders({ Authorization: token });
    // --- DEBUG ---
    console.log(`[AUTHSERVICE] login - Attempting to GET /api/me for user '${username}'`);

    // Make the /api/me call. This serves two purposes:
    // 1. It validates the credentials (backend returns 401 if bad).
    // 2. It fetches the user's roles in the same request.
    return this.http.get<User>(`${this.apiUrl}/me`, { headers }).pipe(
      tap((user) => {
        // --- DEBUG ---
        console.log(`[AUTHSERVICE] login - Success. User roles received.`, user);
        // On success, store the token and user data
        this.currentAuthToken = token;
        sessionStorage.setItem('authToken', token);
        this.currentUserSubject.next(user);
      }),
      catchError((err: HttpErrorResponse) => {
        // --- DEBUG ---
        console.log(`[AUTHSERVICE] login - Failed. Status: ${err.status}`, err);
        // On failure, clear any existing data
        this.logout();
        
        // --- FIX: Re-throw the original HttpErrorResponse ---
        // This allows the component to see the status code.
        return throwError(() => err); 
      })
    );
  }

  /**
   * Logs the user out, clears all stored credentials,
   * and navigates to the login page.
   */
  logout(): void {
    // --- DEBUG ---
    console.log('[AUTHSERVICE] logout - Clearing session and navigating to /login');
    this.currentAuthToken = null;
    this.currentUserSubject.next(null);
    sessionStorage.removeItem('authToken');
    // We use /login as the path, which will be defined in app-routing
    this.router.navigate(['/login']);
  }

  /**
   * FIX: New public method to get the token directly from session storage.
   * This allows the AuthGuard to check for a token without relying on
   * the service's async initialization.
   */
  public getAuthTokenFromStorage(): string | null {
    return sessionStorage.getItem('authToken');
  }


  /**
   * Fetches the current user's data using the stored auth token.
   * Used to re-authenticate a user when the app loads.
   * FIX: Made public so the AuthGuard can call it.
   */
  public fetchCurrentUser(): Observable<User | null> {
    const token = this.getAuthTokenFromStorage();
    if (!token) {
      return of(null);
    }
    
    // Set the in-memory token for subsequent requests by other services
    this.currentAuthToken = token;
    const headers = new HttpHeaders({ Authorization: this.currentAuthToken });
    
    return this.http.get<User>(`${this.apiUrl}/me`, { headers }).pipe(
      tap((user) => {
        // Store the user data
        this.currentUserSubject.next(user);
      }),
      catchError(() => {
        // If the token is invalid (e.g., expired, user deleted),
        // log the user out.
        this.logout();
        return of(null);
      })
    );
  }

  /**
   * Gets the current user object.
   * @returns The current User object or null.
   */
  public getCurrentUser(): User | null {
    return this.currentUserSubject.value;
  }

  /**
   * Gets the current Basic Auth token.
   * This is used by other services (like DatabaseService)
   * to authenticate their own requests.
   * @returns The auth token string (e.g., "Basic ...") or null.
   */
  public getAuthToken(): string | null {
    return this.currentAuthToken;
  }

  /**
   * A helper to check if the current user has a specific role.
   * @param role The role to check.
   * @returns True if the user has the role, false otherwise.
   */
  public hasRole(role: keyof User): boolean {
    const user = this.getCurrentUser();
    return user ? !!user[role] : false;
  }

  changeOwnPassword(newPassword: string): Observable<any> {
    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      Authorization: this.getAuthToken() || '',
    });
    return this.http.patch(`${this.apiUrl}/me`, { password: newPassword }, { headers });
  }

  getUsers(): Observable<User[]> {
    const headers = new HttpHeaders({ Authorization: this.getAuthToken() || '' });
    return this.http.get<User[]>(`${this.apiUrl}/users`, { headers });
  }

  createUser(userData: any): Observable<User> {
    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      Authorization: this.getAuthToken() || '',
    });
    return this.http.post<User>(`${this.apiUrl}/user`, userData, { headers });
  }

  updateUser(userId: number, updates: any): Observable<User> {
    const headers = new HttpHeaders({
      'Content-Type': 'application/json',
      Authorization: this.getAuthToken() || '',
    });
    return this.http.patch<User>(`${this.apiUrl}/user?id=${userId}`, updates, { headers });
  }

  deleteUser(userId: number): Observable<{ message: string }> {
    const headers = new HttpHeaders({ Authorization: this.getAuthToken() || '' });
    return this.http.delete<{ message: string }>(`${this.apiUrl}/user?id=${userId}`, { headers });
  }
}