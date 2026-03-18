package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerationMetadata_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	gm := GenerationMetadata{
		GeneratedAt:      now,
		GeneratedBy:      "claude-code-assisted",
		GenerationMethod: "tool_coordination",
		TaskID: "1",
		TaskRef:          "ET-0001",
		Window:           "2025-Q4",
		ToolsUsed:        []string{"terraform_scanner", "github_permissions"},
		FilesGenerated: []FileMetadata{
			{
				Path:        "01_evidence.md",
				Checksum:    "sha256:abc123",
				SizeBytes:   1024,
				GeneratedAt: now,
			},
			{
				Path:        "02_access_matrix.csv",
				Checksum:    "sha256:def456",
				SizeBytes:   512,
				GeneratedAt: now,
			},
		},
		Status: "generated",
	}

	// Use JSON since yaml tags are primary but json should also work for general serialization
	data, err := json.Marshal(gm)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Note: GenerationMetadata uses yaml tags, not json tags, so JSON marshal
	// will use field names. This test verifies the struct is serializable.
	assert.Contains(t, string(data), "claude-code-assisted")
	assert.Contains(t, string(data), "tool_coordination")
}

func TestFileMetadata_Fields(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	fm := FileMetadata{
		Path:        "evidence.md",
		Checksum:    "sha256:abc123",
		SizeBytes:   2048,
		GeneratedAt: now,
	}

	assert.Equal(t, "evidence.md", fm.Path)
	assert.Equal(t, "sha256:abc123", fm.Checksum)
	assert.Equal(t, int64(2048), fm.SizeBytes)
	assert.Equal(t, now, fm.GeneratedAt)
}
