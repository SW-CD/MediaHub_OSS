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

describe('DatabaseSettingsComponent', () => {
  let component: DatabaseSettingsComponent;
  let fixture: ComponentFixture<DatabaseSettingsComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockModalService: jasmine.SpyObj<ModalService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;
  
  const mockDb: Database = {
    name: 'SettingsDB',
    file_format: 'jpeg',
    custom_fields: [],
    housekeeping: { interval: '1h', disk_space: '100G', max_age: '365d' },
    stats: { image_count: 123, total_disk_space_bytes: 123456789 },
  };

  const adminUser: User = { id: 1, username: 'admin', can_view: true, can_create: true, can_edit: true, can_delete: true };
  const viewUser: User = { id: 2, username: 'viewer', can_view: true, can_create: false, can_edit: false, can_delete: false };

  // Helper to set up the component with a specific user
  async function setupComponent(user: User) {
    const dbServiceSpy = jasmine.createSpyObj('DatabaseService', ['updateDatabase', 'runHousekeeping', 'deleteDatabase'], {
      selectedDatabase$: of(mockDb)
    });

    const authServiceSpy = jasmine.createSpyObj('AuthService', [], {
      currentUser$: of(user)
    });

    const modalServiceSpy = jasmine.createSpyObj('ModalService', ['openWithData']);
    const notificationServiceSpy = jasmine.createSpyObj('NotificationService', ['showSuccess', 'showError']);

    await TestBed.configureTestingModule({
      declarations: [ DatabaseSettingsComponent ],
      imports: [ RouterTestingModule, FormsModule, ReactiveFormsModule ],
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
    mockNotificationService = TestBed.inject(NotificationService) as jasmine.SpyObj<NotificationService>;
    
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

    it('should enable form fields and show action buttons for admin', () => {
      expect(component.housekeepingForm.enabled).toBeTrue();
      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.querySelector('.run-hk-btn')).toBeTruthy();
      expect(compiled.querySelector('.delete-db-btn')).toBeTruthy();
    });

    it('should call updateDatabase on form save', () => {
      mockDatabaseService.updateDatabase.and.returnValue(of(mockDb));
      component.housekeepingForm.patchValue({ interval: '2h' });
      component.onSave();
      expect(mockDatabaseService.updateDatabase).toHaveBeenCalledWith(mockDb.name, { interval: '2h', disk_space: '100G', max_age: '365d' });
      expect(mockNotificationService.showSuccess).toHaveBeenCalledWith('Housekeeping rules updated successfully.');
    });

    it('should call runHousekeeping when button is clicked', () => {
      const report: HousekeepingReport = { database_name: mockDb.name, images_deleted: 10, space_freed_bytes: 1000, message: 'Done' };
      mockDatabaseService.runHousekeeping.and.returnValue(of(report));
      component.onRunHousekeeping();
      expect(mockDatabaseService.runHousekeeping).toHaveBeenCalledWith(mockDb.name);
      expect(mockNotificationService.showSuccess).toHaveBeenCalledWith(report.message);
    });

    it('should open confirmation modal on delete', () => {
      component.onDeleteDatabase();
      expect(mockModalService.openWithData).toHaveBeenCalledWith('confirmation-modal', jasmine.any(Object));
    });
  });

  describe('with View-Only User', () => {
    beforeEach(async () => {
      await setupComponent(viewUser);
    });

    it('should disable form fields for view-only user', () => {
      expect(component.housekeepingForm.disabled).toBeTrue();
    });

    it('should hide action buttons for view-only user', () => {
      const compiled = fixture.nativeElement as HTMLElement;
      expect(compiled.querySelector('.run-hk-btn')).toBeFalsy();
      expect(compiled.querySelector('.delete-db-btn')).toBeFalsy();
      expect(compiled.querySelector('.save-btn')).toBeFalsy();
    });
  });
});
