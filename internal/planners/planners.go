// Package planners contains chunk-4.4 per-goal-type planners that turn
// the current strategic goal (selected by chunk 4.2 from the chunk 4.3
// catalog) into concrete tactical actions per mob round tick.
//
// Each goal type's planner lives in its own <type>.go file. Each file's
// init() calls RegisterPlanner. main.go pulls these registrations via
// a blank import.
//
// Planners are stateless from the framework's perspective. For multi-
// step plans, write intermediate progress to mob.Character.MiscData
// under the convention "plan:<goal_type>:<key>". State is wiped
// automatically on goal switch via ClearPlanState (registered into
// goals.Recompute via SetPlanStateClear at boot).
//
// See docs/superpowers/specs/2026-05-27-mob-aliveness-4.4-strategic-tactical-translation-design.md
package planners

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// BTreeStatus mirrors the behavior-tree status enum (re-exported to
// avoid forcing planners to import internal/behaviortree, which would
// create a cycle).
type BTreeStatus int

const (
	StatusFailure BTreeStatus = iota
	StatusSuccess
	StatusRunning
)

// PlanResult is what a planner returns each tick.
type PlanResult struct {
	// Command to execute this tick (empty string = no action; btree falls
	// through to the next node). Executed via mob.Command(cmd) by the
	// try_goal_planner btree action.
	Command string

	// Status propagated as the try_goal_planner btree action's result.
	Status BTreeStatus
}

// PlanFn is the per-tick planner. Stateless from the framework's
// perspective — for multi-step plans, write to mob.Character.MiscData
// under "plan:<goal_type>:" prefix.
type PlanFn func(mob *mobs.Mob, goal *goals.Goal) PlanResult

var (
	registryMu sync.RWMutex
	registry   = map[string]PlanFn{}
)

// RegisterPlanner registers a planner for a goal type. Called from each
// per-type planner file's init() function. Late registrations overwrite
// earlier ones (last-write-wins; useful for test override).
func RegisterPlanner(goalType string, fn PlanFn) {
	registryMu.Lock()
	registry[goalType] = fn
	registryMu.Unlock()
}

// LookupPlanner returns the registered planner for a goal type, or nil
// if none. Called by the try_goal_planner btree action.
func LookupPlanner(goalType string) PlanFn {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[goalType]
}
