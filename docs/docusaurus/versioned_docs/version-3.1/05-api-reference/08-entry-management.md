---
sidebar_position: 8
title: Single Entry Management
---

# Single Entry Management Endpoints

Endpoints for uploading, inspecting metadata, updating, downloading raw files or previews, and deleting individual media entries.

---

## `POST /api/database/{database_id}/entry`

Uploads a single media file into a database using `multipart/form-data`.

* **Role Required**: `can_create` on database
* **Path Parameters**:
  * `database_id` (string, required): ULID of the target database.
* **Headers**:
  * `Content-Type`: `multipart/form-data; boundary=...`
* **Form Data Parts**:
  1. `metadata` (text field): Serialized JSON string containing entry timestamp, original filename, and custom field values.
     ```json
     {
       "timestamp": 1780713653123,
       "filename": "my_song.wav",
       "custom_fields": {
         "artist": "Demo",
         "album": "Demo Album"
       }
     }
     ```
  2. `file` (binary file part): Raw binary content of the file.

### Response Case 1: Synchronous Processing (`201 Created`)
Returned for small, synchronous uploads.
```json
{
  "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
  "id": 10232,
  "timestamp": 1780713653123,
  "created_at": 1780713655000,
  "updated_at": 1780713655000,
  "filesize": 8945000,
  "preview_filesize": 800,
  "mime_type": "audio/flac",
  "filename": "my_song.wav",
  "status": "ready",
  "media_fields": {
    "duration": 150.2,
    "channels": 2
  },
  "custom_fields": {
    "artist": "Demo",
    "album": "Demo Album"
  }
}
```

### Response Case 2: Asynchronous Processing (`202 Accepted`)
Returned for large files or queued jobs. The client should poll `GET /api/database/{database_id}/entry/{id}` until `status` becomes `"ready"`.

---

## `GET /api/database/{database_id}/entry/{id}`

Retrieves all metadata for a single entry.

* **Role Required**: `can_view` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
  * `id` (integer, required): Unique entry ID.
* **Response (`200 OK`)**: Returns entry metadata object.

---

## `PATCH /api/database/{database_id}/entry/{id}`

Updates mutable metadata (`timestamp`, `filename`, `custom_fields`) of an existing entry.

* **Role Required**: `can_edit` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
  * `id` (integer, required): Unique entry ID.
* **Request Body**:
  ```json
  {
    "timestamp": 1790000000123,
    "filename": "a_new_name.flac",
    "custom_fields": {
      "artist": "A new artist"
    }
  }
  ```
* **Response (`200 OK`)**: Returns updated entry metadata object.

---

## `GET /api/database/{database_id}/entry/{id}/file`

Retrieves the raw file binary. Supports **HTTP Range Requests** (streaming/seeking) and **Content Negotiation** (binary vs JSON Base64).

* **Role Required**: `can_view` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
  * `id` (integer, required): Unique entry ID.
* **Headers (Optional)**:
  * `Accept`: `*/*` (default binary download) or `application/json` (Base64 data wrapper).
  * `Range`: `bytes=0-1023` (Triggers `206 Partial Content` streaming response when `Accept` is binary).

### Binary Response (`200 OK` / `206 Partial Content`)
Returns binary stream with standard headers (`Content-Type`, `Content-Length`, `Content-Range`, `Content-Disposition`).

### JSON Base64 Response (`200 OK` when `Accept: application/json`)
```json
{
  "filename": "my_song.wav",
  "mime_type": "audio/wav",
  "size": 8945000,
  "data": "data:audio/wav;base64,UklGRi..."
}
```

---

## `GET /api/database/{database_id}/entry/{id}/preview`

Retrieves the generated preview thumbnail for an entry (e.g. `image/webp`). Supports **Content Negotiation**.

* **Role Required**: `can_view` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
  * `id` (integer, required): Unique entry ID.
* **Headers (Optional)**:
  * `Accept`: `*/*` (default binary image) or `application/json` (Base64 wrapper).

### Response (`200 OK`)
Returns binary image payload (with `Cache-Control: no-cache`) or JSON Base64 string payload.

---

## `DELETE /api/database/{database_id}/entry/{id}`

Deletes a media entry, removing its stored file, preview, and metadata from database.

* **Role Required**: `can_delete` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
  * `id` (integer, required): Unique entry ID.
* **Response (`200 OK`)**:
  ```json
  {
    "message": "Entry '10232' from database '01HGFB9Z5W7ABCDEFGHJKMNPQR' was successfully deleted."
  }
  ```
