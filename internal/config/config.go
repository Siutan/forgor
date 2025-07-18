package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the overall configuration structure
type Config struct {
	DefaultProfile string             `yaml:"default_profile" mapstructure:"default_profile"`
	Profiles       map[string]Profile `yaml:"profiles" mapstructure:"profiles"`
	History        HistoryConfig      `yaml:"history" mapstructure:"history"`
	Security       SecurityConfig     `yaml:"security" mapstructure:"security"`
	Output         OutputConfig       `yaml:"output" mapstructure:"output"`
	CustomTools    CustomToolsConfig  `yaml:"custom_tools" mapstructure:"custom_tools"`
}

// Profile represents an LLM provider profile
type Profile struct {
	Provider    string  `yaml:"provider" mapstructure:"provider"`
	APIKey      string  `yaml:"api_key" mapstructure:"api_key"`
	Model       string  `yaml:"model" mapstructure:"model"`
	MaxTokens   int     `yaml:"max_tokens" mapstructure:"max_tokens"`
	Temperature float64 `yaml:"temperature" mapstructure:"temperature"`
	Endpoint    string  `yaml:"endpoint,omitempty" mapstructure:"endpoint"`
}

// HistoryConfig represents shell history configuration
type HistoryConfig struct {
	MaxCommands int      `yaml:"max_commands" mapstructure:"max_commands"`
	Shells      []string `yaml:"shells" mapstructure:"shells"`
}

// SecurityConfig represents security and privacy settings
type SecurityConfig struct {
	RedactSensitive bool     `yaml:"redact_sensitive" mapstructure:"redact_sensitive"`
	Filters         []string `yaml:"filters" mapstructure:"filters"`
}

// CustomToolsConfig represents user-defined custom tools
type CustomToolsConfig struct {
	PackageManagers  []string `yaml:"package_managers" mapstructure:"package_managers"`
	Languages        []string `yaml:"languages" mapstructure:"languages"`
	DevelopmentTools []string `yaml:"development_tools" mapstructure:"development_tools"`
	SystemCommands   []string `yaml:"system_commands" mapstructure:"system_commands"`
	ContainerTools   []string `yaml:"container_tools" mapstructure:"container_tools"`
	CloudTools       []string `yaml:"cloud_tools" mapstructure:"cloud_tools"`
	DatabaseTools    []string `yaml:"database_tools" mapstructure:"database_tools"`
	NetworkTools     []string `yaml:"network_tools" mapstructure:"network_tools"`
	Other            []string `yaml:"other" mapstructure:"other"`
}

// OutputConfig represents output formatting configuration
type OutputConfig struct {
	Format           string `yaml:"format" mapstructure:"format"`
	ConfirmBeforeRun bool   `yaml:"confirm_before_run" mapstructure:"confirm_before_run"`
}

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	config := &Config{}

	// Set defaults
	setDefaults()

	// Unmarshal the configuration
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.DefaultProfile == "" {
		return fmt.Errorf("default_profile must be specified")
	}

	if _, exists := c.Profiles[c.DefaultProfile]; !exists {
		return fmt.Errorf("default profile '%s' not found in profiles", c.DefaultProfile)
	}

	for name, profile := range c.Profiles {
		if err := profile.Validate(); err != nil {
			return fmt.Errorf("invalid profile '%s': %w", name, err)
		}
	}

	return nil
}

// Validate checks if a profile configuration is valid
func (p *Profile) Validate() error {
	if p.Provider == "" {
		return fmt.Errorf("provider must be specified")
	}

	if p.Model == "" {
		return fmt.Errorf("model must be specified")
	}

	// Provider-specific validation
	switch p.Provider {
	case "openai", "anthropic", "gemini", "google":
		if p.APIKey == "" {
			return fmt.Errorf("api_key is required for %s provider", p.Provider)
		}
	case "local":
		if p.Endpoint == "" {
			return fmt.Errorf("endpoint is required for local provider")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", p.Provider)
	}

	return nil
}

// GetProfile returns the specified profile or the default profile
func (c *Config) GetProfile(name string) (Profile, error) {
	if name == "" || name == "default" {
		name = c.DefaultProfile
	}

	profile, exists := c.Profiles[name]
	if !exists {
		return Profile{}, fmt.Errorf("profile '%s' not found", name)
	}

	return profile, nil
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	defaultConfig := getDefaultConfig()

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created default config at %s\n", configPath)
	return nil
}

// setDefaults sets default values for viper
func setDefaults() {
	viper.SetDefault("default_profile", "openai")
	viper.SetDefault("history.max_commands", 10)
	viper.SetDefault("history.shells", []string{"bash", "zsh", "fish"})
	viper.SetDefault("security.redact_sensitive", true)
	viper.SetDefault("security.filters", []string{"password", "token", "secret", "key"})
	viper.SetDefault("output.format", "plain")
	viper.SetDefault("output.confirm_before_run", false)
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "forgor"), nil
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *Config {
	return &Config{
		DefaultProfile: "openai",
		Profiles: map[string]Profile{
			"openai": {
				Provider:    "openai",
				APIKey:      "${OPENAI_API_KEY}",
				Model:       "gpt-4",
				MaxTokens:   150,
				Temperature: 0.1,
			},
			"anthropic": {
				Provider:    "anthropic",
				APIKey:      "${ANTHROPIC_API_KEY}",
				Model:       "claude-3-sonnet-20240229",
				MaxTokens:   150,
				Temperature: 0.1,
			},
			"gemini": {
				Provider:    "gemini",
				APIKey:      "${GOOGLE_AI_API_KEY}",
				Model:       "gemini-1.5-pro",
				MaxTokens:   150,
				Temperature: 0.1,
			},
			"local": {
				Provider:  "local",
				Endpoint:  "http://localhost:11434",
				Model:     "codellama",
				MaxTokens: 150,
			},
		},
		History: HistoryConfig{
			MaxCommands: 10,
			Shells:      []string{"bash", "zsh", "fish"},
		},
		Security: SecurityConfig{
			RedactSensitive: true,
			Filters:         []string{"password", "token", "secret", "key"},
		},
		Output: OutputConfig{
			Format:           "plain",
			ConfirmBeforeRun: false,
		},
	}
}

// SaveConfig saves the configuration to the config file
func SaveConfig(config *Config) error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Validate the configuration before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveLastCommand saves the last generated command to cache
func SaveLastCommand(command string) error {
	if command == "" {
		return nil // Don't save empty commands
	}

	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cachePath := filepath.Join(configDir, "last_command")

	if err := os.WriteFile(cachePath, []byte(command), 0644); err != nil {
		return fmt.Errorf("failed to write last command cache: %w", err)
	}

	return nil
}

// LoadLastCommand loads the last generated command from cache
func LoadLastCommand() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	cachePath := filepath.Join(configDir, "last_command")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no previous command found. Generate a command first with: forgor \"your query\"")
		}
		return "", fmt.Errorf("failed to read last command cache: %w", err)
	}

	command := strings.TrimSpace(string(data))
	if command == "" {
		return "", fmt.Errorf("no previous command found")
	}

	return command, nil
}
