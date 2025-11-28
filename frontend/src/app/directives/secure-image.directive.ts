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
import { HttpClient, HttpHeaders } from '@angular/common/http'; // <-- Added HttpHeaders
import { Subscription, BehaviorSubject } from 'rxjs';
import { switchMap, filter } from 'rxjs/operators';

/**
 * Directive to load images/media securely using the JwtInterceptor.
 * It handles Blob URL creation and REVOCATION to prevent memory leaks.
 * Usage: <img [secureSrc]="'/api/entry/preview?...'" (imageError)="handleError()">
 */
@Directive({
  selector: '[secureSrc]',
  standalone: true
})
export class SecureImageDirective implements OnChanges, OnDestroy {
  @Input() secureSrc: string | null = null;
  @Output() imageError = new EventEmitter<void>();

  private currentUrlSubject = new BehaviorSubject<string | null>(null);
  private subscription: Subscription;
  private currentObjectUrl: string | null = null;

  constructor(
    private el: ElementRef,
    private http: HttpClient,
    private renderer: Renderer2
  ) {
    this.subscription = this.currentUrlSubject
      .pipe(
        filter(url => !!url),
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
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['secureSrc']) {
      if (!this.secureSrc) {
        this.revokeCurrentUrl();
        this.renderer.removeAttribute(this.el.nativeElement, 'src');
      }
      this.currentUrlSubject.next(this.secureSrc);
    }
  }

  ngOnDestroy(): void {
    this.subscription.unsubscribe();
    this.revokeCurrentUrl();
  }

  private revokeCurrentUrl(): void {
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = null;
    }
  }
}