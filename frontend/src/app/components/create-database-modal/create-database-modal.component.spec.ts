// frontend/src/app/components/create-database-modal/create-database-modal.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { of, throwError } from 'rxjs';
import { CreateDatabaseModalComponent } from './create-database-modal.component';
import { DatabaseService } from '../../services/database.service';
import { ModalService } from '../../services/modal.service';
import { NotificationService } from '../../services/notification.service';
import { Database } from '../../models/api.models';

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
      declarations: [ CreateDatabaseModalComponent ],
      imports: [ ReactiveFormsModule ],
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
    
    fixture.detectChanges(); // This calls ngOnInit
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize the form with default values', () => {
    expect(component.databaseForm).toBeDefined();
    expect(component.databaseForm.get('name')?.value).toBe('');
    expect(component.databaseForm.get('file_format')?.value).toBe('jpeg');
    expect(component.databaseForm.get('housekeeping.interval')?.value).toBe('1h');
    expect(component.customFields.length).toBe(0);
  });

  it('should make the name field required', () => {
    const nameControl = component.databaseForm.get('name');
    nameControl?.setValue('');
    expect(nameControl?.valid).toBeFalsy();
    nameControl?.setValue('TestDB');
    expect(nameControl?.valid).toBeTruthy();
  });

  it('should add a custom field to the form array', () => {
    component.addCustomField();
    expect(component.customFields.length).toBe(1);
    const firstField = component.customFields.at(0);
    expect(firstField.get('name')?.value).toBe('');
    expect(firstField.get('type')?.value).toBe('TEXT');
  });

  it('should remove a custom field from the form array', () => {
    component.addCustomField();
    expect(component.customFields.length).toBe(1);
    component.removeCustomField(0);
    expect(component.customFields.length).toBe(0);
  });

  it('should call createDatabase on valid form submission', () => {
    const newDb: Database = { name: 'NewTestDB', file_format: 'png', housekeeping: { interval: '2h', disk_space: '50G', max_age: '90d' }, custom_fields: [] };
    mockDatabaseService.createDatabase.and.returnValue(of(newDb));

    component.databaseForm.setValue({
      name: 'NewTestDB',
      file_format: 'png',
      housekeeping: {
        interval: '2h',
        disk_space: '50G',
        max_age: '90d'
      },
      custom_fields: []
    });

    component.onSubmit();

    expect(mockDatabaseService.createDatabase).toHaveBeenCalled();
    expect(mockNotificationService.showSuccess).toHaveBeenCalledWith("Database 'NewTestDB' created successfully.");
    expect(mockModalService.close).toHaveBeenCalledWith('create-database-modal');
  });

  it('should show an error if database creation fails', () => {
    const errorResponse = { error: 'Database name already exists' };
    mockDatabaseService.createDatabase.and.returnValue(throwError(() => errorResponse));
    
    component.databaseForm.get('name')?.setValue('ExistingDB');
    component.onSubmit();

    expect(component.errorMessage).toBe('Database name already exists');
    expect(mockNotificationService.showError).toHaveBeenCalledWith('Database name already exists');
  });

  it('should not submit an invalid form', () => {
    component.databaseForm.get('name')?.setValue(''); // Make form invalid
    component.onSubmit();
    expect(mockDatabaseService.createDatabase).not.toHaveBeenCalled();
  });
});
