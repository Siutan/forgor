package cmd

import (
	"fmt"
	"os"
	"time"

	"forgor/internal/daemon"
	"forgor/internal/utils"

	"github.com/spf13/cobra"
)

var (
	wtfLast   int
	wtfDryRun bool
	wtfFull   bool
)

var wtfCmd = &cobra.Command{
	Use:   "wtf",
	Short: "Explain the last failed command",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.EnsureDaemonRunning(); err != nil {
			return err
		}
		c := daemon.NewClient()
		entries, err := c.QueryLast(wtfLast)
		if err != nil {
			return err
		}
		var failed *daemon.BufferEntry
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].ExitCode != 0 {
				failed = entries[i]
				break
			}
		}
		if failed == nil {
			fmt.Println("No failed commands found")
			return nil
		}

		var output string
		if failed.OutputPath != "" {
			b, err := os.ReadFile(failed.OutputPath)
			if err != nil {
				return err
			}
			output = string(b)
		} else {
			// ask user to rerun using immediate key input, avoid sensitive
			if daemon.NewSanitizer().DetectSensitiveCommand(failed.Command) {
				fmt.Println("Cannot capture output automatically for sensitive command. Rerun manually if safe.")
				return nil
			}
			if utils.PromptYesNo("Rerun command to capture output?", false) {
				fmt.Println("Capturing output via rerun...")
				if _, err := c.Capture(failed.ID); err != nil {
					return fmt.Errorf("capture failed: %w", err)
				}
				updated, err := c.QueryByID(failed.ID)
				if err != nil {
					return err
				}
				if updated.OutputPath == "" {
					return fmt.Errorf("capture produced no output")
				}
				b, err := os.ReadFile(updated.OutputPath)
				if err != nil {
					return err
				}
				output = string(b)
				failed = updated
			} else {
				fmt.Println("Skipped capture.")
				return nil
			}
		}

		if wtfDryRun {
			fmt.Println("Command:")
			fmt.Println(failed.Command)
			fmt.Println()
			fmt.Println("Output:")
			if !wtfFull && len(output) > 500 {
				fmt.Print(output[:500])
				fmt.Println("\n... (truncated)")
			} else {
				fmt.Print(output)
			}
			return nil
		}

		// Sanitize output before any display/LLM
		s := daemon.NewSanitizer()
		sanitized, detections := s.Sanitize(output)
		output = sanitized
		if len(detections) > 0 {
			fmt.Println("Sensitive data detected and redacted.")
		}

		// Build a simple prompt for now; LLM integration exists elsewhere in the codebase
		prompt := fmt.Sprintf("I ran this command and it failed. Explain and suggest fixes.\n\nCommand: %s\nExit Code: %d\nWorking Directory: %s\nShell: %s\n\nOutput:\n%s\n",
			failed.Command, failed.ExitCode, failed.CWD, failed.Shell, output)

		// Call the existing LLM pipeline through root flow when available; placeholder here
		_ = prompt
		// For now, display summary and captured output path to aid debugging
		fmt.Println("Analysis request prepared. Integrate with LLM providers to complete.")
		fmt.Printf("Command: %s\n", failed.Command)
		fmt.Printf("Exit: %d\n", failed.ExitCode)
		fmt.Printf("Output path: %s\n", failed.OutputPath)
		fmt.Printf("When: %s ago\n", humanizeDuration(time.Since(failed.Timestamp)))
		return nil
	},
}

func init() {
	wtfCmd.Flags().IntVarP(&wtfLast, "last", "n", 1, "Number of commands to analyze")
	wtfCmd.Flags().BoolVar(&wtfDryRun, "dry-run", false, "Show what would be sent without calling LLM")
	wtfCmd.Flags().BoolVar(&wtfFull, "full", false, "Include full output (no truncation)")
	rootCmd.AddCommand(wtfCmd)
}
