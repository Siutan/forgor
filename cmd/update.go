package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"forgor/internal/utils"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update forgor to the latest version",
	Long:  `Checks for the latest release of forgor on GitHub and, if a newer version is found, downloads and installs it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	fmt.Printf("Checking for new releases of forgor...\n")

	// Get the latest release information from GitHub
	latestRelease, err := utils.GetLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	// Compare versions
	currentVersion := Version
	if latestRelease.TagName == currentVersion {
		fmt.Printf("✅ You are already using the latest version of forgor: %s\n", utils.Styled(currentVersion, utils.StyleSuccess))
		return nil
	}

	fmt.Printf("A new version is available: %s (current: %s)\n",
		utils.Styled(latestRelease.TagName, utils.StyleSuccess),
		utils.Styled(currentVersion, utils.StyleWarning),
	)

	// Find the correct asset for the current OS and architecture
	binaryName := fmt.Sprintf("forgor_%s_%s", runtime.GOOS, runtime.GOARCH)

	var assetURL, assetName string
	for _, asset := range latestRelease.Assets {
		compareAssetName := strings.TrimSuffix(asset.Name, ".tar.gz")
		if strings.EqualFold(compareAssetName, binaryName) {
			assetURL = asset.DownloadURL
			assetName = asset.Name
			break
		}
	}

	if assetURL == "" {
		return fmt.Errorf("could not find a release asset for your system: %s", binaryName)
	}

	// 1. Ask for user confirmation.
	fmt.Printf("Update Forgor to the latest version? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "" && response != "y" && response != "yes" {
		fmt.Printf("❌ Update cancelled\n")
		return nil
	}

	// 2. Download the binary from assetURL into a temp directory.
	fmt.Printf("⬇️  Downloading %s...\n", utils.Styled(assetName, utils.StyleHighlight))
	downloadedArchivePath, err := utils.DownloadUpdate(assetURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	// Ensure the temporary directory is cleaned up
	tempDir := filepath.Dir(downloadedArchivePath)
	defer os.RemoveAll(tempDir)

	// 3. Unzip/untar if necessary.
	fmt.Printf("📦 Extracting archive...\n")
	err = utils.ExtractTarGz(downloadedArchivePath, tempDir)
	if err != nil {
		return fmt.Errorf("failed to extract update: %w", err)
	}

	// 4. Replace the current executable with the new one.
	currentExec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find current executable path: %w", err)
	}

	newExecPath := filepath.Join(tempDir, "forgor")
	fmt.Printf("🚀 Replacing current version at %s...\n", utils.Styled(currentExec, utils.StyleSubtle))
	err = os.Rename(newExecPath, currentExec)
	if err != nil {
		return fmt.Errorf("failed to replace executable (you may need to run with sudo or as an administrator): %w", err)
	}

	fmt.Printf("✅ Forgor has been successfully updated to version %s!\n", utils.Styled(latestRelease.TagName, utils.StyleSuccess))
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
