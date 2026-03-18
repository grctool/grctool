// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LogLevel tests ---

func TestLogLevel_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{TraceLevel, "TRACE"},
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestParseLogLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"trace", TraceLevel},
		{"TRACE", TraceLevel},
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"WARN", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"unknown", InfoLevel},
		{"", InfoLevel},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, ParseLogLevel(tt.input), "for input: %s", tt.input)
	}
}

func TestParseZerologLevel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, zerolog.TraceLevel, parseZerologLevel(TraceLevel))
	assert.Equal(t, zerolog.DebugLevel, parseZerologLevel(DebugLevel))
	assert.Equal(t, zerolog.InfoLevel, parseZerologLevel(InfoLevel))
	assert.Equal(t, zerolog.WarnLevel, parseZerologLevel(WarnLevel))
	assert.Equal(t, zerolog.ErrorLevel, parseZerologLevel(ErrorLevel))
	assert.Equal(t, zerolog.InfoLevel, parseZerologLevel(LogLevel(99)))
}

// --- Field helper tests ---

func TestFieldHelpers(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Field{Key: "name", Value: "val"}, String("name", "val"))
	assert.Equal(t, Field{Key: "count", Value: 42}, Int("count", 42))
	assert.Equal(t, Field{Key: "dur", Value: int64(1500)}, Duration("dur", 1500*time.Millisecond))
	assert.Equal(t, Field{Key: "error", Value: "oops"}, Error(errors.New("oops")))
	assert.Equal(t, Field{Key: "operation", Value: "sync"}, Operation("sync"))
	assert.Equal(t, Field{Key: "request_id", Value: "req-123"}, RequestID("req-123"))
}

// --- DefaultConfig tests ---

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	assert.Equal(t, InfoLevel, cfg.Level)
	assert.Equal(t, "text", cfg.Format)
	assert.Equal(t, "stderr", cfg.Output)
	assert.True(t, cfg.SanitizeURLs)
	assert.Contains(t, cfg.RedactFields, "password")
	assert.Contains(t, cfg.RedactFields, "token")
	assert.False(t, cfg.ShowCaller)
	assert.Equal(t, 100, cfg.BufferSize)
	assert.Equal(t, "5s", cfg.FlushInterval)
}

// --- DefaultLogFilePath tests ---

func TestDefaultLogFilePath(t *testing.T) {
	t.Parallel()
	path := DefaultLogFilePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "grctool.log")
}

// --- New / NewTestLogger tests ---

func TestNew_DefaultConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	log, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestNewTestLogger(t *testing.T) {
	t.Parallel()
	log, err := NewTestLogger()
	require.NoError(t, err)
	assert.NotNil(t, log)
}

// --- ZerologLogger tests ---

func TestZerologLogger_Creation_Stdout(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: DebugLevel, Format: "json", Output: "stdout"}
	log, err := NewZerologLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestZerologLogger_Creation_Stderr(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: InfoLevel, Format: "text", Output: "stderr"}
	log, err := NewZerologLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestZerologLogger_Creation_File(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	cfg := &Config{Level: DebugLevel, Format: "json", Output: "file", FilePath: logFile}
	log, err := NewZerologLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)

	// Verify file was created
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

func TestZerologLogger_Creation_File_NoPath(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: DebugLevel, Format: "json", Output: "file", FilePath: ""}
	_, err := NewZerologLogger(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_path required")
}

func TestZerologLogger_Creation_DefaultOutput(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: InfoLevel, Format: "json", Output: "other"}
	log, err := NewZerologLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestZerologLogger_Creation_WithCaller(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: DebugLevel, Format: "json", Output: "stderr", ShowCaller: true}
	log, err := NewZerologLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestZerologLogger_LogMethods(t *testing.T) {
	t.Parallel()
	// Create logger writing to a buffer so we can verify output
	var buf bytes.Buffer
	log := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	// All methods should not panic
	log.Trace("trace message")
	log.Debug("debug message")
	log.Info("info message")
	log.Warn("warn message")
	log.Error("error message")

	output := buf.String()
	assert.Contains(t, output, "trace message")
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestZerologLogger_LogMethods_WithFields(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	log.Info("with fields",
		String("key", "value"),
		Int("count", 42),
		Field{Key: "bool_field", Value: true},
		Field{Key: "float_field", Value: 3.14},
		Field{Key: "int64_field", Value: int64(1234567890)},
		Field{Key: "time_field", Value: time.Now()},
		Field{Key: "duration_field", Value: 5 * time.Second},
		Field{Key: "error_field", Value: errors.New("test error")},
		Field{Key: "slice_field", Value: []string{"a", "b"}},
	)

	output := buf.String()
	assert.Contains(t, output, "with fields")
	assert.Contains(t, output, "value")
}

func TestZerologLogger_WithFields(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	enriched := base.WithFields(String("component", "test"))
	require.NotNil(t, enriched)

	enriched.Info("enriched log")
	assert.Contains(t, buf.String(), "test")
}

func TestZerologLogger_WithContext(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger(),
	}

	ctx := context.Background()
	enriched := base.WithContext(ctx)
	require.NotNil(t, enriched)
}

func TestZerologLogger_WithComponent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	enriched := base.WithComponent("my-component")
	require.NotNil(t, enriched)

	enriched.Info("component log")
	assert.Contains(t, buf.String(), "my-component")
}

func TestZerologLogger_RequestLogger(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	reqLog := base.RequestLogger("req-abc")
	require.NotNil(t, reqLog)

	reqLog.Info("request log")
	assert.Contains(t, buf.String(), "req-abc")
}

func TestZerologLogger_DumpJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	base.DumpJSON(map[string]string{"key": "value"}, "dump test")
	assert.Contains(t, buf.String(), "dump test")
}

// --- Tracer tests ---

func TestZerologTracer(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	tracer := base.TraceOperation("test-op")
	require.NotNil(t, tracer)

	// Step
	tracer.Step("step1", String("detail", "something"))
	assert.Contains(t, buf.String(), "operation_step")

	// Duration should be > 0
	assert.True(t, tracer.Duration() >= 0)

	// Success
	tracer.Success(String("result", "ok"))
	assert.Contains(t, buf.String(), "operation_completed")
}

func TestZerologTracer_Error(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	tracer := base.TraceOperation("failing-op")
	tracer.Step("step1")
	tracer.Error(errors.New("something failed"))
	assert.Contains(t, buf.String(), "operation_failed")
}

// --- TimingLogger tests ---

func TestZerologTimingLogger(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	timing := base.Timing("perf-test")
	require.NotNil(t, timing)

	timing.Mark("checkpoint1")
	timing.Mark("checkpoint2", String("info", "extra"))
	timing.Complete(String("result", "done"))

	output := buf.String()
	assert.Contains(t, output, "timing_checkpoint")
	assert.Contains(t, output, "timing_completed")
}

func TestZerologTimingLogger_Abandon(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	timing := base.Timing("abandoned-op")
	timing.Mark("started")
	timing.Abandon("user cancelled")

	output := buf.String()
	assert.Contains(t, output, "timing_abandoned")
	assert.Contains(t, output, "user cancelled")
}

func TestZerologTimingLogger_CompleteWithoutCheckpoints(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	timing := base.Timing("quick-op")
	timing.Complete()

	assert.Contains(t, buf.String(), "timing_completed")
}

// --- operationTracer tests (from logger.go) ---

func TestOperationTracer(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: TraceLevel, Format: "json", Output: "stderr"}
	base, err := New(cfg)
	require.NoError(t, err)

	tracer := base.TraceOperation("test-operation")
	require.NotNil(t, tracer)

	tracer.Step("init")
	tracer.Step("process", String("items", "5"))

	dur := tracer.Duration()
	assert.True(t, dur >= 0)

	tracer.Success(String("total", "5"))
}

func TestOperationTracer_Error(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: TraceLevel, Format: "json", Output: "stderr"}
	base, err := New(cfg)
	require.NoError(t, err)

	tracer := base.TraceOperation("failing-operation")
	tracer.Step("attempt")
	tracer.Error(errors.New("failed"))
}

// --- timingLogger tests (from logger.go) ---

func TestTimingLogger(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: TraceLevel, Format: "json", Output: "stderr"}
	base, err := New(cfg)
	require.NoError(t, err)

	timing := base.Timing("sync-timing")
	timing.Mark("start")
	timing.Mark("middle")
	timing.Complete()
}

func TestTimingLogger_Abandon(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: TraceLevel, Format: "json", Output: "stderr"}
	base, err := New(cfg)
	require.NoError(t, err)

	timing := base.Timing("abandoned-timing")
	timing.Mark("start")
	timing.Abandon("timeout")
}

// --- MultiLogger tests ---

func TestNewMultiLogger_NoLoggers(t *testing.T) {
	t.Parallel()
	ml := NewMultiLogger()
	require.NotNil(t, ml)

	// Should not panic on any operation
	ml.Trace("trace")
	ml.Debug("debug")
	ml.Info("info")
	ml.Warn("warn")
	ml.Error("error")
	ml.DumpJSON(nil, "dump")
}

func TestNewMultiLogger_NilLoggers(t *testing.T) {
	t.Parallel()
	ml := NewMultiLogger(nil, nil, nil)
	require.NotNil(t, ml)
	ml.Info("should not panic")
}

func TestMultiLogger_RoutesToAll(t *testing.T) {
	t.Parallel()
	var buf1, buf2 bytes.Buffer
	l1 := &ZerologLogger{logger: zerolog.New(&buf1).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	l2 := &ZerologLogger{logger: zerolog.New(&buf2).With().Timestamp().Logger().Level(zerolog.TraceLevel)}

	ml := NewMultiLogger(l1, l2)
	ml.Info("test message")

	assert.Contains(t, buf1.String(), "test message")
	assert.Contains(t, buf2.String(), "test message")
}

func TestMultiLogger_AllMethods(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	ml.Trace("t")
	ml.Debug("d")
	ml.Info("i")
	ml.Warn("w")
	ml.Error("e")
	ml.DumpJSON(map[string]string{"k": "v"}, "dump")

	output := buf.String()
	assert.Contains(t, output, "\"t\"")
	assert.Contains(t, output, "\"d\"")
	assert.Contains(t, output, "\"i\"")
	assert.Contains(t, output, "\"w\"")
	assert.Contains(t, output, "\"e\"")
}

func TestMultiLogger_WithFields(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	enriched := ml.WithFields(String("key", "val"))
	require.NotNil(t, enriched)
	enriched.Info("enriched")
	assert.Contains(t, buf.String(), "val")
}

func TestMultiLogger_WithContext(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger()}
	ml := NewMultiLogger(l)

	enriched := ml.WithContext(context.Background())
	require.NotNil(t, enriched)
}

func TestMultiLogger_WithComponent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	enriched := ml.WithComponent("comp")
	require.NotNil(t, enriched)
	enriched.Info("component log")
	assert.Contains(t, buf.String(), "comp")
}

func TestMultiLogger_RequestLogger(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	reqLog := ml.RequestLogger("req-1")
	require.NotNil(t, reqLog)
	reqLog.Info("req log")
	assert.Contains(t, buf.String(), "req-1")
}

func TestMultiLogger_TraceOperation(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	tracer := ml.TraceOperation("multi-op")
	require.NotNil(t, tracer)
	tracer.Step("s1")
	tracer.Success()

	output := buf.String()
	assert.Contains(t, output, "operation_step")
	assert.Contains(t, output, "operation_completed")
}

func TestMultiTracer_Error(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	tracer := ml.TraceOperation("failing-multi-op")
	tracer.Error(errors.New("multi fail"))
	assert.Contains(t, buf.String(), "operation_failed")
}

func TestMultiTracer_Duration_Empty(t *testing.T) {
	t.Parallel()
	mt := &multiTracer{tracers: []Tracer{}}
	assert.Equal(t, time.Duration(0), mt.Duration())
}

func TestMultiLogger_Timing(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	timing := ml.Timing("multi-timing")
	require.NotNil(t, timing)
	timing.Mark("cp1")
	timing.Complete()
	assert.Contains(t, buf.String(), "timing_completed")
}

func TestMultiTimingLogger_Abandon(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	ml := NewMultiLogger(l)

	timing := ml.Timing("abandon-multi")
	timing.Mark("start")
	timing.Abandon("cancelled")
	assert.Contains(t, buf.String(), "timing_abandoned")
}

// --- CreateMultiLogger tests ---

func TestCreateMultiLogger_ConsoleOnly(t *testing.T) {
	t.Parallel()
	cfg := &Config{Level: InfoLevel, Format: "json", Output: "stderr"}
	log, err := CreateMultiLogger(cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestCreateMultiLogger_FileOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &Config{Level: DebugLevel, Format: "json", Output: "file", FilePath: filepath.Join(dir, "test.log")}
	log, err := CreateMultiLogger(nil, cfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestCreateMultiLogger_Both(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	consoleCfg := &Config{Level: InfoLevel, Format: "json", Output: "stderr"}
	fileCfg := &Config{Level: DebugLevel, Format: "json", Output: "file", FilePath: filepath.Join(dir, "test.log")}

	log, err := CreateMultiLogger(consoleCfg, fileCfg)
	require.NoError(t, err)
	assert.NotNil(t, log)
}

func TestCreateMultiLogger_NoLoggers(t *testing.T) {
	t.Parallel()
	_, err := CreateMultiLogger(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no loggers configured")
}

// --- RedactionHook tests ---

func TestRedactionHook_DoesNotCrash(t *testing.T) {
	t.Parallel()
	hook := RedactionHook{}
	// Just verify it doesn't panic
	hook.Run(nil, zerolog.InfoLevel, "test message")
}

// --- Global logger tests ---

func TestInitGlobal(t *testing.T) {
	// Not parallel since it modifies global state
	oldDefault := defaultLogger
	defer func() { defaultLogger = oldDefault }()

	cfg := DefaultConfig()
	err := InitGlobal(cfg)
	require.NoError(t, err)
	assert.NotNil(t, defaultLogger)
}

func TestInitGlobalWithLogger(t *testing.T) {
	oldDefault := defaultLogger
	defer func() { defaultLogger = oldDefault }()

	var buf bytes.Buffer
	custom := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger()}
	InitGlobalWithLogger(custom)
	assert.Equal(t, custom, defaultLogger)
}

func TestGlobalConvenienceFunctions(t *testing.T) {
	oldDefault := defaultLogger
	defer func() { defaultLogger = oldDefault }()

	var buf bytes.Buffer
	log := &ZerologLogger{logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel)}
	defaultLogger = log

	Trace("global trace")
	Debug("global debug")
	Info("global info")
	Warn("global warn")

	output := buf.String()
	assert.Contains(t, output, "global trace")
	assert.Contains(t, output, "global debug")
	assert.Contains(t, output, "global info")
	assert.Contains(t, output, "global warn")
}

func TestGlobalConvenienceFunctions_NilLogger(t *testing.T) {
	oldDefault := defaultLogger
	defer func() { defaultLogger = oldDefault }()

	defaultLogger = nil

	// Should not panic
	Trace("trace")
	Debug("debug")
	Info("info")
	Warn("warn")
}

func TestWithComponent_Global(t *testing.T) {
	oldDefault := defaultLogger
	defer func() { defaultLogger = oldDefault }()

	cfg := DefaultConfig()
	log, err := New(cfg)
	require.NoError(t, err)
	defaultLogger = log

	comp := WithComponent("my-comp")
	assert.NotNil(t, comp)
}

func TestWithComponent_Global_NilLogger(t *testing.T) {
	oldDefault := defaultLogger
	defer func() { defaultLogger = oldDefault }()

	defaultLogger = nil
	comp := WithComponent("comp")
	assert.Nil(t, comp)
}

// --- addFieldToContext type coverage ---

func TestZerologLogger_AddFieldToContext_AllTypes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	base := &ZerologLogger{
		logger: zerolog.New(&buf).With().Timestamp().Logger().Level(zerolog.TraceLevel),
	}

	enriched := base.WithFields(
		String("str", "hello"),
		Int("int_val", 42),
		Field{Key: "int64_val", Value: int64(999)},
		Field{Key: "float_val", Value: 3.14},
		Field{Key: "bool_val", Value: true},
		Field{Key: "err_val", Value: errors.New("err")},
		Field{Key: "dur_val", Value: 5 * time.Second},
		Field{Key: "time_val", Value: time.Now()},
		Field{Key: "other_val", Value: []string{"a", "b"}},
	)
	require.NotNil(t, enriched)
	enriched.Info("typed fields test")
}
