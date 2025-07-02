package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"forgor/internal/config"
	"forgor/internal/llm"
	"forgor/internal/utils"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	verbose     bool
	profile     string
	history     int
	interactive bool
	explain     bool
	format      string
	confirm     bool
	localOnly   bool
	forceRun    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "forgor",
	Short: "LLM-powered memory jogger for command line",
	Long: `forgor (ff) is a CLI tool that lets you type natural language prompts 
and get bash-friendly commands back, powered by an LLM.

Examples:
  forgor "find all txt files with hello in them"
  ff "show me how to make a new tmux session called dev"
  ff --history 2 "fix the above command"
  ff -R "list all files in current directory"  # Force run the generated command`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuery(args[0])
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/forgor/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Query flags
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "default", "config profile to use")
	rootCmd.Flags().IntVar(&history, "history", 0, "number of commands from history to include")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "interactive mode with follow-ups")
	rootCmd.Flags().BoolVarP(&explain, "explain", "e", false, "explain the command instead of just returning it")
	rootCmd.Flags().StringVarP(&format, "format", "f", "plain", "output format: plain, json")
	rootCmd.Flags().BoolVarP(&confirm, "confirm", "c", false, "ask for confirmation before showing command")
	rootCmd.Flags().BoolVar(&localOnly, "local-only", false, "don't send data to external APIs")

	// Execution flags (uppercase for potentially unsafe operations)
	rootCmd.Flags().BoolVarP(&forceRun, "force-run", "R", false, "immediately run the generated command (DANGEROUS)")

	// Bind flags to viper
	viper.BindPFlag("profile", rootCmd.Flags().Lookup("profile"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// runQuery processes a natural language query and generates a command
func runQuery(query string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		if verbose {
			fmt.Printf("Error loading config: %v\n", err)
			fmt.Println("💡 Run 'forgor config init' to create a default configuration")
		}
		// Create a minimal default config for basic functionality
		cfg = &config.Config{
			DefaultProfile: "openai",
			Profiles: map[string]config.Profile{
				"openai": {
					Provider:    "openai",
					APIKey:      "${OPENAI_API_KEY}",
					Model:       "gpt-4",
					MaxTokens:   150,
					Temperature: 0.1,
				},
			},
		}
	}

	if verbose {
		fmt.Printf("🔍 Processing query: %s\n", query)
		fmt.Printf("📋 Using profile: %s\n", profile)
	}

	// Create LLM factory
	factory := llm.NewFactory(cfg)

	// Get the provider
	provider, err := factory.GetProvider(profile)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	if verbose {
		info := provider.GetProviderInfo()
		fmt.Printf("🤖 Using %s with model %s\n", info.Name, info.Metadata["model"])
	}

	// Build request context
	ctx := context.Background()

	// Build enhanced context with tool detection
	requestContext := llm.BuildContextFromSystem()
	// TODO: Add command history when implemented
	requestContext = llm.EnhanceContextWithHistory(requestContext, []string{})

	request := &llm.Request{
		Query:   query,
		Context: requestContext,
		Options: llm.RequestOptions{
			MaxTokens:          150,
			Temperature:        0.1,
			IncludeExplanation: explain,
			SafetyLevel:        "moderate",
		},
	}

	// Override options from profile
	if profileConfig, err := cfg.GetProfile(profile); err == nil {
		if profileConfig.MaxTokens > 0 {
			request.Options.MaxTokens = profileConfig.MaxTokens
		}
		if profileConfig.Temperature >= 0 {
			request.Options.Temperature = profileConfig.Temperature
		}
	}

	if verbose {
		fmt.Printf("🌍 Context: %s on %s (%s) in %s\n",
			request.Context.Shell, request.Context.OS, request.Context.Architecture, request.Context.WorkingDirectory)
		if request.Context.ToolsSummary != "" {
			fmt.Printf("🔧 Tools: %s\n", request.Context.ToolsSummary)
		}
	}

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Generate command
	var response *llm.Response
	if explain {
		// If explain flag is set and query looks like a command, explain it instead
		if isLikelyCommand(query) {
			response, err = provider.ExplainCommand(ctx, query)
		} else {
			response, err = provider.GenerateCommand(ctx, request)
		}
	} else {
		response, err = provider.GenerateCommand(ctx, request)
	}

	if err != nil {
		return fmt.Errorf("failed to generate command: %w", err)
	}

	// Display results
	return displayResponse(response, explain)
}

// isLikelyCommand checks if the input looks like a shell command
func isLikelyCommand(input string) bool {
	// Simple heuristic: if it starts with common command patterns
	commandPrefixes := []string{
		"ls", "cd", "mkdir", "rm", "cp", "mv", "find", "grep", "cat", "echo",
		"ps", "kill", "top", "df", "du", "tar", "git", "docker", "npm", "pip",
		"sudo", "chmod", "chown", "ssh", "scp", "curl", "wget",
	}

	for _, prefix := range commandPrefixes {
		if len(input) >= len(prefix) && input[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// displayResponse formats and displays the LLM response
func displayResponse(response *llm.Response, isExplanation bool) error {
	// Handle explanation display
	if isExplanation {
		// Display explanation
		fmt.Printf("📖 Command explanation:\n")
		fmt.Printf("Command: %s\n\n", response.Command)
		fmt.Printf("%s\n", response.Explanation)

		// If force-run is also enabled, continue to execution
		if !forceRun {
			return nil
		}
		fmt.Println() // Add spacing before execution
	} else {
		// Display generated command with optional explanation
		if explain && response.Explanation != "" {
			fmt.Printf("💡 %s\n\n", response.Explanation)
		}
	}

	// Show warnings if any
	if len(response.Warnings) > 0 {
		fmt.Printf("⚠️  Warnings:\n")
		for _, warning := range response.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	// Show the command (unless we already showed it in explanation mode)
	if !isExplanation {
		fmt.Printf("%s\n", response.Command)
	}

	// Save the command to cache for later use with 'forgor run'
	if response.Command != "" {
		if err := config.SaveLastCommand(response.Command); err != nil && verbose {
			fmt.Printf("Warning: Failed to cache command: %v\n", err)
		}
	}

	// Show confidence and usage info in verbose mode
	if verbose {
		fmt.Printf("\n📊 Confidence: %.1f%%\n", response.Confidence*100)
		if response.Usage != nil {
			fmt.Printf("🔢 Tokens: %d prompt + %d completion = %d total\n",
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}
	}

	// Handle command execution
	if forceRun {
		fmt.Printf("\n🚀 Executing command...\n")
		return executeCommand(response.Command, response.Warnings)
	}

	// Offer to run the command (don't show if we're in explanation mode and not force-running)
	if !isExplanation && response.Command != "" {
		fmt.Printf("\n💡 Run this command? Use 'forgor run' or 'ff -R \"%s\"'\n",
			response.Command)
	}

	return nil
}

// executeCommand runs a shell command with safety checks
func executeCommand(command string, warnings []string) error {
	if command == "" {
		return fmt.Errorf("no command to execute")
	}

	// Safety checks
	if isDangerousCommand(command) {
		fmt.Printf("⚠️  DANGEROUS COMMAND DETECTED!\n")
		fmt.Printf("Command: %s\n", command)

		if !forceRun {
			fmt.Printf("This command may be destructive. Continue? (type 'yes' to confirm): ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}

			if strings.TrimSpace(strings.ToLower(response)) != "yes" {
				fmt.Printf("❌ Command execution cancelled\n")
				return nil
			}
		} else {
			fmt.Printf("⚠️  Force execution enabled - proceeding with dangerous command\n")
		}
	} else if !forceRun {
		// For non-dangerous commands, still ask for confirmation unless forced
		fmt.Printf("Execute: %s\n", command)
		fmt.Printf("Continue? [Y/n]: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "" && response != "y" && response != "yes" {
			fmt.Printf("❌ Command execution cancelled\n")
			return nil
		}
	}

	// Show warnings again before execution
	if len(warnings) > 0 {
		fmt.Printf("⚠️  Final warnings:\n")
		for _, warning := range warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	// Execute the command
	fmt.Printf("⚡ Executing: %s\n", command)
	fmt.Println("─────────────────────────────────────")

	cmd := exec.Command(utils.GetCurrentShell(), "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	fmt.Println("─────────────────────────────────────")

	if err != nil {
		fmt.Printf("❌ Command failed: %v\n", err)
		return err
	}

	fmt.Printf("✅ Command completed successfully\n")
	return nil
}

// isDangerousCommand checks if a command is potentially destructive
func isDangerousCommand(command string) bool {
	dangerousPatterns := []string{
		"rm -rf", "rm -fr", "rm -r", "rm -f",
		"sudo rm", "sudo delete", "sudo dd",
		"mkfs", "fdisk", "parted",
		"shutdown", "reboot", "halt",
		"chmod 777", "chmod -R 777",
		"chown -R", "find . -delete", "find / -delete",
		"> /dev/", "dd if=", "dd of=",
		"curl.*|.*sh", "wget.*|.*sh", "curl.*|.*bash", "wget.*|.*bash",
		":(){ :|:& };:", // fork bomb
		"mv /*", "cp /* ",
		"truncate", "shred",
	}

	command = strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".forgor" (without extension).
		viper.AddConfigPath(home + "/.config/forgor")
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
