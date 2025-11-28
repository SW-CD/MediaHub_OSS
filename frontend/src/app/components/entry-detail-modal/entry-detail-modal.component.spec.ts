// frontend/src/app/components/entry-detail-modal/entry-detail-modal.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { CommonModule } from '@angular/common'; // Required for *ngIf/ngSwitch
import { DomSanitizer } from '@angular/platform-browser';
import { HttpClientTestingModule } from '@angular/common/http/testing'; // To prevent HTTP errors

import { EntryDetailModalComponent } from './entry-detail-modal.component';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { Entry, Database, User } from '../../models/api.models';
import { ContentType, EntryStatus } from '../../models/enums';
import { ConfirmationModalComponent } from '../confirmation-modal/confirmation-modal.component';
import { EditEntryModalComponent } from '../edit-entry-modal/edit-entry-modal.component';
import { FormatBytesPipe } from '../../pipes/format-bytes.pipe';
import { Component, Input } from '@angular/core';

// Mock the app-modal component to avoid template errors
@Component({
  selector: 'app-modal',
  template: '<ng-content></ng-content>',
  standalone: false
})
class MockModalComponent {
  @Input() modalId: string = '';
  @Input() modalTitle: string = '';
}

describe('EntryDetailModalComponent', () => {
  let component: EntryDetailModalComponent;
  let fixture: ComponentFixture<EntryDetailModalComponent>;
  
  // Spies
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockModalService: jasmine.SpyObj<ModalService>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;
  let mockSanitizer: jasmine.SpyObj<DomSanitizer>;

  // Mock Data
  const mockUser: User = { 
    id: 1, 
    username: 'admin', 
    can_view: true, 
    can_create: true, 
    can_edit: true, 
    can_delete: true, 
    is_admin: true 
  };

  const mockDb: Database = {
    name: 'TestDB',
    content_type: ContentType.Image,
    config: {},
    housekeeping: {} as any,
    custom_fields: []
  };

  const mockEntry: Entry = {
    id: 101,
    timestamp: 1234567890,
    mime_type: 'image/jpeg',
    filesize: 5000,
    status: EntryStatus.Ready,
    filename: 'test.jpg'
  };

  beforeEach(async () => {
    // 1. Create Spies
    // We define ALL methods we expect to be called here.
    mockDatabaseService = jasmine.createSpyObj('DatabaseService', 
      ['getEntryMeta', 'getEntryFileBlob', 'getEntryPreviewUrl', 'deleteEntry', 'clearSelectedEntry'], 
      {
        selectedDatabase$: of(mockDb),
        selectedEntry$: of(mockEntry) // Default selection
      }
    );

    mockModalService = jasmine.createSpyObj('ModalService', ['open', 'close', 'getModalEvents']);
    mockAuthService = jasmine.createSpyObj('AuthService', [], { currentUser$: of(mockUser) });
    mockNotificationService = jasmine.createSpyObj('NotificationService', ['showError', 'showSuccess']);
    mockSanitizer = jasmine.createSpyObj('DomSanitizer', ['bypassSecurityTrustUrl']);

    // 2. Configure default return values
    // Crucial: ensure getEntryFileBlob returns an Observable, or the component throws "reading 'pipe' of undefined"
    mockDatabaseService.getEntryMeta.and.returnValue(of(mockEntry));
    mockDatabaseService.getEntryFileBlob.and.returnValue(of(new Blob(['test content'], { type: 'image/jpeg' })));
    mockDatabaseService.getEntryPreviewUrl.and.returnValue('mock-preview-url');
    mockSanitizer.bypassSecurityTrustUrl.and.returnValue('safe-url');

    await TestBed.configureTestingModule({
      declarations: [ EntryDetailModalComponent, MockModalComponent ],
      imports: [ 
        CommonModule, // Required for structural directives
        HttpClientTestingModule, // Required for preventing HTTP injection errors
        FormatBytesPipe // Standalone pipe
      ],
      providers: [
        { provide: DatabaseService, useValue: mockDatabaseService },
        { provide: ModalService, useValue: mockModalService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: NotificationService, useValue: mockNotificationService },
        { provide: DomSanitizer, useValue: mockSanitizer }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(EntryDetailModalComponent);
    component = fixture.componentInstance;
    
    // 3. Trigger initial data binding (ngOnInit)
    fixture.detectChanges(); 
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('onEdit', () => {
    beforeEach(() => {
      // Reset state for specific tests
      component.entryForMetadata = mockEntry;
      component.currentDatabase = mockDb;
    });

    it('should open EditEntryModal if status is "Ready"', () => {
      // Act
      component.onEdit();

      // Assert
      expect(mockModalService.open).toHaveBeenCalledWith(EditEntryModalComponent.MODAL_ID);
    });

    it('should show error and NOT open modal if status is "Processing"', () => {
      // Arrange
      component.entryForMetadata = { ...mockEntry, status: EntryStatus.Processing };

      // Act
      component.onEdit();

      // Assert
      expect(mockNotificationService.showError).toHaveBeenCalledWith(jasmine.stringMatching(/processing/));
      expect(mockModalService.open).not.toHaveBeenCalled();
    });

    it('should do nothing if entry metadata is missing', () => {
      // Arrange
      component.entryForMetadata = null;

      // Act
      component.onEdit();

      // Assert
      expect(mockModalService.open).not.toHaveBeenCalled();
    });
  });

  describe('onDelete', () => {
    beforeEach(() => {
      component.entryForMetadata = mockEntry;
      component.currentDatabase = mockDb;
    });

    it('should show error if entry status is "processing"', () => {
      component.entryForMetadata = { ...mockEntry, status: EntryStatus.Processing };
      component.onDelete();
      expect(mockNotificationService.showError).toHaveBeenCalledWith(jasmine.stringMatching(/processing/));
      expect(mockModalService.open).not.toHaveBeenCalled();
    });

    it('should open confirmation modal if status is "ready"', () => {
      mockModalService.open.and.returnValue(of(false)); 
      component.onDelete();
      expect(mockModalService.open).toHaveBeenCalledWith(
        ConfirmationModalComponent.MODAL_ID, 
        jasmine.objectContaining({ title: 'Delete Entry' })
      );
    });

    it('should NOT call delete API if user cancels confirmation', () => {
      mockModalService.open.and.returnValue(of(false)); // User clicks Cancel
      component.onDelete();
      expect(mockDatabaseService.deleteEntry).not.toHaveBeenCalled();
    });

    it('should call delete API and close modal if user confirms', () => {
      mockModalService.open.and.returnValue(of(true)); // User clicks Confirm
      mockDatabaseService.deleteEntry.and.returnValue(of({ message: 'Deleted' }));

      component.onDelete();

      expect(component.isLoadingFile).toBeTrue();
      expect(mockDatabaseService.deleteEntry).toHaveBeenCalledWith(mockDb.name, mockEntry.id);
      expect(mockModalService.close).toHaveBeenCalled();
      expect(mockDatabaseService.clearSelectedEntry).toHaveBeenCalled();
    });

    it('should reset loading state if delete API fails', () => {
      mockModalService.open.and.returnValue(of(true)); 
      mockDatabaseService.deleteEntry.and.returnValue(throwError(() => new Error('API Error')));

      component.onDelete();

      expect(mockDatabaseService.deleteEntry).toHaveBeenCalled();
      expect(component.isLoadingFile).toBeFalse();
      expect(mockModalService.close).not.toHaveBeenCalled();
    });
  });

  describe('UI Permissions', () => {
    it('should expose currentUser$ for template usage', (done) => {
      component.currentUser$.subscribe(user => {
        expect(user).toEqual(mockUser);
        expect(user?.can_delete).toBeTrue();
        expect(user?.can_edit).toBeTrue();
        done();
      });
    });
  });
});