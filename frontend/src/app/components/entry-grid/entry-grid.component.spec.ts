// entry-grid.component.spec.ts

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
      imports: [EntryGridComponent], // Standalone
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

  it('should add entry ID to failedImageIds on onImageError', () => {
    expect(component.failedImageIds.has(101)).toBeFalse();
    
    component.onImageError(101);
    
    expect(component.failedImageIds.has(101)).toBeTrue();
  });

  it('should clear failedImageIds when inputs change', () => {
    // Arrange: Add a failure
    component.onImageError(101);
    expect(component.failedImageIds.size).toBe(1);

    // Act: Trigger ngOnChanges
    const changes: SimpleChanges = {
      dbName: new SimpleChange('OldDB', 'NewDB', false)
    };
    component.ngOnChanges(changes);

    // Assert: Set should be empty
    expect(component.failedImageIds.size).toBe(0);
  });

  describe('getEntryTitle', () => {
    it('should return correct title for Ready status', () => {
      const entry = { ...mockEntry, status: EntryStatus.Ready };
      expect(component.getEntryTitle(entry)).toContain('View details');
    });

    it('should return correct title for Processing status', () => {
      const entry = { ...mockEntry, status: EntryStatus.Processing };
      expect(component.getEntryTitle(entry)).toContain('Processing...');
    });

    it('should return correct title for Error status', () => {
      const entry = { ...mockEntry, status: EntryStatus.Error };
      expect(component.getEntryTitle(entry)).toContain('Processing Failed');
    });

    it('should return No Preview title if ID is in failedImageIds', () => {
      const entry = { ...mockEntry, status: EntryStatus.Ready };
      component.onImageError(entry.id);
      expect(component.getEntryTitle(entry)).toContain('No Preview Available');
    });
  });
});