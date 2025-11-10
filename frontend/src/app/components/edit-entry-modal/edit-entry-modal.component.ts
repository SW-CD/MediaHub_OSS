// frontend/src/app/components/edit-entry-modal/edit-entry-modal.component.ts
// filepath: frontend/src/app/components/edit-entry-modal/edit-entry-modal.component.ts
import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, filter, finalize } from 'rxjs/operators';
// UPDATED: Image to Entry
import { Database, Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';

@Component({
  selector: 'app-edit-entry-modal', // RENAMED
  templateUrl: './edit-entry-modal.component.html', // RENAMED
  styleUrls: ['./edit-entry-modal.component.css'], // RENAMED
  standalone: false,
})
export class EditEntryModalComponent implements OnInit, OnDestroy { // RENAMED
  public static readonly MODAL_ID = 'editEntryModal'; // RENAMED
  editForm: FormGroup;
  isLoading = false;

  public currentDatabase: Database | null = null;
  public currentEntry: Entry | null = null; // RENAMED
  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private modalService: ModalService
  ) {
    this.editForm = this.fb.group({}); // Initialize empty
  }

  ngOnInit(): void {
    // Subscribe to database changes to build the form structure
    this.databaseService.selectedDatabase$
      .pipe(
        takeUntil(this.destroy$),
        filter((db): db is Database => !!db)
      )
      .subscribe(db => {
        this.currentDatabase = db;
        this.initializeForm(db);
        // If an entry was already selected, patch the newly initialized form
        if (this.currentEntry) {
            this.patchForm(this.currentEntry);
        }
      });

    // Subscribe to entry changes to populate the form
    // RENAMED: selectedImage$ to selectedEntry$
    this.databaseService.selectedEntry$
      .pipe(
        takeUntil(this.destroy$),
        filter((entry): entry is Entry => !!entry && !!this.currentDatabase && this.editForm.controls['timestamp'] != null) // Ensure form is initialized
      )
      .subscribe(entry => {
        this.currentEntry = entry;
        this.patchForm(entry);
      });
  }

  /**
   * Converts a Date object to a string format suitable for a `datetime-local` input (`YYYY-MM-DDTHH:mm`).
   * This uses the browser's local time, not UTC, which is what the input expects.
   */
  private getLocalISOString(date: Date): string {
    const offset = date.getTimezoneOffset();
    const shiftedDate = new Date(date.getTime() - (offset * 60 * 1000));
    return shiftedDate.toISOString().slice(0, 16);
  }

  /**
   * Dynamically builds the form controls based on the database's schema.
   * UPDATED: To add standard fields based on content_type.
   */
  private initializeForm(db: Database): void {
    // Clear existing controls
    Object.keys(this.editForm.controls).forEach(key => {
      this.editForm.removeControl(key);
    });

    // Add standard controls
    this.editForm.addControl('timestamp', this.fb.control('', Validators.required));
    this.editForm.addControl('filename', this.fb.control('')); // <-- ADDED

    // NEW: Add dynamic standard controls based on content_type
    if (db.content_type === 'image') {
      this.editForm.addControl('width', this.fb.control(0, Validators.required));
      this.editForm.addControl('height', this.fb.control(0, Validators.required));
    } else if (db.content_type === 'audio') {
      this.editForm.addControl('duration_sec', this.fb.control(0, Validators.required));
      this.editForm.addControl('channels', this.fb.control(null));
    }
    // 'file' type has no extra standard fields to edit

    // Add controls for custom fields
    db.custom_fields.forEach(field => {
      // Default boolean to false
      const defaultValue = field.type === 'BOOLEAN' ? false : '';
      this.editForm.addControl(field.name, this.fb.control(defaultValue));
    });
  }

  /**
   * Populates the form with data from the selected entry.
   */
  private patchForm(entry: Entry): void {
    if (!this.editForm) return; // Guard against race conditions

    // Convert the UTC Unix timestamp to a local time string for the input
    const localDateTime = this.getLocalISOString(new Date(entry.timestamp * 1000));

    const patchData: { [key: string]: any } = {
      timestamp: localDateTime,
      filename: entry.filename ?? '' // <-- ADDED
    };

    // NEW: Add standard fields to patchData
    if (this.currentDatabase?.content_type === 'image') {
      patchData['width'] = entry.width;
      patchData['height'] = entry.height;
    } else if (this.currentDatabase?.content_type === 'audio') {
      patchData['duration_sec'] = entry.duration_sec;
      patchData['channels'] = entry.channels;
    }

    // Populate custom field values
    this.currentDatabase?.custom_fields.forEach(field => {
       // Convert backend 0/1 to boolean for select
       if (field.type === 'BOOLEAN') {
           patchData[field.name] = entry[field.name] === 1 || entry[field.name] === true;
       } else {
           patchData[field.name] = entry[field.name] ?? ''; // Use nullish coalescing for default
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
    // The `timestamp` from the form is a local time string. `new Date()` correctly parses this.
    // `.getTime()` returns the UTC milliseconds, which we convert to a Unix timestamp (seconds) for the backend.
    const updates = {
        ...formValue,
        timestamp: Math.floor(new Date(formValue.timestamp).getTime() / 1000)
    };

    // Ensure boolean select values are sent as booleans
    this.currentDatabase.custom_fields.forEach(field => {
      if (field.type === 'BOOLEAN' && updates.hasOwnProperty(field.name)) {
        updates[field.name] = !!updates[field.name]; // Convert truthy/falsy to true/false
      }
    });

    // Use RENAMED service method
    this.databaseService.updateEntry(this.currentDatabase.name, this.currentEntry.id, updates)
      .pipe(finalize(() => this.isLoading = false))
      .subscribe(() => {
        this.modalService.close(true); // Signal success
        // Service triggers list refresh
      });
  }

  closeModal(): void {
    this.modalService.close(false); // Signal cancellation
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}