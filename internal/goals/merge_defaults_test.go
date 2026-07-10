package goals

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// mergeDefaultsLookup returns tank_taunter defaults (survival@90) for the
// TestMergeArchetypeDefaults scenarios below.
func mergeDefaultsLookup(m *mobs.Mob) []GoalDefault {
	if m == nil {
		return nil
	}
	switch m.BehaviorArchetype {
	case "tank_taunter":
		return []GoalDefault{{Type: "survival", Priority: 90}}
	}
	return nil
}

// TestMergeArchetypeDefaults: the exported wrapper merges the mob's
// CURRENT archetype's defaults — used by the archetype-shift path after
// swapping BehaviorArchetype. A pre-existing goal of a DIFFERENT type
// survives, and the missing default type is added.
func TestMergeArchetypeDefaults(t *testing.T) {
	ClearCache()
	const mobId = 99805
	const name = "mergeprobe"

	SetArchetypeDefaultsLookup(mergeDefaultsLookup)
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

// TestMergeArchetypeDefaults_SameTypeHigherPriorityKept proves the
// documented same-type semantics: a pre-existing goal of the SAME type
// as an archetype default, at STRICTLY HIGHER priority, blocks the
// default (Add's ConflictError, swallowed by the merge) — it is neither
// displaced nor duplicated.
func TestMergeArchetypeDefaults_SameTypeHigherPriorityKept(t *testing.T) {
	ClearCache()
	const mobId = 99806
	const name = "mergeprobe_prio"

	SetArchetypeDefaultsLookup(mergeDefaultsLookup)
	t.Cleanup(func() { SetArchetypeDefaultsLookup(nil) })

	// Pre-existing survival goal OUTRANKING the archetype default (90).
	if _, err := Add(mobId, name, &Goal{Type: "survival", Priority: 95}); err != nil {
		t.Fatalf("Add(survival@95): %v", err)
	}

	m := &mobs.Mob{MobId: mobs.MobId(mobId), BehaviorArchetype: "tank_taunter"}
	m.Character.Name = name
	MergeArchetypeDefaults(mobId, name, m)

	var survival []*Goal
	for _, g := range GoalsOf(mobId, name) {
		if g.Type == "survival" {
			survival = append(survival, g)
		}
	}
	if len(survival) != 1 {
		t.Fatalf("expected exactly 1 survival goal after merge, got %d", len(survival))
	}
	if survival[0].Priority != 95 {
		t.Errorf("survival goal priority: got %d, want 95 (lower-priority default should be skipped, not displace)",
			survival[0].Priority)
	}
}
