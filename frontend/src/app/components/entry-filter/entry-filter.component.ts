import { Component, Input, Output, EventEmitter, OnChanges, SimpleChanges, ChangeDetectionStrategy, ChangeDetectorRef, OnInit, HostListener } from '@angular/core';
import { FormBuilder, FormGroup, FormArray, Validators, AbstractControl } from '@angular/forms';
import { SearchFilter } from '../../models';

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
export class EntryFilterComponent implements OnInit, OnChanges {
  @Input() availableFilters: AvailableFilter[] = [];
  @Input() isLoading = false;
  
  @Output() filterApplied = new EventEmitter<FilterChangedEvent>();

  public filterForm: FormGroup;
  public isCollapsed = false;

  constructor(private fb: FormBuilder, private cdr: ChangeDetectorRef) {
    this.filterForm = this.fb.group({
      limitPerPage: [24, [Validators.required, Validators.min(1)]],
      tstart: [''],
      tend: [''],
      customFilters: this.fb.array([])
    });
  }

  ngOnInit(): void {
    this.checkMobileState();
    
    // Automatically re-evaluate the active filters count if the form changes
    this.filterForm.valueChanges.subscribe(() => {
      this.cdr.markForCheck();
    });
  }

  // Determine initial state based on window size
  private checkMobileState(): void {
    if (window.innerWidth < 768) {
      this.isCollapsed = true;
    }
  }

  toggleCollapse(): void {
    this.isCollapsed = !this.isCollapsed;
  }

  /**
   * Dynamically calculates how many filter conditions are actively set
   */
  get activeFiltersCount(): number {
    let count = 0;
    const formValue = this.filterForm.value;
    
    if (formValue.tstart) count++;
    if (formValue.tend) count++;

    formValue.customFilters.forEach((filter: any) => {
      // Only count custom filters that actually have a field and a value selected
      if (filter.field && filter.value !== null && String(filter.value).trim() !== '') {
        count++;
      }
    });
    
    return count;
  }

  ngOnChanges(changes: SimpleChanges): void {
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

    newGroup.get('field')?.valueChanges.subscribe(() => {
      const fieldType = this.getSelectedFieldTypeForGroup(newGroup);
      const defaultOp = this.getOperatorsForFieldType(fieldType)[0] || '=';
      
      newGroup.get('operator')?.setValue(defaultOp);
      
      if (fieldType === 'BOOLEAN') {
        newGroup.get('value')?.setValue('true'); 
      } else {
        newGroup.get('value')?.setValue('');
      }
    });

    this.customFilters.push(newGroup);
    
    // Automatically expand the panel if a user clicks "+ Add Filter" programmatically 
    // or if a filter is added via another method
    if (this.isCollapsed) {
      this.isCollapsed = false;
    }
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
    
    // Auto-collapse on mobile after applying to get the filter out of the way of the results
    if (window.innerWidth < 768) {
      this.isCollapsed = true;
    }
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
          } else if (fieldDefinition.type === 'TEXT' && filter.operator === 'LIKE') {
            filterValue = `%${filterValue}%`;
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
      
      return date.getTime(); 
    } catch (e) {
      return null;
    }
  }
}