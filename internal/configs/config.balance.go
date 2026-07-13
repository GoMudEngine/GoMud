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
	ClinchDodgePenalty           ConfigFloat `yaml:"ClinchDodgePenalty"`           // Dodge score multiplier while clinched (default 0.80)
	ClinchParryPenalty           ConfigFloat `yaml:"ClinchParryPenalty"`           // Parry score multiplier while clinched (default 0.83)
	ClinchBlockPenalty           ConfigFloat `yaml:"ClinchBlockPenalty"`           // Block score multiplier while clinched (default 0.85)
	GroundedDodgePenalty         ConfigFloat `yaml:"GroundedDodgePenalty"`         // Dodge score multiplier while grounded (default 0.75)
	GroundedParryPenalty         ConfigFloat `yaml:"GroundedParryPenalty"`         // Parry score multiplier while grounded (default 0.77)
	GroundedBlockPenalty         ConfigFloat `yaml:"GroundedBlockPenalty"`         // Block score multiplier while grounded (default 0.80)
	// GrappleStaminaLowThreshold is the stamina fraction (0.0–1.0) below
	// which a character is considered "low stamina" for grapple purposes.
	// Used by IsLowGrappleStamina() and the mob_low_grapple_stamina btree
	// primitive (T5) and Position_Messaging (T7).
	GrappleStaminaLowThreshold ConfigFloat `yaml:"GrappleStaminaLowThreshold"` // Stamina fraction floor for grapple stamina warning (default 0.25)

	// ── REACH UTILITY CURVE (chunk 4c) ───────────────────────────────────────
	// See internal/combat/reach.go for the formula and the design spec for
	// reasoning.

	// ReachStandingGrappleRadius is the effective radius (meters) at
	// which a weapon stops fitting in Clinch / BackStanding positions.
	// Default 0.5 — about chest-to-chest distance in a clinch.
	ReachStandingGrappleRadius ConfigFloat `yaml:"reach_standing_grapple_radius"`

	// ReachGroundGrappleRadius is the effective radius (meters) at
	// which a weapon stops fitting in any ground grapple (Mount,
	// SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround,
	// HalfGuard, Guard). Default 0.3 — body-on-body distance.
	ReachGroundGrappleRadius ConfigFloat `yaml:"reach_ground_grapple_radius"`

	// ReachUtilityFloor caps the minimum damage multiplier from the
	// reach curve. Without a floor, a pike in mount would multiply
	// damage by ~0.1 — a floor at 0.15 ensures even the longest
	// weapon can poke for chip damage (pommel jab, hilt-strike).
	// Tunable; smoke may push this lower.
	ReachUtilityFloor ConfigFloat `yaml:"reach_utility_floor"`

	// ── CHUNK 4D: SUBMISSION TICK ────────────────────────────────────────────
	// Per-round opportunistic submission attempts gated on the chunk-4b
	// control-axis drift roll. See spec
	// docs/superpowers/specs/2026-05-18-state-chunk-4d-submission-rework-design.md
	SubmissionAttemptAlpha ConfigFloat `yaml:"submission_attempt_alpha"`  // Min drift-margin (std devs) that opens a sub window (either side)
	SubmissionAttemptCritZ ConfigFloat `yaml:"submission_attempt_crit_z"` // Defender-side shortcut: drift z >= this opens a bottom-sub window regardless of margin
	SubSkillWeight         ConfigFloat `yaml:"sub_skill_weight"`          // Unarmed-combat skill contribution multiplier in the sub roll
	SubBadZThreshold       ConfigFloat `yaml:"sub_bad_z_threshold"`       // Z-score below which the sub roll's bad-tier (attempter falls prone) fires
	SubCritZThreshold      ConfigFloat `yaml:"sub_crit_z_threshold"`      // Z-score at or above which the sub roll's crit-tier (recipient stunned) fires
	SubGoldLossFraction    ConfigFloat `yaml:"sub_gold_loss_fraction"`    // Fraction of carried gold transferred to the aggressor on subdue/cripple
	BrokenLimbBuffDuration ConfigInt   `yaml:"broken_limb_buff_duration"` // Duration in rounds for the broken-limb buff; expires naturally via standard buff tick

	// ── CHUNK 4E: THIRD-PARTY INTERFERENCE ──────────────────────────────────
	// See docs/superpowers/specs/ chunk-4e design.

	// ControlDegradeOnOutsideHit enables chunk 4e §5: third-party damage on
	// a grapple controller shifts their ControlLevel one step toward Neutral
	// per disrupted round. Set false to disable the mechanic for tuning.
	ControlDegradeOnOutsideHit ConfigBool `yaml:"ControlDegradeOnOutsideHit"`

	// SubInterruptDamageThresholdPct is the fraction of HealthMax that
	// constitutes "above-threshold" third-party damage for chunk 4e §7
	// sub interrupt. Below this, damage doesn't break a sub setup; at or
	// above, the sub outcome is forced to Bad tier. A crit also triggers
	// the override regardless of threshold. Default 0.10 (10% of max HP).
	// Set 0 to disable threshold-path (crit-only).
	SubInterruptDamageThresholdPct ConfigFloat `yaml:"SubInterruptDamageThresholdPct"`

	// ── GRAPPLE CONTROL AXIS (chunk 4b) ──────────────────────────────────────
	// Per-round drift mechanics — see
	// docs/superpowers/specs/2026-05-16-state-chunk-4b-position-control-axis-design.md
	GrappleStaminaPenaltyMax        ConfigFloat `yaml:"GrappleStaminaPenaltyMax"`        // Max roll-mult reduction at 0% stamina (default 0.60)
	GrappleStaminaPenaltyCurve      ConfigFloat `yaml:"GrappleStaminaPenaltyCurve"`      // Exponent shape of stamina penalty curve (default 1.5)
	GrappleEncumbrancePenaltyMax    ConfigFloat `yaml:"GrappleEncumbrancePenaltyMax"`    // Max roll-mult reduction at max encumbrance (default 0.80)
	GrappleEncumbrancePenaltyCurve  ConfigFloat `yaml:"GrappleEncumbrancePenaltyCurve"`  // Exponent shape of encumbrance penalty curve (default 1.5)
	GrappleStaminaCostPerRound      ConfigInt   `yaml:"GrappleStaminaCostPerRound"`      // Base per-round stamina cost in grapples (default 5)
	GrappleControllerCostMultiplier ConfigFloat `yaml:"GrappleControllerCostMultiplier"` // Controller's per-round cost multiplier (default 1.0)
	GrappleControlledCostMultiplier ConfigFloat `yaml:"GrappleControlledCostMultiplier"` // Controlled's per-round cost multiplier (default 2.0)
	PositionConsistencyCheckRounds  ConfigInt   `yaml:"PositionConsistencyCheckRounds"`  // How often the periodic invariant checker runs (default 10)

	// ── COMBAT: SPECIAL MOVES ────────────────────────────────────────────────
	SpecialMoveCooldown     ConfigInt   `yaml:"SpecialMoveCooldown"`     // Shared cooldown rounds for bash/trip/kick (default 5)
	FlightOpposedEdge       ConfigInt   `yaml:"FlightOpposedEdge"`       // Winged Flight: melee opposed-roll edge a flyer gets over the earthbound (default 25)
	FlightMoveStaminaMult   ConfigFloat `yaml:"FlightMoveStaminaMult"`   // Winged Flight: move-stamina cost multiplier while flying (default 0.5)
	FlightFleeStaminaMult   ConfigFloat `yaml:"FlightFleeStaminaMult"`   // Winged Flight: flee-stamina cost multiplier while flying (default 0.5)
	TauntHoldRounds         ConfigInt   `yaml:"TauntHoldRounds"`         // Rounds a successful taunt pins the target's aggro onto the taunter (default 4)
	BashDamagePercent       ConfigFloat `yaml:"BashDamagePercent"`       // Fraction of normal melee damage (default 0.50)
	BashKnockdownChance     ConfigInt   `yaml:"BashKnockdownChance"`     // Base % knockdown chance (default 40)
	TripDamagePercent       ConfigFloat `yaml:"TripDamagePercent"`       // Fraction of normal melee damage (default 0.25)
	TripKnockdownChance     ConfigInt   `yaml:"TripKnockdownChance"`     // Base % knockdown chance (default 60)
	KickDamagePercent       ConfigFloat `yaml:"KickDamagePercent"`       // Fraction of normal melee damage (default 0.80)
	KickKnockdownChance     ConfigInt   `yaml:"KickKnockdownChance"`     // Base % knockdown chance (default 35)
	StompDamagePercent      ConfigFloat `yaml:"StompDamagePercent"`      // Stomp damage when target is prone (default 1.20)
	KneeDamagePercent       ConfigFloat `yaml:"KneeDamagePercent"`       // Knee damage in grapple (default 1.00)
	CoupDeGraceRounds       ConfigInt   `yaml:"CoupDeGraceRounds"`       // Rounds before mob finishes downed player (default 1; 0=disabled)
	DrainHealRatio          ConfigFloat `yaml:"DrainHealRatio"`          // Fraction of drain damage the attacker heals (lifesteal), default 0.75
	ThrottleInterruptChance ConfigFloat `yaml:"ThrottleInterruptChance"` // Chance throttle interrupts a casting victim, default 0.75

	// ── COMBAT: RANGED ───────────────────────────────────────────────────────
	RangedShotScale          ConfigFloat `yaml:"RangedShotScale"`          // Global multiplier on all ranged shot damage (default 1.0)
	RangedShieldDefenseBonus ConfigInt   `yaml:"RangedShieldDefenseBonus"` // Flat defense-score bonus vs ranged when defender has a shield (default 15)

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
	HiddenMoveStaminaMultiplier    ConfigFloat `yaml:"HiddenMoveStaminaMultiplier"`    // Extra stamina cost multiplier for moving while hidden (default 3.0)
	SneakModEmitsLightDarkRoom     ConfigFloat `yaml:"SneakModEmitsLightDarkRoom"`     // Sneak score multiplier: sneaker emits light, room dark (default 0.5)
	SneakModEmitsLightLitRoom      ConfigFloat `yaml:"SneakModEmitsLightLitRoom"`      // Sneak score multiplier: sneaker emits light, room lit (default 0.85)
	SneakModNoLightLitRoom         ConfigFloat `yaml:"SneakModNoLightLitRoom"`         // Sneak score multiplier: sneaker dark, room lit (default 0.9)

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
	UnarmedBaseDamage                 ConfigFloat `yaml:"UnarmedBaseDamage"`                 // Base damage before stat bonuses (default 2.0)
	UnarmedStrengthDivisor            ConfigFloat `yaml:"UnarmedStrengthDivisor"`            // Str / this = damage bonus (default 25.0)
	UnarmedSkillDivisor               ConfigFloat `yaml:"UnarmedSkillDivisor"`               // Skill / this = damage bonus (default 10.0)
	UnarmedBaseVariance               ConfigFloat `yaml:"UnarmedBaseVariance"`               // Base randomness of unarmed hits (default 3.0)
	UnarmedDamageMultiplier           ConfigFloat `yaml:"UnarmedDamageMultiplier"`           // Fist damage multiplier for new pipeline (default 0.30)
	UnarmedSpeedMultiplier            ConfigFloat `yaml:"UnarmedSpeedMultiplier"`            // Unarmed attack speed — slightly faster than light weapons (default 1.4)
	HasteSwingMultiplier              ConfigFloat `yaml:"HasteSwingMultiplier"`              // Swing count multiplier when haste buff is active (default 1.50)
	SkillMultiplierBase               ConfigFloat `yaml:"SkillMultiplierBase"`               // Skill multiplier at rank 0 (default 1.0)
	SkillMultiplierMax                ConfigFloat `yaml:"SkillMultiplierMax"`                // Skill multiplier at soft cap (default 3.0)
	SkillWeight                       ConfigFloat `yaml:"SkillWeight"`                       // Global multiplier on skill contributions in additive formulas (default 2.0)
	MeleeDamageScale                  ConfigFloat `yaml:"MeleeDamageScale"`                  // Physical damage scale. Stats ~100, so 0.30 yields ~30 raw per swing (default 0.30)
	SpellDamageScale                  ConfigFloat `yaml:"SpellDamageScale"`                  // Flat multiplier on spell damage output (default 1.0 = no change)
	RhetoricDamageScale               ConfigFloat `yaml:"RhetoricDamageScale"`               // Flat multiplier on conviction/taunt damage output (default 1.0 = no change)
	MobDamageMultiplier               ConfigFloat `yaml:"MobDamageMultiplier"`               // Extra multiplier applied to NPC melee damage only (default 1.0 = same as players)
	GlobalDamageMultiplier            ConfigFloat `yaml:"GlobalDamageMultiplier"`            // Master multiplier applied to ALL damage channels (default 1.0)
	PhysicalMitigationCap             ConfigFloat `yaml:"PhysicalMitigationCap"`             // Max physical mitigation % (default 0.75)
	MagicalMitigationCap              ConfigFloat `yaml:"MagicalMitigationCap"`              // Max magical mitigation % (default 0.75)
	ConvictionMitigationCap           ConfigFloat `yaml:"ConvictionMitigationCap"`           // Max conviction mitigation % (default 0.75)
	SpellAvoidanceDamageMultiplier    ConfigFloat `yaml:"SpellAvoidanceDamageMultiplier"`    // Damage multiplier on successful spell deflection (default 0.50)
	RhetoricAvoidanceDamageMultiplier ConfigFloat `yaml:"RhetoricAvoidanceDamageMultiplier"` // Damage multiplier on successful stoic resolve (default 0.50)
	ResourcePenaltyCurve              ConfigFloat `yaml:"ResourcePenaltyCurve"`              // Exponent for resource depletion penalty curve (default 2.0)
	HealthPenaltyMax                  ConfigFloat `yaml:"HealthPenaltyMax"`                  // Max melee damage penalty at 0% HP (default 0.28)
	StaminaPenaltyMax                 ConfigFloat `yaml:"StaminaPenaltyMax"`                 // Max attack count + hit rate penalty at 0% SP (default 0.28)
	ConvictionPenaltyMax              ConfigFloat `yaml:"ConvictionPenaltyMax"`              // Max taunt/spell penalty at 0% CP (default 0.28)

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
	SkillSoftCap             ConfigInt   `yaml:"SkillSoftCap"`             // Virtual ranks where progression slows sharply (default 50)
	StatSoftCap              ConfigInt   `yaml:"StatSoftCap"`              // Stat value where progression slows sharply (default 150)
	UsesPerRank              ConfigInt   `yaml:"UsesPerRank"`              // Skill/stat uses that equal one virtual rank (default 25)
	BaseProgressionChance    ConfigFloat `yaml:"BaseProgressionChance"`    // Starting chance to progress at rank 0 (default 0.30)
	StatProgressionRate      ConfigFloat `yaml:"StatProgressionRate"`      // Global multiplier on STAT progression chance; skills unaffected (default 1.0)
	ProgressionDecayBelowCap ConfigFloat `yaml:"ProgressionDecayBelowCap"` // Exponential steepness below soft cap (default 3.0)
	ProgressionDecayAboveCap ConfigFloat `yaml:"ProgressionDecayAboveCap"` // Exponential steepness above soft cap (default 2.0)
	StatSoftCapThreshold     ConfigInt   `yaml:"StatSoftCapThreshold"`     // Raw stat value where adjusted formula kicks in (default 105)
	StatSoftCapMultiplier    ConfigFloat `yaml:"StatSoftCapMultiplier"`    // Multiplier in: 100 + sqrt(raw-100) * this (default 2.0)
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
	StatRollMean   ConfigFloat `yaml:"StatRollMean"`   // Mean for stat rolls at character creation (default 100.0)
	StatRollStdDev ConfigFloat `yaml:"StatRollStdDev"` // Std dev for stat rolls (default 15.0)
	StatRollMin    ConfigFloat `yaml:"StatRollMin"`    // Minimum stat value from rolls (default 70.0)
	StatRollMax    ConfigFloat `yaml:"StatRollMax"`    // Maximum stat value from rolls (default 130.0)
	StartingHealth ConfigInt   `yaml:"StartingHealth"` // Initial health points at character creation (default 10)

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
	// ── MUTATION GRAPH (drift + opposition) ───────────────────────────────────
	MutationAffinityPerSkillUse    ConfigFloat `yaml:"MutationAffinityPerSkillUse"`    // drift affinity added per cluster-relevant skill use (default 1.0)
	MutationAffinityPerCombatEvent ConfigFloat `yaml:"MutationAffinityPerCombatEvent"` // drift affinity per combat behavior event — tank/dodge/control (default 1.0)
	MutationAffinityPerRarity      ConfigFloat `yaml:"MutationAffinityPerRarity"`      // affinity required per point of rarity to unlock a mutation (default 6.0 — r3 entry ~18, r9 apex/Extra Arms ~54; earned via drift)
	MutationAffinityDecay          ConfigFloat `yaml:"MutationAffinityDecay"`          // per-tick multiplicative decay of cluster affinity (default 0.98)
	MutationBodyConvictionDecayMax ConfigFloat `yaml:"MutationBodyConvictionDecayMax"` // max fraction of ConvictionMax lost to deep Body commitment (default 0.9)
	MutationBeliefGearDecayMax     ConfigFloat `yaml:"MutationBeliefGearDecayMax"`     // max fraction of gear effectiveness lost to deep Belief commitment (default 0.9)
	MutationPoleDecayRef           ConfigFloat `yaml:"MutationPoleDecayRef"`           // pole-depth at which decay reaches half its max (default 60.0)
	MutationLevel2Multiplier     ConfigFloat `yaml:"MutationLevel2Multiplier"`     // Effect scaling at level 2 (default 1.5)
	MutationLevel3Multiplier     ConfigFloat `yaml:"MutationLevel3Multiplier"`     // Effect scaling at level 3 (default 2.0)
	MutationLevel4Multiplier     ConfigFloat `yaml:"MutationLevel4Multiplier"`     // Effect scaling at level 4 (default 2.5)

	// ── SPELLCASTING ─────────────────────────────────────────────────────
	SpellDiscoveryBaseChance ConfigFloat `yaml:"SpellDiscoveryBaseChance"` // Base % to discover a new spell per successful cast (default 5.0)
	SpellDiscoveryDecayRate  ConfigFloat `yaml:"SpellDiscoveryDecayRate"`  // Decay per known spell: chance = base / (1 + known*this) (default 0.1)

	// ── DISCOVERY OFFSET (shared: spells + recipes) ──────────────────────────
	DiscoveryPerceptionScale ConfigFloat `yaml:"DiscoveryPerceptionScale"` // Raw Per contribution reaches 1.0 at (Per - 100) / this (default 200)
	DiscoverySkillScale      ConfigFloat `yaml:"DiscoverySkillScale"`      // Raw skill contribution reaches 1.0 at rank / this (default 100)
	DiscoveryMaxDecayOffset  ConfigFloat `yaml:"DiscoveryMaxDecayOffset"`  // Hard ceiling on combined offset; effective decay floor = Decay × (1 - this) (default 0.8)

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

	// ── ENCHANT SALVAGE BANDS ─────────────────────────────────────────────────
	// Tiered potion → enchanting-mat mapping. Potions are bucketed by their
	// alchemy-recipe skill_minimum into four bands, each with per-mat drop %s.
	EnchantSalvageBand2Min         ConfigInt `yaml:"EnchantSalvageBand2Min,omitempty"`         // potion skill_min >= this → band 2 (default 10)
	EnchantSalvageBand3Min         ConfigInt `yaml:"EnchantSalvageBand3Min,omitempty"`         // band 3 (default 18)
	EnchantSalvageBand4Min         ConfigInt `yaml:"EnchantSalvageBand4Min,omitempty"`         // band 4 (default 28)
	EnchantSalvageBand2SettingPct  ConfigInt `yaml:"EnchantSalvageBand2SettingPct,omitempty"`  // % chance to yield chrysalis-setting in band 2 (default 25)
	EnchantSalvageBand3SettingPct  ConfigInt `yaml:"EnchantSalvageBand3SettingPct,omitempty"`  // % chance to yield chrysalis-setting in band 3 (default 35)
	EnchantSalvageBand3CatalystPct ConfigInt `yaml:"EnchantSalvageBand3CatalystPct,omitempty"` // % chance to yield mutation-catalyst in band 3 (default 12)
	EnchantSalvageBand4CatalystPct ConfigInt `yaml:"EnchantSalvageBand4CatalystPct,omitempty"` // % chance to yield mutation-catalyst in band 4 (default 40)
	EnchantSalvageBand4SettingPct  ConfigInt `yaml:"EnchantSalvageBand4SettingPct,omitempty"`  // % chance to yield chrysalis-setting in band 4 (default 30)
	EnchantSalvageBand4CorePct     ConfigInt `yaml:"EnchantSalvageBand4CorePct,omitempty"`     // % chance to yield chrysalis-core in band 4 (default 8)

	// ── WORLD EVENTS ─────────────────────────────────────────────────────────
	WorldEventBufferSize ConfigInt `yaml:"WorldEventBufferSize"` // Max events in the ring buffer (default 200)

	// ── CHARACTER MANAGEMENT ──────────────────────────────────────────────────
	CharacterRenameCooldownHours ConfigInt `yaml:"CharacterRenameCooldownHours"` // Hours between player renames (0 disables; default 168 = 7 days)

	// ── MOB MUTATIONS ────────────────────────────────────────────────────────
	MobMutationEnabled ConfigBool  `yaml:"MobMutationEnabled"` // Enable mob mutation acquisition in combat (default false)
	MobMutationRate    ConfigFloat `yaml:"MobMutationRate"`    // Multiplier on mutation progress vs players (default 0.3)

	// ── MOB AI ───────────────────────────────────────────────────────────────
	CombatMemoryDuration            ConfigInt   `yaml:"CombatMemoryDuration"`            // Rounds before combat memory expires (default 300)
	MobAIEnabled                    ConfigBool  `yaml:"MobAIEnabled"`                    // Global toggle for reactive AI system (default true)
	MobReactionDelayMin             ConfigFloat `yaml:"MobReactionDelayMin"`             // Min reaction delay in seconds (default 0.25)
	MobReactionDelayMax             ConfigFloat `yaml:"MobReactionDelayMax"`             // Max reaction delay in seconds (default 4.0)
	MobBTreeReactionBase            ConfigFloat `yaml:"MobBTreeReactionBase"`            // Base reaction delay in seconds for behavior tree mobs (default 3.0)
	MobBTreeReactionPerceptionScale ConfigInt   `yaml:"MobBTreeReactionPerceptionScale"` // Perception divisor for reaction delay (default 100)

	// ── MOB SCHEDULES (chunk 3.2) ────────────────────────────────────────────
	// ScheduleMaxPathRetries is the number of consecutive failed pathto
	// attempts a scheduled mob will tolerate before falling back to
	// `pathto home`. Default 20 (≈80 seconds at the default tick rate).
	// See chunk 3.2 spec.
	ScheduleMaxPathRetries ConfigInt `yaml:"ScheduleMaxPathRetries"`

	// ── MOB SLEEP & WAKE (chunk 3.3) ──────────────────────────────────────────
	// SleepRegenMultiplier multiplies HP/SP/CP per-round percentage regen
	// when the bearer has the Sleeping flag. Default 5.0 — sleep is the
	// dominant recovery mechanic. Chunk 3.3.
	SleepRegenMultiplier ConfigFloat `yaml:"SleepRegenMultiplier"`

	// ScheduleWakeGraceRounds is the cooldown (in rounds) during which a
	// scheduled mob will not re-sleep after a forced wake. Prevents the
	// schedule executor from immediately re-applying Sleeping when the
	// player interacts with a sleeping NPC. Default 50 (~200 sec real-time
	// at default tick rate). Chunk 3.3.
	ScheduleWakeGraceRounds ConfigInt `yaml:"ScheduleWakeGraceRounds"`

	// ── NPC-NPC IDLE CONVERSATIONS (chunk 3.6) ───────────────────────────────────
	// ConversationBaseChancePct is the per-tick percentage chance that a
	// fully-idle NPC will attempt to start an idle conversation with an
	// in-room partner that has a relationship edge. Default 1.0 → ~once
	// per 100 ticks per NPC. Chunk 3.6.
	ConversationBaseChancePct ConfigFloat `yaml:"ConversationBaseChancePct"`

	// ConversationPlayerArrivalBoostPct is the percentage chance that
	// a conversation will start when a player arrives in a room with 2+
	// relateable, idle NPCs. Default 25. Chunk 3.6.
	ConversationPlayerArrivalBoostPct ConfigInt `yaml:"ConversationPlayerArrivalBoostPct"`

	// ConversationCooldownRounds is the cooldown applied to both NPCs
	// after a conversation completes, before either can initiate another.
	// Default 50 (~200 sec real-time). Chunk 3.6.
	ConversationCooldownRounds ConfigInt `yaml:"ConversationCooldownRounds"`

	// ── GOAL SELECTION (chunk 4.2) ────────────────────────────────────────────
	// GoalSelectSwitchMargin is the minimum effective-score advantage a
	// challenger goal must have over the current goal to displace it.
	// Hysteresis safety against goal-thrash on noisy scoring inputs.
	// Default 5.0; floats so weights/contextMod can produce fractional
	// scores. Chunk 4.2.
	GoalSelectSwitchMargin ConfigFloat `yaml:"GoalSelectSwitchMargin"`

	// GoalSelectMinHoldRounds is the minimum number of rounds the
	// current goal must be held before any switch is allowed. ≈ 5 min
	// at default tick rate (default 100). Chunk 4.2.
	GoalSelectMinHoldRounds ConfigInt `yaml:"GoalSelectMinHoldRounds"`

	// GoalSelectTickEnabled is the master kill-switch for the tick-driven
	// recompute path. Eager recompute on Add/Remove/Clear still fires
	// when false (cache stays consistent). Default true. Chunk 4.2.
	GoalSelectTickEnabled ConfigBool `yaml:"GoalSelectTickEnabled"`

	// GoalPruneIntervalRounds is how often (in rounds) the per-mob goal
	// prune sweep runs. Staggered per mob to avoid a synchronized spike.
	// chunk 4.6.
	GoalPruneIntervalRounds ConfigInt `yaml:"GoalPruneIntervalRounds"`

	// GoalAbandonDormantRounds is how many consecutive rounds a goal's
	// context score may stay at ~0 before it is abandoned (pruned).
	// chunk 4.6.
	GoalAbandonDormantRounds ConfigInt `yaml:"GoalAbandonDormantRounds"`

	// GuardWarnGraceRounds is how many rounds a warned (Cold-rep) player may
	// remain present before a guard escalates the warning to an attack. 5.1a.
	GuardWarnGraceRounds ConfigInt `yaml:"GuardWarnGraceRounds"`

	// JusticeCrimeLookbackRounds is the recency window for the unresolved-crime
	// "wanted" signal used by guard enforcement. 5.1a.
	JusticeCrimeLookbackRounds ConfigInt `yaml:"JusticeCrimeLookbackRounds"`

	// JusticeBountyExpiryRounds is how long an auto-declared town-faction
	// bounty stays open before lapsing. 5.1b.
	JusticeBountyExpiryRounds ConfigInt `yaml:"JusticeBountyExpiryRounds"`

	// JusticeBountyMurderMult scales an auto-bounty's gold when triggered by
	// an identified murder. 5.1b.
	JusticeBountyMurderMult ConfigFloat `yaml:"JusticeBountyMurderMult"`

	// JusticeBountyRepMultMax is the max gold multiplier at maximum hostility
	// (rep -100) for rep-triggered auto-bounties. 5.1b.
	JusticeBountyRepMultMax ConfigFloat `yaml:"JusticeBountyRepMultMax"`

	// ArrestResistGraceRounds is the window between a guard declaring an arrest
	// and hauling the player off, during which the player may fight back. 5.1c.
	ArrestResistGraceRounds ConfigInt `yaml:"ArrestResistGraceRounds"`

	// JusticeFineDecayPerRound is gold the arrest fine drops per round served;
	// sentence length = fine / this. 5.1c.
	JusticeFineDecayPerRound ConfigInt `yaml:"JusticeFineDecayPerRound"`

	// JusticeArrestRepReset is the reputation floor restored with the issuing
	// faction when a sentence is served/paid (only raises, never lowers). 5.1c.
	JusticeArrestRepReset ConfigInt `yaml:"JusticeArrestRepReset"`

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
	CrafterEnabled       ConfigBool `yaml:"CrafterEnabled"`       // Enable mob autonomous crafting (default true)
	CrafterRareThreshold ConfigInt  `yaml:"CrafterRareThreshold"` // SkillMinimum at or above which a craft is considered rare (default 3)

	// Deprecated: replaced by RestockCadenceTier{50,40,30,20,10}*. Kept
	// only so old config.yaml files load without error. Remove after
	// one deploy cycle.
	CrafterMaterialRestockRate ConfigInt `yaml:"CrafterMaterialRestockRate"`

	// Per-rarity-tier restock cadences (game-time hours). Replaces the
	// single CrafterMaterialRestockRate. Higher rarity tiers (= more
	// common) fire faster; lower tiers (= rarer) fire slowly as a
	// backstop on top of forager / player-sale input.
	RestockCadenceTier50Hours ConfigInt `yaml:"RestockCadenceTier50Hours"`
	RestockCadenceTier40Hours ConfigInt `yaml:"RestockCadenceTier40Hours"`
	RestockCadenceTier30Hours ConfigInt `yaml:"RestockCadenceTier30Hours"`
	RestockCadenceTier20Hours ConfigInt `yaml:"RestockCadenceTier20Hours"`
	RestockCadenceTier10Days  ConfigInt `yaml:"RestockCadenceTier10Days"`

	// ── GOSSIP SYSTEM ────────────────────────────────────────────────────────
	GossipIntervalRounds ConfigInt `yaml:"GossipIntervalRounds"` // Rounds between gossip broadcasts for "gossiper" group mobs (default 75)

	// ── MOON PHASES ───────────────────────────────────────────────────────────
	MoonStatModMax          ConfigFloat `yaml:"MoonStatModMax"`          // Max fractional stat modifier from moon phases, e.g. 0.05 = ±5% (default 0.05)
	CarryCapacityMultiplier ConfigFloat `yaml:"CarryCapacityMultiplier"` // Strength multiplier for carry capacity in lbs (default 0.65)

	// ── TOXICITY ────────────────────────────────────────────────────────────
	ToxicityDecayPerTick      ConfigFloat `yaml:"ToxicityDecayPerTick"`      // Points decayed per regen tick (default 1.0)
	ToxicityBaseMax           ConfigFloat `yaml:"ToxicityBaseMax"`           // Base max before vitality bonus (default 100)
	ToxicityVitalityScale     ConfigFloat `yaml:"ToxicityVitalityScale"`     // Vitality divisor for max bonus (default 5)
	ToxicitySicknessDamagePct ConfigFloat `yaml:"ToxicitySicknessDamagePct"` // % max-HP/tick acute harm at top band (default 0.02)
	ToxicityHighDecaySlowMult ConfigFloat `yaml:"ToxicityHighDecaySlowMult"` // decay multiplier when toxicity >= 75% (default 0.5 = clears slower)

	// ── BLOOM ───────────────────────────────────────────────────────────────
	BloomAddictionPerDose      ConfigInt   `yaml:"BloomAddictionPerDose"`      // addiction gained per dose (default 1)
	BloomAddictionDecayRounds  ConfigInt   `yaml:"BloomAddictionDecayRounds"`  // rounds of abstinence per -1 addiction (default 300)
	BloomWithdrawalOnsetRounds ConfigInt   `yaml:"BloomWithdrawalOnsetRounds"` // abstinence rounds before withdrawal (default 60)
	BloomMutationAdvanceChance ConfigFloat `yaml:"BloomMutationAdvanceChance"` // chance/dose to advance strongest mutation (default 0.50)
	BloomNewMutationChance     ConfigFloat `yaml:"BloomNewMutationChance"`     // chance/dose to instead grant a new mutation (default 0.10)
	BloomCommunionRounds       ConfigInt   `yaml:"BloomCommunionRounds"`       // communion high duration in rounds (default 30)
	BloomCrashRoundsMult       ConfigFloat `yaml:"BloomCrashRoundsMult"`       // crash duration = communion * this (default 2.5)

	// ── MANIFESTATION / COMPANION SCALING ───────────────────────────────────
	ManifestStatScaleChaFactor   ConfigInt   `yaml:"ManifestStatScaleChaFactor"`   // Charisma divisor for companion stat scaling (default 150)
	ManifestStatScaleSkillFactor ConfigFloat `yaml:"ManifestStatScaleSkillFactor"` // Manifestation skill additive factor (default 0.02)

	// ── COMPANION CONVICTION ECONOMY ────────────────────────────────────────
	CompanionReserveSkillPct      ConfigFloat `yaml:"CompanionReserveSkillPct"`      // Reservation reduction per manifestation skill point (default 0.01)
	CompanionReserveSkillCap      ConfigFloat `yaml:"CompanionReserveSkillCap"`      // Max reduction from skill (default 0.55, cap reached at manifestation 55)
	CompanionReserveMutPctPerRank ConfigFloat `yaml:"CompanionReserveMutPctPerRank"` // Reservation reduction per Manifester mutation rank (default 0.06)
	CompanionReserveMutCap        ConfigFloat `yaml:"CompanionReserveMutCap"`        // Max reduction from the Manifester mutation (default 0.24)
	CompanionReserveTotalCap      ConfigFloat `yaml:"CompanionReserveTotalCap"`      // Combined reduction ceiling (default 0.79)
	CompanionSoftCap              ConfigInt   `yaml:"CompanionSoftCap"`              // Soft count backstop; real limit is the Conviction budget (default 5)
	CompanionSoftCapApex          ConfigInt   `yaml:"CompanionSoftCapApex"`          // Soft count backstop with the Manifester apex (default 7)
	CompanionReserveDefault       ConfigInt   `yaml:"CompanionReserveDefault"`       // Fallback reserve when a summon spell omits summon_conviction_reserve (default 350)
	CompanionCastingFloorPct      ConfigFloat `yaml:"CompanionCastingFloorPct"`      // Conviction fraction kept unreservable as casting headroom (default 0.0)
	HomunculusCraftScale          ConfigFloat `yaml:"HomunculusCraftScale"`          // Chrysifier: homunculus statpool = (sum of crafting-skill levels) * this (default 4.0)
	HomunculusConvictionReserve   ConfigInt   `yaml:"HomunculusConvictionReserve"`   // Chrysifier: base Conviction the homunculus reserves before reduction (default 1000)

	// ── SHOP ECONOMY ─────────────────────────────────────────────────────────
	ShopBuyRatio              ConfigFloat `yaml:"ShopBuyRatio,omitempty"`              // Base buy/sell spread: NPC buy offer = baseValue * BuyRatio * scarcityMult (default 0.50)
	ShopPriceFloor            ConfigFloat `yaml:"ShopPriceFloor,omitempty"`            // Minimum scarcity multiplier when stock is very high (default 0.25)
	ShopPriceCeiling          ConfigFloat `yaml:"ShopPriceCeiling,omitempty"`          // Maximum scarcity multiplier when stock is zero (default 5.0)
	ShopAbundanceThreshold    ConfigFloat `yaml:"ShopAbundanceThreshold,omitempty"`    // Stock/restock ratio at which price hits the floor (default 3.0)
	ShopMaterialReserve       ConfigInt   `yaml:"ShopMaterialReserve,omitempty"`       // Units of each material a crafter mob reserves before selling (default 1)
	DefaultPricingBaselineQty ConfigInt   `yaml:"DefaultPricingBaselineQty,omitempty"` // Pricing baseline (scarcity-curve denominator) for stock entries with RestockQty==0, e.g. crafted/caravan-delivered goods (default 3)
	// CrafterIngredientReservePct is the fraction of an ingredient's
	// MaxStock the crafter mob keeps in reserve when deciding whether to
	// craft. Prevents the crafter from consuming its own stock to a level
	// where players can't buy. Per-ingredient check, floor of 1.
	CrafterIngredientReservePct ConfigFloat `yaml:"CrafterIngredientReservePct"`        // Fraction of MaxStock kept as reserve (default 0.25)
	ShopGoldReserveRatio        ConfigFloat `yaml:"ShopGoldReserveRatio,omitempty"`     // Fraction of gold pool a shop keeps in reserve before buying (default 0.50)
	ShopMaxStockMultiplier      ConfigFloat `yaml:"ShopMaxStockMultiplier,omitempty"`   // Global multiplier on EffectiveMaxStock — chunk 3.8 bumped to 2.0 to give the cross-city caravan room to build surplus (default 2.0)
	ShopOverstockDecayRounds    ConfigInt   `yaml:"ShopOverstockDecayRounds,omitempty"` // Rounds an over-baseline stock entry must sit un-grown before one unit decays (default 21600 ≈ several in-game days)
	ShopOverstockDecayQty       ConfigInt   `yaml:"ShopOverstockDecayQty,omitempty"`    // Units removed per decay fire (default 1)
	BarterMaxDiscount           ConfigFloat `yaml:"BarterMaxDiscount,omitempty"`        // Max fractional price reduction a player can get via bartering (default 0.15)
	BarterMaxBonus              ConfigFloat `yaml:"BarterMaxBonus,omitempty"`           // Max fractional sell-price bonus a player can get via bartering (default 0.15)
	StorageFeePerItem           ConfigInt   `yaml:"StorageFeePerItem"`                  // Gold charged per stored item per game month (default 1)

	// ── WAREHOUSES (Stage 3 ferry system) ────────────────────────────────────
	WarehouseItemCap      ConfigInt `yaml:"WarehouseItemCap,omitempty"`      // Per-item stock cap in city warehouses (default 4,000,000 — effectively unbounded)
	WarehouseAccrualHours ConfigInt `yaml:"WarehouseAccrualHours,omitempty"` // Game-hours between ambient accrual ticks (default 2)

	// ── LOOT ──────────────────────────────────────────────────────────────────
	LootBudgetScalar ConfigFloat `yaml:"LootBudgetScalar"` // Multiplier for sqrt(goldPaid) loot budget (default 7.0)
	GoldPerAffixPoint ConfigFloat `yaml:"GoldPerAffixPoint"` // Gold value per affix cost-point on instance/affixed loot (default 3.0)
	ShopAffixedStockCap ConfigInt `yaml:"ShopAffixedStockCap"` // Max per-instance affixed items a shop resells before evicting oldest (default 8)

	// ── INSTANCES ────────────────────────────────────────────────────────────
	InstanceStatPoolCap ConfigInt `yaml:"InstanceStatPoolCap"` // Max stat pool per mob in instances (default 50000, 0=uncapped)

	// ── CRASH SITE (#22) ─────────────────────────────────────────────────────
	CrashSiteSuppressionFactor ConfigFloat `yaml:"CrashSiteSuppressionFactor"` // inside the buried hull (#22), spell power and mutation combat bonuses are scaled to this fraction; 0=fully suppressed, 1=no effect (default 0.35)

	// ── BOSS INTERRUPTS ──────────────────────────────────────────────────────
	// General-purpose disruptor allowlists (not crash-site-specific). Any mob
	// mid-fold-cast (Character.IsCasting() / Activity.IsCasting()) has its
	// cast interrupted via actions.InterruptTargetCast when hit by a thrown
	// item whose id is in BossInterruptItemIds, or a player spell whose id is
	// in BossInterruptSpellIds. Generic melee never interrupts. Defaults set
	// in validateMisc(); see IsBossInterruptItem / IsBossInterruptSpell.
	BossInterruptItemIds  []int    `yaml:"BossInterruptItemIds"`
	BossInterruptSpellIds []string `yaml:"BossInterruptSpellIds"`

	// ── CARAVAN SYSTEM ───────────────────────────────────────────────────────
	// CaravanServedZones lists zone display names whose vendor mobs do NOT
	// auto-restock — they restock only on caravan visit. Mobs in zones not
	// in this list keep the legacy per-mob restock tick.
	CaravanServedZones []string `yaml:"CaravanServedZones"`

	// CaravanDepotDwellRounds is the number of rounds the caravan rests at
	// each depot between transit legs. ~720 ≈ 48 min real ≈ a full game
	// day. Stage 3.1 doubled this from 360 so foragers are the day-to-day
	// supply pipeline; caravans now arrive about once per game day.
	CaravanDepotDwellRounds ConfigInt `yaml:"CaravanDepotDwellRounds"`

	// FernwayPickupDwellRounds is the dwell time at the Fernway forager
	// meeting point (North Road 4038) on each transit leg. Default 6.
	FernwayPickupDwellRounds ConfigInt `yaml:"FernwayPickupDwellRounds"`

	// ── FORAGER SYSTEM (Stage 3.1) ───────────────────────────────────────────

	// ForagerCarryThresholdPct is the carry-capacity ratio (0.0-1.0) at
	// which the forager heads home for delivery. Default 0.75.
	ForagerCarryThresholdPct ConfigFloat `yaml:"ForagerCarryThresholdPct"`

	// ForagerHPRecallThresholdPct is the HP ratio (0.0-1.0) below which
	// the forager casts fold-recall as an emergency escape. Default 0.50.
	ForagerHPRecallThresholdPct ConfigFloat `yaml:"ForagerHPRecallThresholdPct"`

	// ForagerHealPotionThresholdPct is the HP ratio (0.0-1.0) below which
	// the forager auto-drinks a healing salve. Default 0.75.
	ForagerHealPotionThresholdPct ConfigFloat `yaml:"ForagerHealPotionThresholdPct"`

	// ForagerWaitTimeoutRounds is the maximum rounds the Fernway forager
	// idles at the meeting point waiting for the caravan before recalling
	// home with the satchel. Default 150.
	ForagerWaitTimeoutRounds ConfigInt `yaml:"ForagerWaitTimeoutRounds"`

	// ForagerRestCarryThreshold (Stage 3.4) is the carry-capacity ratio
	// (0.0-1.0) above which the forager stays resting at home instead of
	// cycling back to forage. Prevents futile foraging loops when local
	// vendors are saturated (MaxStock cap reached). Default 0.5.
	ForagerRestCarryThreshold ConfigFloat `yaml:"ForagerRestCarryThreshold"`

	// ChestBackpressureResumePct (Stage 5.4) is the storage-lockbox fill
	// fraction (0.0-1.0) at/below which a rested forager is allowed to
	// start a new gather cycle. While the chest is fuller than this, the
	// forager stays resting until the vendor-backfill drains it. Default 0.9.
	ChestBackpressureResumePct ConfigFloat `yaml:"ChestBackpressureResumePct,omitempty"`

	// ForagerRestDurationRounds is how long a forager stays at sanctuary
	// before re-entering the territory. Gated by HP-full and carry-ratio
	// checks too, so the actual rest can be longer if the forager
	// arrived hurt or over-encumbered. Default 40 rounds (~3 real
	// minutes / ~1 game-hour at the default round cadence).
	ForagerRestDurationRounds ConfigInt `yaml:"ForagerRestDurationRounds"`

	// ForagerLockboxCapacity caps how many items a sanctuary lockbox
	// can hold. When the box is full, the forager falls back to the
	// Stage 3.4 rest-extension behavior until a player picks the box
	// open and clears space.
	ForagerLockboxCapacity ConfigInt `yaml:"ForagerLockboxCapacity"`

	// ForagerStuckThresholdRounds is the watchdog timeout. If a forager
	// sits in the same state for more than this many rounds, it is
	// force-reset to Recalling so it heads home, dumps satchel, and
	// re-cycles. Logs a Warn on reset for ops visibility.
	ForagerStuckThresholdRounds ConfigInt `yaml:"ForagerStuckThresholdRounds"`

	// ForagerStoringWatchdogRounds is the maximum number of rounds a
	// forager will spend in StateStoring before bailing to StateRecalling.
	// Prevents infinite loops if the chest workflow stalls (e.g.,
	// persistently full chest or unreachable chest room).
	ForagerStoringWatchdogRounds ConfigInt `yaml:"ForagerStoringWatchdogRounds"`

	// ── ECONOMY HEALTH DASHBOARD ─────────────────────────────────────────────
	EconomySnapshotIntervalHours ConfigInt   `yaml:"EconomySnapshotIntervalHours"` // Wall-clock cadence (default 1)
	EconomySnapshotRetentionDays ConfigInt   `yaml:"EconomySnapshotRetentionDays"` // Auto-snapshot retention (default 30)
	EconomyScoreWeightShop       ConfigFloat `yaml:"EconomyScoreWeightShop"`       // Overall-score weight for shops (default 0.6)
	EconomyScoreWeightCaravan    ConfigFloat `yaml:"EconomyScoreWeightCaravan"`    // (default 0.2)
	EconomyScoreWeightForager    ConfigFloat `yaml:"EconomyScoreWeightForager"`    // (default 0.2)

	// ── ECONOMY SCORING — TtR TARGETS ────────────────────────────────────────
	// Time-to-Refill targets per rarity tier (game-time). Throughput score
	// penalizes items that take longer than their tier's target to refill.
	// Tier 50 (common) should refill in ~3 game-hours; tier 10 (rare) in ~7 game-days.
	TtRTargetTier50Hours ConfigInt `yaml:"TtRTargetTier50Hours"`
	TtRTargetTier40Hours ConfigInt `yaml:"TtRTargetTier40Hours"`
	TtRTargetTier30Hours ConfigInt `yaml:"TtRTargetTier30Hours"`
	TtRTargetTier20Days  ConfigInt `yaml:"TtRTargetTier20Days"`
	TtRTargetTier10Days  ConfigInt `yaml:"TtRTargetTier10Days"`

	// TtRWindowGameDays is the rolling window of completed depletion→refill
	// events used by ThroughputScore (default 7 game-days).
	TtRWindowGameDays ConfigInt `yaml:"TtRWindowGameDays"`

	// ── ECONOMY SCORING — LOGISTICS ──────────────────────────────────────────
	// LogisticsStuckRounds is the round count after which a caravan or forager
	// is considered stuck; the stuck multiplier is applied to its health score.
	// Default 3000 (more aggressive than the MVP 5000).
	LogisticsStuckRounds ConfigInt `yaml:"LogisticsStuckRounds"`
	// LogisticsStuckMultiplier scales the base logistics score when stuck
	// (default 0.4 — a stuck entity reads ~40% of its healthy score).
	LogisticsStuckMultiplier ConfigFloat `yaml:"LogisticsStuckMultiplier"`

	// ── ECONOMY SCORING — OVERALL BLEND ─────────────────────────────────────
	// Weights for the five-axis OverallScore blend. Must sum to 1.0.
	// Logistics is not blended here; it is displayed as a standalone panel.
	ScoreWeightStock      ConfigFloat `yaml:"ScoreWeightStock"`
	ScoreWeightInput      ConfigFloat `yaml:"ScoreWeightInput"`
	ScoreWeightThroughput ConfigFloat `yaml:"ScoreWeightThroughput"`
	ScoreWeightShopGold   ConfigFloat `yaml:"ScoreWeightShopGold"`

	// ── OPINIONS / DISPOSITION ───────────────────────────────────────────────
	OpinionAttackBump              ConfigInt `yaml:"OpinionAttackBump"`              // Disposition delta when a player initiates aggression on a mob (default -15)
	DispositionDecayHalfLifeRounds ConfigInt `yaml:"DispositionDecayHalfLifeRounds"` // Rounds for one half-life of disposition decay toward default (default 100000; 0 disables decay)

	// ── FACTIONS ─────────────────────────────────────────────────────────────
	FactionMemberKillRep       ConfigInt `yaml:"FactionMemberKillRep"`       // Rep delta when a player kills a member of a defined faction (default -10) — DEPRECATED, retained for any non-citizen faction fallback path. Citizen factions use CrimeRepDeltaMurder via internal/crimes.
	CrimeRepDeltaMurder        ConfigInt `yaml:"CrimeRepDeltaMurder"`        // Rep delta on murder crime with identified perpetrator (default -25)
	CrimeRepDeltaAssault       ConfigInt `yaml:"CrimeRepDeltaAssault"`       // Rep delta on assault crime with identified perpetrator (default -10)
	CrimeRepDeltaTheft         ConfigInt `yaml:"CrimeRepDeltaTheft"`         // Rep delta on theft crime with identified perpetrator (default -5)
	CrimeStaleAfterRounds      ConfigInt `yaml:"CrimeStaleAfterRounds"`      // Rounds after which an unresolved crime is auto-snapped to stale (default 7884000 — ~365 game-days at 4-second rounds)
	KnowledgeObservationLogMax ConfigInt `yaml:"KnowledgeObservationLogMax"` // Max observation log entries per record (FIFO) (default 32)

	// ── BOUNTIES ────────────────────────────────────────────────────────────
	BountyGoldDefaultMultiplier ConfigFloat `yaml:"BountyGoldDefaultMultiplier"` // Multiplier on statpool for default gold reward (default 0.5)
	BountyGoldFloor             ConfigInt   `yaml:"BountyGoldFloor"`             // Minimum gold floor for any bounty (default 50)

	// ── BOUNTY HUNTING (5.2) ────────────────────────────────────────────
	BountyHunterGoldThreshold      ConfigInt   `yaml:"BountyHunterGoldThreshold"`      // Single open bounty >= this dispatches a hunter (default 500)
	BountyHunterBaseStatpool       ConfigInt   `yaml:"BountyHunterBaseStatpool"`       // Base of hunter scaled statpool (default 250)
	BountyHunterStatpoolPerGold    ConfigFloat `yaml:"BountyHunterStatpoolPerGold"`    // Statpool added per gold of triggering bounty (default 0.25)
	BountyHunterMinStatpool        ConfigInt   `yaml:"BountyHunterMinStatpool"`        // Clamp floor for hunter statpool (default 300)
	BountyHunterMaxStatpool        ConfigInt   `yaml:"BountyHunterMaxStatpool"`        // Clamp ceiling for hunter statpool (default 500)
	BountyHunterRedispatchCooldown ConfigInt   `yaml:"BountyHunterRedispatchCooldown"` // Rounds before re-dispatch after a hunter dies (default 500)
	BountyHunterGearGoldDivisor    ConfigInt   `yaml:"BountyHunterGearGoldDivisor"`    // gearGold = statpool / this, fed to GenerateAffixedItem (default 5)

	// ── EQUIPMENT-AWARE SHOPPING (5.3) ──────────────────────────────────
	MobUpgradeGoldReserve ConfigInt   `yaml:"MobUpgradeGoldReserve"` // Gold a shopping mob keeps in reserve, won't spend below (default 50)
	MobUpgradeMinDelta    ConfigFloat `yaml:"MobUpgradeMinDelta"`    // Minimum itemvalue swap delta worth buying (default 1.0)

	// ── FACTS & EVENTS ──────────────────────────────────────────────────────
	FactsHeardEventsMax ConfigInt `yaml:"FactsHeardEventsMax"` // Max facts_heard events per mobs instance (default 32)

	// ── PINNACLE ITEMS ───────────────────────────────────────────────────
	BandolierAttuneRounds         ConfigInt `yaml:"BandolierAttuneRounds"`         // Rounds of re-attunement after bandolier contents change (default 100)
	SentientChatterCooldownRounds ConfigInt `yaml:"SentientChatterCooldownRounds"` // Min rounds between sentient item lines (default 20)
	SentientChatterChancePct      ConfigInt `yaml:"SentientChatterChancePct"`      // Percent chance per eligible round that a sentient item speaks (default 15)
}

func (b *Balance) Validate() {
	b.validateCombat()
	b.validateProgression()
	b.validateMobs()
	b.validateSpells()
	b.validateDiscovery()
	b.validateShops()
	b.validateMisc()
}

func GetBalanceConfig() Balance {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()

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

// IsCaravanServedZone reports whether the named zone is in the
// CaravanServedZones list. Case-sensitive — match the zone display
// name exactly.
func (b Balance) IsCaravanServedZone(zone string) bool {
	for _, z := range b.CaravanServedZones {
		if z == zone {
			return true
		}
	}
	return false
}

// IsBossInterruptItem reports whether the given item id is a configured
// boss-interrupt disruptor (see BossInterruptItemIds).
func (b Balance) IsBossInterruptItem(itemId int) bool {
	for _, id := range b.BossInterruptItemIds {
		if id == itemId {
			return true
		}
	}
	return false
}

// IsBossInterruptSpell reports whether the given spell id is a configured
// boss-interrupt disruption spell (see BossInterruptSpellIds).
func (b Balance) IsBossInterruptSpell(spellId string) bool {
	for _, id := range b.BossInterruptSpellIds {
		if id == spellId {
			return true
		}
	}
	return false
}
