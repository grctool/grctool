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

// Package sync provides cross-provider conflict detection and resolution
// for GRCTool's multi-provider synchronization engine.
package sync

import (
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
)

// ConflictPolicy defines how conflicts are resolved.
type ConflictPolicy string

const (
	// ConflictPolicyLocalWins always keeps the local version.
	ConflictPolicyLocalWins ConflictPolicy = "local_wins"
	// ConflictPolicyRemoteWins always keeps the remote version.
	ConflictPolicyRemoteWins ConflictPolicy = "remote_wins"
	// ConflictPolicyNewestWins keeps whichever version was modified most recently.
	ConflictPolicyNewestWins ConflictPolicy = "newest_wins"
	// ConflictPolicyManual leaves conflicts unresolved for manual intervention.
	ConflictPolicyManual ConflictPolicy = "manual"
)

// ConflictClassification describes the type of conflict detected.
type ConflictClassification string

const (
	// ClassificationInSync means local and remote are identical.
	ClassificationInSync ConflictClassification = "in_sync"
	// ClassificationLocalOnly means the entity exists locally but not remotely.
	ClassificationLocalOnly ConflictClassification = "local_only"
	// ClassificationRemoteOnly means the entity exists remotely but not locally.
	ClassificationRemoteOnly ConflictClassification = "remote_only"
	// ClassificationLocalChanged means only the local version changed since last sync.
	ClassificationLocalChanged ConflictClassification = "local_changed"
	// ClassificationRemoteChanged means only the remote version changed since last sync.
	ClassificationRemoteChanged ConflictClassification = "remote_changed"
	// ClassificationBothChanged means both local and remote changed since last sync.
	ClassificationBothChanged ConflictClassification = "both_changed"
)

// Conflict represents a detected conflict between local and remote state.
type Conflict struct {
	EntityType     string                 // "policy", "control", "evidence_task"
	EntityID       string
	Provider       string
	Classification ConflictClassification
	LocalHash      string
	RemoteHash     string
	LocalModified  time.Time
	RemoteModified time.Time
}

// Resolution represents the outcome of resolving a conflict.
type Resolution struct {
	Conflict   Conflict
	Policy     ConflictPolicy
	Winner     string // "local", "remote", or "" (unresolved)
	ResolvedAt time.Time
}

// ConflictDetector compares local and remote state to detect conflicts.
type ConflictDetector struct{}

// NewConflictDetector creates a new ConflictDetector.
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConflicts compares local sync metadata against remote changes and
// returns a list of conflicts. Entities that are in sync are not included
// in the result.
//
// For each remote ChangeEntry the detector looks up the entity's local
// SyncMetadata to classify the change:
//   - No local hash for the provider  -> ClassificationRemoteOnly
//   - Remote hash == local stored hash -> check if local hash changed
//   - Both hashes differ from stored   -> ClassificationBothChanged
//   - Only remote changed              -> ClassificationRemoteChanged
//   - Only local changed               -> ClassificationLocalChanged
func (d *ConflictDetector) DetectConflicts(
	localMeta *domain.SyncMetadata,
	remoteChanges []interfaces.ChangeEntry,
	provider string,
) []Conflict {
	var conflicts []Conflict

	for _, rc := range remoteChanges {
		c := d.classifyChange(localMeta, rc, provider)
		if c.Classification == ClassificationInSync {
			continue
		}
		conflicts = append(conflicts, c)
	}

	return conflicts
}

// classifyChange determines the conflict classification for a single
// remote change entry against local sync metadata.
func (d *ConflictDetector) classifyChange(
	localMeta *domain.SyncMetadata,
	change interfaces.ChangeEntry,
	provider string,
) Conflict {
	c := Conflict{
		EntityType:     change.EntityType,
		EntityID:       change.EntityID,
		Provider:       provider,
		RemoteHash:     change.Hash,
		RemoteModified: change.ModifiedAt,
	}

	// If no local metadata exists at all, everything is remote-only.
	if localMeta == nil {
		c.Classification = ClassificationRemoteOnly
		return c
	}

	storedHash, hasStoredHash := localMeta.ContentHash[provider]
	if !hasStoredHash {
		// No local hash for this provider — entity is new from remote.
		c.Classification = ClassificationRemoteOnly
		return c
	}

	// storedHash is the hash we recorded at last sync.
	// change.Hash is the current remote hash.
	// We need to know if the local content changed since last sync.
	// The local current hash is whatever is in ContentHash right now;
	// if it still equals storedHash, local hasn't changed.
	// But we track per-provider hashes — the "local" hash is the
	// entity's current content hash stored separately from the provider hash.
	// In our model, ContentHash[provider] is updated at sync time.
	// A local modification would change the entity but not update
	// ContentHash[provider], so we detect local changes by comparing
	// the entity's current content hash (passed via a "local" pseudo-provider
	// or via the stored hash itself).
	//
	// For this framework, we use a convention: ContentHash["local"] holds
	// the current local content hash. If "local" key is absent, we treat
	// local as unchanged from the stored provider hash.
	localCurrentHash, hasLocalHash := localMeta.ContentHash["local"]
	if !hasLocalHash {
		// No explicit local hash tracked — assume local matches stored.
		localCurrentHash = storedHash
	}
	c.LocalHash = localCurrentHash

	remoteChanged := change.Hash != storedHash
	localChanged := localCurrentHash != storedHash

	switch {
	case !remoteChanged && !localChanged:
		c.Classification = ClassificationInSync
	case remoteChanged && localChanged:
		c.Classification = ClassificationBothChanged
	case remoteChanged:
		c.Classification = ClassificationRemoteChanged
	case localChanged:
		c.Classification = ClassificationLocalChanged
	}

	// Populate local modified time from sync metadata if available.
	if syncTime, ok := localMeta.LastSyncTime[provider]; ok {
		c.LocalModified = syncTime
	}

	return c
}

// ConflictResolver applies a policy to resolve conflicts.
type ConflictResolver struct {
	DefaultPolicy ConflictPolicy
}

// NewConflictResolver creates a new ConflictResolver with the given default policy.
func NewConflictResolver(policy ConflictPolicy) *ConflictResolver {
	return &ConflictResolver{DefaultPolicy: policy}
}

// Resolve applies the conflict policy and returns a resolution.
func (r *ConflictResolver) Resolve(conflict Conflict) Resolution {
	res := Resolution{
		Conflict:   conflict,
		Policy:     r.DefaultPolicy,
		ResolvedAt: time.Now(),
	}

	switch r.DefaultPolicy {
	case ConflictPolicyLocalWins:
		res.Winner = "local"
	case ConflictPolicyRemoteWins:
		res.Winner = "remote"
	case ConflictPolicyNewestWins:
		if conflict.LocalModified.After(conflict.RemoteModified) {
			res.Winner = "local"
		} else if conflict.RemoteModified.After(conflict.LocalModified) {
			res.Winner = "remote"
		} else {
			// Tiebreak: local wins when timestamps are equal.
			res.Winner = "local"
		}
	case ConflictPolicyManual:
		res.Winner = ""
	}

	return res
}

// ResolveAll resolves a batch of conflicts and returns all resolutions.
func (r *ConflictResolver) ResolveAll(conflicts []Conflict) []Resolution {
	resolutions := make([]Resolution, 0, len(conflicts))
	for _, c := range conflicts {
		resolutions = append(resolutions, r.Resolve(c))
	}
	return resolutions
}
