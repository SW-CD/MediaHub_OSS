// frontend/src/app/components/entry-detail-modal/entry-detail-modal.component.ts

import { Component, OnDestroy, OnInit } from '@angular/core';
import { Subject, Observable, of } from 'rxjs'; // Added Observable
import { switchMap, takeUntil, map, take, filter } from 'rxjs/operators'; // Added take, filter
import { Database, Entry, User } from '../../models/api.models'; // Added User
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { AuthService } from '../../services/auth.service'; // Added
import { NotificationService } from '../../services/notification.service'; // Added
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component'; // Added

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
  public metadataKeys: string[] = [];
  public isLoadingFile: boolean = true;
  
  // NEW: Observable for RBAC in the template
  public currentUser$: Observable<User | null>;

  public previewUrl: string | null = null;

  private currentDatabase: Database | null = null;
  private currentObjectUrl: string | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private modalService: ModalService,
    private databaseService: DatabaseService,
    private sanitizer: DomSanitizer,
    private authService: AuthService, // NEW
    private notificationService: NotificationService // NEW
  ) {
    // Initialize the user stream
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
        this.metadataKeys = [];
        this.entryForMetadata = null;

        if (entry && this.currentDatabase) {
          return this.databaseService.getEntryMeta(this.currentDatabase.name, entry.id).pipe(
            switchMap(metaEntry => {
              this.entryForMetadata = metaEntry;
              this.metadataKeys = Object.keys(metaEntry).filter(key => key !== 'id').sort();
              
              this.previewUrl = this.getPreviewUrl();

              return this.databaseService.getEntryFileBlob(this.currentDatabase!.name, entry.id).pipe(
                map(blob => {
                  this.currentObjectUrl = URL.createObjectURL(blob);
                  this.fileUrl = this.sanitizer.bypassSecurityTrustUrl(this.currentObjectUrl);
                  this.isLoadingFile = false;
                  return 'loaded';
                })
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

  // NEW: Handle Deletion
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

    // Open Confirmation Modal and wait for result
    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(
        take(1),
        filter(confirmed => confirmed === true)
      )
      .subscribe(() => {
        // We set isLoadingFile to true to disable buttons while the request is in flight
        this.isLoadingFile = true; 
        
        this.databaseService.deleteEntry(this.currentDatabase!.name, this.entryForMetadata!.id)
          .subscribe({
            next: () => {
              // Success! The service handles the list refresh.
              // We just close this modal.
              this.closeModal();
            },
            error: () => {
              // On error, re-enable the controls
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
    this.metadataKeys = [];
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.revokeFileObjectUrl();
  }
}