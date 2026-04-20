import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators, FormArray } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, finalize } from 'rxjs/operators';
import { User, Permission } from '../../models'; // UPDATED: Use index barrel
import { AuthService } from '../../services/auth.service';
import { DatabaseService } from '../../services/database.service'; // NEW: Required to fetch DB list
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
  
  public availableDatabases: string[] = []; // NEW: Keep track of databases

  private userIdToEdit: number | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private databaseService: DatabaseService,
    private modalService: ModalService,
    private notificationService: NotificationService
  ) {
    this.userForm = this.fb.group({
      username: ['', [Validators.required, Validators.pattern(/^[a-zA-Z0-9_]+$/)]],
      password: ['', [Validators.minLength(8)]], 
      is_admin: [false],
      permissions: this.fb.array([]) // NEW: FormArray for DB-scoped permissions
    });
  }

  ngOnInit(): void {
    // Keep an up-to-date list of available databases to build the permissions form
    this.databaseService.databases$
      .pipe(takeUntil(this.destroy$))
      .subscribe(dbs => {
        this.availableDatabases = dbs.map(db => db.name);
      });

    this.modalService.getModalEvents(UserFormComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe((event: ModalEvent) => {
        if (event.action === 'open') {
          this.setupFormForMode(event.data);
        }
      });
  }

  // Helper to easily access the FormArray in the HTML template
  get permissions(): FormArray {
    return this.userForm.get('permissions') as FormArray;
  }

  private setupFormForMode(data: any): void {
    this.userForm.reset();
    this.permissions.clear(); // Clear old permissions
    
    this.isEditMode = data?.isEditMode || false;
    this.userIdToEdit = null;

    const userPerms: Permission[] = data?.user?.permissions || [];

    // Dynamically build a permission group for every database currently in the system
    this.availableDatabases.forEach(dbName => {
      const existing = userPerms.find(p => p.database_name === dbName);
      
      this.permissions.push(this.fb.group({
        database_name: [dbName],
        can_view: [existing?.can_view || false],
        can_create: [existing?.can_create || false],
        can_edit: [existing?.can_edit || false],
        can_delete: [existing?.can_delete || false]
      }));
    });

    if (this.isEditMode && data.user) {
      this.userIdToEdit = data.user.id;
      this.userForm.patchValue({
        username: data.user.username,
        is_admin: data.user.is_admin
      });
      this.userForm.get('password')?.setValidators([Validators.minLength(8)]);
    } else {
      this.userForm.patchValue({ is_admin: false });
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
          this.closeModal(true); 
        }
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