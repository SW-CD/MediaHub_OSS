// frontend/src/app/components/entry-grid/entry-grid.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SimpleChange, SimpleChanges } from '@angular/core';
import { EntryGridComponent } from './entry-grid.component';
import { DatabaseService } from '../../services/database.service';
import { EntryStatus } from '../../models/enums';
import { Entry } from '../../models/api.models';

describe('EntryGridComponent', () => {
  let component: EntryGridComponent;
  let fixture: ComponentFixture<EntryGridComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;

  const mockEntry: Entry = {
    id: 101,
    timestamp: 1234567890,
    mime_type: 'image/jpeg',
    filesize: 5000,
    status: EntryStatus.Ready
  };

  beforeEach(async () => {
    mockDatabaseService = jasmine.createSpyObj('DatabaseService', ['getEntryPreviewUrl']);
    
    await TestBed.configureTestingModule({
      imports: [EntryGridComponent], 
      providers: [
        { provide: DatabaseService, useValue: mockDatabaseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(EntryGridComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('Selection Logic', () => {
    it('isSelected should return true if id is in selectedIds set', () => {
      component.selectedIds = new Set([101]);
      expect(component.isSelected(mockEntry)).toBeTrue();
    });

    it('isSelected should return false if id is NOT in selectedIds set', () => {
      component.selectedIds = new Set([999]);
      expect(component.isSelected(mockEntry)).toBeFalse();
    });

    it('onCheckboxClick should emit toggleSelection and stop propagation', () => {
      spyOn(component.toggleSelection, 'emit');
      const mockEvent = jasmine.createSpyObj('MouseEvent', ['stopPropagation']);

      component.onCheckboxClick(mockEntry, mockEvent);

      expect(mockEvent.stopPropagation).toHaveBeenCalled();
      expect(component.toggleSelection.emit).toHaveBeenCalledWith({ entry: mockEntry, event: mockEvent });
    });
  });

  it('should add entry ID to failedImageIds on onImageError', () => {
    expect(component.failedImageIds.has(101)).toBeFalse();
    component.onImageError(101);
    expect(component.failedImageIds.has(101)).toBeTrue();
  });

  it('should clear failedImageIds when inputs change', () => {
    component.onImageError(101);
    expect(component.failedImageIds.size).toBe(1);

    const changes: SimpleChanges = {
      dbName: new SimpleChange('OldDB', 'NewDB', false)
    };
    component.ngOnChanges(changes);

    expect(component.failedImageIds.size).toBe(0);
  });
});