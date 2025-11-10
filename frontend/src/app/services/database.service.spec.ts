// frontend/src/app/services/database.service.spec.ts

import { TestBed } from '@angular/core/testing';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { DatabaseService } from './database.service';
import { AuthService } from './auth.service';
import { NotificationService } from './notification.service';
import { Database, Image } from '../models/api.models';

describe('DatabaseService', () => {
  let service: DatabaseService;
  let httpMock: HttpTestingController;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let mockNotificationService: jasmine.SpyObj<NotificationService>;

  const mockDatabases: Database[] = [
    {
      name: 'TestDB1',
      file_format: 'jpeg',
      housekeeping: { interval: '1h', disk_space: '100G', max_age: '365d' },
      custom_fields: [],
    },
  ];

  beforeEach(() => {
    // Create spies for the service dependencies
    const authServiceSpy = jasmine.createSpyObj('AuthService', [
      'getAuthToken',
      'logout',
    ]);
    const notificationServiceSpy = jasmine.createSpyObj('NotificationService', [
      'showSuccess',
      'showError',
      'showGlobalError',
    ]);

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule, RouterTestingModule],
      providers: [
        DatabaseService,
        { provide: AuthService, useValue: authServiceSpy },
        { provide: NotificationService, useValue: notificationServiceSpy },
      ],
    });

    service = TestBed.inject(DatabaseService);
    httpMock = TestBed.inject(HttpTestingController);
    mockAuthService = TestBed.inject(
      AuthService
    ) as jasmine.SpyObj<AuthService>;
    mockNotificationService = TestBed.inject(
      NotificationService
    ) as jasmine.SpyObj<NotificationService>;

    // Default mock for getAuthToken to return a valid token
    mockAuthService.getAuthToken.and.returnValue('Basic dXNlcjpwYXNz');
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

  describe('getImages', () => {
    it('should fetch images with correct filters', (done) => {
        const mockImages: Image[] = [{id: 1, description: 'test', timestamp: 123, width: 100, height: 100, filesize: 1024}];
        const filters = { limit: 50, search: 'test' };

        service.getImages('TestDB1', filters).subscribe(images => {
            expect(images.length).toBe(1);
            done();
        });

        const req = httpMock.expectOne(r => r.url === '/api/database/images');
        expect(req.request.method).toBe('GET');
        expect(req.request.params.get('name')).toBe('TestDB1');
        expect(req.request.params.get('limit')).toBe('50');
        expect(req.request.params.get('search')).toBe('test');
        req.flush(mockImages);
    });
  });

  describe('createDatabase', () => {
    it('should send a POST request to create a database', (done) => {
        const newDbData: Partial<Database> = { name: 'NewDB', file_format: 'png' };
        const responseDb: Database = { ...newDbData, housekeeping: {} as any, custom_fields: [] };

        service.createDatabase(newDbData).subscribe(db => {
            expect(db.name).toBe('NewDB');
            expect(mockNotificationService.showSuccess).toHaveBeenCalled();
            done();
        });

        const req = httpMock.expectOne('/api/database');
        expect(req.request.method).toBe('POST');
        req.flush(responseDb);

        // Expect a call to load databases after creation
        const req2 = httpMock.expectOne('/api/databases');
        req2.flush([]);
    });
  });
  
  describe('deleteImage', () => {
    it('should send a DELETE request to remove an image', (done) => {
      const dbName = 'TestDB1';
      const imageId = 123;

      service.deleteImage(dbName, imageId).subscribe(response => {
        expect(response.message).toBe('Image deleted.');
        expect(mockNotificationService.showSuccess).toHaveBeenCalledWith('Image deleted.');
        done();
      });

      const req = httpMock.expectOne(`/api/image?database_name=${dbName}&id=${imageId}`);
      expect(req.request.method).toBe('DELETE');
      req.flush({ message: 'Image deleted.' });
    });
  });
  
  describe('Error Handling', () => {
    it('should handle 401 Unauthorized by logging out', (done) => {
      service.loadDatabases().subscribe({
        next: () => fail('should have failed with 401'),
        error: (err) => {
          expect(mockNotificationService.showGlobalError).toHaveBeenCalled();
          expect(mockAuthService.logout).toHaveBeenCalled();
          done();
        }
      });

      const req = httpMock.expectOne('/api/databases');
      req.flush({ error: 'Unauthorized' }, { status: 401, statusText: 'Unauthorized' });
    });
  });

});
