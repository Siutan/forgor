package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"time"
)

// BootstrapContext contains fast-loading system basics
type BootstrapContext struct {
	OS           string    `json:"os"`
	Architecture string    `json:"arch"`
	Shell        string    `json:"shell"`
	User         string    `json:"user"`
	HomeDir      string    `json:"home_dir"`
	CreatedAt    time.Time `json:"created_at"`
	Version      int       `json:"version"`
}

// LoadBootstrapCache loads the bootstrap context from disk
// Returns nil if cache doesn't exist or is invalid
func LoadBootstrapCache() (*BootstrapContext, error) {
	path := getBootstrapCachePath()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read bootstrap cache: %w", err)
	}

	var ctx BootstrapContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse bootstrap cache: %w", err)
	}

	// Validate version
	if ctx.Version != CacheVersion {
		return nil, fmt.Errorf("cache version mismatch: got %d, expected %d", ctx.Version, CacheVersion)
	}

	return &ctx, nil
}

// SaveBootstrapCache saves the bootstrap context to disk
func SaveBootstrapCache(ctx *BootstrapContext) error {
	if ctx == nil {
		return fmt.Errorf("cannot save nil bootstrap context")
	}

	// Ensure cache directory exists
	if err := ensureCacheDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Set metadata
	ctx.Version = CacheVersion
	if ctx.CreatedAt.IsZero() {
		ctx.CreatedAt = time.Now()
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bootstrap context: %w", err)
	}

	path := getBootstrapCachePath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write bootstrap cache: %w", err)
	}

	return nil
}

// BuildBootstrapContext creates a new bootstrap context using fast operations only
func BuildBootstrapContext() *BootstrapContext {
	ctx := &BootstrapContext{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Shell:        detectShellFast(),
		CreatedAt:    time.Now(),
		Version:      CacheVersion,
	}

	// Try to get user info (usually fast, but can fail)
	if currentUser, err := user.Current(); err == nil {
		ctx.User = currentUser.Username
		ctx.HomeDir = currentUser.HomeDir
	} else {
		// Fallback to environment variables
		ctx.User = os.Getenv("USER")
		if ctx.User == "" {
			ctx.User = os.Getenv("USERNAME") // Windows
		}
		ctx.HomeDir = os.Getenv("HOME")
		if ctx.HomeDir == "" {
			ctx.HomeDir = os.Getenv("USERPROFILE") // Windows
		}
	}

	return ctx
}

// LoadBootstrapCacheOrDefault loads cache or returns defaults
func LoadBootstrapCacheOrDefault() *BootstrapContext {
	if ctx, err := LoadBootstrapCache(); err == nil {
		return ctx
	}

	ctx := GetDefaultBootstrapContext()

	go func() {
		if err := SaveBootstrapCache(ctx); err != nil {
			// Only log if verbose mode
			if os.Getenv("FORGOR_VERBOSE") == "true" {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save bootstrap cache: %v\n", err)
			}
		}
	}()

	return ctx
}



// detectShellFast detects the current shell using only environment variables
func detectShellFast() string {
	// Check SHELL environment variable first (Unix/Linux/macOS)
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Check for common shell indicators
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	}
	if os.Getenv("BASH_VERSION") != "" {
		return "bash"
	}

	// Windows shells
	if runtime.GOOS == "windows" {
		if os.Getenv("PSModulePath") != "" {
			return "powershell"
		}
		return "cmd.exe"
	}

	// Generic fallback based on OS
	return getShellFromEnv()
}

// getShellFromEnv returns a shell based on environment and OS
// This is a pure fallback that doesn't depend on any external state
func getShellFromEnv() string {
	// Priority: SHELL env var > OS-specific default
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// OS-specific defaults
	switch runtime.GOOS {
	case "windows":
		if os.Getenv("PSModulePath") != "" {
			return "powershell.exe"
		}
		return "cmd.exe"
	case "darwin":
		// Going to make the assumption that macOS default shell is zsh. will change if needed. 
		return "zsh"
	case "linux", "freebsd", "openbsd", "netbsd":
		return "bash"
	default:
		return "/bin/sh"
	}
}

// RefreshBootstrapCache forces a refresh of the bootstrap cache
// This is rarely needed since bootstrap data usually never changes... "usually"
func RefreshBootstrapCache() error {
	ctx := BuildBootstrapContext()
	return SaveBootstrapCache(ctx)
}

// IsBootstrapCacheValid checks if the bootstrap cache exists and is valid
func IsBootstrapCacheValid() bool {
	_, err := LoadBootstrapCache()
	return err == nil
}