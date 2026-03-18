// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package accountablehq

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/providers"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Stub Client ---

type stubAHQClient struct {
	policies   []AHQPolicy
	connErr    error
	listErr    error
	getErr     error
	createErr  error
	updateErr  error
	deleteErr  error
	created    []AHQPolicy
	updated    map[string]*AHQPolicy
	deleted    []string
}

func newStubClient() *stubAHQClient {
	return &stubAHQClient{updated: make(map[string]*AHQPolicy)}
}

func (c *stubAHQClient) TestConnection(ctx context.Context) error { return c.connErr }

func (c *stubAHQClient) ListPolicies(ctx context.Context, page, pageSize int) ([]AHQPolicy, int, error) {
	if c.listErr != nil {
		return nil, 0, c.listErr
	}
	total := len(c.policies)
	if pageSize <= 0 {
		return c.policies, total, nil
	}
	start := page * pageSize
	if start >= total {
		return nil, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return c.policies[start:end], total, nil
}

func (c *stubAHQClient) GetPolicy(ctx context.Context, id string) (*AHQPolicy, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	for _, p := range c.policies {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("policy not found: %s", id)
}

func (c *stubAHQClient) CreatePolicy(ctx context.Context, policy *AHQPolicy) (string, error) {
	if c.createErr != nil {
		return "", c.createErr
	}
	id := fmt.Sprintf("ahq-new-%d", len(c.created)+1)
	c.created = append(c.created, *policy)
	return id, nil
}

func (c *stubAHQClient) UpdatePolicy(ctx context.Context, id string, policy *AHQPolicy) error {
	if c.updateErr != nil {
		return c.updateErr
	}
	c.updated[id] = policy
	return nil
}

func (c *stubAHQClient) DeletePolicy(ctx context.Context, id string) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	c.deleted = append(c.deleted, id)
	return nil
}

// --- Tests ---

func TestAccountableHQ_Name(t *testing.T) {
	t.Parallel()
	p := NewAccountableHQSyncProvider(newStubClient(), testhelpers.NewStubLogger())
	assert.Equal(t, "accountablehq", p.Name())
}

func TestAccountableHQ_TestConnection(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		p := NewAccountableHQSyncProvider(newStubClient(), testhelpers.NewStubLogger())
		assert.NoError(t, p.TestConnection(context.Background()))
	})
	t.Run("failure", func(t *testing.T) {
		c := newStubClient()
		c.connErr = fmt.Errorf("unauthorized")
		p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())
		assert.Error(t, p.TestConnection(context.Background()))
	})
}

func TestAccountableHQ_ListPolicies(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	c.policies = []AHQPolicy{
		{ID: "ahq-1", Title: "Access Control", Content: "Policy text", Status: "active", Version: 3},
		{ID: "ahq-2", Title: "Data Protection", Content: "Another policy", Status: "draft", Version: 1},
	}
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	policies, total, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, policies, 2)
	assert.Equal(t, "Access Control", policies[0].Name)
	assert.Equal(t, "ahq-1", policies[0].ExternalIDs["accountablehq"])
	assert.NotNil(t, policies[0].SyncMetadata)
}

func TestAccountableHQ_ListPolicies_Pagination(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	for i := 0; i < 5; i++ {
		c.policies = append(c.policies, AHQPolicy{ID: fmt.Sprintf("ahq-%d", i), Title: fmt.Sprintf("Policy %d", i)})
	}
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	policies, total, err := p.ListPolicies(context.Background(), interfaces.ListOptions{Page: 0, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, policies, 2)
}

func TestAccountableHQ_ListPolicies_Error(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	c.listErr = fmt.Errorf("rate limited")
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	_, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestAccountableHQ_GetPolicy(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	c.policies = []AHQPolicy{{ID: "ahq-1", Title: "Test Policy", Content: "Content", Version: 2}}
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	policy, err := p.GetPolicy(context.Background(), "ahq-1")
	require.NoError(t, err)
	assert.Equal(t, "ahq-1", policy.ID)
	assert.Equal(t, "Test Policy", policy.Name)
	assert.Equal(t, "ahq-1", policy.ExternalIDs["accountablehq"])
}

func TestAccountableHQ_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	p := NewAccountableHQSyncProvider(newStubClient(), testhelpers.NewStubLogger())
	_, err := p.GetPolicy(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestAccountableHQ_PushPolicy_Create(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	policy := &domain.Policy{ID: "POL-001", Name: "New Policy", Content: "# New Policy"}
	err := p.PushPolicy(context.Background(), policy)
	require.NoError(t, err)
	assert.Len(t, c.created, 1)
	assert.Equal(t, "ahq-new-1", policy.ExternalIDs["accountablehq"])
}

func TestAccountableHQ_PushPolicy_Update(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	policy := &domain.Policy{
		ID:          "POL-001",
		Name:        "Updated Policy",
		Content:     "# Updated",
		ExternalIDs: map[string]string{"accountablehq": "ahq-existing"},
	}
	err := p.PushPolicy(context.Background(), policy)
	require.NoError(t, err)
	assert.Contains(t, c.updated, "ahq-existing")
}

func TestAccountableHQ_PushPolicy_CreateError(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	c.createErr = fmt.Errorf("quota exceeded")
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	err := p.PushPolicy(context.Background(), &domain.Policy{Name: "Test"})
	assert.Error(t, err)
}

func TestAccountableHQ_PushPolicy_UpdateError(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	c.updateErr = fmt.Errorf("forbidden")
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	err := p.PushPolicy(context.Background(), &domain.Policy{ExternalIDs: map[string]string{"accountablehq": "ahq-1"}})
	assert.Error(t, err)
}

func TestAccountableHQ_DeletePolicy(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	err := p.DeletePolicy(context.Background(), "ahq-delete-me")
	require.NoError(t, err)
	assert.Contains(t, c.deleted, "ahq-delete-me")
}

func TestAccountableHQ_DetectChanges(t *testing.T) {
	t.Parallel()
	now := time.Now()
	c := newStubClient()
	c.policies = []AHQPolicy{
		{ID: "ahq-1", Title: "Changed", Version: 5, UpdatedAt: now},
		{ID: "ahq-2", Title: "Old", Version: 1, UpdatedAt: now.Add(-2 * time.Hour)},
	}
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	changes, err := p.DetectChanges(context.Background(), now.Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "accountablehq", changes.Provider)
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, "ahq-1", changes.Changes[0].EntityID)
}

func TestAccountableHQ_DetectChanges_Error(t *testing.T) {
	t.Parallel()
	c := newStubClient()
	c.listErr = fmt.Errorf("timeout")
	p := NewAccountableHQSyncProvider(c, testhelpers.NewStubLogger())

	_, err := p.DetectChanges(context.Background(), time.Now())
	assert.Error(t, err)
}

func TestAccountableHQ_ControlsNotManaged(t *testing.T) {
	t.Parallel()
	p := NewAccountableHQSyncProvider(newStubClient(), testhelpers.NewStubLogger())

	controls, n, err := p.ListControls(context.Background(), interfaces.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Nil(t, controls)

	_, err = p.GetControl(context.Background(), "any")
	assert.Error(t, err)

	assert.Error(t, p.PushControl(context.Background(), &domain.Control{}))
	assert.Error(t, p.DeleteControl(context.Background(), "any"))
}

func TestAccountableHQ_EvidenceTasksNotManaged(t *testing.T) {
	t.Parallel()
	p := NewAccountableHQSyncProvider(newStubClient(), testhelpers.NewStubLogger())

	tasks, n, err := p.ListEvidenceTasks(context.Background(), interfaces.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Nil(t, tasks)

	_, err = p.GetEvidenceTask(context.Background(), "any")
	assert.Error(t, err)

	assert.Error(t, p.PushEvidenceTask(context.Background(), &domain.EvidenceTask{}))
	assert.Error(t, p.DeleteEvidenceTask(context.Background(), "any"))
}

func TestAccountableHQ_RegisterWith(t *testing.T) {
	t.Parallel()
	reg := providers.NewProviderRegistry()
	p := NewAccountableHQSyncProvider(newStubClient(), testhelpers.NewStubLogger())

	require.NoError(t, p.RegisterWith(reg))
	assert.True(t, reg.Has("accountablehq"))

	sp, err := reg.GetSyncProvider("accountablehq")
	require.NoError(t, err)
	assert.Equal(t, "accountablehq", sp.Name())
}

func TestAccountableHQ_CompileTimeInterface(t *testing.T) {
	var _ interfaces.SyncProvider = (*AccountableHQSyncProvider)(nil)
}

func TestConvertToDomain(t *testing.T) {
	t.Parallel()
	ap := AHQPolicy{
		ID:        "ahq-99",
		Title:     "Test Policy",
		Content:   "# Test",
		Status:    "active",
		Version:   7,
		Category:  "Security",
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC),
	}
	pol := convertToDomain(ap)
	assert.Equal(t, "ahq-99", pol.ID)
	assert.Equal(t, "Test Policy", pol.Name)
	assert.Equal(t, "active", pol.Status)
	assert.Equal(t, "Security", pol.Category)
	assert.Equal(t, "ahq-99", pol.ExternalIDs["accountablehq"])
	assert.Equal(t, "v7", pol.SyncMetadata.ContentHash["accountablehq"])
}

func TestConvertFromDomain(t *testing.T) {
	t.Parallel()
	pol := &domain.Policy{Name: "My Policy", Content: "# Content", Status: "draft", Category: "Compliance"}
	ahq := convertFromDomain(pol)
	assert.Equal(t, "My Policy", ahq.Title)
	assert.Equal(t, "# Content", ahq.Content)
	assert.Equal(t, "draft", ahq.Status)
	assert.Equal(t, "Compliance", ahq.Category)
}
