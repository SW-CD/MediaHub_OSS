-- +goose Up
CREATE TABLE IF NOT EXISTS databases (
    id TEXT(26) PRIMARY KEY NOT NULL, -- ULID
    name TEXT(32) UNIQUE NOT NULL CHECK(length(name) <= 32),
    content_type TEXT NOT NULL DEFAULT 'image',
    hk_interval INTEGER NOT NULL DEFAULT 3600000, -- 1 hour in milliseconds
    hk_disk_space INTEGER NOT NULL DEFAULT 107374182400, -- 100GB in bytes
    hk_max_age INTEGER NOT NULL DEFAULT 31536000000, -- 365 days in milliseconds
    
    -- Extracted config fields
    create_preview BOOLEAN NOT NULL DEFAULT 0,
    auto_conversion TEXT NOT NULL DEFAULT 'none',
    
    custom_fields TEXT NOT NULL DEFAULT '[]',
    hk_last_run INTEGER NOT NULL DEFAULT 0, -- UNIX epoch in milliseconds
    
    -- Denormalized stats
    entry_count INTEGER NOT NULL DEFAULT 0,
    total_disk_space_bytes INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
    password_hash TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS database_permissions (
    user_id INTEGER NOT NULL,
    database_id TEXT(26) NOT NULL,
    can_view BOOLEAN NOT NULL DEFAULT 0,
    can_create BOOLEAN NOT NULL DEFAULT 0,
    can_edit BOOLEAN NOT NULL DEFAULT 0,
    can_delete BOOLEAN NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, database_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash TEXT UNIQUE NOT NULL, 
    expiry INTEGER NOT NULL, -- UNIX epoch in milliseconds
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER NOT NULL DEFAULT (CAST(unixepoch('subsec') * 1000 AS INTEGER)), -- UNIX epoch in milliseconds
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    resource TEXT NOT NULL,
    details TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS database_permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS databases;
DROP TABLE IF EXISTS audit_logs;