package configs

import (
	"strconv"
	"strings"
)

type GamePlay struct {
	AllowItemBuffRemoval ConfigBool `yaml:"AllowItemBuffRemoval"`
	// Death related settings
	Death GameplayDeath `yaml:"Death"`

	LivesStart     ConfigInt `yaml:"LivesStart"`     // Starting permadeath lives
	LivesMax       ConfigInt `yaml:"LivesMax"`       // Maximum permadeath lives
	LivesOnLevelUp ConfigInt `yaml:"LivesOnLevelUp"` // # lives gained on level up
	PricePerLife   ConfigInt `yaml:"PricePerLife"`   // Price in gold to buy new lives
	// Shops/Conatiners
	ShopRestockRate  ConfigString `yaml:"ShopRestockRate"`  // Default time it takes to restock 1 quantity in shops
	ContainerSizeMax ConfigInt    `yaml:"ContainerSizeMax"` // How many objects containers can hold before overflowing
	// Alt chars
	MaxAltCharacters ConfigInt `yaml:"MaxAltCharacters"` // How many characters beyond the default character can they create?
	// Combat
	ConsistentAttackMessages ConfigBool `yaml:"ConsistentAttackMessages"` // Whether each weapon has consistent attack messages

	// PVP Restrictions
	PVP             ConfigString `yaml:"PVP"`
	PVPMinimumLevel ConfigInt    `yaml:"PVPMinimumLevel"`
	// XpScale (difficulty)
	XPScale           ConfigFloat `yaml:"XPScale"`
	MobConverseChance ConfigInt `yaml:"MobConverseChance"` // Chance 1-100 of attempting to converse when idle

	// Skill Progression
	UseSkillProgression ConfigBool `yaml:"UseSkillProgression"` // Enable skill/stat progression checks (messages only until Stage 3.3)
}

type GameplayDeath struct {
	EquipmentDropChance ConfigFloat  `yaml:"EquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	AlwaysDropBackpack  ConfigBool   `yaml:"AlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	XPPenalty           ConfigString `yaml:"XPPenalty"`           // Possible values are: none, level, 10%, 25%, 50%, 75%, 90%, 100%
	ProtectionLevels    ConfigInt    `yaml:"ProtectionLevels"`    // How many levels is the user protected from death penalties for?
	PermaDeath          ConfigBool   `yaml:"PermaDeath"`          // Is permadeath enabled?
	CorpsesEnabled      ConfigBool   `yaml:"CorpsesEnabled"`      // Whether corpses are left behind after mob/player deaths
	CorpseDecayTime     ConfigString `yaml:"CorpseDecayTime"`     // How long until corpses decay to dust (go away)
}

func (g *GamePlay) Validate() {

	// Ignore AllowItemBuffRemoval
	// Ignore OnDeathAlwaysDropBackpack
	// Ignore ConsistentAttackMessages
	// Ignore CorpsesEnabled

	if g.Death.EquipmentDropChance < 0.0 || g.Death.EquipmentDropChance > 1.0 {
		g.Death.EquipmentDropChance = 0.0 // default
	}

	g.Death.XPPenalty.Set(strings.ToLower(string(g.Death.XPPenalty)))

	if g.Death.XPPenalty != `none` && g.Death.XPPenalty != `level` {
		// If not a valid percent, set to default
		if !strings.HasSuffix(string(g.Death.XPPenalty), `%`) {
			g.Death.XPPenalty = `none` // default
		} else {
			// If not a valid percent, set to default
			percent, err := strconv.ParseInt(string(g.Death.XPPenalty)[0:len(g.Death.XPPenalty)-1], 10, 64)
			if err != nil || percent < 0 || percent > 100 {
				g.Death.XPPenalty = `none` // default
			}
		}
	}

	if g.Death.ProtectionLevels < 0 {
		g.Death.ProtectionLevels = 0 // default
	}

	if g.LivesStart < 0 {
		g.LivesStart = 0
	}

	if g.LivesMax < 0 {
		g.LivesMax = 0
	}

	if g.LivesOnLevelUp < 0 {
		g.LivesOnLevelUp = 0
	}

	if g.PricePerLife < 1 {
		g.PricePerLife = 1
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

	if g.PVP != PVPEnabled && g.PVP != PVPDisabled && g.PVP != PVPLimited {
		if g.PVP == PVPOff {
			g.PVP = PVPDisabled
		} else {
			g.PVP = PVPEnabled
		}
	}

	if int(g.PVPMinimumLevel) < 0 {
		g.PVPMinimumLevel = 0
	}

	if g.XPScale <= 0 {
		g.XPScale = 100
	}

	if g.MobConverseChance < 0 {
		g.MobConverseChance = 0
	} else if g.MobConverseChance > 100 {
		g.MobConverseChance = 100
	}

}

func GetGamePlayConfig() GamePlay {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.GamePlay
}
