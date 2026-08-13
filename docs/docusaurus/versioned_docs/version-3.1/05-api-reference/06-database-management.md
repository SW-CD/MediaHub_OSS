---
sidebar_position: 6
title: Database Management Endpoints
---

# Database Management Endpoints

Endpoints for creating, listing, inspecting, updating, deleting databases, and manually triggering housekeeping tasks.

---

## `GET /api/databases`

Retrieves a list of all databases accessible to the current user (filtered by user permissions unless the user is global `IsAdmin`).

* **Role Required**: Authenticated User
* **Response (`200 OK`)**:
  ```json
  [
    {
      "id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
      "name": "Audio_Archive",
      "content_type": "audio",
      "n_max_queued": 0,
      "housekeeping": {
        "interval": "24h",
        "disk_space": "500G",
        "max_age": "730d"
      },
      "config": {
        "create_preview": true,
        "auto_conversion": "audio/flac"
      },
      "custom_fields": [
        { "id": 0, "name": "description", "type": "TEXT", "is_indexed": false }
      ],
      "stats": {
        "entry_count": 850,
        "total_disk_space_bytes": 1234567890
      }
    }
  ]
  ```

---

## `POST /api/database`

Creates a new database along with its dedicated entries storage table.

* **Role Required**: `IsAdmin` (Global)
* **Request Body**:
  ```json
  {
    "name": "MyAudioDatabase",
    "content_type": "audio",
    "n_max_queued": 0,
    "housekeeping": {
      "interval": "1h",
      "disk_space": "100G",
      "max_age": "0d"
    },
    "config": {
      "create_preview": true,
      "auto_conversion": "audio/flac"
    },
    "custom_fields": [
      { "name": "artist", "type": "TEXT", "is_indexed": true },
      { "name": "album", "type": "TEXT", "is_indexed": true },
      { "name": "song_name", "type": "TEXT", "is_indexed": false }
    ]
  }
  ```
  *(Note: Supported `content_type` values are `image`, `audio`, `video`, and `file`).*
* **Response (`201 Created`)**:
  ```json
  {
    "id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
    "name": "MyAudioDatabase",
    "content_type": "audio",
    "n_max_queued": 0,
    "housekeeping": { ... },
    "config": { ... },
    "custom_fields": [ ... ]
  }
  ```

---

## `GET /api/database/{id}`

Retrieves details, custom field schema, and live stats for a specific database.

* **Role Required**: Database permission (`can_view`, `can_create`, `can_edit`, `can_delete`, `can_admin`) or global `IsAdmin`
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
* **Response (`200 OK`)**: Returns database object including live storage stats.

---

## `PUT /api/database/{id}`

Updates settings for an existing database (`name`, `n_max_queued`, `housekeeping`, `config`). `content_type` cannot be changed after creation.

* **Role Required**: Global `IsAdmin` OR `can_admin` on database
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
* **Request Body**:
  ```json
  {
    "name": "MyUpdatedAudioDatabase",
    "n_max_queued": 0,
    "config": {
      "create_preview": false,
      "auto_conversion": ""
    },
    "housekeeping": {
      "interval": "2h",
      "disk_space": "150G",
      "max_age": "0d"
    }
  }
  ```
* **Response (`200 OK`)**: Returns updated database object.

---

## `DELETE /api/database/{id}`

Deletes a database, removing its storage folder, previews, dedicated entries table, and configuration.

* **Role Required**: `IsAdmin` (Global)
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
* **Response (`200 OK`)**:
  ```json
  {
    "message": "Database 'MyAudioDatabase' (ID: 01HGFB9Z5W7ABCDEFGHJKMNPQR) and all its contents were successfully deleted."
  }
  ```

---

## `POST /api/database/{id}/housekeeping`

Manually triggers housekeeping execution for a specific database.

* **Role Required**: `can_delete` on database or global `IsAdmin`
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
* **Response (`200 OK`)**:
  ```json
  {
    "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
    "database_name": "MyUpdatedAudioDatabase",
    "entries_deleted": 75,
    "space_freed_bytes": 210456789,
    "message": "Housekeeping complete. 75 entries deleted due to age or disk space limits."
  }
  ```
