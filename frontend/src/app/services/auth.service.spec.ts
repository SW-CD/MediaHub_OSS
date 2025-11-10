// frontend/src/app/services/auth.service.spec.ts

import { TestBed } from '@angular/core/testing';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { Router } from '@angular/router';

import { AuthService } from './auth.service';
import { User } from '../models/api.models';

describe('AuthService', () => {
  let service: AuthService;
  let httpMock: HttpTestingController;
  let router: Router;

  const mockUser: User = {
    id: 1,
    username: 'test',
    can_view: true,
    can_create: false,
    can_edit: false,
    can_delete: false,
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule, RouterTestingModule.withRoutes([])],
      providers: [AuthService],
    });
    service = TestBed.inject(AuthService);
    httpMock = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
  });

  afterEach(() => {
    // Verify that there are no outstanding HTTP requests.
    httpMock.verify();
    // Clear session storage after each test
    sessionStorage.clear();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('login', () => {
    it('should authenticate user, store token, and update currentUser$', (done) => {
      service.login('test', 'password').subscribe((user) => {
        expect(user).toEqual(mockUser);
        expect(service.getCurrentUser()).toEqual(mockUser);
        expect(sessionStorage.getItem('authToken')).toBe('Basic ' + btoa('test:password'));
        done();
      });

      // Expect a GET request to /api/me and flush it with mock data
      const req = httpMock.expectOne('/api/me');
      expect(req.request.method).toBe('GET');
      expect(req.request.headers.get('Authorization')).toBe(
        'Basic ' + btoa('test:password')
      );
      req.flush(mockUser);
    });

    it('should handle login failure and clear session data', (done) => {
      // Spy on logout to ensure it's called
      spyOn(service, 'logout').and.callThrough();

      service.login('test', 'wrongpassword').subscribe({
        next: () => fail('login should have failed'),
        error: (err) => {
          expect(err.message).toBe('Invalid username or password');
          expect(service.getCurrentUser()).toBeNull();
          expect(sessionStorage.getItem('authToken')).toBeNull();
          // Because logout is called on error, it triggers navigation
          // expect(service.logout).toHaveBeenCalled();
          done();
        },
      });

      const req = httpMock.expectOne('/api/me');
      req.flush({ error: 'Unauthorized' }, { status: 401, statusText: 'Unauthorized' });
    });
  });

  describe('logout', () => {
    it('should clear user data, remove token, and navigate to login', () => {
      // First, simulate a login to set state
      service.login('test', 'password').subscribe();
      const req = httpMock.expectOne('/api/me');
      req.flush(mockUser);
      
      expect(service.getCurrentUser()).not.toBeNull();

      // Spy on router navigation
      const navigateSpy = spyOn(router, 'navigate');

      // Now, test logout
      service.logout();

      expect(service.getCurrentUser()).toBeNull();
      expect(sessionStorage.getItem('authToken')).toBeNull();
      expect(navigateSpy).toHaveBeenCalledWith(['/login']);
    });
  });

  describe('hasRole', () => {
    it('should return true if the user has the role', (done) => {
      service.login('test', 'password').subscribe(() => {
        expect(service.hasRole('can_view')).toBeTrue();
        done();
      });
      const req = httpMock.expectOne('/api/me');
      req.flush(mockUser);
    });

    it('should return false if the user does not have the role', (done) => {
      service.login('test', 'password').subscribe(() => {
        expect(service.hasRole('can_create')).toBeFalse();
        done();
      });
      const req = httpMock.expectOne('/api/me');
      req.flush(mockUser);
    });

    it('should return false if no user is logged in', () => {
      expect(service.hasRole('can_view')).toBeFalse();
    });
  });
});
