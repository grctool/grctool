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
	"fmt"

	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// terraformAtmosCmd represents the atmos-stack-analyzer command
var terraformAtmosCmd = &cobra.Command{
	Use:   "atmos-stack-analyzer",
	Short: "Analyzes Atmos stack configurations and multi-environment Terraform deployments for security compliance",
	Long: `The atmos-stack-analyzer tool provides comprehensive analysis of Atmos stack configurations with:

• Multi-environment stack analysis
• Configuration drift detection between environments
• Security compliance validation
• SOC2 and ISO27001 framework mapping
• Cross-environment consistency checking
• Stack dependency analysis

This tool analyzes Atmos stack configurations and identifies:
- Environment-specific configuration variations
- Security misconfigurations across environments
- Compliance gaps and violations
- Configuration drift between dev/staging/prod
- Best practice deviations
- Infrastructure consistency patterns

Output formats: detailed_json, summary_markdown, compliance_csv, drift_report`,
	RunE: runTerraformAtmos,
}

// terraformAtmosFlags defines the command-line flags
type terraformAtmosFlags struct {
	environments         []string
	stackNames           []string
	securityFocus        string
	complianceFrameworks []string
	includeDriftAnalysis bool
	outputFormat         string
}

var terraformAtmosOpts = terraformAtmosFlags{}

func init() {
	toolCmd.AddCommand(terraformAtmosCmd)

	// Environment and stack selection
	terraformAtmosCmd.Flags().StringSliceVar(&terraformAtmosOpts.environments, "environments", nil,
		"Specific environments to analyze (e.g., dev,staging,prod). Leave empty for all environments")
	terraformAtmosCmd.Flags().StringSliceVar(&terraformAtmosOpts.stackNames, "stack-names", nil,
		"Specific stack names to analyze (e.g., vpc,app,db). Leave empty for all stacks")

	// Analysis options
	terraformAtmosCmd.Flags().StringVar(&terraformAtmosOpts.securityFocus, "security-focus", "all",
		"Security domain to focus on: encryption, network, iam, monitoring, backup, all")
	terraformAtmosCmd.Flags().StringSliceVar(&terraformAtmosOpts.complianceFrameworks, "compliance-frameworks", nil,
		"Compliance frameworks to check against (e.g., SOC2,ISO27001,PCI)")
	terraformAtmosCmd.Flags().BoolVar(&terraformAtmosOpts.includeDriftAnalysis, "include-drift-analysis", true,
		"Include configuration drift analysis between environments")

	// Output options
	terraformAtmosCmd.Flags().StringVar(&terraformAtmosOpts.outputFormat, "output-format", "detailed_json",
		"Output format: detailed_json, summary_markdown, compliance_csv, drift_report")
}

// runTerraformAtmos executes the atmos stack analyzer tool
func runTerraformAtmos(cmd *cobra.Command, args []string) error {
	// Validate parameters
	if err := validateTerraformAtmosParams(); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Build tool parameters
	params := map[string]interface{}{
		"environments":           terraformAtmosOpts.environments,
		"stack_names":            terraformAtmosOpts.stackNames,
		"security_focus":         terraformAtmosOpts.securityFocus,
		"compliance_frameworks":  terraformAtmosOpts.complianceFrameworks,
		"include_drift_analysis": terraformAtmosOpts.includeDriftAnalysis,
		"output_format":          terraformAtmosOpts.outputFormat,
	}

	// Execute tool
	validationRules := map[string]tools.ValidationRule{
		"security_focus": {
			Required:      true,
			Type:          "string",
			AllowedValues: []string{"encryption", "network", "iam", "monitoring", "backup", "all"},
		},
		"output_format": {
			Required:      true,
			Type:          "string",
			AllowedValues: []string{"detailed_json", "summary_markdown", "compliance_csv", "drift_report"},
		},
		"include_drift_analysis": BoolRule,
	}

	return ValidateAndExecuteTool(cmd, "atmos-stack-analyzer", params, validationRules)
}

// validateTerraformAtmosParams validates the command parameters
func validateTerraformAtmosParams() error {
	// Validate security focus
	validSecurityFocus := map[string]bool{
		"encryption": true,
		"network":    true,
		"iam":        true,
		"monitoring": true,
		"backup":     true,
		"all":        true,
	}
	if !validSecurityFocus[terraformAtmosOpts.securityFocus] {
		return fmt.Errorf("invalid security focus '%s', must be one of: encryption, network, iam, monitoring, backup, all",
			terraformAtmosOpts.securityFocus)
	}

	// Validate output format
	validFormats := map[string]bool{
		"detailed_json":    true,
		"summary_markdown": true,
		"compliance_csv":   true,
		"drift_report":     true,
	}
	if !validFormats[terraformAtmosOpts.outputFormat] {
		return fmt.Errorf("invalid output format '%s', must be one of: detailed_json, summary_markdown, compliance_csv, drift_report",
			terraformAtmosOpts.outputFormat)
	}

	// Validate compliance frameworks if provided
	if len(terraformAtmosOpts.complianceFrameworks) > 0 {
		validFrameworks := map[string]bool{
			"SOC2":     true,
			"ISO27001": true,
			"PCI":      true,
			"HIPAA":    true,
			"GDPR":     true,
			"NIST":     true,
		}

		for _, framework := range terraformAtmosOpts.complianceFrameworks {
			if !validFrameworks[framework] {
				return fmt.Errorf("invalid compliance framework '%s'", framework)
			}
		}
	}

	return nil
}

// terraformAtmosDriftCmd analyzes configuration drift between environments
var terraformAtmosDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Analyze configuration drift between Atmos environments",
	Long: `Analyzes configuration drift between environments in Atmos stacks.

This command focuses specifically on identifying configuration variations:
- Compares stack configurations across environments
- Identifies unintended configuration differences
- Highlights security-relevant drift
- Reports inconsistent resource configurations
- Provides environment parity analysis

Example:
  grctool tool atmos-stack-analyzer drift
  grctool tool atmos-stack-analyzer drift --environments dev,staging,prod
  grctool tool atmos-stack-analyzer drift --stack-names vpc,app`,
	RunE: runTerraformAtmosDrift,
}

func init() {
	terraformAtmosCmd.AddCommand(terraformAtmosDriftCmd)

	terraformAtmosDriftCmd.Flags().StringSliceVar(&terraformAtmosOpts.environments, "environments", nil,
		"Specific environments to compare for drift")
	terraformAtmosDriftCmd.Flags().StringSliceVar(&terraformAtmosOpts.stackNames, "stack-names", nil,
		"Specific stacks to analyze for drift")
}

// runTerraformAtmosDrift executes drift analysis
func runTerraformAtmosDrift(cmd *cobra.Command, args []string) error {
	// Configure for drift analysis
	terraformAtmosOpts.securityFocus = "all"
	terraformAtmosOpts.includeDriftAnalysis = true
	terraformAtmosOpts.outputFormat = "drift_report"
	terraformAtmosOpts.complianceFrameworks = nil

	return runTerraformAtmos(cmd, args)
}

// terraformAtmosSecurityCmd focuses on security analysis
var terraformAtmosSecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "Security-focused analysis of Atmos stack configurations",
	Long: `Performs comprehensive security analysis of Atmos stack configurations.

This command focuses specifically on security aspects:
- Identifies security misconfigurations across environments
- Maps findings to compliance frameworks
- Analyzes encryption, IAM, network security settings
- Checks for security best practices
- Reports environment-specific security gaps

Example:
  grctool tool atmos-stack-analyzer security
  grctool tool atmos-stack-analyzer security --security-focus encryption
  grctool tool atmos-stack-analyzer security --compliance-frameworks SOC2,ISO27001`,
	RunE: runTerraformAtmosSecurity,
}

func init() {
	terraformAtmosCmd.AddCommand(terraformAtmosSecurityCmd)

	terraformAtmosSecurityCmd.Flags().StringVar(&terraformAtmosOpts.securityFocus, "security-focus", "all",
		"Security domain to focus on")
	terraformAtmosSecurityCmd.Flags().StringSliceVar(&terraformAtmosOpts.complianceFrameworks, "compliance-frameworks", nil,
		"Compliance frameworks to check against")
}

// runTerraformAtmosSecurity executes security-focused analysis
func runTerraformAtmosSecurity(cmd *cobra.Command, args []string) error {
	// Configure for security analysis
	terraformAtmosOpts.includeDriftAnalysis = false
	terraformAtmosOpts.outputFormat = "summary_markdown"

	// Use common compliance frameworks if not specified
	if len(terraformAtmosOpts.complianceFrameworks) == 0 {
		terraformAtmosOpts.complianceFrameworks = []string{"SOC2", "ISO27001"}
	}

	return runTerraformAtmos(cmd, args)
}

// terraformAtmosComplianceCmd focuses on compliance mapping
var terraformAtmosComplianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Analyze Atmos stacks for compliance framework alignment",
	Long: `Analyzes Atmos stack configurations for compliance with specific frameworks.

This command provides compliance-focused analysis:
- Maps stack configurations to compliance controls
- Identifies compliance gaps across environments
- Reports framework-specific violations
- Generates compliance coverage reports
- Provides remediation guidance

Example:
  grctool tool atmos-stack-analyzer compliance
  grctool tool atmos-stack-analyzer compliance --compliance-frameworks SOC2
  grctool tool atmos-stack-analyzer compliance --compliance-frameworks SOC2,ISO27001 --output-format compliance_csv`,
	RunE: runTerraformAtmosCompliance,
}

func init() {
	terraformAtmosCmd.AddCommand(terraformAtmosComplianceCmd)

	terraformAtmosComplianceCmd.Flags().StringSliceVar(&terraformAtmosOpts.complianceFrameworks, "compliance-frameworks", nil,
		"Compliance frameworks to analyze (required)")
	terraformAtmosComplianceCmd.Flags().StringVar(&terraformAtmosOpts.outputFormat, "output-format", "compliance_csv",
		"Output format for compliance report")

	terraformAtmosComplianceCmd.MarkFlagRequired("compliance-frameworks")
}

// runTerraformAtmosCompliance executes compliance analysis
func runTerraformAtmosCompliance(cmd *cobra.Command, args []string) error {
	// Configure for compliance analysis
	terraformAtmosOpts.securityFocus = "all"
	terraformAtmosOpts.includeDriftAnalysis = false

	return runTerraformAtmos(cmd, args)
}
