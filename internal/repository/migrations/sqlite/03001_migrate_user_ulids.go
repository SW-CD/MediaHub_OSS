package sqlitemigrations

import (
	"context"
	"database/sql"
	"fmt"
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up03001, down03001)
}

func up03001(ctx context.Context, tx *sql.Tx) error {
	// 1. Query all existing users
	type oldUser struct {
		id           int64
		username     string
		passwordHash string
		isAdmin      bool
	}
	userRows, err := tx.QueryContext(ctx, "SELECT id, username, password_hash, is_admin FROM users;")
	if err != nil {
		return fmt.Errorf("failed to query existing users: %w", err)
	}
	var oldUsers []oldUser
	for userRows.Next() {
		var u oldUser
		if err := userRows.Scan(&u.id, &u.username, &u.passwordHash, &u.isAdmin); err != nil {
			userRows.Close()
			return fmt.Errorf("failed to scan user: %w", err)
		}
		oldUsers = append(oldUsers, u)
	}
	userRows.Close()

	// 2. Query all existing database permissions
	type oldPerm struct {
		userID     int64
		databaseID string
		canView    bool
		canCreate  bool
		canEdit    bool
		canDelete  bool
	}
	permRows, err := tx.QueryContext(ctx, "SELECT user_id, database_id, can_view, can_create, can_edit, can_delete FROM database_permissions;")
	if err != nil {
		permRows = nil
	}
	var oldPerms []oldPerm
	if permRows != nil {
		for permRows.Next() {
			var p oldPerm
			if err := permRows.Scan(&p.userID, &p.databaseID, &p.canView, &p.canCreate, &p.canEdit, &p.canDelete); err != nil {
				permRows.Close()
				return fmt.Errorf("failed to scan permission: %w", err)
			}
			oldPerms = append(oldPerms, p)
		}
		permRows.Close()
	}

	// 3. Query all existing refresh tokens
	type oldToken struct {
		id        int64
		userID    int64
		tokenHash string
		expiry    int64
	}
	tokenRows, err := tx.QueryContext(ctx, "SELECT id, user_id, token_hash, expiry FROM refresh_tokens;")
	if err != nil {
		tokenRows = nil
	}
	var oldTokens []oldToken
	if tokenRows != nil {
		for tokenRows.Next() {
			var t oldToken
			if err := tokenRows.Scan(&t.id, &t.userID, &t.tokenHash, &t.expiry); err != nil {
				tokenRows.Close()
				return fmt.Errorf("failed to scan refresh token: %w", err)
			}
			oldTokens = append(oldTokens, t)
		}
		tokenRows.Close()
	}

	// 4. Generate mapping: Old Integer ID -> New ULID
	idMap := make(map[int64]repository.ULID)
	for _, u := range oldUsers {
		idMap[u.id] = repository.ULID(shared.GenerateULID())
	}

	// 5. Rename old tables to temporary names
	if _, err := tx.ExecContext(ctx, "ALTER TABLE refresh_tokens RENAME TO refresh_tokens_old;"); err != nil {
		return fmt.Errorf("failed to rename refresh_tokens to refresh_tokens_old: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE database_permissions RENAME TO database_permissions_old;"); err != nil {
		return fmt.Errorf("failed to rename database_permissions to database_permissions_old: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE users RENAME TO users_old;"); err != nil {
		return fmt.Errorf("failed to rename users to users_old: %w", err)
	}

	// 6. Create new tables with correct final names and constraints
	createUsersNew := `
	CREATE TABLE users (
		id VARCHAR(26) PRIMARY KEY NOT NULL,
		username VARCHAR(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
		password_hash TEXT NOT NULL,
		is_admin BOOLEAN NOT NULL DEFAULT 0,
		is_service_account BOOLEAN NOT NULL DEFAULT 0
	);`
	if _, err := tx.ExecContext(ctx, createUsersNew); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	createPermsNew := `
	CREATE TABLE database_permissions (
		user_id VARCHAR(26) NOT NULL,
		database_id TEXT(26) NOT NULL,
		can_view BOOLEAN NOT NULL DEFAULT 0,
		can_create BOOLEAN NOT NULL DEFAULT 0,
		can_edit BOOLEAN NOT NULL DEFAULT 0,
		can_delete BOOLEAN NOT NULL DEFAULT 0,
		PRIMARY KEY (user_id, database_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
	);`
	if _, err := tx.ExecContext(ctx, createPermsNew); err != nil {
		return fmt.Errorf("failed to create database_permissions table: %w", err)
	}

	createTokensNew := `
	CREATE TABLE refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id VARCHAR(26) NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expiry INTEGER NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	if _, err := tx.ExecContext(ctx, createTokensNew); err != nil {
		return fmt.Errorf("failed to create refresh_tokens table: %w", err)
	}

	// 7. Insert migrated users (must be inserted first to satisfy foreign keys)
	for _, u := range oldUsers {
		newID := idMap[u.id]
		_, err := tx.ExecContext(ctx, "INSERT INTO users (id, username, password_hash, is_admin, is_service_account) VALUES (?, ?, ?, ?, 0);",
			newID.String(), u.username, u.passwordHash, u.isAdmin)
		if err != nil {
			return fmt.Errorf("failed to insert user into users: %w", err)
		}
	}

	// 8. Insert migrated permissions
	for _, p := range oldPerms {
		newUserID, exists := idMap[p.userID]
		if !exists {
			continue // skip orphan permissions
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO database_permissions (user_id, database_id, can_view, can_create, can_edit, can_delete) VALUES (?, ?, ?, ?, ?, ?);",
			newUserID.String(), p.databaseID, p.canView, p.canCreate, p.canEdit, p.canDelete)
		if err != nil {
			return fmt.Errorf("failed to insert permission into database_permissions: %w", err)
		}
	}

	// 9. Insert migrated refresh tokens
	for _, t := range oldTokens {
		newUserID, exists := idMap[t.userID]
		if !exists {
			continue // skip orphan tokens
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expiry) VALUES (?, ?, ?);",
			newUserID.String(), t.tokenHash, t.expiry)
		if err != nil {
			return fmt.Errorf("failed to insert refresh token into refresh_tokens: %w", err)
		}
	}

	// 10. Drop old tables
	if _, err := tx.ExecContext(ctx, "DROP TABLE refresh_tokens_old;"); err != nil {
		return fmt.Errorf("failed to drop refresh_tokens_old: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DROP TABLE database_permissions_old;"); err != nil {
		return fmt.Errorf("failed to drop database_permissions_old: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DROP TABLE users_old;"); err != nil {
		return fmt.Errorf("failed to drop users_old: %w", err)
	}

	// 11. Create indexes
	if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);"); err != nil {
		return fmt.Errorf("failed to recreate index on refresh_tokens: %w", err)
	}

	return nil
}

func down03001(ctx context.Context, tx *sql.Tx) error {
	// 1. Query all existing users
	type newUserStruct struct {
		id           string
		username     string
		passwordHash string
		isAdmin      bool
	}
	userRows, err := tx.QueryContext(ctx, "SELECT id, username, password_hash, is_admin FROM users;")
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	var newUsers []newUserStruct
	for userRows.Next() {
		var u newUserStruct
		if err := userRows.Scan(&u.id, &u.username, &u.passwordHash, &u.isAdmin); err != nil {
			userRows.Close()
			return fmt.Errorf("failed to scan user: %w", err)
		}
		newUsers = append(newUsers, u)
	}
	userRows.Close()

	// 2. Query all existing database permissions
	type newPermStruct struct {
		userID     string
		databaseID string
		canView    bool
		canCreate  bool
		canEdit    bool
		canDelete  bool
	}
	permRows, err := tx.QueryContext(ctx, "SELECT user_id, database_id, can_view, can_create, can_edit, can_delete FROM database_permissions;")
	if err != nil {
		permRows = nil
	}
	var newPerms []newPermStruct
	if permRows != nil {
		for permRows.Next() {
			var p newPermStruct
			if err := permRows.Scan(&p.userID, &p.databaseID, &p.canView, &p.canCreate, &p.canEdit, &p.canDelete); err != nil {
				permRows.Close()
				return fmt.Errorf("failed to scan permission: %w", err)
			}
			newPerms = append(newPerms, p)
		}
		permRows.Close()
	}

	// 3. Query all existing refresh tokens
	type newTokenStruct struct {
		userID    string
		tokenHash string
		expiry    int64
	}
	tokenRows, err := tx.QueryContext(ctx, "SELECT user_id, token_hash, expiry FROM refresh_tokens;")
	if err != nil {
		tokenRows = nil
	}
	var newTokens []newTokenStruct
	if tokenRows != nil {
		for tokenRows.Next() {
			var t newTokenStruct
			if err := tokenRows.Scan(&t.userID, &t.tokenHash, &t.expiry); err != nil {
				tokenRows.Close()
				return fmt.Errorf("failed to scan refresh token: %w", err)
			}
			newTokens = append(newTokens, t)
		}
		tokenRows.Close()
	}

	// 4. Generate mapping: ULID -> Old Integer ID (using sequential integers)
	idMap := make(map[string]int64)
	var nextID int64 = 1
	for _, u := range newUsers {
		idMap[u.id] = nextID
		nextID++
	}

	// 5. Rename current tables to temporary names
	if _, err := tx.ExecContext(ctx, "ALTER TABLE refresh_tokens RENAME TO refresh_tokens_new;"); err != nil {
		return fmt.Errorf("failed to rename refresh_tokens: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE database_permissions RENAME TO database_permissions_new;"); err != nil {
		return fmt.Errorf("failed to rename database_permissions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE users RENAME TO users_new;"); err != nil {
		return fmt.Errorf("failed to rename users: %w", err)
	}

	// 6. Create old tables with original names and constraints
	createUsersOld := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT(64) UNIQUE NOT NULL CHECK(length(username) > 0 AND length(username) <= 64),
		password_hash TEXT NOT NULL,
		is_admin BOOLEAN NOT NULL DEFAULT 0
	);`
	if _, err := tx.ExecContext(ctx, createUsersOld); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	createPermsOld := `
	CREATE TABLE database_permissions (
		user_id INTEGER NOT NULL,
		database_id TEXT(26) NOT NULL,
		can_view BOOLEAN NOT NULL DEFAULT 0,
		can_create BOOLEAN NOT NULL DEFAULT 0,
		can_edit BOOLEAN NOT NULL DEFAULT 0,
		can_delete BOOLEAN NOT NULL DEFAULT 0,
		PRIMARY KEY (user_id, database_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
	);`
	if _, err := tx.ExecContext(ctx, createPermsOld); err != nil {
		return fmt.Errorf("failed to create database_permissions table: %w", err)
	}

	createTokensOld := `
	CREATE TABLE refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token_hash TEXT UNIQUE NOT NULL,
		expiry INTEGER NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	if _, err := tx.ExecContext(ctx, createTokensOld); err != nil {
		return fmt.Errorf("failed to create refresh_tokens table: %w", err)
	}

	// 7. Insert users (must be inserted first to satisfy foreign keys)
	for _, u := range newUsers {
		oldID := idMap[u.id]
		_, err := tx.ExecContext(ctx, "INSERT INTO users (id, username, password_hash, is_admin) VALUES (?, ?, ?, ?);",
			oldID, u.username, u.passwordHash, u.isAdmin)
		if err != nil {
			return fmt.Errorf("failed to insert user into users: %w", err)
		}
	}

	// 8. Insert permissions
	for _, p := range newPerms {
		oldUserID, exists := idMap[p.userID]
		if !exists {
			continue
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO database_permissions (user_id, database_id, can_view, can_create, can_edit, can_delete) VALUES (?, ?, ?, ?, ?, ?);",
			oldUserID, p.databaseID, p.canView, p.canCreate, p.canEdit, p.canDelete)
		if err != nil {
			return fmt.Errorf("failed to insert permission into database_permissions: %w", err)
		}
	}

	// 9. Insert refresh tokens
	for _, t := range newTokens {
		oldUserID, exists := idMap[t.userID]
		if !exists {
			continue
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expiry) VALUES (?, ?, ?);",
			oldUserID, t.tokenHash, t.expiry)
		if err != nil {
			return fmt.Errorf("failed to insert token into refresh_tokens: %w", err)
		}
	}

	// 10. Drop temporary tables
	if _, err := tx.ExecContext(ctx, "DROP TABLE refresh_tokens_new;"); err != nil {
		return fmt.Errorf("failed to drop refresh_tokens_new: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DROP TABLE database_permissions_new;"); err != nil {
		return fmt.Errorf("failed to drop database_permissions_new: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DROP TABLE users_new;"); err != nil {
		return fmt.Errorf("failed to drop users_new: %w", err)
	}

	// 11. Create indexes
	if _, err := tx.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);"); err != nil {
		return fmt.Errorf("failed to recreate index on refresh_tokens: %w", err)
	}

	return nil
}
