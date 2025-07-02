package cmd

import (
	"context"
	"fmt"
	"os"
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
  ff --history 2 "fix the above command"`,
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
	request := &llm.Request{
		Query: query,
		Context: llm.Context{
			OS:               utils.GetOperatingSystem(),
			Shell:            utils.GetCurrentShell(),
			WorkingDirectory: utils.GetWorkingDirectory(),
			History:          []string{}, // TODO: Implement history reading
		},
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
		fmt.Printf("🌍 Context: %s on %s in %s\n",
			request.Context.Shell, request.Context.OS, request.Context.WorkingDirectory)
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
	if isExplanation {
		// Display explanation
		fmt.Printf("📖 Command explanation:\n")
		fmt.Printf("Command: %s\n\n", response.Command)
		fmt.Printf("%s\n", response.Explanation)
	} else {
		// Display generated command
		if explain && response.Explanation != "" {
			fmt.Printf("💡 %s\n\n", response.Explanation)
		}

		// Show warnings if any
		if len(response.Warnings) > 0 {
			fmt.Printf("⚠️  Warnings:\n")
			for _, warning := range response.Warnings {
				fmt.Printf("  • %s\n", warning)
			}
			fmt.Println()
		}

		fmt.Printf("%s\n", response.Command)
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

	return nil
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
