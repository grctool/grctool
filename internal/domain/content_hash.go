// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

// metadataKeys are JSON field names excluded from content hash computation.
// These fields track sync/lifecycle state, not entity content.
var metadataKeys = map[string]bool{
	"sync_metadata":  true,
	"external_ids":   true,
	"lifecycle_state": true,
}

// ComputeContentHash computes a deterministic SHA-256 hash of an entity's
// content fields (excluding metadata like SyncMetadata, ExternalIDs, and
// LifecycleState). The entity is first marshaled to JSON, then metadata
// keys are stripped, and the remaining fields are serialized with sorted
// keys for determinism.
func ComputeContentHash(entity interface{}) (string, error) {
	// Marshal entity to JSON, then unmarshal to a generic map so we can
	// strip metadata fields before hashing.
	data, err := json.Marshal(entity)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entity for hashing: %w", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("failed to unmarshal entity for hashing: %w", err)
	}

	// Remove metadata fields.
	for key := range metadataKeys {
		delete(m, key)
	}

	// Re-serialize with sorted keys for determinism.
	canonical, err := canonicalJSON(m)
	if err != nil {
		return "", fmt.Errorf("failed to produce canonical JSON: %w", err)
	}

	hash := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", hash), nil
}

// canonicalJSON produces a deterministic JSON representation with sorted keys.
func canonicalJSON(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		buf := []byte{'{'}
		for i, k := range keys {
			if i > 0 {
				buf = append(buf, ',')
			}
			keyJSON, _ := json.Marshal(k)
			buf = append(buf, keyJSON...)
			buf = append(buf, ':')
			valJSON, err := canonicalJSON(val[k])
			if err != nil {
				return nil, err
			}
			buf = append(buf, valJSON...)
		}
		buf = append(buf, '}')
		return buf, nil

	case []interface{}:
		buf := []byte{'['}
		for i, item := range val {
			if i > 0 {
				buf = append(buf, ',')
			}
			itemJSON, err := canonicalJSON(item)
			if err != nil {
				return nil, err
			}
			buf = append(buf, itemJSON...)
		}
		buf = append(buf, ']')
		return buf, nil

	default:
		// Primitives (string, number, bool, null) — use standard marshal.
		return json.Marshal(v)
	}
}
