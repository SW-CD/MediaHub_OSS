import { Component, OnDestroy, OnInit, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil, filter, finalize } from 'rxjs/operators';
import { Database, CustomField } from '../../models'; 
import { EntryService } from '../../services/entry.service';
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
    private enryService: EntryService,
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
           if (event.data?.droppedFile) {
              this.handleFile(event.data.droppedFile);
           } else {
              this.resetFileState();
           }
        }
      });
  }

  private updateFileAcceptString(contentType: string): void {
    if (contentType === 'image') {
      this.fileAcceptString = 'image/jpeg,image/png,image/gif,image/webp';
    } else if (contentType === 'audio') {
      this.fileAcceptString = 'audio/mpeg,audio/wav,audio/flac,audio/opus,audio/ogg';
    } else if (contentType === 'video') {
      this.fileAcceptString = 'video/mp4,video/x-matroska,video/webm,video/ogg,video/quicktime,video/x-msvideo,video/x-flv';
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
    
    this.resetFileState(); 

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
    this.uploadForm.get('file')?.setValue(null, { emitEvent: false });
  }

  onFileSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    const fileList: FileList | null = element.files;
    
    if (fileList && fileList.length > 0) {
      const file = fileList[0];
      this.handleFile(file);
    }
  }

  handleFile(file: File | null): void {
    if (!file) {
      this.resetFileState();
      return;
    }

    if (this.currentDatabase && !isMimeTypeAllowed(this.currentDatabase.content_type, file.type)) {
        this.notificationService.showError(`Invalid file type (${file.type}). Allowed: ${this.currentDatabase.content_type}`);
        return; 
    }

    this.selectedFile = file;
    this.selectedFileName = file.name;
    
    this.uploadForm.patchValue({ file: this.selectedFile });
    
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

    // Destructure to separate the core fields from the dynamic custom fields
    const { timestamp, file, ...rawCustomFields } = this.uploadForm.value;

    const custom_fields: Record<string, any> = {};

    this.currentDatabase.custom_fields.forEach(field => {
        if (rawCustomFields.hasOwnProperty(field.name)) {
            let value = rawCustomFields[field.name];
            
            // Strictly enforce data types based on the schema before sending to the backend
            if (field.type === 'BOOLEAN') {
                value = !!value; 
            } else if ((field.type === 'INTEGER' || field.type === 'REAL') && value !== '' && value !== null) {
                value = Number(value);
            }
            
            custom_fields[field.name] = value;
        }
    });

    const metadata = {
        timestamp: new Date(timestamp).getTime(),
        filename: this.selectedFile.name,
        custom_fields: custom_fields 
    };

    // UPDATED: Passing this.currentDatabase.id instead of .name
    this.enryService.uploadEntry(this.currentDatabase.id, metadata as any, this.selectedFile)
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