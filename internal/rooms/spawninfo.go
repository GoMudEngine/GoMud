package rooms

// SpawnInfo is one entry in a room's spawn list.
//
// The json tags exist for the admin web builder, which edits spawn lists as
// part of the room payload. Without them Go marshals the capitalised field
// names, leaving this the only payload in the builder not in camelCase — a
// trap for anyone writing client code against it. The yaml tags remain the
// authored on-disk shape and are unaffected.
type SpawnInfo struct {
	MobId        int      `yaml:"mobid,omitempty" json:"mobId,omitempty"`                // Mob template Id to spawn
	InstanceId   int      `yaml:"-" json:"instanceId,omitempty"`                         // Mob instance Id that was spawned (tracks whether exists currently)
	Container    string   `yaml:"container,omitempty" json:"container,omitempty"`        // If set, any item or gold spawned will go into the container.
	ItemId       int      `yaml:"itemid,omitempty" json:"itemId,omitempty"`              // Item template Id to spawn on the floor
	Gold         int      `yaml:"gold,omitempty" json:"gold,omitempty"`                  // How much gold to spawn on the floor
	Message      string   `yaml:"message,omitempty" json:"message,omitempty"`            // (optional) message to display to the room when this creature spawns, instead of a default
	Name         string   `yaml:"name,omitempty" json:"name,omitempty"`                  // (optional) if set, will override the mob's name
	ForceHostile bool     `yaml:"forcehostile,omitempty" json:"forceHostile,omitempty"`  // (optional) if true, forces the mob to be hostile.
	MaxWander    int      `yaml:"maxwander,omitempty" json:"maxWander,omitempty"`        // (optional) if set, will override the mob's max wander distance
	IdleCommands []string `yaml:"idlecommands,omitempty" json:"idleCommands,omitempty"`  // (optional) list of commands to override the default of the mob. Useful when you need a mob to be more unique.
	ScriptTag    string   `yaml:"scripttag,omitempty" json:"scriptTag,omitempty"`        // (optional) if set, will override the mob's script tag
	QuestFlags   []string `yaml:"questflags,omitempty,flow" json:"questFlags,omitempty"` // (optional) list of quest flags to set on the mob
	BuffIds      []int    `yaml:"buffids,omitempty,flow" json:"buffIds,omitempty"`       // (optional) list of buffs the mob always has active
	StatPool     int      `yaml:"statpool,omitempty" json:"statPool,omitempty"`          // (optional) force this mob to a specific stat pool
	StatPoolMod  int      `yaml:"statpoolmod,omitempty" json:"statPoolMod,omitempty"`    // (optional) modify this mob's stat pool by this amount
	// spawn tracking and rate
	DespawnedRound uint64 `yaml:"-" json:"despawnedRound,omitempty"`                  // When this mob was last despawned (killed)
	RespawnRate    string `yaml:"respawnrate,omitempty" json:"respawnRate,omitempty"` // How long until it respawns when not present?
}
