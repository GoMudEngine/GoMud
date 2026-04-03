package actions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// FindExitByName calls keywords.TryDirectionAlias, which dereferences a
	// package-level pointer that is nil unless LoadAliases has been called.
	// Change to the project root so LoadAliases can find the default
	// keywords.yaml (at _datafiles/world/default/keywords.yaml).
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile = …/internal/actions/economy_test.go; walk up to project root.
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	if abs, err := filepath.Abs(projectRoot); err == nil {
		_ = os.Chdir(abs)
	}
	keywords.LoadAliases()
}

// ---------------------------------------------------------------------------
// Minimal Actor stub for economy tests
// ---------------------------------------------------------------------------

// stubActor implements the Actor interface using plain character/room pointers.
// It discards all messages — this is sufficient for the economy action tests
// which only care about inventory and gold state.
type stubActor struct {
	char *characters.Character
	room *rooms.Room
}

func newStubActor(char *characters.Character, room *rooms.Room) *stubActor {
	return &stubActor{char: char, room: room}
}

func (a *stubActor) GetCharacter() *characters.Character    { return a.char }
func (a *stubActor) GetRoom() *rooms.Room                   { return a.room }
func (a *stubActor) SendText(_ string)                      {}
func (a *stubActor) SendRoomText(_ string, _ bool)          {}
func (a *stubActor) SendRoomCommunication(_ string, _ bool) {}
func (a *stubActor) GetName() string                        { return "TestActor" }
func (a *stubActor) IsPlayer() bool                         { return false }
func (a *stubActor) GetUserId() int                         { return 0 }
func (a *stubActor) GetMobInstanceId() int                  { return 0 }
func (a *stubActor) AddBuff(_ int, _ string)                {}

// ---------------------------------------------------------------------------
// Item constructors for equip/remove tests
// ---------------------------------------------------------------------------

// newWeaponItem creates a test item with Type=Weapon so Wear() accepts it.
// Subtype is Shooting so HandsRequired() returns early before calling
// species.GetSpecies(), which panics when no species data is loaded in tests.
// Hands=1 keeps it a one-handed weapon.
func newWeaponItem(id int) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId:  id,
			Name:    "test sword",
			Type:    items.Weapon,
			Subtype: items.Shooting,
			Hands:   1,
			Weight:  0.5,
		},
	}
}

// newWeaponItemNamed is like newWeaponItem but with a distinct name so two
// weapons are still distinguishable via FindInBackpack / FindOnBody.
func newWeaponItemNamed(id int, name string) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId:  id,
			Name:    name,
			Type:    items.Weapon,
			Subtype: items.Shooting,
			Hands:   1,
			Weight:  0.5,
		},
	}
}

// countEquipped returns the number of non-zero equipment slots on a character.
func countEquipped(c *characters.Character) int {
	count := 0
	eq := c.Equipment
	for _, itm := range []items.Item{
		eq.Weapon, eq.Offhand,
		eq.Head, eq.Neck, eq.Shoulders, eq.Body, eq.Back,
		eq.Belt, eq.Wrist1, eq.Wrist2, eq.Gloves,
		eq.Ring, eq.Ring2, eq.Legs, eq.Feet,
	} {
		if itm.ItemId != 0 {
			count++
		}
	}
	return count
}

// totalItems returns the sum of backpack + equipped items + floor items.
func totalItems(c *characters.Character, r *rooms.Room) int {
	return countCharItems(c) + countEquipped(c) + countFloorItems(r)
}

// ---------------------------------------------------------------------------
// DropItem tests
// ---------------------------------------------------------------------------

// TestDropItem_Happy verifies an item moves from backpack to floor and
// the total item count is conserved.
func TestDropItem_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	itm := newTestItem(10)
	char.Items = append(char.Items, itm)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	result := DropItem(actor, "test item")

	require.True(t, result.Found, "item should be found")
	require.NoError(t, result.Err)

	afterTotal := countCharItems(char) + countFloorItems(room)

	assert.Equal(t, 0, countCharItems(char), "char should have no backpack items")
	assert.Equal(t, 1, countFloorItems(room), "floor should have 1 item")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// TestDropItem_NotFound verifies Found=false when the item is absent.
func TestDropItem_NotFound(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	result := DropItem(actor, "missing item")

	assert.False(t, result.Found, "should not find a missing item")

	afterTotal := countCharItems(char) + countFloorItems(room)
	assert.Equal(t, beforeTotal, afterTotal, "conservation: counts unchanged")
}

// ---------------------------------------------------------------------------
// RemoveEquipment tests
// ---------------------------------------------------------------------------

// TestRemoveEquipment_Happy verifies an equipped weapon moves to the backpack.
func TestRemoveEquipment_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	sword := newWeaponItem(20)
	char.Equipment.Weapon = sword

	beforeEquipped := countEquipped(char)
	beforeBackpack := countCharItems(char)

	result := RemoveEquipment(actor, "test sword")

	require.True(t, result.Found, "item should be found on body")
	assert.True(t, result.Removed, "item should be removed")

	assert.Equal(t, beforeEquipped-1, countEquipped(char), "one fewer equipped item")
	assert.Equal(t, beforeBackpack+1, countCharItems(char), "one more backpack item")
	assert.Equal(t, items.Item{}, char.Equipment.Weapon, "weapon slot should be empty")
}

// TestRemoveEquipment_NotFound verifies Found=false when nothing matches.
func TestRemoveEquipment_NotFound(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := RemoveEquipment(actor, "invisible blade")

	assert.False(t, result.Found, "should not find unequipped item")
	assert.Equal(t, 0, countEquipped(char), "no equipped items changed")
}

// ---------------------------------------------------------------------------
// EquipItem tests
// ---------------------------------------------------------------------------

// TestEquipItem_Happy verifies a backpack item moves to the weapon slot.
func TestEquipItem_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	sword := newWeaponItem(30)
	char.Items = append(char.Items, sword)

	beforeBackpack := countCharItems(char)

	result := EquipItem(actor, "test sword")

	require.True(t, result.Found, "item should be found in backpack")
	assert.True(t, result.Equipped, "item should be equipped")

	assert.Equal(t, beforeBackpack-1, countCharItems(char), "one fewer backpack item")
	assert.Equal(t, 1, countEquipped(char), "one item now equipped")
}

// TestEquipItem_NotFound verifies Found=false when item absent from backpack.
func TestEquipItem_NotFound(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := EquipItem(actor, "ghost weapon")

	assert.False(t, result.Found, "should not find a missing item")
	assert.Equal(t, 0, countEquipped(char), "nothing equipped")
}

// TestEquipItem_Displaces verifies that equipping when both weapon and offhand
// slots are occupied displaces the old main weapon to the backpack and the
// total item count is conserved. New characters always have WeaponCombat
// skill >= 1 (dual-wield enabled), so we pre-fill both hands before equipping
// a third sword to force a true displacement.
func TestEquipItem_Displaces(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	mainSword := newWeaponItemNamed(31, "main sword")
	offSword := newWeaponItemNamed(32, "off sword")
	newSword := newWeaponItemNamed(33, "new sword")

	// Fill both hands so the next equip must displace.
	char.Equipment.Weapon = mainSword
	char.Equipment.Offhand = offSword
	char.Items = append(char.Items, newSword)

	beforeTotal := totalItems(char, room)

	result := EquipItem(actor, "new sword")

	require.True(t, result.Found, "new sword should be found")
	assert.True(t, result.Equipped, "new sword should be equipped")

	afterTotal := totalItems(char, room)

	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
	// The new sword must be present somewhere on the body.
	eq := char.Equipment
	equippedIds := map[int]bool{
		eq.Weapon.ItemId:  true,
		eq.Offhand.ItemId: true,
	}
	assert.True(t, equippedIds[newSword.ItemId], "new sword should be on the body")
}

// ---------------------------------------------------------------------------
// GetItemFromFloor tests
// ---------------------------------------------------------------------------

// TestGetItemFromFloor_Happy verifies a floor item moves to the backpack.
func TestGetItemFromFloor_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	itm := newTestItem(40)
	room.Items = append(room.Items, itm)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	result := GetItemFromFloor(actor, "test item", false)

	require.True(t, result.Found, "item should be found on floor")
	require.NoError(t, result.Err)

	afterTotal := countCharItems(char) + countFloorItems(room)

	assert.Equal(t, 1, countCharItems(char), "char should have 1 item")
	assert.Equal(t, 0, countFloorItems(room), "floor should be empty")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// TestGetItemFromFloor_NotFound verifies Found=false when floor is empty.
func TestGetItemFromFloor_NotFound(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	result := GetItemFromFloor(actor, "phantom item", false)

	assert.False(t, result.Found, "should not find item on empty floor")

	afterTotal := countCharItems(char) + countFloorItems(room)
	assert.Equal(t, beforeTotal, afterTotal, "conservation: counts unchanged")
}

// ---------------------------------------------------------------------------
// GetGoldFromFloor tests
// ---------------------------------------------------------------------------

// TestGetGoldFromFloor_Happy verifies gold moves from floor to character wallet.
func TestGetGoldFromFloor_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	actor := newStubActor(char, room)

	char.Gold = 0
	room.Gold = 75

	total := totalGold(char, room)

	err := GetGoldFromFloor(actor, 75)
	require.NoError(t, err)

	assert.Equal(t, 75, char.Gold, "char should have 75 gold")
	assert.Equal(t, 0, room.Gold, "floor should have 0 gold")
	assert.Equal(t, total, totalGold(char, room), "conservation: total gold unchanged")
}

// ---------------------------------------------------------------------------
// GiveItemToChar tests
// ---------------------------------------------------------------------------

// TestGiveItemToChar_Happy verifies an item moves from source to destination
// and the combined item count is conserved.
func TestGiveItemToChar_Happy(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()
	room := newTestRoom()
	actor := newStubActor(charA, room)

	itm := newTestItem(50)
	charA.Items = append(charA.Items, itm)

	beforeTotal := countCharItems(charA) + countCharItems(charB)

	result := GiveItemToChar(actor, "test item", charB, 0, 0)

	require.True(t, result.Found, "item should be found")
	require.NoError(t, result.Err)

	afterTotal := countCharItems(charA) + countCharItems(charB)

	assert.Equal(t, 0, countCharItems(charA), "source should have no items")
	assert.Equal(t, 1, countCharItems(charB), "destination should have 1 item")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// TestGiveItemToChar_NotFound verifies Found=false when the item is absent.
func TestGiveItemToChar_NotFound(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()
	room := newTestRoom()
	actor := newStubActor(charA, room)

	beforeTotal := countCharItems(charA) + countCharItems(charB)

	result := GiveItemToChar(actor, "missing item", charB, 0, 0)

	assert.False(t, result.Found, "should not find missing item")

	afterTotal := countCharItems(charA) + countCharItems(charB)
	assert.Equal(t, beforeTotal, afterTotal, "conservation: counts unchanged")
}

// ---------------------------------------------------------------------------
// GiveGoldToChar tests
// ---------------------------------------------------------------------------

// TestGiveGoldToChar_Happy verifies gold moves from actor to target and total
// is conserved.
func TestGiveGoldToChar_Happy(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()
	room := newTestRoom()
	actor := newStubActor(charA, room)

	charA.Gold = 200
	charB.Gold = 0

	total := charA.Gold + charB.Gold

	err := GiveGoldToChar(actor, 100, charB)
	require.NoError(t, err)

	assert.Equal(t, 100, charA.Gold, "source should have 100 remaining")
	assert.Equal(t, 100, charB.Gold, "destination should have 100")
	assert.Equal(t, total, charA.Gold+charB.Gold, "conservation: total gold unchanged")
}

// TestGiveGoldToChar_Insufficient verifies an error when source lacks funds
// and balances are unchanged.
func TestGiveGoldToChar_Insufficient(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()
	room := newTestRoom()
	actor := newStubActor(charA, room)

	charA.Gold = 50
	charB.Gold = 0

	total := charA.Gold + charB.Gold

	err := GiveGoldToChar(actor, 200, charB)
	assert.Error(t, err, "should error on insufficient gold")
	assert.Equal(t, 50, charA.Gold, "source gold unchanged")
	assert.Equal(t, 0, charB.Gold, "destination gold unchanged")
	assert.Equal(t, total, charA.Gold+charB.Gold, "conservation: total gold unchanged")
}

// ---------------------------------------------------------------------------
// FindExit tests
// ---------------------------------------------------------------------------

// TestFindExit_Found verifies that a room with a known exit returns Found=true
// and the correct target RoomId.
func TestFindExit_Found(t *testing.T) {
	room := &rooms.Room{
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 42},
		},
	}

	result := FindExit(room, "north")

	assert.True(t, result.Found, "north exit should be found")
	assert.Equal(t, 42, result.RoomId, "exit should lead to room 42")
	assert.Equal(t, "north", result.ExitName, "exit name should be 'north'")
}

// TestFindExit_NotFound verifies that searching for a non-existent exit
// returns Found=false.
func TestFindExit_NotFound(t *testing.T) {
	room := &rooms.Room{
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 1},
		},
	}

	result := FindExit(room, "south")

	assert.False(t, result.Found, "south exit should not be found")
	assert.Equal(t, 0, result.RoomId, "RoomId should be zero on not-found")
}
