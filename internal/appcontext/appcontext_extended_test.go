// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package appcontext

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Context key roundtrip tests ---

func TestWithOutput_GetOutput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var buf bytes.Buffer
	ctx = WithOutput(ctx, &buf)

	w := GetOutput(ctx)
	require.NotNil(t, w)
	_, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", buf.String())
}

func TestWithError_GetError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	var buf bytes.Buffer
	ctx = WithError(ctx, &buf)

	w := GetError(ctx)
	require.NotNil(t, w)
	_, err := w.Write([]byte("error msg"))
	require.NoError(t, err)
	assert.Equal(t, "error msg", buf.String())
}

func TestContextKeys_Isolation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Set different values for each key
	ctx = WithLogger(ctx, "test-logger")
	ctx = WithConfig(ctx, "test-config")
	ctx = WithVCRConfig(ctx, "test-vcr")
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")
	ctx = WithOperation(ctx, "sync")

	// Verify each retrieves the right value
	assert.Equal(t, "test-logger", GetLogger(ctx))
	assert.Equal(t, "test-config", GetConfig(ctx))
	assert.Equal(t, "test-vcr", GetVCRConfig(ctx))
	assert.Equal(t, "req-123", GetRequestID(ctx))
	assert.Equal(t, "user-456", GetUserID(ctx))
	assert.Equal(t, "sync", GetOperation(ctx))
}

func TestContextKeys_Overwrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ctx = WithRequestID(ctx, "first")
	assert.Equal(t, "first", GetRequestID(ctx))

	ctx = WithRequestID(ctx, "second")
	assert.Equal(t, "second", GetRequestID(ctx))
}

// --- CommandContext tests ---

func TestCommandContext(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test-cmd",
		Short: "A test command",
	}

	ctx := CommandContext(cmd)
	require.NotNil(t, ctx)

	// Should have command
	assert.Equal(t, cmd, GetCommand(ctx))

	// Should have operation name from command
	assert.Equal(t, "test-cmd", GetOperation(ctx))

	// Should have a request ID
	reqID := GetRequestID(ctx)
	assert.NotEmpty(t, reqID)
	assert.True(t, strings.HasPrefix(reqID, "req-"), "request ID should start with 'req-'")

	// Should have output writer
	assert.NotNil(t, GetOutput(ctx))

	// Should have error writer
	assert.NotNil(t, GetError(ctx))
}

// --- PreRunE tests ---

func TestPreRunE(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "pre-run-test",
		Short: "Test PreRunE",
	}

	err := PreRunE(cmd, []string{})
	require.NoError(t, err)

	// Command should have context set
	ctx := cmd.Context()
	require.NotNil(t, ctx)

	// Context should be enriched
	assert.Equal(t, cmd, GetCommand(ctx))
	assert.Equal(t, "pre-run-test", GetOperation(ctx))
	assert.NotEmpty(t, GetRequestID(ctx))
}

// --- WrapRunE tests ---

func TestWrapRunE(t *testing.T) {
	var capturedCtx context.Context
	innerFunc := func(cmd *cobra.Command, args []string) error {
		capturedCtx = cmd.Context()
		return nil
	}

	wrapped := WrapRunE(innerFunc)
	cmd := &cobra.Command{Use: "wrapped-test"}

	err := wrapped(cmd, []string{})
	require.NoError(t, err)
	require.NotNil(t, capturedCtx)

	// Should have context enrichment
	assert.Equal(t, cmd, GetCommand(capturedCtx))
	assert.NotEmpty(t, GetRequestID(capturedCtx))
}

func TestWrapRunE_ExistingContext(t *testing.T) {
	// If context already exists, WrapRunE should not overwrite it
	var capturedReqID string
	innerFunc := func(cmd *cobra.Command, args []string) error {
		capturedReqID = GetRequestID(cmd.Context())
		return nil
	}

	wrapped := WrapRunE(innerFunc)
	cmd := &cobra.Command{Use: "existing-ctx"}

	// Pre-set context
	preCtx := context.WithValue(context.Background(), requestIDKey, "pre-existing-id")
	cmd.SetContext(preCtx)

	err := wrapped(cmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "pre-existing-id", capturedReqID)
}

// --- GetOrCreateContext tests ---

func TestGetOrCreateContext_CreatesNew(t *testing.T) {
	cmd := &cobra.Command{Use: "get-or-create"}
	// No context set

	ctx := GetOrCreateContext(cmd)
	require.NotNil(t, ctx)
	assert.Equal(t, cmd, GetCommand(ctx))
	assert.NotEmpty(t, GetRequestID(ctx))
}

func TestGetOrCreateContext_ReturnsExisting(t *testing.T) {
	cmd := &cobra.Command{Use: "existing"}
	existing := context.WithValue(context.Background(), requestIDKey, "existing-req")
	cmd.SetContext(existing)

	ctx := GetOrCreateContext(cmd)
	assert.Equal(t, "existing-req", GetRequestID(ctx))
}

// --- EnrichContext tests ---

func TestEnrichContext_NilCommand(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	enriched := EnrichContext(ctx, nil)
	// Should not panic, command should be nil
	assert.Nil(t, GetCommand(enriched))
}

func TestEnrichContext_SetsOutputAndError(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	ctx := EnrichContext(context.Background(), cmd)

	outWriter := GetOutput(ctx)
	require.NotNil(t, outWriter)

	errWriter := GetError(ctx)
	require.NotNil(t, errWriter)
}

// --- Output interface tests ---

func TestCommandOutput(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(&buf)

	out := &CommandOutput{cmd: cmd}
	out.Print("hello")
	assert.Equal(t, "hello", buf.String())

	buf.Reset()
	out.Printf("hi %s", "world")
	assert.Equal(t, "hi world", buf.String())

	buf.Reset()
	out.Println("line")
	assert.Equal(t, "line\n", buf.String())
}

func TestWriterOutput(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	out := &WriterOutput{writer: &buf}

	out.Print("hello")
	assert.Equal(t, "hello", buf.String())

	buf.Reset()
	out.Printf("hi %s", "world")
	assert.Equal(t, "hi world", buf.String())

	buf.Reset()
	out.Println("line")
	assert.Equal(t, "line\n", buf.String())
}

func TestWriterOutput_ErrorMethods(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	out := &WriterOutput{writer: &buf}

	out.PrintErr("err")
	assert.Equal(t, "err", buf.String())

	buf.Reset()
	out.PrintfErr("err %d", 42)
	assert.Equal(t, "err 42", buf.String())

	buf.Reset()
	out.PrintlnErr("err line")
	assert.Equal(t, "err line\n", buf.String())
}

func TestStdOutput_DoesNotPanic(t *testing.T) {
	// Just verify StdOutput methods don't panic - we can't easily capture stdout
	out := &StdOutput{}
	// These write to real stdout but shouldn't panic
	out.Print("")
	out.Printf("%s", "")
	out.Println("")
}

// --- ServiceOutput tests ---

func TestServiceOutput_WithOutputWriter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := context.Background()
	ctx = WithOutput(ctx, &buf)

	out := ServiceOutput(ctx)
	require.NotNil(t, out)
	out.Print("via service output")
	assert.Equal(t, "via service output", buf.String())
}

func TestServiceOutput_WithCommand(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(&buf)

	ctx := context.Background()
	ctx = WithCommand(ctx, cmd)

	out := ServiceOutput(ctx)
	require.NotNil(t, out)
	out.Print("via command")
	assert.Equal(t, "via command", buf.String())
}

func TestServiceOutput_FallbackToStdout(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	out := ServiceOutput(ctx)
	require.NotNil(t, out)
	// Should be StdOutput
	_, ok := out.(*StdOutput)
	assert.True(t, ok)
}

// --- Convenience functions ---

func TestPrint_Convenience(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := WithOutput(context.Background(), &buf)

	Print(ctx, "hello")
	assert.Equal(t, "hello", buf.String())
}

func TestPrintf_Convenience(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := WithOutput(context.Background(), &buf)

	Printf(ctx, "hi %s", "world")
	assert.Equal(t, "hi world", buf.String())
}

func TestPrintln_Convenience(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := WithOutput(context.Background(), &buf)

	Println(ctx, "line")
	assert.Equal(t, "line\n", buf.String())
}

// --- ServiceErrorOutput tests ---

func TestServiceErrorOutput_WithCommand(t *testing.T) {
	var errBuf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}
	cmd.SetErr(&errBuf)

	ctx := context.Background()
	ctx = WithCommand(ctx, cmd)

	out := ServiceErrorOutput(ctx)
	require.NotNil(t, out)
	out.PrintErr("error message")
	assert.Equal(t, "error message", errBuf.String())
}

func TestServiceErrorOutput_WithErrorWriter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := context.Background()
	ctx = WithError(ctx, &buf)

	out := ServiceErrorOutput(ctx)
	require.NotNil(t, out)
	out.PrintErr("error via writer")
	assert.Equal(t, "error via writer", buf.String())
}

func TestServiceErrorOutput_FallbackToStderr(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	out := ServiceErrorOutput(ctx)
	require.NotNil(t, out)
}

func TestCommandErrorOutput(t *testing.T) {
	var errBuf bytes.Buffer
	cmd := &cobra.Command{Use: "test"}
	cmd.SetErr(&errBuf)

	out := &CommandErrorOutput{cmd: cmd}

	out.PrintErr("e1")
	assert.Equal(t, "e1", errBuf.String())

	errBuf.Reset()
	out.PrintfErr("e%d", 2)
	assert.Equal(t, "e2", errBuf.String())

	errBuf.Reset()
	out.PrintlnErr("e3")
	assert.Equal(t, "e3\n", errBuf.String())
}

// --- GenerateRequestID tests ---

func TestGenerateRequestID(t *testing.T) {
	t.Parallel()
	id := GenerateRequestID()
	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "req-"))
	// Should be unique
	id2 := GenerateRequestID()
	assert.NotEqual(t, id, id2)
}

func TestGenerateRequestID_Format(t *testing.T) {
	t.Parallel()
	id := GenerateRequestID()
	// Should be "req-" + 16 hex chars (8 bytes)
	assert.Len(t, id, 4+16) // "req-" + 16 hex chars
}

// --- GenerateOperationID tests ---

func TestGenerateOperationID(t *testing.T) {
	t.Parallel()
	id := GenerateOperationID("sync")
	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "sync-"))

	id2 := GenerateOperationID("sync")
	assert.NotEqual(t, id, id2, "IDs should be unique")
}

func TestGenerateOperationID_Format(t *testing.T) {
	t.Parallel()
	id := GenerateOperationID("test")
	// Should be "test-" + 8 hex chars (4 bytes)
	assert.Len(t, id, 5+8) // "test-" + 8 hex chars
}
