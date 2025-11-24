// frontend/src/app/components/entry-detail-modal/entry-detail-modal.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of, Subject, throwError } from 'rxjs';
import { DomSanitizer } from '@angular/platform-browser';
import { EntryDetailModalComponent } from './entry-detail-modal.component';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { Entry, Database, User } from '../../models/api.models';
import { ContentType, EntryStatus } from '../../models/enums';
import { ConfirmationModalComponent } from '../confirmation-modal/confirmation-modal.component';
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
    // Create Spies
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

    // Mock returns
    mockDatabaseService.getEntryMeta.and.returnValue(of(mockEntry));
    mockDatabaseService.getEntryFileBlob.and.returnValue(of(new Blob(['test content'])));
    mockDatabaseService.getEntryPreviewUrl.and.returnValue('mock-preview-url');
    mockSanitizer.bypassSecurityTrustUrl.and.returnValue('safe-url');

    await TestBed.configureTestingModule({
      declarations: [ EntryDetailModalComponent, MockModalComponent ],
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
    fixture.detectChanges(); // Trigger ngOnInit
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('onDelete', () => {
    beforeEach(() => {
      // Ensure the component has loaded the entry data needed for deletion
      component.entryForMetadata = mockEntry;
      // We need to manually set currentDatabase if it wasn't captured in ngOnInit correctly during test setup
      // (Though the mock observable should have handled it, accessing the private var or ensuring state is safe)
      // Accessing private via 'any' for test setup:
      (component as any).currentDatabase = mockDb;
    });

    it('should show error if entry status is "processing"', () => {
      // Arrange
      component.entryForMetadata = { ...mockEntry, status: EntryStatus.Processing };

      // Act
      component.onDelete();

      // Assert
      expect(mockNotificationService.showError).toHaveBeenCalledWith(jasmine.stringMatching(/processing/));
      expect(mockModalService.open).not.toHaveBeenCalled();
    });

    it('should open confirmation modal if status is "ready"', () => {
      // Arrange
      // Mock the modal to return an empty observable (neither true nor false yet) to stop execution there
      mockModalService.open.and.returnValue(of(false)); 

      // Act
      component.onDelete();

      // Assert
      expect(mockModalService.open).toHaveBeenCalledWith(
        ConfirmationModalComponent.MODAL_ID, 
        jasmine.objectContaining({ title: 'Delete Entry' })
      );
    });

    it('should NOT call delete API if user cancels confirmation', () => {
      // Arrange
      mockModalService.open.and.returnValue(of(false)); // User clicks Cancel

      // Act
      component.onDelete();

      // Assert
      expect(mockDatabaseService.deleteEntry).not.toHaveBeenCalled();
    });

    it('should call delete API and close modal if user confirms', () => {
      // Arrange
      mockModalService.open.and.returnValue(of(true)); // User clicks Confirm
      mockDatabaseService.deleteEntry.and.returnValue(of({ message: 'Deleted' }));

      // Act
      component.onDelete();

      // Assert
      expect(component.isLoadingFile).toBeTrue(); // UI should lock
      expect(mockDatabaseService.deleteEntry).toHaveBeenCalledWith(mockDb.name, mockEntry.id);
      expect(mockModalService.close).toHaveBeenCalled(); // Component should close itself
      expect(mockDatabaseService.clearSelectedEntry).toHaveBeenCalled(); // Cleanup
    });

    it('should reset loading state if delete API fails', () => {
      // Arrange
      mockModalService.open.and.returnValue(of(true)); // User clicks Confirm
      mockDatabaseService.deleteEntry.and.returnValue(throwError(() => new Error('API Error')));

      // Act
      component.onDelete();

      // Assert
      expect(mockDatabaseService.deleteEntry).toHaveBeenCalled();
      expect(component.isLoadingFile).toBeFalse(); // Should unlock UI on error
      expect(mockModalService.close).not.toHaveBeenCalled(); // Should stay open so user sees context
    });
  });

  describe('UI Permissions', () => {
    it('should expose currentUser$ for template usage', (done) => {
      component.currentUser$.subscribe(user => {
        expect(user).toEqual(mockUser);
        expect(user?.can_delete).toBeTrue();
        done();
      });
    });
  });
});