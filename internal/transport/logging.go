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

package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

// LoggingTransport wraps an http.RoundTripper to log requests and responses
type LoggingTransport struct {
	Transport http.RoundTripper
	Logger    logger.Logger
}

// NewLoggingTransport creates a new logging transport
func NewLoggingTransport(transport http.RoundTripper, log logger.Logger) *LoggingTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if log == nil {
		log = logger.WithComponent("http")
		// If global logger isn't initialized, create a default one
		if log == nil {
			defaultConfig := logger.DefaultConfig()
			newLog, err := logger.New(defaultConfig)
			if err == nil {
				log = newLog.WithComponent("http")
			}
		}
	}
	return &LoggingTransport{
		Transport: transport,
		Logger:    log,
	}
}

// RoundTrip implements the http.RoundTripper interface
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Clone the request to avoid mutating the original
	reqClone := req.Clone(req.Context())

	// Read and restore request body if present
	var reqBody []byte
	if req.Body != nil {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			t.Logger.Error("failed to read request body for logging", logger.Error(err))
		} else {
			req.Body = io.NopCloser(bytes.NewReader(reqBody))
			reqClone.Body = io.NopCloser(bytes.NewReader(reqBody))
		}
	}

	// Debug log: method, URL, and basic info
	t.Logger.Debug("http request",
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.String("host", req.Host),
		logger.Int("content_length", int(req.ContentLength)),
	)

	// Trace log: full request in curl format (no redaction as requested)
	t.logRequestAsCurl(reqClone, reqBody)

	// Execute the request
	resp, err := t.Transport.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		t.Logger.Debug("http request failed",
			logger.String("method", req.Method),
			logger.String("url", req.URL.String()),
			logger.Duration("duration", duration),
			logger.Error(err),
		)
		return nil, err
	}

	// Clone response body for logging
	var respBody []byte
	if resp.Body != nil {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Logger.Error("failed to read response body for logging", logger.Error(err))
		} else {
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
		}
	}

	// Debug log: response code and basic info
	t.Logger.Debug("http response",
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.Int("status_code", resp.StatusCode),
		logger.String("status", resp.Status),
		logger.Duration("duration", duration),
		logger.Int("content_length", int(resp.ContentLength)),
	)

	// Trace log: full response headers and body
	t.logResponse(resp, respBody)

	return resp, nil
}

// logRequestAsCurl logs the request in curl format at trace level
func (t *LoggingTransport) logRequestAsCurl(req *http.Request, body []byte) {
	var curlCmd strings.Builder

	// Start with curl command
	curlCmd.WriteString("curl")

	// Add method if not GET
	if req.Method != "GET" {
		curlCmd.WriteString(fmt.Sprintf(" -X %s", req.Method))
	}

	// Add headers (no redaction as requested)
	for key, values := range req.Header {
		for _, value := range values {
			curlCmd.WriteString(fmt.Sprintf(" -H '%s: %s'", key, value))
		}
	}

	// Add body if present
	if len(body) > 0 {
		// Escape single quotes in body
		bodyStr := strings.ReplaceAll(string(body), "'", "'\"'\"'")
		curlCmd.WriteString(fmt.Sprintf(" -d '%s'", bodyStr))
	}

	// Add URL
	curlCmd.WriteString(fmt.Sprintf(" '%s'", req.URL.String()))

	// Also log the raw request dump
	reqDump, err := httputil.DumpRequestOut(req, len(body) > 0)
	if err != nil {
		t.Logger.Trace("http request (curl format)",
			logger.String("curl", curlCmd.String()),
			logger.String("error", "failed to dump request"),
		)
	} else {
		t.Logger.Trace("http request details",
			logger.String("curl", curlCmd.String()),
			logger.String("raw_request", string(reqDump)),
		)
	}
}

// logResponse logs the full response at trace level
func (t *LoggingTransport) logResponse(resp *http.Response, body []byte) {
	// Create a response dump
	respDump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		t.Logger.Trace("http response details",
			logger.String("error", "failed to dump response"),
		)
		return
	}

	// Log response headers and body separately for clarity
	t.Logger.Trace("http response details",
		logger.String("raw_headers", string(respDump)),
		logger.String("body", string(body)),
		logger.Int("body_length", len(body)),
	)
}
