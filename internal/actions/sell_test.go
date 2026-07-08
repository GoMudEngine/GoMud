package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// affixedSellPrice must be spec.Value * ShopBuyRatio (fixed spread), independent
// of scarcity — the Stage-2 melt price for instance loot.
func TestAffixedSellPrice(t *testing.T) {
	cfg := shops.DefaultPricingConfig() // BuyRatio 0.50
	it := items.Item{ItemId: 1, Affixed: true, Spec: &items.ItemSpec{Value: 400}}
	if got := affixedSellPrice(it, cfg); got != 200 { // 400 * 0.50
		t.Errorf("affixedSellPrice = %d; want 200", got)
	}
}

// ─── Sell test infrastructure ───────────────────────────────────────────────
//
// The gold-gate + skip-filter behaviors are exercised against the LEGACY shop
// path (Character.Shop / GetSellPrice / StockItem), which is fully reachable in
// a go-test unit context: it needs only seeded item specs and a hand-built mob
// with a non-empty Character.Shop. The living-economy ShopInventory path
// (resolveMerchant → shops.GetShopInventory → EvaluateBuyRules) requires loaded
// shop data files that go test does NOT provide, so — matching the chunk-2.1
// buy-lift precedent — those absolute-price branches are deferred to the
// in-game smoke. The gold-gate DIRECTION (mob sale leaves merchant gold
// unchanged; player sale drains it) is what we assert here, on the legacy path
// where merchant gold lives on mob.Character.Gold.

const sellTestItemId = 70001      // normal sellable, Value=100
const sellTestQuestItemId = 70002 // quest item (QuestToken set)
const sellTestComponentId = 70003 // is_component material
const sellTestZeroValueId = 70004 // Value==0

// seedSellItemSpecs registers the item specs used by the sell tests.
func seedSellItemSpecs() func() {
	return items.SeedItemsForTest(map[int]*items.ItemSpec{
		sellTestItemId: {
			ItemId: sellTestItemId,
			Name:   "iron sword",
			Type:   items.Weapon,
			Value:  100,
		},
		sellTestQuestItemId: {
			ItemId:     sellTestQuestItemId,
			Name:       "sacred relic",
			Type:       items.Object,
			Value:      100,
			QuestToken: "5-start",
		},
		sellTestComponentId: {
			ItemId:      sellTestComponentId,
			Name:        "iron ore",
			Type:        items.Object,
			Value:       50,
			IsComponent: true,
		},
		sellTestZeroValueId: {
			ItemId: sellTestZeroValueId,
			Name:   "worthless trinket",
			Type:   items.Object,
			Value:  0,
		},
	})
}

// seedSellRoom builds a minimal one-room world for the sell tests and returns
// a cleanup function.
func seedSellRoom(t *testing.T) func() {
	t.Helper()
	room := &rooms.Room{
		RoomId: 1,
		Zone:   "TestZone",
		Title:  "Market",
		Exits:  map[string]exit.RoomExit{},
	}
	return rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room},
		map[string]*rooms.ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}}},
		},
	)
}

// seedSellMerchant injects a legacy-shop merchant into room 1 carrying the
// normal sell item in its sale list (so GetSellPrice > 0). Returns a cleanup.
func seedSellMerchant(t *testing.T, merchantGold int) func() {
	t.Helper()
	const merchantMobId = 2
	const merchantInstId = 301

	merchant := &mobs.Mob{
		MobId:      merchantMobId,
		InstanceId: merchantInstId,
		HomeRoomId: 1,
		Zone:       "TestZone",
		Character: characters.Character{
			Name:   "Merchant",
			RoomId: 1,
			Gold:   merchantGold,
			Buffs:  buffs.New(),
			Shop: characters.Shop{
				{ItemId: sellTestItemId, Price: 100, Quantity: 0, QuantityMax: 0},
			},
		},
	}
	merchant.Character.HealthMax.Value = 100
	merchant.Character.Health = 100

	mobs.SetInstanceForTest(merchantInstId, merchant)
	room := rooms.LoadRoom(1)
	require.NotNil(t, room, "room 1 must exist")
	room.AddMob(merchantInstId)

	return func() {
		room.RemoveMob(merchantInstId)
		mobs.SetInstanceForTest(merchantInstId, nil)
	}
}

// merchantInstance returns the seeded merchant for gold assertions.
func merchantInstance() *mobs.Mob { return mobs.GetInstance(301) }

// newSellerActor builds an Actor backed by a hand-built character with the
// given items. isPlayer toggles the gold-model gate.
func newSellerActor(t *testing.T, isPlayer bool, itemIds ...int) Actor {
	t.Helper()
	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	if isPlayer {
		u := users.NewTestUser(1, "seller", "Seller", 1)
		u.Character.RoomId = 1
		u.Character.Gold = 0
		u.Character.Buffs = buffs.New()
		for _, id := range itemIds {
			require.True(t, u.Character.StoreItem(items.New(id)), "store item %d", id)
		}
		return &UserActor{User: u, Room: room}
	}

	c := &characters.Character{
		Name:   "MobSeller",
		RoomId: 1,
		Gold:   0,
		Buffs:  buffs.New(),
	}
	for _, id := range itemIds {
		require.True(t, c.StoreItem(items.New(id)), "store item %d", id)
	}
	m := &mobs.Mob{MobId: 9, InstanceId: 309, Character: *c}
	return &MobActor{Mob: m, Room: room}
}

// ─── Tests ──────────────────────────────────────────────────────────────────

func TestSell_MobSale_DoesNotDrainShopGold(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	// Merchant with plenty of gold so the only thing under test is whether the
	// MOB sale touches it.
	defer seedSellMerchant(t, 1000)()

	mobSeller := newSellerActor(t, false, sellTestItemId)
	startMerchantGold := merchantInstance().Character.Gold

	res := Sell(mobSeller, SellOptions{ItemName: "iron sword", Quantity: 1})

	assert.Equal(t, 1, res.Sold, "mob should sell its one item")
	assert.NotEqual(t, SellStopMerchantBroke, res.Reason, "mob sale must never report merchant-broke")
	assert.Greater(t, mobSeller.GetCharacter().Gold, 0, "seller gold should increase")
	assert.Equal(t, startMerchantGold, merchantInstance().Character.Gold,
		"mob sale must NOT drain merchant gold")
}

func TestSell_PlayerSale_DrainsShopGold(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	playerSeller := newSellerActor(t, true, sellTestItemId)
	startMerchantGold := merchantInstance().Character.Gold

	res := Sell(playerSeller, SellOptions{ItemName: "iron sword", Quantity: 1})

	assert.Equal(t, 1, res.Sold, "player should sell its one item")
	assert.Greater(t, playerSeller.GetCharacter().Gold, 0, "seller gold should increase")
	assert.Less(t, merchantInstance().Character.Gold, startMerchantGold,
		"player sale MUST drain merchant gold")
}

func TestSell_SellAllSellable_SkipsQuestAndComponentAndZeroValue(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	// Mob carrying a quest item, a component, a zero-value item, and one normal.
	mobSeller := newSellerActor(t, false,
		sellTestQuestItemId, sellTestComponentId, sellTestZeroValueId, sellTestItemId)

	res := Sell(mobSeller, SellOptions{SellAllSellable: true})

	assert.Equal(t, 1, res.Sold, "only the normal sellable should sell")

	char := mobSeller.GetCharacter()
	// The quest / component / zero-value items must remain in inventory.
	remaining := map[int]bool{}
	for _, it := range char.Items {
		remaining[it.ItemId] = true
	}
	assert.True(t, remaining[sellTestQuestItemId], "quest item must be skipped")
	assert.True(t, remaining[sellTestComponentId], "component must be skipped")
	assert.True(t, remaining[sellTestZeroValueId], "zero-value item must be skipped")
	assert.False(t, remaining[sellTestItemId], "normal item should have sold")
}

func TestSell_NoMerchant(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	// No merchant seeded.

	seller := newSellerActor(t, true, sellTestItemId)

	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: 1})

	assert.Equal(t, 0, res.Sold)
	assert.Equal(t, SellStopNoMerchant, res.Reason)
}

// TestSell_PartialFill_ReportsSoldAll verifies that requesting more than the
// seller has (e.g. "sell 5" with only 3) reports SellStopSoldAll, NOT
// SellStopNoItem — so the player still gets a sale confirmation. Regression
// guard for the wrapper's `case SellStopNoItem: return` short-circuit.
func TestSell_PartialFill_ReportsSoldAll(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 100000)()

	seller := newSellerActor(t, false, sellTestItemId, sellTestItemId, sellTestItemId)

	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: 5})

	assert.Equal(t, 3, res.Sold, "should sell all 3 it had")
	assert.Equal(t, SellStopSoldAll, res.Reason,
		"running out mid-loop after selling some is a normal completion")
}

// TestSell_NamedNoItem verifies a genuine no-item request reports
// SellStopNoItem with Sold==0.
func TestSell_NamedNoItem(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true) // carrying nothing

	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: 1})

	assert.Equal(t, 0, res.Sold)
	assert.Equal(t, SellStopNoItem, res.Reason)
}

func TestSell_NoRoom(t *testing.T) {
	// An actor with no room → SellStopNoMerchant.
	seller := &MobActor{Mob: &mobs.Mob{}, Room: nil}
	res := Sell(seller, SellOptions{ItemName: "anything", Quantity: 1})
	assert.Equal(t, SellStopNoMerchant, res.Reason)
	assert.Equal(t, 0, res.Sold)
}

// TestSell_QuestItemRejectedNotNoMerchant verifies that selling a quest item
// returns SellStopRejected (not SellStopNoMerchant) regardless of whether a
// merchant is present. Regression guard for the 5.4 regression where
// resolveMerchant returned nil for quest items (zero-price offer) before the
// quest-token check was reached, causing the player to see "There's no merchant
// here." instead of "Quest items cannot be sold!".
func TestSell_QuestItemRejectedNotNoMerchant(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	// Player seller with a quest item, merchant present in the room.
	playerSeller := newSellerActor(t, true, sellTestQuestItemId)

	res := Sell(playerSeller, SellOptions{ItemName: "sacred relic", Quantity: 1})

	assert.Equal(t, 0, res.Sold, "quest item must never be sold")
	assert.Equal(t, SellStopRejected, res.Reason,
		"must report SellStopRejected, not SellStopNoMerchant")
}

// TestSell_PresentMerchantWontBuy_RejectedNotNoMerchant verifies that when a
// merchant IS present but won't pay for the offered item (non-positive offer —
// here a zero-value trinket), the result is SellStopRejected (the merchant
// gives a refusal) and NOT SellStopNoMerchant ("There's no merchant here.").
// Regression guard for the resolveMerchant probe-gate shortcut that collapsed
// a present-but-unwilling merchant into the no-merchant message.
func TestSell_PresentMerchantWontBuy_RejectedNotNoMerchant(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)() // present, stocks iron sword (Weapon) only

	// Seller offers a zero-value item the merchant will never pay for.
	seller := newSellerActor(t, true, sellTestZeroValueId)

	res := Sell(seller, SellOptions{ItemName: "worthless trinket", Quantity: 1})

	assert.Equal(t, 0, res.Sold, "an unwilling merchant buys nothing")
	assert.Equal(t, SellStopRejected, res.Reason,
		"merchant present but unwilling must report Rejected, not NoMerchant")
}

// TestSell_QuestItemRejectedNoMerchantPresent verifies the same rejection even
// when no merchant is in the room — the quest-token gate must fire BEFORE the
// merchant resolution so the message is always correct.
func TestSell_QuestItemRejectedNoMerchantPresent(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	// No merchant seeded.

	playerSeller := newSellerActor(t, true, sellTestQuestItemId)

	res := Sell(playerSeller, SellOptions{ItemName: "sacred relic", Quantity: 1})

	assert.Equal(t, 0, res.Sold, "quest item must never be sold")
	assert.Equal(t, SellStopRejected, res.Reason,
		"must report SellStopRejected even when no merchant is present")
}
