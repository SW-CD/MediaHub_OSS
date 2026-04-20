import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, finalize } from 'rxjs/operators';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-change-password-modal',
  templateUrl: './change-password-modal.component.html',
  styleUrls: ['./change-password-modal.component.css'],
  standalone: false,
})
export class ChangePasswordModalComponent implements OnInit, OnDestroy {
  public static readonly MODAL_ID = 'changePasswordModal';
  public passwordForm: FormGroup;
  public isLoading = false;

  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private modalService: ModalService,
    private notificationService: NotificationService
  ) {
    // Added the oldPassword field to the form validation
    this.passwordForm = this.fb.group({
      oldPassword: ['', [Validators.required]],
      newPassword: ['', [Validators.required, Validators.minLength(8)]],
    });
  }

  ngOnInit(): void {
    this.modalService.getModalEvents(ChangePasswordModalComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe(event => {
        if (event.action === 'open') {
          this.passwordForm.reset();
          this.isLoading = false;
        }
      });
  }

  onSubmit(): void {
    if (this.passwordForm.invalid) {
      this.passwordForm.markAllAsTouched();
      return;
    }

    this.isLoading = true;
    const oldPassword = this.passwordForm.value.oldPassword;
    const newPassword = this.passwordForm.value.newPassword;

    // Call the updated service method with both passwords
    this.authService.changeOwnPassword(oldPassword, newPassword).pipe(
      finalize(() => this.isLoading = false)
    ).subscribe({
      next: () => {
        this.notificationService.showSuccess('Password changed successfully!');
        this.closeModal(true);
      },
      error: (err: HttpErrorResponse) => {
        if (err.status === 401) {
           this.notificationService.showError('Incorrect current password.');
        }
        console.error('Password change failed:', err);
      },
    });
  }

  closeModal(result: boolean = false): void {
    this.modalService.close(result);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}