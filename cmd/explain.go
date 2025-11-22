package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"forgor/internal/daemon"

	"github.com/spf13/cobra"
)

var (
	explainFile string
	explainLast int
)

var explainCmd = &cobra.Command{
	Use:   "explain [query]",
	Short: "Explain a specific error or pattern",
	RunE: func(cmd *cobra.Command, args []string) error {
		var input string
		var source string
		if explainFile != "" {
			b, err := os.ReadFile(explainFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			input = string(b)
			source = explainFile
		} else if fi, _ := os.Stdin.Stat(); (fi.Mode() & os.ModeCharDevice) == 0 {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			input = string(b)
			source = "stdin"
		} else if len(args) > 0 {
			query := strings.Join(args, " ")
			return explainFromHistory(query)
		} else {
			return fmt.Errorf("no input provided (use argument, pipe, or --file)")
		}

		fmt.Printf("Analyzing error from %s\n\n", source)
		s := daemon.NewSanitizer()
		sanitized, _ := s.Sanitize(input)
		// For now, just echo sanitized content; LLM integration can be added later
		fmt.Println(sanitized)
		return nil
	},
}

func explainFromHistory(query string) error {
	if err := daemon.EnsureDaemonRunning(); err != nil {
		return err
	}
	c := daemon.NewClient()
	entries, err := c.QueryLast(explainLast)
	if err != nil {
		return err
	}
	var matches []*daemon.BufferEntry
	for _, e := range entries {
		if strings.Contains(e.Command, query) {
			matches = append(matches, e)
			continue
		}
		if e.OutputPath != "" {
			if b, err := os.ReadFile(e.OutputPath); err == nil {
				if strings.Contains(string(b), query) {
					matches = append(matches, e)
				}
			}
		}
	}
	if len(matches) == 0 {
		fmt.Printf("No commands found matching %q\n", query)
		return nil
	}
	fmt.Printf("Found %d matching command(s):\n\n", len(matches))
	for i, e := range matches {
		fmt.Printf("%d. %s (exit: %d)\n", i+1, e.Command, e.ExitCode)
	}
	return nil
}

func init() {
	explainCmd.Flags().StringVar(&explainFile, "file", "", "Analyze errors from file")
	explainCmd.Flags().IntVarP(&explainLast, "last", "n", 10, "Search last N commands")
	rootCmd.AddCommand(explainCmd)
}
