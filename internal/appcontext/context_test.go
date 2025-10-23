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

package appcontext

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	t.Run("Logger storage and retrieval", func(t *testing.T) {
		// Create a test logger (using a mock)
		log := "test-logger" // Using string as a simple mock

		// Store and retrieve
		ctx := WithLogger(ctx, log)
		retrieved := GetLogger(ctx)
		assert.Equal(t, log, retrieved)
	})

	t.Run("Command storage and retrieval", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:   "test",
			Short: "Test command",
		}

		ctx := WithCommand(ctx, cmd)
		retrieved := GetCommand(ctx)
		assert.Equal(t, cmd, retrieved)
	})

	t.Run("Config storage and retrieval", func(t *testing.T) {
		cfg := map[string]string{
			"test": "config",
		}

		ctx := WithConfig(ctx, cfg)
		retrieved := GetConfig(ctx)
		assert.Equal(t, cfg, retrieved)
	})

	t.Run("VCR Config storage and retrieval", func(t *testing.T) {
		vcrCfg := map[string]interface{}{
			"enabled": true,
			"mode":    "playback",
		}

		ctx := WithVCRConfig(ctx, vcrCfg)
		retrieved := GetVCRConfig(ctx)
		assert.Equal(t, vcrCfg, retrieved)
	})

	t.Run("Request ID storage and retrieval", func(t *testing.T) {
		reqID := "req-12345"
		ctx := WithRequestID(ctx, reqID)
		retrieved := GetRequestID(ctx)
		assert.Equal(t, reqID, retrieved)
	})

	t.Run("Operation storage and retrieval", func(t *testing.T) {
		operation := "sync"
		ctx := WithOperation(ctx, operation)
		retrieved := GetOperation(ctx)
		assert.Equal(t, operation, retrieved)
	})

	t.Run("User ID storage and retrieval", func(t *testing.T) {
		userID := "user-123"
		ctx := WithUserID(ctx, userID)
		retrieved := GetUserID(ctx)
		assert.Equal(t, userID, retrieved)
	})
}

func TestEnrichContext(t *testing.T) {
	ctx := context.Background()
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	enriched := EnrichContext(ctx, cmd)

	// Should have command
	assert.Equal(t, cmd, GetCommand(enriched))

	// Should have operation name from command
	assert.Equal(t, "test", GetOperation(enriched))

	// Should have output writer
	assert.NotNil(t, GetOutput(enriched))

	// Should have error writer
	assert.NotNil(t, GetError(enriched))
}

func TestNilSafety(t *testing.T) {
	ctx := context.Background()

	// All getters should return zero values when nothing is stored
	assert.Nil(t, GetLogger(ctx))
	assert.Nil(t, GetCommand(ctx))
	assert.Nil(t, GetConfig(ctx))
	assert.Nil(t, GetVCRConfig(ctx))
	assert.Nil(t, GetOutput(ctx))
	assert.Nil(t, GetError(ctx))
	assert.Equal(t, "", GetRequestID(ctx))
	assert.Equal(t, "", GetUserID(ctx))
	assert.Equal(t, "", GetOperation(ctx))

	// EnrichContext should handle nil command
	enriched := EnrichContext(ctx, nil)
	assert.Nil(t, GetCommand(enriched))
}
