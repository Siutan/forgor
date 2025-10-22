package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ToolContext contains detected system tools and capabilities
// This is Tier 2 cache, stale data is acceptable
type ToolContext struct {
	PackageManagers  []string          `json:"package_managers"`
	Languages        []LanguageRuntime `json:"languages"`
	DevelopmentTools []Tool            `json:"development_tools"`
	ContainerTools   []string          `json:"container_tools"`
	CloudTools       []string          `json:"cloud_tools"`
	DatabaseTools    []string          `json:"database_tools"`
	NetworkTools     []string          `json:"network_tools"`
	SystemCommands   []string          `json:"system_commands"`
	Available        map[string]bool   `json:"available"`
	LastScan         time.Time         `json:"last_scan"`
	ScanDurationMs   int64             `json:"scan_duration_ms"`
	Version          int               `json:"version"`
}

// LanguageRuntime represents a programming language runtime
type LanguageRuntime struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// Tool represents an available development tool
type Tool struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// LoadToolCache loads the tool detection cache from disk
// Returns nil if cache doesn't exist or is invalid
func LoadToolCache() (*ToolContext, error) {
	path := getToolCachePath()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool cache: %w", err)
	}

	var ctx ToolContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse tool cache: %w", err)
	}

	// Validate version
	if ctx.Version != CacheVersion {
		return nil, fmt.Errorf("cache version mismatch: got %d, expected %d", ctx.Version, CacheVersion)
	}

	return &ctx, nil
}

// SaveToolCache saves the tool detection cache to disk
func SaveToolCache(ctx *ToolContext) error {
	if ctx == nil {
		return fmt.Errorf("cannot save nil tool context")
	}

	// Ensure cache directory exists
	if err := ensureCacheDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Set metadata
	ctx.Version = CacheVersion
	if ctx.LastScan.IsZero() {
		ctx.LastScan = time.Now()
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tool context: %w", err)
	}

	path := getToolCachePath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tool cache: %w", err)
	}

	return nil
}

// LoadToolCacheOrEmpty loads tool cache or returns empty context
// Returns stale cache if available, empty if not, prefer this method to fetch tool cache
func LoadToolCacheOrEmpty(maxAge time.Duration) *ToolContext {
	ctx, err := LoadToolCache()

	if err != nil {
		// Cache doesn't exist or is corrupt - return empty
		if os.Getenv("FORGOR_VERBOSE") == "true" {
			fmt.Fprintf(os.Stderr, "Tool cache miss: %v\n", err)
		}
		return GetMinimalToolContext()
	}

	// Check if stale and trigger background refresh if needed
	if ctx.IsStale(maxAge) {
		if os.Getenv("FORGOR_VERBOSE") == "true" {
			age := time.Since(ctx.LastScan)
			fmt.Fprintf(os.Stderr, "Tool cache stale (age: %v), will refresh in background\n", age)
		}
		// Note: Background refresh is handled by scheduler, not here
		// We just return the stale cache - it's still useful
	}

	return ctx
}

// IsStale checks if the tool cache needs refresh based on age
func (tc *ToolContext) IsStale(maxAge time.Duration) bool {
	if tc == nil {
		return true
	}
	if tc.LastScan.IsZero() {
		return true
	}
	return time.Since(tc.LastScan) > maxAge
}

// GetMinimalToolContext returns an empty but valid tool context
// This is used when cache is missing or corrupt
func GetMinimalToolContext() *ToolContext {
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
		Version:          CacheVersion,
	}
}

// IsEmpty checks if the tool context has any detected tools
func (tc *ToolContext) IsEmpty() bool {
	if tc == nil {
		return true
	}
	return len(tc.PackageManagers) == 0 &&
		len(tc.Languages) == 0 &&
		len(tc.DevelopmentTools) == 0 &&
		len(tc.ContainerTools) == 0 &&
		len(tc.CloudTools) == 0 &&
		len(tc.DatabaseTools) == 0 &&
		len(tc.NetworkTools) == 0 &&
		len(tc.SystemCommands) == 0
}

// GetAge returns how long ago the tool cache was last scanned
func (tc *ToolContext) GetAge() time.Duration {
	if tc == nil || tc.LastScan.IsZero() {
		return 0
	}
	return time.Since(tc.LastScan)
}

// IsToolAvailable checks if a specific tool is available
func (tc *ToolContext) IsToolAvailable(tool string) bool {
	if tc == nil || tc.Available == nil {
		return false
	}
	available, exists := tc.Available[tool]
	return exists && available
}

// GetToolCount returns the total number of detected tools
func (tc *ToolContext) GetToolCount() int {
	if tc == nil {
		return 0
	}

	count := 0
	count += len(tc.PackageManagers)
	count += len(tc.Languages)
	count += len(tc.DevelopmentTools)
	count += len(tc.ContainerTools)
	count += len(tc.CloudTools)
	count += len(tc.DatabaseTools)
	count += len(tc.NetworkTools)
	count += len(tc.SystemCommands)

	return count
}

// GetSummary returns a human-readable summary of detected tools
func (tc *ToolContext) GetSummary() string {
	if tc == nil || tc.IsEmpty() {
		return "No tools detected"
	}

	summary := fmt.Sprintf("%d tools detected", tc.GetToolCount())

	if !tc.LastScan.IsZero() {
		age := time.Since(tc.LastScan)
		if age < time.Minute {
			summary += " (just now)"
		} else if age < time.Hour {
			summary += fmt.Sprintf(" (%d minutes ago)", int(age.Minutes()))
		} else if age < 24*time.Hour {
			summary += fmt.Sprintf(" (%d hours ago)", int(age.Hours()))
		} else {
			summary += fmt.Sprintf(" (%d days ago)", int(age.Hours()/24))
		}
	}

	return summary
}

// NeedsScan checks if the cache has never been scanned
func (tc *ToolContext) NeedsScan() bool {
	if tc == nil {
		return true
	}
	return tc.LastScan.IsZero()
}

// IsToolCacheValid checks if the tool cache exists and is valid
func IsToolCacheValid() bool {
	_, err := LoadToolCache()
	return err == nil
}
