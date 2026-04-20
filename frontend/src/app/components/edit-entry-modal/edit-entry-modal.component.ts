import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, filter, finalize } from 'rxjs/operators';
import { Database, Entry } from '../../models';
import { EntryService } from '../../services/entry.service';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';

@Component({
  selector: 'app-edit-entry-modal',
  templateUrl: './edit-entry-modal.component.html',
  styleUrls: ['./edit-entry-modal.component.css'],
  standalone: false,
})
export class EditEntryModalComponent implements OnInit, OnDestroy {
  public static readonly MODAL_ID = 'editEntryModal';
  editForm: FormGroup;
  isLoading = false;

  public currentDatabase: Database | null = null;
  public currentEntry: Entry | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private entryService: EntryService,
    private modalService: ModalService
  ) {
    // FIXED: Removed media_fields from the form structure entirely
    this.editForm = this.fb.group({
      timestamp: ['', Validators.required],
      filename: [''],
      custom_fields: this.fb.group({})
    }); 
  }

  ngOnInit(): void {
    this.databaseService.selectedDatabase$
      .pipe(
        takeUntil(this.destroy$),
        filter((db): db is Database => !!db)
      )
      .subscribe(db => {
        this.currentDatabase = db;
        this.initializeForm(db);
        
        if (this.currentEntry) {
            this.patchForm(this.currentEntry);
        }
      });

    this.entryService.selectedEntry$
      .pipe(
        takeUntil(this.destroy$),
        filter((entry): entry is Entry => !!entry && !!this.currentDatabase)
      )
      .subscribe(entry => {
        this.currentEntry = entry;
        this.patchForm(entry);
      });
  }

  private getLocalISOString(date: Date): string {
    const offset = date.getTimezoneOffset();
    const shiftedDate = new Date(date.getTime() - (offset * 60 * 1000));
    return shiftedDate.toISOString().slice(0, 16);
  }

  /**
   * Dynamically builds nested form controls based on the database's schema.
   */
  private initializeForm(db: Database): void {
    const customGroup = this.fb.group({});

    // Add custom fields
    db.custom_fields.forEach(field => {
      const defaultValue = field.type === 'BOOLEAN' ? false : '';
      customGroup.addControl(field.name, this.fb.control(defaultValue));
    });

    this.editForm.setControl('custom_fields', customGroup);
  }

  /**
   * Populates the nested form with data from the selected entry.
   */
  private patchForm(entry: Entry): void {
    if (!this.editForm || !this.currentDatabase) return;

    // FIXED: Removed the * 1000 multiplier, as timestamps are already in milliseconds
    const localDateTime = this.getLocalISOString(new Date(entry.timestamp));

    const patchData: any = {
      timestamp: localDateTime,
      filename: entry.filename ?? '',
      custom_fields: {}
    };

    // Safely map custom fields if they exist
    this.currentDatabase.custom_fields.forEach(field => {
       if (entry.custom_fields && entry.custom_fields[field.name] !== undefined) {
         let val = entry.custom_fields[field.name];
         if (field.type === 'BOOLEAN') {
           val = (val === 1 || val === true);
         }
         patchData.custom_fields[field.name] = val;
       }
    });

    this.editForm.patchValue(patchData);
  }

  onSubmit(): void {
    if (this.editForm.invalid || !this.currentDatabase || !this.currentEntry) {
      return;
    }

    this.isLoading = true;
    const formValue = this.editForm.value;

    // Build the clean update payload without media_fields
    const updates: any = {
        filename: formValue.filename,
        // FIXED: Removed the / 1000 divisor to keep the payload in milliseconds
        timestamp: new Date(formValue.timestamp).getTime(),
        custom_fields: { ...formValue.custom_fields }
    };

    // Ensure correct data types for the backend
    this.currentDatabase.custom_fields.forEach(field => {
      const val = updates.custom_fields[field.name];
      if (field.type === 'BOOLEAN') {
        updates.custom_fields[field.name] = !!val;
      } else if ((field.type === 'INTEGER' || field.type === 'REAL') && val !== '' && val !== null) {
        updates.custom_fields[field.name] = Number(val);
      }
    });

    this.entryService.updateEntry(this.currentDatabase.name, this.currentEntry.id, updates)
      .pipe(finalize(() => this.isLoading = false))
      .subscribe(() => {
        this.modalService.close(true); 
      });
  }

  closeModal(): void {
    this.modalService.close(false); 
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}