// frontend/src/app/components/api-key-modal/api-key-modal.component.ts

import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, finalize } from 'rxjs/operators';
import { ApiKey } from '../../models';
import { AuthService } from '../../services/auth.service';
import { ModalService, ModalEvent } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';

@Component({
  selector: 'app-api-key-modal',
  templateUrl: './api-key-modal.component.html',
  styleUrls: ['./api-key-modal.component.css'],
  standalone: false
})
export class ApiKeyModalComponent implements OnInit, OnDestroy {
  public static readonly MODAL_ID = 'apiKeyModal';

  public keyForm: FormGroup;
  public isEditMode = false;
  public isLoading = false;
  public userId: string | null = null;
  public keyIdToEdit: string | null = null;

  // Token Reveal State
  public plaintextToken: string | null = null;
  public isTokenSavedConfirmed = false;
  public tokenCopied = false;

  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private modalService: ModalService,
    private notificationService: NotificationService
  ) {
    this.keyForm = this.fb.group({
      name: ['', [Validators.required, Validators.maxLength(64)]],
      expires_at: [''],
      scope_view: [false],
      scope_create: [false],
      scope_edit: [false],
      scope_delete: [false],
      scope_admin: [false]
    });
  }

  ngOnInit(): void {
    this.modalService.getModalEvents(ApiKeyModalComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe((event: ModalEvent) => {
        if (event.action === 'open') {
          this.setupForm(event.data);
        }
      });
  }

  private setupForm(data: any): void {
    this.keyForm.reset({
      name: '',
      expires_at: '',
      scope_view: false,
      scope_create: false,
      scope_edit: false,
      scope_delete: false,
      scope_admin: false
    });
    
    this.userId = data?.userId || null;
    this.isEditMode = !!data?.apiKey;
    this.keyIdToEdit = data?.apiKey?.id || null;
    this.plaintextToken = null;
    this.isTokenSavedConfirmed = false;
    this.tokenCopied = false;

    if (this.isEditMode && data.apiKey) {
      const key: ApiKey = data.apiKey;
      let expiryString = '';
      if (key.expires_at) {
        // Convert timestamp (ms) to YYYY-MM-DDThh:mm format for datetime-local
        const date = new Date(key.expires_at);
        const pad = (n: number) => n.toString().padStart(2, '0');
        expiryString = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
      }

      this.keyForm.patchValue({
        name: key.name,
        expires_at: expiryString,
        scope_view: key.scope_view,
        scope_create: key.scope_create,
        scope_edit: key.scope_edit,
        scope_delete: key.scope_delete,
        scope_admin: key.scope_admin
      });
    }
  }

  public copyToClipboard(): void {
    if (!this.plaintextToken) return;
    
    navigator.clipboard.writeText(this.plaintextToken).then(() => {
      this.tokenCopied = true;
      this.notificationService.showSuccess('Plaintext token copied to clipboard!');
      setTimeout(() => this.tokenCopied = false, 3000);
    }).catch(err => {
      console.error('Failed to copy token to clipboard:', err);
      this.notificationService.showError('Could not copy automatically. Please select and copy manually.');
    });
  }

  public onSubmit(): void {
    if (this.keyForm.invalid || !this.userId) {
      this.keyForm.markAllAsTouched();
      return;
    }

    this.isLoading = true;
    const formVal = this.keyForm.value;
    
    // Process expires_at to ms timestamp if set
    let expiresAtMs: number | null = null;
    if (formVal.expires_at) {
      expiresAtMs = new Date(formVal.expires_at).getTime();
    }

    const payload = {
      name: formVal.name,
      expires_at: expiresAtMs,
      scope_view: !!formVal.scope_view,
      scope_create: !!formVal.scope_create,
      scope_edit: !!formVal.scope_edit,
      scope_delete: !!formVal.scope_delete,
      scope_admin: !!formVal.scope_admin
    };

    if (this.isEditMode && this.keyIdToEdit) {
      this.authService.updateUserKey(this.userId, this.keyIdToEdit, payload)
        .pipe(finalize(() => this.isLoading = false))
        .subscribe({
          next: () => {
            this.notificationService.showSuccess('API key updated successfully!');
            this.modalService.close(true);
          },
          error: (err) => {
            console.error('Failed to update API key', err);
            this.notificationService.showError('Could not update API key.');
          }
        });
    } else {
      this.authService.createUserKey(this.userId, payload)
        .pipe(finalize(() => this.isLoading = false))
        .subscribe({
          next: (resKey) => {
            this.notificationService.showSuccess('API key created successfully!');
            // Show the token reveal UI
            this.plaintextToken = resKey.token || null;
            if (!this.plaintextToken) {
              // fallback if backend didn't return a plaintext token
              this.modalService.close(true);
            } else {
              // Notify parent immediately that key has been created to refresh key lists in the background
              this.modalService.emitResult(true);
            }
          },
          error: (err) => {
            console.error('Failed to create API key', err);
            this.notificationService.showError('Could not create API key.');
          }
        });
    }
  }

  public closeModal(): void {
    // If we've generated a key, ensure they confirmed copying it before letting them exit
    if (this.plaintextToken && !this.isTokenSavedConfirmed) {
      this.notificationService.showError('Please confirm you have saved your API key first.');
      return;
    }
    this.modalService.close(true);
  }

  public cancelModal(): void {
    this.modalService.close(false);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}
