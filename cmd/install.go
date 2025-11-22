package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"forgor/internal/daemon"
	"forgor/internal/utils"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install shell integration",
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Forgor Installation Wizard")
	fmt.Println()

	shell := os.Getenv("SHELL")
	shellName := filepath.Base(shell)
	fmt.Printf("1. Detecting shell: %s\n", shellName)

	var hookScript string
	switch shellName {
	case "zsh":
		hookScript = "scripts/hooks/zsh-hook.zsh"
	case "bash":
		hookScript = "scripts/hooks/bash-hook.bash"
	default:
		return fmt.Errorf("unsupported shell: %s (supported: zsh, bash)", shellName)
	}

	rcFile := getRCFilePath(shellName)
	if rcFile == "" {
		return fmt.Errorf("could not locate shell rc file")
	}

	fmt.Print("2. Checking for existing installation...")
	if isAlreadyInstalled(rcFile) {
		fmt.Println(" found")
		if !utils.PromptYesNo("Reinstall?", false) {
			return fmt.Errorf("installation cancelled")
		}
	} else {
		fmt.Println(" none")
	}

	// Backup
	backupPath := rcFile + ".backup-forgor"
	fmt.Printf("3. Backing up shell config to %s\n", backupPath)
	if err := installCopyFile(rcFile, backupPath); err != nil {
		// best-effort backup
		_ = err
	}

	// Install hook
	fmt.Println("4. Installing shell hooks...")
	if err := installHook(rcFile, hookScript); err != nil {
		return fmt.Errorf("failed to install hook: %w", err)
	}

	// Ensure config dir
	cfgDir := filepath.Join(userHome(), ".config", "forgor")
	if err := os.MkdirAll(filepath.Join(cfgDir, "hooks"), 0o700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Copy hook into config dir
	hookDest := filepath.Join(cfgDir, "hooks", filepath.Base(hookScript))
	if err := installCopyFile(hookScript, hookDest); err != nil {
		return fmt.Errorf("failed to copy hook: %w", err)
	}

	// Start daemon
	fmt.Println("5. Starting daemon...")
	d := daemon.NewDaemon()
	if err := d.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Simple integration test prompt
	fmt.Println()
	fmt.Println("6. Test: run a command like 'ls' in your shell and press Enter here")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Println("Installation complete. Restart your shell: exec $SHELL")
	return nil
}

func getRCFilePath(shell string) string {
	switch shell {
	case "zsh":
		return filepath.Join(userHome(), ".zshrc")
	case "bash":
		return filepath.Join(userHome(), ".bashrc")
	default:
		return ""
	}
}

func isAlreadyInstalled(rcFile string) bool {
	b, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "forgor")
}

func installHook(rcFile, hookScript string) error {
	cfgDir := filepath.Join(userHome(), ".config", "forgor")
	hookPath := filepath.Join(cfgDir, "hooks", filepath.Base(hookScript))
	loader := fmt.Sprintf("\n# Added by Forgor - do not edit manually\nif [ -f \"%s\" ]; then\n  . \"%s\"\nfi\n", hookPath, hookPath)
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(loader)
	return err
}

// copyFile duplicates a file (local helper name adjusted to avoid conflict with cmd/config.go)
func installCopyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func userHome() string {
	h, _ := os.UserHomeDir()
	return h
}
