package shared

import (
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		hasErr   bool
	}{
		{"1024", 1024, false},
		{"1024B", 1024, false},
		{"1024 bytes", 1024, false},
		{"1024 BYTE", 1024, false},
		{"1K", 1024, false},
		{"1KB", 1024, false},
		{"10 MB", 10 * 1024 * 1024, false},
		{"2GB", 2 * 1024 * 1024 * 1024, false},
		{"1TB", 1 * 1024 * 1024 * 1024 * 1024, false},
		{"invalid", 0, true},
		{"-5MB", 0, true},
		{"10XB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if (err != nil) != tt.hasErr {
				t.Fatalf("ParseSize(%q) error = %v, wantErr %v", tt.input, err, tt.hasErr)
			}
			if !tt.hasErr && got != tt.expected {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasErr   bool
	}{
		{"0", 0, false},
		{"0d", 0, false},
		{"30s", 30 * time.Second, false},
		{"15 secs", 15 * time.Second, false},
		{"10m", 10 * time.Minute, false},
		{"45 mins", 45 * time.Minute, false},
		{"2h", 2 * time.Hour, false},
		{"24 hours", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"14 days", 14 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"-5h", 0, true},
		{"10xyz", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if (err != nil) != tt.hasErr {
				t.Fatalf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.hasErr)
			}
			if !tt.hasErr && got != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
