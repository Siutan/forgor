package cmd

import (
	"fmt"

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
		fmt.Printf("forgor version %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build date: %s\n", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
