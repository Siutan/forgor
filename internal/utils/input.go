package utils

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// PromptYesNo displays a prompt and returns on single key press 'y' or 'n'.
// If defaultYes is true, Enter or any other key returns true; otherwise false.
func PromptYesNo(prompt string, defaultYes bool) bool {
	var suffix string
	if defaultYes {
		suffix = " [Y/n] "
	} else {
		suffix = " [y/N] "
	}
	fmt.Print(prompt)
	fmt.Print(suffix)
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// fallback to normal scan
		var s string
		fmt.Scanln(&s)
		if s == "y" || s == "Y" {
			return true
		}
		if s == "n" || s == "N" {
			return false
		}
		return defaultYes
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	b := make([]byte, 1)
	if _, err := os.Stdin.Read(b); err != nil {
		return defaultYes
	}
	switch b[0] {
	case 'y', 'Y':
		fmt.Println("y")
		return true
	case 'n', 'N':
		fmt.Println("n")
		return false
	case '\r', '\n':
		fmt.Println("")
		return defaultYes
	default:
		fmt.Println("")
		return defaultYes
	}
}
