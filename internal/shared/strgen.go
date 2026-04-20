package shared

import (
	"fmt"
	"strings"
	"time"
)

// convert a number representing the number of bytes into a string
// representation, e.g., "12MB"
func BytesToString(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}

	// Define our suffixes
	units := []string{"B", "K", "M", "G", "T", "P"}
	value := float64(b)
	unitIndex := 0

	// Divide by 1024 until the value is less than 1024, or we run out of units
	for value >= unit && unitIndex < len(units)-1 {
		value /= unit
		unitIndex++
	}

	// Format to 1 decimal place (e.g., "1.5"), but use TrimSuffix to drop ".0"
	// so whole numbers look cleaner (e.g., "12MB" instead of "12.0MB")
	formattedValue := strings.TrimSuffix(fmt.Sprintf("%.1f", value), ".0")

	return formattedValue + units[unitIndex]
}

func DurationToString(d time.Duration) string {
	if d == 0 {
		return "0"
	}

	// Calculate units using Go's built-in time constants
	days := d / (24 * time.Hour)
	d %= 24 * time.Hour

	hours := d / time.Hour
	d %= time.Hour

	minutes := d / time.Minute
	d %= time.Minute

	seconds := d / time.Second

	// Build the output string dynamically, ignoring units that are 0
	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dmin", minutes))
	}

	// Include seconds if it's > 0, or if the duration was entirely less than a minute
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	// Join the parts with a space (e.g., "1d 12h")
	return strings.Join(parts, " ")
}
