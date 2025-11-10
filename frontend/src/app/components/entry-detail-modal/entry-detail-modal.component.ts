// frontend/src/app/components/entry-detail-modal/entry-detail-modal.component.ts

import { Component, OnDestroy, OnInit } from '@angular/core';
import { Observable, Subject, of } from 'rxjs';
import { switchMap, takeUntil, filter, map, tap } from 'rxjs/operators';
// UPDATED: Image to Entry
import { Database, Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';

@Component({
  selector: 'app-entry-detail-modal', // RENAMED
  templateUrl: './entry-detail-modal.component.html', // RENAMED
  styleUrls: ['./entry-detail-modal.component.css'], // RENAMED
  standalone: false,
})
export class EntryDetailModalComponent implements OnInit, OnDestroy { // RENAMED
  public static readonly MODAL_ID = 'entryDetailModal'; // RENAMED

  // UPDATED: imageForMetadata to entryForMetadata
  public entryForMetadata: Entry | null = null;
  public fileUrl: SafeUrl | null = null; // RENAMED: imageUrl to fileUrl
  public metadataKeys: string[] = [];
  public isLoadingFile: boolean = true; // RENAMED: isLoadingImage to isLoadingFile

  // NEW: Property for the preview (waveform) URL
  public previewUrl: string | null = null;

  private currentDatabase: Database | null = null;
  private currentObjectUrl: string | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private modalService: ModalService,
    private databaseService: DatabaseService,
    private sanitizer: DomSanitizer
  ) {}

  ngOnInit(): void {
    // Keep track of the currently selected database
    this.databaseService.selectedDatabase$
      .pipe(takeUntil(this.destroy$))
      .subscribe((db) => (this.currentDatabase = db));

    // --- REWORKED LOGIC ---
    // When the selected entry changes, this is our trigger.
    // RENAMED: selectedImage$ to selectedEntry$
    this.databaseService.selectedEntry$.pipe(
      takeUntil(this.destroy$),
      // SwitchMap will cancel previous pending fetches if a new entry is selected quickly
      switchMap(entry => {
        // Revoke any previous object URL to prevent memory leaks
        this.revokeFileObjectUrl();
        
        // 1. Immediately reset the state to "loading"
        this.fileUrl = null;
        this.previewUrl = null; // NEW: Clear preview URL
        this.isLoadingFile = true;
        this.metadataKeys = [];
        this.entryForMetadata = null;

        if (entry && this.currentDatabase) {
          // 2. Fetch metadata first
          // RENAMED: getImageMeta to getEntryMeta
          return this.databaseService.getEntryMeta(this.currentDatabase.name, entry.id).pipe(
            // 3. Once metadata is here, fetch the blob
            switchMap(metaEntry => {
              this.entryForMetadata = metaEntry; // Set metadata for template
              this.metadataKeys = Object.keys(metaEntry).filter(key => key !== 'id').sort();
              
              // NEW: Get the preview URL (for audio waveforms)
              this.previewUrl = this.getPreviewUrl();

              // RENAMED: getImageBlob to getEntryFileBlob
              return this.databaseService.getEntryFileBlob(this.currentDatabase!.name, entry.id).pipe(
                map(blob => {
                  // 5. Got both! Return final state.
                  this.currentObjectUrl = URL.createObjectURL(blob);
                  this.fileUrl = this.sanitizer.bypassSecurityTrustUrl(this.currentObjectUrl);
                  this.isLoadingFile = false; // Done loading
                  return 'loaded'; // Return something to the outer subscription
                })
              );
            })
          );
        } else {
          // No entry selected (e.g., on modal close), just ensure state is cleared
          this.isLoadingFile = false; // Nothing to load
          return of(null); // Return an observable that completes
        }
      })
    ).subscribe(); // The state-setting logic is all inside the switchMap chain.
  }

  /**
   * NEW: Helper to get the preview URL (for waveforms).
   * Note: This is NOT a blob URL, it's a direct URL to the /api/entry/preview endpoint.
   * We let the <img src> handle auth via the secureImage pipe.
   */
  public getPreviewUrl(): string | null {
    if (this.currentDatabase && this.entryForMetadata) {
      // Use RENAMED service method
      return this.databaseService.getEntryPreviewUrl(this.currentDatabase.name, this.entryForMetadata.id);
    }
    return null;
  }

  /**
   * Revokes the current object URL to free up browser memory.
   */
  // RENAMED: revokeImageObjectUrl to revokeFileObjectUrl
  private revokeFileObjectUrl(): void {
    if (this.currentObjectUrl) {
      URL.revokeObjectURL(this.currentObjectUrl);
      this.currentObjectUrl = null;
    }
  }

  closeModal(): void {
    this.modalService.close();
    this.databaseService.clearSelectedEntry(); // RENAMED
    this.revokeFileObjectUrl(); // Clean up memory on close

    // --- ADDED ---
    // Reset loading state for next time
    this.fileUrl = null;
    this.previewUrl = null;
    this.isLoadingFile = true;
    this.entryForMetadata = null;
    this.metadataKeys = [];
    // --- END ADDED ---
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.revokeFileObjectUrl(); // Clean up memory on destroy
    
    // --- ADDED ---
    // Reset loading state
    this.fileUrl = null;
    this.previewUrl = null;
    this.isLoadingFile = true;
    this.entryForMetadata = null;
    this.metadataKeys = [];
    // --- END ADDED ---
  }
}
