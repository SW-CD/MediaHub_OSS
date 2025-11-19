# Concept: Frontend Migration to JWT Authentication

## 1\. 🎯 Objective

Refactor the Angular frontend to switch from **Basic Authentication** to **JWT (JSON Web Tokens)**. This involves managing Access and Refresh tokens, handling token expiration automatically, and cleaning up existing services to remove manual header injection.

## 2\. 🏗️ Architectural Changes

### Current State (Basic Auth)

  * **AuthService:** Encodes `username:password` to Base64 and stores it.
  * **DatabaseService:** Manually retrieves the token and adds `Authorization: Basic ...` to every request.
  * **SecureImagePipe:** Manually retrieves the token for image fetching.

### Target State (JWT + Interceptor)

  * **AuthService:** Manages `login` (fetch tokens), `logout` (revoke tokens), and token storage (`localStorage`).
  * **JwtInterceptor (New):** A single file that:
    1.  Intercepts **all** outgoing HTTP requests.
    2.  Injects `Authorization: Bearer <access_token>`.
    3.  Catches `401 Unauthorized` errors.
    4.  Triggers the **Token Refresh** flow seamlessly without the user noticing.
  * **DatabaseService & Pipe:** Stripped of auth logic. They simply make HTTP requests.

-----

## 3\. 🛠️ Implementation Steps

### Step 1: Update Data Models

We need a TypeScript interface for the response coming from `POST /api/token`.

**File:** `frontend/src/app/models/api.models.ts`

```typescript
export interface TokenResponse {
  access_token: string;
  refresh_token: string;
}
```

### Step 2: Refactor AuthService

The `AuthService` needs to completely change its internal logic. It will no longer store a "Basic" string.

**File:** `frontend/src/app/services/auth.service.ts`

  * **Storage:** Use `localStorage` instead of `sessionStorage` to persist login across tab closes (standard for JWT), or stick to `sessionStorage` for stricter security.
  * **Login (`login()`):**
      * Input: `username`, `password`.
      * Action: Call `POST /api/token` with Basic Auth headers (one-time use).
      * On Success: Save `access_token` and `refresh_token`. Fetch User details.
  * **Logout (`logout()`):**
      * Action: Call `POST /api/logout` sending the `refresh_token`.
      * Cleanup: Remove tokens from storage, navigate to `/login`.
  * **Refresh (`refreshToken()`):**
      * Action: Call `POST /api/token/refresh` with `{ refresh_token: ... }`.
      * On Success: Update stored tokens.
      * On Failure: Force logout.
  * **Helpers:** Add `getAccessToken()` and `getRefreshToken()`.

### Step 3: Create the JWT Interceptor

This is the most complex but valuable part. It handles the "Session Expired" scenario automatically.

**File:** `frontend/src/app/interceptors/jwt.interceptor.ts` (New File)

  * **Logic:**
    1.  Clones the request and adds `Authorization: Bearer <token>`.
    2.  `catchError`: If error is `401`:
          * If already refreshing: Wait for the refresh to finish (using a `BehaviorSubject`).
          * If not refreshing: Set `isRefreshing = true`, call `authService.refreshToken()`.
          * When token returns: Retry the failed request with the new token.
          * If refresh fails: `authService.logout()`.

### Step 4: Register the Interceptor

Angular needs to know about the interceptor.

**File:** `frontend/src/app/app.module.ts`

  * Import `HTTP_INTERCEPTORS` from `@angular/common/http`.
  * Import `JwtInterceptor`.
  * Add to `providers`:
    ```typescript
    providers: [
      { provide: HTTP_INTERCEPTORS, useClass: JwtInterceptor, multi: true }
    ],
    ```

### Step 5: Clean Up DatabaseService

Remove the manual header construction. This makes the service much smaller.

**File:** `frontend/src/app/services/database.service.ts`

  * **Remove:** `getAuthHeaders()` method.
  * **Remove:** All calls to `this.authService.getAuthToken()`.
  * **Refactor:** All `this.http.get/post` calls should just pass the body/params. The Interceptor handles the headers.

### Step 6: Clean Up SecureImagePipe

Similar to the service, the pipe shouldn't manually handle tokens.

**File:** `frontend/src/app/pipes/secure-image.pipe.ts`

  * **Refactor:** `HttpClient` will now go through the Interceptor, so we just need to make the `GET` request.
  * *Note:* The pipe uses `HttpClient` to fetch a `Blob`. The interceptor *will* work here automatically, so we can remove the manual header injection code.

-----

## 4\. 🧪 Testing Strategy

Since we are changing the core communication layer, manual testing is required after implementation:

1.  **Login Flow:** Ensure logging in stores two tokens in the browser Application/Storage tab.
2.  **Normal Usage:** Click around. Ensure `Authorization: Bearer ...` is present in Network tab request headers.
3.  **Refresh Flow (Hard to test manually, but crucial):**
      * Login.
      * Manually modify the `access_token` in LocalStorage (delete the last character) to invalidate it.
      * Click a button that fetches data (e.g., refresh dashboard).
      * **Expected:**
          * Network tab shows a red `401` for the data request.
          * Immediately followed by a `POST /api/token/refresh`.
          * Immediately followed by a retry of the data request (Success).
          * User sees no error, app continues working.
4.  **Logout Flow:** Ensure clicking logout calls the API and clears local storage.

-----

## 5\. ⚠️ Potential Pitfalls & Solutions

  * **Circular Dependency:** `JwtInterceptor` depends on `AuthService`, and `AuthService` uses `HttpClient`.
      * *Solution:* The Interceptor should rely on `AuthService` for token management, but `AuthService` should use a generic `HttpClient`. This is usually fine in modern Angular. If an issue arises, we can inject `Injector` to lazy-load `AuthService` inside the interceptor.
  * **Concurrent Refreshes:** If the page loads and fires 5 requests, and the token is expired, we don't want to call `/refresh` 5 times.
      * *Solution:* The `isRefreshing` flag and `refreshTokenSubject` logic in the Interceptor (Step 3) specifically prevents this.
