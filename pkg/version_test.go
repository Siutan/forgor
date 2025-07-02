package pkg

import (
	"testing"
)

// Version returns the current version for testing
func Version() string {
	return "test"
}

func TestVersion(t *testing.T) {
	version := Version()
	if version == "" {
		t.Error("Version should not be empty")
	}

	if version != "test" {
		t.Errorf("Expected version 'test', got '%s'", version)
	}
}

func TestVersionFormat(t *testing.T) {
	version := Version()

	// Test that version is a valid string
	if len(version) == 0 {
		t.Error("Version should have non-zero length")
	}

	// Test that version contains only valid characters (for this test case)
	for _, char := range version {
		if !(char >= 'a' && char <= 'z') && !(char >= '0' && char <= '9') && char != '.' && char != '-' {
			t.Errorf("Version contains invalid character: %c", char)
		}
	}
}
