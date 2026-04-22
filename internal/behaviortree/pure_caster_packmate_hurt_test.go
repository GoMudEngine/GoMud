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

const pureCasterYAML2 = "../../_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml"

// TestPureCaster_PackmateHurt_HealsWoundedPackmate verifies that when a
// same-room packmate is wounded below 40% HP, pure_caster's packmate_hurt
// handler casts a heal_friendly spell targeting that packmate
// (most_wounded_packmate resolver).
func TestPureCaster_PackmateHurt_HealsWoundedPackmate(t *testing.T) {
	defer seedCasterSpellsForPackmateTest(t)()
	LoadArchetypeForTest(t, "pure_caster", pureCasterYAML2)

	// Seed the caster with a heal_friendly spell so the category
	// candidate list is non-empty.
	caster, cleanup := seedCasterMobForPackmateTest(t, 90401, map[string]int{
		"heal": 4, // heal_friendly category
	})
	caster.Routine = "wraith_pack"
	caster.Character.HealthMax.Value = 100
	defer cleanup()
	defer events.DrainQueuedInputsForTest(caster.InstanceId)

	// Seed a wounded packmate in the same room.
	wounded := &mobs.Mob{
		InstanceId: 90402,
		Routine:    "wraith_pack",
		Character: characters.Character{
			RoomId: caster.Character.RoomId,
			Health: 30, // 30% of 100 — below 40% threshold
		},
	}
	wounded.Character.HealthMax.Value = 100
	defer events.DrainQueuedInputsForTest(wounded.InstanceId)
	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		caster.InstanceId: caster,
		90402:             wounded,
	})
	defer cleanup2()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	// The cast_best_in_category action with target=most_wounded_packmate
	// issues a `cast <spell> #<instId>` command. Verify via the queued
	// input.
	cmd := events.InspectQueuedInputForTest(caster.InstanceId, "cast ")
	if cmd == "" {
		t.Fatalf("expected a heal cast queued; nothing cast")
	}
	// The command should include "#90402" (the wounded packmate's instance id).
	if !strings.Contains(cmd, "#90402") {
		t.Fatalf("expected cast to target packmate #90402, got %q", cmd)
	}
}

// TestPureCaster_PackmateHurt_NoWoundedPackmate_EngagesAttacker verifies that
// when no packmate is wounded, pure_caster's packmate_hurt handler falls
// through to engaging the attacker via the existing attack action (sets Aggro).
func TestPureCaster_PackmateHurt_NoWoundedPackmate_EngagesAttacker(t *testing.T) {
	defer seedCasterSpellsForPackmateTest(t)()
	LoadArchetypeForTest(t, "pure_caster", pureCasterYAML2)

	caster, cleanup := seedCasterMobForPackmateTest(t, 90403, map[string]int{
		"heal": 4,
	})
	caster.Routine = "wraith_pack"
	caster.Character.HealthMax.Value = 100
	defer cleanup()
	defer events.DrainQueuedInputsForTest(caster.InstanceId)

	healthy := &mobs.Mob{
		InstanceId: 90404,
		Routine:    "wraith_pack",
		Character: characters.Character{
			RoomId: caster.Character.RoomId,
			Health: 95, // 95% — above 40% threshold
		},
	}
	healthy.Character.HealthMax.Value = 100
	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		caster.InstanceId: caster,
		90404:             healthy,
	})
	defer cleanup2()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	// Drain delayedActions so the attack sets Aggro synchronously.
	DrainAllDelayedActionsForTest(t)

	// No heal should have been cast (threshold not met).
	cmd := events.InspectQueuedInputForTest(caster.InstanceId, "cast ")
	if cmd != "" {
		t.Fatalf("no heal should have been cast; got: %q", cmd)
	}
	// Attack should have been triggered (sets Aggro).
	if caster.Character.Aggro == nil {
		t.Fatalf("expected mob.Aggro to be set on attacker; got nil")
	}
	if caster.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", caster.Character.Aggro.UserId)
	}
}

// seedCasterSpellsForPackmateTest seeds the minimal spells for packmate_hurt
// testing (heal with heal_friendly category).
func seedCasterSpellsForPackmateTest(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"heal": {
			SpellId: "heal", Name: "Mend Flesh",
			Type: spells.HelpSingle, Cost: 25, BaseFolds: 4,
			EffectType: "heal", EffectMagnitude: 3,
			Categories: []string{"heal_friendly"},
		},
	})
}

// seedCasterMobForPackmateTest seeds a mob with BehaviorArchetype: "pure_caster"
// and the provided spellbook for packmate_hurt testing. HP starts at 100/100.
func seedCasterMobForPackmateTest(t *testing.T, instanceId int, spellbook map[string]int) (*mobs.Mob, func()) {
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
