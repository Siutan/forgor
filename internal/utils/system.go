package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
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

// ToolContext represents available tools and capabilities
type ToolContext struct {
	PackageManagers  []string          `json:"package_managers"`
	Languages        []LanguageRuntime `json:"languages"`
	DevelopmentTools []Tool            `json:"development_tools"`
	SystemCommands   []string          `json:"system_commands"`
	ContainerTools   []string          `json:"container_tools"`
	CloudTools       []string          `json:"cloud_tools"`
	DatabaseTools    []string          `json:"database_tools"`
	NetworkTools     []string          `json:"network_tools"`
	Available        map[string]bool   `json:"available"`
	LastChecked      time.Time         `json:"last_checked"`
}

// LanguageRuntime represents a programming language runtime
type LanguageRuntime struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// Tool represents an available tool or application
type Tool struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

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

	// Determine lock type
	lockType := syscall.LOCK_SH // Shared lock for read
	if write {
		lockType = syscall.LOCK_EX // Exclusive lock for write
	}

	// Try to acquire lock with timeout
	lockType |= syscall.LOCK_NB // Non-blocking

	for i := 0; i < 50; i++ { // Try for up to 5 seconds
		if err := syscall.Flock(int(lockFd.Fd()), lockType); err == nil {
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
		syscall.Flock(int(lockFd.Fd()), syscall.LOCK_UN)
		lockFd.Close()
	}
}

// GetSystemContext returns comprehensive system information with persistent caching
func GetSystemContext() *SystemContext {
	verbose := isVerboseMode()

	// First check in-memory cache
	contextCacheMutex.RLock()
	if systemContextCache != nil && time.Since(cacheTimestamp) < cacheExpiration {
		defer contextCacheMutex.RUnlock()
		return systemContextCache
	}
	contextCacheMutex.RUnlock()

	// Try to load from persistent cache
	if cached, err := loadPersistentCache(); err == nil && cached != nil {
		age := time.Since(cacheTimestamp)
		if verbose {
			fmt.Printf("📁 Loaded system context from cache (age: %v)\n", age)
		}

		// Check if we should trigger background refresh
		if age > cacheExpiration && backgroundRefreshEnabled {
			if atomic.CompareAndSwapInt32(&refreshInProgress, 0, 1) {
				go func() {
					defer atomic.StoreInt32(&refreshInProgress, 0)
					if verbose {
						fmt.Printf("🔄 Refreshing system context in background...\n")
					}
					refreshSystemContextInternal(false) // silent refresh
				}()
			}
		}

		return cached
	}

	// No valid cache - must refresh synchronously
	if verbose {
		fmt.Printf("🔍 Building system context (no valid cache found)...\n")
	}

	return refreshSystemContextInternal(verbose)
}

// refreshSystemContextInternal performs the actual cache refresh
func refreshSystemContextInternal(verbose bool) *SystemContext {
	contextCacheMutex.Lock()
	defer contextCacheMutex.Unlock()

	// Double-check after acquiring write lock
	if systemContextCache != nil && time.Since(cacheTimestamp) < cacheExpiration {
		return systemContextCache
	}

	var timer *Timer
	if verbose {
		timer = NewTimer("System Context", verbose)
		defer timer.PrintSummary()
	}

	// Get user information
	var userStep *StepTimer
	if verbose && timer != nil {
		userStep = timer.StartStep("User Detection")
	}

	currentUser, err := user.Current()
	var username, homeDir string
	if err == nil {
		username = currentUser.Username
		homeDir = currentUser.HomeDir
	} else {
		username = os.Getenv("USER")
		homeDir = os.Getenv("HOME")
	}

	if userStep != nil {
		userStep.End()
	}

	// Get working directory
	var dirStep *StepTimer
	if verbose && timer != nil {
		dirStep = timer.StartStep("Directory Detection")
	}

	wd, _ := os.Getwd()

	if dirStep != nil {
		dirStep.End()
	}

	// Detect tools
	var toolsStep *StepTimer
	if verbose && timer != nil {
		toolsStep = timer.StartStep("Tool Detection")
	}

	tools := gatherToolContext()

	if toolsStep != nil {
		toolsStep.End()
	}

	// Build the context
	var buildStep *StepTimer
	if verbose && timer != nil {
		buildStep = timer.StartStep("Context Assembly")
	}

	systemContextCache = &SystemContext{
		OS:               runtime.GOOS,
		Architecture:     runtime.GOARCH,
		Shell:            GetCurrentShell(),
		User:             username,
		HomeDirectory:    homeDir,
		WorkingDirectory: wd,
		Environment:      getRelevantEnvironment(),
		Tools:            tools,
	}

	if buildStep != nil {
		buildStep.End()
	}

	cacheTimestamp = time.Now()

	// Save to persistent cache
	var saveStep *StepTimer
	if verbose && timer != nil {
		saveStep = timer.StartStep("Cache Save")
	}

	if err := savePersistentCache(systemContextCache); err != nil {
		if verbose {
			fmt.Printf("⚠️  Failed to save cache: %v\n", err)
		}
	} else if verbose {
		fmt.Printf("💾 Saved system context to persistent cache\n")
	}

	if saveStep != nil {
		saveStep.End()
	}

	return systemContextCache
}

// RefreshSystemContext forces a refresh of the system context cache
func RefreshSystemContext() *SystemContext {
	if isVerboseMode() {
		fmt.Printf("🔄 Forcing system context refresh...\n")
	}

	contextCacheMutex.Lock()
	// Force cache expiry
	cacheTimestamp = time.Time{}
	systemContextCache = nil
	contextCacheMutex.Unlock()

	return refreshSystemContextInternal(isVerboseMode())
}

// RefreshSystemContextBackground triggers a background refresh without blocking
func RefreshSystemContextBackground() {
	if atomic.CompareAndSwapInt32(&refreshInProgress, 0, 1) {
		go func() {
			defer atomic.StoreInt32(&refreshInProgress, 0)
			if isVerboseMode() {
				fmt.Printf("🔄 Starting background system context refresh...\n")
			}
			refreshSystemContextInternal(false)
			if isVerboseMode() {
				fmt.Printf("✅ Background system context refresh completed\n")
			}
		}()
	} else if isVerboseMode() {
		fmt.Printf("⏳ Background refresh already in progress\n")
	}
}

// IsRefreshInProgress returns true if a background refresh is currently running
func IsRefreshInProgress() bool {
	return atomic.LoadInt32(&refreshInProgress) == 1
}

// SetBackgroundRefreshEnabled enables or disables background refreshing
func SetBackgroundRefreshEnabled(enabled bool) {
	backgroundRefreshEnabled = enabled
}

// GetCacheAge returns how old the current cache is
func GetCacheAge() time.Duration {
	// First check in-memory cache without holding lock during external calls
	contextCacheMutex.RLock()
	hasInMemoryCache := systemContextCache != nil && !cacheTimestamp.IsZero()
	var memoryAge time.Duration
	if hasInMemoryCache {
		memoryAge = time.Since(cacheTimestamp)
	}
	contextCacheMutex.RUnlock()

	if hasInMemoryCache {
		return memoryAge
	}

	// No in-memory cache, try to check persistent cache without loading it
	if err := initPersistentCache(); err != nil {
		return 0
	}

	// Check if cache file exists and get its age
	if _, err := os.Stat(cacheFile); err == nil {
		// Read the cache file to get timestamp without loading into memory
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			return 0
		}

		var cached CachedSystemContext
		if err := json.Unmarshal(data, &cached); err != nil {
			return 0
		}

		return time.Since(cached.Timestamp)
	}

	return 0
}

// gatherToolContext detects available tools and capabilities
func gatherToolContext() ToolContext {
	tools := ToolContext{
		Available:   make(map[string]bool),
		LastChecked: time.Now(),
	}

	// Detect package managers
	tools.PackageManagers = detectPackageManagers()

	// Detect programming languages
	tools.Languages = detectLanguageRuntimes()

	// Detect development tools
	tools.DevelopmentTools = detectDevelopmentTools()

	// Detect system commands
	tools.SystemCommands = detectSystemCommands()

	// Detect container tools
	tools.ContainerTools = detectContainerTools()

	// Detect cloud tools
	tools.CloudTools = detectCloudTools()

	// Detect database tools
	tools.DatabaseTools = detectDatabaseTools()

	// Detect network tools
	tools.NetworkTools = detectNetworkTools()

	// Build availability map
	buildAvailabilityMap(&tools)

	return tools
}

// detectPackageManagers identifies available package managers
func detectPackageManagers() []string {
	managers := []string{}
	candidates := []string{
		"brew", "apt", "apt-get", "yum", "dnf", "pacman", "zypper",
		"npm", "pip", "pip3", "gem", "cargo", "go", "composer",
		"yarn", "bun", "pnpm", "bundle", "poetry", "pipenv",
	}

	for _, manager := range candidates {
		if isCommandAvailable(manager) {
			managers = append(managers, manager)
		}
	}

	return managers
}

// detectLanguageRuntimes identifies available programming language runtimes
func detectLanguageRuntimes() []LanguageRuntime {
	runtimes := []LanguageRuntime{}

	languages := map[string][]string{
		"python":  {"python", "python3"},
		"node":    {"node"},
		"go":      {"go"},
		"java":    {"java"},
		"ruby":    {"ruby"},
		"php":     {"php"},
		"rust":    {"rustc"},
		"kotlin":  {"kotlinc"},
		"scala":   {"scala"},
		"swift":   {"swift"},
		"dart":    {"dart"},
		"dotnet":  {"dotnet"},
		"perl":    {"perl"},
		"lua":     {"lua"},
		"r":       {"R", "Rscript"},
		"julia":   {"julia"},
		"elixir":  {"elixir"},
		"erlang":  {"erl"},
		"haskell": {"ghc"},
		"clojure": {"clojure"},
		"nim":     {"nim"},
		"zig":     {"zig"},
	}

	for lang, commands := range languages {
		for _, cmd := range commands {
			if path, err := exec.LookPath(cmd); err == nil {
				version := getLanguageVersion(lang, cmd)
				runtimes = append(runtimes, LanguageRuntime{
					Name:    lang,
					Version: version,
					Path:    path,
				})
				break // Only add one runtime per language
			}
		}
	}

	return runtimes
}

// detectDevelopmentTools identifies available development tools
func detectDevelopmentTools() []Tool {
	tools := []Tool{}

	devTools := map[string]string{
		"git":       "Version control system",
		"svn":       "Subversion version control",
		"make":      "Build automation tool",
		"cmake":     "Cross-platform build system",
		"gradle":    "Build automation tool for Java",
		"maven":     "Build automation tool for Java",
		"ansible":   "Configuration management tool",
		"terraform": "Infrastructure as code tool",
		"vagrant":   "Development environment manager",
		"tmux":      "Terminal multiplexer",
		"screen":    "Terminal multiplexer",
		"vim":       "Text editor",
		"nvim":      "Neovim text editor",
		"emacs":     "Text editor",
		"code":      "Visual Studio Code",
		"subl":      "Sublime Text",
		"atom":      "Atom editor",
	}

	for tool, description := range devTools {
		if path, err := exec.LookPath(tool); err == nil {
			version := getToolVersion(tool)
			tools = append(tools, Tool{
				Name:        tool,
				Version:     version,
				Path:        path,
				Description: description,
			})
		}
	}

	return tools
}

// detectSystemCommands identifies common system commands
func detectSystemCommands() []string {
	commands := []string{}
	candidates := []string{
		"ls", "cd", "pwd", "mkdir", "rmdir", "rm", "cp", "mv", "ln",
		"find", "grep", "awk", "sed", "sort", "uniq", "head", "tail",
		"cat", "less", "more", "file", "which", "whereis", "locate",
		"ps", "top", "htop", "kill", "killall", "jobs", "bg", "fg",
		"df", "du", "mount", "umount", "lsblk", "fdisk",
		"tar", "gzip", "gunzip", "zip", "unzip", "7z",
		"chmod", "chown", "chgrp", "umask", "id", "whoami", "groups",
		"date", "cal", "uptime", "uname", "hostname", "who", "w",
		"history", "alias", "unalias", "export", "env", "printenv",
		"echo", "printf", "read", "test", "true", "false",
		"ssh", "scp", "rsync", "curl", "wget", "ping", "traceroute",
		"netstat", "ss", "lsof", "iptables", "firewall-cmd",
	}

	for _, cmd := range candidates {
		if isCommandAvailable(cmd) {
			commands = append(commands, cmd)
		}
	}

	return commands
}

// detectContainerTools identifies container and orchestration tools
func detectContainerTools() []string {
	tools := []string{}
	candidates := []string{
		"docker", "podman", "buildah", "skopeo",
		"kubectl", "helm", "minikube", "kind", "k3s",
		"docker-compose", "docker-machine",
		"containerd", "cri-o", "runc",
	}

	for _, tool := range candidates {
		if isCommandAvailable(tool) {
			tools = append(tools, tool)
		}
	}

	return tools
}

// detectCloudTools identifies cloud platform tools
func detectCloudTools() []string {
	tools := []string{}
	candidates := []string{
		"aws", "az", "gcloud", "gsutil",
		"doctl", "linode-cli", "vultr-cli",
		"heroku", "cf", "oc",
		"sam", "serverless", "pulumi",
	}

	for _, tool := range candidates {
		if isCommandAvailable(tool) {
			tools = append(tools, tool)
		}
	}

	return tools
}

// detectDatabaseTools identifies database tools and clients
func detectDatabaseTools() []string {
	tools := []string{}
	candidates := []string{
		"mysql", "mariadb", "psql", "sqlite3",
		"mongo", "mongosh", "redis-cli",
		"influx", "cqlsh", "snowsql",
		"sqlplus", "isql", "bcp",
	}

	for _, tool := range candidates {
		if isCommandAvailable(tool) {
			tools = append(tools, tool)
		}
	}

	return tools
}

// detectNetworkTools identifies network utilities
func detectNetworkTools() []string {
	tools := []string{}
	candidates := []string{
		"curl", "wget", "httpie", "http",
		"nc", "netcat", "nmap", "tcpdump",
		"wireshark", "tshark", "dig", "nslookup",
		"telnet", "ssh", "scp", "rsync",
		"iperf", "iperf3", "mtr", "traceroute",
	}

	for _, tool := range candidates {
		if isCommandAvailable(tool) {
			tools = append(tools, tool)
		}
	}

	return tools
}

// buildAvailabilityMap creates a map of all available tools
func buildAvailabilityMap(tools *ToolContext) {
	// Add all detected tools to availability map
	for _, pm := range tools.PackageManagers {
		tools.Available[pm] = true
	}
	for _, lang := range tools.Languages {
		tools.Available[lang.Name] = true
	}
	for _, tool := range tools.DevelopmentTools {
		tools.Available[tool.Name] = true
	}
	for _, cmd := range tools.SystemCommands {
		tools.Available[cmd] = true
	}
	for _, tool := range tools.ContainerTools {
		tools.Available[tool] = true
	}
	for _, tool := range tools.CloudTools {
		tools.Available[tool] = true
	}
	for _, tool := range tools.DatabaseTools {
		tools.Available[tool] = true
	}
	for _, tool := range tools.NetworkTools {
		tools.Available[tool] = true
	}
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
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
func IsToolAvailable(tool string) bool {
	context := GetSystemContext()
	available, exists := context.Tools.Available[tool]
	return exists && available
}

// isVerboseMode checks if verbose mode is enabled from environment or context
func isVerboseMode() bool {
	// Check environment variable
	return os.Getenv("FORGOR_VERBOSE") == "true"
}

// GetCacheInfo returns information about the persistent cache
func GetCacheInfo() CacheInfo {
	if err := initPersistentCache(); err != nil {
		return CacheInfo{}
	}

	info := CacheInfo{
		CacheDir: cacheDir,
		FilePath: cacheFile,
		LockFile: lockFile,
	}

	if stat, err := os.Stat(cacheFile); err == nil {
		info.FileExists = true
		info.FileSize = stat.Size()
		info.FileModTime = stat.ModTime()
	}

	return info
}

// ClearPersistentCache removes the persistent cache file
func ClearPersistentCache() error {
	if err := initPersistentCache(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Acquire write lock to ensure safe deletion
	lockFd, err := acquireFileLock(lockFile, true)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer releaseFileLock(lockFd)

	// Clear in-memory cache
	contextCacheMutex.Lock()
	systemContextCache = nil
	cacheTimestamp = time.Time{}
	contextCacheMutex.Unlock()

	// Remove cache file if it exists
	if _, err := os.Stat(cacheFile); err == nil {
		if err := os.Remove(cacheFile); err != nil {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
	}

	// Remove temp files if they exist
	if _, err := os.Stat(cacheFile + ".tmp"); err == nil {
		os.Remove(cacheFile + ".tmp")
	}

	return nil
}
