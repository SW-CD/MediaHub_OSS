// frontend/src/app/components/sidebar/sidebar.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { of } from 'rxjs';
import { SidebarComponent } from './sidebar.component';
import { DatabaseService } from '../../services/database.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { AppInfoService } from '../../services/app-info.service';
import { User, Database } from '../../models/api.models';
import { ContentType } from '../../models/enums';
import { Component, Input } from '@angular/core';

// Mock CreateDatabaseModalComponent to avoid pulling in its dependencies
@Component({
    selector: 'app-create-database-modal',
    template: '',
    standalone: false
})
class MockCreateDatabaseModalComponent {}

describe('SidebarComponent', () => {
  let component: SidebarComponent;
  let fixture: ComponentFixture<SidebarComponent>;
  let mockDatabaseService: jasmine.SpyObj<DatabaseService>;
  let mockAuthService: jasmine.SpyObj<AuthService>;

  const mockDatabases: Database[] = [
    { name: 'DB1', content_type: ContentType.Image, config: {}, custom_fields: [], housekeeping: {} as any },
    { name: 'DB2', content_type: ContentType.Audio, config: {}, custom_fields: [], housekeeping: {} as any },
  ];

  const adminUser: User = { id: 1, username: 'admin', can_view: true, can_create: true, can_edit: true, can_delete: true, is_admin: true };

  beforeEach(async () => {
    const dbServiceSpy = jasmine.createSpyObj('DatabaseService', ['selectDatabase', 'loadDatabases'], {
      databases$: of(mockDatabases),
      selectedDatabase$: of(mockDatabases[0])
    });
    // Fix: loadDatabases must return an observable
    dbServiceSpy.loadDatabases.and.returnValue(of(mockDatabases));

    const authServiceSpy = jasmine.createSpyObj('AuthService', ['logout'], {
      currentUser$: of(adminUser)
    });

    const modalServiceSpy = jasmine.createSpyObj('ModalService', ['open']);
    const appInfoServiceSpy = jasmine.createSpyObj('AppInfoService', ['loadInfo'], {
      info$: of(null)
    });

    await TestBed.configureTestingModule({
      declarations: [ SidebarComponent, MockCreateDatabaseModalComponent ], // Declare mock
      imports: [ RouterTestingModule ],
      providers: [
        { provide: DatabaseService, useValue: dbServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        { provide: ModalService, useValue: modalServiceSpy },
        { provide: AppInfoService, useValue: appInfoServiceSpy }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SidebarComponent);
    component = fixture.componentInstance;
    mockDatabaseService = TestBed.inject(DatabaseService) as jasmine.SpyObj<DatabaseService>;
    
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display a list of databases', () => {
    const compiled = fixture.nativeElement as HTMLElement;
    const dbLinks = compiled.querySelectorAll('.db-link');
    expect(dbLinks.length).toBe(2);
    expect(dbLinks[0].textContent).toContain('DB1');
  });
});