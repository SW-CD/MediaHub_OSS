// frontend/src/app/services/database.service.spec.ts

import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Router } from '@angular/router'; // Import Router
import { DatabaseService } from './database.service';
import { AuthService } from './auth.service';
import { NotificationService } from './notification.service';
import { Database, Entry, SearchRequest } from '../models/api.models';
import { ContentType, EntryStatus } from '../models/enums';

describe('DatabaseService', () => {
  let service: DatabaseService;
  let httpMock: HttpTestingController;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;
  let mockRouter: jasmine.SpyObj<Router>; // Spy for Router

  const mockDatabases: Database[] = [
    {
      name: 'TestDB1',
      content_type: ContentType.Image,
      config: {},
      housekeeping: { interval: '1h', disk_space: '100G', max_age: '365d' },
      custom_fields: [],
    },
  ];

  beforeEach(() => {
    const authServiceSpy = jasmine.createSpyObj('AuthService', ['getAccessToken', 'logout']);
    const notificationServiceSpy = jasmine.createSpyObj('NotificationService', ['showSuccess', 'showError', 'showGlobalError']);
    const routerSpy = jasmine.createSpyObj('Router', ['navigate']); // Create Router spy

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule], // No RouterTestingModule needed if we mock Router
      providers: [
        DatabaseService,
        { provide: AuthService, useValue: authServiceSpy },
        { provide: NotificationService, useValue: notificationServiceSpy },
        { provide: Router, useValue: routerSpy } // Provide the spy
      ],
    });

    service = TestBed.inject(DatabaseService);
    httpMock = TestBed.inject(HttpTestingController);
    mockAuthService = TestBed.inject(AuthService) as jasmine.SpyObj<AuthService>;
    mockNotificationService = TestBed.inject(NotificationService) as jasmine.SpyObj<NotificationService>;
    mockRouter = TestBed.inject(Router) as jasmine.SpyObj<Router>;
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('loadDatabases', () => {
    it('should fetch and return a list of databases', (done) => {
      service.loadDatabases().subscribe((dbs) => {
        expect(dbs.length).toBe(1);
        expect(dbs[0].name).toBe('TestDB1');
        done();
      });

      const req = httpMock.expectOne('/api/databases');
      expect(req.request.method).toBe('GET');
      req.flush(mockDatabases);
    });
  });

  describe('searchEntries', () => {
    it('should fetch entries with correct payload', (done) => {
        const mockEntries: Entry[] = [{
            id: 1, 
            timestamp: 123, 
            mime_type: 'image/jpeg', 
            filesize: 1000, 
            status: EntryStatus.Ready 
        }];
        
        const payload: SearchRequest = {
            pagination: { limit: 50, offset: 0 },
            sort: { field: 'timestamp', direction: 'desc' }
        };

        service.searchEntries('TestDB1', payload).subscribe(entries => {
            expect(entries.length).toBe(1);
            done();
        });

        const req = httpMock.expectOne(r => r.url === '/api/database/entries/search' && r.params.get('name') === 'TestDB1');
        expect(req.request.method).toBe('POST');
        expect(req.request.body).toEqual(payload);
        req.flush(mockEntries);
    });
  });

  describe('createDatabase', () => {
    it('should send a POST request and navigate on success', (done) => {
        const newDbData: Partial<Database> = { name: 'NewDB', content_type: ContentType.Image };
        const responseDb: Database = { 
            name: 'NewDB', 
            content_type: ContentType.Image, 
            config: {},
            housekeeping: {} as any, 
            custom_fields: [] 
        };

        service.createDatabase(newDbData).subscribe(db => {
            expect(db.name).toBe('NewDB');
            expect(mockNotificationService.showSuccess).toHaveBeenCalled();
            expect(mockRouter.navigate).toHaveBeenCalledWith(['/dashboard/db', 'NewDB']); // Verify navigation
            done();
        });

        const req = httpMock.expectOne('/api/database');
        expect(req.request.method).toBe('POST');
        req.flush(responseDb);

        const req2 = httpMock.expectOne('/api/databases'); // It reloads DBs
        req2.flush([]);
    });
  });
  
  describe('deleteEntry', () => {
    it('should send a DELETE request', (done) => {
      const dbName = 'TestDB1';
      const entryId = 123;

      service.deleteEntry(dbName, entryId).subscribe(response => {
        expect(response.message).toBe('Entry deleted.');
        expect(mockNotificationService.showSuccess).toHaveBeenCalled();
        done();
      });

      const req = httpMock.expectOne(`/api/entry?database_name=${dbName}&id=${entryId}`);
      expect(req.request.method).toBe('DELETE');
      req.flush({ message: 'Entry deleted.' });
    });
  });
});