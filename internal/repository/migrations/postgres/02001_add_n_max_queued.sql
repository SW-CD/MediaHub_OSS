-- Migration: Add n_max_queued column to databases
-- Description: Updates databases table schema for queue conversions configuration support.
--   - Up: Adds the 'n_max_queued' integer column to the 'databases' table.
--   - Down: Drops the 'n_max_queued' column from the 'databases' table.
--
-- TODO replace with Go version, similar to how it is done for sqlite
-- +goose Up
ALTER TABLE databases ADD COLUMN n_max_queued INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE databases DROP COLUMN n_max_queued;
