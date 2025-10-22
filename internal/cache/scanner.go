package cache

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// scanAllTools performs comprehensive tool detection
// Note that this operation can take some time, especially on systems with many tools installed.
func scanAllTools() (*ToolContext, error) {
	start := time.Now()

	tools := &ToolContext{
		Available:   make(map[string]bool),
		LastScan:    time.Now(),
		Version:     CacheVersion,
	}

	// Detect all tool categories
	tools.PackageManagers = detectPackageManagers()
	tools.Languages = detectLanguageRuntimes()
	tools.DevelopmentTools = detectDevelopmentTools()
	tools.SystemCommands = detectSystemCommands()
	tools.ContainerTools = detectContainerTools()
	tools.CloudTools = detectCloudTools()
	tools.DatabaseTools = detectDatabaseTools()
	tools.NetworkTools = detectNetworkTools()

	// Build availability map
	buildAvailabilityMap(tools)

	// Record scan duration
	tools.ScanDurationMs = time.Since(start).Milliseconds()

	return tools, nil
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

	// Probably need to find a better way to do this
	// For now, these are the most used languages (according to chat gpt)
	languages := map[string][]string{
		"python":  {"python", "python3"},
		"node":    {"node"},
		"bun":     {"bun"},
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
		"nim":     {"nim"}, // this isnt popular i just added it because i like it
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
		"netstat", "ss", "lsof", "iptables", "firewall-cmd", "forgor",
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
	if tools.Available == nil {
		tools.Available = make(map[string]bool)
	}

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
	var versionFlag string
	
	// Languages use different version flags for whatever reason
	switch language {
	case "go":
		versionFlag = "version"
	default:
		versionFlag = "--version"
	}

	// Try to get version with timeout
	cmd := exec.Command(command, versionFlag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	// Parse version from output (first line usually contains version)
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		version := strings.TrimSpace(lines[0])
		// Limit version string length
		if len(version) > 100 {
			version = version[:100]
		}
		return version
	}

	return "unknown"
}

// getToolVersion attempts to get the version of a tool
func getToolVersion(tool string) string {
	// Try common version flags
	versionFlags := []string{"--version", "-version", "version", "-V", "-v"}

	for _, flag := range versionFlags {
		cmd := exec.Command(tool, flag)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			// Get first line of output
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				version := strings.TrimSpace(lines[0])
				// Limit version string length
				if len(version) > 100 {
					version = version[:100]
				}
				return version
			}
		}
	}

	return "unknown"
}

// ScanAndCacheTools performs a full scan and saves to cache
// This is a convenience function for manual refresh operations
func ScanAndCacheTools() (*ToolContext, error) {
	tools, err := scanAllTools()
	if err != nil {
		return nil, err
	}

	if err := SaveToolCache(tools); err != nil {
		return tools, fmt.Errorf("scan succeeded but cache save failed: %w", err)
	}

	return tools, nil
}

// QuickScanEssentialTools performs a quick scan of only essential tools
// This can be used for a faster initial scan if needed
func QuickScanEssentialTools() *ToolContext {
	tools := &ToolContext{
		Available: make(map[string]bool),
		LastScan:  time.Now(),
		Version:   CacheVersion,
	}

	// Only scan most common tools
	essentialTools := []string{
		"git", "docker", "kubectl",
		"python", "python3", "node", "go",
		"npm", "pip", "brew", "apt",
	}

	for _, tool := range essentialTools {
		if isCommandAvailable(tool) {
			tools.Available[tool] = true
		}
	}

	return tools
}