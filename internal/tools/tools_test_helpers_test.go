// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"github.com/grctool/grctool/internal/config"
)

// newTestConfig creates a minimal config.Config suitable for testing.
func newTestConfig(dataDir string) *config.Config {
	return &config.Config{
		Storage: config.StorageConfig{
			DataDir: dataDir,
		},
	}
}
