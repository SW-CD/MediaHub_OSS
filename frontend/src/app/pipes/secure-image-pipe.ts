// frontend/src/app/pipes/secure-image.pipe.ts
import { Pipe, PipeTransform } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http'; // <-- Import HttpHeaders
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { Observable, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';
import { AuthService } from '../services/auth.service'; // <-- IMPORT AUTH SERVICE

@Pipe({
  name: 'secureImage',
  standalone: true 
})
export class SecureImagePipe implements PipeTransform {

  constructor(
    private http: HttpClient,
    private sanitizer: DomSanitizer,
    private authService: AuthService // <-- INJECT AUTH SERVICE
  ) {}

  transform(url: string | null | undefined): Observable<SafeUrl | null> {
    if (!url) {
      return of(null);
    }

    // --- FIX: Create authentication headers ---
    const token = this.authService.getAuthToken();
    if (!token) {
      // If no token, don't even try. This prevents errors on logout.
      return of(null); 
    }
    const headers = new HttpHeaders({ Authorization: token });
    // --- END FIX ---


    // Get the image data as a Blob, now WITH auth headers
    return this.http.get(url, { headers, responseType: 'blob' }).pipe( // <-- PASS HEADERS
      map(response => {
        // Create a local blob URL from the image data
        const objectUrl = URL.createObjectURL(response);
        // Sanitize the URL before binding it to the <img> src
        return this.sanitizer.bypassSecurityTrustUrl(objectUrl);
      }),
      catchError(error => {
        // On error (e.g., 404, 500), just return null.
        console.error('Error loading secure image:', url, error);
        return of(null);
      })
    );
  }
}