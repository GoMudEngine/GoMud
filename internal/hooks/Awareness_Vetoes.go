package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// wireAwarenessVetoes registers the Activity-check callback that
// vetoes Awareness Visible → Concealing when the character is
// busy with a multi-round activity (casting a spell, crafting an
// item, etc.).
//
// Chunk-3 (Activity machine) will repoint this callback to the
// proper Activity machine query. For chunk 1 it consults the
// existing CastingState/CraftingState pointers on Character.
//
// The check returns TRUE when the character is "Free" (i.e., no
// active multi-round activity) and Awareness's Concealing
// transition should proceed.
func wireAwarenessVetoes(c *characters.Character) {
	c.Awareness.RegisterActivityCheck(func() bool {
		return c.CastingState == nil && c.CraftingState == nil
	})
}

func init() {
	characters.OnCharacterCreated(wireAwarenessVetoes)
}
