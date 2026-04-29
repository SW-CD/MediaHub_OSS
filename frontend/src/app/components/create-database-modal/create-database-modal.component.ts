import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, FormArray, Validators } from '@angular/forms';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { finalize } from 'rxjs/operators';
import { DatabaseConfig } from '../../models';

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
      // UPDATED: Removed the strict Regex pattern since name is now just a display label!
      // Added a maxLength just for standard safety.
      name: ['', [Validators.required, Validators.maxLength(100)]],
      content_type: ['image', Validators.required],
      create_preview: [true],
      auto_conversion: [''], 

      housekeeping: this.fb.group({
        interval_value: [1, [Validators.required, Validators.min(0)]],
        interval_unit: ['h'],
        disk_space_value: [100, [Validators.required, Validators.min(0)]],
        disk_space_unit: ['G'],
        max_age_value: [365, [Validators.required, Validators.min(0)]],
        max_age_unit: ['d'],
      }),
      custom_fields: this.fb.array([]),
    });
  }

  ngOnInit(): void {
    this.createDbForm.get('content_type')?.valueChanges.subscribe(() => {
        this.createDbForm.get('auto_conversion')?.setValue('');
    });
  }

  get customFields(): FormArray {
    return this.createDbForm.get('custom_fields') as FormArray;
  }

  addCustomField(): void {
    const fieldGroup = this.fb.group({
      name: ['', [Validators.required, Validators.pattern(/^[a-zA-Z_][a-zA-Z0-9_]*$/)]],
      type: ['TEXT', Validators.required],
    });
    this.customFields.push(fieldGroup);
  }

  removeCustomField(index: number): void {
    this.customFields.removeAt(index);
  }

  onSubmit(): void {
    if (this.createDbForm.invalid) {
      this.createDbForm.markAllAsTouched();
      return;
    }

    this.isLoading = true;
    const formValue = this.createDbForm.value;

    const config: DatabaseConfig = {};
    if (['image', 'audio', 'video'].includes(formValue.content_type)) {
      config.create_preview = formValue.create_preview;
      config.auto_conversion = formValue.auto_conversion;
    }

    const hkForm = formValue.housekeeping;
    const reconstructedHousekeeping = {
      interval: hkForm.interval_value > 0 ? `${hkForm.interval_value}${hkForm.interval_unit}` : "0",
      disk_space: hkForm.disk_space_value > 0 ? `${hkForm.disk_space_value}${hkForm.disk_space_unit}` : "0",
      max_age: hkForm.max_age_value > 0 ? `${hkForm.max_age_value}${hkForm.max_age_unit}` : "0",
    };

    const payload = {
      name: formValue.name,
      content_type: formValue.content_type,
      config: config, 
      housekeeping: reconstructedHousekeeping, 
      custom_fields: formValue.custom_fields,
    };

    this.databaseService.createDatabase(payload)
      .pipe(
        finalize(() => this.isLoading = false)
      )
      .subscribe({
        next: () => {
          this.modalService.close(true); 
          
          this.createDbForm.reset({
            name: '',
            content_type: 'image',
            create_preview: true, 
            auto_conversion: '',
            housekeeping: { 
              interval_value: 1, interval_unit: 'h', 
              disk_space_value: 100, disk_space_unit: 'G', 
              max_age_value: 365, max_age_unit: 'd' 
            } 
          });
          this.customFields.clear();
        }
      });
  }

  closeModal(): void {
    this.modalService.close(false);
  }
}