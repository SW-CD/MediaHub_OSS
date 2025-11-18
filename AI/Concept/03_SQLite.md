## SQLite tables

Here are the `CREATE TABLE` commands for the SQLite database that would power this application.

```sql
-- Stores the database configurations, custom schemas, and type-specific settings
CREATE TABLE IF NOT EXISTS databases (
    name TEXT(255) PRIMARY KEY NOT NULL CHECK(length(name) <= 255),
    
    -- Type of content stored. Determines the schema of the dynamic table.
    content_type TEXT NOT NULL CHECK(content_type IN (
        'image', 
        'audio',
        'file'
    )),
    
    hk_interval TEXT NOT NULL DEFAULT '1h',
    hk_disk_space TEXT NOT NULL DEFAULT '100G',
    hk_max_age TEXT NOT NULL DEFAULT '365d',
    
    -- Stores the JSON object for type-specific config 
    -- (e.g., {"create_preview": true, "auto_conversion": "flac"})
    config TEXT NOT NULL DEFAULT '{}',
    
    -- Stores the JSON array of custom field definitions
    custom_fields TEXT NOT NULL DEFAULT '[]' 
);

-- NOTE: A new 'entries' table is dynamically created for *each* database,
-- based on its 'content_type'.
--
-- Example 1: POST /api/database with name: "MyImageDB", content_type: 'image'
/*
CREATE TABLE IF NOT EXISTS "entries_MyImageDB" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    filename TEXT NOT NULL DEFAULT '',
    
    -- NEW: Tracks async processing
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN (
        'processing', 
        'ready', 
        'error'
    )),
    
    mime_type TEXT NOT NULL CHECK(mime_type IN (
        'image/jpeg', 
        'image/png', 
        'image/gif', 
        'image/webp'
    )),
    
    -- Image-specific fields (will be 0 until 'ready')
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,

    -- Custom columns are appended here
);
*/

-- Example 2: POST /api/database with name: "MyAudioDB", content_type: 'audio'
/*
CREATE TABLE IF NOT EXISTS "entries_MyAudioDB" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    filename TEXT NOT NULL DEFAULT '',

    -- NEW: Tracks async processing
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN (
        'processing', 
        'ready', 
        'error'
    )),
    
    mime_type TEXT NOT NULL CHECK(mime_type IN (
        'audio/mpeg', 
        'audio/wav', 
        'audio/flac',
        'audio/opus',
        'audio/ogg'
    )),
    
    -- Audio-specific fields (will be 0 until 'ready')
    duration_sec REAL NOT NULL,
    channels INTEGER,

    -- Custom columns are appended here
);
*/

-- Example 3: POST /api/database with name: "MyFileDB", content_type: 'file'
/*
CREATE TABLE IF NOT EXISTS "entries_MyFileDB" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL,
    filesize INTEGER NOT NULL,
    filename TEXT NOT NULL DEFAULT '',

    -- NEW: Tracks async processing
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN (
        'processing', 
        'ready', 
        'error'
    )),
    
    -- No CHECK constraint, allows any type
    mime_type TEXT NOT NULL, 

    -- No media-specific fields

    -- Custom columns are appended here
);
*/

-- An index is also dynamically created for fast time queries on all tables
/*
CREATE INDEX IF NOT EXISTS "idx_entries_MyAudioDB_time"
    ON "entries_MyAudioDB"(timestamp);
*/

-- NEW: An index should also be created for the 'status' column
/*
CREATE INDEX IF NOT EXISTS "idx_entries_MyAudioDB_status"
    ON "entries_MyAudioDB"(status);
*/


-- Stores user accounts and their roles in a single, flat table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
    -- Stores the securely hashed password (e.g., bcrypt)
    -- The salt is included in the hash string.
    password_hash TEXT NOT NULL,
    -- Boolean flags (0 or 1) for each role
    can_view BOOLEAN NOT NULL DEFAULT 0,
    can_create BOOLEAN NOT NULL DEFAULT 0,
    can_edit BOOLEAN NOT NULL DEFAULT 0,
    can_delete BOOLEAN NOT NULL DEFAULT 0,
    is_admin BOOLEAN NOT NULL DEFAULT 0
);

```
