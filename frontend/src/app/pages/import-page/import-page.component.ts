import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { HttpEvent, HttpEventType, HttpResponse } from '@angular/common/http';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

// Corrected default imports to resolve TypeScript construction errors
import JSZip from 'jszip';
import Papa from 'papaparse';

import { Database } from '../../models';
import { DatabaseService } from '../../services/database.service';
import { EntryService } from '../../services/entry.service';
import { NotificationService } from '../../services/notification.service';

@Component({
  selector: 'app-import-page',
  templateUrl: './import-page.component.html',
  styleUrls: ['./import-page.component.css'],
  standalone: false
})
export class ImportPageComponent implements OnInit, OnDestroy {
  public currentDatabase: Database | null = null;
  private destroy$ = new Subject<void>();

  // --- Native Stepper State ---
  public currentStep = 1;

  // Forms
  public configStepForm: FormGroup;
  public mappingStepForm: FormGroup;

  // File & CSV State
  public selectedFile: File | null = null;
  public isParsingZip = false;
  public customCsvHeaders: string[] = [];
  
  private readonly STANDARD_HEADERS = [
    'id', 'filename', 'timestamp', 'filesize', 'previewsize', 'mime_type', 'status', 'width', 'height', 'duration', 'channels'
  ];

  // Upload State
  public isUploading = false;
  public uploadProgress = 0;
  public isUploadComplete = false;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private entryService: EntryService,
    private notificationService: NotificationService,
    private cdr: ChangeDetectorRef
  ) {
    this.configStepForm = this.fb.group({
      mode: ['generate_new', Validators.required],
      unmapped_fields: ['ignore', Validators.required]
    });

    this.mappingStepForm = this.fb.group({});
  }

  ngOnInit(): void {
    const dbId = this.route.snapshot.paramMap.get('id');
    if (dbId) {
      this.databaseService.selectDatabase(dbId)
        .pipe(takeUntil(this.destroy$))
        .subscribe(db => {
          this.currentDatabase = db;
          this.cdr.markForCheck();
        });
    }
  }

  // --- Stepper Navigation ---
  
  public nextStep(): void {
    this.currentStep++;
  }

  public prevStep(): void {
    this.currentStep--;
  }

  // --- File Handling ---

  /**
   * Safely handles the raw browser event from the <input type="file"> element.
   */
  public onFileInputChange(event: Event): void {
    const element = event.target as HTMLInputElement;
    const fileList: FileList | null = element.files;
    
    if (fileList && fileList.length > 0) {
      this.onFileSelected(fileList[0]);
    }
    
    // Optional: Reset the input value so the user can select the same file again if they remove it
    element.value = '';
  }

  public onFileSelected(file: File): void {
    if (!file.name.toLowerCase().endsWith('.zip')) {
      this.notificationService.showError('Please select a valid .zip archive.');
      return;
    }

    this.selectedFile = file;
    this.isParsingZip = true;
    this.cdr.detectChanges();

    const zip = new JSZip();
    
    // Using explicit types in the promise chain to satisfy strict TypeScript rules
    zip.loadAsync(file).then((loadedZip: JSZip) => {
      const csvFile = loadedZip.file('entries.csv');
      
      if (!csvFile) {
        this.notificationService.showError('The archive does not contain an entries.csv file in the root folder.');
        this.resetFile();
        return;
      }
      return csvFile.async('text');
    }).then((csvText: string | undefined) => {
      if (csvText) {
        this.extractCsvHeaders(csvText);
      }
    }).catch((err: Error) => {
      console.error('Failed to parse ZIP:', err);
      this.notificationService.showError('Failed to read the ZIP archive. It might be corrupted.');
      this.resetFile();
    });
  }

  public resetFile(): void {
    this.selectedFile = null;
    this.isParsingZip = false;
    this.customCsvHeaders = [];
    this.currentStep = 1;
    this.cdr.detectChanges();
  }

  private extractCsvHeaders(csvText: string): void {
    Papa.parse(csvText, {
      header: true,
      preview: 1,
      skipEmptyLines: true,
      complete: (results) => {
        const allHeaders = results.meta.fields || [];
        this.customCsvHeaders = allHeaders.filter(
          header => !this.STANDARD_HEADERS.includes(header.toLowerCase())
        );

        this.buildMappingForm();
        this.isParsingZip = false;
        this.cdr.detectChanges();
      }
    });
  }

  private buildMappingForm(): void {
    Object.keys(this.mappingStepForm.controls).forEach(key => {
      this.mappingStepForm.removeControl(key);
    });

    this.customCsvHeaders.forEach(csvHeader => {
      this.mappingStepForm.addControl(csvHeader, new FormControl(''));
    });
  }

  // --- Upload Logic ---

  public startImport(): void {
    if (!this.currentDatabase || !this.selectedFile) return;

    this.isUploading = true;
    this.uploadProgress = 0;

    const custom_field_mapping: Record<string, string> = {};
    const mappingValues = this.mappingStepForm.value;
    
    Object.keys(mappingValues).forEach(csvHeader => {
      if (mappingValues[csvHeader]) {
        custom_field_mapping[csvHeader] = mappingValues[csvHeader];
      }
    });

    const config = {
      mode: this.configStepForm.value.mode,
      unmapped_fields: this.configStepForm.value.unmapped_fields,
      custom_field_mapping: custom_field_mapping
    };

    this.entryService.importEntries(this.currentDatabase.id, this.selectedFile, config)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (event: HttpEvent<any>) => {
          if (event.type === HttpEventType.UploadProgress && event.total) {
            this.uploadProgress = Math.round((100 * event.loaded) / event.total);
            this.cdr.markForCheck();
          } else if (event instanceof HttpResponse) {
            this.isUploading = false;
            this.isUploadComplete = true;
            this.notificationService.showSuccess('Archive uploaded successfully!');
            this.cdr.markForCheck();
          }
        },
        error: (err) => {
          this.isUploading = false;
          this.cdr.markForCheck();
          
          // Try to extract a meaningful error message from the backend response
          const errorMessage = err.error?.message || err.error?.error || err.message || 'An unexpected error occurred during the upload.';
          
          // Display the error to the user
          this.notificationService.showError(`Upload failed: ${errorMessage}`);
        }
      });
  }

  public navigateBack(): void {
    if (this.currentDatabase) {
      this.router.navigate(['/dashboard/settings', this.currentDatabase.id]);
    }
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}