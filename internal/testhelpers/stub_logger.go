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

package testhelpers

import (
	"context"
	"sync"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

// Compile-time assertion that StubLogger implements logger.Logger.
var _ logger.Logger = (*StubLogger)(nil)

// LogEntry captures a single log call for later assertion in tests.
type LogEntry struct {
	Level   string
	Message string
	Fields  []logger.Field
}

// StubLogger implements the logger.Logger interface as a silent no-op logger
// suitable for tests. When CaptureEntries is true, log calls are recorded in
// the Entries slice for assertion.
type StubLogger struct {
	mu             sync.Mutex
	CaptureEntries bool
	Entries        []LogEntry
	component      string
}

// NewStubLogger creates a silent logger that discards all output.
func NewStubLogger() *StubLogger {
	return &StubLogger{}
}

// NewCapturingLogger creates a logger that records all entries for assertion.
func NewCapturingLogger() *StubLogger {
	return &StubLogger{CaptureEntries: true}
}

func (l *StubLogger) record(level, msg string, fields []logger.Field) {
	if !l.CaptureEntries {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Entries = append(l.Entries, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  fields,
	})
}

func (l *StubLogger) Trace(msg string, fields ...logger.Field) { l.record("TRACE", msg, fields) }
func (l *StubLogger) Debug(msg string, fields ...logger.Field) { l.record("DEBUG", msg, fields) }
func (l *StubLogger) Info(msg string, fields ...logger.Field)  { l.record("INFO", msg, fields) }
func (l *StubLogger) Warn(msg string, fields ...logger.Field)  { l.record("WARN", msg, fields) }
func (l *StubLogger) Error(msg string, fields ...logger.Field) { l.record("ERROR", msg, fields) }

func (l *StubLogger) WithFields(_ ...logger.Field) logger.Logger  { return l }
func (l *StubLogger) WithContext(_ context.Context) logger.Logger  { return l }
func (l *StubLogger) WithComponent(_ string) logger.Logger         { return l }
func (l *StubLogger) TraceOperation(_ string) logger.Tracer        { return &stubTracer{} }
func (l *StubLogger) RequestLogger(_ string) logger.Logger         { return l }
func (l *StubLogger) DumpJSON(_ interface{}, _ string)             {}
func (l *StubLogger) Timing(_ string) logger.TimingLogger          { return &stubTimingLogger{} }

// HasEntry returns true if an entry with the given level and message was captured.
func (l *StubLogger) HasEntry(level, msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.Entries {
		if e.Level == level && e.Message == msg {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// stubTracer implements logger.Tracer
// ---------------------------------------------------------------------------

type stubTracer struct {
	start time.Time
}

func (t *stubTracer) Step(_ string, _ ...logger.Field)  {}
func (t *stubTracer) Success(_ ...logger.Field)         {}
func (t *stubTracer) Error(_ error, _ ...logger.Field)  {}
func (t *stubTracer) Duration() time.Duration           { return time.Since(t.start) }

// ---------------------------------------------------------------------------
// stubTimingLogger implements logger.TimingLogger
// ---------------------------------------------------------------------------

type stubTimingLogger struct{}

func (t *stubTimingLogger) Mark(_ string, _ ...logger.Field) {}
func (t *stubTimingLogger) Complete(_ ...logger.Field)       {}
func (t *stubTimingLogger) Abandon(_ string)                 {}
