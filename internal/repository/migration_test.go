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
	dbPath := "test_migration.db"
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	cfg := &config.Config{Database: config.DatabaseConfig{Path: dbPath}}
	repo, err := NewRepository(cfg)
	assert.NoError(t, err)
	defer repo.Close()

	// 1. New DB should be invalid (needs migration)
	// ValidateSchema checks goose.GetDBVersion. On a fresh DB, goose creates its table
	// and returns 0. If embedded migrations exist (> v0), this returns error.
	err = repo.ValidateSchema()
	assert.Error(t, err, "Fresh DB should be considered outdated")
	assert.Contains(t, err.Error(), "database schema is outdated")

	// 2. Apply Migrations (Simulate "migrate up")
	// Using the internal helper we created for repository tests
	applyTestMigrations(t, repo)

	// 3. Verify Schema is now Valid
	err = repo.ValidateSchema()
	assert.NoError(t, err, "DB should be valid after applying migrations")

	// 4. Verify a table exists (Sanity check)
	var tableName string
	err = repo.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	assert.NoError(t, err, "Users table should exist after migration")
	assert.Equal(t, "users", tableName)
}
