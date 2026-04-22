package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const lookoutYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml"

// TestLookout_PackmateHurt_CallsForHelpAndEngages verifies the lookout's
// packmate_hurt handler: queues `callforhelp` first, then engages the
// attacker via the existing attack action (sets Aggro).
func TestLookout_PackmateHurt_CallsForHelpAndEngages(t *testing.T) {
	LoadArchetypeForTest(t, "lookout", lookoutYAML)

	mob, cleanup := seedLookoutMob(t, 90501)
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	DrainAllDelayedActionsForTest(t)

	// callforhelp should be queued.
	callCmd := events.InspectQueuedInputForTest(mob.InstanceId, "callforhelp")
	if callCmd == "" {
		t.Fatalf("lookout should queue callforhelp on packmate_hurt; nothing queued")
	}

	// Aggro should be set on the attacker.
	if mob.Character.Aggro == nil {
		t.Fatalf("lookout should set mob.Aggro on attacker; got nil")
	}
	if mob.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", mob.Character.Aggro.UserId)
	}
}

// TestLookout_MobHurt_CallsForHelpAndEngages verifies the lookout's
// mob_hurt handler: when the lookout itself is attacked, same behavior —
// callforhelp then engage (the attacker's UserId is already available
// via ctx.Event).
func TestLookout_MobHurt_CallsForHelpAndEngages(t *testing.T) {
	LoadArchetypeForTest(t, "lookout", lookoutYAML)

	mob, cleanup := seedLookoutMob(t, 90502)
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{
		EventType: "mob_hurt",
		UserId:    99,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(mob_hurt): expected Success, got false")
	}
	DrainAllDelayedActionsForTest(t)

	callCmd := events.InspectQueuedInputForTest(mob.InstanceId, "callforhelp")
	if callCmd == "" {
		t.Fatalf("lookout should callforhelp on mob_hurt; nothing queued")
	}
	if mob.Character.Aggro == nil || mob.Character.Aggro.UserId != 99 {
		t.Fatalf("expected Aggro.UserId=99; got %+v", mob.Character.Aggro)
	}
}

// seedLookoutMob seeds a mob with BehaviorArchetype set to "lookout".
// Returns the mob pointer and a cleanup function.
func seedLookoutMob(t *testing.T, instanceId int) (*mobs.Mob, func()) {
	t.Helper()
	m := &mobs.Mob{
		MobId:             mobs.MobId(300 + instanceId),
		InstanceId:        instanceId,
		BehaviorArchetype: "lookout",
	}
	m.Character.Name = "testmob"
	m.Character.Conviction = 500
	m.Character.Buffs = buffs.New()
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{300 + instanceId: m},
		map[int]*mobs.Mob{instanceId: m},
	)
	return m, cleanup
}

