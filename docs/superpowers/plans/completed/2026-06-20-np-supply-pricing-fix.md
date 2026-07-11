# NP Supply — Fresh-Entry Pricing Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop crafted / caravan-delivered goods (`StockEntry.RestockQty == 0`) from pricing at ~5× ("not for sale") by decoupling the pricing baseline from the ticker restock quantity.

**Architecture:** A new tunable `Balance.DefaultPricingBaselineQty` (default 3) flows through `PricingConfig.DefaultBaselineQty`. `EffectiveRestock` takes that baseline as a parameter and returns it (instead of `MaxStock/2`) when `RestockQty == 0`. The field's *ticker* meaning ("no auto-refill") is untouched — no restock-mechanic code calls `EffectiveRestock` (only the buy-pricing and list-display paths do).

**Tech Stack:** Go, `testify/assert`. Spec: `docs/superpowers/specs/2026-06-20-np-supply-pricing-fix-design.md`.

**Branch:** continue on `feature/new-plymouth-planning` (or a fresh `feature/np-supply-pricing-fix` if you prefer to isolate the engine change — author's choice at execution).

---

### Task 1: Add the `DefaultPricingBaselineQty` Balance config knob

**Files:**
- Modify: `internal/configs/config.balance.go` (Balance struct, near `ShopMaterialReserve` ~line 477)
- Modify: `internal/configs/config.balance.shops.go` (`validateShops`, the "SHOP ECONOMY" block ~line 24)
- Test: `internal/configs/config.balance_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/configs/config.balance_test.go` (match the file's existing import block; it is `package configs` and uses `testify/assert`):

```go
func TestValidateShops_DefaultPricingBaselineQtyDefault(t *testing.T) {
	b := &Balance{}
	b.validateShops()
	assert.Equal(t, 3, int(b.DefaultPricingBaselineQty))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/configs/ -run TestValidateShops_DefaultPricingBaselineQtyDefault -v`
Expected: FAIL to compile — `b.DefaultPricingBaselineQty` undefined.

- [ ] **Step 3: Add the struct field**

In `internal/configs/config.balance.go`, immediately after the `ShopMaterialReserve` field (~line 477), add:

```go
	DefaultPricingBaselineQty ConfigInt `yaml:"DefaultPricingBaselineQty,omitempty"` // Pricing baseline (scarcity-curve denominator) for stock entries with RestockQty==0, e.g. crafted/caravan-delivered goods (default 3)
```

- [ ] **Step 4: Add the default in `validateShops`**

In `internal/configs/config.balance.shops.go`, inside `validateShops()`, in the "SHOP ECONOMY" block right after the `CrafterIngredientReservePct` default (~line 24), add:

```go
	if b.DefaultPricingBaselineQty < 1 {
		b.DefaultPricingBaselineQty = 3
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/configs/ -run TestValidateShops_DefaultPricingBaselineQtyDefault -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.shops.go internal/configs/config.balance_test.go
git commit -m "feat(shops): add DefaultPricingBaselineQty balance knob (default 3)"
```

---

### Task 2: Carry the baseline on `PricingConfig`

**Files:**
- Modify: `internal/shops/pricing.go` (`PricingConfig` struct:10, `PricingConfigFromBalance`:19, `DefaultPricingConfig`:38)
- Test: `internal/shops/pricing_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/shops/pricing_test.go`:

```go
func TestDefaultPricingConfig_BaselineQty(t *testing.T) {
	assert.Equal(t, 3, DefaultPricingConfig().DefaultBaselineQty)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/shops/ -run TestDefaultPricingConfig_BaselineQty -v`
Expected: FAIL to compile — `DefaultBaselineQty` undefined on `PricingConfig`.

- [ ] **Step 3: Add the field and wire it**

In `internal/shops/pricing.go`, add the field to `PricingConfig` (after `AbundanceThreshold`):

```go
	AbundanceThreshold  float64 // Stock/restock ratio for full abundance (default 3.0)
	DefaultBaselineQty  int     // Pricing baseline for RestockQty==0 entries (default 3)
```

In `DefaultPricingConfig()`, add the field:

```go
	return PricingConfig{
		BuyRatio:           0.50,
		PriceFloor:         0.25,
		PriceCeiling:       5.0,
		AbundanceThreshold: 3.0,
		DefaultBaselineQty: 3,
	}
```

In `PricingConfigFromBalance()`, after the `ShopAbundanceThreshold` block (before `return cfg`):

```go
	if int(b.DefaultPricingBaselineQty) > 0 {
		cfg.DefaultBaselineQty = int(b.DefaultPricingBaselineQty)
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/shops/ -run TestDefaultPricingConfig_BaselineQty -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/shops/pricing.go internal/shops/pricing_test.go
git commit -m "feat(shops): carry DefaultBaselineQty on PricingConfig"
```

---

### Task 3: Change `EffectiveRestock` + update both callers

`EffectiveRestock` and its in-package caller `tryPurchaseFromInventory` are both in `package actions`, so the signature change and that caller must change together to keep the package compiling. The cross-package caller `buildShopStockFromInventory` (`internal/usercommands`) is fixed in the same task so the full build stays green for the commit.

**Files:**
- Modify: `internal/actions/buy.go` (`EffectiveRestock`:499, `tryPurchaseFromInventory`:539)
- Modify: `internal/usercommands/list.go` (`buildShopStockFromInventory`:112)
- Test: `internal/actions/buy_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/actions/buy_test.go` (match the file's existing import block; add `"github.com/GoMudEngine/GoMud/internal/shops"` if not already imported):

```go
func TestEffectiveRestock_UsesRestockQtyWhenPositive(t *testing.T) {
	e := &shops.StockEntry{RestockQty: 7, MaxStock: 20}
	assert.Equal(t, 7, EffectiveRestock(e, 3))
}

func TestEffectiveRestock_UsesDefaultBaselineWhenZero(t *testing.T) {
	// The bug: previously returned MaxStock/2 (=10). Now returns the baseline.
	e := &shops.StockEntry{RestockQty: 0, MaxStock: 20}
	assert.Equal(t, 3, EffectiveRestock(e, 3))
}

func TestEffectiveRestock_ClampsBaselineToOne(t *testing.T) {
	e := &shops.StockEntry{RestockQty: 0, MaxStock: 20}
	assert.Equal(t, 1, EffectiveRestock(e, 0))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/actions/ -run TestEffectiveRestock -v`
Expected: FAIL to compile — `EffectiveRestock` takes 1 arg, tests pass 2.

- [ ] **Step 3: Change `EffectiveRestock`**

In `internal/actions/buy.go`, replace the function (currently lines 496–508):

```go
// EffectiveRestock returns the normalizer (baseline) for pricing
// calculations. Materials use RestockQty; goods with no ticker restock
// (RestockQty==0) — crafted or caravan-delivered — use defaultBaseline so a
// low-volume stock is not priced as if perpetually scarce.
func EffectiveRestock(entry *shops.StockEntry, defaultBaseline int) int {
	if entry.RestockQty > 0 {
		return entry.RestockQty
	}
	if defaultBaseline < 1 {
		defaultBaseline = 1
	}
	return defaultBaseline
}
```

- [ ] **Step 4: Update the in-package caller**

In `internal/actions/buy.go`, `tryPurchaseFromInventory` (line 539), change:

```go
		restock := EffectiveRestock(entry)
```
to:
```go
		restock := EffectiveRestock(entry, cfg.DefaultBaselineQty)
```
(`cfg` is already built at line 514 via `shops.PricingConfigFromBalance()`.)

- [ ] **Step 5: Update the cross-package caller**

In `internal/usercommands/list.go`, `buildShopStockFromInventory` (line 112), change:

```go
		restock := actions.EffectiveRestock(&entry)
```
to:
```go
		restock := actions.EffectiveRestock(&entry, cfg.DefaultBaselineQty)
```
(`cfg` is already built at line 100 via `shops.PricingConfigFromBalance()`.)

- [ ] **Step 6: Run tests + full build**

Run: `go test ./internal/actions/ -run TestEffectiveRestock -v`
Expected: PASS

Run: `go build ./...`
Expected: builds clean (confirms both callers updated, no other callers exist).

- [ ] **Step 7: Commit**

```bash
git add internal/actions/buy.go internal/actions/buy_test.go internal/usercommands/list.go
git commit -m "fix(shops): price RestockQty==0 stock off DefaultBaselineQty, not MaxStock/2"
```

---

### Task 4: Regression tests documenting the pricing effect

Locks in the §1 design table from the spec so a future curve change can't silently reintroduce the bug.

**Files:**
- Test: `internal/shops/pricing_test.go`

- [ ] **Step 1: Write the tests**

Add to `internal/shops/pricing_test.go`:

```go
func TestScarcityMultiplier_Baseline3_ModestStockIsAffordable(t *testing.T) {
	cfg := DefaultPricingConfig()
	// 5 units at baseline 3 → ratio 1.67 → comfortably below base price.
	assert.Less(t, ScarcityMultiplier(5, 3, cfg), 1.5)
}

func TestScarcityMultiplier_Baseline3_NineUnitsHitsFloor(t *testing.T) {
	cfg := DefaultPricingConfig()
	// 9 units at baseline 3 → ratio 3.0 == AbundanceThreshold → floor.
	assert.Equal(t, cfg.PriceFloor, ScarcityMultiplier(9, 3, cfg))
}

func TestScarcityMultiplier_Baseline3_LastUnitBounded(t *testing.T) {
	cfg := DefaultPricingConfig()
	// A single unit still carries a premium, but never exceeds the ceiling.
	m := ScarcityMultiplier(1, 3, cfg)
	assert.Greater(t, m, 1.0)
	assert.LessOrEqual(t, m, cfg.PriceCeiling)
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/shops/ -run TestScarcityMultiplier_Baseline3 -v`
Expected: PASS (3 tests)

- [ ] **Step 3: Run the full shops + actions + configs suites**

Run: `go test ./internal/shops/ ./internal/actions/ ./internal/configs/`
Expected: all PASS (existing `TestScarcityMultiplier_*` unchanged).

- [ ] **Step 4: Commit**

```bash
git add internal/shops/pricing_test.go
git commit -m "test(shops): lock in baseline-3 pricing band for low-volume goods"
```

---

## Self-Review

**Spec coverage:**
- §1 config knob `DefaultPricingBaselineQty` default 3 → Task 1. ✅
- §1 threaded baseline (not hidden config read) → carried on `PricingConfig`, passed as param → Tasks 2–3. ✅
- §1 `EffectiveRestock` returns baseline when `RestockQty==0` → Task 3. ✅
- §1 both callers updated → Task 3 steps 4–5. ✅
- §3 test plan (EffectiveRestock cases + scarcity band + config default) → Tasks 1,3,4. ✅
- §2 NP wiring recipe → intentionally NOT in this plan (built during the Docks district build). ✅

**Placeholder scan:** none — every step has exact file:line, full code, and a runnable command.

**Type consistency:** `EffectiveRestock(entry *shops.StockEntry, defaultBaseline int) int` used identically in Task 3 def + both callers. `PricingConfig.DefaultBaselineQty int` defined in Task 2, consumed in Task 3 via `cfg.DefaultBaselineQty`. `Balance.DefaultPricingBaselineQty ConfigInt` defined Task 1, read in Task 2 via `int(b.DefaultPricingBaselineQty)`. Consistent.

> One execution note: confirm `internal/actions/buy_test.go` already imports `testify/assert` and add the `internal/shops` import if missing (Task 3 Step 1). If the file uses a different assertion style, match it.
