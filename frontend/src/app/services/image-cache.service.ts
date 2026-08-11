// frontend/src/app/services/image-cache.service.ts
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { map, shareReplay, catchError, take } from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class ImageCacheService {
  private cache = new Map<string, Observable<string>>();

  constructor(private http: HttpClient) {}

  /**
   * Fetches an image blob from the given URL securely and returns a cached Object URL.
   * Multiple calls for the same URL share the exact same Observable and Blob URL.
   */
  public getBlobUrl(url: string): Observable<string> {
    if (this.cache.has(url)) {
      return this.cache.get(url)!;
    }

    const stream$ = this.http.get(url, {
      responseType: 'blob',
      headers: new HttpHeaders({ 'Accept': '*/*' })
    }).pipe(
      map(blob => URL.createObjectURL(blob)),
      shareReplay(1),
      catchError(err => {
        this.cache.delete(url);
        return throwError(() => err);
      })
    );

    this.cache.set(url, stream$);
    return stream$;
  }

  /**
   * Invalidates and revokes a single cached image URL.
   */
  public invalidate(url: string): void {
    const stream$ = this.cache.get(url);
    if (stream$) {
      stream$.pipe(take(1)).subscribe({
        next: blobUrl => {
          try { URL.revokeObjectURL(blobUrl); } catch (_) {}
        }
      });
      this.cache.delete(url);
    }
  }

  /**
   * Clears and revokes all cached Blob URLs in memory.
   */
  public clearAll(): void {
    this.cache.forEach(stream$ => {
      stream$.pipe(take(1)).subscribe({
        next: blobUrl => {
          try { URL.revokeObjectURL(blobUrl); } catch (_) {}
        }
      });
    });
    this.cache.clear();
  }
}
