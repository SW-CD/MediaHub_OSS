import { Component, OnDestroy, OnInit, ChangeDetectorRef, ChangeDetectionStrategy } from '@angular/core';
import { Notification, NotificationService } from '../../services/notification.service';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { trigger, transition, style, animate } from '@angular/animations';

@Component({
  selector: 'app-notification-host',
  templateUrl: './notification-host.component.html',
  styleUrls: ['./notification-host.component.css'],
  standalone: false,
  changeDetection: ChangeDetectionStrategy.OnPush, // NEW: Huge performance boost for a root-level component
  animations: [
    trigger('toastAnimation', [
      transition(':enter', [
        style({ transform: 'translateY(100%)', opacity: 0 }),
        animate('300ms ease-out', style({ transform: 'translateY(0)', opacity: 1 })),
      ]),
      transition(':leave', [
        animate('300ms ease-in', style({ transform: 'translateY(100%)', opacity: 0 })),
      ]),
    ]),
  ],
})
export class NotificationHostComponent implements OnInit, OnDestroy {
  public globalError: string | null = null;
  public toast: Notification | null = null;
  
  private destroy$ = new Subject<void>();
  private toastTimer?: ReturnType<typeof setTimeout>; // UPDATED: Strict typing instead of 'any'

  constructor(
    private notificationService: NotificationService,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.notificationService.globalError$
      .pipe(takeUntil(this.destroy$))
      .subscribe(message => {
        this.globalError = message;
        this.cdr.markForCheck(); // Inform Angular to update the view
      });

    this.notificationService.toastNotification$
      .pipe(takeUntil(this.destroy$))
      .subscribe(notification => {
        this.showToast(notification);
      });
  }

  private showToast(notification: Notification | null): void {
    if (this.toastTimer) {
      clearTimeout(this.toastTimer);
    }
    
    this.toast = notification;
    this.cdr.markForCheck(); // Inform Angular to update the view

    if (notification) {
      this.toastTimer = setTimeout(() => {
        this.toast = null;
        this.cdr.markForCheck(); // Inform Angular to update the view when hiding
      }, 4000);
    }
  }

  clearGlobalError(): void { 
    this.notificationService.clearGlobalError(); 
    this.globalError = null;
    this.cdr.markForCheck();
  }
  
  clearToast(): void { 
    if (this.toastTimer) {
      clearTimeout(this.toastTimer); 
    }
    this.toast = null; 
    this.cdr.markForCheck();
  }
  
  ngOnDestroy(): void { 
    this.destroy$.next();
    this.destroy$.complete();
    if (this.toastTimer) {
      clearTimeout(this.toastTimer); 
    }
  }
}