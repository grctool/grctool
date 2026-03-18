// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package testhelpers

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
)

// Compile-time assertions.
var _ interfaces.DataProvider = (*StubFullProvider)(nil)
var _ interfaces.RelationshipQuerier = (*StubFullProvider)(nil)
var _ interfaces.EvidenceSubmitter = (*StubFullProvider)(nil)

// StubFullProvider extends StubDataProvider with RelationshipQuerier and
// EvidenceSubmitter capabilities for testing rich workflows. Use this when
// tests need cross-entity queries or evidence submission; use StubDataProvider
// when basic list/get is sufficient.
type StubFullProvider struct {
	*StubDataProvider

	// Relationship data
	ControlToTasks    map[string][]string // controlID → []taskID
	PolicyToControls  map[string][]string // policyID → []controlID
	ControlToPolicies map[string][]string // controlID → []policyID

	// Evidence submission data
	Submissions []StubSubmission
	Attachments map[string][]interfaces.Attachment // taskID → attachments
	Files       map[string]string                  // attachmentID → content
	SubmitErr   error
}

// StubSubmission records a submission made through SubmitEvidence.
type StubSubmission struct {
	TaskID   string
	Content  string
	Metadata interfaces.SubmissionMetadata
}

// NewStubFullProvider creates a full-featured stub provider.
func NewStubFullProvider(name string) *StubFullProvider {
	return &StubFullProvider{
		StubDataProvider:  NewStubDataProvider(name),
		ControlToTasks:    make(map[string][]string),
		PolicyToControls:  make(map[string][]string),
		ControlToPolicies: make(map[string][]string),
		Attachments:       make(map[string][]interfaces.Attachment),
		Files:             make(map[string]string),
	}
}

// --- RelationshipQuerier ---

func (s *StubFullProvider) GetEvidenceTasksByControl(ctx context.Context, controlID string) ([]domain.EvidenceTask, error) {
	taskIDs := s.ControlToTasks[controlID]
	var tasks []domain.EvidenceTask
	for _, id := range taskIDs {
		if t, ok := s.Tasks[id]; ok {
			tasks = append(tasks, *t)
		}
	}
	return tasks, nil
}

func (s *StubFullProvider) GetControlsByPolicy(ctx context.Context, policyID string) ([]domain.Control, error) {
	controlIDs := s.PolicyToControls[policyID]
	var controls []domain.Control
	for _, id := range controlIDs {
		if c, ok := s.Controls[id]; ok {
			controls = append(controls, *c)
		}
	}
	return controls, nil
}

func (s *StubFullProvider) GetPoliciesByControl(ctx context.Context, controlID string) ([]domain.Policy, error) {
	policyIDs := s.ControlToPolicies[controlID]
	var policies []domain.Policy
	for _, id := range policyIDs {
		if p, ok := s.Policies[id]; ok {
			policies = append(policies, *p)
		}
	}
	return policies, nil
}

// --- EvidenceSubmitter ---

func (s *StubFullProvider) SubmitEvidence(ctx context.Context, taskID string, file io.Reader, meta interfaces.SubmissionMetadata) error {
	if s.SubmitErr != nil {
		return s.SubmitErr
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read evidence file: %w", err)
	}
	s.Submissions = append(s.Submissions, StubSubmission{
		TaskID:   taskID,
		Content:  string(content),
		Metadata: meta,
	})
	return nil
}

func (s *StubFullProvider) ListAttachments(ctx context.Context, taskID string, opts interfaces.ListOptions) ([]interfaces.Attachment, int, error) {
	atts := s.Attachments[taskID]
	return atts, len(atts), nil
}

func (s *StubFullProvider) DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, string, error) {
	content, ok := s.Files[attachmentID]
	if !ok {
		return nil, "", fmt.Errorf("attachment not found: %s", attachmentID)
	}
	return io.NopCloser(strings.NewReader(content)), attachmentID + ".dat", nil
}
