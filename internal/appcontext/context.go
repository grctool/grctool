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
	"io"

	"github.com/spf13/cobra"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

// Context key constants
const (
	loggerKey    contextKey = "grctool.logger"
	commandKey   contextKey = "grctool.cobra.command"
	configKey    contextKey = "grctool.config"
	vcrConfigKey contextKey = "grctool.vcr.config"
	outputKey    contextKey = "grctool.output.writer"
	requestIDKey contextKey = "grctool.request.id"
	userIDKey    contextKey = "grctool.user.id"
	operationKey contextKey = "grctool.operation.name"
	errorKey     contextKey = "grctool.error.writer"
)

// WithLogger stores a logger in the context
// The logger parameter is interface{} to avoid circular imports
func WithLogger(ctx context.Context, log interface{}) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

// GetLogger retrieves the logger from context, returns nil if not found
// Returns interface{} to avoid circular imports - caller must type assert
func GetLogger(ctx context.Context) interface{} {
	return ctx.Value(loggerKey)
}

// WithCommand stores a cobra command in the context
func WithCommand(ctx context.Context, cmd *cobra.Command) context.Context {
	return context.WithValue(ctx, commandKey, cmd)
}

// GetCommand retrieves the cobra command from context
func GetCommand(ctx context.Context) *cobra.Command {
	if cmd, ok := ctx.Value(commandKey).(*cobra.Command); ok {
		return cmd
	}
	return nil
}

// WithConfig stores the application configuration in the context
// The config parameter is interface{} to avoid circular imports
func WithConfig(ctx context.Context, cfg interface{}) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

// GetConfig retrieves the configuration from context
// Returns interface{} to avoid circular imports - caller must type assert
func GetConfig(ctx context.Context) interface{} {
	return ctx.Value(configKey)
}

// WithVCRConfig stores VCR configuration in the context
// The vcrCfg parameter is interface{} to avoid circular imports
func WithVCRConfig(ctx context.Context, vcrCfg interface{}) context.Context {
	return context.WithValue(ctx, vcrConfigKey, vcrCfg)
}

// GetVCRConfig retrieves VCR configuration from context
// Returns interface{} to avoid circular imports - caller must type assert
func GetVCRConfig(ctx context.Context) interface{} {
	return ctx.Value(vcrConfigKey)
}

// WithOutput stores an output writer in the context
func WithOutput(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, outputKey, w)
}

// GetOutput retrieves the output writer from context
func GetOutput(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(outputKey).(io.Writer); ok {
		return w
	}
	return nil
}

// WithError stores an error writer in the context
func WithError(ctx context.Context, w io.Writer) context.Context {
	return context.WithValue(ctx, errorKey, w)
}

// GetError retrieves the error writer from context
func GetError(ctx context.Context) io.Writer {
	if w, ok := ctx.Value(errorKey).(io.Writer); ok {
		return w
	}
	return nil
}

// WithRequestID stores a request ID in the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID stores a user ID in the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// WithOperation stores an operation name in the context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationKey, operation)
}

// GetOperation retrieves the operation name from context
func GetOperation(ctx context.Context) string {
	if op, ok := ctx.Value(operationKey).(string); ok {
		return op
	}
	return ""
}

// EnrichContext creates a new context with common values from a cobra command
func EnrichContext(ctx context.Context, cmd *cobra.Command) context.Context {
	// Add the command itself
	ctx = WithCommand(ctx, cmd)

	// Add output and error writers from the command
	if cmd != nil {
		ctx = WithOutput(ctx, cmd.OutOrStdout())
		ctx = WithError(ctx, cmd.ErrOrStderr())

		// Set operation name from command
		ctx = WithOperation(ctx, cmd.Name())
	}

	return ctx
}
