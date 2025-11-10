// frontend/src/app/components/entry-list/entry-list.component.ts
import { Component, OnDestroy, OnInit, ChangeDetectorRef } from '@angular/core';
import { ActivatedRoute, ParamMap } from '@angular/router';
import { FormBuilder, FormGroup, FormArray, Validators, AbstractControl } from '@angular/forms'; 
import { Observable, of, Subject, merge } from 'rxjs';
import { switchMap, takeUntil, finalize, filter, take, map, distinctUntilChanged, tap } from 'rxjs/operators';
import { Database, User, CustomField, SearchRequest, SearchFilter, Entry } from '../../models/api.models'; 
import { DatabaseService } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service'; 
import { UploadEntryModalComponent } from '../upload-entry-modal/upload-entry-modal.component';
import { EntryDetailModalComponent } from '../entry-detail-modal/entry-detail-modal.component';
import { EditEntryModalComponent } from '../edit-entry-modal/edit-entry-modal.component';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';

// Define the structure for available filters, including standard ones
interface AvailableFilter {
    name: string;
    type: 'TEXT' | 'INTEGER' | 'REAL' | 'BOOLEAN';
}

@Component({
  selector: 'app-entry-list', // RENAMED
  templateUrl: './entry-list.component.html', // RENAMED
  styleUrls: ['./entry-list.component.css'], // RENAMED
  standalone: false,
})
export class EntryListComponent implements OnInit, OnDestroy { // RENAMED
  public currentUser$: Observable<User | null>;
  public entriesToShow: Entry[] = []; // RENAMED
  public currentUser: User | null = null; 

  public filterForm: FormGroup;
  public isLoading = true;
  public tableColumns: string[] = [];
  public availableFilters: AvailableFilter[] = [];

  public dbName: string | null = null;
  private currentDb: Database | null = null;

  private destroy$ = new Subject<void>();
  private manualFetchTrigger$ = new Subject<void>(); 

  // --- UPDATED: No longer need to subscribe to processingEntries$ here ---
  // public processingEntries$: Observable<number[]>;

  // --- STATE PROPERTIES ---
  public viewMode: 'grid' | 'list' = 'grid'; 
  public currentPage = 1;
  public imagesPerPage = 24; // This is now the 'limit' for backend pagination
  public hasNextPage = false; 

  constructor(
    private route: ActivatedRoute,
    private databaseService: DatabaseService,
    private authService: AuthService,
    private modalService: ModalService,
    private notificationService: NotificationService, 
    private fb: FormBuilder,
    private cdr: ChangeDetectorRef
  ) {
    this.currentUser$ = this.authService.currentUser$;
    // --- UPDATED: No longer need to subscribe to processingEntries$ here ---
    // this.processingEntries$ = this.databaseService.processingEntries$;

    this.filterForm = this.fb.group({
      // UPDATED: This is now 'limit per page'
      limitPerPage: [24, [Validators.required, Validators.min(1)]],
      tstart: [''],
      tend: [''],  
      customFilters: this.fb.array([]) 
    });
  }

  ngOnInit(): void {
    // --- Store the current user ---
    this.currentUser$.pipe(takeUntil(this.destroy$)).subscribe(user => {
      this.currentUser = user;
      if (this.currentDb) {
        this.setupTableColumns(this.currentDb);
      }
    });

    // React to route parameter changes
    this.route.paramMap.pipe(
      takeUntil(this.destroy$),
      map((params: ParamMap) => params.get('name')),
      distinctUntilChanged(), 
      tap(name => this.setupForNewDatabase(name)),
      filter((name): name is string => !!name), 
      // Main fetch logic
      switchMap(name =>
        merge(of(null), this.manualFetchTrigger$, this.databaseService.refreshRequired$).pipe(
          tap(() => {
            this.isLoading = true; 
            this.cdr.markForCheck();
          }),
          switchMap(() => {
            if (!this.dbName || this.dbName !== name) {
              return of([]); 
            }
            const searchPayload = this.buildSearchPayload();
            if (!searchPayload) {
               this.isLoading = false;
               this.cdr.markForCheck();
               return of([]);
            }
            // Use RENAMED service method
            return this.databaseService.searchEntries(name, searchPayload);
          })
        )
      )
    ).subscribe({
      next: entries => {
        // --- UPDATED: Handle backend pagination response ---
        this.imagesPerPage = this.filterForm.get('limitPerPage')?.value || 24;
        
        // --- NEW: Check if we received the "extra" item ---
        if (entries && entries.length > this.imagesPerPage) {
          // Yes, there is a next page
          this.hasNextPage = true;
          // Slice the array to only show the requested number of items
          this.entriesToShow = entries.slice(0, this.imagesPerPage);
        } else {
          // No, this is the last page
          this.hasNextPage = false;
          this.entriesToShow = entries || []; // Assign directly
        }
        // ---
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

  // Resets component state when the selected database changes
  private setupForNewDatabase(name: string | null): void {
    this.dbName = name;
    this.entriesToShow = [];
    this.currentPage = 1; 
    this.hasNextPage = false;

    this.isLoading = !!name; 
    this.customFilters.clear();
    
    this.filterForm.patchValue({
        tstart: '',
        tend: '',
        limitPerPage: 24 
    }, { emitEvent: false });


    if (name) {
      this.databaseService.selectDatabase(name).pipe(take(1)).subscribe(db => {
        if (db) {
          this.currentDb = db;
          this.setupTableColumns(db); // Configure table headers
          // --- UPDATED: Dynamically build available filters based on content_type ---
          this.availableFilters = [
              { name: 'timestamp', type: 'INTEGER' },
              { name: 'filesize', type: 'INTEGER' },
              { name: 'mime_type', type: 'TEXT' },
              { name: 'filename', type: 'TEXT' }, // <-- ADDED
              { name: 'status', type: 'TEXT' }, // <-- ADDED FOR ASYNC
          ];
          if (db.content_type === 'image') {
              this.availableFilters.push({ name: 'width', type: 'INTEGER' });
              this.availableFilters.push({ name: 'height', type: 'INTEGER' });
          } else if (db.content_type === 'audio') {
              this.availableFilters.push({ name: 'duration_sec', type: 'REAL' });
              this.availableFilters.push({ name: 'channels', type: 'INTEGER' });
          }
          // Add custom fields
          this.availableFilters.push(...db.custom_fields);
          this.availableFilters.sort((a, b) => a.name.localeCompare(b.name));
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

  /**
   * Builds the SearchRequest payload for the POST endpoint.
   * UPDATED: To remove the default status filter.
   */
  private buildSearchPayload(): SearchRequest | null {
    if (this.filterForm.invalid) {
      this.filterForm.markAllAsTouched();
      return null;
    }

    const formValue = this.filterForm.value;
    
    // --- UPDATED: Start with an empty filter list ---
    const conditions: SearchFilter[] = [];
    // --- END UPDATE ---


    // 1. Add time filters
    if (formValue.tstart) {
      const tstartUnix = this.datetimeLocalToUnix(formValue.tstart);
      if (tstartUnix !== null) {
        conditions.push({ field: 'timestamp', operator: '>=', value: tstartUnix });
      }
    }
    if (formValue.tend) {
      const tendUnix = this.datetimeLocalToUnix(formValue.tend);
      if (tendUnix !== null) {
        conditions.push({ field: 'timestamp', operator: '<=', value: tendUnix });
      }
    }

    // 2. Add custom filters
    formValue.customFilters.forEach((filter: any, index: number) => {
      if (filter.field && filter.value !== null && String(filter.value).trim() !== '') {
        const fieldDefinition = this.availableFilters.find(f => f.name === filter.field);
        let filterValue: any = String(filter.value).trim(); 

        // --- UPDATED: No longer need to check for default status filter ---
        // --- END UPDATE ---

        if (fieldDefinition) {
           if (fieldDefinition.type === 'INTEGER' || fieldDefinition.type === 'REAL') {
             const num = Number(filterValue);
             if (!isNaN(num)) { filterValue = num; }
           } else if (fieldDefinition.type === 'BOOLEAN') {
               const lowerVal = filterValue.toLowerCase();
               if (lowerVal === 'true' || lowerVal === '1') { filterValue = true; }
               else if (lowerVal === 'false' || lowerVal === '0') { filterValue = false; }
           }
        }
        conditions.push({
          field: filter.field,
          operator: filter.operator, 
          value: filterValue
        });
      }
    });

    // --- UPDATED: Backend pagination ---
    this.imagesPerPage = formValue.limitPerPage;
    const payload: SearchRequest = {
      pagination: {
        // --- NEW: Request n+1 items ---
        limit: this.imagesPerPage + 1,
        offset: (this.currentPage - 1) * this.imagesPerPage // Calculate offset
      },
      sort: { field: 'timestamp', direction: 'desc' }
    };

    if (conditions.length > 0) {
      payload.filter = {
        operator: 'and', 
        conditions: conditions
      };
    }
    return payload;
  }

  // Helper to convert datetime-local string to Unix timestamp (seconds)
  private datetimeLocalToUnix(dateTimeLocal: string): number | null {
      try {
          const date = new Date(dateTimeLocal);
          if (isNaN(date.getTime())) {
              return null;
          }
          return Math.floor(date.getTime() / 1000);
      } catch (e) {
          return null;
      }
  }


  // --- FormArray Management ---
  get customFilters(): FormArray {
    return this.filterForm.get('customFilters') as FormArray;
  }

  addCustomFilter(): void {
    const newGroup = this.fb.group({
        field: ['', Validators.required],
        operator: ['='], 
        value: ['', Validators.required]
    });

    newGroup.get('field')?.valueChanges.pipe(
        takeUntil(this.destroy$)
    ).subscribe(fieldName => {
        const fieldType = this.getSelectedFieldTypeForGroup(newGroup);
        const defaultOp = this.getOperatorsForFieldType(fieldType)[0] || '=';
        newGroup.get('operator')?.setValue(defaultOp);
        this.cdr.markForCheck(); 
    });

    this.customFilters.push(newGroup);
  }

  removeCustomFilter(index: number): void {
    this.customFilters.removeAt(index);
  }

  // --- Operator Logic ---
  getOperatorsForFieldType(fieldType: string | null | undefined): string[] {
    switch (fieldType) {
      case 'INTEGER':
      case 'REAL':
        return ['=', '!=', '>', '>=', '<', '<='];
      case 'BOOLEAN':
        return ['=', '!='];
      case 'TEXT':
        return ['=', '!=', 'LIKE']; 
      default:
        return ['=', '!=']; 
    }
  }

  getSelectedFieldType(index: number): string | null {
      const group = this.customFilters.at(index);
      return this.getSelectedFieldTypeForGroup(group);
  }

   private getSelectedFieldTypeForGroup(group: AbstractControl | null): string | null {
       if (!group) return null;
       const fieldName = group.get('field')?.value;
       const fieldDefinition = this.availableFilters.find(f => f.name === fieldName);
       return fieldDefinition ? fieldDefinition.type : null;
   }


  // --- Event Handlers ---
  applyFilters(): void {
    if (this.filterForm.invalid) {
        this.filterForm.markAllAsTouched(); 
        return;
    }
    this.currentPage = 1; // Reset to page 1
    this.manualFetchTrigger$.next(); 
  }

  // --- UI Setup ---
  private setupTableColumns(db: Database): void {
    // --- ADDED 'status' to standard columns ---
    let standardColumns = ['id', 'timestamp', 'filename', 'mime_type', 'filesize', 'status'];
    
    // --- UPDATED: Add dynamic columns based on content_type ---
    if (db.content_type === 'image') {
      standardColumns.push('width', 'height');
    } else if (db.content_type === 'audio') {
      standardColumns.push('duration_sec', 'channels');
    }

    const customColumns = db.custom_fields.map(field => field.name);
    
    // --- Use the stored this.currentUser ---
    const actionColumn = (this.currentUser && (this.currentUser.can_edit || this.currentUser.can_delete)) ? ['actions'] : [];

    // Add 'Preview' column
    this.tableColumns = ['Preview', ...standardColumns, ...customColumns, ...actionColumn];
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  // --- Modal Opening Methods (Event Handlers from Children) ---
  openUploadModal(): void {
    if (this.dbName) {
      // Use RENAMED modal ID
      this.modalService.open(UploadEntryModalComponent.MODAL_ID);
    } else {
      this.notificationService.showInfo('Please select a database first.');
    }
  }

  openEditModal(entry: Entry): void { // RENAMED
    if (this.dbName) {
      // --- ADDED: Check for processing status ---
      if (entry.status === 'processing') {
        this.notificationService.showError('Cannot edit an entry that is still processing.');
        return;
      }
      // --- END ADDED ---
      this.databaseService.selectEntry(entry); // RENAMED
      // Use RENAMED modal ID
      this.modalService.open(EditEntryModalComponent.MODAL_ID);
    }
  }

openDeleteConfirm(entry: Entry): void { // RENAMED
    if (!this.currentDb) return;

    // --- ADDED: Check for processing status ---
    if (entry.status === 'processing') {
      this.notificationService.showError('Cannot delete an entry that is still processing.');
      return;
    }
    // --- END ADDED ---

    const modalData: ConfirmationModalData = {
      message: `Are you sure you want to delete entry ${entry.id} from '${this.currentDb.name}'? This action cannot be undone.`
    };
    this.modalService.open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(
        take(1), 
        filter(confirmed => confirmed === true) 
      )
      .subscribe(() => {
        if (this.currentDb) { 
          // Use RENAMED service method
          this.databaseService.deleteEntry(this.currentDb.name, entry.id).subscribe();
        }
      });
  }

  openDetailModal(entry: Entry): void { // RENAMED
     if (this.dbName) {
        // --- ADDED: Check for processing status ---
        if (entry.status === 'processing') {
          this.notificationService.showInfo('This entry is still processing. Details will be available when complete.');
          return;
        }
        // --- END ADDED ---
        this.databaseService.selectEntry(entry); // RENAMED
        // Use RENAMED modal ID
        this.modalService.open(EntryDetailModalComponent.MODAL_ID);
     }
  }

  // --- Utility Methods ---
  getCustomFilterGroup(index: number): AbstractControl | null {
      return this.customFilters.at(index);
  }

  // --- PUBLIC HELPER METHODS ---
  public setViewMode(mode: 'grid' | 'list'): void {
    this.viewMode = mode;
    this.cdr.markForCheck();
  }

  // --- UPDATED: Backend pagination logic ---
  public nextPage(): void {
    if (this.hasNextPage) {
      this.currentPage++;
      this.manualFetchTrigger$.next(); 
    }
  }

  public prevPage(): void {
    if (this.currentPage > 1) {
      this.currentPage--;
      this.manualFetchTrigger$.next(); 
    }
  }
}