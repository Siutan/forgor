package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"forgor/internal/cache"
	"forgor/internal/config"
	"forgor/internal/llm"
	"forgor/internal/utils"

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
		fmt.Println("Edit ~/.config/forgor/config.yaml to customize your settings")
		fmt.Println("Set your API keys in environment variables (e.g., OPENAI_API_KEY)")
		
		// Perform initial system scan
		fmt.Println("\n🔍 Scanning system for tools (this may take a moment)...")
		start := time.Now()
		
		tools, err := cache.ScanAndCacheTools()
		duration := time.Since(start)
		
		if err != nil {
			fmt.Printf("⚠️  Warning: Tool scan failed: %v\n", err)
		} else {
			fmt.Printf("✓ Scan completed in %v\n", duration)
			
			// Show detected tools
			if !tools.IsEmpty() {
				fmt.Println("\nDetected tools:")
				if len(tools.PackageManagers) > 0 {
					fmt.Printf("   • Package managers: %v\n", tools.PackageManagers)
				}
				if len(tools.Languages) > 0 {
					langs := make([]string, 0, len(tools.Languages))
					for _, l := range tools.Languages {
						langs = append(langs, l.Name)
					}
					fmt.Printf("   • Languages: %v\n", langs)
				}
				if len(tools.ContainerTools) > 0 {
					fmt.Printf("   • Container tools: %v\n", tools.ContainerTools)
				}
				if len(tools.CloudTools) > 0 {
					fmt.Printf("   • Cloud tools: %v\n", tools.CloudTools)
				}
			}
		}
		
		fmt.Println("\nInitialization complete!")
		fmt.Println("\nTry: forgor list all files")
	},
}

// configRefreshCmd represents the config refresh command
var configRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh system tool detection cache",
	Long: `Scans your system for available tools and updates the cache.
	
This command performs a comprehensive scan of your system to detect:
- Package managers (brew, apt, npm, pip, etc.)
- Programming languages (python, node, go, etc.)
- Development tools (git, docker, kubectl, etc.)
- Container and cloud tools

The scan typically takes 1-3 seconds depending on your system.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Scanning system for tools...")
		
		start := time.Now()
		
		// Force immediate refresh
		if err := utils.RefreshSystemContext(); err != nil {
			fmt.Printf("Refresh failed: %v\n", err)
			return
		}
		
		duration := time.Since(start)
		
		fmt.Printf("Cache refreshed in %v\n", duration)
		
		// Show what was detected
		tools := cache.LoadToolCacheOrEmpty(cache.ToolCacheMaxAge)
		if !tools.IsEmpty() {
			fmt.Printf("\nDetected %d tools:\n", tools.GetToolCount())
			
			if len(tools.PackageManagers) > 0 {
				fmt.Printf("   • Package managers: %v\n", tools.PackageManagers)
			}
			if len(tools.Languages) > 0 {
				langs := make([]string, 0, len(tools.Languages))
				for _, l := range tools.Languages {
					langs = append(langs, l.Name)
				}
				fmt.Printf("   • Languages: %v\n", langs)
			}
			if len(tools.DevelopmentTools) > 0 {
				devTools := make([]string, 0, len(tools.DevelopmentTools))
				for _, t := range tools.DevelopmentTools {
					devTools = append(devTools, t.Name)
				}
				fmt.Printf("   • Dev tools: %v\n", devTools)
			}
			if len(tools.ContainerTools) > 0 {
				fmt.Printf("   • Container tools: %v\n", tools.ContainerTools)
			}
			if len(tools.CloudTools) > 0 {
				fmt.Printf("   • Cloud tools: %v\n", tools.CloudTools)
			}
		}
		
		fmt.Println("\nTool cache updated successfully!")
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

// configCompletionCmd represents the config completion command
var configCompletionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Setup shell completion automatically",
	Long: `Automatically setup shell completion by adding the necessary lines to your shell configuration.

Supported shells: bash, zsh, fish

If no shell is specified, it will auto-detect your current shell.

Examples:
  forgor config completion         # Auto-detect and setup for current shell
  forgor config completion zsh     # Setup for zsh specifically
  forgor config completion bash    # Setup for bash specifically`,
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var targetShell string

		if len(args) > 0 {
			targetShell = args[0]
		} else {
			// Auto-detect shell
			shell := os.Getenv("SHELL")
			if shell == "" {
				return fmt.Errorf("could not detect shell. Please specify shell explicitly: forgor config completion [bash|zsh|fish]")
			}
			targetShell = filepath.Base(shell)
		}

		// Validate shell
		switch targetShell {
		case "bash", "zsh", "fish":
			// supported
		default:
			return fmt.Errorf("unsupported shell: %s. Supported shells: bash, zsh, fish", targetShell)
		}

		fmt.Printf("🚀 Setting up %s completion for forgor...\n\n", targetShell)

		return setupShellCompletion(targetShell)
	},
}

func setupShellCompletion(shell string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	switch shell {
	case "bash":
		return setupBashCompletion(homeDir)
	case "zsh":
		return setupZshCompletion(homeDir)
	case "fish":
		return setupFishCompletion(homeDir)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func setupBashCompletion(homeDir string) error {
	configFile := filepath.Join(homeDir, ".bashrc")

	// Check if .bash_profile exists and .bashrc doesn't, use .bash_profile instead (common on macOS)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		profileFile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(profileFile); err == nil {
			configFile = profileFile
		}
	}

	// Create completion file
	completionDir := filepath.Join(homeDir, ".config", "forgor")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return fmt.Errorf("failed to create completion directory: %w", err)
	}

	completionFile := filepath.Join(completionDir, "completion.bash")
	file, err := os.Create(completionFile)
	if err != nil {
		return fmt.Errorf("failed to create completion file: %w", err)
	}
	defer file.Close()

	// Generate completion script directly using Cobra
	if err := rootCmd.GenBashCompletion(file); err != nil {
		return fmt.Errorf("failed to generate bash completion: %w", err)
	}

	// Add sourcing line to shell config
	completionLine := fmt.Sprintf(`# forgor shell completion
if [ -f "%s" ]; then
    source "%s"
fi`, completionFile, completionFile)

	return addCompletionToFile(configFile, completionLine, "bash")
}

func setupZshCompletion(homeDir string) error {
	configFile := filepath.Join(homeDir, ".zshrc")

	// Create completion file
	completionDir := filepath.Join(homeDir, ".config", "forgor")
	if err := os.MkdirAll(completionDir, 0755); err != nil {
		return fmt.Errorf("failed to create completion directory: %w", err)
	}

	completionFile := filepath.Join(completionDir, "completion.zsh")
	file, err := os.Create(completionFile)
	if err != nil {
		return fmt.Errorf("failed to create completion file: %w", err)
	}
	defer file.Close()

	// Generate completion script directly using Cobra
	if err := rootCmd.GenZshCompletion(file); err != nil {
		return fmt.Errorf("failed to generate zsh completion: %w", err)
	}

	// Add sourcing line to shell config
	completionLine := fmt.Sprintf(`# forgor shell completion
if [ -f "%s" ]; then
    source "%s"
fi`, completionFile, completionFile)

	return addCompletionToFile(configFile, completionLine, "zsh")
}

func setupFishCompletion(homeDir string) error {
	// Fish uses a different approach - we create a completion file
	fishConfigDir := filepath.Join(homeDir, ".config", "fish", "completions")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(fishConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create fish completions directory: %w", err)
	}

	completionFile := filepath.Join(fishConfigDir, "forgor.fish")

	// Create completion file
	file, err := os.Create(completionFile)
	if err != nil {
		return fmt.Errorf("failed to create completion file: %w", err)
	}
	defer file.Close()

	// Generate completion script directly using Cobra
	if err := rootCmd.GenFishCompletion(file, true); err != nil {
		return fmt.Errorf("failed to generate fish completion: %w", err)
	}

	fmt.Printf("✅ Fish completion installed to %s\n", completionFile)
	fmt.Printf("🔄 Restart your fish shell or run: source %s\n", completionFile)

	return nil
}

func addCompletionToFile(configFile, completionLines, shell string) error {
	// Check if completion is already set up
	if isCompletionAlreadySetup(configFile) {
		fmt.Printf("✅ forgor completion is already set up in %s\n", configFile)
		return nil
	}

	// Create backup
	backupFile := configFile + ".forgor-backup"
	if err := copyFile(configFile, backupFile); err == nil {
		fmt.Printf("📋 Created backup: %s\n", backupFile)
	}

	// Add completion lines
	file, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", configFile, err)
	}
	defer file.Close()

	_, err = file.WriteString("\n" + completionLines + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to %s: %w", configFile, err)
	}

	fmt.Printf("✅ Added forgor completion to %s\n", configFile)
	fmt.Printf("🔄 Run 'source %s' or restart your %s shell to enable completion\n", configFile, shell)

	// Try to source the file automatically
	if shell == "bash" || shell == "zsh" {
		fmt.Printf("🚀 Attempting to source the file automatically...\n")
		cmd := exec.Command(shell, "-c", fmt.Sprintf("source %s", configFile))
		if err := cmd.Run(); err == nil {
			fmt.Printf("✨ Completion should now be active in your current session!\n")
		} else {
			fmt.Printf("⚠️  Could not auto-source. Please restart your shell or run: source %s\n", configFile)
		}
	}

	return nil
}

func isCompletionAlreadySetup(configFile string) bool {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return false
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		return false
	}

	return strings.Contains(string(content), "forgor completion")
}

func copyFile(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return err
	}

	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, input, 0644)
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetDefaultCmd)
	configCmd.AddCommand(configListProvidersCmd)
	configCmd.AddCommand(configToolsCmd)
	configCmd.AddCommand(configRefreshCmd)
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
