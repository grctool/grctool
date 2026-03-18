// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"context"
	"fmt"
)

// FolderStructure manages the Google Drive folder hierarchy for GRCTool data.
type FolderStructure struct {
	RootFolderID     string
	PoliciesFolderID string
	ControlsFolderID string
	EvidenceFolderID string
	AuditorFolderID  string // read-only share for auditors
}

// DefaultFolderNames defines the standard subfolder names.
var DefaultFolderNames = map[string]string{
	"policies": "Policies",
	"controls": "Controls",
	"evidence": "Evidence Tasks",
	"auditor":  "Auditor (Read-Only)",
}

// folderMIMEType is the Google Drive MIME type for folders.
const folderMIMEType = "application/vnd.google-apps.folder"

// ResolveFolders checks if the expected subfolder structure exists under root.
// Returns the folder IDs for each subfolder. If a subfolder doesn't exist,
// its ID is left empty (the caller decides whether to create it).
func ResolveFolders(ctx context.Context, client DriveClient, rootFolderID string) (*FolderStructure, error) {
	files, err := client.ListFiles(ctx, rootFolderID, folderMIMEType)
	if err != nil {
		return nil, fmt.Errorf("list subfolders under root %s: %w", rootFolderID, err)
	}

	// Build a name-to-ID lookup from the returned folder entries.
	nameToID := make(map[string]string, len(files))
	for _, f := range files {
		nameToID[f.Name] = f.ID
	}

	fs := &FolderStructure{
		RootFolderID:     rootFolderID,
		PoliciesFolderID: nameToID[DefaultFolderNames["policies"]],
		ControlsFolderID: nameToID[DefaultFolderNames["controls"]],
		EvidenceFolderID: nameToID[DefaultFolderNames["evidence"]],
		AuditorFolderID:  nameToID[DefaultFolderNames["auditor"]],
	}

	return fs, nil
}
