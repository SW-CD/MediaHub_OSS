// frontend/src/app/components/database-settings/database-settings.component.ts
import { Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Observable, Subject, of } from 'rxjs';
import { switchMap, takeUntil, filter, take, finalize } from 'rxjs/operators';
import { Database, User, DatabaseConfig } from '../../models/api.models';
import { DatabaseService, DatabaseUpdatePayload } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { ConfirmationModalComponent, ConfirmationModalData } from '../confirmation-modal/confirmation-modal.component';

@Component({
  selector: 'app-database-settings',
  templateUrl: './database-settings.component.html',
  styleUrls: ['./database-settings.component.css'],
  standalone: false
})
export class DatabaseSettingsComponent implements OnInit, OnDestroy {
  public selectedDatabase$: Observable<Database | null>;
  public currentUser$: Observable<User | null>;
  public settingsForm: FormGroup;
  public isLoading = false;

  public currentDb: Database | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private route: ActivatedRoute,
    private databaseService: DatabaseService,
    private authService: AuthService,
    private modalService: ModalService,
    private fb: FormBuilder
  ) {
    this.selectedDatabase$ = this.databaseService.selectedDatabase$;
    this.currentUser$ = this.authService.currentUser$;

    // --- UPDATED: Renamed 'create_previews' to 'create_preview' ---
    this.settingsForm = this.fb.group({
      // Config fields
      convert_to_jpeg: [false],
      create_preview: [true], // RENAMED
      auto_conversion: ['none'],

      // Housekeeping fields
      housekeeping: this.fb.group({
        interval: ['', Validators.required],
        disk_space: ['', Validators.required],
        max_age: ['', Validators.required],
      }),
    });
  }

  ngOnInit(): void {
    // When the route parameter changes, select the new database
    this.route.paramMap
      .pipe(
        takeUntil(this.destroy$),
        switchMap((params) => {
          const name = params.get('name');
          // Clear form and set loading when navigating
          this.settingsForm.reset();
          this.isLoading = true; 
          return name ? this.databaseService.selectDatabase(name) : of(null);
        })
      )
      .subscribe(); // The tap in the service handles the subject update

    // When the selected database changes, patch the form
    this.selectedDatabase$
      .pipe(
        takeUntil(this.destroy$),
        // No filter here, handle null case explicitly
      )
      .subscribe((db) => {
          this.isLoading = false; // Stop loading once data (or null) arrives
          if (db) {
            this.currentDb = db;
            // --- UPDATED: Patch config object and housekeeping ---
            // This will now correctly patch 'create_preview'
            this.settingsForm.patchValue({
              ...db.config, // Patches 'create_preview'
              housekeeping: db.housekeeping,
            });
            this.updateFormEnabledState();
          } else {
            // Handle case where database is not found or cleared
            this.currentDb = null;
            this.settingsForm.reset(); // Clear form if no DB selected
            this.settingsForm.disable(); // Disable if no DB
          }
        });

    // When the user changes, update form editability
    this.currentUser$.pipe(takeUntil(this.destroy$)).subscribe(() => {
      this.updateFormEnabledState();
    });
  }

  private updateFormEnabledState(): void {
    const user = this.authService.getCurrentUser();
    // Also check if there's a currentDb to edit
    if (user && !user.can_edit || !this.currentDb) {
      this.settingsForm.disable();
    } else if (user && user.can_edit && this.currentDb) {
      this.settingsForm.enable();
    }
  }

  onSaveSettings(): void {
    if (this.settingsForm.invalid || !this.currentDb) {
      return;
    }
    this.isLoading = true;

    // --- UPDATED: Build dynamic config object ---
    const formValue = this.settingsForm.value;
    const config: DatabaseConfig = {};
    
    // Dynamically add config properties based on content type
    if (this.currentDb.content_type === 'image') {
      config.convert_to_jpeg = formValue.convert_to_jpeg;
      config.create_preview = formValue.create_preview; // RENAMED
    } else if (this.currentDb.content_type === 'audio') {
      config.auto_conversion = formValue.auto_conversion;
      config.create_preview = formValue.create_preview; // RENAMED
    }
    // 'file' type has no config

    const payload: DatabaseUpdatePayload = {
      config: config,
      housekeeping: this.settingsForm.get('housekeeping')?.value,
    };

    this.databaseService
      .updateDatabase(this.currentDb.name, payload)
      .pipe(finalize(() => (this.isLoading = false)))
      .subscribe(() => {
        this.settingsForm.markAsPristine(); // Mark form as saved
      });
  }

  onTriggerHousekeeping(): void {
    if (!this.currentDb) return;
    this.isLoading = true;
    this.databaseService
      .triggerHousekeeping(this.currentDb.name)
      .pipe(finalize(() => (this.isLoading = false)))
      .subscribe();
  }

  onDeleteDatabase(): void {
    if (!this.currentDb) return;

    const modalData: ConfirmationModalData = {
      title: 'Confirm Database Deletion',
      message: `This action will permanently delete the '${this.currentDb.name}' database and all entries within it. This action cannot be undone.`,
    };

    this.modalService
      .open(ConfirmationModalComponent.MODAL_ID, modalData)
      .pipe(
        take(1),
        filter((confirmed) => confirmed === true)
      )
      .subscribe(() => {
        if (this.currentDb) {
          this.isLoading = true;
          this.databaseService
            .deleteDatabase(this.currentDb.name)
            .pipe(finalize(() => (this.isLoading = false)))
            .subscribe();
        }
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}