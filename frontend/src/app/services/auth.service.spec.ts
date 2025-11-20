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
    is_admin: false // Added
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
    httpMock.verify();
    sessionStorage.clear();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('login', () => {
    it('should authenticate user, store token, and update currentUser$', (done) => {
      service.login('test', 'password').subscribe((user) => {
        expect(user).toEqual(mockUser);
        // Auth tokens are now in localStorage, not sessionStorage.
        expect(localStorage.getItem('access_token')).toBe('access123');
        done();
      });

      // 1. Expect POST to /api/token
      const reqToken = httpMock.expectOne('/api/token');
      expect(reqToken.request.method).toBe('POST');
      expect(reqToken.request.headers.get('Authorization')).toBe('Basic ' + btoa('test:password'));
      reqToken.flush({ access_token: 'access123', refresh_token: 'refresh123' });

      // 2. Expect GET to /api/me
      const reqMe = httpMock.expectOne('/api/me');
      expect(reqMe.request.method).toBe('GET');
      reqMe.flush(mockUser);
    });
  });
});