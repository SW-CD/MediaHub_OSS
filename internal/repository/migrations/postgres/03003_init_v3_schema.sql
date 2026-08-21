-- Migration: Initialize V3 Postgres Schema
-- Description: Creates all core database tables for PostgreSQL at schema version 3.3 (3003), including:
--   - databases: Virtual database configurations and statistics (including n_max_queued)
--   - database_custom_fields: Custom metadata field definitions (uses SMALLINT for field_id)
--   - users: User accounts (ULID IDs, admin and service account flags)
--   - database_permissions: Database-specific permissions (including can_admin)
--   - refresh_tokens: Stateful JWT refresh tokens
--   - api_keys: User and service account API keys with scopes
--   - audit_logs: System audit logging
--   - system_locks: Database-backed distributed locks for Kubernetes multi-node deployment
--
-- Rationale:
--   CHECK constraints are used instead of PostgreSQL ENUM types for metadata tables.
--   This ensures Goose database migrations remain 100% transactional and idempotent
--   without requiring non-transactional ALTER TYPE commands or custom driver type casting.
--
-- +goose Up
CREATE TABLE IF NOT EXISTS databases (
    id VARCHAR(26) PRIMARY KEY NOT NULL, -- ULID
    name VARCHAR(64) UNIQUE NOT NULL CHECK(length(name) > 0 AND length(name) <= 64),
    content_type VARCHAR(32) NOT NULL DEFAULT 'image',
    hk_interval BIGINT NOT NULL DEFAULT 3600000, -- 1 hour in milliseconds
    hk_disk_space BIGINT NOT NULL DEFAULT 107374182400, -- 100GB in bytes
    hk_max_age BIGINT NOT NULL DEFAULT 0, -- 0 = disabled, value is in milliseconds if provided
    
    create_preview BOOLEAN NOT NULL DEFAULT FALSE,
    auto_conversion VARCHAR(32) NOT NULL DEFAULT 'none',
    n_max_queued INTEGER NOT NULL DEFAULT 0,
    
    hk_last_run BIGINT NOT NULL DEFAULT 0,
    
    entry_count BIGINT NOT NULL DEFAULT 0,
    total_disk_space_bytes BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS database_custom_fields (
    database_id VARCHAR(26) NOT NULL,
    field_id SMALLINT NOT NULL CHECK(field_id >= 0 AND field_id <= 254), -- SMALLINT (2 bytes) for compact storage
    name VARCHAR(64) NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('TEXT', 'INTEGER', 'REAL', 'BOOLEAN')),
    is_indexed BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (database_id, field_id),
    FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE,
    UNIQUE (database_id, name)
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(26) PRIMARY KEY NOT NULL, -- ULID
    username VARCHAR(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
    password_hash TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    is_service_account BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS database_permissions (
    user_id VARCHAR(26) NOT NULL,
    database_id VARCHAR(26) NOT NULL,
    can_view BOOLEAN NOT NULL DEFAULT FALSE,
    can_create BOOLEAN NOT NULL DEFAULT FALSE,
    can_edit BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete BOOLEAN NOT NULL DEFAULT FALSE,
    can_admin BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, database_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    token_hash TEXT UNIQUE NOT NULL, 
    expiry BIGINT NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(26) PRIMARY KEY NOT NULL, -- ULID
    user_id VARCHAR(26) NOT NULL,
    name VARCHAR(64) NOT NULL,
    key_hash TEXT UNIQUE NOT NULL,
    key_hint VARCHAR(16) NOT NULL,
    scope_view BOOLEAN NOT NULL DEFAULT FALSE,
    scope_create BOOLEAN NOT NULL DEFAULT FALSE,
    scope_edit BOOLEAN NOT NULL DEFAULT FALSE,
    scope_delete BOOLEAN NOT NULL DEFAULT FALSE,
    scope_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at BIGINT NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000 AS BIGINT),
    expires_at BIGINT,
    last_used_at BIGINT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    timestamp BIGINT NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000 AS BIGINT),
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    resource TEXT NOT NULL,
    details TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);

CREATE TABLE IF NOT EXISTS system_locks (
    lock_name VARCHAR(64) PRIMARY KEY,
    locked_at BIGINT NOT NULL DEFAULT CAST(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000 AS BIGINT),
    locked_by VARCHAR(128) NOT NULL,
    expires_at BIGINT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS system_locks;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS database_permissions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS database_custom_fields;
DROP TABLE IF EXISTS databases;
