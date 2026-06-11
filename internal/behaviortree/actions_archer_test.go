package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// rangedBow builds a shooting-subtype weapon Item with the given loaded state.
func rangedBow(loaded bool) items.Item {
	return items.Item{
		ItemId: 1,
		Loaded: loaded,
		Spec: &items.ItemSpec{
			ItemId:  1,
			Type:    items.Weapon,
			Subtype: items.Shooting,
			AmmoTag: "arrows",
		},
	}
}

// arrowBundle builds an ammo bundle Item matching rangedBow.
func arrowBundle() items.Item {
	return items.Item{
		ItemId: 2,
		Uses:   10,
		Spec: &items.ItemSpec{
			ItemId:  2,
			Type:    items.Ammo,
			AmmoTag: "arrows",
		},
	}
}

// seedTargetMob registers a target mob in the given room and arranges cleanup.
func seedTargetMob(t *testing.T, instanceId, roomId int) *mobs.Mob {
	t.Helper()
	target := &mobs.Mob{MobId: 2, InstanceId: instanceId}
	target.Character.Name = "Target"
	target.Character.RoomId = roomId
	target.Character.HealthMax.Value = 100
	target.Character.Health = 100
	mobs.SetInstanceForTest(instanceId, target)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })
	return target
}

func queuedCmds(instanceId int) []string {
	return events.DrainQueuedInputsForTest(instanceId)
}

func contains(cmds []string, want string) bool {
	for _, c := range cmds {
		if c == want {
			return true
		}
	}
	return false
}

// ─── registration ────────────────────────────────────────────────────────────

func TestArcherActions_Registered(t *testing.T) {
	for _, name := range []string{"try_fire", "try_reload", "keep_distance"} {
		if LookupAction(name) == nil {
			t.Errorf("%s must be registered in actionRegistry", name)
		}
	}
}

// ─── try_fire ─────────────────────────────────────────────────────────────────

func TestActTryFire_SameRoomLoaded_FiresAndSucceeds(t *testing.T) {
	const room = 10
	target := seedTargetMob(t, 401, room)

	mob := newTestMob(t)
	mob.Character.RoomId = room
	mob.Character.Equipment.Weapon = rangedBow(true)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
	queuedCmds(mob.InstanceId) // clear

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: room}
	if got := LookupAction("try_fire")(nil, ctx); got != Success {
		t.Fatalf("try_fire = %v, want Success", got)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "shoot Target") {
		t.Errorf("expected 'shoot Target' queued, got %v", cmds)
	}
}

func TestActTryFire_Unloaded_Fails(t *testing.T) {
	const room = 10
	target := seedTargetMob(t, 402, room)

	mob := newTestMob(t)
	mob.Character.RoomId = room
	mob.Character.Equipment.Weapon = rangedBow(false) // unloaded
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: room}
	if got := LookupAction("try_fire")(nil, ctx); got != Failure {
		t.Errorf("try_fire with unloaded weapon = %v, want Failure", got)
	}
}

func TestActTryFire_NoAggro_Fails(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.RoomId = 10
	mob.Character.Equipment.Weapon = rangedBow(true)
	mob.Character.Aggro = nil

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: 10}
	if got := LookupAction("try_fire")(nil, ctx); got != Failure {
		t.Errorf("try_fire with no aggro = %v, want Failure", got)
	}
}

func TestActTryFire_CrossRoomAdjacent_FiresDirectional(t *testing.T) {
	const here, there = 10, 11
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{
			"north": {RoomId: there},
		}},
		there: {RoomId: there, Zone: "test", Exits: map[string]exit.RoomExit{
			"south": {RoomId: here},
		}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	target := seedTargetMob(t, 403, there) // one exit (north) away

	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.Equipment.Weapon = rangedBow(true)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
	queuedCmds(mob.InstanceId)

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: here}
	if got := LookupAction("try_fire")(nil, ctx); got != Success {
		t.Fatalf("try_fire cross-room = %v, want Success", got)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "shoot Target north") {
		t.Errorf("expected 'shoot Target north' queued, got %v", cmds)
	}
}

// ─── try_reload ───────────────────────────────────────────────────────────────

func TestActTryReload_UnloadedWithAmmoNotEngaged_ReloadsAndSucceeds(t *testing.T) {
	const here = 10
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.Equipment.Weapon = rangedBow(false)
	mob.Character.Items = []items.Item{arrowBundle()}
	mob.Character.Aggro = nil // not engaged
	queuedCmds(mob.InstanceId)

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: here}
	if got := LookupAction("try_reload")(nil, ctx); got != Success {
		t.Fatalf("try_reload = %v, want Success", got)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "reload") {
		t.Errorf("expected 'reload' queued, got %v", cmds)
	}
}

func TestActTryReload_AlreadyLoaded_Fails(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.RoomId = 10
	mob.Character.Equipment.Weapon = rangedBow(true) // loaded
	mob.Character.Items = []items.Item{arrowBundle()}

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: 10}
	if got := LookupAction("try_reload")(nil, ctx); got != Failure {
		t.Errorf("try_reload with loaded weapon = %v, want Failure", got)
	}
}

func TestActTryReload_NoAmmo_Fails(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.RoomId = 10
	mob.Character.Equipment.Weapon = rangedBow(false)
	mob.Character.Items = nil // no ammo

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: 10}
	if got := LookupAction("try_reload")(nil, ctx); got != Failure {
		t.Errorf("try_reload without ammo = %v, want Failure", got)
	}
}

func TestActTryReload_MeleeEngaged_Fails(t *testing.T) {
	const here = 10
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	target := seedTargetMob(t, 404, here) // in the same room == engaged

	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.Equipment.Weapon = rangedBow(false)
	mob.Character.Items = []items.Item{arrowBundle()}
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: here}
	if got := LookupAction("try_reload")(nil, ctx); got != Failure {
		t.Errorf("try_reload while melee-engaged = %v, want Failure", got)
	}
}

// ─── keep_distance ────────────────────────────────────────────────────────────

func TestActKeepDistance_EngagedAndHealthy_KitesAndBreadcrumbs(t *testing.T) {
	const here, there = 10, 11
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{
			"north": {RoomId: there},
		}},
		there: {RoomId: there, Zone: "test", Exits: map[string]exit.RoomExit{
			"south": {RoomId: here},
		}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	target := seedTargetMob(t, 405, here) // closed into melee

	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 100 // healthy
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
	queuedCmds(mob.InstanceId)

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: here}
	if got := LookupAction("keep_distance")(nil, ctx); got != Success {
		t.Fatalf("keep_distance = %v, want Success", got)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "go north") {
		t.Errorf("expected 'go north' queued, got %v", cmds)
	}
	if from := getMiscDataIntForager(mob, keyArcherFallbackFrom); from != here {
		t.Errorf("breadcrumb archer_fallback_from = %d, want %d", from, here)
	}
}

func TestActKeepDistance_NotEngaged_Fails(t *testing.T) {
	const here = 10
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{
			"north": {RoomId: 11},
		}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 100
	mob.Character.Aggro = nil // no one closed on us

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: here}
	if got := LookupAction("keep_distance")(nil, ctx); got != Failure {
		t.Errorf("keep_distance not engaged = %v, want Failure", got)
	}
}

func TestActKeepDistance_Hurt_Fails(t *testing.T) {
	const here = 10
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{
			"north": {RoomId: 11},
		}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	target := seedTargetMob(t, 406, here)

	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.HealthMax.Value = 100
	mob.Character.Health = 20 // below default 50% threshold
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId, RoomId: here}
	if got := LookupAction("keep_distance")(nil, ctx); got != Failure {
		t.Errorf("keep_distance while hurt = %v, want Failure", got)
	}
}
