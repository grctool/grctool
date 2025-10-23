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
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information set by main package
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// SetVersionInfo sets the version information from the main package
func SetVersionInfo(v, bt, gc string) {
	version = v
	buildTime = bt
	gitCommit = gc
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display detailed version information including the version number,
build time, git commit, and Go runtime version.`,
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")

		if short {
			fmt.Println(version)
		} else {
			fmt.Printf("grctool version %s\n", version)
			fmt.Printf("  Build Time: %s\n", buildTime)
			fmt.Printf("  Git Commit: %s\n", gitCommit)
			fmt.Printf("  Go Version: %s\n", runtime.Version())
			fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Add flag for short version output
	versionCmd.Flags().BoolP("short", "s", false, "Print only the version number")
}
