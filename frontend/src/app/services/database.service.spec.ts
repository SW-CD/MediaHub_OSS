// frontend/src/app/services/database.service.spec.ts

import { TestBed, fakeAsync, tick, discardPeriodicTasks } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { Router } from '@angular/router'; 
import { DatabaseService } from './database.service';
import { AuthService } from './auth.service';
import { NotificationService } from './notification.service';
import { Database, Entry, SearchRequest } from '../models/api.models';
import { ContentType, EntryStatus } from '../models/enums';
import { of } from 'rxjs';

describe('DatabaseService', () => {
  let service: DatabaseService;
  let httpMock: HttpTestingController;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;
  let mockRouter: jasmine.SpyObj<Router>;

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
    const notificationServiceSpy = jasmine.createSpyObj('NotificationService', ['showSuccess', 'showError', 'showGlobalError', 'showInfo']);
    const routerSpy = jasmine.createSpyObj('Router', ['navigate']);

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule], 
      providers: [
        DatabaseService,
        { provide: AuthService, useValue: authServiceSpy },
        { provide: NotificationService, useValue: notificationServiceSpy },
        { provide: Router, useValue: routerSpy }
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
            expect(mockRouter.navigate).toHaveBeenCalledWith(['/dashboard/db', 'NewDB']);
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

  // --- NEW: Bulk Action Tests ---

  describe('bulkDeleteEntries', () => {
    it('should send a POST request to delete multiple entries', (done) => {
      const dbName = 'TestDB1';
      const idsToDelete = [101, 102, 103];

      service.bulkDeleteEntries(dbName, idsToDelete).subscribe(() => {
        expect(mockNotificationService.showSuccess).toHaveBeenCalledWith('Successfully deleted 3 entries.');
        done();
      });

      const req = httpMock.expectOne(r => r.url === '/api/database/entries/delete' && r.params.get('name') === dbName);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ ids: idsToDelete });
      
      req.flush({ database_name: dbName, deleted_count: 3, message: 'Deleted' });
    });
  });

  describe('bulkExportEntries', () => {
    it('should send a POST request and return a Blob', (done) => {
      const dbName = 'TestDB1';
      const idsToExport = [201, 202];
      const mockBlob = new Blob(['zip content'], { type: 'application/zip' });

      service.bulkExportEntries(dbName, idsToExport).subscribe(blob => {
        expect(blob).toEqual(mockBlob);
        done();
      });

      const req = httpMock.expectOne(r => r.url === '/api/database/entries/export' && r.params.get('name') === dbName);
      expect(req.request.method).toBe('POST');
      expect(req.request.responseType).toBe('blob'); // Crucial check
      expect(req.request.body).toEqual({ ids: idsToExport });

      req.flush(mockBlob);
    });
  });

  // --- End Bulk Action Tests ---

  describe('uploadEntry', () => {
    const dbName = 'TestDB1';
    const file = new File([''], 'test.png', { type: 'image/png' });
    const metadata = { timestamp: 123456 };

    it('should handle 201 Created with Ready status (No Polling)', () => {
      service.uploadEntry(dbName, metadata, file).subscribe();

      const req = httpMock.expectOne(r => r.url === '/api/entry');
      expect(req.request.method).toBe('POST');
      
      req.flush({ id: 100, status: 'ready' }, { status: 201, statusText: 'Created' });

      httpMock.expectNone('/api/entry/meta');
    });

    it('should handle 201 Created with Processing status (Triggers Polling)', fakeAsync(() => {
      service.uploadEntry(dbName, metadata, file).subscribe();

      const req = httpMock.expectOne(r => r.url === '/api/entry');
      req.flush({ id: 101, status: 'processing' }, { status: 201, statusText: 'Created' });

      tick(2000);

      const pollReq = httpMock.expectOne(r => r.url === '/api/entry/meta' && r.params.get('id') === '101');
      expect(pollReq.request.method).toBe('GET');
      
      pollReq.flush({ id: 101, status: 'ready' });

      expect(mockNotificationService.showSuccess).toHaveBeenCalledWith('Entry uploaded successfully.');
      
      discardPeriodicTasks();
    }));

    it('should handle 202 Accepted (Async/Large File) and trigger polling', fakeAsync(() => {
      service.uploadEntry(dbName, metadata, file).subscribe();

      const req = httpMock.expectOne(r => r.url === '/api/entry');
      req.flush({ id: 102, status: 'processing' }, { status: 202, statusText: 'Accepted' });

      expect(mockNotificationService.showInfo).toHaveBeenCalledWith(jasmine.stringMatching(/processing/));

      tick(2000);

      const pollReq = httpMock.expectOne(r => r.url === '/api/entry/meta' && r.params.get('id') === '102');
      pollReq.flush({ id: 102, status: 'ready' });

      discardPeriodicTasks();
    }));
  });
});