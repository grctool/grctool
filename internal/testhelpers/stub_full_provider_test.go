// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package testhelpers

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubFullProvider_ImplementsRelationshipQuerier(t *testing.T) {
	t.Parallel()
	var p interfaces.DataProvider = NewStubFullProvider("test")
	_, ok := p.(interfaces.RelationshipQuerier)
	assert.True(t, ok)
}

func TestStubFullProvider_ImplementsEvidenceSubmitter(t *testing.T) {
	t.Parallel()
	var p interfaces.DataProvider = NewStubFullProvider("test")
	_, ok := p.(interfaces.EvidenceSubmitter)
	assert.True(t, ok)
}

func TestStubFullProvider_InheritsDataProvider(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.Policies["POL-001"] = &domain.Policy{ID: "POL-001", Name: "Test Policy"}

	pol, err := fp.GetPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	assert.Equal(t, "Test Policy", pol.Name)
}

func TestStubFullProvider_GetEvidenceTasksByControl(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.Tasks["ET-0047"] = &domain.EvidenceTask{ID: "ET-0047", Name: "GitHub Access"}
	fp.Tasks["ET-0048"] = &domain.EvidenceTask{ID: "ET-0048", Name: "GitHub Workflows"}
	fp.ControlToTasks["CC-06.1"] = []string{"ET-0047", "ET-0048"}

	tasks, err := fp.GetEvidenceTasksByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "ET-0047", tasks[0].ID)
}

func TestStubFullProvider_GetEvidenceTasksByControl_Empty(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	tasks, err := fp.GetEvidenceTasksByControl(context.Background(), "UNKNOWN")
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestStubFullProvider_GetEvidenceTasksByControl_MissingTask(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.ControlToTasks["CC-06.1"] = []string{"ET-GONE"} // task not in Tasks map
	tasks, err := fp.GetEvidenceTasksByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	assert.Empty(t, tasks) // silently skips missing
}

func TestStubFullProvider_GetControlsByPolicy(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.Controls["CC-06.1"] = &domain.Control{ID: "CC-06.1", Name: "Logical Access"}
	fp.PolicyToControls["POL-001"] = []string{"CC-06.1"}

	controls, err := fp.GetControlsByPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	assert.Len(t, controls, 1)
	assert.Equal(t, "Logical Access", controls[0].Name)
}

func TestStubFullProvider_GetPoliciesByControl(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.Policies["POL-001"] = &domain.Policy{ID: "POL-001", Name: "Access Control"}
	fp.ControlToPolicies["CC-06.1"] = []string{"POL-001"}

	policies, err := fp.GetPoliciesByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	assert.Len(t, policies, 1)
	assert.Equal(t, "Access Control", policies[0].Name)
}

func TestStubFullProvider_SubmitEvidence(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	meta := interfaces.SubmissionMetadata{
		CollectedDate: "2026-03-18",
		Window:        "2026-Q1",
	}

	err := fp.SubmitEvidence(context.Background(), "ET-0047",
		bytes.NewReader([]byte("evidence data")), meta)
	require.NoError(t, err)

	assert.Len(t, fp.Submissions, 1)
	assert.Equal(t, "ET-0047", fp.Submissions[0].TaskID)
	assert.Equal(t, "evidence data", fp.Submissions[0].Content)
	assert.Equal(t, "2026-Q1", fp.Submissions[0].Metadata.Window)
}

func TestStubFullProvider_SubmitEvidence_Error(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.SubmitErr = assert.AnError

	err := fp.SubmitEvidence(context.Background(), "ET-0001",
		bytes.NewReader([]byte("data")), interfaces.SubmissionMetadata{})
	assert.Error(t, err)
}

func TestStubFullProvider_ListAttachments(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.Attachments["ET-0047"] = []interfaces.Attachment{
		{ID: "att-1", Filename: "report.csv"},
		{ID: "att-2", Filename: "screenshot.png"},
	}

	atts, total, err := fp.ListAttachments(context.Background(), "ET-0047", interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, atts, 2)
}

func TestStubFullProvider_ListAttachments_Empty(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	atts, total, err := fp.ListAttachments(context.Background(), "UNKNOWN", interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, atts)
}

func TestStubFullProvider_DownloadAttachment(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	fp.Files["att-1"] = "user,role\nadmin,full"

	reader, filename, err := fp.DownloadAttachment(context.Background(), "att-1")
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "att-1.dat", filename)
	content, _ := io.ReadAll(reader)
	assert.Contains(t, string(content), "admin")
}

func TestStubFullProvider_DownloadAttachment_NotFound(t *testing.T) {
	t.Parallel()
	fp := NewStubFullProvider("full")
	_, _, err := fp.DownloadAttachment(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestStubFullProvider_FullWorkflow(t *testing.T) {
	t.Parallel()
	// Simulate: list controls for a policy → get tasks for a control → submit evidence
	fp := NewStubFullProvider("workflow")

	pol := &domain.Policy{ID: "POL-001", Name: "Access Control"}
	ctrl := &domain.Control{ID: "CC-06.1", Name: "Logical Access"}
	task := &domain.EvidenceTask{ID: "ET-0047", Name: "GitHub Access Controls"}

	fp.Policies["POL-001"] = pol
	fp.Controls["CC-06.1"] = ctrl
	fp.Tasks["ET-0047"] = task
	fp.PolicyToControls["POL-001"] = []string{"CC-06.1"}
	fp.ControlToTasks["CC-06.1"] = []string{"ET-0047"}

	// Step 1: Get controls for policy
	controls, err := fp.GetControlsByPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	require.Len(t, controls, 1)

	// Step 2: Get tasks for control
	tasks, err := fp.GetEvidenceTasksByControl(context.Background(), controls[0].ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Step 3: Submit evidence for task
	err = fp.SubmitEvidence(context.Background(), tasks[0].ID,
		bytes.NewReader([]byte("collected evidence")),
		interfaces.SubmissionMetadata{CollectedDate: "2026-03-18"})
	require.NoError(t, err)

	assert.Len(t, fp.Submissions, 1)
	assert.Equal(t, "ET-0047", fp.Submissions[0].TaskID)
}
