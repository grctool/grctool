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

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interpolation"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create interpolator with nested variables
	flatVars := cfg.Interpolation.GetFlatVariables()
	fmt.Printf("Flattened variables:\n")
	for k, v := range flatVars {
		fmt.Printf("  %s = %s\n", k, v)
	}

	interpolatorConfig := interpolation.InterpolatorConfig{
		Variables:         flatVars,
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}

	interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)

	// Read test content
	content, err := os.ReadFile("test-policy-variables.md")
	if err != nil {
		log.Fatalf("Failed to read test file: %v", err)
	}

	// Interpolate
	result, err := interpolator.Interpolate(string(content))
	if err != nil {
		log.Fatalf("Failed to interpolate: %v", err)
	}

	fmt.Printf("\n=== INTERPOLATED RESULT ===\n")
	fmt.Println(result)
}
