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

package testhelpers

import (
	"context"
	"fmt"
	"sort"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
)

// Compile-time assertion that StubDataProvider implements DataProvider.
var _ interfaces.DataProvider = (*StubDataProvider)(nil)

// StubDataProvider is a reusable in-memory implementation of the DataProvider
// interface for use in tests across packages. Set ConnError to simulate
// connection failures.
type StubDataProvider struct {
	ProviderName string
	Policies     map[string]*domain.Policy
	Controls     map[string]*domain.Control
	Tasks        map[string]*domain.EvidenceTask
	ConnError    error // Set to simulate connection failure
}

// NewStubDataProvider creates a StubDataProvider with the given name and empty maps.
func NewStubDataProvider(name string) *StubDataProvider {
	return &StubDataProvider{
		ProviderName: name,
		Policies:     make(map[string]*domain.Policy),
		Controls:     make(map[string]*domain.Control),
		Tasks:        make(map[string]*domain.EvidenceTask),
	}
}

func (s *StubDataProvider) Name() string {
	return s.ProviderName
}

// Capabilities returns full support for all entity types (read-only).
func (s *StubDataProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsPolicies:      true,
		SupportsControls:      true,
		SupportsEvidenceTasks: true,
		SupportsWrite:         false,
		SupportsChangeDetect:  false,
	}
}

func (s *StubDataProvider) TestConnection(_ context.Context) error {
	return s.ConnError
}

func (s *StubDataProvider) ListPolicies(_ context.Context, opts interfaces.ListOptions) ([]domain.Policy, int, error) {
	if s.ConnError != nil {
		return nil, 0, s.ConnError
	}
	all := s.allPolicies(opts)
	total := len(all)
	paged := paginate(total, opts.Page, opts.PageSize)
	if paged.start >= total {
		return nil, total, nil
	}
	return all[paged.start:paged.end], total, nil
}

func (s *StubDataProvider) GetPolicy(_ context.Context, id string) (*domain.Policy, error) {
	if s.ConnError != nil {
		return nil, s.ConnError
	}
	p, ok := s.Policies[id]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", id)
	}
	return p, nil
}

func (s *StubDataProvider) ListControls(_ context.Context, opts interfaces.ListOptions) ([]domain.Control, int, error) {
	if s.ConnError != nil {
		return nil, 0, s.ConnError
	}
	all := s.allControls(opts)
	total := len(all)
	paged := paginate(total, opts.Page, opts.PageSize)
	if paged.start >= total {
		return nil, total, nil
	}
	return all[paged.start:paged.end], total, nil
}

func (s *StubDataProvider) GetControl(_ context.Context, id string) (*domain.Control, error) {
	if s.ConnError != nil {
		return nil, s.ConnError
	}
	c, ok := s.Controls[id]
	if !ok {
		return nil, fmt.Errorf("control not found: %s", id)
	}
	return c, nil
}

func (s *StubDataProvider) ListEvidenceTasks(_ context.Context, opts interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	if s.ConnError != nil {
		return nil, 0, s.ConnError
	}
	all := s.allTasks(opts)
	total := len(all)
	paged := paginate(total, opts.Page, opts.PageSize)
	if paged.start >= total {
		return nil, total, nil
	}
	return all[paged.start:paged.end], total, nil
}

func (s *StubDataProvider) GetEvidenceTask(_ context.Context, id string) (*domain.EvidenceTask, error) {
	if s.ConnError != nil {
		return nil, s.ConnError
	}
	t, ok := s.Tasks[id]
	if !ok {
		return nil, fmt.Errorf("evidence task not found: %s", id)
	}
	return t, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type pageRange struct {
	start, end int
}

// paginate calculates start/end indices for a page. Page is 1-based.
// PageSize <= 0 means return all items (start=0, end=total).
func paginate(total, page, pageSize int) pageRange {
	if pageSize <= 0 {
		return pageRange{start: 0, end: total}
	}
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return pageRange{start: start, end: end}
}

func (s *StubDataProvider) allPolicies(opts interfaces.ListOptions) []domain.Policy {
	result := make([]domain.Policy, 0, len(s.Policies))
	for _, p := range s.Policies {
		if opts.Framework != "" && p.Framework != opts.Framework {
			continue
		}
		if opts.Status != "" && p.Status != opts.Status {
			continue
		}
		if opts.Category != "" && p.Category != opts.Category {
			continue
		}
		result = append(result, *p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *StubDataProvider) allControls(opts interfaces.ListOptions) []domain.Control {
	result := make([]domain.Control, 0, len(s.Controls))
	for _, c := range s.Controls {
		if opts.Framework != "" && c.Framework != opts.Framework {
			continue
		}
		if opts.Status != "" && c.Status != opts.Status {
			continue
		}
		if opts.Category != "" && c.Category != opts.Category {
			continue
		}
		result = append(result, *c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *StubDataProvider) allTasks(opts interfaces.ListOptions) []domain.EvidenceTask {
	result := make([]domain.EvidenceTask, 0, len(s.Tasks))
	for _, t := range s.Tasks {
		if opts.Framework != "" && t.Framework != opts.Framework {
			continue
		}
		if opts.Status != "" && t.Status != opts.Status {
			continue
		}
		if opts.Category != "" && t.Category != opts.Category {
			continue
		}
		result = append(result, *t)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
