package cache

import (
	"os"
	"os/user"
	"runtime"
	"time"
)

// GetDefaultBootstrapContext returns a bootstrap context with fast defaults
// This is used when cache is missing or corrupt
func GetDefaultBootstrapContext() *BootstrapContext {
	ctx := &BootstrapContext{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Shell:        getDefaultShell(),
		CreatedAt:    time.Now(),
		Version:      CacheVersion,
	}

	// Try to get user info quickly
	if currentUser, err := user.Current(); err == nil {
		ctx.User = currentUser.Username
		ctx.HomeDir = currentUser.HomeDir
	} else {
		// Fallback to environment variables
		ctx.User = getDefaultUser()
		ctx.HomeDir = getDefaultHomeDir()
	}

	return ctx
}

// GetDefaultToolContext returns an empty tool context with safe defaults
// This is used when cache is missing or tool scanning has never occurred
func GetDefaultToolContext() *ToolContext {
	return &ToolContext{
		PackageManagers:  []string{},
		Languages:        []LanguageRuntime{},
		DevelopmentTools: []Tool{},
		ContainerTools:   []string{},
		CloudTools:       []string{},
		DatabaseTools:    []string{},
		NetworkTools:     []string{},
		SystemCommands:   []string{},
		Available:        make(map[string]bool),
		LastScan:         time.Time{}, // Zero time indicates never scanned
		ScanDurationMs:   0,
		Version:          CacheVersion,
	}
}

// getDefaultShell returns the default shell for the current OS
// This doesn't execute any processes, just returns OS-appropriate defaults
func getDefaultShell() string {
	// First try environment variable (fastest)
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Check for shell-specific environment variables
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	}
	if os.Getenv("BASH_VERSION") != "" {
		return "bash"
	}

	// OS-specific defaults
	switch runtime.GOOS {
	case "windows":
		// Check for PowerShell
		if os.Getenv("PSModulePath") != "" {
			return "powershell.exe"
		}
		// Check for common shell environment
		if comspec := os.Getenv("COMSPEC"); comspec != "" {
			return comspec
		}
		return "cmd.exe"
	case "darwin":
	// Going to make the assumption that macOS default shell is zsh. will change if needed.
		return "zsh"
	case "linux":
		return "bash"
	case "freebsd", "openbsd", "netbsd":
		return "sh"
	default:
		return "/bin/sh"
	}
}

// getDefaultUser returns the current username
func getDefaultUser() string {
	// Try environment variables (fastest)
	if username := os.Getenv("USER"); username != "" {
		return username
	}
	if username := os.Getenv("USERNAME"); username != "" { // Windows
		return username
	}
	if username := os.Getenv("LOGNAME"); username != "" { // Unix alternative
		return username
	}

	// Last resort: try user.Current() again (might succeed in different context)
	if currentUser, err := user.Current(); err == nil {
		return currentUser.Username
	}

	// Absolute fallback
	return "unknown"
}

// getDefaultHomeDir returns the user's home directory
func getDefaultHomeDir() string {
	// Try environment variables (fastest)
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" { // Windows
		return home
	}

	// Try user.Current() again
	if currentUser, err := user.Current(); err == nil {
		return currentUser.HomeDir
	}

	// Try os.UserHomeDir() as fallback
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}

	// OS-specific absolute fallbacks
	switch runtime.GOOS {
	case "windows":
		// Windows fallback
		if homedrive := os.Getenv("HOMEDRIVE"); homedrive != "" {
			if homepath := os.Getenv("HOMEPATH"); homepath != "" {
				return homedrive + homepath
			}
		}
		return "C:\\Users\\Default"
	default:
		// Unix-like systems
		return "/tmp"
	}
}

// GetCommonToolDefaults returns a tool context with commonly available tools
// This is NOT used by default, but can be useful for testing or offline mode
func GetCommonToolDefaults() *ToolContext {
	ctx := GetDefaultToolContext()

	// Add common tools that are likely to be available
	// These are conservative guesses and won't be used in normal operation
	switch runtime.GOOS {
	case "darwin", "linux", "freebsd", "openbsd", "netbsd":
		ctx.SystemCommands = []string{"ls", "cd", "pwd", "echo", "cat", "grep", "find"}
	case "windows":
		ctx.SystemCommands = []string{"dir", "cd", "echo", "type", "find"}
	}

	return ctx
}

// ValidateBootstrapContext checks if a bootstrap context has valid data
func ValidateBootstrapContext(ctx *BootstrapContext) error {
	if ctx == nil {
		return &CacheError{Op: "validate", Err: "bootstrap context is nil"}
	}
	if ctx.OS == "" {
		return &CacheError{Op: "validate", Err: "OS is empty"}
	}
	if ctx.Architecture == "" {
		return &CacheError{Op: "validate", Err: "architecture is empty"}
	}
	if ctx.Shell == "" {
		return &CacheError{Op: "validate", Err: "shell is empty"}
	}
	return nil
}

// ValidateToolContext checks if a tool context has valid structure
func ValidateToolContext(ctx *ToolContext) error {
	if ctx == nil {
		return &CacheError{Op: "validate", Err: "tool context is nil"}
	}
	if ctx.Available == nil {
		ctx.Available = make(map[string]bool)
	}
	return nil
}

// CacheError represents a cache operation error
type CacheError struct {
	Op  string // Operation that failed (e.g., "load", "save", "validate")
	Err string // Error description
}

func (e *CacheError) Error() string {
	return "cache " + e.Op + ": " + e.Err
}

// SanitizeBootstrapContext ensures a bootstrap context has valid values
// If any fields are empty, they're filled with defaults
func SanitizeBootstrapContext(ctx *BootstrapContext) *BootstrapContext {
	if ctx == nil {
		return GetDefaultBootstrapContext()
	}

	if ctx.OS == "" {
		ctx.OS = runtime.GOOS
	}
	if ctx.Architecture == "" {
		ctx.Architecture = runtime.GOARCH
	}
	if ctx.Shell == "" {
		ctx.Shell = getDefaultShell()
	}
	if ctx.User == "" {
		ctx.User = getDefaultUser()
	}
	if ctx.HomeDir == "" {
		ctx.HomeDir = getDefaultHomeDir()
	}
	if ctx.Version == 0 {
		ctx.Version = CacheVersion
	}

	return ctx
}

// SanitizeToolContext ensures a tool context has valid structure
func SanitizeToolContext(ctx *ToolContext) *ToolContext {
	if ctx == nil {
		return GetDefaultToolContext()
	}

	// Ensure maps are initialized
	if ctx.Available == nil {
		ctx.Available = make(map[string]bool)
	}

	// Ensure slices are initialized (not nil)
	if ctx.PackageManagers == nil {
		ctx.PackageManagers = []string{}
	}
	if ctx.Languages == nil {
		ctx.Languages = []LanguageRuntime{}
	}
	if ctx.DevelopmentTools == nil {
		ctx.DevelopmentTools = []Tool{}
	}
	if ctx.ContainerTools == nil {
		ctx.ContainerTools = []string{}
	}
	if ctx.CloudTools == nil {
		ctx.CloudTools = []string{}
	}
	if ctx.DatabaseTools == nil {
		ctx.DatabaseTools = []string{}
	}
	if ctx.NetworkTools == nil {
		ctx.NetworkTools = []string{}
	}
	if ctx.SystemCommands == nil {
		ctx.SystemCommands = []string{}
	}
	if ctx.Version == 0 {
		ctx.Version = CacheVersion
	}

	return ctx
}
