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
func GenerateAffixedItem(baseItemId int, goldPaid int, scalar float64, goldPerPoint float64) Item {
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

	// Build weighted selection table. Base weights favor the item's
	// primary role: weapons prefer damage, armor prefers mitigation.
	// The snowball mechanic increases a bonus's weight each time it's
	// picked, creating focused items instead of thin spreads.
	weights := make(map[string]int, len(eligible))
	for _, b := range eligible {
		switch b.Category {
		case CategoryWeapon, CategoryWeaponCaster:
			weights[b.Name] = 5 // damage mult heavily favored on weapons
		case CategoryArmor:
			weights[b.Name] = 5 // mitigations heavily favored on armor
		case CategoryAny:
			if len(b.Name) > 6 && b.Name[:6] == "skill_" {
				weights[b.Name] = 1 // skills rare
			} else {
				weights[b.Name] = 2 // stats moderate
			}
		}
	}

	// Spend the budget one purchase at a time with weighted selection.
	budget := rolledBudget
	for budget > 0 {
		// Collect affordable candidates and their weights.
		type weightedCandidate struct {
			bonus  BonusType
			weight int
		}
		candidates := make([]weightedCandidate, 0, len(eligible))
		totalWeight := 0
		for _, b := range eligible {
			if b.Cost <= budget {
				w := weights[b.Name]
				if w < 1 {
					w = 1
				}
				candidates = append(candidates, weightedCandidate{b, w})
				totalWeight += w
			}
		}
		if len(candidates) == 0 || totalWeight == 0 {
			break
		}

		// Weighted random selection.
		roll := util.Rand(totalWeight)
		var chosen BonusType
		for _, wc := range candidates {
			roll -= wc.weight
			if roll < 0 {
				chosen = wc.bonus
				break
			}
		}

		budget -= chosen.Cost
		applyBonus(&specCopy, chosen.Name)

		// Snowball: increase weight of chosen bonus for next iteration.
		weights[chosen.Name]++
	}

	// Value scales with the affix power added (Stage 1: shops-trade-affixed-gear).
	specCopy.Value = AffixValue(specCopy, baseSpec, goldPerPoint)

	item.Spec = &specCopy
	item.Affixed = true

	// Set display prefix based on dominant bonus type.
	prefix := getAffixPrefix(&specCopy, &baseSpec)
	item.Adjectives = append(item.Adjectives, prefix)

	return item
}

// AffixValue computes the gold value of an affixed item instance: the base
// template value plus a premium for each affix cost-point of power added above
// the base, priced at goldPerPoint gold per point. affixedSpec is the item's
// per-instance spec; baseSpec is the unmodified template. Only positive deltas
// count (affixes never subtract). See the design spec's Calibration section.
func AffixValue(affixedSpec, baseSpec ItemSpec, goldPerPoint float64) int {
	pts := affixPoints(affixedSpec, baseSpec)
	return baseSpec.Value + int(math.Round(float64(pts)*goldPerPoint))
}

// affixPoints reconstructs the affix cost-point budget spent on affixedSpec vs
// baseSpec, using the same per-affix cost weights as allBonusTypes:
// damage_mult_phys 8 (+0.05/rank), the spell half of damage_mult_both 4,
// mitigations 5, stats 3, skills 12.
func affixPoints(affixedSpec, baseSpec ItemSpec) int {
	pts := 0

	if d := affixedSpec.DamageMultiplier - baseSpec.DamageMultiplier; d > 0 {
		pts += int(math.Round(d/0.05)) * 8
	}
	if d := affixedSpec.SpellDamageMultiplier - baseSpec.SpellDamageMultiplier; d > 0 {
		pts += int(math.Round(d/0.05)) * 4
	}
	if d := affixedSpec.PhysicalMitigation - baseSpec.PhysicalMitigation; d > 0 {
		pts += d * 5
	}
	if d := affixedSpec.MagicalMitigation - baseSpec.MagicalMitigation; d > 0 {
		pts += d * 5
	}
	if d := affixedSpec.ConvictionMitigation - baseSpec.ConvictionMitigation; d > 0 {
		pts += d * 5
	}
	for k, v := range affixedSpec.StatMods {
		base := 0
		if baseSpec.StatMods != nil {
			base = baseSpec.StatMods[k]
		}
		if d := v - base; d > 0 {
			if isSkillMod(k) {
				pts += d * 12
			} else {
				pts += d * 3
			}
		}
	}
	return pts
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

// isSkillMod returns true if the stat mod name is a skill (not a stat).
func isSkillMod(name string) bool {
	skills := []string{
		"weapon-combat", "unarmed-combat", "skullduggery",
		"spellcasting", "rhetoric", "manifestation",
	}
	for _, s := range skills {
		if s == name {
			return true
		}
	}
	return false
}

// getAffixPrefix returns a descriptive prefix based on what type of
// bonus dominated the point spend.
func getAffixPrefix(spec *ItemSpec, baseSpec *ItemSpec) string {
	dmgScore := 0
	mitScore := 0
	statScore := 0
	skillScore := 0

	// Check damage mult increase
	if spec.DamageMultiplier > baseSpec.DamageMultiplier {
		dmgScore += 10
	}
	if spec.SpellDamageMultiplier > baseSpec.SpellDamageMultiplier {
		dmgScore += 10
	}

	// Check mitigation increases
	if spec.PhysicalMitigation > baseSpec.PhysicalMitigation {
		mitScore += spec.PhysicalMitigation - baseSpec.PhysicalMitigation
	}
	if spec.MagicalMitigation > baseSpec.MagicalMitigation {
		mitScore += spec.MagicalMitigation - baseSpec.MagicalMitigation
	}
	if spec.ConvictionMitigation > baseSpec.ConvictionMitigation {
		mitScore += spec.ConvictionMitigation - baseSpec.ConvictionMitigation
	}

	// Check stat/skill mods
	for k, v := range spec.StatMods {
		baseVal := 0
		if baseSpec.StatMods != nil {
			baseVal = baseSpec.StatMods[k]
		}
		added := v - baseVal
		if added > 0 {
			// Skills cost 12 pts, stats cost 3 — weight accordingly
			if isSkillMod(k) {
				skillScore += added * 4
			} else {
				statScore += added
			}
		}
	}

	// Pick dominant category
	best := dmgScore
	prefix := "Keen"
	if mitScore > best {
		best = mitScore
		prefix = "Warding"
	}
	if statScore > best {
		best = statScore
		prefix = "Empowered"
	}
	if skillScore > best {
		prefix = "Masterwork"
	}

	return prefix
}
