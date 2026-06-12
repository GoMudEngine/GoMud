package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// TestCombatSkillTagForItem_Shooting verifies that shooting-subtype weapons
// (bows, crossbows, guns) train the revived ranged-combat skill rather than
// weapon-combat.
func TestCombatSkillTagForItem_Shooting(t *testing.T) {
	bow := items.Item{ItemId: 1, Spec: &items.ItemSpec{ItemId: 1, Type: items.Weapon, Subtype: items.Shooting}}
	if got := CombatSkillTagForItem(bow); got != skills.RangedCombat {
		t.Errorf("shooting weapon skill tag = %v, want ranged-combat", got)
	}
}
