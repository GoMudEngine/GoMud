# Affixed Gear — Stage 2 (Sellability / Melt Path) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let players sell affix-scaled instance loot for gold scaled to its
affix value (`spec.Value × ShopBuyRatio`), WITHOUT un-protecting other custom-
spec items (enchanted / blob / uses-modified). The item is consumed on sale
("melt") — it is NOT added to shop stock (that's Stage 3's per-instance resale).

**Architecture:** Affixed items get an explicit `Item.Affixed` marker (set in
`GenerateAffixedItem`). The sell path (`internal/actions/sell.go`) treats an
affixed item as a special case: it is allowed past the `IsSpecial` reject and
the merchant probe, priced by a fixed spread off its (Stage-1 stamped) value,
and — for the melt path — its base-ID stock line is NOT incremented. `IsSpecial`
itself is untouched; every non-affixed custom-spec item stays blocked exactly as
today.

**Tech Stack:** Go. `internal/items`, `internal/actions/sell.go`, `internal/shops`.

**Spec:** `docs/superpowers/specs/2026-07-08-shops-trade-affixed-gear-design.md`
(Stage 2 row + "Pricing model for unique affixed items").

**Depends on:** Stage 1 (merged) — `spec.Value` on affixed items is already the
scaled `AffixValue`.

---

## Task 1: `Item.Affixed` marker

**Files:**
- Modify: `internal/items/items.go` (struct field)
- Modify: `internal/items/affixgen.go` (set it)
- Test: `internal/items/affixgen_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/items/affixgen_test.go`:

```go
func TestGenerateAffixedItem_SetsAffixedMarker(t *testing.T) {
	cleanup := SeedItemsForTest(map[int]*ItemSpec{
		9100: {ItemId: 9100, Name: "Test Torc", Type: Neck, Value: 85},
	})
	defer cleanup()

	it := GenerateAffixedItem(9100, 200, 7.0, 3.0)
	if !it.Affixed {
		t.Error("expected Affixed=true on a budgeted affixed item")
	}

	// A zero-budget generation returns a plain item — not marked affixed.
	plain := GenerateAffixedItem(9100, 0, 7.0, 3.0)
	if plain.Affixed {
		t.Error("expected Affixed=false when no affixes were applied (goldPaid 0)")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/ -run TestGenerateAffixedItem_SetsAffixedMarker -v`
Expected: FAIL — `it.Affixed undefined`.

- [ ] **Step 3: Add the field**

In `internal/items/items.go`, in the `Item` struct (near `Spec`), add:

```go
	Affixed       bool           `yaml:"affixed,omitempty"`      // Instance-loot affix-scaled item (sellable + value-scaled; distinct from enchanted)
```

- [ ] **Step 4: Set it in `GenerateAffixedItem`**

In `internal/items/affixgen.go`, in the block that runs only when bonuses were
applied (right where `item.Spec = &specCopy` and the value stamp are), add:

```go
	specCopy.Value = AffixValue(specCopy, baseSpec, goldPerPoint)

	item.Spec = &specCopy
	item.Affixed = true
```

The two early-return paths (`rawBudget <= 0`, `len(eligible) == 0`) return the
plain `item` — leave them; a no-affix item stays `Affixed=false`.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/items/ -run 'TestGenerateAffixedItem' -v`
Expected: PASS (marker test + the Stage-1 stamping test).

- [ ] **Step 6: Commit**

```bash
git add internal/items/items.go internal/items/affixgen.go internal/items/affixgen_test.go
git commit -m "feat(items): Item.Affixed marker set by GenerateAffixedItem"
```

---

## Task 2: Affixed sell-price helper + merchant-probe allowance

The merchant probe (`findBuyingMerchant`, `sell.go:~95`) decides whether any
merchant will buy the item. It must recognize affixed items (priced by the fixed
spread) so an affixed item finds a buyer.

**Files:**
- Modify: `internal/actions/sell.go`
- Test: `internal/actions/sell_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/actions/sell_test.go` (reuse the existing test helpers there
— it already stands up a merchant with a shop; mirror an existing test's setup):

```go
// affixedSellPrice must be spec.Value * ShopBuyRatio (fixed spread), independent
// of scarcity — the Stage-2 melt price for instance loot.
func TestAffixedSellPrice(t *testing.T) {
	cfg := shops.DefaultPricingConfig() // BuyRatio 0.50
	it := items.Item{ItemId: 1, Affixed: true, Spec: &items.ItemSpec{Value: 400}}
	got := affixedSellPrice(it, cfg)
	if got != 200 { // 400 * 0.50
		t.Errorf("affixedSellPrice = %d; want 200", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestAffixedSellPrice -v`
Expected: FAIL — `undefined: affixedSellPrice`.

- [ ] **Step 3: Implement the helper + probe allowance**

In `internal/actions/sell.go`, add the helper (near the top-level funcs):

```go
// affixedSellPrice is the fixed-spread price a shop pays for an affix-scaled
// instance item: its (Stage-1 stamped) value times the buy/sell spread. It
// deliberately bypasses the scarcity curve and the legacy 25% cap — unique
// affixed gear is priced on value, not commodity stock levels.
func affixedSellPrice(item items.Item, cfg shops.PricingConfig) int {
	price := int(math.Ceil(float64(item.GetSpec().Value) * cfg.BuyRatio))
	if price < 1 {
		price = 1
	}
	return price
}
```

Then in `findBuyingMerchant`, give affixed items a value-based probe so any
merchant will buy them. Replace the probe-value block:

```go
		var probeValue int
		if probe.Affixed {
			probeValue = affixedSellPrice(probe, shops.PricingConfigFromBalance())
		} else if shopInv != nil {
			cfg := shops.PricingConfigFromBalance()
			wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
			offer := shops.EvaluateBuyRules(probe, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
			probeValue = offer.Price
		} else {
			probeValue = mob.GetSellPrice(probe)
		}
```

Add `"math"` to the imports if not present.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/actions/ -run TestAffixedSellPrice -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/sell.go internal/actions/sell_test.go
git commit -m "feat(sell): affixedSellPrice fixed-spread helper + merchant-probe allowance"
```

---

## Task 3: Sell an affixed item (allow past IsSpecial, price, melt)

Wire the affixed case into the sell-execution function: allow past the
`IsSpecial` reject, price via `affixedSellPrice`, and skip the base-ID stock
increment (melt). Enchanted/blob/uses items stay rejected.

**Files:**
- Modify: `internal/actions/sell.go` (the sell-execution function, ~L238–330)
- Test: `internal/actions/sell_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/actions/sell_test.go`:

Uses the real API confirmed in `sell_test.go`: `Sell(seller Actor, opts
SellOptions) SellResult`, the `newSellerActor(t, isPlayer, itemIds...)` helper,
and `sellTestItemId` (a base item the harness seeds as sellable at a
merchant-with-shop in room 1). The affixed item is injected onto the seller's
character after building the actor.

```go
// TestSell_AffixedItem_MeltPath: an affixed item sells for spec.Value*BuyRatio,
// leaves the player's inventory, and (Stage 2) is NOT added to shop stock.
func TestSell_AffixedItem_MeltPath(t *testing.T) {
	cleanup := seedSellRegistries(t) // whatever the existing sell tests use; see file top
	defer cleanup()

	seller := newSellerActor(t, true) // player, no starting items
	char := seller.GetCharacter()

	// Inject an affixed instance of the sellable base item (Value 400, marked).
	affixed := items.New(sellTestItemId)
	affixed.Affixed = true
	affixed.Spec = &items.ItemSpec{Value: 400, Type: items.Weapon}
	char.StoreItem(affixed)
	goldBefore := char.Gold

	res := Sell(seller, SellOptions{ItemName: affixed.GetSpec().NameSimple, Quantity: 1})

	if char.Gold != goldBefore+200 { // 400 * ShopBuyRatio 0.50
		t.Errorf("gold delta = %d; want 200 (res=%+v)", char.Gold-goldBefore, res)
	}
	if _, has := char.FindInBackpack(affixed.GetSpec().NameSimple); has {
		t.Error("affixed item should be gone from the backpack after sale")
	}
}

// Control: a non-affixed custom-spec item (Spec!=nil, Affixed=false) stays
// rejected — proves the IsSpecial protection is intact.
func TestSell_NonAffixedCustomSpec_Rejected(t *testing.T) {
	cleanup := seedSellRegistries(t)
	defer cleanup()

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	special := items.New(sellTestItemId)
	special.Spec = &items.ItemSpec{Value: 400} // Affixed stays false
	char.StoreItem(special)
	goldBefore := char.Gold

	Sell(seller, SellOptions{ItemName: special.GetSpec().NameSimple, Quantity: 1})
	if char.Gold != goldBefore {
		t.Error("a non-affixed custom-spec item must not sell")
	}
}
```

(Read `sell_test.go`'s exact seed helper name + `sellTestItemId` value and the
`Actor` accessor for the character — the assertions are the contract: affixed →
paid `Value×BuyRatio`, removed, not stocked; non-affixed custom-spec → rejected,
no payout.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestSell_AffixedItem_MeltPath -v`
Expected: FAIL — affixed item currently rejected by the `IsSpecial` guard.

- [ ] **Step 3: Allow affixed past IsSpecial + price it**

In the sell-execution function, change the reject guard (`sell.go:242`):

```go
	if item.IsSpecial() && !item.Affixed {
		merchantSay(room, mob, "I'm afraid I don't buy those.")
		return 0, SellStopRejected
	}
```

Then, at the top of the pricing block (before the `if shopInv != nil` branch),
short-circuit affixed items:

```go
	var sellValue int
	var buyReason string
	if item.Affixed {
		sellValue = affixedSellPrice(item, shops.PricingConfigFromBalance())
	} else if shopInv != nil {
		// ... existing EvaluateBuyRules block unchanged ...
	} else {
		sellValue = mob.GetSellPrice(item)
	}
```

- [ ] **Step 4: Skip base-ID stocking for affixed (melt)**

Wrap the merchant-side stock update (`sell.go:~302 if shopInv != nil { ... }`)
so affixed items are not stocked as a base ItemId:

```go
	// Stock update (merchant side). Affixed items are melted in Stage 2 — they
	// are NOT stocked as a base ItemId (Stage 3 adds per-instance resale stock).
	if shopInv != nil && !item.Affixed {
		// ... existing AddStockAtRound logic unchanged ...
	}
```

Leave the gold-affordability gate, `char.Gold += sellValue`, `char.RemoveItem`,
and the `ItemOwnership`/`EquipmentChange` events unchanged — they already handle
the payout + removal correctly for affixed items.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/actions/ -run 'TestSell|TestAffixed' -v`
Expected: PASS — affixed melt path works; existing sell tests still green; the
non-affixed-custom-spec control is still rejected.

- [ ] **Step 6: Build + broader suite**

Run: `go build ./... && go test ./internal/actions/ ./internal/items/ ./internal/mobs/`
Expected: build OK, all green.

- [ ] **Step 7: Commit**

```bash
git add internal/actions/sell.go internal/actions/sell_test.go
git commit -m "feat(sell): affixed instance loot is sellable (melt path, fixed-spread price)"
```

---

## Definition of Done (Stage 2)

- Affixed items carry `Item.Affixed = true`.
- A player can sell an affixed item for `spec.Value × ShopBuyRatio`; the item
  leaves inventory and is NOT added to shop stock (melt).
- `IsSpecial` is unchanged; enchanted / blob / uses-modified items remain
  unsellable (regression asserted).
- `go build ./...` clean; `actions` / `items` / `mobs` suites green.

## Divergences From Spec (this stage)

- Affixed pricing bypasses BOTH `EvaluateBuyRules` (scarcity) and
  `GetSellPrice` (25% cap) via `affixedSellPrice` — the fixed spread the spec's
  pricing-model section requires for unique items.
- Melt (no stocking) is the Stage-2 behavior; Stage 3 replaces the skipped
  `AddStockAtRound` with real per-instance stock so the shop can resell.

## Notes for Stage 3

- The skipped stocking in Task 3 Step 4 is the exact seam Stage 3 fills: instead
  of `if shopInv != nil && !item.Affixed`, Stage 3 adds an
  `else if item.Affixed` that stores the full per-instance `items.Item` on a
  new `ShopItem` field at `affixedSellPrice`, and the buy path lists/sells it at
  `spec.Value × 1.0`.
