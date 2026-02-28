package configs

// Balance holds every numeric gameplay-balance constant that was previously
// hardcoded in Go source.  All fields have defaults equal to the prior
// hardcoded values, so behaviour is unchanged unless a field is overridden
// in config.yaml or config-overrides.yaml.
type Balance struct {
	// ── COMBAT ────────────────────────────────────────────────────────────────
	// Legacy unarmed knobs — still used by GetDefaultDistributionDamage() for
	// attack count and crit buff calculation. Damage values are overridden by
	// the unified pipeline (UnarmedDamageMultiplier + CalcRawDamage).
	UnarmedBaseDamage       ConfigFloat `yaml:"UnarmedBaseDamage"`       // Base damage before stat bonuses (default 2.0)
	UnarmedStrengthDivisor  ConfigFloat `yaml:"UnarmedStrengthDivisor"`  // Str / this = damage bonus (default 25.0)
	UnarmedSkillDivisor     ConfigFloat `yaml:"UnarmedSkillDivisor"`     // Skill / this = damage bonus (default 10.0)
	UnarmedBaseVariance        ConfigFloat `yaml:"UnarmedBaseVariance"`        // Base randomness of unarmed hits (default 3.0)
	UnarmedDamageMultiplier    ConfigFloat `yaml:"UnarmedDamageMultiplier"`    // Fist damage multiplier for new pipeline (default 0.30)
	SkillMultiplierBase        ConfigFloat `yaml:"SkillMultiplierBase"`        // Skill multiplier at rank 0 (default 1.0)
	SkillMultiplierMax         ConfigFloat `yaml:"SkillMultiplierMax"`         // Skill multiplier at soft cap (default 3.0)
	MeleeDamageScale           ConfigFloat `yaml:"MeleeDamageScale"`            // Physical damage scale. Stats ~100, so 0.30 yields ~30 raw per swing (default 0.30)
	SpellDamageScale           ConfigFloat `yaml:"SpellDamageScale"`            // Flat multiplier on spell damage output (default 1.0 = no change)
	RhetoricDamageScale        ConfigFloat `yaml:"RhetoricDamageScale"`         // Flat multiplier on conviction/taunt damage output (default 1.0 = no change)
	MobDamageMultiplier        ConfigFloat `yaml:"MobDamageMultiplier"`         // Extra multiplier applied to NPC melee damage only (default 1.0 = same as players)
	PhysicalMitigationCap      ConfigFloat `yaml:"PhysicalMitigationCap"`     // Max physical mitigation % (default 0.75)
	MagicalMitigationCap       ConfigFloat `yaml:"MagicalMitigationCap"`      // Max magical mitigation % (default 0.75)
	ConvictionMitigationCap    ConfigFloat `yaml:"ConvictionMitigationCap"`   // Max conviction mitigation % (default 0.75)
	ResourcePenaltyCurve       ConfigFloat `yaml:"ResourcePenaltyCurve"`     // Exponent for resource depletion penalty curve (default 2.0)
	HealthPenaltyMax           ConfigFloat `yaml:"HealthPenaltyMax"`         // Max melee damage penalty at 0% HP (default 0.28)
	StaminaPenaltyMax          ConfigFloat `yaml:"StaminaPenaltyMax"`        // Max attack count + hit rate penalty at 0% SP (default 0.28)
	ConvictionPenaltyMax       ConfigFloat `yaml:"ConvictionPenaltyMax"`     // Max taunt/spell penalty at 0% CP (default 0.28)

	// ── REGEN RATES ──────────────────────────────────────────────────────────
	PlayerHealthRegenPct     ConfigFloat `yaml:"PlayerHealthRegenPct"`     // Fraction of HealthMax regen'd per tick — players (default 0.01)
	PlayerStaminaRegenPct    ConfigFloat `yaml:"PlayerStaminaRegenPct"`    // Fraction of StaminaMax regen'd per tick — players (default 0.01)
	PlayerConvictionRegenPct ConfigFloat `yaml:"PlayerConvictionRegenPct"` // Fraction of ConvictionMax regen'd per tick — players (default 0.01)
	MobHealthRegenPct        ConfigFloat `yaml:"MobHealthRegenPct"`        // Fraction of HealthMax regen'd per tick — NPCs (default 0.01)
	MobStaminaRegenPct       ConfigFloat `yaml:"MobStaminaRegenPct"`       // Fraction of StaminaMax regen'd per tick — NPCs (default 0.01)
	MobConvictionRegenPct    ConfigFloat `yaml:"MobConvictionRegenPct"`    // Fraction of ConvictionMax regen'd per tick — NPCs (default 0.01)

	// ── STAMINA & CONVICTION ──────────────────────────────────────────────────
	MovementBaseStaminaCost  ConfigFloat `yaml:"MovementBaseStaminaCost"`  // Flat cost to move on normal terrain (default 2.0)
	MovementMaxStaminaCost   ConfigFloat `yaml:"MovementMaxStaminaCost"`   // Ceiling for any single move action (default 20.0)
	UnarmedAttackStaminaCost ConfigInt   `yaml:"UnarmedAttackStaminaCost"` // Stamina per unarmed attack (default 4)

	// ── RESOURCE MAXIMUMS ─────────────────────────────────────────────────────
	HealthBase          ConfigInt `yaml:"HealthBase"`          // Flat HP before stat contribution (default 5)
	HealthPerStrength   ConfigInt `yaml:"HealthPerStrength"`   // Strength multiplier toward HealthMax (default 1)
	HealthPerVitality   ConfigInt `yaml:"HealthPerVitality"`   // Vitality multiplier toward HealthMax (default 4)
	StaminaBase         ConfigInt `yaml:"StaminaBase"`         // Flat stamina before stat contribution (default 5)
	StaminaPerStrength  ConfigInt `yaml:"StaminaPerStrength"`  // Strength multiplier toward StaminaMax (default 1)
	StaminaPerWillpower ConfigInt `yaml:"StaminaPerWillpower"` // Willpower multiplier toward StaminaMax (default 1)
	StaminaPerVitality  ConfigInt `yaml:"StaminaPerVitality"`  // Vitality multiplier toward StaminaMax (default 3)
	ConvictionBase      ConfigInt `yaml:"ConvictionBase"`      // Flat conviction before stat contribution (default 5)
	ConvictionPerWilCha ConfigInt `yaml:"ConvictionPerWilCha"` // (Wil+Cha) multiplier toward ConvictionMax (default 2)

	// ── PROGRESSION ───────────────────────────────────────────────────────────
	SkillSoftCap              ConfigInt   `yaml:"SkillSoftCap"`              // Virtual ranks where progression slows sharply (default 50)
	StatSoftCap               ConfigInt   `yaml:"StatSoftCap"`               // Stat value where progression slows sharply (default 150)
	UsesPerRank               ConfigInt   `yaml:"UsesPerRank"`               // Skill/stat uses that equal one virtual rank (default 25)
	BaseProgressionChance     ConfigFloat `yaml:"BaseProgressionChance"`     // Starting chance to progress at rank 0 (default 0.30)
	ProgressionDecayBelowCap  ConfigFloat `yaml:"ProgressionDecayBelowCap"`  // Exponential steepness below soft cap (default 3.0)
	ProgressionDecayAboveCap  ConfigFloat `yaml:"ProgressionDecayAboveCap"`  // Exponential steepness above soft cap (default 2.0)
	StatSoftCapThreshold      ConfigInt   `yaml:"StatSoftCapThreshold"`      // Raw stat value where adjusted formula kicks in (default 105)
	StatSoftCapMultiplier     ConfigFloat `yaml:"StatSoftCapMultiplier"`     // Multiplier in: 100 + sqrt(raw-100) * this (default 2.0)
	MobProgressionEnabled    ConfigBool  `yaml:"MobProgressionEnabled"`    // Enable mob stat/skill progression (default true)
	MobProgressionRate       ConfigFloat `yaml:"MobProgressionRate"`       // Multiplier on progression chance vs players (default 0.5)
	MobStatCap               ConfigInt   `yaml:"MobStatCap"`               // Hard cap on mob stats from progression (default 200)
	MobSkillCap              ConfigInt   `yaml:"MobSkillCap"`              // Hard cap on mob skill level from progression (default 3)
	MobSaveIntervalRounds    ConfigInt   `yaml:"MobSaveIntervalRounds"`    // Rounds between periodic mob instance saves (default 100)
	MobInstanceMaxAgeDays    ConfigInt   `yaml:"MobInstanceMaxAgeDays"`    // Max age in days before stale instance files are pruned (default 7)

	// ── CHARACTER CREATION ────────────────────────────────────────────────────
	StatRollMean      ConfigFloat `yaml:"StatRollMean"`      // Mean for stat rolls at character creation (default 100.0)
	StatRollStdDev    ConfigFloat `yaml:"StatRollStdDev"`    // Std dev for stat rolls (default 15.0)
	StatRollMin       ConfigFloat `yaml:"StatRollMin"`       // Minimum stat value from rolls (default 70.0)
	StatRollMax       ConfigFloat `yaml:"StatRollMax"`       // Maximum stat value from rolls (default 130.0)
	StartingHealth    ConfigInt   `yaml:"StartingHealth"`    // Initial health points at character creation (default 10)

	// ── CRAFTING ──────────────────────────────────────────────────────────────
	CraftingBaseSuccessChance  ConfigInt `yaml:"CraftingBaseSuccessChance"`  // % before skill adjustment (default 50)
	CraftingSkillBonusPerLevel ConfigInt `yaml:"CraftingSkillBonusPerLevel"` // +% per skill level above recipe minimum (default 5)
	CraftingMinSuccessChance   ConfigInt `yaml:"CraftingMinSuccessChance"`   // Floor (default 5)
	CraftingMaxSuccessChance   ConfigInt `yaml:"CraftingMaxSuccessChance"`   // Ceiling (default 95)

	// ── RECIPE DISCOVERY ─────────────────────────────────────────────────────
	RecipeDiscoveryBaseChance ConfigFloat `yaml:"RecipeDiscoveryBaseChance"` // Base % to discover a new recipe per successful craft (default 8.0)
	RecipeDiscoveryDecayRate  ConfigFloat `yaml:"RecipeDiscoveryDecayRate"`  // Decay per known recipe: chance = base / (1 + known*this) (default 0.1)

	// ── MUTATIONS ─────────────────────────────────────────────────────────────
	MutationBaseProgress         ConfigFloat `yaml:"MutationBaseProgress"`         // Progress needed for first mutation (default 50.0)
	MutationProgressScale        ConfigFloat `yaml:"MutationProgressScale"`        // Each additional mutation costs Scale^n more (default 1.5)
	MutationMaxCount             ConfigInt   `yaml:"MutationMaxCount"`             // Max simultaneous mutations per character (default 5)
	MutationMaxLevel             ConfigInt   `yaml:"MutationMaxLevel"`             // Max level any single mutation can reach (default 3)
	MutationProgressGainPerRound ConfigFloat `yaml:"MutationProgressGainPerRound"` // Progress added per combat round (default 1.0)
	MutationLevel2Multiplier     ConfigFloat `yaml:"MutationLevel2Multiplier"`     // Effect scaling at level 2 (default 1.5)
	MutationLevel3Multiplier     ConfigFloat `yaml:"MutationLevel3Multiplier"`     // Effect scaling at level 3 (default 2.0)

	// ── SPELLCASTING ─────────────────────────────────────────────────────
	SpellDiscoveryBaseChance        ConfigFloat `yaml:"SpellDiscoveryBaseChance"`        // Base % to discover a new spell per successful cast (default 5.0)
	SpellDiscoveryDecayRate         ConfigFloat `yaml:"SpellDiscoveryDecayRate"`         // Decay per known spell: chance = base / (1 + known*this) (default 0.1)
	SpellInitiationBase             ConfigInt   `yaml:"SpellInitiationBase"`             // Base % chance to initiate a spell (default 60)
	SpellInitiationWillpowerDivisor ConfigInt   `yaml:"SpellInitiationWillpowerDivisor"` // Willpower / this = initiation bonus (default 4)
	SpellInitiationSkillFactor      ConfigInt   `yaml:"SpellInitiationSkillFactor"`      // Spellcasting level * this = initiation bonus (default 5)
	SpellConcentrationBase          ConfigInt   `yaml:"SpellConcentrationBase"`          // Base % concentration chance when struck (default 50)
	SpellFoldsSkillFactor           ConfigInt   `yaml:"SpellFoldsSkillFactor"`           // Skill * this in folds-per-round calc (default 25)
	SpellAttackSkillFactor          ConfigInt   `yaml:"SpellAttackSkillFactor"`          // Skill * this in spell attack mean (default 3)
	SpellProficiencyCastsPerPoint   ConfigInt   `yaml:"SpellProficiencyCastsPerPoint"`   // Casts needed per 1 proficiency point (default 50)

	// ── ENCHANTMENTS ─────────────────────────────────────────────────────────
	EnchantTierUpBaseChance     ConfigFloat `yaml:"EnchantTierUpBaseChance"`     // Chance per use (once threshold met) to advance tier (default 0.02)
	EnchantTierUsesBase         ConfigInt   `yaml:"EnchantTierUsesBase"`         // Uses needed for tier 0→1 (default 25)
	EnchantTierUsesScale        ConfigFloat `yaml:"EnchantTierUsesScale"`        // Multiplier per tier for uses threshold (default 2.5)
	EnchantRemovalPenaltyRounds ConfigInt   `yaml:"EnchantRemovalPenaltyRounds"` // Rounds of withdrawal after disenchant (default 50)
	EnchantMaxTier              ConfigInt   `yaml:"EnchantMaxTier"`              // Maximum tier enchantments can reach (default 4)

	// ── WORLD EVENTS ─────────────────────────────────────────────────────────
	WorldEventBufferSize ConfigInt `yaml:"WorldEventBufferSize"` // Max events in the ring buffer (default 200)

	// ── MOB MUTATIONS ────────────────────────────────────────────────────────
	MobMutationEnabled ConfigBool  `yaml:"MobMutationEnabled"` // Enable mob mutation acquisition in combat (default false)
	MobMutationRate    ConfigFloat `yaml:"MobMutationRate"`    // Multiplier on mutation progress vs players (default 0.3)

	// ── PACK SCALING ─────────────────────────────────────────────────────────
	PackScalingEnabled   ConfigBool `yaml:"PackScalingEnabled"`   // Enable pack survival bonuses (default true)
	PackSurvivalRounds   ConfigInt  `yaml:"PackSurvivalRounds"`   // Consecutive rounds together before bonus (default 10)
	PackBonusTrainingPts ConfigInt  `yaml:"PackBonusTrainingPts"` // Training points awarded per pack bonus (default 1)
	PackMaxBonus         ConfigInt  `yaml:"PackMaxBonus"`         // Max total pack bonus training points (default 5)

	// ── CRAFTER MOBS ─────────────────────────────────────────────────────────
	CrafterEnabled              ConfigBool `yaml:"CrafterEnabled"`              // Enable mob autonomous crafting (default true)
	CrafterMaterialRestockRate  ConfigInt  `yaml:"CrafterMaterialRestockRate"`  // Rounds between material restocks and craft attempts (default 200)
	CrafterRareThreshold        ConfigInt  `yaml:"CrafterRareThreshold"`        // SkillMinimum at or above which a craft is considered rare (default 3)

	// ── MOON PHASES ───────────────────────────────────────────────────────────
	MoonStatModMax ConfigFloat `yaml:"MoonStatModMax"` // Max fractional stat modifier from moon phases, e.g. 0.05 = ±5% (default 0.05)
}

func (b *Balance) Validate() {
	// ── COMBAT ────────────────────────────────────────────────────────────────
	if b.UnarmedBaseDamage <= 0 {
		b.UnarmedBaseDamage = 2.0
	}
	if b.UnarmedStrengthDivisor <= 0 {
		b.UnarmedStrengthDivisor = 25.0
	}
	if b.UnarmedSkillDivisor <= 0 {
		b.UnarmedSkillDivisor = 10.0
	}
	if b.UnarmedBaseVariance <= 0 {
		b.UnarmedBaseVariance = 3.0
	}
	if b.UnarmedDamageMultiplier <= 0 {
		b.UnarmedDamageMultiplier = 0.30
	}
	if b.SkillMultiplierBase <= 0 {
		b.SkillMultiplierBase = 1.0
	}
	if b.SkillMultiplierMax <= 0 {
		b.SkillMultiplierMax = 3.0
	}
	if b.MeleeDamageScale <= 0 {
		b.MeleeDamageScale = 0.30
	}
	if b.SpellDamageScale <= 0 {
		b.SpellDamageScale = 1.0
	}
	if b.RhetoricDamageScale <= 0 {
		b.RhetoricDamageScale = 1.0
	}
	if b.MobDamageMultiplier <= 0 {
		b.MobDamageMultiplier = 1.0
	}
	if b.PhysicalMitigationCap <= 0 || b.PhysicalMitigationCap > 1.0 {
		b.PhysicalMitigationCap = 0.75
	}
	if b.MagicalMitigationCap <= 0 || b.MagicalMitigationCap > 1.0 {
		b.MagicalMitigationCap = 0.75
	}
	if b.ConvictionMitigationCap <= 0 || b.ConvictionMitigationCap > 1.0 {
		b.ConvictionMitigationCap = 0.75
	}
	if b.ResourcePenaltyCurve <= 0 {
		b.ResourcePenaltyCurve = 2.0
	}
	if b.HealthPenaltyMax <= 0 || b.HealthPenaltyMax > 1.0 {
		b.HealthPenaltyMax = 0.28
	}
	if b.StaminaPenaltyMax <= 0 || b.StaminaPenaltyMax > 1.0 {
		b.StaminaPenaltyMax = 0.28
	}
	if b.ConvictionPenaltyMax <= 0 || b.ConvictionPenaltyMax > 1.0 {
		b.ConvictionPenaltyMax = 0.28
	}

	// ── REGEN RATES ──────────────────────────────────────────────────────────
	clampPct := func(v *ConfigFloat, def ConfigFloat) {
		if *v <= 0 {
			*v = def
		}
		if *v > 1.0 {
			*v = 1.0
		}
	}
	clampPct(&b.PlayerHealthRegenPct, 0.01)
	clampPct(&b.PlayerStaminaRegenPct, 0.01)
	clampPct(&b.PlayerConvictionRegenPct, 0.01)
	clampPct(&b.MobHealthRegenPct, 0.01)
	clampPct(&b.MobStaminaRegenPct, 0.01)
	clampPct(&b.MobConvictionRegenPct, 0.01)

	// ── STAMINA & CONVICTION ──────────────────────────────────────────────────
	if b.MovementBaseStaminaCost <= 0 {
		b.MovementBaseStaminaCost = 2.0
	}
	if b.MovementMaxStaminaCost <= 0 {
		b.MovementMaxStaminaCost = 20.0
	}
	if b.UnarmedAttackStaminaCost < 1 {
		b.UnarmedAttackStaminaCost = 4
	}

	// ── RESOURCE MAXIMUMS ─────────────────────────────────────────────────────
	if b.HealthBase < 0 {
		b.HealthBase = 5
	}
	if b.HealthPerStrength < 0 {
		b.HealthPerStrength = 1
	}
	if b.HealthPerVitality < 0 {
		b.HealthPerVitality = 4
	}
	if b.StaminaBase < 0 {
		b.StaminaBase = 5
	}
	if b.StaminaPerStrength < 0 {
		b.StaminaPerStrength = 1
	}
	if b.StaminaPerWillpower < 0 {
		b.StaminaPerWillpower = 1
	}
	if b.StaminaPerVitality < 0 {
		b.StaminaPerVitality = 3
	}
	if b.ConvictionBase < 0 {
		b.ConvictionBase = 5
	}
	if b.ConvictionPerWilCha < 0 {
		b.ConvictionPerWilCha = 2
	}

	// ── PROGRESSION ───────────────────────────────────────────────────────────
	if b.SkillSoftCap < 1 {
		b.SkillSoftCap = 50
	}
	if b.StatSoftCap < 1 {
		b.StatSoftCap = 150
	}
	if b.UsesPerRank < 1 {
		b.UsesPerRank = 25
	}
	if b.BaseProgressionChance <= 0 || b.BaseProgressionChance > 1.0 {
		b.BaseProgressionChance = 0.30
	}
	if b.ProgressionDecayBelowCap <= 0 {
		b.ProgressionDecayBelowCap = 3.0
	}
	if b.ProgressionDecayAboveCap <= 0 {
		b.ProgressionDecayAboveCap = 2.0
	}
	if b.StatSoftCapThreshold < 100 {
		b.StatSoftCapThreshold = 105
	}
	if b.StatSoftCapMultiplier <= 0 {
		b.StatSoftCapMultiplier = 2.0
	}
	if b.MobProgressionRate <= 0 || b.MobProgressionRate > 1.0 {
		b.MobProgressionRate = 0.5
	}
	if b.MobStatCap < 1 {
		b.MobStatCap = 200
	}
	if b.MobSkillCap < 1 {
		b.MobSkillCap = 3
	}
	if b.MobSaveIntervalRounds < 1 {
		b.MobSaveIntervalRounds = 100
	}
	if b.MobInstanceMaxAgeDays < 1 {
		b.MobInstanceMaxAgeDays = 7
	}

	// ── CHARACTER CREATION ────────────────────────────────────────────────────
	if b.StatRollMean <= 0 {
		b.StatRollMean = 100.0
	}
	if b.StatRollStdDev <= 0 {
		b.StatRollStdDev = 15.0
	}
	if b.StatRollMin <= 0 {
		b.StatRollMin = 70.0
	}
	if b.StatRollMax <= 0 {
		b.StatRollMax = 130.0
	}
	if b.StartingHealth < 1 {
		b.StartingHealth = 10
	}

	// ── CRAFTING ──────────────────────────────────────────────────────────────
	if b.CraftingBaseSuccessChance <= 0 || b.CraftingBaseSuccessChance > 100 {
		b.CraftingBaseSuccessChance = 50
	}
	if b.CraftingSkillBonusPerLevel <= 0 {
		b.CraftingSkillBonusPerLevel = 5
	}
	if b.CraftingMinSuccessChance < 1 {
		b.CraftingMinSuccessChance = 5
	}
	if b.CraftingMaxSuccessChance <= 0 || b.CraftingMaxSuccessChance > 100 {
		b.CraftingMaxSuccessChance = 95
	}

	// ── RECIPE DISCOVERY ─────────────────────────────────────────────────────
	if b.RecipeDiscoveryBaseChance <= 0 {
		b.RecipeDiscoveryBaseChance = 8.0
	}
	if b.RecipeDiscoveryDecayRate <= 0 {
		b.RecipeDiscoveryDecayRate = 0.1
	}

	// ── MUTATIONS ─────────────────────────────────────────────────────────────
	if b.MutationBaseProgress <= 0 {
		b.MutationBaseProgress = 50.0
	}
	if b.MutationProgressScale <= 0 {
		b.MutationProgressScale = 1.5
	}
	if b.MutationMaxCount < 1 {
		b.MutationMaxCount = 5
	}
	if b.MutationMaxLevel < 1 {
		b.MutationMaxLevel = 3
	}
	if b.MutationProgressGainPerRound <= 0 {
		b.MutationProgressGainPerRound = 1.0
	}
	if b.MutationLevel2Multiplier <= 0 {
		b.MutationLevel2Multiplier = 1.5
	}
	if b.MutationLevel3Multiplier <= 0 {
		b.MutationLevel3Multiplier = 2.0
	}

	// ── SPELLCASTING ─────────────────────────────────────────────────────
	if b.SpellDiscoveryBaseChance <= 0 {
		b.SpellDiscoveryBaseChance = 5.0
	}
	if b.SpellDiscoveryDecayRate <= 0 {
		b.SpellDiscoveryDecayRate = 0.1
	}
	if b.SpellInitiationBase <= 0 {
		b.SpellInitiationBase = 60
	}
	if b.SpellInitiationWillpowerDivisor < 1 {
		b.SpellInitiationWillpowerDivisor = 4
	}
	if b.SpellInitiationSkillFactor < 1 {
		b.SpellInitiationSkillFactor = 5
	}
	if b.SpellConcentrationBase <= 0 {
		b.SpellConcentrationBase = 50
	}
	if b.SpellFoldsSkillFactor < 1 {
		b.SpellFoldsSkillFactor = 25
	}
	if b.SpellAttackSkillFactor < 1 {
		b.SpellAttackSkillFactor = 3
	}
	if b.SpellProficiencyCastsPerPoint < 1 {
		b.SpellProficiencyCastsPerPoint = 50
	}

	// ── ENCHANTMENTS ─────────────────────────────────────────────────────────
	if b.EnchantTierUpBaseChance <= 0 {
		b.EnchantTierUpBaseChance = 0.02
	}
	if b.EnchantTierUsesBase < 1 {
		b.EnchantTierUsesBase = 25
	}
	if b.EnchantTierUsesScale <= 0 {
		b.EnchantTierUsesScale = 2.5
	}
	if b.EnchantRemovalPenaltyRounds < 1 {
		b.EnchantRemovalPenaltyRounds = 50
	}
	if b.EnchantMaxTier < 1 {
		b.EnchantMaxTier = 4
	}

	// ── WORLD EVENTS ─────────────────────────────────────────────────────────
	if b.WorldEventBufferSize < 10 {
		b.WorldEventBufferSize = 200
	}

	// ── MOB MUTATIONS ────────────────────────────────────────────────────────
	if b.MobMutationRate <= 0 || b.MobMutationRate > 1.0 {
		b.MobMutationRate = 0.3
	}

	// ── PACK SCALING ─────────────────────────────────────────────────────────
	if b.PackSurvivalRounds < 1 {
		b.PackSurvivalRounds = 10
	}
	if b.PackBonusTrainingPts < 1 {
		b.PackBonusTrainingPts = 1
	}
	if b.PackMaxBonus < 1 {
		b.PackMaxBonus = 5
	}

	// ── CRAFTER MOBS ─────────────────────────────────────────────────────────
	if b.CrafterMaterialRestockRate < 1 {
		b.CrafterMaterialRestockRate = 200
	}
	if b.CrafterRareThreshold < 1 {
		b.CrafterRareThreshold = 3
	}

	// ── MOON PHASES ───────────────────────────────────────────────────────────
	if b.MoonStatModMax <= 0 {
		b.MoonStatModMax = 0.05
	}
}

func GetBalanceConfig() Balance {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Balance
}
