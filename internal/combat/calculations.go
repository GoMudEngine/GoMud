package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PowerScore computes an absolute power score for a character.
// Uses the real damage pipeline so it automatically tracks config tuning.
// Components: Offense, Defense, Durability, Skills, Mutations, KD Ratio.
func PowerScore(char characters.Character) float64 {
	bal := configs.GetBalanceConfig()
	softCap := float64(bal.SkillSoftCap)
	if softCap <= 0 {
		softCap = 50
	}

	// ── OFFENSE ──────────────────────────────────────────────────────────
	physAtk := 0.0

	addWeaponDPS := func(weapon items.Item, isOffhand bool) {
		spec := weapon.GetSpec()
		damageMult := spec.DamageMultiplier
		if damageMult <= 0 {
			damageMult = 0.30
		}
		weaponSpeed := spec.SpeedMultiplier
		if weaponSpeed <= 0 {
			weaponSpeed = 1.0
		}

		dex := float64(char.Stats.Dexterity.ValueAdj)
		combatSkill := float64(char.GetCombatSkillLevel()) * float64(bal.SkillWeight)
		swings := 1.0 + (dex-50.0)/100.0*weaponSpeed*(1.0+combatSkill/softCap)
		if isOffhand {
			wcLevel := float64(char.GetSkillLevel(skills.WeaponCombat)) * float64(bal.SkillWeight)
			dualWieldMod := 0.5 + (wcLevel/50.0)*0.5
			swings *= dualWieldMod
		}
		if swings < 1 {
			swings = 1
		}
		if swings > 4 {
			swings = 4
		}

		rawDmg := CalcRawDamage(char.Stats.Strength.ValueAdj, char.GetCombatSkillLevel(), damageMult, ChannelPhysical)
		physAtk += swings * rawDmg
	}

	if char.Equipment.Weapon.ItemId > 0 {
		addWeaponDPS(char.Equipment.Weapon, false)
	} else {
		addWeaponDPS(items.Item{}, false)
	}
	if char.Equipment.Offhand.ItemId > 0 {
		addWeaponDPS(char.Equipment.Offhand, true)
	}
	if char.ExtraArms >= 1 && char.Equipment.ExtraArm1.ItemId > 0 {
		addWeaponDPS(char.Equipment.ExtraArm1, true)
	}
	if char.ExtraArms >= 2 && char.Equipment.ExtraArm2.ItemId > 0 {
		addWeaponDPS(char.Equipment.ExtraArm2, true)
	}
	if char.ExtraArms >= 3 && char.Equipment.ExtraArm3.ItemId > 0 {
		addWeaponDPS(char.Equipment.ExtraArm3, true)
	}
	if char.ExtraArms >= 4 && char.Equipment.ExtraArm4.ItemId > 0 {
		addWeaponDPS(char.Equipment.ExtraArm4, true)
	}
	physAtk *= (1.0 + mutations.GetDamageMultiplier(char.Mutations))

	// Normalize all three channels to comparable scale
	spellcastingRank := char.GetSkillLevel(skills.Spellcasting)
	spellMult := 1.0
	if char.Equipment.Weapon.ItemId > 0 {
		sdm := char.Equipment.Weapon.GetSpec().SpellDamageMultiplier
		if sdm > 0 {
			spellMult = sdm * mutations.GearEffectivenessMultiplier(char.Mutations)
		}
	}
	magAtk := CalcRawDamage(char.Stats.Willpower.ValueAdj, spellcastingRank, spellMult, ChannelMagical)

	rhetoricRank := char.GetSkillLevel(skills.Rhetoric)
	convAtk := CalcRawDamage(char.Stats.Charisma.ValueAdj, rhetoricRank, 0.5, ChannelConviction)

	// Normalize: physical already includes swing count so it's higher per-round.
	// Magical/conviction are single-cast values. Weight them equally.
	offenseScore := physAtk + magAtk*0.5 + convAtk*0.5

	// ── DEFENSE ──────────────────────────────────────────────────────────
	avgMit := (char.GetPhysicalMitigation() + char.GetMagicalMitigation() + char.GetConvictionMitigation()) / 3.0
	dodgeScore := char.GetDefenseScore(characters.DefenseDodge)
	parryScore := char.GetDefenseScore(characters.DefenseParry)
	blockScore := char.GetDefenseScore(characters.DefenseBlock)
	defenseAvoidance := (dodgeScore + parryScore + blockScore) / 3.0
	naturalArmor := mutations.GetNaturalArmor(char.Mutations)
	defenseScore := (avgMit * 300) + (defenseAvoidance * 2) + float64(naturalArmor)

	// ── DURABILITY ───────────────────────────────────────────────────────
	durabilityScore := float64(char.HealthMax.Value) +
		float64(char.StaminaMax.Value)*0.5 +
		float64(char.ConvictionMax.Value)*0.5

	// ── SKILLS ───────────────────────────────────────────────────────────
	totalRanks := 0
	for _, rank := range char.GetAllSkillRanks() {
		totalRanks += rank
	}
	skillScore := math.Sqrt(float64(totalRanks)) * 25.0

	// ── MUTATIONS ────────────────────────────────────────────────────────
	totalMutLevels := 0
	for _, level := range char.Mutations {
		if level > 0 {
			totalMutLevels += level
		}
	}
	mutationScore := float64(totalMutLevels) * 20.0

	// ── KD RATIO ─────────────────────────────────────────────────────────
	deaths := char.KD.TotalDeaths
	if deaths < 1 {
		deaths = 1
	}
	kdRatio := float64(char.KD.TotalKills) / float64(deaths)
	kdScore := math.Min(kdRatio*10.0, 50.0)

	return offenseScore + defenseScore + durabilityScore + skillScore + mutationScore + kdScore
}

func ChanceToTame(s *users.UserRecord, t *mobs.Mob) int {

	var MOD_SKILL_MIN int = 1   // Minimum base tame ability
	var MOD_SKILL_MAX int = 100 // Maximum base tame ability

	var MOD_SIZE_SMALL int = 0    // Modifier for small creatures
	var MOD_SIZE_MEDIUM int = -10 // Modifier for medium creatures
	var MOD_SIZE_LARGE int = -25  // Modifier for large creatures

	var MOD_SKILLDIFF_MIN int = -4 // Lowest skill delta modifier
	var MOD_SKILLDIFF_MAX int = 4  // Highest skill delta modifier
	var SKILL_SCALE int = 6        // Scale factor to preserve impact range (4*6=24, similar to old ±25)

	var MOD_HEALTHPERCENT_MAX float64 = 50 // Highest possible bonus for target HP being reduced

	var FACTOR_IS_AGGRO float64 = .50 // Overall reduction of chance if target is aggro

	proficiencyModifier := s.Character.MobMastery.GetTame(int(t.MobId))

	if proficiencyModifier < MOD_SKILL_MIN {
		proficiencyModifier = MOD_SKILL_MIN
	} else if proficiencyModifier > MOD_SKILL_MAX {
		proficiencyModifier = MOD_SKILL_MAX
	}

	speciesInfo := species.GetSpecies(s.Character.SpeciesId)

	sizeModifier := 0
	switch speciesInfo.Size {
	case species.Large:
		sizeModifier = MOD_SIZE_LARGE
	case species.Small:
		sizeModifier = MOD_SIZE_SMALL
	case species.Medium:
	default:
		sizeModifier = MOD_SIZE_MEDIUM
	}

	// Taming is now handled via spellcasting; use spellcasting skill vs mob combat skill
	tamerSkill := s.Character.GetSkillLevel(skills.Spellcasting)
	mobSkill := t.Character.GetCombatSkillLevel()
	skillDiff := tamerSkill - mobSkill
	if skillDiff > MOD_SKILLDIFF_MAX {
		skillDiff = MOD_SKILLDIFF_MAX
	} else if skillDiff < MOD_SKILLDIFF_MIN {
		skillDiff = MOD_SKILLDIFF_MIN
	}
	scaledSkillDiff := skillDiff * SKILL_SCALE

	healthModifier := MOD_HEALTHPERCENT_MAX - math.Ceil(float64(s.Character.Health)/float64(s.Character.HealthMax.Value)*MOD_HEALTHPERCENT_MAX)

	var aggroModifier float64 = 1
	if t.Character.IsAggro(s.UserId, 0) {
		aggroModifier = FACTOR_IS_AGGRO
	}

	return int(math.Ceil((float64(proficiencyModifier) + float64(scaledSkillDiff) + healthModifier + float64(sizeModifier)) * aggroModifier))
}

// ChanceToSwitchTarget calculates the percentage chance (0-100) that a character
// can successfully switch targets mid-combat.
// Success is based on:
// - Combat skill (60% weight) - Higher skill = smoother transitions
// - Dexterity (40% weight) - Agility helps reposition quickly
//
// Base chance: 50%
// Skill bonus: +0.3% per combat skill level (max +30% at skill 100)
// Dexterity bonus: +0.2% per dexterity point (max +20% at dex 100)
//
// Returns: chance out of 100 (e.g., 75 = 75% chance)
func ChanceToSwitchTarget(c *characters.Character) int {
	const baseChance float64 = 50.0
	const skillWeight float64 = 0.3 // 0.3% per skill level
	const dexWeight float64 = 0.2   // 0.2% per dex point

	combatSkill := c.GetCombatSkillLevel()
	dexterity := c.Stats.Dexterity.ValueAdj

	skillBonus := float64(combatSkill) * skillWeight
	dexBonus := float64(dexterity) * dexWeight

	totalChance := baseChance + skillBonus + dexBonus

	// Cap at 95% (always a small chance of failure)
	if totalChance > 95.0 {
		totalChance = 95.0
	}

	// Floor at 25% (even unskilled fighters have some chance)
	if totalChance < 25.0 {
		totalChance = 25.0
	}

	return int(totalChance)
}
