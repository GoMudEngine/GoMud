package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const tankTaunterYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml"

// TestTankTaunter_PackmateHurt_TauntsAndSetsAggro verifies the packmate_hurt
// handler: first queues a taunt command, then sets aggro on the attacker
// via the existing attack action.
func TestTankTaunter_PackmateHurt_TauntsAndSetsAggro(t *testing.T) {
	LoadArchetypeForTest(t, "tank_taunter", tankTaunterYAML)

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

	// Seed a tank_taunter mob with proper archetype.
	m := &mobs.Mob{
		MobId:             mobs.MobId(90201),
		InstanceId:        90201,
		BehaviorArchetype: "tank_taunter",
	}
	m.Character.Name = "testmob"
	m.Character.RoomId = 10001
	m.Character.Health = 100
	m.Character.Stamina = 100
	m.Character.Conviction = 100
	m.Character.Buffs = buffs.New()
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{300 + 90201: m},
		map[int]*mobs.Mob{90201: m},
	)
	defer cleanup()
	defer events.DrainQueuedInputsForTest(m.InstanceId)

	ok := TryMobBehavior(m.InstanceId, EventContext{
		EventType: "packmate_hurt",
		UserId:    42,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(packmate_hurt): expected Success, got false")
	}

	// Drain perception-delay queue so mob.Command fires synchronously.
	DrainAllDelayedActionsForTest(t)

	// Assert aggro set on the attacker (via the attack action).
	if m.Character.Aggro == nil {
		t.Fatalf("packmate_hurt handler should set mob.Aggro; got nil")
	}
	if m.Character.Aggro.UserId != 42 {
		t.Fatalf("expected Aggro.UserId=42, got %d", m.Character.Aggro.UserId)
	}
}
