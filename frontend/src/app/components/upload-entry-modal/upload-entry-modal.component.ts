// frontend/src/app/components/upload-entry-modal/upload-entry-modal.component.ts
import { Component, OnDestroy, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, filter, finalize } from 'rxjs/operators';
import { Database, CustomField, Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';

@Component({
  selector: 'app-upload-entry-modal', // RENAMED
  templateUrl: './upload-entry-modal.component.html', // RENAMED
  styleUrls: ['./upload-entry-modal.component.css'], // RENAMED
  standalone: false,
})
export class UploadEntryModalComponent implements OnInit, OnDestroy { // RENAMED
  public static readonly MODAL_ID = 'uploadEntryModal'; // RENAMED
  uploadForm: FormGroup;
  isLoading = false;
  selectedFile: File | null = null;
  public selectedFileName: string | null = null;
  currentDatabase: Database | null = null;
  private destroy$ = new Subject<void>();
  
  // NEW: Property to hold the dynamic accept string
  public fileAcceptString: string | null = null;

  constructor(
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private modalService: ModalService
  ) {
    this.uploadForm = this.fb.group({}); // Initialize empty
  }

  ngOnInit(): void {
    // Listen for the selected database to change
    this.databaseService.selectedDatabase$
      .pipe(
        takeUntil(this.destroy$),
        filter((db): db is Database => db !== null)
      )
      .subscribe((db: Database) => {
        this.currentDatabase = db;
        this.initializeForm(); // Rebuild the form when the database changes
        this.updateFileAcceptString(db.content_type); // NEW: Update accept string
      });
  }

  /**
   * NEW: Sets the file input's 'accept' property based on content_type.
   */
  private updateFileAcceptString(contentType: 'image' | 'audio' | 'file'): void {
    if (contentType === 'image') {
      this.fileAcceptString = 'image/jpeg,image/png,image/gif,image/webp';
    } else if (contentType === 'audio') {
      this.fileAcceptString = 'audio/mpeg,audio/wav,audio/flac,audio/opus,audio/ogg';
    } else {
      this.fileAcceptString = null; // Allow all files
    }
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
   * Initializes or re-initializes the upload form based on the current database's schema.
   */
  private initializeForm(): void {
    // Clear any existing form controls
    Object.keys(this.uploadForm.controls).forEach(key => {
      this.uploadForm.removeControl(key);
    });

    // Add standard controls. Default the timestamp to the user's current local time.
    this.uploadForm.addControl('timestamp', this.fb.control(this.getLocalISOString(new Date()), Validators.required));
    // UPDATED: 'imageFile' to 'file'
    this.uploadForm.addControl('file', this.fb.control(null, Validators.required));
    this.selectedFileName = null;

    // Add controls for custom fields defined in the database
    if (this.currentDatabase) {
      this.currentDatabase.custom_fields.forEach((field: CustomField) => {
        // Set default for boolean checkbox to false
        const defaultValue = field.type === 'BOOLEAN' ? false : '';
        this.uploadForm.addControl(field.name, this.fb.control(defaultValue));
      });
    }
  }

  /**
   * Handles the file input change event to store the selected file.
   */
  onFileSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    const fileList: FileList | null = element.files;
    if (fileList && fileList.length > 0) {
      this.selectedFile = fileList[0];
      this.selectedFileName = this.selectedFile.name;
      // Update the form control value specifically for validation
      this.uploadForm.patchValue({ file: this.selectedFile }); // UPDATED: 'file'
      this.uploadForm.get('file')?.markAsTouched(); // Mark as touched after selection
    } else {
      this.selectedFile = null;
      this.selectedFileName = null;
      this.uploadForm.patchValue({ file: null }); // UPDATED: 'file'
    }
  }

  /**
   * Handles form submission by constructing the multipart/form-data payload and calling the service.
   * UPDATED: To handle new async flow.
   */
  onSubmit(): void {
    if (this.uploadForm.invalid || !this.currentDatabase || !this.selectedFile) {
      this.uploadForm.markAllAsTouched(); // Ensure validation messages show
      console.warn("Upload form invalid or missing data:", { invalid: this.uploadForm.invalid, db: !!this.currentDatabase, file: !!this.selectedFile });
      return;
    }

    this.isLoading = true;

    // Separate the file and timestamp from the rest of the form (which are custom fields)
    // UPDATED: 'imageFile' to 'file', added 'status' to destructuring
    const { timestamp, file, ...customFields } = this.uploadForm.value;

    // The `timestamp` from the form is a local time string. `new Date()` correctly parses this.
    // `.getTime()` returns the UTC milliseconds, which we convert to a Unix timestamp (seconds) for the backend.
    const metadata = {
        timestamp: Math.floor(new Date(timestamp).getTime() / 1000),
        ...customFields
    };

    // Make sure boolean values are actual booleans in the JSON payload
    this.currentDatabase.custom_fields.forEach(field => {
        if (field.type === 'BOOLEAN' && metadata.hasOwnProperty(field.name)) {
            metadata[field.name] = !!metadata[field.name]; // Convert checkbox value to true/false
        }
    });

    // Use RENAMED service method
    // UPDATED: Service now returns Observable<void> and handles notifications.
    this.databaseService.uploadEntry(this.currentDatabase.name, metadata, this.selectedFile)
      .pipe(finalize(() => this.isLoading = false))
      .subscribe({
        next: () => {
          // The service handles success/info notifications.
          // Just close the modal.
          this.closeModal();
        },
        error: () => {
          // The service's global handler will show the error toast.
          // Loading is finalized, form remains open for user to retry.
        }
      });
  }

  closeModal(): void {
    this.modalService.close();
    // Reset form on close
    if (this.currentDatabase) {
      this.initializeForm();
    }
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}