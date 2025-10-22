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
		bootstrapAge, toolsAge := utils.GetCacheAge()
		refreshing := utils.IsRefreshInProgress()
		cacheInfo := utils.GetCacheInfo()

		fmt.Printf("%s\n", utils.Box("SYSTEM CONTEXT CACHE STATUS", "", utils.StyleInfo))

		// Bootstrap cache status
		fmt.Printf("\n%s\n", utils.Styled("Bootstrap Cache:", utils.StyleInfo))
		if bootstrapAge == 0 {
			fmt.Printf("  %s Not available\n", utils.Styled("[STATUS]", utils.StyleWarning))
		} else {
			fmt.Printf("  %s Available (age: %v)\n", utils.Styled("[STATUS]", utils.StyleSuccess), bootstrapAge)
		}

		// Tool cache status
		fmt.Printf("\n%s\n", utils.Styled("Tool Cache:", utils.StyleInfo))
		if toolsAge == 0 {
			fmt.Printf("  %s No tools detected\n", utils.Styled("[STATUS]", utils.StyleWarning))
			fmt.Printf("  %s Run 'forgor config refresh' to scan\n", utils.Styled("[TIP]", utils.StyleInfo))
		} else {
			fmt.Printf("  %s Available\n", utils.Styled("[STATUS]", utils.StyleSuccess))
			fmt.Printf("  %s %v ago\n", utils.Styled("Last scan:", utils.StyleInfo), toolsAge)

			expiry := 24 * time.Hour

			if toolsAge < expiry {
				fmt.Printf("  %s Fresh\n", utils.Styled("Freshness:", utils.StyleSuccess))
			} else {
				fmt.Printf("  %s Stale (consider refresh)\n", utils.Styled("Freshness:", utils.StyleWarning))
			}
		}

		// Background refresh status
		fmt.Printf("\n%s\n", utils.Styled("Background Refresh:", utils.StyleInfo))
		if refreshing {
			fmt.Printf("  %s In progress\n", utils.Styled("[STATUS]", utils.StyleInfo))
		} else {
			fmt.Printf("  %s Idle\n", utils.Styled("[STATUS]", utils.StyleSubtle))
		}

		// Cache directory info
		if cacheDir, ok := cacheInfo["cache_dir"].(string); ok && cacheDir != "" {
			fmt.Printf("\n%s\n", utils.Divider("CACHE FILES", utils.StyleInfo))
			fmt.Printf("%s %s\n", utils.Styled("Directory:", utils.StyleInfo), cacheDir)
			
			if bootstrapExists, ok := cacheInfo["bootstrap_exists"].(bool); ok && bootstrapExists {
				if size, ok := cacheInfo["bootstrap_size"].(int64); ok {
					fmt.Printf("%s %.1f KB\n", utils.Styled("Bootstrap:", utils.StyleInfo), float64(size)/1024)
				}
			}
			
			if toolsExists, ok := cacheInfo["tools_exists"].(bool); ok && toolsExists {
				if size, ok := cacheInfo["tools_size"].(int64); ok {
					fmt.Printf("%s %.1f KB\n", utils.Styled("Tools:", utils.StyleInfo), float64(size)/1024)
				}
			}
		}

		fmt.Printf("\n%s Three-tier cache architecture for zero-latency command generation.\n",
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
		
		if cacheDir, ok := cacheInfo["cache_dir"].(string); ok {
			fmt.Printf("%s %s\n", utils.Styled("Cache Directory:", utils.StyleInfo), cacheDir)
		}

		// Bootstrap cache info
		fmt.Printf("\n%s\n", utils.Styled("Bootstrap Cache:", utils.StyleInfo))
		if exists, ok := cacheInfo["bootstrap_exists"].(bool); ok && exists {
			fmt.Printf("  %s Exists\n", utils.Styled("[STATUS]", utils.StyleSuccess))
			if size, ok := cacheInfo["bootstrap_size"].(int64); ok && size > 0 {
				fmt.Printf("  %s %.1f KB\n", utils.Styled("Size:", utils.StyleInfo), float64(size)/1024)
			}
		} else {
			fmt.Printf("  %s Not found (will be created)\n", utils.Styled("[STATUS]", utils.StyleWarning))
		}

		// Tool cache info
		fmt.Printf("\n%s\n", utils.Styled("Tool Cache:", utils.StyleInfo))
		if exists, ok := cacheInfo["tools_exists"].(bool); ok && exists {
			fmt.Printf("  %s Exists\n", utils.Styled("[STATUS]", utils.StyleSuccess))
			if size, ok := cacheInfo["tools_size"].(int64); ok && size > 0 {
				fmt.Printf("  %s %.1f KB\n", utils.Styled("Size:", utils.StyleInfo), float64(size)/1024)
			}
		} else {
			fmt.Printf("  %s Not found\n", utils.Styled("[STATUS]", utils.StyleWarning))
			fmt.Printf("  %s Run 'forgor config refresh' to scan tools\n", utils.Styled("[TIP]", utils.StyleInfo))
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
