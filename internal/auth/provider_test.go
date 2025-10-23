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

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

func TestNoAuthProvider(t *testing.T) {
	provider := NewNoAuthProvider("test-tool", "local")

	// Test basic properties
	if provider.Name() != "test-tool" {
		t.Errorf("Expected name 'test-tool', got '%s'", provider.Name())
	}

	if provider.IsAuthRequired() {
		t.Error("Expected IsAuthRequired to be false for NoAuthProvider")
	}

	// Test status
	ctx := context.Background()
	status := provider.GetStatus(ctx)

	if !status.Authenticated {
		t.Error("Expected NoAuthProvider to always be authenticated")
	}

	if status.Provider != "test-tool" {
		t.Errorf("Expected provider name 'test-tool', got '%s'", status.Provider)
	}

	if status.Source != "local" {
		t.Errorf("Expected source 'local', got '%s'", status.Source)
	}

	if status.TokenPresent {
		t.Error("Expected TokenPresent to be false for NoAuthProvider")
	}

	// Test operations (should all be no-ops)
	if err := provider.Authenticate(ctx); err != nil {
		t.Errorf("Expected Authenticate to be no-op, got error: %v", err)
	}

	if err := provider.ValidateAuth(ctx); err != nil {
		t.Errorf("Expected ValidateAuth to be no-op, got error: %v", err)
	}

	if err := provider.ClearAuth(); err != nil {
		t.Errorf("Expected ClearAuth to be no-op, got error: %v", err)
	}
}

func TestGitHubAuthProviderWithoutToken(t *testing.T) {
	// Create a temporary directory for cache
	cacheDir := t.TempDir()

	// Mock logger
	logger := &mockLogger{}

	provider := NewGitHubAuthProvider("", cacheDir, logger)

	// Test basic properties
	if provider.Name() != "github" {
		t.Errorf("Expected name 'github', got '%s'", provider.Name())
	}

	if !provider.IsAuthRequired() {
		t.Error("Expected IsAuthRequired to be true for GitHubAuthProvider")
	}

	// Test status without token
	ctx := context.Background()
	status := provider.GetStatus(ctx)

	if status.Authenticated {
		t.Error("Expected not authenticated without token")
	}

	if status.TokenPresent {
		t.Error("Expected TokenPresent to be false without token")
	}

	if status.Provider != "github" {
		t.Errorf("Expected provider name 'github', got '%s'", status.Provider)
	}

	// Test authentication without token should fail
	if err := provider.Authenticate(ctx); err == nil {
		t.Error("Expected Authenticate to fail without token")
	}
}

func TestTugboatAuthProviderWithoutToken(t *testing.T) {
	// Create a temporary directory for cache
	cacheDir := t.TempDir()

	// Mock logger
	logger := &mockLogger{}

	provider := NewTugboatAuthProvider("", "https://api.tugboat.com", cacheDir, logger)

	// Test basic properties
	if provider.Name() != "tugboat" {
		t.Errorf("Expected name 'tugboat', got '%s'", provider.Name())
	}

	if !provider.IsAuthRequired() {
		t.Error("Expected IsAuthRequired to be true for TugboatAuthProvider")
	}

	// Test status without token
	ctx := context.Background()
	status := provider.GetStatus(ctx)

	if status.Authenticated {
		t.Error("Expected not authenticated without token")
	}

	if status.TokenPresent {
		t.Error("Expected TokenPresent to be false without token")
	}

	if status.Provider != "tugboat" {
		t.Errorf("Expected provider name 'tugboat', got '%s'", status.Provider)
	}

	// Test authentication without token should fail
	if err := provider.Authenticate(ctx); err == nil {
		t.Error("Expected Authenticate to fail without token")
	}
}

// mockLogger is a simple logger implementation for testing
type mockLogger struct{}

func (l *mockLogger) Trace(msg string, fields ...logger.Field)        {}
func (l *mockLogger) Debug(msg string, fields ...logger.Field)        {}
func (l *mockLogger) Info(msg string, fields ...logger.Field)         {}
func (l *mockLogger) Warn(msg string, fields ...logger.Field)         {}
func (l *mockLogger) Error(msg string, fields ...logger.Field)        {}
func (l *mockLogger) WithFields(fields ...logger.Field) logger.Logger { return l }
func (l *mockLogger) WithContext(ctx context.Context) logger.Logger   { return l }
func (l *mockLogger) WithComponent(component string) logger.Logger    { return l }
func (l *mockLogger) TraceOperation(operation string) logger.Tracer   { return &mockTracer{} }
func (l *mockLogger) RequestLogger(requestID string) logger.Logger    { return l }
func (l *mockLogger) DumpJSON(obj interface{}, msg string)            {}
func (l *mockLogger) Timing(operation string) logger.TimingLogger     { return &mockTimingLogger{} }

// mockTracer implements logger.Tracer for testing
type mockTracer struct{}

func (t *mockTracer) Step(step string, fields ...logger.Field) {}
func (t *mockTracer) Success(fields ...logger.Field)           {}
func (t *mockTracer) Error(err error, fields ...logger.Field)  {}
func (t *mockTracer) Duration() time.Duration                  { return 0 }

// mockTimingLogger implements logger.TimingLogger for testing
type mockTimingLogger struct{}

func (t *mockTimingLogger) Mark(checkpoint string, fields ...logger.Field) {}
func (t *mockTimingLogger) Complete(fields ...logger.Field)                {}
func (t *mockTimingLogger) Abandon(reason string)                          {}
