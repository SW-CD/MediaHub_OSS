
### Database endpoints

#### POST /api/database

Creates a new database. This includes creating a new folder, adding an entry to the main `databases` table, and **creating a new, dedicated SQLite table** (e.g., `entries_MyDatabase`) with the standard and custom fields appropriate for the specified `content_type`.

**Role Required: `CanCreate`**

##### Request Body

```json
{
    "name": "MyAudioDatabase",
    "content_type": "audio",
    "housekeeping": {
        "interval": "1h",
        "disk_space": "100G",
        "max_age": "365d"
    },
    "config": {
        "create_preview": true,
        "auto_conversion": "flac"
    },
    "custom_fields": [
        { "name": "artist", "type": "TEXT" },
        { "name": "album", "type": "TEXT" }
    ]
}
```

  * **name** (string, required): The unique identifier.
  * **content\_type** (string, required): The type of content this database will store. Supported types: `image`, `audio`, `file`. This determines the dynamic table schema and allowed MIME types.
  * **housekeeping** (object, optional):
      * **interval** (string, optional): Default: "1h".
      * **disk\_space** (string, optional): Default: "100GB".
      * **max\_age** (string, optional): Default: "365 days".
  * **config** (object, optional): A JSON object for type-specific settings.
      * **For `content_type: 'image'`**:
          * `create_preview` (boolean, optional): Default: `true`.
          * `convert_to_jpeg` (boolean, optional): Default: `false`.
      * **For `content_type: 'audio'`**:
          * `create_preview` (boolean, optional): Default: `true`.
          * `auto_conversion` (string, optional): Default: `"none"`. Other values: `"flac"`, `"opus"`. **(Note: Setting this to anything other than "none" requires FFmpeg to be installed on the server.)**
      * **For `content_type: 'file'`**:
          * No options available. `config` should be `{}`.
  * **custom\_fields** (array, optional): An array of objects defining custom columns for this database's entry table.
      * **name** (string, required): The name of the custom field.
      * **type** (string, required): The SQLite data type. Supported types: `TEXT`, `INTEGER`, `REAL`, `BOOLEAN`.

##### Success Response

**Status 201 - Created**

```json
{
    "name": "MyAudioDatabase",
    "content_type": "audio",
    "housekeeping": {
        "interval": "1h",
        "disk_space": "100G",
        "max_age": "365d"
    },
    "config": {
        "create_preview": true,
        "auto_conversion": "flac"
    },
    "custom_fields": [
        { "name": "artist", "type": "TEXT" },
        { "name": "album", "type": "TEXT" }
    ]
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Invalid JSON, missing 'name' or 'content\_type', invalid `content_type`, invalid `config` options for the type, or invalid `custom_fields`.
  * **Status 400 - Bad Request**: The user attempted to enable an `auto_conversion` (e.g., "flac") but the server does not have **FFmpeg** installed.
    ```json
    {
        "error": "Cannot enable 'auto_conversion': FFmpeg is not available on the server."
    }
    ```
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanCreate` role.
  * **Status 409 - Conflict**: Database name already in use.

-----

#### DELETE /api/database

Deletes an entire database. This includes deleting its folder, all files on disk, its preview folder, its dedicated `entries_[name]` table from SQLite, and its entry from the `databases` table.
**Role Required: `CanDelete`**

##### Request

`DELETE /api/database?name=$name`

  * **name** (query param, required): The name of the database to delete.

##### Success Response

**Status 200 - OK**

```json
{
    "message": "Database 'MyAudioDatabase' and all its contents were successfully deleted."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanDelete` role.
  * **Status 404 - Not Found**: Database 'MyAudioDatabase' not found.

-----

#### PUT /api/database

Updates the housekeeping properties and type-specific `config` of an existing database. (Note: `custom_fields` and `content_type` cannot be modified after creation).
**Role Required: `CanEdit`**

##### Request

`PUT /api/database?name=$name`

  * **name** (query param, required): The name of the database to update.

##### Request Body

```json
{
    "config": {
        "create_preview": false,
        "auto_conversion": "none"
    },
    "housekeeping": {
        "interval": "2h",
        "disk_space": "150G"
    }
}
```

##### Success Response

**Status 200 - OK**

Returns the full, updated database object.

```json
{
    "name": "MyAudioDatabase",
    "content_type": "audio",
    "housekeeping": {
        "interval": "2h",
        "disk_space": "150G",
        "max_age": "365d"
    },
    "config": {
        "create_preview": false,
        "auto_conversion": "none"
    },
    "custom_fields": [
        { "name": "artist", "type": "TEXT" },
        { "name": "album", "type": "TEXT" }
    ]
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'name' param or invalid request body (e.g., invalid `config` for the type).
  * **Status 400 - Bad Request**: The user attempted to update `config` to enable an `auto_conversion` but the server does not have **FFmpeg** installed.
    ```json
    {
        "error": "Cannot enable 'auto_conversion': FFmpeg is not available on the server."
    }
    ```
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanEdit` role.
  * **Status 404 - Not Found**: Database 'MyAudioDatabase' not found.

-----

#### GET /api/database

Retrieves the properties and stats of a single database.
**Role Required: `CanView`**

##### Request

`GET /api/database?name=$name`

  * **name** (query param, required): The name of the database to retrieve.

##### Success Response

**Status 200 - OK**

Returns the database object, including its custom schema and live statistics.

```json
{
    "name": "MyAudioDatabase",
    "content_type": "audio",
    "housekeeping": {
        "interval": "1h",
        "disk_space": "100G",
        "max_age": "365d"
    },
    "config": {
        "create_preview": true,
        "auto_conversion": "flac"
    },
    "custom_fields": [
        { "name": "artist", "type": "TEXT" },
        { "name": "album", "type": "TEXT" }
    ],
    "stats": {
        "entry_count": 1520,
        "total_disk_space_bytes": 4501234567
    }
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.
  * **Status 404 - Not Found**: Database 'MyAudioDatabase' not found.

-----

#### GET /api/databases

Retrieves a list of all available databases and their stats.
**Role Required: `CanView`**

##### Request

`GET /api/databases`

##### Success Response

**Status 200 - OK**

Returns an array of all database objects.

```json
[
    {
        "name": "Sensor_Images",
        "content_type": "image",
        "housekeeping": {
            "interval": "1h",
            "disk_space": "100G",
            "max_age": "365d"
        },
        "config": {
            "create_preview": true,
            "convert_to_jpeg": false
        },
        "custom_fields": [
            { "name": "sensor_id", "type": "TEXT" }
        ],
        "stats": {
            "entry_count": 1520,
            "total_disk_space_bytes": 4501234567
        }
    },
    {
        "name": "Audio_Archive",
        "content_type": "audio",
        "housekeeping": {
            "interval": "24h",
            "disk_space": "500G",
            "max_age": "730d"
        },
        "config": {
            "create_preview": true,
            "auto_conversion": "flac"
        },
        "custom_fields": [
            { "name": "description", "type": "TEXT" }
        ],
        "stats": {
            "entry_count": 850,
            "total_disk_space_bytes": 1234567890
        }
    }
]
```

##### Error Responses

  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.

-----

#### GET /api/database/entries

Queries a database for a list of entry metadata. This endpoint provides basic filtering by timestamp and pagination. For complex, custom-field-based filtering, use the `POST /api/database/entries/search` endpoint.

**Role Required: `CanView`**

##### Request

`GET /api/database/entries?name=MyAudioDatabase&limit=10&order=desc&tstart=1750713600`

  * **name** (query param, required): The name of the database to query (determines which `entries_[name]` table to use).
  * **tstart, tend** (optional): ISO 8601 string or Unix epoch to filter by timestamp.
  * **limit, offset** (optional): Integers for pagination.
  * **order** (optional): "asc" or "desc" (default).

##### Success Response

**Status 200 - OK**

Returns an array of entry metadata objects, including any custom fields.

```json
[
    {
        "timestamp": 1750713653,
        "id": 10101,
        "filesize": 4500000,
        "mime_type": "audio/flac",
        "status": "ready",
        "duration_sec": 180.5,
        "artist": "Demo",
        "album": "Demo Album"
    }
]
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'name' param or invalid parameter formats (e.g., bad timestamp, non-integer limit).
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.
  * **Status 404 - Not Found**: Database 'MyAudioDatabase' not found.

-----

#### POST /api/database/entries/search

Retrieves a list of entry metadata matching the complex, nested filter criteria provided in the request body. This is the recommended endpoint for all custom-field-based queries.

**Role Required: `CanView`**

##### Request

`POST /api/database/entries/search?name=$name`

  * **name** (query param, required): The name of the database to search (determines which `entries_[name]` table to use).

##### Request Body

A JSON body defining the filter, sort, and pagination logic. The server will whitelist `field` names based on the database's `content_type` (e.g., `duration_sec`, `status`, `artist`).
**Example Logic:** `(duration_sec > 60.0 AND artist = "Demo")`
A `pagination.limit` parameter must be provided.

**JSON Request Body:**

```json
{
  "filter": {
    "operator": "and",
    "conditions": [
      { "field": "duration_sec", "operator": ">", "value": 60.0 },
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

To prevent SQL injection, the server **must** use parameterized queries and whitelist all `field` and `operator` strings provided in the request body.

##### Success Response

**Status 200 - OK**

The response body contains the list of matching results (even if the list is empty).

```json
[
    {
        "timestamp": 1750713653,
        "id": 10101,
        "filesize": 4500000,
        "mime_type": "audio/flac",
        "status": "ready",
        "duration_sec": 180.5,
        "artist": "Demo",
        "album": "Demo Album"
    }
]
```

##### Error Responses

  * **Status 400 - Bad Request**: The request was malformed. This is returned if:
      * Missing 'name' query parameter.
      * The JSON is invalid.
      * The user specifies a `field` that is not in the server's field whitelist for this `content_type`.
      * The user specifies an `operator` that is not in the server's operator whitelist.
      * The `pagiantion.limit` parameter is missing.
      * The data types are incorrect.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanView` role.
  * **Status 404 - Not Found**: Database 'MyAudioDatabase' not found.
  * **Status 500 - Internal Server Error**: The server encountered an unexpected error while trying to process the request.

-----

#### POST /api/database/housekeeping

Manually triggers the housekeeping task for a specific database.
**Role Required: `CanDelete`**

##### Request

`POST /api/database/housekeeping?name=$name`

  * **name** (query param, required): The name of the database to run housekeeping on.

##### Success Response

**Status 200 - OK**

Returns a report of actions taken.

```json
{
    "database_name": "MyAudioDatabase",
    "entries_deleted": 75,
    "space_freed_bytes": 210456789,
    "message": "Housekeeping complete. 75 entries deleted due to age or disk space limits."
}
```

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks `CanDelete` role.
  * **Status 404 - Not Found**: Database 'MyAudioDatabase' not found.

-----

#### GET /api/database/export

**New in v1.2**: streams the entire contents of a database (files and metadata) as a ZIP archive.
**Performance Note**: This endpoint uses `io.Pipe` to stream data directly from disk to the HTTP response. It does **not** load the files into RAM, allowing for multi-gigabyte exports on low-memory devices.

**Role Required: `CanView`** (Note: Might restrict to `IsAdmin` in future depending on policy, currently matches View access).

##### Request

`GET /api/database/export?name=$name`

  * **name** (query param, required): The name of the database to export.

##### Success Response

**Status 200 - OK**

  * **Content-Type:** `application/zip`
  * **Content-Disposition:** `attachment; filename="MyAudioDatabase_export.zip"`
  * **Body:** A binary ZIP stream containing:
    * Folder structure: `YYYY/MM/ID` (matching storage).
    * `_metadata.json`: A dump of the database configuration.
    * `entries.json`: A dump of all entry metadata.

##### Error Responses

  * **Status 400 - Bad Request**: Missing 'name' query parameter.
  * **Status 401 - Unauthorized**: Authentication failed.
  * **Status 403 - Forbidden**: User lacks permissions.
  * **Status 404 - Not Found**: Database not found.
  * **Status 500 - Internal Server Error**: ZIP streaming failed.