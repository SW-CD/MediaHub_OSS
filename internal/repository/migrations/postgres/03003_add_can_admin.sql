-- Migration: Add can_admin to database_permissions
-- Description: Adds a database-specific admin role for non-global admins

-- +goose Up
ALTER TABLE database_permissions ADD COLUMN can_admin BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE database_permissions DROP COLUMN can_admin;
