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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

// ZerologLogger wraps zerolog for our logging needs
type ZerologLogger struct {
	logger zerolog.Logger
}

// NewZerologLogger creates a new zerolog-based logger
func NewZerologLogger(config *Config) (*ZerologLogger, error) {
	// Set global time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Configure zerolog level
	level := parseZerologLevel(config.Level)

	// Create the appropriate writer based on config
	var writer io.Writer
	switch config.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		if config.FilePath == "" {
			return nil, fmt.Errorf("file_path required when output is 'file'")
		}
		// Ensure directory exists
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		// Use diode writer for non-blocking writes
		writer = diode.NewWriter(file, 1000, 10*time.Millisecond, func(missed int) {
			fmt.Fprintf(os.Stderr, "zerolog: dropped %d log messages\n", missed)
		})
	default:
		writer = os.Stderr
	}

	// Configure output format
	if config.Format == "text" {
		writer = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
			NoColor:    config.Output == "file", // No colors in file output
		}
	}

	// Create logger with context
	logger := zerolog.New(writer).With().Timestamp().Logger().Level(level)

	// Add caller info if requested
	if config.ShowCaller {
		logger = logger.With().Caller().Logger()
	}

	return &ZerologLogger{logger: logger}, nil
}

// parseZerologLevel converts our LogLevel to zerolog.Level
func parseZerologLevel(level LogLevel) zerolog.Level {
	switch level {
	case TraceLevel:
		return zerolog.TraceLevel
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Implementation of our Logger interface

func (l *ZerologLogger) Trace(msg string, fields ...Field) {
	event := l.logger.Trace()
	l.addFields(event, fields...).Msg(msg)
}

func (l *ZerologLogger) Debug(msg string, fields ...Field) {
	event := l.logger.Debug()
	l.addFields(event, fields...).Msg(msg)
}

func (l *ZerologLogger) Info(msg string, fields ...Field) {
	event := l.logger.Info()
	l.addFields(event, fields...).Msg(msg)
}

func (l *ZerologLogger) Warn(msg string, fields ...Field) {
	event := l.logger.Warn()
	l.addFields(event, fields...).Msg(msg)
}

func (l *ZerologLogger) Error(msg string, fields ...Field) {
	event := l.logger.Error()
	l.addFields(event, fields...).Msg(msg)
}

func (l *ZerologLogger) WithFields(fields ...Field) Logger {
	newLogger := l.logger.With()
	for _, field := range fields {
		newLogger = l.addFieldToContext(newLogger, field)
	}
	return &ZerologLogger{logger: newLogger.Logger()}
}

func (l *ZerologLogger) WithContext(ctx context.Context) Logger {
	return &ZerologLogger{logger: l.logger.With().Ctx(ctx).Logger()}
}

func (l *ZerologLogger) WithComponent(component string) Logger {
	return &ZerologLogger{logger: l.logger.With().Str("component", component).Logger()}
}

// TraceOperation implementation
func (l *ZerologLogger) TraceOperation(operation string) Tracer {
	return &zerologTracer{
		logger:    l,
		operation: operation,
		startTime: time.Now(),
		steps:     make([]string, 0),
	}
}

func (l *ZerologLogger) RequestLogger(requestID string) Logger {
	return &ZerologLogger{logger: l.logger.With().Str("request_id", requestID).Logger()}
}

func (l *ZerologLogger) DumpJSON(obj interface{}, msg string) {
	l.logger.Debug().Interface("data", obj).Msg(msg)
}

func (l *ZerologLogger) Timing(operation string) TimingLogger {
	return &zerologTimingLogger{
		logger:      l,
		operation:   operation,
		startTime:   time.Now(),
		checkpoints: make([]checkpoint, 0),
	}
}

// Helper method to add fields to a zerolog event
func (l *ZerologLogger) addFields(event *zerolog.Event, fields ...Field) *zerolog.Event {
	for _, field := range fields {
		event = l.addFieldToEvent(event, field)
	}
	return event
}

// Helper method to add a field to a zerolog event
func (l *ZerologLogger) addFieldToEvent(event *zerolog.Event, field Field) *zerolog.Event {
	switch v := field.Value.(type) {
	case string:
		return event.Str(field.Key, v)
	case int:
		return event.Int(field.Key, v)
	case int64:
		return event.Int64(field.Key, v)
	case float64:
		return event.Float64(field.Key, v)
	case bool:
		return event.Bool(field.Key, v)
	case error:
		return event.Err(v)
	case time.Duration:
		return event.Dur(field.Key, v)
	case time.Time:
		return event.Time(field.Key, v)
	default:
		return event.Interface(field.Key, v)
	}
}

// Helper method to add a field to a zerolog context
func (l *ZerologLogger) addFieldToContext(ctx zerolog.Context, field Field) zerolog.Context {
	switch v := field.Value.(type) {
	case string:
		return ctx.Str(field.Key, v)
	case int:
		return ctx.Int(field.Key, v)
	case int64:
		return ctx.Int64(field.Key, v)
	case float64:
		return ctx.Float64(field.Key, v)
	case bool:
		return ctx.Bool(field.Key, v)
	case error:
		return ctx.Err(v)
	case time.Duration:
		return ctx.Dur(field.Key, v)
	case time.Time:
		return ctx.Time(field.Key, v)
	default:
		return ctx.Interface(field.Key, v)
	}
}

// zerologTracer implements the Tracer interface
type zerologTracer struct {
	logger    *ZerologLogger
	operation string
	startTime time.Time
	steps     []string
}

func (t *zerologTracer) Step(step string, fields ...Field) {
	t.steps = append(t.steps, step)
	event := t.logger.logger.Debug().
		Str("operation", t.operation).
		Str("step", step).
		Dur("elapsed", time.Since(t.startTime)).
		Int("step_count", len(t.steps))

	t.logger.addFields(event, fields...).Msg("operation_step")
}

func (t *zerologTracer) Success(fields ...Field) {
	event := t.logger.logger.Info().
		Str("operation", t.operation).
		Dur("duration", time.Since(t.startTime)).
		Int("total_steps", len(t.steps)).
		Str("outcome", "success")

	t.logger.addFields(event, fields...).Msg("operation_completed")
}

func (t *zerologTracer) Error(err error, fields ...Field) {
	event := t.logger.logger.Error().
		Str("operation", t.operation).
		Dur("duration", time.Since(t.startTime)).
		Int("total_steps", len(t.steps)).
		Str("outcome", "error").
		Err(err)

	t.logger.addFields(event, fields...).Msg("operation_failed")
}

func (t *zerologTracer) Duration() time.Duration {
	return time.Since(t.startTime)
}

// zerologTimingLogger implements the TimingLogger interface
type zerologTimingLogger struct {
	logger      *ZerologLogger
	operation   string
	startTime   time.Time
	checkpoints []checkpoint
}

func (t *zerologTimingLogger) Mark(name string, fields ...Field) {
	t.checkpoints = append(t.checkpoints, checkpoint{
		name:      name,
		timestamp: time.Now(),
		fields:    fields,
	})

	event := t.logger.logger.Debug().
		Str("operation", t.operation).
		Str("checkpoint", name).
		Dur("elapsed", time.Since(t.startTime)).
		Int("checkpoint_count", len(t.checkpoints))

	t.logger.addFields(event, fields...).Msg("timing_checkpoint")
}

func (t *zerologTimingLogger) Complete(fields ...Field) {
	event := t.logger.logger.Info().
		Str("operation", t.operation).
		Dur("total_duration", time.Since(t.startTime)).
		Int("total_checkpoints", len(t.checkpoints))

	if len(t.checkpoints) > 0 {
		// Add timing breakdown
		breakdown := make([]map[string]interface{}, 0, len(t.checkpoints))
		lastTime := t.startTime

		for _, cp := range t.checkpoints {
			stepDuration := cp.timestamp.Sub(lastTime)
			breakdown = append(breakdown, map[string]interface{}{
				"checkpoint":  cp.name,
				"duration_ms": stepDuration.Milliseconds(),
				"elapsed_ms":  cp.timestamp.Sub(t.startTime).Milliseconds(),
			})
			lastTime = cp.timestamp
		}

		event = event.Interface("timing_breakdown", breakdown)
	}

	t.logger.addFields(event, fields...).Msg("timing_completed")
}

func (t *zerologTimingLogger) Abandon(reason string) {
	t.logger.logger.Warn().
		Str("operation", t.operation).
		Dur("duration", time.Since(t.startTime)).
		Str("reason", reason).
		Int("checkpoints_reached", len(t.checkpoints)).
		Msg("timing_abandoned")
}

// CreateMultiLogger creates a multi-output logger with different levels
func CreateMultiLogger(consoleConfig, fileConfig *Config) (Logger, error) {
	var loggers []Logger

	// Create console logger
	if consoleConfig != nil {
		consoleLogger, err := NewZerologLogger(consoleConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create console logger: %w", err)
		}
		loggers = append(loggers, consoleLogger)
	}

	// Create file logger
	if fileConfig != nil {
		fileLogger, err := NewZerologLogger(fileConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create file logger: %w", err)
		}
		loggers = append(loggers, fileLogger)
	}

	if len(loggers) == 0 {
		return nil, fmt.Errorf("no loggers configured")
	}

	if len(loggers) == 1 {
		return loggers[0], nil
	}

	return NewMultiLogger(loggers...), nil
}

// Hook for sensitive field redaction
type RedactionHook struct {
}

func (h RedactionHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// This would require more complex implementation to intercept and modify fields
	// For now, we'll handle redaction at the field addition level
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "trace":
		return TraceLevel
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}
