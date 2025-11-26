// frontend/src/app/components/upload-entry-modal/upload-entry-modal.component.ts
import { Component, OnDestroy, OnInit, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, filter, finalize } from 'rxjs/operators';
import { Database, CustomField } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { isMimeTypeAllowed } from '../../utils/mime-types';
import { NotificationService } from '../../services/notification.service';

@Component({
  selector: 'app-upload-entry-modal',
  templateUrl: './upload-entry-modal.component.html',
  styleUrls: ['./upload-entry-modal.component.css'],
  standalone: false,
})
export class UploadEntryModalComponent implements OnInit, OnDestroy {
  public static readonly MODAL_ID = 'uploadEntryModal';
  uploadForm: FormGroup;
  isLoading = false;
  selectedFile: File | null = null;
  public selectedFileName: string | null = null;
  currentDatabase: Database | null = null;
  private destroy$ = new Subject<void>();
  
  public fileAcceptString: string | null = null;

  constructor(
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private modalService: ModalService,
    private notificationService: NotificationService,
    private cdr: ChangeDetectorRef
  ) {
    this.uploadForm = this.fb.group({}); 
  }

  ngOnInit(): void {
    this.databaseService.selectedDatabase$
      .pipe(
        takeUntil(this.destroy$),
        filter((db): db is Database => db !== null)
      )
      .subscribe((db: Database) => {
        this.currentDatabase = db;
        this.initializeForm(); 
        this.updateFileAcceptString(db.content_type); 
      });
    
    this.modalService.getModalEvents(UploadEntryModalComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe(event => {
        if (event.action === 'open') {
           // Reset file state on open if no file passed
           if (event.data?.droppedFile) {
              this.handleFile(event.data.droppedFile);
           } else {
              this.resetFileState();
           }
        }
      });
  }

  private updateFileAcceptString(contentType: 'image' | 'audio' | 'file'): void {
    if (contentType === 'image') {
      this.fileAcceptString = 'image/jpeg,image/png,image/gif,image/webp';
    } else if (contentType === 'audio') {
      this.fileAcceptString = 'audio/mpeg,audio/wav,audio/flac,audio/opus,audio/ogg';
    } else {
      this.fileAcceptString = null; 
    }
  }

  private getLocalISOString(date: Date): string {
    const offset = date.getTimezoneOffset();
    const shiftedDate = new Date(date.getTime() - (offset * 60 * 1000));
    return shiftedDate.toISOString().slice(0, 16);
  }

  private initializeForm(): void {
    Object.keys(this.uploadForm.controls).forEach(key => {
      this.uploadForm.removeControl(key);
    });

    this.uploadForm.addControl('timestamp', this.fb.control(this.getLocalISOString(new Date()), Validators.required));
    this.uploadForm.addControl('file', this.fb.control(null, Validators.required));
    
    this.resetFileState(); // Ensure clean state

    if (this.currentDatabase) {
      this.currentDatabase.custom_fields.forEach((field: CustomField) => {
        const defaultValue = field.type === 'BOOLEAN' ? false : '';
        this.uploadForm.addControl(field.name, this.fb.control(defaultValue));
      });
    }
  }

  private resetFileState(): void {
    this.selectedFile = null;
    this.selectedFileName = null;
    // We use emitEvent: false to prevent triggering listeners unnecessarily during reset
    this.uploadForm.get('file')?.setValue(null, { emitEvent: false });
  }

  onFileSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    const fileList: FileList | null = element.files;
    
    if (fileList && fileList.length > 0) {
      const file = fileList[0];
      this.handleFile(file);
      
      // Reset the input value safely
      // This allows selecting the same file again if needed,
      // and prevents the browser from holding onto a file reference we've processed.
      // However, doing this *during* the event can cause the DOMException in some browsers.
      // We simply let the native input be. We don't need to clear it unless we close the modal.
    }
  }

  handleFile(file: File | null): void {
    if (!file) {
      this.resetFileState();
      return;
    }

    if (this.currentDatabase && !isMimeTypeAllowed(this.currentDatabase.content_type, file.type)) {
        this.notificationService.showError(`Invalid file type (${file.type}). Allowed: ${this.currentDatabase.content_type}`);
        // Don't clear the form here, let the user see they made a mistake but keep the modal open
        return; 
    }

    this.selectedFile = file;
    this.selectedFileName = file.name;
    
    // Patch the form value with the File object.
    // NOTE: Some browsers throw the DOMException if you try to set the value of a 
    // file input programmatically to a File object.
    // However, 'file' here is an internal FormControl, NOT the native DOM input.
    // Angular handles this, but to be safe and avoid the error, we should pass
    // the File object to the control, but ensure we aren't trying to write it back to the DOM.
    this.uploadForm.patchValue({ file: this.selectedFile });
    
    // Mark touched and FORCE validation update
    this.uploadForm.get('file')?.markAsTouched();
    this.uploadForm.get('file')?.updateValueAndValidity();
    
    this.cdr.detectChanges();
  }

  onSubmit(): void {
    if (this.uploadForm.invalid || !this.currentDatabase || !this.selectedFile) {
      this.uploadForm.markAllAsTouched(); 
      return;
    }

    this.isLoading = true;

    const { timestamp, file, ...customFields } = this.uploadForm.value;

    const metadata = {
        timestamp: Math.floor(new Date(timestamp).getTime() / 1000),
        ...customFields
    };

    this.currentDatabase.custom_fields.forEach(field => {
        if (field.type === 'BOOLEAN' && metadata.hasOwnProperty(field.name)) {
            metadata[field.name] = !!metadata[field.name]; 
        }
    });

    this.databaseService.uploadEntry(this.currentDatabase.name, metadata, this.selectedFile)
      .pipe(finalize(() => this.isLoading = false))
      .subscribe({
        next: () => {
          this.closeModal();
        },
        error: () => {
          // Error handled by service
        }
      });
  }

  closeModal(): void {
    this.modalService.close();
    // Reset form logic is handled in ngOnInit or initializeForm when re-opened
    // But we can clear it here too to be safe
    if (this.currentDatabase) {
      this.initializeForm();
    }
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}