// frontend/src/app/services/entry.service.ts

import { Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { BehaviorSubject, Observable, Subject, throwError, timer } from 'rxjs';
import { catchError, tap, map, switchMap, filter, take, finalize } from 'rxjs/operators';
import { Entry, SearchRequest, PartialEntryResponse } from '../models';
import { NotificationService } from './notification.service';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root',
})
export class EntryService {
  private readonly apiUrl = 'api';

  // State specific to entries
  private selectedEntrySubject = new BehaviorSubject<Entry | null>(null);
  private refreshNotifier = new Subject<void>();
  private processingEntriesSubject = new BehaviorSubject<number[]>([]);

  public selectedEntry$ = this.selectedEntrySubject.asObservable();
  public refreshRequired$ = this.refreshNotifier.asObservable();
  public processingEntries$ = this.processingEntriesSubject.asObservable();

  constructor(
    private http: HttpClient,
    private notificationService: NotificationService,
    private authService: AuthService
  ) {}

  public triggerImageListRefresh(): void {
    this.refreshNotifier.next();
  }

  /**
   * Centralized error handler for HTTP requests.
   */
  private handleError(error: HttpErrorResponse): Observable<never> {
    console.error("[DEBUG] EntryService: Full HTTP Error Response:", error);

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

  // --- ENTRIES (BULK) ENDPOINTS ---

  public searchEntries(dbName: string, payload: SearchRequest): Observable<Entry[]> {
    return this.http
      .post<Entry[]>(`${this.apiUrl}/database/${dbName}/entries/search`, payload)
      .pipe(catchError((err) => this.handleError(err)));
  }

  public bulkDeleteEntries(dbName: string, ids: number[]): Observable<any> {
    return this.http.post(`${this.apiUrl}/database/${dbName}/entries/delete`, { ids }).pipe(
      tap(() => {
        this.notificationService.showSuccess(`Successfully deleted ${ids.length} entries.`);
        this.triggerImageListRefresh();
      }),
      catchError((err) => this.handleError(err))
    );
  }

  public bulkExportEntries(dbName: string, ids: number[]): Observable<Blob> {
    return this.http.post(`${this.apiUrl}/database/${dbName}/entries/export`, { ids }, { 
      responseType: 'blob' 
    }).pipe(
      catchError((err) => this.handleError(err))
    );
  }

  // --- ENTRY (SINGLE) ENDPOINTS ---

  public selectEntry(entry: Entry): void {
    this.selectedEntrySubject.next(entry);
  }

  public clearSelectedEntry(): void {
    this.selectedEntrySubject.next(null);
  }

  /**
   * Uploads a file with associated metadata.
   * Sends both payload and binary using multipart/form-data.
   */
  public uploadEntry(dbName: string, metadata: Omit<Entry, 'id' | 'width' | 'height' | 'filesize' | 'mime_type' | 'status'>, file: File): Observable<void> {
    const formData = new FormData();
    
    formData.append('metadata', JSON.stringify(metadata)); 
    formData.append('file', file, file.name);

    return this.http.post<Entry | PartialEntryResponse>(`${this.apiUrl}/database/${dbName}/entry`, formData, { 
      observe: 'response'
    }).pipe(
      tap(response => {
        if (response.status === 201) {
          const entry = response.body as Entry;
          this.notificationService.showSuccess('Entry uploaded successfully.');
          
          if (entry && entry.status === 'processing') {
             this.addProcessingEntry(entry.id);
             this.pollForEntryStatus(dbName, entry.id);
          }
          this.triggerImageListRefresh();
        }
        
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
    return this.http
      .get<Entry>(`${this.apiUrl}/database/${dbName}/entry/${entryId}`)
      .pipe(catchError((err) => this.handleError(err)));
  }

  public getEntryFileBlob(dbName: string, entryId: number): Observable<Blob> {
    return this.http.get(`${this.apiUrl}/database/${dbName}/entry/${entryId}/file`, {
        responseType: 'blob',
        headers: new HttpHeaders({ 'Accept': '*/*' })
      })
      .pipe(catchError((err) => this.handleError(err)));
  }

  /**
   * Generates a direct URL to the file endpoint, bypassing Angular's HttpClient.
   * This is crucial for <video> and <audio> tags so the browser can utilize 
   * HTTP Range requests (streaming) instead of downloading the entire file into memory.
   */
  public getEntryFileUrl(dbName: string, entryId: number): string {
    const baseUrl = `${this.apiUrl}/database/${dbName}/entry/${entryId}/file`;
    const token = this.authService.getAccessToken(); 
    
    // Append the token as a query parameter so the browser's native media engine can authenticate
    if (token) {
      return `${baseUrl}?token=${encodeURIComponent(token)}`;
    }
    
    return baseUrl;
  }

  public getEntryPreviewUrl(dbName: string, entryId: number): string {
    return `${this.apiUrl}/database/${dbName}/entry/${entryId}/preview`;
  }

  public updateEntry(dbName: string, entryId: number, updates: Partial<Entry>): Observable<Entry> {
    return this.http.patch<Entry>(`${this.apiUrl}/database/${dbName}/entry/${entryId}`, updates).pipe(
      tap(() => {
        this.notificationService.showSuccess(`Entry ${entryId} updated successfully.`);
        this.triggerImageListRefresh();
      }),
      catchError(err => this.handleError(err))
    );
  }

  public deleteEntry(dbName: string, entryId: number): Observable<{ message: string }> {
    return this.http.delete<{ message: string }>(`${this.apiUrl}/database/${dbName}/entry/${entryId}`).pipe(
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
    timer(2000, 2000).pipe(
      switchMap(() => this.getEntryMeta(dbName, entryId)),
      filter(entry => entry.status !== 'processing'),
      take(1),
      take(30),
      finalize(() => {
        if (this.processingEntriesSubject.value.includes(entryId)) {
           this.removeProcessingEntry(entryId);
        }
      })
    ).subscribe({
      next: entry => {
        this.removeProcessingEntry(entry.id);
        if (entry.status === 'ready') {
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