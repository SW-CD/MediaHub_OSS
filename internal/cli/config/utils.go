package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseSize parses a size string (e.g., "100G", "500MB") into bytes.
// Duplicated here to keep the config package self-contained and dependency-free.
func parseSize(sizeStr string) (int64, error) {
	re := regexp.MustCompile(`(?i)^(\d+)\s*(K|M|G|T)?B?$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(sizeStr))

	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
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
