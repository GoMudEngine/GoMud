package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// archetypeYAML is the path to the melee_self_buff archetype YAML.
// Go tests run with cwd = package directory (internal/behaviortree/).
const archetypeYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml"

// Integration tests for the melee_self_buff archetype.
//
// All three tests use the full end-to-end pipeline:
//  1. LoadArchetypeForTest loads the real melee_self_buff.yaml
//  2. A mob with BehaviorArchetype:"melee_self_buff" is seeded
//  3. TryMobBehavior fires mob_combat_round → selector evaluates children
//  4. cast_best_in_category runs synchronously (not in delayedActions),
//     so mob.Command fires inline → "cast X" is queued in events
//  5. events.InspectQueuedInputForTest verifies the cast command
//
// Test 1: fresh mob with an offense spell → casts self_offense
// Test 2: surge buff already active, only defense spells → selector
//         falls through offense (Failure) → casts self_defense
// Test 3: defense-only mob → offense always Failure → casts self_defense

// seedArchetypeSpells installs the four real spells used by melee_self_buff.
// Returns a cleanup function.
func seedArchetypeSpells(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"conviction-surge": {
			SpellId: "conviction-surge", Name: "Conviction Surge",
			Type: spells.HelpSingle, Cost: 35, BaseFolds: 4,
			EffectType: "buff", BuffIds: []int{26},
			Categories: []string{"self_offense"},
		},
		"iron-will": {
			SpellId: "iron-will", Name: "Iron Will",
			Type: spells.HelpSingle, Cost: 45, BaseFolds: 6,
			EffectType: "buff", BuffIds: []int{27},
			Categories: []string{"self_defense"},
		},
		"conviction-ward": {
			SpellId: "conviction-ward", Name: "Conviction Ward",
			Type: spells.HelpSingle, Cost: 30, BaseFolds: 4,
			EffectType: "shield",
			Categories: []string{"self_defense"},
		},
		"conviction-armor": {
			SpellId: "conviction-armor", Name: "Conviction Armor",
			Type: spells.HelpSingle, Cost: 50, BaseFolds: 6,
			EffectType: "buff", BuffIds: []int{38},
			Categories: []string{"self_defense"},
		},
	})
}

// seedArchetypeMob seeds a mob with BehaviorArchetype set to "melee_self_buff"
// and the provided spellbook. Returns the mob pointer and a cleanup function.
func seedArchetypeMob(t *testing.T, instanceId int, spellbook map[string]int) (*mobs.Mob, func()) {
	t.Helper()
	m := &mobs.Mob{
		MobId:             mobs.MobId(300 + instanceId),
		InstanceId:        instanceId,
		BehaviorArchetype: "melee_self_buff",
	}
	m.Character.Name = "testmob"
	m.Character.Conviction = 500
	m.Character.SpellBook = spellbook
	m.Character.Buffs = buffs.New()
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{300 + instanceId: m},
		map[int]*mobs.Mob{instanceId: m},
	)
	return m, cleanup
}

// seedBuffOnChar seeds a buff as active on the character, using the same
// pattern as action_cast_best_in_category_test.go: directly set Buffs.List
// then call Validate(true) to rebuild the buffIds index. Also seeds the buff
// spec so the buffs package doesn't panic on lookup.
func seedBuffOnChar(t *testing.T, char *characters.Character, buffId int) func() {
	t.Helper()
	cleanupBuff := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		buffId: {BuffId: buffId, Name: "TestBuff"},
	})
	char.Buffs.List = append(char.Buffs.List, &buffs.Buff{
		BuffId:       buffId,
		TriggersLeft: 5,
	})
	char.Buffs.Validate(true)
	if !char.HasBuff(buffId) {
		t.Fatalf("seedBuffOnChar: HasBuff(%d) false after seeding — setup broken", buffId)
	}
	return cleanupBuff
}

// TestMeleeSelfBuff_FreshMobCastsSelfOffense is a full end-to-end pipeline
// test: real YAML loaded, TryMobBehavior fires, delayed action drained, and
// the queued "cast conviction-surge" is verified.
func TestMeleeSelfBuff_FreshMobCastsSelfOffense(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff", archetypeYAML)

	mob, cleanup := seedArchetypeMob(t, 90001, map[string]int{
		"conviction-surge": 3, // self_offense — only offense spell
		"conviction-ward":  4, // self_defense — fallback (never reached this round)
		"iron-will":        4, // self_defense — fallback (never reached this round)
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success (tree handled event), got false")
	}
	// Flush the perception-delay queue so mob.Command fires synchronously.
	DrainAllDelayedActionsForTest(t)

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast conviction-surge") {
		t.Fatalf("fresh mob should cast conviction-surge (only self_offense), got %q", cmd)
	}
}

// TestMeleeSelfBuff_WithSurgeActiveCastsIronWill verifies the full selector
// fallthrough: when the offense buff (surge, buff 26) is already active, the
// offense child returns Failure and the selector falls through to defense.
// The defense action picks iron-will (score 6×45=270) over conviction-ward
// (score 4×30=120).
func TestMeleeSelfBuff_WithSurgeActiveCastsIronWill(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff", archetypeYAML)

	mob, cleanup := seedArchetypeMob(t, 90002, map[string]int{
		"conviction-ward": 4, // self_defense, score 120
		"iron-will":       4, // self_defense, score 270 — should win
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	// Mark surge buff 26 as active so the offense child finds no eligible
	// spell and returns Failure, forcing the selector to try defense.
	cleanupBuff := seedBuffOnChar(t, &mob.Character, 26)
	defer cleanupBuff()

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success (tree handled event), got false")
	}

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast iron-will") {
		t.Fatalf("highest-scoring defense (iron-will, score 270) should be cast, got %q", cmd)
	}
}

// TestMeleeSelfBuff_AirElementalCastsDefenseOnly verifies that a mob with
// only self_defense spells correctly falls through the offense child (Failure)
// and casts the highest-scoring defense spell (iron-will, 270 > ward, 120).
// This covers the air/fire elemental archetype case.
func TestMeleeSelfBuff_AirElementalCastsDefenseOnly(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff", archetypeYAML)

	mob, cleanup := seedArchetypeMob(t, 90003, map[string]int{
		"iron-will":       4, // self_defense, score 6×45=270
		"conviction-ward": 4, // self_defense, score 4×30=120
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success (tree handled event), got false")
	}

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast iron-will") {
		t.Fatalf("defense-only mob should cast iron-will (score 270 > ward 120), got %q", cmd)
	}
}
