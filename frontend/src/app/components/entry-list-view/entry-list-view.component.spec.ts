// entry-list-view.component.spec.ts
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SimpleChange, SimpleChanges } from '@angular/core';
import { EntryListViewComponent } from './entry-list-view.component';
import { DatabaseService } from '../../services/database.service';

describe('EntryListViewComponent', () => {
  let component: EntryListViewComponent;
  let fixture: ComponentFixture<EntryListViewComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;

  beforeEach(async () => {
    mockDatabaseService = jasmine.createSpyObj('DatabaseService', ['getEntryPreviewUrl']);

    await TestBed.configureTestingModule({
      imports: [EntryListViewComponent], // Standalone
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

  it('should track failed images via onImageError', () => {
    component.onImageError(500);
    expect(component.failedImageIds.has(500)).toBeTrue();
  });

  it('should reset failed images on change', () => {
    component.onImageError(500);
    
    // Simulate entries changing (e.g. pagination or filter)
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

    const mockEntry: any = { id: 1 };

    component.onEntryClick(mockEntry);
    expect(component.entryClicked.emit).toHaveBeenCalledWith(mockEntry);

    component.onEditClick(mockEntry);
    expect(component.editClicked.emit).toHaveBeenCalledWith(mockEntry);

    component.onDeleteClick(mockEntry);
    expect(component.deleteClicked.emit).toHaveBeenCalledWith(mockEntry);
  });
});