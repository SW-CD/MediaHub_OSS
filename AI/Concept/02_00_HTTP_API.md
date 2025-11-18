## API description

### Authentication

All API endpoints (unless specified) are protected using role-based access control (RBAC) and are available under the `/api` path. The server supports a **Hybrid Authentication** model. You may authenticate using either:

1.  **Basic Authentication**: Send the standard `Authorization: Basic <base64_credentials>` header.
2.  **Bearer Token (JWT)**: Send an `Authorization: Bearer <access_token>` header.

**Token Behavior:**
* **Access Tokens:** Short-lived (default 5 mins). Stateless. Used for authorizing API requests.
* **Refresh Tokens:** Long-lived (default 24 hours). **Stateful**. The hash of the refresh token is stored in the database. This allows the server to revoke sessions (Logout).

Users and their roles are defined in the SQLite database. The available roles are stored as boolean flags on the user's record:

  * **CanView**: Allows read-only access (e.g., `GET` requests).
  * **CanCreate**: Allows creating new resources (e.g., `POST /api/database`, `POST /api/entry`).
  * **CanEdit**: Allows modifying existing resources (e.g., `PUT /api/database`, `PATCH /api/entry`).
  * **CanDelete**: Allows deleting resources (e.g., `DELETE /api/database`, `DELETE /api/entry`, `POST /api/database/housekeeping`).
  * **IsAdmin**: Allows access to user management endpoints (e.g., `POST /api/user`, `GET /api/users`).

Endpoints will return a **Status 401 - Unauthorized** if no valid authentication is provided, or a **Status 403 - Forbidden** if the authenticated user does not have the required role.

### Error Responses

All error responses (4xx, 5xx) return a JSON body with a standard structure:

```json
{
    "error": "A brief, human-readable error message"
}
```

-----

### Authentication & Token Endpoints

#### POST /api/token

Exchanges Basic Auth credentials for a JWT Access/Refresh token pair.
**Role Required: None (Public, but requires valid Basic Auth headers)**

##### Request

`POST /api/token`

  * **Headers:** `Authorization: Basic <base64(username:password)>`

##### Success Response

**Status 200 - OK**

```json
{
    "access_token": "eyJhbGciOiJIUzI1Ni...",
    "refresh_token": "eyJhbGciOiJIUzI1Ni..."
}
```

##### Error Responses

  * **Status 401 - Unauthorized**: Invalid username or password, or missing Basic Auth header.

-----

#### POST /api/token/refresh

Uses a valid Refresh Token to obtain a new Access/Refresh token pair. The old refresh token is immediately revoked (Token Rotation).
**Role Required: None (Public)**

##### Request

`POST /api/token/refresh`

##### Request Body

```json
{
    "refresh_token": "eyJhbGciOiJIUzI1Ni..."
}
```

##### Success Response

**Status 200 - OK**

```json
{
    "access_token": "new_access_token...",
    "refresh_token": "new_refresh_token..."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Invalid JSON body.
  * **Status 401 - Unauthorized**: The refresh token is invalid, expired, or has been revoked (logged out).

-----

#### POST /logout

Revokes a Refresh Token, effectively logging the user out.
**Role Required: None (Authenticated)**

##### Request

`POST /api/logout`

  * **Headers:** `Authorization: Bearer <access_token>` (or Basic Auth)

##### Request Body

```json
{
    "refresh_token": "eyJhbGciOiJIUzI1Ni..."
}
```

##### Success Response

**Status 200 - OK**

```json
{
    "message": "Logged out successfully."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Invalid JSON body.
  * **Status 401 - Unauthorized**: Invalid Access Token.
