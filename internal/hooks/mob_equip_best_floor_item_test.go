package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestEquipBestFloorItem_NilRoom(t *testing.T) {
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	if EquipBestFloorItem(m, nil) {
		t.Errorf("expected false for nil room")
	}
}

func TestEquipBestFloorItem_EmptyRoom(t *testing.T) {
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	r := &rooms.Room{RoomId: 99999}
	if EquipBestFloorItem(m, r) {
		t.Errorf("expected false for empty room")
	}
}

func TestEquipBestFloorItem_NonEligibleMob(t *testing.T) {
	// Non-combat archetype mob → CanScanFloorLoot false → skip.
	m := &mobs.Mob{BehaviorArchetype: "noncombat_shopkeeper"}
	m.Character = characters.Character{}
	r := &rooms.Room{RoomId: 99999}
	r.Items = []items.Item{{ItemId: 1}}
	if EquipBestFloorItem(m, r) {
		t.Errorf("expected false for non-eligible mob")
	}
}

func TestEquipBestFloorItem_InCombatSkips(t *testing.T) {
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	// Set up an aggro to indicate in-combat state.
	m.Character.Aggro = &characters.Aggro{UserId: 1}
	r := &rooms.Room{RoomId: 99999}
	r.Items = []items.Item{{ItemId: 1}}
	if EquipBestFloorItem(m, r) {
		t.Errorf("expected false for in-combat mob")
	}
}

func TestEquipBestFloorItem_RealSwap_FixtureLimited(t *testing.T) {
	t.Skip("requires loaded item specs to score IsUpgrade meaningfully; covered by Task 6 smoke")
}
