package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// DamageChannel represents the type of damage being dealt.
type DamageChannel int

const (
	ChannelPhysical   DamageChannel = iota // Melee/ranged — scales with Strength
	ChannelMagical                         // Spells — scales with Willpower
	ChannelConviction                      // Social combat — scales with Charisma
)

// SkillMultiplier returns the damage multiplier for a given skill rank.
// Uses a sqrt curve: mult = base + (max - base) * sqrt(rank / softCap)
// Rank 0 → base (1.0), Rank 50 → max (3.0)
func SkillMultiplier(rank int) float64 {
	bal := configs.GetBalanceConfig()
	base := float64(bal.SkillMultiplierBase)
	max := float64(bal.SkillMultiplierMax)
	softCap := float64(bal.SkillSoftCap)

	if softCap <= 0 {
		softCap = 50.0
	}
	if rank <= 0 {
		return base
	}

	r := float64(rank)
	if r > softCap {
		r = softCap
	}

	return base + (max-base)*math.Sqrt(r/softCap)
}

// DamageScale returns the configured damage scale for a given channel.
// All channels use the same formula: stat × SkillMult × itemMult × scale.
// Physical default is 0.30 (stats ~100, so 100×1.0×1.0×0.30 = 30 raw).
// Magical / Conviction defaults are 1.0 (flat multipliers).
func DamageScale(channel DamageChannel) float64 {
	bal := configs.GetBalanceConfig()
	switch channel {
	case ChannelPhysical:
		return float64(bal.MeleeDamageScale)
	case ChannelMagical:
		return float64(bal.SpellDamageScale)
	case ChannelConviction:
		return float64(bal.RhetoricDamageScale)
	default:
		return 1.0
	}
}

// CalcRawDamage computes raw damage before mitigation and variance.
// All channels use the same unified formula:
//
//	raw = stat × SkillMultiplier(rank) × itemMult × ChannelScale
//
// The per-channel scale absorbs any normalization:
//   - Physical: 0.30 (stats ~100, so 100×1.0×1.0×0.30 = 30 raw per swing)
//   - Magical:  1.00 (100×1.0×1.0×1.00 = 100 raw)
//   - Conviction: 1.00 (100×1.0×0.5×1.00 = 50 raw for taunt)
func CalcRawDamage(stat int, skillRank int, itemMult float64, channel DamageChannel) float64 {
	if itemMult <= 0 {
		itemMult = 0.30 // fallback to unarmed-level multiplier
	}
	scale := DamageScale(channel)
	statFactor := float64(stat)

	globalMult := float64(configs.GetBalanceConfig().GlobalDamageMultiplier)
	return statFactor * SkillMultiplier(skillRank) * itemMult * scale * globalMult
}

// ResourceMultiplier returns a smooth penalty multiplier based on how depleted
// a resource pool is. Returns 1.0 at full, decreasing to (1 - penaltyMax) at 0%.
// Uses a quadratic curve (configurable exponent) so penalties are barely
// noticeable above 50% but ramp up as the pool empties.
func ResourceMultiplier(current, max int, penaltyMax float64) float64 {
	if max <= 0 {
		return 1.0
	}
	ratio := float64(current) / float64(max)
	if ratio >= 1.0 {
		return 1.0
	}
	if ratio < 0.0 {
		ratio = 0.0
	}
	curve := float64(configs.GetBalanceConfig().ResourcePenaltyCurve)
	return 1.0 - penaltyMax*math.Pow(1.0-ratio, curve)
}

// ApplyMitigation reduces raw damage by a mitigation percentage, capped.
// final = raw × (1 - min(mitigationPct, cap))
// mitigationPct and cap are fractions (0.0–1.0).
func ApplyMitigation(rawDmg float64, mitigationPct float64, cap float64) float64 {
	if mitigationPct < 0 {
		mitigationPct = 0
	}
	if cap <= 0 {
		cap = 0.75
	}
	if mitigationPct > cap {
		mitigationPct = cap
	}
	return rawDmg * (1.0 - mitigationPct)
}

// MitigationCap returns the configured cap for a given damage channel.
func MitigationCap(channel DamageChannel) float64 {
	bal := configs.GetBalanceConfig()
	switch channel {
	case ChannelPhysical:
		return float64(bal.PhysicalMitigationCap)
	case ChannelMagical:
		return float64(bal.MagicalMitigationCap)
	case ChannelConviction:
		return float64(bal.ConvictionMitigationCap)
	default:
		return 0.75
	}
}

// GetConvictionDamageDescription converts conviction damage to descriptive text.
func GetConvictionDamageDescription(damageAmount int, targetMaxConviction int) string {
	if targetMaxConviction <= 0 {
		return "a mild rebuke"
	}

	pct := float64(damageAmount) / float64(targetMaxConviction) * 100

	switch {
	case pct < 5:
		return "a feeble jab at their resolve"
	case pct < 15:
		return "a stinging insult"
	case pct < 30:
		return "a rattling verbal assault"
	case pct < 50:
		return "a crushing blow to their confidence"
	case pct < 75:
		return "a devastating attack on their will"
	default:
		return "a soul-shattering tirade"
	}
}
