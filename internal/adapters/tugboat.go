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

package adapters

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/domain"
	tugboatmodels "github.com/grctool/grctool/internal/tugboat/models"
)

// TugboatToDomain converts Tugboat API models to domain models
type TugboatToDomain struct{}

// NewTugboatToDomain creates a new adapter for converting Tugboat API models to domain models
func NewTugboatToDomain() *TugboatToDomain {
	return &TugboatToDomain{}
}

// ConvertPolicy converts a Tugboat API Policy or PolicyDetails to a domain Policy
func (t *TugboatToDomain) ConvertPolicy(apiPolicy interface{}) domain.Policy {
	switch p := apiPolicy.(type) {
	case tugboatmodels.Policy:
		return domain.Policy{
			ID:          p.ID.String(),
			Name:        p.Name,
			Description: p.Description,
			Framework:   p.Framework,
			Status:      p.Status,
			CreatedAt:   p.CreatedAt.Time,
			UpdatedAt:   p.UpdatedAt.Time,
		}
	case tugboatmodels.PolicyDetails:
		policy := domain.Policy{
			ID:          p.ID.String(),
			Name:        p.Name,
			Description: p.Summary, // Use summary as description
			Framework:   p.Framework,
			Status:      p.Status,
			CreatedAt:   p.CreatedAt.Time,
			UpdatedAt:   p.UpdatedAt.Time,
			// Reference and categorization fields from API
			Category:         p.Category,
			MasterPolicyID:   p.MasterPolicyID.String(),
			VersionNum:       p.VersionNum,
			MasterVersionNum: p.MasterVersionNum,
			// Content fields
			Summary:          p.Summary,
			Content:          p.Details,
			DeprecationNotes: p.DeprecationNotes,
			// Relationships
			Tags:      t.convertPolicyTags(p.Tags),
			Assignees: t.convertPolicyAssignees(p.Assignees),
			Reviewers: t.convertPolicyReviewers(p.Reviewers),
		}

		// Set version from current version if available
		if p.CurrentVersion != nil {
			policy.Version = p.CurrentVersion.Version
			policy.CurrentVersion = t.convertPolicyVersion(p.CurrentVersion)
			// Only use current_version.content if details field is empty
			if policy.Content == "" {
				policy.Content = p.CurrentVersion.Content
			}
		}

		// Set latest version if available
		if p.LatestVersion != nil {
			policy.LatestVersion = t.convertPolicyVersion(p.LatestVersion)
		}

		// Set association counts if available
		if p.AssociationCounts != nil {
			policy.ControlCount = p.AssociationCounts.Controls
			policy.ProcedureCount = p.AssociationCounts.Procedures
			policy.EvidenceCount = p.AssociationCounts.Evidence
		}

		// Set usage statistics if available
		if p.Usage != nil {
			policy.ViewCount = p.Usage.ViewCount
			policy.LastViewedAt = &p.Usage.LastViewedAt.Time
			policy.DownloadCount = p.Usage.DownloadCount
			policy.LastDownloadedAt = &p.Usage.LastDownloaded.Time
			policy.ReferenceCount = p.Usage.ReferenceCount
			policy.LastReferencedAt = &p.Usage.LastReferenced.Time
		}

		return policy
	default:
		return domain.Policy{}
	}
}

// ConvertControl converts a Tugboat API Control or ControlDetails to a domain Control
func (t *TugboatToDomain) ConvertControl(apiControl interface{}) domain.Control {
	switch c := apiControl.(type) {
	case tugboatmodels.Control:
		var implementedDate, testedDate *time.Time

		// Handle implemented_date conversion (can be null or date)
		if c.ImplementedDate != nil {
			if parsedTime, err := time.Parse(time.RFC3339, *c.ImplementedDate); err == nil {
				implementedDate = &parsedTime
			}
		}

		// Handle tested_date conversion (can be null or date)
		if c.TestedDate != nil {
			if parsedTime, err := time.Parse(time.RFC3339, *c.TestedDate); err == nil {
				testedDate = &parsedTime
			}
		}

		// Handle risk_level conversion (can be null or string)
		riskLevel := ""
		if c.RiskLevel != nil {
			riskLevel = *c.RiskLevel
		}

		return domain.Control{
			ID:                c.ID,
			Name:              c.Name,
			Description:       c.Body, // API uses "body" field
			Category:          c.Category,
			Framework:         c.Framework,
			Status:            c.Status,
			Risk:              c.Risk,
			RiskLevel:         riskLevel,
			Help:              c.Help,
			IsAutoImplemented: c.IsAutoImplemented,
			ImplementedDate:   implementedDate,
			TestedDate:        testedDate,
			Codes:             c.Codes,
		}
	case tugboatmodels.ControlDetails:
		// First convert the basic control
		control := t.ConvertControl(c.Control)

		// Add detailed fields
		control.MasterVersionNum = c.MasterVersionNum
		control.MasterControlID = c.MasterControlID
		control.OrgID = c.OrgID
		control.OrgScopeID = c.OrgScopeID
		control.DeprecationNotes = c.DeprecationNotes
		// Relationships
		control.Tags = t.convertControlTags(c.Tags)
		control.Assignees = t.convertControlAssignees(c.Assignees)
		control.AuditProjects = t.convertAuditProjects(c.AuditProjects)
		control.JiraIssues = t.convertJiraIssues(c.JiraIssues)
		control.FrameworkCodes = t.convertFrameworkCodes(c.FrameworkCodes)
		// Master content and associations
		control.MasterContent = t.convertControlMasterContent(c.MasterContent)
		control.Associations = t.convertControlAssociations(c.Associations)
		control.OrgEvidenceMetrics = t.convertControlEvidenceMetrics(c.OrgEvidenceMetrics)

		// Handle OrgScope
		if c.OrgScope != nil {
			control.OrgScope = t.convertOrgScope(c.OrgScope)
		}

		// Handle recommended evidence count (can be null or int)
		if c.RecommendedEvidenceCount != nil {
			control.RecommendedEvidenceCount = *c.RecommendedEvidenceCount
		}

		// Handle open incident count (can be null or int)
		if c.OpenIncidentCount != nil {
			control.OpenIncidentCount = *c.OpenIncidentCount
		}

		// Handle usage statistics
		if c.Usage != nil {
			control.ViewCount = c.Usage.ViewCount
			control.LastViewedAt = &c.Usage.LastViewedAt.Time
			control.DownloadCount = c.Usage.DownloadCount
			control.LastDownloadedAt = &c.Usage.LastDownloaded.Time
			control.ReferenceCount = c.Usage.ReferenceCount
			control.LastReferencedAt = &c.Usage.LastReferenced.Time
		}

		// Handle embedded evidence tasks (try both possible fields)
		var relatedEvidenceTasks []domain.EvidenceTask

		// Try evidence_tasks field first
		if c.EvidenceTasks != nil {
			relatedEvidenceTasks = t.convertEmbeddedEvidenceTasks(c.EvidenceTasks)
		}

		// Try org_evidence field as fallback
		if len(relatedEvidenceTasks) == 0 && c.OrgEvidence != nil {
			relatedEvidenceTasks = t.convertEmbeddedEvidenceTasks(c.OrgEvidence)
		}

		control.RelatedEvidenceTasks = relatedEvidenceTasks

		return control
	default:
		return domain.Control{}
	}
}

// ConvertEvidenceTask converts a Tugboat API EvidenceTask or EvidenceTaskDetails to a domain EvidenceTask
func (t *TugboatToDomain) ConvertEvidenceTask(apiTask interface{}) domain.EvidenceTask {
	switch task := apiTask.(type) {
	case tugboatmodels.EvidenceTask:
		var lastCollected, nextDue *time.Time

		// Handle last_collected (can be null or ISO date string)
		if task.LastCollected != nil && *task.LastCollected != "" {
			if parsedTime, err := time.Parse(time.RFC3339, *task.LastCollected); err == nil {
				lastCollected = &parsedTime
			}
		}

		// Handle next_due (can be null or ISO date string)
		if task.NextDue != nil && *task.NextDue != "" {
			if parsedTime, err := time.Parse(time.RFC3339, *task.NextDue); err == nil {
				nextDue = &parsedTime
			}
		}

		// Parse created_at and updated_at from ISO strings
		createdAt, _ := time.Parse(time.RFC3339, task.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, task.UpdatedAt)

		// Derive status from completed field (API doesn't return status directly)
		status := task.Status // Use if provided
		if status == "" {
			if task.Completed {
				status = "completed"
			} else {
				status = "pending"
			}
		}

		// Derive priority from collection_interval (API doesn't return priority directly)
		priority := task.Priority // Use if provided
		if priority == "" {
			switch task.CollectionInterval {
			case "year":
				priority = "low"
			case "quarter":
				priority = "medium"
			case "month", "week":
				priority = "high"
			default:
				priority = "medium"
			}
		}

		// Compute next_due from last_collected + collection_interval if not provided
		if nextDue == nil && lastCollected != nil {
			nextDue = computeNextDue(lastCollected, task.CollectionInterval)
		}

		domainTask := domain.EvidenceTask{
			ID:                 task.ID,
			Name:               task.Name,
			Description:        task.Description,
			Guidance:           "", // Not available in simplified API model
			CollectionInterval: task.CollectionInterval,
			Priority:           priority, // ← DERIVED from collection_interval
			Framework:          task.Framework,
			Status:             status, // ← DERIVED from completed field
			Completed:          task.Completed,
			LastCollected:      lastCollected,
			NextDue:            nextDue, // ← COMPUTED from last_collected + interval
			DueDaysBefore:      0,       // Not available in simplified API model
			AdHoc:              false,   // Not available in simplified API model
			Sensitive:          false,   // Not available in simplified API model
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
		}

		// Handle assignees - can be []string or []EvidenceAssignee depending on embeds
		if task.Assignees != nil {
			domainTask.Assignees = t.convertAssigneesInterface(task.Assignees)
		}

		// Handle tags - can be []string or []EvidenceTag depending on embeds
		if task.Tags != nil {
			domainTask.Tags = t.convertTagsInterface(task.Tags)
		}

		// Handle AEC status - can be string or AecStatus object depending on embeds
		if task.AecStatus != nil {
			domainTask.AecStatus = t.convertAecStatus(task.AecStatus)
		}

		// Automatically assign category, complexity, and collection type
		domainTask.AssignCategory()
		domainTask.AssignComplexityLevel()
		domainTask.AssignCollectionType()

		return domainTask
	case tugboatmodels.EvidenceTaskDetails:
		// First convert the basic task
		baseTask := t.ConvertEvidenceTask(task.EvidenceTask)

		// Extract guidance from master_content if available
		if task.MasterContent != nil && task.MasterContent.Guidance != "" {
			baseTask.Guidance = task.MasterContent.Guidance
		}

		// Convert ParentOrgControls to []string and []Control
		var controlIDs []string
		var relatedControls []domain.Control

		// Use ParentOrgControls from embeds=org_controls (preferred)
		if len(task.ParentOrgControls) > 0 {
			for _, tugboatControl := range task.ParentOrgControls {
				// Convert tugboat Control to domain Control
				domainControl := t.ConvertControl(tugboatControl)
				relatedControls = append(relatedControls, domainControl)
				controlIDs = append(controlIDs, fmt.Sprintf("%d", domainControl.ID))
			}
		} else if task.Controls != nil {
			// Fallback to legacy Controls interface{} field for backward compatibility
			if controlList, ok := task.Controls.([]interface{}); ok {
				for _, ctrl := range controlList {
					// Handle control ID strings
					if ctrlStr, ok := ctrl.(string); ok {
						controlIDs = append(controlIDs, ctrlStr)
					}
					// Handle embedded control objects
					if ctrlMap, ok := ctrl.(map[string]interface{}); ok {
						// Convert embedded control object to domain.Control
						control := t.convertEmbeddedControl(ctrlMap)
						if control.ID != 0 {
							relatedControls = append(relatedControls, control)
							// Also add the ID to controlIDs for backward compatibility
							controlIDs = append(controlIDs, fmt.Sprintf("%d", control.ID))
						}
					}
				}
			}
		}

		// Derive framework from related controls if not already set
		if baseTask.Framework == "" && len(relatedControls) > 0 {
			frameworks := make(map[string]int)
			for _, ctrl := range relatedControls {
				// Check framework field on control
				if ctrl.Framework != "" {
					frameworks[ctrl.Framework]++
				}
				// Also check framework_codes for more accurate framework detection
				for _, fc := range ctrl.FrameworkCodes {
					if fc.Framework != "" {
						frameworks[fc.Framework]++
					}
				}
			}
			// Use the most common framework
			maxCount := 0
			for fw, count := range frameworks {
				if count > maxCount {
					baseTask.Framework = fw // ← DERIVED from related controls (e.g., "SOC2")
					maxCount = count
				}
			}
		}

		// Add detailed fields
		baseTask.Controls = controlIDs
		baseTask.RelatedControls = relatedControls
		baseTask.Assignees = t.convertEvidenceAssignees(task.Assignees)
		baseTask.Tags = t.convertEvidenceTags(task.Tags)
		baseTask.AuditProjects = t.convertAuditProjectsInterface(task.AuditProjects)
		baseTask.JiraIssues = t.convertJiraIssuesInterface(task.JiraIssues)
		baseTask.FrameworkCodes = t.convertFrameworkCodes(task.FrameworkCodes)
		if task.LastRemindedAt != nil {
			baseTask.LastRemindedAt = &task.LastRemindedAt.Time
		}
		baseTask.SupportedIntegrations = t.convertSupportedIntegrations(task.SupportedIntegrations)
		baseTask.AecStatus = t.convertAecStatus(task.AecStatus)
		baseTask.SubtaskMetadata = t.convertSubtaskMetadata(task.SubtaskMetadata)
		baseTask.MasterContent = t.convertEvidenceTaskMasterContent(task.MasterContent)
		baseTask.Associations = t.convertEvidenceTaskAssociations(task.Associations)

		// Handle org scope
		if task.OrgScope != nil {
			baseTask.OrgScope = t.convertOrgScope(task.OrgScope)
		}

		// Handle open incident count (evidence tasks don't typically have this field)
		// baseTask.OpenIncidentCount is already 0 by default

		// Handle usage statistics if available
		if task.Usage != nil {
			if usageObj, ok := task.Usage.(map[string]interface{}); ok {
				if viewCount, hasViews := usageObj["view_count"].(float64); hasViews {
					baseTask.ViewCount = int(viewCount)
				}
				if downloadCount, hasDownloads := usageObj["download_count"].(float64); hasDownloads {
					baseTask.DownloadCount = int(downloadCount)
				}
				if referenceCount, hasRefs := usageObj["reference_count"].(float64); hasRefs {
					baseTask.ReferenceCount = int(referenceCount)
				}
				// Handle time fields if present
				if lastViewed, hasLastViewed := usageObj["last_viewed_at"].(string); hasLastViewed {
					if parsedTime, err := time.Parse(time.RFC3339, lastViewed); err == nil {
						baseTask.LastViewedAt = &parsedTime
					}
				}
				if lastDownloaded, hasLastDownloaded := usageObj["last_downloaded_at"].(string); hasLastDownloaded {
					if parsedTime, err := time.Parse(time.RFC3339, lastDownloaded); err == nil {
						baseTask.LastDownloadedAt = &parsedTime
					}
				}
				if lastReferenced, hasLastReferenced := usageObj["last_referenced_at"].(string); hasLastReferenced {
					if parsedTime, err := time.Parse(time.RFC3339, lastReferenced); err == nil {
						baseTask.LastReferencedAt = &parsedTime
					}
				}
			}
		}

		// Automatically assign category, complexity, and collection type
		baseTask.AssignCategory()
		baseTask.AssignComplexityLevel()
		baseTask.AssignCollectionType()

		return baseTask
	default:
		return domain.EvidenceTask{}
	}
}

// Helper conversion functions

func (t *TugboatToDomain) convertPolicyVersion(apiVersion *tugboatmodels.PolicyVersion) *domain.PolicyVersion {
	if apiVersion == nil {
		return nil
	}

	return &domain.PolicyVersion{
		ID:        apiVersion.ID,
		Version:   apiVersion.Version,
		Content:   apiVersion.Content,
		Status:    apiVersion.Status,
		CreatedAt: apiVersion.CreatedAt.Time,
		CreatedBy: apiVersion.CreatedBy,
	}
}

func (t *TugboatToDomain) convertPolicyTags(apiTags []tugboatmodels.PolicyTag) []domain.Tag {
	tags := make([]domain.Tag, len(apiTags))
	for i, apiTag := range apiTags {
		tags[i] = domain.Tag{
			ID:    apiTag.ID,
			Name:  apiTag.Name,
			Color: apiTag.Color,
		}
	}
	return tags
}

func (t *TugboatToDomain) convertPolicyAssignees(apiAssignees []tugboatmodels.PolicyAssignee) []domain.Person {
	assignees := make([]domain.Person, len(apiAssignees))
	for i, apiAssignee := range apiAssignees {
		assignees[i] = domain.Person{
			ID:         apiAssignee.ID,
			Name:       apiAssignee.Name,
			Email:      apiAssignee.Email,
			Role:       apiAssignee.Role,
			AssignedAt: &apiAssignee.AssignedAt.Time,
		}
	}
	return assignees
}

func (t *TugboatToDomain) convertPolicyReviewers(apiReviewers []tugboatmodels.PolicyReviewer) []domain.Person {
	reviewers := make([]domain.Person, len(apiReviewers))
	for i, apiReviewer := range apiReviewers {
		reviewers[i] = domain.Person{
			ID:    apiReviewer.ID,
			Name:  apiReviewer.Name,
			Email: apiReviewer.Email,
			Role:  apiReviewer.Status, // Using status as role for reviewers
		}
	}
	return reviewers
}

func (t *TugboatToDomain) convertControlTags(apiTags interface{}) []domain.Tag {
	if apiTags == nil {
		return []domain.Tag{}
	}

	// Handle interface{} type - could be array or null
	if tagArray, ok := apiTags.([]interface{}); ok {
		tags := make([]domain.Tag, 0, len(tagArray))
		for _, item := range tagArray {
			if tagMap, ok := item.(map[string]interface{}); ok {
				tag := domain.Tag{}
				if id, ok := tagMap["id"].(string); ok {
					tag.ID = id
				}
				if name, ok := tagMap["name"].(string); ok {
					tag.Name = name
				}
				if color, ok := tagMap["color"].(string); ok {
					tag.Color = color
				}
				tags = append(tags, tag)
			}
		}
		return tags
	}

	return []domain.Tag{}
}

func (t *TugboatToDomain) convertControlAssignees(apiAssignees []tugboatmodels.ControlAssignee) []domain.Person {
	assignees := make([]domain.Person, len(apiAssignees))
	for i, apiAssignee := range apiAssignees {
		assignees[i] = domain.Person{
			ID:         apiAssignee.ID.String(),
			Name:       apiAssignee.Name,
			Email:      apiAssignee.Email,
			Role:       apiAssignee.Role,
			AssignedAt: &apiAssignee.AssignedAt.Time,
		}
	}
	return assignees
}

func (t *TugboatToDomain) convertAuditProjects(apiProjects []tugboatmodels.AuditProject) []domain.AuditProject {
	projects := make([]domain.AuditProject, len(apiProjects))
	for i, apiProject := range apiProjects {
		projects[i] = domain.AuditProject{
			ID:          apiProject.ID.String(),
			Name:        apiProject.Name,
			Status:      apiProject.Status,
			StartDate:   apiProject.StartDate.Time,
			EndDate:     apiProject.EndDate.Time,
			Description: apiProject.Description,
		}
	}
	return projects
}

func (t *TugboatToDomain) convertJiraIssues(apiIssues []tugboatmodels.JiraIssue) []domain.JiraIssue {
	issues := make([]domain.JiraIssue, len(apiIssues))
	for i, apiIssue := range apiIssues {
		issues[i] = domain.JiraIssue{
			ID:        apiIssue.ID,
			Key:       apiIssue.Key,
			Summary:   apiIssue.Summary,
			Status:    apiIssue.Status,
			Priority:  apiIssue.Priority,
			IssueType: apiIssue.IssueType,
			CreatedAt: apiIssue.CreatedAt.Time,
			UpdatedAt: apiIssue.UpdatedAt.Time,
			Assignee:  apiIssue.Assignee,
			Reporter:  apiIssue.Reporter,
		}
	}
	return issues
}

func (t *TugboatToDomain) convertFrameworkCodes(apiCodes interface{}) []domain.FrameworkCode {
	if apiCodes == nil {
		return []domain.FrameworkCode{}
	}

	// The API returns framework_codes as an array of objects with:
	// - framework_id (int)
	// - framework_name (string)
	// - code (string)
	// - master_control_id (int)

	// Try to unmarshal into tugboat models first (handles type conversion)
	data, err := json.Marshal(apiCodes)
	if err != nil {
		return []domain.FrameworkCode{}
	}

	var tugboatCodes []tugboatmodels.FrameworkCode
	if err := json.Unmarshal(data, &tugboatCodes); err != nil {
		return []domain.FrameworkCode{}
	}

	// Convert to domain models
	codes := make([]domain.FrameworkCode, 0, len(tugboatCodes))
	for _, tc := range tugboatCodes {
		// Only add if we have at least code and framework
		if tc.Code != "" && tc.Framework != "" {
			codes = append(codes, domain.FrameworkCode{
				ID:        tc.ID.String(), // Convert IntOrString to string
				Code:      tc.Code,        // Direct mapping
				Framework: tc.Framework,   // framework_name from API
				Name:      tc.Name,        // Optional field
			})
		}
	}

	return codes
}

func (t *TugboatToDomain) convertOrgScope(apiScope *tugboatmodels.OrgScope) *domain.OrgScope {
	if apiScope == nil {
		return nil
	}

	return &domain.OrgScope{
		ID:          apiScope.ID,
		Name:        apiScope.Name,
		Description: apiScope.Description,
		Type:        apiScope.Type,
	}
}

func (t *TugboatToDomain) convertEvidenceAssignees(apiAssignees []tugboatmodels.EvidenceAssignee) []domain.Person {
	assignees := make([]domain.Person, len(apiAssignees))
	for i, apiAssignee := range apiAssignees {
		// Convert ID to string regardless of original type
		var idStr string
		switch id := apiAssignee.ID.(type) {
		case string:
			idStr = id
		case int:
			idStr = fmt.Sprintf("%d", id)
		case float64:
			idStr = fmt.Sprintf("%.0f", id)
		default:
			idStr = fmt.Sprintf("%v", id)
		}

		assignees[i] = domain.Person{
			ID:         idStr,
			Name:       apiAssignee.Name,
			Email:      apiAssignee.Email,
			Role:       apiAssignee.Role,
			AssignedAt: &apiAssignee.AssignedAt.Time,
		}
	}
	return assignees
}

func (t *TugboatToDomain) convertEvidenceTags(apiTags []tugboatmodels.EvidenceTag) []domain.Tag {
	tags := make([]domain.Tag, len(apiTags))
	for i, apiTag := range apiTags {
		tags[i] = domain.Tag{
			ID:    apiTag.ID,
			Name:  apiTag.Name,
			Color: apiTag.Color,
		}
	}
	return tags
}

// convertAssigneesInterface handles assignees that can be either []string or []EvidenceAssignee
func (t *TugboatToDomain) convertAssigneesInterface(apiAssignees interface{}) []domain.Person {
	if apiAssignees == nil {
		return nil
	}

	// Handle []interface{} (JSON unmarshaling)
	if assigneeList, ok := apiAssignees.([]interface{}); ok {
		var persons []domain.Person
		for _, item := range assigneeList {
			// Handle string case (just IDs, no embed)
			if assigneeStr, ok := item.(string); ok {
				persons = append(persons, domain.Person{
					ID:   assigneeStr,
					Name: "", // Name not available without embeds
				})
				continue
			}

			// Handle object case (full assignee with embeds)
			if assigneeMap, ok := item.(map[string]interface{}); ok {
				person := domain.Person{}

				// Convert ID (can be string, int, or float64)
				if id, hasID := assigneeMap["id"]; hasID {
					switch v := id.(type) {
					case string:
						person.ID = v
					case int:
						person.ID = fmt.Sprintf("%d", v)
					case float64:
						person.ID = fmt.Sprintf("%.0f", v)
					default:
						person.ID = fmt.Sprintf("%v", v)
					}
				}

				if name, hasName := assigneeMap["name"].(string); hasName {
					person.Name = name
				}
				if email, hasEmail := assigneeMap["email"].(string); hasEmail {
					person.Email = email
				}
				if role, hasRole := assigneeMap["role"].(string); hasRole {
					person.Role = role
				}

				persons = append(persons, person)
			}
		}
		return persons
	}

	return nil
}

// convertTagsInterface handles tags that can be either []string or []EvidenceTag
func (t *TugboatToDomain) convertTagsInterface(apiTags interface{}) []domain.Tag {
	if apiTags == nil {
		return nil
	}

	// Handle []interface{} (JSON unmarshaling)
	if tagList, ok := apiTags.([]interface{}); ok {
		var tags []domain.Tag
		for _, item := range tagList {
			// Handle string case (just names, no embed)
			if tagStr, ok := item.(string); ok {
				tags = append(tags, domain.Tag{
					Name: tagStr,
				})
				continue
			}

			// Handle object case (full tag with embeds)
			if tagMap, ok := item.(map[string]interface{}); ok {
				tag := domain.Tag{}
				if id, hasID := tagMap["id"].(string); hasID {
					tag.ID = id
				}
				if name, hasName := tagMap["name"].(string); hasName {
					tag.Name = name
				}
				if color, hasColor := tagMap["color"].(string); hasColor {
					tag.Color = color
				}
				tags = append(tags, tag)
			}
		}
		return tags
	}

	return nil
}

func (t *TugboatToDomain) convertSupportedIntegrations(apiIntegrations []tugboatmodels.SupportedIntegration) []domain.Integration {
	integrations := make([]domain.Integration, len(apiIntegrations))
	for i, apiIntegration := range apiIntegrations {
		integrations[i] = domain.Integration{
			ID:          apiIntegration.ID,
			Name:        apiIntegration.Name,
			Type:        apiIntegration.Type,
			Description: apiIntegration.Description,
			Enabled:     apiIntegration.Enabled,
			Config:      apiIntegration.Config,
		}
	}
	return integrations
}

func (t *TugboatToDomain) convertAecStatus(apiStatus interface{}) *domain.AecStatus {
	if apiStatus == nil {
		return nil
	}

	// Handle string case (e.g., "na", "enabled", "disabled")
	if statusStr, ok := apiStatus.(string); ok {
		return &domain.AecStatus{
			Status: statusStr,
		}
	}

	// Handle object case
	if statusObj, ok := apiStatus.(map[string]interface{}); ok {
		status := &domain.AecStatus{}
		if id, hasID := statusObj["id"].(string); hasID {
			status.ID = id
		}
		if statusVal, hasStatus := statusObj["status"].(string); hasStatus {
			status.Status = statusVal
		}
		if errMsg, hasErr := statusObj["error_message"].(string); hasErr {
			status.ErrorMessage = errMsg
		}
		if successRuns, hasSuccess := statusObj["successful_runs"].(float64); hasSuccess {
			status.SuccessfulRuns = int(successRuns)
		}
		if failedRuns, hasFailed := statusObj["failed_runs"].(float64); hasFailed {
			status.FailedRuns = int(failedRuns)
		}
		// Handle time fields
		if timeStr, ok := statusObj["last_executed"].(string); ok && timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				status.LastExecuted = &t
			}
		}
		if timeStr, ok := statusObj["next_scheduled"].(string); ok && timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				status.NextScheduled = &t
			}
		}
		if timeStr, ok := statusObj["last_successful_run"].(string); ok && timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				status.LastSuccessfulRun = &t
			}
		}
		return status
	}

	return nil
}

func (t *TugboatToDomain) convertSubtaskMetadata(apiMetadata *tugboatmodels.SubtaskMetadata) *domain.SubtaskMetadata {
	if apiMetadata == nil {
		return nil
	}

	return &domain.SubtaskMetadata{
		TotalSubtasks:     apiMetadata.TotalSubtasks,
		CompletedSubtasks: apiMetadata.CompletedSubtasks,
		PendingSubtasks:   apiMetadata.PendingSubtasks,
		OverdueSubtasks:   apiMetadata.OverdueSubtasks,
	}
}

func (t *TugboatToDomain) convertControlMasterContent(apiContent *tugboatmodels.ControlMasterContent) *domain.ControlMasterContent {
	if apiContent == nil {
		return nil
	}

	return &domain.ControlMasterContent{
		Help:        apiContent.Help,
		Guidance:    apiContent.Guidance,
		Description: apiContent.Description,
	}
}

func (t *TugboatToDomain) convertControlAssociations(apiAssociations *tugboatmodels.ControlAssociations) *domain.ControlAssociations {
	if apiAssociations == nil {
		return nil
	}

	return &domain.ControlAssociations{
		Policies:   apiAssociations.Policies,
		Procedures: apiAssociations.Procedures,
		Evidence:   apiAssociations.Evidence,
		Risks:      apiAssociations.Risks,
	}
}

func (t *TugboatToDomain) convertControlEvidenceMetrics(apiMetrics *tugboatmodels.ControlEvidenceMetrics) *domain.ControlEvidenceMetrics {
	if apiMetrics == nil {
		return nil
	}

	return &domain.ControlEvidenceMetrics{
		TotalCount:    apiMetrics.TotalCount,
		CompleteCount: apiMetrics.CompleteCount,
		OverdueCount:  apiMetrics.OverdueCount,
	}
}

func (t *TugboatToDomain) convertEvidenceTaskMasterContent(apiContent *tugboatmodels.MasterContent) *domain.EvidenceTaskMasterContent {
	if apiContent == nil {
		return nil
	}

	return &domain.EvidenceTaskMasterContent{
		Guidance:    apiContent.Guidance,
		Description: apiContent.Description,
		Help:        apiContent.Help,
	}
}

func (t *TugboatToDomain) convertEvidenceTaskAssociations(apiAssociations interface{}) *domain.EvidenceTaskAssociations {
	if apiAssociations == nil {
		return nil
	}

	// Handle interface{} type - could be array or object
	if assocObj, ok := apiAssociations.(map[string]interface{}); ok {
		associations := &domain.EvidenceTaskAssociations{}
		if controls, hasControls := assocObj["controls"].(float64); hasControls {
			associations.Controls = int(controls)
		}
		if policies, hasPolicies := assocObj["policies"].(float64); hasPolicies {
			associations.Policies = int(policies)
		}
		if procedures, hasProcedures := assocObj["procedures"].(float64); hasProcedures {
			associations.Procedures = int(procedures)
		}
		return associations
	}

	return nil
}

func (t *TugboatToDomain) convertAuditProjectsInterface(apiProjects interface{}) []domain.AuditProject {
	if apiProjects == nil {
		return []domain.AuditProject{}
	}

	// Handle interface{} type - could be array or null
	if projectArray, ok := apiProjects.([]interface{}); ok {
		projects := make([]domain.AuditProject, 0, len(projectArray))
		for _, item := range projectArray {
			if projectMap, ok := item.(map[string]interface{}); ok {
				project := domain.AuditProject{}
				if id, ok := projectMap["id"].(string); ok {
					project.ID = id
				}
				if name, ok := projectMap["name"].(string); ok {
					project.Name = name
				}
				if status, ok := projectMap["status"].(string); ok {
					project.Status = status
				}
				if description, ok := projectMap["description"].(string); ok {
					project.Description = description
				}
				// Handle dates if present
				projects = append(projects, project)
			}
		}
		return projects
	}

	return []domain.AuditProject{}
}

func (t *TugboatToDomain) convertJiraIssuesInterface(apiIssues interface{}) []domain.JiraIssue {
	if apiIssues == nil {
		return []domain.JiraIssue{}
	}

	// Handle interface{} type - could be array or null
	if issueArray, ok := apiIssues.([]interface{}); ok {
		issues := make([]domain.JiraIssue, 0, len(issueArray))
		for _, item := range issueArray {
			if issueMap, ok := item.(map[string]interface{}); ok {
				issue := domain.JiraIssue{}
				if id, ok := issueMap["id"].(string); ok {
					issue.ID = id
				}
				if key, ok := issueMap["key"].(string); ok {
					issue.Key = key
				}
				if summary, ok := issueMap["summary"].(string); ok {
					issue.Summary = summary
				}
				if status, ok := issueMap["status"].(string); ok {
					issue.Status = status
				}
				if priority, ok := issueMap["priority"].(string); ok {
					issue.Priority = priority
				}
				issues = append(issues, issue)
			}
		}
		return issues
	}

	return []domain.JiraIssue{}
}

// convertEmbeddedControl converts an embedded control object from interface{} to domain.Control
func (t *TugboatToDomain) convertEmbeddedControl(ctrlMap map[string]interface{}) domain.Control {
	control := domain.Control{}

	// Basic fields
	if id, ok := ctrlMap["id"].(float64); ok {
		control.ID = int(id)
	}
	if name, ok := ctrlMap["name"].(string); ok {
		control.Name = name
	}
	if body, ok := ctrlMap["body"].(string); ok {
		control.Description = body // API uses "body" field for description
	}
	if category, ok := ctrlMap["category"].(string); ok {
		control.Category = category
	}
	if status, ok := ctrlMap["status"].(string); ok {
		control.Status = status
	}
	if framework, ok := ctrlMap["framework"].(string); ok {
		control.Framework = framework
	}
	if risk, ok := ctrlMap["risk"].(string); ok {
		control.Risk = risk
	}
	if help, ok := ctrlMap["help"].(string); ok {
		control.Help = help
	}
	if isAuto, ok := ctrlMap["is_auto_implemented"].(bool); ok {
		control.IsAutoImplemented = isAuto
	}
	if codes, ok := ctrlMap["codes"].(string); ok {
		control.Codes = codes
	}

	// Handle risk_level (can be null or string)
	if riskLevel, ok := ctrlMap["risk_level"].(string); ok {
		control.RiskLevel = riskLevel
	}

	// Handle API metadata fields
	if masterVersionNum, ok := ctrlMap["master_version_num"].(float64); ok {
		control.MasterVersionNum = int(masterVersionNum)
	}
	if masterControlID, ok := ctrlMap["master_control_id"].(float64); ok {
		control.MasterControlID = int(masterControlID)
	}
	if orgID, ok := ctrlMap["org_id"].(float64); ok {
		control.OrgID = int(orgID)
	}
	if orgScopeID, ok := ctrlMap["org_scope_id"].(float64); ok {
		control.OrgScopeID = int(orgScopeID)
	}

	// Handle date fields (implemented_date, tested_date)
	if implementedDate, ok := ctrlMap["implemented_date"].(string); ok && implementedDate != "" {
		if parsedTime, err := time.Parse(time.RFC3339, implementedDate); err == nil {
			control.ImplementedDate = &parsedTime
		}
	}
	if testedDate, ok := ctrlMap["tested_date"].(string); ok && testedDate != "" {
		if parsedTime, err := time.Parse(time.RFC3339, testedDate); err == nil {
			control.TestedDate = &parsedTime
		}
	}

	// Handle embedded objects
	if masterContent, ok := ctrlMap["master_content"].(map[string]interface{}); ok {
		content := &domain.ControlMasterContent{}
		if guidance, hasGuidance := masterContent["guidance"].(string); hasGuidance {
			content.Guidance = guidance
		}
		if helpText, hasHelp := masterContent["help"].(string); hasHelp {
			content.Help = helpText
		}
		if description, hasDesc := masterContent["description"].(string); hasDesc {
			content.Description = description
		}
		control.MasterContent = content
	}

	// Handle framework codes
	if frameworkCodes, ok := ctrlMap["framework_codes"].([]interface{}); ok {
		control.FrameworkCodes = t.convertFrameworkCodes(frameworkCodes)
	}

	// Handle tags
	if tags, ok := ctrlMap["tags"].([]interface{}); ok {
		control.Tags = t.convertControlTags(tags)
	}

	return control
}

// convertEmbeddedEvidenceTasks converts embedded evidence task objects from interface{} to []domain.EvidenceTask
func (t *TugboatToDomain) convertEmbeddedEvidenceTasks(evidenceData interface{}) []domain.EvidenceTask {
	if evidenceData == nil {
		return []domain.EvidenceTask{}
	}

	// Handle interface{} type - could be array or null
	if evidenceArray, ok := evidenceData.([]interface{}); ok {
		tasks := make([]domain.EvidenceTask, 0, len(evidenceArray))
		for _, item := range evidenceArray {
			if taskMap, ok := item.(map[string]interface{}); ok {
				task := t.convertEmbeddedEvidenceTask(taskMap)
				if task.ID != 0 {
					tasks = append(tasks, task)
				}
			}
		}
		return tasks
	}

	return []domain.EvidenceTask{}
}

// convertEmbeddedEvidenceTask converts an embedded evidence task object from interface{} to domain.EvidenceTask
func (t *TugboatToDomain) convertEmbeddedEvidenceTask(taskMap map[string]interface{}) domain.EvidenceTask {
	task := domain.EvidenceTask{}

	// Basic fields
	if id, ok := taskMap["id"].(float64); ok {
		task.ID = int(id)
	}
	if name, ok := taskMap["name"].(string); ok {
		task.Name = name
	}
	if description, ok := taskMap["description"].(string); ok {
		task.Description = description
	}
	if collectionInterval, ok := taskMap["collection_interval"].(string); ok {
		task.CollectionInterval = collectionInterval
	}
	if priority, ok := taskMap["priority"].(string); ok {
		task.Priority = priority
	}
	if framework, ok := taskMap["framework"].(string); ok {
		task.Framework = framework
	}
	if status, ok := taskMap["status"].(string); ok {
		task.Status = status
	}
	if completed, ok := taskMap["completed"].(bool); ok {
		task.Completed = completed
	}

	// Handle date fields
	if lastCollected, ok := taskMap["last_collected"].(string); ok && lastCollected != "" {
		if parsedTime, err := time.Parse(time.RFC3339, lastCollected); err == nil {
			task.LastCollected = &parsedTime
		}
	}
	if nextDue, ok := taskMap["next_due"].(string); ok && nextDue != "" {
		if parsedTime, err := time.Parse(time.RFC3339, nextDue); err == nil {
			task.NextDue = &parsedTime
		}
	}
	if createdAt, ok := taskMap["created_at"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, createdAt); err == nil {
			task.CreatedAt = parsedTime
		}
	}
	if updatedAt, ok := taskMap["updated_at"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			task.UpdatedAt = parsedTime
		}
	}

	// Handle API metadata fields
	if masterVersionNum, ok := taskMap["master_version_num"].(float64); ok {
		task.MasterVersionNum = int(masterVersionNum)
	}
	if masterEvidenceID, ok := taskMap["master_evidence_id"].(float64); ok {
		task.MasterEvidenceID = int(masterEvidenceID)
	}
	if orgID, ok := taskMap["org_id"].(float64); ok {
		task.OrgID = int(orgID)
	}
	if orgScopeID, ok := taskMap["org_scope_id"].(float64); ok {
		task.OrgScopeID = int(orgScopeID)
	}

	// Handle embedded objects
	if masterContent, ok := taskMap["master_content"].(map[string]interface{}); ok {
		content := &domain.EvidenceTaskMasterContent{}
		if guidance, hasGuidance := masterContent["guidance"].(string); hasGuidance {
			content.Guidance = guidance
			task.Guidance = guidance // Also set the top-level guidance field
		}
		if description, hasDesc := masterContent["description"].(string); hasDesc {
			content.Description = description
		}
		if help, hasHelp := masterContent["help"].(string); hasHelp {
			content.Help = help
		}
		task.MasterContent = content
	}

	// Handle controls (these would be control IDs that relate back to the parent control)
	if controls, ok := taskMap["controls"].([]interface{}); ok {
		controlIDs := make([]string, 0, len(controls))
		for _, ctrl := range controls {
			if ctrlStr, ok := ctrl.(string); ok {
				controlIDs = append(controlIDs, ctrlStr)
			} else if ctrlID, ok := ctrl.(float64); ok {
				controlIDs = append(controlIDs, fmt.Sprintf("%.0f", ctrlID))
			}
		}
		task.Controls = controlIDs
	}

	return task
}

// computeNextDue calculates the next due date based on last collected date and collection interval
func computeNextDue(lastCollected *time.Time, interval string) *time.Time {
	if lastCollected == nil {
		return nil
	}

	var nextDue time.Time
	switch interval {
	case "year":
		nextDue = lastCollected.AddDate(1, 0, 0)
	case "quarter":
		nextDue = lastCollected.AddDate(0, 3, 0)
	case "month":
		nextDue = lastCollected.AddDate(0, 1, 0)
	case "week":
		nextDue = lastCollected.AddDate(0, 0, 7)
	case "day":
		nextDue = lastCollected.AddDate(0, 0, 1)
	default:
		// Unknown interval, don't compute
		return nil
	}

	return &nextDue
}
