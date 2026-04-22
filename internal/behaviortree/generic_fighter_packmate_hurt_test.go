package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const genericFighterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml"

// TestGenericFighter_PackmateHurt_SetsAggroOnAttacker verifies the new
// packmate_hurt handler: when a packmate is hurt, the generic_fighter
// sets aggro on the attacker (via the existing actAttack action,
// which reads ctx.Event.UserId).

func TestGenericFighter_PackmateHurt_SetsAggroOnAttacker(t *testing.T) {
	LoadArchetypeForTest(t, "generic_fighter", genericFighterYAML)

	// Disable grace period checks for this test.
	characters.SetUserUntargetableCheck(nil)
	t.Cleanup(func() { characters.SetUserUntargetableCheck(nil) })

	// Seed a test room
	testRoom := &rooms.Room{
		RoomId: 10001,
		Title:  "Test Room",
		Zone:   "test_zone",
	}
	cleanupRoom := rooms.SeedRoomsForTest(map[int]*rooms.Room{10001: testRoom}, map[string]*rooms.ZoneConfig{})
	defer cleanupRoom()

	m := &mobs.Mob{
		MobId:             mobs.MobId(90101),
		InstanceId:        90101,
		BehaviorArchetype: "generic_fighter",
	}
	m.Character.Name = "testmob"
	m.Character.RoomId = 10001
	m.Character.Health = 100
	m.Character.Stamina = 100
	m.Character.Conviction = 100
	m.Character.Buffs = buffs.New()
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{300 + 90101: m},
		map[int]*mobs.Mob{90101: m},
	)
	defer cleanup()

	ok := TryMobBehavior(m.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	// The "attack" action is in delayedActions, so it runs asynchronously.
	// Drain the queue to execute it synchronously for the test.
	DrainAllDelayedActionsForTest(t)

	// actAttack sets Aggro directly — verify via mob state.
	if m.Character.Aggro == nil {
		t.Fatalf("packmate_hurt handler should set mob.Aggro; got nil")
	}
	if m.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", m.Character.Aggro.UserId)
	}
}
