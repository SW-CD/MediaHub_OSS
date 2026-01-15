// internal/logging/logging.go
package logging

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Init initializes the logger with a specific level.
func NewLogger(level string) *logrus.Logger {

	var log = logrus.New()

	// Set the log format.
	// Using JSON format for structured logging.
	log.SetFormatter(&logrus.JSONFormatter{})

	// Set the output.
	// Default is stderr, but can be set to a file.
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.TraceLevel)

	switch strings.ToLower(level) {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
	return log
}
