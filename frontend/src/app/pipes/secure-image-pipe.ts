// frontend/src/app/pipes/secure-image-pipe.ts
import { Pipe, PipeTransform } from '@angular/core';
import { HttpClient } from '@angular/common/http'; // Removed HttpHeaders
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { Observable, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';

@Pipe({
  name: 'secureImage',
  standalone: true 
})
export class SecureImagePipe implements PipeTransform {

  constructor(
    private http: HttpClient,
    private sanitizer: DomSanitizer
    // Removed AuthService injection as headers are now handled by the Interceptor
  ) {}

  transform(url: string | null | undefined): Observable<SafeUrl | null> {
    if (!url) {
      return of(null);
    }

    // The JwtInterceptor will automatically attach the Bearer token to this request.
    return this.http.get(url, { responseType: 'blob' }).pipe(
      map(response => {
        // Create a local blob URL from the image data
        const objectUrl = URL.createObjectURL(response);
        // Sanitize the URL before binding it to the <img> src
        return this.sanitizer.bypassSecurityTrustUrl(objectUrl);
      }),
      catchError(error => {
        console.error('Error loading secure image:', url, error);
        return of(null);
      })
    );
  }
}