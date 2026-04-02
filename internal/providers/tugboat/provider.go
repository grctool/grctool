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

package tugboat

import (
	"context"
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
)

// Compile-time assertion that TugboatDataProvider implements interfaces.DataProvider.
var _ interfaces.DataProvider = (*TugboatDataProvider)(nil)

// TugboatDataProvider implements interfaces.DataProvider by wrapping
// the existing tugboat.Client and adapters.TugboatToDomain.
type TugboatDataProvider struct {
	client  *tugboatclient.Client
	adapter *adapters.TugboatToDomain
	orgID   string
	logger  logger.Logger
}

// NewTugboatDataProvider creates a provider wrapping an existing client.
func NewTugboatDataProvider(client *tugboatclient.Client, adapter *adapters.TugboatToDomain, orgID string, log logger.Logger) *TugboatDataProvider {
	return &TugboatDataProvider{
		client:  client,
		adapter: adapter,
		orgID:   orgID,
		logger:  log,
	}
}

// Name returns the unique identifier for this provider.
func (p *TugboatDataProvider) Name() string {
	return "tugboat"
}

// Capabilities reports that Tugboat supports all entity types (read-only).
func (p *TugboatDataProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsPolicies:      true,
		SupportsControls:      true,
		SupportsEvidenceTasks: true,
		SupportsWrite:         false,
		SupportsChangeDetect:  false,
	}
}

// TestConnection verifies the Tugboat API is reachable by fetching
// a minimal page of policies.
func (p *TugboatDataProvider) TestConnection(ctx context.Context) error {
	_, err := p.client.GetPolicies(ctx, &tugboatclient.PolicyListOptions{
		Org:      p.orgID,
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		return fmt.Errorf("tugboat connection test failed: %w", err)
	}
	return nil
}

// ensureSyncMetadata initializes SyncMetadata on an entity if nil.
func ensureSyncMetadata(sm **domain.SyncMetadata) {
	if *sm == nil {
		*sm = &domain.SyncMetadata{}
	}
	if (*sm).ContentHash == nil {
		(*sm).ContentHash = make(map[string]string)
	}
	if (*sm).LastSyncTime == nil {
		(*sm).LastSyncTime = make(map[string]time.Time)
	}
}

// setPolicyContentHash computes and sets the content hash on a policy.
func (p *TugboatDataProvider) setPolicyContentHash(policy *domain.Policy) {
	ensureSyncMetadata(&policy.SyncMetadata)
	hash, err := domain.ComputeContentHash(policy)
	if err != nil {
		p.logger.Warn("failed to compute content hash for policy",
			logger.String("id", policy.ID), logger.String("error", err.Error()))
		return
	}
	policy.SyncMetadata.ContentHash["tugboat"] = hash
	policy.SyncMetadata.LastSyncTime["tugboat"] = time.Now()
}

// setControlContentHash computes and sets the content hash on a control.
func (p *TugboatDataProvider) setControlContentHash(control *domain.Control) {
	ensureSyncMetadata(&control.SyncMetadata)
	hash, err := domain.ComputeContentHash(control)
	if err != nil {
		p.logger.Warn("failed to compute content hash for control",
			logger.String("id", control.ID), logger.String("error", err.Error()))
		return
	}
	control.SyncMetadata.ContentHash["tugboat"] = hash
	control.SyncMetadata.LastSyncTime["tugboat"] = time.Now()
}

// setTaskContentHash computes and sets the content hash on an evidence task.
func (p *TugboatDataProvider) setTaskContentHash(task *domain.EvidenceTask) {
	ensureSyncMetadata(&task.SyncMetadata)
	hash, err := domain.ComputeContentHash(task)
	if err != nil {
		p.logger.Warn("failed to compute content hash for evidence task",
			logger.String("id", task.ID), logger.String("error", err.Error()))
		return
	}
	task.SyncMetadata.ContentHash["tugboat"] = hash
	task.SyncMetadata.LastSyncTime["tugboat"] = time.Now()
}

// ListPolicies wraps client.GetPolicies with ListOptions to PolicyListOptions translation.
func (p *TugboatDataProvider) ListPolicies(ctx context.Context, opts interfaces.ListOptions) ([]domain.Policy, int, error) {
	clientOpts := &tugboatclient.PolicyListOptions{
		Org:       p.orgID,
		Page:      opts.Page,
		PageSize:  opts.PageSize,
		Framework: opts.Framework,
		Status:    opts.Status,
	}

	p.logger.Debug("ListPolicies called",
		logger.Int("page", opts.Page),
		logger.Int("page_size", opts.PageSize),
		logger.String("framework", opts.Framework))

	results, err := p.client.GetPolicies(ctx, clientOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list policies: %w", err)
	}

	policies := make([]domain.Policy, 0, len(results))
	for _, r := range results {
		pol := p.adapter.ConvertPolicy(r)
		p.setPolicyContentHash(&pol)
		policies = append(policies, pol)
	}

	// The underlying client does not expose the total count from pagination
	// metadata, so we return len(results) as the count for this page.
	return policies, len(policies), nil
}

// GetPolicy retrieves a single policy by its provider-native ID.
func (p *TugboatDataProvider) GetPolicy(ctx context.Context, id string) (*domain.Policy, error) {
	p.logger.Debug("GetPolicy called", logger.String("id", id))

	details, err := p.client.GetPolicyDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy %s: %w", id, err)
	}

	policy := p.adapter.ConvertPolicy(*details)
	p.setPolicyContentHash(&policy)
	return &policy, nil
}

// ListControls wraps client.GetControls with ListOptions to ControlListOptions translation.
func (p *TugboatDataProvider) ListControls(ctx context.Context, opts interfaces.ListOptions) ([]domain.Control, int, error) {
	clientOpts := &tugboatclient.ControlListOptions{
		Org:       p.orgID,
		Page:      opts.Page,
		PageSize:  opts.PageSize,
		Framework: opts.Framework,
		Status:    opts.Status,
	}

	p.logger.Debug("ListControls called",
		logger.Int("page", opts.Page),
		logger.Int("page_size", opts.PageSize),
		logger.String("framework", opts.Framework))

	results, err := p.client.GetControls(ctx, clientOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list controls: %w", err)
	}

	controls := make([]domain.Control, 0, len(results))
	for _, r := range results {
		ctrl := p.adapter.ConvertControl(r)
		p.setControlContentHash(&ctrl)
		controls = append(controls, ctrl)
	}

	return controls, len(controls), nil
}

// GetControl retrieves a single control by its provider-native ID.
func (p *TugboatDataProvider) GetControl(ctx context.Context, id string) (*domain.Control, error) {
	p.logger.Debug("GetControl called", logger.String("id", id))

	details, err := p.client.GetControlDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get control %s: %w", id, err)
	}

	control := p.adapter.ConvertControl(*details)
	p.setControlContentHash(&control)
	return &control, nil
}

// ListEvidenceTasks wraps client.GetEvidenceTasks with ListOptions translation.
func (p *TugboatDataProvider) ListEvidenceTasks(ctx context.Context, opts interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	clientOpts := &tugboatclient.EvidenceTaskListOptions{
		Org:      p.orgID,
		Page:     opts.Page,
		PageSize: opts.PageSize,
	}

	p.logger.Debug("ListEvidenceTasks called",
		logger.Int("page", opts.Page),
		logger.Int("page_size", opts.PageSize))

	results, err := p.client.GetEvidenceTasks(ctx, clientOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list evidence tasks: %w", err)
	}

	tasks := make([]domain.EvidenceTask, 0, len(results))
	for _, r := range results {
		task := p.adapter.ConvertEvidenceTask(r)
		p.setTaskContentHash(&task)
		tasks = append(tasks, task)
	}

	return tasks, len(tasks), nil
}

// GetEvidenceTask retrieves a single evidence task by its provider-native ID.
func (p *TugboatDataProvider) GetEvidenceTask(ctx context.Context, id string) (*domain.EvidenceTask, error) {
	p.logger.Debug("GetEvidenceTask called", logger.String("id", id))

	details, err := p.client.GetEvidenceTaskDetails(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence task %s: %w", id, err)
	}

	task := p.adapter.ConvertEvidenceTask(*details)
	p.setTaskContentHash(&task)
	return &task, nil
}
