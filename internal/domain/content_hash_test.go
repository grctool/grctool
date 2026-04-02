// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeContentHash_Policy_NonEmpty(t *testing.T) {
	p := Policy{
		ID:   "12345",
		Name: "Access Control Policy",
	}

	hash, err := ComputeContentHash(p)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 hex = 64 chars
}

func TestComputeContentHash_Deterministic(t *testing.T) {
	p := Policy{
		ID:          "12345",
		Name:        "Access Control Policy",
		Description: "Defines access controls",
		Framework:   "SOC2",
	}

	hash1, err := ComputeContentHash(p)
	require.NoError(t, err)

	hash2, err := ComputeContentHash(p)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "same entity should produce same hash")
}

func TestComputeContentHash_DifferentContent_DifferentHash(t *testing.T) {
	p1 := Policy{ID: "1", Name: "Policy A"}
	p2 := Policy{ID: "1", Name: "Policy B"}

	hash1, _ := ComputeContentHash(p1)
	hash2, _ := ComputeContentHash(p2)

	assert.NotEqual(t, hash1, hash2, "different content should produce different hash")
}

func TestComputeContentHash_ExcludesMetadata(t *testing.T) {
	// Same content, different metadata — hashes should be equal.
	p1 := Policy{
		ID:   "12345",
		Name: "Test Policy",
	}

	p2 := Policy{
		ID:   "12345",
		Name: "Test Policy",
		ExternalIDs: map[string]string{"tugboat": "99"},
		SyncMetadata: &SyncMetadata{
			ContentHash:  map[string]string{"tugboat": "old-hash"},
			LastSyncTime: map[string]time.Time{"tugboat": time.Now()},
		},
		LifecycleState: "published",
	}

	hash1, err := ComputeContentHash(p1)
	require.NoError(t, err)

	hash2, err := ComputeContentHash(p2)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2,
		"metadata fields (ExternalIDs, SyncMetadata, LifecycleState) should be excluded from hash")
}

func TestComputeContentHash_Control(t *testing.T) {
	c := Control{
		ID:          "778805",
		ReferenceID: "CC-06.1",
		Name:        "Logical Access Controls",
	}

	hash, err := ComputeContentHash(c)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestComputeContentHash_EvidenceTask(t *testing.T) {
	et := EvidenceTask{
		ID:          "327992",
		ReferenceID: "ET-0047",
		Name:        "GitHub Repository Access Controls",
	}

	hash, err := ComputeContentHash(et)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}
