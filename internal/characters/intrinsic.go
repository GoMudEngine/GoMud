package characters

import (
	"github.com/GoMudEngine/GoMud/internal/species"
)

// ApplyIntrinsicMutations merges the species's intrinsic mutations
// additively into the character's Mutations map. Cap-aware: each
// combined rank is clamped to the mutation's max rank (default cap = 4,
// matching the chunk-2.2a convention for ranked mutations).
//
// Called from mob spawn AND player creation after all other
// mutation logic (curated SpawnMutations from YAML + random
// roll + persistent acquired). No-op if species is nil or has
// no intrinsic_mutations.
func (c *Character) ApplyIntrinsicMutations(sp *species.Species) {
	if sp == nil || len(sp.IntrinsicMutations) == 0 {
		return
	}
	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}
	cap := 4 // default cap matches the chunk-2.2a convention; no per-mutation max field exists
	for id, intrinsicRank := range sp.IntrinsicMutations {
		combined := c.Mutations[id] + intrinsicRank
		if combined > cap {
			combined = cap
		}
		c.Mutations[id] = combined
	}
}
