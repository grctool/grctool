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

package providers

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/grctool/grctool/internal/interfaces"
)

// ProviderRegistry manages registered DataProvider and SyncProvider instances.
// Thread-safe for concurrent access.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]interfaces.DataProvider
}

// NewProviderRegistry creates an empty registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]interfaces.DataProvider),
	}
}

// Register adds a provider. Returns error if name already registered.
func (r *ProviderRegistry) Register(provider interfaces.DataProvider) error {
	name := provider.Name()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider already registered: %s", name)
	}
	r.providers[name] = provider
	return nil
}

// Get returns a provider by name, or error if not found.
func (r *ProviderRegistry) Get(name string) (interfaces.DataProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return p, nil
}

// GetSyncProvider returns the provider as a SyncProvider, or error if not found
// or if the provider doesn't implement SyncProvider.
func (r *ProviderRegistry) GetSyncProvider(name string) (interfaces.SyncProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	sp, ok := p.(interfaces.SyncProvider)
	if !ok {
		return nil, fmt.Errorf("provider %s does not implement SyncProvider", name)
	}
	return sp, nil
}

// List returns all registered provider names, sorted for deterministic output.
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListSyncProviders returns names of providers that implement SyncProvider,
// sorted for deterministic output.
func (r *ProviderRegistry) ListSyncProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name, p := range r.providers {
		if _, ok := p.(interfaces.SyncProvider); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// Remove unregisters a provider. No error if not found.
func (r *ProviderRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.providers, name)
}

// Count returns the number of registered providers.
func (r *ProviderRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.providers)
}

// Has checks if a provider is registered.
func (r *ProviderRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.providers[name]
	return ok
}

// HealthCheck tests connectivity for all registered providers.
// Returns a map of provider name to error (nil means healthy).
func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]error {
	r.mu.RLock()
	// Copy the map to avoid holding the lock during potentially slow calls.
	snapshot := make(map[string]interfaces.DataProvider, len(r.providers))
	for name, p := range r.providers {
		snapshot[name] = p
	}
	r.mu.RUnlock()

	results := make(map[string]error, len(snapshot))
	for name, p := range snapshot {
		results[name] = p.TestConnection(ctx)
	}
	return results
}
