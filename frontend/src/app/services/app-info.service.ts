import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, of } from 'rxjs';
import { tap, catchError } from 'rxjs/operators';
import { AppInfo } from '../models';

@Injectable({
  providedIn: 'root'
})
export class AppInfoService {
  private readonly apiUrl = 'api/info';
  
  private infoSubject = new BehaviorSubject<AppInfo | null>(null);
  public info$ = this.infoSubject.asObservable();

  constructor(private http: HttpClient) { }

  loadInfo(): Observable<AppInfo | null> {
    if (this.infoSubject.value) {
      return of(this.infoSubject.value);
    }

    return this.http.get<AppInfo>(this.apiUrl).pipe(
      tap(info => {
        this.infoSubject.next(info);
      }),
      catchError(err => {
        console.error("Failed to load app info:", err);
        this.infoSubject.next(null);
        return of(null);
      })
    );
  }
}