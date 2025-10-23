// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	githubAPIURL    = "https://api.github.com/repos/grctool/grctool/releases/latest"
	installerURL    = "https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh"
	updateCheckFile = ".grctool_update_check"
)

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install grctool updates",
	Long: `Check for the latest version of grctool and optionally install it.

The update command can check for new versions and install them using the
official installer script.`,
}

// checkCmd represents the update check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for available updates",
	Long: `Check if a new version of grctool is available.

This command queries the GitHub API to determine if a newer version
is available compared to the currently installed version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdateCheck(false)
	},
}

// installCmd represents the update install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the latest version",
	Long: `Download and install the latest version of grctool.

This command downloads the official installer script and executes it
to install the latest version. By default, it will prompt for confirmation
before installing. Use --yes to skip the confirmation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		systemWide, _ := cmd.Flags().GetBool("system")
		return runUpdateInstall(yes, systemWide)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.AddCommand(checkCmd)
	updateCmd.AddCommand(installCmd)

	// Flags for install command
	installCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	installCmd.Flags().Bool("system", false, "Install system-wide to /usr/local/bin (requires sudo)")
}

// runUpdateCheck checks for available updates
func runUpdateCheck(silent bool) error {
	if !silent {
		fmt.Println("Checking for updates...")
	}

	// Get latest release info
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Get current version
	currentVersion := version
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersionClean := strings.TrimPrefix(currentVersion, "v")

	if !silent {
		fmt.Printf("\nCurrent version: %s\n", currentVersion)
		fmt.Printf("Latest version:  %s\n", latestVersion)
		fmt.Printf("Published:       %s\n", release.PublishedAt.Format("2006-01-02"))
		fmt.Println()
	}

	// Compare versions
	if currentVersionClean == "dev" || currentVersionClean == "unknown" {
		if !silent {
			fmt.Println("⚠️  Running development version - cannot determine if update is available")
			fmt.Printf("Latest stable version is %s\n", latestVersion)
			fmt.Printf("Visit: %s\n", release.HTMLURL)
		}
		return nil
	}

	if compareVersions(currentVersionClean, latestVersion) < 0 {
		if !silent {
			fmt.Printf("🆕 Update available: %s → %s\n", currentVersion, latestVersion)
			fmt.Println()
			fmt.Println("To update, run:")
			fmt.Println("  grctool update install")
			fmt.Println()
			fmt.Printf("Release notes: %s\n", release.HTMLURL)
		}
		return nil
	}

	if !silent {
		fmt.Println("✅ You are running the latest version")
	}

	return nil
}

// runUpdateInstall installs the latest version
func runUpdateInstall(skipConfirmation bool, systemWide bool) error {
	// First check what updates are available
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch release information: %w", err)
	}

	currentVersion := version
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest version:  %s\n", latestVersion)
	fmt.Println()

	// Check if already on latest
	currentVersionClean := strings.TrimPrefix(currentVersion, "v")
	if currentVersionClean != "dev" && currentVersionClean != "unknown" {
		if compareVersions(currentVersionClean, latestVersion) >= 0 {
			fmt.Println("✅ Already running the latest version")
			return nil
		}
	}

	// Confirm installation
	if !skipConfirmation {
		fmt.Printf("Install version %s? [y/N]: ", latestVersion)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))

		if response != "y" && response != "yes" {
			fmt.Println("Installation cancelled")
			return nil
		}
	}

	fmt.Println()
	fmt.Println("Downloading installer...")

	// Download installer script
	installerPath, err := downloadInstaller()
	if err != nil {
		return fmt.Errorf("failed to download installer: %w", err)
	}
	defer os.Remove(installerPath)

	// Make installer executable
	if err := os.Chmod(installerPath, 0755); err != nil {
		return fmt.Errorf("failed to make installer executable: %w", err)
	}

	// Run installer
	fmt.Println("Running installer...")
	fmt.Println()

	args := []string{}
	if systemWide {
		args = append(args, "--system")
	}

	cmd := exec.Command(installerPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installer failed: %w", err)
	}

	return nil
}

// fetchLatestRelease fetches the latest release information from GitHub
func fetchLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent to avoid GitHub API rate limiting
	req.Header.Set("User-Agent", fmt.Sprintf("grctool/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// downloadInstaller downloads the installer script to a temporary file
func downloadInstaller() (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(installerURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download installer: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "grctool-install-*.sh")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Copy installer content
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// compareVersions compares two semantic version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Simple lexicographic comparison
	// For more robust comparison, could use github.com/hashicorp/go-version
	// but keeping dependencies minimal

	// Strip 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split on dots
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 string

		if i < len(parts1) {
			p1 = parts1[i]
		} else {
			p1 = "0"
		}

		if i < len(parts2) {
			p2 = parts2[i]
		} else {
			p2 = "0"
		}

		// Remove any pre-release suffixes (e.g., "1-beta")
		p1 = strings.Split(p1, "-")[0]
		p2 = strings.Split(p2, "-")[0]

		if p1 < p2 {
			return -1
		}
		if p1 > p2 {
			return 1
		}
	}

	return 0
}
