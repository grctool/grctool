// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"testing"
)

// --- Validate edge cases ---

func TestStateMachine_Validate_EmptyStates(t *testing.T) {
	t.Parallel()

	sm := &StateMachine{
		EntityType:  "test",
		States:      []State{},
		Transitions: []Transition{},
		Initial:     "draft",
		Terminal:    []State{},
	}

	err := sm.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty states with non-empty initial, got nil")
	}
}

func TestStateMachine_Validate_NilTransitions(t *testing.T) {
	t.Parallel()

	sm := &StateMachine{
		EntityType:  "test",
		States:      []State{"a", "b"},
		Transitions: nil,
		Initial:     "a",
		Terminal:    []State{"b"},
	}

	err := sm.Validate()
	if err != nil {
		t.Errorf("expected no error for nil transitions (valid), got: %v", err)
	}
}

func TestStateMachine_Validate_DuplicateStates(t *testing.T) {
	t.Parallel()

	// Duplicate states should not cause a problem for Validate (it uses a set).
	sm := &StateMachine{
		EntityType:  "test",
		States:      []State{"a", "a", "b"},
		Transitions: []Transition{{From: "a", To: "b"}},
		Initial:     "a",
		Terminal:    []State{"b"},
	}

	err := sm.Validate()
	if err != nil {
		t.Errorf("expected no error with duplicate states, got: %v", err)
	}
}

func TestStateMachine_Validate_EmptyInitial(t *testing.T) {
	t.Parallel()

	sm := &StateMachine{
		EntityType:  "test",
		States:      []State{"a"},
		Transitions: []Transition{},
		Initial:     "",
		Terminal:    []State{},
	}

	err := sm.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty initial state, got nil")
	}
}

// --- IsValidState with empty string ---

func TestStateMachine_IsValidState_EmptyString(t *testing.T) {
	t.Parallel()

	machines := []*StateMachine{
		NewPolicyStateMachine(),
		NewControlStateMachine(),
		NewEvidenceTaskStateMachine(),
	}

	for _, sm := range machines {
		if sm.IsValidState("") {
			t.Errorf("%s: expected empty string to be invalid state", sm.EntityType)
		}
	}
}

// --- ValidTransitionsFrom for terminal states across all machines ---

func TestValidTransitionsFrom_TerminalStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		machine  *StateMachine
		terminal State
	}{
		{"policy retired", NewPolicyStateMachine(), PolicyRetired},
		{"control deprecated", NewControlStateMachine(), ControlDeprecated},
		{"evidence accepted", NewEvidenceTaskStateMachine(), EvidenceAccepted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transitions := tt.machine.ValidTransitionsFrom(tt.terminal)
			if len(transitions) != 0 {
				t.Errorf("expected 0 transitions from terminal state %q, got %v", tt.terminal, transitions)
			}
		})
	}
}

// --- Exhaustive invalid transition tests for all three state machines ---

func TestControlStateMachine_InvalidTransitions(t *testing.T) {
	t.Parallel()

	sm := NewControlStateMachine()

	invalid := []Transition{
		{From: ControlDefined, To: ControlTested},
		{From: ControlDefined, To: ControlEffective},
		{From: ControlDefined, To: ControlDeprecated},
		{From: ControlImplemented, To: ControlDefined},
		{From: ControlImplemented, To: ControlEffective},
		{From: ControlImplemented, To: ControlDeprecated},
		{From: ControlTested, To: ControlDefined},
		{From: ControlTested, To: ControlDeprecated},
		{From: ControlEffective, To: ControlDefined},
		{From: ControlEffective, To: ControlImplemented},
		{From: ControlDeprecated, To: ControlDefined},
		{From: ControlDeprecated, To: ControlImplemented},
		{From: ControlDeprecated, To: ControlTested},
		{From: ControlDeprecated, To: ControlEffective},
	}

	for _, tr := range invalid {
		if sm.CanTransition(tr.From, tr.To) {
			t.Errorf("expected transition %s -> %s to be invalid", tr.From, tr.To)
		}
	}
}

func TestEvidenceTaskStateMachine_InvalidTransitions(t *testing.T) {
	t.Parallel()

	sm := NewEvidenceTaskStateMachine()

	invalid := []Transition{
		{From: EvidenceScheduled, To: EvidenceCollected},
		{From: EvidenceScheduled, To: EvidenceValidated},
		{From: EvidenceScheduled, To: EvidenceSubmitted},
		{From: EvidenceScheduled, To: EvidenceAccepted},
		{From: EvidenceScheduled, To: EvidenceRejected},
		{From: EvidenceCollecting, To: EvidenceScheduled},
		{From: EvidenceCollecting, To: EvidenceValidated},
		{From: EvidenceCollecting, To: EvidenceSubmitted},
		{From: EvidenceCollecting, To: EvidenceAccepted},
		{From: EvidenceCollecting, To: EvidenceRejected},
		{From: EvidenceCollected, To: EvidenceScheduled},
		{From: EvidenceCollected, To: EvidenceSubmitted},
		{From: EvidenceCollected, To: EvidenceAccepted},
		{From: EvidenceCollected, To: EvidenceRejected},
		{From: EvidenceValidated, To: EvidenceScheduled},
		{From: EvidenceValidated, To: EvidenceCollecting},
		{From: EvidenceValidated, To: EvidenceCollected},
		{From: EvidenceValidated, To: EvidenceAccepted},
		{From: EvidenceValidated, To: EvidenceRejected},
		{From: EvidenceSubmitted, To: EvidenceScheduled},
		{From: EvidenceSubmitted, To: EvidenceCollecting},
		{From: EvidenceSubmitted, To: EvidenceCollected},
		{From: EvidenceSubmitted, To: EvidenceValidated},
		{From: EvidenceAccepted, To: EvidenceScheduled},
		{From: EvidenceAccepted, To: EvidenceCollecting},
		{From: EvidenceAccepted, To: EvidenceCollected},
		{From: EvidenceAccepted, To: EvidenceValidated},
		{From: EvidenceAccepted, To: EvidenceSubmitted},
		{From: EvidenceAccepted, To: EvidenceRejected},
		{From: EvidenceRejected, To: EvidenceScheduled},
		{From: EvidenceRejected, To: EvidenceCollected},
		{From: EvidenceRejected, To: EvidenceValidated},
		{From: EvidenceRejected, To: EvidenceSubmitted},
		{From: EvidenceRejected, To: EvidenceAccepted},
	}

	for _, tr := range invalid {
		if sm.CanTransition(tr.From, tr.To) {
			t.Errorf("expected transition %s -> %s to be invalid", tr.From, tr.To)
		}
	}
}

// --- Cross-machine state checks ---

func TestPolicyStatesNotValidInControlMachine(t *testing.T) {
	t.Parallel()

	sm := NewControlStateMachine()
	policyStates := []State{PolicyDraft, PolicyReview, PolicyApproved, PolicyPublished, PolicyRetired}

	for _, s := range policyStates {
		if sm.IsValidState(s) {
			t.Errorf("policy state %q should not be valid in control state machine", s)
		}
	}
}

// --- IsTerminal for non-terminal states ---

func TestIsTerminal_NonTerminalStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		machine *StateMachine
		states  []State
	}{
		{"policy non-terminal", NewPolicyStateMachine(), []State{PolicyDraft, PolicyReview, PolicyApproved, PolicyPublished}},
		{"control non-terminal", NewControlStateMachine(), []State{ControlDefined, ControlImplemented, ControlTested, ControlEffective}},
		{"evidence non-terminal", NewEvidenceTaskStateMachine(), []State{EvidenceScheduled, EvidenceCollecting, EvidenceCollected, EvidenceValidated, EvidenceSubmitted, EvidenceRejected}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, s := range tt.states {
				if tt.machine.IsTerminal(s) {
					t.Errorf("state %q should not be terminal", s)
				}
			}
		})
	}
}

// --- IsTerminal with unknown state ---

func TestIsTerminal_UnknownState(t *testing.T) {
	t.Parallel()

	sm := NewPolicyStateMachine()
	if sm.IsTerminal("nonexistent") {
		t.Error("expected unknown state to not be terminal")
	}
	if sm.IsTerminal("") {
		t.Error("expected empty string to not be terminal")
	}
}

// --- ValidTransitionsFrom with unknown state ---

func TestValidTransitionsFrom_UnknownState(t *testing.T) {
	t.Parallel()

	sm := NewPolicyStateMachine()
	transitions := sm.ValidTransitionsFrom("nonexistent")
	if len(transitions) != 0 {
		t.Errorf("expected 0 transitions from unknown state, got %v", transitions)
	}
}

// --- CanTransition with unknown states ---

func TestCanTransition_UnknownStates(t *testing.T) {
	t.Parallel()

	sm := NewPolicyStateMachine()
	if sm.CanTransition("nonexistent", PolicyReview) {
		t.Error("expected false for transition from unknown state")
	}
	if sm.CanTransition(PolicyDraft, "nonexistent") {
		t.Error("expected false for transition to unknown state")
	}
	if sm.CanTransition("foo", "bar") {
		t.Error("expected false for transition between unknown states")
	}
}
