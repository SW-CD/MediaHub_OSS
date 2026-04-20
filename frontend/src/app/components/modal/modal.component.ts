import { Component, Input, OnDestroy, OnInit, HostListener, ChangeDetectorRef, ChangeDetectionStrategy } from '@angular/core';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { ModalService, ModalEvent } from '../../services/modal.service';

@Component({
  selector: 'app-modal',
  templateUrl: './modal.component.html',
  styleUrls: ['./modal.component.css'],
  standalone: false,
  changeDetection: ChangeDetectionStrategy.OnPush // Optimization for better performance
})
export class ModalComponent implements OnInit, OnDestroy {
  @Input() modalId: string = '';
  @Input() modalTitle: string = 'Modal';
  
  public isOpen = false;
  private destroy$ = new Subject<void>();

  constructor(
    private modalService: ModalService,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.modalService.getModalEvents(this.modalId)
      .pipe(takeUntil(this.destroy$))
      .subscribe((event: ModalEvent) => {
        if (event.action === 'open') {
          this.isOpen = true;
          document.body.classList.add('modal-open'); // Prevent background scrolling
        } else if (event.action === 'close') {
          this.isOpen = false;
          document.body.classList.remove('modal-open'); // Restore background scrolling
        }
        
        // Inform Angular to update the view since we are using OnPush
        this.cdr.markForCheck(); 
      });
  }

  // UX Improvement: Close modal on 'Escape' key press
  @HostListener('document:keydown.escape')
  onEscKeydown(): void {
    if (this.isOpen) {
      this.closeModal();
    }
  }

  closeModal(): void {
    // Calling the service will emit the 'close' event, updating the state cleanly
    this.modalService.close();
  }

  onContentClick(event: MouseEvent): void {
    // Prevent clicks inside the modal content area from bubbling up and closing the overlay
    event.stopPropagation();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    // Safety cleanup just in case the component is destroyed while open
    document.body.classList.remove('modal-open'); 
  }
}