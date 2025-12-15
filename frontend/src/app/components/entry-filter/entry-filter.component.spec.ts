// frontend/src/app/components/entry-filter/entry-filter.component.spec.ts
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { EntryFilterComponent, AvailableFilter } from './entry-filter.component';
import { SimpleChange, SimpleChanges } from '@angular/core';

describe('EntryFilterComponent', () => {
  let component: EntryFilterComponent;
  let fixture: ComponentFixture<EntryFilterComponent>;

  const mockAvailableFilters: AvailableFilter[] = [
    { name: 'timestamp', type: 'INTEGER' },
    { name: 'artist', type: 'TEXT' },
    { name: 'is_valid', type: 'BOOLEAN' }
  ];

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [EntryFilterComponent],
      imports: [ReactiveFormsModule]
    }).compileComponents();

    fixture = TestBed.createComponent(EntryFilterComponent);
    component = fixture.componentInstance;
    component.availableFilters = mockAvailableFilters;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with default values', () => {
    expect(component.filterForm.get('limitPerPage')?.value).toBe(24);
    expect(component.customFilters.length).toBe(0);
  });

  it('should reset custom filters when availableFilters input changes', () => {
    // Add a filter
    component.addCustomFilter();
    expect(component.customFilters.length).toBe(1);

    // Simulate input change
    const changes: SimpleChanges = {
      availableFilters: new SimpleChange(mockAvailableFilters, [], false)
    };
    component.ngOnChanges(changes);

    // Should be cleared
    expect(component.customFilters.length).toBe(0);
  });

  describe('Custom Filters', () => {
    it('should add a custom filter row', () => {
      component.addCustomFilter();
      expect(component.customFilters.length).toBe(1);
      const group = component.customFilters.at(0);
      expect(group.get('field')?.value).toBe('');
      expect(group.get('operator')?.value).toBe('=');
    });

    it('should remove a custom filter row', () => {
      component.addCustomFilter();
      component.removeCustomFilter(0);
      expect(component.customFilters.length).toBe(0);
    });

    it('should update operator default when field changes', () => {
      component.addCustomFilter();
      const group = component.customFilters.at(0);

      // Select TEXT field
      group.get('field')?.setValue('artist');
      // Logic inside component sets default op for TEXT (usually '=')
      expect(group.get('operator')?.value).toBe('=');
    });
  });

  describe('applyFilters', () => {
    it('should not emit if form is invalid', () => {
      spyOn(component.filterApplied, 'emit');
      component.filterForm.get('limitPerPage')?.setValue(null); // Invalid
      component.applyFilters();
      expect(component.filterApplied.emit).not.toHaveBeenCalled();
    });

    it('should emit correct event structure', () => {
      spyOn(component.filterApplied, 'emit');
      
      // Setup form
      component.filterForm.patchValue({
        limitPerPage: 50,
        tstart: '2023-01-01T12:00'
      });

      // Add custom filter
      component.addCustomFilter();
      component.customFilters.at(0).patchValue({
        field: 'artist',
        operator: '=',
        value: 'Test Artist'
      });

      component.applyFilters();

      expect(component.filterApplied.emit).toHaveBeenCalledWith(jasmine.objectContaining({
        limit: 50,
        filter: jasmine.objectContaining({
          operator: 'and',
          conditions: jasmine.arrayContaining([
            // Timestamp check
            jasmine.objectContaining({ field: 'timestamp', operator: '>=' }),
            // Custom field check
            jasmine.objectContaining({ field: 'artist', operator: '=', value: 'Test Artist' })
          ])
        })
      }));
    });
  });
});