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

package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
)

// TypedTerraformTool provides Terraform configuration scanning with typed requests/responses
type TypedTerraformTool struct {
	*TerraformTool // Embed the original tool for compatibility
}

// NewTypedTerraformTool creates a new TypedTerraformTool
func NewTypedTerraformTool(cfg *config.Config, log logger.Logger) *TypedTerraformTool {
	return &TypedTerraformTool{
		TerraformTool: &TerraformTool{
			config: &cfg.Evidence.Tools.Terraform,
			logger: log,
		},
	}
}

// ExecuteTyped runs the Terraform scanner with a typed request
func (tt *TypedTerraformTool) ExecuteTyped(ctx context.Context, req types.Request) (types.Response, error) {
	// Type assert to TerraformRequest
	terraformReq, ok := req.(*types.TerraformRequest)
	if !ok {
		return types.NewErrorResponse(tt.Name(), "invalid request type, expected TerraformRequest", nil), nil
	}

	tt.logger.Debug("Executing Terraform scanner with typed request",
		logger.Field{Key: "resource_types", Value: terraformReq.ResourceTypes},
		logger.Field{Key: "control_codes", Value: terraformReq.ControlCodes},
		logger.Field{Key: "output_format", Value: terraformReq.OutputFormat})

	// Use scan paths from request if provided, otherwise use config
	scanPaths := terraformReq.ScanPaths
	if len(scanPaths) == 0 {
		scanPaths = tt.config.ScanPaths
	}

	// Execute scan
	var results []models.TerraformScanResult
	var err error

	if len(terraformReq.ControlCodes) > 0 {
		// Scan for specific security controls
		results, err = tt.scanForControlsTyped(ctx, terraformReq.ControlCodes, terraformReq.ResourceTypes, scanPaths)
		if err != nil {
			return types.NewErrorResponse(tt.Name(), fmt.Sprintf("failed to scan for security controls: %v", err), nil), nil
		}
	} else {
		// Scan for specific resource types or all resources
		results, err = tt.scanForResourcesTyped(ctx, terraformReq.ResourceTypes, scanPaths)
		if err != nil {
			return types.NewErrorResponse(tt.Name(), fmt.Sprintf("failed to scan resources: %v", err), nil), nil
		}
	}

	// Generate report (no git hash in typed version)
	report, err := tt.GenerateEvidenceReport(results, terraformReq.OutputFormat, "")
	if err != nil {
		return types.NewErrorResponse(tt.Name(), fmt.Sprintf("failed to generate report: %v", err), nil), nil
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "terraform",
		Resource:    fmt.Sprintf("Scanned %d Terraform resources", len(results)),
		Content:     report,
		Relevance:   tt.calculateRelevance(results),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"resource_count": len(results),
			"scan_paths":     scanPaths,
			"format":         terraformReq.OutputFormat,
		},
	}

	// Create typed response
	response := &types.TerraformResponse{
		ToolResponse: types.NewSuccessResponse(tt.Name(), report, source, map[string]interface{}{
			"request_type": "TerraformRequest",
		}),
		Results:          results,
		ResourceCount:    len(results),
		SecurityFindings: tt.countSecurityFindings(results),
		ScannedPaths:     scanPaths,
		Format:           terraformReq.OutputFormat,
	}

	return response, nil
}

// scanForControlsTyped scans for resources related to specific security controls (typed version)
func (tt *TypedTerraformTool) scanForControlsTyped(ctx context.Context, controlCodes, resourceTypes []string, scanPaths []string) ([]models.TerraformScanResult, error) {
	// For this implementation, we'll use a simplified approach similar to the original
	// In a full implementation, you'd load security mappings from configuration

	var allResults []models.TerraformScanResult

	for _, scanPath := range scanPaths {
		results, err := tt.scanPathTyped(ctx, scanPath, resourceTypes)
		if err != nil {
			tt.logger.Warn("Failed to scan path",
				logger.Field{Key: "path", Value: scanPath},
				logger.Field{Key: "error", Value: err})
			continue
		}
		allResults = append(allResults, results...)
	}

	// Filter by control codes
	var filteredResults []models.TerraformScanResult
	for _, result := range allResults {
		for _, relevantControl := range result.SecurityRelevance {
			for _, requestedControl := range controlCodes {
				if relevantControl == requestedControl {
					filteredResults = append(filteredResults, result)
					goto nextResult
				}
			}
		}
	nextResult:
	}

	return filteredResults, nil
}

// scanForResourcesTyped scans for specific resource types (typed version)
func (tt *TypedTerraformTool) scanForResourcesTyped(ctx context.Context, resourceTypes []string, scanPaths []string) ([]models.TerraformScanResult, error) {
	var allResults []models.TerraformScanResult

	for _, scanPath := range scanPaths {
		results, err := tt.scanPathTyped(ctx, scanPath, resourceTypes)
		if err != nil {
			tt.logger.Warn("Failed to scan path",
				logger.Field{Key: "path", Value: scanPath},
				logger.Field{Key: "error", Value: err})
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// scanPathTyped scans a specific path with typed parameters
func (tt *TypedTerraformTool) scanPathTyped(ctx context.Context, scanPath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
	// Reuse the existing scanPath method from the embedded TerraformTool
	return tt.scanPath(ctx, scanPath, resourceTypes)
}

// countSecurityFindings counts the number of security-relevant findings
func (tt *TypedTerraformTool) countSecurityFindings(results []models.TerraformScanResult) int {
	count := 0
	for _, result := range results {
		if len(result.SecurityRelevance) > 0 {
			count++
		}
	}
	return count
}

// Ensure TypedTerraformTool implements both Tool and types.Tool interfaces
var _ Tool = (*TypedTerraformTool)(nil)
var _ types.TypedTool = (*TypedTerraformTool)(nil)
