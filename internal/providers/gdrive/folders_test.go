// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFolders_AllPresent(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.files = []DriveFile{
		{ID: "folder-pol", Name: "Policies", MimeType: folderMIMEType},
		{ID: "folder-ctrl", Name: "Controls", MimeType: folderMIMEType},
		{ID: "folder-ev", Name: "Evidence Tasks", MimeType: folderMIMEType},
		{ID: "folder-aud", Name: "Auditor (Read-Only)", MimeType: folderMIMEType},
	}

	fs, err := ResolveFolders(context.Background(), client, "root-123")
	require.NoError(t, err)
	assert.Equal(t, "root-123", fs.RootFolderID)
	assert.Equal(t, "folder-pol", fs.PoliciesFolderID)
	assert.Equal(t, "folder-ctrl", fs.ControlsFolderID)
	assert.Equal(t, "folder-ev", fs.EvidenceFolderID)
	assert.Equal(t, "folder-aud", fs.AuditorFolderID)
}

func TestResolveFolders_Partial(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.files = []DriveFile{
		{ID: "folder-pol", Name: "Policies", MimeType: folderMIMEType},
		// Controls, Evidence Tasks, and Auditor are missing
	}

	fs, err := ResolveFolders(context.Background(), client, "root-456")
	require.NoError(t, err)
	assert.Equal(t, "root-456", fs.RootFolderID)
	assert.Equal(t, "folder-pol", fs.PoliciesFolderID)
	assert.Empty(t, fs.ControlsFolderID)
	assert.Empty(t, fs.EvidenceFolderID)
	assert.Empty(t, fs.AuditorFolderID)
}

func TestResolveFolders_Empty(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	// No files at all

	fs, err := ResolveFolders(context.Background(), client, "root-789")
	require.NoError(t, err)
	assert.Equal(t, "root-789", fs.RootFolderID)
	assert.Empty(t, fs.PoliciesFolderID)
	assert.Empty(t, fs.ControlsFolderID)
	assert.Empty(t, fs.EvidenceFolderID)
	assert.Empty(t, fs.AuditorFolderID)
}

func TestResolveFolders_Error(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.listErr = fmt.Errorf("API quota exceeded")

	fs, err := ResolveFolders(context.Background(), client, "root-err")
	assert.Nil(t, fs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API quota exceeded")
}
