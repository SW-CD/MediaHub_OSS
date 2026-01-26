# SQLite Schema & Migrations

The database schema is managed via **Migrations** (using `pressly/goose`). The application embeds `.sql` migration files and applies them via the `migrate` CLI command. If a mismatching schema version is found on a different command, an error is triggered with a request to use the `migrate` command first, or downgrade to the correct version of the application. Both SQLite and PostreSQL are supported in the future.

**Important:** Do not manually modify the schema in the codebase. Create a new versioned migration file (e.g., `002_add_permissions.sql`) to change the structure.

## 1. Static Tables (Managed by Migrations)

These tables exist once in the global `mediahub.db` file.

```sql
-- Tracks migration history (Managed by Goose)
CREATE TABLE IF NOT EXISTS goose_db_version (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version_id INTEGER NOT NULL,
    is_applied BOOLEAN NOT NULL,
    tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stores the database configurations
CREATE TABLE IF NOT EXISTS databases (
    name TEXT(32) PRIMARY KEY NOT NULL CHECK(length(name) <= 32),
    
    -- Type of content stored. Determines the schema of the dynamic table.
    content_type TEXT NOT NULL DEFAULT 'image',
    
    hk_interval TEXT NOT NULL DEFAULT '1h',
    hk_disk_space TEXT NOT NULL DEFAULT '100G',
    hk_max_age TEXT NOT NULL DEFAULT '365d',
    
    -- Stores the configuration for auto-conversion 
    create_preview BOOLEAN NOT NULL DEFAULT 0,
    auto_conversion TEXT NOT NULL DEFAULT 'none',
    
    -- Stores the JSON array of custom field definitions
    custom_fields TEXT NOT NULL DEFAULT '[]',
    
    last_hk_run TIMESTAMP NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    
    -- Denormalized Stats (New in v1.2 for performance)
    entry_count INTEGER NOT NULL DEFAULT 0,
    total_disk_space_bytes INTEGER NOT NULL DEFAULT 0
);

-- Users Table, TODO update regarding per database user roles
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
    password_hash TEXT NOT NULL,
    -- Roles
    can_view BOOLEAN NOT NULL DEFAULT 0,
    can_create BOOLEAN NOT NULL DEFAULT 0,
    can_edit BOOLEAN NOT NULL DEFAULT 0,
    can_delete BOOLEAN NOT NULL DEFAULT 0,
    is_admin BOOLEAN NOT NULL DEFAULT 0
);

-- Refresh Tokens (Stateful Auth)
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash TEXT UNIQUE NOT NULL, 
    expiry TIMESTAMP NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
```

## 2\. Dynamic Entry Tables (Managed by Repository Code)

A new `entries` table is dynamically created for *each* database when `POST /api/database` is called. The schema depends on the `content_type`.

**Note:** Since v1.3, we use a Query Builder (Squirrel) to interact with these tables to ensure compatibility with future Postgres drivers, but the SQLite structure remains as follows.

### Common Fields

All entry tables share these core columns:

  * `id`: Unique identifier.
  * `timestamp`: Unix timestamp of the entry.
  * `filesize`: Size in bytes.
  * `filename`: Original filename.
  * `status`: **(New in v1.2)** Tracks async processing (`processing`, `ready`, `error`).

-----

### Type: `image`

**Example Table:** `entries_MyImageDB`

```sql
CREATE TABLE IF NOT EXISTS "entries_MyImageDB" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    filename TEXT NOT NULL DEFAULT '',
    
    -- Async Processing Status
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN (
        'processing', 
        'ready', 
        'error'
    )),
    
    -- Image Specifics
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    
    mime_type TEXT NOT NULL CHECK(mime_type IN (
        'image/jpeg', 
        'image/png', 
        'image/gif', 
        'image/webp'
    ))

    -- Custom columns are appended here (e.g., "score" REAL)
);
```

-----

### Type: `audio`

**Example Table:** `entries_MyAudioDB`

```sql
CREATE TABLE IF NOT EXISTS "entries_MyAudioDB" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    filename TEXT NOT NULL DEFAULT '',

    -- Async Processing Status
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN (
        'processing', 
        'ready', 
        'error'
    )),
    
    -- Audio Specifics
    duration_sec REAL NOT NULL,
    channels INTEGER,
    
    mime_type TEXT NOT NULL CHECK(mime_type IN (
        'audio/mpeg', 
        'audio/wav', 
        'audio/flac', 
        'audio/x-flac',
        'audio/opus', 
        'audio/ogg',
        'application/ogg'
    ))

    -- Custom columns are appended here
);
```

-----

### Type: `file`

**Example Table:** `entries_MyFileDB`

```sql
CREATE TABLE IF NOT EXISTS "entries_MyFileDB" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    filename TEXT NOT NULL DEFAULT '',

    -- Async Processing Status
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN (
        'processing', 
        'ready', 
        'error'
    )),
    
    -- Generic File (No Check Constraint)
    mime_type TEXT NOT NULL 

    -- Custom columns are appended here
);
```

-----

### Indexes

Indexes are dynamically created to ensure fast filtering and sorting.

1.  **Time Index (Standard):**

    ```sql
    CREATE INDEX IF NOT EXISTS "idx_entries_MyAudioDB_time"
        ON "entries_MyAudioDB"(timestamp);
    ```

2.  **Status Index (Standard):**
    Used by the recovery tool to find stuck "processing" entries.

    ```sql
    CREATE INDEX IF NOT EXISTS "idx_entries_MyAudioDB_status"
        ON "entries_MyAudioDB"(status);
    ```

3.  **Custom Field Indexes:**
    Created for every custom field defined in the database config.

    ```sql
    CREATE INDEX IF NOT EXISTS "idx_entries_MyAudioDB_artist"
        ON "entries_MyAudioDB"("artist");
    ```
