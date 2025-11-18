## API description

### Authentication

All API endpoints (unless specified) are protected using role-based access control (RBAC) and are available under the `/api` path. The server will authenticate the user (e.g., via Basic Authentication or a Bearer Token) and check their permissions.

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
