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
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	TraceLevel LogLevel = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l LogLevel) String() string {
	switch l {
	case TraceLevel:
		return "TRACE"
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Field represents a structured log field
type Field struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// Logger interface for structured logging
type Logger interface {
	Trace(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
	WithComponent(component string) Logger

	// Development/Debug utilities
	TraceOperation(operation string) Tracer
	RequestLogger(requestID string) Logger
	DumpJSON(obj interface{}, msg string)
	Timing(operation string) TimingLogger
}

// Tracer for operation lifecycle logging
type Tracer interface {
	Step(step string, fields ...Field)
	Success(fields ...Field)
	Error(err error, fields ...Field)
	Duration() time.Duration
}

// TimingLogger for performance monitoring
type TimingLogger interface {
	Mark(checkpoint string, fields ...Field)
	Complete(fields ...Field)
	Abandon(reason string)
}

// Config holds logger configuration
type Config struct {
	Level         LogLevel `yaml:"level"`
	Format        string   `yaml:"format"` // "text" or "json"
	Output        string   `yaml:"output"` // "stdout", "stderr", "file"
	FilePath      string   `yaml:"file_path"`
	SanitizeURLs  bool     `yaml:"sanitize_urls"`
	RedactFields  []string `yaml:"redact_fields"`
	ShowCaller    bool     `yaml:"show_caller"` // Show file:line for debug
	BufferSize    int      `yaml:"buffer_size"`
	FlushInterval string   `yaml:"flush_interval"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Level:         InfoLevel,
		Format:        "text",
		Output:        "stderr",
		SanitizeURLs:  true,
		RedactFields:  []string{"password", "token", "key", "secret", "api_key", "cookie"},
		ShowCaller:    false,
		BufferSize:    100,
		FlushInterval: "5s",
	}
}

// DefaultLogFilePath returns OS-appropriate log file location following platform conventions
func DefaultLogFilePath() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Logs/grctool/grctool.log
		home, err := os.UserHomeDir()
		if err != nil {
			return "./grctool.log"
		}
		return filepath.Join(home, "Library", "Logs", "grctool", "grctool.log")
	case "linux":
		// Linux: ~/.local/state/grctool/grctool.log (XDG Base Directory spec)
		home, err := os.UserHomeDir()
		if err != nil {
			return "./grctool.log"
		}
		return filepath.Join(home, ".local", "state", "grctool", "grctool.log")
	case "windows":
		// Windows: %LOCALAPPDATA%\grctool\grctool.log
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "./grctool.log"
		}
		return filepath.Join(localAppData, "grctool", "grctool.log")
	default:
		// Fallback to current directory
		return "./grctool.log"
	}
}

// New creates a new logger with the given configuration
func New(config *Config) (Logger, error) {
	return NewZerologLogger(config)
}

// NewTestLogger creates a minimal logger for testing
func NewTestLogger() (Logger, error) {
	config := &Config{
		Level:  ErrorLevel, // Only show errors during tests
		Format: "text",
		Output: "stderr",
	}
	return New(config)
}

// Helper functions for creating fields

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.Milliseconds()}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

func Operation(name string) Field {
	return Field{Key: "operation", Value: name}
}

func RequestID(id string) Field {
	return Field{Key: "request_id", Value: id}
}

// Global logger instance for convenience
var defaultLogger Logger

// InitGlobal initializes the global logger
func InitGlobal(config *Config) error {
	logger, err := New(config)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// InitGlobalWithLogger sets the global logger to a specific logger instance
func InitGlobalWithLogger(logger Logger) {
	defaultLogger = logger
}

// Global convenience functions
func Trace(msg string, fields ...Field) {
	if defaultLogger != nil {
		defaultLogger.Trace(msg, fields...)
	}
}

func Debug(msg string, fields ...Field) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...Field) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...Field) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, fields...)
	}
}

// Note: Error() is already used for field creation, so we don't have a global Error function
// Use logger.WithComponent("component").Error() instead

func WithComponent(component string) Logger {
	if defaultLogger != nil {
		return defaultLogger.WithComponent(component)
	}
	return nil
}

// operationTracer implements the Tracer interface
type operationTracer struct {
	logger    Logger
	operation string
	startTime time.Time
	steps     []string
}

func (t *operationTracer) Step(step string, fields ...Field) {
	t.steps = append(t.steps, step)
	allFields := append([]Field{
		Operation(t.operation),
		String("step", step),
		Duration("elapsed", time.Since(t.startTime)),
		Int("step_count", len(t.steps)),
	}, fields...)

	t.logger.Debug("operation_step", allFields...)
}

func (t *operationTracer) Success(fields ...Field) {
	duration := time.Since(t.startTime)
	allFields := append([]Field{
		Operation(t.operation),
		Duration("duration", duration),
		Int("total_steps", len(t.steps)),
		String("outcome", "success"),
	}, fields...)

	t.logger.Info("operation_completed", allFields...)
}

func (t *operationTracer) Error(err error, fields ...Field) {
	duration := time.Since(t.startTime)
	allFields := append([]Field{
		Operation(t.operation),
		Duration("duration", duration),
		Int("total_steps", len(t.steps)),
		String("outcome", "error"),
		Error(err),
	}, fields...)

	t.logger.Error("operation_failed", allFields...)
}

func (t *operationTracer) Duration() time.Duration {
	return time.Since(t.startTime)
}

// checkpoint represents a timing checkpoint
type checkpoint struct {
	name      string
	timestamp time.Time
	fields    []Field
}

// timingLogger implements the TimingLogger interface
type timingLogger struct {
	logger      Logger
	operation   string
	startTime   time.Time
	checkpoints []checkpoint
}

func (t *timingLogger) Mark(name string, fields ...Field) {
	t.checkpoints = append(t.checkpoints, checkpoint{
		name:      name,
		timestamp: time.Now(),
		fields:    fields,
	})

	elapsed := time.Since(t.startTime)
	allFields := append([]Field{
		Operation(t.operation),
		String("checkpoint", name),
		Duration("elapsed", elapsed),
		Int("checkpoint_count", len(t.checkpoints)),
	}, fields...)

	t.logger.Debug("timing_checkpoint", allFields...)
}

func (t *timingLogger) Complete(fields ...Field) {
	totalDuration := time.Since(t.startTime)

	// Log completion with timing summary
	allFields := append([]Field{
		Operation(t.operation),
		Duration("total_duration", totalDuration),
		Int("total_checkpoints", len(t.checkpoints)),
	}, fields...)

	if len(t.checkpoints) > 0 {
		// Add timing breakdown
		var breakdown []map[string]interface{}
		var lastTime = t.startTime

		for _, cp := range t.checkpoints {
			stepDuration := cp.timestamp.Sub(lastTime)
			breakdown = append(breakdown, map[string]interface{}{
				"checkpoint":  cp.name,
				"duration_ms": stepDuration.Milliseconds(),
				"elapsed_ms":  cp.timestamp.Sub(t.startTime).Milliseconds(),
			})
			lastTime = cp.timestamp
		}

		allFields = append(allFields, Field{Key: "timing_breakdown", Value: breakdown})
	}

	t.logger.Info("timing_completed", allFields...)
}

func (t *timingLogger) Abandon(reason string) {
	duration := time.Since(t.startTime)
	t.logger.Warn("timing_abandoned",
		Operation(t.operation),
		Duration("duration", duration),
		String("reason", reason),
		Int("checkpoints_reached", len(t.checkpoints)),
	)
}
