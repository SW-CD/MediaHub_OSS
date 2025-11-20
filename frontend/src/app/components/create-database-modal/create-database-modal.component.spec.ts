// frontend/src/app/components/create-database-modal/create-database-modal.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { of } from 'rxjs';
import { CreateDatabaseModalComponent } from './create-database-modal.component';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { Database } from '../../models/api.models';
import { ContentType } from '../../models/enums';
import { IntervalPickerComponent } from '../interval-picker/interval-picker.component';
import { Component, Input } from '@angular/core';

// Mock for app-modal
@Component({
  selector: 'app-modal',
  template: '<ng-content></ng-content>',
  standalone: false
})
class MockModalComponent {
  @Input() modalId: string = '';
  @Input() modalTitle: string = '';
}

describe('CreateDatabaseModalComponent', () => {
  let component: CreateDatabaseModalComponent;
  let fixture: ComponentFixture<CreateDatabaseModalComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockModalService: jasmine.SpyObj<ModalService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;

  beforeEach(async () => {
    const dbServiceSpy = jasmine.createSpyObj('DatabaseService', ['createDatabase']);
    const modalServiceSpy = jasmine.createSpyObj('ModalService', ['close']);
    const notificationServiceSpy = jasmine.createSpyObj('NotificationService', ['showSuccess', 'showError']);

    await TestBed.configureTestingModule({
      declarations: [ CreateDatabaseModalComponent, MockModalComponent ], // Add MockModalComponent
      imports: [ ReactiveFormsModule, IntervalPickerComponent ],
      providers: [
        { provide: DatabaseService, useValue: dbServiceSpy },
        { provide: ModalService, useValue: modalServiceSpy },
        { provide: NotificationService, useValue: notificationServiceSpy },
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(CreateDatabaseModalComponent);
    component = fixture.componentInstance;
    mockDatabaseService = TestBed.inject(DatabaseService) as jasmine.SpyObj<DatabaseService>;
    mockModalService = TestBed.inject(ModalService) as jasmine.SpyObj<ModalService>;
    mockNotificationService = TestBed.inject(NotificationService) as jasmine.SpyObj<NotificationService>;
    
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize the form with default values', () => {
    expect(component.createDbForm).toBeDefined();
    expect(component.createDbForm.get('name')?.value).toBe('');
    expect(component.createDbForm.get('content_type')?.value).toBe('image');
    expect(component.createDbForm.get('housekeeping.interval')?.value).toBe('10m');
  });

  it('should make the name field required', () => {
    const nameControl = component.createDbForm.get('name');
    nameControl?.setValue('');
    expect(nameControl?.valid).toBeFalsy();
    nameControl?.setValue('TestDB');
    expect(nameControl?.valid).toBeTruthy();
  });

  it('should add a custom field', () => {
    component.addCustomField();
    expect(component.customFields.length).toBe(1);
  });

  it('should remove a custom field', () => {
    component.addCustomField();
    component.removeCustomField(0);
    expect(component.customFields.length).toBe(0);
  });

  it('should call createDatabase on valid form submission', () => {
    const newDb: Database = { 
        name: 'NewTestDB', 
        content_type: ContentType.Image,
        config: { convert_to_jpeg: false, create_preview: true },
        housekeeping: { interval: '2h', disk_space: '50G', max_age: '90d' }, 
        custom_fields: [] 
    };
    mockDatabaseService.createDatabase.and.returnValue(of(newDb));

    component.createDbForm.patchValue({
      name: 'NewTestDB',
      content_type: 'image',
      housekeeping: {
        interval: '2h',
        disk_space: '50G',
        max_age: '90d'
      }
    });

    component.onSubmit();

    expect(mockDatabaseService.createDatabase).toHaveBeenCalled();
    expect(mockModalService.close).toHaveBeenCalledWith(true);
  });

  it('should not submit an invalid form', () => {
    component.createDbForm.get('name')?.setValue('');
    component.onSubmit();
    expect(mockDatabaseService.createDatabase).not.toHaveBeenCalled();
  });
});