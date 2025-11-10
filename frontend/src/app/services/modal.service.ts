// frontend/src/app/services/modal.service.ts

import { Injectable } from '@angular/core';
import { Subject, Observable } from 'rxjs';
import { filter } from 'rxjs/operators';

export interface ModalEvent {
  id: string;
  action: 'open' | 'close';
  data?: any;
}

@Injectable({
  providedIn: 'root',
})
export class ModalService {
  private modalEventSubject = new Subject<ModalEvent>();
  private resultSubject!: Subject<boolean>;

  constructor() {}

  /**
   * Opens a modal and returns an Observable that emits the result on close.
   * @param id The unique ID of the modal to open.
   * @param data Optional data to pass to the modal.
   * @returns An Observable that emits `true` for confirm, `false` for cancel/close.
   */
  open(id: string, data?: any): Observable<boolean> {
    this.resultSubject = new Subject<boolean>();
    this.modalEventSubject.next({ id, action: 'open', data });
    return this.resultSubject.asObservable();
  }

  /**
   * Closes the currently active modal and emits the result.
   * @param result The result to emit (true for confirm, false for cancel).
   */
  close(result: boolean = false): void {
    if (this.resultSubject) {
      this.resultSubject.next(result);
      this.resultSubject.complete();
    }
    // Broadcast a general close event for the modal component to hide itself.
    this.modalEventSubject.next({ id: '', action: 'close' });
  }

  /**
   * Allows a modal component to subscribe to its specific open/close events.
   */
  getModalEvents(id: string): Observable<ModalEvent> {
    return this.modalEventSubject.asObservable().pipe(
      filter(event => event.action === 'close' || event.id === id)
    );
  }
}

