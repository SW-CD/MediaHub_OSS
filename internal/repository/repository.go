// filepath: internal/repository/repository.go
// Package repository provides the functionality for interacting with the SQLite database.
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"mediahub/internal/config"
	"regexp"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/patrickmn/go-cache"
)

var SafeNameRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

// ErrInvalidFilter is returned when a search filter is malformed or invalid.
var ErrInvalidFilter = errors.New("invalid filter")

// Repository provides access to the database.
type Repository struct {
	DB    *sql.DB
	Cache *cache.Cache
	Cfg   *config.Config
}

// NewRepository creates a new database service instance.
// It opens the database file and ensures the schema is created.
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
	}

	if err := repo.initDB(); err != nil {
		return nil, fmt.Errorf("could not initialize database schema: %w", err)
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
