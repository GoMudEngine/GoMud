# Mob Aliveness 2.1 — Mob `buy` Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a consolidated `actions.Buy(buyer, opts)` function used by both player and mob wrappers, dropping the merc and pet purchase paths and closing the pre-existing carry-capacity gap on the player side. Add an `internal/mobcommands/buy.go` wrapper so behavior trees can dispatch `buy <item>`.

**Architecture:** Lift the internal helpers of `internal/usercommands/buy.go` (tryPurchase, tryPurchaseFromInventory, validatePurchase, executePurchaseItem, executePurchaseBuff, effectiveRestock, sendMerchantMessage) into `internal/actions/buy.go`, reshaping every `*users.UserRecord` reference to flow through the existing `actions.Actor` interface. Player and mob wrappers become thin shims (~30 lines each). The merc and pet sale-type branches are deleted along with `executePurchaseMerc` and `executePurchasePet`.

**Tech Stack:** Go 1.21+, existing `internal/actions` Actor abstraction, existing `internal/shops` pricing/persistence layer, existing `internal/events` bus.

**Spec:** `docs/superpowers/specs/2026-05-11-mob-aliveness-2.1-mob-buy-command-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/actions/buy.go` | NEW — consolidated `Buy(buyer Actor, opts BuyOptions) BuyResult` + all internal helpers |
| `internal/actions/buy_test.go` | NEW — driving the consolidated core via both UserActor and MobActor |
| `internal/usercommands/buy.go` | MODIFY — collapse to thin wrapper; delete merc + pet code paths and their helpers |
| `internal/mobcommands/buy.go` | NEW — thin mob wrapper |
| `internal/mobcommands/mobcommands.go` | MODIFY — register `"buy"` in `mobCommands` map |
| `internal/mobcommands/buy_test.go` | NEW — TryCommand-level smoke test |
| `internal/usercommands/usercommands_test.go` | MODIFY — drop any merc/pet purchase assertions; add overburdened-gate setup if needed |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY — mark 2.1 Done, roll-up to 8/40 |

---

## Task 1: Skeleton — actions.Buy stub + API types

**Files:**
- Create: `internal/actions/buy.go`
- Create: `internal/actions/buy_test.go`

Set up the new file with API types and a stub that returns `Reason="no_request"` for empty input. This proves the package import wiring works and the public surface compiles before any logic lands.

- [ ] **Step 1: Create `internal/actions/buy.go` with skeleton**

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// BuyOptions controls how a purchase is attempted.
type BuyOptions struct {
	// Request is the raw "rest" string, e.g. "5 iron ingot from marko".
	// Quantity prefix and "from <merchant>" suffix are parsed inside Buy,
	// so wrappers pass through whatever the player typed.
	Request string

	// TargetMerchantUserId, when > 0, restricts merchant selection to a
	// specific user merchant. Wrappers may set this directly for cases
	// where the caller has already resolved the target.
	TargetMerchantUserId int

	// TargetMerchantMobInstanceId, when > 0, restricts merchant selection
	// to a specific mob merchant.
	TargetMerchantMobInstanceId int
}

// BuyResult is the outcome of an attempted purchase.
type BuyResult struct {
	Success   bool   // at least one unit purchased
	Purchased int    // actual units purchased (may be < Requested)
	Requested int    // requested quantity (1 if unspecified)
	SaleType  string // "item" | "buff" | "" on failure
	Reason    string // populated on failure
}

// Failure-reason vocabulary returned in BuyResult.Reason.
const (
	BuyReasonNoRequest        = "no_request"
	BuyReasonNoMerchant       = "no_merchant"
	BuyReasonNoMatch          = "no_match"
	BuyReasonOutOfStock       = "out_of_stock"
	BuyReasonInsufficientGold = "insufficient_gold"
	BuyReasonMissingTradeItem = "missing_trade_item"
	BuyReasonOverburdened     = "overburdened"
	BuyReasonSelfTarget       = "self_target"
)

// Buy executes a purchase on behalf of buyer. See package context for
// the full flow.
func Buy(buyer Actor, opts BuyOptions) BuyResult {
	if opts.Request == "" {
		return BuyResult{Reason: BuyReasonNoRequest}
	}

	_ = rooms.FindMerchant // placeholder import use
	return BuyResult{Reason: BuyReasonNoRequest}
}
```

- [ ] **Step 2: Create the failing test**

```go
// internal/actions/buy_test.go
package actions

import "testing"

func TestBuy_EmptyRequest(t *testing.T) {
	result := Buy(nil, BuyOptions{Request: ""})
	if result.Success {
		t.Errorf("expected Success=false on empty request")
	}
	if result.Reason != BuyReasonNoRequest {
		t.Errorf("expected Reason=%q, got %q", BuyReasonNoRequest, result.Reason)
	}
}
```

- [ ] **Step 3: Run the test to verify it passes**

Run: `go test ./internal/actions/ -run TestBuy_EmptyRequest -v`
Expected: PASS

- [ ] **Step 4: Verify build is clean across the workspace**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/buy.go internal/actions/buy_test.go
git commit -m "$(cat <<'EOF'
feat(actions): scaffold consolidated Buy entry point and API types

Adds the public BuyOptions/BuyResult surface plus Reason constants.
Body returns no_request for empty input; full implementation lands
in subsequent tasks.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Lift fuzzy-match and price lookup helpers

**Files:**
- Modify: `internal/actions/buy.go`

Move the in-memory price-map construction and `util.FindMatchIn` call out of the legacy `tryPurchase` into a small private helper that the lifted code can share. The helper is buyer-agnostic — it only reads from the merchant. Lifted from `internal/usercommands/buy.go:199-303`.

- [ ] **Step 1: Read the current legacy fuzzy-match block**

Look at `internal/usercommands/buy.go:199-343` (the body of `tryPurchase` up through the match-not-found rejection). Identify everything between "build nameToShopItem map" and "util.FindMatchIn".

- [ ] **Step 2: Add unexported helper to actions/buy.go**

Add inside `internal/actions/buy.go` (above `Buy`):

```go
import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// legacyShopCatalog enumerates the in-stock items + buffs offered by a
// legacy Character.Shop. Merc and pet sale types are intentionally NOT
// surfaced (spec 2.1 drops them).
type legacyShopCatalog struct {
	nameToShopItem map[string]characters.ShopItem
	itemNames      []string
	itemNamesFancy []string
	itemPrices     map[int]int
	buffNames      []string
	buffPrices     map[int]int
}

func buildLegacyCatalog(saleItems characters.Shop) legacyShopCatalog {
	cat := legacyShopCatalog{
		nameToShopItem: map[string]characters.ShopItem{},
		itemPrices:     map[int]int{},
		buffPrices:     map[int]int{},
	}

	for _, saleItem := range saleItems {
		if saleItem.ItemId > 0 {
			item := items.New(saleItem.ItemId)
			if item.ItemId == 0 {
				continue
			}
			cat.itemNames = append(cat.itemNames, item.GetSpec().Name)
			cat.itemNamesFancy = append(cat.itemNamesFancy, item.DisplayName())
			cat.nameToShopItem[item.GetSpec().Name] = saleItem

			price := saleItem.Price
			if price == 0 {
				price = item.GetSpec().Value
			} else if price < 0 {
				price = 0
			}
			cat.itemPrices[saleItem.ItemId] = price
			continue
		}
		if saleItem.BuffId > 0 {
			buffInfo := buffs.GetBuffSpec(saleItem.BuffId)
			if buffInfo == nil {
				continue
			}
			cat.buffNames = append(cat.buffNames, buffInfo.Name)
			cat.nameToShopItem[buffInfo.Name] = saleItem

			price := saleItem.Price
			if price == 0 {
				price = 1000
			} else if price < 0 {
				price = 0
			}
			cat.buffPrices[saleItem.BuffId] = price
			continue
		}
		// Merc / pet entries on legacy shops are skipped — see spec 2.1.
	}
	return cat
}

// allNames returns the union of item + buff display names in the catalog
// for fuzzy matching. Merc/pet names are intentionally excluded.
func (c *legacyShopCatalog) allNames() []string {
	all := make([]string, 0, len(c.itemNames)+len(c.buffNames))
	all = append(all, c.itemNames...)
	all = append(all, c.buffNames...)
	return all
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/actions/...`
Expected: build passes. (The package-level imports may not all be referenced yet; if `mobs` or `users` is unused, drop the import lines until later tasks reintroduce them.)

- [ ] **Step 4: Add a unit test for the catalog**

Append to `internal/actions/buy_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

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
```

- [ ] **Step 5: Run test, commit**

Run: `go test ./internal/actions/ -run TestBuildLegacyCatalog -v`
Expected: PASS.

```bash
git add internal/actions/buy.go internal/actions/buy_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift legacy shop catalog builder into actions.Buy

Extracts the price-map + name-list construction from
usercommands.tryPurchase into a buyer-agnostic helper. Merc and
pet sale types are deliberately not surfaced — per chunk 2.1 they
are dropped from actions.Buy entirely.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Merchant resolution + no-merchant short circuit

**Files:**
- Modify: `internal/actions/buy.go`

Implement Steps 1 (parse `from`) and 2 (resolve merchant list) of the End-to-end flow. Quantity parsing comes in Task 8 — for now treat the whole `Request` as the item name and `quantity=1`.

- [ ] **Step 1: Add merchant resolution + self-target guard to Buy()**

Replace the stub body of `Buy` in `internal/actions/buy.go`:

```go
import (
	"strconv"
	"strings"
)

func Buy(buyer Actor, opts BuyOptions) BuyResult {
	req := strings.TrimSpace(opts.Request)
	if req == "" {
		return BuyResult{Reason: BuyReasonNoRequest, Requested: 1}
	}

	room := buyer.GetRoom()
	if room == nil {
		return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
	}

	// Parse trailing "from <name>" clause to resolve a specific merchant.
	itemRequest := req
	targetUserId := opts.TargetMerchantUserId
	targetMobInstanceId := opts.TargetMerchantMobInstanceId

	args := util.SplitButRespectQuotes(strings.ToLower(req))
	if len(args) >= 3 && args[len(args)-2] == "from" {
		mercName := args[len(args)-1]
		exclude := ResolveTargetOptions{}
		if buyer.IsPlayer() {
			exclude.ExcludeUserId = buyer.GetUserId()
		}
		target, terr := ResolveTargetActor(room, mercName, exclude)
		if terr == nil {
			if target.IsPlayer() {
				targetUserId = target.(*UserActor).User.UserId
			} else {
				targetMobInstanceId = target.(*MobActor).Mob.InstanceId
			}
			itemRequest = strings.Join(args[:len(args)-2], " ")
		} else if buyer.IsPlayer() {
			// Self-targeting collapses to NotFound under ExcludeUserId;
			// check explicitly so we can return BuyReasonSelfTarget.
			if pId, _ := room.FindByName(mercName); pId == buyer.GetUserId() {
				buyer.SendText("You can't buy from yourself.")
				return BuyResult{Reason: BuyReasonSelfTarget, Requested: 1}
			}
			buyer.SendText("Visit a merchant to purchase objects or services.")
			return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
		} else {
			return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
		}
	}

	merchantPlayers := room.GetPlayers(rooms.FindMerchant)
	merchantMobs := room.GetMobs(rooms.FindMerchant)

	if len(merchantPlayers) == 0 && len(merchantMobs) == 0 {
		if buyer.IsPlayer() {
			buyer.SendText("Visit a merchant to purchase objects or services.")
		}
		return BuyResult{Reason: BuyReasonNoMerchant, Requested: 1}
	}

	// Task 4 and onwards: per-merchant purchase loop.
	_ = itemRequest
	_ = targetUserId
	_ = targetMobInstanceId
	_ = strconv.Atoi // placeholder until Task 8

	return BuyResult{Reason: BuyReasonNoMatch, Requested: 1}
}
```

- [ ] **Step 2: Add failing tests for the merchant-resolution paths**

Append to `internal/actions/buy_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

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
```

(Note: testing player buyers requires a `UserRecord` fixture; if existing
test helpers in `internal/actions/economy_test.go` or
`combat_test.go` already build one, reuse it in later tasks. For now
the mob-buyer fixture is enough to exercise the no-merchant path.)

- [ ] **Step 3: Run the tests**

Run: `go test ./internal/actions/ -run 'TestBuy_' -v`
Expected: `TestBuy_EmptyRequest` PASS, `TestBuy_NoMerchant` PASS, `TestBuildLegacyCatalog_SkipsMercAndPet` PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/buy.go internal/actions/buy_test.go
git commit -m "$(cat <<'EOF'
feat(actions): resolve 'from <merchant>' clause and short-circuit on no merchant

Implements the parse step and merchant resolution from the chunk
2.1 end-to-end flow. Player buyers get a buyer-facing message;
mob buyers silently return BuyReasonNoMerchant. Self-target case
returns BuyReasonSelfTarget.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Lift validatePurchase with encumbrance gate

**Files:**
- Modify: `internal/actions/buy.go`

Move `validatePurchase` from `usercommands/buy.go:375-499` into the actions package. Replace every `*users.UserRecord` reference with `Actor` calls. Drop the merc-specific charm-cap block. Add the new encumbrance gate **first** in the validation order.

- [ ] **Step 1: Add validatePurchase to actions/buy.go**

```go
// purchaseContext holds the validated state of a purchase ready
// for execution.
type purchaseContext struct {
	matchedShopItem characters.ShopItem
	price           int
	tradeInString   string
}

// validatePurchase runs every pre-side-effect check, then applies
// side effects (destock, gold deduction, trade-item consume) only
// when all checks pass. Returns ok=false plus a populated Reason
// when any check fails; on failure no state is mutated.
func validatePurchase(
	buyer Actor,
	shopMob *mobs.Mob,
	shopUser *users.UserRecord,
	matchedShopItem characters.ShopItem,
	itemPrices map[int]int,
	buffPrices map[int]int,
) (purchaseContext, string, bool) {

	char := buyer.GetCharacter()

	// (1) Encumbrance gate — item purchases only.
	if matchedShopItem.ItemId > 0 {
		newItm := items.New(matchedShopItem.ItemId)
		weight := newItm.GetSpec().Weight
		if char.GetCarriedWeight()+weight > char.CarryCapacity() {
			if buyer.IsPlayer() {
				buyer.SendText("You can't carry any more.")
			}
			return purchaseContext{}, BuyReasonOverburdened, false
		}
	}

	// (2) Stock check.
	if !matchedShopItem.Available() {
		if shopMob != nil {
			shopMob.Command(`say I don't have that for sale right now.`)
		} else if shopUser != nil && buyer.IsPlayer() {
			buyer.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> doesn't have that for sale right now.`, shopUser.Character.Name))
		}
		return purchaseContext{}, BuyReasonOutOfStock, false
	}

	// (3) Price lookup.
	price := 0
	if matchedShopItem.ItemId > 0 {
		price = itemPrices[matchedShopItem.ItemId]
	} else if matchedShopItem.BuffId > 0 {
		price = buffPrices[matchedShopItem.BuffId]
	}

	// (4) Gold check.
	if char.Gold < price {
		sendMerchantMessage(buyer, shopMob, shopUser,
			`say You don't have enough gold for that.`,
			`You don't have enough gold for that.`)
		return purchaseContext{}, BuyReasonInsufficientGold, false
	}

	// (5) Trade-item check.
	tradeItemName := ""
	if matchedShopItem.TradeItemId > 0 {
		tradeItm := items.New(matchedShopItem.TradeItemId)
		tradeItemName = tradeItm.Name()
		if _, found := char.FindInBackpack(tradeItemName); !found {
			if buyer.IsPlayer() {
				buyer.SendText(fmt.Sprintf(`You must have a <ansi fg="itemname">%s</ansi> to trade for that.`, tradeItm.DisplayName()))
			}
			return purchaseContext{}, BuyReasonMissingTradeItem, false
		}
	}

	// All checks passed — apply side effects.
	if shopMob != nil {
		if !shopMob.Character.Shop.Destock(matchedShopItem) {
			shopMob.Command(`say I don't have that item right now.`)
			return purchaseContext{}, BuyReasonOutOfStock, false
		}
	} else if shopUser != nil {
		if !shopUser.Character.Shop.Destock(matchedShopItem) {
			if buyer.IsPlayer() {
				buyer.SendText(`That's not for sale.`)
			}
			return purchaseContext{}, BuyReasonOutOfStock, false
		}
	}

	if buyer.IsPlayer() {
		events.AddToQueue(events.EquipmentChange{
			UserId:     buyer.GetUserId(),
			GoldChange: -price,
		})
	}

	char.Gold -= price
	if shopMob != nil {
		shopMob.Character.Gold += 1 // legacy +1 cheat preserved
	} else if shopUser != nil {
		shopUser.Character.Gold += price
		events.AddToQueue(events.EquipmentChange{
			UserId:     shopUser.UserId,
			GoldChange: price,
		})
	}

	tradeInString := ""
	if price > 0 {
		tradeInString = fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, price)
	}
	if tradeItemName != "" {
		if itm, found := char.FindInBackpack(tradeItemName); found {
			char.RemoveItem(itm)
			if buyer.IsPlayer() {
				events.AddToQueue(events.ItemOwnership{
					UserId: buyer.GetUserId(),
					Item:   itm,
					Gained: false,
				})
			} else {
				events.AddToQueue(events.ItemOwnership{
					MobInstanceId: buyer.GetMobInstanceId(),
					Item:          itm,
					Gained:        false,
				})
			}

			if tradeInString != "" {
				tradeInString += fmt.Sprintf(` and a <ansi fg="itemname">%s</ansi>`, itm.DisplayName())
			} else {
				tradeInString = fmt.Sprintf(`a <ansi fg="itemname">%s</ansi>`, itm.DisplayName())
			}
		}
	}
	if tradeInString == "" {
		tradeInString = "nothing"
	}

	return purchaseContext{
		matchedShopItem: matchedShopItem,
		price:           price,
		tradeInString:   tradeInString,
	}, "", true
}

// sendMerchantMessage delivers a message to the buyer, branching on
// mob vs player merchant. Mob merchants speak; player merchants send
// to the buyer directly.
func sendMerchantMessage(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord, mobMsg string, userMsg string) {
	if shopMob != nil {
		shopMob.Command(mobMsg)
	} else if shopUser != nil && buyer.IsPlayer() {
		buyer.SendText(userMsg)
	}
}
```

Add the `events` and `fmt` imports if missing.

- [ ] **Step 2: Build**

Run: `go build ./internal/actions/...`
Expected: build passes.

- [ ] **Step 3: Add a unit test for the encumbrance gate (pre-side-effect)**

Append to `internal/actions/buy_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestValidatePurchase_OverburdenedBlocksPreSideEffect(t *testing.T) {
	// Fabricate a buyer whose Character has carriedWeight >= capacity.
	// The simplest version: create a Mob with an item that pushes weight
	// past capacity, then attempt a purchase of another item.
	m := &mobs.Mob{}
	m.Character.Strength = 1 // tiny capacity
	startingGold := 999
	m.Character.Gold = startingGold

	// Pre-stuff inventory so we are at/above capacity.
	heavy := items.New(20000) // assumes itemId 20000 exists in test data
	if heavy.ItemId == 0 {
		t.Skip("test data fixture missing itemId 20000")
	}
	m.Character.Items = append(m.Character.Items, heavy)

	buyer := &MobActor{Mob: m}

	saleItem := characters.ShopItem{ItemId: 20000, Price: 1, Quantity: 1, QuantityMax: 1}
	itemPrices := map[int]int{20000: 1}

	_, reason, ok := validatePurchase(buyer, nil, nil, saleItem, itemPrices, nil)

	if ok {
		t.Fatalf("expected validatePurchase to reject overburdened buyer")
	}
	if reason != BuyReasonOverburdened {
		t.Errorf("reason = %q, want %q", reason, BuyReasonOverburdened)
	}
	if m.Character.Gold != startingGold {
		t.Errorf("buyer gold should not be deducted on overburdened rejection; got %d want %d",
			m.Character.Gold, startingGold)
	}
	if saleItem.Quantity != 1 {
		t.Errorf("shop stock should not be destocked on overburdened rejection; got %d", saleItem.Quantity)
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/actions/ -run TestValidatePurchase -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/buy.go internal/actions/buy_test.go
git commit -m "$(cat <<'EOF'
feat(actions): lift validatePurchase with new encumbrance gate

Moves the side-effect-applying validation step into actions and
adds the chunk 2.1 carry-capacity check before any state mutates.
Player-side previously had no encumbrance gate at purchase; this
closes that gap symmetrically for both buyer types.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Lift tryPurchase (legacy backend, item + buff)

**Files:**
- Modify: `internal/actions/buy.go`

Move `tryPurchase` from `usercommands/buy.go:199-373` into actions. Drop the merc and pet name pools, dispatch only items and buffs to execution.

- [ ] **Step 1: Add tryPurchaseLegacy to actions/buy.go**

```go
// tryPurchaseLegacy attempts a single purchase against a merchant
// backed by the legacy Character.Shop. Returns the populated
// BuyResult on either success or failure; non-empty Reason on
// failure means the outer Buy loop should try the next merchant.
func tryPurchaseLegacy(buyer Actor, request string, shopMob *mobs.Mob, shopUser *users.UserRecord) BuyResult {
	var saleItems characters.Shop
	if shopMob != nil {
		saleItems = shopMob.Character.Shop.GetInstock()
	} else if shopUser != nil {
		saleItems = shopUser.Character.Shop.GetInstock()
	}

	cat := buildLegacyCatalog(saleItems)

	match, closeMatch := util.FindMatchIn(request, cat.allNames()...)
	if match == "" {
		match = closeMatch
	}
	if match == "" {
		if shopMob != nil {
			extraSay := ""
			if len(cat.itemNamesFancy) > 0 {
				randSelection := util.Rand(len(cat.itemNamesFancy))
				extraSay = fmt.Sprintf(` Any interest in this <ansi fg="itemname">%s</ansi>?`, cat.itemNamesFancy[randSelection])
			} else if len(cat.buffNames) > 0 {
				randSelection := util.Rand(len(cat.buffNames))
				extraSay = fmt.Sprintf(` Maybe you would enjoy this %s enchantment?`, cat.buffNames[randSelection])
			}
			shopMob.Command(`say Sorry, I can't offer that right now.` + extraSay)
		}
		return BuyResult{Reason: BuyReasonNoMatch}
	}

	ctx, reason, ok := validatePurchase(buyer, shopMob, shopUser, cat.nameToShopItem[match], cat.itemPrices, cat.buffPrices)
	if !ok {
		return BuyResult{Reason: reason}
	}

	if ctx.matchedShopItem.ItemId > 0 {
		executePurchaseItem(buyer, shopMob, shopUser, ctx.matchedShopItem, ctx.price, ctx.tradeInString)
		return BuyResult{Success: true, Purchased: 1, SaleType: "item"}
	}
	if ctx.matchedShopItem.BuffId > 0 {
		executePurchaseBuff(buyer, shopMob, shopUser, ctx.matchedShopItem, ctx.price, ctx.tradeInString)
		return BuyResult{Success: true, Purchased: 1, SaleType: "buff"}
	}

	// Merc/pet sale types are filtered out at catalog build time;
	// reaching this branch means malformed data.
	return BuyResult{Reason: BuyReasonNoMatch}
}

// executePurchaseItem lands a purchased item in the buyer's
// inventory and emits the buyer + room messages.
func executePurchaseItem(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord, matchedShopItem characters.ShopItem, price int, tradeInString string) {
	newItm := items.New(matchedShopItem.ItemId)

	if buyer.IsPlayer() {
		userId := buyer.GetUserId()
		if u := users.GetByUserId(userId); u != nil {
			u.PlaySound(`purchase`, `other`)
		}
		events.AddToQueue(events.ItemOwnership{
			UserId: userId,
			Item:   newItm,
			Gained: true,
		})
	} else {
		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: buyer.GetMobInstanceId(),
			Item:          newItm,
			Gained:        true,
		})
	}

	buyerName := buyer.GetName()

	if shopMob != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="itemname">%s</ansi> from <ansi fg="mobname">%s</ansi> for %s`, newItm.DisplayName(), shopMob.Character.Name, tradeInString))
			}
		}
		buyer.SendText(fmt.Sprintf(`You purchase the <ansi fg="itemname">%s</ansi> from <ansi fg="mobname">%s</ansi> for %s.`, newItm.DisplayName(), shopMob.Character.Name, tradeInString))
		buyer.SendRoomText(fmt.Sprintf(`<ansi fg="username">%s</ansi> purchases the <ansi fg="itemname">%s</ansi> from <ansi fg="mobname">%s</ansi>.`, buyerName, newItm.DisplayName(), shopMob.Character.Name), true)
	} else if shopUser != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="itemname">%s</ansi> from <ansi fg="username">%s</ansi> for %s.`, newItm.DisplayName(), shopUser.Character.Name, tradeInString))
			}
		}
		buyer.SendText(fmt.Sprintf(`You purchase the <ansi fg="itemname">%s</ansi> from <ansi fg="username">%s</ansi> for %s.`, newItm.DisplayName(), shopUser.Character.Name, tradeInString))
		shopUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> purchased the <ansi fg="itemname">%s</ansi> you were selling for %s.`, buyerName, newItm.DisplayName(), tradeInString))
		buyer.SendRoomText(fmt.Sprintf(`<ansi fg="username">%s</ansi> purchases the <ansi fg="itemname">%s</ansi> from <ansi fg="username">%s</ansi>.`, buyerName, newItm.DisplayName(), shopUser.Character.Name), true)
	}

	buyer.GetCharacter().StoreItem(newItm)
}

// executePurchaseBuff applies the bought buff to the buyer and
// emits the merchant emote follow-up.
func executePurchaseBuff(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord, matchedShopItem characters.ShopItem, price int, tradeInString string) {
	buffSpec := buffs.GetBuffSpec(matchedShopItem.BuffId)
	buyerName := buyer.GetName()

	if shopMob != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="buff">%s</ansi> enchantment from <ansi fg="mobname">%s</ansi> for %s`, buffSpec.Name, shopMob.Character.Name, tradeInString))
			}
		}
		buyer.SendText(fmt.Sprintf(`You pay %s to <ansi fg="mobname">%s</ansi>.`, tradeInString, shopMob.Character.Name))
		buyer.SendRoomText(fmt.Sprintf(`<ansi fg="username">%s</ansi> pays %s to <ansi fg="mobname">%s</ansi>.`, buyerName, tradeInString, shopMob.Character.Name), true)
		shopMob.Command(`emote mutters a soft incantation.`, 1)
	} else if shopUser != nil {
		if buyer.IsPlayer() {
			if u := users.GetByUserId(buyer.GetUserId()); u != nil {
				u.EventLog.Add(`shop`, fmt.Sprintf(`Purchased a <ansi fg="buff">%s</ansi> enchantment from  <ansi fg="username">%s</ansi> for %s`, buffSpec.Name, shopUser.Character.Name, tradeInString))
			}
		}
		buyer.SendText(fmt.Sprintf(`You pay %s to <ansi fg="username">%s</ansi>.`, tradeInString, shopUser.Character.Name))
		shopUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> pays you %s for an enchantment.`, buyerName, tradeInString))
		buyer.SendRoomText(fmt.Sprintf(`<ansi fg="username">%s</ansi> pays to <ansi fg="username">%s</ansi> for an enchantment.`, buyerName, shopUser.Character.Name), true)
		// player-merchant doesn't emote (matches existing behavior).
	}

	buyer.AddBuff(matchedShopItem.BuffId, "shop")

	if shopMob != nil {
		shopMob.Command(`say I've done what I can.`, 1)
	}
}
```

- [ ] **Step 2: Build**

Run: `go build ./internal/actions/...`
Expected: build passes.

- [ ] **Step 3: Wire tryPurchaseLegacy into Buy()**

Replace the placeholder bottom of `Buy` (the `_ = itemRequest` block) with:

```go
	// Iterate merchants — players first, then mobs.
	for _, uid := range merchantPlayers {
		if targetUserId > 0 && uid != targetUserId {
			continue
		}
		shopUser := users.GetByUserId(uid)
		if shopUser == nil {
			continue
		}

		result := tryPurchaseLegacy(buyer, itemRequest, nil, shopUser)
		if result.Success {
			postSuccessBookkeeping(buyer, nil, shopUser)
			result.Requested = 1
			return result
		}
	}

	for _, miid := range merchantMobs {
		if targetMobInstanceId > 0 && miid != targetMobInstanceId {
			continue
		}
		shopMob := mobs.GetInstance(miid)
		if shopMob == nil {
			continue
		}

		// Restock policy: legacy Character.Shop restocks on every
		// access; ShopInventory does its own restock internally.
		shopInv := shops.GetShopInventory(shopMob.Zone, int(shopMob.MobId), shopMob.HomeRoomId)
		if shopInv == nil {
			shopMob.Character.Shop.Restock()
			result := tryPurchaseLegacy(buyer, itemRequest, shopMob, nil)
			if result.Success {
				postSuccessBookkeeping(buyer, shopMob, nil)
				result.Requested = 1
				return result
			}
		}
		// Task 6 wires the ShopInventory path here.
	}

	return BuyResult{Reason: BuyReasonNoMatch, Requested: 1}
}

// postSuccessBookkeeping runs after any successful purchase: skill
// progression for the buyer, charisma stat-use on the merchant mob,
// and quest-engine notification gated by buyer.IsPlayer().
func postSuccessBookkeeping(buyer Actor, shopMob *mobs.Mob, shopUser *users.UserRecord) {
	buyer.OnSkillUse("bartering")

	if shopMob != nil {
		shopMob.Character.OnStatUse("charisma", 0)
	}

	if buyer.IsPlayer() {
		userId := buyer.GetUserId()
		if u := users.GetByUserId(userId); u != nil {
			room := buyer.GetRoom()
			bridge := questengine.NewGameBridge(u, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  userId,
				RoomId:  room.RoomId,
				Command: "buy",
			}, bridge, bridge)
		}
	}
}
```

Add imports for `shops` and `questengine`.

- [ ] **Step 4: Build**

Run: `go build ./internal/actions/...`
Expected: build passes.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/buy.go
git commit -m "$(cat <<'EOF'
feat(actions): lift legacy tryPurchase + executePurchaseItem/Buff

Migrates the legacy Character.Shop purchase path into actions,
using the Actor abstraction throughout. Drops the merc + pet
dispatch branches per spec 2.1. ShopInventory dispatch is added
in Task 6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Lift tryPurchaseFromInventory (ShopInventory backend)

**Files:**
- Modify: `internal/actions/buy.go`

Move `tryPurchaseFromInventory` from `usercommands/buy.go:667-800` into actions. Replace `*users.UserRecord` references with `Actor`. Skip the player-merchant case — ShopInventory is mob-merchant-only.

- [ ] **Step 1: Add the function to actions/buy.go**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// effectiveRestock returns the normalizer for pricing calculations.
// Materials use RestockQty; crafted goods (RestockQty==0) use half
// MaxStock.
func effectiveRestock(entry *shops.StockEntry) int {
	if entry.RestockQty > 0 {
		return entry.RestockQty
	}
	norm := entry.MaxStock / 2
	if norm < 1 {
		norm = 1
	}
	return norm
}

// tryPurchaseFromInventory attempts a single purchase against a
// ShopInventory-backed mob merchant. Buff/merc/pet purchases are
// NOT handled here — ShopInventory only carries items.
func tryPurchaseFromInventory(buyer Actor, request string, shopMob *mobs.Mob, shopInv *shops.ShopInventory) BuyResult {
	cfg := shops.PricingConfigFromBalance()

	type invEntry struct {
		entry     *shops.StockEntry
		item      items.Item
		plainName string
		price     int
	}

	var available []invEntry
	var itemNames []string
	var itemNamesFancy []string

	char := buyer.GetCharacter()

	for i := range shopInv.Stock {
		entry := &shopInv.Stock[i]
		if entry.Current <= 0 {
			continue
		}
		itm := items.New(entry.ItemId)
		if itm.ItemId == 0 {
			continue
		}
		spec := itm.GetSpec()
		restock := effectiveRestock(entry)
		basePrice := shops.CalcSellPrice(spec.Value, entry.Current, restock, cfg)

		// Bartering discount — works symmetrically for both buyer types.
		barterSkill := char.GetSkillLevel(skills.Bartering)
		if barterSkill > 0 {
			discount := float64(barterSkill) / 50.0 * 0.15 // max 15% at skill 50
			basePrice = shops.ApplyBarterSellDiscount(basePrice, discount)
		}

		available = append(available, invEntry{
			entry:     entry,
			item:      itm,
			plainName: spec.Name,
			price:     basePrice,
		})
		itemNames = append(itemNames, spec.Name)
		itemNamesFancy = append(itemNamesFancy, itm.DisplayName())
	}

	match, closeMatch := util.FindMatchIn(request, itemNames...)
	if match == "" {
		match = closeMatch
	}
	if match == "" {
		if shopMob != nil {
			extraSay := ""
			if len(itemNamesFancy) > 0 {
				randSelection := util.Rand(len(itemNamesFancy))
				extraSay = fmt.Sprintf(` Any interest in this <ansi fg="itemname">%s</ansi>?`, itemNamesFancy[randSelection])
			}
			shopMob.Command(`say Sorry, I can't offer that right now.` + extraSay)
		}
		return BuyResult{Reason: BuyReasonNoMatch}
	}

	var matched *invEntry
	for i := range available {
		if available[i].plainName == match {
			matched = &available[i]
			break
		}
	}
	if matched == nil {
		return BuyResult{Reason: BuyReasonNoMatch}
	}

	// Encumbrance gate — same pre-side-effect check as the legacy
	// path, since ShopInventory bypasses validatePurchase.
	if char.GetCarriedWeight()+matched.item.GetSpec().Weight > char.CarryCapacity() {
		if buyer.IsPlayer() {
			buyer.SendText("You can't carry any more.")
		}
		return BuyResult{Reason: BuyReasonOverburdened}
	}

	if matched.entry.Current <= 0 {
		if shopMob != nil {
			shopMob.Command(`say I don't have that for sale right now.`)
		}
		return BuyResult{Reason: BuyReasonOutOfStock}
	}

	if char.Gold < matched.price {
		if shopMob != nil {
			shopMob.Command(`say You don't have enough gold for that.`)
		} else if buyer.IsPlayer() {
			buyer.SendText(`You don't have enough gold for that.`)
		}
		return BuyResult{Reason: BuyReasonInsufficientGold}
	}

	if shopInv.RemoveStockAtRound(matched.entry.ItemId, 1, util.GetRoundCount()) == 0 {
		if shopMob != nil {
			shopMob.Command(`say I don't have that item right now.`)
		}
		return BuyResult{Reason: BuyReasonOutOfStock}
	}
	shopInv.SalesCount++

	if buyer.IsPlayer() {
		events.AddToQueue(events.EquipmentChange{
			UserId:     buyer.GetUserId(),
			GoldChange: -matched.price,
		})
	}
	char.Gold -= matched.price
	shopInv.Gold += matched.price

	if err := shops.SaveShop(shopInv.Zone, shopInv.MobId, shopInv.RoomId); err != nil {
		mudlog.Error("PURCHASE", "msg", "SaveShop failed", "error", err)
	}

	tradeInString := fmt.Sprintf(`<ansi fg="gold">%d gold</ansi>`, matched.price)
	if matched.price == 0 {
		tradeInString = "nothing"
	}

	executePurchaseItem(buyer, shopMob, nil,
		characters.ShopItem{ItemId: matched.entry.ItemId},
		matched.price, tradeInString)

	return BuyResult{Success: true, Purchased: 1, SaleType: "item"}
}
```

Add the `mudlog` import.

- [ ] **Step 2: Wire ShopInventory dispatch into Buy()**

In the mob-merchant loop in `Buy`, replace the comment `// Task 6 wires the ShopInventory path here.` with:

```go
		if shopInv != nil {
			result := tryPurchaseFromInventory(buyer, itemRequest, shopMob, shopInv)
			if result.Success {
				postSuccessBookkeeping(buyer, shopMob, nil)
				result.Requested = 1
				return result
			}
		}
```

So the full mob-merchant block reads:

```go
	for _, miid := range merchantMobs {
		if targetMobInstanceId > 0 && miid != targetMobInstanceId {
			continue
		}
		shopMob := mobs.GetInstance(miid)
		if shopMob == nil {
			continue
		}

		shopInv := shops.GetShopInventory(shopMob.Zone, int(shopMob.MobId), shopMob.HomeRoomId)
		if shopInv != nil {
			result := tryPurchaseFromInventory(buyer, itemRequest, shopMob, shopInv)
			if result.Success {
				postSuccessBookkeeping(buyer, shopMob, nil)
				result.Requested = 1
				return result
			}
		} else {
			shopMob.Character.Shop.Restock()
			result := tryPurchaseLegacy(buyer, itemRequest, shopMob, nil)
			if result.Success {
				postSuccessBookkeeping(buyer, shopMob, nil)
				result.Requested = 1
				return result
			}
		}
	}
```

- [ ] **Step 3: Build**

Run: `go build ./internal/actions/...`
Expected: build passes.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/buy.go
git commit -m "$(cat <<'EOF'
feat(actions): lift tryPurchaseFromInventory for ShopInventory backend

Adds the dynamic-pricing path to actions.Buy. Encumbrance gate
applies symmetrically here too. Bartering discount works the same
way for player and mob buyers via Character.GetSkillLevel.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Collapse usercommands/buy.go to wrapper; delete merc + pet code

**Files:**
- Modify: `internal/usercommands/buy.go`

Replace the entire body of `usercommands/buy.go` with a thin wrapper that calls `actions.Buy`. Delete the now-unused helpers (`tryPurchase`, `tryPurchaseFromInventory`, `validatePurchase`, `executePurchaseItem`, `executePurchaseBuff`, `executePurchaseMerc`, `executePurchasePet`, `effectiveRestock`, `sendMerchantMessage`, `purchaseContext`). The wrapper preserves the empty-request fallthrough to `List(...)`.

- [ ] **Step 1: Read the current file to confirm what's being replaced**

Run: `wc -l internal/usercommands/buy.go`
Expected: ~866 lines.

- [ ] **Step 2: Replace the file with the thin wrapper**

Overwrite `internal/usercommands/buy.go` with:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Buy is the player-side wrapper. All purchase logic lives in
// actions.Buy; this entry point handles the empty-request
// fall-through to List(...) and constructs the buyer Actor.
func Buy(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		return List(rest, user, room, flags)
	}

	actor := &actions.UserActor{User: user, Room: room}
	actions.Buy(actor, actions.BuyOptions{Request: rest})
	return true, nil
}
```

- [ ] **Step 3: Build the whole module**

Run: `go build ./...`
Expected: build passes. If there are compile errors referencing now-deleted helpers (e.g., a test imported `tryPurchase`), those tests will need a small adaptation — fix in place.

- [ ] **Step 4: Run the existing player buy tests**

Run: `go test ./internal/usercommands/ -run TestBuy -v`
Expected: PASS for `TestBuy`, `TestBuyBranches`, `TestBuyEmptyArgs`, `TestBuyNoMerchant`. If `TestBuyBranches` previously asserted merc/pet purchase, those subtests fail — open the test file, delete only the merc/pet subtests, and re-run.

Common follow-ups expected:
- Any test that asserted `executePurchasePet` was called → delete subtest, dead code.
- Any test that overburdened a character and expected the buy to succeed → adjust setup (drop weight, raise CarryCapacity, or assert the new overburdened-rejection behavior).

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/buy.go internal/usercommands/usercommands_test.go
git commit -m "$(cat <<'EOF'
refactor(usercommands): collapse buy to actions.Buy wrapper

Deletes ~830 lines of helpers (tryPurchase, validatePurchase,
executePurchase*, etc.) — all migrated to internal/actions/buy.go.
Drops merc and pet purchase paths per spec 2.1. Existing
TestBuy* tests stay green; merc/pet assertions in TestBuyBranches
removed alongside their now-deleted code paths.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Quantity parsing

**Files:**
- Modify: `internal/actions/buy.go`
- Modify: `internal/actions/buy_test.go`

Restore the `buy N <item>` behavior. The current `Buy()` treats the whole request as the item name; this task adds the leading-integer parse and the per-unit retry loop with partial-success messaging.

- [ ] **Step 1: Add quantity parsing at the top of Buy()**

After the `req` trim/empty check and before the `from` clause parsing, insert:

```go
	// Parse leading quantity: "buy 5 iron ingot".
	quantity := 1
	args0 := strings.SplitN(strings.TrimSpace(req), " ", 2)
	if len(args0) == 2 {
		if n, err := strconv.Atoi(args0[0]); err == nil && n >= 1 {
			quantity = n
			req = args0[1]
		}
	}
```

(`strconv` is already imported as a placeholder; remove the `_ = strconv.Atoi` line if still present.)

- [ ] **Step 2: Wrap the merchant loop in a quantity-aware retry**

Restructure the rest of `Buy()` so that after the first successful purchase, the function loops up to `quantity` times against the same merchant. Replace the per-merchant `if result.Success { return result }` pattern with:

```go
	tryMerchant := func(attempt func() BuyResult) (BuyResult, bool) {
		first := attempt()
		if !first.Success {
			return first, false
		}
		purchased := 1
		for purchased < quantity {
			next := attempt()
			if !next.Success {
				break
			}
			purchased++
		}
		if quantity > 1 && purchased < quantity {
			if buyer.IsPlayer() {
				buyer.SendText(fmt.Sprintf(`<ansi fg="yellow">Purchased %d of %d before running short.</ansi>`, purchased, quantity))
			}
		}
		return BuyResult{
			Success:   true,
			Purchased: purchased,
			Requested: quantity,
			SaleType:  first.SaleType,
		}, true
	}

	for _, uid := range merchantPlayers {
		if targetUserId > 0 && uid != targetUserId {
			continue
		}
		shopUser := users.GetByUserId(uid)
		if shopUser == nil {
			continue
		}
		result, sold := tryMerchant(func() BuyResult {
			r := tryPurchaseLegacy(buyer, itemRequest, nil, shopUser)
			if r.Success {
				postSuccessBookkeeping(buyer, nil, shopUser)
			}
			return r
		})
		if sold {
			return result
		}
	}

	for _, miid := range merchantMobs {
		if targetMobInstanceId > 0 && miid != targetMobInstanceId {
			continue
		}
		shopMob := mobs.GetInstance(miid)
		if shopMob == nil {
			continue
		}

		shopInv := shops.GetShopInventory(shopMob.Zone, int(shopMob.MobId), shopMob.HomeRoomId)
		if shopInv != nil {
			result, sold := tryMerchant(func() BuyResult {
				r := tryPurchaseFromInventory(buyer, itemRequest, shopMob, shopInv)
				if r.Success {
					postSuccessBookkeeping(buyer, shopMob, nil)
				}
				return r
			})
			if sold {
				return result
			}
		} else {
			result, sold := tryMerchant(func() BuyResult {
				shopMob.Character.Shop.Restock()
				r := tryPurchaseLegacy(buyer, itemRequest, shopMob, nil)
				if r.Success {
					postSuccessBookkeeping(buyer, shopMob, nil)
				}
				return r
			})
			if sold {
				return result
			}
		}
	}

	return BuyResult{Reason: BuyReasonNoMatch, Requested: quantity}
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: build passes.

- [ ] **Step 4: Add a quantity-parsing test**

Append to `internal/actions/buy_test.go`:

```go
func TestBuy_QuantityParse(t *testing.T) {
	// We can't easily exercise the full purchase path without a shop
	// fixture; the parsing behavior is observable via Requested even
	// when the purchase fails for unrelated reasons.
	room := newEmptyTestRoom(t)
	mobActor := &MobActor{Room: room}

	result := Buy(mobActor, BuyOptions{Request: "5 iron ingot"})
	if result.Requested != 5 {
		t.Errorf("Requested = %d, want 5", result.Requested)
	}
	if result.Success {
		t.Errorf("expected Success=false with no merchant in room")
	}
}
```

- [ ] **Step 5: Run tests, commit**

Run: `go test ./internal/actions/ -run TestBuy -v`
Expected: all pass.

```bash
git add internal/actions/buy.go internal/actions/buy_test.go
git commit -m "$(cat <<'EOF'
feat(actions): support buy N <item> quantity loop

Restores the per-unit retry behavior from the player wrapper and
ships it symmetrically for mob buyers. Partial-success yellow
message gated to player buyers.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: New mobcommands/buy.go wrapper + registration

**Files:**
- Create: `internal/mobcommands/buy.go`
- Modify: `internal/mobcommands/mobcommands.go`

Wire the mob-side entry point. Wrapper is ~15 lines; registration is a single line addition in the `mobCommands` map.

- [ ] **Step 1: Create `internal/mobcommands/buy.go`**

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Buy is the mob-side entry into actions.Buy. The mob has no
// list-fall-through (mobs don't read text), so empty rest is a
// silent no-op.
func Buy(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if rest == "" {
		return true, nil
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Buy(actor, actions.BuyOptions{Request: rest})
	return true, nil
}
```

- [ ] **Step 2: Register in mobcommands.go**

In `internal/mobcommands/mobcommands.go`, inside the `mobCommands` map literal, add the new entry. Insert it after `"break"` to keep alphabetical-ish order:

```go
		"bash":           {Bash, false},
		"bite":           {Bite, false},
		"befriend":       {Befriend, false},
		"break":          {Break, false},
		"buy":            {Buy, false}, // NEW: chunk 2.1
		"broadcast":      {Broadcast, false},
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: build passes.

- [ ] **Step 4: Confirm the dispatcher sees the new command**

Run: `go test ./internal/mobcommands/ -run TestCommandIsReadyNamesAreMobCommands -v`
Expected: PASS (the test doesn't include `buy`, so this just confirms the existing list isn't broken).

- [ ] **Step 5: Commit**

```bash
git add internal/mobcommands/buy.go internal/mobcommands/mobcommands.go
git commit -m "$(cat <<'EOF'
feat(mobcommands): add buy command wrapper

Thin wrapper around actions.Buy. Empty rest is a silent no-op
(mobs don't read 'list'). AllowedWhenDowned: false matches the
other shop-adjacent verbs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: actions/buy_test.go — full coverage table

**Files:**
- Modify: `internal/actions/buy_test.go`

Flesh out the test table from the spec. The earlier task-scoped tests cover empty-request, no-merchant, overburdened, and quantity parse. This task adds the rest: insufficient gold, no-match, and the full happy paths through both backends.

Several cases need shared fixtures (a shop-bearing test mob, a buyer with gold, an item template with non-zero weight). The pattern in `internal/actions/economy_test.go` is a good reference for setting up a UserRecord; for MobActor, the existing `combat_test.go` shows how to build a mob from a spec.

The tests below assume a small set of test fixtures that may need to be added to the test data dir alongside this task. Where a fixture isn't trivially available, mark the test `t.Skip` with a TODO referring to the fixture and capture the gap as a follow-on memory entry.

- [ ] **Step 1: Add the happy-path tests for legacy backend**

Append to `internal/actions/buy_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/users"
)

// newShopMob builds a mob with a single item in its legacy
// Character.Shop. itemId must resolve to a valid item in the test
// data dir.
func newShopMob(t *testing.T, itemId int, price int, stock int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{}
	m.Character.Name = "Test Merchant"
	m.Character.Shop = characters.Shop{
		{ItemId: itemId, Price: price, Quantity: stock, QuantityMax: stock},
	}
	return m
}

func TestBuy_LegacyItem_HappyPath_PlayerBuyer(t *testing.T) {
	const itemId = 20000 // adjust to any cheap test item available
	itm := items.New(itemId)
	if itm.ItemId == 0 {
		t.Skip("test fixture itemId 20000 missing")
	}

	room := &rooms.Room{RoomId: 99999}
	shopMob := newShopMob(t, itemId, 10, 5)
	room.AddMob(shopMob.InstanceId)
	defer mobs.DestroyInstance(shopMob.InstanceId)
	// Real implementation: register the mob with the room machinery
	// so room.GetMobs(FindMerchant) returns it. The exact wiring
	// depends on the test harness; reuse helpers from combat_test.go.

	user := &users.UserRecord{}
	user.UserId = 1
	user.Character = &characters.Character{}
	user.Character.Gold = 100
	user.Character.Strength = 100 // healthy carry capacity
	buyer := &UserActor{User: user, Room: room}

	result := Buy(buyer, BuyOptions{Request: itm.GetSpec().Name})

	if !result.Success {
		t.Fatalf("expected Success=true, got Reason=%q", result.Reason)
	}
	if result.SaleType != "item" {
		t.Errorf("SaleType = %q, want item", result.SaleType)
	}
	if user.Character.Gold != 90 {
		t.Errorf("user gold = %d, want 90", user.Character.Gold)
	}
	if shopMob.Character.Gold != 1 {
		t.Errorf("legacy +1 merchant gold cheat: got %d want 1", shopMob.Character.Gold)
	}
}
```

(Adapt the harness boilerplate to match what's actually in
`combat_test.go` — the snippet above is the *shape* of the test;
the *setup* must match local conventions to avoid panics during
`room.GetMobs(FindMerchant)`.)

- [ ] **Step 2: Add the mob-buyer happy-path test**

```go
func TestBuy_LegacyItem_HappyPath_MobBuyer(t *testing.T) {
	const itemId = 20000
	itm := items.New(itemId)
	if itm.ItemId == 0 {
		t.Skip("test fixture itemId 20000 missing")
	}

	room := &rooms.Room{RoomId: 99999}
	shopMob := newShopMob(t, itemId, 10, 5)
	// Register shopMob in room.

	buyerMob := &mobs.Mob{}
	buyerMob.Character.Name = "Bandit"
	buyerMob.Character.Gold = 100
	buyerMob.Character.Strength = 100
	buyer := &MobActor{Mob: buyerMob, Room: room}

	result := Buy(buyer, BuyOptions{Request: itm.GetSpec().Name})

	if !result.Success {
		t.Fatalf("expected Success=true, got Reason=%q", result.Reason)
	}
	if buyerMob.Character.Gold != 90 {
		t.Errorf("mob buyer gold = %d, want 90", buyerMob.Character.Gold)
	}
	// Mob buyer should NOT fire the events.EquipmentChange (no prompt
	// to update); harness should not see the event. Asserting this
	// requires a hooked event queue — defer to integration smoke if
	// the unit test harness doesn't capture events.
}
```

- [ ] **Step 3: Add insufficient-gold and no-match tests**

```go
func TestBuy_InsufficientGold(t *testing.T) {
	const itemId = 20000
	itm := items.New(itemId)
	if itm.ItemId == 0 {
		t.Skip("test fixture itemId 20000 missing")
	}

	room := &rooms.Room{RoomId: 99999}
	shopMob := newShopMob(t, itemId, 1000, 5)
	// Register.

	user := &users.UserRecord{}
	user.UserId = 1
	user.Character = &characters.Character{}
	user.Character.Gold = 5 // not enough
	user.Character.Strength = 100
	buyer := &UserActor{User: user, Room: room}

	result := Buy(buyer, BuyOptions{Request: itm.GetSpec().Name})

	if result.Success {
		t.Fatalf("expected failure")
	}
	if result.Reason != BuyReasonInsufficientGold {
		t.Errorf("Reason = %q, want %q", result.Reason, BuyReasonInsufficientGold)
	}
	if user.Character.Gold != 5 {
		t.Errorf("gold should not be deducted; got %d", user.Character.Gold)
	}
	if shopMob.Character.Shop[0].Quantity != 5 {
		t.Errorf("stock should not be destocked; got %d", shopMob.Character.Shop[0].Quantity)
	}
}

func TestBuy_NoMatch(t *testing.T) {
	room := &rooms.Room{RoomId: 99999}
	shopMob := newShopMob(t, 20000, 10, 5)
	// Register.

	user := &users.UserRecord{}
	user.UserId = 1
	user.Character = &characters.Character{}
	user.Character.Gold = 100
	user.Character.Strength = 100
	buyer := &UserActor{User: user, Room: room}

	result := Buy(buyer, BuyOptions{Request: "zorpfish"})

	if result.Success {
		t.Fatalf("expected failure")
	}
	if result.Reason != BuyReasonNoMatch {
		t.Errorf("Reason = %q, want %q", result.Reason, BuyReasonNoMatch)
	}
}
```

- [ ] **Step 4: Run all actions tests**

Run: `go test ./internal/actions/ -run TestBuy -v`
Expected: all PASS or SKIP. Skipped tests indicate missing fixtures.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/buy_test.go
git commit -m "$(cat <<'EOF'
test(actions): full coverage table for actions.Buy

Adds happy-path, insufficient-gold, no-match, mob-buyer, and
encumbrance-gate tests. Tests that need fixtures missing from the
test data dir use t.Skip with a TODO marker; gaps captured for a
follow-on fixture-authoring memory entry if any remain after a
local run.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: mobcommands/buy_test.go — TryCommand smoke

**Files:**
- Create: `internal/mobcommands/buy_test.go`

One TryCommand-level test confirming the mob wrapper wires correctly. Thorough cases live in actions/buy_test.go; this test only validates end-to-end through the mob command pipeline.

- [ ] **Step 1: Create the file**

```go
// internal/mobcommands/buy_test.go
package mobcommands

import "testing"

func TestBuyRegistered(t *testing.T) {
	all := GetAllMobCommands()
	found := false
	for _, cmd := range all {
		if cmd == "buy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'buy' to be registered in mobCommands")
	}
}

func TestBuyEmptyRest_NoOp(t *testing.T) {
	// Mob buy with empty rest is a silent no-op. Direct call (not
	// through TryCommand) since TryCommand requires a live mob
	// instance which the harness doesn't set up trivially.
	handled, err := Buy("", nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handled {
		t.Errorf("expected handled=true for empty rest no-op")
	}
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/mobcommands/ -run TestBuy -v`
Expected: both PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/mobcommands/buy_test.go
git commit -m "$(cat <<'EOF'
test(mobcommands): smoke registration + empty-rest no-op for buy

Confirms the wrapper is dispatch-visible and the empty-rest path
returns handled=true silently. Thorough purchase semantics covered
by actions/buy_test.go.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Manual smoke test in running server

**Files:**
- None (manual verification)

Per the CLAUDE.md pre-push SOP: boot the server locally and confirm it
starts cleanly past data-file loading, then exercise the new verb.

- [ ] **Step 1: Build the server**

Run: `go build -o dogmud.exe .`
Expected: clean build.

- [ ] **Step 2: Start the server and watch loader output**

Run: `./dogmud.exe` (or `go run .`)
Expected: see `mobs.LoadDataFiles() loadedCount=...`, `items.LoadDataFiles() loadedCount=...` etc. without panics.

- [ ] **Step 3: Connect and run the player-side regression**

Connect as a test player, walk to a known shop (e.g., Thornwall blacksmith). Run:
```
buy iron ingot
buy 3 iron ingot
buy iron ingot from <merchant-name>
```
Expected: items appear in inventory; gold decreases; partial-success message on quantity > stock.

- [ ] **Step 4: Run the mob-side regression**

From the admin console:
```
mob <instanceId> buy iron ingot
```
where `<instanceId>` is a test mob spawned in the same shop room with gold (use admin tools to grant gold).
Expected: mob's `Character.Gold` decreases (verify via `mob inspect`); ingot appears in mob's `Character.Items`; room broadcast text shows nearby players "Bandit purchases the iron ingot from Marko."

- [ ] **Step 5: Run the encumbrance regression**

Load up a test player to overburdened, attempt to buy a heavy item. Expected: "You can't carry any more." no gold deducted, no stock removed.

- [ ] **Step 6: Kill the server, no commit**

```bash
# Ctrl-C or kill the process
```

Per the kill-test-servers SOP, clean up any `dogmud*.exe` / `go run` processes.

---

## Task 13: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

Mark 2.1 Done, update the roll-up.

- [ ] **Step 1: Update the progress tracker row**

In the Progress tracker table, change the `2.1` row's Status column from `Not started` to `Done`.

- [ ] **Step 2: Update the chunk mini-brief**

In the "### 2.1 Mob `buy` command" section, change `**Status:** Not started • **Size:** M` to `**Status:** Done (2026-05-11) • **Size:** M`. Append a `**Shipped:**` paragraph similar to the 1.x entries, summarizing:
- `internal/actions/buy.go` with `actions.Buy(buyer Actor, opts BuyOptions) BuyResult`
- legacy + ShopInventory backends supported symmetrically
- item + buff sale types; merc + pet paths deleted
- new symmetric carry-capacity gate in `validatePurchase`
- thin wrappers at `internal/usercommands/buy.go` and `internal/mobcommands/buy.go`
- registered in `mobCommands` map; smoke test green

- [ ] **Step 3: Update the roll-up**

Change `**Roll-up:** 7 / 40 done • 0 in progress • 33 not started.` to `**Roll-up:** 8 / 40 done • 0 in progress • 32 not started.`

- [ ] **Step 4: Add MEMORY.md follow-on for deferred paid-merc work**

If the work surfaced a need to design proper paid-merc hiring (it did — that's the explicit deferral), append to `MEMORY.md`'s "Loose Followups" or "Features & Content" section a new row pointing to a new
`memory/project_paid_merc_hiring.md`. Author that memory file with:

```markdown
---
name: Paid Merc Hiring
description: Replace dropped buy-merc path with a proper paid contractor model
type: project
---

Chunk 2.1 (mob `buy` command) dropped the legacy merc purchase path
from `actions.Buy`. No current shop YAML sells mercs, so the deletion
was content-zero-impact. A future chunk should design proper
paid-merc semantics:

- Hire fee + periodic upkeep (gold drain per N rounds).
- Loyalty meter; walks away if unpaid.
- May be a new `hire` verb rather than reusing `buy`.

**Why:** The legacy "buy a merc → permanent charm" model is a
pre-aliveness-era artifact; making mercs paid contractors fits the
aliveness framework (factions, opinions, knowledge) better.

**How to apply:** When designing the merc-hiring chunk, do NOT
revive `executePurchaseMerc` as-is. Start from the paid-contractor
model.
```

- [ ] **Step 5: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md MEMORY.md memory/project_paid_merc_hiring.md
git commit -m "$(cat <<'EOF'
docs(roadmap): mark chunk 2.1 (mob buy command) as Done

Ships consolidated actions.Buy covering items + buffs, with the
merc and pet sale paths dropped. Symmetric carry-capacity gate
closes a pre-existing player-side gap. Roll-up moves to 8/40.
Deferred paid-merc hiring captured as a memory entry.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (run before declaring done)

- [ ] `go build ./...` passes clean.
- [ ] `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ -v` all green (skips OK only for missing test fixtures).
- [ ] `internal/usercommands/buy.go` ends up around 30 lines.
- [ ] `internal/mobcommands/buy.go` ends up around 20 lines.
- [ ] `internal/actions/buy.go` has no `*users.UserRecord` parameter on `Buy` itself (Actor only); concrete `*users.UserRecord` and `*mobs.Mob` are limited to helper signatures that need shop-side state.
- [ ] No `executePurchaseMerc` or `executePurchasePet` remain anywhere.
- [ ] No `tryPurchase` / `tryPurchaseFromInventory` / `validatePurchase` symbols remain in `internal/usercommands/`.
- [ ] Manual smoke test (Task 12) confirmed: player buy, quantity buy, mob buy, encumbrance reject all worked in a live server.
- [ ] MEMORY.md `project_paid_merc_hiring` entry captured.
- [ ] Roadmap roll-up updated.
