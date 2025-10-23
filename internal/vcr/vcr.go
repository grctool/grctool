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

package vcr

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/logger"
	"gopkg.in/yaml.v3"
)

// Mode defines the VCR recording mode
type Mode string

const (
	// ModeOff disables VCR completely
	ModeOff Mode = "off"
	// ModeRecord records requests to cassettes
	ModeRecord Mode = "record"
	// ModePlayback plays back from existing cassettes
	ModePlayback Mode = "playback"
	// ModeRecordOnce records if cassette doesn't exist, otherwise plays back
	ModeRecordOnce Mode = "record_once"
)

// Config holds VCR configuration
type Config struct {
	Enabled         bool     `yaml:"enabled"`
	Mode            Mode     `yaml:"mode"`
	CassetteDir     string   `yaml:"cassette_dir"`
	CassetteName    string   `yaml:"cassette_name,omitempty"` // Optional: force specific cassette file
	SanitizeHeaders bool     `yaml:"sanitize_headers"`
	SanitizeParams  bool     `yaml:"sanitize_params"`
	RedactHeaders   []string `yaml:"redact_headers"`
	RedactParams    []string `yaml:"redact_params"`
	MatchMethod     bool     `yaml:"match_method"`
	MatchURI        bool     `yaml:"match_uri"`
	MatchQuery      bool     `yaml:"match_query"`
	MatchHeaders    bool     `yaml:"match_headers"`
	MatchBody       bool     `yaml:"match_body"`
}

// DefaultConfig returns default VCR configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:         false,
		Mode:            ModeOff,
		CassetteDir:     "docs/.cache/vcr",
		SanitizeHeaders: true,
		SanitizeParams:  true,
		RedactHeaders:   []string{"authorization", "cookie", "x-api-key", "token"},
		RedactParams:    []string{"api_key", "token", "password", "secret"},
		MatchMethod:     true,
		MatchURI:        true,
		MatchQuery:      false,
		MatchHeaders:    false,
		MatchBody:       false,
	}
}

// FromEnvironment creates VCR config from VCR_MODE environment variable
// This is the recommended way to control VCR in tests and development.
// Returns nil if VCR_MODE is not set or is "off".
func FromEnvironment() *Config {
	vcrMode := os.Getenv("VCR_MODE")
	if vcrMode == "" || vcrMode == "off" {
		// VCR disabled
		return nil
	}

	// Start with defaults
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Mode = Mode(vcrMode)

	// Allow override of cassette directory
	if cassetteDir := os.Getenv("VCR_CASSETTE_DIR"); cassetteDir != "" {
		cfg.CassetteDir = cassetteDir
	}

	return cfg
}

// Request represents a recorded HTTP request
type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// Response represents a recorded HTTP response
type Response struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// Interaction represents a single HTTP request/response pair
type Interaction struct {
	Request   Request   `json:"request"`
	Response  Response  `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// Cassette represents a collection of HTTP interactions
type Cassette struct {
	Name         string        `json:"name"`
	Interactions []Interaction `json:"interactions"`
	RecordedAt   time.Time     `json:"recorded_at"`
}

// Go-VCR compatible format structures (for loading existing cassettes)
type goVCRRequest struct {
	Body    *string             `yaml:"body"`
	Form    map[string][]string `yaml:"form"`
	Headers map[string][]string `yaml:"headers"`
	URL     string              `yaml:"uri"`
	Method  string              `yaml:"method"`
}

type goVCRResponse struct {
	Body struct {
		String string `yaml:"string"`
	} `yaml:"body"`
	Code    int                 `yaml:"code"`
	Headers map[string][]string `yaml:"headers"`
}

type goVCRInteraction struct {
	Request  goVCRRequest  `yaml:"request"`
	Response goVCRResponse `yaml:"response"`
}

type goVCRCassette struct {
	Interactions []goVCRInteraction `yaml:"interactions"`
}

// VCR handles HTTP request recording and playback
type VCR struct {
	config    *Config
	cassettes map[string]*Cassette
	logger    logger.Logger
	transport http.RoundTripper
}

// New creates a new VCR instance
func New(config *Config) *VCR {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize logger with fallback for testing
	vcrLogger := logger.WithComponent("vcr")
	if vcrLogger == nil {
		// Fallback to default logger for testing
		defaultLogger, _ := logger.New(logger.DefaultConfig())
		vcrLogger = defaultLogger.WithComponent("vcr")
	}

	return &VCR{
		config:    config,
		cassettes: make(map[string]*Cassette),
		logger:    vcrLogger,
		transport: http.DefaultTransport,
	}
}

// RoundTrip implements http.RoundTripper interface
func (v *VCR) RoundTrip(req *http.Request) (*http.Response, error) {
	if !v.config.Enabled || v.config.Mode == ModeOff {
		// VCR disabled, pass through to default transport
		return v.transport.RoundTrip(req)
	}

	// Use configured cassette name if provided, otherwise generate from request
	cassetteName := v.config.CassetteName
	if cassetteName == "" {
		cassetteName = v.generateCassetteName(req)
	}

	tracer := v.logger.TraceOperation("vcr_request")
	tracer.Step("request_processing",
		logger.String("cassette", cassetteName),
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.String("mode", string(v.config.Mode)),
	)

	switch v.config.Mode {
	case ModePlayback:
		resp, err := v.playback(cassetteName, req)
		if err != nil {
			tracer.Error(err, logger.String("operation", "playback"))
			return nil, fmt.Errorf("VCR playback failed: %w", err)
		}
		tracer.Success(logger.String("source", "cassette"))
		return resp, nil

	case ModeRecord:
		resp, err := v.recordAndExecute(cassetteName, req)
		if err != nil {
			tracer.Error(err, logger.String("operation", "record"))
			return nil, fmt.Errorf("VCR record failed: %w", err)
		}
		tracer.Success(logger.String("source", "live_recorded"))
		return resp, nil

	case ModeRecordOnce:
		if v.cassetteExists(cassetteName) {
			resp, err := v.playback(cassetteName, req)
			if err != nil {
				tracer.Error(err, logger.String("operation", "playback"))
				return nil, fmt.Errorf("VCR playback failed: %w", err)
			}
			tracer.Success(logger.String("source", "cassette"))
			return resp, nil
		} else {
			resp, err := v.recordAndExecute(cassetteName, req)
			if err != nil {
				tracer.Error(err, logger.String("operation", "record"))
				return nil, fmt.Errorf("VCR record failed: %w", err)
			}
			tracer.Success(logger.String("source", "live_recorded"))
			return resp, nil
		}

	default:
		tracer.Error(fmt.Errorf("unknown VCR mode: %s", v.config.Mode))
		return v.transport.RoundTrip(req)
	}
}

// generateCassetteName creates a unique name for the cassette based on request
func (v *VCR) generateCassetteName(req *http.Request) string {
	// Create a deterministic hash based on method, URL path, and normalized query parameters
	h := sha256.New()

	// Add method
	h.Write([]byte(req.Method))
	h.Write([]byte("|"))

	// Add URL path (without scheme/host for portability)
	h.Write([]byte(req.URL.Path))
	h.Write([]byte("|"))

	// Add sorted query parameters (excluding sensitive ones)
	if req.URL.RawQuery != "" {
		params := req.URL.Query()
		var keys []string
		for k := range params {
			// Skip potentially sensitive parameters
			if !v.isSensitiveParam(k) {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)

		for _, k := range keys {
			h.Write([]byte(k))
			h.Write([]byte("="))
			values := params[k]
			sort.Strings(values)
			h.Write([]byte(strings.Join(values, ",")))
			h.Write([]byte("&"))
		}
	}

	// Create readable prefix from URL
	urlParts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	var prefix strings.Builder
	prefix.WriteString(strings.ToLower(req.Method))
	prefix.WriteString("_")

	for i, part := range urlParts {
		if i > 2 { // Limit prefix length
			break
		}
		if part != "" {
			// Sanitize part for filename
			cleanPart := strings.ReplaceAll(part, "-", "_")
			cleanPart = strings.ReplaceAll(cleanPart, ".", "_")
			prefix.WriteString(cleanPart)
			if i < len(urlParts)-1 && i < 2 {
				prefix.WriteString("_")
			}
		}
	}

	// Add hash suffix for uniqueness
	hash := fmt.Sprintf("%x", h.Sum(nil))[:12]

	return fmt.Sprintf("%s_%s.json", prefix.String(), hash)
}

// isSensitiveParam checks if a query parameter contains sensitive data
func (v *VCR) isSensitiveParam(param string) bool {
	paramLower := strings.ToLower(param)

	for _, sensitive := range v.config.RedactParams {
		if strings.Contains(paramLower, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// isSensitiveHeader checks if a header contains sensitive data
func (v *VCR) isSensitiveHeader(header string) bool {
	headerLower := strings.ToLower(header)

	for _, sensitive := range v.config.RedactHeaders {
		if strings.Contains(headerLower, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// cassetteExists checks if a cassette file exists
func (v *VCR) cassetteExists(name string) bool {
	path := filepath.Join(v.config.CassetteDir, name)
	_, err := os.Stat(path)
	return err == nil
}

// playback replays a request from a cassette
func (v *VCR) playback(cassetteName string, req *http.Request) (*http.Response, error) {
	cassette, err := v.loadCassette(cassetteName)
	if err != nil {
		return nil, fmt.Errorf("failed to load cassette %s: %w", cassetteName, err)
	}

	// Find matching interaction
	interaction := v.findMatchingInteraction(cassette, req)
	if interaction == nil {
		return nil, fmt.Errorf("no matching interaction found in cassette %s for %s %s",
			cassetteName, req.Method, req.URL.String())
	}

	v.logger.Debug("playing back interaction",
		logger.String("cassette", cassetteName),
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.Int("status_code", interaction.Response.StatusCode),
	)

	// Create response from recorded interaction
	resp := &http.Response{
		Status:     interaction.Response.Status,
		StatusCode: interaction.Response.StatusCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(interaction.Response.Body)),
		Request:    req,
	}

	// Set headers
	for k, v := range interaction.Response.Headers {
		resp.Header.Set(k, v)
	}

	return resp, nil
}

// recordAndExecute executes the request and records it to a cassette
func (v *VCR) recordAndExecute(cassetteName string, req *http.Request) (*http.Response, error) {
	// Execute the actual request
	resp, err := v.transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()

	// Create new body reader for the returned response
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// Record the interaction
	if err := v.recordInteraction(cassetteName, req, resp, string(body)); err != nil {
		v.logger.Warn("failed to record interaction",
			logger.Error(err),
			logger.String("cassette", cassetteName),
		)
		// Don't fail the request if recording fails
	}

	return resp, nil
}

// recordInteraction saves an HTTP interaction to a cassette
func (v *VCR) recordInteraction(cassetteName string, req *http.Request, resp *http.Response, responseBody string) error {
	// Read request body if present
	var requestBody string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		requestBody = string(bodyBytes)
	}

	// Create interaction
	interaction := Interaction{
		Request: Request{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: v.sanitizeHeaders(req.Header),
			Body:    requestBody,
		},
		Response: Response{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    v.sanitizeHeaders(resp.Header),
			Body:       responseBody,
		},
		Timestamp: time.Now(),
	}

	// Load existing cassette or create new one
	cassette, err := v.loadCassette(cassetteName)
	if err != nil {
		// Create new cassette
		cassette = &Cassette{
			Name:         cassetteName,
			Interactions: []Interaction{},
			RecordedAt:   time.Now(),
		}
	}

	// Add interaction to cassette
	cassette.Interactions = append(cassette.Interactions, interaction)

	// Save cassette
	if err := v.saveCassette(cassetteName, cassette); err != nil {
		return fmt.Errorf("failed to save cassette: %w", err)
	}

	v.logger.Debug("recorded interaction",
		logger.String("cassette", cassetteName),
		logger.String("method", req.Method),
		logger.String("url", req.URL.String()),
		logger.Int("status_code", resp.StatusCode),
		logger.Int("total_interactions", len(cassette.Interactions)),
	)

	return nil
}

// sanitizeHeaders removes sensitive headers from recorded interactions
func (v *VCR) sanitizeHeaders(headers http.Header) map[string]string {
	sanitized := make(map[string]string)

	for k, headerValues := range headers {
		key := strings.ToLower(k)
		isSensitive := false

		if v.config.SanitizeHeaders {
			for _, sensitive := range v.config.RedactHeaders {
				if strings.Contains(key, strings.ToLower(sensitive)) {
					isSensitive = true
					break
				}
			}
		}

		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else if len(headerValues) > 0 {
			sanitized[k] = headerValues[0] // Take first value
		}
	}

	return sanitized
}

// findMatchingInteraction finds an interaction that matches the request
func (v *VCR) findMatchingInteraction(cassette *Cassette, req *http.Request) *Interaction {
	for _, interaction := range cassette.Interactions {
		if v.requestMatches(&interaction.Request, req) {
			return &interaction
		}
	}
	return nil
}

// requestMatches checks if a recorded request matches the current request
func (v *VCR) requestMatches(recorded *Request, current *http.Request) bool {
	// Check method if configured
	if v.config.MatchMethod && recorded.Method != current.Method {
		return false
	}

	// Parse URLs for comparison
	recordedURL, err := url.Parse(recorded.URL)
	if err != nil {
		return false
	}

	// Check URI path if configured
	if v.config.MatchURI && recordedURL.Path != current.URL.Path {
		return false
	}

	// Check query parameters if configured
	if v.config.MatchQuery {
		recordedParams := recordedURL.Query()
		currentParams := current.URL.Query()

		for k, recordedValues := range recordedParams {
			if v.isSensitiveParam(k) {
				continue
			}

			currentValues, exists := currentParams[k]
			if !exists {
				return false
			}

			sort.Strings(recordedValues)
			sort.Strings(currentValues)

			if len(recordedValues) != len(currentValues) {
				return false
			}

			for i, val := range recordedValues {
				if val != currentValues[i] {
					return false
				}
			}
		}
	}

	// Check headers if configured
	if v.config.MatchHeaders {
		for k, recordedValues := range recorded.Headers {
			if v.isSensitiveHeader(k) {
				continue
			}
			currentValues := current.Header[k]
			if !reflect.DeepEqual(recordedValues, currentValues) {
				return false
			}
		}
	}

	// Check body if configured
	if v.config.MatchBody && recorded.Body != "" {
		currentBody, _ := io.ReadAll(current.Body)
		current.Body = io.NopCloser(bytes.NewReader(currentBody))
		if recorded.Body != string(currentBody) {
			return false
		}
	}

	return true
}

// loadCassette loads a cassette from disk
func (v *VCR) loadCassette(name string) (*Cassette, error) {
	// Check cache first
	if cassette, exists := v.cassettes[name]; exists {
		return cassette, nil
	}

	path := filepath.Join(v.config.CassetteDir, name)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Provide a helpful error message when cassette is missing
			return nil, fmt.Errorf(`cassette not found: %s

VCR is in %s mode but the cassette file is missing.

To fix this, you need to record the cassette first:
  VCR_MODE=record make test-record

This will create the cassette files by making real API calls.
Make sure you have valid credentials set:
  export GITHUB_TOKEN=your_token_here
  export TUGBOAT_BEARER=your_token_here

After recording, you can run tests in playback mode:
  make test-integration  (uses VCR playback by default)`,
				path, v.config.Mode)
		}
		return nil, fmt.Errorf("failed to read cassette file %s: %w", path, err)
	}

	var cassette *Cassette

	// Try loading as YAML (go-vcr format) first
	if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		var goVCR goVCRCassette
		if err := yaml.Unmarshal(data, &goVCR); err == nil {
			// Convert go-vcr format to our internal format
			cassette = v.convertGoVCRCassette(name, &goVCR)
		} else {
			return nil, fmt.Errorf("failed to unmarshal YAML cassette %s: %w", name, err)
		}
	} else {
		// Try JSON format
		var jsonCassette Cassette
		if err := json.Unmarshal(data, &jsonCassette); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cassette %s: %w", name, err)
		}
		cassette = &jsonCassette
	}

	// Cache the cassette
	v.cassettes[name] = cassette

	return cassette, nil
}

// convertGoVCRCassette converts go-vcr format to our internal format
func (v *VCR) convertGoVCRCassette(name string, goVCR *goVCRCassette) *Cassette {
	cassette := &Cassette{
		Name:         name,
		Interactions: make([]Interaction, len(goVCR.Interactions)),
		RecordedAt:   time.Now(),
	}

	for i, gv := range goVCR.Interactions {
		// Convert headers from []string to single string
		reqHeaders := make(map[string]string)
		for k, v := range gv.Request.Headers {
			if len(v) > 0 {
				reqHeaders[k] = v[0] // Take first value
			}
		}

		respHeaders := make(map[string]string)
		for k, v := range gv.Response.Headers {
			if len(v) > 0 {
				respHeaders[k] = v[0] // Take first value
			}
		}

		// Get request body
		reqBody := ""
		if gv.Request.Body != nil {
			reqBody = *gv.Request.Body
		}

		// Determine status code and status string
		statusCode := gv.Response.Code
		if statusCode == 0 {
			statusCode = 200 // Default to 200 if not set
		}
		status := fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))

		cassette.Interactions[i] = Interaction{
			Request: Request{
				Method:  gv.Request.Method,
				URL:     gv.Request.URL,
				Headers: reqHeaders,
				Body:    reqBody,
			},
			Response: Response{
				StatusCode: statusCode,
				Status:     status,
				Headers:    respHeaders,
				Body:       gv.Response.Body.String,
			},
			Timestamp: time.Now(),
		}
	}

	return cassette
}

// saveCassette saves a cassette to disk
func (v *VCR) saveCassette(name string, cassette *Cassette) error {
	// Ensure directory exists
	if err := os.MkdirAll(v.config.CassetteDir, 0755); err != nil {
		return fmt.Errorf("failed to create cassette directory: %w", err)
	}

	path := filepath.Join(v.config.CassetteDir, name)

	data, err := json.MarshalIndent(cassette, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cassette: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cassette file: %w", err)
	}

	// Update cache
	v.cassettes[name] = cassette

	return nil
}

// Stats returns statistics about loaded cassettes
func (v *VCR) Stats() map[string]interface{} {
	stats := map[string]interface{}{
		"mode":             string(v.config.Mode),
		"enabled":          v.config.Enabled,
		"cassette_dir":     v.config.CassetteDir,
		"loaded_cassettes": len(v.cassettes),
	}

	var totalInteractions int
	for _, cassette := range v.cassettes {
		totalInteractions += len(cassette.Interactions)
	}
	stats["total_interactions"] = totalInteractions

	return stats
}
