package tests

import (
	"runtime"
	"strings"
	"testing"

	"forgor/internal/utils"
)

func TestGetOperatingSystem(t *testing.T) {
	os := utils.GetOperatingSystem()

	// Should return one of the expected values
	validOS := []string{"Linux", "macOS", "Windows", "Unknown"}
	isValid := false
	for _, valid := range validOS {
		if os == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		t.Errorf("GetOperatingSystem() returned unexpected value: %s", os)
	}

	// Should match runtime.GOOS mapping
	switch runtime.GOOS {
	case "linux":
		if os != "Linux" {
			t.Errorf("Expected 'Linux' for linux, got '%s'", os)
		}
	case "darwin":
		if os != "macOS" {
			t.Errorf("Expected 'macOS' for darwin, got '%s'", os)
		}
	case "windows":
		if os != "Windows" {
			t.Errorf("Expected 'Windows' for windows, got '%s'", os)
		}
	default:
		if os != "Unknown" {
			t.Errorf("Expected 'Unknown' for unknown OS, got '%s'", os)
		}
	}
}

func TestNormalizeShellName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bash", "bash"},
		{"/bin/bash", "bash"},
		{"/usr/local/bin/bash", "bash"},
		{"zsh", "zsh"},
		{"/bin/zsh", "zsh"},
		{"/usr/local/bin/zsh", "zsh"},
		{"fish", "fish"},
		{"/usr/local/bin/fish", "fish"},
		{"sh", "sh"},
		{"/bin/sh", "/bin/sh"}, // Doesn't contain "bash", "zsh", etc. so returns as-is
		{"powershell", "powershell"},
		{"pwsh", "powershell"}, // pwsh maps to powershell
		{"cmd", "cmd"},
		{"unknown-shell", "unknown-shell"},
		{"", ""},
	}

	for _, test := range tests {
		result := utils.NormalizeShellName(test.input)
		if result != test.expected {
			t.Errorf("NormalizeShellName(%s) = %s; want %s", test.input, result, test.expected)
		}
	}
}

func TestIsShellSupported(t *testing.T) {
	tests := []struct {
		shell    string
		expected bool
	}{
		{"bash", true},
		{"zsh", true},
		{"fish", true},
		{"sh", false},         // Only bash, zsh, fish are supported
		{"powershell", false}, // Only bash, zsh, fish are supported
		{"pwsh", false},       // Only bash, zsh, fish are supported
		{"cmd", false},
		{"unknown", false},
		{"", false},
	}

	for _, test := range tests {
		result := utils.IsShellSupported(test.shell)
		if result != test.expected {
			t.Errorf("IsShellSupported(%s) = %v; want %v", test.shell, result, test.expected)
		}
	}
}

func TestGetWorkingDirectory(t *testing.T) {
	wd := utils.GetWorkingDirectory()

	if wd == "" {
		t.Error("GetWorkingDirectory() should not return empty string")
	}

	// Should be an absolute path
	if wd[0] != '/' && !(len(wd) >= 3 && wd[1] == ':') { // Unix absolute or Windows drive
		t.Errorf("GetWorkingDirectory() should return absolute path, got: %s", wd)
	}
}

func TestReadShellHistory(t *testing.T) {
	// Test with zero maxCommands
	commands, err := utils.ReadShellHistory("bash", 0)
	if err != nil {
		t.Errorf("ReadShellHistory with 0 maxCommands should not error: %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("ReadShellHistory with 0 maxCommands should return empty slice, got %d commands", len(commands))
	}

	// Test with unsupported shell
	commands, err = utils.ReadShellHistory("unsupported", 5)
	if err != nil {
		t.Errorf("ReadShellHistory with unsupported shell should not error: %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("ReadShellHistory with unsupported shell should return empty slice, got %d commands", len(commands))
	}
}

func TestGetCurrentShellHistory(t *testing.T) {
	// Test with zero maxCommands
	commands, err := utils.GetCurrentShellHistory(0)
	if err != nil {
		t.Errorf("GetCurrentShellHistory with 0 maxCommands should not error: %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("GetCurrentShellHistory with 0 maxCommands should return empty slice, got %d commands", len(commands))
	}

	// Test with small number - this might return actual history or empty slice
	commands, err = utils.GetCurrentShellHistory(1)
	if err != nil {
		t.Errorf("GetCurrentShellHistory should not error: %v", err)
	}
	// Don't check length as it depends on whether history file exists and has content
}

func TestFilterSensitiveCommands(t *testing.T) {
	// This test would need to be added if we export the filterSensitiveCommands function
	// For now, we test it indirectly through GetCurrentShellHistory
	commands, err := utils.GetCurrentShellHistory(10)
	if err != nil {
		t.Errorf("GetCurrentShellHistory should not error: %v", err)
	}

	// Check that no sensitive commands are returned
	sensitivePatterns := []string{
		"password", "passwd", "pass",
		"token", "secret", "key",
		"api_key", "apikey", "apisecret",
	}

	for _, cmd := range commands {
		cmdLower := strings.ToLower(cmd)
		for _, pattern := range sensitivePatterns {
			if strings.Contains(cmdLower, pattern) {
				t.Errorf("Sensitive command found in filtered history: %s", cmd)
			}
		}
	}
}
