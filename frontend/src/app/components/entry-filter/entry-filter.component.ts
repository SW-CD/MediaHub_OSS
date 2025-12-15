import { Component, Input, Output, EventEmitter, OnChanges, SimpleChanges, ChangeDetectionStrategy, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, FormArray, Validators, AbstractControl } from '@angular/forms';
import { SearchFilter } from '../../models/api.models';

export interface FilterChangedEvent {
  filter: SearchFilter | undefined;
  limit: number;
}

export interface AvailableFilter {
  name: string;
  type: 'TEXT' | 'INTEGER' | 'REAL' | 'BOOLEAN';
}

@Component({
  selector: 'app-entry-filter',
  templateUrl: './entry-filter.component.html',
  styleUrls: ['./entry-filter.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: false
})
export class EntryFilterComponent implements OnChanges {
  @Input() availableFilters: AvailableFilter[] = [];
  @Input() isLoading = false;
  
  @Output() filterApplied = new EventEmitter<FilterChangedEvent>();

  public filterForm: FormGroup;

  constructor(private fb: FormBuilder, private cdr: ChangeDetectorRef) {
    this.filterForm = this.fb.group({
      limitPerPage: [24, [Validators.required, Validators.min(1)]],
      tstart: [''],
      tend: [''],
      customFilters: this.fb.array([])
    });
  }

  ngOnChanges(changes: SimpleChanges): void {
    // If the available filters change (e.g. DB changed), reset the custom filters
    if (changes['availableFilters'] && !changes['availableFilters'].firstChange) {
      this.customFilters.clear();
      this.filterForm.patchValue({ tstart: '', tend: '' });
    }
  }

  get customFilters(): FormArray {
    return this.filterForm.get('customFilters') as FormArray;
  }

  addCustomFilter(): void {
    const newGroup = this.fb.group({
      field: ['', Validators.required],
      operator: ['='],
      value: ['', Validators.required]
    });

    // Update operator when field changes
    newGroup.get('field')?.valueChanges.subscribe(() => {
      const fieldType = this.getSelectedFieldTypeForGroup(newGroup);
      const defaultOp = this.getOperatorsForFieldType(fieldType)[0] || '=';
      newGroup.get('operator')?.setValue(defaultOp);
    });

    this.customFilters.push(newGroup);
  }

  removeCustomFilter(index: number): void {
    this.customFilters.removeAt(index);
  }

  applyFilters(): void {
    if (this.filterForm.invalid) {
      this.filterForm.markAllAsTouched();
      return;
    }

    const event = this.buildFilterEvent();
    this.filterApplied.emit(event);
  }

  private buildFilterEvent(): FilterChangedEvent {
    const formValue = this.filterForm.value;
    const conditions: SearchFilter[] = [];

    // 1. Time Filters
    if (formValue.tstart) {
      const tstartUnix = this.datetimeLocalToUnix(formValue.tstart);
      if (tstartUnix !== null) {
        conditions.push({ field: 'timestamp', operator: '>=', value: tstartUnix });
      }
    }
    if (formValue.tend) {
      const tendUnix = this.datetimeLocalToUnix(formValue.tend);
      if (tendUnix !== null) {
        conditions.push({ field: 'timestamp', operator: '<=', value: tendUnix });
      }
    }

    // 2. Custom Filters
    formValue.customFilters.forEach((filter: any) => {
      if (filter.field && filter.value !== null && String(filter.value).trim() !== '') {
        const fieldDefinition = this.availableFilters.find(f => f.name === filter.field);
        let filterValue: any = String(filter.value).trim();

        if (fieldDefinition) {
          if (fieldDefinition.type === 'INTEGER' || fieldDefinition.type === 'REAL') {
            const num = Number(filterValue);
            if (!isNaN(num)) { filterValue = num; }
          } else if (fieldDefinition.type === 'BOOLEAN') {
            const lowerVal = filterValue.toLowerCase();
            if (lowerVal === 'true' || lowerVal === '1') { filterValue = true; }
            else if (lowerVal === 'false' || lowerVal === '0') { filterValue = false; }
          }
        }
        conditions.push({
          field: filter.field,
          operator: filter.operator,
          value: filterValue
        });
      }
    });

    let filterObject: SearchFilter | undefined;
    if (conditions.length > 0) {
      filterObject = {
        operator: 'and',
        conditions: conditions
      };
    }

    return {
      filter: filterObject,
      limit: formValue.limitPerPage
    };
  }

  // --- Helpers ---

  getOperatorsForFieldType(fieldType: string | null | undefined): string[] {
    switch (fieldType) {
      case 'INTEGER':
      case 'REAL':
        return ['=', '!=', '>', '>=', '<', '<='];
      case 'BOOLEAN':
        return ['=', '!='];
      case 'TEXT':
        return ['=', '!=', 'LIKE'];
      default:
        return ['=', '!='];
    }
  }

  getSelectedFieldType(index: number): string | null {
    const group = this.customFilters.at(index);
    return this.getSelectedFieldTypeForGroup(group);
  }

  private getSelectedFieldTypeForGroup(group: AbstractControl | null): string | null {
    if (!group) return null;
    const fieldName = group.get('field')?.value;
    const fieldDefinition = this.availableFilters.find(f => f.name === fieldName);
    return fieldDefinition ? fieldDefinition.type : null;
  }

  private datetimeLocalToUnix(dateTimeLocal: string): number | null {
    try {
      const date = new Date(dateTimeLocal);
      if (isNaN(date.getTime())) return null;
      return Math.floor(date.getTime() / 1000);
    } catch (e) {
      return null;
    }
  }
}