package cmd

import (
	"bufio"
	"fmt"
	"os"

	"forgor/internal/daemon"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage daemon process",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		d := daemon.NewDaemon()
		return d.Start()
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		d := daemon.NewDaemon()
		return d.Stop()
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := daemon.NewClient()
		// basic status by querying stats
		_, err := c.QueryLast(1)
		if err != nil {
			fmt.Println("Daemon is not running")
			return nil
		}
		fmt.Println("Daemon is running")
		return nil
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show daemon logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		// tail the log file (best-effort)
		logPath := daemon.DefaultDaemonConfig().LogPath
		f, err := os.Open(logPath)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
			if len(lines) > 1000 {
				lines = lines[1:]
			}
		}
		for _, l := range lines {
			fmt.Println(l)
		}
		return scanner.Err()
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	// hidden internal child command to run daemon in foreground
	childCmd := &cobra.Command{Use: "child", Hidden: true, RunE: func(cmd *cobra.Command, args []string) error {
		d := daemon.NewDaemon()
		return d.RunChild()
	}}
	daemonCmd.AddCommand(childCmd)
	rootCmd.AddCommand(daemonCmd)
}
