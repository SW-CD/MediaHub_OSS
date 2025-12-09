// filepath: internal/cli/root_test.go
package cli

import (
	"mediahub/internal/config"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Helper to reset the global config and flags between tests
func resetGlobals() {
	cfg = nil
	port = 0
	logLevel = ""
	cfgFile = "config.toml" // Default
}

func TestConfigPrecedence(t *testing.T) {
	// We cannot easily run RootCmd.Execute() in tests because it calls os.Exit on failure
	// and runs the server. Instead, we test the initializeConfig and applyOverrides logic.

	t.Run("Defaults", func(t *testing.T) {
		resetGlobals()
		// Mock a non-existent config file to trigger defaults
		cfgFile = "nonexistent.toml"

		cmd := &cobra.Command{}
		err := initializeConfig(cmd)
		assert.NoError(t, err)

		assert.Equal(t, 8080, cfg.Server.Port)     // Default
		assert.Equal(t, "info", cfg.Logging.Level) // Default
	})

	t.Run("Environment Overrides Defaults", func(t *testing.T) {
		resetGlobals()
		cfgFile = "nonexistent.toml"

		os.Setenv("FDB_PORT", "9090")
		os.Setenv("FDB_LOG_LEVEL", "warn")
		defer os.Unsetenv("FDB_PORT")
		defer os.Unsetenv("FDB_LOG_LEVEL")

		cmd := &cobra.Command{}
		err := initializeConfig(cmd)
		assert.NoError(t, err)

		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, "warn", cfg.Logging.Level)
	})

	t.Run("Flags Override Environment", func(t *testing.T) {
		resetGlobals()
		cfgFile = "nonexistent.toml"

		// Set Env
		os.Setenv("FDB_PORT", "9090")
		defer os.Unsetenv("FDB_PORT")

		// Set Flag (Simulate parsing)
		port = 7070

		cmd := &cobra.Command{}
		// We must not set Changed() manually on the command without defining flags,
		// but applyOverrides checks the global variables bound to flags.

		err := initializeConfig(cmd)
		assert.NoError(t, err)

		assert.Equal(t, 7070, cfg.Server.Port)
	})

	t.Run("Config File Loading", func(t *testing.T) {
		resetGlobals()

		// Create a temporary config file
		content := []byte(`
[server]
port = 6060
[logging]
level = "error"
`)
		tmpFile := "test_config.toml"
		err := os.WriteFile(tmpFile, content, 0644)
		assert.NoError(t, err)
		defer os.Remove(tmpFile)

		cfgFile = tmpFile

		cmd := &cobra.Command{}
		err = initializeConfig(cmd)
		assert.NoError(t, err)

		assert.Equal(t, 6060, cfg.Server.Port)
		assert.Equal(t, "error", cfg.Logging.Level)
	})
}

func TestApplyOverrides(t *testing.T) {
	// Direct test of the applyOverrides logic
	c := &config.Config{
		Server:  config.ServerConfig{Port: 8080},
		Logging: config.LoggingConfig{Level: "info"},
	}

	// Reset global flags
	port = 9999
	logLevel = "debug"
	auditEnabled = true

	cmd := &cobra.Command{}
	// Determine if flag was "changed" is hard to mock on a bare command struct
	// so we rely on the fact that applyOverrides checks the global vars directly
	// for values != zero value (mostly).
	// Exception: boolean flags often need cmd.Flags().Changed check.
	// We'll skip the boolean test here if it relies strictly on .Changed() logic
	// unless we fully set up the FlagSet.

	applyOverrides(c, cmd)

	assert.Equal(t, 9999, c.Server.Port)
	assert.Equal(t, "debug", c.Logging.Level)
}
