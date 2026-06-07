package characters

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// ItemHandleSigil prefixes an opaque item handle (the item's UUID string).
// When a finder receives a target beginning with this sigil, it resolves the
// remainder against the actor's OWN reachable item collections only (backpack,
// worn, bandolier, component-bag contents) — never another player's items.
// This lets the web inventory panel target an exact item instance so it can
// never grab the wrong duplicate. The sigil is unused by command target /
// emote / say parsing (verified against usercommands + items disambiguation).
const ItemHandleSigil = "@"

// isItemHandle reports whether s is an item-handle target. When it is, the
// returned string is the bare handle (the UUID string) with the sigil and any
// surrounding whitespace trimmed.
func isItemHandle(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, ItemHandleSigil) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(s, ItemHandleSigil)), true
}

// itemMatchesHandle reports whether itm is a real item whose UUID equals the
// given handle string.
func itemMatchesHandle(itm items.Item, handle string) bool {
	return itm.ItemId > 0 && itm.UUID.String() == handle
}

// GetComponentBagContents returns a copy of the items currently stored inside
// the equipped component bag. Returns an empty slice when no bag is equipped or
// it is empty.
func (c *Character) GetComponentBagContents() []items.Item {
	return append([]items.Item{}, c.ComponentItems...)
}
