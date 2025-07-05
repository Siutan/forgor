package utils

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubRepo   = "Siutan/forgor"
	githubApiURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
)

// ReleaseInfo holds information about a GitHub release
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// httpGet performs a GET request and returns the response body
func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "forgor-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status from %s: %s\n%s", url, resp.Status, string(body))
	}

	return io.ReadAll(resp.Body)
}

// GetLatestVersion returns the latest version info of forgor from GitHub.
func GetLatestVersion() (*ReleaseInfo, error) {
	body, err := httpGet(githubApiURL)
	if err != nil {
		return nil, err
	}

	var release ReleaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse json response: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("could not find latest version tag in GitHub response")
	}
	// GitHub tags are often prefixed with 'v', so we remove it for comparison
	release.TagName = strings.TrimPrefix(release.TagName, "v")

	return &release, nil
}

// CheckForUpdates checks for updates to forgor and prints a message to the console.
// This is intended for non-interactive checks, like in the 'version' command.
func CheckForUpdates(currentVersion string) {
	if currentVersion == "dev" || currentVersion == "unknown" {
		fmt.Printf("\n%s %s\n",
			Styled("ℹ️", StyleInfo),
			Styled("Development version - update check skipped.", StyleSubtle))
		return
	}

	latestRelease, err := GetLatestVersion()
	if err != nil {
		// Non-blocking, just print a warning.
		fmt.Printf("\n%s Could not check for updates: %s\n", Styled("[WARN]", StyleWarning), err)
		return
	}

	// Using semantic version comparison would be better, but direct comparison is a good start.
	if latestRelease.TagName == currentVersion {
		fmt.Printf("\n%s Forgor is up to date (version %s)\n", Styled("✅", StyleSuccess), currentVersion)
		return
	}

	fmt.Printf("\n%s A new version of forgor is available: %s (current: %s)\n",
		Styled("🔄", StyleInfo),
		Styled(latestRelease.TagName, StyleSuccess),
		Styled(currentVersion, StyleWarning))
	fmt.Printf("   To update, run: %s\n", Styled("forgor update", StyleCommand))
}

// DownloadUpdate downloads a file from a URL to a new temporary directory and returns the path to the downloaded file.
func DownloadUpdate(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to perform GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status from %s: %s", url, resp.Status)
	}

	// Create a temporary directory for the update
	tmpDir, err := os.MkdirTemp("", "forgor-update-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create the file in the temp directory
	filePath := filepath.Join(tmpDir, filepath.Base(url))
	file, err := os.Create(filePath)
	if err != nil {
		// If file creation fails, we should clean up the temp directory
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create file in temp dir: %w", err)
	}
	defer file.Close()

	// Write the downloaded content to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		// If copy fails, we should clean up the temp directory and file
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write download to file: %w", err)
	}

	return file.Name(), nil
}

// ExtractTarGz extracts a gzipped tar file to a destination directory.
func ExtractTarGz(src, dest string) error {
	// Open the gzipped tar file
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer r.Close()

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create the destination directory
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract the tar file
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Check if the file is a directory
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(filepath.Join(dest, header.Name), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Extract the file
		target := filepath.Join(dest, header.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(file, tr)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}
