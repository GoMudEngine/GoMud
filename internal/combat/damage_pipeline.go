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

// CalcRawDamage computes raw damage before mitigation and variance.
// raw = stat × SkillMultiplier(rank) × itemMult
func CalcRawDamage(stat int, skillRank int, itemMult float64) float64 {
	if itemMult <= 0 {
		itemMult = 0.30 // fallback to unarmed-level multiplier
	}
	return float64(stat) * SkillMultiplier(skillRank) * itemMult
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
