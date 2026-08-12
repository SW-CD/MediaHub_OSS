---
sidebar_position: 9
title: Entries Querying & Bulk Operations
---

# Entries Querying & Bulk Operations

Endpoints for querying, structured searching, exporting, importing, and bulk deleting media entries.

---

## `GET /api/database/{database_id}/entries`

Queries a database for a paginated list of entry metadata with time-range filtering and sorting.

* **Role Required**: `can_view` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
* **Query Parameters**:
  * `limit` (integer, optional): Pagination limit.
  * `offset` (integer, optional): Pagination offset.
  * `order` (string, optional): Sort direction (`asc` or `desc`, default `desc`).
  * `sort_by` (string, optional): Field to sort by (`timestamp`, `created_at`, `updated_at`, default `timestamp`).
  * `time_field` (string, optional): Field to filter with `tstart`/`tend` (`timestamp`, `created_at`, `updated_at`, default `timestamp`).
  * `tstart` (integer/string, optional): Start timestamp in milliseconds or ISO8601 string.
  * `tend` (integer/string, optional): End timestamp in milliseconds or ISO8601 string.
* **Response (`200 OK`)**:
  ```json
  [
    {
      "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
      "id": 10232,
      "timestamp": 1790000000000,
      "created_at": 1790000500000,
      "updated_at": 1790001000000,
      "filesize": 8945000,
      "mime_type": "audio/flac",
      "filename": "a_new_name.flac",
      "status": "ready",
      "media_fields": {
        "duration": 150.2,
        "channels": 2
      },
      "custom_fields": {
        "artist": "A new artist",
        "album": "Demo Album"
      }
    }
  ]
  ```

---

## `POST /api/database/{database_id}/entries/search`

Executes structured, multi-condition search queries across native and custom metadata fields.

* **Role Required**: `can_view` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
* **Request Body**:
  ```json
  {
    "filter": {
      "operator": "and",
      "conditions": [
        { "field": "duration", "operator": ">", "value": 60.0 },
        { "field": "artist", "operator": "=", "value": "Demo" }
      ]
    },
    "sort": {
      "field": "timestamp",
      "direction": "desc"
    },
    "pagination": {
      "offset": 0,
      "limit": 30
    }
  }
  ```
* **Response (`200 OK`)**: Returns array of matching entry metadata objects.

---

## `POST /api/database/{database_id}/entries/export`

Streams a ZIP archive containing media files and an `entries.csv` metadata sheet for a specified set of entry IDs.

* **Role Required**: `can_view` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
* **Request Body**:
  ```json
  {
    "ids": [10232, 10233, 10234, 10500]
  }
  ```
* **Response (`200 OK`)**: Binary stream with `Content-Type: application/zip` and `Content-Disposition: attachment; filename="{database_id}_export.zip"`.

---

## `POST /api/database/{database_id}/entries/import`

Bulk-imports media files and metadata from an uploaded ZIP archive containing `entries.csv`.

* **Role Required**: `can_create` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
* **Form Data Parts**:
  * `file` (required): Uploaded ZIP archive.
  * `config` (optional JSON string): Import options (`mode`: `"generate_new"` or `"skip"`, `custom_field_mapping`, `unmapped_fields`: `"ignore"` or `"fail"`).
* **Response (`202 Accepted`)**:
  ```json
  {
    "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
    "message": "Import job started successfully. The archive is being processed in the background."
  }
  ```

---

## `POST /api/database/{database_id}/entries/delete`

Deletes multiple entries in a single atomic database operation.

* **Role Required**: `can_delete` on database
* **Path Parameters**:
  * `database_id` (string, required): Database ULID.
* **Request Body**:
  ```json
  {
    "ids": [10232, 10233, 10234, 10500]
  }
  ```
* **Response (`200 OK`)**:
  ```json
  {
    "database_id": "01HGFB9Z5W7ABCDEFGHJKMNPQR",
    "deleted_count": 4,
    "space_freed_bytes": 15000000,
    "message": "Successfully deleted 4 entries.",
    "errors": ""
  }
  ```
