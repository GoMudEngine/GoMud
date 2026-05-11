package itemvalue

// PhysicalBruiser fits stat=fighting or behavior in
// {generic_fighter, melee_self_buff, leader}.
var PhysicalBruiser = WeightProfile{
	Name:                       "PhysicalBruiser",
	PhysicalDamageWeight:       1.0,
	SpellDamageWeight:          0.1,
	PhysicalMitigationWeight:   1.0,
	MagicalMitigationWeight:    0.6,
	ConvictionMitigationWeight: 0.4,
	StatWeights: map[string]float64{
		"strength":   1.5,
		"dexterity":  1.2,
		"vitality":   1.3,
		"willpower":  0.3,
		"charisma":   0.3,
		"perception": 0.7,
	},
	WeightPenaltyPerLb:     0.5,
	EncumbranceTierPenalty: 25,
	DualWieldBonus:         80,
	ShieldBonus:            20,
	TwoHandedBonus:         60,
}

// PhysicalTank fits stat=tank or behavior=tank_taunter.
var PhysicalTank = WeightProfile{
	Name:                       "PhysicalTank",
	PhysicalDamageWeight:       0.5,
	SpellDamageWeight:          0.1,
	PhysicalMitigationWeight:   1.2,
	MagicalMitigationWeight:    1.0,
	ConvictionMitigationWeight: 1.2,
	StatWeights: map[string]float64{
		"vitality":   1.5,
		"strength":   1.0,
		"charisma":   1.3,
		"willpower":  0.5,
		"dexterity":  0.7,
		"perception": 0.5,
	},
	WeightPenaltyPerLb:     0.2,
	EncumbranceTierPenalty: 10,
	DualWieldBonus:         0,
	ShieldBonus:            80,
	TwoHandedBonus:         -20,
}

// Stealth fits behavior in {ambusher, lookout}.
var Stealth = WeightProfile{
	Name:                       "Stealth",
	PhysicalDamageWeight:       1.1,
	SpellDamageWeight:          0.1,
	PhysicalMitigationWeight:   0.6,
	MagicalMitigationWeight:    0.4,
	ConvictionMitigationWeight: 0.3,
	StatWeights: map[string]float64{
		"dexterity":  1.5,
		"perception": 1.3,
		"strength":   1.0,
		"vitality":   0.4,
		"willpower":  0.3,
		"charisma":   0.3,
	},
	WeightPenaltyPerLb:     1.8,
	EncumbranceTierPenalty: 80,
	DualWieldBonus:         100,
	ShieldBonus:            0,
	TwoHandedBonus:         -40,
}

// MagicalPure fits behavior=pure_caster.
var MagicalPure = WeightProfile{
	Name:                       "MagicalPure",
	PhysicalDamageWeight:       0.2,
	SpellDamageWeight:          1.5,
	PhysicalMitigationWeight:   0.5,
	MagicalMitigationWeight:    1.0,
	ConvictionMitigationWeight: 0.5,
	StatWeights: map[string]float64{
		"willpower":  1.5,
		"perception": 1.2,
		"charisma":   1.0,
		"vitality":   0.5,
		"dexterity":  0.5,
		"strength":   0.3,
	},
	WeightPenaltyPerLb:     1.5,
	EncumbranceTierPenalty: 60,
	DualWieldBonus:         70,
	ShieldBonus:            -40,
	TwoHandedBonus:         80,
}

// MagicalSupport fits behavior=support_caster, or stat=casting
// fallback when no behavior label is set.
var MagicalSupport = WeightProfile{
	Name:                       "MagicalSupport",
	PhysicalDamageWeight:       0.2,
	SpellDamageWeight:          1.2,
	PhysicalMitigationWeight:   0.7,
	MagicalMitigationWeight:    1.0,
	ConvictionMitigationWeight: 0.8,
	StatWeights: map[string]float64{
		"willpower":  1.4,
		"charisma":   1.3,
		"perception": 1.1,
		"vitality":   0.8,
		"dexterity":  0.5,
		"strength":   0.3,
	},
	WeightPenaltyPerLb:     1.5,
	EncumbranceTierPenalty: 60,
	DualWieldBonus:         -80,
	ShieldBonus:            80,
	TwoHandedBonus:         -20,
}

// Neutral is the default for empty archetypes,
// combat_passive, prey, and all noncombat_* roles.
var Neutral = WeightProfile{
	Name:                       "Neutral",
	PhysicalDamageWeight:       0.7,
	SpellDamageWeight:          0.5,
	PhysicalMitigationWeight:   0.7,
	MagicalMitigationWeight:    0.7,
	ConvictionMitigationWeight: 0.7,
	StatWeights:                map[string]float64{}, // all stats default to 1.0
	WeightPenaltyPerLb:         1.0,
	EncumbranceTierPenalty:     35,
	DualWieldBonus:             20,
	ShieldBonus:                20,
	TwoHandedBonus:             0,
}

// ProfileFor resolves a mob's archetype fields to a named
// WeightProfile. Behavior archetype takes precedence; stat
// archetype is the fallback when no behavior label is set.
// Empty input returns Neutral.
func ProfileFor(statArchetype, behaviorArchetype string) WeightProfile {
	// (1) Behavior archetype takes precedence.
	switch behaviorArchetype {
	case "tank_taunter":
		return PhysicalTank
	case "pure_caster":
		return MagicalPure
	case "support_caster":
		return MagicalSupport
	case "ambusher", "lookout":
		return Stealth
	case "generic_fighter", "melee_self_buff", "leader":
		return PhysicalBruiser
	case "combat_passive", "prey",
		"noncombat_passive", "noncombat_questgiver",
		"noncombat_shopkeeper":
		return Neutral
	}
	// (2) Stat archetype fallback when no behavior label.
	switch statArchetype {
	case "fighting":
		return PhysicalBruiser
	case "casting":
		return MagicalSupport
	case "tank":
		return PhysicalTank
	}
	// (3) Default.
	return Neutral
}
