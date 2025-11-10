// frontend/src/app/components/create-database-modal/create-database-modal.component.ts
import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, FormArray, Validators } from '@angular/forms';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { finalize } from 'rxjs/operators';
import { DatabaseConfig } from '../../models/api.models';

@Component({
  selector: 'app-create-database-modal',
  templateUrl: './create-database-modal.component.html',
  styleUrls: ['./create-database-modal.component.css'],
  standalone: false,
})
export class CreateDatabaseModalComponent implements OnInit {
  public static readonly MODAL_ID = 'createDatabaseModal';
  createDbForm: FormGroup;
  isLoading = false;

  constructor(
    private fb: FormBuilder,
    private databaseService: DatabaseService,
    private modalService: ModalService
  ) {
    this.createDbForm = this.fb.group({
      name: ['', [Validators.required, Validators.pattern(/^[a-zA-Z0-9_]+$/)]],
      // NEW: Content type selector
      content_type: ['image', Validators.required],
      
      // Config-related fields
      convert_to_jpeg: [false],
      // UPDATED: Renamed 'create_previews' to 'create_preview'
      create_preview: [true], // RENAMED
      auto_conversion: ['none'],

      // Housekeeping and Custom Fields (unchanged)
      housekeeping: this.fb.group({
        interval: ['10m', Validators.required], 
        disk_space: ['100G', Validators.required],
        max_age: ['365d', Validators.required],
      }),
      custom_fields: this.fb.array([]),
    });
  }

  ngOnInit(): void {
    // ngOnInit is now free for other logic if needed.
  }

  // Getter for easy access to the custom_fields FormArray
  get customFields(): FormArray {
    return this.createDbForm.get('custom_fields') as FormArray;
  }

  /**
   * Adds a new, empty custom field group to the FormArray.
   */
  addCustomField(): void {
    const fieldGroup = this.fb.group({
      name: ['', [Validators.required, Validators.pattern(/^[a-zA-Z_][a-zA-Z0-9_]*$/)]],
      type: ['TEXT', Validators.required],
    });
    this.customFields.push(fieldGroup);
  }

  /**
   * Removes a custom field from the FormArray at a given index.
   */
  removeCustomField(index: number): void {
    this.customFields.removeAt(index);
  }

  /**
   * Handles the form submission.
   * UPDATED: To build the dynamic config object.
   */
  onSubmit(): void {
    if (this.createDbForm.invalid) {
      this.createDbForm.markAllAsTouched(); // Show validation errors
      return;
    }

    this.isLoading = true;
    
    const formValue = this.createDbForm.value;

    // 1. Build the dynamic config object based on content_type
    const config: DatabaseConfig = {};
    if (formValue.content_type === 'image') {
      config.convert_to_jpeg = formValue.convert_to_jpeg;
      // UPDATED: Read from 'create_preview'
      config.create_preview = formValue.create_preview; // RENAMED
    } else if (formValue.content_type === 'audio') {
      config.auto_conversion = formValue.auto_conversion;
      // UPDATED: Read from 'create_preview'
      config.create_preview = formValue.create_preview; // RENAMED
    }
    // 'file' content_type has no config options

    // 2. Build the final payload for the API
    const payload = {
      name: formValue.name,
      content_type: formValue.content_type,
      config: config, // The new dynamic config object
      housekeeping: formValue.housekeeping,
      custom_fields: formValue.custom_fields,
    };

    this.databaseService.createDatabase(payload)
      .pipe(
        finalize(() => this.isLoading = false)
      )
      .subscribe({
        next: () => {
          this.modalService.close(true); // Signal success
          // Reset form with defaults
          this.createDbForm.reset({
            name: '',
            content_type: 'image',
            convert_to_jpeg: false,
            // UPDATED: Reset 'create_preview'
            create_preview: true, // RENAMED
            auto_conversion: 'none',
            housekeeping: { interval: '10m', disk_space: '100G', max_age: '365d' } 
          });
          this.customFields.clear(); // Clear the custom fields array separately
        },
        // Errors are handled by the service
      });
  }

  closeModal(): void {
    this.modalService.close(false); // Signal cancellation
  }
}

