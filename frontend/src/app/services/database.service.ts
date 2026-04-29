// frontend/src/app/services/database.service.ts

import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { BehaviorSubject, Observable, throwError, of } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';
import { Database, HousekeepingReport, DatabaseConfig } from '../models';
import { NotificationService } from './notification.service';
import { Router } from '@angular/router';

export interface DatabaseUpdatePayload {
  config?: DatabaseConfig;
  housekeeping?: any;
}

@Injectable({
  providedIn: 'root',
})
export class DatabaseService {
  private readonly apiUrl = 'api';

  // State specific to databases
  private databasesSubject = new BehaviorSubject<Database[]>([]);
  private selectedDatabaseSubject = new BehaviorSubject<Database | null>(null);

  public databases$ = this.databasesSubject.asObservable();
  public selectedDatabase$ = this.selectedDatabaseSubject.asObservable();

  constructor(
    private http: HttpClient,
    private notificationService: NotificationService,
    private router: Router
  ) {}

  /**
   * Centralized error handler for HTTP requests.
   */
  private handleError(error: HttpErrorResponse): Observable<never> {
    console.error("[DEBUG] DatabaseService: Full HTTP Error Response:", error);

    let errorMessage: string;
    let isAuthError = false;

    if (error.error && typeof error.error.error === 'string') {
        errorMessage = error.error.error;
    } else if (error.status === 0) {
      errorMessage = 'Network error or backend unreachable. Check CORS or server status.';
    } else if (error.status === 401) {
      errorMessage = 'Authentication failed or session expired. Please log in again.';
      isAuthError = true;
    } else if (error.status === 403) {
      errorMessage = 'Forbidden: You lack permission for this action.';
    } else if (error.status === 400) {
      errorMessage = `Bad Request: ${error.error?.error || 'Invalid input.'}`;
    } else if (error.status >= 500) {
      errorMessage = `Server Error (${error.status}): ${error.statusText}. Please try again later.`;
    } else if (error.statusText) {
      errorMessage = `Error ${error.status}: ${error.statusText}`;
    } else {
      errorMessage = 'An unknown error occurred.';
    }

    if (isAuthError) {
      this.notificationService.showGlobalError(errorMessage);
    } else {
      this.notificationService.showError(errorMessage);
    }

    return throwError(() => new Error(errorMessage));
  }

  // --- DATABASE ENDPOINTS ---

  public loadDatabases(): Observable<Database[]> {
    return this.http
      .get<Database[]>(`${this.apiUrl}/databases`)
      .pipe(
        tap((databases) => this.databasesSubject.next(databases || [])),
        catchError((err) => this.handleError(err))
      );
  }

  /**
   * Fetches a single database by its ULID.
   */
  public selectDatabase(id: string): Observable<Database | null> {
    return this.http
      .get<Database>(`${this.apiUrl}/database/${id}`)
      .pipe(
        tap((db) => this.selectedDatabaseSubject.next(db)),
        catchError((err) => {
          this.selectedDatabaseSubject.next(null);
          this.handleError(err);
          return of(null);
        })
      );
  }

  /**
   * Creates a new database and navigates to its ID-based route.
   */
  public createDatabase(dbData: Partial<Database>): Observable<Database> {
    return this.http.post<Database>(`${this.apiUrl}/database`, dbData).pipe(
      tap(newDb => {
        this.notificationService.showSuccess(`Database '${newDb.name}' created successfully.`);
        this.loadDatabases().subscribe();
        
        // Navigate using the newly generated ULID instead of the name
        this.router.navigate(['/dashboard/db', (newDb as any).id]); 
      }),
      catchError((err) => this.handleError(err))
    );
  }

  /**
   * Updates an existing database using its ULID.
   */
  public updateDatabase(id: string, updates: DatabaseUpdatePayload): Observable<Database> {
    return this.http.put<Database>(`${this.apiUrl}/database/${id}`, updates).pipe(
      tap(updatedDb => {
        // Update the currently selected DB if it matches the ID
        if ((this.selectedDatabaseSubject.value as any)?.id === id) {
            this.selectedDatabaseSubject.next(updatedDb);
        }
        
        // Update the specific database in our local array list
        const currentDbs = this.databasesSubject.value;
        const index = currentDbs.findIndex((db: any) => db.id === id);
        if (index > -1) {
            currentDbs[index] = updatedDb;
            this.databasesSubject.next([...currentDbs]);
        }
        this.notificationService.showSuccess(`Database '${updatedDb.name}' settings updated successfully.`);
      }),
      catchError((err) => this.handleError(err))
    );
  }

  /**
   * Deletes a database by its ULID.
   */
  public deleteDatabase(id: string): Observable<{ message: string }> {
    return this.http.delete<{ message: string }>(`${this.apiUrl}/database/${id}`).pipe(
      tap((res) => {
        // Utilizing the backend's explicit message string for the notification
        this.notificationService.showSuccess(res.message);
        
        if ((this.selectedDatabaseSubject.value as any)?.id === id) {
            this.selectedDatabaseSubject.next(null);
        }
        this.loadDatabases().subscribe();
        this.router.navigate(['/dashboard']);
      }),
      catchError((err) => this.handleError(err))
    );
  }

  /**
   * Triggers the housekeeping background worker using the ULID.
   */
  public triggerHousekeeping(id: string): Observable<HousekeepingReport> {
    return this.http.post<HousekeepingReport>(`${this.apiUrl}/database/${id}/housekeeping`, null).pipe(
      tap(report => {
        this.notificationService.showSuccess(report.message || `Housekeeping complete.`);
        if ((this.selectedDatabaseSubject.value as any)?.id === id) {
            this.selectDatabase(id).subscribe();
        }
      }),
      catchError((err) => this.handleError(err))
    );
  }
}