// filepath: internal/repository/schema.go
package repository

// initDB creates the necessary tables and indexes if they don't already exist.
func (s *Repository) initDB() error {
	schema := `
		CREATE TABLE IF NOT EXISTS databases (
			name TEXT(32) PRIMARY KEY NOT NULL CHECK(length(name) <= 32),
			content_type TEXT NOT NULL DEFAULT 'image',
			config TEXT NOT NULL DEFAULT '{}',
			hk_interval TEXT NOT NULL DEFAULT '1h',
			hk_disk_space TEXT NOT NULL DEFAULT '100G',
			hk_max_age TEXT NOT NULL DEFAULT '365d',
			custom_fields TEXT NOT NULL DEFAULT '[]',
			last_hk_run TIMESTAMP NOT NULL DEFAULT '1970-01-01T00:00:00Z',

			-- NEW: Denormalized stats for performance
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
	`
	_, err := s.DB.Exec(schema)
	return err
}
