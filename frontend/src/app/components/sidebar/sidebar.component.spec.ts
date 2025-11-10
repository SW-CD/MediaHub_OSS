// frontend/src/app/components/sidebar/sidebar.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { of } from 'rxjs';
import { SidebarComponent } from './sidebar.component';
import { DatabaseService } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { User, Database } from '../../models/api.models';
import { Component, Input } from '@angular/core';

// Mock child component to satisfy the template
@Component({
  selector: 'app-create-database-modal',
  template: ''
})
class MockCreateDatabaseModalComponent {}

describe('SidebarComponent', () => {
  let component: SidebarComponent;
  let fixture: ComponentFixture<SidebarComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockModalService: jasmine.SpyObj<ModalService>;

  const mockDatabases: Database[] = [
    { name: 'DB1', file_format: 'jpeg', custom_fields: [], housekeeping: {} as any },
    { name: 'DB2', file_format: 'png', custom_fields: [], housekeeping: {} as any },
  ];

  const adminUser: User = { id: 1, username: 'admin', can_view: true, can_create: true, can_edit: true, can_delete: true };
  const viewUser: User = { id: 2, username: 'viewer', can_view: true, can_create: false, can_edit: false, can_delete: false };

  beforeEach(async () => {
    const dbServiceSpy = jasmine.createSpyObj('DatabaseService', ['selectDatabase', 'loadDatabases'], {
      // Use 'of()' to make databases$ an observable
      databases$: of(mockDatabases),
      selectedDatabase$: of(mockDatabases[0])
    });

    const authServiceSpy = jasmine.createSpyObj('AuthService', ['logout'], {
      currentUser$: of(adminUser) // Default to admin user
    });

    const modalServiceSpy = jasmine.createSpyObj('ModalService', ['open']);

    await TestBed.configureTestingModule({
      declarations: [ SidebarComponent, MockCreateDatabaseModalComponent ],
      imports: [ RouterTestingModule ],
      providers: [
        { provide: DatabaseService, useValue: dbServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        { provide: ModalService, useValue: modalServiceSpy },
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SidebarComponent);
    component = fixture.componentInstance;
    mockDatabaseService = TestBed.inject(DatabaseService) as jasmine.SpyObj<DatabaseService>;
    mockAuthService = TestBed.inject(AuthService) as jasmine.SpyObj<AuthService>;
    mockModalService = TestBed.inject(ModalService) as jasmine.SpyObj<ModalService>;
    
    fixture.detectChanges(); // Initial data binding
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display a list of databases', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const dbLinks = compiled.querySelectorAll('.db-link');
    expect(dbLinks.length).toBe(2);
    expect(dbLinks[0].textContent).toContain('DB1');
    expect(dbLinks[1].textContent).toContain('DB2');
  });

  it('should highlight the selected database', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const activeLink = compiled.querySelector('.db-link.active');
    expect(activeLink).toBeTruthy();
    expect(activeLink?.textContent).toContain('DB1');
  });

  it('should show the "Create" button for a user with can_create role', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const createButton = compiled.querySelector('.create-db-btn');
    expect(createButton).toBeTruthy();
  });

  it('should hide the "Create" button for a user without can_create role', () => {
    // Override the auth service's observable for this test
    (Object.getOwnPropertyDescriptor(mockAuthService, 'currentUser$')?.get as jasmine.Spy).and.returnValue(of(viewUser));
    
    fixture.detectChanges(); // Rerun change detection with the new user

    const compiled = fixture.nativeElement as HTMLElement;
    const createButton = compiled.querySelector('.create-db-btn');
    expect(createButton).toBeFalsy();
  });

  it('should call DatabaseService.selectDatabase when a database link is clicked', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const db2Link = compiled.querySelectorAll('.db-link')[1] as HTMLElement;
    db2Link.click();

    expect(mockDatabaseService.selectDatabase).toHaveBeenCalledWith('DB2');
  });

  it('should call AuthService.logout when the logout button is clicked', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const logoutButton = compiled.querySelector('.logout-btn') as HTMLElement;
    logoutButton.click();
    
    expect(mockAuthService.logout).toHaveBeenCalled();
  });

  it('should call ModalService.open when the create button is clicked', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const createButton = compiled.querySelector('.create-db-btn') as HTMLElement;
    createButton.click();

    expect(mockModalService.open).toHaveBeenCalled();
  });
});
