package llm

import (
	"context"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// GenerateCommand generates a shell command from a natural language query
	GenerateCommand(ctx context.Context, request *Request) (*Response, error)

	// ExplainCommand explains what a command does
	ExplainCommand(ctx context.Context, command string) (*Response, error)

	// GetProviderInfo returns information about the provider
	GetProviderInfo() ProviderInfo
}

// Request represents a query to the LLM
type Request struct {
	// The user's natural language query
	Query string `json:"query"`

	// Context information
	Context Context `json:"context"`

	// Options for this specific request
	Options RequestOptions `json:"options"`
}

// Context provides environmental context for the command generation
type Context struct {
	// Operating system (linux, darwin, windows)
	OS string `json:"os"`

	// Shell type (bash, zsh, fish, etc.)
	Shell string `json:"shell"`

	// Current working directory
	WorkingDirectory string `json:"working_directory"`

	// Recent command history
	History []string `json:"history,omitempty"`

	// Additional context from user
	UserContext string `json:"user_context,omitempty"`

	// System architecture
	Architecture string `json:"architecture,omitempty"`

	// User information
	User string `json:"user,omitempty"`

	// Home directory
	HomeDirectory string `json:"home_directory,omitempty"`

	// Available package managers
	PackageManagers []string `json:"package_managers,omitempty"`

	// Available programming languages
	Languages []string `json:"languages,omitempty"`

	// Available development tools
	DevelopmentTools []string `json:"development_tools,omitempty"`

	// Available container tools
	ContainerTools []string `json:"container_tools,omitempty"`

	// Available cloud tools
	CloudTools []string `json:"cloud_tools,omitempty"`

	// Available database tools
	DatabaseTools []string `json:"database_tools,omitempty"`

	// Available network tools
	NetworkTools []string `json:"network_tools,omitempty"`

	// Tool availability map for quick lookups
	ToolsAvailable map[string]bool `json:"tools_available,omitempty"`

	// Tool context summary for prompts
	ToolsSummary string `json:"tools_summary,omitempty"`

	// Relevant environment variables
	Environment map[string]string `json:"environment,omitempty"`
}

// RequestOptions contains options for the request
type RequestOptions struct {
	// Maximum tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature for randomness (0.0 to 1.0)
	Temperature float64 `json:"temperature,omitempty"`

	// Whether to include explanations
	IncludeExplanation bool `json:"include_explanation,omitempty"`

	// Safety level (strict, moderate, permissive)
	SafetyLevel string `json:"safety_level,omitempty"`
}

// Response represents the LLM's response
type Response struct {
	// The generated command
	Command string `json:"command"`

	// Explanation of what the command does
	Explanation string `json:"explanation,omitempty"`

	// Alternative commands
	Alternatives []string `json:"alternatives,omitempty"`

	// Confidence score (0.0 to 1.0)
	Confidence float64 `json:"confidence,omitempty"`

	// Danger level assessment
	DangerLevel DangerLevel `json:"danger_level,omitempty"`

	// Reason for the danger assessment
	DangerReason string `json:"danger_reason,omitempty"`

	// Safety warnings
	Warnings []string `json:"warnings,omitempty"`

	// Provider-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Token usage information
	Usage *Usage `json:"usage,omitempty"`
}

// DangerLevel represents the assessed danger level of a command
type DangerLevel string

const (
	DangerLevelSafe     DangerLevel = "safe"     // Safe command, no risks
	DangerLevelLow      DangerLevel = "low"      // Minor risks, generally safe
	DangerLevelMedium   DangerLevel = "medium"   // Moderate risks, requires caution
	DangerLevelHigh     DangerLevel = "high"     // High risks, potentially destructive
	DangerLevelCritical DangerLevel = "critical" // Critical risks, very dangerous
)

// GetDangerLevelValue returns the numeric value for danger level comparison
func GetDangerLevelValue(level DangerLevel) int {
	switch level {
	case DangerLevelSafe:
		return 0
	case DangerLevelLow:
		return 1
	case DangerLevelMedium:
		return 2
	case DangerLevelHigh:
		return 3
	case DangerLevelCritical:
		return 4
	default:
		return 0 // Default to safe
	}
}

// IsAtLeastLevel checks if the current danger level is at least the specified level
func (d DangerLevel) IsAtLeastLevel(level DangerLevel) bool {
	return GetDangerLevelValue(d) >= GetDangerLevelValue(level)
}

// DangerAssessment contains the result of command danger analysis
type DangerAssessment struct {
	Level       DangerLevel `json:"level"`
	Confidence  float64     `json:"confidence"`
	Reason      string      `json:"reason"`
	Factors     []string    `json:"factors,omitempty"`
	Mitigations []string    `json:"mitigations,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderInfo contains information about the provider
type ProviderInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Models       []string          `json:"models"`
	Capabilities []string          `json:"capabilities"`
	Limits       map[string]int    `json:"limits"`
	Metadata     map[string]string `json:"metadata"`
}

// Error types for LLM operations
type Error struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    string    `json:"code,omitempty"`
	Cause   error     `json:"-"`
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ErrorType represents different types of LLM errors
type ErrorType string

const (
	ErrorTypeAuth         ErrorType = "auth"          // Authentication/authorization errors
	ErrorTypeRateLimit    ErrorType = "rate_limit"    // Rate limiting errors
	ErrorTypeQuota        ErrorType = "quota"         // Quota exceeded errors
	ErrorTypeNetwork      ErrorType = "network"       // Network/connectivity errors
	ErrorTypeTimeout      ErrorType = "timeout"       // Request timeout errors
	ErrorTypeInvalidInput ErrorType = "invalid_input" // Invalid input errors
	ErrorTypeModel        ErrorType = "model"         // Model-specific errors
	ErrorTypeUnknown      ErrorType = "unknown"       // Unknown errors
	ErrorTypeSafety       ErrorType = "safety"        // Safety/content filtering errors
)
