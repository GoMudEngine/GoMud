package characters

import (
	"github.com/GoMudEngine/GoMud/internal/species"
)

// MutationMaxRank is the absolute cap for any mutation rank, matching
// the chunk-2.2a convention (incorporeal 1-4, extra-arms 1-4, etc.).
// Future mutations exceeding rank 4 will require adding a MaxRank
// field to MutationSpec and consulting it here.
const MutationMaxRank = 4

// ApplyIntrinsicMutations merges the species's intrinsic mutations
// additively into the character's Mutations map. Cap-aware: each
// combined rank is clamped to MutationMaxRank.
//
// Called from mob spawn AND player creation after all other
// mutation logic (curated SpawnMutations from YAML + random
// roll + persistent acquired). No-op if species is nil or has
// no intrinsic_mutations.
func (c *Character) ApplyIntrinsicMutations(sp *species.Species) {
	if sp == nil || sp.MutationImmune || len(sp.IntrinsicMutations) == 0 {
		return
	}
	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}
	for id, intrinsicRank := range sp.IntrinsicMutations {
		combined := c.Mutations[id] + intrinsicRank
		if combined > MutationMaxRank {
			combined = MutationMaxRank
		}
		c.Mutations[id] = combined
	}
}
