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

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/lifecycle"
	"github.com/grctool/grctool/internal/storage"
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
The new state is persisted to the entity's storage file.

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

	entityTypes := []string{"policy", "control", "evidence_task"}
	if len(args) == 1 {
		entityTypes = []string{args[0]}
	}

	for i, et := range entityTypes {
		if err := printStateMachine(cmd, et); err != nil {
			return err
		}
		// Try to show stored entity states if storage is available.
		printEntityStates(cmd, et)
		if i < len(entityTypes)-1 {
			fmt.Fprintln(out)
		}
	}

	return nil
}

// printEntityStates loads stored entities and prints their lifecycle states.
func printEntityStates(cmd *cobra.Command, entityType string) {
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return // silently skip if config not available
	}

	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return
	}

	sm, err := getStateMachine(entityType)
	if err != nil {
		return
	}

	type entityInfo struct {
		id    string
		name  string
		state string
	}

	var entities []entityInfo

	switch entityType {
	case "policy":
		policies, err := store.GetAllPolicies()
		if err != nil || len(policies) == 0 {
			return
		}
		for _, p := range policies {
			state := p.LifecycleState
			if state == "" {
				state = string(sm.Initial)
			}
			ref := p.ReferenceID
			if ref == "" {
				ref = p.ID
			}
			entities = append(entities, entityInfo{id: ref, name: p.Name, state: state})
		}
	case "control":
		controls, err := store.GetAllControls()
		if err != nil || len(controls) == 0 {
			return
		}
		for _, c := range controls {
			state := c.LifecycleState
			if state == "" {
				state = string(sm.Initial)
			}
			ref := c.ReferenceID
			if ref == "" {
				ref = c.ID
			}
			entities = append(entities, entityInfo{id: ref, name: c.Name, state: state})
		}
	case "evidence_task":
		tasks, err := store.GetAllEvidenceTasks()
		if err != nil || len(tasks) == 0 {
			return
		}
		for _, t := range tasks {
			state := t.LifecycleState
			if state == "" {
				state = string(sm.Initial)
			}
			ref := t.ReferenceID
			if ref == "" {
				ref = t.ID
			}
			entities = append(entities, entityInfo{id: ref, name: t.Name, state: state})
		}
	}

	if len(entities) == 0 {
		return
	}

	// Count entities by state.
	stateCounts := make(map[string]int)
	for _, e := range entities {
		stateCounts[e.state]++
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Entity states (%d %ss):\n", len(entities), entityType)
	for _, s := range sm.States {
		count := stateCounts[string(s)]
		if count > 0 {
			fmt.Fprintf(out, "  %s: %d\n", s, count)
		}
	}
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

	// Load config and storage to read/write entity state.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Look up current lifecycle state from the entity.
	currentState, err := getEntityLifecycleState(store, entityType, entityID, sm)
	if err != nil {
		return err
	}

	if !sm.CanTransition(currentState, targetState) {
		validNext := sm.ValidTransitionsFrom(currentState)
		return fmt.Errorf("cannot transition %s %s from %q to %q (valid transitions: %s)",
			entityType, entityID, currentState, newState, joinStates(validNext))
	}

	// Persist the state change.
	if err := setEntityLifecycleState(store, entityType, entityID, string(targetState)); err != nil {
		return err
	}

	fmt.Fprintf(out, "Transitioned %s %s: %s -> %s\n", entityType, entityID, currentState, targetState)

	return nil
}

// getEntityLifecycleState reads the lifecycle state for an entity from storage.
// If the entity has no lifecycle state set, returns the state machine's initial state.
func getEntityLifecycleState(store *storage.Storage, entityType, entityID string, sm *lifecycle.StateMachine) (lifecycle.State, error) {
	switch entityType {
	case "policy":
		p, err := store.GetPolicy(entityID)
		if err != nil {
			return "", fmt.Errorf("failed to get policy %s: %w", entityID, err)
		}
		if p.LifecycleState != "" {
			return lifecycle.State(p.LifecycleState), nil
		}
		return sm.Initial, nil
	case "control":
		c, err := store.GetControl(entityID)
		if err != nil {
			return "", fmt.Errorf("failed to get control %s: %w", entityID, err)
		}
		if c.LifecycleState != "" {
			return lifecycle.State(c.LifecycleState), nil
		}
		return sm.Initial, nil
	case "evidence_task":
		t, err := store.GetEvidenceTask(entityID)
		if err != nil {
			return "", fmt.Errorf("failed to get evidence task %s: %w", entityID, err)
		}
		if t.LifecycleState != "" {
			return lifecycle.State(t.LifecycleState), nil
		}
		return sm.Initial, nil
	default:
		return "", fmt.Errorf("unknown entity type: %s", entityType)
	}
}

// setEntityLifecycleState writes the lifecycle state for an entity to storage.
func setEntityLifecycleState(store *storage.Storage, entityType, entityID, state string) error {
	switch entityType {
	case "policy":
		p, err := store.GetPolicy(entityID)
		if err != nil {
			return fmt.Errorf("failed to get policy %s: %w", entityID, err)
		}
		p.LifecycleState = state
		return store.SavePolicy(p)
	case "control":
		c, err := store.GetControl(entityID)
		if err != nil {
			return fmt.Errorf("failed to get control %s: %w", entityID, err)
		}
		c.LifecycleState = state
		return store.SaveControl(c)
	case "evidence_task":
		t, err := store.GetEvidenceTask(entityID)
		if err != nil {
			return fmt.Errorf("failed to get evidence task %s: %w", entityID, err)
		}
		t.LifecycleState = state
		return store.SaveEvidenceTask(t)
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}
}

// joinStates formats a slice of States as a comma-separated string.
func joinStates(states []lifecycle.State) string {
	parts := make([]string, len(states))
	for i, s := range states {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}
