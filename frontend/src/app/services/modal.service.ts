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
  private resultSubjects = new Map<string, Subject<boolean>>();

  constructor() {}

  /**
   * Opens a modal and returns an Observable that emits the result on close.
   * @param id The unique ID of the modal to open.
   * @param data Optional data to pass to the modal.
   * @returns An Observable that emits `true` for confirm, `false` for cancel/close.
   */
  open(id: string, data?: any): Observable<boolean> {
    const subject = new Subject<boolean>();
    this.resultSubjects.set(id, subject);
    this.modalEventSubject.next({ id, action: 'open', data });
    return subject.asObservable();
  }

  /**
   * Emits a result to the result subject without closing the modal.
   */
  emitResult(result: boolean, id?: string): void {
    if (id && this.resultSubjects.has(id)) {
      this.resultSubjects.get(id)!.next(result);
    } else if (this.resultSubjects.size > 0) {
      const lastKey = Array.from(this.resultSubjects.keys()).pop()!;
      this.resultSubjects.get(lastKey)!.next(result);
    }
  }

  /**
   * Closes the currently active modal and emits the result.
   * @param result The result to emit (true for confirm, false for cancel).
   * @param id Optional specific modal id to close.
   */
  close(result: boolean = false, id?: string): void {
    const targetId = id || Array.from(this.resultSubjects.keys()).pop();
    if (targetId && this.resultSubjects.has(targetId)) {
      const subject = this.resultSubjects.get(targetId)!;
      subject.next(result);
      subject.complete();
      this.resultSubjects.delete(targetId);
    }
    // Broadcast a general close event for the modal component to hide itself.
    this.modalEventSubject.next({ id: targetId || '', action: 'close' });
  }

  /**
   * Allows a modal component to subscribe to its specific open/close events.
   */
  getModalEvents(id: string): Observable<ModalEvent> {
    return this.modalEventSubject.asObservable().pipe(
      filter(event => event.id === id || (!event.id && event.action === 'close'))
    );
  }
}

