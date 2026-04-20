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

  public selectDatabase(name: string): Observable<Database | null> {
    return this.http
      .get<Database>(`${this.apiUrl}/database/${name}`)
      .pipe(
        tap((db) => this.selectedDatabaseSubject.next(db)),
        catchError((err) => {
          this.selectedDatabaseSubject.next(null);
          this.handleError(err);
          return of(null);
        })
      );
  }

  public createDatabase(dbData: Partial<Database>): Observable<Database> {
    return this.http.post<Database>(`${this.apiUrl}/database`, dbData).pipe(
      tap(newDb => {
        this.notificationService.showSuccess(`Database '${newDb.name}' created successfully.`);
        this.loadDatabases().subscribe();
        this.router.navigate(['/dashboard/db', newDb.name]);
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public updateDatabase(dbName: string, updates: DatabaseUpdatePayload): Observable<Database> {
    return this.http.put<Database>(`${this.apiUrl}/database/${dbName}`, updates).pipe(
      tap(updatedDb => {
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectedDatabaseSubject.next(updatedDb);
        }
        const currentDbs = this.databasesSubject.value;
        const index = currentDbs.findIndex(db => db.name === dbName);
        if (index > -1) {
            currentDbs[index] = updatedDb;
            this.databasesSubject.next([...currentDbs]);
        }
        this.notificationService.showSuccess(`Database '${dbName}' settings updated successfully.`);
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public deleteDatabase(dbName: string): Observable<{ message: string }> {
    return this.http.delete<{ message: string }>(`${this.apiUrl}/database/${dbName}`).pipe(
      tap(() => {
        this.notificationService.showSuccess(`Database '${dbName}' was successfully deleted.`);
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectedDatabaseSubject.next(null);
        }
        this.loadDatabases().subscribe();
        this.router.navigate(['/dashboard']);
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public triggerHousekeeping(dbName: string): Observable<HousekeepingReport> {
    return this.http.post<HousekeepingReport>(`${this.apiUrl}/database/${dbName}/housekeeping`, null).pipe(
      tap(report => {
        this.notificationService.showSuccess(report.message || `Housekeeping complete for '${dbName}'.`);
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectDatabase(dbName).subscribe();
        }
      }),
      catchError((err) => this.handleError(err))
    );
  }
}