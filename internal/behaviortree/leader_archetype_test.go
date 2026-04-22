package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const leaderYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/leader.yaml"

// TestLeader_PackmateHurt_RallyOrWarcryThenEngage verifies the leader's
// packmate_hurt handler: queues rally or warcry (command_best_of fires the
// first ready one — CommandIsReady skips if the buff is already active),
// then engages the attacker via the existing attack action (sets Aggro).
func TestLeader_PackmateHurt_RallyOrWarcryThenEngage(t *testing.T) {
	LoadArchetypeForTest(t, "leader", leaderYAML)

	mob, cleanup := seedLeaderMob(t, 90701)
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

	// command_best_of issues the first ready command. Either rally or
	// warcry should be queued.
	rally := events.InspectQueuedInputForTest(mob.InstanceId, "rally")
	warcry := events.InspectQueuedInputForTest(mob.InstanceId, "warcry")
	if rally == "" && warcry == "" {
		t.Fatalf("expected rally or warcry to be queued on packmate_hurt; got neither")
	}

	// Attack should set Aggro.
	if mob.Character.Aggro == nil {
		t.Fatalf("expected mob.Aggro to be set; got nil")
	}
	if mob.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", mob.Character.Aggro.UserId)
	}
}

// TestLeader_MobHurt_RallyOrWarcryThenEngage verifies the leader's
// mob_hurt handler: when the leader itself is attacked, same behavior —
// rally or warcry then engage (the attacker's UserId is already available
// via ctx.Event).
func TestLeader_MobHurt_RallyOrWarcryThenEngage(t *testing.T) {
	LoadArchetypeForTest(t, "leader", leaderYAML)

	mob, cleanup := seedLeaderMob(t, 90702)
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

	rally := events.InspectQueuedInputForTest(mob.InstanceId, "rally")
	warcry := events.InspectQueuedInputForTest(mob.InstanceId, "warcry")
	if rally == "" && warcry == "" {
		t.Fatalf("expected rally or warcry to be queued on mob_hurt; got neither")
	}
	if mob.Character.Aggro == nil || mob.Character.Aggro.UserId != 99 {
		t.Fatalf("expected Aggro.UserId=99; got %+v", mob.Character.Aggro)
	}
}

// seedLeaderMob seeds a mob with BehaviorArchetype set to "leader".
// Returns the mob pointer and a cleanup function.
func seedLeaderMob(t *testing.T, instanceId int) (*mobs.Mob, func()) {
	t.Helper()
	m := &mobs.Mob{
		MobId:             mobs.MobId(300 + instanceId),
		InstanceId:        instanceId,
		BehaviorArchetype: "leader",
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
