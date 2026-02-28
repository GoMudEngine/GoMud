package configs

type GamePlay struct {
	AllowItemBuffRemoval ConfigBool `yaml:"AllowItemBuffRemoval"`
	// Death related settings
	Death GameplayDeath `yaml:"Death"`

	LivesStart     ConfigInt `yaml:"LivesStart"`     // Starting permadeath lives
	LivesMax       ConfigInt `yaml:"LivesMax"`       // Maximum permadeath lives
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
	PVPMinimumSkillRanks ConfigInt `yaml:"PVPMinimumSkillRanks"` // Minimum total skill ranks to engage in PVP
	MobConverseChance    ConfigInt `yaml:"MobConverseChance"` // Chance 1-100 of attempting to converse when idle

	// Skill Progression
	UseSkillProgression ConfigBool `yaml:"UseSkillProgression"` // Enable skill/stat progression checks on skill/stat use
	DualProgressionMode ConfigBool `yaml:"DualProgressionMode"` // When true, progression checks grant actual skill/stat increases (requires UseSkillProgression)

	// RollSpread is the master randomness knob. See internal/dice/dice.go for full documentation.
	// Controls stdDev = stat * RollSpread for every stat-based roll in the game.
	// Default 0.15 (15%). Valid range 0.05–0.50. Requires server restart.
	RollSpread ConfigFloat `yaml:"RollSpread"`

	// Spell Cost Scaling
	SpellConvictionCostMultiplier ConfigFloat `yaml:"SpellConvictionCostMultiplier"` // Global multiplier for spell conviction costs (default 1.0)
	SpellHealthCostMultiplier     ConfigFloat `yaml:"SpellHealthCostMultiplier"`     // Global multiplier for spell health costs (default 1.0)

	// Defense stamina cost multipliers (Stage 7.1)
	DodgeMultiplier ConfigFloat `yaml:"DodgeMultiplier"` // Default 0.9 (base cost: 2 stamina)
	ParryMultiplier ConfigFloat `yaml:"ParryMultiplier"` // Default 0.9 (base cost: 4 stamina)
	BlockMultiplier ConfigFloat `yaml:"BlockMultiplier"` // Default 0.9 (base cost: 5 stamina)

	// Defense effectiveness multipliers — applied to the defense score before
	// the opposed roll, independent of prone/grapple modifiers.
	DodgeEffectiveness ConfigFloat `yaml:"DodgeEffectiveness"` // Default 1.0 (1.0 = no change)
	ParryEffectiveness ConfigFloat `yaml:"ParryEffectiveness"` // Default 1.0
	BlockEffectiveness ConfigFloat `yaml:"BlockEffectiveness"` // Default 1.0

	// Stage 7.5: Prone condition effects
	ProneAttackMultiplier   ConfigFloat `yaml:"ProneAttackMultiplier"`   // Default: 0.80 (multiplier on attack score while prone)
	ProneDodgePenalty       ConfigFloat `yaml:"ProneDodgePenalty"`       // Default: 0.70 (multiplier: 0.70 = keep 70% of dodge)
	ProneParryPenalty       ConfigFloat `yaml:"ProneParryPenalty"`       // Default: 0.80 (multiplier: 0.80 = keep 80% of parry)
	ProneBlockPenalty       ConfigFloat `yaml:"ProneBlockPenalty"`       // Default: 0.90 (multiplier: 0.90 = keep 90% of block)
	ProneDamagePenalty      ConfigFloat `yaml:"ProneDamagePenalty"`      // Default: 0.80 (damage multiplier)
	ProneVulnerabilityMultiplier ConfigFloat `yaml:"ProneVulnerabilityMultiplier"` // Default: 1.15 (multiplier on attack score vs prone target)
	StandStaminaCost        ConfigFloat `yaml:"StandStaminaCost"`        // Default: 0.15 (15% of max stamina)
	StandMinStamina         ConfigFloat `yaml:"StandMinStamina"`         // Default: 0.15 (minimum 15% stamina)

	// Defense floor: minimum probability any defense succeeds
	MinDefenseChance ConfigFloat `yaml:"MinDefenseChance"` // Default: 0.15 (15% floor even when massively outclassed)

	// Attack hit floor: minimum probability any attack hits
	MinAttackHitChance ConfigFloat `yaml:"MinAttackHitChance"` // Default: 0.15 (15% floor even when massively outclassed)

	// Stage 8.5: Third-party grapple vulnerability
	ThirdPartyGrapplePenalty ConfigFloat `yaml:"ThirdPartyGrapplePenalty"` // Default: 0.70 (-30% defense when entangled)

	// Stage 7.5: Special move parameters
	SpecialMoveCooldown ConfigInt   `yaml:"SpecialMoveCooldown"` // Default: 5 (shared cooldown for bash/trip/kick)
	BashDamagePercent   ConfigFloat `yaml:"BashDamagePercent"`   // Default: 0.50
	BashKnockdownChance ConfigInt   `yaml:"BashKnockdownChance"` // Default: 40
	TripDamagePercent   ConfigFloat `yaml:"TripDamagePercent"`   // Default: 0.25
	TripKnockdownChance ConfigInt   `yaml:"TripKnockdownChance"` // Default: 60
	KickDamagePercent   ConfigFloat `yaml:"KickDamagePercent"`   // Default: 0.40
	KickKnockdownChance ConfigInt   `yaml:"KickKnockdownChance"` // Default: 35

	// Coup de Grâce: rounds a mob waits before finishing a downed player (0 = disabled)
	CoupDeGraceRounds ConfigInt `yaml:"CoupDeGraceRounds"` // Default: 1
}

type GameplayDeath struct {
	EquipmentDropChance ConfigFloat  `yaml:"EquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	AlwaysDropBackpack  ConfigBool   `yaml:"AlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	ProtectionSkillRanks ConfigInt  `yaml:"ProtectionSkillRanks"` // Total skill ranks below which death penalties are waived
	PermaDeath           ConfigBool `yaml:"PermaDeath"`           // Is permadeath enabled?
	CorpsesEnabled      ConfigBool   `yaml:"CorpsesEnabled"`      // Whether corpses are left behind after mob/player deaths
	CorpseDecayTime     ConfigString `yaml:"CorpseDecayTime"`     // How long until corpses decay to dust (go away)
	// DOGMud death penalties (Stage 20.1)
	StatDecayMin          ConfigInt `yaml:"StatDecayMin"`          // Min Training loss on death (default 1)
	StatDecayMax          ConfigInt `yaml:"StatDecayMax"`          // Max Training loss on death (default 2)
	SkillRustCount        ConfigInt `yaml:"SkillRustCount"`        // Number of skills to decay on death (default 1)
	SkillRustAmount       ConfigInt `yaml:"SkillRustAmount"`       // Skill ranks lost per decayed skill (default 1)
	SkillRecencyThreshold ConfigInt `yaml:"SkillRecencyThreshold"` // Use count above which skills are protected (default 50)
	DeathsShadowBuffId    ConfigInt `yaml:"DeathsShadowBuffId"`    // Buff ID for Death's Shadow debuff (default 25)
}

func (g *GamePlay) Validate() {

	// Ignore AllowItemBuffRemoval
	// Ignore OnDeathAlwaysDropBackpack
	// Ignore ConsistentAttackMessages
	// Ignore CorpsesEnabled

	if g.Death.EquipmentDropChance < 0.0 || g.Death.EquipmentDropChance > 1.0 {
		g.Death.EquipmentDropChance = 0.0 // default
	}

	if g.Death.ProtectionSkillRanks < 0 {
		g.Death.ProtectionSkillRanks = 10 // default
	}

	if g.LivesStart < 0 {
		g.LivesStart = 0
	}

	if g.LivesMax < 0 {
		g.LivesMax = 0
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

	if g.MobConverseChance < 0 {
		g.MobConverseChance = 0
	} else if g.MobConverseChance > 100 {
		g.MobConverseChance = 100
	}

	// RollSpread: clamp to sensible range; default 0.15
	if g.RollSpread < 0.05 || g.RollSpread > 0.50 {
		g.RollSpread = 0.15
	}

	// Spell cost multipliers - default to 1.0 if not set or invalid
	if g.SpellConvictionCostMultiplier <= 0 {
		g.SpellConvictionCostMultiplier = 1.0
	}
	if g.SpellHealthCostMultiplier <= 0 {
		g.SpellHealthCostMultiplier = 1.0
	}

	// Defense cost multipliers - default to 0.9 if not set or invalid (Stage 7.1)
	if g.DodgeMultiplier <= 0 {
		g.DodgeMultiplier = 0.9
	}
	if g.ParryMultiplier <= 0 {
		g.ParryMultiplier = 0.9
	}
	if g.BlockMultiplier <= 0 {
		g.BlockMultiplier = 0.9
	}
	if g.DodgeEffectiveness <= 0 {
		g.DodgeEffectiveness = 1.0
	}
	if g.ParryEffectiveness <= 0 {
		g.ParryEffectiveness = 1.0
	}
	if g.BlockEffectiveness <= 0 {
		g.BlockEffectiveness = 1.0
	}

	// Stage 7.5: Prone condition effects - set defaults if invalid
	if g.ProneAttackMultiplier <= 0 {
		g.ProneAttackMultiplier = 0.80
	}
	if g.ProneDodgePenalty <= 0 || g.ProneDodgePenalty > 1.0 {
		g.ProneDodgePenalty = 0.70
	}
	if g.ProneParryPenalty <= 0 || g.ProneParryPenalty > 1.0 {
		g.ProneParryPenalty = 0.80
	}
	if g.ProneBlockPenalty <= 0 || g.ProneBlockPenalty > 1.0 {
		g.ProneBlockPenalty = 0.90
	}
	if g.ProneDamagePenalty <= 0 || g.ProneDamagePenalty > 1.0 {
		g.ProneDamagePenalty = 0.80
	}
	if g.ProneVulnerabilityMultiplier <= 0 {
		g.ProneVulnerabilityMultiplier = 1.15
	}
	if g.StandStaminaCost <= 0 || g.StandStaminaCost > 1.0 {
		g.StandStaminaCost = 0.15
	}
	if g.StandMinStamina <= 0 || g.StandMinStamina > 1.0 {
		g.StandMinStamina = 0.15
	}

	// Defense floor
	if g.MinDefenseChance < 0 || g.MinDefenseChance > 0.50 {
		g.MinDefenseChance = 0.15
	}

	// Attack hit floor
	if g.MinAttackHitChance < 0 || g.MinAttackHitChance > 0.50 {
		g.MinAttackHitChance = 0.15
	}

	// Stage 7.5: Special move parameters - set defaults if invalid
	if g.SpecialMoveCooldown < 1 {
		g.SpecialMoveCooldown = 5
	}
	if g.BashDamagePercent <= 0 || g.BashDamagePercent > 1.0 {
		g.BashDamagePercent = 0.50
	}
	if g.BashKnockdownChance < 0 || g.BashKnockdownChance > 100 {
		g.BashKnockdownChance = 40
	}
	if g.TripDamagePercent <= 0 || g.TripDamagePercent > 1.0 {
		g.TripDamagePercent = 0.25
	}
	if g.TripKnockdownChance < 0 || g.TripKnockdownChance > 100 {
		g.TripKnockdownChance = 60
	}
	if g.KickDamagePercent <= 0 || g.KickDamagePercent > 1.0 {
		g.KickDamagePercent = 0.40
	}
	if g.KickKnockdownChance < 0 || g.KickKnockdownChance > 100 {
		g.KickKnockdownChance = 35
	}

	// Coup de Grâce: default 1 round grace period
	if g.CoupDeGraceRounds < 0 {
		g.CoupDeGraceRounds = 1
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
