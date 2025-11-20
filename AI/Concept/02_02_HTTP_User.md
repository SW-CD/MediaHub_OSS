
### User endpoints

#### GET /api/me

Retrieves the user record and all assigned roles for the currently authenticated user. This allows the frontend to dynamically adjust the UI based on the user's permissions.

**Role Required: None (any authenticated user)**

##### Request

`GET /api/me`

##### Success Response

**Status 200 - OK**

Returns the user object from the `users` table (excluding password hashes/salts).

```json
{
    "id": 1,
    "username": "admin",
    "can_view": true,
    "can_create": true,
    "can_edit": true,
    "can_delete": true,
    "is_admin": true
}
```

*Another example for a read-only user:*

```json
{
    "id": 3,
    "username": "viewer",
    "can_view": true,
    "can_create": false,
    "can_edit": false,
    "can_delete": false,
    "is_admin": false
}
```

##### Error Responses

  * **Status 401 - Unauthorized**: Authentication failed (no credentials provided, or token/password is invalid).
    ```json
    {
        "error": "Authentication failed"
    }
    ```

-----

#### PATCH /api/me

Updates the password for the currently authenticated user.

**Role Required: None (any authenticated user)**

##### Request

`PATCH /api/me`

##### Request Body

```json
{
    "password": "my-new-secure-password-123"
}
```

##### Success Response

**Status 200 - OK**

```json
{
    "message": "Password updated successfully."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Invalid JSON or missing 'password' field.
  * **Status 401 - Unauthorized**: Authentication failed.

-----

#### GET /api/users

Retrieves a list of all user accounts and their assigned roles.
**Role Required: `IsAdmin`**

##### Request

`GET /api/users`

##### Success Response

**Status 200 - OK**

Returns an array of all user objects (excluding password info).

```json
[
    {
        "id": 1,
        "username": "admin",
        "can_view": true,
        "can_create": true,
        "can_edit": true,
        "can_delete": true,
        "is_admin": true
    },
    {
        "id": 3,
        "username": "viewer",
        "can_view": true,
        "can_create": false,
        "can_edit": false,
        "can_delete": false,
        "is_admin": false
    }
]
```

##### Error Responses

  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `IsAdmin` role.

-----

#### POST /api/user

Creates a new user account.
**Role Required: `IsAdmin`**

##### Request

`POST /api/user`

##### Request Body

```json
{
    "username": "new_editor",
    "password": "a-strong-password-123",
    "can_view": true,
    "can_create": true,
    "can_edit": true,
    "can_delete": false,
    "is_admin": false
}
```

##### Success Response

**Status 201 - Created**

Returns the new user object (excluding password).

```json
{
    "id": 4,
    "username": "new_editor",
    "can_view": true,
    "can_create": true,
    "can_edit": true,
    "can_delete": false,
    "is_admin": false
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Invalid JSON, or missing `username` or `password`.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `IsAdmin` role.
  * **Status 409 - Conflict**: Username 'new\_editor' already exists.

-----

#### PATCH /api/user

Updates an existing user's roles or password.
**Role Required: `IsAdmin`**

##### Request

`PATCH /api/user?id=4`

  * **id** (query param, required): The ID of the user to update.

##### Request Body (Updating Roles)

A JSON object containing only the fields to be updated.

```json
{
    "can_delete": true,
    "is_admin": true
}
```

##### Request Body (Updating Password)

```json
{
    "password": "a-new-stronger-password"
}
```

##### Success Response

**Status 200 - OK**

Returns the full, updated user object.

```json
{
    "id": 4,
    "username": "new_editor",
    "can_view": true,
    "can_create": true,
    "can_edit": true,
    "can_delete": true,
    "is_admin": true
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' param or invalid JSON body.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `IsAdmin` role.
  * **Status 404 - Not Found**: User with 'id=4' not found.
  * **Status 409 - Conflict**: Cannot remove `IsAdmin` from the last remaining admin user.

-----

#### DELETE /api/user

Deletes a user account.
**Role Required: `IsAdmin`**

##### Request

`DELETE /api/user?id=4`

  * **id** (query param, required): The ID of the user to delete.

##### Success Response

**Status 200 - OK**

```json
{
    "message": "User 'new_editor' (ID: 4) was successfully deleted."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `IsAdmin` role.
  * **Status 404 - Not Found**: User with 'id=4' not found.
  * **Status 409 - Conflict**: Cannot delete the last remaining admin user.
