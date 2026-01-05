// frontend/src/app/components/entry-list-view/entry-list-view.component.spec.ts
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SimpleChange, SimpleChanges } from '@angular/core';
import { EntryListViewComponent } from './entry-list-view.component';
import { DatabaseService } from '../../services/database.service';
import { Entry } from '../../models/api.models';
import { EntryStatus } from '../../models/enums';

describe('EntryListViewComponent', () => {
  let component: EntryListViewComponent;
  let fixture: ComponentFixture<EntryListViewComponent>;
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
      imports: [EntryListViewComponent],
      providers: [
        { provide: DatabaseService, useValue: mockDatabaseService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(EntryListViewComponent);
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

    it('onCheckboxClick should emit toggleSelection and stop propagation', () => {
      spyOn(component.toggleSelection, 'emit');
      const mockEvent = jasmine.createSpyObj('MouseEvent', ['stopPropagation']);

      component.onCheckboxClick(mockEntry, mockEvent);

      expect(mockEvent.stopPropagation).toHaveBeenCalled();
      expect(component.toggleSelection.emit).toHaveBeenCalledWith({ entry: mockEntry, event: mockEvent });
    });
  });

  it('should track failed images via onImageError', () => {
    component.onImageError(500);
    expect(component.failedImageIds.has(500)).toBeTrue();
  });

  it('should reset failed images on change', () => {
    component.onImageError(500);
    const changes: SimpleChanges = {
      entries: new SimpleChange([], [], false)
    };
    component.ngOnChanges(changes);
    expect(component.failedImageIds.size).toBe(0);
  });
  
  it('should emit correct events on actions', () => {
    spyOn(component.entryClicked, 'emit');
    spyOn(component.editClicked, 'emit');
    spyOn(component.deleteClicked, 'emit');

    const mockEntryAny: any = { id: 1 };

    component.onEntryClick(mockEntryAny);
    expect(component.entryClicked.emit).toHaveBeenCalledWith(mockEntryAny);

    component.onEditClick(mockEntryAny);
    expect(component.editClicked.emit).toHaveBeenCalledWith(mockEntryAny);

    component.onDeleteClick(mockEntryAny);
    expect(component.deleteClicked.emit).toHaveBeenCalledWith(mockEntryAny);
  });
});