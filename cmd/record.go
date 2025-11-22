package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Control command recording",
}

var recordStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Enable full output capture",
	RunE: func(cmd *cobra.Command, args []string) error {
		// For now this toggles mode via config in future work; placeholder
		fmt.Println("Full capture mode not yet configurable; use PTY session when available.")
		return nil
	},
}

var recordStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop output capture (keep metadata only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Reverting to metadata-only mode.")
		return nil
	},
}

var recordStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show recording status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Recording: metadata-only (lazy capture)")
		return nil
	},
}

func init() {
	recordCmd.AddCommand(recordStartCmd)
	recordCmd.AddCommand(recordStopCmd)
	recordCmd.AddCommand(recordStatusCmd)
	rootCmd.AddCommand(recordCmd)
}
