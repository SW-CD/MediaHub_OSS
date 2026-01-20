-- filepath: internal/db/migrations/001_init_v1_1.sql
-- +goose Up
CREATE TABLE IF NOT EXISTS databases (
    name TEXT(32) PRIMARY KEY NOT NULL CHECK(length(name) <= 32),
    content_type TEXT NOT NULL DEFAULT 'image',
    hk_interval TEXT NOT NULL DEFAULT '1h',
    hk_disk_space TEXT NOT NULL DEFAULT '100G',
    hk_max_age TEXT NOT NULL DEFAULT '365d',
    config TEXT NOT NULL DEFAULT '{}',
    custom_fields TEXT NOT NULL DEFAULT '[]',
    last_hk_run TIMESTAMP NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    
    -- Denormalized stats (Existing in v1.1)
    entry_count INTEGER NOT NULL DEFAULT 0,
    total_disk_space_bytes INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
    password_hash TEXT NOT NULL,
    can_view BOOLEAN NOT NULL DEFAULT 0,
    can_create BOOLEAN NOT NULL DEFAULT 0,
    can_edit BOOLEAN NOT NULL DEFAULT 0,
    can_delete BOOLEAN NOT NULL DEFAULT 0,
    is_admin BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash TEXT UNIQUE NOT NULL, 
    expiry TIMESTAMP NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS databases;