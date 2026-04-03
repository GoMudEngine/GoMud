package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Counting helpers
// ---------------------------------------------------------------------------

func countCharItems(c *characters.Character) int {
	return len(c.Items) + len(c.ComponentItems) + len(c.PotionItems)
}

func countFloorItems(r *rooms.Room) int {
	return len(r.Items)
}

func totalGold(c *characters.Character, r *rooms.Room) int {
	return c.Gold + r.Gold
}

// newTestItem creates a lightweight test item with a given ID.
// The Spec field is set inline so no data files are required.
func newTestItem(id int) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId: id,
			Name:   "test item",
			Weight: 0.1, // very light — won't trigger capacity limits
		},
	}
}

// newHeavyTestItem creates a test item whose weight exceeds any reasonable
// carry capacity when Strength is set to 1.
//
// With Strength=1 and the default CarryCapacityMultiplier (0.65):
//   capacity = 1 * 0.65 = 0.65
//   StoreItem blocks when newWeight > capacity * 2.0 = 1.3
//
// A single 2-pound item therefore triggers the capacity check.
func newHeavyTestItem(id int) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId: id,
			Name:   "heavy item",
			Weight: 2.0,
		},
	}
}

// newTestChar creates a minimal character with default stats.
func newTestChar() *characters.Character {
	c := characters.New()
	return c
}

// newWeakChar creates a character with Strength 1 so that a heavy item
// immediately exceeds carry capacity.
func newWeakChar() *characters.Character {
	c := characters.New()
	c.Stats.Strength.ValueAdj = 1
	return c
}

// newTestRoom creates a minimal room suitable for item / gold operations.
func newTestRoom() *rooms.Room {
	return &rooms.Room{}
}

// ---------------------------------------------------------------------------
// TransferItemToFloor
// ---------------------------------------------------------------------------

// TestTransferItemToFloor_Happy verifies that an item in the character's
// backpack moves to the room floor and that the total item count is conserved.
func TestTransferItemToFloor_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()

	itm := newTestItem(1)
	char.Items = append(char.Items, itm)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	err := TransferItemToFloor(itm, char, 1, 0, room)
	require.NoError(t, err)

	afterTotal := countCharItems(char) + countFloorItems(room)

	assert.Equal(t, 0, countCharItems(char), "char should have no items")
	assert.Equal(t, 1, countFloorItems(room), "floor should have 1 item")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// TestTransferItemToFloor_NotFound verifies that transferring an item the
// character does not own returns an error and leaves counts unchanged.
func TestTransferItemToFloor_NotFound(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()

	itm := newTestItem(99)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	err := TransferItemToFloor(itm, char, 1, 0, room)
	assert.Error(t, err, "should error when item not in inventory")

	afterTotal := countCharItems(char) + countFloorItems(room)
	assert.Equal(t, beforeTotal, afterTotal, "conservation: counts unchanged on failure")
}

// ---------------------------------------------------------------------------
// TransferItemToBackpack
// ---------------------------------------------------------------------------

// TestTransferItemToBackpack_Happy verifies that an item on the room floor
// moves into the character's backpack and that the total item count is conserved.
func TestTransferItemToBackpack_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()

	itm := newTestItem(2)
	room.Items = append(room.Items, itm)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	removeFrom := func(i items.Item) {
		room.RemoveItem(i, false)
	}
	rollback := func(i items.Item) {
		room.AddItem(i, false)
	}

	err := TransferItemToBackpack(itm, char, 1, 0, removeFrom, rollback)
	require.NoError(t, err)

	afterTotal := countCharItems(char) + countFloorItems(room)

	assert.Equal(t, 1, countCharItems(char), "char should have 1 item")
	assert.Equal(t, 0, countFloorItems(room), "floor should be empty")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// TestTransferItemToBackpack_CapacityFull verifies that when the character
// cannot store the item the rollback fires, the item returns to the floor,
// and the total item count is conserved.
func TestTransferItemToBackpack_CapacityFull(t *testing.T) {
	char := newWeakChar()
	room := newTestRoom()

	itm := newHeavyTestItem(3)
	room.Items = append(room.Items, itm)

	beforeTotal := countCharItems(char) + countFloorItems(room)

	rollbackCalled := false
	removeFrom := func(i items.Item) {
		room.RemoveItem(i, false)
	}
	rollback := func(i items.Item) {
		rollbackCalled = true
		room.AddItem(i, false)
	}

	err := TransferItemToBackpack(itm, char, 1, 0, removeFrom, rollback)
	assert.Error(t, err, "should error when capacity exceeded")
	assert.True(t, rollbackCalled, "rollback must be called on failure")

	afterTotal := countCharItems(char) + countFloorItems(room)

	assert.Equal(t, 0, countCharItems(char), "char should have no items after rollback")
	assert.Equal(t, 1, countFloorItems(room), "item should be back on floor after rollback")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// ---------------------------------------------------------------------------
// TransferItemBetweenChars
// ---------------------------------------------------------------------------

// TestTransferItemBetweenChars_Happy verifies that an item moves from one
// character to another and the combined item count is conserved.
func TestTransferItemBetweenChars_Happy(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()

	itm := newTestItem(4)
	charA.Items = append(charA.Items, itm)

	beforeTotal := countCharItems(charA) + countCharItems(charB)

	err := TransferItemBetweenChars(itm, charA, 1, 0, charB, 2, 0)
	require.NoError(t, err)

	afterTotal := countCharItems(charA) + countCharItems(charB)

	assert.Equal(t, 0, countCharItems(charA), "source should have no items")
	assert.Equal(t, 1, countCharItems(charB), "destination should have 1 item")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// TestTransferItemBetweenChars_NotFound verifies that an error is returned
// and counts are unchanged when the item is not in the source's inventory.
func TestTransferItemBetweenChars_NotFound(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()

	itm := newTestItem(99)

	beforeTotal := countCharItems(charA) + countCharItems(charB)

	err := TransferItemBetweenChars(itm, charA, 1, 0, charB, 2, 0)
	assert.Error(t, err, "should error when item not in source")

	afterTotal := countCharItems(charA) + countCharItems(charB)
	assert.Equal(t, beforeTotal, afterTotal, "conservation: counts unchanged")
}

// TestTransferItemBetweenChars_DestFull verifies that when the destination
// cannot accept the item it is rolled back to the source and the combined
// item count is conserved.
func TestTransferItemBetweenChars_DestFull(t *testing.T) {
	charA := newTestChar()
	charB := newWeakChar() // weak char cannot carry a heavy item

	itm := newHeavyTestItem(5)
	charA.Items = append(charA.Items, itm)

	beforeTotal := countCharItems(charA) + countCharItems(charB)

	err := TransferItemBetweenChars(itm, charA, 1, 0, charB, 2, 0)
	assert.Error(t, err, "should error when destination is full")

	afterTotal := countCharItems(charA) + countCharItems(charB)

	assert.Equal(t, 1, countCharItems(charA), "item should be rolled back to source")
	assert.Equal(t, 0, countCharItems(charB), "destination should still be empty")
	assert.Equal(t, beforeTotal, afterTotal, "conservation: total items unchanged")
}

// ---------------------------------------------------------------------------
// TransferGold
// ---------------------------------------------------------------------------

// TestTransferGold_Happy verifies that gold moves from one character to
// another and the combined total is conserved.
func TestTransferGold_Happy(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()
	// Override the default starting gold to set known, controlled values.
	charA.Gold = 100
	charB.Gold = 0

	total := charA.Gold + charB.Gold

	err := TransferGold(50, charA, charB)
	require.NoError(t, err)

	assert.Equal(t, 50, charA.Gold, "source should have 50 gold remaining")
	assert.Equal(t, 50, charB.Gold, "destination should have 50 gold")
	assert.Equal(t, total, charA.Gold+charB.Gold, "conservation: total gold unchanged")
}

// TestTransferGold_Insufficient verifies that an error is returned when the
// source does not have enough gold, and balances are unchanged.
func TestTransferGold_Insufficient(t *testing.T) {
	charA := newTestChar()
	charB := newTestChar()
	// Override the default starting gold to set known, controlled values.
	charA.Gold = 100
	charB.Gold = 0

	total := charA.Gold + charB.Gold

	err := TransferGold(200, charA, charB)
	assert.Error(t, err, "should error on insufficient gold")
	assert.Equal(t, 100, charA.Gold, "source gold unchanged")
	assert.Equal(t, 0, charB.Gold, "destination gold unchanged")
	assert.Equal(t, total, charA.Gold+charB.Gold, "conservation: total gold unchanged")
}

// ---------------------------------------------------------------------------
// FloorPickupGold
// ---------------------------------------------------------------------------

// TestFloorPickupGold_Happy verifies that gold moves from the floor to the
// character and the combined total is conserved.
func TestFloorPickupGold_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	// Override the default starting gold to set known, controlled values.
	char.Gold = 0
	room.Gold = 50

	total := totalGold(char, room)

	err := FloorPickupGold(50, room, char)
	require.NoError(t, err)

	assert.Equal(t, 50, char.Gold, "char should have 50 gold")
	assert.Equal(t, 0, room.Gold, "floor should have 0 gold")
	assert.Equal(t, total, totalGold(char, room), "conservation: total gold unchanged")
}

// TestFloorDropGold_Happy verifies that gold moves from the character to the
// floor and the combined total is conserved.
func TestFloorDropGold_Happy(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	// Override the default starting gold to set known, controlled values.
	char.Gold = 100
	room.Gold = 0

	total := totalGold(char, room)

	err := FloorDropGold(50, char, room)
	require.NoError(t, err)

	assert.Equal(t, 50, char.Gold, "char should have 50 gold remaining")
	assert.Equal(t, 50, room.Gold, "floor should have 50 gold")
	assert.Equal(t, total, totalGold(char, room), "conservation: total gold unchanged")
}

// TestFloorPickupGold_Insufficient verifies that an error is returned when
// the floor has less gold than requested, and balances are unchanged.
func TestFloorPickupGold_Insufficient(t *testing.T) {
	char := newTestChar()
	room := newTestRoom()
	// Override the default starting gold to set known, controlled values.
	char.Gold = 0
	room.Gold = 10

	total := totalGold(char, room)

	err := FloorPickupGold(50, room, char)
	assert.Error(t, err, "should error when floor has insufficient gold")
	assert.Equal(t, 0, char.Gold, "char gold unchanged")
	assert.Equal(t, 10, room.Gold, "floor gold unchanged")
	assert.Equal(t, total, totalGold(char, room), "conservation: total gold unchanged")
}
