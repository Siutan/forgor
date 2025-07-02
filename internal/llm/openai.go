package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client  *resty.Client
	apiKey  string
	model   string
	baseURL string
}

// OpenAI API request/response structures
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Authorization", "Bearer "+apiKey)
	client.SetHeader("Content-Type", "application/json")

	return &OpenAIProvider{
		client:  client,
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
	}
}

// GenerateCommand generates a shell command from a natural language query
func (p *OpenAIProvider) GenerateCommand(ctx context.Context, request *Request) (*Response, error) {
	prompt := p.buildCommandPrompt(request)

	openAIReq := openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: getSystemPrompt(request.Context),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   request.Options.MaxTokens,
		Temperature: request.Options.Temperature,
		Stream:      false,
	}

	var resp openAIResponse
	restResp, err := p.client.R().
		SetContext(ctx).
		SetBody(openAIReq).
		SetResult(&resp).
		Post(p.baseURL + "/chat/completions")

	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: "Failed to call OpenAI API",
			Cause:   err,
		}
	}

	if restResp.IsError() {
		return nil, p.handleAPIError(restResp, &resp)
	}

	if len(resp.Choices) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "No response from OpenAI",
		}
	}

	choice := resp.Choices[0]
	command, explanation, llmDangerLevel, llmDangerReason := p.parseResponse(choice.Message.Content, request.Options.IncludeExplanation)

	return &Response{
		Command:      command,
		Explanation:  explanation,
		Confidence:   p.calculateConfidence(choice.FinishReason),
		DangerLevel:  llmDangerLevel,
		DangerReason: llmDangerReason,
		Warnings:     p.checkSafety(command),
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: map[string]interface{}{
			"model":            resp.Model,
			"finish_reason":    choice.FinishReason,
			"llm_danger_level": string(llmDangerLevel),
		},
	}, nil
}

// ExplainCommand explains what a command does
func (p *OpenAIProvider) ExplainCommand(ctx context.Context, command string) (*Response, error) {
	prompt := fmt.Sprintf("Explain what this shell command does:\n\n%s\n\nProvide a clear, concise explanation of what this command accomplishes.", command)

	openAIReq := openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant that explains shell commands clearly and concisely.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   300,
		Temperature: 0.1,
		Stream:      false,
	}

	var resp openAIResponse
	restResp, err := p.client.R().
		SetContext(ctx).
		SetBody(openAIReq).
		SetResult(&resp).
		Post(p.baseURL + "/chat/completions")

	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: "Failed to call OpenAI API",
			Cause:   err,
		}
	}

	if restResp.IsError() {
		return nil, p.handleAPIError(restResp, &resp)
	}

	if len(resp.Choices) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "No response from OpenAI",
		}
	}

	return &Response{
		Command:     command,
		Explanation: strings.TrimSpace(resp.Choices[0].Message.Content),
		Confidence:  1.0, // High confidence for explanations
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// GetProviderInfo returns information about the OpenAI provider
func (p *OpenAIProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:    "OpenAI",
		Version: "1.0.0",
		Models:  []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		Capabilities: []string{
			"command_generation",
			"command_explanation",
			"context_awareness",
			"safety_filtering",
		},
		Limits: map[string]int{
			"max_tokens":      4096,
			"max_history":     10,
			"timeout_seconds": 30,
		},
		Metadata: map[string]string{
			"provider": "openai",
			"model":    p.model,
		},
	}
}

// buildCommandPrompt constructs the prompt for command generation
func (p *OpenAIProvider) buildCommandPrompt(request *Request) string {
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

	// Add response format instructions
	parts = append(parts, "\nPlease respond in this exact format:")
	parts = append(parts, "COMMAND: [the shell command]")
	if request.Options.IncludeExplanation {
		parts = append(parts, "EXPLANATION: [brief explanation]")
	}
	parts = append(parts, "DANGER_LEVEL: [safe/low/medium/high/critical]")
	parts = append(parts, "DANGER_REASON: [reason for the danger level assessment]")

	return strings.Join(parts, "\n")
}

// parseResponse extracts command, explanation, and danger assessment from the response
func (p *OpenAIProvider) parseResponse(content string, includeExplanation bool) (command, explanation string, dangerLevel DangerLevel, dangerReason string) {
	content = strings.TrimSpace(content)
	lines := strings.Split(content, "\n")

	// Default values
	dangerLevel = DangerLevelSafe
	dangerReason = "No specific assessment provided"

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "COMMAND:") {
			command = strings.TrimSpace(strings.TrimPrefix(line, "COMMAND:"))
		} else if strings.HasPrefix(line, "EXPLANATION:") && includeExplanation {
			explanation = strings.TrimSpace(strings.TrimPrefix(line, "EXPLANATION:"))
		} else if strings.HasPrefix(line, "DANGER_LEVEL:") {
			levelStr := strings.TrimSpace(strings.TrimPrefix(line, "DANGER_LEVEL:"))
			switch strings.ToLower(levelStr) {
			case "safe":
				dangerLevel = DangerLevelSafe
			case "low":
				dangerLevel = DangerLevelLow
			case "medium":
				dangerLevel = DangerLevelMedium
			case "high":
				dangerLevel = DangerLevelHigh
			case "critical":
				dangerLevel = DangerLevelCritical
			}
		} else if strings.HasPrefix(line, "DANGER_REASON:") {
			dangerReason = strings.TrimSpace(strings.TrimPrefix(line, "DANGER_REASON:"))
		}
	}

	// Fallback: if no structured response, treat whole content as command
	if command == "" {
		command = content
	}

	// Clean up command (remove code block markers if present)
	command = strings.TrimPrefix(command, "```bash")
	command = strings.TrimPrefix(command, "```sh")
	command = strings.TrimPrefix(command, "```")
	command = strings.TrimSuffix(command, "```")
	command = strings.TrimSpace(command)

	return command, explanation, dangerLevel, dangerReason
}

// calculateConfidence estimates confidence based on finish reason
func (p *OpenAIProvider) calculateConfidence(finishReason string) float64 {
	switch finishReason {
	case "stop":
		return 0.9
	case "length":
		return 0.7
	case "content_filter":
		return 0.3
	default:
		return 0.5
	}
}

// checkSafety performs basic safety checks on commands
func (p *OpenAIProvider) checkSafety(command string) []string {
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

// handleAPIError converts OpenAI API errors to our error format
func (p *OpenAIProvider) handleAPIError(resp *resty.Response, apiResp *openAIResponse) error {
	if apiResp.Error != nil {
		var errorType ErrorType
		switch apiResp.Error.Type {
		case "invalid_request_error":
			errorType = ErrorTypeInvalidInput
		case "authentication_error":
			errorType = ErrorTypeAuth
		case "permission_error":
			errorType = ErrorTypeAuth
		case "rate_limit_error":
			errorType = ErrorTypeRateLimit
		case "quota_exceeded":
			errorType = ErrorTypeQuota
		case "server_error":
			errorType = ErrorTypeModel
		default:
			errorType = ErrorTypeUnknown
		}

		return &Error{
			Type:    errorType,
			Message: apiResp.Error.Message,
			Code:    apiResp.Error.Code,
		}
	}

	return &Error{
		Type:    ErrorTypeNetwork,
		Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode(), resp.String()),
	}
}

// getSystemPrompt returns the system prompt for command generation
func getSystemPrompt(context Context) string {
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
1. Return only the command, no extra text or formatting
2. Ensure commands are safe and won't cause system damage
3. Use appropriate flags and options for the target OS and shell
4. Prefer tools and commands that are actually available on this system
5. Take advantage of available package managers, languages, and tools when relevant
6. If the request is unclear, make reasonable assumptions based on the available tools
7. Consider the user's environment and available capabilities when generating commands

Examples:
- "find all txt files" → find . -name "*.txt"
- "show disk usage" → df -h
- "list running processes" → ps aux
- "compress this folder" → tar -czf archive.tar.gz .
- "install package" → use appropriate package manager (brew, apt, yum, etc.)
- "run container" → use docker, podman, or available container runtime

Remember: Safety first - avoid destructive operations unless explicitly requested. Use tools that are actually available on this system.`

	return basePrompt
}
