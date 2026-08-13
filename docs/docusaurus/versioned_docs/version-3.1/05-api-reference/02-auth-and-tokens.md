---
sidebar_position: 2
title: Auth & Token Endpoints
---

# Authentication & Token Endpoints

Endpoints for acquiring, refreshing, and revoking JSON Web Tokens (JWT).

---

## `POST /api/token`

Obtain an Access and Refresh JWT pair using local Basic Auth or OIDC IdP Token Exchange.

* **Role Required**: None (Public)

### Request Method A: Local Credentials (Basic Auth)
* **Header**: `Authorization: Basic <base64(username:password)>`
* **Body**: None

### Request Method B: OIDC Token Exchange (Keycloak)
* **Header**: `Content-Type: application/json`
* **Body**:
  ```json
  {
    "idp_token": "eyJhbGciOi..."
  }
  ```

### Response (`200 OK`)
```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "user": {
    "id": "01J2A3X9D4B5C6E7F8G9H0J1K2",
    "username": "operator",
    "is_admin": false,
    "permissions": [
      {
        "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
        "can_view": true,
        "can_create": true,
        "can_edit": false,
        "can_delete": false,
        "can_admin": false
      }
    ]
  }
}
```

---

## `POST /api/token/refresh`

Exchange a valid Refresh Token for a new Access/Refresh token pair (Token Rotation).

* **Role Required**: None (Public)
* **Request Body**:
  ```json
  {
    "refresh_token": "eyJhbGciOi..."
  }
  ```
* **Response (`200 OK`)**:
  ```json
  {
    "access_token": "new_access_token...",
    "refresh_token": "new_refresh_token..."
  }
  ```

---

## `POST /api/logout`

Revoke a Refresh Token and end the current user session.

* **Role Required**: Authenticated User
* **Header**: `Authorization: Bearer <access_token>`
* **Request Body**:
  ```json
  {
    "refresh_token": "eyJhbGciOi..."
  }
  ```
* **Response (`200 OK`)**:
  ```json
  {
    "message": "Logged out successfully."
  }
  ```
