// frontend/src/app/components/entry-detail-modal/entry-detail-modal.component.ts
import { Component, OnDestroy, OnInit } from '@angular/core';
import { Subject, of } from 'rxjs';
import { switchMap, takeUntil, map } from 'rxjs/operators';
import { Database, Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';

// NOTE: We still import SecureImageDirective implicitly if it's in AppModule,
// but since this is a Modal inside AppModule, we rely on the module setup.
// Unlike the standalone Grid/List components, this component is declared in AppModule.

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

  // URL string for the preview (waveform), processed by the Directive in the template
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
              
              // The directive will handle fetching this URL securely
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
    
    this.fileUrl = null;
    this.previewUrl = null;
    this.isLoadingFile = true;
    this.entryForMetadata = null;
    this.metadataKeys = [];
  }
}