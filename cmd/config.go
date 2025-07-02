package cmd

import (
	"fmt"

	"forgor/internal/config"
	"forgor/internal/llm"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage forgor configuration",
	Long: `Manage forgor configuration files and profiles.

Examples:
  forgor config init                  # Create default config file
  forgor config show                  # Show current configuration
  forgor config set-default openai    # Set default provider
  forgor config list-providers        # List available providers`,
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Long:  `Create a default configuration file in ~/.config/forgor/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.CreateDefaultConfig(); err != nil {
			fmt.Printf("Error creating config: %v\n", err)
			return
		}
		fmt.Println("✅ Default configuration created successfully!")
		fmt.Println("📝 Edit ~/.config/forgor/config.yaml to customize your settings")
		fmt.Println("🔑 Set your API keys in environment variables (e.g., OPENAI_API_KEY)")
	},
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration including all profiles`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			fmt.Println("💡 Run 'forgor config init' to create a default configuration")
			return
		}

		fmt.Printf("📋 Current Configuration\n")
		fmt.Printf("Default Profile: %s\n\n", cfg.DefaultProfile)

		fmt.Printf("🔧 Profiles:\n")
		for name, profile := range cfg.Profiles {
			marker := ""
			if name == cfg.DefaultProfile {
				marker = " (default)"
			}
			fmt.Printf("  %s%s:\n", name, marker)
			fmt.Printf("    Provider: %s\n", profile.Provider)
			fmt.Printf("    Model: %s\n", profile.Model)
			if profile.APIKey != "" {
				if profile.APIKey == "${OPENAI_API_KEY}" || profile.APIKey == "${ANTHROPIC_API_KEY}" || profile.APIKey == "${GOOGLE_AI_API_KEY}" {
					fmt.Printf("    API Key: %s\n", profile.APIKey)
				} else {
					fmt.Printf("    API Key: %s***\n", profile.APIKey[:min(4, len(profile.APIKey))])
				}
			}
			if profile.Endpoint != "" {
				fmt.Printf("    Endpoint: %s\n", profile.Endpoint)
			}
			fmt.Printf("    Max Tokens: %d\n", profile.MaxTokens)
			fmt.Printf("    Temperature: %.1f\n\n", profile.Temperature)
		}

		fmt.Printf("📚 History: Max %d commands from %v shells\n",
			cfg.History.MaxCommands, cfg.History.Shells)
		fmt.Printf("🔒 Security: Redact sensitive data = %v\n", cfg.Security.RedactSensitive)
		fmt.Printf("📤 Output: Format = %s\n", cfg.Output.Format)
	},
}

// configSetDefaultCmd represents the config set-default command
var configSetDefaultCmd = &cobra.Command{
	Use:   "set-default <profile>",
	Short: "Set the default provider profile",
	Long: `Set which provider profile to use by default.

Examples:
  forgor config set-default openai     # Use OpenAI as default
  forgor config set-default anthropic  # Use Anthropic as default
  forgor config set-default gemini     # Use Gemini as default`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		// Load current config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check if profile exists
		if _, exists := cfg.Profiles[profileName]; !exists {
			fmt.Printf("❌ Profile '%s' not found\n\n", profileName)
			fmt.Printf("Available profiles:\n")
			for name := range cfg.Profiles {
				fmt.Printf("  • %s\n", name)
			}
			return fmt.Errorf("profile '%s' does not exist", profileName)
		}

		// Validate the profile before setting as default
		factory := llm.NewFactory(cfg)
		if err := factory.ValidateProvider(profileName); err != nil {
			fmt.Printf("⚠️  Warning: Profile '%s' has validation issues: %v\n", profileName, err)
			fmt.Printf("Setting as default anyway, but you may need to fix the configuration.\n\n")
		}

		// Update the default profile
		cfg.DefaultProfile = profileName

		// Save the updated config
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✅ Default provider set to '%s'\n", profileName)

		// Show provider info
		if provider, err := factory.GetProvider(profileName); err == nil {
			info := provider.GetProviderInfo()
			fmt.Printf("🤖 Using %s with model %s\n", info.Name, info.Metadata["model"])
		}

		return nil
	},
}

// configListProvidersCmd represents the config list-providers command
var configListProvidersCmd = &cobra.Command{
	Use:   "list-providers",
	Short: "List all available provider profiles",
	Long:  `Show all configured provider profiles with their status`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			fmt.Println("💡 Run 'forgor config init' to create a default configuration")
			return
		}

		factory := llm.NewFactory(cfg)

		fmt.Printf("📋 Available Provider Profiles\n\n")

		for name, profile := range cfg.Profiles {
			status := "✅"
			statusMsg := "Ready"

			// Check if this is the default
			isDefault := name == cfg.DefaultProfile

			// Validate the provider
			if err := factory.ValidateProvider(name); err != nil {
				status = "⚠️"
				statusMsg = fmt.Sprintf("Issue: %v", err)
			}

			defaultMarker := ""
			if isDefault {
				defaultMarker = " (default)"
			}

			fmt.Printf("%s %s%s\n", status, name, defaultMarker)
			fmt.Printf("    Provider: %s\n", profile.Provider)
			fmt.Printf("    Model: %s\n", profile.Model)
			fmt.Printf("    Status: %s\n", statusMsg)
			fmt.Printf("\n")
		}

		fmt.Printf("💡 Use 'forgor config set-default <profile>' to change the default\n")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetDefaultCmd)
	configCmd.AddCommand(configListProvidersCmd)
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
