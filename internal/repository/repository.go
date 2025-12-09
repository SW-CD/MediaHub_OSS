// filepath: internal/repository/repository.go
// Package repository provides the functionality for interacting with the SQLite database.
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"mediahub/internal/config"
	"mediahub/internal/db/migrations" // Import the embedded migrations
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/patrickmn/go-cache"
	"github.com/pressly/goose/v3"
)

var SafeNameRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

// ErrInvalidFilter is returned when a search filter is malformed or invalid.
var ErrInvalidFilter = errors.New("invalid filter")

// Repository provides access to the database.
type Repository struct {
	DB      *sql.DB
	Cache   *cache.Cache
	Cfg     *config.Config
	Builder squirrel.StatementBuilderType // SQL Query Builder
}

// NewRepository creates a new database service instance.
// It opens the database file. Note: It does NOT automatically migrate.
// The caller must run ValidateSchema or the 'migrate' CLI command.
func NewRepository(cfg *config.Config) (*Repository, error) {
	// Add `_journal=WAL` for better concurrent performance (many readers, one writer).
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal=WAL", cfg.Database.Path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	// Set connection pool properties
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	repo := &Repository{
		DB:    db,
		Cache: cache.New(5*time.Minute, 10*time.Minute),
		Cfg:   cfg,
		// Initialize Squirrel with Question mark placeholders for SQLite
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}

	return repo, nil
}

// Close closes the database connection.
func (s *Repository) Close() {
	s.DB.Close()
}

// BeginTx starts a new database transaction.
func (s *Repository) BeginTx() (*Tx, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// ValidateSchema checks if the database schema is up to date with the binary.
// It returns nil if the versions match.
func (s *Repository) ValidateSchema() error {
	// Configure Goose to use the embedded filesystem
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// 1. Get current DB version
	currentVersion, err := goose.GetDBVersion(s.DB)
	if err != nil {
		return fmt.Errorf("failed to get current database version: %w", err)
	}

	// 2. Determine the latest version in the binary
	latestVersion, err := getLatestMigrationVersion()
	if err != nil {
		return fmt.Errorf("failed to determine latest migration version: %w", err)
	}

	// 3. Compare
	if currentVersion < latestVersion {
		return fmt.Errorf("database schema is outdated (DB: v%d, App: v%d). Please run './mediahub migrate up' to update", currentVersion, latestVersion)
	}
	if currentVersion > latestVersion {
		return fmt.Errorf("database schema is newer than this application binary (DB: v%d, App: v%d). Please upgrade the application", currentVersion, latestVersion)
	}

	return nil
}

// getLatestMigrationVersion scans the embedded FS to find the highest version number.
func getLatestMigrationVersion() (int64, error) {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return 0, err
	}

	var maxVersion int64 = 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		// Parse filename like "001_init.sql" -> 1
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		ver, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue // Skip files that don't start with a number
		}
		if ver > maxVersion {
			maxVersion = ver
		}
	}
	return maxVersion, nil
}
