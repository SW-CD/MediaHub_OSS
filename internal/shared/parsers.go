package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseSize parses a size string (e.g., "100G", "500MB", "1024 bytes") into bytes.
func ParseSize(sizeStr string) (uint64, error) {
	// (?i) makes it case-insensitive.
	// \s* allows optional spaces between the number and the unit.
	// ([a-z]*) captures any alphabetical characters that follow the number.
	re := regexp.MustCompile(`(?i)^(\d+)\s*([a-z]*)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(sizeStr))

	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	value, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %s", matches[1])
	}

	unit := ""
	if len(matches) > 2 {
		unit = strings.ToUpper(matches[2]) // Normalize to uppercase for the switch
	}

	switch unit {
	case "T", "TB":
		return value * (1 << 40), nil
	case "G", "GB":
		return value * (1 << 30), nil
	case "M", "MB":
		return value * (1 << 20), nil
	case "K", "KB":
		return value * (1 << 10), nil
	case "", "B", "BYTE", "BYTES":
		return value, nil
	default:
		return 0, fmt.Errorf("unsupported size unit: %s", unit)
	}
}

// ParseDuration parses a duration string with support for days and various aliases
// (e.g., "30d", "24 hours", "15 mins").
func ParseDuration(durationStr string) (time.Duration, error) {
	trimmedStr := strings.TrimSpace(durationStr)

	// Handle "0" as a special case for "disabled"
	if trimmedStr == "0" {
		return 0, nil
	}

	// Capture the number and any trailing alphabetical characters
	re := regexp.MustCompile(`(?i)^(\d+)\s*([a-z]+)$`)
	matches := re.FindStringSubmatch(trimmedStr)

	if len(matches) < 3 {
		return 0, fmt.Errorf("invalid duration format: %s", durationStr)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", matches[1])
	}

	// If value is 0 (e.g., "0d"), return 0 duration
	if value == 0 {
		return 0, nil
	}

	unit := strings.ToLower(matches[2]) // Normalize to lowercase for the switch
	switch unit {
	case "d", "day", "days":
		return time.Duration(value) * 24 * time.Hour, nil
	case "h", "hr", "hrs", "hour", "hours":
		return time.Duration(value) * time.Hour, nil
	case "m", "min", "mins", "minute", "minutes":
		return time.Duration(value) * time.Minute, nil
	case "s", "sec", "secs", "second", "seconds":
		return time.Duration(value) * time.Second, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}
}
