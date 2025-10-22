package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"forgor/internal/cache"
)

// SystemContext represents comprehensive system information
type SystemContext struct {
	OS               string            `json:"os"`
	Shell            string            `json:"shell"`
	Architecture     string            `json:"architecture"`
	WorkingDirectory string            `json:"working_directory"`
	User             string            `json:"user"`
	HomeDirectory    string            `json:"home_directory"`
	Tools            ToolContext       `json:"tools"`
	Environment      map[string]string `json:"environment"`
}

// ToolContext is an alias for cache.ToolContext for backward compatibility
type ToolContext = cache.ToolContext

// LanguageRuntime is an alias for cache.LanguageRuntime for backward compatibility
type LanguageRuntime = cache.LanguageRuntime

// Tool is an alias for cache.Tool for backward compatibility
type Tool = cache.Tool

var (
	systemContextCache *SystemContext
	contextCacheMutex  sync.RWMutex
	cacheExpiration    = 20 * time.Minute
	cacheTimestamp     time.Time

	// Background refresh control
	refreshInProgress        int32 // atomic flag
	backgroundRefreshEnabled = true
	gracePeriod              = 1 * time.Minute // Grace period to use stale cache while refreshing

	// Persistent cache settings
	cacheDir      string
	cacheFile     string
	lockFile      string
	initCacheOnce sync.Once
)

// CachedSystemContext represents the persistent cache structure
type CachedSystemContext struct {
	Context   *SystemContext `json:"context"`
	Timestamp time.Time      `json:"timestamp"`
	Version   string         `json:"version"`
}

// CacheInfo represents information about the persistent cache
type CacheInfo struct {
	CacheDir    string    `json:"cache_dir"`
	FilePath    string    `json:"file_path"`
	LockFile    string    `json:"lock_file"`
	FileExists  bool      `json:"file_exists"`
	FileSize    int64     `json:"file_size"`
	FileModTime time.Time `json:"file_mod_time"`
}

// initPersistentCache initializes the persistent cache directory and file paths
func initPersistentCache() error {
	var err error
	initCacheOnce.Do(func() {
		// Get user cache directory
		var userCacheDir string
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			userCacheDir = xdgCache
		} else {
			var homeDir string
			if currentUser, userErr := user.Current(); userErr == nil {
				homeDir = currentUser.HomeDir
			} else {
				homeDir = os.Getenv("HOME")
			}
			userCacheDir = filepath.Join(homeDir, ".cache")
		}

		cacheDir = filepath.Join(userCacheDir, "forgor")
		cacheFile = filepath.Join(cacheDir, "system-context.json")
		lockFile = filepath.Join(cacheDir, "system-context.lock")

		// Create cache directory if it doesn't exist
		err = os.MkdirAll(cacheDir, 0755)
	})
	return err
}

// loadPersistentCache loads the system context from persistent cache
func loadPersistentCache() (*SystemContext, error) {
	if err := initPersistentCache(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil, nil // No cache file
	}

	// Acquire read lock
	lockFd, err := acquireFileLock(lockFile, false)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer releaseFileLock(lockFd)

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse cache
	var cached CachedSystemContext
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	// Validate cache version and age
	if cached.Version != "1.0" {
		return nil, fmt.Errorf("cache version mismatch")
	}

	age := time.Since(cached.Timestamp)
	if age > cacheExpiration+gracePeriod {
		return nil, fmt.Errorf("cache too old: %v", age)
	}

	// Update in-memory cache
	contextCacheMutex.Lock()
	systemContextCache = cached.Context
	cacheTimestamp = cached.Timestamp
	contextCacheMutex.Unlock()

	return cached.Context, nil
}

// savePersistentCache saves the system context to persistent cache
func savePersistentCache(context *SystemContext) error {
	if err := initPersistentCache(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Acquire write lock
	lockFd, err := acquireFileLock(lockFile, true)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer releaseFileLock(lockFd)

	// Create cache structure
	cached := CachedSystemContext{
		Context:   context,
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to temporary file first
	tempFile := cacheFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp cache: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, cacheFile); err != nil {
		os.Remove(tempFile) // Cleanup temp file
		return fmt.Errorf("failed to update cache: %w", err)
	}

	return nil
}

// acquireFileLock acquires a file lock (exclusive if write=true, shared if write=false)
func acquireFileLock(lockFile string, write bool) (*os.File, error) {
	// Create lock file if it doesn't exist
	lockFd, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// Try to acquire lock non-blocking with timeout/retries.
	// On platforms where platformLockFile is a no-op, this will succeed immediately.
	for i := 0; i < 50; i++ { // Try for up to 5 seconds
		if err := platformLockFile(lockFd, write, true); err == nil {
			return lockFd, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	lockFd.Close()
	return nil, fmt.Errorf("timeout acquiring file lock")
}

// releaseFileLock releases a file lock
func releaseFileLock(lockFd *os.File) {
	if lockFd != nil {
		_ = platformUnlockFile(lockFd)
		lockFd.Close()
	}
}

// GetSystemContext returns comprehensive system information with persistent caching
func GetSystemContext() *SystemContext {
	verbose := isVerboseMode()

	// TIER 1: Load bootstrap context (never blocks)
	bootstrap := cache.LoadBootstrapCacheOrDefault()

	// TIER 2: Load tool context (never blocks, stale is OK)
	tools := cache.LoadToolCacheOrEmpty(cache.ToolCacheMaxAge)

	// Check if we should trigger background refresh
	if tools.IsStale(cache.ToolCacheMaxAge) || tools.NeedsScan() {
		if verbose {
			if tools.NeedsScan() {
				fmt.Fprintf(os.Stderr, "🔍 Tool cache never scanned, scheduling background scan...\n")
			} else {
				age := tools.GetAge()
				fmt.Fprintf(os.Stderr, "⏰ Tool cache is stale (age: %v), scheduling background refresh...\n", age.Round(time.Hour))
			}
		}
		// Trigger background refresh (non-blocking)
		go cache.MaybeRefresh()
	}

	// TIER 3: Get dynamic context (must be fresh)
	wd, _ := os.Getwd()
	env := getRelevantEnvironment()

	// Combine all tiers
	ctx := &SystemContext{
		OS:               bootstrap.OS,
		Shell:            bootstrap.Shell,
		Architecture:     bootstrap.Architecture,
		User:             bootstrap.User,
		HomeDirectory:    bootstrap.HomeDir,
		WorkingDirectory: wd,
		Tools:            *tools,
		Environment:      env,
	}

	if verbose {
		if !tools.IsEmpty() {
			fmt.Fprintf(os.Stderr, "   📦 %s\n", tools.GetSummary())
		}
	}

	return ctx
}



// RefreshSystemContext forces a refresh of the system context cache
func RefreshSystemContext() error {
	verbose := isVerboseMode()

	if verbose {
		fmt.Fprintf(os.Stderr, "🔄 Refreshing system context...\n")
	}

	// Refresh bootstrap cache (fast)
	if err := cache.RefreshBootstrapCache(); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "⚠️  Bootstrap refresh failed: %v\n", err)
		}
	} else if verbose {
		fmt.Fprintf(os.Stderr, "✅ Bootstrap cache refreshed\n")
	}

	// Refresh tool cache (slow - this is the main operation)
	if err := cache.ForceRefresh(); err != nil {
		return fmt.Errorf("tool refresh failed: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✅ System context refreshed\n")
	}

	return nil
}

// RefreshSystemContextBackground triggers a background refresh without blocking
func RefreshSystemContextBackground() {
	cache.MaybeRefresh()
}

// IsRefreshInProgress checks if a background refresh is currently running
func IsRefreshInProgress() bool {
	return cache.IsRefreshInProgress()
}

// SetBackgroundRefreshEnabled enables or disables background refreshing
func SetBackgroundRefreshEnabled(enabled bool) {
	cache.GetGlobalScheduler().SetEnabled(enabled)
}

// GetCacheAge returns how old the caches are (bootstrap, tools)
func GetCacheAge() (bootstrap time.Duration, tools time.Duration) {
	info := cache.GetCacheInfo()
	
	if age, ok := info["bootstrap_age"].(time.Duration); ok {
		bootstrap = age
	}
	
	if age, ok := info["tools_age"].(time.Duration); ok {
		tools = age
	}
	
	return
}



// getLanguageVersion attempts to get the version of a language runtime with timeout
func getLanguageVersion(language, command string) string {
	versionArgs := map[string][]string{
		"python": {"--version"},
		"node":   {"--version"},
		"go":     {"version"},
		"java":   {"-version"},
		"ruby":   {"--version"},
		"php":    {"--version"},
		"rustc":  {"--version"},
		"dotnet": {"--version"},
		"swift":  {"--version"},
		"dart":   {"--version"},
		"julia":  {"--version"},
		"elixir": {"--version"},
		"nim":    {"--version"},
		"zig":    {"version"},
	}

	args, exists := versionArgs[language]
	if !exists {
		args = []string{"--version"}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	// Extract version from output (simplified)
	version := strings.TrimSpace(string(output))
	lines := strings.Split(version, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return "unknown"
}

// getToolVersion attempts to get the version of a tool with timeout
func getToolVersion(tool string) string {
	// Try common version flags
	versionFlags := []string{"--version", "-version", "-V", "-v", "version"}

	for _, flag := range versionFlags {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cmd := exec.CommandContext(ctx, tool, flag)
		output, err := cmd.CombinedOutput()
		cancel() // Clean up immediately after each attempt

		if err == nil {
			version := strings.TrimSpace(string(output))
			lines := strings.Split(version, "\n")
			if len(lines) > 0 && lines[0] != "" {
				return strings.TrimSpace(lines[0])
			}
		}
	}

	return "unknown"
}

// TODO: leaving this here for now but i should probably remove it
// getHomeDirectory returns the user's home directory
func getHomeDirectory() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return "."
}

// getRelevantEnvironment returns environment variables relevant for command generation
func getRelevantEnvironment() map[string]string {
	env := make(map[string]string)

	relevantVars := []string{
		"PATH", "USER", "HOME", "SHELL", "TERM", "LANG", "LC_ALL",
		"EDITOR", "VISUAL", "PAGER", "BROWSER",
		"GOPATH", "GOROOT", "JAVA_HOME", "PYTHON_PATH", "NODE_PATH",
		"VIRTUAL_ENV", "CONDA_DEFAULT_ENV",
		"DOCKER_HOST", "KUBECONFIG", "AWS_PROFILE", "AZURE_SUBSCRIPTION_ID",
	}

	for _, varName := range relevantVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}

	return env
}

// GetToolContextSummary returns a concise summary of available tools for prompts
func GetToolContextSummary() string {
	context := GetSystemContext()
	summary := []string{}

	if len(context.Tools.PackageManagers) > 0 {
		summary = append(summary, "Package managers: "+strings.Join(context.Tools.PackageManagers, ", "))
	}

	if len(context.Tools.Languages) > 0 {
		langs := make([]string, len(context.Tools.Languages))
		for i, lang := range context.Tools.Languages {
			langs[i] = lang.Name
		}
		summary = append(summary, "Languages: "+strings.Join(langs, ", "))
	}

	if len(context.Tools.ContainerTools) > 0 {
		summary = append(summary, "Containers: "+strings.Join(context.Tools.ContainerTools, ", "))
	}

	if len(context.Tools.CloudTools) > 0 {
		summary = append(summary, "Cloud tools: "+strings.Join(context.Tools.CloudTools, ", "))
	}

	if len(summary) == 0 {
		return "Standard system commands available"
	}

	return strings.Join(summary, "; ")
}

// IsToolAvailable checks if a specific tool is available
// IsToolAvailable checks if a tool is available in the context
func IsToolAvailable(tool string) bool {
	tools := cache.LoadToolCacheOrEmpty(cache.ToolCacheMaxAge)
	return tools.IsToolAvailable(tool)
}

// isVerboseMode checks if verbose mode is enabled from environment or context
func isVerboseMode() bool {
	// Check environment variable
	return os.Getenv("FORGOR_VERBOSE") == "true"
}

// GetCacheInfo returns information about the persistent cache
func GetCacheInfo() map[string]interface{} {
	info := cache.GetCacheInfo()
	
	// Add scheduler info
	scheduler := cache.GetGlobalScheduler()
	info["refresh_in_progress"] = scheduler.IsRefreshInProgress()
	info["refresh_status"] = scheduler.GetRefreshStatus()
	info["refresh_interval"] = scheduler.GetInterval().String()
	info["needs_refresh"] = scheduler.NeedsRefresh()
	
	return info
}

// ClearPersistentCache removes the persistent cache file
func ClearPersistentCache() error {
	return cache.ClearCache()
}

// InitializeCache initializes the cache system with configuration
func InitializeCache(refreshInterval time.Duration, enableBackgroundRefresh bool) {
	cache.InitGlobalScheduler(refreshInterval)
	
	if !enableBackgroundRefresh {
		cache.GetGlobalScheduler().SetEnabled(false)
	}
	
	// Check if bootstrap cache exists, if not create it
	if !cache.IsBootstrapCacheValid() {
		bootstrap := cache.BuildBootstrapContext()
		if err := cache.SaveBootstrapCache(bootstrap); err != nil && isVerboseMode() {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create bootstrap cache: %v\n", err)
		}
	}
	
	// Check if tool cache exists, if not schedule background scan
	if !cache.IsToolCacheValid() {
		if isVerboseMode() {
			fmt.Fprintf(os.Stderr, "Tool cache not found, scheduling initial scan...\n")
		}
		go cache.MaybeRefresh()
	}
}
