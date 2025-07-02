package cmd

import (
	"fmt"

	"forgor/internal/config"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage forgor configuration",
	Long: `Manage forgor configuration files and profiles.

Examples:
  forgor config init    # Create default config file
  forgor config show    # Show current configuration`,
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
				if profile.APIKey == "${OPENAI_API_KEY}" || profile.APIKey == "${ANTHROPIC_API_KEY}" {
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

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
