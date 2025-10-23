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
	"encoding/json"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Test Evidence model JSON marshaling/unmarshaling
func TestEvidence_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	evidence := Evidence{
		ID:          "ev-001",
		TaskID:      "task-001",
		Type:        "terraform",
		Name:        "Test Evidence",
		Description: "Test description",
		Content:     "Test content",
		FilePath:    "/path/to/file",
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Marshal to JSON
	data, err := json.Marshal(evidence)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal from JSON
	var decoded Evidence
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, evidence.ID, decoded.ID)
	assert.Equal(t, evidence.TaskID, decoded.TaskID)
	assert.Equal(t, evidence.Type, decoded.Type)
	assert.Equal(t, evidence.Name, decoded.Name)
	assert.Equal(t, evidence.Description, decoded.Description)
	assert.Equal(t, evidence.Content, decoded.Content)
	assert.Equal(t, evidence.FilePath, decoded.FilePath)
	assert.Equal(t, evidence.Status, decoded.Status)
	assert.Equal(t, evidence.Metadata, decoded.Metadata)
	assert.True(t, evidence.CreatedAt.Equal(decoded.CreatedAt))
	assert.True(t, evidence.UpdatedAt.Equal(decoded.UpdatedAt))
}

// Test Evidence with empty/nil values
func TestEvidence_EmptyValues(t *testing.T) {
	evidence := Evidence{
		ID:       "ev-001",
		TaskID:   "task-001",
		Type:     "terraform",
		Name:     "Test Evidence",
		Metadata: map[string]string{},
		Status:   "draft",
	}

	data, err := json.Marshal(evidence)
	require.NoError(t, err)

	var decoded Evidence
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "", decoded.Description)
	assert.Equal(t, "", decoded.Content)
	assert.Equal(t, "", decoded.FilePath)
	assert.NotNil(t, decoded.Metadata)
	assert.Empty(t, decoded.Metadata)
}

// Test TerraformEvidence model
func TestTerraformEvidence_JSONMarshaling(t *testing.T) {
	tfEvidence := TerraformEvidence{
		EvidenceType: "iam_policy",
		ResourceType: "aws_iam_role",
		ResourceName: "test-role",
		FilePath:     "/path/to/main.tf",
		LineNumbers:  "10-25",
		Content:      "resource \"aws_iam_role\" \"test-role\" { ... }",
		Controls:     []string{"CC6.1", "CC6.2"},
	}

	data, err := json.Marshal(tfEvidence)
	require.NoError(t, err)

	var decoded TerraformEvidence
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, tfEvidence.EvidenceType, decoded.EvidenceType)
	assert.Equal(t, tfEvidence.ResourceType, decoded.ResourceType)
	assert.Equal(t, tfEvidence.ResourceName, decoded.ResourceName)
	assert.Equal(t, tfEvidence.FilePath, decoded.FilePath)
	assert.Equal(t, tfEvidence.LineNumbers, decoded.LineNumbers)
	assert.Equal(t, tfEvidence.Content, decoded.Content)
	assert.Equal(t, tfEvidence.Controls, decoded.Controls)
}

// Test EvidenceContext model
func TestEvidenceContext_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ctx := EvidenceContext{
		Task: EvidenceTaskDetails{
			EvidenceTask: EvidenceTask{
				ID:          123,
				Name:        "Test Task",
				Description: "Test description",
				Created:     now,
			},
		},
		Controls:         []Control{},
		Policies:         []Policy{},
		FrameworkReqs:    []string{"SOC2", "ISO27001"},
		PreviousEvidence: []string{},
		SecurityMappings: SecurityMappings{
			SOC2: map[string]SecurityControlMapping{
				"CC6.1": {
					TerraformResources: []string{"aws_iam_role", "aws_iam_policy"},
					Description:        "Access control",
					Requirements:       []string{"MFA required", "Least privilege"},
				},
			},
		},
		OutputFormat: "csv",
	}

	data, err := json.Marshal(ctx)
	require.NoError(t, err)

	var decoded EvidenceContext
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ctx.Task.ID, decoded.Task.ID)
	assert.Equal(t, ctx.Task.Name, decoded.Task.Name)
	assert.Equal(t, ctx.FrameworkReqs, decoded.FrameworkReqs)
	assert.Equal(t, ctx.OutputFormat, decoded.OutputFormat)
	assert.NotNil(t, decoded.SecurityMappings.SOC2)
	assert.Contains(t, decoded.SecurityMappings.SOC2, "CC6.1")
}

// Test IntOrString type
func TestIntOrString_String(t *testing.T) {
	tests := []struct {
		name     string
		input    IntOrString
		expected string
	}{
		{
			name:     "numeric string",
			input:    IntOrString("123"),
			expected: "123",
		},
		{
			name:     "text string",
			input:    IntOrString("test"),
			expected: "test",
		},
		{
			name:     "empty string",
			input:    IntOrString(""),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test IntOrString ToInt conversion
func TestIntOrString_ToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    IntOrString
		expected int
	}{
		{
			name:     "valid number",
			input:    IntOrString("123"),
			expected: 123,
		},
		{
			name:     "zero",
			input:    IntOrString("0"),
			expected: 0,
		},
		{
			name:     "invalid - returns default",
			input:    IntOrString("not-a-number"),
			expected: 0,
		},
		{
			name:     "empty - returns default",
			input:    IntOrString(""),
			expected: 0,
		},
		{
			name:     "negative number",
			input:    IntOrString("-42"),
			expected: -42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToInt()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test GitHubAccessControlMatrix model
func TestGitHubAccessControlMatrix_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	matrix := GitHubAccessControlMatrix{
		Repository: GitHubRepositoryInfo{
			Name:          "test-repo",
			FullName:      "org/test-repo",
			Owner:         "org",
			Private:       true,
			DefaultBranch: "main",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Collaborators: []GitHubCollaborator{
			{
				Login: "user1",
				ID:    123,
				Type:  "User",
				Permissions: GitHubPermissions{
					Permission: "admin",
					Admin:      true,
					Push:       true,
					Pull:       true,
				},
			},
		},
		Teams: []GitHubTeam{
			{
				ID:   456,
				Name: "Engineering",
				Slug: "engineering",
			},
		},
		SecuritySettings: GitHubSecuritySettings{
			VulnerabilityAlertsEnabled: true,
			SecretScanningEnabled:      true,
		},
		ExtractedAt: now,
		AccessSummary: GitHubAccessSummary{
			TotalCollaborators: 1,
			TotalTeams:         1,
			AdminUsers:         []string{"user1"},
		},
	}

	data, err := json.Marshal(matrix)
	require.NoError(t, err)

	var decoded GitHubAccessControlMatrix
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, matrix.Repository.Name, decoded.Repository.Name)
	assert.Equal(t, matrix.Repository.FullName, decoded.Repository.FullName)
	assert.Equal(t, len(matrix.Collaborators), len(decoded.Collaborators))
	assert.Equal(t, matrix.Collaborators[0].Login, decoded.Collaborators[0].Login)
	assert.Equal(t, len(matrix.Teams), len(decoded.Teams))
	assert.True(t, decoded.SecuritySettings.VulnerabilityAlertsEnabled)
}

// Test GeneratedEvidence model
func TestGeneratedEvidence_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	evidence := GeneratedEvidence{
		TaskID:          123,
		GeneratedAt:     now,
		GeneratedBy:     "claude-3-5-sonnet",
		EvidenceFormat:  "csv",
		EvidenceContent: "header1,header2\nvalue1,value2",
		SourcesUsed: []EvidenceSource{
			{
				Type:        "terraform",
				Resource:    "/path/to/main.tf",
				Content:     "resource content",
				Relevance:   0.95,
				ExtractedAt: now,
			},
		},
		Reasoning:       "Evidence collected from Terraform files",
		Completeness:    0.85,
		QualityScore:    "high",
		OutputDirectory: "/path/to/output",
		Status:          "ready",
		CreatedAt:       now,
		UpdatedAt:       now,
		ToolsUsed:       []string{"terraform_scanner", "github_searcher"},
		Conversation: []ConversationEntry{
			{
				Role:      "user",
				Content:   "Generate evidence",
				Timestamp: now,
			},
		},
	}

	data, err := json.Marshal(evidence)
	require.NoError(t, err)

	var decoded GeneratedEvidence
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, evidence.TaskID, decoded.TaskID)
	assert.Equal(t, evidence.GeneratedBy, decoded.GeneratedBy)
	assert.Equal(t, evidence.EvidenceFormat, decoded.EvidenceFormat)
	assert.Equal(t, evidence.Completeness, decoded.Completeness)
	assert.Equal(t, evidence.QualityScore, decoded.QualityScore)
	assert.Equal(t, evidence.Status, decoded.Status)
	assert.Equal(t, len(evidence.SourcesUsed), len(decoded.SourcesUsed))
	assert.Equal(t, len(evidence.ToolsUsed), len(decoded.ToolsUsed))
	assert.Equal(t, len(evidence.Conversation), len(decoded.Conversation))
}

// Test TerraformResource model with MultiAZ and HA config
func TestTerraformResource_JSONMarshaling(t *testing.T) {
	resource := TerraformResource{
		Type: "aws_rds_instance",
		Name: "main-db",
		Config: map[string]interface{}{
			"engine":         "postgres",
			"instance_class": "db.t3.micro",
		},
		Tags: map[string]interface{}{
			"Environment": "production",
			"Team":        "platform",
		},
		FilePath:          "/path/to/database.tf",
		SecurityRelevance: []string{"CC6.6", "CC7.2"},
		MultiAZConfig: &MultiAZConfiguration{
			IsMultiAZ:         true,
			AvailabilityZones: []string{"us-east-1a", "us-east-1b"},
		},
		HAConfig: &HighAvailabilityConfig{
			HasFailover:        true,
			HasLoadBalancing:   false,
			HasAutoScaling:     false,
			HasBackup:          true,
			ReplicationEnabled: true,
		},
	}

	data, err := json.Marshal(resource)
	require.NoError(t, err)

	var decoded TerraformResource
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resource.Type, decoded.Type)
	assert.Equal(t, resource.Name, decoded.Name)
	assert.Equal(t, resource.FilePath, decoded.FilePath)
	assert.NotNil(t, decoded.MultiAZConfig)
	assert.True(t, decoded.MultiAZConfig.IsMultiAZ)
	assert.Equal(t, 2, len(decoded.MultiAZConfig.AvailabilityZones))
	assert.NotNil(t, decoded.HAConfig)
	assert.True(t, decoded.HAConfig.HasFailover)
}

// Test ConvertHCLRange function
func TestConvertHCLRange(t *testing.T) {
	hclRange := hcl.Range{
		Filename: "main.tf",
		Start: hcl.Pos{
			Line:   10,
			Column: 5,
			Byte:   120,
		},
		End: hcl.Pos{
			Line:   20,
			Column: 15,
			Byte:   250,
		},
	}

	result := ConvertHCLRange(hclRange)

	assert.NotNil(t, result)
	assert.Equal(t, "main.tf", result.Filename)
	assert.Equal(t, 10, result.Start.Line)
	assert.Equal(t, 5, result.Start.Column)
	assert.Equal(t, 120, result.Start.Byte)
	assert.Equal(t, 20, result.End.Line)
	assert.Equal(t, 15, result.End.Column)
	assert.Equal(t, 250, result.End.Byte)
}

// Test ConvertHCLDiagnostics function
func TestConvertHCLDiagnostics(t *testing.T) {
	diagnostics := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Test error",
			Detail:   "This is a test error",
			Subject: &hcl.Range{
				Filename: "test.tf",
				Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
				End:      hcl.Pos{Line: 1, Column: 10, Byte: 10},
			},
		},
		{
			Severity: hcl.DiagWarning,
			Summary:  "Test warning",
			Detail:   "This is a test warning",
		},
	}

	result := ConvertHCLDiagnostics(diagnostics)

	assert.Len(t, result, 2)
	assert.Equal(t, "error", result[0].Severity)
	assert.Equal(t, "Test error", result[0].Summary)
	assert.Equal(t, "This is a test error", result[0].Detail)
	assert.NotNil(t, result[0].Subject)
	assert.Equal(t, "test.tf", result[0].Subject.Filename)

	assert.Equal(t, "warning", result[1].Severity)
	assert.Equal(t, "Test warning", result[1].Summary)
	assert.Nil(t, result[1].Subject)
}

// Test GetCTYTypeString function
func TestGetCTYTypeString(t *testing.T) {
	tests := []struct {
		name     string
		input    cty.Type
		expected string
	}{
		{
			name:     "string type",
			input:    cty.String,
			expected: "string",
		},
		{
			name:     "number type",
			input:    cty.Number,
			expected: "number",
		},
		{
			name:     "bool type",
			input:    cty.Bool,
			expected: "bool",
		},
		{
			name:     "nil type",
			input:    cty.NilType,
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCTYTypeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test EvidenceSource with metadata
func TestEvidenceSource_WithMetadata(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	source := EvidenceSource{
		Type:        "github",
		Resource:    "issue-123",
		Content:     "Security policy documentation",
		Relevance:   0.9,
		ExtractedAt: now,
		Metadata: map[string]interface{}{
			"issue_number": 123,
			"labels":       []string{"security", "compliance"},
			"state":        "closed",
		},
	}

	data, err := json.Marshal(source)
	require.NoError(t, err)

	var decoded EvidenceSource
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, source.Type, decoded.Type)
	assert.Equal(t, source.Resource, decoded.Resource)
	assert.Equal(t, source.Relevance, decoded.Relevance)
	assert.NotNil(t, decoded.Metadata)
	assert.Contains(t, decoded.Metadata, "issue_number")
}

// Test ClaudeRequest and ClaudeResponse models
func TestClaude_Models(t *testing.T) {
	request := ClaudeRequest{
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   4096,
		Temperature: 0.7,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: "Generate evidence for SOC2 compliance",
			},
		},
		Tools: []ClaudeTool{
			{
				Name:        "terraform_scanner",
				Description: "Scan Terraform files",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{"type": "string"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded ClaudeRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, request.Model, decoded.Model)
	assert.Equal(t, request.MaxTokens, decoded.MaxTokens)
	assert.Equal(t, len(request.Messages), len(decoded.Messages))
	assert.Equal(t, len(request.Tools), len(decoded.Tools))

	// Test ClaudeResponse
	response := ClaudeResponse{
		ID:    "msg-123",
		Type:  "message",
		Role:  "assistant",
		Model: "claude-3-5-sonnet-20241022",
		Content: []ClaudeContent{
			{
				Type: "text",
				Text: "Evidence generated successfully",
			},
		},
		StopReason: "end_turn",
		Usage: ClaudeUsage{
			InputTokens:  100,
			OutputTokens: 50,
		},
	}

	data, err = json.Marshal(response)
	require.NoError(t, err)

	var decodedResp ClaudeResponse
	err = json.Unmarshal(data, &decodedResp)
	require.NoError(t, err)

	assert.Equal(t, response.ID, decodedResp.ID)
	assert.Equal(t, response.Model, decodedResp.Model)
	assert.Equal(t, len(response.Content), len(decodedResp.Content))
	assert.Equal(t, response.Usage.InputTokens, decodedResp.Usage.InputTokens)
}

// Test SecurityFinding model
func TestSecurityFinding_JSONMarshaling(t *testing.T) {
	finding := SecurityFinding{
		Resource: ResourceIdentifier{
			Type:     "resource",
			Kind:     "aws_s3_bucket",
			Name:     "data-bucket",
			FilePath: "/path/to/storage.tf",
		},
		FindingType:    "encryption",
		Severity:       "high",
		Title:          "S3 bucket encryption not enabled",
		Description:    "The S3 bucket does not have server-side encryption enabled",
		Recommendation: "Enable AES256 or KMS encryption",
		ControlCodes:   []string{"CC6.6", "CC6.7"},
		CVSSScore:      7.5,
		Evidence: map[string]interface{}{
			"bucket_name":       "data-bucket",
			"encryption_status": "disabled",
		},
	}

	data, err := json.Marshal(finding)
	require.NoError(t, err)

	var decoded SecurityFinding
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, finding.FindingType, decoded.FindingType)
	assert.Equal(t, finding.Severity, decoded.Severity)
	assert.Equal(t, finding.Title, decoded.Title)
	assert.Equal(t, finding.Resource.Kind, decoded.Resource.Kind)
	assert.Equal(t, finding.CVSSScore, decoded.CVSSScore)
	assert.Equal(t, len(finding.ControlCodes), len(decoded.ControlCodes))
}

// Test InfrastructureTopology with comprehensive data
func TestInfrastructureTopology_JSONMarshaling(t *testing.T) {
	topology := InfrastructureTopology{
		Regions: []RegionTopology{
			{
				Region:            "us-east-1",
				AvailabilityZones: []string{"us-east-1a", "us-east-1b"},
				VPCs:              []string{"vpc-123"},
				ResourceCounts: map[string]int{
					"aws_instance": 5,
					"aws_rds":      2,
				},
			},
		},
		Databases: []DatabaseInfo{
			{
				Name:              "main-db",
				Engine:            "postgres",
				MultiAZ:           true,
				BackupRetention:   7,
				EncryptionEnabled: true,
				MonitoringEnabled: true,
			},
		},
		EncryptionSummary: &EncryptionSummary{
			AtRestEncryption: &EncryptionStatus{
				Enabled:   true,
				Coverage:  85.5,
				Resources: []string{"aws_rds", "aws_s3_bucket"},
			},
			KeyManagement: &KeyManagementInfo{
				CustomerManaged: true,
				KeyRotation:     true,
			},
		},
		HAAnalysis: &HAAnalysis{
			MultiAZCoverage:      75.0,
			LoadBalancedServices: 3,
			AutoScalingGroups:    2,
			DatabaseReplication:  1,
		},
	}

	data, err := json.Marshal(topology)
	require.NoError(t, err)

	var decoded InfrastructureTopology
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, len(topology.Regions), len(decoded.Regions))
	assert.Equal(t, topology.Regions[0].Region, decoded.Regions[0].Region)
	assert.NotNil(t, decoded.EncryptionSummary)
	assert.True(t, decoded.EncryptionSummary.AtRestEncryption.Enabled)
	assert.NotNil(t, decoded.HAAnalysis)
	assert.Equal(t, topology.HAAnalysis.MultiAZCoverage, decoded.HAAnalysis.MultiAZCoverage)
}

// Test EvidenceDataPackage model (prompt-as-data pattern)
func TestEvidenceDataPackage_JSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	pkg := EvidenceDataPackage{
		TaskID:      123,
		GeneratedAt: now,
		Prompt:      "Generate SOC2 compliance evidence",
		ToolOutputs: []ToolOutput{
			{
				ToolName:  "terraform_scanner",
				Result:    "Found 10 resources",
				Success:   true,
				Timestamp: now,
				Metadata: map[string]interface{}{
					"resource_count": 10,
				},
			},
		},
		Sources: []EvidenceSource{
			{
				Type:        "terraform",
				Resource:    "/path/to/main.tf",
				Relevance:   0.9,
				ExtractedAt: now,
			},
		},
		Metadata: map[string]interface{}{
			"tool_count":      1,
			"generation_mode": "prompt-as-data",
		},
	}

	data, err := json.Marshal(pkg)
	require.NoError(t, err)

	var decoded EvidenceDataPackage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pkg.TaskID, decoded.TaskID)
	assert.Equal(t, pkg.Prompt, decoded.Prompt)
	assert.Equal(t, len(pkg.ToolOutputs), len(decoded.ToolOutputs))
	assert.Equal(t, len(pkg.Sources), len(decoded.Sources))
	assert.NotNil(t, decoded.Metadata)
}
