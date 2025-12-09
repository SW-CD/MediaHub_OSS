// filepath: internal/config/config_test.go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"8MB", 8 * 1024 * 1024, false},
		{"512KB", 512 * 1024, false},
		{"1GB", 1 * 1024 * 1024 * 1024, false},
		{"100", 100, false},        // Bytes
		{"1024B", 1024, false},     // Bytes with suffix
		{" 4 MB ", 4194304, false}, // Spaces
		{"8mb", 8388608, false},    // Lowercase
		{"invalid", 0, true},
		{"10XB", 0, true},
		{"-10MB", 0, true}, // Regex expects digits, not negatives
	}

	for _, tc := range tests {
		val, err := parseSize(tc.input)
		if tc.hasError {
			assert.Error(t, err, "Expected error for input: %s", tc.input)
		} else {
			assert.NoError(t, err, "Unexpected error for input: %s", tc.input)
			assert.Equal(t, tc.expected, val, "Mismatch for input: %s", tc.input)
		}
	}
}

func TestConfig_ParseAndValidate(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				MaxSyncUploadSize: "10MB",
			},
		}
		err := cfg.ParseAndValidate()
		assert.NoError(t, err)
		assert.Equal(t, int64(10485760), cfg.MaxSyncUploadSizeBytes)
	})

	t.Run("Default Fallback", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				MaxSyncUploadSize: "", // Empty
			},
		}
		err := cfg.ParseAndValidate()
		assert.NoError(t, err)
		assert.Equal(t, "8MB", cfg.Server.MaxSyncUploadSize)
		assert.Equal(t, int64(8388608), cfg.MaxSyncUploadSizeBytes)
	})

	t.Run("Invalid Config", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				MaxSyncUploadSize: "NotASize",
			},
		}
		err := cfg.ParseAndValidate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid max_sync_upload_size")
	})
}
