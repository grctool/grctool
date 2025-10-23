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

// Relationships defines the connections between different entities
type Relationships struct {
	PolicyToProcedures map[string][]string `json:"policy_to_procedures"`
	PolicyToControls   map[string][]string `json:"policy_to_controls"`
	ProcedureToTasks   map[string][]string `json:"procedure_to_tasks"`
	ControlToTasks     map[string][]string `json:"control_to_tasks"`
	TaskToEvidence     map[string][]string `json:"task_to_evidence"`
}
