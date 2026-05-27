// Package goals — chunk 4.1 of the mob aliveness roadmap.
//
// Provides a persistent, queryable list of typed goals per mob
// template. Ships with an empty predicate/conflict registry; chunk 4.3
// fills it with concrete goal types. No behavior-tree integration yet
// (chunk 4.4).
package goals

import (
	"errors"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// Goal is one strategic intent owned by a mob template. Immutable once
// added — "updates" go through Remove + Add.
type Goal struct {
	Id         string         `yaml:"id"`
	OwnerMobId int            `yaml:"-"` // stamped at load time from MobGoals.MobId
	Type       string         `yaml:"type"`
	Priority   int            `yaml:"priority"`
	Params     map[string]any `yaml:"params,omitempty"`
	CreatedAt  time.Time      `yaml:"created_at"`
	ExpiresAt  time.Time      `yaml:"expires_at,omitempty"`
}

// MobGoals is the on-disk shape — one file per mob template.
type MobGoals struct {
	MobId             int     `yaml:"mob_id"`
	NextGoalId        int     `yaml:"next_goal_id"`
	CurrentGoalId     string  `yaml:"current_goal_id,omitempty"`     // chunk 4.2 — selection state
	CurrentSinceRound uint64  `yaml:"current_since_round,omitempty"` // chunk 4.2 — round when current became current
	LastSwitchRound   uint64  `yaml:"last_switch_round,omitempty"`   // chunk 4.2 — round of most recent switch
	Goals             []*Goal `yaml:"goals"`
}

// PredicateFn evaluates whether a goal is currently satisfied. Same
// goal + same mob state → same answer. No side effects — IsSatisfied
// may be called from any context.
type PredicateFn func(g *Goal, mob *mobs.Mob) bool

// ContextScoreFn returns a non-negative multiplier for a goal in the
// current mob context. 0 effectively suppresses the goal from selection
// this tick (e.g. "revenge target not in zone"). Must be pure(ish):
// same goal + same mob state → same answer. Side effects forbidden —
// may be called from any context.
//
// Chunk 4.2 — 4.3 will register concrete implementations per goal type.
type ContextScoreFn func(g *Goal, mob *mobs.Mob) float64

// ParamSchema declares one expected key on Goal.Params for type-aware
// validation. Used by ValidateParams (chunk 4.3).
type ParamSchema struct {
	Key      string
	Required bool
	GoType   string // "int" | "string" | "[]string" | "float64" | "bool"
}

// ErrBadParams is returned by Add when a goal's params don't match its
// type's declared ParamSchema.
type ErrBadParams struct {
	Key          string
	ExpectedType string
	GotType      string
	Reason       string // optional — e.g. "missing required key"
}

func (e *ErrBadParams) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("goals.ErrBadParams: key=%q %s", e.Key, e.Reason)
	}
	return fmt.Sprintf("goals.ErrBadParams: key=%q expected=%s got=%s",
		e.Key, e.ExpectedType, e.GotType)
}

// GoalTypeMeta is registered once per goal type by chunk 4.3's catalog.
type GoalTypeMeta struct {
	Predicate     PredicateFn
	ConflictsWith []string       // type names this goal type conflicts with
	ContextScore  ContextScoreFn // chunk 4.2 — optional; nil = always 1.0
	Params        []ParamSchema  // chunk 4.3 — optional; nil = no validation
}

// AddResult reports what happened on a successful Add.
type AddResult struct {
	Added     *Goal    // the newly-added goal (Id assigned)
	Displaced []string // goal ids removed because new one preempted
}

// ConflictError is returned by Add when an existing goal blocks the
// new one (priority ≤ existing's). Carries the blocker so the admin
// command can render "Blocked by goal <id> (type=<t>, priority=<p>)".
type ConflictError struct {
	BlockerGoalId string
	BlockerType   string
	BlockerPrio   int
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("goals: blocked by goal %s (type=%s, priority=%d)",
		e.BlockerGoalId, e.BlockerType, e.BlockerPrio)
}

// Sentinel returned by Remove when the goal id wasn't found. Tests can
// match it via errors.Is. Production callers usually ignore Remove's
// error since "remove what's not there" is a no-op.
var ErrGoalNotFound = errors.New("goals: goal id not found")
