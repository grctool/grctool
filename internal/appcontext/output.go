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
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Output interface provides methods for user-facing output
type Output interface {
	Print(args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})
}

// CommandOutput wraps a cobra.Command for output
type CommandOutput struct {
	cmd *cobra.Command
}

// Print outputs to the command's output writer
func (c *CommandOutput) Print(args ...interface{}) {
	fmt.Fprint(c.cmd.OutOrStdout(), args...)
}

// Printf outputs formatted text to the command's output writer
func (c *CommandOutput) Printf(format string, args ...interface{}) {
	fmt.Fprintf(c.cmd.OutOrStdout(), format, args...)
}

// Println outputs to the command's output writer with a newline
func (c *CommandOutput) Println(args ...interface{}) {
	fmt.Fprintln(c.cmd.OutOrStdout(), args...)
}

// WriterOutput wraps an io.Writer for output
type WriterOutput struct {
	writer io.Writer
}

// Print outputs to the writer
func (w *WriterOutput) Print(args ...interface{}) {
	fmt.Fprint(w.writer, args...)
}

// Printf outputs formatted text to the writer
func (w *WriterOutput) Printf(format string, args ...interface{}) {
	fmt.Fprintf(w.writer, format, args...)
}

// Println outputs to the writer with a newline
func (w *WriterOutput) Println(args ...interface{}) {
	fmt.Fprintln(w.writer, args...)
}

// PrintErr outputs to the writer (for ErrorOutput interface)
func (w *WriterOutput) PrintErr(args ...interface{}) {
	fmt.Fprint(w.writer, args...)
}

// PrintfErr outputs formatted text to the writer (for ErrorOutput interface)
func (w *WriterOutput) PrintfErr(format string, args ...interface{}) {
	fmt.Fprintf(w.writer, format, args...)
}

// PrintlnErr outputs to the writer with a newline (for ErrorOutput interface)
func (w *WriterOutput) PrintlnErr(args ...interface{}) {
	fmt.Fprintln(w.writer, args...)
}

// StdOutput provides output to stdout
type StdOutput struct{}

// Print outputs to stdout
func (s *StdOutput) Print(args ...interface{}) {
	fmt.Print(args...)
}

// Printf outputs formatted text to stdout
func (s *StdOutput) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Println outputs to stdout with a newline
func (s *StdOutput) Println(args ...interface{}) {
	fmt.Println(args...)
}

// ServiceOutput extracts an Output implementation from the context
// It checks for: output writer > cobra command > stdout fallback
// Note: This is primarily for services that need to output to users but don't have direct access to the cobra command
// Command handlers should continue using cmd.Println() directly
func ServiceOutput(ctx context.Context) Output {
	// First, check for an explicit output writer
	if w := GetOutput(ctx); w != nil {
		return &WriterOutput{writer: w}
	}

	// Next, check for a cobra command
	if cmd := GetCommand(ctx); cmd != nil {
		return &CommandOutput{cmd: cmd}
	}

	// Fallback to stdout
	return &StdOutput{}
}

// Print is a convenience function that prints to the output from context
func Print(ctx context.Context, args ...interface{}) {
	ServiceOutput(ctx).Print(args...)
}

// Printf is a convenience function that prints formatted text to the output from context
func Printf(ctx context.Context, format string, args ...interface{}) {
	ServiceOutput(ctx).Printf(format, args...)
}

// Println is a convenience function that prints with newline to the output from context
func Println(ctx context.Context, args ...interface{}) {
	ServiceOutput(ctx).Println(args...)
}

// ErrorOutput provides access to error output stream
type ErrorOutput interface {
	PrintErr(args ...interface{})
	PrintfErr(format string, args ...interface{})
	PrintlnErr(args ...interface{})
}

// CommandErrorOutput wraps a cobra.Command for error output
type CommandErrorOutput struct {
	cmd *cobra.Command
}

// PrintErr outputs to the command's error writer
func (c *CommandErrorOutput) PrintErr(args ...interface{}) {
	fmt.Fprint(c.cmd.ErrOrStderr(), args...)
}

// PrintfErr outputs formatted text to the command's error writer
func (c *CommandErrorOutput) PrintfErr(format string, args ...interface{}) {
	fmt.Fprintf(c.cmd.ErrOrStderr(), format, args...)
}

// PrintlnErr outputs to the command's error writer with a newline
func (c *CommandErrorOutput) PrintlnErr(args ...interface{}) {
	fmt.Fprintln(c.cmd.ErrOrStderr(), args...)
}

// ServiceErrorOutput extracts an ErrorOutput implementation from the context
func ServiceErrorOutput(ctx context.Context) ErrorOutput {
	// Check for a cobra command first
	if cmd := GetCommand(ctx); cmd != nil {
		return &CommandErrorOutput{cmd: cmd}
	}

	// Check for an explicit error writer
	if w := GetError(ctx); w != nil {
		return &WriterOutput{writer: w}
	}

	// Fallback to stderr
	return &WriterOutput{writer: os.Stderr}
}
