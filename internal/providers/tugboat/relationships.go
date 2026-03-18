// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"context"
	"fmt"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
)

// Compile-time assertion that TugboatDataProvider implements RelationshipQuerier.
var _ interfaces.RelationshipQuerier = (*TugboatDataProvider)(nil)

// GetEvidenceTasksByControl returns evidence tasks linked to a control.
// Uses the Tugboat API's native filtering: GET /api/org_evidence/?controls={controlID}
func (p *TugboatDataProvider) GetEvidenceTasksByControl(ctx context.Context, controlID string) ([]domain.EvidenceTask, error) {
	apiTasks, err := p.client.GetEvidenceTasksByControl(ctx, controlID, p.orgID)
	if err != nil {
		return nil, fmt.Errorf("get tasks by control %s: %w", controlID, err)
	}

	tasks := make([]domain.EvidenceTask, 0, len(apiTasks))
	for _, at := range apiTasks {
		tasks = append(tasks, p.adapter.ConvertEvidenceTask(at))
	}
	return tasks, nil
}

// GetControlsByPolicy returns controls implementing a policy.
// Tugboat doesn't have a direct API for this. We fetch the policy details
// (which include association counts) and then fetch all controls and filter
// by those that reference this policy. For now, returns an empty slice with
// a logged warning since the API doesn't expose control-policy relationships
// in a queryable way.
func (p *TugboatDataProvider) GetControlsByPolicy(ctx context.Context, policyID string) ([]domain.Control, error) {
	// Tugboat API doesn't have a /api/org_control/?policy={id} filter.
	// We'd need to fetch ALL controls and check their associations, which
	// is expensive. Return empty for now — callers should use local storage
	// for this relationship.
	p.logger.Info("GetControlsByPolicy: Tugboat API does not support direct policy→control queries; use local storage for this relationship",
		logger.Field{Key: "policy_id", Value: policyID})
	return nil, nil
}

// GetPoliciesByControl returns policies that a control implements.
// Similar to GetControlsByPolicy — Tugboat doesn't expose this as a
// filterable API. Returns empty with a logged note.
func (p *TugboatDataProvider) GetPoliciesByControl(ctx context.Context, controlID string) ([]domain.Policy, error) {
	p.logger.Info("GetPoliciesByControl: Tugboat API does not support direct control→policy queries; use local storage for this relationship",
		logger.Field{Key: "control_id", Value: controlID})
	return nil, nil
}
