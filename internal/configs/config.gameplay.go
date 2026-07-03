package configs

type GamePlay struct {
	AllowItemBuffRemoval ConfigBool `yaml:"AllowItemBuffRemoval"`
	// Death related settings
	Death GameplayDeath `yaml:"Death"`

	// Shops/Conatiners
	ShopRestockRate  ConfigString `yaml:"ShopRestockRate"`  // Default time it takes to restock 1 quantity in shops
	ContainerSizeMax ConfigInt    `yaml:"ContainerSizeMax"` // How many objects containers can hold before overflowing
	// Alt chars
	MaxAltCharacters ConfigInt `yaml:"MaxAltCharacters"` // How many characters beyond the default character can they create?
	// PVP Restrictions
	PVP             ConfigString `yaml:"PVP"`
	PVPMinimumSkillRanks ConfigInt `yaml:"PVPMinimumSkillRanks"` // Minimum total skill ranks to engage in PVP

	// Skill Progression
	UseSkillProgression ConfigBool `yaml:"UseSkillProgression"` // Enable skill/stat progression checks on skill/stat use
	DualProgressionMode ConfigBool `yaml:"DualProgressionMode"` // When true, progression checks grant actual skill/stat increases (requires UseSkillProgression)

	// Cartesian map consistency enforcement
	MapConsistencyEnforce ConfigString `yaml:"MapConsistencyEnforce"` // "off" | "warn" (default) | "panic" — startup Cartesian-consistency enforcement level

	FerriesEnabled ConfigBool `yaml:"FerriesEnabled"` // Master toggle for the ferry vessel controller + boarding (Stage 1 ferry system)

	WarehousesEnabled ConfigBool `yaml:"WarehousesEnabled"` // Master toggle for warehouse buffer pools (Stage 3 ferry system)
}

type GameplayDeath struct {
	EquipmentDropChance ConfigFloat  `yaml:"EquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	AlwaysDropBackpack  ConfigBool   `yaml:"AlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	ProtectionSkillRanks ConfigInt  `yaml:"ProtectionSkillRanks"` // Total skill ranks below which death penalties are waived
	CorpsesEnabled      ConfigBool   `yaml:"CorpsesEnabled"`      // Whether corpses are left behind after mob/player deaths
	CorpseDecayTime     ConfigString `yaml:"CorpseDecayTime"`     // How long until corpses decay to dust (go away)
	// DOGMud death penalties (Stage 20.1)
	StatDecayMin          ConfigInt `yaml:"StatDecayMin"`          // Min Training loss on death (default 1)
	StatDecayMax          ConfigInt `yaml:"StatDecayMax"`          // Max Training loss on death (default 2)
	SkillRustCount        ConfigInt `yaml:"SkillRustCount"`        // Number of skills to decay on death (default 1)
	SkillRustAmount       ConfigInt `yaml:"SkillRustAmount"`       // Skill ranks lost per decayed skill (default 1)
	SkillRecencyThreshold ConfigInt `yaml:"SkillRecencyThreshold"` // Use count above which skills are protected (default 50)
	DeathsShadowBuffId    ConfigInt `yaml:"DeathsShadowBuffId"`    // Buff ID for Death's Shadow debuff (default 25)
	RespawnPoolFraction   ConfigFloat `yaml:"RespawnPoolFraction"` // Fraction of max pools (Health/Stamina/Conviction) restored on respawn (default 0.05). Keeps "death run" strategies honest — players respawn weakened and have to recover before their next attempt.
	RespawnGraceRounds    ConfigInt   `yaml:"RespawnGraceRounds"`  // Rounds of no-aggro-target protection after respawn (default 3). Set to 0 to disable grace period.
}

func (g *GamePlay) Validate() {

	// Ignore AllowItemBuffRemoval
	// Ignore OnDeathAlwaysDropBackpack
	// Ignore ConsistentAttackMessages
	// Ignore CorpsesEnabled

	if g.Death.EquipmentDropChance < 0.0 || g.Death.EquipmentDropChance > 1.0 {
		g.Death.EquipmentDropChance = 0.0 // default
	}

	if g.Death.RespawnPoolFraction <= 0.0 || g.Death.RespawnPoolFraction > 1.0 {
		g.Death.RespawnPoolFraction = 0.05 // default — respawn at 5% of max pools
	}

	if g.Death.RespawnGraceRounds < 0 {
		g.Death.RespawnGraceRounds = 3 // default — 3 rounds of grace protection
	}

	if g.Death.ProtectionSkillRanks < 0 {
		g.Death.ProtectionSkillRanks = 10 // default
	}

	if g.ShopRestockRate == `` {
		g.ShopRestockRate = `6 hours`
	}

	if g.ContainerSizeMax < 1 {
		g.ContainerSizeMax = 1
	}

	if g.MaxAltCharacters < 0 {
		g.MaxAltCharacters = 0
	}

	if g.Death.CorpseDecayTime == `` {
		g.Death.CorpseDecayTime = `1 hour`
	}

	// DOGMud death penalty defaults (Stage 20.1)
	if g.Death.StatDecayMin < 1 {
		g.Death.StatDecayMin = 1
	}
	if g.Death.StatDecayMax < g.Death.StatDecayMin {
		g.Death.StatDecayMax = 2
	}
	if g.Death.SkillRustCount < 0 {
		g.Death.SkillRustCount = 1
	}
	if g.Death.SkillRustAmount < 1 {
		g.Death.SkillRustAmount = 1
	}
	if g.Death.SkillRecencyThreshold < 1 {
		g.Death.SkillRecencyThreshold = 50
	}
	if g.Death.DeathsShadowBuffId < 1 {
		g.Death.DeathsShadowBuffId = 25
	}

	if g.PVP != PVPEnabled && g.PVP != PVPDisabled && g.PVP != PVPLimited {
		if g.PVP == PVPOff {
			g.PVP = PVPDisabled
		} else {
			g.PVP = PVPEnabled
		}
	}

	if int(g.PVPMinimumSkillRanks) < 0 {
		g.PVPMinimumSkillRanks = 0
	}

	switch string(g.MapConsistencyEnforce) {
	case "off", "warn", "panic":
		// valid
	default:
		g.MapConsistencyEnforce = "warn"
	}

}

func GetGamePlayConfig() GamePlay {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.GamePlay
}
