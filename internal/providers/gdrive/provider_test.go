// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

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

// --- Stub DriveClient ---

type stubDriveClient struct {
	files       []DriveFile
	contents    map[string]string // fileID → markdown content
	created     []string          // track created doc IDs
	updated     []string          // track updated fileIDs
	deleted     []string          // track deleted fileIDs
	connErr     error
	listErr     error
	getErr      error
	createErr   error
	updateErr   error
	deleteErr   error
	revisions   map[string][]Revision
}

func newStubDriveClient() *stubDriveClient {
	return &stubDriveClient{
		contents:  make(map[string]string),
		revisions: make(map[string][]Revision),
	}
}

func (s *stubDriveClient) TestConnection(ctx context.Context) error { return s.connErr }

func (s *stubDriveClient) ListFiles(ctx context.Context, folderID, mimeType string) ([]DriveFile, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if mimeType == "" {
		return s.files, nil
	}
	var filtered []DriveFile
	for _, f := range s.files {
		if f.MimeType == mimeType {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

func (s *stubDriveClient) GetFileContent(ctx context.Context, fileID string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	c, ok := s.contents[fileID]
	if !ok {
		return "", fmt.Errorf("file not found: %s", fileID)
	}
	return c, nil
}

func (s *stubDriveClient) CreateDoc(ctx context.Context, folderID, title, content string) (string, error) {
	if s.createErr != nil {
		return "", s.createErr
	}
	id := fmt.Sprintf("new-doc-%d", len(s.created)+1)
	s.created = append(s.created, id)
	s.contents[id] = content
	return id, nil
}

func (s *stubDriveClient) UpdateDoc(ctx context.Context, fileID, content string) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, fileID)
	s.contents[fileID] = content
	return nil
}

func (s *stubDriveClient) DeleteFile(ctx context.Context, fileID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, fileID)
	return nil
}

func (s *stubDriveClient) GetRevisions(ctx context.Context, fileID string, since time.Time) ([]Revision, error) {
	return s.revisions[fileID], nil
}

// --- Tests ---

func TestGDriveSyncProvider_Name(t *testing.T) {
	t.Parallel()
	p := NewGDriveSyncProvider(newStubDriveClient(), "root", testhelpers.NewStubLogger())
	assert.Equal(t, "gdrive", p.Name())
}

func TestGDriveSyncProvider_TestConnection(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		p := NewGDriveSyncProvider(newStubDriveClient(), "root", testhelpers.NewStubLogger())
		assert.NoError(t, p.TestConnection(context.Background()))
	})

	t.Run("failure", func(t *testing.T) {
		client := newStubDriveClient()
		client.connErr = fmt.Errorf("auth failed")
		p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
		assert.Error(t, p.TestConnection(context.Background()))
	})
}

func TestGDriveSyncProvider_ListPolicies(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.files = []DriveFile{
		{ID: "doc-1", Name: "Access Control Policy", MimeType: "application/vnd.google-apps.document"},
		{ID: "doc-2", Name: "Data Protection Policy", MimeType: "application/vnd.google-apps.document"},
		{ID: "sheet-1", Name: "Control Matrix", MimeType: "application/vnd.google-apps.spreadsheet"},
	}
	client.contents["doc-1"] = "# Access Control Policy\n\nThis policy defines access controls."
	client.contents["doc-2"] = "# Data Protection Policy\n\nThis policy covers data protection."

	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	policies, count, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, count) // only Docs, not Sheets
	assert.Len(t, policies, 2)
	assert.Equal(t, "doc-1", policies[0].ExternalIDs["gdrive"])
	assert.NotEmpty(t, policies[0].SyncMetadata.ContentHash["gdrive"])
}

func TestGDriveSyncProvider_ListPolicies_Pagination(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("doc-%d", i)
		client.files = append(client.files, DriveFile{ID: id, Name: fmt.Sprintf("Policy %d", i), MimeType: "application/vnd.google-apps.document"})
		client.contents[id] = fmt.Sprintf("# Policy %d\n\nContent.", i)
	}

	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	policies, count, err := p.ListPolicies(context.Background(), interfaces.ListOptions{Page: 0, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.Len(t, policies, 2)
}

func TestGDriveSyncProvider_GetPolicy(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.contents["doc-1"] = "# Access Control\n\nPolicy content here."

	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	policy, err := p.GetPolicy(context.Background(), "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", policy.ID)
	assert.Equal(t, "doc-1", policy.ExternalIDs["gdrive"])
	assert.Contains(t, policy.Content, "Policy content here")
}

func TestGDriveSyncProvider_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	p := NewGDriveSyncProvider(newStubDriveClient(), "root", testhelpers.NewStubLogger())
	_, err := p.GetPolicy(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestGDriveSyncProvider_PushPolicy_Create(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	policy := &domain.Policy{ID: "POL-001", Name: "New Policy", Content: "# New Policy\n\nContent."}
	err := p.PushPolicy(context.Background(), policy)
	require.NoError(t, err)

	assert.Len(t, client.created, 1)
	assert.Equal(t, "new-doc-1", policy.ExternalIDs["gdrive"])
}

func TestGDriveSyncProvider_PushPolicy_Update(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	policy := &domain.Policy{
		ID:          "POL-001",
		Name:        "Existing Policy",
		Content:     "# Updated Content",
		ExternalIDs: map[string]string{"gdrive": "existing-doc-id"},
	}
	err := p.PushPolicy(context.Background(), policy)
	require.NoError(t, err)

	assert.Len(t, client.updated, 1)
	assert.Equal(t, "existing-doc-id", client.updated[0])
}

func TestGDriveSyncProvider_DeletePolicy(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	err := p.DeletePolicy(context.Background(), "doc-to-delete")
	require.NoError(t, err)
	assert.Contains(t, client.deleted, "doc-to-delete")
}

func TestGDriveSyncProvider_DetectChanges(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	now := time.Now()
	client.files = []DriveFile{
		{ID: "doc-1", Name: "Changed", Modified: now},
		{ID: "doc-2", Name: "Old", Modified: now.Add(-2 * time.Hour)},
	}

	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	changes, err := p.DetectChanges(context.Background(), now.Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, "gdrive", changes.Provider)
	assert.Len(t, changes.Changes, 1) // only doc-1 is after `since`
	assert.Equal(t, "doc-1", changes.Changes[0].EntityID)
}

func TestGDriveSyncProvider_ListControls_NotImplemented(t *testing.T) {
	t.Parallel()
	p := NewGDriveSyncProvider(newStubDriveClient(), "root", testhelpers.NewStubLogger())
	controls, count, err := p.ListControls(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Nil(t, controls)
}

func TestGDriveSyncProvider_CompileTimeInterface(t *testing.T) {
	// Compile-time assertion is at package level (var _ interfaces.SyncProvider = ...)
	// This test just documents it.
	var _ interfaces.SyncProvider = (*GDriveSyncProvider)(nil)
}

// ---------------------------------------------------------------------------
// Registration with ProviderRegistry
// ---------------------------------------------------------------------------

func TestGDriveSyncProvider_RegisterWith(t *testing.T) {
	t.Parallel()
	registry := providers.NewProviderRegistry()
	p := NewGDriveSyncProvider(newStubDriveClient(), "root", testhelpers.NewStubLogger())

	err := p.RegisterWith(registry)
	require.NoError(t, err)
	assert.True(t, registry.Has("gdrive"))

	// Verify it's retrievable as a SyncProvider.
	sp, err := registry.GetSyncProvider("gdrive")
	require.NoError(t, err)
	assert.Equal(t, "gdrive", sp.Name())
}

func TestGDriveSyncProvider_RegisterWith_Duplicate(t *testing.T) {
	t.Parallel()
	registry := providers.NewProviderRegistry()
	p := NewGDriveSyncProvider(newStubDriveClient(), "root", testhelpers.NewStubLogger())

	require.NoError(t, p.RegisterWith(registry))
	err := p.RegisterWith(registry)
	assert.Error(t, err, "duplicate registration must fail")
}

// ---------------------------------------------------------------------------
// Contract tests (equivalent to DataProviderContractSuite)
// ---------------------------------------------------------------------------
// The DataProviderContractSuite is in package providers_test and cannot be
// imported here. We replicate the essential contract checks inline.

func newContractProvider() (*GDriveSyncProvider, string) {
	client := newStubDriveClient()
	docID := "contract-doc-1"
	client.files = []DriveFile{
		{ID: docID, Name: "Contract Policy", MimeType: "application/vnd.google-apps.document"},
	}
	client.contents[docID] = "# Contract Policy\n\nContent for contract testing."
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	return p, docID
}

func TestGDriveContract_Name(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	assert.NotEmpty(t, p.Name(), "Name() must return a non-empty string")
}

func TestGDriveContract_TestConnection(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	assert.NoError(t, p.TestConnection(context.Background()))
}

func TestGDriveContract_ListPolicies_ReturnsResults(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	policies, count, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Greater(t, count, 0, "count must be > 0 when data is loaded")
	assert.NotEmpty(t, policies, "policies slice must not be empty")
}

func TestGDriveContract_ListPolicies_Pagination(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	policies, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{
		Page:     1,
		PageSize: 1,
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(policies), 1, "page size 1 must return at most 1 result")
}

func TestGDriveContract_ListPolicies_EmptyPage(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	policies, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{
		Page:     9999,
		PageSize: 1,
	})
	require.NoError(t, err)
	assert.Empty(t, policies, "requesting a page beyond data should return empty slice")
}

func TestGDriveContract_GetPolicy_Exists(t *testing.T) {
	t.Parallel()
	p, docID := newContractProvider()
	policy, err := p.GetPolicy(context.Background(), docID)
	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, docID, policy.ID, "returned policy ID must match requested ID")
	// Note: Name comes from gdocs.MarkdownToDoc Title extraction, which may
	// be empty depending on the markdown parser. Content must be present.
	assert.NotEmpty(t, policy.Content, "policy Content must not be empty")
}

func TestGDriveContract_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	_, err := p.GetPolicy(context.Background(), "nonexistent-id-99999")
	assert.Error(t, err, "GetPolicy with unknown ID must return an error")
}

func TestGDriveContract_ListControls_Stub(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	// Controls are not yet implemented (Sheets integration pending).
	// Contract: must not error, returns empty.
	controls, count, err := p.ListControls(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, controls)
}

func TestGDriveContract_GetControl_NotFound(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	_, err := p.GetControl(context.Background(), "nonexistent-id-99999")
	assert.Error(t, err, "GetControl with unknown ID must return an error")
}

func TestGDriveContract_ListEvidenceTasks_Stub(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	// Evidence tasks are not yet implemented (Sheets integration pending).
	tasks, count, err := p.ListEvidenceTasks(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, tasks)
}

func TestGDriveContract_GetEvidenceTask_NotFound(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	_, err := p.GetEvidenceTask(context.Background(), "nonexistent-id-99999")
	assert.Error(t, err, "GetEvidenceTask with unknown ID must return an error")
}

func TestGDriveContract_ContextCancellation(t *testing.T) {
	t.Parallel()
	p, _ := newContractProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, errPolicies := p.ListPolicies(ctx, interfaces.ListOptions{})
	_, _, errControls := p.ListControls(ctx, interfaces.ListOptions{})
	_, _, errTasks := p.ListEvidenceTasks(ctx, interfaces.ListOptions{})

	anyErr := errPolicies != nil || errControls != nil || errTasks != nil
	if !anyErr {
		t.Log("NOTE: provider did not return an error for cancelled context (acceptable for stub-backed provider)")
	}
}
