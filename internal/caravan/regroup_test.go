package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// seedLeaderAndStrandedWagon builds a 2-room test fixture:
//   - leader (mob 357 / instance 80357) at room A
//   - wagon  (mob 374 / instance 80374) at room B (stranded)
//
// Returns a cleanup func that must be deferred.
func seedLeaderAndStrandedWagon(t *testing.T, roomA, roomB int) (*mobs.Mob, *mobs.Mob, func()) {
	t.Helper()

	ra := &rooms.Room{RoomId: roomA, Zone: "TestZone", Title: "Room A", Exits: map[string]exit.RoomExit{}}
	rb := &rooms.Room{RoomId: roomB, Zone: "TestZone", Title: "Room B", Exits: map[string]exit.RoomExit{}}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomA: ra, roomB: rb},
		map[string]*rooms.ZoneConfig{},
	)

	const leaderInstId = 80357
	const wagonInstId = 80374
	leader := &mobs.Mob{
		MobId:      mobs.MobId(LeaderMobId), // 357
		InstanceId: leaderInstId,
		HomeRoomId: roomA,
		Zone:       "TestZone",
		PatrolId:   CaravanPatrolId,
	}
	leader.Character.Name = "Ketil"
	leader.Character.RoomId = roomA
	leader.Character.Buffs = buffs.New()
	ra.AddMob(leader.InstanceId)

	wagon := &mobs.Mob{
		MobId:      mobs.MobId(WagonMobId), // 374
		InstanceId: wagonInstId,
		HomeRoomId: roomA,
		Zone:       "TestZone",
	}
	wagon.Character.Name = "caravan wagon"
	wagon.Character.RoomId = roomB
	wagon.Character.Buffs = buffs.New()
	rb.AddMob(wagon.InstanceId)

	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{
			LeaderMobId: leader,
			WagonMobId:  wagon,
		},
		map[int]*mobs.Mob{
			leaderInstId: leader,
			wagonInstId:  wagon,
		},
	)

	return leader, wagon, func() {
		cleanMobs()
		cleanRooms()
	}
}

func TestForceRegroupCrew_MovesStrandedWagon(t *testing.T) {
	leader, wagon, cleanup := seedLeaderAndStrandedWagon(t, 7001, 7002)
	defer cleanup()

	if wagon.Character.RoomId != 7002 {
		t.Fatalf("precondition: wagon should start at room 7002, got %d", wagon.Character.RoomId)
	}

	ForceRegroupCrew(leader)

	if wagon.Character.RoomId != 7001 {
		t.Errorf("after regroup, wagon.RoomId = %d, want 7001 (leader's room)", wagon.Character.RoomId)
	}
}

func TestForceRegroupCrew_SkipsAlreadyCoLocated(t *testing.T) {
	leader, wagon, cleanup := seedLeaderAndStrandedWagon(t, 7003, 7004)
	defer cleanup()

	// Manually colocate the wagon with the leader before regrouping.
	wagon.Character.RoomId = 7003

	// Call should be a no-op (idempotent).
	ForceRegroupCrew(leader)

	if wagon.Character.RoomId != 7003 {
		t.Errorf("wagon should remain at 7003 (already co-located), got %d", wagon.Character.RoomId)
	}
}

func TestForceRegroupCrew_NilLeaderNoOps(t *testing.T) {
	// Just verify no panic.
	ForceRegroupCrew(nil)
}

func TestHandleDepotArrival_FreshRespawnMarkerTriggersRegroup(t *testing.T) {
	leader, wagon, cleanup := seedLeaderAndStrandedWagon(t, 7005, 7006)
	defer cleanup()

	leader.Character.SetMiscData("patrol_fresh_respawn", true)

	arrival := events.PatrolWaypointArrival{
		MobInstanceId: leader.InstanceId,
		PatrolId:      CaravanPatrolId,
		WaypointIdx:   0,
		RoomId:        leader.Character.RoomId,
		ArrivalEvent:  "caravan_depot",
	}
	handleDepotArrival(leader, arrival)

	if wagon.Character.RoomId != 7005 {
		t.Errorf("expected wagon to be regrouped to 7005, got %d", wagon.Character.RoomId)
	}
	cleared, _ := leader.Character.GetMiscData("patrol_fresh_respawn").(bool)
	if cleared {
		t.Errorf("expected patrol_fresh_respawn to be cleared after regroup, still true")
	}
}

func TestHandleDepotArrival_NoMarkerSkipsRegroup(t *testing.T) {
	leader, wagon, cleanup := seedLeaderAndStrandedWagon(t, 7007, 7008)
	defer cleanup()

	// No patrol_fresh_respawn marker set.

	arrival := events.PatrolWaypointArrival{
		MobInstanceId: leader.InstanceId,
		PatrolId:      CaravanPatrolId,
		WaypointIdx:   0,
		RoomId:        leader.Character.RoomId,
		ArrivalEvent:  "caravan_depot",
	}
	handleDepotArrival(leader, arrival)

	if wagon.Character.RoomId != 7008 {
		t.Errorf("expected wagon to stay at 7008 (no marker), got %d", wagon.Character.RoomId)
	}
}

func TestHandleDepotArrival_NonZeroWaypointNoOps(t *testing.T) {
	leader, wagon, cleanup := seedLeaderAndStrandedWagon(t, 7009, 7010)
	defer cleanup()

	// Marker set, but waypoint isn't 0 — should still no-op.
	leader.Character.SetMiscData("patrol_fresh_respawn", true)

	arrival := events.PatrolWaypointArrival{
		MobInstanceId: leader.InstanceId,
		PatrolId:      CaravanPatrolId,
		WaypointIdx:   11, // Stillwater depot, not Thornwall start
		RoomId:        leader.Character.RoomId,
		ArrivalEvent:  "caravan_depot",
	}
	handleDepotArrival(leader, arrival)

	if wagon.Character.RoomId != 7010 {
		t.Errorf("expected wagon to stay at 7010 (non-zero waypoint), got %d", wagon.Character.RoomId)
	}
	// Marker should also remain set, since wp0 didn't fire.
	stillSet, _ := leader.Character.GetMiscData("patrol_fresh_respawn").(bool)
	if !stillSet {
		t.Errorf("expected patrol_fresh_respawn to remain true (wp11 didn't consume it)")
	}
}
