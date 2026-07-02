package migrations

import (
	"strings"
	"testing"
)

func TestCheckVersion(t *testing.T) {
	// Case 1: Match RequiredVersion
	err := CheckVersion(RequiredVersion)
	if err != nil {
		t.Fatalf("expected no error for version %d, got: %v", RequiredVersion, err)
	}

	// Case 2: Version older than RequiredVersion
	err = CheckVersion(RequiredVersion - 1)
	if err == nil {
		t.Fatalf("expected error for older version %d, got nil", RequiredVersion-1)
	}
	if !strings.Contains(err.Error(), "older") || !strings.Contains(err.Error(), "migrate up") {
		t.Errorf("unexpected error message for older version: %v", err)
	}

	// Case-3: Version newer than RequiredVersion
	err = CheckVersion(RequiredVersion + 1)
	if err == nil {
		t.Fatalf("expected error for newer version %d, got nil", RequiredVersion+1)
	}
	if !strings.Contains(err.Error(), "newer") || !strings.Contains(err.Error(), "migrate down") {
		t.Errorf("unexpected error message for newer version: %v", err)
	}
}
