package mutations

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
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
// rarity. Rarer/deeper keystones demand more affinity.
func depthThreshold(rarity int) float64 {
	return float64(rarity) * float64(configs.GetBalanceConfig().MutationAffinityPerRarity)
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
// its best cluster affinity clears its rarity-based depth threshold. Weight
// scales with rarity (commoner = heavier) plus the surplus affinity, so a
// strongly-expressed cluster dominates the roll. aff must already fold in
// owned-gravity (see EffectiveAffinity). Pass nil sp to skip body filtering.
func GetGraphPool(owned map[string]int, aff map[string]float64, sp *species.Species) []string {
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
		a := affinityFor(spec, aff)
		if a < depthThreshold(spec.Rarity) {
			continue
		}
		weight := 11 - spec.Rarity
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
