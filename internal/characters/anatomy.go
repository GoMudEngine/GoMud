package characters

import "github.com/GoMudEngine/GoMud/internal/species"

// HasBodyPart reports whether this character's species declares the given
// canonical body-part tag (e.g. "arms", "legs", "mouth"). Used to gate which
// combat moves a creature's anatomy permits (humanoid technique moves need
// "arms"; trip/kick need "legs"). Returns false if the species is unknown.
func (c *Character) HasBodyPart(part string) bool {
	sp := species.GetSpecies(c.SpeciesId)
	if sp == nil {
		return false
	}
	for _, p := range sp.BodyParts {
		if p == part {
			return true
		}
	}
	return false
}
