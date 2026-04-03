// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/grctool/grctool/internal/domain"
)

// SyncScope defines include/exclude filters for selective sync.
// Per FEAT-002 US-5, operators can control which entities are synced to Drive.
type SyncScope struct {
	// Policies scope
	Policies EntityScope `json:"policies" yaml:"policies"`
	// Controls scope
	Controls EntityScope `json:"controls" yaml:"controls"`
	// EvidenceTasks scope
	EvidenceTasks EntityScope `json:"evidence_tasks" yaml:"evidence_tasks"`
}

// EntityScope defines include/exclude patterns for a single entity type.
type EntityScope struct {
	// Enabled controls whether this entity type is synced at all.
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Include patterns (glob-style on reference ID). Empty means include all (when enabled).
	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	// Exclude patterns (glob-style on reference ID). Applied after include.
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	// TagsInclude filters to entities with at least one of these tags. Empty means no tag filter.
	TagsInclude []string `json:"tags_include,omitempty" yaml:"tags_include,omitempty"`
	// TagsExclude excludes entities with any of these tags.
	TagsExclude []string `json:"tags_exclude,omitempty" yaml:"tags_exclude,omitempty"`
}

// DefaultSyncScope returns the default scope where all entity types are enabled
// with no filters.
func DefaultSyncScope() SyncScope {
	return SyncScope{
		Policies:      EntityScope{Enabled: true},
		Controls:      EntityScope{Enabled: true},
		EvidenceTasks: EntityScope{Enabled: true},
	}
}

// PolicyInScope returns true if the given policy passes the scope filters.
func (s *SyncScope) PolicyInScope(policy *domain.Policy) bool {
	if !s.Policies.Enabled {
		return false
	}
	tags := tagNames(policy.Tags)
	return s.Policies.matches(policy.ReferenceID, tags)
}

// ControlInScope returns true if the given control passes the scope filters.
func (s *SyncScope) ControlInScope(control *domain.Control) bool {
	if !s.Controls.Enabled {
		return false
	}
	tags := tagNames(control.Tags)
	return s.Controls.matches(control.ReferenceID, tags)
}

// EvidenceTaskInScope returns true if the given task passes the scope filters.
func (s *SyncScope) EvidenceTaskInScope(task *domain.EvidenceTask) bool {
	if !s.EvidenceTasks.Enabled {
		return false
	}
	tags := tagNames(task.Tags)
	return s.EvidenceTasks.matches(task.ReferenceID, tags)
}

// matches checks if a reference ID and tag set pass the include/exclude filters.
func (es *EntityScope) matches(refID string, tags []string) bool {
	// Include filter: if patterns are specified, refID must match at least one.
	if len(es.Include) > 0 {
		if !matchesAnyPattern(refID, es.Include) {
			return false
		}
	}

	// Exclude filter: if refID matches any exclude pattern, reject.
	if len(es.Exclude) > 0 {
		if matchesAnyPattern(refID, es.Exclude) {
			return false
		}
	}

	// Tag include: entity must have at least one of the specified tags.
	if len(es.TagsInclude) > 0 {
		if !hasAnyTag(tags, es.TagsInclude) {
			return false
		}
	}

	// Tag exclude: entity must not have any of the specified tags.
	if len(es.TagsExclude) > 0 {
		if hasAnyTag(tags, es.TagsExclude) {
			return false
		}
	}

	return true
}

// ValidatePatterns checks that all Include and Exclude glob patterns in the
// EntityScope are valid filepath.Match patterns. Returns an error describing
// the first invalid pattern found, or nil if all patterns are valid.
func (es *EntityScope) ValidatePatterns() error {
	for _, p := range es.Include {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("invalid include pattern %q: %w", p, err)
		}
	}
	for _, p := range es.Exclude {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}
	return nil
}

// ValidatePatterns checks all glob patterns across all entity scopes.
// Returns an error describing the first invalid pattern found.
func (s *SyncScope) ValidatePatterns() error {
	for name, es := range map[string]*EntityScope{
		"policies":       &s.Policies,
		"controls":       &s.Controls,
		"evidence_tasks": &s.EvidenceTasks,
	} {
		if err := es.ValidatePatterns(); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

// matchesAnyPattern checks if s matches any of the given glob patterns.
// Patterns are assumed to be pre-validated via ValidatePatterns.
func matchesAnyPattern(s string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, s); matched {
			return true
		}
	}
	return false
}

// hasAnyTag checks if any tag in entityTags appears in filterTags (case-insensitive).
func hasAnyTag(entityTags, filterTags []string) bool {
	for _, et := range entityTags {
		for _, ft := range filterTags {
			if strings.EqualFold(et, ft) {
				return true
			}
		}
	}
	return false
}

// tagNames extracts tag name strings from domain.Tag slices.
func tagNames(tags []domain.Tag) []string {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	return names
}
