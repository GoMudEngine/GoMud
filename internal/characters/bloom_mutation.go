package characters

// bloom_mutation.go — Bloom drug mutation-acceleration effect.
// Called by the consume hook when a character drinks Bloom.

import (
	"math/rand"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// BloomAdvanceMutation accelerates the character's mutation progression
// (the Bloom drug's signature effect). It advances the character's strongest
// existing mutation one level (cap-aware), or seeds a new mid/high-tier
// mutation if the character has none — or if all owned mutations are already
// at the global maximum level.
//
// "Strongest" means highest current level; ties are broken alphabetically by
// mutation ID for determinism. The global cap is Balance.MutationMaxLevel.
//
// When seeding a new mutation, candidates are weighted by Rarity (higher
// rarity = higher weight), giving Bloom its signature bias toward mid/high-
// tier mutations rather than the common-weighted pool of natural discovery.
// Conflict and species body-part requirements are respected.
//
// Returns the mutationId advanced/granted and its new level, or ("", 0) if
// nothing could change (all owned are at cap and no new candidate exists).
//
// rng is injectable for deterministic testing; pass nil to use the default
// global math/rand source.
func (c *Character) BloomAdvanceMutation(rng *rand.Rand) (string, int) {
	maxLevel := int(configs.GetBalanceConfig().MutationMaxLevel)

	// ── Has owned mutations: advance the strongest one below the cap ──────────
	if len(c.Mutations) > 0 {
		type ownedEntry struct {
			id    string
			level int
		}
		entries := make([]ownedEntry, 0, len(c.Mutations))
		for id, lvl := range c.Mutations {
			entries = append(entries, ownedEntry{id, lvl})
		}
		// Descending level; ties → ascending id for determinism.
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].level != entries[j].level {
				return entries[i].level > entries[j].level
			}
			return entries[i].id < entries[j].id
		})

		for _, e := range entries {
			if e.level < maxLevel {
				c.Mutations[e.id] = e.level + 1
				return e.id, e.level + 1
			}
		}
		// All owned mutations are at cap — fall through to seed a new one.
	}

	return c.BloomSeedNewMutation(rng)
}

// BloomSeedNewMutation grants the character a brand-new mutation drawn from a
// Rarity-weighted pool (biased toward mid/high-tier), respecting ownership,
// conflicts, and species body-part requirements. Used both as BloomAdvanceMutation's
// fall-through (no owned / all-capped) AND directly by the dose hook's separate
// BloomNewMutationChance roll (the "Bloom occasionally pushes a whole new change"
// path). Returns (id, 1) or ("", 0) if no candidate exists. rng nil → global rand.
func (c *Character) BloomSeedNewMutation(rng *rand.Rand) (string, int) {
	allSpecs := mutations.GetAll()
	if len(allSpecs) == 0 {
		return "", 0
	}

	sp := species.GetSpecies(c.SpeciesId) // CanApplyTo(nil) is fail-open

	type candidate struct {
		id     string
		weight int
	}
	candidates := make([]candidate, 0, len(allSpecs))
	for id, spec := range allSpecs {
		if _, owned := c.Mutations[id]; owned {
			continue
		}
		if mutations.HasConflict(c.Mutations, id) {
			continue
		}
		if !spec.CanApplyTo(sp) {
			continue
		}
		// Weight = Rarity (1=common … 10=very rare). Bloom biases toward
		// impactful mutations so higher-rarity specs receive proportionally
		// more weight (inverse of the natural-discovery pool).
		w := spec.Rarity
		if w < 1 {
			w = 1
		}
		candidates = append(candidates, candidate{id, w})
	}
	if len(candidates) == 0 {
		return "", 0
	}

	// Sort by id so the pool layout is deterministic before random selection.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].id < candidates[j].id
	})

	totalWeight := 0
	for _, cand := range candidates {
		totalWeight += cand.weight
	}

	var pick int
	if rng != nil {
		pick = rng.Intn(totalWeight)
	} else {
		pick = rand.Intn(totalWeight)
	}

	for _, cand := range candidates {
		pick -= cand.weight
		if pick < 0 {
			if c.Mutations == nil {
				c.Mutations = make(map[string]int)
			}
			c.Mutations[cand.id] = 1
			return cand.id, 1
		}
	}

	// Unreachable (totalWeight > 0 guarantees an early return above), but
	// belt-and-suspenders: grant the last candidate.
	last := candidates[len(candidates)-1]
	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}
	c.Mutations[last.id] = 1
	return last.id, 1
}
