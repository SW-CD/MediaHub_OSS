---
sidebar_position: 7
title: Custom Fields Endpoints
---

# Custom Fields Endpoints

Endpoints for defining and modifying custom metadata field schemas on a database. Custom fields support data types `TEXT`, `INTEGER`, `REAL`, and `BOOLEAN`. Each database supports up to 255 custom fields identified by integer IDs (`0` to `254`).

---

## `GET /api/database/{id}/fields`

Retrieves a list of all custom fields defined for a database.

* **Role Required**: Database permission (`can_view`, `can_create`, `can_edit`, `can_delete`, `can_admin`) or global `IsAdmin`
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
* **Response (`200 OK`)**:
  ```json
  [
    { "id": 0, "name": "artist", "type": "TEXT", "is_indexed": true },
    { "id": 1, "name": "album", "type": "TEXT", "is_indexed": true },
    { "id": 2, "name": "song_name", "type": "TEXT", "is_indexed": false }
  ]
  ```

---

## `POST /api/database/{id}/field`

Adds a new custom field to an existing database schema.

* **Role Required**: Global `IsAdmin` OR `can_admin` on database
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
* **Request Body**:
  ```json
  {
    "name": "genre",
    "type": "TEXT",
    "is_indexed": false
  }
  ```
  *(Note: `is_indexed` defaults to `true` if omitted).*
* **Response (`201 Created`)**:
  ```json
  {
    "id": 2,
    "name": "genre",
    "type": "TEXT",
    "is_indexed": false
  }
  ```

---

## `PATCH /api/database/{id}/field/{field_id}`

Updates the name or indexing status of an existing custom field.

* **Role Required**: Global `IsAdmin` OR `can_admin` on database
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
  * `field_id` (integer, required): The numeric ID of the custom field (0 to 254).
* **Request Body**:
  ```json
  {
    "name": "primary_genre",
    "is_indexed": true
  }
  ```
* **Response (`200 OK`)**: Returns updated field schema object.

---

## `DELETE /api/database/{id}/field/{field_id}`

Permanently drops a custom field and its stored data across all entries in the database.

* **Role Required**: Global `IsAdmin` OR `can_admin` on database
* **Path Parameters**:
  * `id` (string, required): ULID of the database.
  * `field_id` (integer, required): The numeric ID of the custom field.
* **Response (`200 OK`)**:
  ```json
  {
    "message": "Field 'primary_genre' (ID: 2) was successfully deleted from database 'MyAudioDatabase'."
  }
  ```
