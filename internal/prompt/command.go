package prompt

import (
	"fmt"
	"forgor/internal/history"
	"strings"
)

// Request represents a command generation request
type Request struct {
	Query   string
	Context RequestContext
	Options RequestOptions
}

// RequestContext contains contextual information for the request
type RequestContext struct {
	WorkingDirectory string
	History          []history.HistoryEntry
	UserContext      string
}

// RequestOptions contains options for the request
type RequestOptions struct {
	IncludeExplanation bool
	MaxTokens          int
	Temperature        float64
}

func formatHistoryForPrompt(historyEntries []history.HistoryEntry) string {
	if len(historyEntries) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "\n\nHere is the recent command history (most recent last):")
	for _, entry := range historyEntries {
		status := ""
		if entry.ExitCode > 0 {
			status = fmt.Sprintf(" (FAILED with exit code %d)", entry.ExitCode)
		} else if entry.ExitCode == 0 {
			status = " (SUCCESS)"
		}
		// If ExitCode is -1 (unknown), no status is added.
		parts = append(parts, fmt.Sprintf("- `%s`%s", entry.Command, status))
	}
	parts = append(parts, "\n\nPay special attention to any FAILED commands and try to fix them based on the user's request.")
	return strings.Join(parts, "\n")
}

// BuildCommandPrompt constructs the prompt for command generation
func BuildCommandPrompt(request *Request) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Convert this natural language request to a shell command:\n\n%s", request.Query))

	// Add context information
	if request.Context.WorkingDirectory != "" {
		parts = append(parts, fmt.Sprintf("\nCurrent directory: %s", request.Context.WorkingDirectory))
	}

	// Add command history if available
	parts = append(parts, formatHistoryForPrompt(request.Context.History))

	// Add user context if provided
	if request.Context.UserContext != "" {
		parts = append(parts, fmt.Sprintf("\nAdditional context: %s", request.Context.UserContext))
	}

	return strings.Join(parts, "\n")
}

// BuildOpenAICommandPrompt builds the OpenAI-specific command prompt with structured output
func BuildOpenAICommandPrompt(request *Request) string {
	basePrompt := BuildCommandPrompt(request)

	// Add OpenAI-specific response format instructions
	var formatParts []string
	formatParts = append(formatParts, "\nPlease respond in this exact format:")
	formatParts = append(formatParts, "COMMAND: [the shell command]")

	if request.Options.IncludeExplanation {
		formatParts = append(formatParts, "EXPLANATION: [brief explanation]")
	}

	formatParts = append(formatParts, "DANGER_LEVEL: [safe/low/medium/high/critical]")
	formatParts = append(formatParts, "DANGER_REASON: [reason for the danger level assessment]")

	return basePrompt + strings.Join(formatParts, "\n")
}

// BuildAnthropicCommandPrompt builds the Anthropic-specific command prompt
func BuildAnthropicCommandPrompt(request *Request) string {
	basePrompt := BuildCommandPrompt(request)

	// Add Anthropic-specific response format instructions
	if request.Options.IncludeExplanation {
		basePrompt += "\nRespond with the command followed by a brief explanation separated by '||'."
	} else {
		basePrompt += "\nRespond with only the shell command, no explanation."
	}

	return basePrompt
}

// BuildGeminiCommandPrompt builds the Gemini-specific command prompt
func BuildGeminiCommandPrompt(request *Request) string {
	basePrompt := BuildCommandPrompt(request)

	// Add Gemini-specific response format instructions (same as Anthropic)
	if request.Options.IncludeExplanation {
		basePrompt += "\nRespond with the command followed by a brief explanation separated by '||'."
	} else {
		basePrompt += "\nRespond with only the shell command, no explanation."
	}

	return basePrompt
}
