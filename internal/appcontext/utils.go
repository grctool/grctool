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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateRequestID creates a unique request ID for tracing
func GenerateRequestID() string {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	// Return as hex string with prefix
	return fmt.Sprintf("req-%s", hex.EncodeToString(bytes))
}

// GenerateOperationID creates a unique operation ID
func GenerateOperationID(operationName string) string {
	// Generate 4 random bytes for shorter IDs
	bytes := make([]byte, 4)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp
		return fmt.Sprintf("%s-%d", operationName, time.Now().UnixNano())
	}

	// Return operation name with random suffix
	return fmt.Sprintf("%s-%s", operationName, hex.EncodeToString(bytes))
}
