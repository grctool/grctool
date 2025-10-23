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

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// IntOrString is a type that can unmarshal from either an integer or a string in JSON
// This is used to handle API responses where Tugboat Logic sometimes returns
// integers and sometimes returns strings for the same field
type IntOrString string

// UnmarshalJSON implements json.Unmarshaler interface
func (i *IntOrString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*i = IntOrString(s)
		return nil
	}

	// Try to unmarshal as integer
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*i = IntOrString(strconv.Itoa(n))
		return nil
	}

	// Try to unmarshal as float (sometimes APIs return IDs as floats)
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*i = IntOrString(fmt.Sprintf("%.0f", f))
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into IntOrString", string(data))
}

// MarshalJSON implements json.Marshaler interface
func (i IntOrString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(i))
}

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

// FlexibleTime is a time type that can handle both RFC3339 timestamps and date-only strings
// This is needed because Tugboat Logic API sometimes returns full timestamps (e.g., "2021-12-17T14:42:01.016197Z")
// and sometimes returns date-only strings (e.g., "2026-03-24")
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler interface for flexible time parsing
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// If empty string, set to zero time
	if s == "" {
		ft.Time = time.Time{}
		return nil
	}

	// Try parsing as RFC3339 first (full timestamp)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		ft.Time = t
		return nil
	}

	// Try parsing as date-only format
	if t, err := time.Parse("2006-01-02", s); err == nil {
		ft.Time = t
		return nil
	}

	// Try parsing with nanoseconds (Tugboat API format)
	if t, err := time.Parse("2006-01-02T15:04:05.999999Z", s); err == nil {
		ft.Time = t
		return nil
	}

	// Try parsing without nanoseconds
	if t, err := time.Parse("2006-01-02T15:04:05Z", s); err == nil {
		ft.Time = t
		return nil
	}

	return fmt.Errorf("cannot parse time string: %s", s)
}

// MarshalJSON implements json.Marshaler interface
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.IsZero() {
		return json.Marshal(nil)
	}
	return json.Marshal(ft.Format(time.RFC3339))
}
