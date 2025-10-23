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

package models

import (
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// TerraformModule represents a complete Terraform module or configuration
type TerraformModule struct {
	FilePath          string                 `json:"file_path"`
	Resources         []TerraformResource    `json:"resources"`
	DataSources       []TerraformDataSource  `json:"data_sources"`
	Variables         []TerraformVariable    `json:"variables"`
	Outputs           []TerraformOutput      `json:"outputs"`
	Locals            []TerraformLocal       `json:"locals"`
	ModuleCalls       []TerraformModuleCall  `json:"module_calls"`
	ProviderConfigs   []TerraformProvider    `json:"provider_configs"`
	ParsedAt          time.Time              `json:"parsed_at"`
	HCLDiagnostics    []HCLDiagnostic        `json:"hcl_diagnostics,omitempty"`
	SecurityRelevance []string               `json:"security_relevance"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// TerraformResource represents a Terraform resource block
type TerraformResource struct {
	Type              string                  `json:"type"`
	Name              string                  `json:"name"`
	Config            map[string]interface{}  `json:"config"`
	DependsOn         []string                `json:"depends_on,omitempty"`
	Count             *TerraformExpression    `json:"count,omitempty"`
	ForEach           *TerraformExpression    `json:"for_each,omitempty"`
	Provider          string                  `json:"provider,omitempty"`
	Lifecycle         *TerraformLifecycle     `json:"lifecycle,omitempty"`
	Provisioners      []TerraformProvisioner  `json:"provisioners,omitempty"`
	Tags              map[string]interface{}  `json:"tags,omitempty"`
	FilePath          string                  `json:"file_path"`
	LineRange         *HCLRange               `json:"line_range"`
	SecurityRelevance []string                `json:"security_relevance"`
	MultiAZConfig     *MultiAZConfiguration   `json:"multi_az_config,omitempty"`
	HAConfig          *HighAvailabilityConfig `json:"ha_config,omitempty"`
}

// TerraformDataSource represents a Terraform data source block
type TerraformDataSource struct {
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Config    map[string]interface{} `json:"config"`
	DependsOn []string               `json:"depends_on,omitempty"`
	Count     *TerraformExpression   `json:"count,omitempty"`
	ForEach   *TerraformExpression   `json:"for_each,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	FilePath  string                 `json:"file_path"`
	LineRange *HCLRange              `json:"line_range"`
}

// TerraformVariable represents a Terraform variable declaration
type TerraformVariable struct {
	Name        string                `json:"name"`
	Type        string                `json:"type,omitempty"`
	Description string                `json:"description,omitempty"`
	Default     interface{}           `json:"default,omitempty"`
	Validation  []TerraformValidation `json:"validation,omitempty"`
	Sensitive   bool                  `json:"sensitive,omitempty"`
	Nullable    bool                  `json:"nullable,omitempty"`
	FilePath    string                `json:"file_path"`
	LineRange   *HCLRange             `json:"line_range"`
}

// TerraformOutput represents a Terraform output declaration
type TerraformOutput struct {
	Name         string               `json:"name"`
	Value        *TerraformExpression `json:"value"`
	Description  string               `json:"description,omitempty"`
	Sensitive    bool                 `json:"sensitive,omitempty"`
	DependsOn    []string             `json:"depends_on,omitempty"`
	Precondition []TerraformCondition `json:"precondition,omitempty"`
	FilePath     string               `json:"file_path"`
	LineRange    *HCLRange            `json:"line_range"`
}

// TerraformLocal represents a Terraform locals block
type TerraformLocal struct {
	Name      string               `json:"name"`
	Value     *TerraformExpression `json:"value"`
	FilePath  string               `json:"file_path"`
	LineRange *HCLRange            `json:"line_range"`
}

// TerraformModuleCall represents a Terraform module call
type TerraformModuleCall struct {
	Name      string                 `json:"name"`
	Source    string                 `json:"source"`
	Version   string                 `json:"version,omitempty"`
	Config    map[string]interface{} `json:"config"`
	Count     *TerraformExpression   `json:"count,omitempty"`
	ForEach   *TerraformExpression   `json:"for_each,omitempty"`
	Providers map[string]string      `json:"providers,omitempty"`
	DependsOn []string               `json:"depends_on,omitempty"`
	FilePath  string                 `json:"file_path"`
	LineRange *HCLRange              `json:"line_range"`
}

// TerraformProvider represents a Terraform provider configuration
type TerraformProvider struct {
	Name      string                 `json:"name"`
	Alias     string                 `json:"alias,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Config    map[string]interface{} `json:"config"`
	FilePath  string                 `json:"file_path"`
	LineRange *HCLRange              `json:"line_range"`
}

// TerraformLifecycle represents a Terraform lifecycle configuration
type TerraformLifecycle struct {
	CreateBeforeDestroy bool                 `json:"create_before_destroy,omitempty"`
	PreventDestroy      bool                 `json:"prevent_destroy,omitempty"`
	IgnoreChanges       []string             `json:"ignore_changes,omitempty"`
	ReplaceTriggeredBy  []string             `json:"replace_triggered_by,omitempty"`
	Precondition        []TerraformCondition `json:"precondition,omitempty"`
	Postcondition       []TerraformCondition `json:"postcondition,omitempty"`
}

// TerraformProvisioner represents a Terraform provisioner
type TerraformProvisioner struct {
	Type       string                 `json:"type"`
	Config     map[string]interface{} `json:"config"`
	When       string                 `json:"when,omitempty"`
	OnFailure  string                 `json:"on_failure,omitempty"`
	Connection *TerraformConnection   `json:"connection,omitempty"`
}

// TerraformConnection represents a Terraform connection block
type TerraformConnection struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// TerraformValidation represents a Terraform variable validation
type TerraformValidation struct {
	Condition    *TerraformExpression `json:"condition"`
	ErrorMessage string               `json:"error_message"`
}

// TerraformCondition represents a Terraform condition (precondition/postcondition)
type TerraformCondition struct {
	Condition    *TerraformExpression `json:"condition"`
	ErrorMessage string               `json:"error_message"`
}

// TerraformExpression represents a Terraform expression with metadata
type TerraformExpression struct {
	Raw          string               `json:"raw"`
	Value        interface{}          `json:"value,omitempty"`
	Variables    []string             `json:"variables,omitempty"`
	Functions    []string             `json:"functions,omitempty"`
	References   []TerraformReference `json:"references,omitempty"`
	IsConstant   bool                 `json:"is_constant"`
	HasSensitive bool                 `json:"has_sensitive"`
	CTYType      string               `json:"cty_type,omitempty"`
}

// TerraformReference represents a reference to another resource/data/variable
type TerraformReference struct {
	Type      string `json:"type"` // "resource", "data", "var", "local", "module"
	Name      string `json:"name"`
	Attribute string `json:"attribute,omitempty"`
	Key       string `json:"key,omitempty"`
}

// MultiAZConfiguration represents multi-availability zone configuration
type MultiAZConfiguration struct {
	IsMultiAZ           bool                `json:"is_multi_az"`
	AvailabilityZones   []string            `json:"availability_zones,omitempty"`
	AZReferences        []string            `json:"az_references,omitempty"`
	SubnetConfiguration *SubnetConfig       `json:"subnet_configuration,omitempty"`
	LoadBalancerConfig  *LoadBalancerConfig `json:"load_balancer_config,omitempty"`
	AutoScalingConfig   *AutoScalingConfig  `json:"auto_scaling_config,omitempty"`
}

// HighAvailabilityConfig represents high availability configuration
type HighAvailabilityConfig struct {
	HasFailover        bool `json:"has_failover"`
	HasLoadBalancing   bool `json:"has_load_balancing"`
	HasAutoScaling     bool `json:"has_auto_scaling"`
	HasBackup          bool `json:"has_backup"`
	HasMonitoring      bool `json:"has_monitoring"`
	ReplicationEnabled bool `json:"replication_enabled"`
	ClusteringEnabled  bool `json:"clustering_enabled"`
	DisasterRecovery   bool `json:"disaster_recovery"`
}

// SubnetConfig represents subnet configuration for multi-AZ
type SubnetConfig struct {
	PrivateSubnets    []string `json:"private_subnets,omitempty"`
	PublicSubnets     []string `json:"public_subnets,omitempty"`
	DatabaseSubnets   []string `json:"database_subnets,omitempty"`
	IsSpreadAcrossAZs bool     `json:"is_spread_across_azs"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Type              string             `json:"type"`   // "application", "network", "classic"
	Scheme            string             `json:"scheme"` // "internet-facing", "internal"
	CrossZoneEnabled  bool               `json:"cross_zone_enabled"`
	TargetGroups      []string           `json:"target_groups,omitempty"`
	HealthCheckConfig *HealthCheckConfig `json:"health_check_config,omitempty"`
}

// AutoScalingConfig represents auto scaling configuration
type AutoScalingConfig struct {
	MinSize                int      `json:"min_size"`
	MaxSize                int      `json:"max_size"`
	DesiredCapacity        int      `json:"desired_capacity"`
	HealthCheckType        string   `json:"health_check_type,omitempty"`
	HealthCheckGracePeriod int      `json:"health_check_grace_period,omitempty"`
	AvailabilityZones      []string `json:"availability_zones,omitempty"`
	TargetGroupARNs        []string `json:"target_group_arns,omitempty"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled            bool   `json:"enabled"`
	HealthyThreshold   int    `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold int    `json:"unhealthy_threshold,omitempty"`
	Timeout            int    `json:"timeout,omitempty"`
	Interval           int    `json:"interval,omitempty"`
	Path               string `json:"path,omitempty"`
	Protocol           string `json:"protocol,omitempty"`
	Port               string `json:"port,omitempty"`
	Matcher            string `json:"matcher,omitempty"`
}

// HCLRange represents a range in HCL source code
type HCLRange struct {
	Filename string `json:"filename"`
	Start    HCLPos `json:"start"`
	End      HCLPos `json:"end"`
}

// HCLPos represents a position in HCL source code
type HCLPos struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Byte   int `json:"byte"`
}

// HCLDiagnostic represents an HCL parsing diagnostic/error
type HCLDiagnostic struct {
	Severity string    `json:"severity"`
	Summary  string    `json:"summary"`
	Detail   string    `json:"detail,omitempty"`
	Subject  *HCLRange `json:"subject,omitempty"`
	Context  *HCLRange `json:"context,omitempty"`
}

// TerraformParseResult represents the result of parsing Terraform files
type TerraformParseResult struct {
	Modules           []TerraformModule       `json:"modules"`
	Dependencies      []ResourceDependency    `json:"dependencies"`
	SecurityFindings  []SecurityFinding       `json:"security_findings"`
	InfrastructureMap *InfrastructureTopology `json:"infrastructure_map"`
	ParseSummary      *ParseSummary           `json:"parse_summary"`
	ParsedAt          time.Time               `json:"parsed_at"`
	ToolVersion       string                  `json:"tool_version"`
}

// ResourceDependency represents dependencies between resources
type ResourceDependency struct {
	From           ResourceIdentifier `json:"from"`
	To             ResourceIdentifier `json:"to"`
	DependencyType string             `json:"dependency_type"` // "explicit", "implicit", "data"
	Attribute      string             `json:"attribute,omitempty"`
}

// ResourceIdentifier uniquely identifies a Terraform resource
type ResourceIdentifier struct {
	Type     string `json:"type"` // "resource", "data", "module"
	Kind     string `json:"kind"` // e.g., "aws_instance", "aws_vpc"
	Name     string `json:"name"`
	Module   string `json:"module,omitempty"`
	FilePath string `json:"file_path"`
}

// SecurityFinding represents a security-relevant finding
type SecurityFinding struct {
	Resource       ResourceIdentifier     `json:"resource"`
	FindingType    string                 `json:"finding_type"` // "encryption", "access_control", "network", etc.
	Severity       string                 `json:"severity"`     // "high", "medium", "low", "info"
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Recommendation string                 `json:"recommendation,omitempty"`
	ControlCodes   []string               `json:"control_codes"` // SOC2 control codes
	CWEReference   string                 `json:"cwe_reference,omitempty"`
	CVSSScore      float64                `json:"cvss_score,omitempty"`
	Evidence       map[string]interface{} `json:"evidence"`
}

// InfrastructureTopology represents the overall infrastructure layout
type InfrastructureTopology struct {
	Regions           []RegionTopology    `json:"regions"`
	Networks          []NetworkTopology   `json:"networks"`
	SecurityGroups    []SecurityGroupInfo `json:"security_groups"`
	LoadBalancers     []LoadBalancerInfo  `json:"load_balancers"`
	Databases         []DatabaseInfo      `json:"databases"`
	ComputeInstances  []ComputeInfo       `json:"compute_instances"`
	StorageSystems    []StorageInfo       `json:"storage_systems"`
	IAMResources      []IAMResourceInfo   `json:"iam_resources"`
	EncryptionSummary *EncryptionSummary  `json:"encryption_summary"`
	HAAnalysis        *HAAnalysis         `json:"ha_analysis"`
}

// RegionTopology represents the topology of a specific region
type RegionTopology struct {
	Region            string               `json:"region"`
	AvailabilityZones []string             `json:"availability_zones"`
	VPCs              []string             `json:"vpcs"`
	ResourceCounts    map[string]int       `json:"resource_counts"`
	MultiAZResources  []ResourceIdentifier `json:"multi_az_resources"`
}

// NetworkTopology represents network configuration
type NetworkTopology struct {
	VPCId            string           `json:"vpc_id"`
	CIDR             string           `json:"cidr,omitempty"`
	Subnets          []SubnetInfo     `json:"subnets"`
	InternetGateways []string         `json:"internet_gateways"`
	NATGateways      []string         `json:"nat_gateways"`
	RouteTables      []RouteTableInfo `json:"route_tables"`
	NetworkACLs      []string         `json:"network_acls"`
	VPCPeering       []string         `json:"vpc_peering,omitempty"`
	VPNConnections   []string         `json:"vpn_connections,omitempty"`
}

// SubnetInfo represents subnet information
type SubnetInfo struct {
	Name             string `json:"name"`
	CIDR             string `json:"cidr,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
	Type             string `json:"type"` // "public", "private", "database"
	IsPublic         bool   `json:"is_public"`
}

// RouteTableInfo represents route table information
type RouteTableInfo struct {
	Name    string   `json:"name"`
	Routes  []string `json:"routes,omitempty"`
	Subnets []string `json:"subnets,omitempty"`
}

// SecurityGroupInfo represents security group information
type SecurityGroupInfo struct {
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	VPC          string               `json:"vpc,omitempty"`
	IngressRules []SecurityRule       `json:"ingress_rules,omitempty"`
	EgressRules  []SecurityRule       `json:"egress_rules,omitempty"`
	References   []ResourceIdentifier `json:"references,omitempty"`
}

// SecurityRule represents a security group rule
type SecurityRule struct {
	Protocol    string   `json:"protocol"`
	FromPort    string   `json:"from_port,omitempty"`
	ToPort      string   `json:"to_port,omitempty"`
	CIDRBlocks  []string `json:"cidr_blocks,omitempty"`
	SourceSG    string   `json:"source_sg,omitempty"`
	Description string   `json:"description,omitempty"`
}

// LoadBalancerInfo represents load balancer information
type LoadBalancerInfo struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Scheme           string            `json:"scheme"`
	Subnets          []string          `json:"subnets,omitempty"`
	SecurityGroups   []string          `json:"security_groups,omitempty"`
	TargetGroups     []TargetGroupInfo `json:"target_groups,omitempty"`
	IsMultiAZ        bool              `json:"is_multi_az"`
	CrossZoneEnabled bool              `json:"cross_zone_enabled"`
}

// TargetGroupInfo represents target group information
type TargetGroupInfo struct {
	Name        string             `json:"name"`
	Port        int                `json:"port,omitempty"`
	Protocol    string             `json:"protocol,omitempty"`
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty"`
	TargetType  string             `json:"target_type,omitempty"`
}

// DatabaseInfo represents database information
type DatabaseInfo struct {
	Name              string   `json:"name"`
	Engine            string   `json:"engine,omitempty"`
	EngineVersion     string   `json:"engine_version,omitempty"`
	MultiAZ           bool     `json:"multi_az"`
	BackupRetention   int      `json:"backup_retention,omitempty"`
	EncryptionEnabled bool     `json:"encryption_enabled"`
	SubnetGroup       string   `json:"subnet_group,omitempty"`
	SecurityGroups    []string `json:"security_groups,omitempty"`
	MonitoringEnabled bool     `json:"monitoring_enabled"`
}

// ComputeInfo represents compute instance information
type ComputeInfo struct {
	Name               string   `json:"name"`
	InstanceType       string   `json:"instance_type,omitempty"`
	AMI                string   `json:"ami,omitempty"`
	Subnet             string   `json:"subnet,omitempty"`
	SecurityGroups     []string `json:"security_groups,omitempty"`
	IAMInstanceProfile string   `json:"iam_instance_profile,omitempty"`
	UserData           bool     `json:"has_user_data"`
	EBSOptimized       bool     `json:"ebs_optimized"`
	MonitoringEnabled  bool     `json:"monitoring_enabled"`
}

// StorageInfo represents storage information
type StorageInfo struct {
	Name              string `json:"name"`
	Type              string `json:"type"` // "s3", "ebs", "efs", etc.
	EncryptionEnabled bool   `json:"encryption_enabled"`
	EncryptionKey     string `json:"encryption_key,omitempty"`
	BackupEnabled     bool   `json:"backup_enabled"`
	Versioning        bool   `json:"versioning,omitempty"`
	AccessLogging     bool   `json:"access_logging"`
}

// IAMResourceInfo represents IAM resource information
type IAMResourceInfo struct {
	Name        string               `json:"name"`
	Type        string               `json:"type"` // "role", "policy", "user", "group"
	Policies    []string             `json:"policies,omitempty"`
	AssumeRole  *AssumeRoleInfo      `json:"assume_role,omitempty"`
	Permissions []string             `json:"permissions,omitempty"`
	IsService   bool                 `json:"is_service"`
	References  []ResourceIdentifier `json:"references,omitempty"`
}

// AssumeRoleInfo represents assume role policy information
type AssumeRoleInfo struct {
	Principals []string               `json:"principals,omitempty"`
	Actions    []string               `json:"actions,omitempty"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// EncryptionSummary represents overall encryption status
type EncryptionSummary struct {
	AtRestEncryption    *EncryptionStatus  `json:"at_rest_encryption"`
	InTransitEncryption *EncryptionStatus  `json:"in_transit_encryption"`
	KeyManagement       *KeyManagementInfo `json:"key_management"`
}

// EncryptionStatus represents encryption status
type EncryptionStatus struct {
	Enabled      bool     `json:"enabled"`
	Coverage     float64  `json:"coverage_percentage"`
	Resources    []string `json:"encrypted_resources,omitempty"`
	NonCompliant []string `json:"non_compliant_resources,omitempty"`
}

// KeyManagementInfo represents key management information
type KeyManagementInfo struct {
	KMSKeys         []string `json:"kms_keys,omitempty"`
	CustomerManaged bool     `json:"customer_managed"`
	KeyRotation     bool     `json:"key_rotation_enabled"`
	CrossRegionKeys bool     `json:"cross_region_keys"`
}

// HAAnalysis represents high availability analysis
type HAAnalysis struct {
	MultiAZCoverage       float64              `json:"multi_az_coverage_percentage"`
	LoadBalancedServices  int                  `json:"load_balanced_services"`
	AutoScalingGroups     int                  `json:"auto_scaling_groups"`
	DatabaseReplication   int                  `json:"database_replication_count"`
	BackupSystems         int                  `json:"backup_systems_count"`
	SinglePointsOfFailure []ResourceIdentifier `json:"single_points_of_failure"`
}

// ParseSummary represents summary statistics from parsing
type ParseSummary struct {
	FilesProcessed   int            `json:"files_processed"`
	TotalResources   int            `json:"total_resources"`
	ResourceCounts   map[string]int `json:"resource_counts_by_type"`
	ModuleCounts     int            `json:"module_counts"`
	ErrorCount       int            `json:"error_count"`
	WarningCount     int            `json:"warning_count"`
	ParseDuration    time.Duration  `json:"parse_duration"`
	SecurityFindings int            `json:"security_findings_count"`
	HAFindings       map[string]int `json:"ha_findings"`
}

// ConvertHCLRange converts an hcl.Range to our HCLRange model
func ConvertHCLRange(hclRange hcl.Range) *HCLRange {
	return &HCLRange{
		Filename: hclRange.Filename,
		Start: HCLPos{
			Line:   hclRange.Start.Line,
			Column: hclRange.Start.Column,
			Byte:   hclRange.Start.Byte,
		},
		End: HCLPos{
			Line:   hclRange.End.Line,
			Column: hclRange.End.Column,
			Byte:   hclRange.End.Byte,
		},
	}
}

// ConvertHCLDiagnostics converts HCL diagnostics to our model
func ConvertHCLDiagnostics(diagnostics hcl.Diagnostics) []HCLDiagnostic {
	var result []HCLDiagnostic
	for _, diag := range diagnostics {
		hclDiag := HCLDiagnostic{
			Summary: diag.Summary,
			Detail:  diag.Detail,
		}

		switch diag.Severity {
		case hcl.DiagError:
			hclDiag.Severity = "error"
		case hcl.DiagWarning:
			hclDiag.Severity = "warning"
		default:
			hclDiag.Severity = "info"
		}

		if diag.Subject != nil {
			hclDiag.Subject = ConvertHCLRange(*diag.Subject)
		}
		if diag.Context != nil {
			hclDiag.Context = ConvertHCLRange(*diag.Context)
		}

		result = append(result, hclDiag)
	}
	return result
}

// GetCTYTypeString returns a string representation of a cty.Type
func GetCTYTypeString(ctyType cty.Type) string {
	if ctyType == cty.NilType {
		return "unknown"
	}
	return ctyType.FriendlyName()
}
