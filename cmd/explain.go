package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"forgor/internal/security"

	"github.com/spf13/cobra"
)

var explainFile string

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
			input = strings.Join(args, " ")
			source = "argument"
		} else {
			return fmt.Errorf("no input provided (use argument, pipe, or --file)")
		}

		fmt.Printf("Analyzing error from %s\n\n", source)
		s := security.NewSanitizer()
		sanitized, _ := s.Sanitize(input)
		// For now, just echo sanitized content; LLM integration can be added later
		fmt.Println(sanitized)
		return nil
	},
}

func init() {
	explainCmd.Flags().StringVar(&explainFile, "file", "", "Analyze errors from file")
	rootCmd.AddCommand(explainCmd)
}
