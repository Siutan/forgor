package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// GeminiProvider implements the Provider interface for Google AI Gemini
type GeminiProvider struct {
	client  *resty.Client
	apiKey  string
	model   string
	baseURL string
}

// Gemini API request/response structures
type geminiRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig  `json:"generationConfig,omitempty"`
	SafetySettings    []geminiSafetySetting    `json:"safetySettings,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
}

type geminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type geminiResponse struct {
	Candidates     []geminiCandidate     `json:"candidates"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
	Error          *geminiError          `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	Index         int                  `json:"index"`
	SafetyRatings []geminiSafetyRating `json:"safetyRatings"`
}

type geminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type geminiPromptFeedback struct {
	SafetyRatings []geminiSafetyRating `json:"safetyRatings"`
	BlockReason   string               `json:"blockReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NewGeminiProvider creates a new Google AI Gemini provider
func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Content-Type", "application/json")

	return &GeminiProvider{
		client:  client,
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}

// GenerateCommand generates a shell command from a natural language query
func (p *GeminiProvider) GenerateCommand(ctx context.Context, request *Request) (*Response, error) {
	prompt := p.buildCommandPrompt(request)

	geminiReq := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
				Role: "user",
			},
		},
		SystemInstruction: &geminiSystemInstruction{
			Parts: []geminiPart{
				{Text: getSystemPrompt(request.Context.OS, request.Context.Shell)},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     request.Options.Temperature,
			MaxOutputTokens: request.Options.MaxTokens,
			TopP:            0.8,
			TopK:            40,
		},
		SafetySettings: []geminiSafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	var resp geminiResponse
	restResp, err := p.client.R().
		SetContext(ctx).
		SetBody(geminiReq).
		SetResult(&resp).
		Post(url)

	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: "Failed to call Gemini API",
			Cause:   err,
		}
	}

	if restResp.IsError() {
		return nil, p.handleAPIError(restResp, &resp)
	}

	if len(resp.Candidates) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "No response from Gemini",
		}
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "Empty response from Gemini",
		}
	}

	content := candidate.Content.Parts[0].Text
	command, explanation := p.parseResponse(content, request.Options.IncludeExplanation)

	var usage *Usage
	if resp.UsageMetadata != nil {
		usage = &Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &Response{
		Command:     command,
		Explanation: explanation,
		Confidence:  p.calculateConfidence(candidate.FinishReason),
		Warnings:    p.checkSafety(command),
		Usage:       usage,
		Metadata: map[string]interface{}{
			"model":         p.model,
			"finish_reason": candidate.FinishReason,
		},
	}, nil
}

// ExplainCommand explains what a command does
func (p *GeminiProvider) ExplainCommand(ctx context.Context, command string) (*Response, error) {
	prompt := fmt.Sprintf("Explain what this shell command does:\n\n%s\n\nProvide a clear, concise explanation of what this command accomplishes.", command)

	geminiReq := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
				Role: "user",
			},
		},
		SystemInstruction: &geminiSystemInstruction{
			Parts: []geminiPart{
				{Text: "You are a helpful assistant that explains shell commands clearly and concisely."},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     0.1,
			MaxOutputTokens: 300,
		},
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	var resp geminiResponse
	restResp, err := p.client.R().
		SetContext(ctx).
		SetBody(geminiReq).
		SetResult(&resp).
		Post(url)

	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: "Failed to call Gemini API",
			Cause:   err,
		}
	}

	if restResp.IsError() {
		return nil, p.handleAPIError(restResp, &resp)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, &Error{
			Type:    ErrorTypeModel,
			Message: "No response from Gemini",
		}
	}

	var usage *Usage
	if resp.UsageMetadata != nil {
		usage = &Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &Response{
		Command:     command,
		Explanation: strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text),
		Confidence:  1.0, // High confidence for explanations
		Usage:       usage,
	}, nil
}

// GetProviderInfo returns information about the Gemini provider
func (p *GeminiProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:    "Google AI",
		Version: "1.0.0",
		Models:  []string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.0-pro"},
		Capabilities: []string{
			"command_generation",
			"command_explanation",
			"context_awareness",
			"safety_filtering",
		},
		Limits: map[string]int{
			"max_tokens":      8192,
			"max_history":     10,
			"timeout_seconds": 30,
		},
		Metadata: map[string]string{
			"provider": "google",
			"model":    p.model,
		},
	}
}

// buildCommandPrompt constructs the prompt for command generation
func (p *GeminiProvider) buildCommandPrompt(request *Request) string {
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

	if request.Options.IncludeExplanation {
		parts = append(parts, "\nRespond with the command followed by a brief explanation separated by '||'.")
	} else {
		parts = append(parts, "\nRespond with only the shell command, no explanation.")
	}

	return strings.Join(parts, "\n")
}

// parseResponse extracts command and explanation from the response
func (p *GeminiProvider) parseResponse(content string, includeExplanation bool) (command, explanation string) {
	content = strings.TrimSpace(content)

	if includeExplanation && strings.Contains(content, "||") {
		parts := strings.SplitN(content, "||", 2)
		command = strings.TrimSpace(parts[0])
		explanation = strings.TrimSpace(parts[1])
	} else {
		command = content
	}

	// Clean up command (remove code block markers if present)
	command = strings.TrimPrefix(command, "```bash")
	command = strings.TrimPrefix(command, "```sh")
	command = strings.TrimPrefix(command, "```")
	command = strings.TrimSuffix(command, "```")
	command = strings.TrimSpace(command)

	return command, explanation
}

// calculateConfidence estimates confidence based on finish reason
func (p *GeminiProvider) calculateConfidence(finishReason string) float64 {
	switch finishReason {
	case "STOP":
		return 0.9
	case "MAX_TOKENS":
		return 0.7
	case "SAFETY":
		return 0.3
	case "RECITATION":
		return 0.4
	default:
		return 0.5
	}
}

// checkSafety performs basic safety checks on commands
func (p *GeminiProvider) checkSafety(command string) []string {
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

// handleAPIError converts Gemini API errors to our error format
func (p *GeminiProvider) handleAPIError(resp *resty.Response, apiResp *geminiResponse) error {
	if apiResp.Error != nil {
		var errorType ErrorType
		switch apiResp.Error.Code {
		case 400:
			errorType = ErrorTypeInvalidInput
		case 401, 403:
			errorType = ErrorTypeAuth
		case 429:
			errorType = ErrorTypeRateLimit
		case 500, 503:
			errorType = ErrorTypeModel
		default:
			errorType = ErrorTypeUnknown
		}

		return &Error{
			Type:    errorType,
			Message: apiResp.Error.Message,
			Code:    fmt.Sprintf("%d", apiResp.Error.Code),
		}
	}

	return &Error{
		Type:    ErrorTypeNetwork,
		Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode(), resp.String()),
	}
}
