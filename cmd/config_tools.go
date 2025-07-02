package cmd

import (
	"fmt"
	"forgor/internal/config"
	"forgor/internal/utils"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Note: configCmd is defined in config.go

// configToolsCmd represents the config tools command
var configToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage custom tools configuration",
	Long:  `Manage custom tools that should be included in system detection.`,
}

// configCacheCmd represents the config cache command
var configCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage system context cache",
	Long:  `Manage the system context cache for better performance.`,
}

// configToolsListCmd lists custom tools
var configToolsListCmd = &cobra.Command{
	Use:   "list [category]",
	Short: "List custom tools",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var category string
		if len(args) > 0 {
			category = args[0]
		}

		return config.ListCustomTools(category)
	},
}

// configToolsAddCmd adds custom tools
var configToolsAddCmd = &cobra.Command{
	Use:   "add <category> <tool1,tool2,...>",
	Short: "Add custom tools to a category",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		category := args[0]
		toolsStr := args[1]

		// Parse comma-separated tools
		toolsList := strings.Split(toolsStr, ",")
		for i, tool := range toolsList {
			toolsList[i] = strings.TrimSpace(tool)
		}

		err := config.AddCustomTools(category, toolsList)
		if err != nil {
			return fmt.Errorf("failed to add tools: %w", err)
		}

		// Trigger background cache refresh to include new tools
		if verbose {
			fmt.Println("🔄 Triggering background cache refresh...")
		}
		utils.RefreshSystemContextBackground()

		return nil
	},
}

// configToolsRemoveCmd removes custom tools
var configToolsRemoveCmd = &cobra.Command{
	Use:   "remove <category> <tool1,tool2,...>",
	Short: "Remove custom tools from a category",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		category := args[0]
		toolsStr := args[1]

		// Parse comma-separated tools
		toolsList := strings.Split(toolsStr, ",")
		for i, tool := range toolsList {
			toolsList[i] = strings.TrimSpace(tool)
		}

		err := config.RemoveCustomTools(category, toolsList)
		if err != nil {
			return fmt.Errorf("failed to remove tools: %w", err)
		}

		// Trigger background cache refresh to update tools list
		if verbose {
			fmt.Println("🔄 Triggering background cache refresh...")
		}
		utils.RefreshSystemContextBackground()

		return nil
	},
}

// configToolsClearCmd clears custom tools
var configToolsClearCmd = &cobra.Command{
	Use:   "clear <category|all>",
	Short: "Clear custom tools from a category or all categories",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		err := config.ClearCustomTools(target)
		if err != nil {
			return fmt.Errorf("failed to clear tools: %w", err)
		}

		// Trigger background cache refresh to update tools list
		if verbose {
			fmt.Println("🔄 Triggering background cache refresh...")
		}
		utils.RefreshSystemContextBackground()

		return nil
	},
}

// configToolsCategoriesCmd lists available tool categories
var configToolsCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List available tool categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		categories := []string{
			"package_managers", "languages", "development_tools",
			"system_commands", "container_tools", "cloud_tools",
			"database_tools", "network_tools", "other",
		}

		fmt.Println("Available tool categories:")
		for _, category := range categories {
			fmt.Printf("  • %s\n", category)
		}

		return nil
	},
}

// configCacheStatusCmd shows cache status
var configCacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system context cache status",
	RunE: func(cmd *cobra.Command, args []string) error {
		age := utils.GetCacheAge()
		refreshing := utils.IsRefreshInProgress()
		cacheInfo := utils.GetCacheInfo()

		fmt.Printf("%s\n", utils.Box("SYSTEM CONTEXT CACHE STATUS", "", utils.StyleInfo))

		if age == 0 {
			fmt.Printf("%s No cache available\n", utils.Styled("[STATUS]", utils.StyleError))
		} else {
			fmt.Printf("%s Cache available\n", utils.Styled("[STATUS]", utils.StyleSuccess))
			fmt.Printf("%s %v\n", utils.Styled("Age:", utils.StyleInfo), age)

			expiry := 5 * time.Minute
			grace := 1 * time.Minute

			if age < expiry {
				remaining := expiry - age
				fmt.Printf("%s Fresh (expires in %v)\n",
					utils.Styled("Freshness:", utils.StyleSuccess), remaining)
			} else if age < expiry+grace {
				fmt.Printf("%s Stale but usable (refresh window)\n",
					utils.Styled("Freshness:", utils.StyleWarning))
			} else {
				fmt.Printf("%s Expired (will rebuild on next use)\n",
					utils.Styled("Freshness:", utils.StyleError))
			}
		}

		if refreshing {
			fmt.Printf("%s In progress\n", utils.Styled("Background Refresh:", utils.StyleInfo))
		} else {
			fmt.Printf("%s Idle\n", utils.Styled("Background Refresh:", utils.StyleSubtle))
		}

		fmt.Printf("%s %v\n", utils.Styled("Cache Expiry:", utils.StyleSubtle), 5*time.Minute)
		fmt.Printf("%s %v\n", utils.Styled("Grace Period:", utils.StyleSubtle), 1*time.Minute)

		if cacheInfo.FilePath != "" {
			fmt.Printf("\n%s\n", utils.Divider("PERSISTENT CACHE", utils.StyleInfo))
			fmt.Printf("%s %s\n", utils.Styled("Location:", utils.StyleInfo), cacheInfo.FilePath)
			if cacheInfo.FileSize > 0 {
				fmt.Printf("%s %.1f KB\n", utils.Styled("Size:", utils.StyleInfo), float64(cacheInfo.FileSize)/1024)
			}
			if !cacheInfo.FileModTime.IsZero() {
				fmt.Printf("%s %v\n", utils.Styled("Last Modified:", utils.StyleInfo),
					cacheInfo.FileModTime.Format("2006-01-02 15:04:05"))
			}
		}

		fmt.Printf("\n%s Cache persists between command invocations for optimal performance.\n",
			utils.Styled("[INFO]", utils.StyleInfo))

		return nil
	},
}

// configCacheRefreshCmd forces cache refresh
var configCacheRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force refresh of system context cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		background, _ := cmd.Flags().GetBool("background")

		if background {
			fmt.Printf("%s Starting background cache refresh...\n", utils.Styled("[INFO]", utils.StyleInfo))
			utils.RefreshSystemContextBackground()
			fmt.Printf("%s Background refresh initiated\n", utils.Styled("[SUCCESS]", utils.StyleSuccess))
		} else {
			fmt.Printf("%s Refreshing system context cache...\n", utils.Styled("[INFO]", utils.StyleInfo))
			start := time.Now()
			utils.RefreshSystemContext()
			duration := time.Since(start)
			fmt.Printf("%s Cache refreshed in %v\n", utils.Styled("[SUCCESS]", utils.StyleSuccess), duration)
			fmt.Printf("%s Updated persistent cache file\n", utils.Styled("[INFO]", utils.StyleInfo))
		}

		return nil
	},
}

// configCacheClearCmd clears the cache
var configCacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the system context cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := utils.ClearPersistentCache()
		if err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		fmt.Printf("%s System context cache cleared\n", utils.Styled("[SUCCESS]", utils.StyleSuccess))
		fmt.Printf("%s Next command will rebuild cache from scratch\n", utils.Styled("[INFO]", utils.StyleInfo))
		return nil
	},
}

// configCacheLocationCmd shows cache file location
var configCacheLocationCmd = &cobra.Command{
	Use:   "location",
	Short: "Show cache file location",
	RunE: func(cmd *cobra.Command, args []string) error {
		cacheInfo := utils.GetCacheInfo()

		fmt.Printf("%s\n", utils.Box("CACHE FILE LOCATIONS", "", utils.StyleInfo))
		fmt.Printf("%s %s\n", utils.Styled("Cache Directory:", utils.StyleInfo), cacheInfo.CacheDir)
		fmt.Printf("%s %s\n", utils.Styled("Cache File:", utils.StyleInfo), cacheInfo.FilePath)
		fmt.Printf("%s %s\n", utils.Styled("Lock File:", utils.StyleInfo), cacheInfo.LockFile)

		if cacheInfo.FileExists {
			fmt.Printf("\n%s Cache file exists\n", utils.Styled("[STATUS]", utils.StyleSuccess))
			if cacheInfo.FileSize > 0 {
				fmt.Printf("%s %.1f KB\n", utils.Styled("Size:", utils.StyleInfo), float64(cacheInfo.FileSize)/1024)
			}
		} else {
			fmt.Printf("\n%s Cache file does not exist\n", utils.Styled("[STATUS]", utils.StyleError))
		}

		return nil
	},
}

func init() {
	// Add to existing configCmd (defined in config.go)
	configCmd.AddCommand(configToolsCmd)
	configCmd.AddCommand(configCacheCmd)

	// Tools subcommands
	configToolsCmd.AddCommand(configToolsCategoriesCmd)
	configToolsCmd.AddCommand(configToolsAddCmd)
	configToolsCmd.AddCommand(configToolsRemoveCmd)
	configToolsCmd.AddCommand(configToolsListCmd)
	configToolsCmd.AddCommand(configToolsClearCmd)

	// Cache subcommands
	configCacheCmd.AddCommand(configCacheStatusCmd)
	configCacheCmd.AddCommand(configCacheRefreshCmd)
	configCacheCmd.AddCommand(configCacheClearCmd)
	configCacheCmd.AddCommand(configCacheLocationCmd)

	// Flags
	configCacheRefreshCmd.Flags().BoolP("background", "b", false, "Refresh in background")
}
