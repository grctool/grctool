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

package terraform

import (
	"time"

	"github.com/grctool/grctool/internal/models"
)

// Common types shared across terraform tools

// EnhancedTerraformResult extends TerraformScanResult with additional context
type EnhancedTerraformResult struct {
	models.TerraformScanResult
	BoundedSnippet string `json:"bounded_snippet"`
	ContextLines   int    `json:"context_lines"`
}

// SecurityAnalysisResult contains the results of comprehensive security analysis
type SecurityAnalysisResult struct {
	AnalysisTimestamp   time.Time                     `json:"analysis_timestamp"`
	SecurityDomain      string                        `json:"security_domain"`
	RequestedControls   []string                      `json:"requested_controls"`
	RequestedTasks      []string                      `json:"requested_tasks"`
	SecurityResources   []SecurityResource            `json:"security_resources"`
	EncryptionConfigs   []EncryptionConfig            `json:"encryption_configs"`
	IAMConfigs          []IAMConfig                   `json:"iam_configs"`
	NetworkConfigs      []NetworkConfig               `json:"network_configs"`
	BackupConfigs       []BackupConfig                `json:"backup_configs"`
	MonitoringConfigs   []MonitoringConfig            `json:"monitoring_configs"`
	FilesAnalyzed       []string                      `json:"files_analyzed"`
	SOC2ControlMapping  map[string][]SecurityResource `json:"soc2_control_mapping"`
	EvidenceTaskMapping map[string][]SecurityResource `json:"evidence_task_mapping"`
	ComplianceGaps      []ComplianceGap               `json:"compliance_gaps"`
}

// SecurityResource represents a generic security resource configuration
type SecurityResource struct {
	ResourceType      string                 `json:"resource_type"`
	ResourceName      string                 `json:"resource_name"`
	FilePath          string                 `json:"file_path"`
	LineRange         string                 `json:"line_range"`
	SecurityRelevance []string               `json:"security_relevance"`
	Configuration     map[string]interface{} `json:"configuration"`
	SecurityFindings  []SecurityFinding      `json:"security_findings"`
}

// SecurityFinding represents a security issue or observation
type SecurityFinding struct {
	Type           string   `json:"type"`
	Severity       string   `json:"severity"`
	Description    string   `json:"description"`
	Recommendation string   `json:"recommendation"`
	SOC2Controls   []string `json:"soc2_controls"`
}

// EncryptionConfig represents encryption-specific configuration
type EncryptionConfig struct {
	ResourceType        string `json:"resource_type"`
	ResourceName        string `json:"resource_name"`
	FilePath            string `json:"file_path"`
	LineRange           string `json:"line_range"`
	EncryptionEnabled   bool   `json:"encryption_enabled"`
	EncryptionType      string `json:"encryption_type"`
	KMSKeyID            string `json:"kms_key_id"`
	EncryptionInTransit bool   `json:"encryption_in_transit"`
}

// IAMConfig represents IAM-specific configuration
type IAMConfig struct {
	ResourceType      string   `json:"resource_type"`
	ResourceName      string   `json:"resource_name"`
	FilePath          string   `json:"file_path"`
	LineRange         string   `json:"line_range"`
	AssumeRolePolicy  string   `json:"assume_role_policy"`
	Policies          []string `json:"policies"`
	ManagedPolicyARNs string   `json:"managed_policy_arns"`
	Permissions       []string `json:"permissions"`
}

// NetworkConfig represents network-specific configuration
type NetworkConfig struct {
	ResourceType      string        `json:"resource_type"`
	ResourceName      string        `json:"resource_name"`
	FilePath          string        `json:"file_path"`
	LineRange         string        `json:"line_range"`
	VPCID             string        `json:"vpc_id"`
	IngressRules      []NetworkRule `json:"ingress_rules"`
	EgressRules       []NetworkRule `json:"egress_rules"`
	SecurityGroups    []string      `json:"security_groups"`
	AvailabilityZones []string      `json:"availability_zones"`
}

// NetworkRule represents a network access rule
type NetworkRule struct {
	Protocol string   `json:"protocol"`
	FromPort string   `json:"from_port"`
	ToPort   string   `json:"to_port"`
	CIDR     []string `json:"cidr_blocks"`
}

// BackupConfig represents backup-specific configuration
type BackupConfig struct {
	ResourceType      string `json:"resource_type"`
	ResourceName      string `json:"resource_name"`
	FilePath          string `json:"file_path"`
	LineRange         string `json:"line_range"`
	BackupEnabled     bool   `json:"backup_enabled"`
	RetentionPeriod   string `json:"retention_period"`
	VersioningEnabled bool   `json:"versioning_enabled"`
}

// MonitoringConfig represents monitoring-specific configuration
type MonitoringConfig struct {
	ResourceType      string `json:"resource_type"`
	ResourceName      string `json:"resource_name"`
	FilePath          string `json:"file_path"`
	LineRange         string `json:"line_range"`
	LoggingEnabled    bool   `json:"logging_enabled"`
	MonitoringEnabled bool   `json:"monitoring_enabled"`
	AlertsConfigured  bool   `json:"alerts_configured"`
	RetentionPeriod   string `json:"retention_period"`
	EventTypes        string `json:"event_types"`
}

// ComplianceGap represents a potential compliance gap
type ComplianceGap struct {
	Type            string   `json:"type"`
	Severity        string   `json:"severity"`
	Description     string   `json:"description"`
	SOC2Controls    []string `json:"soc2_controls"`
	EvidenceTasks   []string `json:"evidence_tasks"`
	Recommendations []string `json:"recommendations"`
}

// TerraformSnippet represents a suggested Terraform configuration snippet
type TerraformSnippet struct {
	ResourceType     string   `json:"resource_type"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	ControlCodes     []string `json:"control_codes"`
	Configuration    string   `json:"configuration"`
	ExamplePath      string   `json:"example_path"`
	SecurityFeatures []string `json:"security_features"`
}

// AnalysisStrategy defines the interface for different analysis strategies
type AnalysisStrategy interface {
	Analyze(results []models.TerraformScanResult) ([]models.TerraformScanResult, error)
}

// BaseAnalysisStrategy provides common analysis functionality
type BaseAnalysisStrategy struct {
	logger interface{} // Will be logger.Logger when used
}

// EnhancedAnalysisStrategy provides enhanced analysis with caching and filtering
type EnhancedAnalysisStrategy struct {
	BaseAnalysisStrategy
	cacheDir     string
	cacheStorage interface{} // Will be *storage.Storage when used
}

// HCLAnalysisStrategy provides comprehensive HCL parsing and analysis
type HCLAnalysisStrategy struct {
	BaseAnalysisStrategy
	parser interface{} // Will be *hclparse.Parser when used
}

// SecurityAnalysisStrategy provides focused security analysis
type SecurityAnalysisStrategy struct {
	BaseAnalysisStrategy
	securityDomain string
	controlCodes   []string
}

// Atmos-specific types for stack configuration analysis

// AtmosStackAnalysis contains the results of Atmos stack analysis
type AtmosStackAnalysis struct {
	AnalysisTimestamp  time.Time                        `json:"analysis_timestamp"`
	Environments       []string                         `json:"environments"`
	Stacks             []AtmosStack                     `json:"stacks"`
	DriftIssues        []ConfigurationDrift             `json:"drift_issues"`
	SecurityFindings   []StackSecurityFinding           `json:"security_findings"`
	ComplianceMapping  map[string][]AtmosStack          `json:"compliance_mapping"`
	ComplianceCoverage float64                          `json:"compliance_coverage"`
	EnvironmentMatrix  map[string]map[string]AtmosStack `json:"environment_matrix"`
}

// AtmosStack represents a single Atmos stack configuration
type AtmosStack struct {
	StackName          string                       `json:"stack_name"`
	Environment        string                       `json:"environment"`
	FilePath           string                       `json:"file_path"`
	Component          string                       `json:"component"`
	Backend            map[string]interface{}       `json:"backend"`
	Workspace          string                       `json:"workspace"`
	Variables          map[string]interface{}       `json:"variables"`
	TerraformResources []models.TerraformScanResult `json:"terraform_resources"`
	SecurityFindings   []SecurityFinding            `json:"security_findings"`
}

// ConfigurationDrift represents drift between environment configurations
type ConfigurationDrift struct {
	StackName           string `json:"stack_name"`
	BaselineEnvironment string `json:"baseline_environment"`
	CompareEnvironment  string `json:"compare_environment"`
	DriftType           string `json:"drift_type"`
	Description         string `json:"description"`
	Severity            string `json:"severity"`
	Recommendation      string `json:"recommendation"`
}

// StackSecurityFinding represents a security finding specific to a stack
type StackSecurityFinding struct {
	StackName       string          `json:"stack_name"`
	Environment     string          `json:"environment"`
	SecurityFinding SecurityFinding `json:"security_finding"`
}

// Security indexing types for infrastructure resource indexing

// SecurityIndex represents a comprehensive security index of infrastructure resources
type SecurityIndex struct {
	IndexTimestamp     time.Time                           `json:"index_timestamp"`
	QueryType          string                              `json:"query_type"`
	IndexedResources   []IndexedResource                   `json:"indexed_resources"`
	SecurityAttributes map[string]SecurityAttributeDetails `json:"security_attributes"`
	ControlMapping     map[string][]IndexedResource        `json:"control_mapping"`
	FrameworkMapping   map[string][]IndexedResource        `json:"framework_mapping"`
	RiskDistribution   map[string]int                      `json:"risk_distribution"`
	ComplianceCoverage float64                             `json:"compliance_coverage"`
	EnvironmentStats   map[string]EnvironmentStats         `json:"environment_stats"`
}

// IndexedResource represents a resource indexed by security attributes
type IndexedResource struct {
	ResourceID         string                 `json:"resource_id"`
	ResourceType       string                 `json:"resource_type"`
	ResourceName       string                 `json:"resource_name"`
	FilePath           string                 `json:"file_path"`
	LineRange          string                 `json:"line_range"`
	Environment        string                 `json:"environment"`
	SecurityAttributes []string               `json:"security_attributes"`
	ControlRelevance   []string               `json:"control_relevance"`
	RiskLevel          string                 `json:"risk_level"`
	ComplianceStatus   string                 `json:"compliance_status"`
	Configuration      map[string]interface{} `json:"configuration"`
	LastModified       time.Time              `json:"last_modified"`
}

// SecurityAttributeDetails provides details about a security attribute
type SecurityAttributeDetails struct {
	AttributeName string    `json:"attribute_name"`
	ResourceCount int       `json:"resource_count"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
}

// EnvironmentStats provides statistics for a specific environment
type EnvironmentStats struct {
	Environment      string         `json:"environment"`
	ResourceCount    int            `json:"resource_count"`
	RiskDistribution map[string]int `json:"risk_distribution"`
}

// SecurityIndexQuery represents a query for the security index
type SecurityIndexQuery struct {
	ControlCodes         []string `json:"control_codes"`
	SecurityAttributes   []string `json:"security_attributes"`
	ComplianceFrameworks []string `json:"compliance_frameworks"`
	RiskLevels           []string `json:"risk_levels"`
	Environments         []string `json:"environments"`
	ResourceTypes        []string `json:"resource_types"`
	IncludeMetadata      bool     `json:"include_metadata"`
}

// Evidence query types for Claude query interface

// EvidenceQuery represents a query for finding compliance evidence
type EvidenceQuery struct {
	EvidenceType         string   `json:"evidence_type"`
	NaturalLanguageQuery string   `json:"natural_language_query"`
	ControlCodes         []string `json:"control_codes"`
	EvidenceTasks        []string `json:"evidence_tasks"`
	Frameworks           []string `json:"frameworks"`
	Environments         []string `json:"environments"`
	RiskLevel            string   `json:"risk_level"`
	DetailLevel          string   `json:"detail_level"`
	IncludeRemediation   bool     `json:"include_remediation"`
}

// EvidenceQueryResult contains the results of an evidence query
type EvidenceQueryResult struct {
	QueryTimestamp  time.Time          `json:"query_timestamp"`
	EvidenceType    string             `json:"evidence_type"`
	Resources       []EvidenceResource `json:"resources"`
	Gaps            []EvidenceGap      `json:"gaps"`
	Recommendations []string           `json:"recommendations"`
	ConfidenceScore float64            `json:"confidence_score"`
	Summary         string             `json:"summary"`
}

// EvidenceResource represents a resource that provides compliance evidence
type EvidenceResource struct {
	ResourceID       string                 `json:"resource_id"`
	ResourceType     string                 `json:"resource_type"`
	ResourceName     string                 `json:"resource_name"`
	FilePath         string                 `json:"file_path"`
	LineRange        string                 `json:"line_range"`
	Environment      string                 `json:"environment"`
	ControlRelevance []string               `json:"control_relevance"`
	SecurityConfig   map[string]interface{} `json:"security_config"`
	RiskLevel        string                 `json:"risk_level"`
	ComplianceStatus string                 `json:"compliance_status"`
	EvidenceQuality  string                 `json:"evidence_quality"`
}

// EvidenceGap represents a gap in compliance evidence
type EvidenceGap struct {
	ControlCode    string `json:"control_code"`
	GapType        string `json:"gap_type"`
	Description    string `json:"description"`
	Severity       string `json:"severity"`
	Recommendation string `json:"recommendation"`
}

// QueryContext represents extracted context from natural language queries
type QueryContext struct {
	ExtractedControls     []string `json:"extracted_controls"`
	ExtractedFrameworks   []string `json:"extracted_frameworks"`
	ExtractedAttributes   []string `json:"extracted_attributes"`
	ExtractedEnvironments []string `json:"extracted_environments"`
	Intent                string   `json:"intent"`
}
