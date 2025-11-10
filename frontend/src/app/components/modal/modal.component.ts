import { Component, Input, OnDestroy, OnInit } from '@angular/core';
import { Subscription } from 'rxjs';
import { ModalService, ModalEvent } from '../../services/modal.service';

@Component({
  selector: 'app-modal',
  templateUrl: './modal.component.html',
  styleUrls: ['./modal.component.css'],
  standalone: false,
})
export class ModalComponent implements OnInit, OnDestroy {
  @Input() modalId: string = '';
  @Input() modalTitle: string = 'Modal';
  isOpen = false;
  private modalSubscription!: Subscription;

  constructor(private modalService: ModalService) {}

  ngOnInit(): void {
    this.modalSubscription = this.modalService.getModalEvents(this.modalId).subscribe(
      (event: ModalEvent) => {
        if (event.action === 'open') {
          this.isOpen = true;
        } else if (event.action === 'close') {
          this.isOpen = false;
        }
      }
    );
  }

  closeModal(): void {
    this.modalService.close();
  }

  onContentClick(event: MouseEvent): void {
    event.stopPropagation();
  }

  ngOnDestroy(): void {
    this.modalSubscription.unsubscribe();
  }
}