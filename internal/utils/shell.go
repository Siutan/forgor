package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetCurrentShell attempts to detect the current shell
func GetCurrentShell() string {
	// Check SHELL environment variable first
	shell := os.Getenv("SHELL")
	if shell != "" {
		return filepath.Base(shell)
	}

	// Fallback to runtime detection
	switch runtime.GOOS {
	case "windows":
		return "cmd"
	default:
		return "bash" // Default fallback
	}
}

// GetOperatingSystem returns the operating system name
func GetOperatingSystem() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	default:
		return runtime.GOOS
	}
}

// GetWorkingDirectory returns the current working directory
func GetWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// IsShellSupported checks if the shell is supported for history reading
func IsShellSupported(shell string) bool {
	supportedShells := []string{"bash", "zsh", "fish"}
	shell = strings.ToLower(shell)

	for _, supported := range supportedShells {
		if shell == supported {
			return true
		}
	}
	return false
}

// GetShellHistoryFile returns the path to the shell history file
func GetShellHistoryFile(shell string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	shell = strings.ToLower(shell)
	switch shell {
	case "bash":
		return filepath.Join(homeDir, ".bash_history")
	case "zsh":
		return filepath.Join(homeDir, ".zsh_history")
	case "fish":
		return filepath.Join(homeDir, ".local", "share", "fish", "fish_history")
	default:
		return ""
	}
}

// DetectShellFromProcess attempts to detect shell from process information
func DetectShellFromProcess() string {
	// Check parent process (on Unix systems)
	ppid := os.Getppid()
	if ppid == 0 {
		return GetCurrentShell()
	}

	// Try to read process information
	// This is a simplified approach - in a real implementation,
	// you might want to use more sophisticated process detection
	return GetCurrentShell()
}

// NormalizeShellName normalizes shell names to standard format
func NormalizeShellName(shell string) string {
	shell = strings.ToLower(strings.TrimSpace(shell))

	// Handle common variations
	switch {
	case strings.Contains(shell, "bash"):
		return "bash"
	case strings.Contains(shell, "zsh"):
		return "zsh"
	case strings.Contains(shell, "fish"):
		return "fish"
	case strings.Contains(shell, "cmd"):
		return "cmd"
	case strings.Contains(shell, "powershell") || strings.Contains(shell, "pwsh"):
		return "powershell"
	default:
		return shell
	}
}

// GetShellVersion attempts to get the version of the current shell
func GetShellVersion(shell string) string {
	// This would require executing shell commands to get version info
	// For now, return empty string - can be implemented later if needed
	return ""
}

// GetEnvironmentInfo returns useful environment information for context
func GetEnvironmentInfo() map[string]string {
	info := map[string]string{
		"os":    GetOperatingSystem(),
		"shell": GetCurrentShell(),
		"arch":  runtime.GOARCH,
		"pwd":   GetWorkingDirectory(),
	}

	// Add useful environment variables if set
	if user := os.Getenv("USER"); user != "" {
		info["user"] = user
	}
	if home := os.Getenv("HOME"); home != "" {
		info["home"] = home
	}
	if path := os.Getenv("PATH"); path != "" {
		// Don't include full PATH as it can be very long
		info["has_path"] = "true"
	}

	return info
}
