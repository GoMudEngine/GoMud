# Affixed Gear — Stage 3 (Shop Resale) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a player sells an affix-scaled instance item, the shop stores the
**exact per-instance item** and resells it at a markup — instead of melting it
(Stage 2). Extend the living-economy `ShopInventory` with a per-instance affixed
stock list, thread it through sell → stock and buy ← stock, persist it, and cap
clutter.

**Architecture:** A new `ShopInventory.AffixedStock []AffixedStockEntry` holds
full `items.Item` copies + their relist price. The existing ItemId-keyed
`Stock []StockEntry` is untouched (base commodities). Sell (living-economy path)
appends to `AffixedStock` at `AffixValue × ShopBuyRatio`; buy lists those entries
and, on purchase, hands the stored item (fresh UUID) at `AffixValue × 1.0` and
removes the entry. Persists via YAML. Legacy `characters.Shop` merchants keep the
Stage-2 melt behavior (out of scope — they lack persistence).

**Tech Stack:** Go. `internal/shops`, `internal/actions/{sell,buy}.go`.
Test seeding via `shops.RegisterShop` (in-memory, no data files needed).

**Spec:** `docs/superpowers/specs/2026-07-08-shops-trade-affixed-gear-design.md`
(Stage 3 row + pricing model).

**Depends on:** Stages 1–2 (merged) — `Item.Affixed`, stamped `spec.Value`,
`affixedSellPrice`, and the Stage-2 melt-skip seam in `sell.go`.

---

## Task 1: `AffixedStockEntry` + `ShopInventory` methods (data model)

**Files:**
- Modify: `internal/shops/shopinventory.go`
- Test: `internal/shops/shopinventory_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/shops/shopinventory_test.go`:

```go
func TestAffixedStock_AddListRemove(t *testing.T) {
	si := &ShopInventory{}
	a := items.Item{ItemId: 10, Affixed: true, Spec: &items.ItemSpec{Value: 400, Name: "Keen Torc"}}
	b := items.Item{ItemId: 11, Affixed: true, Spec: &items.ItemSpec{Value: 300, Name: "Warding Ring"}}

	si.AddAffixedStock(a, 200, 100) // price 200, cap 100
	si.AddAffixedStock(b, 150, 100)
	if len(si.AffixedStock) != 2 {
		t.Fatalf("want 2 affixed entries, got %d", len(si.AffixedStock))
	}

	// Remove by index returns the stored item and shrinks the list.
	got, ok := si.RemoveAffixedStock(0)
	if !ok || got.ItemId != 10 {
		t.Fatalf("RemoveAffixedStock(0) = %+v, %v", got, ok)
	}
	if len(si.AffixedStock) != 1 || si.AffixedStock[0].Item.ItemId != 11 {
		t.Fatalf("after remove, want [11], got %+v", si.AffixedStock)
	}
}

// The clutter cap evicts the OLDEST entry when full, so a busy shop doesn't
// accumulate unbounded per-instance stock.
func TestAffixedStock_CapEvictsOldest(t *testing.T) {
	si := &ShopInventory{}
	for i := 0; i < 5; i++ {
		si.AddAffixedStock(items.Item{ItemId: 100 + i, Affixed: true,
			Spec: &items.ItemSpec{Value: 100}}, 50, 3) // cap 3
	}
	if len(si.AffixedStock) != 3 {
		t.Fatalf("cap 3: want 3 entries, got %d", len(si.AffixedStock))
	}
	// Oldest (100, 101) evicted; newest three (102,103,104) remain in order.
	for i, want := range []int{102, 103, 104} {
		if si.AffixedStock[i].Item.ItemId != want {
			t.Errorf("entry %d = %d; want %d", i, si.AffixedStock[i].Item.ItemId, want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/shops/ -run TestAffixedStock -v`
Expected: FAIL — undefined `AddAffixedStock`/`RemoveAffixedStock`/`AffixedStock`.

- [ ] **Step 3: Implement**

In `internal/shops/shopinventory.go`, add the type + field + methods:

```go
// AffixedStockEntry is one unique affix-scaled item a shop bought from a player
// and will resell. Unlike StockEntry (ItemId-keyed commodities), it carries the
// full per-instance item so the exact affixes survive.
type AffixedStockEntry struct {
	Item       items.Item `yaml:"item"`
	Price      int        `yaml:"price"`                 // relist price (AffixValue x 1.0)
	AddedRound uint64     `yaml:"added_round,omitempty"` // for age-based clutter eviction
}
```

Add to the `ShopInventory` struct (near `Stock`):

```go
	AffixedStock []AffixedStockEntry `yaml:"affixed_stock,omitempty"` // unique bought-back affixed items for resale
```

Add the methods:

```go
// AddAffixedStock appends a bought-back affixed item at relist price, evicting
// the oldest entry (FIFO) when the list is at cap. cap <= 0 means no cap.
func (si *ShopInventory) AddAffixedStock(item items.Item, price, cap int) {
	si.AffixedStock = append(si.AffixedStock, AffixedStockEntry{
		Item:       item,
		Price:      price,
		AddedRound: util.GetRoundCount(),
	})
	if cap > 0 {
		for len(si.AffixedStock) > cap {
			si.AffixedStock = si.AffixedStock[1:] // drop oldest
		}
	}
}

// RemoveAffixedStock removes and returns the entry at idx (e.g. on purchase).
func (si *ShopInventory) RemoveAffixedStock(idx int) (items.Item, bool) {
	if idx < 0 || idx >= len(si.AffixedStock) {
		return items.Item{}, false
	}
	it := si.AffixedStock[idx].Item
	si.AffixedStock = append(si.AffixedStock[:idx], si.AffixedStock[idx+1:]...)
	return it, true
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/shops/ -run TestAffixedStock -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/shopinventory_test.go
git commit -m "feat(shops): ShopInventory.AffixedStock (per-instance resale stock) + FIFO cap"
```

---

## Task 2: Sell → affixed stock (living-economy path)

Replace the Stage-2 melt-skip so a living-economy shop STORES the sold affixed
item for resale. Legacy `characters.Shop` merchants keep melting.

**Files:**
- Modify: `internal/actions/sell.go`
- Modify: `internal/configs/config.balance.go` + `config.balance.misc.go` (cap knob)
- Test: `internal/actions/sell_test.go`

- [ ] **Step 1: Add the clutter-cap config knob**

Field (near `GoldPerAffixPoint`): 
```go
	ShopAffixedStockCap ConfigInt `yaml:"ShopAffixedStockCap"` // Max per-instance affixed items a shop resells (default 8)
```
Default (near the `GoldPerAffixPoint` default):
```go
	if b.ShopAffixedStockCap <= 0 {
		b.ShopAffixedStockCap = 8
	}
```

- [ ] **Step 2: Write the failing test**

The existing sell tests use the LEGACY path (shopInv nil). For this we need a
living-economy shop — seed one with `shops.RegisterShop`. Append to
`sell_test.go`:

```go
func TestSell_AffixedItem_StocksForResale(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 100000)()

	// Register a living-economy shop for the seeded merchant (mob 2, room 1).
	si := shops.RegisterShop("TestZone", 2, 1, shops.ShopInventory{Gold: 100000})

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	require.True(t, char.StoreItem(newAffixedInstance(400)))

	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: 1})
	require.Equal(t, SellStopSoldAll, res.Reason)

	// The affixed item is now in the shop's per-instance resale stock at the
	// fixed-spread buy price (400 * 0.5 = 200), NOT the base ItemId stock.
	if len(si.AffixedStock) != 1 {
		t.Fatalf("want 1 affixed stock entry, got %d", len(si.AffixedStock))
	}
	if si.AffixedStock[0].Price != 200 {
		t.Errorf("affixed stock price = %d; want 200", si.AffixedStock[0].Price)
	}
	if si.AffixedStock[0].Item.GetSpec().Value != 400 {
		t.Errorf("stored affixed value = %d; want 400", si.AffixedStock[0].Item.GetSpec().Value)
	}
}
```

(Confirm `RegisterShop` makes `GetShopInventory("TestZone",2,1)` return `si` so
the sell path's `resolveMerchant` picks the living-economy branch. If the sell
path also needs the merchant flagged, mirror how `persistence_test.go` seeds a
live shop.)

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestSell_AffixedItem_StocksForResale -v`
Expected: FAIL — the item is currently melted (AffixedStock stays empty).

- [ ] **Step 4: Implement**

In `sell.go`, in the stock-update block, replace the melt-skip
(`if !item.Affixed { ... }`) so an affixed item on the living-economy path is
stored for resale. The relist price is its value at full (`AffixValue × 1.0` =
`item.GetSpec().Value`):

```go
	if item.Affixed {
		if shopInv != nil {
			cap := int(configs.GetBalanceConfig().ShopAffixedStockCap)
			shopInv.AddAffixedStock(item, item.GetSpec().Value, cap)
			shopInv.BuysCount++
			if err := shops.SaveShop(shopInv.Zone, shopInv.MobId, shopInv.RoomId); err != nil {
				mudlog.Error("SELL", "msg", "SaveShop failed", "error", err)
			}
		}
		// Legacy-shop merchant (shopInv == nil): item is melted (consumed), as Stage 2.
	} else if shopInv != nil {
		// ... existing base-ItemId stock-update block, unchanged ...
	} else {
		mob.Character.Shop.StockItem(item.ItemId)
	}
```

Add the `configs` import to `sell.go` if not already present.

- [ ] **Step 5: Run + build**

Run: `go test ./internal/actions/ -run TestSell -v && go build ./...`
Expected: PASS (new test + all Stage-2 sell tests, including the legacy melt
path which still has `shopInv == nil`).

- [ ] **Step 6: Commit**

```bash
git add internal/actions/sell.go internal/actions/sell_test.go internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "feat(sell): living-economy shops store affixed loot for resale (ShopAffixedStockCap)"
```

---

## Task 3: Buy ← affixed stock (list + purchase the exact item)

List affixed stock in the buy menu and, on purchase, hand the buyer the stored
per-instance item (fresh UUID) at its relist price, removing the entry.

**Files:**
- Modify: `internal/actions/buy.go`
- Test: `internal/actions/buy_test.go` (or the existing buy test file)

- [ ] **Step 1: Write the failing test**

Seed a living-economy shop with one affixed entry (via `RegisterShop`), then
`Buy` it by name and assert the buyer receives an item whose spec value matches
the stored affixed item (not the base), gold is deducted by the relist price,
and the entry is gone. (Mirror the existing buy tests' actor/room setup.)

```go
func TestBuy_AffixedStockItem(t *testing.T) {
	// ... seed room + merchant + buyer with enough gold (mirror existing buy tests) ...
	si := shops.RegisterShop("TestZone", 2, 1, shops.ShopInventory{Gold: 1000})
	si.AddAffixedStock(items.Item{ItemId: sellTestItemId, Affixed: true,
		Spec: &items.ItemSpec{Value: 400, Name: "iron sword", NameSimple: "sword"}}, 400, 8)

	buyer := /* player actor with >=400 gold */
	res := Buy(buyer, BuyOptions{ItemName: "iron sword", Quantity: 1})
	require.True(t, res.Success)

	// Buyer holds the affixed item (value 400, per-instance spec), paid 400.
	got, has := buyer.GetCharacter().FindInBackpack("iron sword")
	require.True(t, has)
	if got.GetSpec().Value != 400 || !got.Affixed {
		t.Errorf("bought item not the affixed instance: value=%d affixed=%v", got.GetSpec().Value, got.Affixed)
	}
	if len(si.AffixedStock) != 0 {
		t.Errorf("affixed entry should be removed after purchase")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestBuy_AffixedStockItem -v`
Expected: FAIL — buy doesn't list/sell affixed stock yet.

- [ ] **Step 3: Implement — list affixed entries**

In the buy listing loop (buy.go ~515, after the `shopInv.Stock` loop), append the
affixed entries to `available`/`itemNames`. Add an `affixedIdx int` to the
`invEntry` struct (default -1) to mark affixed rows:

```go
	for i := range shopInv.AffixedStock {
		e := &shopInv.AffixedStock[i]
		spec := e.Item.GetSpec()
		price := e.Price
		if barterSkill := char.GetSkillLevel(skills.Bartering); barterSkill > 0 {
			discount := float64(barterSkill) / 50.0 * 0.15
			price = shops.ApplyBarterSellDiscount(price, discount)
		}
		available = append(available, invEntry{
			item:       e.Item,
			plainName:  spec.Name,
			price:      price,
			affixedIdx: i, // marks this as an affixed-stock row
		})
		itemNames = append(itemNames, spec.Name)
		itemNamesFancy = append(itemNamesFancy, e.Item.DisplayName())
	}
```

(Add `affixedIdx int` to the `invEntry` type; initialize the base-stock rows'
`affixedIdx` to `-1`.)

- [ ] **Step 4: Implement — purchase an affixed entry**

In the purchase completion (buy.go ~581+), branch when `matched.affixedIdx >= 0`:
take the stored item (don't `items.New`), assign a fresh UUID, remove the entry,
handle gold + persist. Factor the "give item + messages" so the affixed item is
handed as-is:

```go
	if matched.affixedIdx >= 0 {
		if char.Gold < matched.price { /* insufficient-gold message + return */ }
		bought, ok := shopInv.RemoveAffixedStock(matched.affixedIdx)
		if !ok { /* out-of-stock return */ }
		bought.UUID = items.NewItemUUID() // fresh identity (UUID is yaml:"-")
		shopInv.SalesCount++
		char.Gold -= matched.price
		shopInv.Gold += matched.price
		if !char.StoreItem(bought) { /* rollback: re-add, refund, overburdened return */ }
		// emit ItemOwnership(gained) + buyer/room purchase messages (mirror executePurchaseItem)
		_ = shops.SaveShop(shopInv.Zone, shopInv.MobId, shopInv.RoomId)
		return BuyResult{Success: true, Purchased: 1, SaleType: "item"}
	}
```

Use the real UUID constructor (`items.New` does `uuid.New(items.UUIDItem)`);
expose/reuse it as `items.NewItemUUID()` or inline `uuid.New(items.UUIDItem)`
with the import. Verify against `items.New`.

- [ ] **Step 5: Run + build**

Run: `go test ./internal/actions/ -run 'TestBuy' -v && go build ./...`
Expected: PASS (new affixed-buy test + existing buy tests).

- [ ] **Step 6: Commit**

```bash
git add internal/actions/buy.go internal/actions/buy_test.go
git commit -m "feat(buy): shops list + resell per-instance affixed stock (exact item, fresh UUID)"
```

---

## Task 4: Persistence round-trip

Prove an `AffixedStock` entry survives `SaveShop` → reload (so bought-back gear
isn't lost on restart).

**Files:**
- Test: `internal/shops/persistence_test.go`

- [ ] **Step 1: Write the test**

Mirror the existing persistence test's save/reload helper. Register a shop, add
an affixed entry, `SaveShop`, evict the in-memory copy, reload, and assert the
`AffixedStock` entry (ItemId, Price, spec Value/affix mods) round-tripped.

```go
func TestPersistence_AffixedStockRoundTrip(t *testing.T) {
	// ... use the same temp-dir override + save/reload the existing persistence tests use ...
	si := shops.RegisterShop(zone, mobId, roomId, shops.ShopInventory{Gold: 500})
	si.AddAffixedStock(items.Item{ItemId: 10, Affixed: true,
		Spec: &items.ItemSpec{Value: 400, PhysicalMitigation: 5}}, 200, 8)
	require.NoError(t, shops.SaveShop(zone, mobId, roomId))

	// force reload (clear cache / re-Get) — mirror the existing test's mechanism
	reloaded := reloadShop(t, zone, mobId, roomId)
	require.Len(t, reloaded.AffixedStock, 1)
	e := reloaded.AffixedStock[0]
	if e.Price != 200 || e.Item.GetSpec().Value != 400 || e.Item.GetSpec().PhysicalMitigation != 5 {
		t.Errorf("affixed entry did not round-trip: %+v", e)
	}
}
```

(Read `persistence_test.go` for the exact temp-dir override + reload mechanism;
the assertion is the contract: `AffixedStock` + its per-instance spec persist.)

- [ ] **Step 2: Run**

Run: `go test ./internal/shops/ -run TestPersistence_AffixedStockRoundTrip -v`
Expected: PASS. If the per-instance spec doesn't round-trip, verify `items.Item`
+ `ItemSpec` YAML tags cover the affix fields (`overrides`/`statmods`/mitigations
are tagged — they should).

- [ ] **Step 3: Commit**

```bash
git add internal/shops/persistence_test.go
git commit -m "test(shops): affixed resale stock round-trips through SaveShop/reload"
```

---

## Definition of Done (Stage 3)

- `ShopInventory.AffixedStock` holds per-instance affixed items with a FIFO
  clutter cap (`ShopAffixedStockCap`, default 8).
- Selling an affixed item to a living-economy shop stores it for resale at
  `AffixValue × ShopBuyRatio`; buying it hands back the exact affixed item (fresh
  UUID) at `AffixValue × 1.0` and removes the entry.
- Affixed stock survives `SaveShop` → reload.
- Legacy `characters.Shop` merchants keep the Stage-2 melt behavior.
- `go build ./...` clean; `shops` + `actions` suites green; full `go test ./...`
  green.

## Divergences / Notes

- **Scope: living-economy shops only.** Legacy `characters.Shop` merchants (no
  persistence) keep melting affixed items — extending them is not worth it.
- **UUID regenerates on purchase** (`items.Item.UUID` is `yaml:"-"`), so a
  reloaded-then-bought item gets a fresh identity — correct, since it's a new
  item in the buyer's hands.
- **Relist uses full `AffixValue`** (`× 1.0`); the shop's margin is the half it
  underpaid at buy (`× ShopBuyRatio`). No scarcity curve (unique items).
- **In-game smoke** still recommended before prod: the ShopInventory buy/sell
  paths are only reachable in go-test via `RegisterShop` fixtures; a live shop
  round-trip (sell an oasis item, see it on the shelf, rebuy it) is the final
  confidence check.
