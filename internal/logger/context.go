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

package logger

import (
	"context"

	"github.com/grctool/grctool/internal/appcontext"
)

// FromContext retrieves a logger from the context with automatic enrichment
// If no logger is found, returns the default global logger
func FromContext(ctx context.Context) Logger {
	// Try to get logger from context first
	var log Logger
	if logInterface := appcontext.GetLogger(ctx); logInterface != nil {
		// Type assert to Logger
		if typedLog, ok := logInterface.(Logger); ok {
			log = typedLog
		}
	}

	if log == nil {
		// Fallback to default logger
		log = defaultLogger
		if log == nil {
			// If even default logger is nil, create a basic one
			cfg := DefaultConfig()
			log, _ = New(cfg)
		}
	}

	// Auto-enrich with context values

	// Add request ID if present
	if reqID := appcontext.GetRequestID(ctx); reqID != "" {
		log = log.WithFields(RequestID(reqID))
	}

	// Add operation name if present
	if operation := appcontext.GetOperation(ctx); operation != "" {
		log = log.WithFields(Operation(operation))
	}

	// Add user ID if present
	if userID := appcontext.GetUserID(ctx); userID != "" {
		log = log.WithFields(String("user_id", userID))
	}

	return log
}

// FromContextOrNew retrieves a logger from context or creates a new one with the given component
func FromContextOrNew(ctx context.Context, component string) Logger {
	log := FromContext(ctx)
	if component != "" {
		log = log.WithComponent(component)
	}
	return log
}

// WithContext stores a logger in the context (convenience wrapper)
func WithContext(ctx context.Context, log Logger) context.Context {
	return appcontext.WithLogger(ctx, log)
}

// EnrichLogger takes a logger and enriches it with context values
func EnrichLogger(ctx context.Context, log Logger) Logger {
	// Add request ID if present
	if reqID := appcontext.GetRequestID(ctx); reqID != "" {
		log = log.WithFields(RequestID(reqID))
	}

	// Add operation name if present
	if operation := appcontext.GetOperation(ctx); operation != "" {
		log = log.WithFields(Operation(operation))
	}

	// Add user ID if present
	if userID := appcontext.GetUserID(ctx); userID != "" {
		log = log.WithFields(String("user_id", userID))
	}

	return log
}
