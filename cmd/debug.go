package cmd

import (
	"encoding/json"
	"fmt"

	"forgor/internal/daemon"

	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Show daemon state and recent buffer info",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.EnsureDaemonRunning(); err != nil {
			return err
		}
		client := daemon.NewClient()
		// Get stats via query
		// Reuse QueryLast(0) to fetch all for now; then summarize
		entries, err := client.QueryLast(0)
		if err != nil {
			return err
		}
		// Basic summary
		fmt.Printf("Total entries: %d\n", len(entries))
		failed := 0
		withOutput := 0
		for _, e := range entries {
			if e.ExitCode != 0 {
				failed++
			}
			if e.OutputPath != "" {
				withOutput++
			}
		}
		fmt.Printf("Failed: %d\n", failed)
		fmt.Printf("With output: %d\n", withOutput)
		// Print last 5 entries JSON for debugging
		n := 5
		if len(entries) < n {
			n = len(entries)
		}
		tail := entries[len(entries)-n:]
		b, _ := json.MarshalIndent(tail, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
}
