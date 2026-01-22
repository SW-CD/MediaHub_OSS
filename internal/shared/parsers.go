package shared

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// parseSize parses a size string (e.g., "100G", "500MB") into bytes.
// Duplicated here to keep the config package self-contained and dependency-free.
func ParseSize(sizeStr string) (uint64, error) {
	re := regexp.MustCompile(`(?i)^(\d+)\s*(K|M|G|T)?B?$`)
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
		unit = strings.ToUpper(matches[2])
	}

	switch unit {
	case "T":
		return value * (1 << 40), nil
	case "G":
		return value * (1 << 30), nil
	case "M":
		return value * (1 << 20), nil
	case "K":
		return value * (1 << 10), nil
	default:
		return value, nil
	}
}

// shared.ParseDuration parses a duration string with support for days
// (e.g., "30d", "24h") into a time.Duration. If you dont need support for "d", you can
// just use time.ParseDuration .
// A special value of "0" is allowed and returns 0 duration (disabling the check).
func ParseDuration(durationStr string) (time.Duration, error) {
	trimmedStr := strings.TrimSpace(durationStr)
	// Handle "0" as a special case for "disabled"
	if trimmedStr == "0" {
		return 0, nil
	}

	re := regexp.MustCompile(`^(\d+)\s*(d|h|m|s)$`)
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

	unit := matches[2]
	switch unit {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "s":
		return time.Duration(value) * time.Second, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}
}
