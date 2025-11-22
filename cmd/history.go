package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"forgor/internal/daemon"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Browse recorded command history",
}

var historyListCmd = &cobra.Command{
	Use:   "list [n]",
	Short: "List last N recorded commands (default: all)",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.EnsureDaemonRunning(); err != nil {
			return err
		}
		n := 0
		if len(args) == 1 {
			v, err := strconv.Atoi(args[0])
			if err != nil || v < 0 {
				return fmt.Errorf("invalid number: %s", args[0])
			}
			n = v
		}
		c := daemon.NewClient()
		entries, err := c.QueryLast(n)
		if err != nil {
			return err
		}
		for _, e := range entries {
			fmt.Printf("ID: %s\n", e.ID)
			fmt.Printf("Command: %s\n", e.Command)
			fmt.Printf("Exit Code: %d\n", e.ExitCode)
			if e.Duration > 0 {
				fmt.Printf("Duration: %s\n", time.Duration(e.Duration))
			}
			if e.OutputPath != "" {
				fmt.Printf("Captured: yes (%d bytes)\n", e.OutputSize)
			} else {
				fmt.Println("Captured: no")
			}
			if !e.Timestamp.IsZero() {
				ago := time.Since(e.Timestamp)
				fmt.Printf("When: %s ago\n", humanizeDuration(ago))
			}
			fmt.Println()
		}
		return nil
	},
}

var historyOutputCmd = &cobra.Command{
	Use:   "output <id>",
	Short: "Show captured output for a command ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.EnsureDaemonRunning(); err != nil {
			return err
		}
		id := args[0]
		c := daemon.NewClient()
		entry, err := c.QueryByID(id)
		if err != nil {
			return err
		}
		if entry.OutputPath == "" {
			return fmt.Errorf("no output captured for %s", id)
		}
		data, err := os.ReadFile(entry.OutputPath)
		if err != nil {
			return err
		}
		fmt.Printf("%s", string(data))
		return nil
	},
}

func init() {
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyOutputCmd)
	rootCmd.AddCommand(historyCmd)
}

func humanizeDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
