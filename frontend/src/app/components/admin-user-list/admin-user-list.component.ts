// frontend/src/app/components/admin-user-list/admin-user-list.component.ts

import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, FormArray, Validators } from '@angular/forms';
import { Subject, combineLatest } from 'rxjs';
import { takeUntil, finalize, filter, take } from 'rxjs/operators';
import { User, Permission, Database, ApiKey } from '../../models';
import { AuthService } from '../../services/auth.service';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';
import { ApiKeyModalComponent } from '../api-key-modal/api-key-modal.component';

@Component({
  selector: 'app-admin-user-list',
  templateUrl: './admin-user-list.component.html',
  styleUrls: ['./admin-user-list.component.css'],
  standalone: false,
})
export class AdminUserListComponent implements OnInit, OnDestroy {
  public users: User[] = [];
  public availableDatabases: { id: string, name: string }[] = [];
  public isLoading = true;
  
  // Master-Detail State
  public selectedUser: User | null = null;
  public isNewUser = false;
  public detailForm: FormGroup;
  public isSaving = false;

  // Sidebar List Tab (Standard Users vs Service Accounts)
  public activeListTab: 'standard' | 'service' = 'standard';
  
  // Detail Pane Tabs (Settings vs API Keys)
  public activeDetailTab: 'settings' | 'keys' = 'settings';
  public userKeys: ApiKey[] = [];
  public isKeysLoading = false;

  private destroy$ = new Subject<void>();

  constructor(
    private authService: AuthService,
    private databaseService: DatabaseService,
    private modalService: ModalService,
    private notificationService: NotificationService,
    private fb: FormBuilder,
    private cdr: ChangeDetectorRef
  ) {
    // Initialize the Reactive Form for the detail pane
    this.detailForm = this.fb.group({
      id: [null],
      username: ['', [Validators.required, Validators.pattern(/^[a-zA-Z0-9_]+$/)]],
      password: [''], // Will be required dynamically for new standard users
      is_admin: [false],
      is_service_account: [false],
      permissions: this.fb.array([])
    });
  }

  ngOnInit(): void {
    this.loadData();

    // Listen to changes on the is_admin toggle to disable/enable the permissions table
    this.detailForm.get('is_admin')?.valueChanges
      .pipe(takeUntil(this.destroy$))
      .subscribe(isAdmin => {
        if (isAdmin) {
          this.permissions.disable();
        } else {
          this.permissions.enable();
        }
      });
  }

  /**
   * Sets the active sidebar user list tab and reloads.
   */
  setListTab(tab: 'standard' | 'service'): void {
    this.activeListTab = tab;
    this.clearSelection();
    this.loadData();
  }

  /**
   * Sets the active detail pane tab.
   */
  setDetailTab(tab: 'settings' | 'keys'): void {
    this.activeDetailTab = tab;
    if (tab === 'keys') {
      this.loadUserKeys();
    }
  }

  /**
   * Fetches both users and databases simultaneously.
   */
  loadData(): void {
    this.isLoading = true;
    this.cdr.markForCheck();
    
    const filterServiceAccount = this.activeListTab === 'service';

    combineLatest([
      this.authService.getUsers(filterServiceAccount),
      this.databaseService.databases$ 
    ])
    .pipe(
      take(1),
      takeUntil(this.destroy$),
      finalize(() => {
        this.isLoading = false;
        this.cdr.markForCheck();
      })
    )
    .subscribe({
      next: ([users, databases]) => {
        this.users = users;
        this.availableDatabases = databases.map(db => ({ id: db.id, name: db.name }));
        
        // If we reloaded data and a user is selected, refresh their form data
        if (this.selectedUser) {
          const refreshedUser = this.users.find(u => u.id === this.selectedUser!.id);
          if (refreshedUser) {
            this.selectUser(refreshedUser);
            if (this.activeDetailTab === 'keys') {
              this.loadUserKeys();
            }
          } else {
            this.clearSelection();
          }
        }
      },
      error: (err) => {
        console.error('Failed to load admin user data:', err);
        this.notificationService.showError('Could not load user data.');
      }
    });
  }

  // --- Form Array Getter ---
  get permissions(): FormArray {
    return this.detailForm.get('permissions') as FormArray;
  }

  // --- Master Pane Actions ---

  trackById(index: number, user: User): string {
    return user.id;
  }

  selectUser(user: User): void {
    this.isNewUser = false;
    this.selectedUser = user;
    this.activeDetailTab = 'settings';
    this.buildForm(user);
    this.loadUserKeys();
  }

  createNewUser(): void {
    this.isNewUser = true;
    this.selectedUser = null; 
    this.activeDetailTab = 'settings';
    this.userKeys = [];
    
    const isService = this.activeListTab === 'service';
    const emptyUser: Partial<User> = {
      username: '',
      is_admin: false,
      is_service_account: isService,
      permissions: []
    };
    
    this.buildForm(emptyUser as User);
    
    if (isService) {
      // Password field is hidden and bypassed for service accounts
      this.detailForm.get('password')?.clearValidators();
    } else {
      this.detailForm.get('password')?.setValidators([Validators.required, Validators.minLength(8)]);
    }
    this.detailForm.get('password')?.updateValueAndValidity();
  }

  clearSelection(): void {
    this.selectedUser = null;
    this.isNewUser = false;
    this.activeDetailTab = 'settings';
    this.detailForm.reset();
    this.permissions.clear();
    this.userKeys = [];
  }

  // --- Detail Pane Logic ---

  private buildForm(user: User): void {
    this.detailForm.reset();
    this.permissions.clear();

    // Dynamically build a permission group for every available database
    this.availableDatabases.forEach(db => {
      const existingPerm = user.permissions?.find(p => p.database_id === db.id);
      
      this.permissions.push(this.fb.group({
        database_id: [db.id],
        database_name: [db.name],
        can_view: [existingPerm?.can_view || false],
        can_create: [existingPerm?.can_create || false],
        can_edit: [existingPerm?.can_edit || false],
        can_delete: [existingPerm?.can_delete || false]
      }));
    });

    this.detailForm.patchValue({
      id: user.id || null,
      username: user.username || '',
      is_admin: user.is_admin || false,
      is_service_account: user.is_service_account || false,
      password: ''
    });

    // Handle password field validation on edit
    if (!this.isNewUser) {
      if (user.is_service_account) {
        this.detailForm.get('password')?.clearValidators();
      } else {
        this.detailForm.get('password')?.setValidators([Validators.minLength(8)]);
      }
      this.detailForm.get('password')?.updateValueAndValidity();
    }
  }

  toggleColumnAll(permissionType: 'can_view' | 'can_create' | 'can_edit' | 'can_delete'): void {
    if (this.detailForm.get('is_admin')?.value) return;

    const allTrue = this.permissions.controls.every(ctrl => ctrl.get(permissionType)?.value === true);
    const newValue = !allTrue;

    this.permissions.controls.forEach(ctrl => {
      ctrl.get(permissionType)?.setValue(newValue);
    });
    
    this.detailForm.markAsDirty();
  }

  onSaveUser(): void {
    if (this.detailForm.invalid) {
      this.detailForm.markAllAsTouched();
      return;
    }

    this.isSaving = true;
    
    const formData = JSON.parse(JSON.stringify(this.detailForm.getRawValue()));

    // Bypasses password for service accounts or if password wasn't provided for edit
    if (formData.is_service_account || !formData.password) {
      delete formData.password;
    }

    if (formData.permissions) {
      formData.permissions = formData.permissions.map((p: any) => {
        const { database_name, ...rest } = p;
        return rest;
      });
    }

    const apiCall$ = this.isNewUser
      ? this.authService.createUser(formData)
      : this.authService.updateUser(formData.id, formData);

    apiCall$.pipe(finalize(() => this.isSaving = false)).subscribe({
      next: (savedUser) => {
        const action = this.isNewUser ? 'created' : 'updated';
        this.notificationService.showSuccess(`User ${savedUser.username} ${action} successfully!`);
        this.loadData();
      },
      error: (err) => {
        console.error('Failed to save user:', err);
      }
    });
  }

  openDeleteConfirm(): void {
    const userToDelete = this.selectedUser;
    if (!userToDelete) return;

    const modalData: ConfirmationModalData = {
      title: 'Delete User',
      message: `Are you sure you want to delete the user "${userToDelete.username}"? This action cannot be undone.`,
    };

    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(take(1), filter(isConfirmed => isConfirmed === true))
      .subscribe(() => {
        this.authService.deleteUser(userToDelete.id).subscribe({
          next: () => {
            this.notificationService.showSuccess(`User deleted successfully.`);
            this.clearSelection();
            this.loadData();
          },
          error: (err) => {
            console.error('Failed to delete user', err);
          }
        });
      });
  }

  // --- API Key Management (Delegated) ---

  loadUserKeys(): void {
    if (!this.selectedUser) return;
    this.isKeysLoading = true;
    this.authService.getUserKeys(this.selectedUser.id)
      .pipe(
        takeUntil(this.destroy$),
        finalize(() => {
          this.isKeysLoading = false;
          this.cdr.markForCheck();
        })
      )
      .subscribe({
        next: (keys) => {
          this.userKeys = keys || [];
        },
        error: (err) => {
          console.error('Failed to load user keys', err);
          this.notificationService.showError('Could not load user API keys.');
        }
      });
  }

  openCreateKeyModal(): void {
    if (!this.selectedUser) return;
    this.modalService.open(ApiKeyModalComponent.MODAL_ID, { userId: this.selectedUser.id })
      .pipe(take(1))
      .subscribe(created => {
        if (created) {
          this.loadUserKeys();
        }
      });
  }

  openEditKeyModal(key: ApiKey): void {
    if (!this.selectedUser) return;
    this.modalService.open(ApiKeyModalComponent.MODAL_ID, { userId: this.selectedUser.id, apiKey: key })
      .pipe(take(1))
      .subscribe(updated => {
        if (updated) {
          this.loadUserKeys();
        }
      });
  }

  openRevokeKeyConfirm(key: ApiKey): void {
    if (!this.selectedUser) return;

    const modalData: ConfirmationModalData = {
      title: 'Revoke User API Key',
      message: `Are you sure you want to revoke/delete the API key "${key.name}" on behalf of this user?`
    };

    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(take(1), filter(isConfirmed => isConfirmed === true))
      .subscribe(() => {
        this.authService.deleteUserKey(this.selectedUser!.id, key.id).subscribe({
          next: () => {
            this.notificationService.showSuccess('API key revoked successfully.');
            this.loadUserKeys();
          },
          error: (err) => {
            console.error('Failed to revoke key', err);
            this.notificationService.showError('Could not revoke key.');
          }
        });
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}