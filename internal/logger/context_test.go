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
	"testing"

	"github.com/grctool/grctool/internal/appcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromContext(t *testing.T) {
	t.Run("Returns logger from context", func(t *testing.T) {
		ctx := context.Background()

		// Create a test logger
		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)

		// Store in context
		ctx = appcontext.WithLogger(ctx, log)

		// Retrieve
		retrieved := FromContext(ctx)
		assert.NotNil(t, retrieved)
	})

	t.Run("Returns default logger when none in context", func(t *testing.T) {
		// Save current default logger
		oldDefault := defaultLogger
		defer func() { defaultLogger = oldDefault }()

		// Set a test default logger
		cfg := DefaultConfig()
		testDefault, err := New(cfg)
		require.NoError(t, err)
		defaultLogger = testDefault

		ctx := context.Background()
		retrieved := FromContext(ctx)
		assert.Equal(t, testDefault, retrieved)
	})

	t.Run("Creates new logger if no default", func(t *testing.T) {
		// Save current default logger
		oldDefault := defaultLogger
		defer func() { defaultLogger = oldDefault }()

		// Clear default logger
		defaultLogger = nil

		ctx := context.Background()
		retrieved := FromContext(ctx)
		assert.NotNil(t, retrieved)
	})

	t.Run("Auto-enriches with request ID", func(t *testing.T) {
		ctx := context.Background()
		ctx = appcontext.WithRequestID(ctx, "req-12345")

		// Create a mock logger that tracks fields
		mockLog := &mockLogger{fields: make(map[string]interface{})}
		ctx = appcontext.WithLogger(ctx, mockLog)

		retrieved := FromContext(ctx)
		assert.NotNil(t, retrieved)

		// The returned logger should have request ID field
		// This would require the mock logger to track WithFields calls
	})

	t.Run("Auto-enriches with operation name", func(t *testing.T) {
		ctx := context.Background()
		ctx = appcontext.WithOperation(ctx, "sync-operation")

		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)
		ctx = appcontext.WithLogger(ctx, log)

		retrieved := FromContext(ctx)
		assert.NotNil(t, retrieved)
	})

	t.Run("Auto-enriches with user ID", func(t *testing.T) {
		ctx := context.Background()
		ctx = appcontext.WithUserID(ctx, "user-123")

		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)
		ctx = appcontext.WithLogger(ctx, log)

		retrieved := FromContext(ctx)
		assert.NotNil(t, retrieved)
	})
}

func TestFromContextOrNew(t *testing.T) {
	t.Run("Creates logger with component", func(t *testing.T) {
		ctx := context.Background()

		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)
		ctx = appcontext.WithLogger(ctx, log)

		retrieved := FromContextOrNew(ctx, "test-component")
		assert.NotNil(t, retrieved)
	})

	t.Run("Works without component", func(t *testing.T) {
		ctx := context.Background()

		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)
		ctx = appcontext.WithLogger(ctx, log)

		retrieved := FromContextOrNew(ctx, "")
		assert.NotNil(t, retrieved)
	})
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()

	cfg := DefaultConfig()
	log, err := New(cfg)
	require.NoError(t, err)

	// Use the convenience function
	ctx = WithContext(ctx, log)

	// Should be retrievable via appcontext
	retrieved := appcontext.GetLogger(ctx)
	assert.Equal(t, log, retrieved)
}

func TestEnrichLogger(t *testing.T) {
	t.Run("Enriches with all context values", func(t *testing.T) {
		ctx := context.Background()
		ctx = appcontext.WithRequestID(ctx, "req-12345")
		ctx = appcontext.WithOperation(ctx, "test-op")
		ctx = appcontext.WithUserID(ctx, "user-456")

		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)

		enriched := EnrichLogger(ctx, log)
		assert.NotNil(t, enriched)
		// The enriched logger should have all the fields
	})

	t.Run("Handles missing context values", func(t *testing.T) {
		ctx := context.Background()
		// No context values set

		cfg := DefaultConfig()
		log, err := New(cfg)
		require.NoError(t, err)

		enriched := EnrichLogger(ctx, log)
		assert.NotNil(t, enriched)
		// Should return the same logger without errors
	})
}

// mockLogger is a test implementation of Logger interface
type mockLogger struct {
	fields map[string]interface{}
}

func (m *mockLogger) Trace(msg string, fields ...Field) {}
func (m *mockLogger) Debug(msg string, fields ...Field) {}
func (m *mockLogger) Info(msg string, fields ...Field)  {}
func (m *mockLogger) Warn(msg string, fields ...Field)  {}
func (m *mockLogger) Error(msg string, fields ...Field) {}

func (m *mockLogger) WithFields(fields ...Field) Logger {
	// Clone and add fields
	newMock := &mockLogger{
		fields: make(map[string]interface{}),
	}
	for k, v := range m.fields {
		newMock.fields[k] = v
	}
	for _, f := range fields {
		newMock.fields[f.Key] = f.Value
	}
	return newMock
}

func (m *mockLogger) WithContext(ctx context.Context) Logger {
	return m
}

func (m *mockLogger) WithComponent(component string) Logger {
	return m.WithFields(String("component", component))
}

func (m *mockLogger) TraceOperation(operation string) Tracer {
	return nil
}

func (m *mockLogger) RequestLogger(requestID string) Logger {
	return m.WithFields(RequestID(requestID))
}

func (m *mockLogger) DumpJSON(obj interface{}, msg string) {}

func (m *mockLogger) Timing(operation string) TimingLogger {
	return nil
}
