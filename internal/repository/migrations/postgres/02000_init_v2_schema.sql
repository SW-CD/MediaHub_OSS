-- +goose Up
CREATE TABLE IF NOT EXISTS databases (
    name VARCHAR(32) PRIMARY KEY NOT NULL CHECK(length(name) <= 32),
    content_type VARCHAR(32) NOT NULL DEFAULT 'image',
    hk_interval INTEGER NOT NULL DEFAULT 3600000, -- 1 hour in milliseconds
    hk_disk_space INTEGER NOT NULL DEFAULT 107374182400, -- 100GB in bytes
    hk_max_age INTEGER NOT NULL DEFAULT 3153600000, -- 365 days in milliseconds
    
    -- Extracted config fields
    create_preview BOOLEAN NOT NULL DEFAULT FALSE,
    auto_conversion VARCHAR(32) NOT NULL DEFAULT 'none',
    
    custom_fields TEXT NOT NULL DEFAULT '[]',
    hk_last_run INTEGER NOT NULL DEFAULT 0,
    
    -- Denormalized stats
    entry_count BIGINT NOT NULL DEFAULT 0,
    total_disk_space_bytes BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
    password_hash TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS database_permissions (
    user_id INTEGER NOT NULL,
    database_name VARCHAR(32) NOT NULL,
    can_view BOOLEAN NOT NULL DEFAULT FALSE,
    can_create BOOLEAN NOT NULL DEFAULT FALSE,
    can_edit BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, database_name),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (database_name) REFERENCES databases(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token_hash TEXT UNIQUE NOT NULL, 
    expiry INTEGER NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp BIGINT NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000 AS BIGINT),
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    resource TEXT NOT NULL,
    details TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);

-- System locks table for distributed locking (PostgreSQL only)
CREATE TABLE IF NOT EXISTS system_locks (
    lock_name VARCHAR(64) PRIMARY KEY,
    locked_at INTEGER NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000 AS INTEGER), -- TODO verify if this is the best way to store timestamps in Postgres for our use case
    locked_by VARCHAR(128) NOT NULL,
    expires_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS system_locks;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS database_permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS databases;
DROP TABLE IF EXISTS audit_logs;