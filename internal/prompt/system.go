package prompt

import (
	"fmt"
	"strings"
)

// Context represents the system context for prompt generation
type Context struct {
	OS               string
	Shell            string
	Architecture     string
	User             string
	WorkingDirectory string
	ToolsSummary     string
	PackageManagers  []string
	Languages        []string
	ContainerTools   []string
	CloudTools       []string
}

// GetSystemPrompt returns the enhanced system prompt for command generation
func GetSystemPrompt(context Context) string {
	basePrompt := fmt.Sprintf(`You are a helpful shell command assistant. Convert natural language requests into safe, executable shell commands for %s using %s.

System Information:
- OS: %s (%s architecture)
- Shell: %s
- User: %s
- Working Directory: %s`, context.OS, context.Shell, context.OS, context.Architecture, context.Shell, context.User, context.WorkingDirectory)

	// Add tool context if available
	if context.ToolsSummary != "" {
		basePrompt += fmt.Sprintf(`
- Available Tools: %s`, context.ToolsSummary)
	}

	// Add package managers if available
	if len(context.PackageManagers) > 0 {
		basePrompt += fmt.Sprintf(`
- Package Managers: %s`, strings.Join(context.PackageManagers, ", "))
	}

	// Add programming languages if available
	if len(context.Languages) > 0 {
		basePrompt += fmt.Sprintf(`
- Programming Languages: %s`, strings.Join(context.Languages, ", "))
	}

	// Add container tools if available
	if len(context.ContainerTools) > 0 {
		basePrompt += fmt.Sprintf(`
- Container Tools: %s`, strings.Join(context.ContainerTools, ", "))
	}

	// Add cloud tools if available
	if len(context.CloudTools) > 0 {
		basePrompt += fmt.Sprintf(`
- Cloud Tools: %s`, strings.Join(context.CloudTools, ", "))
	}

	basePrompt += `

Rules:
1. Return only the command, no extra text or formatting unless specifically requested
2. Ensure commands are safe and won't cause system damage
3. Use appropriate flags and options for the target OS and shell
4. Prefer tools and commands that are actually available on this system
5. Take advantage of available package managers, languages, and tools when relevant
6. If the request is unclear, make reasonable assumptions based on the available tools

IMPORTANT - Command Path and Alias Guidelines:
7. When creating aliases, assume commands are already in PATH unless explicitly told otherwise
8. For commands like "forgor", "git", "docker" etc., use the bare command name (e.g., "alias ff=forgor")
9. Only use full paths when explicitly specified or when dealing with local scripts/files
10. Trailing slashes indicate directories (e.g., "cd /path/to/dir/")
11. When referencing executables, assume they're in PATH unless context suggests otherwise
12. For alias creation specifically:
    - "make X an alias to Y" → "alias X=Y" (assuming Y is in PATH)
    - "alias X to /full/path/Y" → "alias X=/full/path/Y" (when full path given)
    - Never default to current directory paths for well-known commands
13. Use double quotes for aliases to enclose the alias, if needed, escape the inner quotes

Examples:
- "find all txt files" → find . -name "*.txt"
- "show disk usage" → df -h
- "list running processes" → ps aux
- "compress this folder" → tar -czf archive.tar.gz .
- "install package" → use appropriate package manager (brew, apt, yum, etc.)
- "run container" → use docker, podman, or available container runtime
- "make ff an alias to forgor" → alias ff=forgor
- "create alias for git status" → alias gs='git status'
- "alias ll to list files with details" → alias ll='ls -la'

Remember: Safety first - avoid destructive operations unless explicitly requested. Use tools that are actually available on this system. For alias creation, trust that commands mentioned are properly installed and available in PATH.`

	return basePrompt
}
