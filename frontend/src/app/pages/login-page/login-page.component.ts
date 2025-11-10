// src/app/pages/login-page/login-page.component.ts
import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { finalize } from 'rxjs/operators';
import { HttpErrorResponse } from '@angular/common/http'; // Import HttpErrorResponse

@Component({
  selector: 'app-login-page',
  templateUrl: './login-page.component.html',
  styleUrls: ['./login-page.component.css'],
  standalone: false
})
export class LoginPageComponent implements OnInit {
  loginForm: FormGroup; // <-- FIX: Removed `!`
  isLoading = false;
  loginError: string | null = null;

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private router: Router,
    private notificationService: NotificationService
  ) {
    console.log('[LOGIN] Constructor - Initializing form.');
    // FIX: Initialize the form in the constructor to prevent race conditions.
    this.loginForm = this.fb.group({
      username: ['', Validators.required],
      password: ['', Validators.required],
    });
  }

  ngOnInit(): void {
    console.log('[LOGIN] ngOnInit - Component Initialized.');
    if (this.authService.getCurrentUser()) {
      console.log('[LOGIN] User is already logged in. Navigating to dashboard.');
      this.router.navigate(['/dashboard']);
    }
  }

  onSubmit(): void {
    console.log('[LOGIN] onSubmit - Login form submitted.');
    if (this.loginForm.invalid) {
      console.log('[LOGIN] Form is invalid. Aborting submission.');
      return;
    }
    this.isLoading = true;
    this.loginError = null;
    this.notificationService.clearGlobalError();
    const { username, password } = this.loginForm.value;
    console.log(`[LOGIN] Calling authService.login for user: '${username}'`);
    this.authService
      .login(username, password)
      .pipe(
        finalize(() => {
          this.isLoading = false;
          console.log('[LOGIN] Login request finalized.');
        })
      )
      .subscribe({
        next: (user) => {
          console.log('[LOGIN] Login successful. Navigating to dashboard.');
          this.router.navigate(['/dashboard']);
        },
        // --- FIX: Update error handler to expect HttpErrorResponse ---
        error: (err: HttpErrorResponse) => {
          console.error('[LOGIN] Login failed:', err); // This log will now show the full HttpErrorResponse

          if (err.status === 401) {
            this.loginError = 'Invalid username or password.';
          } else if (err.status && err.status >= 500) {
            this.loginError = 'A server error occurred. Please try again later.';
            this.notificationService.showGlobalError(
              'Server error. Please try again later.'
            );
          } else {
            this.loginError = 'An unexpected error occurred.';
          }
        },
      });
  }
}