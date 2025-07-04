package utils

import (
	"bufio"
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

// ReadShellHistory reads the last N commands from the shell history file
func ReadShellHistory(shell string, maxCommands int) ([]string, error) {
	if maxCommands <= 0 {
		return []string{}, nil
	}

	// Get the history file path
	historyFile := GetShellHistoryFile(shell)
	if historyFile == "" {
		return []string{}, nil // No history file found for this shell
	}

	// Check if the history file exists
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return []string{}, nil // History file doesn't exist
	}

	// Handle different shell history formats
	shell = strings.ToLower(shell)
	switch shell {
	case "zsh":
		return readZshHistory(historyFile, maxCommands)
	case "fish":
		return readFishHistory(historyFile, maxCommands)
	case "bash":
		fallthrough
	default:
		return readBashHistory(historyFile, maxCommands)
	}
}

// readBashHistory reads bash history (simple line-by-line format)
func readBashHistory(historyFile string, maxCommands int) ([]string, error) {
	file, err := os.Open(historyFile)
	if err != nil {
		return []string{}, err
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)

	// Read all lines and collect commands
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			commands = append(commands, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return []string{}, err
	}

	// Return the last N commands (or all if less than N)
	if len(commands) <= maxCommands {
		return commands, nil
	}

	// Return the last maxCommands
	return commands[len(commands)-maxCommands:], nil
}

// readZshHistory reads zsh history (extended format with timestamps)
func readZshHistory(historyFile string, maxCommands int) ([]string, error) {
	file, err := os.Open(historyFile)
	if err != nil {
		return []string{}, err
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)

	// Read all lines and collect commands
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Zsh history format: ": timestamp:duration;command"
		// We need to extract just the command part
		if strings.HasPrefix(line, ": ") {
			// Find the last semicolon which separates metadata from command
			lastSemicolon := strings.LastIndex(line, ";")
			if lastSemicolon != -1 && lastSemicolon < len(line)-1 {
				command := strings.TrimSpace(line[lastSemicolon+1:])
				if command != "" {
					commands = append(commands, command)
				}
			}
		} else {
			// Fallback: treat as regular command
			commands = append(commands, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return []string{}, err
	}

	// Return the last N commands (or all if less than N)
	if len(commands) <= maxCommands {
		return commands, nil
	}

	// Return the last maxCommands
	return commands[len(commands)-maxCommands:], nil
}

// readFishHistory reads fish history (SQLite database format)
func readFishHistory(historyFile string, maxCommands int) ([]string, error) {
	// Fish uses SQLite database, which is complex to parse
	// For now, we'll return empty slice and log a message
	// TODO: Implement SQLite parsing for fish history
	return []string{}, nil
}

// GetCurrentShellHistory reads history from the current shell
func GetCurrentShellHistory(maxCommands int) ([]string, error) {
	shell := GetCurrentShell()
	commands, err := ReadShellHistory(shell, maxCommands)
	if err != nil {
		return commands, err
	}

	// Filter out sensitive information
	return filterSensitiveCommands(commands), nil
}

// filterSensitiveCommands removes commands that might contain sensitive information
func filterSensitiveCommands(commands []string) []string {
	sensitivePatterns := []string{
		"password", "passwd", "pass",
		"token", "secret", "key",
		"api_key", "apikey", "apisecret",
		"ssh-keygen", "ssh-add",
		"gpg", "openssl",
		"mysql -p", "psql -W",
		"docker login",
		"aws configure",
		"kubectl config",
	}

	var filtered []string
	for _, cmd := range commands {
		cmdLower := strings.ToLower(cmd)
		isSensitive := false

		for _, pattern := range sensitivePatterns {
			if strings.Contains(cmdLower, pattern) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			filtered = append(filtered, cmd)
		}
	}

	return filtered
}
