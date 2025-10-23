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
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// IndexQuery provides fast query operations on persisted Terraform security indices
type IndexQuery struct {
	index *PersistedIndex
}

// NewIndexQuery creates a new index query instance
func NewIndexQuery(index *PersistedIndex) *IndexQuery {
	return &IndexQuery{
		index: index,
	}
}

// QueryResult represents the result of a query operation
type QueryResult struct {
	Resources []IndexedResource      `json:"resources"`
	Count     int                    `json:"count"`
	QueryTime time.Duration          `json:"query_time_ms"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ByControl queries resources by SOC2 control codes
func (iq *IndexQuery) ByControl(controlCodes ...string) *QueryResult {
	start := time.Now()
	var resources []IndexedResource
	seen := make(map[string]bool)

	for _, control := range controlCodes {
		if controlResources, exists := iq.index.Index.ControlMapping[control]; exists {
			for _, res := range controlResources {
				if !seen[res.ResourceID] {
					resources = append(resources, res)
					seen[res.ResourceID] = true
				}
			}
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":    "by_control",
			"control_codes": controlCodes,
		},
	}
}

// ByAttribute queries resources by security attributes
func (iq *IndexQuery) ByAttribute(attributes ...string) *QueryResult {
	start := time.Now()
	var resources []IndexedResource
	attrSet := make(map[string]bool)

	for _, attr := range attributes {
		attrSet[attr] = true
	}

	for _, resource := range iq.index.Index.IndexedResources {
		// Check if resource has any of the requested attributes
		for _, resAttr := range resource.SecurityAttributes {
			if attrSet[resAttr] {
				resources = append(resources, resource)
				break
			}
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type": "by_attribute",
			"attributes": attributes,
		},
	}
}

// ByResourceType queries resources by their Terraform resource type
func (iq *IndexQuery) ByResourceType(resourceTypes ...string) *QueryResult {
	start := time.Now()
	var resources []IndexedResource
	typeSet := make(map[string]bool)

	for _, rt := range resourceTypes {
		typeSet[rt] = true
	}

	for _, resource := range iq.index.Index.IndexedResources {
		if typeSet[resource.ResourceType] {
			resources = append(resources, resource)
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":     "by_resource_type",
			"resource_types": resourceTypes,
		},
	}
}

// ByEnvironment queries resources by environment (prod, staging, dev, etc.)
func (iq *IndexQuery) ByEnvironment(environments ...string) *QueryResult {
	start := time.Now()
	var resources []IndexedResource
	envSet := make(map[string]bool)

	for _, env := range environments {
		envSet[env] = true
	}

	for _, resource := range iq.index.Index.IndexedResources {
		if envSet[resource.Environment] {
			resources = append(resources, resource)
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":   "by_environment",
			"environments": environments,
		},
	}
}

// ByRiskLevel queries resources by risk level (high, medium, low)
func (iq *IndexQuery) ByRiskLevel(riskLevels ...string) *QueryResult {
	start := time.Now()
	var resources []IndexedResource
	riskSet := make(map[string]bool)

	for _, risk := range riskLevels {
		riskSet[risk] = true
	}

	for _, resource := range iq.index.Index.IndexedResources {
		if riskSet[resource.RiskLevel] {
			resources = append(resources, resource)
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":  "by_risk_level",
			"risk_levels": riskLevels,
		},
	}
}

// ByComplianceStatus queries resources by compliance status (compliant, partially_compliant, non_compliant)
func (iq *IndexQuery) ByComplianceStatus(statuses ...string) *QueryResult {
	start := time.Now()
	var resources []IndexedResource
	statusSet := make(map[string]bool)

	for _, status := range statuses {
		statusSet[status] = true
	}

	for _, resource := range iq.index.Index.IndexedResources {
		if statusSet[resource.ComplianceStatus] {
			resources = append(resources, resource)
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":        "by_compliance_status",
			"compliance_status": statuses,
		},
	}
}

// ByEvidenceTask queries resources relevant to a specific evidence task
// This maps evidence tasks to their required controls and attributes
func (iq *IndexQuery) ByEvidenceTask(evidenceTaskRef string) *QueryResult {
	start := time.Now()

	// Map evidence tasks to their control requirements
	// This should eventually be loaded from configuration
	evidenceTaskMapping := map[string]struct {
		Controls   []string
		Attributes []string
	}{
		"ET-001": {
			Controls:   []string{"CC6.1", "CC6.6", "CC6.8", "CC7.2", "SO2"},
			Attributes: []string{"encryption", "network_security", "monitoring", "access_control", "high_availability"},
		},
		"ET-021": {
			Controls:   []string{"CC6.8"},
			Attributes: []string{"encryption", "ssl_tls"},
		},
		"ET-023": {
			Controls:   []string{"CC6.8"},
			Attributes: []string{"encryption", "data_protection"},
		},
		"ET-047": {
			Controls:   []string{"CC6.1", "CC6.3"},
			Attributes: []string{"access_control", "identity_management"},
		},
		"ET-071": {
			Controls:   []string{"CC6.6", "CC7.1"},
			Attributes: []string{"network_security", "access_control"},
		},
	}

	taskMapping, exists := evidenceTaskMapping[evidenceTaskRef]
	if !exists {
		return &QueryResult{
			Resources: []IndexedResource{},
			Count:     0,
			QueryTime: time.Since(start),
			Metadata: map[string]interface{}{
				"query_type":        "by_evidence_task",
				"evidence_task_ref": evidenceTaskRef,
				"error":             fmt.Sprintf("unknown evidence task: %s", evidenceTaskRef),
			},
		}
	}

	// Combine control and attribute queries
	seen := make(map[string]bool)
	var resources []IndexedResource

	// Get resources by controls
	for _, control := range taskMapping.Controls {
		if controlResources, exists := iq.index.Index.ControlMapping[control]; exists {
			for _, res := range controlResources {
				if !seen[res.ResourceID] {
					resources = append(resources, res)
					seen[res.ResourceID] = true
				}
			}
		}
	}

	// Add resources by attributes that weren't already included
	attrSet := make(map[string]bool)
	for _, attr := range taskMapping.Attributes {
		attrSet[attr] = true
	}

	for _, resource := range iq.index.Index.IndexedResources {
		if seen[resource.ResourceID] {
			continue
		}

		for _, resAttr := range resource.SecurityAttributes {
			if attrSet[resAttr] {
				resources = append(resources, resource)
				seen[resource.ResourceID] = true
				break
			}
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":        "by_evidence_task",
			"evidence_task_ref": evidenceTaskRef,
			"controls":          taskMapping.Controls,
			"attributes":        taskMapping.Attributes,
		},
	}
}

// Intersect returns resources that appear in all query results (AND operation)
func Intersect(results ...*QueryResult) *QueryResult {
	start := time.Now()

	if len(results) == 0 {
		return &QueryResult{
			Resources: []IndexedResource{},
			Count:     0,
			QueryTime: time.Since(start),
		}
	}

	if len(results) == 1 {
		return results[0]
	}

	// Build set from first result
	resourceSet := make(map[string]IndexedResource)
	for _, res := range results[0].Resources {
		resourceSet[res.ResourceID] = res
	}

	// Intersect with remaining results
	for i := 1; i < len(results); i++ {
		nextSet := make(map[string]bool)
		for _, res := range results[i].Resources {
			nextSet[res.ResourceID] = true
		}

		// Remove resources not in next set
		for id := range resourceSet {
			if !nextSet[id] {
				delete(resourceSet, id)
			}
		}
	}

	// Convert to slice
	var resources []IndexedResource
	for _, res := range resourceSet {
		resources = append(resources, res)
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":   "intersect",
			"input_counts": extractCounts(results),
		},
	}
}

// Union returns resources that appear in any query result (OR operation)
func Union(results ...*QueryResult) *QueryResult {
	start := time.Now()

	resourceSet := make(map[string]IndexedResource)
	for _, result := range results {
		for _, res := range result.Resources {
			resourceSet[res.ResourceID] = res
		}
	}

	var resources []IndexedResource
	for _, res := range resourceSet {
		resources = append(resources, res)
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":   "union",
			"input_counts": extractCounts(results),
		},
	}
}

// Exclude returns resources in the first result that are not in the second (A - B)
func Exclude(base *QueryResult, exclude *QueryResult) *QueryResult {
	start := time.Now()

	excludeSet := make(map[string]bool)
	for _, res := range exclude.Resources {
		excludeSet[res.ResourceID] = true
	}

	var resources []IndexedResource
	for _, res := range base.Resources {
		if !excludeSet[res.ResourceID] {
			resources = append(resources, res)
		}
	}

	return &QueryResult{
		Resources: resources,
		Count:     len(resources),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":    "exclude",
			"base_count":    base.Count,
			"exclude_count": exclude.Count,
		},
	}
}

// Filter applies a custom filter function to a query result
func (qr *QueryResult) Filter(filterFunc func(IndexedResource) bool) *QueryResult {
	start := time.Now()

	var filtered []IndexedResource
	for _, res := range qr.Resources {
		if filterFunc(res) {
			filtered = append(filtered, res)
		}
	}

	return &QueryResult{
		Resources: filtered,
		Count:     len(filtered),
		QueryTime: time.Since(start),
		Metadata: map[string]interface{}{
			"query_type":     "filter",
			"original_count": qr.Count,
		},
	}
}

// Sort sorts resources by a given field
func (qr *QueryResult) Sort(field string, ascending bool) *QueryResult {
	sorted := make([]IndexedResource, len(qr.Resources))
	copy(sorted, qr.Resources)

	sort.Slice(sorted, func(i, j int) bool {
		var less bool

		switch field {
		case "resource_id":
			less = sorted[i].ResourceID < sorted[j].ResourceID
		case "resource_type":
			less = sorted[i].ResourceType < sorted[j].ResourceType
		case "environment":
			less = sorted[i].Environment < sorted[j].Environment
		case "risk_level":
			// Custom ordering: high > medium > low
			riskOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
			less = riskOrder[sorted[i].RiskLevel] < riskOrder[sorted[j].RiskLevel]
		case "compliance_status":
			// Custom ordering: non_compliant > partially_compliant > compliant
			statusOrder := map[string]int{
				"non_compliant":       3,
				"partially_compliant": 2,
				"compliant":           1,
			}
			less = statusOrder[sorted[i].ComplianceStatus] < statusOrder[sorted[j].ComplianceStatus]
		case "file_path":
			less = sorted[i].FilePath < sorted[j].FilePath
		default:
			// Default to resource ID
			less = sorted[i].ResourceID < sorted[j].ResourceID
		}

		if !ascending {
			less = !less
		}

		return less
	})

	return &QueryResult{
		Resources: sorted,
		Count:     len(sorted),
		QueryTime: qr.QueryTime,
		Metadata:  qr.Metadata,
	}
}

// Limit limits the number of results returned
func (qr *QueryResult) Limit(n int) *QueryResult {
	if n >= len(qr.Resources) {
		return qr
	}

	return &QueryResult{
		Resources: qr.Resources[:n],
		Count:     n,
		QueryTime: qr.QueryTime,
		Metadata: map[string]interface{}{
			"query_type":     "limit",
			"original_count": qr.Count,
			"limit":          n,
		},
	}
}

// AggregateByRisk aggregates resources by risk level
func (qr *QueryResult) AggregateByRisk() map[string]int {
	agg := make(map[string]int)
	for _, res := range qr.Resources {
		agg[res.RiskLevel]++
	}
	return agg
}

// AggregateByEnvironment aggregates resources by environment
func (qr *QueryResult) AggregateByEnvironment() map[string]int {
	agg := make(map[string]int)
	for _, res := range qr.Resources {
		agg[res.Environment]++
	}
	return agg
}

// AggregateByComplianceStatus aggregates resources by compliance status
func (qr *QueryResult) AggregateByComplianceStatus() map[string]int {
	agg := make(map[string]int)
	for _, res := range qr.Resources {
		agg[res.ComplianceStatus]++
	}
	return agg
}

// AggregateByResourceType aggregates resources by resource type
func (qr *QueryResult) AggregateByResourceType() map[string]int {
	agg := make(map[string]int)
	for _, res := range qr.Resources {
		agg[res.ResourceType]++
	}
	return agg
}

// Summary generates a text summary of the query result
func (qr *QueryResult) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Query Results: %d resources (%.2fms)\n\n", qr.Count, float64(qr.QueryTime.Microseconds())/1000.0))

	if qr.Count == 0 {
		sb.WriteString("No resources found.\n")
		return sb.String()
	}

	// Risk distribution
	riskAgg := qr.AggregateByRisk()
	sb.WriteString("Risk Distribution:\n")
	for _, risk := range []string{"high", "medium", "low"} {
		if count, exists := riskAgg[risk]; exists {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", cases.Title(language.English).String(risk), count))
		}
	}
	sb.WriteString("\n")

	// Compliance distribution
	complianceAgg := qr.AggregateByComplianceStatus()
	sb.WriteString("Compliance Status:\n")
	for status, count := range complianceAgg {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", status, count))
	}
	sb.WriteString("\n")

	// Environment distribution
	envAgg := qr.AggregateByEnvironment()
	sb.WriteString("Environment Distribution:\n")
	for env, count := range envAgg {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", env, count))
	}

	return sb.String()
}

// extractCounts is a helper to extract counts from multiple query results
func extractCounts(results []*QueryResult) []int {
	counts := make([]int, len(results))
	for i, r := range results {
		counts[i] = r.Count
	}
	return counts
}
