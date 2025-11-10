// frontend/src/app/components/user-form/user-form.component.ts
import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject, Subscription } from 'rxjs';
import { takeUntil, finalize } from 'rxjs/operators';
import { User } from '../../models/api.models';
import { AuthService } from '../../services/auth.service';
import { ModalService, ModalEvent } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';

@Component({
  selector: 'app-user-form',
  templateUrl: './user-form.component.html',
  styleUrls: ['./user-form.component.css'],
  standalone: false,
})
export class UserFormComponent implements OnInit, OnDestroy {
  static readonly MODAL_ID = 'userFormModal';

  public userForm: FormGroup;
  public isEditMode = false;
  public isLoading = false;
  private userIdToEdit: number | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private modalService: ModalService,
    private notificationService: NotificationService
  ) {
    this.userForm = this.fb.group({
      username: ['', [Validators.required, Validators.pattern(/^[a-zA-Z0-9_]+$/)]],
      password: ['', [Validators.minLength(8)]], // Required only for create mode, handled dynamically
      can_view: [true],
      can_create: [false],
      can_edit: [false],
      can_delete: [false],
      is_admin: [false],
    });
  }

  ngOnInit(): void {
    this.modalService.getModalEvents(UserFormComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe((event: ModalEvent) => {
        if (event.action === 'open') {
          this.setupFormForMode(event.data);
        }
      });
  }

  private setupFormForMode(data: any): void {
    this.userForm.reset();
    this.isEditMode = data?.isEditMode || false;
    this.userIdToEdit = null;

    if (this.isEditMode && data.user) {
      this.userIdToEdit = data.user.id;
      this.userForm.patchValue(data.user);
      this.userForm.get('password')?.setValidators([Validators.minLength(8)]); // Make password optional for edit
    } else {
      // Create mode
      this.userForm.patchValue({ can_view: true }); // Default role
      this.userForm.get('password')?.setValidators([Validators.required, Validators.minLength(8)]);
    }
    this.userForm.get('password')?.updateValueAndValidity();
  }

  onSubmit(): void {
    if (this.userForm.invalid) {
      this.userForm.markAllAsTouched();
      return;
    }

    this.isLoading = true;
    const formData = this.userForm.value;

    // Don't send an empty password string on update
    if (this.isEditMode && !formData.password) {
      delete formData.password;
    }

    const apiCall$ = this.isEditMode
      ? this.authService.updateUser(this.userIdToEdit!, formData)
      : this.authService.createUser(formData);

    apiCall$.pipe(finalize(() => this.isLoading = false))
      .subscribe({
        next: () => {
          const action = this.isEditMode ? 'updated' : 'created';
          this.notificationService.showSuccess(`User ${action} successfully!`);
          this.closeModal(true); // Signal success
        },
        // Errors are handled by the global error handler in the service
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

