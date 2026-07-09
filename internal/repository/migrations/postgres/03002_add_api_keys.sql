-- Migration: Add API Keys Table
-- Description: Creates the api_keys table for standard users and service accounts.
--
-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(26) PRIMARY KEY NOT NULL, -- ULID
    user_id VARCHAR(26) NOT NULL, -- Links to the Service Account or standard user
    
    name VARCHAR(64) NOT NULL,
    key_hash TEXT UNIQUE NOT NULL, -- SHA-256 hash of the token secret
    key_hint VARCHAR(16) NOT NULL, -- Stores 'srv_...a1b2'
    
    -- Token Scopes (The filters)
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

-- +goose Down
DROP TABLE IF EXISTS api_keys;
