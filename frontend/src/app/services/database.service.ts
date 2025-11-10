// frontend/src/app/services/database.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders, HttpParams, HttpErrorResponse } from '@angular/common/http';
import { BehaviorSubject, Observable, Subject, throwError, of, timer } from 'rxjs';
import { catchError, tap, map, switchMap, filter, take, finalize } from 'rxjs/operators';
import { Database, Entry, ApiError, Housekeeping, HousekeepingReport, SearchRequest, DatabaseConfig, PartialEntryResponse } from '../models/api.models';
import { AuthService } from './auth.service';
import { NotificationService } from './notification.service';
import { Router } from '@angular/router';

/**
 * UPDATED: Payload for PUT /api/database
 * Now includes config object.
 */
export interface DatabaseUpdatePayload {
  config?: DatabaseConfig;
  housekeeping?: Housekeeping;
}

/**
 * Manages all API interactions related to databases and entries.
 * REFACTOR: Renamed all "Image" methods to "Entry".
 */
@Injectable({
  providedIn: 'root',
})
export class DatabaseService {
  private readonly apiUrl = '/api';

  private databasesSubject = new BehaviorSubject<Database[]>([]);
  private selectedDatabaseSubject = new BehaviorSubject<Database | null>(null);
  // RENAMED: selectedImageSubject -> selectedEntrySubject
  private selectedEntrySubject = new BehaviorSubject<Entry | null>(null);
  private refreshNotifier = new Subject<void>();

  // --- ADDED FOR ASYNC UPLOAD ---
  private processingEntriesSubject = new BehaviorSubject<number[]>([]);
  public processingEntries$ = this.processingEntriesSubject.asObservable();
  // --- END ADDED ---

  public databases$ = this.databasesSubject.asObservable();
  public selectedDatabase$ = this.selectedDatabaseSubject.asObservable();
  // RENAMED: selectedImage$ -> selectedEntry$
  public selectedEntry$ = this.selectedEntrySubject.asObservable();
  public refreshRequired$ = this.refreshNotifier.asObservable();

  constructor(
    private http: HttpClient,
    private authService: AuthService,
    private notificationService: NotificationService,
    private router: Router
  ) {}

  public triggerImageListRefresh(): void {
    this.refreshNotifier.next();
  }

  private getAuthHeaders(isJson: boolean = false): HttpHeaders {
    const token = this.authService.getAuthToken();
    if (!token) {
      console.error("[DEBUG] DatabaseService: Auth token is missing in getAuthHeaders."); // Log error
      this.notificationService.showGlobalError('Authentication token is missing.');
      this.authService.logout();
      return new HttpHeaders();
    }
    let headers = new HttpHeaders({ Authorization: token });
    if (isJson) {
      headers = headers.set('Content-Type', 'application/json');
    }
    return headers;
  }

  // Centralized error handling for HTTP requests
  private handleError(error: HttpErrorResponse): Observable<never> {
    console.error("[DEBUG] DatabaseService: Full HTTP Error Response:", error); // Log full error object

    let errorMessage: string;
    let isAuthError = false;

    // Prioritize specific backend error message if available
    if (error.error && typeof error.error.error === 'string') {
        errorMessage = error.error.error;
    }
    // Handle specific status codes
    else if (error.status === 0) {
      errorMessage = 'Network error or backend unreachable. Check CORS or server status.';
    } else if (error.status === 401) {
      errorMessage = 'Authentication failed or session expired. Please log in again.';
      isAuthError = true; // Flag for logout
    } else if (error.status === 403) {
      errorMessage = 'Forbidden: You lack permission for this action.';
    } else if (error.status === 400) {
      errorMessage = `Bad Request: ${error.error?.error || 'Invalid input.'}`; // Use backend message if possible
    } else if (error.status >= 500) {
      errorMessage = `Server Error (${error.status}): ${error.statusText}. Please try again later.`;
    } else if (error.statusText) {
      errorMessage = `Error ${error.status}: ${error.statusText}`;
    } else {
      errorMessage = 'An unknown error occurred.';
    }

    console.error("[DEBUG] DatabaseService: Parsed error message:", errorMessage);

    // Handle side effects based on error type
    if (isAuthError) {
      this.notificationService.showGlobalError(errorMessage);
      this.authService.logout();
    } else {
      // Show error as a toast for non-auth issues
      this.notificationService.showError(errorMessage);
    }

    // Re-throw the error message string for component-level handling if needed
    return throwError(() => new Error(errorMessage));
  }


  public loadDatabases(): Observable<Database[]> {
    const headers = this.getAuthHeaders();
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing')); // Prevent request without token

    return this.http
      .get<Database[]>(`${this.apiUrl}/databases`, { headers })
      .pipe(
        tap((databases) => {
          this.databasesSubject.next(databases || []); // Ensure it's an array
        }),
        catchError((err) => this.handleError(err))
      );
  }

  public selectDatabase(name: string): Observable<Database | null> { // Allow null return on error
    console.log(`[DEBUG] DatabaseService: selectDatabase called for name: '${name}'`);
    const headers = this.getAuthHeaders();
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams().set('name', name);

    return this.http
      .get<Database>(`${this.apiUrl}/database`, { headers, params })
      .pipe(
        tap((db) => {
          console.log(`[DEBUG] DatabaseService: selectDatabase HTTP GET successful for '${name}'. Tapping to update subject.`, db);
          this.selectedDatabaseSubject.next(db);
        }),
        catchError((err) => {
          console.error(`[DEBUG] DatabaseService: selectDatabase HTTP GET failed for '${name}'.`, err);
          this.selectedDatabaseSubject.next(null); // Clear selection on error
          this.handleError(err); // Still show error to user
          return of(null); // Return null observable on error
        })
      );
  }

  /**
   * Searches entries using the complex search endpoint.
   */
  public searchEntries(dbName: string, payload: SearchRequest): Observable<Entry[]> {
    const headers = this.getAuthHeaders(true); // Set Content-Type: application/json
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams().set('name', dbName);

    console.log(`[DEBUG] DatabaseService: Calling POST /api/database/entries/search for db '${dbName}'`);
    console.log("[DEBUG] DatabaseService: Search Payload:", JSON.stringify(payload, null, 2));


    return this.http
      .post<Entry[]>(`${this.apiUrl}/database/entries/search`, payload, { headers, params })
      .pipe(
        tap(entries => console.log(`[DEBUG] DatabaseService: Received ${entries?.length ?? 0} entries from search.`)),
        catchError((err) => this.handleError(err)) // Use centralized error handler
      );
  }

  public selectEntry(entry: Entry): void {
    this.selectedEntrySubject.next(entry);
  }

  public clearSelectedEntry(): void {
    this.selectedEntrySubject.next(null);
  }

  public createDatabase(dbData: Partial<Database>): Observable<Database> {
    const headers = this.getAuthHeaders(true); // Set Content-Type: application/json
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    return this.http.post<Database>(`${this.apiUrl}/database`, dbData, { headers }).pipe(
      tap(newDb => {
        this.notificationService.showSuccess(`Database '${newDb.name}' created successfully.`);
        this.loadDatabases().subscribe(); // Refresh the list
        this.router.navigate(['/dashboard/db', newDb.name]); // Navigate to the new DB
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public updateDatabase(dbName: string, updates: DatabaseUpdatePayload): Observable<Database> {
    const headers = this.getAuthHeaders(true); // Set Content-Type: application/json
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams().set('name', dbName);

    return this.http.put<Database>(`${this.apiUrl}/database`, updates, { headers, params }).pipe(
      tap(updatedDb => {
        // Update the selected DB subject if it matches
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectedDatabaseSubject.next(updatedDb);
        }
        // Update the main list as well
        const currentDbs = this.databasesSubject.value;
        const index = currentDbs.findIndex(db => db.name === dbName);
        if (index > -1) {
            currentDbs[index] = updatedDb;
            this.databasesSubject.next([...currentDbs]); // Emit new array
        }
        this.notificationService.showSuccess(`Database '${dbName}' settings updated successfully.`);
      }),
      catchError((err) => this.handleError(err))
    );
  }


  public triggerHousekeeping(dbName: string): Observable<HousekeepingReport> {
    const headers = this.getAuthHeaders();
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams().set('name', dbName);

    return this.http.post<HousekeepingReport>(`${this.apiUrl}/database/housekeeping`, null, { headers, params }).pipe(
      tap(report => {
        this.notificationService.showSuccess(report.message || `Housekeeping complete for '${dbName}'.`);
        // Refresh the currently selected DB to update stats
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectDatabase(dbName).subscribe();
        }
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public deleteDatabase(dbName: string): Observable<{ message: string }> {
    const headers = this.getAuthHeaders();
    if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams().set('name', dbName);

    return this.http.delete<{ message: string }>(`${this.apiUrl}/database`, { headers, params }).pipe(
      tap(() => {
        this.notificationService.showSuccess(`Database '${dbName}' was successfully deleted.`);
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectedDatabaseSubject.next(null); // Clear selection if deleted
        }
        this.loadDatabases().subscribe(); // Refresh list
        this.router.navigate(['/dashboard']); // Navigate away
      }),
      catchError((err) => this.handleError(err))
    );
  }

  /**
   * UPDATED: To handle 201 (sync) and 202 (async) responses
   * and trigger an immediate refresh for 202.
   */
  public uploadEntry(dbName: string, metadata: Omit<Entry, 'id' | 'width' | 'height' | 'filesize' | 'mime_type' | 'status'>, file: File): Observable<void> {
    const headers = this.getAuthHeaders(); // Don't set Content-Type, browser does it for FormData
     if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams().set('database_name', dbName);

    const formData = new FormData();
    formData.append('metadata', JSON.stringify(metadata));
    formData.append('file', file, file.name);

    return this.http.post<Entry | PartialEntryResponse>(`${this.apiUrl}/entry`, formData, { 
      headers, 
      params,
      observe: 'response' // <-- Set observe: 'response'
    }).pipe(
      tap(response => {
        // Handle 201 Created (Sync)
        if (response.status === 201) {
          this.notificationService.showSuccess('Entry uploaded successfully.');
          this.triggerImageListRefresh();
        }
        
        // Handle 202 Accepted (Async)
        if (response.status === 202) {
          const partialEntry = response.body as PartialEntryResponse;
          this.addProcessingEntry(partialEntry.id);
          this.notificationService.showInfo(`Large file (ID: ${partialEntry.id}) is processing...`);
          // Start polling
          this.pollForEntryStatus(dbName, partialEntry.id);
          // --- UPDATED: Trigger refresh immediately to show "processing" state ---
          this.triggerImageListRefresh();
          // --- END UPDATE ---
        }
      }),
      map(() => void 0), // Transform the result to void for the component
      catchError((err) => this.handleError(err))
    );
  }

  public getEntryMeta(dbName: string, entryId: number): Observable<Entry> {
    const headers = this.getAuthHeaders();
     if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http
      .get<Entry>(`${this.apiUrl}/entry/meta`, { headers, params })
      .pipe(catchError((err) => this.handleError(err)));
  }

  public getEntryFileBlob(dbName: string, entryId: number): Observable<Blob> {
    const headers = this.getAuthHeaders();
     if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    // Request the file data as a Blob
    return this.http.get(`${this.apiUrl}/entry/file`, {
        headers,
        params,
        responseType: 'blob' // Important: Angular handles the Blob response
      })
      .pipe(catchError((err) => this.handleError(err)));
  }

  public getEntryPreviewUrl(dbName: string, entryId: number): string {
    return `${this.apiUrl}/entry/preview?database_name=${dbName}&id=${entryId}`;
  }

  public updateEntry(dbName: string, entryId: number, updates: Partial<Entry>): Observable<Entry> {
    const headers = this.getAuthHeaders(true); // Set Content-Type: application/json
     if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http.patch<Entry>(`${this.apiUrl}/entry`, updates, { headers, params }).pipe(
      tap(() => {
        this.notificationService.showSuccess(`Entry ${entryId} updated successfully.`);
        this.triggerImageListRefresh(); // Notify list component
      }),
      catchError(err => this.handleError(err))
    );
  }

  public deleteEntry(dbName: string, entryId: number): Observable<{ message: string }> {
    const headers = this.getAuthHeaders();
     if (!headers.has('Authorization')) return throwError(() => new Error('Auth token missing'));

    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http.delete<{ message: string }>(`${this.apiUrl}/entry`, { headers, params }).pipe(
      tap(res => {
        this.notificationService.showSuccess(res.message || `Entry ${entryId} deleted.`);
        this.triggerImageListRefresh(); // Notify list component
      }),
      catchError(err => this.handleError(err))
    );
  }

  // --- ADDED HELPER METHODS FOR ASYNC UPLOAD ---

  private addProcessingEntry(id: number): void {
    const current = this.processingEntriesSubject.value;
    if (!current.includes(id)) {
      this.processingEntriesSubject.next([...current, id]);
    }
  }

  private removeProcessingEntry(id: number): void {
    const current = this.processingEntriesSubject.value;
    this.processingEntriesSubject.next(current.filter(entryId => entryId !== id));
  }

  private pollForEntryStatus(dbName: string, entryId: number): void {
    // Poll every 2 seconds (2000ms)
    timer(2000, 2000).pipe(
      // Switch to the getEntryMeta call
      switchMap(() => this.getEntryMeta(dbName, entryId)),
      // Stop polling once status is no longer 'processing'
      filter(entry => entry.status !== 'processing'),
      // Take only the first such emission
      take(1),
      // Add a timeout (e.g., 30 attempts = 1 minute)
      take(30),
      // Finalize block runs on completion or timeout
      finalize(() => {
        // Check if polling timed out (entry is still in processing list)
        if (this.processingEntriesSubject.value.includes(entryId)) {
           this.removeProcessingEntry(entryId);
           this.notificationService.showError(`Polling for entry ${entryId} timed out.`);
        }
      })
    ).subscribe({
      next: entry => {
        this.removeProcessingEntry(entry.id);
        if (entry.status === 'ready') {
          this.notificationService.showSuccess(`Entry ${entry.id} processing complete.`);
          this.triggerImageListRefresh();
        } else if (entry.status === 'error') {
          this.notificationService.showError(`Entry ${entry.id} failed to process.`);
          // Trigger a refresh so the list can show the 'error' state
          this.triggerImageListRefresh();
        }
      },
      error: err => {
        // Handle errors during polling (e.g., 404 if entry failed/deleted)
        this.removeProcessingEntry(entryId);
        // The error itself is already shown by handleError in getEntryMeta
        // We just ensure it's removed from the processing list.
        console.error(`Error polling for entry ${entryId}:`, err);
      }
    });
  }
}