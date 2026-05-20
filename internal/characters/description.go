package characters

import (
	"fmt"
	"slices"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	descriptionCache = map[string]string{} // key is a hash, value is the description
)

// GetDescription returns the character's description. It may be a hash-prefixed
// value (h:hash) that points to another description location.
func (c *Character) GetDescription() string {

	desc := c.Description
	if strings.HasPrefix(desc, `h:`) {
		hash := strings.TrimPrefix(desc, `h:`)
		desc = descriptionCache[hash]
	}

	// Normalize line breaks: collapse single newlines to spaces
	// (continuation wrapping) while preserving double newlines
	// (intentional paragraph breaks).
	desc = strings.ReplaceAll(desc, "\r\n", "\n")
	desc = strings.ReplaceAll(desc, "\n\n", "\x00") // protect paragraph breaks
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.ReplaceAll(desc, "\x00", "\n\n") // restore paragraph breaks
	for strings.Contains(desc, "  ") {
		desc = strings.ReplaceAll(desc, "  ", " ")
	}

	return desc
}

// GetMutationVisuals returns a space-joined string of all owned mutation visual
// descriptors, sorted by mutation id for deterministic output. Returns "" if
// no mutations have a visual field. Used by the description template (Stage 12.2).
func (c *Character) GetMutationVisuals() string {
	if len(c.Mutations) == 0 {
		return ""
	}
	ids := make([]string, 0, len(c.Mutations))
	for id := range c.Mutations {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		if spec := mutations.GetMutation(id); spec != nil && spec.Visual != "" {
			parts = append(parts, spec.Visual)
		}
	}
	return strings.Join(parts, " ")
}

// GetHealthAppearance returns a string describing the character's current health state.
// USERNAME appears to be <BLANK>
func (c *Character) GetHealthAppearance() string {

	className := util.HealthClass(c.Health, c.HealthMax.Value)
	pct := int(float64(c.Health) / float64(c.HealthMax.Value) * 100)

	if pct < 15 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> looks like they're <ansi fg="%s">about to die!</ansi>`, c.Name, className)
	}

	if pct < 50 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> looks to be in <ansi fg="%s">pretty bad shape.</ansi>`, c.Name, className)
	}

	if pct < 80 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> has some <ansi fg="%s">cuts and bruises.</ansi>`, c.Name, className)
	}

	if pct < 100 {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi> has <ansi fg="%s">a few scratches.</ansi>`, c.Name, className)
	}

	return fmt.Sprintf(`<ansi fg="username">%s</ansi> is in <ansi fg="%s">perfect health.</ansi>`, c.Name, className)
}

// CacheDescription should only be used for mobs, not players.
// Hashes the description and stores it centrally.
// This saves a lot of memory because many descriptions are duplicates.
func (c *Character) CacheDescription() {
	hash := util.Hash(c.Description)
	if _, ok := descriptionCache[hash]; !ok {
		descriptionCache[hash] = c.Description
	}
	c.Description = fmt.Sprintf(`h:%s`, hash)
}

func (c *Character) HasAdjective(adj string) bool {
	return slices.Contains(c.Adjectives, adj)
}

func (c *Character) SetAdjective(adj string, addToList bool) {
	if c.Adjectives == nil {
		c.Adjectives = []string{}
	}
	for i, a := range c.Adjectives {
		if a == adj {
			if addToList {
				return
			} else {
				c.Adjectives = slices.Delete(c.Adjectives, i, i+1)
				return
			}
		}
	}
	if addToList {
		c.Adjectives = append(c.Adjectives, adj)
	}
}

func (c *Character) GetAdjectives() []string {

	retAdjectives := []string{}

	// Start dynamic adjectives
	if c.Health < 1 {
		retAdjectives = append(retAdjectives, `downed`)
	}

	if len(c.Shop) > 0 {
		retAdjectives = append(retAdjectives, `shop`)
	}

	if c.HasFlagFromAnySource(buffs.EmitsLight) {
		retAdjectives = append(retAdjectives, `lit`)
	}

	if c.IsHidden() {
		retAdjectives = append(retAdjectives, `hidden`)
	}

	if c.HasBuffFlag(buffs.Poison) {
		retAdjectives = append(retAdjectives, `poisoned`)
	}
	// End dynamic adjectives

	retAdjectives = append(retAdjectives, c.Adjectives...)

	return retAdjectives
}

func (c *Character) Species() string {
	if r := species.GetSpecies(c.SpeciesId); r != nil {
		return r.Name
	}
	return `Ghostly Spirit`
}

func (c *Character) BarterPrice(startPrice int) int {
	factor := (float64(c.Stats.Charisma.ValueAdj) / 3) / 100 // 100 = 33% discount, 0 = 0% discount, 300 = 100% discount
	if factor > .75 {
		factor = .75
	}
	return int(factor * float64(startPrice))
}
