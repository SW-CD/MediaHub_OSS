---
sidebar_position: 10
title: Audit Logging Endpoint
---

# Audit Logging Endpoint

Endpoint for querying system audit trail records tracking administrative and user actions across the platform.

---

## `GET /api/audit`

Retrieves a paginated list of system audit log entries.

* **Role Required**: `IsAdmin` (Global)
* **Query Parameters**:
  * `limit` (integer, optional): Number of audit records to return.
  * `offset` (integer, optional): Pagination offset.
  * `order` (string, optional): Sort direction (`asc` or `desc`, default `desc`).
  * `tstart` (integer, optional): Start timestamp filter in milliseconds epoch.
  * `tend` (integer, optional): End timestamp filter in milliseconds epoch.

### Response (`200 OK`)

```json
[
  {
    "id": 1042,
    "timestamp": 1780713653123,
    "action": "delete_entry",
    "actor": "admin",
    "resource": "01HGFB9Z5W7ABCDEFGHJKMNPQR/entry/10232",
    "details": {
      "filesize_freed": 8945000,
      "filename": "my_song.wav"
    }
  },
  {
    "id": 1041,
    "timestamp": 1780713653123,
    "action": "user_login",
    "actor": "editor_bob",
    "resource": "system",
    "details": {
      "method": "local",
      "ip_address": "192.168.1.50"
    }
  }
]
```
