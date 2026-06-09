package configs

type LootGoblin struct {
	// Item/floor cleanup
	RoomId             ConfigInt  `yaml:"RoomId"`             // The room the loot goblin spawns in
	RoundCount         ConfigInt  `yaml:"RoundCount"`         // How often to spawn a loot goblin
	MinimumItems       ConfigInt  `yaml:"MinimumItems"`       // How many items on the ground to attract the loot goblin
	MinimumGold        ConfigInt  `yaml:"MinimumGold"`        // How much gold on the ground to attract the loot goblin
	IncludeRecentRooms ConfigBool `yaml:"IncludeRecentRooms"` // should the goblin include rooms that have been visited recently?

}

func (l *LootGoblin) Validate() {

	// Ignore RoomId
	// Ignore IncludeRecentRooms

	if l.RoundCount < 10 {
		l.RoundCount = 10 // default
	}

	if l.MinimumItems < 1 {
		l.MinimumItems = 2 // default
	}

	if l.MinimumGold < 1 {
		l.MinimumGold = 100 // default
	}

}

func GetLootGoblinConfig() LootGoblin {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.LootGoblin
}
