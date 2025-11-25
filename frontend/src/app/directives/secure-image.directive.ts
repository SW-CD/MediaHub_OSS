// frontend/src/app/directives/secure-image.directive.ts
import {
  Directive,
  ElementRef,
  Input,
  Output,
  EventEmitter,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  Renderer2
} from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Subscription, BehaviorSubject } from 'rxjs';
import { switchMap, filter } from 'rxjs/operators';

/**
 * Directive to load images/media securely using the JwtInterceptor.
 * It handles Blob URL creation and REVOCATION to prevent memory leaks.
 * * It also emits an error event if the image fails to load (e.g. 404),
 * allowing the parent component to show a fallback.
 *
 * Usage: <img [secureSrc]="'/api/entry/preview?...'" (imageError)="handleError()">
 */
@Directive({
  selector: '[secureSrc]',
  standalone: true
})
export class SecureImageDirective implements OnChanges, OnDestroy {
  @Input() secureSrc: string | null = null;
  @Output() imageError = new EventEmitter<void>(); // <-- Emits when loading fails

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
          // Set a loading state
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
          
          // Emit the error so the parent can handle it (e.g., show placeholder)
          this.imageError.emit();
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