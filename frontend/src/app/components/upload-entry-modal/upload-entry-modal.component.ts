// frontend/src/app/components/upload-entry-modal/upload-entry-modal.component.ts
import { Component, OnDestroy, OnInit, ChangeDetectorRef } from '@angular/core'; // Added ChangeDetectorRef
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
    private cdr: ChangeDetectorRef // Inject ChangeDetectorRef
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
        if (event.action === 'open' && event.data?.droppedFile) {
           this.handleFile(event.data.droppedFile);
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
    this.selectedFileName = null;
    this.selectedFile = null; // Reset file

    if (this.currentDatabase) {
      this.currentDatabase.custom_fields.forEach((field: CustomField) => {
        const defaultValue = field.type === 'BOOLEAN' ? false : '';
        this.uploadForm.addControl(field.name, this.fb.control(defaultValue));
      });
    }
  }

  onFileSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    const fileList: FileList | null = element.files;
    if (fileList && fileList.length > 0) {
      this.handleFile(fileList[0]);
    } else {
      this.handleFile(null);
    }
  }

  handleFile(file: File | null): void {
    if (!file) {
      this.selectedFile = null;
      this.selectedFileName = null;
      this.uploadForm.patchValue({ file: null });
      return;
    }

    if (this.currentDatabase && !isMimeTypeAllowed(this.currentDatabase.content_type, file.type)) {
        this.notificationService.showError(`Invalid file type (${file.type}). Allowed: ${this.currentDatabase.content_type}`);
        return; 
    }

    this.selectedFile = file;
    this.selectedFileName = file.name;
    
    // Patch the form value
    this.uploadForm.patchValue({ file: this.selectedFile });
    
    // Mark touched and FORCE validation update
    this.uploadForm.get('file')?.markAsTouched();
    this.uploadForm.get('file')?.updateValueAndValidity();
    
    // Manually trigger change detection to ensure the button state updates in the view
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
    if (this.currentDatabase) {
      this.initializeForm();
    }
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}