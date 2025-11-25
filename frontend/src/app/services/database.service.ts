// frontend/src/app/services/database.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpParams, HttpErrorResponse } from '@angular/common/http'; // Removed HttpHeaders usage
import { BehaviorSubject, Observable, Subject, throwError, of, timer } from 'rxjs';
import { catchError, tap, map, switchMap, filter, take, finalize } from 'rxjs/operators';
import { Database, Entry, HousekeepingReport, SearchRequest, DatabaseConfig, PartialEntryResponse } from '../models/api.models';
import { AuthService } from './auth.service';
import { NotificationService } from './notification.service';
import { Router } from '@angular/router';

export interface DatabaseUpdatePayload {
  config?: DatabaseConfig;
  housekeeping?: any; // Using any here to match previous usage or strict Housekeeping type
}

/**
 * Manages all API interactions related to databases and entries.
 */
@Injectable({
  providedIn: 'root',
})
export class DatabaseService {
  private readonly apiUrl = '/api';

  private databasesSubject = new BehaviorSubject<Database[]>([]);
  private selectedDatabaseSubject = new BehaviorSubject<Database | null>(null);
  private selectedEntrySubject = new BehaviorSubject<Entry | null>(null);
  private refreshNotifier = new Subject<void>();

  private processingEntriesSubject = new BehaviorSubject<number[]>([]);
  public processingEntries$ = this.processingEntriesSubject.asObservable();

  public databases$ = this.databasesSubject.asObservable();
  public selectedDatabase$ = this.selectedDatabaseSubject.asObservable();
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

  // Centralized error handling for HTTP requests
  private handleError(error: HttpErrorResponse): Observable<never> {
    console.error("[DEBUG] DatabaseService: Full HTTP Error Response:", error);

    let errorMessage: string;
    let isAuthError = false;

    if (error.error && typeof error.error.error === 'string') {
        errorMessage = error.error.error;
    } else if (error.status === 0) {
      errorMessage = 'Network error or backend unreachable. Check CORS or server status.';
    } else if (error.status === 401) {
      // 401s are mostly handled by the interceptor now, but if they bubble up here,
      // it means refresh failed.
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
      // The interceptor usually calls logout, but we can double check here
      // We don't call authService.logout() here to avoid loops if the interceptor is already handling it.
    } else {
      this.notificationService.showError(errorMessage);
    }

    return throwError(() => new Error(errorMessage));
  }


  public loadDatabases(): Observable<Database[]> {
    // Interceptor handles Auth
    return this.http
      .get<Database[]>(`${this.apiUrl}/databases`)
      .pipe(
        tap((databases) => {
          this.databasesSubject.next(databases || []);
        }),
        catchError((err) => this.handleError(err))
      );
  }

  public selectDatabase(name: string): Observable<Database | null> {
    const params = new HttpParams().set('name', name);

    return this.http
      .get<Database>(`${this.apiUrl}/database`, { params })
      .pipe(
        tap((db) => {
          this.selectedDatabaseSubject.next(db);
        }),
        catchError((err) => {
          this.selectedDatabaseSubject.next(null);
          this.handleError(err);
          return of(null);
        })
      );
  }

  public searchEntries(dbName: string, payload: SearchRequest): Observable<Entry[]> {
    const params = new HttpParams().set('name', dbName);

    return this.http
      .post<Entry[]>(`${this.apiUrl}/database/entries/search`, payload, { params })
      .pipe(
        catchError((err) => this.handleError(err))
      );
  }

  public selectEntry(entry: Entry): void {
    this.selectedEntrySubject.next(entry);
  }

  public clearSelectedEntry(): void {
    this.selectedEntrySubject.next(null);
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
    const params = new HttpParams().set('name', dbName);

    return this.http.put<Database>(`${this.apiUrl}/database`, updates, { params }).pipe(
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


  public triggerHousekeeping(dbName: string): Observable<HousekeepingReport> {
    const params = new HttpParams().set('name', dbName);

    return this.http.post<HousekeepingReport>(`${this.apiUrl}/database/housekeeping`, null, { params }).pipe(
      tap(report => {
        this.notificationService.showSuccess(report.message || `Housekeeping complete for '${dbName}'.`);
        if (this.selectedDatabaseSubject.value?.name === dbName) {
            this.selectDatabase(dbName).subscribe();
        }
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public deleteDatabase(dbName: string): Observable<{ message: string }> {
    const params = new HttpParams().set('name', dbName);

    return this.http.delete<{ message: string }>(`${this.apiUrl}/database`, { params }).pipe(
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

  public uploadEntry(dbName: string, metadata: Omit<Entry, 'id' | 'width' | 'height' | 'filesize' | 'mime_type' | 'status'>, file: File): Observable<void> {
    const params = new HttpParams().set('database_name', dbName);

    const formData = new FormData();
    formData.append('metadata', JSON.stringify(metadata));
    formData.append('file', file, file.name);

    return this.http.post<Entry | PartialEntryResponse>(`${this.apiUrl}/entry`, formData, { 
      params,
      observe: 'response'
    }).pipe(
      tap(response => {
        // Case 1: Synchronous Creation (201)
        if (response.status === 201) {
          const entry = response.body as Entry;
          this.notificationService.showSuccess('Entry uploaded successfully.');
          
          // FIX: Check if the synchronously created entry is still processing (e.g. waiting for preview)
          if (entry && entry.status === 'processing') {
             this.addProcessingEntry(entry.id);
             // We don't need to show an info toast here, the success message is enough,
             // but we MUST start polling so the preview appears automatically.
             this.pollForEntryStatus(dbName, entry.id);
          }

          this.triggerImageListRefresh();
        }
        
        // Case 2: Asynchronous Accepted (202)
        if (response.status === 202) {
          const partialEntry = response.body as PartialEntryResponse;
          this.addProcessingEntry(partialEntry.id);
          this.notificationService.showInfo(`Large file (ID: ${partialEntry.id}) is processing...`);
          this.pollForEntryStatus(dbName, partialEntry.id);
          this.triggerImageListRefresh();
        }
      }),
      map(() => void 0),
      catchError((err) => this.handleError(err))
    );
  }

  public getEntryMeta(dbName: string, entryId: number): Observable<Entry> {
    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http
      .get<Entry>(`${this.apiUrl}/entry/meta`, { params })
      .pipe(catchError((err) => this.handleError(err)));
  }

  public getEntryFileBlob(dbName: string, entryId: number): Observable<Blob> {
    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http.get(`${this.apiUrl}/entry/file`, {
        params,
        responseType: 'blob'
      })
      .pipe(catchError((err) => this.handleError(err)));
  }

  public getEntryPreviewUrl(dbName: string, entryId: number): string {
    return `${this.apiUrl}/entry/preview?database_name=${dbName}&id=${entryId}`;
  }

  public updateEntry(dbName: string, entryId: number, updates: Partial<Entry>): Observable<Entry> {
    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http.patch<Entry>(`${this.apiUrl}/entry`, updates, { params }).pipe(
      tap(() => {
        this.notificationService.showSuccess(`Entry ${entryId} updated successfully.`);
        this.triggerImageListRefresh();
      }),
      catchError(err => this.handleError(err))
    );
  }

  public deleteEntry(dbName: string, entryId: number): Observable<{ message: string }> {
    const params = new HttpParams()
      .set('database_name', dbName)
      .set('id', entryId.toString());

    return this.http.delete<{ message: string }>(`${this.apiUrl}/entry`, { params }).pipe(
      tap(res => {
        this.notificationService.showSuccess(res.message || `Entry ${entryId} deleted.`);
        this.triggerImageListRefresh();
      }),
      catchError(err => this.handleError(err))
    );
  }

  // --- HELPER METHODS FOR ASYNC UPLOAD ---

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
    // Poll every 2 seconds, starting after 2 seconds
    timer(2000, 2000).pipe(
      switchMap(() => this.getEntryMeta(dbName, entryId)),
      // Continue polling while status is 'processing'
      filter(entry => entry.status !== 'processing'),
      // Take the first non-processing status (ready or error)
      take(1),
      // Safety: Stop polling after 30 attempts (60 seconds)
      take(30),
      finalize(() => {
        // If we exit and the ID is still in our processing list, it likely timed out
        if (this.processingEntriesSubject.value.includes(entryId)) {
           this.removeProcessingEntry(entryId);
           // Optional: Notify user of timeout, but silent fail might be better for UX if it's just slow
           // this.notificationService.showError(`Polling for entry ${entryId} timed out.`);
        }
      })
    ).subscribe({
      next: entry => {
        this.removeProcessingEntry(entry.id);
        if (entry.status === 'ready') {
          // Success: Refresh the list so the spinner is replaced by the preview
          this.triggerImageListRefresh();
        } else if (entry.status === 'error') {
          this.notificationService.showError(`Entry ${entry.id} failed to process.`);
          this.triggerImageListRefresh();
        }
      },
      error: err => {
        this.removeProcessingEntry(entryId);
        console.error(`Error polling for entry ${entryId}:`, err);
      }
    });
  }
}