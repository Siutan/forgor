package llm

import (
	"fmt"
	"os"
	"strings"

	"forgor/internal/config"
)

// Factory manages the creation and selection of LLM providers
type Factory struct {
	providers map[string]Provider
	config    *config.Config
}

// NewFactory creates a new LLM provider factory
func NewFactory(cfg *config.Config) *Factory {
	return &Factory{
		providers: make(map[string]Provider),
		config:    cfg,
	}
}

// GetProvider returns a provider for the specified profile name
func (f *Factory) GetProvider(profileName string) (Provider, error) {
	if profileName == "" || profileName == "default" {
		profileName = f.config.DefaultProfile
	}

	// Check if provider already exists in cache
	if provider, exists := f.providers[profileName]; exists {
		return provider, nil
	}

	// Get profile configuration
	profile, err := f.config.GetProfile(profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile '%s': %w", profileName, err)
	}

	// Create provider based on configuration
	provider, err := f.createProvider(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider for profile '%s': %w", profileName, err)
	}

	// Cache the provider
	f.providers[profileName] = provider

	return provider, nil
}

// GetDefaultProvider returns the default provider
func (f *Factory) GetDefaultProvider() (Provider, error) {
	return f.GetProvider(f.config.DefaultProfile)
}

// ListProviders returns information about all configured providers
func (f *Factory) ListProviders() map[string]ProviderInfo {
	info := make(map[string]ProviderInfo)

	for name := range f.config.Profiles {
		provider, err := f.GetProvider(name)
		if err != nil {
			// If we can't create the provider, include error info
			info[name] = ProviderInfo{
				Name:    "Error",
				Version: "0.0.0",
				Models:  []string{},
				Metadata: map[string]string{
					"error": err.Error(),
				},
			}
			continue
		}
		info[name] = provider.GetProviderInfo()
	}

	return info
}

// ValidateProvider checks if a provider configuration is valid
func (f *Factory) ValidateProvider(profileName string) error {
	profile, err := f.config.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Basic validation
	if err := profile.Validate(); err != nil {
		return err
	}

	// Provider-specific validation
	switch profile.Provider {
	case "openai":
		return f.validateOpenAI(profile)
	case "anthropic":
		return f.validateAnthropic(profile)
	case "gemini", "google":
		return f.validateGemini(profile)
	default:
		return fmt.Errorf("unsupported provider: %s", profile.Provider)
	}
}

// createProvider creates a new provider instance based on the profile
func (f *Factory) createProvider(profile config.Profile) (Provider, error) {
	// Expand environment variables in API key
	apiKey := os.ExpandEnv(profile.APIKey)

	switch profile.Provider {
	case "openai":
		return NewOpenAIProvider(apiKey, profile.Model), nil

	case "anthropic":
		return NewAnthropicProvider(apiKey, profile.Model), nil

	case "gemini", "google":
		return NewGeminiProvider(apiKey, profile.Model), nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", profile.Provider)
	}
}

// validateOpenAI validates OpenAI provider configuration
func (f *Factory) validateOpenAI(profile config.Profile) error {
	apiKey := os.ExpandEnv(profile.APIKey)
	if apiKey == "" {
		return fmt.Errorf("openAI API key not found. Set OPENAI_API_KEY environment variable or add api_key to config")
	}

	validModels := []string{
		"gpt-4", "gpt-4-turbo", "gpt-4-turbo-preview",
		"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
	}

	if !contains(validModels, profile.Model) {
		return fmt.Errorf("invalid OpenAI model: %s. Valid models: %s",
			profile.Model, strings.Join(validModels, ", "))
	}

	return nil
}

// validateAnthropic validates Anthropic provider configuration
func (f *Factory) validateAnthropic(profile config.Profile) error {
	apiKey := os.ExpandEnv(profile.APIKey)
	if apiKey == "" {
		return fmt.Errorf("anthropic API key not found. Set ANTHROPIC_API_KEY environment variable or add api_key to config")
	}

	validModels := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	if !contains(validModels, profile.Model) {
		return fmt.Errorf("invalid Anthropic model: %s. Valid models: %s",
			profile.Model, strings.Join(validModels, ", "))
	}

	return nil
}

// validateGemini validates Google AI/Gemini provider configuration
func (f *Factory) validateGemini(profile config.Profile) error {
	apiKey := os.ExpandEnv(profile.APIKey)
	if apiKey == "" {
		return fmt.Errorf("google AI API key not found. Set GOOGLE_AI_API_KEY environment variable or add api_key to config")
	}

	validModels := []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
		"gemini-2.0-flash-exp",
		"gemini-2.5-flash-lite-preview-06-17",
		"gemini-exp-1114",
	}

	if !contains(validModels, profile.Model) {
		return fmt.Errorf("invalid Gemini model: %s. Valid models: %s",
			profile.Model, strings.Join(validModels, ", "))
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetSupportedProviders returns a list of all supported provider types
func GetSupportedProviders() []string {
	return []string{"openai", "anthropic", "gemini", "google"}
}

// GetDefaultModels returns default models for each provider type
func GetDefaultModels() map[string]string {
	return map[string]string{
		"openai":    "gpt-4.1",
		"anthropic": "claude-3.5-sonnet",
		"gemini":    "gemini-2.5-flash-lite-preview-06-17",
		"google":    "gemini-2.5-flash",
	}
}

// GetProviderCapabilities returns capabilities for each provider type
func GetProviderCapabilities() map[string][]string {
	return map[string][]string{
		"openai": {
			"command_generation",
			"command_explanation",
			"context_awareness",
			"safety_filtering",
		},
		"anthropic": {
			"command_generation",
			"command_explanation",
			"context_awareness",
			"safety_filtering",
			"advanced_reasoning",
		},
		"gemini": {
			"command_generation",
			"command_explanation",
			"context_awareness",
			"safety_filtering",
			"multimodal",
		},
		"google": {
			"command_generation",
			"command_explanation",
			"context_awareness",
			"safety_filtering",
			"multimodal",
		},
	}
}
