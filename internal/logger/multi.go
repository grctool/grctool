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
	"time"
)

// multiLogger implements Logger interface and forwards all calls to multiple loggers
type multiLogger struct {
	loggers []Logger
}

// NewMultiLogger creates a new logger that writes to multiple destinations
func NewMultiLogger(loggers ...Logger) Logger {
	// Filter out nil loggers
	var validLoggers []Logger
	for _, l := range loggers {
		if l != nil {
			validLoggers = append(validLoggers, l)
		}
	}

	if len(validLoggers) == 0 {
		// Return a no-op logger if no valid loggers provided
		return &multiLogger{loggers: []Logger{}}
	}

	return &multiLogger{
		loggers: validLoggers,
	}
}

func (m *multiLogger) Trace(msg string, fields ...Field) {
	for _, logger := range m.loggers {
		logger.Trace(msg, fields...)
	}
}

func (m *multiLogger) Debug(msg string, fields ...Field) {
	for _, logger := range m.loggers {
		logger.Debug(msg, fields...)
	}
}

func (m *multiLogger) Info(msg string, fields ...Field) {
	for _, logger := range m.loggers {
		logger.Info(msg, fields...)
	}
}

func (m *multiLogger) Warn(msg string, fields ...Field) {
	for _, logger := range m.loggers {
		logger.Warn(msg, fields...)
	}
}

func (m *multiLogger) Error(msg string, fields ...Field) {
	for _, logger := range m.loggers {
		logger.Error(msg, fields...)
	}
}

func (m *multiLogger) WithFields(fields ...Field) Logger {
	var newLoggers []Logger
	for _, logger := range m.loggers {
		newLoggers = append(newLoggers, logger.WithFields(fields...))
	}
	return &multiLogger{loggers: newLoggers}
}

func (m *multiLogger) WithContext(ctx context.Context) Logger {
	var newLoggers []Logger
	for _, logger := range m.loggers {
		newLoggers = append(newLoggers, logger.WithContext(ctx))
	}
	return &multiLogger{loggers: newLoggers}
}

func (m *multiLogger) WithComponent(component string) Logger {
	var newLoggers []Logger
	for _, logger := range m.loggers {
		newLoggers = append(newLoggers, logger.WithComponent(component))
	}
	return &multiLogger{loggers: newLoggers}
}

func (m *multiLogger) TraceOperation(operation string) Tracer {
	// For tracers, we'll return a multi-tracer that forwards to all
	var tracers []Tracer
	for _, logger := range m.loggers {
		tracers = append(tracers, logger.TraceOperation(operation))
	}
	return &multiTracer{tracers: tracers}
}

func (m *multiLogger) RequestLogger(requestID string) Logger {
	var newLoggers []Logger
	for _, logger := range m.loggers {
		newLoggers = append(newLoggers, logger.RequestLogger(requestID))
	}
	return &multiLogger{loggers: newLoggers}
}

func (m *multiLogger) DumpJSON(obj interface{}, msg string) {
	for _, logger := range m.loggers {
		logger.DumpJSON(obj, msg)
	}
}

func (m *multiLogger) Timing(operation string) TimingLogger {
	// For timing loggers, we'll return a multi-timing logger
	var timingLoggers []TimingLogger
	for _, logger := range m.loggers {
		timingLoggers = append(timingLoggers, logger.Timing(operation))
	}
	return &multiTimingLogger{timingLoggers: timingLoggers}
}

// multiTracer implements Tracer interface for multiple tracers
type multiTracer struct {
	tracers []Tracer
}

func (m *multiTracer) Step(step string, fields ...Field) {
	for _, tracer := range m.tracers {
		tracer.Step(step, fields...)
	}
}

func (m *multiTracer) Success(fields ...Field) {
	for _, tracer := range m.tracers {
		tracer.Success(fields...)
	}
}

func (m *multiTracer) Error(err error, fields ...Field) {
	for _, tracer := range m.tracers {
		tracer.Error(err, fields...)
	}
}

func (m *multiTracer) Duration() time.Duration {
	// Return the duration from the first tracer (they should all be the same)
	if len(m.tracers) > 0 {
		return m.tracers[0].Duration()
	}
	return 0
}

// multiTimingLogger implements TimingLogger interface for multiple timing loggers
type multiTimingLogger struct {
	timingLoggers []TimingLogger
}

func (m *multiTimingLogger) Mark(checkpoint string, fields ...Field) {
	for _, tl := range m.timingLoggers {
		tl.Mark(checkpoint, fields...)
	}
}

func (m *multiTimingLogger) Complete(fields ...Field) {
	for _, tl := range m.timingLoggers {
		tl.Complete(fields...)
	}
}

func (m *multiTimingLogger) Abandon(reason string) {
	for _, tl := range m.timingLoggers {
		tl.Abandon(reason)
	}
}
