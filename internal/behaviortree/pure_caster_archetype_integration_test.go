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

const pureCasterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml"

// seedCasterSpells seeds the spells the pure_caster archetype uses.
// Includes Phase 4's self_defense spells (iron-will, conviction-ward) plus
// the new self_heal + harm_single + harm_multi additions. Replaces the
// entire allSpells map via SeedSpellsForTest.
func seedCasterSpells(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		// self_defense (reused from Phase 4)
		"iron-will": {
			SpellId: "iron-will", Name: "Iron Will",
			Type: spells.HelpSingle, Cost: 45, BaseFolds: 6,
			EffectType: "buff", BuffIds: []int{27},
			Categories: []string{"self_defense"},
		},
		"conviction-ward": {
			SpellId: "conviction-ward", Name: "Conviction Ward",
			Type: spells.HelpSingle, Cost: 30, BaseFolds: 4,
			EffectType: "shield", EffectMagnitude: 75,
			Categories: []string{"self_defense"},
		},
		// self_heal
		"heal": {
			SpellId: "heal", Name: "Heal",
			Type: spells.HelpSingle, Cost: 40, BaseFolds: 5,
			EffectType: "heal", EffectMagnitude: 3,
			Categories: []string{"self_heal"},
		},
		// harm_single
		"mind-spike": {
			SpellId: "mind-spike", Name: "Mind Spike",
			Type: spells.HarmSingle, Cost: 35, BaseFolds: 4,
			EffectType: "damage",
			Categories: []string{"harm_single"},
		},
		// harm_multi
		"sparks": {
			SpellId: "sparks", Name: "Sparks",
			Type: spells.HarmArea, Cost: 75, BaseFolds: 4,
			EffectType: "damage",
			Categories: []string{"harm_multi"},
		},
	})
}

// seedCasterMob seeds a mob with BehaviorArchetype: "pure_caster" and the
// provided spellbook. HP starts at 100/100 (full). Caller can mutate
// m.Character.Health for low-HP tests.
func seedCasterMob(t *testing.T, instanceId int, spellbook map[string]int) (*mobs.Mob, func()) {
	t.Helper()
	m := &mobs.Mob{
		MobId:             mobs.MobId(400 + instanceId),
		InstanceId:        instanceId,
		BehaviorArchetype: "pure_caster",
	}
	m.Character.Name = "testcaster"
	m.Character.Conviction = 500
	m.Character.Health = 100
	m.Character.HealthMax.Value = 100
	m.Character.SpellBook = spellbook
	m.Character.Buffs = buffs.New()
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{400 + instanceId: m},
		map[int]*mobs.Mob{instanceId: m},
	)
	return m, cleanup
}

// TestPureCaster_FullHP_MaintainsDefenseFirst verifies that a full-HP,
// unbuffed caster casts its top-scoring self_defense spell first — heal
// branch is gated by mob_health_below (HP not < 40%), so the selector
// moves to self_defense.
func TestPureCaster_FullHP_MaintainsDefenseFirst(t *testing.T) {
	defer seedCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster", pureCasterYAML)

	mob, cleanup := seedCasterMob(t, 91001, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
		"heal":            3,
		"mind-spike":      5,
		"sparks":          3,
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success, got false")
	}
	DrainAllDelayedActionsForTest(t)

	// iron-will (6×45=270) > conviction-ward (4×30=120) → iron-will wins.
	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast iron-will") {
		t.Fatalf("full-HP fresh caster should cast iron-will (top defense), got %q", cmd)
	}
}

// TestPureCaster_LowHP_EmergencyHeal verifies that when HP drops below 40%
// the heal branch of the selector fires first, preempting defense.
func TestPureCaster_LowHP_EmergencyHeal(t *testing.T) {
	defer seedCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster", pureCasterYAML)

	mob, cleanup := seedCasterMob(t, 91002, map[string]int{
		"conviction-ward": 4,
		"heal":            3,
		"mind-spike":      5,
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	// Drop HP below 40% (30/100 = 30%).
	mob.Character.Health = 30

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success, got false")
	}
	DrainAllDelayedActionsForTest(t)

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast heal") {
		t.Fatalf("low-HP caster should cast heal first, got %q", cmd)
	}
}

// TestPureCaster_DefenseCovered_SingleEnemy_CastsHarmSingle verifies that
// when defense buffs are active, HP is fine, and no "multiple enemies"
// condition fires (1 actor = caster itself), the selector reaches the
// harm_single branch.
func TestPureCaster_DefenseCovered_SingleEnemy_CastsHarmSingle(t *testing.T) {
	defer seedCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster", pureCasterYAML)

	mob, cleanup := seedCasterMob(t, 91003, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
		"heal":            3,
		"mind-spike":      5,
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	// Activate iron-will (buff 27) and conviction-ward (ConditionShield).
	defer seedBuffOnChar(t, &mob.Character, 27)()
	mob.Character.AddCondition(characters.ConditionShield, 20, 75, "test")

	// No room setup → multiple_enemies condition returns Failure (no other
	// actors), so the AoE branch skips. Selector drops to harm_single.
	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success, got false")
	}
	DrainAllDelayedActionsForTest(t)

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast mind-spike") {
		t.Fatalf("defense-covered, single-enemy caster should cast mind-spike, got %q", cmd)
	}
}

// TestPureCaster_NoCandidates_FallsThrough verifies that when every branch
// has no viable candidates (all defense active, no harm spells known,
// HP fine) the selector returns Failure so legacy combat runs.
func TestPureCaster_NoCandidates_FallsThrough(t *testing.T) {
	defer seedCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster", pureCasterYAML)

	// Caster with only self_defense spells — no harm, no heal. All defense active.
	mob, cleanup := seedCasterMob(t, 91004, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	defer seedBuffOnChar(t, &mob.Character, 27)()
	mob.Character.AddCondition(characters.ConditionShield, 20, 75, "test")

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if ok {
		t.Fatalf("TryMobBehavior: expected Failure (all branches declined), got Success")
	}
}
