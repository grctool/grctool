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

package types

import (
	"github.com/go-playground/validator/v10"
)

// Request is the base interface for all tool request types
type Request interface {
	Validate() error
}

// Response is the base interface for all tool response types
type Response interface {
	GetContent() string
	GetMetadata() map[string]interface{}
}

// BaseRequest provides common validation functionality
type BaseRequest struct{}

// validate provides a shared validator instance
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct performs validation on any struct using the shared validator
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
