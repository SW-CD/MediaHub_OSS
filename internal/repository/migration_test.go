// filepath: internal/repository/migration_test.go
package repository

import (
	"mediahub/internal/config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSchema(t *testing.T) {
	// Setup a clean DB
	dbPath := "test_validate_schema.db"
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	cfg := &config.Config{Database: config.DatabaseConfig{Path: dbPath}}
	repo, err := NewRepository(cfg)
	assert.NoError(t, err)
	defer repo.Close()

	// 1. New DB should be invalid (needs migration)
	err = repo.ValidateSchema()
	assert.Error(t, err, "Fresh DB should be considered outdated")
	assert.Contains(t, err.Error(), "database schema is outdated")

	// 2. Apply Migrations (Simulate "migrate up")
	applyTestMigrations(t, repo)

	// 3. Verify Schema is now Valid
	err = repo.ValidateSchema()
	assert.NoError(t, err, "DB should be valid after applying migrations")
}

func TestEnsureSchemaBootstrapped(t *testing.T) {
	t.Run("Fresh Database", func(t *testing.T) {
		dbPath := "test_bootstrap_fresh.db"
		os.Remove(dbPath)
		defer os.Remove(dbPath)

		cfg := &config.Config{Database: config.DatabaseConfig{Path: dbPath}}
		repo, err := NewRepository(cfg)
		assert.NoError(t, err)
		defer repo.Close()

		// Act: Run bootstrap on empty DB
		err = repo.EnsureSchemaBootstrapped()
		assert.NoError(t, err)

		// Assert: Schema should now be valid (fully migrated)
		err = repo.ValidateSchema()
		assert.NoError(t, err, "Fresh DB should be fully migrated after bootstrap")

		// Assert: Users table should exist
		var tableName string
		err = repo.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
		assert.NoError(t, err)
		assert.Equal(t, "users", tableName)
	})

	t.Run("Existing Database (Skip)", func(t *testing.T) {
		dbPath := "test_bootstrap_existing.db"
		os.Remove(dbPath)
		defer os.Remove(dbPath)

		cfg := &config.Config{Database: config.DatabaseConfig{Path: dbPath}}
		repo, err := NewRepository(cfg)
		assert.NoError(t, err)
		defer repo.Close()

		// Arrange: Simulate an "existing" DB by manually creating the version table.
		// We do NOT create the 'users' table. This simulates a state where the DB
		// is "initialized" but might be outdated or broken, and we want to ensure
		// the bootstrap logic respects the existence of the table and does NOTHING.
		_, err = repo.DB.Exec("CREATE TABLE goose_db_version (id INTEGER PRIMARY KEY, version_id INTEGER, is_applied BOOLEAN, tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP);")
		assert.NoError(t, err)

		// Act: Run bootstrap
		err = repo.EnsureSchemaBootstrapped()
		assert.NoError(t, err)

		// Assert: The 'users' table should STILL NOT exist.
		// If bootstrap ran, it would have created 'users'.
		var name string
		err = repo.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&name)
		assert.Error(t, err, "Bootstrap should have skipped migration")

		// ValidateSchema should fail (because we tricked it into thinking it exists, but it has no version data)
		// This proves we handed control over to the manual migration process.
		err = repo.ValidateSchema()
		assert.Error(t, err)
	})
}
