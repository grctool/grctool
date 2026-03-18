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

import (
	"testing"
)

// --- Policy State Machine Tests ---

func TestPolicyStateMachine_ValidTransitions(t *testing.T) {
	sm := NewPolicyStateMachine()

	validTransitions := []Transition{
		{From: PolicyDraft, To: PolicyReview},
		{From: PolicyReview, To: PolicyApproved},
		{From: PolicyReview, To: PolicyDraft},
		{From: PolicyApproved, To: PolicyPublished},
		{From: PolicyPublished, To: PolicyReview},
		{From: PolicyPublished, To: PolicyRetired},
	}

	for _, tr := range validTransitions {
		if !sm.CanTransition(tr.From, tr.To) {
			t.Errorf("expected transition %s -> %s to be valid", tr.From, tr.To)
		}
	}
}

func TestPolicyStateMachine_InvalidTransitions(t *testing.T) {
	sm := NewPolicyStateMachine()

	invalidTransitions := []Transition{
		{From: PolicyDraft, To: PolicyPublished},   // skip states
		{From: PolicyDraft, To: PolicyApproved},     // skip review
		{From: PolicyDraft, To: PolicyRetired},      // skip all
		{From: PolicyApproved, To: PolicyDraft},     // no backward from approved to draft
		{From: PolicyRetired, To: PolicyDraft},      // terminal cannot transition
		{From: PolicyRetired, To: PolicyPublished},  // terminal cannot transition
	}

	for _, tr := range invalidTransitions {
		if sm.CanTransition(tr.From, tr.To) {
			t.Errorf("expected transition %s -> %s to be invalid", tr.From, tr.To)
		}
	}
}

func TestPolicyStateMachine_BackTransition(t *testing.T) {
	sm := NewPolicyStateMachine()

	// review -> draft (reject back) is allowed
	if !sm.CanTransition(PolicyReview, PolicyDraft) {
		t.Error("expected review -> draft (reject back) to be valid")
	}

	// published -> review (trigger re-review) is allowed
	if !sm.CanTransition(PolicyPublished, PolicyReview) {
		t.Error("expected published -> review (re-review) to be valid")
	}
}

func TestPolicyStateMachine_TerminalState(t *testing.T) {
	sm := NewPolicyStateMachine()

	if !sm.IsTerminal(PolicyRetired) {
		t.Error("expected retired to be a terminal state")
	}

	transitions := sm.ValidTransitionsFrom(PolicyRetired)
	if len(transitions) != 0 {
		t.Errorf("expected no outgoing transitions from retired, got %v", transitions)
	}
}

// --- Control State Machine Tests ---

func TestControlStateMachine_ValidTransitions(t *testing.T) {
	sm := NewControlStateMachine()

	// Full forward lifecycle
	validTransitions := []Transition{
		{From: ControlDefined, To: ControlImplemented},
		{From: ControlImplemented, To: ControlTested},
		{From: ControlTested, To: ControlEffective},
		{From: ControlEffective, To: ControlDeprecated},
		{From: ControlEffective, To: ControlTested},
		{From: ControlTested, To: ControlImplemented},
	}

	for _, tr := range validTransitions {
		if !sm.CanTransition(tr.From, tr.To) {
			t.Errorf("expected transition %s -> %s to be valid", tr.From, tr.To)
		}
	}
}

func TestControlStateMachine_FailedTesting(t *testing.T) {
	sm := NewControlStateMachine()

	// tested -> implemented (failed testing) is allowed
	if !sm.CanTransition(ControlTested, ControlImplemented) {
		t.Error("expected tested -> implemented (failed testing) to be valid")
	}
}

func TestControlStateMachine_Deprecated(t *testing.T) {
	sm := NewControlStateMachine()

	if !sm.IsTerminal(ControlDeprecated) {
		t.Error("expected deprecated to be a terminal state")
	}

	transitions := sm.ValidTransitionsFrom(ControlDeprecated)
	if len(transitions) != 0 {
		t.Errorf("expected no outgoing transitions from deprecated, got %v", transitions)
	}
}

// --- Evidence Task State Machine Tests ---

func TestEvidenceTaskStateMachine_ValidTransitions(t *testing.T) {
	sm := NewEvidenceTaskStateMachine()

	// Full forward lifecycle
	validTransitions := []Transition{
		{From: EvidenceScheduled, To: EvidenceCollecting},
		{From: EvidenceCollecting, To: EvidenceCollected},
		{From: EvidenceCollected, To: EvidenceValidated},
		{From: EvidenceValidated, To: EvidenceSubmitted},
		{From: EvidenceSubmitted, To: EvidenceAccepted},
		{From: EvidenceSubmitted, To: EvidenceRejected},
		// Back transitions
		{From: EvidenceCollected, To: EvidenceCollecting},
		{From: EvidenceRejected, To: EvidenceCollecting},
	}

	for _, tr := range validTransitions {
		if !sm.CanTransition(tr.From, tr.To) {
			t.Errorf("expected transition %s -> %s to be valid", tr.From, tr.To)
		}
	}
}

func TestEvidenceTaskStateMachine_Rejection(t *testing.T) {
	sm := NewEvidenceTaskStateMachine()

	// submitted -> rejected is valid
	if !sm.CanTransition(EvidenceSubmitted, EvidenceRejected) {
		t.Error("expected submitted -> rejected to be valid")
	}

	// rejected -> collecting (retry) is valid
	if !sm.CanTransition(EvidenceRejected, EvidenceCollecting) {
		t.Error("expected rejected -> collecting (retry) to be valid")
	}
}

// --- Generic State Machine Tests ---

func TestStateMachine_IsValidState(t *testing.T) {
	sm := NewPolicyStateMachine()

	// Known states
	for _, s := range sm.States {
		if !sm.IsValidState(s) {
			t.Errorf("expected state %q to be valid", s)
		}
	}

	// Unknown state
	if sm.IsValidState("nonexistent") {
		t.Error("expected state 'nonexistent' to be invalid")
	}

	if sm.IsValidState("") {
		t.Error("expected empty string state to be invalid")
	}
}

func TestStateMachine_IsTerminal(t *testing.T) {
	sm := NewPolicyStateMachine()

	// Terminal state
	if !sm.IsTerminal(PolicyRetired) {
		t.Error("expected retired to be terminal")
	}

	// Non-terminal states
	nonTerminal := []State{PolicyDraft, PolicyReview, PolicyApproved, PolicyPublished}
	for _, s := range nonTerminal {
		if sm.IsTerminal(s) {
			t.Errorf("expected state %q to NOT be terminal", s)
		}
	}
}

func TestStateMachine_ValidTransitionsFrom(t *testing.T) {
	sm := NewPolicyStateMachine()

	// Draft should only transition to review
	transitions := sm.ValidTransitionsFrom(PolicyDraft)
	if len(transitions) != 1 || transitions[0] != PolicyReview {
		t.Errorf("expected draft to have exactly [review], got %v", transitions)
	}

	// Review can go to approved or draft
	transitions = sm.ValidTransitionsFrom(PolicyReview)
	if len(transitions) != 2 {
		t.Errorf("expected review to have 2 transitions, got %d: %v", len(transitions), transitions)
	}

	// Published can go to review or retired
	transitions = sm.ValidTransitionsFrom(PolicyPublished)
	if len(transitions) != 2 {
		t.Errorf("expected published to have 2 transitions, got %d: %v", len(transitions), transitions)
	}

	// Retired has no outgoing transitions
	transitions = sm.ValidTransitionsFrom(PolicyRetired)
	if len(transitions) != 0 {
		t.Errorf("expected retired to have 0 transitions, got %d: %v", len(transitions), transitions)
	}
}

func TestStateMachine_Validate(t *testing.T) {
	// All well-formed machines should pass validation
	machines := []*StateMachine{
		NewPolicyStateMachine(),
		NewControlStateMachine(),
		NewEvidenceTaskStateMachine(),
	}

	for _, sm := range machines {
		if err := sm.Validate(); err != nil {
			t.Errorf("expected %s state machine to be valid, got error: %v", sm.EntityType, err)
		}
	}
}

func TestStateMachine_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		machine *StateMachine
		wantErr string
	}{
		{
			name: "transition references undefined from-state",
			machine: &StateMachine{
				EntityType: "test",
				States:     []State{"a", "b"},
				Transitions: []Transition{
					{From: "x", To: "b"}, // "x" not in States
				},
				Initial:  "a",
				Terminal: []State{"b"},
			},
			wantErr: "transition references undefined from-state",
		},
		{
			name: "transition references undefined to-state",
			machine: &StateMachine{
				EntityType: "test",
				States:     []State{"a", "b"},
				Transitions: []Transition{
					{From: "a", To: "z"}, // "z" not in States
				},
				Initial:  "a",
				Terminal: []State{"b"},
			},
			wantErr: "transition references undefined to-state",
		},
		{
			name: "initial state not in states",
			machine: &StateMachine{
				EntityType:  "test",
				States:      []State{"a", "b"},
				Transitions: []Transition{{From: "a", To: "b"}},
				Initial:     "missing",
				Terminal:    []State{"b"},
			},
			wantErr: "initial state",
		},
		{
			name: "terminal state not in states",
			machine: &StateMachine{
				EntityType:  "test",
				States:      []State{"a", "b"},
				Transitions: []Transition{{From: "a", To: "b"}},
				Initial:     "a",
				Terminal:    []State{"missing"},
			},
			wantErr: "terminal state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.machine.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestStateMachine_CanTransition_SameState(t *testing.T) {
	machines := []*StateMachine{
		NewPolicyStateMachine(),
		NewControlStateMachine(),
		NewEvidenceTaskStateMachine(),
	}

	for _, sm := range machines {
		for _, s := range sm.States {
			if sm.CanTransition(s, s) {
				t.Errorf("%s: expected self-loop %s -> %s to be invalid (no self-loops)", sm.EntityType, s, s)
			}
		}
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
