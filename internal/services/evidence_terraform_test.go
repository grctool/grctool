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

package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDataService implements DataService interface for testing
type mockDataService struct{}

func (m *mockDataService) GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error) {
	return nil, nil
}

func (m *mockDataService) GetAllEvidenceTasks(ctx context.Context) ([]domain.EvidenceTask, error) {
	return nil, nil
}

func (m *mockDataService) FilterEvidenceTasks(ctx context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error) {
	return nil, nil
}

func (m *mockDataService) GetControl(ctx context.Context, controlID string) (*domain.Control, error) {
	return nil, nil
}

func (m *mockDataService) GetAllControls(ctx context.Context) ([]domain.Control, error) {
	return nil, nil
}

func (m *mockDataService) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	return nil, nil
}

func (m *mockDataService) GetAllPolicies(ctx context.Context) ([]domain.Policy, error) {
	return nil, nil
}

func (m *mockDataService) GetRelationships(ctx context.Context, sourceType, sourceID string) ([]domain.Relationship, error) {
	return nil, nil
}

func (m *mockDataService) SaveEvidenceRecord(ctx context.Context, record *domain.EvidenceRecord) error {
	return nil
}

func (m *mockDataService) GetEvidenceRecords(ctx context.Context, taskID int) ([]domain.EvidenceRecord, error) {
	return nil, nil
}

func TestSaveTerraformSnippets(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test service
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				OutputDir:        tempDir,
				IncludeReasoning: true,
			},
		},
	}
	log, _ := logger.New(logger.DefaultConfig())

	// Create a mock data service
	mockDataService := &mockDataService{}
	service, err := NewEvidenceService(mockDataService, cfg, log)
	require.NoError(t, err)

	// Test terraform content with multiple resources
	terraformContent := `
Resource Type,Resource Name,File Path,Line Range,Security Controls,Key Configuration
aws_autoscaling_group,web_asg,/terraform/prod/autoscaling.tf,10-45,SO2,min_size=2; max_size=10; desired_capacity=4
aws_autoscaling_policy,web_scale_up,/terraform/prod/autoscaling.tf,47-65,SO2,scaling_adjustment=2; adjustment_type=ChangeInCapacity
aws_appautoscaling_target,ecs_target,/terraform/prod/ecs.tf,20-35,SO2,min_capacity=1; max_capacity=20; scalable_dimension=ecs:service:DesiredCount

The following terraform resources were found:

resource "aws_autoscaling_group" "web_asg" {
  name                = "web-autoscaling-group"
  min_size            = 2
  max_size            = 10
  desired_capacity    = 4
  health_check_type   = "ELB"
  health_check_grace_period = 300
  
  launch_template {
    id      = aws_launch_template.web.id
    version = "$Latest"
  }
  
  vpc_zone_identifier = aws_subnet.private[*].id
  
  tag {
    key                 = "Name"
    value               = "web-asg-instance"
    propagate_at_launch = true
  }
}

resource "aws_autoscaling_policy" "web_scale_up" {
  name                   = "web-scale-up"
  scaling_adjustment     = 2
  adjustment_type        = "ChangeInCapacity"
  cooldown               = 300
  autoscaling_group_name = aws_autoscaling_group.web_asg.name
}

resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = 20
  min_capacity       = 1
  resource_id        = "service/default/web-service"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}
`

	// Create test evidence with terraform source
	evidence := &models.GeneratedEvidence{
		TaskID:          87,
		GeneratedAt:     time.Now(),
		GeneratedBy:     "test",
		EvidenceFormat:  "csv",
		EvidenceContent: "Test evidence content",
		SourcesUsed: []models.EvidenceSource{
			{
				Type:    "terraform",
				Content: terraformContent,
			},
		},
	}

	// Save the evidence
	err = service.saveGeneratedEvidence(evidence)
	require.NoError(t, err)

	// Verify the main directory was created
	taskDir := filepath.Join(tempDir, "ET87")
	assert.DirExists(t, taskDir)

	// Verify terraform snippets directory was created
	terraformDir := filepath.Join(taskDir, "terraform_snippets")
	assert.DirExists(t, terraformDir)

	// Verify individual terraform files were created
	expectedFiles := []string{
		"01_aws_autoscaling_group_web_asg.tf",
		"02_aws_autoscaling_policy_web_scale_up.tf",
		"03_aws_appautoscaling_target_ecs_target.tf",
		"00_summary.md",
	}

	for _, fileName := range expectedFiles {
		filePath := filepath.Join(terraformDir, fileName)
		assert.FileExists(t, filePath, "Expected file %s to exist", fileName)
	}

	// Verify content of one of the terraform files
	asgContent, err := os.ReadFile(filepath.Join(terraformDir, "01_aws_autoscaling_group_web_asg.tf"))
	require.NoError(t, err)
	assert.Contains(t, string(asgContent), "resource \"aws_autoscaling_group\" \"web_asg\"")
	assert.Contains(t, string(asgContent), "min_size            = 2")
	assert.Contains(t, string(asgContent), "max_size            = 10")

	// Verify summary file content
	summaryContent, err := os.ReadFile(filepath.Join(terraformDir, "00_summary.md"))
	require.NoError(t, err)
	assert.Contains(t, string(summaryContent), "# Terraform Resources Summary")
	assert.Contains(t, string(summaryContent), "Total resources found: 3")
	assert.Contains(t, string(summaryContent), "`aws_autoscaling_group.web_asg`")
}

func TestSaveTerraformSnippets_NoTerraformContent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test service
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				OutputDir: tempDir,
			},
		},
	}
	log, _ := logger.New(logger.DefaultConfig())

	// Create a mock data service
	mockDataService := &mockDataService{}
	service, err := NewEvidenceService(mockDataService, cfg, log)
	require.NoError(t, err)

	// Create test evidence without terraform source
	evidence := &models.GeneratedEvidence{
		TaskID:          88,
		GeneratedAt:     time.Now(),
		GeneratedBy:     "test",
		EvidenceFormat:  "markdown",
		EvidenceContent: "Test evidence content without terraform",
		SourcesUsed: []models.EvidenceSource{
			{
				Type:    "manual",
				Content: "Manual evidence content",
			},
		},
	}

	// Save the evidence
	err = service.saveGeneratedEvidence(evidence)
	require.NoError(t, err)

	// Verify the main directory was created
	taskDir := filepath.Join(tempDir, "ET88")
	assert.DirExists(t, taskDir)

	// Verify terraform snippets directory was NOT created
	terraformDir := filepath.Join(taskDir, "terraform_snippets")
	assert.NoDirExists(t, terraformDir)
}
