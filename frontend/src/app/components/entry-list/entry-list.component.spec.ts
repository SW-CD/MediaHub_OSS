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
import { Database, User } from '../../models/api.models';
import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { UploadEntryModalComponent } from '../upload-entry-modal/upload-entry-modal.component';
import { ContentType } from '../../models/enums';

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

  beforeEach(async () => {
    mockDatabaseService = jasmine.createSpyObj('DatabaseService', ['selectDatabase', 'searchEntries'], {
      refreshRequired$: of(null)
    });
    mockAuthService = jasmine.createSpyObj('AuthService', [], {
      currentUser$: of(mockUser)
    });
    mockModalService = jasmine.createSpyObj('ModalService', ['open']);
    mockNotificationService = jasmine.createSpyObj('NotificationService', ['showInfo', 'showError']);

    await TestBed.configureTestingModule({
      declarations: [ EntryListComponent ],
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
      schemas: [CUSTOM_ELEMENTS_SCHEMA] // Ignore child components like app-entry-grid
    })
    .compileComponents();

    fixture = TestBed.createComponent(EntryListComponent);
    component = fixture.componentInstance;
    
    // Setup initial state for DB selection
    mockDatabaseService.selectDatabase.and.returnValue(of(mockDb));
    mockDatabaseService.searchEntries.and.returnValue(of([]));
    
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('onFileDropped', () => {
    it('should show error if user cannot create entries', () => {
        // Arrange: User without create permission
        const readOnlyUser = { ...mockUser, can_create: false };
        component.currentUser = readOnlyUser;
        
        const file = new File([''], 'test.jpg', { type: 'image/jpeg' });
        
        // Act
        component.onFileDropped(file);

        // Assert
        expect(mockNotificationService.showInfo).toHaveBeenCalledWith('You cannot upload files here.');
        expect(mockModalService.open).not.toHaveBeenCalled();
    });

    it('should show error if mime type is invalid for database', () => {
        // Arrange: DB is 'image', File is 'text/plain'
        const file = new File([''], 'notes.txt', { type: 'text/plain' });
        
        // Act
        component.onFileDropped(file);

        // Assert
        expect(mockNotificationService.showError).toHaveBeenCalled();
        expect(mockModalService.open).not.toHaveBeenCalled();
    });

    it('should open Upload Modal with file data if valid', () => {
        // Arrange: DB is 'image', File is 'image/png'
        const file = new File([''], 'pic.png', { type: 'image/png' });
        
        // Act
        component.onFileDropped(file);

        // Assert
        expect(mockModalService.open).toHaveBeenCalledWith(
            UploadEntryModalComponent.MODAL_ID, 
            { droppedFile: file }
        );
    });
  });
});