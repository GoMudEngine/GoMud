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

const supportCasterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/support_caster.yaml"

// TestSupportCaster_PackmateHurt_ShieldsTankingPackmate verifies that when
// a same-room packmate is tanking (has Aggro set), support_caster's
// packmate_hurt handler casts a buff_friendly (shield) spell on the tanking
// packmate, taking priority over healing wounded packmates.
func TestSupportCaster_PackmateHurt_ShieldsTankingPackmate(t *testing.T) {
	defer seedSupportCasterSpells(t)()
	LoadArchetypeForTest(t, "support_caster", supportCasterYAML)

	// The caster has both heal_friendly and buff_friendly available.
	// If the archetype prioritizes correctly, the tanking packmate
	// triggers the buff_friendly branch (shield the tank) — even if
	// another packmate is wounded below 70%, the tank-shielding
	// priority wins.
	caster, cleanup := seedSupportCasterMob(t, 90601, map[string]int{
		"mend-wounds":     4, // heal_friendly
		"conviction-ward": 4, // buff_friendly
	})
	caster.Routine = "bandit_camp_guard"
	caster.Character.HealthMax.Value = 100
	defer cleanup()
	defer events.DrainQueuedInputsForTest(caster.InstanceId)

	tanking := &mobs.Mob{
		InstanceId: 90602,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: caster.Character.RoomId,
			Health: 100,
			Aggro:  &characters.Aggro{UserId: 42, Type: characters.DefaultAttack},
		},
	}
	tanking.Character.HealthMax.Value = 100

	// Also a lightly-wounded packmate to prove tank-priority wins.
	wounded := &mobs.Mob{
		InstanceId: 90603,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: caster.Character.RoomId,
			Health: 50, // 50% — would trigger heal if no tank
		},
	}
	wounded.Character.HealthMax.Value = 100

	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		caster.InstanceId: caster,
		90602:             tanking,
		90603:             wounded,
	})
	defer cleanup2()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	cmd := events.InspectQueuedInputForTest(caster.InstanceId, "cast ")
	if cmd == "" {
		t.Fatalf("expected a cast command; nothing queued")
	}
	if !strings.Contains(cmd, "#90602") {
		t.Fatalf("expected cast to target tanking packmate #90602, got: %q", cmd)
	}
}

// TestSupportCaster_PackmateHurt_HealsWoundedWhenNoTank verifies that when
// no packmate is tanking, but a wounded packmate is below 70% HP,
// support_caster's packmate_hurt handler casts a heal_friendly spell on the
// most_wounded_packmate.
func TestSupportCaster_PackmateHurt_HealsWoundedWhenNoTank(t *testing.T) {
	defer seedSupportCasterSpells(t)()
	LoadArchetypeForTest(t, "support_caster", supportCasterYAML)

	caster, cleanup := seedSupportCasterMob(t, 90604, map[string]int{
		"mend-wounds":     4,
		"conviction-ward": 4,
	})
	caster.Routine = "bandit_camp_guard"
	caster.Character.HealthMax.Value = 100
	defer cleanup()
	defer events.DrainQueuedInputsForTest(caster.InstanceId)

	wounded := &mobs.Mob{
		InstanceId: 90605,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: caster.Character.RoomId,
			Health: 50, // 50% — below 70% threshold
		},
	}
	wounded.Character.HealthMax.Value = 100

	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		caster.InstanceId: caster,
		90605:             wounded,
	})
	defer cleanup2()

	ok := TryMobBehavior(caster.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	cmd := events.InspectQueuedInputForTest(caster.InstanceId, "cast ")
	if cmd == "" {
		t.Fatalf("expected a heal cast; nothing queued")
	}
	if !strings.Contains(cmd, "#90605") {
		t.Fatalf("expected cast to target wounded packmate #90605, got: %q", cmd)
	}
}

// TestSupportCaster_PackmateHurt_EngagesAttackerWhenNoSupportNeeded verifies
// that when no packmate is tanking and no packmate is wounded, support_caster's
// packmate_hurt handler engages the attacker via the existing attack action
// (sets Aggro).
func TestSupportCaster_PackmateHurt_EngagesAttackerWhenNoSupportNeeded(t *testing.T) {
	defer seedSupportCasterSpells(t)()
	LoadArchetypeForTest(t, "support_caster", supportCasterYAML)

	caster, cleanup := seedSupportCasterMob(t, 90606, map[string]int{
		"mend-wounds":     4,
		"conviction-ward": 4,
	})
	caster.Routine = "bandit_camp_guard"
	caster.Character.HealthMax.Value = 100
	defer cleanup()
	defer events.DrainQueuedInputsForTest(caster.InstanceId)

	healthy := &mobs.Mob{
		InstanceId: 90607,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: caster.Character.RoomId,
			Health: 100, // full HP, no Aggro — no support needed
		},
	}
	healthy.Character.HealthMax.Value = 100
	cleanup2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		caster.InstanceId: caster,
		90607:             healthy,
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

	cmd := events.InspectQueuedInputForTest(caster.InstanceId, "cast ")
	if cmd != "" {
		t.Fatalf("no cast should be queued; got: %q", cmd)
	}
	if caster.Character.Aggro == nil {
		t.Fatalf("expected Aggro to be set on attacker; got nil")
	}
	if caster.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", caster.Character.Aggro.UserId)
	}
}

// seedSupportCasterSpells seeds the minimal spells for support_caster testing
// (heal_friendly and buff_friendly categories).
func seedSupportCasterSpells(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"mend-wounds": {
			SpellId: "mend-wounds", Name: "Mend Wounds",
			Type: spells.HelpSingle, Cost: 40, BaseFolds: 4,
			EffectType: "heal", EffectMagnitude: 5,
			Categories: []string{"heal_friendly"},
		},
		"conviction-ward": {
			SpellId: "conviction-ward", Name: "Conviction Ward",
			Type: spells.HelpSingle, Cost: 30, BaseFolds: 4,
			EffectType: "shield", EffectMagnitude: 75,
			Categories: []string{"buff_friendly"},
		},
	})
}

// seedSupportCasterMob seeds a mob with BehaviorArchetype: "support_caster"
// and the provided spellbook for packmate_hurt testing. HP starts at 100/100.
func seedSupportCasterMob(t *testing.T, instanceId int, spellbook map[string]int) (*mobs.Mob, func()) {
	t.Helper()
	m := &mobs.Mob{
		MobId:             mobs.MobId(400 + instanceId),
		InstanceId:        instanceId,
		BehaviorArchetype: "support_caster",
	}
	m.Character.Name = "testsupport"
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
