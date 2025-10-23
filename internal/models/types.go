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

package models

import "strconv"

// IntOrString is a type that can unmarshal from either an integer or a string in JSON
// This is a minimal implementation for model compatibility
type IntOrString string

// String returns the string representation
func (i IntOrString) String() string {
	return string(i)
}

// ToInt converts the IntOrString to an integer
func (i IntOrString) ToInt() int {
	if val, err := strconv.Atoi(string(i)); err == nil {
		return val
	}
	return 0 // Default value if conversion fails
}
