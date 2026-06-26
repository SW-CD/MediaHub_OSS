-- TODO replace with Go version, similar to how it is done for sqlite
-- +goose Up
ALTER TABLE databases ADD COLUMN n_max_queued INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE databases DROP COLUMN n_max_queued;
