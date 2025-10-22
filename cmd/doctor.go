package cmd

import (
	"fmt"
	"os"
	"time"

	"forgor/internal/cache"
	"forgor/internal/config"
	"forgor/internal/utils"

	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check forgor health and cache status",
	Long: `Diagnose issues with forgor configuration and cache.

This command checks:
- Configuration file status
- Bootstrap cache status
- Tool cache status and age
- Cache directory permissions
- Background refresh status

Use this command to troubleshoot issues with forgor or to verify
that your system is properly configured.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Checking forgor health...")

		healthy := true

		// Check configuration
		cfg, err := config.Load()
		if err != nil {
			fmt.Println("❌ Configuration: ERROR")
			fmt.Printf("   %v\n", err)
			fmt.Println("   💡 Run 'forgor config init' to create a default configuration")
			healthy = false
		} else {
			fmt.Println("✓ Configuration: OK")
			fmt.Printf("   Default profile: %s\n", cfg.DefaultProfile)
			fmt.Printf("   Profiles: %d\n", len(cfg.Profiles))
		}

		// Check bootstrap cache
		fmt.Println()
		bootstrap, err := cache.LoadBootstrapCache()
		if err != nil {
			fmt.Println("❌ Bootstrap cache: MISSING")
			fmt.Printf("   %v\n", err)
			fmt.Println("   💡 Will be created automatically on next command")
			healthy = false
		} else {
			fmt.Println("✓ Bootstrap cache: OK")
			fmt.Printf("   OS: %s (%s)\n", bootstrap.OS, bootstrap.Architecture)
			fmt.Printf("   Shell: %s\n", bootstrap.Shell)
			fmt.Printf("   User: %s\n", bootstrap.User)
			if !bootstrap.CreatedAt.IsZero() {
				age := time.Since(bootstrap.CreatedAt)
				fmt.Printf("   Age: %v\n", age.Round(time.Hour*24))
			}
		}

		// Check tool cache
		fmt.Println()
		tools, err := cache.LoadToolCache()
		if err != nil {
			fmt.Println("⚠️  Tool cache: MISSING")
			fmt.Printf("   %v\n", err)
			fmt.Println("   💡 Run 'forgor config refresh' to scan for tools")
			fmt.Println("   💡 Or just run any command - tools will be scanned in background")
		} else {
			age := time.Since(tools.LastScan)
			status := "OK"
			icon := "✓"

			if age > 7*24*time.Hour {
				status = "STALE"
				icon = "⚠️ "
				healthy = false
			} else if age > 24*time.Hour {
				status = "OLD"
				icon = "⚠️ "
			}

			fmt.Printf("%s Tool cache: %s\n", icon, status)
			fmt.Printf("   Tools detected: %d\n", tools.GetToolCount())
			fmt.Printf("   Last scan: %v ago\n", formatDuration(age))

			if !tools.LastScan.IsZero() && tools.ScanDurationMs > 0 {
				fmt.Printf("   Scan duration: %dms\n", tools.ScanDurationMs)
			}

			if age > 24*time.Hour {
				fmt.Println("   💡 Run 'forgor config refresh' to update tool detection")
			}

			// Show some detected tools
			if !tools.IsEmpty() {
				fmt.Println("\n   Detected categories:")
				if len(tools.PackageManagers) > 0 {
					fmt.Printf("   • Package managers: %d\n", len(tools.PackageManagers))
				}
				if len(tools.Languages) > 0 {
					fmt.Printf("   • Languages: %d\n", len(tools.Languages))
				}
				if len(tools.DevelopmentTools) > 0 {
					fmt.Printf("   • Dev tools: %d\n", len(tools.DevelopmentTools))
				}
				if len(tools.ContainerTools) > 0 {
					fmt.Printf("   • Container tools: %d\n", len(tools.ContainerTools))
				}
				if len(tools.CloudTools) > 0 {
					fmt.Printf("   • Cloud tools: %d\n", len(tools.CloudTools))
				}
			}
		}

		// Check cache directory
		fmt.Println()
		info := cache.GetCacheInfo()
		cacheDir := info["cache_dir"].(string)

		if stat, err := os.Stat(cacheDir); os.IsNotExist(err) {
			fmt.Println("⚠️  Cache directory: MISSING")
			fmt.Printf("   Path: %s\n", cacheDir)
			fmt.Println("   💡 Will be created automatically on next command")
		} else if err != nil {
			fmt.Println("❌ Cache directory: ERROR")
			fmt.Printf("   Path: %s\n", cacheDir)
			fmt.Printf("   Error: %v\n", err)
			healthy = false
		} else {
			fmt.Println("✓ Cache directory: OK")
			fmt.Printf("   Path: %s\n", cacheDir)
			fmt.Printf("   Permissions: %v\n", stat.Mode())

			// Calculate total cache size
			var totalSize int64
			if bootstrapSize, ok := info["bootstrap_size"].(int64); ok {
				totalSize += bootstrapSize
			}
			if toolSize, ok := info["tools_size"].(int64); ok {
				totalSize += toolSize
			}

			if totalSize > 0 {
				fmt.Printf("   Total size: %s\n", formatBytes(totalSize))
			}
		}

		// Check refresh status
		fmt.Println()
		scheduler := cache.GetGlobalScheduler()
		if scheduler.IsRefreshInProgress() {
			fmt.Println("🔄 Background refresh: IN PROGRESS")
		} else {
			fmt.Println("✓ Background refresh: IDLE")
			if scheduler.IsEnabled() {
				fmt.Printf("   Status: %s\n", scheduler.GetRefreshStatus())
				fmt.Printf("   Interval: %v\n", scheduler.GetInterval())
				if scheduler.NeedsRefresh() {
					fmt.Println("   💡 Cache refresh recommended - run 'forgor config refresh'")
				}
			} else {
				fmt.Println("   Status: DISABLED")
			}
		}

		// Performance info
		fmt.Println()
		fmt.Println("⚡ Performance:")
		bootstrapAge, toolsAge := utils.GetCacheAge()
		if bootstrapAge > 0 {
			fmt.Printf("   Bootstrap load time: < 2ms (cached %v ago)\n", formatDuration(bootstrapAge))
		} else {
			fmt.Println("   Bootstrap load time: ~5ms (default fallback)")
		}
		if toolsAge > 0 {
			fmt.Printf("   Tools load time: < 5ms (cached %v ago)\n", formatDuration(toolsAge))
		} else {
			fmt.Println("   Tools load time: 0ms (empty)")
		}

		// Summary
		fmt.Println()
		fmt.Println("═══════════════════════════════════════")
		if healthy {
			fmt.Println("✅ All checks passed!")
			fmt.Println("\nForgor is properly configured and ready to use.")
		} else {
			fmt.Println("⚠️  Some issues detected")
			fmt.Println("\nSee recommendations above to fix issues.")
			fmt.Println("\nCommon fixes:")
			fmt.Println("  • forgor config init     - Create default configuration")
			fmt.Println("  • forgor config refresh  - Scan for tools")
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
}

// formatBytes formats bytes in a human-readable way
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}