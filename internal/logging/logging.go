// internal/logging/logging.go
package logging

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Log is a pre-configured logger instance.
var Log = logrus.New()

func init() {
	// Set the log format.
	// Using JSON format for structured logging.
	Log.SetFormatter(&logrus.JSONFormatter{})

	// Set the output.
	// Default is stderr, but can be set to a file.
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.TraceLevel)
}

// Init initializes the logger with a specific level.
func Init(level string) {
	switch strings.ToLower(level) {
	case "trace":
		Log.SetLevel(logrus.TraceLevel)
	case "debug":
		Log.SetLevel(logrus.DebugLevel)
	case "info":
		Log.SetLevel(logrus.InfoLevel)
	case "warn":
		Log.SetLevel(logrus.WarnLevel)
	case "error":
		Log.SetLevel(logrus.ErrorLevel)
	default:
		Log.SetLevel(logrus.InfoLevel)
	}
}
