### Entry endpoints

#### POST /api/entry

Uploads a new file to a specific database using `multipart/form-data`. This allows for sending both JSON metadata (including custom fields) and the raw file binary in a single request.

The backend validates the file's MIME type and intelligently detects the upload size to determine the processing strategy:

  * **Small Files (In-Memory):** Are processed **synchronously**. The backend performs all conversion, preview generation, and metadata extraction *before* returning. This request will be slower, but the full entry metadata is returned upon completion with a `201 Created` status.
  * **Large Files (On-Disk):** Are processed **asynchronously**. The backend secures the file for background processing and returns a `202 Accepted` status *immediately*. The client must then poll the `GET /api/entry/meta` endpoint until the entry's `status` field changes from `"processing"` to `"ready"`.

**Role Required: `CanCreate`**

##### Request

`POST /api/entry?database_name=$name`

  * **database\_name** (query param, required): The name of the database.

##### Headers

  * `Content-Type` (required): `multipart/form-data; boundary=...`

##### Request Body

A `multipart/form-data` body with two parts:

1.  **`metadata` part**:
      * `Content-Type: application/json`
      * Body: A JSON string containing the entry metadata.
    <!-- end list -->
    ```json
    {
        "timestamp": 1780713653,
        "artist": "Demo",
        "album": "Demo Album"
    }
    ```
2.  **`file` part**:
      * `Content-Type`: Media type of the *uploaded* file (e.g., `audio/wav`, `image/jpeg`).
      * `Content-Disposition`: `form-data; name="file"; filename="my_song.wav"` (The `filename` is extracted and saved).
      * Body: The raw binary data of the file.

##### Success Response (Case 1: Synchronous)

**Status 201 - Created**

Returned for small, in-memory files. Returns the full metadata object for the newly created entry, including all generated fields and a `status` of `"ready"`.
This response indicated only that the file is uploaded and metadata was generated, it does not indicated that the preview image was generated already (in case the database config is set to generate previews).

```json
{
    "database_name": "MyAudioDatabase",
    "timestamp": 1780713653,
    "id": 10232,
    "filesize": 8945000,
    "mime_type": "audio/flac",
    "filename": "my_song.wav",
    "duration_sec": 150.2,
    "channels": 2,
    "status": "ready",
    "artist": "Demo",
    "album": "Demo Album"
}
```

##### Success Response (Case 2: Asynchronous)

**Status 202 - Accepted**

Returned for large, on-disk files. Returns a **partial** metadata object containing only the `id`, `timestamp`, `status`, and any user-provided metadata.

**The client is now responsible for polling the `GET /api/entry/meta?id={id}` endpoint** until the `status` field is `"ready"`.

```json
{
    "database_name": "MyAudioDatabase",
    "timestamp": 1780713653,
    "id": 10233,
    "status": "processing",
    "custom_fields": {
      "artist": "Demo",
      "album": "Demo Album"
    }
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'database\_name', `multipart/form-data` body, `metadata` part, or `file` part. Invalid JSON in `metadata`. Metadata fields do not match the database schema.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanCreate` role.
  * **Status 404 - Not Found**: 'database\_name' does not exist.
  * **Status 415 - Unsupported Media Type**: Invalid or unrecognized file format in the `file` part, or its MIME type is not in the server's allowlist for this database's `content_type`.
  * **Status 500 - Internal Server Error**: Failed to hand off the file to the background worker or create a temporary file.

-----

#### DELETE /api/entry

Deletes a single file from disk and its metadata from the corresponding database table. This also deletes the associated preview file, if one exists.
**Role Required: `CanDelete`**

##### Request

`DELETE /api/entry?database_name=$name&id=$id`

  * **database\_name** (query param, required): The name of the database the entry belongs to.
  * **id** (query param, required): The unique ID of the entry to delete.

##### Success Response

**Status 200 - OK**

```json
{
    "message": "Entry '10232' from database 'MyAudioDatabase' was successfully deleted."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' or 'database\_name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanDelete` role.
  * **Status 404 - Not Found**: Entry '10232' or database 'MyAudioDatabase' not found.

-----

#### GET /api/entry/file

Retrieves the raw **original** file.
**Role Required: `CanView`**

##### Request

`GET /api/entry/file?database_name=$name&id=$id`

  * **database\_name** (query param, required): The name of the database the entry belongs to.
  * **id** (query param, required): The unique ID of the entry to retrieve.

##### Success Response

**Status 200 - OK**

  * **Headers:**
      * `Content-Type`: The stored MIME type of the file (e.g., `audio/flac`, `image/jpeg`).
      * `Content-Length`: (filesize in bytes)
      * `Content-Disposition`: `attachment; filename="my_song.wav"` (Tells the browser to save the file with its original name. This header is omitted if the stored filename is empty).
  * **Body:** The raw binary data of the file.

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' or 'database\_name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.
  * **Status 404 - Not Found**: Entry '10232' or database 'MyAudioDatabase' not found.

-----

#### GET /api/entry/preview

Retrieves the generated preview for an entry (e.g., image thumbnail, audio waveform). This is the recommended endpoint for displaying entries in a gallery or list view.
**Role Required: `CanView`**

##### Request

`GET /api/entry/preview?database_name=$name&id=$id`

  * **database\_name** (query param, required): The name of the database the entry belongs to.
  * **id** (query param, required): The unique ID of the entry to retrieve the preview for.

##### Success Response

**Status 200 - OK**

  * **Headers:**
      * `Content-Type`: `image/jpeg`
      * `Content-Length`: (filesize in bytes)
  * **Body:** The raw binary data of the preview (thumbnail or waveform).

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' or 'database\_name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.
  * **Status 404 - Not Found**: Entry '10232', database 'MyAudioDatabase', or the preview file for this entry was not found (or the database type 'file' does not support previews).

-----

#### GET /api/entry/meta

Retrieves all metadata for a single entry, including custom fields. This is the efficient way to get entry details without downloading the file binary.
**Role Required: `CanView`**

##### Request

`GET /api/entry/meta?database_name=$name&id=$id`

  * **database\_name** (query param, required): The name of the database the entry belongs to.
  * **id** (query param, required): The unique ID of the entry to retrieve metadata for.

##### Success Response

**Status 200 - OK**

Returns the full entry metadata object.

```json
{
    "database_name": "MyAudioDatabase",
    "timestamp": 1780713653,
    "id": 10232,
    "filesize": 8945000,
    "mime_type": "audio/flac",
    "filename": "my_song.wav",
    "status": "ready",
    "duration_sec": 150.2,
    "channels": 2,
    "artist": "Demo",
    "album": "Demo Album"
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' or 'database\_name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.
  * **Status 404 - Not Found**: Entry '10232' or database 'MyAudioDatabase' not found.

-----

#### PATCH /api/entry

Updates the mutable metadata of an existing entry (e.g., `timestamp`, `filename`, and any custom fields). Read-only, generated fields like `filesize`, `mime_type`, `width`, `height`, and `status` cannot be modified.
**Role Required: `CanEdit`**

##### Request

`PATCH /api/entry?database_name=$name&id=$id`

  * **database\_name** (query param, required): The name of the database the entry belongs to.
  * **id** (query param, required): The ID of the entry to update.

##### Request Body

A JSON object containing only the fields to be updated. (Note: `filename` can also be updated).

```json
{
    "timestamp": 1790000000,
    "artist": "A new artist",
    "filename": "a_new_name.flac"
}
```

##### Success Response

**Status 200 - OK**

Returns the full, updated entry metadata object.

```json
{
    "database_name": "MyAudioDatabase",
    "timestamp": 1790000000,
    "id": 10232,
    "filesize": 8945000,
    "mime_type": "audio/flac",
    "filename": "a_new_name.flac",
    "status": "ready",
    "duration_sec": 150.2,
    "channels": 2,
    "artist": "A new artist",
    "album": "Demo Album"
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'id' or 'database\_name' param, invalid JSON, or invalid field values (e.g., non-existent custom field).
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanEdit` role.
  * **Status 404 - Not Found**: Entry '10232' or database 'MyAudioDatabase' not found.
