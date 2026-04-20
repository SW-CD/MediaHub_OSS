import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, FormArray, Validators } from '@angular/forms';
import { Subject, combineLatest } from 'rxjs';
import { takeUntil, finalize, filter, take } from 'rxjs/operators';
import { User, Permission, Database } from '../../models';
import { AuthService } from '../../services/auth.service';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';

@Component({
  selector: 'app-admin-user-list',
  templateUrl: './admin-user-list.component.html',
  styleUrls: ['./admin-user-list.component.css'],
  standalone: false,
})
export class AdminUserListComponent implements OnInit, OnDestroy {
  public users: User[] = [];
  public availableDatabases: string[] = [];
  public isLoading = true;
  
  // Master-Detail State
  public selectedUser: User | null = null;
  public isNewUser = false;
  public detailForm: FormGroup;
  public isSaving = false;

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
      password: [''], // Will be required dynamically for new users
      is_admin: [false],
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
   * Fetches both users and databases simultaneously.
   */
/**
   * Fetches both users and databases simultaneously.
   */
  loadData(): void {
    this.isLoading = true;
    this.cdr.markForCheck(); // Ensure the UI knows we are loading
    
    combineLatest([
      this.authService.getUsers(),
      this.databaseService.databases$ 
    ])
    .pipe(
      take(1), // <--- THE FIX: Forces the stream to complete after 1 emission
      takeUntil(this.destroy$),
      finalize(() => {
        this.isLoading = false;
        this.cdr.markForCheck();
      })
    )
    .subscribe({
      next: ([users, databases]) => {
        this.users = users;
        this.availableDatabases = databases.map(db => db.name);
        
        // If we reloaded data and a user is selected, refresh their form data
        if (this.selectedUser) {
          const refreshedUser = this.users.find(u => u.id === this.selectedUser!.id);
          if (refreshedUser) {
            this.selectUser(refreshedUser);
          } else {
            this.clearSelection();
          }
        }
      },
      error: (err) => {
        console.error('Failed to load admin user data:', err);
      }
    });
  }

  // --- Form Array Getter ---
  get permissions(): FormArray {
    return this.detailForm.get('permissions') as FormArray;
  }

  // --- Master Pane Actions ---

  trackById(index: number, user: User): number {
    return user.id;
  }

  selectUser(user: User): void {
    this.isNewUser = false;
    this.selectedUser = user;
    this.buildForm(user);
  }

  createNewUser(): void {
    this.isNewUser = true;
    this.selectedUser = null; // Clears the selection highlight in the list
    
    const emptyUser: Partial<User> = {
      username: '',
      is_admin: false,
      permissions: []
    };
    
    this.buildForm(emptyUser as User);
    // Password is required for a new user
    this.detailForm.get('password')?.setValidators([Validators.required, Validators.minLength(8)]);
    this.detailForm.get('password')?.updateValueAndValidity();
  }

  clearSelection(): void {
    this.selectedUser = null;
    this.isNewUser = false;
    this.detailForm.reset();
    this.permissions.clear();
  }

  // --- Detail Pane Logic ---

  private buildForm(user: User): void {
    this.detailForm.reset();
    this.permissions.clear();

    // Dynamically build a permission group for every available database
    this.availableDatabases.forEach(dbName => {
      const existingPerm = user.permissions?.find(p => p.database_name === dbName);
      
      this.permissions.push(this.fb.group({
        database_name: [dbName],
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
      password: '' // Never populate the password field
    });

    // Make password optional for editing existing users
    if (!this.isNewUser) {
      this.detailForm.get('password')?.setValidators([Validators.minLength(8)]);
      this.detailForm.get('password')?.updateValueAndValidity();
    }
  }

  /**
   * Toggles a specific permission flag for ALL databases in the form.
   */
  toggleColumnAll(permissionType: 'can_view' | 'can_create' | 'can_edit' | 'can_delete'): void {
    if (this.detailForm.get('is_admin')?.value) return; // Prevent toggling if disabled

    // Check if they are all currently true
    const allTrue = this.permissions.controls.every(ctrl => ctrl.get(permissionType)?.value === true);
    const newValue = !allTrue; // If all are true, turn them all off. Otherwise, turn them all on.

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
    const formData = this.detailForm.getRawValue(); // getRawValue includes disabled fields (like permissions if is_admin is true)

    // Remove empty password field so backend doesn't try to set it to blank
    if (!formData.password) {
      delete formData.password;
    }

    // Prepare API call
    const apiCall$ = this.isNewUser
      ? this.authService.createUser(formData)
      : this.authService.updateUser(formData.id, formData);

    apiCall$.pipe(finalize(() => this.isSaving = false)).subscribe({
      next: (savedUser) => {
        const action = this.isNewUser ? 'created' : 'updated';
        this.notificationService.showSuccess(`User ${savedUser.username} ${action} successfully!`);
        this.loadData(); // Refresh the list
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
          }
        });
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}