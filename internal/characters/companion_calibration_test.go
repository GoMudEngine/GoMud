package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
)

// TestCompanionEconomy_Calibration encodes the spec §5 rows end-to-end
// (reserve -> budget), so a knob change that breaks the intended progression
// fails loudly rather than silently drifting the game balance.
func TestCompanionEconomy_Calibration(t *testing.T) {
	seedMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"broodmaster": {
			MutationId: "broodmaster", Name: "Broodmaster", Rarity: 5, Pole: "belief",
			Pros: []mutations.MutationEffect{{Type: "companion_reserve_reduction"}},
		},
	})
	defer seedMut()

	// fieldN counts how many companions of the given per-type base cost this
	// character can field at the given ConvictionMax before the budget (or the
	// soft count backstop) refuses the next one.
	fieldN := func(c *Character, base, convMax int) int {
		c.ConvictionMax.Value = convMax
		c.Companions = nil
		n := 0
		for {
			r := c.CalcCompanionReserve(base)
			if !c.CanAffordCompanion(r) {
				break
			}
			n++
			c.AddCompanion(CompanionInfo{InstanceId: n, Name: "p", ConvictionReserve: r})
		}
		return n
	}

	// Newbie: manif 5, Lesser 350, ConvMax 440 -> exactly 1 wolf.
	nb := New()
	nb.Skills[string(skills.Manifestation)] = 5
	assert.Equal(t, 1, fieldN(nb, 350, 440), "newbie fields exactly 1 spirit wolf")

	// Meirok: manif 48, no mutation, Greater 440, ConvMax 547 -> 2 golems.
	me := New()
	me.Skills[string(skills.Manifestation)] = 48
	assert.Equal(t, 2, fieldN(me, 440, 547), "Meirok fields 2 greater golems, not 3")

	// Fully archetyped: manif 55 + rank-4 mutation, ConvMax 600.
	ar := New()
	ar.Skills[string(skills.Manifestation)] = 55
	ar.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 5, fieldN(ar, 440, 600), "archetyped fields 5 greater golems (soft cap binds)")
	assert.Equal(t, 3, fieldN(ar, 735, 600), "archetyped fields 3 elder golems")

	// Absolute unit: manif 65 + rank-4, ConvMax 850 -> 5 elder golems.
	un := New()
	un.Skills[string(skills.Manifestation)] = 65
	un.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 5, fieldN(un, 735, 850), "absolute unit fields 5 elder golems")
}
