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

package cmd

import (
	"fmt"
	"strings"

	"github.com/grctool/grctool/internal/lifecycle"
	"github.com/spf13/cobra"
)

// validEntityTypes lists the entity types accepted by lifecycle commands.
var validEntityTypes = map[string]bool{
	"policy":        true,
	"control":       true,
	"evidence_task": true,
}

// lifecycleCmd is the parent command for lifecycle management.
var lifecycleCmd = &cobra.Command{
	Use:   "lifecycle",
	Short: "Manage compliance entity lifecycles",
	Long: `Manage lifecycle states for compliance entities (policies, controls, evidence tasks).

Each entity type has a defined state machine with valid states and transitions.
Use subcommands to view status and manually transition entities between states.`,
}

// lifecycleStatusCmd shows lifecycle states.
var lifecycleStatusCmd = &cobra.Command{
	Use:   "status [entity-type]",
	Short: "Show lifecycle status of entities",
	Long: `Show the lifecycle state machine for an entity type, including all valid states
and transitions. If no entity type is specified, shows all entity types.

Valid entity types: policy, control, evidence_task`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLifecycleStatus,
}

// lifecycleTransitionCmd manually transitions an entity.
var lifecycleTransitionCmd = &cobra.Command{
	Use:   "transition <entity-type> <entity-id> <new-state>",
	Short: "Transition an entity to a new lifecycle state",
	Long: `Manually transition a compliance entity to a new lifecycle state.

The transition is validated against the state machine to ensure it is allowed.
Currently shows what would happen (actual persistence wired in orchestration bead).

Valid entity types: policy, control, evidence_task`,
	Args: cobra.ExactArgs(3),
	RunE: runLifecycleTransition,
}

func init() {
	rootCmd.AddCommand(lifecycleCmd)
	lifecycleCmd.AddCommand(lifecycleStatusCmd)
	lifecycleCmd.AddCommand(lifecycleTransitionCmd)
}

// getStateMachine returns the state machine for the given entity type.
func getStateMachine(entityType string) (*lifecycle.StateMachine, error) {
	switch entityType {
	case "policy":
		return lifecycle.NewPolicyStateMachine(), nil
	case "control":
		return lifecycle.NewControlStateMachine(), nil
	case "evidence_task":
		return lifecycle.NewEvidenceTaskStateMachine(), nil
	default:
		valid := make([]string, 0, len(validEntityTypes))
		for k := range validEntityTypes {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("unknown entity type %q (valid types: %s)", entityType, strings.Join(valid, ", "))
	}
}

func runLifecycleStatus(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	if len(args) == 1 {
		return printStateMachine(cmd, args[0])
	}

	// Show all entity types.
	for _, et := range []string{"policy", "control", "evidence_task"} {
		if err := printStateMachine(cmd, et); err != nil {
			return err
		}
		fmt.Fprintln(out)
	}

	return nil
}

func printStateMachine(cmd *cobra.Command, entityType string) error {
	out := cmd.OutOrStdout()

	sm, err := getStateMachine(entityType)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "=== %s Lifecycle ===\n", entityType)
	fmt.Fprintf(out, "Initial state: %s\n", sm.Initial)
	fmt.Fprintf(out, "Terminal states: %s\n", joinStates(sm.Terminal))
	fmt.Fprintln(out)

	fmt.Fprintln(out, "States:")
	for _, s := range sm.States {
		marker := " "
		if s == sm.Initial {
			marker = ">"
		}
		if sm.IsTerminal(s) {
			marker = "*"
		}
		transitions := sm.ValidTransitionsFrom(s)
		if len(transitions) > 0 {
			fmt.Fprintf(out, "  %s %s -> %s\n", marker, s, joinStates(transitions))
		} else {
			fmt.Fprintf(out, "  %s %s (terminal)\n", marker, s)
		}
	}

	return nil
}

func runLifecycleTransition(cmd *cobra.Command, args []string) error {
	entityType := args[0]
	entityID := args[1]
	newState := args[2]

	out := cmd.OutOrStdout()

	// Validate entity type.
	if !validEntityTypes[entityType] {
		valid := make([]string, 0, len(validEntityTypes))
		for k := range validEntityTypes {
			valid = append(valid, k)
		}
		return fmt.Errorf("unknown entity type %q (valid types: %s)", entityType, strings.Join(valid, ", "))
	}

	sm, err := getStateMachine(entityType)
	if err != nil {
		return err
	}

	targetState := lifecycle.State(newState)

	// Validate target state exists.
	if !sm.IsValidState(targetState) {
		return fmt.Errorf("invalid state %q for entity type %q (valid states: %s)",
			newState, entityType, joinStates(sm.States))
	}

	// For now, we assume the entity is in the initial state if we don't have
	// persisted state. In practice, the orchestration layer will look up the
	// actual current state from entity metadata.
	// TODO(c6w.3): Look up actual current state from entity metadata.
	currentState := sm.Initial

	if !sm.CanTransition(currentState, targetState) {
		validNext := sm.ValidTransitionsFrom(currentState)
		return fmt.Errorf("cannot transition %s %s from %q to %q (valid transitions: %s)",
			entityType, entityID, currentState, newState, joinStates(validNext))
	}

	// TODO(c6w.3): Actually persist the state change.
	fmt.Fprintf(out, "Transitioning %s %s: %s -> %s\n", entityType, entityID, currentState, targetState)
	fmt.Fprintf(out, "  Transition validated successfully.\n")

	return nil
}

// joinStates formats a slice of States as a comma-separated string.
func joinStates(states []lifecycle.State) string {
	parts := make([]string, len(states))
	for i, s := range states {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}
