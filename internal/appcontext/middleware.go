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

package appcontext

import (
	"context"

	"github.com/spf13/cobra"
)

// CommandContext creates an enriched context for command execution
// Note: This does not set up config or logger due to circular dependencies.
// Commands should:
// 1. Call this function to get basic context
// 2. Load config and add to context
// 3. Create logger and add to context
func CommandContext(cmd *cobra.Command) context.Context {
	// Start with background context
	ctx := context.Background()

	// Enrich with common values from command
	ctx = EnrichContext(ctx, cmd)

	// Generate and add request ID
	ctx = WithRequestID(ctx, GenerateRequestID())

	return ctx
}

// PreRunE is a cobra PreRunE function that sets up context
// Usage: cmd.PreRunE = appcontext.PreRunE
func PreRunE(cmd *cobra.Command, args []string) error {
	ctx := CommandContext(cmd)

	// Store context in command for use in RunE
	cmd.SetContext(ctx)

	// Note: Config and logger should be set up by the command handler

	return nil
}

// WrapRunE wraps a RunE function with context setup
func WrapRunE(runE func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Setup context if not already done
		if cmd.Context() == nil {
			ctx := CommandContext(cmd)
			cmd.SetContext(ctx)
		}

		// Execute the wrapped function
		return runE(cmd, args)
	}
}

// GetOrCreateContext gets the context from the command or creates a new enriched one
func GetOrCreateContext(cmd *cobra.Command) context.Context {
	// Check if context already exists
	ctx := cmd.Context()
	if ctx != nil {
		return ctx
	}

	// Create new context
	ctx = CommandContext(cmd)

	// Store it in the command
	cmd.SetContext(ctx)

	return ctx
}
