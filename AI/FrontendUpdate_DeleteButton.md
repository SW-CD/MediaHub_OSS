# Feature Guide: Delete Button in Entry Detail View

## 1\. 🎯 Objective

Add a "Delete" button to the **Entry Detail Modal** that allows authorized users to permanently remove an entry while viewing its details.

**Requirements:**

  * The button must **only** be visible to users with the `CanDelete` role.
  * The button must be disabled or hidden if the entry is in `processing` status.
  * Clicking the button must trigger the `ConfirmationModal`.
  * Successful deletion must close the modal and refresh the main list.

## 2\. 🛠️ Implementation Steps

### Step 1: Update Component Logic (`entry-detail-modal.component.ts`)

We need to inject the `AuthService` to access user permissions and implement the deletion workflow involving the confirmation modal.

**Changes Required:**

1.  Import `AuthService` and `ConfirmationModalComponent`.
2.  Inject `AuthService` in the constructor.
3.  Expose `currentUser$` to the template.
4.  Implement `onDelete()` method.

<!-- end list -->

```typescript
// src/app/components/entry-detail-modal/entry-detail-modal.component.ts

// 1. Add Imports
import { AuthService } from '../../services/auth.service';
import { User } from '../../models/api.models';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';
import { take, filter } from 'rxjs/operators'; // Ensure these are imported

export class EntryDetailModalComponent implements OnInit, OnDestroy {
  // ... existing properties ...

  // 2. Add User Observable
  public currentUser$: Observable<User | null>;

  constructor(
    private modalService: ModalService,
    private databaseService: DatabaseService,
    private sanitizer: DomSanitizer,
    private authService: AuthService, // <-- 3. Inject AuthService
    private notificationService: NotificationService // <-- Inject NotificationService (if not already there)
  ) {
    this.currentUser$ = this.authService.currentUser$;
  }

  // ... existing ngOnInit ...

  // 4. Implement Delete Logic
  onDelete(): void {
    if (!this.currentDatabase || !this.entryForMetadata) return;

    // Guard: Prevent deleting processing entries
    if (this.entryForMetadata.status === 'processing') {
      this.notificationService.showError('Cannot delete an entry that is still processing.');
      return;
    }

    const modalData: ConfirmationModalData = {
      title: 'Delete Entry',
      message: `Are you sure you want to delete entry ${this.entryForMetadata.id}? This action cannot be undone.`
    };

    // Open Confirmation Modal
    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(
        take(1),
        filter(confirmed => confirmed === true)
      )
      .subscribe(() => {
        // Proceed with deletion
        this.isLoadingFile = true; // Show loading state while deleting
        this.databaseService.deleteEntry(this.currentDatabase!.name, this.entryForMetadata!.id)
          .subscribe({
            next: () => {
              // On success, close this detail modal
              // The service already handles the success notification and list refresh
              this.closeModal();
            },
            error: () => {
              this.isLoadingFile = false; // Re-enable controls on error
            }
          });
      });
  }
}
```

### Step 2: Update Template (`entry-detail-modal.component.html`)

We need to subscribe to `currentUser$` and conditionally render the delete button in the footer.

**Changes Required:**

1.  Wrap the content (or the actions area) to unwrap the `user` observable.
2.  Add the Delete button with the `btn-danger` class.
3.  Add logic to disable the button if `isLoadingFile` is true or status is `processing`.

<!-- end list -->

```html
<div class="modal-actions" *ngIf="currentUser$ | async as user">
  
  <button 
    *ngIf="user.can_delete"
    type="button" 
    class="btn btn-danger"
    style="margin-right: auto;" 
    (click)="onDelete()"
    [disabled]="isLoadingFile || entryForMetadata?.status === 'processing'">
    Delete
  </button>

  <a [href]="fileUrl" 
     [download]="entryForMetadata?.filename || 'download'" 
     class="btn btn-secondary"
     [class.disabled]="isLoadingFile" 
     [style.pointer-events]="isLoadingFile ? 'none' : 'auto'">
     Download
  </a>
  
  <button type="button" class="btn btn-primary" (click)="closeModal()">Close</button>
</div>
```

*Note: `style="margin-right: auto;"` is a quick utility to push the Delete button to the left side of the footer, separating destructive actions from navigation actions. Alternatively, use a spacer div class.*

### Step 3: Testing Checklist

After implementation, verify the following scenarios:

1.  **Role Check:** Log in as a "Viewer" (read-only). Open an entry. **Result:** Delete button is NOT visible.
2.  **Role Check:** Log in as "Admin" or user with `CanDelete`. Open an entry. **Result:** Delete button IS visible.
3.  **Cancellation:** Click Delete, then click "Cancel" in the confirmation modal. **Result:** Detail modal remains open; entry is not deleted.
4.  **Execution:** Click Delete, then click "Confirm". **Result:** Confirmation modal closes, Detail modal closes, Success toast appears, Entry is removed from the Grid/List view.
5.  **Safety:** Try to delete an entry with `status: 'processing'` (if possible to catch one). **Result:** Button should be disabled or show an error toast if clicked.