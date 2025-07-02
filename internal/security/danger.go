package security

import (
	"regexp"
	"strings"

	"forgor/internal/llm"
)

// DangerDetector provides comprehensive command danger assessment
type DangerDetector struct {
	patterns []DangerPattern
}

// DangerPattern represents a dangerous command pattern
type DangerPattern struct {
	Name        string          // Human-readable name
	Pattern     *regexp.Regexp  // Regex pattern to match
	Level       llm.DangerLevel // Danger level
	Reason      string          // Why this is dangerous
	Factors     []string        // Risk factors
	Mitigations []string        // Suggested mitigations
	ContextSafe []string        // Contexts where this might be safe
}

// NewDangerDetector creates a new danger detector with predefined patterns
func NewDangerDetector() *DangerDetector {
	return &DangerDetector{
		patterns: getDangerPatterns(),
	}
}

// AssessCommand performs comprehensive danger assessment of a command
func (d *DangerDetector) AssessCommand(command string, context *llm.Context) llm.DangerAssessment {
	command = strings.TrimSpace(command)
	if command == "" {
		return llm.DangerAssessment{
			Level:      llm.DangerLevelSafe,
			Confidence: 1.0,
			Reason:     "Empty command",
		}
	}

	// Start with pattern-based detection
	patternAssessment := d.assessPatterns(command, context)

	// Add context-based modifications
	contextAssessment := d.assessContext(command, context, patternAssessment)

	// Add heuristic analysis
	finalAssessment := d.assessHeuristics(command, context, contextAssessment)

	return finalAssessment
}

// assessPatterns checks command against known dangerous patterns
func (d *DangerDetector) assessPatterns(command string, context *llm.Context) llm.DangerAssessment {
	lowerCommand := strings.ToLower(command)
	maxLevel := llm.DangerLevelSafe
	var allFactors []string
	var allMitigations []string
	var reasons []string
	confidence := 0.8 // High confidence in pattern matching

	for _, pattern := range d.patterns {
		if pattern.Pattern.MatchString(lowerCommand) {
			// Check if context makes this safe
			if d.isContextSafe(command, context, pattern) {
				continue
			}

			if d.isDangerLevelHigher(pattern.Level, maxLevel) {
				maxLevel = pattern.Level
				reasons = append(reasons, pattern.Reason)
			}

			allFactors = append(allFactors, pattern.Factors...)
			allMitigations = append(allMitigations, pattern.Mitigations...)
		}
	}

	reason := "Safe command"
	if len(reasons) > 0 {
		reason = strings.Join(reasons, "; ")
	}

	return llm.DangerAssessment{
		Level:       maxLevel,
		Confidence:  confidence,
		Reason:      reason,
		Factors:     removeDuplicates(allFactors),
		Mitigations: removeDuplicates(allMitigations),
	}
}

// assessContext modifies danger assessment based on execution context
func (d *DangerDetector) assessContext(command string, context *llm.Context, baseAssessment llm.DangerAssessment) llm.DangerAssessment {
	assessment := baseAssessment

	// Context-specific risk modifications
	if context != nil {
		// Working in temporary directories is generally safer
		if strings.Contains(context.WorkingDirectory, "/tmp") ||
			strings.Contains(context.WorkingDirectory, "/temp") {
			if assessment.Level == llm.DangerLevelHigh {
				assessment.Level = llm.DangerLevelMedium
				assessment.Mitigations = append(assessment.Mitigations, "Operating in temporary directory reduces risk")
			}
		}

		// Root directory operations are more dangerous
		if context.WorkingDirectory == "/" {
			if assessment.Level == llm.DangerLevelMedium {
				assessment.Level = llm.DangerLevelHigh
				assessment.Factors = append(assessment.Factors, "Operating in root directory")
			}
		}

		// Add OS-specific considerations
		if context.OS == "darwin" && strings.Contains(command, "rm") {
			assessment.Mitigations = append(assessment.Mitigations, "Consider using 'trash' command instead of rm on macOS")
		}
	}

	return assessment
}

// assessHeuristics applies additional heuristic analysis
func (d *DangerDetector) assessHeuristics(command string, context *llm.Context, baseAssessment llm.DangerAssessment) llm.DangerAssessment {
	assessment := baseAssessment

	// Multiple dangerous elements increase risk
	dangerousCount := 0
	if strings.Contains(command, "sudo") {
		dangerousCount++
	}
	if strings.Contains(command, "rm") {
		dangerousCount++
	}
	if strings.Contains(command, "-rf") {
		dangerousCount++
	}
	if strings.Contains(command, ">/dev/") {
		dangerousCount++
	}
	if strings.Contains(command, "dd") {
		dangerousCount++
	}

	if dangerousCount >= 2 {
		if assessment.Level == llm.DangerLevelMedium {
			assessment.Level = llm.DangerLevelHigh
		} else if assessment.Level == llm.DangerLevelHigh {
			assessment.Level = llm.DangerLevelCritical
		}
		assessment.Factors = append(assessment.Factors, "Multiple dangerous elements combined")
	}

	// Piped commands with downloads are risky
	if (strings.Contains(command, "curl") || strings.Contains(command, "wget")) &&
		(strings.Contains(command, "| sh") || strings.Contains(command, "| bash")) {
		assessment.Level = llm.DangerLevelCritical
		assessment.Factors = append(assessment.Factors, "Remote code execution via piped download")
		assessment.Mitigations = append(assessment.Mitigations, "Download and inspect scripts before execution")
	}

	// Wildcard with destructive commands
	if strings.Contains(command, "rm") && (strings.Contains(command, "*") || strings.Contains(command, "/*")) {
		if assessment.Level < llm.DangerLevelHigh {
			assessment.Level = llm.DangerLevelHigh
		}
		assessment.Factors = append(assessment.Factors, "Wildcard deletion")
	}

	return assessment
}

// isContextSafe checks if the current context makes a dangerous pattern safe
func (d *DangerDetector) isContextSafe(command string, context *llm.Context, pattern DangerPattern) bool {
	if context == nil || len(pattern.ContextSafe) == 0 {
		return false
	}

	for _, safeContext := range pattern.ContextSafe {
		if strings.Contains(context.WorkingDirectory, safeContext) {
			return true
		}
	}

	return false
}

// isDangerLevelHigher compares danger levels
func (d *DangerDetector) isDangerLevelHigher(level1, level2 llm.DangerLevel) bool {
	levels := map[llm.DangerLevel]int{
		llm.DangerLevelSafe:     0,
		llm.DangerLevelLow:      1,
		llm.DangerLevelMedium:   2,
		llm.DangerLevelHigh:     3,
		llm.DangerLevelCritical: 4,
	}

	return levels[level1] > levels[level2]
}

// getDangerPatterns returns all predefined dangerous patterns
func getDangerPatterns() []DangerPattern {
	return []DangerPattern{
		{
			Name:        "Recursive Force Delete",
			Pattern:     regexp.MustCompile(`rm\s+(-[rf]+|--recursive|--force)`),
			Level:       llm.DangerLevelHigh,
			Reason:      "Recursive force deletion can permanently destroy data",
			Factors:     []string{"Data loss", "Irreversible operation"},
			Mitigations: []string{"Use 'ls' first to preview", "Consider 'trash' command", "Make backups"},
			ContextSafe: []string{"/tmp", "/temp"},
		},
		{
			Name:        "Root Filesystem Operations",
			Pattern:     regexp.MustCompile(`(rm|mv|cp|chmod|chown).*(/\*|/[^/\s]*\*)}`),
			Level:       llm.DangerLevelCritical,
			Reason:      "Operations on root filesystem can break the system",
			Factors:     []string{"System corruption", "Boot failure"},
			Mitigations: []string{"Be extremely specific with paths", "Test in VM first"},
		},
		{
			Name:        "Disk Device Operations",
			Pattern:     regexp.MustCompile(`dd\s+.*(/dev/|if=/dev/|of=/dev/)`),
			Level:       llm.DangerLevelCritical,
			Reason:      "Direct disk operations can destroy data or corrupt systems",
			Factors:     []string{"Data corruption", "System failure"},
			Mitigations: []string{"Double-check device paths", "Unmount devices first", "Use disk imaging tools"},
		},
		{
			Name:        "System Shutdown/Reboot",
			Pattern:     regexp.MustCompile(`(shutdown|reboot|halt|poweroff|init\s+[06])`),
			Level:       llm.DangerLevelMedium,
			Reason:      "System restart will terminate all running processes",
			Factors:     []string{"Work loss", "Service interruption"},
			Mitigations: []string{"Save work first", "Warn other users", "Schedule during maintenance window"},
		},
		{
			Name:        "Permissive Permissions",
			Pattern:     regexp.MustCompile(`chmod\s+(-R\s+)?777`),
			Level:       llm.DangerLevelHigh,
			Reason:      "777 permissions create security vulnerabilities",
			Factors:     []string{"Security risk", "Unauthorized access"},
			Mitigations: []string{"Use specific permissions", "Apply principle of least privilege"},
		},
		{
			Name:        "Remote Code Execution",
			Pattern:     regexp.MustCompile(`(curl|wget).*\|\s*(sh|bash|zsh|fish)`),
			Level:       llm.DangerLevelCritical,
			Reason:      "Executing remote code without inspection is extremely dangerous",
			Factors:     []string{"Malware execution", "System compromise"},
			Mitigations: []string{"Download and inspect first", "Use package managers", "Verify sources"},
		},
		{
			Name:        "Process Termination",
			Pattern:     regexp.MustCompile(`kill(all)?\s+(-[9]+|--signal.*KILL)`),
			Level:       llm.DangerLevelMedium,
			Reason:      "Force killing processes can cause data loss",
			Factors:     []string{"Data loss", "Corrupted files"},
			Mitigations: []string{"Try graceful termination first", "Check for important processes"},
		},
		{
			Name:        "Package Management Risks",
			Pattern:     regexp.MustCompile(`(npm|pip|gem)\s+install.*--global|sudo\s+(npm|pip|gem)`),
			Level:       llm.DangerLevelMedium,
			Reason:      "Global package installation can affect system stability",
			Factors:     []string{"System pollution", "Dependency conflicts"},
			Mitigations: []string{"Use virtual environments", "Check package reputation"},
		},
		{
			Name:        "Archive Extraction",
			Pattern:     regexp.MustCompile(`(tar|unzip)\s+.*(-C\s*/|--directory[= ]*/)`),
			Level:       llm.DangerLevelMedium,
			Reason:      "Extracting archives to root directories can overwrite system files",
			Factors:     []string{"File overwriting", "System corruption"},
			Mitigations: []string{"Extract to safe directories", "List contents first"},
		},
		{
			Name:        "Network Service Binding",
			Pattern:     regexp.MustCompile(`.*--bind.*0\.0\.0\.0|.*--host.*0\.0\.0\.0`),
			Level:       llm.DangerLevelLow,
			Reason:      "Binding to all interfaces exposes services to network",
			Factors:     []string{"Security exposure", "Unauthorized access"},
			Mitigations: []string{"Bind to specific interfaces", "Use firewalls", "Enable authentication"},
		},
		{
			Name:        "Shell History Manipulation",
			Pattern:     regexp.MustCompile(`(history\s+-c|>\s*\$HISTFILE|rm.*\.(bash_|zsh_)?history)`),
			Level:       llm.DangerLevelLow,
			Reason:      "Manipulating shell history can hide malicious activity",
			Factors:     []string{"Audit trail loss", "Forensic difficulty"},
			Mitigations: []string{"Keep separate audit logs", "Use centralized logging"},
		},
	}
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
