import { Component, OnDestroy, OnInit, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Subject, firstValueFrom } from 'rxjs';
import { takeUntil, filter, switchMap } from 'rxjs/operators';
import { Database, CustomField } from '../../models'; 
import { EntryService } from '../../services/entry.service';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { isMimeTypeAllowed, getFileAcceptString } from '../../utils/mime-types';
import { NotificationService } from '../../services/notification.service';
import { AuthService } from '../../services/auth.service';
import { extractMetadata } from '../../utils/metadata-extractor';

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
  selectedFiles: File[] = [];
  isMultipleUpload = false;
  currentDatabase: Database | null = null;
  private destroy$ = new Subject<void>();
  
  public fileAcceptString: string | null = null;

  // Batch upload progress state
  public uploadProgressIndex = 0;
  public totalUploads = 0;
  public currentUploadingFileName = '';
  public uploadProgressPercent = 0;

  constructor(
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private entryService: EntryService,
    private modalService: ModalService,
    private notificationService: NotificationService,
    private cdr: ChangeDetectorRef,
    private authService: AuthService
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
        this.fileAcceptString = getFileAcceptString(db.content_type);
      });
    
    this.modalService.getModalEvents(UploadEntryModalComponent.MODAL_ID)
      .pipe(takeUntil(this.destroy$))
      .subscribe(event => {
        if (event.action === 'open') {
           this.resetProgressState();
           if (event.data?.files && event.data.files.length > 1) {
              this.handleMultipleFiles(event.data.files);
           } else if (event.data?.files && event.data.files.length === 1) {
              this.handleFile(event.data.files[0]);
           } else if (event.data?.file) {
              this.handleFile(event.data.file);
           } else if (event.data?.droppedFiles && event.data.droppedFiles.length > 1) {
              this.handleMultipleFiles(event.data.droppedFiles);
           } else if (event.data?.droppedFiles && event.data.droppedFiles.length === 1) {
              this.handleFile(event.data.droppedFiles[0]);
           } else if (event.data?.droppedFile) {
              this.handleFile(event.data.droppedFile);
           } else {
              this.resetFileState();
           }
        }
      });
  }

  private resetProgressState(): void {
    this.uploadProgressIndex = 0;
    this.totalUploads = 0;
    this.currentUploadingFileName = '';
    this.uploadProgressPercent = 0;
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
    this.selectedFiles = [];
    this.isMultipleUpload = false;
    this.resetProgressState();
    this.uploadForm.get('file')?.setValue(null, { emitEvent: false });
    this.uploadForm.get('timestamp')?.setValidators([Validators.required]);
    this.uploadForm.get('timestamp')?.updateValueAndValidity();
  }

  onFileSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    const fileList: FileList | null = element.files;
    
    if (fileList && fileList.length > 0) {
      const file = fileList[0];
      this.handleFile(file);
    }
  }

  onMultipleFilesSelected(event: Event): void {
    const element = event.currentTarget as HTMLInputElement;
    const fileList: FileList | null = element.files;
    if (fileList && fileList.length > 0) {
      const files = Array.from(fileList).filter(f => 
        this.currentDatabase ? isMimeTypeAllowed(this.currentDatabase.content_type, f.type) : true
      );
      if (files.length === 1) {
        this.handleFile(files[0]);
      } else if (files.length > 1) {
        this.handleMultipleFiles(files);
      }
    }
  }

  async handleFile(file: File | null): Promise<void> {
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
    this.selectedFiles = [];
    this.isMultipleUpload = false;
    
    this.uploadForm.patchValue({ file: this.selectedFile });
    
    this.uploadForm.get('file')?.markAsTouched();
    this.uploadForm.get('file')?.updateValueAndValidity();

    this.uploadForm.get('timestamp')?.setValidators([Validators.required]);
    this.uploadForm.get('timestamp')?.setValue(this.getLocalISOString(new Date()));
    this.uploadForm.get('timestamp')?.updateValueAndValidity();
    
    this.cdr.detectChanges();

    try {
      const ext = await extractMetadata(file);
      if (ext && ext.timestamp) {
        this.uploadForm.patchValue({ timestamp: this.getLocalISOString(ext.timestamp) });
        this.notificationService.showSuccess(`Extracted capture timestamp from file: ${ext.timestamp.toLocaleString()}`);
        this.cdr.detectChanges();
      }
    } catch (err) {
      console.error('[UploadModal] Error auto-extracting metadata:', err);
    }
  }

  handleMultipleFiles(files: File[]): void {
    this.selectedFiles = files;
    this.isMultipleUpload = true;
    this.selectedFile = null;
    this.selectedFileName = null;

    this.uploadForm.patchValue({ file: files[0] });
    this.uploadForm.get('file')?.updateValueAndValidity();

    this.uploadForm.get('timestamp')?.clearValidators();
    this.uploadForm.get('timestamp')?.updateValueAndValidity();
    
    this.cdr.detectChanges();
  }

  onSubmit(): void {
    if (this.uploadForm.invalid || !this.currentDatabase || (!this.selectedFile && this.selectedFiles.length === 0)) {
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
            
            if (field.type === 'BOOLEAN') {
                value = !!value; 
            } else if ((field.type === 'INTEGER' || field.type === 'REAL') && value !== '' && value !== null) {
                value = Number(value);
            }
            
            custom_fields[field.name] = value;
        }
    });

    // Pre-emptively ping the server to ensure our access token is fresh. 
    this.authService.fetchCurrentUser().pipe(
      filter(user => user !== null),
      switchMap(async () => {
        if (this.isMultipleUpload) {
          this.totalUploads = this.selectedFiles.length;
          this.uploadProgressIndex = 0;
          this.uploadProgressPercent = 0;

          for (let i = 0; i < this.selectedFiles.length; i++) {
            const f = this.selectedFiles[i];
            this.uploadProgressIndex = i + 1;
            this.currentUploadingFileName = f.name;
            this.uploadProgressPercent = Math.round((i / this.selectedFiles.length) * 100);
            this.cdr.detectChanges();

            const ext = await extractMetadata(f);
            const ts = ext?.timestamp ? ext.timestamp.getTime() : Date.now();
            const metadata = {
              timestamp: ts,
              filename: f.name,
              custom_fields: custom_fields
            };
            
            await firstValueFrom(this.entryService.uploadEntry(
              this.currentDatabase!.id, 
              metadata as any, 
              f,
              { skipRefresh: true, silentSuccess: true }
            ));

            this.uploadProgressPercent = Math.round(((i + 1) / this.selectedFiles.length) * 100);
            this.cdr.detectChanges();
          }

          this.entryService.triggerImageListRefresh();
          this.notificationService.showSuccess(`Successfully uploaded ${this.selectedFiles.length} entries.`);
        } else {
          const metadata = {
            timestamp: new Date(timestamp).getTime(),
            filename: this.selectedFile!.name,
            custom_fields: custom_fields 
          };
          await firstValueFrom(this.entryService.uploadEntry(this.currentDatabase!.id, metadata as any, this.selectedFile!));
        }
      })
    ).subscribe({
      next: () => {
        this.isLoading = false;
        this.resetProgressState();
        this.closeModal();
      },
      error: (err) => {
        this.isLoading = false;
        if (this.isMultipleUpload) {
          this.entryService.triggerImageListRefresh();
        }
      }
    });
  }

  closeModal(): void {
    if (this.isLoading) return;
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