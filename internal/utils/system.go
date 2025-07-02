package utils

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
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
	cacheExpiration    = 5 * time.Minute
)

// GetSystemContext returns comprehensive system context information
func GetSystemContext() *SystemContext {
	contextCacheMutex.RLock()
	if systemContextCache != nil && time.Since(systemContextCache.Tools.LastChecked) < cacheExpiration {
		defer contextCacheMutex.RUnlock()
		return systemContextCache
	}
	contextCacheMutex.RUnlock()

	contextCacheMutex.Lock()
	defer contextCacheMutex.Unlock()

	// Double-check after acquiring write lock
	if systemContextCache != nil && time.Since(systemContextCache.Tools.LastChecked) < cacheExpiration {
		return systemContextCache
	}

	systemContextCache = buildSystemContext()
	return systemContextCache
}

// buildSystemContext creates a comprehensive system context
func buildSystemContext() *SystemContext {
	context := &SystemContext{
		OS:               GetOperatingSystem(),
		Shell:            GetCurrentShell(),
		Architecture:     runtime.GOARCH,
		WorkingDirectory: GetWorkingDirectory(),
		User:             os.Getenv("USER"),
		HomeDirectory:    getHomeDirectory(),
		Environment:      getRelevantEnvironment(),
	}

	// Gather tool context
	context.Tools = gatherToolContext()

	return context
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

// getLanguageVersion attempts to get the version of a language runtime
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

	cmd := exec.Command(command, args...)
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

// getToolVersion attempts to get the version of a tool
func getToolVersion(tool string) string {
	// Try common version flags
	versionFlags := []string{"--version", "-version", "-V", "-v", "version"}

	for _, flag := range versionFlags {
		cmd := exec.Command(tool, flag)
		output, err := cmd.CombinedOutput()
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

// RefreshSystemContext forces a refresh of the system context cache
func RefreshSystemContext() *SystemContext {
	contextCacheMutex.Lock()
	defer contextCacheMutex.Unlock()

	systemContextCache = buildSystemContext()
	return systemContextCache
}
