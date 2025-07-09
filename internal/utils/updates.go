package utils

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubRepo   = "Siutan/forgor"
	githubApiURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	// Security limits for extraction
	maxDecompressedSize = 1024 * 1024 * 100 // 100MB limit
	maxFileCount        = 1000              // max files in archive
)

// isValidURL validates if the URL is from an allowed domain
func isValidURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTPS
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	// Only allow GitHub domains for security
	allowedHosts := []string{
		"api.github.com",
		"github.com",
		"objects.githubusercontent.com",
	}

	for _, host := range allowedHosts {
		if parsedURL.Host == host {
			return nil
		}
	}

	return fmt.Errorf("URL host %s is not allowed", parsedURL.Host)
}

// sanitizePath validates and sanitizes file paths to prevent directory traversal
func sanitizePath(basePath, userPath string) (string, error) {
	// Clean the path to remove any .. or other traversal attempts
	cleanPath := filepath.Clean(userPath)

	// Check for absolute paths or paths that try to escape the base directory
	if filepath.IsAbs(cleanPath) || strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: %s", userPath)
	}

	// Join with base path and check it's still within the base
	fullPath := filepath.Join(basePath, cleanPath)
	if !strings.HasPrefix(fullPath, filepath.Clean(basePath)+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes base directory: %s", userPath)
	}

	return fullPath, nil
}

// secureRemoveAll safely removes a directory with error handling
func secureRemoveAll(path string) {
	if err := os.RemoveAll(path); err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("Warning: failed to clean up temporary directory %s: %v\n", path, err)
	}
}

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
	// Validate URL before making request
	if err := isValidURL(url); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

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
	// Validate URL before making request
	if err := isValidURL(url); err != nil {
		return "", fmt.Errorf("URL validation failed: %w", err)
	}

	resp, err := http.Get(url) // #nosec G107 - URL is validated above
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

	// Create the file in the temp directory with sanitized path
	fileName := filepath.Base(url)
	if fileName == "." || fileName == "/" {
		secureRemoveAll(tmpDir)
		return "", fmt.Errorf("invalid filename from URL")
	}

	filePath, err := sanitizePath(tmpDir, fileName)
	if err != nil {
		secureRemoveAll(tmpDir)
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	file, err := os.Create(filePath) // #nosec G304 - path is sanitized above
	if err != nil {
		// If file creation fails, we should clean up the temp directory
		secureRemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create file in temp dir: %w", err)
	}
	defer file.Close()

	// Write the downloaded content to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		// If copy fails, we should clean up the temp directory and file
		secureRemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write download to file: %w", err)
	}

	return file.Name(), nil
}

// ExtractTarGz extracts a gzipped tar file to a destination directory.
func ExtractTarGz(src, dest string) error {
	// Validate source path
	if !filepath.IsAbs(src) {
		return fmt.Errorf("source path must be absolute")
	}
	if !filepath.IsAbs(dest) {
		return fmt.Errorf("destination path must be absolute")
	}

	// Open the gzipped tar file
	r, err := os.Open(src) // #nosec G304 - path is validated above
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer r.Close()

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create the destination directory with secure permissions
	if err := os.MkdirAll(dest, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract the tar file with security checks
	tr := tar.NewReader(gzr)
	var totalSize int64
	fileCount := 0

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Security check: limit number of files
		fileCount++
		if fileCount > maxFileCount {
			return fmt.Errorf("archive contains too many files (limit: %d)", maxFileCount)
		}

		// Security check: limit total decompressed size
		totalSize += header.Size
		if totalSize > maxDecompressedSize {
			return fmt.Errorf("archive too large when decompressed (limit: %d bytes)", maxDecompressedSize)
		}

		// Sanitize the path to prevent directory traversal
		target, err := sanitizePath(dest, header.Name)
		if err != nil {
			return fmt.Errorf("invalid path in archive: %w", err)
		}

		// Check if the file is a directory
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0750); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Convert file mode safely to avoid integer overflow
		var fileMode os.FileMode
		if header.Mode >= 0 && header.Mode <= 0777 {
			fileMode = os.FileMode(header.Mode) & 0777
		} else {
			fileMode = 0644 // Default safe permissions
		}

		file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, fileMode) // #nosec G304 - path is sanitized above
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		// Copy with size limit to prevent decompression bombs
		limited := io.LimitReader(tr, header.Size)
		_, err = io.Copy(file, limited) // #nosec G110 - size is limited above
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}

	return nil
}
