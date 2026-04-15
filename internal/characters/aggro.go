package characters

type AggroType int

const (
	// Enumerated Aggro Types
	DefaultAttack AggroType = iota // Regular H2H combat, everything can decay to this. Starts at zero
	Shooting
	SurpriseAttack
	SpellCast
	Flee
)

type SpellAggroInfo struct {
	SpellId              string
	SpellRest            string
	TargetUserIds        []int
	TargetMobInstanceIds []int
}

type Aggro struct {
	Type          AggroType
	MobInstanceId int
	UserId        int
	SpellInfo     SpellAggroInfo // If Type is SpellCast, this is the spell info
	ExitName      string         // For example, firing a weapon in a direction
	RoundsWaiting int            // How many rounds must pass before this triggers
}
