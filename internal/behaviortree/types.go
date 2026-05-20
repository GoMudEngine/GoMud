package behaviortree

import "github.com/GoMudEngine/GoMud/internal/state"

// Result is the return value of a node evaluation.
type Result int

const (
	Success Result = iota
	Failure
	Running
)

// EventContext carries information about the triggering event.
type EventContext struct {
	EventType string         // "player_ask", "mob_idle", "player_give", etc.
	UserId    int            // Triggering player (0 if none)
	MobId     int            // Triggering mob instance (0 if none)
	Text      string         // For ask/say events — the text spoken
	ItemId    int            // For give/show events — the item
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
	Event      EventContext
	MobState   *BehaviorState
	MobId      int    // Mob template ID
	InstanceId int    // Mob instance ID
	RoomId     int    // Current room
	MobName    string // Mob's display name
	Intercepted bool  // Whether the command was intercepted by a behavior tree

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
type NodeDef struct {
	Type     string         `yaml:"type"`
	Event    string         `yaml:"event,omitempty"`
	Children []NodeDef      `yaml:"children,omitempty"`
	Check    string         `yaml:"check,omitempty"`
	Do       string         `yaml:"do,omitempty"`
	Mod      string         `yaml:"mod,omitempty"`
	Child    *NodeDef       `yaml:"child,omitempty"`
	Params   map[string]any `yaml:",inline"`
}

// TreeDef is the top-level YAML structure.
type TreeDef struct {
	Tree NodeDef `yaml:"tree"`
}
