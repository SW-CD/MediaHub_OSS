-- Migration: Add can_admin to database_permissions
-- Description: Adds a database-specific admin role for non-global admins

-- +goose Up
ALTER TABLE database_permissions ADD COLUMN can_admin BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite does not easily support dropping a column without recreating the table,
-- but newer versions of SQLite (3.35.0+) do. Assuming a modern SQLite version.
ALTER TABLE database_permissions DROP COLUMN can_admin;
