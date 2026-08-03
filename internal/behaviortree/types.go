package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/uuid"
)

// Result is the return value of a node evaluation.
type Result int

const (
	Success Result = iota
	Failure
	Running
)

// EventContext carries information about the triggering event.
type EventContext struct {
	EventType string // "player_ask", "mob_idle", "player_give", etc.
	UserId    int    // Triggering player (0 if none)
	MobId     int    // Triggering mob instance (0 if none)
	Text      string // For ask/say events — the text spoken
	ItemId    int    // For give/show events — the item template id
	// ItemUUID identifies the EXACT item instance the event refers to, so
	// handlers (e.g. return_item) can operate on the real, stateful object
	// rather than a fresh copy minted from ItemId. Zero when the producer
	// could not supply one — consumers must fall back to an ItemId match.
	ItemUUID  uuid.UUID
	RoomId    int            // Room where event occurred
	Extra     map[string]any // Extensible context
	Command   string         // Command name for room command interception
	Rest      string         // Command arguments
	Direction string         // Direction for movement events
}

// Node is the interface all behavior tree nodes implement.
type Node interface {
	// Evaluate runs this node with the given context.
	Evaluate(ctx *EvalContext) Result
}

// EvalContext bundles everything a node needs during evaluation.
type EvalContext struct {
	Event       EventContext
	MobState    *BehaviorState
	MobId       int    // Mob template ID
	InstanceId  int    // Mob instance ID
	RoomId      int    // Current room
	MobName     string // Mob's display name
	Intercepted bool   // Whether the command was intercepted by a behavior tree

	// SoftTarget is a non-combat target slot used by archetypes
	// that pick targets for skullduggery (steal/plant/shadow) WITHOUT
	// entering combat. Set by target-picker actions like
	// target_random_player_in_room; read by try_steal/try_plant/try_shadow.
	//
	// CRITICAL DESIGN: Combat target lives on Character.CombatPhase's
	// Engaged state ONLY. To "pick a target without engaging combat,"
	// callers stash it here and pass it to actions as a parameter.
	// Setting Aggro for non-combat purposes is the chunk-2.7 bug class
	// that this slot exists to prevent.
	SoftTarget state.ActorRef
}

// NodeDef is the raw YAML definition of a node, parsed before
// being compiled into a Node.
//
// Note is a durable per-node home for design rationale (5d editor):
// marshal-based writers drop `#` comments, so rationale that should
// survive an editor save lives here. Stripped from runtime params by
// cleanParams (it's in knownFields).
//
// json tags mirror the yaml names — the 5d editor's wire contract.
// encoding/json has no inline, so on the wire Params travels as an
// explicit "params" object; the yaml marshal re-inlines it.
type NodeDef struct {
	Type     string         `yaml:"type" json:"type"`
	Event    string         `yaml:"event,omitempty" json:"event,omitempty"`
	Children []NodeDef      `yaml:"children,omitempty" json:"children,omitempty"`
	Check    string         `yaml:"check,omitempty" json:"check,omitempty"`
	Do       string         `yaml:"do,omitempty" json:"do,omitempty"`
	Mod      string         `yaml:"mod,omitempty" json:"mod,omitempty"`
	Note     string         `yaml:"note,omitempty" json:"note,omitempty"`
	Child    *NodeDef       `yaml:"child,omitempty" json:"child,omitempty"`
	Params   map[string]any `yaml:",inline" json:"params,omitempty"`
}

// TreeDef is the top-level YAML structure for archetype + room +
// per-mob behavior trees.
//
// GoalWeights is chunk-4.2 archetype metadata: per-goal-type score
// multipliers consumed by the goals package's Select function via a
// registered weights-lookup callback. Optional; missing or empty map
// means selection scores at default 1.0 for every goal type.
//
// DefaultGoals is chunk-4.3 archetype metadata: default goals seeded
// onto fresh mobs whose template references this archetype. Consumed
// by internal/goals/ via the SetArchetypeDefaultsLookup callback.
// Notes is the file-level counterpart of NodeDef.Note (5d editor).
type TreeDef struct {
	Notes        string             `yaml:"notes,omitempty" json:"notes,omitempty"`
	Tree         NodeDef            `yaml:"tree" json:"tree"`
	GoalWeights  map[string]float64 `yaml:"goal_weights,omitempty" json:"goal_weights,omitempty"`   // chunk 4.2
	DefaultGoals []GoalDefault      `yaml:"default_goals,omitempty" json:"default_goals,omitempty"` // chunk 4.3
}

// GoalDefault declares one default goal to seed on a fresh mob whose
// template uses this archetype. Consumed by internal/goals/ via the
// SetArchetypeDefaultsLookup callback. Chunk 4.3.
type GoalDefault struct {
	Type     string         `yaml:"type" json:"type"`
	Priority int            `yaml:"priority" json:"priority"`
	Params   map[string]any `yaml:"params,omitempty" json:"params,omitempty"`
}
