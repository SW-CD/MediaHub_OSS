// filepath: internal/repository/token_repo.go
package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// StoreRefreshToken saves the hash of a refresh token to the database.
func (s *Repository) StoreRefreshToken(userID int64, tokenHash string, expiry time.Time) error {
	query := "INSERT INTO refresh_tokens (user_id, token_hash, expiry) VALUES (?, ?, ?)"
	_, err := s.DB.Exec(query, userID, tokenHash, expiry)
	return err
}

// ValidateRefreshToken checks if a token hash exists and is not expired, returning the user ID.
func (s *Repository) ValidateRefreshToken(tokenHash string) (int64, error) {
	query := "SELECT user_id FROM refresh_tokens WHERE token_hash = ? AND expiry > ?"
	var userID int64
	err := s.DB.QueryRow(query, tokenHash, time.Now()).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("token not found or expired")
		}
		return 0, err
	}
	return userID, nil
}

// DeleteRefreshToken removes a specific refresh token hash from the database.
func (s *Repository) DeleteRefreshToken(tokenHash string) error {
	query := "DELETE FROM refresh_tokens WHERE token_hash = ?"
	_, err := s.DB.Exec(query, tokenHash)
	return err
}

// DeleteAllRefreshTokensForUser revokes all sessions for a specific user.
func (s *Repository) DeleteAllRefreshTokensForUser(userID int64) error {
	query := "DELETE FROM refresh_tokens WHERE user_id = ?"
	_, err := s.DB.Exec(query, userID)
	return err
}
