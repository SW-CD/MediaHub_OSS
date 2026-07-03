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
  Renderer2,
  HostListener
} from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Subscription, BehaviorSubject } from 'rxjs';
import { switchMap, filter } from 'rxjs/operators';

/**
 * Directive to load images/media securely using the JwtInterceptor.
 * It handles Blob URL creation and REVOCATION to prevent memory leaks.
 * Usage: <img [secureSrc]="'/api/entry/preview?...'" (imageError)="handleError()">
 * Optimized to load images lazily via IntersectionObserver when entering the viewport.
 */
@Directive({
  selector: '[secureSrc]',
  standalone: true
})
export class SecureImageDirective implements OnChanges, OnDestroy {
  @Input() secureSrc: string | null = null;
  @Output() imageError = new EventEmitter<void>();
  @Output() aspectLoaded = new EventEmitter<number>();

  private currentUrlSubject = new BehaviorSubject<string | null>(null);
  private subscription: Subscription;
  private currentObjectUrl: string | null = null;
  private observer: IntersectionObserver | null = null;
  private isVisible = false;

  constructor(
    private el: ElementRef,
    private http: HttpClient,
    private renderer: Renderer2
  ) {
    this.subscription = this.currentUrlSubject
      .pipe(
        // Only load if a URL is provided AND the element is visible in the viewport
        filter(url => !!url && this.isVisible),
        switchMap(url => {
          this.renderer.addClass(this.el.nativeElement, 'loading-image');
          
          // UPDATED: Explicitly request */* to ensure we get a binary Blob, not JSON Base64
          return this.http.get(url!, { 
            responseType: 'blob',
            headers: new HttpHeaders({ 'Accept': '*/*' }) 
          });
        })
      )
      .subscribe({
        next: (blob) => {
          this.revokeCurrentUrl();
          this.currentObjectUrl = URL.createObjectURL(blob);
          this.renderer.setAttribute(this.el.nativeElement, 'src', this.currentObjectUrl);
          this.renderer.removeClass(this.el.nativeElement, 'loading-image');
        },
        error: (err) => {
          console.error('Error loading secure image:', err);
          this.renderer.removeClass(this.el.nativeElement, 'loading-image');
          this.imageError.emit();
        }
      });

    this.setupIntersectionObserver();
  }

  private setupIntersectionObserver(): void {
    if (typeof IntersectionObserver !== 'undefined') {
      this.observer = new IntersectionObserver(
        (entries) => {
          const entry = entries[0];
          if (entry && entry.isIntersecting) {
            this.isVisible = true;
            // Trigger the behavior subject to load the current URL
            if (this.secureSrc) {
              this.currentUrlSubject.next(this.secureSrc);
            }
            this.disconnectObserver();
          }
        },
        {
          rootMargin: '200px' // Start loading 200px before entering viewport
        }
      );
      this.observer.observe(this.el.nativeElement);
    } else {
      // Fallback for environments/browsers without IntersectionObserver
      this.isVisible = true;
    }
  }

  private disconnectObserver(): void {
    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['secureSrc']) {
      if (!this.secureSrc) {
        this.revokeCurrentUrl();
        this.renderer.removeAttribute(this.el.nativeElement, 'src');
      }
      
      // If already visible, load immediately. Otherwise, observer will trigger it.
      if (this.isVisible) {
        this.currentUrlSubject.next(this.secureSrc);
      }
    }
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
    this.revokeCurrentUrl();
    this.disconnectObserver();
  }

  private revokeCurrentUrl(): void {
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = null;
    }
  }

  @HostListener('load')
  onLoad(): void {
    const img = this.el.nativeElement as HTMLImageElement;
    if (img && img.naturalWidth && img.naturalHeight) {
      const ar = img.naturalWidth / img.naturalHeight;
      this.aspectLoaded.emit(ar);
    }
  }
}