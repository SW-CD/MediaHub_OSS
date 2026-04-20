import { Component, OnInit, OnDestroy } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { AppInfoService } from '../../services/app-info.service';
import { AppInfo } from '../../models';
import { finalize, takeUntil } from 'rxjs/operators';
import { Subject } from 'rxjs';
import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-login-page',
  templateUrl: './login-page.component.html',
  styleUrls: ['./login-page.component.css'],
  standalone: false
})
export class LoginPageComponent implements OnInit, OnDestroy {
  loginForm: FormGroup;
  isLoading = false;
  loginError: string | null = null;
  appInfo: AppInfo | null = null;
  
  private destroy$ = new Subject<void>();

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private router: Router,
    private notificationService: NotificationService,
    private appInfoService: AppInfoService
  ) {
    this.loginForm = this.fb.group({
      username: ['', Validators.required],
      password: ['', Validators.required],
    });
  }

  ngOnInit(): void {
    if (this.authService.getCurrentUser()) {
      this.router.navigate(['/dashboard']);
      return;
    }

    // Load AppInfo to check OIDC settings
    this.appInfoService.loadInfo().pipe(takeUntil(this.destroy$)).subscribe(info => {
      this.appInfo = info;
      
      // UPDATED: Check nested oidc object properties
      if (this.appInfo?.oidc?.enabled && this.appInfo?.oidc?.login_page_disabled) {
        // Automatically redirect to OIDC if the local login page is disabled
        this.redirectToOIDC();
      }
    });
  }

  onSubmit(): void {
    if (this.loginForm.invalid) return;
    
    this.isLoading = true;
    this.loginError = null;
    this.notificationService.clearGlobalError();
    
    const { username, password } = this.loginForm.value;
    
    // UPDATED: Call the new basicAuthLogin method
    this.authService.basicAuthLogin(username, password).pipe(
      finalize(() => this.isLoading = false)
    ).subscribe({
      next: () => this.router.navigate(['/dashboard']),
      error: (err: HttpErrorResponse) => {
        if (err.status === 401) {
          this.loginError = 'Invalid username or password.';
        } else if (err.status === 403) {
          this.loginError = 'Local login is disabled by the server configuration.';
        } else {
          this.loginError = 'A server error occurred. Please try again later.';
        }
      },
    });
  }

  redirectToOIDC(): void {
    // Simplify access by grabbing the nested object
    const oidcConfig = this.appInfo?.oidc;

    // UPDATED: Check the nested properties
    if (!oidcConfig?.issuer_url || !oidcConfig?.client_id) {
      this.loginError = 'OIDC configuration is missing from the server.';
      return;
    }
    
    // Construct the standard OIDC Authorization URL using the nested object
    const authEndpoint = `${oidcConfig.issuer_url}/protocol/openid-connect/auth`;
    const clientId = encodeURIComponent(oidcConfig.client_id);
    const redirectUri = encodeURIComponent(oidcConfig.redirect_url || window.location.origin + '/login');
    
    const oidcUrl = `${authEndpoint}?client_id=${clientId}&redirect_uri=${redirectUri}&response_type=code&scope=openid`;
    
    // Redirect the browser to Keycloak
    window.location.href = oidcUrl;
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}