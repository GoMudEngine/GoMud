package items

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BonusCategory controls which item types a bonus applies to.
type BonusCategory string

const (
	CategoryAny          BonusCategory = "any"
	CategoryArmor        BonusCategory = "armor"
	CategoryWeapon       BonusCategory = "weapon"       // non-caster weapons
	CategoryWeaponCaster BonusCategory = "weapon_caster" // wand/sceptre/staff
)

// BonusType describes a single affix option with its budget cost.
type BonusType struct {
	Name     string
	Cost     int
	Category BonusCategory
}

// allBonusTypes is the master table of every affix the engine can roll.
var allBonusTypes = []BonusType{
	// Weapon affixes
	{Name: "damage_mult_both", Cost: 12, Category: CategoryWeaponCaster},
	{Name: "damage_mult_phys", Cost: 8, Category: CategoryWeapon},
	// Armor affixes
	{Name: "physical_mitigation", Cost: 5, Category: CategoryArmor},
	{Name: "magical_mitigation", Cost: 5, Category: CategoryArmor},
	{Name: "conviction_mitigation", Cost: 5, Category: CategoryArmor},
	// Universal stat affixes
	{Name: "stat_strength", Cost: 3, Category: CategoryAny},
	{Name: "stat_dexterity", Cost: 3, Category: CategoryAny},
	{Name: "stat_perception", Cost: 3, Category: CategoryAny},
	{Name: "stat_vitality", Cost: 3, Category: CategoryAny},
	{Name: "stat_willpower", Cost: 3, Category: CategoryAny},
	{Name: "stat_charisma", Cost: 3, Category: CategoryAny},
	// Universal skill affixes
	{Name: "skill_weapon-combat", Cost: 12, Category: CategoryAny},
	{Name: "skill_unarmed-combat", Cost: 12, Category: CategoryAny},
	{Name: "skill_skullduggery", Cost: 12, Category: CategoryAny},
	{Name: "skill_spellcasting", Cost: 12, Category: CategoryAny},
	{Name: "skill_rhetoric", Cost: 12, Category: CategoryAny},
	{Name: "skill_manifestation", Cost: 12, Category: CategoryAny},
}

// CalcLootBudget converts gold paid into a flat bonus point budget.
// Returns floor(scalar * sqrt(goldPaid)), or 0 if goldPaid <= 0.
func CalcLootBudget(goldPaid int, scalar float64) int {
	if goldPaid <= 0 {
		return 0
	}
	return int(math.Floor(scalar * math.Sqrt(float64(goldPaid))))
}

// GetEligibleBonuses returns the bonus types that are valid for the given
// item classification. CategoryAny bonuses always appear.
func GetEligibleBonuses(isWeapon bool, isCasterWeapon bool) []BonusType {
	out := make([]BonusType, 0, len(allBonusTypes))
	for _, b := range allBonusTypes {
		switch b.Category {
		case CategoryAny:
			out = append(out, b)
		case CategoryWeaponCaster:
			if isWeapon && isCasterWeapon {
				out = append(out, b)
			}
		case CategoryWeapon:
			if isWeapon && !isCasterWeapon {
				out = append(out, b)
			}
		case CategoryArmor:
			if !isWeapon {
				out = append(out, b)
			}
		}
	}
	return out
}

// GenerateAffixedItem creates an item instance from baseItemId with random
// stat bonuses scaled by goldPaid * scalar.
//
// Budget is calculated via CalcLootBudget then given gaussian variance via
// dice.RollStat so the actual amount spent may be slightly above or below.
// Bonuses are drawn randomly from the eligible pool until the budget is
// exhausted.  Each draw of the same bonus type stacks by +1 on the same field.
func GenerateAffixedItem(baseItemId int, goldPaid int, scalar float64) Item {
	item := New(baseItemId)

	// Fetch and copy the base spec so that our Spec field is a full snapshot —
	// GetSpec() returns *i.Spec when non-nil, completely replacing the base.
	baseSpec := item.GetSpec()
	specCopy := baseSpec
	// Ensure the StatMods map is freshly allocated so we don't mutate shared data.
	if specCopy.StatMods != nil {
		newMods := make(map[string]int, len(specCopy.StatMods))
		for k, v := range specCopy.StatMods {
			newMods[k] = v
		}
		specCopy.StatMods = newMods
	} else {
		specCopy.StatMods = make(map[string]int)
	}

	// Calculate raw budget then roll gaussian variance around it.
	rawBudget := CalcLootBudget(goldPaid, scalar)
	if rawBudget <= 0 {
		// No budget, return unmodified item.
		return item
	}
	rolledBudget := int(math.Round(dice.RollStat(float64(rawBudget)).Value))
	if rolledBudget < 1 {
		rolledBudget = 1
	}

	// Determine item classification for bonus eligibility.
	isWeapon := baseSpec.Type == Weapon
	isCasterWeapon := isWeapon && (baseSpec.Subtype == Wand ||
		baseSpec.Subtype == Sceptre ||
		baseSpec.Subtype == Staff)

	eligible := GetEligibleBonuses(isWeapon, isCasterWeapon)
	if len(eligible) == 0 {
		// No eligible bonuses (e.g. a potion) — return plain item.
		return item
	}

	// Spend the budget one purchase at a time.
	budget := rolledBudget
	for budget > 0 {
		// Collect candidates whose cost fits within remaining budget.
		candidates := make([]BonusType, 0, len(eligible))
		for _, b := range eligible {
			if b.Cost <= budget {
				candidates = append(candidates, b)
			}
		}
		if len(candidates) == 0 {
			break
		}

		// Pick a random candidate.
		chosen := candidates[util.Rand(len(candidates))]
		budget -= chosen.Cost

		// Apply the chosen bonus to the spec copy.
		applyBonus(&specCopy, chosen.Name)
	}

	item.Spec = &specCopy
	return item
}

// applyBonus mutates spec by applying one rank of the named bonus.
func applyBonus(spec *ItemSpec, bonusName string) {
	switch bonusName {
	case "damage_mult_phys":
		// Add a fixed increment; 0.05 per rank keeps numbers meaningful.
		spec.DamageMultiplier += 0.05
	case "damage_mult_both":
		spec.DamageMultiplier += 0.05
		spec.SpellDamageMultiplier += 0.05
	case "physical_mitigation":
		spec.PhysicalMitigation++
	case "magical_mitigation":
		spec.MagicalMitigation++
	case "conviction_mitigation":
		spec.ConvictionMitigation++
	// Stats — prefix "stat_" strips to the statmods key.
	case "stat_strength":
		spec.StatMods.Add("strength", 1)
	case "stat_dexterity":
		spec.StatMods.Add("dexterity", 1)
	case "stat_perception":
		spec.StatMods.Add("perception", 1)
	case "stat_vitality":
		spec.StatMods.Add("vitality", 1)
	case "stat_willpower":
		spec.StatMods.Add("willpower", 1)
	case "stat_charisma":
		spec.StatMods.Add("charisma", 1)
	// Skills — prefix "skill_" strips to the statmods key.
	case "skill_weapon-combat":
		spec.StatMods.Add("weapon-combat", 1)
	case "skill_unarmed-combat":
		spec.StatMods.Add("unarmed-combat", 1)
	case "skill_skullduggery":
		spec.StatMods.Add("skullduggery", 1)
	case "skill_spellcasting":
		spec.StatMods.Add("spellcasting", 1)
	case "skill_rhetoric":
		spec.StatMods.Add("rhetoric", 1)
	case "skill_manifestation":
		spec.StatMods.Add("manifestation", 1)
	}
}
