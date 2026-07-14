package auctions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestNpcWallet(t *testing.T) {
	w := &NpcWallet{Balance: 100, Cap: 200}
	if !w.CanAfford(100) || w.CanAfford(101) {
		t.Fatal("CanAfford wrong")
	}
	w.Spend(60)
	if w.Balance != 40 {
		t.Fatalf("Spend: balance=%d want 40", w.Balance)
	}
	w.Refund(1000) // clamps to cap
	if w.Balance != 200 {
		t.Fatalf("Refund clamp: balance=%d want 200", w.Balance)
	}
	w.Regen(1000) // clamps to cap
	if w.Balance != 200 {
		t.Fatalf("Regen clamp: balance=%d want 200", w.Balance)
	}
}

func TestCollector_InterestAndMaxBid(t *testing.T) {
	collectorMinValue = 500
	collectorPremium = 1.0
	c := &collector{name: "Veyd", wallet: &NpcWallet{Balance: 10000, Cap: 10000}}

	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		101: {ItemId: 101, Name: "fine blade", Type: items.Weapon, Value: 800},
		102: {ItemId: 102, Name: "cheap dagger", Type: items.Weapon, Value: 100},
		103: {ItemId: 103, Name: "herb", Type: items.Potion, Value: 900},
	})()
	if !c.Interested(items.New(101)) {
		t.Error("collector should want a valuable weapon")
	}
	if c.Interested(items.New(102)) {
		t.Error("collector should NOT want a cheap weapon (below min value)")
	}
	if c.Interested(items.New(103)) {
		t.Error("collector should NOT want a non-equipment potion")
	}
	if got := c.MaxBid(items.New(101)); got != 800 {
		t.Errorf("MaxBid=%d want 800 (Value*1.0)", got)
	}
}

func TestNextNpcBid(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		201: {ItemId: 201, Name: "prize", Type: items.Weapon, Value: 800},
	})()
	collectorMinValue = 500
	collectorPremium = 1.0
	rich := &collector{name: "Rich", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	broke := &collector{name: "Broke", wallet: &NpcWallet{Balance: 10, Cap: 5000}}
	item := items.New(201)

	b, bid, ok := nextNpcBid([]NpcBuyer{broke, rich}, item, 0, 250, "", false)
	if !ok || b.Name() != "Rich" || bid != 250 {
		t.Fatalf("expected Rich to bid 250, got %v %d %v", b, bid, ok)
	}
	if _, _, ok := nextNpcBid([]NpcBuyer{rich}, item, 800, 801, "Rich", true); ok {
		t.Fatal("nobody should bid past MaxBid")
	}
	if _, _, ok := nextNpcBid([]NpcBuyer{rich}, item, 300, 301, "Rich", true); ok {
		t.Fatal("the current high NPC must not bid against itself")
	}
}

func TestCraftsperson(t *testing.T) {
	craftMinValue = 50
	craftPremium = 1.0
	c := &craftsperson{name: "Ordwin", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		301: {ItemId: 301, Name: "rare ore", IsComponent: true, Value: 200},
		302: {ItemId: 302, Name: "junk scrap", IsComponent: true, Value: 5},
		303: {ItemId: 303, Name: "sword", Type: items.Weapon, Value: 900},
	})()
	if !c.Interested(items.New(301)) {
		t.Error("crafter should want a valuable component")
	}
	if c.Interested(items.New(302)) {
		t.Error("crafter should NOT want a near-worthless component")
	}
	if c.Interested(items.New(303)) {
		t.Error("crafter should NOT want a non-component weapon")
	}
	if c.MaxBid(items.New(301)) != 200 {
		t.Errorf("MaxBid=%d want 200", c.MaxBid(items.New(301)))
	}
	if c.Flavor() != "for their workshop" {
		t.Errorf("flavor=%q", c.Flavor())
	}
}

func TestAdventurer(t *testing.T) {
	advMinValue = 300
	advPremium = 0.9
	a := &adventurer{name: "Kest", wallet: &NpcWallet{Balance: 5000, Cap: 5000}}
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		311: {ItemId: 311, Name: "enchanted blade", Type: items.Weapon, Value: 1000, StatMods: map[string]int{"strength": 5}},
		312: {ItemId: 312, Name: "plain blade", Type: items.Weapon, Value: 1000},
		313: {ItemId: 313, Name: "ore", IsComponent: true, Value: 1000, StatMods: map[string]int{"strength": 5}},
	})()
	if !a.Interested(items.New(311)) {
		t.Error("adventurer should want stat-bearing gear")
	}
	if a.Interested(items.New(312)) {
		t.Error("adventurer should NOT want plain gear (no statmods)")
	}
	if a.Interested(items.New(313)) {
		t.Error("adventurer should NOT want a non-equipment component")
	}
	if a.MaxBid(items.New(311)) != 900 {
		t.Errorf("MaxBid=%d want 900", a.MaxBid(items.New(311)))
	}
	if a.Flavor() != "to gear up" {
		t.Errorf("flavor=%q", a.Flavor())
	}
}

func newShopkeeperTestItem(t *testing.T) (items.Item, func()) {
	t.Helper()
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		901: {ItemId: 901, Name: "iron blade", Type: items.Weapon, Value: 1000,
			VendorCategories: []string{shops.CraftSupportBlacksmithing}},
		902: {ItemId: 902, Name: "no-vendor trinket", Type: items.Ring, Value: 1000},
	})
	return items.New(901), cleanup
}

func TestShopkeeper_InterestedAndMaxBid(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()

	shops.ClearCache()
	defer shops.ClearCache()
	smith := shops.RegisterShop("testzone", 1, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})
	shops.RegisterShop("testzone", 2, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportTailoring})

	sk := &shopkeeper{name: "The Merchants' Guild"}
	if !sk.Interested(item) {
		t.Fatal("shopkeeper should want a blacksmith item when a blacksmith shop has gold")
	}
	want := shops.EvaluateBuyRules(item, smith, "", false, shops.PricingConfigFromBalance(), nil).Price
	if want <= 0 {
		t.Fatalf("test setup: expected a positive counter offer, got %d", want)
	}
	if got := sk.MaxBid(item); got != want {
		t.Errorf("MaxBid=%d want %d (EvaluateBuyRules price)", got, want)
	}
}

func TestShopkeeper_NoMatchingShop(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	shops.RegisterShop("testzone", 2, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportTailoring})

	sk := &shopkeeper{name: "The Merchants' Guild"}
	if sk.Interested(item) {
		t.Error("shopkeeper should NOT want an item no live shop's discipline accepts")
	}
	if got := sk.MaxBid(item); got != 0 {
		t.Errorf("MaxBid=%d want 0 when uninterested", got)
	}
}

func TestShopkeeper_EscrowUsesRealShopGold(t *testing.T) {
	item, cleanup := newShopkeeperTestItem(t)
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	smith := shops.RegisterShop("testzone", 1, 100, shops.ShopInventory{
		Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})

	orig := saveShopFn
	saveShopFn = func(zone string, mobId, roomId int) error { return nil }
	defer func() { saveShopFn = orig }()

	sk := &shopkeeper{name: "The Merchants' Guild"}
	if !sk.Interested(item) {
		t.Fatal("precondition: shopkeeper should be interested")
	}
	if !sk.CanAfford(400) {
		t.Fatal("shop with 5000 gold / 2500 reserve should afford 400")
	}
	sk.Spend(400)
	if smith.Gold != 4600 {
		t.Errorf("after Spend(400) shop gold=%d want 4600", smith.Gold)
	}
	sk.Refund(400)
	if smith.Gold != 5000 {
		t.Errorf("after Refund(400) shop gold=%d want 5000", smith.Gold)
	}
}

func TestShopkeeper_SelectionUpdatesPerItem(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		911: {ItemId: 911, Name: "blade", Type: items.Weapon, Value: 1000,
			VendorCategories: []string{shops.CraftSupportBlacksmithing}},
		912: {ItemId: 912, Name: "cloak", Type: items.Body, Value: 1000,
			VendorCategories: []string{shops.CraftSupportTailoring}},
	})
	defer cleanup()
	shops.ClearCache()
	defer shops.ClearCache()
	shops.RegisterShop("z", 1, 100, shops.ShopInventory{Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportBlacksmithing})
	shops.RegisterShop("z", 2, 100, shops.ShopInventory{Gold: 5000, StartingGold: 5000, CraftSupport: shops.CraftSupportTailoring})

	sk := &shopkeeper{name: "The Merchants' Guild"}
	blade := items.New(911)
	cloak := items.New(912)
	if !sk.Interested(blade) || sk.selectFor(blade).shop.CraftSupport != shops.CraftSupportBlacksmithing {
		t.Fatal("blade should select the blacksmith")
	}
	// Different item UUID must recompute, not return the memoized blacksmith selection.
	if !sk.Interested(cloak) || sk.selectFor(cloak).shop.CraftSupport != shops.CraftSupportTailoring {
		t.Fatal("cloak should re-select the tailor (memoization must invalidate per item UUID)")
	}
}
