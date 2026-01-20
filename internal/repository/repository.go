package repository

import (
	"mediahub/internal/shared"
	"time"
)

type Repository interface {
	Close() error

	// Database
	CreateDatabase(db *Database) (*Database, error)
	GetDatabase(name string) (*Database, error)
	GetDatabases() ([]Database, error)
	UpdateDatabase(db *Database) error
	DeleteDatabase(name string) error
	GetDatabaseStats(name string) (DatabaseStats, error)

	// Entry
	CreateEntry(db *Database, entry *Entry) (*Entry, error)
	GetEntry(db *Database, id int) (*Entry, error)
	GetEntries(db *Database, limit, offset int, order string, tstart, tend int64, customFields []shared.CustomField) ([]Entry, error)
	UpdateEntry(db *Database, entry *Entry) error
	DeleteEntry(db *Database, id int) error
	DeleteEntries(db *Database, ids []int) error
	SearchEntries(dbName string, req *SearchRequest, customFields []shared.CustomField) ([]Entry, error)

	// User
	GetUserByUsername(username string) (*User, error)

	// Token
	StoreRefreshToken(userID int64, tokenHash string, expiry time.Time) error
	ValidateRefreshToken(tokenHash string) (int64, error)
	DeleteRefreshToken(tokenHash string) error
	DeleteAllRefreshTokensForUser(userID int64)

	// Migration
	GetMigrationVersion() error
	MigrateUp() error
	MigrateDown() error
}

func UserExists(s Repository, username string) (bool, error) {
	_, err := s.GetUserByUsername(username)
	if err != nil {
		if err == shared.ErrUserNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
