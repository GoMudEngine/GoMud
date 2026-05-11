package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestBuy_EmptyRequest(t *testing.T) {
	result := Buy(nil, BuyOptions{Request: ""})
	if result.Success {
		t.Errorf("expected Success=false on empty request")
	}
	if result.Reason != BuyReasonNoRequest {
		t.Errorf("expected Reason=%q, got %q", BuyReasonNoRequest, result.Reason)
	}
}

func TestBuildLegacyCatalog_SkipsMercAndPet(t *testing.T) {
	saleItems := characters.Shop{
		// Items and buffs should appear; mercs/pets should be skipped.
		{ItemId: 20000, Price: 50, Quantity: 1, QuantityMax: 1},
		{BuffId: 1, Price: 100, Quantity: 1, QuantityMax: 1},
		{MobId: 100, Price: 250, Quantity: 1, QuantityMax: 1},
		{PetType: "kitten", Price: 10000, Quantity: 1, QuantityMax: 1},
	}
	cat := buildLegacyCatalog(saleItems)

	// Even if itemId 20000 doesn't resolve to a real item (the test data
	// dir might not be initialized), the merc/pet entries must always
	// be absent.
	if _, ok := cat.nameToShopItem["100"]; ok {
		t.Errorf("merc MobId should not appear in nameToShopItem")
	}
	if _, ok := cat.nameToShopItem["kitten"]; ok {
		t.Errorf("pet PetType should not appear in nameToShopItem")
	}
}

func newEmptyTestRoom(t *testing.T) *rooms.Room {
	t.Helper()
	r := &rooms.Room{RoomId: 99999}
	return r
}

func TestBuy_NoMerchant(t *testing.T) {
	room := newEmptyTestRoom(t)
	mobActor := &MobActor{Room: room} // no User, no Mob — only Room matters
	result := Buy(mobActor, BuyOptions{Request: "iron ingot"})
	if result.Success {
		t.Errorf("expected Success=false with no merchant in room")
	}
	if result.Reason != BuyReasonNoMerchant {
		t.Errorf("Reason = %q, want %q", result.Reason, BuyReasonNoMerchant)
	}
}
