// frontend/src/app/components/admin-user-list/admin-user-list.component.ts
import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { Observable, Subject, of } from 'rxjs';
import { takeUntil, finalize, filter, take } from 'rxjs/operators';
import { User } from '../../models/api.models';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { UserFormComponent } from '../user-form/user-form.component';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';
import { NotificationService } from '../../services/notification.service';

@Component({
  selector: 'app-admin-user-list',
  templateUrl: './admin-user-list.component.html',
  styleUrls: ['./admin-user-list.component.css'],
  standalone: false,
})
export class AdminUserListComponent implements OnInit, OnDestroy {
  public users: User[] = [];
  public isLoading = true;
  private destroy$ = new Subject<void>();

  constructor(
    private authService: AuthService,
    private modalService: ModalService,
    private notificationService: NotificationService,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.refreshUsers();
  }

  refreshUsers(): void {
    this.isLoading = true;
    this.authService.getUsers().pipe(
      takeUntil(this.destroy$),
      finalize(() => {
        this.isLoading = false;
        this.cdr.markForCheck();
      })
    ).subscribe(users => {
      this.users = users;
    });
  }

  trackById(index: number, user: User): number {
    return user.id;
  }

  openCreateUserModal(): void {
    this.modalService.open(UserFormComponent.MODAL_ID, { isEditMode: false })
      .pipe(take(1), filter(result => result === true))
      .subscribe(() => this.refreshUsers());
  }

  openEditUserModal(user: User): void {
    // Create a deep copy to prevent accidental modification of the list data
    const userCopy = JSON.parse(JSON.stringify(user));
    this.modalService.open(UserFormComponent.MODAL_ID, { isEditMode: true, user: userCopy })
      .pipe(take(1), filter(result => result === true))
      .subscribe(() => this.refreshUsers());
  }

  openDeleteUserModal(user: User): void {
    const modalData: ConfirmationModalData = {
      title: 'Delete User',
      message: `Are you sure you want to delete the user "${user.username}"? This action cannot be undone.`,
    };

    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(
        take(1),
        filter((isConfirmed) => isConfirmed === true)
      )
      .subscribe(() => {
        this.authService.deleteUser(user.id).subscribe({
          next: () => {
            // Success notification is handled by the service, just refresh
            this.refreshUsers();
          },
          // Errors are handled by the service
        });
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}

