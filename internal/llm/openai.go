package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"forgor/internal/prompt"

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
	// Convert to prompt package request format
	promptReq := &prompt.Request{
		Query: request.Query,
		Context: prompt.RequestContext{
			WorkingDirectory: request.Context.WorkingDirectory,
			History:          request.Context.History,
			UserContext:      request.Context.UserContext,
		},
		Options: prompt.RequestOptions{
			IncludeExplanation: request.Options.IncludeExplanation,
			MaxTokens:          request.Options.MaxTokens,
			Temperature:        request.Options.Temperature,
		},
	}

	userPrompt := prompt.BuildOpenAICommandPrompt(promptReq)

	// Convert to prompt package context format
	promptContext := prompt.Context{
		OS:               request.Context.OS,
		Shell:            request.Context.Shell,
		Architecture:     request.Context.Architecture,
		User:             request.Context.User,
		WorkingDirectory: request.Context.WorkingDirectory,
		ToolsSummary:     request.Context.ToolsSummary,
		PackageManagers:  request.Context.PackageManagers,
		Languages:        request.Context.Languages,
		ContainerTools:   request.Context.ContainerTools,
		CloudTools:       request.Context.CloudTools,
	}

	systemPrompt := prompt.GetSystemPrompt(promptContext)

	openAIReq := openAIRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
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
		Warnings:     prompt.CheckCommandSafety(command),
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

	// Clean up command using centralized function
	command = prompt.CleanCommand(command)

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
