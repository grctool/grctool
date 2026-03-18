// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"github.com/grctool/grctool/internal/interfaces"
)

// RegisterWith registers the GDrive provider with a ProviderRegistry.
// This is a convenience wrapper around registry.Register.
func (p *GDriveSyncProvider) RegisterWith(registry interfaces.ProviderRegistry) error {
	return registry.Register(p)
}
