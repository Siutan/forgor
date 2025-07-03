package tests

import (
	"testing"
)

// Version returns the current version for testing
func pkgVersion() string {
	return "test"
}

func TestPkgVersion(t *testing.T) {
	version := pkgVersion()
	if version == "" {
		t.Error("Version should not be empty")
	}

	if version != "test" {
		t.Errorf("Expected version 'test', got '%s'", version)
	}
}

func TestPkgVersionFormat(t *testing.T) {
	version := pkgVersion()

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
