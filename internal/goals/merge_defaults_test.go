package goals

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestMergeArchetypeDefaults: the exported wrapper additively merges the
// mob's CURRENT archetype's defaults — used by the archetype-shift path
// after swapping BehaviorArchetype. Existing goals survive.
func TestMergeArchetypeDefaults(t *testing.T) {
	const mobId = 99801
	const name = "mergeprobe"

	SetArchetypeDefaultsLookup(func(m *mobs.Mob) []GoalDefault {
		if m == nil {
			return nil
		}
		switch m.BehaviorArchetype {
		case "tank_taunter":
			return []GoalDefault{{Type: "survival", Priority: 90}}
		}
		return nil
	})
	t.Cleanup(func() { SetArchetypeDefaultsLookup(nil) })

	// Pre-existing goal of a different type.
	if _, err := Add(mobId, name, &Goal{Type: "wealth", Priority: 50}); err != nil {
		t.Fatalf("Add(wealth): %v", err)
	}

	m := &mobs.Mob{MobId: mobs.MobId(mobId), BehaviorArchetype: "tank_taunter"}
	m.Character.Name = name
	MergeArchetypeDefaults(mobId, name, m)

	types := map[string]bool{}
	for _, g := range GoalsOf(mobId, name) {
		types[g.Type] = true
	}
	if !types["wealth"] {
		t.Error("pre-existing wealth goal was lost by the merge")
	}
	if !types["survival"] {
		t.Error("archetype default survival goal was not merged in")
	}
}
