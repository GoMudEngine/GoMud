package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestTooTrivialToAuction(t *testing.T) {
	defer func(old int) { auctionMinListValue = old }(auctionMinListValue)
	auctionMinListValue = 100

	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {ItemId: 1, Name: "trinket", Value: 50},   // below floor
		2: {ItemId: 2, Name: "at floor", Value: 100}, // exactly at floor -> listable
		3: {ItemId: 3, Name: "prize", Value: 500},    // above floor
	})
	defer cleanup()

	if !tooTrivialToAuction(items.New(1)) {
		t.Error("value 50 < floor 100 should be too trivial to auction")
	}
	if tooTrivialToAuction(items.New(2)) {
		t.Error("value 100 == floor should be listable")
	}
	if tooTrivialToAuction(items.New(3)) {
		t.Error("value 500 should be listable")
	}
}

func TestTooTrivialToAuction_FloorDisabled(t *testing.T) {
	defer func(old int) { auctionMinListValue = old }(auctionMinListValue)
	auctionMinListValue = 0 // floor off -> everything listable

	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {ItemId: 1, Name: "trinket", Value: 1},
	})
	defer cleanup()

	if tooTrivialToAuction(items.New(1)) {
		t.Error("with the floor at 0, even a 1g item should be listable")
	}
}
