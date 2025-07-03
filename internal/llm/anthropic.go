package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"forgor/internal/prompt"

	"github.com/go-resty/resty/v2"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	client  *resty.Client
	apiKey  string
	model   string
	baseURL string
}

// Anthropic API request/response structures
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence"`
	Usage        anthropicUsage     `json:"usage"`
	Error        *anthropicError    `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("x-api-key", apiKey)
	client.SetHeader("content-type", "application/json")
	client.SetHeader("anthropic-version", "2023-06-01")

	return &AnthropicProvider{
		client:  client,
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1",
	}
}

// GenerateCommand generates a shell command from a natural language query
func (p *AnthropicProvider) GenerateCommand(ctx context.Context, request *Request) (*Response, error) {
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

	userPrompt := prompt.BuildAnthropicCommandPrompt(promptReq)

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

	anthropicReq := anthropicRequest{
		Model:     p.model,
		MaxTokens: request.Options.MaxTokens,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: request.Options.Temperature,
	}

	var resp anthropicResponse
	restResp, err := p.client.R().
		SetContext(ctx).
		SetBody(anthropicReq).
		SetResult(&resp).
		Post(p.baseURL + "/messages")

	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: "Failed to call Anthropic API",
			Cause:   err,
		}
	}

	if restResp.IsError() {
		return nil, p.handleAPIError(restResp, &resp)
	}

	if len(resp.Content) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "No response from Anthropic",
		}
	}

	content := resp.Content[0].Text
	command, explanation := p.parseResponse(content, request.Options.IncludeExplanation)

	return &Response{
		Command:     command,
		Explanation: explanation,
		Confidence:  p.calculateConfidence(resp.StopReason),
		Warnings:    prompt.CheckCommandSafety(command),
		Usage: &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Metadata: map[string]interface{}{
			"model":       resp.Model,
			"stop_reason": resp.StopReason,
		},
	}, nil
}

// ExplainCommand explains what a command does
func (p *AnthropicProvider) ExplainCommand(ctx context.Context, command string) (*Response, error) {
	prompt := fmt.Sprintf("Explain what this shell command does:\n\n%s\n\nProvide a clear, concise explanation of what this command accomplishes.", command)

	anthropicReq := anthropicRequest{
		Model:     p.model,
		MaxTokens: 300,
		System:    "You are a helpful assistant that explains shell commands clearly and concisely.",
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1,
	}

	var resp anthropicResponse
	restResp, err := p.client.R().
		SetContext(ctx).
		SetBody(anthropicReq).
		SetResult(&resp).
		Post(p.baseURL + "/messages")

	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: "Failed to call Anthropic API",
			Cause:   err,
		}
	}

	if restResp.IsError() {
		return nil, p.handleAPIError(restResp, &resp)
	}

	if len(resp.Content) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "No response from Anthropic",
		}
	}

	return &Response{
		Command:     command,
		Explanation: strings.TrimSpace(resp.Content[0].Text),
		Confidence:  1.0, // High confidence for explanations
		Usage: &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}, nil
}

// GetProviderInfo returns information about the Anthropic provider
func (p *AnthropicProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:    "Anthropic",
		Version: "1.0.0",
		Models:  []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
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
			"provider": "anthropic",
			"model":    p.model,
		},
	}
}

// parseResponse extracts command and explanation from the response
func (p *AnthropicProvider) parseResponse(content string, includeExplanation bool) (command, explanation string) {
	content = strings.TrimSpace(content)

	if includeExplanation && strings.Contains(content, "||") {
		parts := strings.SplitN(content, "||", 2)
		command = strings.TrimSpace(parts[0])
		explanation = strings.TrimSpace(parts[1])
	} else {
		command = content
	}

	// Clean up command using centralized function
	command = prompt.CleanCommand(command)

	return command, explanation
}

// calculateConfidence estimates confidence based on stop reason
func (p *AnthropicProvider) calculateConfidence(stopReason string) float64 {
	switch stopReason {
	case "end_turn":
		return 0.9
	case "max_tokens":
		return 0.7
	case "stop_sequence":
		return 0.8
	default:
		return 0.5
	}
}

// handleAPIError converts Anthropic API errors to our error format
func (p *AnthropicProvider) handleAPIError(resp *resty.Response, apiResp *anthropicResponse) error {
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
		case "overloaded_error":
			errorType = ErrorTypeModel
		default:
			errorType = ErrorTypeUnknown
		}

		return &Error{
			Type:    errorType,
			Message: apiResp.Error.Message,
		}
	}

	return &Error{
		Type:    ErrorTypeNetwork,
		Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode(), resp.String()),
	}
}
