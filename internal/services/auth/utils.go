// filepath: internal/services/auth/utils.go
package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateSecret creates a cryptographically secure random string.
// This was previously logic inside main.go, moved here for better organization
// and accessibility by the CLI package.
func GenerateSecret() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
