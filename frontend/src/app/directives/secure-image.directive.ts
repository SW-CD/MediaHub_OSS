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
import { Subscription, BehaviorSubject, of } from 'rxjs';
import { switchMap, filter, catchError } from 'rxjs/operators';
import { ImageCacheService } from '../services/image-cache.service';

/**
 * Directive to load images/media securely using ImageCacheService and JwtInterceptor.
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
  private observer: IntersectionObserver | null = null;
  private isVisible = false;

  constructor(
    private el: ElementRef,
    private imageCacheService: ImageCacheService,
    private renderer: Renderer2
  ) {
    this.subscription = this.currentUrlSubject
      .pipe(
        filter((url): url is string => !!url && this.isVisible),
        switchMap(url => {
          this.renderer.addClass(this.el.nativeElement, 'loading-image');
          return this.imageCacheService.getBlobUrl(url).pipe(
            catchError(err => {
              console.error('Error loading secure image:', err);
              this.renderer.removeClass(this.el.nativeElement, 'loading-image');
              this.imageError.emit();
              return of(null);
            })
          );
        })
      )
      .subscribe({
        next: (blobUrl) => {
          if (blobUrl) {
            this.renderer.setAttribute(this.el.nativeElement, 'src', blobUrl);
            this.renderer.removeClass(this.el.nativeElement, 'loading-image');
          }
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
        this.renderer.removeAttribute(this.el.nativeElement, 'src');
      }
      
      if (this.isVisible) {
        this.currentUrlSubject.next(this.secureSrc);
      }
    }
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
    this.disconnectObserver();
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