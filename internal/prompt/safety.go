package prompt

import (
	"fmt"
	"strings"
)

// CheckCommandSafety performs basic safety checks on commands
// This replaces the duplicated checkSafety functions in each provider
func CheckCommandSafety(command string) []string {
	var warnings []string
	cmd := strings.ToLower(command)

	dangerousPatterns := []string{
		"rm -rf /",
		"sudo rm",
		"dd if=",
		"mkfs",
		"format",
		"> /dev/",
		"shutdown",
		"reboot",
		":(){ :|:& };:",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			warnings = append(warnings, fmt.Sprintf("Potentially dangerous command detected: %s", pattern))
		}
	}

	return warnings
}

// CleanCommand removes common code block markers from command strings
// This is used by response parsers to clean up LLM output
func CleanCommand(command string) string {
	// Remove code block markers if present
	command = strings.TrimPrefix(command, "```bash")
	command = strings.TrimPrefix(command, "```sh")
	command = strings.TrimPrefix(command, "```shell")
	command = strings.TrimPrefix(command, "```")
	command = strings.TrimSuffix(command, "```")
	command = strings.TrimSpace(command)

	return command
}
