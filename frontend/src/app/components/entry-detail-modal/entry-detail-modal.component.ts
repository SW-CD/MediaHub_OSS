// frontend/src/app/components/entry-detail-modal/entry-detail-modal.component.ts

import { Component, OnDestroy, OnInit } from '@angular/core';
import { Subject, Observable, of } from 'rxjs';
import { switchMap, takeUntil, map, take, filter, tap } from 'rxjs/operators';
import { Database, Entry, User } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';
import { EditEntryModalComponent } from '../edit-entry-modal/edit-entry-modal.component'; // <-- IMPORT THIS

@Component({
  selector: 'app-entry-detail-modal',
  templateUrl: './entry-detail-modal.component.html',
  styleUrls: ['./entry-detail-modal.component.css'],
  standalone: false,
})
export class EntryDetailModalComponent implements OnInit, OnDestroy {
  public static readonly MODAL_ID = 'entryDetailModal';

  public entryForMetadata: Entry | null = null;
  public fileUrl: SafeUrl | null = null;
  // REMOVED: metadataKeys is no longer needed as we map fields manually in HTML
  public isLoadingFile: boolean = true;
  
  public currentUser$: Observable<User | null>;
  public previewUrl: string | null = null;
  
  // Changed to public so HTML can access config/custom_fields
  public currentDatabase: Database | null = null; 
  
  private currentObjectUrl: string | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private modalService: ModalService,
    private databaseService: DatabaseService,
    private sanitizer: DomSanitizer,
    private authService: AuthService,
    private notificationService: NotificationService
  ) {
    this.currentUser$ = this.authService.currentUser$;
  }

  ngOnInit(): void {
    this.databaseService.selectedDatabase$
      .pipe(takeUntil(this.destroy$))
      .subscribe((db) => (this.currentDatabase = db));

    this.databaseService.selectedEntry$.pipe(
      takeUntil(this.destroy$),
      switchMap(entry => {
        this.revokeFileObjectUrl();
        
        this.fileUrl = null;
        this.previewUrl = null;
        this.isLoadingFile = true;
        this.entryForMetadata = null;

        if (entry && this.currentDatabase) {
          // Fetch fresh metadata
          return this.databaseService.getEntryMeta(this.currentDatabase.name, entry.id).pipe(
            switchMap(metaEntry => {
              this.entryForMetadata = metaEntry;
              this.previewUrl = this.getPreviewUrl();

              // Fetch the actual file blob
              return this.databaseService.getEntryFileBlob(this.currentDatabase!.name, entry.id).pipe(
                map(blob => {
                  this.currentObjectUrl = URL.createObjectURL(blob);
                  this.fileUrl = this.sanitizer.bypassSecurityTrustUrl(this.currentObjectUrl);
                  this.isLoadingFile = false;
                  return 'loaded';
                }),
                // Handle case where file load fails but metadata loaded ok
                tap({ error: () => this.isLoadingFile = false })
              );
            })
          );
        } else {
          this.isLoadingFile = false;
          return of(null);
        }
      })
    ).subscribe();
  }

  public getPreviewUrl(): string | null {
    if (this.currentDatabase && this.entryForMetadata) {
      return this.databaseService.getEntryPreviewUrl(this.currentDatabase.name, this.entryForMetadata.id);
    }
    return null;
  }

  // --- NEW: Edit Handler ---
  onEdit(): void {
    if (!this.entryForMetadata) return;

    if (this.entryForMetadata.status === 'processing') {
      this.notificationService.showError('Cannot edit an entry that is still processing.');
      return;
    }

    // Open the Edit Modal. 
    // Since both components listen to `selectedEntry$`, updates will propagate automatically 
    // if the service updates the subject after save.
    this.modalService.open(EditEntryModalComponent.MODAL_ID);
  }

  onDelete(): void {
    if (!this.currentDatabase || !this.entryForMetadata) return;

    if (this.entryForMetadata.status === 'processing') {
      this.notificationService.showError('Cannot delete an entry that is still processing.');
      return;
    }

    const modalData: ConfirmationModalData = {
      title: 'Delete Entry',
      message: `Are you sure you want to delete entry ${this.entryForMetadata.id}? This action cannot be undone.`
    };

    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(
        take(1),
        filter(confirmed => confirmed === true)
      )
      .subscribe(() => {
        this.isLoadingFile = true; 
        
        this.databaseService.deleteEntry(this.currentDatabase!.name, this.entryForMetadata!.id)
          .subscribe({
            next: () => {
              this.closeModal();
            },
            error: () => {
              this.isLoadingFile = false;
            }
          });
      });
  }

  private revokeFileObjectUrl(): void {
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = null;
    }
  }

  closeModal(): void {
    this.modalService.close();
    this.databaseService.clearSelectedEntry();
    this.revokeFileObjectUrl();

    this.fileUrl = null;
    this.previewUrl = null;
    this.isLoadingFile = true;
    this.entryForMetadata = null;
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.revokeFileObjectUrl();
  }
}