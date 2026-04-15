package mobai

// TacticRule is a single priority-ordered behavior rule defined in mob YAML.
type TacticRule struct {
	Trigger  string `yaml:"trigger"`  // e.g. "target_casting", "health_below:30"
	Action   string `yaml:"action"`   // e.g. "trip", "cast conviction-spike", "flee"
	Priority int    `yaml:"priority"` // Higher = evaluated first
}

// CombatMemory tracks who the mob was fighting across flee/re-engage cycles.
type CombatMemory struct {
	TargetUserId   int    // Player they were fighting
	TargetMobId    int    // Or mob they were fighting
	LastSeenRoomId int    // Where the target was last seen
	LastSeenRound  uint64 // When they last saw the target
	Grudge         bool   // Should they pursue?
}

// PendingReaction is a queued tactical reaction waiting to fire.
type PendingReaction struct {
	MobInstanceId int
	Action        string // Command to execute
	FireTurn      uint64 // Turn number when this should execute
}
