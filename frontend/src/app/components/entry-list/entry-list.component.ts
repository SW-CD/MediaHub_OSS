// frontend/src/app/components/entry-list/entry-list.component.ts
import { Component, OnDestroy, OnInit, ChangeDetectorRef } from '@angular/core';
import { ActivatedRoute, ParamMap } from '@angular/router';
import { Observable, of, Subject, merge } from 'rxjs';
import { switchMap, takeUntil, finalize, filter, take, map, distinctUntilChanged, tap } from 'rxjs/operators';
import { Database, User, SearchRequest, SearchFilter, Entry } from '../../models/api.models'; 
import { DatabaseService } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service'; 
import { UploadEntryModalComponent } from '../upload-entry-modal/upload-entry-modal.component';
import { EntryDetailModalComponent } from '../entry-detail-modal/entry-detail-modal.component';
import { EditEntryModalComponent } from '../edit-entry-modal/edit-entry-modal.component';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';
import { isMimeTypeAllowed } from '../../utils/mime-types';
import { AvailableFilter, FilterChangedEvent } from '../entry-filter/entry-filter.component'; // Import from child

@Component({
  selector: 'app-entry-list',
  templateUrl: './entry-list.component.html',
  styleUrls: ['./entry-list.component.css'],
  standalone: false,
})
export class EntryListComponent implements OnInit, OnDestroy {
  public currentUser$: Observable<User | null>;
  public entriesToShow: Entry[] = [];
  public currentUser: User | null = null; 

  public isLoading = true;
  public isBulkProcessing = false;
  public tableColumns: string[] = [];
  public availableFilters: AvailableFilter[] = [];

  public dbName: string | null = null;
  private currentDb: Database | null = null;

  private destroy$ = new Subject<void>();
  private manualFetchTrigger$ = new Subject<void>(); 

  // --- STATE PROPERTIES ---
  public viewMode: 'grid' | 'list' = 'grid'; 
  public currentPage = 1;
  public imagesPerPage = 24;
  public hasNextPage = false;
  
  // Track current filter to apply it when paging
  private currentFilterConditions: SearchFilter | undefined;

  // --- SELECTION STATE ---
  public selectedEntryIds = new Set<number>();
  public lastSelectedEntryId: number | null = null;

  constructor(
    private route: ActivatedRoute,
    private databaseService: DatabaseService,
    private authService: AuthService,
    private modalService: ModalService,
    private notificationService: NotificationService, 
    private cdr: ChangeDetectorRef
  ) {
    this.currentUser$ = this.authService.currentUser$;
  }

  ngOnInit(): void {
    this.currentUser$.pipe(takeUntil(this.destroy$)).subscribe(user => {
      this.currentUser = user;
      if (this.currentDb) {
        this.setupTableColumns(this.currentDb);
      }
    });

    this.route.paramMap.pipe(
      takeUntil(this.destroy$),
      map((params: ParamMap) => params.get('name')),
      distinctUntilChanged(), 
      tap(name => this.setupForNewDatabase(name)),
      filter((name): name is string => !!name), 
      switchMap(name =>
        merge(of(null), this.manualFetchTrigger$, this.databaseService.refreshRequired$).pipe(
          tap(() => {
            this.isLoading = true; 
            this.clearSelection();
            this.cdr.markForCheck();
          }),
          switchMap(() => {
            if (!this.dbName || this.dbName !== name) return of([]); 
            
            const searchPayload = this.buildSearchPayload();
            return this.databaseService.searchEntries(name, searchPayload);
          })
        )
      )
    ).subscribe({
      next: entries => {
        if (entries && entries.length > this.imagesPerPage) {
          this.hasNextPage = true;
          this.entriesToShow = entries.slice(0, this.imagesPerPage);
        } else {
          this.hasNextPage = false;
          this.entriesToShow = entries || [];
        }
        this.isLoading = false; 
        this.cdr.markForCheck(); 
      },
      error: (err) => {
        this.isLoading = false; 
        this.entriesToShow = [];
        this.hasNextPage = false;
        this.cdr.markForCheck(); 
      }
    });
  }

  /**
   * Called by the Child Component (EntryFilter) when the user clicks Apply.
   */
  onFilterApplied(event: FilterChangedEvent): void {
    this.currentFilterConditions = event.filter;
    this.imagesPerPage = event.limit;
    this.currentPage = 1; // Reset to page 1 on new filter
    this.manualFetchTrigger$.next();
  }

  private buildSearchPayload(): SearchRequest {
    const payload: SearchRequest = {
      pagination: {
        limit: this.imagesPerPage + 1,
        offset: (this.currentPage - 1) * this.imagesPerPage
      },
      sort: { field: 'timestamp', direction: 'desc' }
    };

    if (this.currentFilterConditions) {
      payload.filter = this.currentFilterConditions;
    }
    return payload;
  }

  private setupForNewDatabase(name: string | null): void {
    this.dbName = name;
    this.entriesToShow = [];
    this.currentPage = 1; 
    this.hasNextPage = false;
    this.clearSelection();
    this.currentFilterConditions = undefined; // Reset filters on DB change

    this.isLoading = !!name; 

    if (name) {
      this.databaseService.selectDatabase(name).pipe(take(1)).subscribe(db => {
        if (db) {
          this.currentDb = db;
          this.setupTableColumns(db);
          this.setupAvailableFilters(db);
        } else {
          this.currentDb = null;
          this.tableColumns = [];
          this.availableFilters = [];
          this.isLoading = false;
        }
        this.cdr.markForCheck();
      });
    } else {
      this.currentDb = null;
      this.tableColumns = [];
      this.availableFilters = [];
      this.isLoading = false;
      this.cdr.markForCheck();
    }
  }

  private setupAvailableFilters(db: Database): void {
    const filters: AvailableFilter[] = [
      { name: 'timestamp', type: 'INTEGER' },
      { name: 'filesize', type: 'INTEGER' },
      { name: 'mime_type', type: 'TEXT' },
      { name: 'filename', type: 'TEXT' }, 
      { name: 'status', type: 'TEXT' },
    ];
    if (db.content_type === 'image') {
      filters.push({ name: 'width', type: 'INTEGER' });
      filters.push({ name: 'height', type: 'INTEGER' });
    } else if (db.content_type === 'audio') {
      filters.push({ name: 'duration_sec', type: 'REAL' });
      filters.push({ name: 'channels', type: 'INTEGER' });
    }
    filters.push(...db.custom_fields);
    filters.sort((a, b) => a.name.localeCompare(b.name));
    this.availableFilters = filters;
  }

  // --- SELECTION LOGIC (Unchanged) ---

  toggleSelection(event: { entry: Entry, event: MouseEvent }): void {
    const { entry, event: mouseEvent } = event;
    const entryId = entry.id;

    if (mouseEvent.shiftKey && this.lastSelectedEntryId !== null) {
      const lastIndex = this.entriesToShow.findIndex(e => e.id === this.lastSelectedEntryId);
      const currentIndex = this.entriesToShow.findIndex(e => e.id === entryId);

      if (lastIndex !== -1 && currentIndex !== -1) {
        const start = Math.min(lastIndex, currentIndex);
        const end = Math.max(lastIndex, currentIndex);
        for (let i = start; i <= end; i++) {
          this.selectedEntryIds.add(this.entriesToShow[i].id);
        }
      }
    } else {
      if (this.selectedEntryIds.has(entryId)) {
        this.selectedEntryIds.delete(entryId);
        this.lastSelectedEntryId = null;
      } else {
        this.selectedEntryIds.add(entryId);
        this.lastSelectedEntryId = entryId;
      }
    }
  }

  clearSelection(): void {
    this.selectedEntryIds.clear();
    this.lastSelectedEntryId = null;
  }

  // --- BULK ACTIONS (Unchanged) ---

  onBulkDownload(): void {
    if (!this.dbName || this.selectedEntryIds.size === 0) return;
    this.isBulkProcessing = true;
    const ids = Array.from(this.selectedEntryIds);
    this.databaseService.bulkExportEntries(this.dbName, ids)
      .pipe(finalize(() => this.isBulkProcessing = false))
      .subscribe({
        next: (blob) => {
          const url = window.URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `${this.dbName}_export.zip`;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          window.URL.revokeObjectURL(url);
          this.notificationService.showSuccess('Export downloaded successfully.');
          this.clearSelection();
        },
        error: () => this.notificationService.showError('Failed to generate export archive.')
      });
  }

  onBulkDelete(): void {
    if (!this.dbName || this.selectedEntryIds.size === 0) return;
    const ids = Array.from(this.selectedEntryIds);
    const modalData: ConfirmationModalData = {
      title: 'Confirm Bulk Deletion',
      message: `Are you sure you want to delete ${ids.length} selected entries from '${this.dbName}'?`
    };
    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(take(1), filter(confirmed => confirmed === true))
      .subscribe(() => {
        if (this.dbName) {
          this.isBulkProcessing = true;
          this.databaseService.bulkDeleteEntries(this.dbName, ids)
            .pipe(finalize(() => this.isBulkProcessing = false))
            .subscribe({ next: () => this.clearSelection() });
        }
      });
  }

  // --- UI Setup & Modals (Unchanged) ---

  private setupTableColumns(db: Database): void {
    let standardColumns = ['id', 'timestamp', 'filename', 'mime_type', 'filesize', 'status'];
    if (db.content_type === 'image') standardColumns.push('width', 'height');
    else if (db.content_type === 'audio') standardColumns.push('duration_sec', 'channels');
    
    const customColumns = db.custom_fields.map(field => field.name);
    const actionColumn = (this.currentUser && (this.currentUser.can_edit || this.currentUser.can_delete)) ? ['actions'] : [];
    this.tableColumns = ['Preview', ...standardColumns, ...customColumns, ...actionColumn];
  }

  openUploadModal(): void {
    if (this.dbName) this.modalService.open(UploadEntryModalComponent.MODAL_ID);
    else this.notificationService.showInfo('Please select a database first.');
  }

  openEditModal(entry: Entry): void {
    if (this.dbName) {
      if (entry.status === 'processing') return this.notificationService.showError('Cannot edit processing entry.');
      this.databaseService.selectEntry(entry);
      this.modalService.open(EditEntryModalComponent.MODAL_ID);
    }
  }

  openDeleteConfirm(entry: Entry): void {
    if (!this.currentDb) return;
    if (entry.status === 'processing') return this.notificationService.showError('Cannot delete processing entry.');
    const modalData = { message: `Delete entry ${entry.id}?` };
    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(take(1), filter(c => c === true))
      .subscribe(() => this.databaseService.deleteEntry(this.currentDb!.name, entry.id).subscribe());
  }

  openDetailModal(entry: Entry): void {
     if (this.dbName) {
        if (entry.status === 'processing') return this.notificationService.showInfo('Entry processing...');
        this.databaseService.selectEntry(entry);
        this.modalService.open(EntryDetailModalComponent.MODAL_ID);
     }
  }

  public setViewMode(mode: 'grid' | 'list'): void {
    this.viewMode = mode;
    this.cdr.markForCheck();
  }

  public nextPage(): void { if (this.hasNextPage) { this.currentPage++; this.manualFetchTrigger$.next(); } }
  public prevPage(): void { if (this.currentPage > 1) { this.currentPage--; this.manualFetchTrigger$.next(); } }

  onFileDropped(file: File): void {
    if (!this.currentDb || !this.currentUser?.can_create) return this.notificationService.showInfo('Cannot upload here.');
    if (!isMimeTypeAllowed(this.currentDb.content_type, file.type)) return this.notificationService.showError(`Invalid file type.`);
    this.modalService.open(UploadEntryModalComponent.MODAL_ID, { droppedFile: file });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}