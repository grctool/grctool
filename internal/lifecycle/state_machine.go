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

package lifecycle

import "fmt"

// State represents a lifecycle state.
type State string

// EntityType identifies which lifecycle applies.
type EntityType string

const (
	EntityPolicy       EntityType = "policy"
	EntityControl      EntityType = "control"
	EntityEvidenceTask EntityType = "evidence_task"
)

// Policy lifecycle states.
const (
	PolicyDraft     State = "draft"
	PolicyReview    State = "review"
	PolicyApproved  State = "approved"
	PolicyPublished State = "published"
	PolicyRetired   State = "retired"
)

// Control lifecycle states.
const (
	ControlDefined     State = "defined"
	ControlImplemented State = "implemented"
	ControlTested      State = "tested"
	ControlEffective   State = "effective"
	ControlDeprecated  State = "deprecated"
)

// Evidence task lifecycle states (extends existing LocalEvidenceState).
const (
	EvidenceScheduled  State = "scheduled"
	EvidenceCollecting State = "collecting"
	EvidenceCollected  State = "collected"
	EvidenceValidated  State = "validated"
	EvidenceSubmitted  State = "submitted"
	EvidenceAccepted   State = "accepted"
	EvidenceRejected   State = "rejected"
)

// Transition defines a valid state change.
type Transition struct {
	From State
	To   State
}

// StateMachine defines valid states and transitions for an entity type.
type StateMachine struct {
	EntityType  EntityType
	States      []State
	Transitions []Transition
	Initial     State
	Terminal    []State // states that cannot transition further
}

// NewPolicyStateMachine returns the policy lifecycle state machine.
func NewPolicyStateMachine() *StateMachine {
	return &StateMachine{
		EntityType: EntityPolicy,
		States: []State{
			PolicyDraft,
			PolicyReview,
			PolicyApproved,
			PolicyPublished,
			PolicyRetired,
		},
		Transitions: []Transition{
			{From: PolicyDraft, To: PolicyReview},
			{From: PolicyReview, To: PolicyApproved},
			{From: PolicyReview, To: PolicyDraft},
			{From: PolicyApproved, To: PolicyPublished},
			{From: PolicyPublished, To: PolicyReview},
			{From: PolicyPublished, To: PolicyRetired},
		},
		Initial:  PolicyDraft,
		Terminal: []State{PolicyRetired},
	}
}

// NewControlStateMachine returns the control lifecycle state machine.
func NewControlStateMachine() *StateMachine {
	return &StateMachine{
		EntityType: EntityControl,
		States: []State{
			ControlDefined,
			ControlImplemented,
			ControlTested,
			ControlEffective,
			ControlDeprecated,
		},
		Transitions: []Transition{
			{From: ControlDefined, To: ControlImplemented},
			{From: ControlImplemented, To: ControlTested},
			{From: ControlTested, To: ControlEffective},
			{From: ControlTested, To: ControlImplemented},
			{From: ControlEffective, To: ControlTested},
			{From: ControlEffective, To: ControlDeprecated},
		},
		Initial:  ControlDefined,
		Terminal: []State{ControlDeprecated},
	}
}

// NewEvidenceTaskStateMachine returns the evidence task lifecycle state machine.
func NewEvidenceTaskStateMachine() *StateMachine {
	return &StateMachine{
		EntityType: EntityEvidenceTask,
		States: []State{
			EvidenceScheduled,
			EvidenceCollecting,
			EvidenceCollected,
			EvidenceValidated,
			EvidenceSubmitted,
			EvidenceAccepted,
			EvidenceRejected,
		},
		Transitions: []Transition{
			{From: EvidenceScheduled, To: EvidenceCollecting},
			{From: EvidenceCollecting, To: EvidenceCollected},
			{From: EvidenceCollected, To: EvidenceValidated},
			{From: EvidenceCollected, To: EvidenceCollecting},
			{From: EvidenceValidated, To: EvidenceSubmitted},
			{From: EvidenceSubmitted, To: EvidenceAccepted},
			{From: EvidenceSubmitted, To: EvidenceRejected},
			{From: EvidenceRejected, To: EvidenceCollecting},
		},
		Initial:  EvidenceScheduled,
		Terminal: []State{EvidenceAccepted},
	}
}

// CanTransition checks if a transition from current to target is valid.
// Self-loops (from == to) are not allowed.
func (sm *StateMachine) CanTransition(from, to State) bool {
	if from == to {
		return false
	}
	for _, t := range sm.Transitions {
		if t.From == from && t.To == to {
			return true
		}
	}
	return false
}

// ValidTransitionsFrom returns all states reachable from the given state.
func (sm *StateMachine) ValidTransitionsFrom(from State) []State {
	var result []State
	for _, t := range sm.Transitions {
		if t.From == from {
			result = append(result, t.To)
		}
	}
	return result
}

// IsValidState checks if a state is defined in this machine.
func (sm *StateMachine) IsValidState(state State) bool {
	for _, s := range sm.States {
		if s == state {
			return true
		}
	}
	return false
}

// IsTerminal checks if a state is a terminal state.
func (sm *StateMachine) IsTerminal(state State) bool {
	for _, s := range sm.Terminal {
		if s == state {
			return true
		}
	}
	return false
}

// Validate checks that the state machine definition is internally consistent.
// It verifies that:
//   - Initial state is in the States list
//   - All Terminal states are in the States list
//   - All Transition From and To states are in the States list
func (sm *StateMachine) Validate() error {
	stateSet := make(map[State]bool, len(sm.States))
	for _, s := range sm.States {
		stateSet[s] = true
	}

	if !stateSet[sm.Initial] {
		return fmt.Errorf("initial state %q is not in the states list", sm.Initial)
	}

	for _, ts := range sm.Terminal {
		if !stateSet[ts] {
			return fmt.Errorf("terminal state %q is not in the states list", ts)
		}
	}

	for _, t := range sm.Transitions {
		if !stateSet[t.From] {
			return fmt.Errorf("transition references undefined from-state %q", t.From)
		}
		if !stateSet[t.To] {
			return fmt.Errorf("transition references undefined to-state %q", t.To)
		}
	}

	return nil
}
