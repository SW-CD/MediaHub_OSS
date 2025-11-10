// frontend/src/app/components/confirmation-modal/confirmation-modal.component.ts

import { Component, OnDestroy, OnInit } from '@angular/core';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { ModalService, ModalEvent } from '../../services/modal.service';

export interface ConfirmationModalData {
  title?: string;
  message: string;
}

@Component({
  selector: 'app-confirmation-modal',
  templateUrl: './confirmation-modal.component.html',
  styleUrls: ['./confirmation-modal.component.css'],
  standalone: false,
})
export class ConfirmationModalComponent implements OnInit, OnDestroy {
  public static readonly MODAL_ID = 'confirmationModal';

  modalData: ConfirmationModalData = { message: 'Are you sure?' };
  private destroy$ = new Subject<void>();

  constructor(private modalService: ModalService) {}

  ngOnInit(): void {
    this.modalService.getModalEvents(ConfirmationModalComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe((event: ModalEvent) => {
        if (event.action === 'open' && event.data) {
          this.modalData = {
            title: event.data.title || 'Confirm Action',
            message: event.data.message || 'Are you sure?'
          };
        }
      });
  }

  onConfirm(): void {
    this.modalService.close(true); // Close with a `true` result
  }

  onCancel(): void {
    this.modalService.close(false); // Close with a `false` result
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}

