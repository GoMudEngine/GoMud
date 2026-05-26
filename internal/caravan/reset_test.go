package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// seedResetFixture builds:
//   - wp0 room (the Thornwall depot, room id 465 to match the canonical patrol)
//   - a "mid-cycle" room where the leader is currently stranded
//   - a stranded wagon room
//   - a registered test patrol with wp0 dwell = 360
//
// Returns leader, wagon, and a cleanup func.
func seedResetFixture(t *testing.T) (*mobs.Mob, *mobs.Mob, func()) {
	t.Helper()

	const wp0Room = 8000
	const midCycleRoom = 8001
	const wagonStrandedRoom = 8002

	depot := &rooms.Room{RoomId: wp0Room, Zone: "TestZone", Title: "Depot", Exits: map[string]exit.RoomExit{}}
	mid := &rooms.Room{RoomId: midCycleRoom, Zone: "TestZone", Title: "Mid Cycle", Exits: map[string]exit.RoomExit{}}
	stranded := &rooms.Room{RoomId: wagonStrandedRoom, Zone: "TestZone", Title: "Stranded", Exits: map[string]exit.RoomExit{}}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{
			wp0Room:           depot,
			midCycleRoom:      mid,
			wagonStrandedRoom: stranded,
		},
		map[string]*rooms.ZoneConfig{},
	)

	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        CaravanPatrolId,
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: wp0Room, DwellRounds: 360, ArrivalEvent: "caravan_depot"},
			// One additional waypoint is enough — the helper only reads wp0.
			{Room: midCycleRoom, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
		},
	})

	const leaderInstId = 81357
	const wagonInstId = 81374
	leader := &mobs.Mob{
		MobId:      mobs.MobId(LeaderMobId),
		InstanceId: leaderInstId,
		HomeRoomId: wp0Room,
		Zone:       "TestZone",
		PatrolId:   CaravanPatrolId,
	}
	leader.Character.Name = "Ketil"
	leader.Character.RoomId = midCycleRoom // start mid-cycle, not at depot
	leader.Character.Buffs = buffs.New()
	mid.AddMob(leader.InstanceId)

	wagon := &mobs.Mob{
		MobId:      mobs.MobId(WagonMobId),
		InstanceId: wagonInstId,
		HomeRoomId: wp0Room,
		Zone:       "TestZone",
	}
	wagon.Character.Name = "caravan wagon"
	wagon.Character.RoomId = wagonStrandedRoom
	wagon.Character.Buffs = buffs.New()
	stranded.AddMob(wagon.InstanceId)

	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{LeaderMobId: leader, WagonMobId: wagon},
		map[int]*mobs.Mob{leaderInstId: leader, wagonInstId: wagon},
	)

	return leader, wagon, func() {
		cleanMobs()
		cleanRooms()
		mobs.UnregisterPatrolForTest(CaravanPatrolId)
	}
}

func TestResetLeaderToDepot_TeleportsLeaderToWp0(t *testing.T) {
	leader, _, cleanup := seedResetFixture(t)
	defer cleanup()

	if leader.Character.RoomId == 8000 {
		t.Fatalf("precondition: leader should start mid-cycle (not at wp0)")
	}

	if ok := ResetLeaderToDepot(leader); !ok {
		t.Fatal("ResetLeaderToDepot returned false")
	}

	if leader.Character.RoomId != 8000 {
		t.Errorf("expected leader at wp0 room 8000, got %d", leader.Character.RoomId)
	}
}

func TestResetLeaderToDepot_RestoresWp0Dwell(t *testing.T) {
	leader, _, cleanup := seedResetFixture(t)
	defer cleanup()

	// Mid-cycle, dwell_remaining was something else (or unset).
	leader.Character.SetMiscData("patrol_dwell_remaining", 3)

	if ok := ResetLeaderToDepot(leader); !ok {
		t.Fatal("ResetLeaderToDepot returned false")
	}

	dwell, _ := leader.Character.GetMiscData("patrol_dwell_remaining").(int)
	if dwell != 360 {
		t.Errorf("expected patrol_dwell_remaining = 360 (wp0 authored value), got %d", dwell)
	}
}

func TestResetLeaderToDepot_RegroupsCrew(t *testing.T) {
	leader, wagon, cleanup := seedResetFixture(t)
	defer cleanup()

	if wagon.Character.RoomId == leader.Character.RoomId {
		t.Fatalf("precondition: wagon should be stranded elsewhere from leader")
	}

	if ok := ResetLeaderToDepot(leader); !ok {
		t.Fatal("ResetLeaderToDepot returned false")
	}

	// After reset, both leader and wagon should be at wp0 (8000).
	if leader.Character.RoomId != 8000 {
		t.Errorf("leader should be at 8000, got %d", leader.Character.RoomId)
	}
	if wagon.Character.RoomId != 8000 {
		t.Errorf("wagon should be regrouped to 8000, got %d", wagon.Character.RoomId)
	}
}

func TestResetLeaderToDepot_ResetsAllPatrolMiscData(t *testing.T) {
	leader, _, cleanup := seedResetFixture(t)
	defer cleanup()

	// Seed messy mid-cycle state on the leader's MiscData.
	leader.Character.SetMiscData("patrol_waypoint_idx", 7)
	leader.Character.SetMiscData("patrol_direction", -1)
	leader.Character.SetMiscData("patrol_path_fail_count", 19)

	if ok := ResetLeaderToDepot(leader); !ok {
		t.Fatal("ResetLeaderToDepot returned false")
	}

	idx, _ := leader.Character.GetMiscData("patrol_waypoint_idx").(int)
	if idx != 0 {
		t.Errorf("patrol_waypoint_idx = %d, want 0", idx)
	}
	dir, _ := leader.Character.GetMiscData("patrol_direction").(int)
	if dir != 1 {
		t.Errorf("patrol_direction = %d, want 1", dir)
	}
	fails, _ := leader.Character.GetMiscData("patrol_path_fail_count").(int)
	if fails != 0 {
		t.Errorf("patrol_path_fail_count = %d, want 0", fails)
	}
}

func TestResetLeaderToDepot_NonCaravanMobReturnsFalse(t *testing.T) {
	leader, _, cleanup := seedResetFixture(t)
	defer cleanup()

	leader.PatrolId = "some_other_patrol"
	if ok := ResetLeaderToDepot(leader); ok {
		t.Errorf("expected ResetLeaderToDepot to return false for non-caravan mob")
	}
}

func TestResetLeaderToDepot_NilMobReturnsFalse(t *testing.T) {
	if ok := ResetLeaderToDepot(nil); ok {
		t.Errorf("expected ResetLeaderToDepot to return false for nil mob")
	}
}
