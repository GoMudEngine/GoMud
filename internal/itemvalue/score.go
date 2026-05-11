package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ItemValue returns the raw value of an item under the given
// profile. Positive AND negative stat mods contribute. Used to
// rank items independently of any mob's current loadout.
func ItemValue(spec items.ItemSpec, profile WeightProfile) float64 {
	score := 0.0

	// (1) Stat mods (positive AND negative contributions).
	for statName, mod := range spec.StatMods {
		weight, ok := profile.StatWeights[statName]
		if !ok {
			weight = 1.0
		}
		score += float64(mod) * weight
	}

	// (2) Physical damage.
	if spec.DamageMultiplier > 0 {
		score += spec.DamageMultiplier * 100 * profile.PhysicalDamageWeight
	}

	// (3) Spell damage (caster weapons).
	if spec.SpellDamageMultiplier > 0 {
		score += spec.SpellDamageMultiplier * 100 * profile.SpellDamageWeight
	}

	// (4) Mitigation channels (1 point per percentage point).
	score += float64(spec.PhysicalMitigation) * profile.PhysicalMitigationWeight
	score += float64(spec.MagicalMitigation) * profile.MagicalMitigationWeight
	score += float64(spec.ConvictionMitigation) * profile.ConvictionMitigationWeight

	// (5) Weight cost (profile-modulated).
	score -= spec.Weight * profile.WeightPenaltyPerLb

	return score
}

// IsUpgrade is sugar over ItemValueDelta(...).Score > 0. Used
// by callers that just want the boolean "should I equip this"
// answer without the SwapDelta detail.
func IsUpgrade(char *characters.Character, profile WeightProfile, candidate items.Item) bool {
	return ItemValueDelta(char, profile, candidate).Score > 0
}
