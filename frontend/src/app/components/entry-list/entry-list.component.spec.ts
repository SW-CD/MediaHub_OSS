// frontend/src/app/components/entry-list/entry-list.component.spec.ts
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { of } from 'rxjs';
import { ActivatedRoute } from '@angular/router';
import { EntryListComponent } from './entry-list.component';
import { DatabaseService } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { Database, User, Entry } from '../../models/api.models';
import { CUSTOM_ELEMENTS_SCHEMA, Component, Input, Output, EventEmitter } from '@angular/core';
import { UploadEntryModalComponent } from '../upload-entry-modal/upload-entry-modal.component';
import { ConfirmationModalComponent } from '../confirmation-modal/confirmation-modal.component';
import { ContentType, EntryStatus } from '../../models/enums';
import { FilterChangedEvent } from '../entry-filter/entry-filter.component';

// Mock the new child component
@Component({
  selector: 'app-entry-filter',
  template: '',
  standalone: false
})
class MockEntryFilterComponent {
  @Input() availableFilters: any[] = [];
  @Input() isLoading = false;
  @Output() filterApplied = new EventEmitter<FilterChangedEvent>();
}

describe('EntryListComponent', () => {
  let component: EntryListComponent;
  let fixture: ComponentFixture<EntryListComponent>;
  
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockModalService: jasmine.SpyObj<ModalService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;

  // Mock Data
  const mockUser: User = { id: 1, username: 'admin', can_view: true, can_create: true, can_edit: true, can_delete: true, is_admin: true };
  const mockDb: Database = { 
    name: 'TestDB', 
    content_type: ContentType.Image, 
    config: {}, 
    housekeeping: {} as any, 
    custom_fields: [] 
  };
  
  const mockEntries: Entry[] = [
    { id: 1, timestamp: 100, mime_type: 'image/jpeg', filesize: 1000, status: EntryStatus.Ready },
    { id: 2, timestamp: 200, mime_type: 'image/jpeg', filesize: 2000, status: EntryStatus.Ready },
  ];

  beforeEach(async () => {
    mockDatabaseService = jasmine.createSpyObj('DatabaseService', 
      ['selectDatabase', 'searchEntries', 'bulkDeleteEntries', 'bulkExportEntries', 'deleteEntry'], 
      { refreshRequired$: of(null) }
    );
    mockAuthService = jasmine.createSpyObj('AuthService', [], {
      currentUser$: of(mockUser)
    });
    mockModalService = jasmine.createSpyObj('ModalService', ['open']);
    mockNotificationService = jasmine.createSpyObj('NotificationService', ['showInfo', 'showError', 'showSuccess']);

    await TestBed.configureTestingModule({
      declarations: [ EntryListComponent, MockEntryFilterComponent ], // Declare mock child
      imports: [ RouterTestingModule, ReactiveFormsModule ],
      providers: [
        { provide: DatabaseService, useValue: mockDatabaseService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: ModalService, useValue: mockModalService },
        { provide: NotificationService, useValue: mockNotificationService },
        {
            provide: ActivatedRoute,
            useValue: { paramMap: of({ get: () => 'TestDB' }) }
        }
      ],
      schemas: [CUSTOM_ELEMENTS_SCHEMA]
    })
    .compileComponents();

    fixture = TestBed.createComponent(EntryListComponent);
    component = fixture.componentInstance;
    
    mockDatabaseService.selectDatabase.and.returnValue(of(mockDb));
    mockDatabaseService.searchEntries.and.returnValue(of(mockEntries));
    
    fixture.detectChanges(); 
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('Integration with EntryFilter', () => {
    it('should call searchEntries when onFilterApplied is triggered', () => {
      // Setup payload from "child"
      const filterEvent: FilterChangedEvent = {
        limit: 10,
        filter: { 
          operator: 'and', 
          conditions: [{ field: 'timestamp', operator: '>', value: 1000 }] 
        }
      };

      // Act: Simulate event from child
      component.onFilterApplied(filterEvent);

      // Assert
      expect(component.imagesPerPage).toBe(10);
      expect(mockDatabaseService.searchEntries).toHaveBeenCalledWith('TestDB', jasmine.objectContaining({
        pagination: { limit: 11, offset: 0 },
        filter: filterEvent.filter
      }));
    });
  });

  describe('Selection Logic', () => {
    it('should toggle selection', () => {
      const entry = mockEntries[0];
      const event = { entry, event: { shiftKey: false } as MouseEvent };

      component.toggleSelection(event);
      expect(component.selectedEntryIds.has(1)).toBeTrue();

      component.toggleSelection(event);
      expect(component.selectedEntryIds.has(1)).toBeFalse();
    });
  });

  describe('Bulk Actions', () => {
    beforeEach(() => {
      component.dbName = 'TestDB';
      component.selectedEntryIds.add(1);
    });

    it('onBulkDelete should show modal and call service', () => {
      mockModalService.open.and.returnValue(of(true)); 
      mockDatabaseService.bulkDeleteEntries.and.returnValue(of({}));

      component.onBulkDelete();

      expect(mockModalService.open).toHaveBeenCalled();
      expect(mockDatabaseService.bulkDeleteEntries).toHaveBeenCalled();
      expect(component.selectedEntryIds.size).toBe(0);
    });
  });
});