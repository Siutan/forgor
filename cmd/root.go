package cmd

import (
	"fmt"
	"os"

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
  ff -h 1 "fix the above command"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Main query logic will go here
		query := args[0]
		if verbose {
			fmt.Printf("Processing query: %s\n", query)
			fmt.Printf("Profile: %s, History: %d, Interactive: %v\n", profile, history, interactive)
		}

		// TODO: Implement actual query processing
		fmt.Printf("🧠 forgor: Query received - \"%s\"\n", query)
		fmt.Println("⚠️  Implementation coming soon! Check IMPLEMENTATION_PLAN.md for progress.")
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
