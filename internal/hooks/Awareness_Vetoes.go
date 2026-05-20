package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// wireAwarenessVetoes registers the Activity-check callback that
// vetoes Awareness Visible → Concealing when the character is
// busy with a multi-round activity (casting a spell, crafting an
// item, etc.).
//
// The check returns TRUE when the character is "Free" (i.e., no
// active multi-round activity) and Awareness's Concealing
// transition should proceed.
func wireAwarenessVetoes(c *characters.Character) {
	c.Awareness.RegisterActivityCheck(func() bool {
		return c.IsFree()
	})
}

func init() {
	characters.OnCharacterCreated(wireAwarenessVetoes)
}
