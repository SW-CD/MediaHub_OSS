// frontend/src/app/services/app-info.service.ts
import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, of } from 'rxjs';
import { tap, catchError } from 'rxjs/operators';
import { AppInfo } from '../models/api.models';

@Injectable({
  providedIn: 'root'
})
export class AppInfoService {
  private readonly apiUrl = '/api/info';
  
  // Cache the info response
  private infoSubject = new BehaviorSubject<AppInfo | null>(null);
  public info$ = this.infoSubject.asObservable();

  constructor(private http: HttpClient) { }

  /**
   * Loads the /api/info data.
   * Only fetches from the network if it hasn't been fetched already.
   */
  loadInfo(): void {
    // If we already have the info, don't fetch again
    if (this.infoSubject.value) {
      return;
    }

    this.http.get<AppInfo>(this.apiUrl).pipe(
      tap(info => {
        this.infoSubject.next(info);
      }),
      catchError(err => {
        console.error("Failed to load app info:", err);
        // We can't log out here as this is unauthenticated
        // Just return null
        this.infoSubject.next(null);
        return of(null);
      })
    ).subscribe();
  }
}
