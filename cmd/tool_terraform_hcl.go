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

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// terraformHCLCmd represents the terraform-hcl-parser command
var terraformHCLCmd = &cobra.Command{
	Use:   "terraform-hcl-parser",
	Short: "Comprehensive HCL parser for Terraform manifests with infrastructure topology analysis",
	Long: `The terraform-hcl-parser tool provides comprehensive parsing of Terraform HCL files with:

• Full HCL syntax parsing using hashicorp/hcl/v2
• Infrastructure topology mapping
• Multi-AZ pattern detection
• High availability analysis  
• Security configuration analysis
• SOC2 control mapping
• Resource dependency analysis

This tool parses .tf files only (NO state files, NO secrets) and extracts:
- All resource declarations with full configuration
- Variables, locals, outputs, and module calls
- Provider configurations and versions
- Multi-availability zone patterns
- High availability configurations
- Security-relevant findings
- Infrastructure relationships

Output formats: detailed (JSON), summary (Markdown), security-only (Markdown)`,
	RunE: runTerraformHCL,
}

// terraformHCLFlags defines the command-line flags
type terraformHCLFlags struct {
	scanPaths       []string
	includeModules  bool
	analyzeSecurity bool
	analyzeHA       bool
	controlMapping  []string
	outputFormat    string
	includeDiags    bool
}

var terraformHCLOpts = terraformHCLFlags{}

func init() {
	toolCmd.AddCommand(terraformHCLCmd)

	// Scan configuration
	terraformHCLCmd.Flags().StringSliceVar(&terraformHCLOpts.scanPaths, "scan-paths", nil,
		"Paths to scan for Terraform files (supports globs)")
	terraformHCLCmd.Flags().BoolVar(&terraformHCLOpts.includeModules, "include-modules", true,
		"Include module dependencies in analysis")

	// Analysis options
	terraformHCLCmd.Flags().BoolVar(&terraformHCLOpts.analyzeSecurity, "analyze-security", true,
		"Perform security analysis and generate findings")
	terraformHCLCmd.Flags().BoolVar(&terraformHCLOpts.analyzeHA, "analyze-ha", true,
		"Analyze high availability and multi-AZ configurations")

	// Control mapping
	terraformHCLCmd.Flags().StringSliceVar(&terraformHCLOpts.controlMapping, "control-mapping", nil,
		"Specific SOC2 control codes to map resources to (e.g., CC6.1,CC6.8)")

	// Output options
	terraformHCLCmd.Flags().StringVar(&terraformHCLOpts.outputFormat, "output-format", "summary",
		"Output format: detailed, summary, or security-only")
	terraformHCLCmd.Flags().BoolVar(&terraformHCLOpts.includeDiags, "include-diagnostics", false,
		"Include HCL parsing diagnostics in output")

	// Mark required flags
	terraformHCLCmd.MarkFlagRequired("scan-paths")
}

// runTerraformHCL executes the terraform HCL parser tool
func runTerraformHCL(cmd *cobra.Command, args []string) error {
	// Validate parameters
	if err := validateTerraformHCLParams(); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Build tool parameters
	params := map[string]interface{}{
		"scan_paths":          terraformHCLOpts.scanPaths,
		"include_modules":     terraformHCLOpts.includeModules,
		"analyze_security":    terraformHCLOpts.analyzeSecurity,
		"analyze_ha":          terraformHCLOpts.analyzeHA,
		"control_mapping":     terraformHCLOpts.controlMapping,
		"output_format":       terraformHCLOpts.outputFormat,
		"include_diagnostics": terraformHCLOpts.includeDiags,
	}

	// Execute tool
	validationRules := map[string]tools.ValidationRule{
		"scan_paths": {
			Required: true,
			Type:     "string_array",
		},
		"include_modules":  BoolRule,
		"analyze_security": BoolRule,
		"analyze_ha":       BoolRule,
		"output_format": {
			Required:      true,
			Type:          "string",
			AllowedValues: []string{"detailed", "summary", "security-only"},
		},
	}

	return ValidateAndExecuteTool(cmd, "terraform-hcl-parser", params, validationRules)
}

// validateTerraformHCLParams validates the command parameters
func validateTerraformHCLParams() error {
	// Validate scan paths
	if len(terraformHCLOpts.scanPaths) == 0 {
		return fmt.Errorf("at least one scan path is required")
	}

	// Validate output format
	validFormats := map[string]bool{
		"detailed":      true,
		"summary":       true,
		"security-only": true,
	}
	if !validFormats[terraformHCLOpts.outputFormat] {
		return fmt.Errorf("invalid output format '%s', must be one of: detailed, summary, security-only",
			terraformHCLOpts.outputFormat)
	}

	// Validate control codes if provided
	if len(terraformHCLOpts.controlMapping) > 0 {
		validControls := map[string]bool{
			"CC1.1": true, "CC1.2": true, "CC1.3": true, "CC1.4": true, "CC1.5": true,
			"CC2.1": true, "CC2.2": true, "CC2.3": true,
			"CC3.1": true, "CC3.2": true, "CC3.3": true, "CC3.4": true,
			"CC4.1": true, "CC4.2": true,
			"CC5.1": true, "CC5.2": true, "CC5.3": true,
			"CC6.1": true, "CC6.2": true, "CC6.3": true, "CC6.4": true, "CC6.5": true,
			"CC6.6": true, "CC6.7": true, "CC6.8": true,
			"CC7.1": true, "CC7.2": true, "CC7.3": true, "CC7.4": true, "CC7.5": true,
			"CC8.1": true,
			"SO2":   true,
		}

		for _, control := range terraformHCLOpts.controlMapping {
			if !validControls[control] {
				return fmt.Errorf("invalid control code '%s'", control)
			}
		}
	}

	return nil
}

// terraformHCLAnalyzeCmd provides a quick analysis subcommand
var terraformHCLAnalyzeCmd = &cobra.Command{
	Use:   "analyze [paths...]",
	Short: "Quick Terraform HCL analysis with smart defaults",
	Long: `Quick analysis of Terraform configurations with smart defaults.

This command provides a simplified interface for common analysis tasks:
- Scans specified paths for .tf files
- Performs comprehensive security and HA analysis
- Outputs summary in markdown format
- Maps findings to common SOC2 controls

Example:
  grctool tool terraform-hcl-parser analyze ./terraform/
  grctool tool terraform-hcl-parser analyze ./infra/ ./modules/
  grctool tool terraform-hcl-parser analyze "**/*.tf"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTerraformHCLAnalyze,
}

func init() {
	terraformHCLCmd.AddCommand(terraformHCLAnalyzeCmd)

	// Analysis specific options
	terraformHCLAnalyzeCmd.Flags().BoolVar(&terraformHCLOpts.analyzeSecurity, "security", true,
		"Enable security analysis")
	terraformHCLAnalyzeCmd.Flags().BoolVar(&terraformHCLOpts.analyzeHA, "high-availability", true,
		"Enable high availability analysis")
	terraformHCLAnalyzeCmd.Flags().StringVar(&terraformHCLOpts.outputFormat, "format", "summary",
		"Output format: detailed, summary, security-only")
}

// runTerraformHCLAnalyze executes quick analysis
func runTerraformHCLAnalyze(cmd *cobra.Command, args []string) error {
	// Use provided paths as scan paths
	terraformHCLOpts.scanPaths = args
	terraformHCLOpts.includeModules = true
	terraformHCLOpts.includeDiags = false

	// Set common SOC2 controls for analysis
	terraformHCLOpts.controlMapping = []string{
		"CC6.1", "CC6.3", "CC6.6", "CC6.7", "CC6.8",
		"CC7.1", "CC7.2", "CC7.4", "SO2",
	}

	return runTerraformHCL(cmd, args)
}

// terraformHCLSecurityCmd focuses on security analysis
var terraformHCLSecurityCmd = &cobra.Command{
	Use:   "security [paths...]",
	Short: "Security-focused analysis of Terraform configurations",
	Long: `Performs comprehensive security analysis of Terraform configurations.

This command focuses specifically on security aspects:
- Identifies misconfigurations and security risks
- Maps findings to SOC2 security controls
- Analyzes encryption, access controls, and network security
- Outputs security-only report with remediation guidance

Example:
  grctool tool terraform-hcl-parser security ./terraform/
  grctool tool terraform-hcl-parser security --controls CC6.8,CC7.1 ./infra/`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTerraformHCLSecurity,
}

func init() {
	terraformHCLCmd.AddCommand(terraformHCLSecurityCmd)

	terraformHCLSecurityCmd.Flags().StringSliceVar(&terraformHCLOpts.controlMapping, "controls", nil,
		"Specific security controls to focus on (e.g., CC6.8,CC7.1)")
}

// runTerraformHCLSecurity executes security-focused analysis
func runTerraformHCLSecurity(cmd *cobra.Command, args []string) error {
	// Configure for security analysis
	terraformHCLOpts.scanPaths = args
	terraformHCLOpts.includeModules = true
	terraformHCLOpts.analyzeSecurity = true
	terraformHCLOpts.analyzeHA = false
	terraformHCLOpts.outputFormat = "security-only"
	terraformHCLOpts.includeDiags = false

	// Use security-focused controls if not specified
	if len(terraformHCLOpts.controlMapping) == 0 {
		terraformHCLOpts.controlMapping = []string{
			"CC6.1", "CC6.2", "CC6.3", "CC6.6", "CC6.7", "CC6.8",
			"CC7.1", "CC7.2", "CC7.4",
		}
	}

	return runTerraformHCL(cmd, args)
}

// terraformHCLTopologyCmd analyzes infrastructure topology
var terraformHCLTopologyCmd = &cobra.Command{
	Use:   "topology [paths...]",
	Short: "Analyze infrastructure topology and high availability patterns",
	Long: `Analyzes infrastructure topology with focus on high availability and multi-AZ patterns.

This command provides detailed infrastructure topology analysis:
- Maps resource relationships and dependencies
- Identifies multi-AZ configurations
- Analyzes high availability patterns
- Reports single points of failure
- Evaluates load balancing and auto-scaling setups

Example:
  grctool tool terraform-hcl-parser topology ./terraform/
  grctool tool terraform-hcl-parser topology --format detailed ./infra/`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTerraformHCLTopology,
}

func init() {
	terraformHCLCmd.AddCommand(terraformHCLTopologyCmd)

	terraformHCLTopologyCmd.Flags().StringVar(&terraformHCLOpts.outputFormat, "format", "summary",
		"Output format for topology analysis")
}

// runTerraformHCLTopology executes topology analysis
func runTerraformHCLTopology(cmd *cobra.Command, args []string) error {
	// Configure for topology/HA analysis
	terraformHCLOpts.scanPaths = args
	terraformHCLOpts.includeModules = true
	terraformHCLOpts.analyzeSecurity = false
	terraformHCLOpts.analyzeHA = true
	terraformHCLOpts.includeDiags = false

	// Focus on availability controls
	terraformHCLOpts.controlMapping = []string{"SO2"}

	return runTerraformHCL(cmd, args)
}

// terraformHCLValidateCmd validates Terraform HCL syntax
var terraformHCLValidateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate Terraform HCL syntax and report parsing issues",
	Long: `Validates Terraform HCL syntax and reports parsing diagnostics.

This command focuses on HCL syntax validation:
- Parses all .tf files for syntax errors
- Reports detailed diagnostic information
- Identifies problematic expressions and blocks
- Provides line-by-line error details

Example:
  grctool tool terraform-hcl-parser validate ./terraform/
  grctool tool terraform-hcl-parser validate --include-warnings ./modules/`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTerraformHCLValidate,
}

func init() {
	terraformHCLCmd.AddCommand(terraformHCLValidateCmd)

	terraformHCLValidateCmd.Flags().BoolVar(&terraformHCLOpts.includeDiags, "include-warnings", false,
		"Include warnings in diagnostic output")
}

// runTerraformHCLValidate executes syntax validation
func runTerraformHCLValidate(cmd *cobra.Command, args []string) error {
	// Configure for validation only
	terraformHCLOpts.scanPaths = args
	terraformHCLOpts.includeModules = false
	terraformHCLOpts.analyzeSecurity = false
	terraformHCLOpts.analyzeHA = false
	terraformHCLOpts.outputFormat = "detailed"
	terraformHCLOpts.controlMapping = nil

	return runTerraformHCL(cmd, args)
}

// Helper function to pretty print JSON output
func prettyPrintJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}
