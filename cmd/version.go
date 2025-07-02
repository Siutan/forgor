package cmd

import (
	"fmt"
	"os"
	"time"

	"forgor/internal/utils"

	"github.com/spf13/cobra"
)

var (
	// Version info - these will be set by build flags
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version, git commit, and build date of forgor`,
	Run: func(cmd *cobra.Command, args []string) {
		showVersion()
	},
}

func showVersion() {
	fmt.Printf("\n%s\n", utils.Divider("FORGOR VERSION INFO", utils.StyleInfo))

	// Parse build date for better formatting
	var buildTime string
	if BuildDate != "unknown" {
		if t, err := time.Parse(time.RFC3339, BuildDate); err == nil {
			buildTime = t.Format("2006-01-02 15:04:05 UTC")
		} else {
			buildTime = BuildDate
		}
	} else {
		buildTime = "unknown"
	}

	// Display version info in a table format
	headers := []string{"Info", "Value"}
	rows := [][]string{
		{utils.Styled("Version", utils.StyleHighlight), utils.Styled(Version, utils.StyleSuccess)},
		{utils.Styled("Git Commit", utils.StyleHighlight), GitCommit},
		{utils.Styled("Build Date", utils.StyleHighlight), buildTime},
		{utils.Styled("Go Version", utils.StyleHighlight), getGoVersion()},
		{utils.Styled("Platform", utils.StyleHighlight), getPlatform()},
	}

	fmt.Printf("%s\n", utils.Table(headers, rows, utils.StyleInfo))

	// Show installation info
	if execPath, err := os.Executable(); err == nil {
		fmt.Printf("%s %s\n",
			utils.Styled("Executable:", utils.StyleSubtle),
			utils.Styled(execPath, utils.StyleSubtle))
	}

	// Show latest version info if available
	showUpdateInfo()

	fmt.Println()
}

func getGoVersion() string {
	// This will be filled in by the Go runtime
	return "go version not available"
}

func getPlatform() string {
	return fmt.Sprintf("%s/%s", getOS(), getArch())
}

func getOS() string {
	switch {
	case isWindows():
		return "windows"
	case isDarwin():
		return "darwin"
	case isLinux():
		return "linux"
	default:
		return "unknown"
	}
}

func getArch() string {
	// This is a simplified approach - in a real implementation,
	// you might want to use runtime.GOARCH
	return "amd64"
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}

func isDarwin() bool {
	if stat, err := os.Stat("/System/Library/CoreServices/SystemVersion.plist"); err == nil && !stat.IsDir() {
		return true
	}
	return false
}

func isLinux() bool {
	if stat, err := os.Stat("/proc/version"); err == nil && !stat.IsDir() {
		return true
	}
	return false
}

func showUpdateInfo() {
	// Check if this is a development version
	if Version == "dev" || Version == "unknown" {
		fmt.Printf("\n%s %s\n",
			utils.Styled("ℹ️", utils.StyleInfo),
			utils.Styled("Development version - built from source", utils.StyleSubtle))
		return
	}

	// In a future enhancement, you could add update checking here
	fmt.Printf("\n%s %s\n",
		utils.Styled("💡", utils.StyleInfo),
		utils.Styled("Check for updates at: https://github.com/YOURUSERNAME/forgor/releases", utils.StyleSubtle))
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
