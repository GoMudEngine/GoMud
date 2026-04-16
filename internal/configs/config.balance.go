package configs

// Balance holds every numeric gameplay-balance constant that was previously
// hardcoded in Go source.  All fields have defaults equal to the prior
// hardcoded values, so behaviour is unchanged unless a field is overridden
// in config.yaml or config-overrides.yaml.
type Balance struct {
	// ── ROLL SPREAD ──────────────────────────────────────────────────────────
	// Master randomness knob. Controls stdDev = stat * RollSpread for every
	// stat-based roll. Default 0.15 (15%). Valid range 0.05–0.50.
	RollSpread ConfigFloat `yaml:"RollSpread"`

	// ── COMBAT: DEFENSE COSTS ────────────────────────────────────────────────
	DodgeMultiplier ConfigFloat `yaml:"DodgeMultiplier"` // Stamina cost multiplier for dodge (default 0.9)
	ParryMultiplier ConfigFloat `yaml:"ParryMultiplier"` // Stamina cost multiplier for parry (default 0.9)
	BlockMultiplier ConfigFloat `yaml:"BlockMultiplier"` // Stamina cost multiplier for block (default 0.9)

	// ── COMBAT: DEFENSE EFFECTIVENESS ────────────────────────────────────────
	DodgeEffectiveness ConfigFloat `yaml:"DodgeEffectiveness"` // Multiplier on dodge score before opposed roll (default 1.0)
	ParryEffectiveness ConfigFloat `yaml:"ParryEffectiveness"` // Multiplier on parry score before opposed roll (default 1.0)
	BlockEffectiveness ConfigFloat `yaml:"BlockEffectiveness"` // Multiplier on block score before opposed roll (default 1.0)
	MinDefenseChance   ConfigFloat `yaml:"MinDefenseChance"`   // Floor probability any defense succeeds (default 0.15)
	MinAttackHitChance ConfigFloat `yaml:"MinAttackHitChance"` // Floor probability any attack hits (default 0.15)

	// ── COMBAT: PRONE & GRAPPLE ──────────────────────────────────────────────
	ProneAttackMultiplier        ConfigFloat `yaml:"ProneAttackMultiplier"`        // Multiplier on attack score while prone (default 0.80)
	ProneDodgePenalty            ConfigFloat `yaml:"ProneDodgePenalty"`            // Multiplier on dodge score while prone (default 0.70)
	ProneParryPenalty            ConfigFloat `yaml:"ProneParryPenalty"`            // Multiplier on parry score while prone (default 0.80)
	ProneBlockPenalty            ConfigFloat `yaml:"ProneBlockPenalty"`            // Multiplier on block score while prone (default 0.90)
	ProneDamagePenalty           ConfigFloat `yaml:"ProneDamagePenalty"`           // Damage multiplier while prone (default 0.80)
	ProneVulnerabilityMultiplier ConfigFloat `yaml:"ProneVulnerabilityMultiplier"` // Multiplier on attack score vs prone target (default 1.15)
	StandStaminaCost             ConfigFloat `yaml:"StandStaminaCost"`             // Fraction of max stamina to stand up (default 0.15)
	StandMinStamina              ConfigFloat `yaml:"StandMinStamina"`              // Minimum fraction of max SP to stand (default 0.15)
	ThirdPartyGrapplePenalty     ConfigFloat `yaml:"ThirdPartyGrapplePenalty"`     // Defense multiplier when grappled vs third party (default 0.70)
	ClinchDodgePenalty          ConfigFloat `yaml:"ClinchDodgePenalty"`          // Dodge score multiplier while clinched (default 0.80)
	ClinchParryPenalty          ConfigFloat `yaml:"ClinchParryPenalty"`          // Parry score multiplier while clinched (default 0.83)
	ClinchBlockPenalty          ConfigFloat `yaml:"ClinchBlockPenalty"`          // Block score multiplier while clinched (default 0.85)
	GroundedDodgePenalty        ConfigFloat `yaml:"GroundedDodgePenalty"`        // Dodge score multiplier while grounded (default 0.75)
	GroundedParryPenalty        ConfigFloat `yaml:"GroundedParryPenalty"`        // Parry score multiplier while grounded (default 0.77)
	GroundedBlockPenalty        ConfigFloat `yaml:"GroundedBlockPenalty"`        // Block score multiplier while grounded (default 0.80)

	// ── COMBAT: SPECIAL MOVES ────────────────────────────────────────────────
	SpecialMoveCooldown ConfigInt   `yaml:"SpecialMoveCooldown"` // Shared cooldown rounds for bash/trip/kick (default 5)
	BashDamagePercent   ConfigFloat `yaml:"BashDamagePercent"`   // Fraction of normal melee damage (default 0.50)
	BashKnockdownChance ConfigInt   `yaml:"BashKnockdownChance"` // Base % knockdown chance (default 40)
	TripDamagePercent   ConfigFloat `yaml:"TripDamagePercent"`   // Fraction of normal melee damage (default 0.25)
	TripKnockdownChance ConfigInt   `yaml:"TripKnockdownChance"` // Base % knockdown chance (default 60)
	KickDamagePercent   ConfigFloat `yaml:"KickDamagePercent"`   // Fraction of normal melee damage (default 0.80)
	KickKnockdownChance ConfigInt   `yaml:"KickKnockdownChance"` // Base % knockdown chance (default 35)
	StompDamagePercent  ConfigFloat `yaml:"StompDamagePercent"`  // Stomp damage when target is prone (default 1.20)
	KneeDamagePercent   ConfigFloat `yaml:"KneeDamagePercent"`   // Knee damage in grapple (default 1.00)
	CoupDeGraceRounds   ConfigInt   `yaml:"CoupDeGraceRounds"`   // Rounds before mob finishes downed player (default 1; 0=disabled)

	// ── SKULLDUGGERY ─────────────────────────────────────────────────────────
	SneakFailCooldown              ConfigInt   `yaml:"SneakFailCooldown"`              // Rounds before sneak retry after failure (default 3)
	SurpriseAttackOffhandPenalty   ConfigFloat `yaml:"SurpriseAttackOffhandPenalty"`   // Hit penalty for offhand surprise attack (default 0.10)
	SurpriseAttackExtraArm1Penalty ConfigFloat `yaml:"SurpriseAttackExtraArm1Penalty"` // Hit penalty for extra arm 1 (default 0.25)
	SurpriseAttackExtraArm2Penalty ConfigFloat `yaml:"SurpriseAttackExtraArm2Penalty"` // Hit penalty for extra arm 2 (default 0.40)
	SurpriseAttackExtraArm3Penalty ConfigFloat `yaml:"SurpriseAttackExtraArm3Penalty"` // Hit penalty for extra arm 3 (default 0.55)
	SurpriseAttackExtraArm4Penalty ConfigFloat `yaml:"SurpriseAttackExtraArm4Penalty"` // Hit penalty for extra arm 4 (default 0.70)
	StealSkillMultiplier           ConfigFloat `yaml:"StealSkillMultiplier"`           // Tuning knob for steal/plant rolls (default 1.0)
	StealHiddenBonus               ConfigInt   `yaml:"StealHiddenBonus"`               // Bonus to attacker score when hidden (default 25)
	StealCooldown                  ConfigInt   `yaml:"StealCooldown"`                  // Steal/plant cooldown in real seconds (default 60)
	ShadowCooldown                 ConfigInt   `yaml:"ShadowCooldown"`                 // Rounds before re-shadowing (default 5)

	// ── COMBAT: SPELL COSTS ──────────────────────────────────────────────────
	SpellConvictionCostMultiplier ConfigFloat `yaml:"SpellConvictionCostMultiplier"` // Global multiplier for spell conviction costs (default 1.0)
	SpellHealthCostMultiplier     ConfigFloat `yaml:"SpellHealthCostMultiplier"`     // Global multiplier for spell health costs (default 1.0)

	// ── COMBAT: DARKNESS ─────────────────────────────────────────────────────
	DarknessCombatPenalty ConfigFloat `yaml:"DarknessCombatPenalty"` // Multiplier on attack AND defense scores when fighting blind (default 0.80)

	// ── COMBAT: MESSAGES ─────────────────────────────────────────────────────
	ConsistentAttackMessages ConfigBool `yaml:"ConsistentAttackMessages"` // Whether each weapon has consistent attack messages

	// ── COMBAT: DAMAGE ───────────────────────────────────────────────────────
	// Legacy unarmed knobs — still used by GetDefaultDistributionDamage() for
	// attack count and crit buff calculation. Damage values are overridden by
	// the unified pipeline (UnarmedDamageMultiplier + CalcRawDamage).
	UnarmedBaseDamage       ConfigFloat `yaml:"UnarmedBaseDamage"`       // Base damage before stat bonuses (default 2.0)
	UnarmedStrengthDivisor  ConfigFloat `yaml:"UnarmedStrengthDivisor"`  // Str / this = damage bonus (default 25.0)
	UnarmedSkillDivisor     ConfigFloat `yaml:"UnarmedSkillDivisor"`     // Skill / this = damage bonus (default 10.0)
	UnarmedBaseVariance        ConfigFloat `yaml:"UnarmedBaseVariance"`        // Base randomness of unarmed hits (default 3.0)
	UnarmedDamageMultiplier    ConfigFloat `yaml:"UnarmedDamageMultiplier"`    // Fist damage multiplier for new pipeline (default 0.30)
	UnarmedSpeedMultiplier     ConfigFloat `yaml:"UnarmedSpeedMultiplier"`     // Unarmed attack speed — slightly faster than light weapons (default 1.4)
	HasteSwingMultiplier       ConfigFloat `yaml:"HasteSwingMultiplier"`       // Swing count multiplier when haste buff is active (default 1.50)
	SkillMultiplierBase        ConfigFloat `yaml:"SkillMultiplierBase"`        // Skill multiplier at rank 0 (default 1.0)
	SkillMultiplierMax         ConfigFloat `yaml:"SkillMultiplierMax"`         // Skill multiplier at soft cap (default 3.0)
	SkillWeight                ConfigFloat `yaml:"SkillWeight"`                // Global multiplier on skill contributions in additive formulas (default 2.0)
	MeleeDamageScale           ConfigFloat `yaml:"MeleeDamageScale"`            // Physical damage scale. Stats ~100, so 0.30 yields ~30 raw per swing (default 0.30)
	SpellDamageScale           ConfigFloat `yaml:"SpellDamageScale"`            // Flat multiplier on spell damage output (default 1.0 = no change)
	RhetoricDamageScale        ConfigFloat `yaml:"RhetoricDamageScale"`         // Flat multiplier on conviction/taunt damage output (default 1.0 = no change)
	MobDamageMultiplier        ConfigFloat `yaml:"MobDamageMultiplier"`         // Extra multiplier applied to NPC melee damage only (default 1.0 = same as players)
	GlobalDamageMultiplier     ConfigFloat `yaml:"GlobalDamageMultiplier"`      // Master multiplier applied to ALL damage channels (default 1.0)
	PhysicalMitigationCap      ConfigFloat `yaml:"PhysicalMitigationCap"`     // Max physical mitigation % (default 0.75)
	MagicalMitigationCap       ConfigFloat `yaml:"MagicalMitigationCap"`      // Max magical mitigation % (default 0.75)
	ConvictionMitigationCap    ConfigFloat `yaml:"ConvictionMitigationCap"`   // Max conviction mitigation % (default 0.75)
	SpellAvoidanceDamageMultiplier    ConfigFloat `yaml:"SpellAvoidanceDamageMultiplier"`    // Damage multiplier on successful spell deflection (default 0.50)
	RhetoricAvoidanceDamageMultiplier ConfigFloat `yaml:"RhetoricAvoidanceDamageMultiplier"` // Damage multiplier on successful stoic resolve (default 0.50)
	ResourcePenaltyCurve       ConfigFloat `yaml:"ResourcePenaltyCurve"`     // Exponent for resource depletion penalty curve (default 2.0)
	HealthPenaltyMax           ConfigFloat `yaml:"HealthPenaltyMax"`         // Max melee damage penalty at 0% HP (default 0.28)
	StaminaPenaltyMax          ConfigFloat `yaml:"StaminaPenaltyMax"`        // Max attack count + hit rate penalty at 0% SP (default 0.28)
	ConvictionPenaltyMax       ConfigFloat `yaml:"ConvictionPenaltyMax"`     // Max taunt/spell penalty at 0% CP (default 0.28)

	// ── REGEN RATES ──────────────────────────────────────────────────────────
	PlayerHealthRegenPct     ConfigFloat `yaml:"PlayerHealthRegenPct"`     // Fraction of HealthMax regen'd per tick — players (default 0.01)
	PlayerStaminaRegenPct    ConfigFloat `yaml:"PlayerStaminaRegenPct"`    // Fraction of StaminaMax regen'd per tick — players (default 0.01)
	PlayerConvictionRegenPct ConfigFloat `yaml:"PlayerConvictionRegenPct"` // Fraction of ConvictionMax regen'd per tick — players (default 0.01)
	MobHealthRegenPct        ConfigFloat `yaml:"MobHealthRegenPct"`        // Fraction of HealthMax regen'd per tick — NPCs (default 0.01)
	MobStaminaRegenPct       ConfigFloat `yaml:"MobStaminaRegenPct"`       // Fraction of StaminaMax regen'd per tick — NPCs (default 0.02)
	MobConvictionRegenPct    ConfigFloat `yaml:"MobConvictionRegenPct"`    // Fraction of ConvictionMax regen'd per tick — NPCs (default 0.02)

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
	RegenProgressionBase     ConfigFloat `yaml:"RegenProgressionBase"`     // Max chance at 0% resource per stat per tick (default 0.005)
	RegenProgressionCurve    ConfigFloat `yaml:"RegenProgressionCurve"`    // Exponent shaping the depletion→chance curve (default 3.0)

	// ── PROGRESSION MULTIPLIERS ──────────────────────────────────────────────
	// Per-stat and per-skill multipliers on progression chance.
	// Use plain float64 maps (not ConfigFloat) for native YAML unmarshaling.
	StatProgressionMultipliers  map[string]float64 `yaml:"StatProgressionMultipliers"`  // Per-stat multiplier on progression chance (default 1.0; dex 0.5)
	SkillProgressionMultipliers map[string]float64 `yaml:"SkillProgressionMultipliers"` // Per-skill multiplier on progression chance — overrides hardcoded defaults

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

	// ── SALVAGE ──────────────────────────────────────────────────────────────
	SalvageMinChance    ConfigFloat `yaml:"SalvageMinChance"`    // Per-ingredient recovery chance at skill 1 (default 0.15)
	SalvageMaxChance    ConfigFloat `yaml:"SalvageMaxChance"`    // Hard cap on per-ingredient chance (default 0.85)
	SalvageSoftCap      ConfigInt   `yaml:"SalvageSoftCap"`      // Skill level for max curve (default 50)
	SalvageGoldPerRound ConfigInt   `yaml:"SalvageGoldPerRound"` // Ingredient gold value per salvage round (default 10)
	SalvageMaxRounds    ConfigInt   `yaml:"SalvageMaxRounds"`    // Maximum salvage rounds (default 5)

	// ── QUEST ENGINE ─────────────────────────────────────────────────────────
	QuestLogLevel          string    `yaml:"QuestLogLevel"`          // verbose, medium, minimal (default verbose)
	QuestChainDepthLimit   ConfigInt `yaml:"QuestChainDepthLimit"`   // max chained grant evaluations per event (default 10)
	QuestPerformanceWarnMs ConfigInt `yaml:"QuestPerformanceWarnMs"` // warn if trigger evaluation exceeds this (default 50)

	// ── MUTATIONS ─────────────────────────────────────────────────────────────
	MutationBaseProgress         ConfigFloat `yaml:"MutationBaseProgress"`         // Progress needed for first mutation (default 50.0)
	MutationProgressScale        ConfigFloat `yaml:"MutationProgressScale"`        // Each additional mutation costs Scale^n more (default 1.5)
	MutationMaxCount             ConfigInt   `yaml:"MutationMaxCount"`             // Max simultaneous mutations per character (default 5)
	MutationMaxLevel             ConfigInt   `yaml:"MutationMaxLevel"`             // Max level any single mutation can reach (default 3)
	MutationDeepenChance         ConfigFloat `yaml:"MutationDeepenChance"`         // Probability of deepening vs new discovery when both possible (default 0.70)
	MutationProgressGainPerRound ConfigFloat `yaml:"MutationProgressGainPerRound"` // Progress added per combat round (default 1.0)
	MutationLevel2Multiplier     ConfigFloat `yaml:"MutationLevel2Multiplier"`     // Effect scaling at level 2 (default 1.5)
	MutationLevel3Multiplier     ConfigFloat `yaml:"MutationLevel3Multiplier"`     // Effect scaling at level 3 (default 2.0)
	MutationLevel4Multiplier     ConfigFloat `yaml:"MutationLevel4Multiplier"`     // Effect scaling at level 4 (default 2.5)

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
	SpellDifficultyProgressionScale ConfigFloat `yaml:"SpellDifficultyProgressionScale"` // Per-point spell difficulty bonus to skill progression (default 0.01)
	CraftDifficultyProgressionScale ConfigFloat `yaml:"CraftDifficultyProgressionScale"` // Per-point recipe skill_minimum bonus to skill progression (default 0.02)
	SelfCastProgressionMultiplier   ConfigFloat `yaml:"SelfCastProgressionMultiplier"`   // Progression multiplier when spell only targets self (default 0.5)

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

	// ── MOB AI ───────────────────────────────────────────────────────────────
	CombatMemoryDuration ConfigInt   `yaml:"CombatMemoryDuration"` // Rounds before combat memory expires (default 300)
	MobAIEnabled         ConfigBool  `yaml:"MobAIEnabled"`         // Global toggle for reactive AI system (default true)
	MobReactionDelayMin  ConfigFloat `yaml:"MobReactionDelayMin"`  // Min reaction delay in seconds (default 0.25)
	MobReactionDelayMax             ConfigFloat `yaml:"MobReactionDelayMax"`             // Max reaction delay in seconds (default 4.0)
	MobBTreeReactionBase            ConfigFloat `yaml:"MobBTreeReactionBase"`            // Base reaction delay in seconds for behavior tree mobs (default 3.0)
	MobBTreeReactionPerceptionScale ConfigInt   `yaml:"MobBTreeReactionPerceptionScale"` // Perception divisor for reaction delay (default 100)

	// ── PACK SCALING ─────────────────────────────────────────────────────────
	PackScalingEnabled   ConfigBool `yaml:"PackScalingEnabled"`   // Enable pack survival bonuses (default true)
	PackSurvivalRounds   ConfigInt  `yaml:"PackSurvivalRounds"`   // Consecutive rounds together before bonus (default 10)
	PackBonusTrainingPts ConfigInt  `yaml:"PackBonusTrainingPts"` // Training points awarded per pack bonus (default 1)
	PackMaxBonus         ConfigInt  `yaml:"PackMaxBonus"`         // Max total pack bonus training points (default 5)

	// ── PACK ROAMING ────────────────────────────────────────────────────────
	PackRoamingEnabled ConfigBool `yaml:"PackRoamingEnabled"` // Enable alpha-follow pack movement (default true)
	PackMaxSize        ConfigInt  `yaml:"PackMaxSize"`        // Max followers per alpha (-1 = unlimited, default -1)
	PackScatterRounds  ConfigInt  `yaml:"PackScatterRounds"`  // Rounds mobs skip wandering after alpha death (default 2)

	// ── CRAFTER MOBS ─────────────────────────────────────────────────────────
	CrafterEnabled              ConfigBool `yaml:"CrafterEnabled"`              // Enable mob autonomous crafting (default true)
	CrafterMaterialRestockRate  ConfigInt  `yaml:"CrafterMaterialRestockRate"`  // Rounds between material restocks and craft attempts (default 200)
	CrafterRareThreshold        ConfigInt  `yaml:"CrafterRareThreshold"`        // SkillMinimum at or above which a craft is considered rare (default 3)

	// ── GOSSIP SYSTEM ────────────────────────────────────────────────────────
	GossipIntervalRounds ConfigInt `yaml:"GossipIntervalRounds"` // Rounds between gossip broadcasts for "gossiper" group mobs (default 75)

	// ── MOON PHASES ───────────────────────────────────────────────────────────
	MoonStatModMax          ConfigFloat `yaml:"MoonStatModMax"`          // Max fractional stat modifier from moon phases, e.g. 0.05 = ±5% (default 0.05)
	CarryCapacityMultiplier ConfigFloat `yaml:"CarryCapacityMultiplier"` // Strength multiplier for carry capacity in lbs (default 0.65)

	// ── TOXICITY ────────────────────────────────────────────────────────────
	ToxicityDecayPerTick  ConfigFloat `yaml:"ToxicityDecayPerTick"`  // Points decayed per regen tick (default 1.0)
	ToxicityBaseMax       ConfigFloat `yaml:"ToxicityBaseMax"`       // Base max before vitality bonus (default 100)
	ToxicityVitalityScale ConfigFloat `yaml:"ToxicityVitalityScale"` // Vitality divisor for max bonus (default 5)

	// ── MANIFESTATION / COMPANION SCALING ───────────────────────────────────
	ManifestStatScaleChaFactor   ConfigInt   `yaml:"ManifestStatScaleChaFactor"`   // Charisma divisor for companion stat scaling (default 150)
	ManifestStatScaleSkillFactor ConfigFloat `yaml:"ManifestStatScaleSkillFactor"` // Manifestation skill additive factor (default 0.02)

	// ── SHOP ECONOMY ─────────────────────────────────────────────────────────
	ShopBuyRatio           ConfigFloat `yaml:"ShopBuyRatio,omitempty"`           // Base buy/sell spread: NPC buy offer = baseValue * BuyRatio * scarcityMult (default 0.50)
	ShopPriceFloor         ConfigFloat `yaml:"ShopPriceFloor,omitempty"`         // Minimum scarcity multiplier when stock is very high (default 0.25)
	ShopPriceCeiling       ConfigFloat `yaml:"ShopPriceCeiling,omitempty"`       // Maximum scarcity multiplier when stock is zero (default 5.0)
	ShopAbundanceThreshold ConfigFloat `yaml:"ShopAbundanceThreshold,omitempty"` // Stock/restock ratio at which price hits the floor (default 3.0)
	ShopMaterialReserve    ConfigInt   `yaml:"ShopMaterialReserve,omitempty"`    // Units of each material a crafter mob reserves before selling (default 1)
	ShopGoldReserveRatio   ConfigFloat `yaml:"ShopGoldReserveRatio,omitempty"`   // Fraction of gold pool a shop keeps in reserve before buying (default 0.50)
	BarterMaxDiscount      ConfigFloat `yaml:"BarterMaxDiscount,omitempty"`      // Max fractional price reduction a player can get via bartering (default 0.15)
	BarterMaxBonus         ConfigFloat `yaml:"BarterMaxBonus,omitempty"`         // Max fractional sell-price bonus a player can get via bartering (default 0.15)
	StorageFeePerItem      ConfigInt   `yaml:"StorageFeePerItem"`                // Gold charged per stored item per game month (default 1)

	// ── LOOT ──────────────────────────────────────────────────────────────────
	LootBudgetScalar ConfigFloat `yaml:"LootBudgetScalar"` // Multiplier for sqrt(goldPaid) loot budget (default 7.0)

	// ── INSTANCES ────────────────────────────────────────────────────────────
	InstanceStatPoolCap ConfigInt `yaml:"InstanceStatPoolCap"` // Max stat pool per mob in instances (default 50000, 0=uncapped)
}

func (b *Balance) Validate() {
	// ── ROLL SPREAD ──────────────────────────────────────────────────────────
	if b.RollSpread < 0.05 || b.RollSpread > 0.50 {
		b.RollSpread = 0.15
	}

	// ── COMBAT: DEFENSE COSTS ────────────────────────────────────────────────
	if b.DodgeMultiplier <= 0 {
		b.DodgeMultiplier = 0.9
	}
	if b.ParryMultiplier <= 0 {
		b.ParryMultiplier = 0.9
	}
	if b.BlockMultiplier <= 0 {
		b.BlockMultiplier = 0.9
	}

	// ── COMBAT: DEFENSE EFFECTIVENESS ────────────────────────────────────────
	if b.DodgeEffectiveness <= 0 {
		b.DodgeEffectiveness = 1.0
	}
	if b.ParryEffectiveness <= 0 {
		b.ParryEffectiveness = 1.0
	}
	if b.BlockEffectiveness <= 0 {
		b.BlockEffectiveness = 1.0
	}

	// ── COMBAT: PRONE & GRAPPLE ──────────────────────────────────────────────
	if b.ProneAttackMultiplier <= 0 {
		b.ProneAttackMultiplier = 0.80
	}
	if b.ProneDodgePenalty <= 0 || b.ProneDodgePenalty > 1.0 {
		b.ProneDodgePenalty = 0.70
	}
	if b.ProneParryPenalty <= 0 || b.ProneParryPenalty > 1.0 {
		b.ProneParryPenalty = 0.80
	}
	if b.ProneBlockPenalty <= 0 || b.ProneBlockPenalty > 1.0 {
		b.ProneBlockPenalty = 0.90
	}
	if b.ProneDamagePenalty <= 0 || b.ProneDamagePenalty > 1.0 {
		b.ProneDamagePenalty = 0.80
	}
	if b.ProneVulnerabilityMultiplier <= 0 {
		b.ProneVulnerabilityMultiplier = 1.15
	}

	// ── COMBAT: SPECIAL MOVES ────────────────────────────────────────────────
	if b.SpecialMoveCooldown < 1 {
		b.SpecialMoveCooldown = 5
	}

	// ── SKULLDUGGERY ─────────────────────────────────────────────────────────
	if b.SneakFailCooldown < 0 {
		b.SneakFailCooldown = 3
	}
	if b.StealSkillMultiplier <= 0 {
		b.StealSkillMultiplier = 1.0
	}
	if b.StealHiddenBonus < 0 {
		b.StealHiddenBonus = 25
	}
	if b.StealCooldown < 0 {
		b.StealCooldown = 60
	}
	if b.ShadowCooldown < 0 {
		b.ShadowCooldown = 5
	}

	// ── COMBAT: DARKNESS ─────────────────────────────────────────────────────
	if b.DarknessCombatPenalty <= 0 || b.DarknessCombatPenalty > 1.0 {
		b.DarknessCombatPenalty = 0.80
	}

	// ── COMBAT: DAMAGE ───────────────────────────────────────────────────────
	if b.MeleeDamageScale <= 0 {
		b.MeleeDamageScale = 0.30
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
	if b.RhetoricAvoidanceDamageMultiplier <= 0 || b.RhetoricAvoidanceDamageMultiplier > 1.0 {
		b.RhetoricAvoidanceDamageMultiplier = 0.50
	}

	// ── REGEN RATES (mob) ────────────────────────────────────────────────────
	clampPct := func(v *ConfigFloat, def ConfigFloat) {
		if *v <= 0 {
			*v = def
		}
		if *v > 1.0 {
			*v = 1.0
		}
	}
	clampPct(&b.MobHealthRegenPct, 0.01)
	clampPct(&b.MobStaminaRegenPct, 0.02)
	clampPct(&b.MobConvictionRegenPct, 0.02)

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
	if b.RegenProgressionBase <= 0 {
		b.RegenProgressionBase = 0.005
	}
	if b.RegenProgressionBase > 1.0 {
		b.RegenProgressionBase = 1.0
	}
	if b.RegenProgressionCurve <= 0 {
		b.RegenProgressionCurve = 3.0
	}

	// ── PROGRESSION MULTIPLIERS ──────────────────────────────────────────────
	if b.StatProgressionMultipliers == nil {
		b.StatProgressionMultipliers = map[string]float64{}
	}
	for k, v := range b.StatProgressionMultipliers {
		if v <= 0 {
			delete(b.StatProgressionMultipliers, k)
		}
	}
	if b.SkillProgressionMultipliers == nil {
		b.SkillProgressionMultipliers = map[string]float64{}
	}
	for k, v := range b.SkillProgressionMultipliers {
		if v <= 0 {
			delete(b.SkillProgressionMultipliers, k)
		}
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
	if b.MutationDeepenChance <= 0 || b.MutationDeepenChance > 1.0 {
		b.MutationDeepenChance = 0.70
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
	if b.MutationLevel4Multiplier <= 0 {
		b.MutationLevel4Multiplier = 2.5
	}

	// ── MOB MUTATIONS ────────────────────────────────────────────────────────
	if b.MobMutationRate <= 0 || b.MobMutationRate > 1.0 {
		b.MobMutationRate = 0.3
	}

	// ── MOB AI ───────────────────────────────────────────────────────────────
	if b.CombatMemoryDuration < 1 {
		b.CombatMemoryDuration = 300
	}
	if !bool(b.MobAIEnabled) {
		b.MobAIEnabled = true
	}
	if b.MobReactionDelayMin <= 0 {
		b.MobReactionDelayMin = 0.25
	}
	if b.MobReactionDelayMax <= 0 {
		b.MobReactionDelayMax = 2.0
	}
	if b.MobBTreeReactionBase <= 0 {
		b.MobBTreeReactionBase = 2.0
	}
	if b.MobBTreeReactionPerceptionScale < 1 {
		b.MobBTreeReactionPerceptionScale = 100
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

	// ── PACK ROAMING ────────────────────────────────────────────────────────
	if b.PackMaxSize == 0 {
		b.PackMaxSize = -1
	}
	if b.PackScatterRounds < 0 {
		b.PackScatterRounds = 2
	}

	// ── GOSSIP SYSTEM ────────────────────────────────────────────────────────
	if b.GossipIntervalRounds < 20 {
		b.GossipIntervalRounds = 75
	}

	// ── MOON PHASES ───────────────────────────────────────────────────────────
	if b.MoonStatModMax <= 0 {
		b.MoonStatModMax = 0.05
	}

	// ── TOXICITY ────────────────────────────────────────────────────────────
	if b.ToxicityDecayPerTick <= 0 {
		b.ToxicityDecayPerTick = 1.0
	}
	if b.ToxicityBaseMax <= 0 {
		b.ToxicityBaseMax = 100
	}
	if b.ToxicityVitalityScale <= 0 {
		b.ToxicityVitalityScale = 5
	}

	// ── MANIFESTATION / COMPANION SCALING ───────────────────────────────────
	if b.ManifestStatScaleChaFactor < 1 {
		b.ManifestStatScaleChaFactor = 150
	}
	if b.ManifestStatScaleSkillFactor <= 0 {
		b.ManifestStatScaleSkillFactor = 0.02
	}

	b.validateSpells()
	b.validateShops()
	b.validateMisc()
}

func GetBalanceConfig() Balance {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Balance
}

// GetStatProgressionMultiplier returns the per-stat progression multiplier
// from config, or 1.0 if the stat has no override.
func (b *Balance) GetStatProgressionMultiplier(statName string) float64 {
	if b.StatProgressionMultipliers != nil {
		if mult, ok := b.StatProgressionMultipliers[statName]; ok {
			return mult
		}
	}
	return 1.0
}

// GetSkillProgressionMultiplier returns the per-skill progression multiplier
// from config, or 0 to signal "use hardcoded default".
func (b *Balance) GetSkillProgressionMultiplier(skillName string) (float64, bool) {
	if b.SkillProgressionMultipliers != nil {
		if mult, ok := b.SkillProgressionMultipliers[skillName]; ok {
			return mult, true
		}
	}
	return 0, false
}
