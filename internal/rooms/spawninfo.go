package rooms

type SpawnInfo struct {
	MobId        int      `yaml:"mobid,omitempty"`           // Mob template Id to spawn
	InstanceId   int      `yaml:"-"`                         // Mob instance Id that was spawned (tracks whether exists currently)
	Container    string   `yaml:"container,omitempty"`       // If set, any item or gold spawned will go into the container.
	ItemId       int      `yaml:"itemid,omitempty"`          // Item template Id to spawn on the floor
	Gold         int      `yaml:"gold,omitempty"`            // How much gold to spawn on the floor
	Message      string   `yaml:"message,omitempty"`         // (optional) message to display to the room when this creature spawns, instead of a default
	Name         string   `yaml:"name,omitempty"`            // (optional) if set, will override the mob's name
	ForceHostile bool     `yaml:"forcehostile,omitempty"`    // (optional) if true, forces the mob to be hostile.
	MaxWander    int      `yaml:"maxwander,omitempty"`       // (optional) if set, will override the mob's max wander distance
	IdleCommands []string `yaml:"idlecommands,omitempty"`    // (optional) list of commands to override the default of the mob. Useful when you need a mob to be more unique.
	ScriptTag    string   `yaml:"scripttag,omitempty"`       // (optional) if set, will override the mob's script tag
	QuestFlags   []string `yaml:"questflags,omitempty,flow"` // (optional) list of quest flags to set on the mob
	BuffIds      []int    `yaml:"buffids,omitempty,flow"`    // (optional) list of buffs the mob always has active
	StatPool     int      `yaml:"statpool,omitempty"`        // (optional) force this mob to a specific stat pool
	StatPoolMod  int      `yaml:"statpoolmod,omitempty"`     // (optional) modify this mob's stat pool by this amount
	// spawn tracking and rate
	DespawnedRound uint64 `yaml:"-"`                     // When this mob was last despawned (killed)
	RespawnRate    string `yaml:"respawnrate,omitempty"` // How long until it respawns when not present?
}
