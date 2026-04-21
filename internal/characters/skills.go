package characters

import (
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"maps"
)

// GetTotalSkillRanks returns the sum of all skill ranks.
func (c *Character) GetTotalSkillRanks() int {
	total := 0
	for _, rank := range c.Skills {
		total += rank
	}
	return total
}

func (c *Character) KnowsFirstAid() bool {
	if r := species.GetSpecies(c.SpeciesId); r != nil {
		return r.KnowsFirstAid
	}
	return false
}

func (c *Character) GetAllSkillRanks() map[string]int {
	retMap := make(map[string]int)
	maps.Copy(retMap, c.Skills)
	return retMap
}

// AttemptRecovery attempts to recover from Prone status.
// Returns (attemptMade, success) — attemptMade=true only if minimum duration
// has passed and a roll was made. success indicates whether the recovery attempt
// succeeded (only meaningful if attemptMade is true).
func (c *Character) AttemptRecovery(statValue int) (bool, bool) {
	// Currently only handles Prone, but future-proofed for grapple/entangle/etc
	if c.CombatPosition != PositionProne {
		return false, false // No condition to recover from
	}

	// Decrement minimum prone duration counter
	if c.PositionRoundsMin > 0 {
		c.PositionRoundsMin--
		// Still in minimum prone period, can't attempt recovery yet
		// Reduce attacks to 1 this round (struggling to stand)
		c.AddCondition(ConditionRecoveryPenalty, 1, 1.0, "prone recovery")
		return false, false // No recovery attempt yet (still in minimum duration)
	}

	// Minimum duration passed, now roll for recovery based on stat
	// Calculate recovery chance using logarithmic formula
	// DEX 25 = 25%, DEX 100 = 50%, DEX 300 = 75%, caps at 90%
	chance := 25.0
	if statValue > 0 {
		chance = 25.0 + 20.0*math.Log(float64(statValue)/25.0)
		if chance > 90.0 {
			chance = 90.0
		}
		if chance < 0 {
			chance = 0
		}
	}

	// Roll for success
	roll := dice.RollStat(50) // Mean of 50
	success := roll.Value < chance

	if success {
		c.CombatPosition = PositionStanding
		c.PositionRoundsMin = 0
	} else {
		// Failed recovery attempt - reduce attacks to 1 this round
		c.AddCondition(ConditionRecoveryPenalty, 1, 1.0, "prone recovery")
	}

	return true, success
}

func (c *Character) GetSkills() map[string]int {
	skillResults := make(map[string]int)
	for skillName, skillLevel := range c.Skills {
		skillResults[skillName] = skillLevel
	}
	return skillResults
}

func (c *Character) SetSkill(skillName string, level int) {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}
	skillName = strings.ToLower(skillName)

	if level == 0 {
		delete(c.Skills, skillName)
		return
	}

	c.Skills[skillName] = level
}

// Increases the skill training counter and returns the new value
func (c *Character) TrainSkill(skillName string, targetLevel ...int) int {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}

	skillName = strings.ToLower(skillName)

	skillLevel := 0

	if lvl, ok := c.Skills[skillName]; ok {
		skillLevel = lvl
	}

	if len(targetLevel) > 0 {

		if skillLevel < targetLevel[0] {
			skillLevel = targetLevel[0]
		}

	} else {

		skillLevel++

	}

	c.Skills[skillName] = skillLevel

	return skillLevel
}

// Gets the current value of the skillname provided
func (c *Character) GetSkillLevel(skillName skills.SkillTag) int {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}

	if level, ok := c.Skills[string(skillName)]; ok {
		return level
	}
	return 0
}

func (c *Character) GetSkillLevelCost(currentLevel int) int {
	return currentLevel
}

// IncreaseSkill increments the named skill by 1.
// No hard cap — progression is governed by the soft cap in CheckSkillProgression.
// Returns true only when the visible rank description actually changes (e.g.
// novice → apprentice), so callers can show "Your X skill reaches Y!" only on
// a genuine tier-up rather than every internal counter increment.
func (c *Character) IncreaseSkill(skillName string) bool {
	if c.Skills == nil {
		c.Skills = make(map[string]int)
	}
	oldLevel := c.Skills[skillName]
	c.Skills[skillName] = oldLevel + 1
	newLevel := c.Skills[skillName]
	return skills.GetSkillRankDescription(newLevel) != skills.GetSkillRankDescription(oldLevel)
}

// IncreaseStat increments the Training field of the named stat by the given amount,
// then recalculates derived values via Validate.
func (c *Character) IncreaseStat(statName string, amount int) bool {
	switch statName {
	case "strength":
		c.Stats.Strength.Training += amount
	case "dexterity":
		c.Stats.Dexterity.Training += amount
	case "perception":
		c.Stats.Perception.Training += amount
	case "vitality":
		c.Stats.Vitality.Training += amount
	case "willpower":
		c.Stats.Willpower.Training += amount
	case "charisma":
		c.Stats.Charisma.Training += amount
	default:
		return false
	}
	c.Validate()
	return true
}

// GetStatValue returns the raw computed Value for the named stat, or 0 if unrecognised.
func (c *Character) GetStatValue(statName string) int {
	switch statName {
	case "strength":
		return c.Stats.Strength.Value
	case "dexterity":
		return c.Stats.Dexterity.Value
	case "perception":
		return c.Stats.Perception.Value
	case "vitality":
		return c.Stats.Vitality.Value
	case "willpower":
		return c.Stats.Willpower.Value
	case "charisma":
		return c.Stats.Charisma.Value
	}
	return 0
}

// GetCombatSkillTag returns the appropriate combat skill tag based on
// the character's equipped weapon type.
func (c *Character) GetCombatSkillTag() skills.SkillTag {
	if c.Equipment.Weapon.ItemId > 0 {
		return CombatSkillTagForItem(c.Equipment.Weapon)
	}
	return skills.UnarmedCombat
}

// CombatSkillTagForItem returns the combat skill tag for a specific weapon item.
func CombatSkillTagForItem(weapon items.Item) skills.SkillTag {
	if weapon.ItemId == 0 {
		return skills.UnarmedCombat
	}
	spec := weapon.GetSpec()
	if spec.Subtype == items.Claws || spec.Subtype == items.Fist {
		return skills.UnarmedCombat
	}
	return skills.WeaponCombat
}

// GetCombatSkillLevel returns an effective combat skill value for use in
// combat formulas. Checks the weapon-appropriate DOG skill first, then
// falls back to legacy Brawling, then minimum 1.
func (c *Character) GetCombatSkillLevel() int {
	if level := c.GetSkillLevel(c.GetCombatSkillTag()); level > 0 {
		return level
	}
	return 1
}

// GetModifiedAttackCount calculates the number of attacks for a weapon
// considering speed multiplier, skill, and dual wielding.
// baseAttacks: The weapon's base attack count
// weaponSpeed: The weapon's speed multiplier (1.0 = unarmed baseline)
// isOffhand: Whether this is the offhand weapon
func (c *Character) GetModifiedAttackCount(baseAttacks int, weaponSpeed float64, isOffhand bool) int {
	attacks := float64(baseAttacks)

	// Apply weapon speed multiplier
	attacks *= weaponSpeed

	// Apply skill modifier (small bonus, max ~10% at skill 50)
	skillLevel := float64(c.GetCombatSkillLevel())
	skillMod := 1.0 + (skillLevel / 50.0) * 0.1
	attacks *= skillMod

	// If offhand, weapon-combat skill governs dual-wield effectiveness
	if isOffhand {
		wcLevel := float64(c.GetSkillLevel(skills.WeaponCombat))
		// Significant modifier: 0.5 at skill 0, 1.0 at skill 25, 1.2 at skill 50
		dualWieldMod := 0.5 + (wcLevel / 50.0) * 0.7
		attacks *= dualWieldMod
	}

	// Minimum 1 attack
	result := int(math.Round(attacks))
	if result < 1 {
		result = 1
	}

	return result
}
