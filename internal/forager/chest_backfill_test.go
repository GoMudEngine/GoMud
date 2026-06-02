package forager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestSelectBackfillTransfers_NeediestGapFirst(t *testing.T) {
	si := &shops.ShopInventory{Stock: []shops.StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 9}, // gap 1
		{ItemId: 20, MaxStock: 10, Current: 2}, // gap 8 (neediest)
	}}
	pool := map[int]int{10: 5, 20: 5}
	got := shops.SelectStockTransfers(si, pool)
	assert.Equal(t, 5, got[20])
	assert.Equal(t, 1, got[10])
}

func TestSelectBackfillTransfers_AllToppedOff(t *testing.T) {
	si := &shops.ShopInventory{Stock: []shops.StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 10},
	}}
	pool := map[int]int{10: 5}
	got := shops.SelectStockTransfers(si, pool)
	assert.Empty(t, got, "no gaps → nothing transferred")
}

func TestSelectBackfillTransfers_OnlyStockedItems(t *testing.T) {
	si := &shops.ShopInventory{Stock: []shops.StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 5},
	}}
	pool := map[int]int{99: 5}
	got := shops.SelectStockTransfers(si, pool)
	assert.Empty(t, got, "vendor only pulls items it already stocks")
}

func TestBackfill_GlobalCrossZone(t *testing.T) {
	const chestZone, chestRoom = "zoneA-5.4cook", 51001
	RegisterChestRoom(chestZone, chestRoom)

	orig := loadRoomFn
	defer func() { loadRoomFn = orig }()
	loadRoomFn = func(id int) *rooms.Room {
		if id != chestRoom {
			return nil
		}
		return &rooms.Room{Containers: map[string]rooms.Container{
			"lockbox": {Items: []items.Item{{ItemId: 40063}, {ItemId: 40063}}},
		}}
	}

	pool, rooms2 := chestPoolAll()
	if pool[40063] != 2 {
		t.Fatalf("global pool should aggregate cross-zone chest, got %v", pool)
	}
	if len(rooms2) == 0 {
		t.Fatalf("expected chest room in global list")
	}
}

func TestChestPoolForZone_AggregatesViaIndex(t *testing.T) {
	const zone = "test-zone-pool-5.4"
	const chestRoom = 49901
	RegisterChestRoom(zone, chestRoom)

	orig := loadRoomFn
	defer func() { loadRoomFn = orig }()
	loadRoomFn = func(id int) *rooms.Room {
		if id != chestRoom {
			return nil
		}
		return &rooms.Room{
			Containers: map[string]rooms.Container{
				"lockbox": {Items: []items.Item{{ItemId: 10}, {ItemId: 10}, {ItemId: 20}}},
			},
		}
	}

	pool, chestRooms := chestPoolForZone(zone)
	assert.Equal(t, 2, pool[10])
	assert.Equal(t, 1, pool[20])
	assert.Contains(t, chestRooms, chestRoom)

	emptyPool, _ := chestPoolForZone("test-zone-unregistered-5.4")
	assert.Empty(t, emptyPool)
}
