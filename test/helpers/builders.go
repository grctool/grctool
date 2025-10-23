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

//go:build !e2e && !functional

package helpers

import (
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/models"
)

// EvidenceTaskBuilder provides a fluent interface for building test EvidenceTask instances
type EvidenceTaskBuilder struct {
	task *models.EvidenceTask
}

// NewEvidenceTaskBuilder creates a new builder with reasonable defaults
func NewEvidenceTaskBuilder() *EvidenceTaskBuilder {
	return &EvidenceTaskBuilder{
		task: &models.EvidenceTask{
			ID:                 1001,
			Name:               "Test Evidence Task",
			Description:        "A test evidence task for unit testing",
			Guidance:           "Follow standard evidence collection procedures",
			AdHoc:              false,
			CollectionInterval: "quarterly",
			DueDaysBefore:      30,
			MasterVersionNum:   1,
			Sensitive:          false,
		},
	}
}

// WithID sets the ID
func (b *EvidenceTaskBuilder) WithID(id int) *EvidenceTaskBuilder {
	b.task.ID = id
	return b
}

// WithName sets the name
func (b *EvidenceTaskBuilder) WithName(name string) *EvidenceTaskBuilder {
	b.task.Name = name
	return b
}

// WithDescription sets the description
func (b *EvidenceTaskBuilder) WithDescription(description string) *EvidenceTaskBuilder {
	b.task.Description = description
	return b
}

// WithGuidance sets the guidance
func (b *EvidenceTaskBuilder) WithGuidance(guidance string) *EvidenceTaskBuilder {
	b.task.Guidance = guidance
	return b
}

// WithCollectionInterval sets the collection interval
func (b *EvidenceTaskBuilder) WithCollectionInterval(interval string) *EvidenceTaskBuilder {
	b.task.CollectionInterval = interval
	return b
}

// WithLastCollected sets the last collected date
func (b *EvidenceTaskBuilder) WithLastCollected(lastCollected string) *EvidenceTaskBuilder {
	b.task.LastCollected = &lastCollected
	return b
}

// WithAdHoc marks the task as ad-hoc
func (b *EvidenceTaskBuilder) WithAdHoc(adHoc bool) *EvidenceTaskBuilder {
	b.task.AdHoc = adHoc
	return b
}

// WithSensitive marks the task as sensitive
func (b *EvidenceTaskBuilder) WithSensitive(sensitive bool) *EvidenceTaskBuilder {
	b.task.Sensitive = sensitive
	return b
}

// WithDueDaysBefore sets the due days before end
func (b *EvidenceTaskBuilder) WithDueDaysBefore(days int) *EvidenceTaskBuilder {
	b.task.DueDaysBefore = days
	return b
}

// WithMasterVersionNum sets the master version number
func (b *EvidenceTaskBuilder) WithMasterVersionNum(version int) *EvidenceTaskBuilder {
	b.task.MasterVersionNum = version
	return b
}

// Build returns the constructed EvidenceTask
func (b *EvidenceTaskBuilder) Build() *models.EvidenceTask {
	// Return a copy to prevent modification of the builder's internal state
	task := *b.task
	if b.task.LastCollected != nil {
		lastCollected := *b.task.LastCollected
		task.LastCollected = &lastCollected
	}
	return &task
}

// EvidenceTaskDetailsBuilder provides a fluent interface for building test EvidenceTaskDetails instances
type EvidenceTaskDetailsBuilder struct {
	details *models.EvidenceTaskDetails
}

// NewEvidenceTaskDetailsBuilder creates a new builder with defaults
func NewEvidenceTaskDetailsBuilder() *EvidenceTaskDetailsBuilder {
	baseTask := NewEvidenceTaskBuilder().Build()
	return &EvidenceTaskDetailsBuilder{
		details: &models.EvidenceTaskDetails{
			EvidenceTask:          *baseTask,
			OpenIncidentCount:     0,
			JiraIssues:            []models.JiraIssue{},
			Assignees:             []models.EvidenceAssignee{},
			SupportedIntegrations: []models.SupportedIntegration{},
			Tags:                  []models.EvidenceTag{},
		},
	}
}

// WithEvidenceTask sets the base evidence task
func (b *EvidenceTaskDetailsBuilder) WithEvidenceTask(task models.EvidenceTask) *EvidenceTaskDetailsBuilder {
	b.details.EvidenceTask = task
	return b
}

// WithMasterContent sets the master content
func (b *EvidenceTaskDetailsBuilder) WithMasterContent(content *models.MasterContent) *EvidenceTaskDetailsBuilder {
	b.details.MasterContent = content
	return b
}

// WithOrgScope sets the org scope
func (b *EvidenceTaskDetailsBuilder) WithOrgScope(orgScope *models.OrgScope) *EvidenceTaskDetailsBuilder {
	b.details.OrgScope = orgScope
	return b
}

// WithTags sets the tags
func (b *EvidenceTaskDetailsBuilder) WithTags(tags []models.EvidenceTag) *EvidenceTaskDetailsBuilder {
	b.details.Tags = tags
	return b
}

// WithAssignees sets the assignees
func (b *EvidenceTaskDetailsBuilder) WithAssignees(assignees []models.EvidenceAssignee) *EvidenceTaskDetailsBuilder {
	b.details.Assignees = assignees
	return b
}

// WithOpenIncidentCount sets the open incident count
func (b *EvidenceTaskDetailsBuilder) WithOpenIncidentCount(count int) *EvidenceTaskDetailsBuilder {
	b.details.OpenIncidentCount = count
	return b
}

// Build returns the constructed EvidenceTaskDetails
func (b *EvidenceTaskDetailsBuilder) Build() *models.EvidenceTaskDetails {
	// Return a copy to prevent modification
	details := *b.details
	return &details
}

// PolicyBuilder provides a fluent interface for building test Policy instances
type PolicyBuilder struct {
	policy *models.Policy
}

// NewPolicyBuilder creates a new builder with reasonable defaults
func NewPolicyBuilder() *PolicyBuilder {
	now := time.Now()
	return &PolicyBuilder{
		policy: &models.Policy{
			ID:          models.IntOrString("2001"),
			Name:        "Test Security Policy",
			Description: "A test security policy for unit testing",
			Framework:   "SOC2",
			Status:      "active",
			Controls:    []models.Control{},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// WithID sets the ID
func (b *PolicyBuilder) WithID(id int) *PolicyBuilder {
	b.policy.ID = models.IntOrString(fmt.Sprintf("%d", id))
	return b
}

// WithStringID sets the ID as string
func (b *PolicyBuilder) WithStringID(id string) *PolicyBuilder {
	b.policy.ID = models.IntOrString(id)
	return b
}

// WithName sets the name
func (b *PolicyBuilder) WithName(name string) *PolicyBuilder {
	b.policy.Name = name
	return b
}

// WithDescription sets the description
func (b *PolicyBuilder) WithDescription(description string) *PolicyBuilder {
	b.policy.Description = description
	return b
}

// WithFramework sets the framework
func (b *PolicyBuilder) WithFramework(framework string) *PolicyBuilder {
	b.policy.Framework = framework
	return b
}

// WithStatus sets the status
func (b *PolicyBuilder) WithStatus(status string) *PolicyBuilder {
	b.policy.Status = status
	return b
}

// WithControls sets the controls
func (b *PolicyBuilder) WithControls(controls []models.Control) *PolicyBuilder {
	b.policy.Controls = controls
	return b
}

// WithCreatedAt sets the created date
func (b *PolicyBuilder) WithCreatedAt(createdAt time.Time) *PolicyBuilder {
	b.policy.CreatedAt = createdAt
	return b
}

// WithUpdatedAt sets the updated date
func (b *PolicyBuilder) WithUpdatedAt(updatedAt time.Time) *PolicyBuilder {
	b.policy.UpdatedAt = updatedAt
	return b
}

// Build returns the constructed Policy
func (b *PolicyBuilder) Build() *models.Policy {
	// Return a copy to prevent modification
	policy := *b.policy
	if len(b.policy.Controls) > 0 {
		policy.Controls = make([]models.Control, len(b.policy.Controls))
		copy(policy.Controls, b.policy.Controls)
	}
	return &policy
}

// ControlBuilder provides a fluent interface for building test Control instances
type ControlBuilder struct {
	control *models.Control
}

// NewControlBuilder creates a new builder with reasonable defaults
func NewControlBuilder() *ControlBuilder {
	return &ControlBuilder{
		control: &models.Control{
			ID:                3001,
			Name:              "Test Control",
			Body:              "This is a test control for unit testing",
			Category:          "Access Control",
			Status:            "implemented",
			Risk:              "medium",
			Help:              "Follow standard control procedures",
			IsAutoImplemented: false,
		},
	}
}

// WithID sets the ID
func (b *ControlBuilder) WithID(id int) *ControlBuilder {
	b.control.ID = id
	return b
}

// WithName sets the name
func (b *ControlBuilder) WithName(name string) *ControlBuilder {
	b.control.Name = name
	return b
}

// WithBody sets the body (description)
func (b *ControlBuilder) WithBody(body string) *ControlBuilder {
	b.control.Body = body
	return b
}

// WithCategory sets the category
func (b *ControlBuilder) WithCategory(category string) *ControlBuilder {
	b.control.Category = category
	return b
}

// WithStatus sets the status
func (b *ControlBuilder) WithStatus(status string) *ControlBuilder {
	b.control.Status = status
	return b
}

// WithRisk sets the risk level
func (b *ControlBuilder) WithRisk(risk string) *ControlBuilder {
	b.control.Risk = risk
	return b
}

// WithRiskLevel sets the risk level as interface
func (b *ControlBuilder) WithRiskLevel(riskLevel interface{}) *ControlBuilder {
	b.control.RiskLevel = riskLevel
	return b
}

// WithHelp sets the help text
func (b *ControlBuilder) WithHelp(help string) *ControlBuilder {
	b.control.Help = help
	return b
}

// WithAutoImplemented marks the control as auto-implemented
func (b *ControlBuilder) WithAutoImplemented(autoImplemented bool) *ControlBuilder {
	b.control.IsAutoImplemented = autoImplemented
	return b
}

// WithImplementedDate sets the implemented date
func (b *ControlBuilder) WithImplementedDate(implementedDate interface{}) *ControlBuilder {
	b.control.ImplementedDate = implementedDate
	return b
}

// Build returns the constructed Control
func (b *ControlBuilder) Build() *models.Control {
	// Return a copy to prevent modification
	control := *b.control
	return &control
}

// EvidenceAssigneeBuilder provides a fluent interface for building test EvidenceAssignee instances
type EvidenceAssigneeBuilder struct {
	assignee *models.EvidenceAssignee
}

// NewEvidenceAssigneeBuilder creates a new builder with defaults
func NewEvidenceAssigneeBuilder() *EvidenceAssigneeBuilder {
	return &EvidenceAssigneeBuilder{
		assignee: &models.EvidenceAssignee{
			ID:    "user-123",
			Name:  "Test User",
			Email: "test@example.com",
			Role:  "evidence_collector",
		},
	}
}

// WithID sets the ID
func (b *EvidenceAssigneeBuilder) WithID(id string) *EvidenceAssigneeBuilder {
	b.assignee.ID = id
	return b
}

// WithName sets the name
func (b *EvidenceAssigneeBuilder) WithName(name string) *EvidenceAssigneeBuilder {
	b.assignee.Name = name
	return b
}

// WithEmail sets the email
func (b *EvidenceAssigneeBuilder) WithEmail(email string) *EvidenceAssigneeBuilder {
	b.assignee.Email = email
	return b
}

// WithRole sets the role
func (b *EvidenceAssigneeBuilder) WithRole(role string) *EvidenceAssigneeBuilder {
	b.assignee.Role = role
	return b
}

// Build returns the constructed EvidenceAssignee
func (b *EvidenceAssigneeBuilder) Build() *models.EvidenceAssignee {
	assignee := *b.assignee
	return &assignee
}

// EvidenceTagBuilder provides a fluent interface for building test EvidenceTag instances
type EvidenceTagBuilder struct {
	tag *models.EvidenceTag
}

// NewEvidenceTagBuilder creates a new builder with defaults
func NewEvidenceTagBuilder() *EvidenceTagBuilder {
	return &EvidenceTagBuilder{
		tag: &models.EvidenceTag{
			ID:    "tag-1",
			Name:  "test-tag",
			Color: "#007bff",
		},
	}
}

// WithID sets the ID
func (b *EvidenceTagBuilder) WithID(id string) *EvidenceTagBuilder {
	b.tag.ID = id
	return b
}

// WithName sets the name
func (b *EvidenceTagBuilder) WithName(name string) *EvidenceTagBuilder {
	b.tag.Name = name
	return b
}

// WithColor sets the color
func (b *EvidenceTagBuilder) WithColor(color string) *EvidenceTagBuilder {
	b.tag.Color = color
	return b
}

// Build returns the constructed EvidenceTag
func (b *EvidenceTagBuilder) Build() *models.EvidenceTag {
	tag := *b.tag
	return &tag
}

// Usage examples:
/*
func TestSomeFunction(t *testing.T) {
	// Build a simple evidence task
	task := helpers.NewEvidenceTaskBuilder().
		WithName("Access Control Review").
		WithCollectionInterval("monthly").
		WithSensitive(true).
		Build()

	// Build a policy with controls
	control := helpers.NewControlBuilder().
		WithName("Password Policy").
		WithCategory("Authentication").
		Build()

	policy := helpers.NewPolicyBuilder().
		WithName("Information Security Policy").
		WithFramework("ISO27001").
		WithControls([]models.Control{*control}).
		Build()

	// Build detailed evidence task with assignees
	assignee := helpers.NewEvidenceAssigneeBuilder().
		WithName("Jane Doe").
		WithRole("security_analyst").
		Build()

	taskDetails := helpers.NewEvidenceTaskDetailsBuilder().
		WithEvidenceTask(*task).
		WithAssignees([]models.EvidenceAssignee{*assignee}).
		WithOpenIncidentCount(2).
		Build()

	// Use in tests...
}
*/
