package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TestGenericFighter_HeardCallforhelp_IssuesGoCommand verifies the new
// heard_callforhelp handler on generic_fighter: it resolves the caller's
// room via ctx.Event.MobId and issues a `go <exit>` command toward the
// connecting exit.
func TestGenericFighter_HeardCallforhelp_IssuesGoCommand(t *testing.T) {
	LoadArchetypeForTest(t, "generic_fighter", genericFighterYAML)

	// Seed rooms: room 1 (caller), room 2 (responder). Responder's
	// room has a south exit to room 1.
	room1 := &rooms.Room{RoomId: 1, Zone: "TestZone"}
	room2 := &rooms.Room{RoomId: 2, Zone: "TestZone", Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}}
	cleanupRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room1, 2: room2}, nil)
	defer cleanupRooms()

	// Caller mob in room 1 — just needs to exist so GetInstance resolves.
	caller := &mobs.Mob{
		InstanceId: 100,
		Character:  characters.Character{RoomId: 1, Health: 50},
	}

	// Responder mob in room 2, generic_fighter archetype.
	responder := &mobs.Mob{
		MobId:             mobs.MobId(300 + 91001),
		InstanceId:        91001,
		BehaviorArchetype: "generic_fighter",
	}
	responder.Character.Name = "testmob"
	responder.Character.RoomId = 2
	responder.Character.Health = 100
	responder.Character.Stamina = 100
	responder.Character.Conviction = 500
	responder.Character.Buffs = buffs.New()

	seed := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		100:   caller,
		91001: responder,
	})
	defer seed()
	defer events.DrainQueuedInputsForTest(responder.InstanceId)

	ok := TryMobBehavior(responder.InstanceId, EventContext{
		EventType: "heard_callforhelp",
		MobId:     100, // caller instance id
	})
	if !ok {
		t.Fatalf("TryMobBehavior(heard_callforhelp): expected Success, got false")
	}

	DrainAllDelayedActionsForTest(t)

	cmd := events.InspectQueuedInputForTest(responder.InstanceId, "go ")
	if cmd == "" {
		t.Fatalf("expected a `go` command queued; nothing")
	}
	// The exit name is whatever exit in room 2 points at room 1.
	// Since room 2 has exit south → room 1, expect "go south".
	if !strings.Contains(cmd, "south") {
		t.Fatalf("expected `go south` toward caller's room; got: %q", cmd)
	}
}

// TestLookout_HeardCallforhelp_IssuesGoCommand verifies the new
// heard_callforhelp handler on lookout: it resolves the caller's
// room via ctx.Event.MobId and issues a `go <exit>` command toward the
// connecting exit.
func TestLookout_HeardCallforhelp_IssuesGoCommand(t *testing.T) {
	LoadArchetypeForTest(t, "lookout", lookoutYAML)

	room1 := &rooms.Room{RoomId: 1, Zone: "TestZone"}
	room2 := &rooms.Room{RoomId: 2, Zone: "TestZone", Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}}
	cleanupRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room1, 2: room2}, nil)
	defer cleanupRooms()

	caller := &mobs.Mob{
		InstanceId: 100,
		Character:  characters.Character{RoomId: 1, Health: 50},
	}

	responder := &mobs.Mob{
		MobId:             mobs.MobId(300 + 91002),
		InstanceId:        91002,
		BehaviorArchetype: "lookout",
	}
	responder.Character.Name = "testmob"
	responder.Character.RoomId = 2
	responder.Character.Health = 100
	responder.Character.Stamina = 100
	responder.Character.Conviction = 500
	responder.Character.Buffs = buffs.New()

	seed := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		100:   caller,
		91002: responder,
	})
	defer seed()
	defer events.DrainQueuedInputsForTest(responder.InstanceId)

	ok := TryMobBehavior(responder.InstanceId, EventContext{
		EventType: "heard_callforhelp",
		MobId:     100,
	})
	if !ok {
		t.Fatalf("TryMobBehavior(heard_callforhelp): expected Success, got false")
	}

	DrainAllDelayedActionsForTest(t)

	cmd := events.InspectQueuedInputForTest(responder.InstanceId, "go ")
	if cmd == "" {
		t.Fatalf("expected a `go` command queued; nothing")
	}
	if !strings.Contains(cmd, "south") {
		t.Fatalf("expected `go south`; got: %q", cmd)
	}
}
