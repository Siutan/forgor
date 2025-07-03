package prompt

import (
	"fmt"
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
	History          []string
	UserContext      string
}

// RequestOptions contains options for the request
type RequestOptions struct {
	IncludeExplanation bool
	MaxTokens          int
	Temperature        float64
}

// BuildCommandPrompt constructs the prompt for command generation
// This replaces the duplicated buildCommandPrompt functions in each provider
func BuildCommandPrompt(request *Request) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Convert this natural language request to a shell command:\n\n%s", request.Query))

	// Add context information
	if request.Context.WorkingDirectory != "" {
		parts = append(parts, fmt.Sprintf("\nCurrent directory: %s", request.Context.WorkingDirectory))
	}

	// Add command history if available
	if len(request.Context.History) > 0 {
		parts = append(parts, "\nRecent command history:")
		for i, cmd := range request.Context.History {
			if i >= 5 { // Limit to last 5 commands
				break
			}
			parts = append(parts, fmt.Sprintf("  %s", cmd))
		}
	}

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
