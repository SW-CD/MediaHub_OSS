---
sidebar_position: 4
title: User Management Endpoints
---

# User Management Endpoints

Endpoints for retrieving current user profile, updating personal credentials, and administrative user account management.

---

## `GET /api/me`

Retrieves the user record and specific database permissions for the currently authenticated user.

* **Role Required**: None (Any Authenticated User)
* **Response (`200 OK`)**:
  ```json
  {
    "id": "01J2A3X9D4B5C6E7F8G9H0J1K2",
    "username": "viewer",
    "is_admin": false,
    "is_service_account": false,
    "permissions": [
      {
        "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
        "can_view": true,
        "can_create": false,
        "can_edit": false,
        "can_delete": false,
        "can_admin": false
      }
    ]
  }
  ```

---

## `PATCH /api/me`

Updates the password for the currently authenticated user.

* **Role Required**: None (Any Authenticated User)
* **Request Body**:
  ```json
  {
    "old_password": "my-old-password",
    "new_password": "my-new-secure-password-123"
  }
  ```
* **Response (`200 OK`)**:
  ```json
  {
    "message": "Password updated successfully."
  }
  ```

---

## `GET /api/users`

Retrieves a list of all registered user accounts and their database permissions.

* **Role Required**: `IsAdmin`
* **Query Parameters**:
  * `is_service_account` (boolean, optional): If `true`, returns only service accounts; if `false`, returns standard users; if omitted, returns all users.
* **Response (`200 OK`)**:
  ```json
  [
    {
      "id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
      "username": "admin",
      "is_admin": true,
      "is_service_account": false,
      "permissions": []
    },
    {
      "id": "01J2A3X9D4B5C6E7F8G9H0J1K2",
      "username": "editor_bob",
      "is_admin": false,
      "is_service_account": false,
      "permissions": [
        {
          "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
          "can_view": true,
          "can_create": true,
          "can_edit": true,
          "can_delete": false,
          "can_admin": false
        }
      ]
    }
  ]
  ```

---

## `POST /api/user`

Creates a new user or service account.

* **Role Required**: `IsAdmin`
* **Request Body**:
  ```json
  {
    "username": "new_editor",
    "password": "a-strong-password-123",
    "is_admin": false,
    "is_service_account": false,
    "permissions": [
      {
        "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
        "can_view": true,
        "can_create": true,
        "can_edit": true,
        "can_delete": false,
        "can_admin": false
      }
    ]
  }
  ```
  *(Note: If `is_service_account` is `true`, `password` is optional).*
* **Response (`201 Created`)**:
  ```json
  {
    "id": "01K3B4Y0E5C6D7F8G9H0J1K2L3",
    "username": "new_editor",
    "is_admin": false,
    "is_service_account": false,
    "permissions": [ ... ]
  }
  ```

---

## `GET /api/user/{id}`

Retrieves details and permissions for a specific user account.

* **Role Required**: `IsAdmin`
* **Path Parameters**:
  * `id` (string, required): The ULID of the user.
* **Response (`200 OK`)**: Returns user object.

---

## `PATCH /api/user/{id}`

Updates an existing user's username, `is_admin` status, password, or database permissions.

* **Role Required**: `IsAdmin`
* **Path Parameters**:
  * `id` (string, required): The ULID of the user to update.
* **Request Body Example (Permissions Upsert/Replace)**:
  ```json
  {
    "username": "UpdatedUsername",
    "is_admin": false,
    "permissions": [
      {
        "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
        "can_view": true,
        "can_create": false,
        "can_edit": false,
        "can_delete": false,
        "can_admin": false
      }
    ]
  }
  ```
* **Response (`200 OK`)**: Returns full updated user object.

---

## `DELETE /api/user/{id}`

Deletes a user account and removes all associated database permissions.

* **Role Required**: `IsAdmin`
* **Path Parameters**:
  * `id` (string, required): The ULID of the user.
* **Response (`200 OK`)**:
  ```json
  {
    "message": "User 'new_editor' (ID: 01K3B4Y0E5C6D7F8G9H0J1K2L3) was successfully deleted."
  }
  ```
