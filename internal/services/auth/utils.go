// filepath: internal/services/auth/utils.go
package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateSecret creates a cryptographically secure random string.
// Use by the CLI package in case no secret was found or configured.
func GenerateSecret() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
