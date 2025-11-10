// frontend/src/app/services/notification.service.ts

import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

/**
 * Defines the structure for a notification message.
 */
export interface Notification {
  message: string;
  type: 'success' | 'error' | 'info';
}

/**
 * Manages the display of global error messages and success toast notifications.
 * This service provides observables for components to subscribe to.
 */
@Injectable({
  providedIn: 'root',
})
export class NotificationService {
  /**
   * BehaviorSubject for critical, global error banners.
   * e.g., "Server is down."
   */
  private globalErrorSubject = new BehaviorSubject<string | null>(null);

  /**
   * BehaviorSubject for temporary "toast" notifications.
   * e.g., "Image uploaded successfully."
   */
  private toastNotificationSubject = new BehaviorSubject<Notification | null>(
    null
  );

  /**
   * Observable for components to listen for global errors.
   */
  public globalError$: Observable<string | null> =
    this.globalErrorSubject.asObservable();

  /**
   * Observable for components to listen for toast notifications.
   */
  public toastNotification$: Observable<Notification | null> =
    this.toastNotificationSubject.asObservable();

  constructor() {}

  /**
   * Shows a global error banner.
   * @param message The error message to display.
   */
  showGlobalError(message: string): void {
    this.globalErrorSubject.next(message);
  }

  /**
   * Clears the global error banner.
   */
  clearGlobalError(): void {
    this.globalErrorSubject.next(null);
  }

  /**
   * Shows a success toast notification.
   * @param message The success message to display.
   */
  showSuccess(message: string): void {
    this.toastNotificationSubject.next({ message, type: 'success' });
  }

  /**
   * Shows an error toast notification.
   * @param message The error message to display.
   */
  showError(message: string): void {
    this.toastNotificationSubject.next({ message, type: 'error' });
  }

  /**
   * Shows an info toast notification.
   * @param message The info message to display.
   */
  showInfo(message: string): void {
    this.toastNotificationSubject.next({ message, type: 'info' });
  }
}