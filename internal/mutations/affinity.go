package mutations

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// DecayAffinity multiplies every affinity by rate, pruning entries that
// fall below a negligible floor so the map does not grow unbounded.
func DecayAffinity(aff map[string]float64, rate float64) {
	for k, v := range aff {
		nv := v * rate
		if nv < 0.01 {
			delete(aff, k)
		} else {
			aff[k] = nv
		}
	}
}

// EffectiveAffinity combines the character's action-driven affinity with the
// gravity of their currently-owned mutations. Returns a fresh map.
func EffectiveAffinity(owned map[string]int, actionAff map[string]float64) map[string]float64 {
	eff := make(map[string]float64, len(actionAff)+4)
	for k, v := range actionAff {
		eff[k] = v
	}
	for k, v := range OwnedGravity(owned) {
		eff[k] += v
	}
	return eff
}

// depthThreshold is the affinity required to unlock a mutation of the given
// rarity. Rarer/deeper mutations demand more affinity.
//
// The curve is QUADRATIC in rarity so the acquisition journey spreads out:
// entry cluster mutations (low rarity) unlock once a playstyle is established,
// while apex + connective-tissue bridge mutations (high rarity) require
// sustained, dedicated drift — not one big fight. A linear curve compressed the
// whole range into a few fights (a rarity-9 mutation was only ~3x a rarity-3
// one), which let apex-class powers like Extra Arms emerge in a player's first
// fight. With MutationAffinityPerRarity=2: r3≈18, r6≈72, r8(bridge)≈128,
// r9(apex)≈162. PROVISIONAL — dial in during playtest.
func depthThreshold(rarity int) float64 {
	r := float64(rarity)
	return r * r * float64(configs.GetBalanceConfig().MutationAffinityPerRarity)
}

// affinityFor returns the best affinity across a mutation's clusters. A
// mutation with no clusters (universal/generalist enabler) is always eligible.
func affinityFor(spec *MutationSpec, aff map[string]float64) float64 {
	if len(spec.Clusters) == 0 {
		return math.MaxFloat64
	}
	best := 0.0
	for _, cl := range spec.Clusters {
		if aff[cl] > best {
			best = aff[cl]
		}
	}
	return best
}

// GetGraphPool builds a weighted acquisition pool from the mutation graph.
// A candidate is included only if: not already owned, not conflicting, its
// body-part requirements fit the species, its prerequisites are owned, AND
// its best cluster affinity clears its rarity-based depth threshold. Base
// weight follows rarity (commoner = heavier) minus the advanced-character
// rarity uplift (calcRarityBonus — preserves parity with GetWeightedPool so
// untagged legacy mutations still behave as before), plus the mutation's
// cluster affinity so a strongly-expressed cluster dominates the roll. aff
// must already fold in owned-gravity (see EffectiveAffinity). Pass nil sp to
// skip body filtering.
func GetGraphPool(owned map[string]int, aff map[string]float64, sp *species.Species) []string {
	rarityBonus := calcRarityBonus(owned)
	moonBucket := gametime.CurrentMoonFlavorBucket()
	pool := make([]string, 0, len(allMutations)*4)
	for id, spec := range allMutations {
		if _, has := owned[id]; has {
			continue
		}
		if HasConflict(owned, id) {
			continue
		}
		if !spec.CanApplyTo(sp) {
			continue
		}
		if !PrereqsMet(owned, spec) {
			continue
		}
		// Moon-gated flavor (Reflect Skin): only the variant matching the
		// current moon bucket is eligible now.
		if spec.MoonFlavor != 0 && spec.MoonFlavor != moonBucket {
			continue
		}
		a := affinityFor(spec, aff)
		if a < depthThreshold(spec.Rarity) {
			continue
		}
		weight := 11 - spec.Rarity - rarityBonus
		if weight < 1 {
			weight = 1
		}
		if a != math.MaxFloat64 {
			weight += int(a) // clustered mutations get louder as their cluster grows
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}
