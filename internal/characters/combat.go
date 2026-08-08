package characters

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
)

func (c *Character) GetDefaultDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {
	// default racial
	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo == nil {
		// GetSpecies is a bare map lookup and returns nil for an unknown or
		// unmigrated SpeciesId. Degrade to a bare profile (stat-derived dice
		// only) rather than panicking mid-combat.
		speciesInfo = &species.Species{}
	}

	attacks = speciesInfo.Damage.Attacks
	dCount = speciesInfo.Damage.DiceCount
	dSides = speciesInfo.Damage.SideCount
	bonus = speciesInfo.Damage.BonusDamage
	buffOnCrit = speciesInfo.Damage.CritBuffIds

	dCount += int(math.Floor((float64(c.Stats.Dexterity.ValueAdj) / 50)))
	dSides += int(math.Floor((float64(c.Stats.Strength.ValueAdj) / 12)))
	bonus += int(math.Floor((float64(c.Stats.Charisma.ValueAdj) / 25)))

	if dCount < speciesInfo.Damage.DiceCount {
		dCount = speciesInfo.Damage.DiceCount
	}
	if dSides < speciesInfo.Damage.SideCount {
		dSides = speciesInfo.Damage.SideCount
	}

	return attacks, dCount, dSides, bonus, buffOnCrit
}

// GetDefaultDistributionDamage returns distribution damage parameters for unarmed combat.
// Uses CalculateUnarmedDamage to scale with Strength and Unarmed Combat skill.
// This provides meaningful progression for unarmed fighters.
func (c *Character) GetDefaultDistributionDamage() (attacks int, baseDamage float64, variance float64, buffOnCrit []int) {
	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo == nil {
		// See GetDefaultDiceRoll: nil for an unknown SpeciesId. The attacks
		// clamp below turns the zero value into a valid single attack.
		speciesInfo = &species.Species{}
	}

	attacks = speciesInfo.Damage.Attacks
	if attacks < 1 {
		attacks = 1
	}
	buffOnCrit = speciesInfo.Damage.CritBuffIds

	// Use skill-based unarmed damage calculation (Stage 7.3)
	baseDamage, variance = c.CalculateUnarmedDamage()

	return attacks, baseDamage, variance, buffOnCrit
}

// CalculateUnarmedDamage returns the base damage and variance for unarmed attacks.
// This function is designed to be extensible for future conditions/mutations.
//
// Formula: baseDamage = baseValue + (Strength / strengthDivisor) + (UnarmedSkill / skillDivisor)
//
// Scaling examples (at 100 Strength):
//   - Skill 0 (untrained):  2 + 4 + 0  = 6 damage
//   - Skill 50 (trained):   2 + 4 + 5  = 11 damage
//   - Skill 100 (master):   2 + 4 + 10 = 16 damage
//
// Extension points for future conditions/mutations:
//   - baseDamage can be modified by multipliers (e.g., "Stone Fists" mutation: baseDamage *= 1.5)
//   - variance can be modified (e.g., "Precise Strikes" condition: variance *= 0.5)
//   - Additional additive bonuses (e.g., "Enhanced Strength" buff: +5 damage)
//
// To add a buff/condition/mutation that affects unarmed damage:
//  1. Check for the buff/condition after base calculation
//  2. Apply multipliers: baseDamage *= multiplier
//  3. Apply additive bonuses: baseDamage += bonus
//  4. Modify variance if needed: variance *= varianceMultiplier
func (c *Character) CalculateUnarmedDamage() (baseDamage float64, variance float64) {
	b := configs.GetBalanceConfig()
	baseValue := float64(b.UnarmedBaseDamage)
	strengthDivisor := float64(b.UnarmedStrengthDivisor)
	skillDivisor := float64(b.UnarmedSkillDivisor)
	baseVariance := float64(b.UnarmedBaseVariance)

	// Calculate base damage from stats and skill
	strengthBonus := float64(c.Stats.Strength.ValueAdj) / strengthDivisor
	skillBonus := float64(c.GetSkillLevel(skills.UnarmedCombat)) / skillDivisor

	baseDamage = baseValue + strengthBonus + skillBonus

	// Base variance scales slightly with skill (more skill = more consistent)
	// High skill fighters are more consistent, low skill fighters are erratic
	skillLevel := c.GetSkillLevel(skills.UnarmedCombat)
	varianceReduction := float64(skillLevel) / 50.0 // 0 at skill 0, 2 at skill 100
	variance = math.Max(1.0, baseVariance-varianceReduction)

	// --- FUTURE EXTENSION POINT: Buffs/Conditions/Mutations ---
	// Example implementations (commented out for now):
	//
	// if c.HasBuffFlag(buffs.StoneFists) {
	//     baseDamage *= 1.5  // Stone Fists: +50% damage
	//     variance *= 1.2    // But less precise (heavier strikes)
	// }
	//
	// if c.HasBuffFlag(buffs.PreciseStrikes) {
	//     variance *= 0.5    // Precise Strikes: Half variance (more consistent)
	// }
	//
	// if c.HasBuffFlag(buffs.EnhancedStrength) {
	//     baseDamage += 5.0  // Flat +5 damage bonus
	// }
	//
	// if c.HasCondition("weakened") {
	//     baseDamage *= 0.7  // Weakened: -30% damage
	// }
	//
	// if c.HasMutation("razor-claws") {
	//     baseDamage += 3.0  // Razor Claws: +3 base damage
	//     variance += 1.0    // Claws add slight variance
	// }
	//
	// if c.HasMutation("padded-fists") {
	//     baseDamage *= 0.8  // Padded Fists: -20% damage
	//     variance *= 0.6    // But more consistent
	// }
	// --- END EXTENSION POINT ---

	// Stage 12.1: Natural weapon bonus from mutations (Clawed Hands etc.)
	baseDamage += mutations.GetNaturalWeaponBonus(c.Mutations)

	// Ensure minimums
	baseDamage = math.Max(1.0, baseDamage)
	variance = math.Max(0.5, variance)

	return baseDamage, variance
}

// GetPhysicalMitigation returns total physical mitigation as a fraction (0.0–1.0).
// Sources: equipment physical_mitigation, mutations, species natural armor,
// shield spells, and physical_mitigation buff statmods.
// Incorporeal mutation scales gear-derived mitigation.
// (The legacy Character.GetDefense / ItemSpec.DamageReduction summation was
// removed 2026-08-03; the two default-world status templates that dynamically
// dispatched .GetDefense were repointed at this method.)
func (c *Character) GetPhysicalMitigation() float64 {
	// Gear-derived: sum of equipment slot PhysicalMitigation.
	gearMit := 0
	slots := []items.Item{
		c.Equipment.Weapon, c.Equipment.Offhand,
		c.Equipment.ExtraArm1, c.Equipment.ExtraArm2,
		c.Equipment.ExtraArm3, c.Equipment.ExtraArm4,
		c.Equipment.Head, c.Equipment.Neck, c.Equipment.Shoulders,
		c.Equipment.Body, c.Equipment.Back, c.Equipment.Belt,
		c.Equipment.Wrist1, c.Equipment.Wrist2,
		c.Equipment.ExtraWrist1, c.Equipment.ExtraWrist2,
		c.Equipment.ExtraWrist3, c.Equipment.ExtraWrist4,
		c.Equipment.Gloves, c.Equipment.Ring, c.Equipment.Ring2,
		c.Equipment.Legs, c.Equipment.Feet, c.Equipment.Tail,
		c.Equipment.ComponentBag,
	}
	for _, slot := range slots {
		if slot.ItemId <= 0 {
			continue
		}
		spec := slot.GetSpec()
		gearMit += spec.PhysicalMitigation
	}
	// Apply gear-effectiveness multiplier to the gear-derived portion.
	gearMit = int(float64(gearMit) * mutations.GearEffectivenessMultiplier(c.Mutations))

	// Non-gear additions (shield spell, mutations, species natural armor, buff
	// statmods). The buff/statmod contribution via c.StatMod("physical_mitigation")
	// mirrors GetMagicalMitigation / GetConvictionMitigation, which have always
	// folded their statmod sibling — physical was the odd one out, so buffs like
	// Cocoon (104) and Ironhide Brew (61) that reserve physical_mitigation as a
	// statmod silently did nothing until this line.
	nonGearMit := int(c.GetConditionMagnitude(ConditionShield))
	nonGearMit += mutations.GetNaturalArmor(c.Mutations)
	nonGearMit += c.StatMod("physical_mitigation")
	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {
		nonGearMit += speciesInfo.NaturalArmor
	}

	return float64(gearMit+nonGearMit) / 100.0
}

// GetMagicalMitigation returns total magical mitigation as a fraction (0.0–1.0).
// Sources: equipment magical_mitigation, mutation magical resistance, buff stat mods.
// Incorporeal mutation scales gear-derived mitigation.
func (c *Character) GetMagicalMitigation() float64 {
	// Gear-derived: sum of equipment slot MagicalMitigation.
	gearMit := 0
	slots := []items.Item{
		c.Equipment.Weapon, c.Equipment.Offhand,
		c.Equipment.ExtraArm1, c.Equipment.ExtraArm2,
		c.Equipment.ExtraArm3, c.Equipment.ExtraArm4,
		c.Equipment.Head, c.Equipment.Neck, c.Equipment.Shoulders,
		c.Equipment.Body, c.Equipment.Back, c.Equipment.Belt,
		c.Equipment.Wrist1, c.Equipment.Wrist2,
		c.Equipment.ExtraWrist1, c.Equipment.ExtraWrist2,
		c.Equipment.ExtraWrist3, c.Equipment.ExtraWrist4,
		c.Equipment.Gloves, c.Equipment.Ring, c.Equipment.Ring2,
		c.Equipment.Legs, c.Equipment.Feet, c.Equipment.Tail,
		c.Equipment.ComponentBag,
	}
	for _, slot := range slots {
		if slot.ItemId <= 0 {
			continue
		}
		spec := slot.GetSpec()
		gearMit += spec.MagicalMitigation
	}
	// Apply gear-effectiveness multiplier to the gear-derived portion.
	gearMit = int(float64(gearMit) * mutations.GearEffectivenessMultiplier(c.Mutations))

	// Non-gear additions (mutation resistance + buff stat mods via c.StatMod).
	// Note: c.StatMod("magical_mitigation") returns a mixed value where the
	// equipment-stat-mod portion is already scaled by Task 4's StatMod change;
	// buff portion is unscaled (correct — buffs aren't gear).
	nonGearMit := int(mutations.GetMagicalResistance(c.Mutations) * 100)
	nonGearMit += c.StatMod("magical_mitigation")

	return float64(gearMit+nonGearMit) / 100.0
}

// GetConvictionMitigation returns total conviction mitigation as a fraction (0.0–1.0).
// Sources: equipment conviction_mitigation, mutation conviction resistance, buff statmods.
// Incorporeal mutation scales gear-derived mitigation.
func (c *Character) GetConvictionMitigation() float64 {
	// Gear-derived: sum of equipment slot ConvictionMitigation.
	gearMit := 0
	slots := []items.Item{
		c.Equipment.Weapon, c.Equipment.Offhand,
		c.Equipment.ExtraArm1, c.Equipment.ExtraArm2,
		c.Equipment.ExtraArm3, c.Equipment.ExtraArm4,
		c.Equipment.Head, c.Equipment.Neck, c.Equipment.Shoulders,
		c.Equipment.Body, c.Equipment.Back, c.Equipment.Belt,
		c.Equipment.Wrist1, c.Equipment.Wrist2,
		c.Equipment.ExtraWrist1, c.Equipment.ExtraWrist2,
		c.Equipment.ExtraWrist3, c.Equipment.ExtraWrist4,
		c.Equipment.Gloves, c.Equipment.Ring, c.Equipment.Ring2,
		c.Equipment.Legs, c.Equipment.Feet, c.Equipment.Tail,
		c.Equipment.ComponentBag,
	}
	for _, slot := range slots {
		if slot.ItemId <= 0 {
			continue
		}
		spec := slot.GetSpec()
		gearMit += spec.ConvictionMitigation
	}
	// Apply gear-effectiveness multiplier to the gear-derived portion.
	gearMit = int(float64(gearMit) * mutations.GearEffectivenessMultiplier(c.Mutations))

	// Non-gear additions (mutation resistance + buff stat mods via c.StatMod).
	nonGearMit := int(mutations.GetConvictionResistance(c.Mutations) * 100)
	nonGearMit += c.StatMod("conviction_mitigation")

	return float64(gearMit+nonGearMit) / 100.0
}

// GetDefenseSequence returns ordered defenses based on equipment (Stage 7.1)
func (c *Character) GetDefenseSequence() []string {
	defenses := []string{}

	// Everyone can dodge
	defenses = append(defenses, DefenseDodge)

	// Unarmed-style fighters (bare hands, fists, claws) only dodge — no parry
	if c.IsUnarmedStyle() {
		return defenses
	}

	// If dual wielding (two weapons)
	if c.IsDualWielding() {
		// dodge → parry main → parry off
		defenses = append(defenses, DefenseParry) // main hand parry
		defenses = append(defenses, DefenseParry) // offhand parry
		return defenses
	}

	// If wielding weapon + shield
	if c.Equipment.Weapon.ItemId > 0 && c.HasShield() {
		// dodge → parry → block
		defenses = append(defenses, DefenseParry)
		defenses = append(defenses, DefenseBlock)
		return defenses
	}

	// If wielding single weapon (no offhand or offhand is not shield/weapon)
	if c.Equipment.Weapon.ItemId > 0 {
		// dodge → parry
		defenses = append(defenses, DefenseParry)
		return defenses
	}

	// Default: just dodge
	return defenses
}

// GetDefenseScore calculates defense score for a given defense type (Stage 7.1)
func (c *Character) GetDefenseScore(defenseType string) float64 {
	dex := float64(c.GetEffectiveDexterity())
	skillWeight := float64(configs.GetBalanceConfig().SkillWeight)

	switch defenseType {
	case DefenseDodge:
		// Dodge: Dexterity + UnarmedCombat skill + mutation dodge modifier
		unarmedSkill := float64(c.GetSkillLevel(skills.UnarmedCombat)) * skillWeight
		score := dex + unarmedSkill + mutations.GetDodgeModifier(c.Mutations)
		// Phase 24.5: Blinded condition reduces dodge
		if c.HasCondition(ConditionBlinded) {
			score *= c.GetConditionMagnitude(ConditionBlinded) // magnitude is 0.5–0.7 = penalty multiplier
		}
		return score

	case DefenseParry:
		// Parry: Dexterity + WeaponCombat skill + best weapon ParryRating
		weaponSkill := float64(c.GetSkillLevel(skills.WeaponCombat)) * skillWeight
		parryRating := c.BestParryRating()
		return dex + weaponSkill + float64(parryRating)

	case DefenseBlock:
		// Block: (Strength + Dexterity)/2 + WeaponCombat skill + best shield BlockRating
		str := float64(c.Stats.Strength.ValueAdj)
		weaponSkill := float64(c.GetSkillLevel(skills.WeaponCombat)) * skillWeight
		blockRating := c.BestBlockRating()
		return (str+dex)/2 + weaponSkill + float64(blockRating)

	default:
		return 0
	}
}
