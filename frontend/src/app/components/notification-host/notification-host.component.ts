import { Component, OnDestroy } from '@angular/core';
import { Notification, NotificationService } from '../../services/notification.service';
import { Subscription } from 'rxjs';
import { trigger, transition, style, animate } from '@angular/animations';

@Component({
  selector: 'app-notification-host',
  templateUrl: './notification-host.component.html',
  styleUrls: ['./notification-host.component.css'],
  standalone: false,
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
export class NotificationHostComponent implements OnDestroy {
  globalError: string | null = null;
  toast: Notification | null = null;
  private subscriptions = new Subscription();
  private toastTimer: any;

  constructor(private notificationService: NotificationService) {
    this.subscriptions.add(this.notificationService.globalError$.subscribe(message => this.globalError = message));
    this.subscriptions.add(this.notificationService.toastNotification$.subscribe(notification => this.showToast(notification)));
  }

  private showToast(notification: Notification | null): void {
    if (this.toastTimer) clearTimeout(this.toastTimer);
    this.toast = notification;
    if (notification) {
      this.toastTimer = setTimeout(() => this.toast = null, 4000);
    }
  }

  clearGlobalError(): void { this.notificationService.clearGlobalError(); }
  clearToast(): void { if (this.toastTimer) clearTimeout(this.toastTimer); this.toast = null; }
  ngOnDestroy(): void { this.subscriptions.unsubscribe(); if (this.toastTimer) clearTimeout(this.toastTimer); }
}
