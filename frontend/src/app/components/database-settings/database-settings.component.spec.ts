// frontend/src/app/components/database-settings/database-settings.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { of } from 'rxjs';
import { ActivatedRoute } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { DatabaseSettingsComponent } from './database-settings.component';
import { DatabaseService } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { User, Database, HousekeepingReport } from '../../models/api.models';
import { ContentType } from '../../models/enums';
import { FormatBytesPipe } from '../../pipes/format-bytes.pipe';
import { IntervalPickerComponent } from '../interval-picker/interval-picker.component';

describe('DatabaseSettingsComponent', () => {
  let component: DatabaseSettingsComponent;
  let fixture: ComponentFixture<DatabaseSettingsComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockModalService: jasmine.SpyObj<ModalService>;
  
  const mockDb: Database = {
    name: 'SettingsDB',
    content_type: ContentType.Image,
    config: { create_preview: true, convert_to_jpeg: false },
    custom_fields: [],
    housekeeping: { interval: '1h', disk_space: '100G', max_age: '365d' },
    stats: { entry_count: 123, total_disk_space_bytes: 123456789 },
  };

  const adminUser: User = { id: 1, username: 'admin', can_view: true, can_create: true, can_edit: true, can_delete: true, is_admin: true };
  const viewUser: User = { id: 2, username: 'viewer', can_view: true, can_create: false, can_edit: false, can_delete: false, is_admin: false };

  async function setupComponent(user: User) {
    const dbServiceSpy = jasmine.createSpyObj('DatabaseService', ['updateDatabase', 'triggerHousekeeping', 'deleteDatabase', 'selectDatabase'], {
      selectedDatabase$: of(mockDb)
    });
    // Fix: selectDatabase must return an observable
    dbServiceSpy.selectDatabase.and.returnValue(of(mockDb));
    // Fix: deleteDatabase must return an observable
    dbServiceSpy.deleteDatabase.and.returnValue(of({ message: 'Deleted' }));

    const authServiceSpy = jasmine.createSpyObj('AuthService', ['getCurrentUser'], {
      currentUser$: of(user)
    });
    authServiceSpy.getCurrentUser.and.returnValue(user);

    const modalServiceSpy = jasmine.createSpyObj('ModalService', ['open']);
    // Fix: modal.open must return an observable
    modalServiceSpy.open.and.returnValue(of(true));

    const notificationServiceSpy = jasmine.createSpyObj('NotificationService', ['showSuccess', 'showError']);

    await TestBed.configureTestingModule({
      declarations: [ DatabaseSettingsComponent ],
      imports: [ RouterTestingModule, FormsModule, ReactiveFormsModule, FormatBytesPipe, IntervalPickerComponent ],
      providers: [
        { provide: DatabaseService, useValue: dbServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        { provide: ModalService, useValue: modalServiceSpy },
        { provide: NotificationService, useValue: notificationServiceSpy },
        {
          provide: ActivatedRoute,
          useValue: { paramMap: of({ get: () => mockDb.name }) },
        },
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(DatabaseSettingsComponent);
    component = fixture.componentInstance;
    mockDatabaseService = TestBed.inject(DatabaseService) as jasmine.SpyObj<DatabaseService>;
    mockAuthService = TestBed.inject(AuthService) as jasmine.SpyObj<AuthService>;
    mockModalService = TestBed.inject(ModalService) as jasmine.SpyObj<ModalService>;
    
    fixture.detectChanges();
  }

  describe('with Admin User', () => {
    beforeEach(async () => {
      await setupComponent(adminUser);
    });

    it('should create and display stats', () => {
      expect(component).toBeTruthy();
      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.querySelector('.stat-value')?.textContent).toContain('123');
    });

    it('should enable form fields', () => {
      expect(component.settingsForm.enabled).toBeTrue();
    });

    it('should call updateDatabase on form save', () => {
      mockDatabaseService.updateDatabase.and.returnValue(of(mockDb));
      component.settingsForm.patchValue({ housekeeping: { interval: '2h' } });
      component.onSaveSettings();
      expect(mockDatabaseService.updateDatabase).toHaveBeenCalled();
    });

    it('should call triggerHousekeeping when button is clicked', () => {
      const report: HousekeepingReport = { database_name: mockDb.name, entries_deleted: 10, space_freed_bytes: 1000, message: 'Done' };
      mockDatabaseService.triggerHousekeeping.and.returnValue(of(report));
      component.onTriggerHousekeeping();
      expect(mockDatabaseService.triggerHousekeeping).toHaveBeenCalledWith(mockDb.name);
    });

    it('should open confirmation modal on delete', () => {
      component.onDeleteDatabase();
      expect(mockModalService.open).toHaveBeenCalled();
      // Because mockModalService.open returns of(true), deleteDatabase should also be called
      expect(mockDatabaseService.deleteDatabase).toHaveBeenCalledWith(mockDb.name);
    });
  });

  describe('with View-Only User', () => {
    beforeEach(async () => {
      await setupComponent(viewUser);
    });

    it('should disable form fields for view-only user', () => {
      expect(component.settingsForm.disabled).toBeTrue();
    });
  });
});