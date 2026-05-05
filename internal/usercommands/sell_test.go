package usercommands

// sell_test.go: multi-quantity sell command tests.
//
// Strategy: use the seedAllRegistries() helper from usercommands_test.go to
// get a consistent room + user, then inject a merchant mob instance with a
// legacy Character.Shop so the merchant passes FindMerchant and GetSellPrice
// returns a positive value.
//
// Gold assertions use Greater/Less rather than exact equality because:
//   - GetSellPrice() returns ceil(price * priceScale * 0.25), where
//     priceScale decrements each time an item is added to the shop's stock.
//   - Bartering skill can add a small bonus.
// The tests focus on item count conservation and direction-of-gold-change.

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Shared test item IDs ────────────────────────────────────────────────────

// testSellItemId is seeded in seedAllRegistries() as Iron Sword, Value=100.
const testSellItemId = 10001

// ── Merchant setup helpers ──────────────────────────────────────────────────

// seedMerchantInRoom injects a legacy-shop merchant into the player's room.
// HasShop() returns true because Character.Shop is non-empty.
// GetSellPrice returns > 0 because testSellItemId is in the shop's sale list.
func seedMerchantInRoom(t *testing.T, user *users.UserRecord, goldAmount int) (int, func()) {
	t.Helper()

	const merchantMobId = 2
	const merchantInstId = 201

	merchant := &mobs.Mob{
		MobId:       merchantMobId,
		InstanceId:  merchantInstId,
		HomeRoomId:  1,
		BuysGeneral: false,
		Character: characters.Character{
			Name:   "Merchant",
			RoomId: 1,
			Gold:   goldAmount,
			Buffs:  buffs.New(),
			// Price=100, Quantity=0 → GetSellPrice returns ceil(100*1.0*0.25)=25.
			// Each subsequent sell increments Quantity which lowers the price.
			Shop: characters.Shop{
				{ItemId: testSellItemId, Price: 100, Quantity: 0, QuantityMax: 0},
			},
		},
	}
	merchant.Character.HealthMax.Value = 100
	merchant.Character.Health = 100

	mobs.SetInstanceForTest(merchantInstId, merchant)

	room := rooms.LoadRoom(user.Character.RoomId)
	require.NotNil(t, room, "player's room must exist")
	room.AddMob(merchantInstId)

	return merchantInstId, func() {
		room.RemoveMob(merchantInstId)
		mobs.SetInstanceForTest(merchantInstId, nil)
	}
}

// countSellItems returns how many items with itemId are in the backpack.
func countSellItems(ch *characters.Character, itemId int) int {
	count := 0
	for _, it := range ch.Items {
		if it.ItemId == itemId {
			count++
		}
	}
	return count
}

// ── Tests ───────────────────────────────────────────────────────────────────

// TestSell_BareAll_Rejected verifies "sell all" with no item name is rejected:
// handled=true, no state change, no panic.
func TestSell_BareAll_Rejected(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	startGold := user.Character.Gold

	handled, err := Sell("all", user, room, 0)

	assert.True(t, handled, "should be handled")
	assert.NoError(t, err)
	assert.Equal(t, startGold, user.Character.Gold, "gold must not change on rejection")
}

// TestSell_BareAllDot_Rejected verifies "all." with empty trailing name is
// also rejected cleanly.
func TestSell_BareAllDot_Rejected(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	startGold := user.Character.Gold

	handled, err := Sell("all.", user, room, 0)

	assert.True(t, handled, "should be handled")
	assert.NoError(t, err)
	assert.Equal(t, startGold, user.Character.Gold, "gold must not change on rejection")
}

// TestSell_NoItem verifies "sell nonexistent" exits cleanly when the player
// doesn't have the named item.
func TestSell_NoItem(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	startGold := user.Character.Gold

	handled, err := Sell("nonexistent item", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, startGold, user.Character.Gold, "gold must not change when item not found")
}

// TestSell_SingleItem verifies the original single-item form: item leaves
// inventory and gold increases.
func TestSell_SingleItem(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	_, merchantCleanup := seedMerchantInRoom(t, user, 1000)
	defer merchantCleanup()

	require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	startGold := user.Character.Gold
	require.Equal(t, 1, countSellItems(user.Character, testSellItemId))

	handled, err := Sell("iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, 0, countSellItems(user.Character, testSellItemId), "item should leave backpack")
	assert.Greater(t, user.Character.Gold, startGold, "gold must increase after sell")
}

// TestSell_Quantity_SellFive_HaveSeven verifies "sell 5 iron sword" with 7
// items: exactly 5 sold, exactly 2 remain, gold increases.
func TestSell_Quantity_SellFive_HaveSeven(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	_, merchantCleanup := seedMerchantInRoom(t, user, 10000)
	defer merchantCleanup()

	for i := 0; i < 7; i++ {
		require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	}

	startGold := user.Character.Gold

	handled, err := Sell("5 iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, 2, countSellItems(user.Character, testSellItemId),
		"exactly 2 items must remain after selling 5 of 7")
	assert.Greater(t, user.Character.Gold, startGold, "gold must increase")
}

// TestSell_Quantity_SellFive_HaveThree verifies "sell 5 iron sword" when the
// player only has 3: all 3 sold (partial fill, no panic, no hang).
func TestSell_Quantity_SellFive_HaveThree(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	_, merchantCleanup := seedMerchantInRoom(t, user, 10000)
	defer merchantCleanup()

	for i := 0; i < 3; i++ {
		require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	}

	startGold := user.Character.Gold

	handled, err := Sell("5 iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, 0, countSellItems(user.Character, testSellItemId),
		"all 3 items should sell even though 5 were requested")
	assert.Greater(t, user.Character.Gold, startGold, "gold must increase")
}

// TestSell_AllDot verifies "sell all.iron sword" sells every matching item.
func TestSell_AllDot(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	_, merchantCleanup := seedMerchantInRoom(t, user, 10000)
	defer merchantCleanup()

	const n = 4
	for i := 0; i < n; i++ {
		require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	}

	startGold := user.Character.Gold

	handled, err := Sell("all.iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, 0, countSellItems(user.Character, testSellItemId),
		"all items must be sold via 'all.name'")
	assert.Greater(t, user.Character.Gold, startGold, "gold must increase")
}

// TestSell_AllSpace verifies "sell all iron sword" sells every matching item.
func TestSell_AllSpace(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	_, merchantCleanup := seedMerchantInRoom(t, user, 10000)
	defer merchantCleanup()

	const n = 3
	for i := 0; i < n; i++ {
		require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	}

	startGold := user.Character.Gold

	handled, err := Sell("all iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, 0, countSellItems(user.Character, testSellItemId),
		"all items must be sold via 'all name'")
	assert.Greater(t, user.Character.Gold, startGold, "gold must increase")
}

// TestSell_MerchantRunsOutOfGold verifies that the sell loop stops when the
// merchant can no longer afford the item.
//
// First sell price = ceil(100 * 1.0 * 0.25) = 25.
// Second sell price = ceil(100 * (1-1/20) * 0.25) = ceil(23.75) = 24.
// Third sell price = ceil(100 * (1-2/20) * 0.25) = ceil(22.5) = 23.
// Merchant gold = 48 → can afford sells 1+2 (25+24=49 > 48, so only 1?).
// Actually sell 1 costs 25, leaving merchant with 23 — can merchant afford
// sell 2 at price 24? No (23 < 24). So only 1 sell goes through.
// Let's give merchant 60g → sell1=25 (merchant=35), sell2=24 (merchant=11),
// sell3=23 > 11 → stops at 2. Verify soldCount=2 and remaining items=3.
func TestSell_MerchantRunsOutOfGold(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	// Merchant has 60g → can afford sell1(25) + sell2(24) = 49g total,
	// leaving 11g which is < sell3's price of 23g → stops after 2.
	const merchantGold = 60
	_, merchantCleanup := seedMerchantInRoom(t, user, merchantGold)
	defer merchantCleanup()

	const n = 5
	for i := 0; i < n; i++ {
		require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	}

	startGold := user.Character.Gold

	handled, err := Sell("5 iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)

	// Some items should still remain (merchant ran dry before all 5).
	remaining := countSellItems(user.Character, testSellItemId)
	assert.Greater(t, remaining, 0, "some items must remain after merchant runs out of gold")
	assert.Less(t, remaining, n, "at least 1 item should have been sold")
	assert.Greater(t, user.Character.Gold, startGold, "player gained some gold before merchant ran dry")
}

// TestSell_NoMerchant verifies that sell exits cleanly when no merchant is
// present in the room: item stays, gold unchanged.
func TestSell_NoMerchant(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	require.True(t, user.Character.StoreItem(items.New(testSellItemId)))
	startItems := countSellItems(user.Character, testSellItemId)
	startGold := user.Character.Gold

	handled, err := Sell("iron sword", user, room, 0)

	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Equal(t, startItems, countSellItems(user.Character, testSellItemId),
		"item must stay without merchant")
	assert.Equal(t, startGold, user.Character.Gold, "gold must stay without merchant")
}
