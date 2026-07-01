package characters

import "github.com/GoMudEngine/GoMud/internal/species"

// ScourMutations strips ALL of the character's mutations (the moon-crash
// "remort"), re-applies species intrinsics, resets mutation progress, and
// grants `charges` reroll charges. While charges remain, re-acquired
// mutations bias hard toward rare ones (see BloomSeedNewMutation).
func (c *Character) ScourMutations(charges int) {
	c.Mutations = map[string]int{}
	if sp := species.GetSpecies(c.SpeciesId); sp != nil {
		c.ApplyIntrinsicMutations(sp)
	}
	c.MutationProgress = 0
	if charges < 0 {
		charges = 0
	}
	c.MutationRerollBonus = charges
}
