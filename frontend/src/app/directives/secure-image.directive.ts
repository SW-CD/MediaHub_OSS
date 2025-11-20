// frontend/src/app/directives/secure-image.directive.ts
import {
  Directive,
  ElementRef,
  Input,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  Renderer2
} from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Subscription, BehaviorSubject } from 'rxjs';
import { switchMap, filter, tap } from 'rxjs/operators';

/**
 * Directive to load images/media securely using the JwtInterceptor.
 * It handles Blob URL creation and REVOCATION to prevent memory leaks.
 *
 * Usage: <img [secureSrc]="'/api/entry/preview?...'">
 */
@Directive({
  selector: '[secureSrc]',
  standalone: true
})
export class SecureImageDirective implements OnChanges, OnDestroy {
  @Input() secureSrc: string | null = null;

  private currentUrlSubject = new BehaviorSubject<string | null>(null);
  private subscription: Subscription;
  private currentObjectUrl: string | null = null;

  constructor(
    private el: ElementRef,
    private http: HttpClient,
    private renderer: Renderer2
  ) {
    // Subscribe to URL changes
    this.subscription = this.currentUrlSubject
      .pipe(
        filter(url => !!url),
        switchMap(url => {
          // Set a loading state or placeholder if desired
          this.renderer.addClass(this.el.nativeElement, 'loading-image');
          
          // Fetch the blob. Interceptor adds Auth headers.
          return this.http.get(url!, { responseType: 'blob' });
        })
      )
      .subscribe({
        next: (blob) => {
          // Revoke previous URL before creating a new one
          this.revokeCurrentUrl();

          // Create new object URL
          this.currentObjectUrl = URL.createObjectURL(blob);
          
          // Set the src attribute of the host element
          this.renderer.setAttribute(this.el.nativeElement, 'src', this.currentObjectUrl);
          this.renderer.removeClass(this.el.nativeElement, 'loading-image');
        },
        error: (err) => {
          console.error('Error loading secure image:', err);
          this.renderer.removeClass(this.el.nativeElement, 'loading-image');
          // Optionally set an error placeholder
        }
      });
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['secureSrc']) {
      // If the input is cleared, clean up immediately
      if (!this.secureSrc) {
        this.revokeCurrentUrl();
        this.renderer.removeAttribute(this.el.nativeElement, 'src');
      }
      this.currentUrlSubject.next(this.secureSrc);
    }
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
    this.revokeCurrentUrl(); // Critical: Free memory when component/element is destroyed
  }

  private revokeCurrentUrl(): void {
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = null;
    }
  }
}