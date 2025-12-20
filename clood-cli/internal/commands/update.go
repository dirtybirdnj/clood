package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dirtybirdnj/clood/internal/output"
	"github.com/dirtybirdnj/clood/internal/tui"
	"github.com/spf13/cobra"
)

const (
	repoOwner = "dirtybirdnj"
	repoName  = "clood"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
	PublishedAt string `json:"published_at"`
}

// CurrentVersion is set by main.go
var CurrentVersion = "dev"

func UpdateCmd() *cobra.Command {
	var checkOnly bool
	var force bool

	cmd := &cobra.Command{
		Use:   "update [version]",
		Short: "Update clood to the latest version",
		Long: `Update clood by downloading the latest release from GitHub.

Examples:
  clood update            # Update to latest version
  clood update --check    # Check for updates without installing
  clood update v0.3.0     # Update to specific version
  clood update --force    # Force reinstall even if current`,
		Run: func(cmd *cobra.Command, args []string) {
			targetVersion := ""
			if len(args) > 0 {
				targetVersion = args[0]
			}

			if checkOnly {
				runUpdateCheck(targetVersion)
				return
			}

			runUpdate(targetVersion, force)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for updates without installing")
	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already on latest")

	return cmd
}

func runUpdateCheck(targetVersion string) {
	currentVersion := CurrentVersion

	release, err := getRelease(targetVersion)
	if err != nil {
		if output.IsJSON() {
			fmt.Printf(`{"error": %q}`, err.Error())
		} else {
			fmt.Println(tui.ErrorStyle.Render("Error checking for updates: " + err.Error()))
		}
		return
	}

	if output.IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"current_version": currentVersion,
			"latest_version":  release.TagName,
			"update_available": release.TagName != currentVersion,
			"published_at":    release.PublishedAt,
		}, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println()
	fmt.Println(tui.RenderHeader("Update Check"))
	fmt.Println()
	fmt.Printf("  Current version: %s\n", currentVersion)
	fmt.Printf("  Latest version:  %s\n", release.TagName)
	fmt.Printf("  Published:       %s\n", formatPublishedDate(release.PublishedAt))
	fmt.Println()

	if release.TagName == currentVersion {
		fmt.Println(tui.SuccessStyle.Render("✓ You're on the latest version"))
	} else {
		fmt.Println(tui.WarningStyle.Render("⬆ Update available"))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Run 'clood update' to install"))
	}
}

func runUpdate(targetVersion string, force bool) {
	currentVersion := CurrentVersion

	if !output.IsJSON() {
		fmt.Println()
		fmt.Println(tui.RenderHeader("clood update"))
		fmt.Println()
	}

	// Get release info
	if !output.IsJSON() {
		fmt.Print("Checking for updates... ")
	}

	release, err := getRelease(targetVersion)
	if err != nil {
		if output.IsJSON() {
			fmt.Printf(`{"error": %q}`, err.Error())
		} else {
			fmt.Println(tui.ErrorStyle.Render("FAILED"))
			fmt.Println(tui.ErrorStyle.Render("Error: " + err.Error()))
		}
		return
	}

	if !output.IsJSON() {
		fmt.Println(tui.SuccessStyle.Render("OK"))
	}

	// Check if update needed
	if release.TagName == currentVersion && !force {
		if output.IsJSON() {
			fmt.Printf(`{"status": "up_to_date", "version": %q}`, currentVersion)
		} else {
			fmt.Println()
			fmt.Println(tui.SuccessStyle.Render("✓ Already on latest version: " + currentVersion))
		}
		return
	}

	// Find the right asset for this platform
	binaryName := getBinaryName()
	var downloadURL string
	var assetSize int64

	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			assetSize = asset.Size
			break
		}
	}

	if downloadURL == "" {
		if output.IsJSON() {
			fmt.Printf(`{"error": "No binary found for %s"}`, binaryName)
		} else {
			fmt.Println(tui.ErrorStyle.Render("No binary available for " + binaryName))
			fmt.Println(tui.MutedStyle.Render("Available binaries:"))
			for _, asset := range release.Assets {
				if strings.HasPrefix(asset.Name, "clood-") {
					fmt.Printf("  - %s\n", asset.Name)
				}
			}
		}
		return
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		if output.IsJSON() {
			fmt.Printf(`{"error": %q}`, err.Error())
		} else {
			fmt.Println(tui.ErrorStyle.Render("Cannot find current executable: " + err.Error()))
		}
		return
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	// Download new binary
	if !output.IsJSON() {
		fmt.Printf("Downloading %s (%.1f MB)... ", release.TagName, float64(assetSize)/1024/1024)
	}

	tempFile, err := downloadBinary(downloadURL)
	if err != nil {
		if output.IsJSON() {
			fmt.Printf(`{"error": %q}`, err.Error())
		} else {
			fmt.Println(tui.ErrorStyle.Render("FAILED"))
			fmt.Println(tui.ErrorStyle.Render("Download failed: " + err.Error()))
		}
		return
	}
	defer os.Remove(tempFile)

	if !output.IsJSON() {
		fmt.Println(tui.SuccessStyle.Render("OK"))
	}

	// Replace current binary
	if !output.IsJSON() {
		fmt.Print("Installing... ")
	}

	if err := replaceBinary(tempFile, execPath); err != nil {
		if output.IsJSON() {
			fmt.Printf(`{"error": %q}`, err.Error())
		} else {
			fmt.Println(tui.ErrorStyle.Render("FAILED"))
			fmt.Println()
			fmt.Println(tui.ErrorStyle.Render("Installation failed: " + err.Error()))
			fmt.Println()
			fmt.Println(tui.MutedStyle.Render("Try with sudo:"))
			fmt.Printf("  sudo mv %s %s\n", tempFile, execPath)
		}
		return
	}

	if !output.IsJSON() {
		fmt.Println(tui.SuccessStyle.Render("OK"))
	}

	// Success
	if output.IsJSON() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"status":           "updated",
			"previous_version": currentVersion,
			"new_version":      release.TagName,
		}, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println()
		fmt.Println(tui.SuccessStyle.Render("✓ Updated to " + release.TagName))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Run 'clood --version' to verify"))
	}
}

func getRelease(version string) (*GitHubRelease, error) {
	var url string
	if version == "" || version == "latest" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	} else {
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", repoOwner, repoName, version)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		if version != "" {
			return nil, fmt.Errorf("release %s not found", version)
		}
		return nil, fmt.Errorf("no releases found")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

func getBinaryName() string {
	name := fmt.Sprintf("clood-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "clood-update-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Copy to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func replaceBinary(newPath, oldPath string) error {
	// On Unix, we can rename over the running binary
	// On Windows, we need a different approach

	// First, try to remove old binary (may fail on Windows)
	backupPath := oldPath + ".old"
	os.Remove(backupPath) // Ignore error

	// Try to rename current to backup
	if err := os.Rename(oldPath, backupPath); err != nil {
		// If we can't rename, try to copy over
		return copyFile(newPath, oldPath)
	}

	// Rename new to current
	if err := os.Rename(newPath, oldPath); err != nil {
		// Restore backup
		os.Rename(backupPath, oldPath)
		return err
	}

	// Clean up backup
	os.Remove(backupPath)
	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

func formatPublishedDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Jan 02, 2006")
}
