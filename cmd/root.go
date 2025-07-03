package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// setupCompletions configures custom completion functions for flags
func setupCompletions() {
	// Profile completion - complete with available profiles from config
	rootCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load()
		if err != nil {
			// Return common defaults if config loading fails
			return []string{"default", "openai", "anthropic", "gemini"}, cobra.ShellCompDirectiveNoFileComp
		}

		var profiles []string
		for name := range cfg.Profiles {
			profiles = append(profiles, name)
		}
		// Always include "default" as an option
		if cfg.DefaultProfile != "" && cfg.DefaultProfile != "default" {
			profiles = append(profiles, "default")
		}

		return profiles, cobra.ShellCompDirectiveNoFileComp
	})

	// Format completion - complete with valid output formats
	rootCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"plain", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	// History completion - suggest reasonable values
	rootCmd.RegisterFlagCompletionFunc("history", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"0", "1", "2", "3", "5", "10"}, cobra.ShellCompDirectiveNoFileComp
	})
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

	// Set up custom completions
	setupCompletions()

	// Bind flags to viper
	viper.BindPFlag("profile", rootCmd.Flags().Lookup("profile"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// runQuery processes a natural language query and generates a command
func runQuery(query string) error {
	// Set verbose environment variable for system detection timing
	if verbose {
		os.Setenv("FORGOR_VERBOSE", "true")
	}

	// Initialize timing for the entire operation
	timer := utils.NewTimer("Command Execution", verbose)
	defer timer.PrintSummary()

	// Load configuration
	configStep := timer.StartStep("Config Loading")
	cfg, err := config.Load()
	if err != nil {
		configStep.EndWithResult("error - using defaults")
		if verbose {
			fmt.Printf("%s Error loading config: %v\n", utils.Styled("[ERROR]", utils.StyleError), err)
			fmt.Printf("%s Run 'forgor config init' to create a default configuration\n", utils.Styled("[TIP]", utils.StyleInfo))
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
	} else {
		configStep.EndWithResult("success")
	}

	if verbose {
		fmt.Printf("\n%s\n", utils.Divider("QUERY PROCESSING", utils.StyleInfo))
		fmt.Printf("%s %s\n", utils.Styled("Query:", utils.StyleInfo), query)
		fmt.Printf("%s %s\n", utils.Styled("Profile:", utils.StyleInfo), profile)
	}

	// Create LLM factory
	providerStep := timer.StartStep("Provider Setup")
	factory := llm.NewFactory(cfg)

	// Get the provider
	provider, err := factory.GetProvider(profile)
	if err != nil {
		providerStep.EndWithResult("error")
		return fmt.Errorf("failed to get provider: %w", err)
	}
	providerStep.EndWithResult("success")

	if verbose {
		info := provider.GetProviderInfo()
		fmt.Printf("%s %s with model %s\n",
			utils.Styled("Provider:", utils.StyleInfo),
			utils.Styled(info.Name, utils.StyleHighlight),
			utils.Styled(info.Metadata["model"], utils.StyleHighlight))
	}

	// Build request context
	ctx := context.Background()

	// Build enhanced context with tool detection
	contextStep := timer.StartStep("System Context Building")
	requestContext := llm.BuildContextFromSystem()
	contextStep.End()

	// Add command history
	historyStep := timer.StartStep("History Processing")
	requestContext = llm.EnhanceContextWithHistory(requestContext, []string{})
	historyStep.End()

	if verbose {
		fmt.Printf("\n%s\n", utils.Divider("SYSTEM CONTEXT", utils.StyleSubtle))
		fmt.Printf("%s %s on %s (%s) in %s\n",
			utils.Styled("Environment:", utils.StyleSubtle),
			requestContext.Shell,
			requestContext.OS,
			requestContext.Architecture,
			requestContext.WorkingDirectory)

		toolSummary := utils.GetToolContextSummary()
		fmt.Printf("%s %s\n", utils.Styled("Tools:", utils.StyleSubtle), toolSummary)
	}

	// Generate response
	llmStep := timer.StartStep("LLM API Request")
	response, err := provider.GenerateCommand(ctx, &llm.Request{
		Query:   query,
		Context: requestContext,
		Options: llm.RequestOptions{
			IncludeExplanation: explain,
			MaxTokens:          150,
		},
	})

	if err != nil {
		llmStep.EndWithResult("error")
		return fmt.Errorf("failed to generate command: %w", err)
	}
	llmStep.EndWithResult("success")

	// Display response
	displayStep := timer.StartStep("Response Display")
	err = displayResponse(response, explain)
	if err != nil {
		displayStep.EndWithResult("error")
		return err
	}
	displayStep.EndWithResult("success")

	return nil
}

//TODO: remove this function
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
		fmt.Printf("\n%s\n", utils.Box("COMMAND EXPLANATION", "", utils.StyleInfo))
		fmt.Printf("%s %s\n\n", utils.Styled("Command:", utils.StyleCommand), response.Command)
		fmt.Printf("%s\n", response.Explanation)

		// If force-run is also enabled, continue to execution
		if !forceRun {
			return nil
		}
		fmt.Println() // Add spacing before execution
	} else {
		// Display generated command with optional explanation
		if explain && response.Explanation != "" {
			fmt.Printf("\n%s %s\n", utils.Styled("Explanation:", utils.StyleInfo), response.Explanation)
		}
	}

	// Show warnings if any
	if len(response.Warnings) > 0 {
		fmt.Printf("\n%s\n", utils.Divider("WARNINGS", utils.StyleWarning))
		warningList := utils.List(response.Warnings, utils.StyleWarning)
		fmt.Printf("%s\n", warningList)
	}

	// Show the command (unless we already showed it in explanation mode)
	if !isExplanation {
		fmt.Printf("\n%s\n", utils.Divider("GENERATED COMMAND", utils.StyleCommand))
		fmt.Printf("%s\n", utils.SimpleBox(response.Command, utils.StyleCommand))
	}

	// Save the command to cache for later use with 'forgor run'
	if response.Command != "" {
		if err := config.SaveLastCommand(response.Command); err != nil && verbose {
			fmt.Printf("%s Failed to cache command: %v\n", utils.Styled("[WARNING]", utils.StyleWarning), err)
		}
	}

	// Show confidence and usage info in verbose mode
	if verbose {
		fmt.Printf("\n%s\n", utils.Divider("RESPONSE DETAILS", utils.StyleSubtle))

		// Confidence
		confidencePercent := response.Confidence * 100
		confidenceStyle := utils.StyleSuccess
		if confidencePercent < 70 {
			confidenceStyle = utils.StyleWarning
		} else if confidencePercent < 50 {
			confidenceStyle = utils.StyleError
		}
		fmt.Printf("%s %s\n",
			utils.Styled("Confidence:", utils.StyleSubtle),
			utils.Styled(fmt.Sprintf("%.1f%%", confidencePercent), confidenceStyle))

		// Token usage
		if response.Usage != nil {
			fmt.Printf("%s %d prompt + %d completion = %d total\n",
				utils.Styled("Tokens:", utils.StyleSubtle),
				response.Usage.PromptTokens,
				response.Usage.CompletionTokens,
				response.Usage.TotalTokens)
		}
	}

	// Handle command execution
	if forceRun {
		fmt.Printf("\n%s\n", utils.Divider("EXECUTING COMMAND", utils.StyleCommand))
		return executeCommand(response.Command, response.Warnings)
	}

	// Offer to run the command (don't show if we're in explanation mode and not force-running)
	if !isExplanation && response.Command != "" {
		fmt.Printf("\n%s\n", utils.Divider("NEXT STEPS", utils.StyleInfo))
		fmt.Printf("%s Use '%s' or '%s'\n",
			utils.Styled("Run this command?", utils.StyleInfo),
			utils.Styled("forgor run", utils.StyleCommand),
			utils.Styled(fmt.Sprintf("ff -R \"%s\"", response.Command), utils.StyleCommand))
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
