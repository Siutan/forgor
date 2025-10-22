package cache

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheVersion is incremented when cache format changes
	CacheVersion = 1

	// DefaultCacheDir is the default cache directory relative to home
	DefaultCacheDir = ".cache/forgor"

	// ToolCacheMaxAge is the default maximum age for tool cache before refresh
	ToolCacheMaxAge = 24 * time.Hour

	// BootstrapCacheFile is the filename for bootstrap cache
	BootstrapCacheFile = "bootstrap.json"

	// ToolCacheFile is the filename for tool cache
	ToolCacheFile = "tools.json"

	// LockFile is the filename for concurrent access control
	LockFile = ".lock"
)

// getCacheDir returns the cache directory path
// Priority: FORGOR_CACHE_DIR env var > default ~/.cache/forgor
func getCacheDir() string {
	if dir := os.Getenv("FORGOR_CACHE_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to temp directory if home is not accessible
		return filepath.Join(os.TempDir(), "forgor_cache")
	}

	return filepath.Join(home, DefaultCacheDir)
}

// getBootstrapCachePath returns the full path to the bootstrap cache file
func getBootstrapCachePath() string {
	return filepath.Join(getCacheDir(), BootstrapCacheFile)
}

// getToolCachePath returns the full path to the tool cache file
func getToolCachePath() string {
	return filepath.Join(getCacheDir(), ToolCacheFile)
}

// getLockFilePath returns the full path to the lock file
func getLockFilePath() string {
	return filepath.Join(getCacheDir(), LockFile)
}

// ensureCacheDir creates the cache directory if it doesn't exist
func ensureCacheDir() error {
	dir := getCacheDir()
	return os.MkdirAll(dir, 0755)
}

// ClearCache removes all cache files
func ClearCache() error {
	dir := getCacheDir()
	return os.RemoveAll(dir)
}

// GetCacheInfo returns information about the cache state
func GetCacheInfo() map[string]any {
	info := make(map[string]any)

	dir := getCacheDir()
	bootstrapPath := getBootstrapCachePath()
	toolPath := getToolCachePath()

	info["cache_dir"] = dir
	info["cache_version"] = CacheVersion

	// Bootstrap cache info
	if stat, err := os.Stat(bootstrapPath); err == nil {
		info["bootstrap_exists"] = true
		info["bootstrap_size"] = stat.Size()
		info["bootstrap_modified"] = stat.ModTime()
		info["bootstrap_age"] = time.Since(stat.ModTime())
	} else {
		info["bootstrap_exists"] = false
	}

	// Tool cache info
	if stat, err := os.Stat(toolPath); err == nil {
		info["tools_exists"] = true
		info["tools_size"] = stat.Size()
		info["tools_modified"] = stat.ModTime()
		info["tools_age"] = time.Since(stat.ModTime())
	} else {
		info["tools_exists"] = false
	}

	return info
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getFileAge returns the age of a file, or zero if it doesn't exist
func getFileAge(path string) time.Duration {
	stat, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return time.Since(stat.ModTime())
}