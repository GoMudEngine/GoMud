# Mob Aliveness 5.4 — NPC Market Participation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let NPCs sell loot through shops (lifting the missing `sell` verb), drain unsold non-material shop overstock over time, and route forager-chest surplus back into vendor stock with a chest-full rest back-pressure.

**Architecture:** Three independent components on the existing shop/forager substrate. (1) `actions.Sell` actor-pattern lift — mirrors the 2.1 `actions.Buy` lift; mob sales credit the seller without draining shop gold. (2) Time-based overstock decay on `StockEntry`, excluding crafting materials. (3) A forager-chest → vendor backfill called from the idle/restock tick, plus a chest-fullness gate on the forager cycle restart.

**Tech Stack:** Go. Tests with the standard `testing` package + `testify` (`github.com/stretchr/testify/assert`), matching existing `internal/actions`, `internal/shops`, `internal/forager` tests. Build: `go build ./...`. Test: `go test ./internal/<pkg>/...`.

**Spec:** `docs/superpowers/specs/2026-06-02-mob-aliveness-5.4-npc-market-participation-design.md`

---

## Verified codebase facts (confirmed before writing — trust these over memory)

- **Actor interface** `internal/actions/actor.go`: `GetCharacter() *characters.Character`, `GetRoom() *rooms.Room`, `SendText(cat messaging.Category, msg string)`, `GetName() string`, `IsPlayer() bool`, `GetUserId() int`, `OnSkillUse(skillName string) bool`, `OnStatUse(statName string) bool`. `MobActor` (`actor_mob.go`) and `UserActor` (`actor_user.go`) implement it; `MobActor.SendText` is a no-op.
- **Buy lift precedent:** `internal/usercommands/buy.go` (player wrapper), `internal/mobcommands/buy.go` (mob wrapper) — both build an Actor and call `actions.Buy(actor, opts)`.
- **Player sell source to lift:** `internal/usercommands/sell.go`. Key helpers: `merchantSay(room, mob, line)`, `sellFindItem(user, name) (items.Item, bool)`, `trySellOne(itemName, user, room, mob, shopInv) (int, sellResult)`, `Sell(rest, user, room, flags)`. The shop-gold deduction is `shopInv.Gold -= sellValue` (line ~133); merchant-broke check is `if merchantGold < sellValue` (line ~126).
- **mobcommands registry:** `internal/mobcommands/mobcommands.go` — map literal entries like `"buy": {Buy, false}`; lookup at `mobCommands[cmd]`.
- **Shop types** `internal/shops/shopinventory.go`: `StockEntry{ ItemId, RestockQty, MaxStock, Current int }` (yaml `item_id`/`restock_qty`/`max_stock`/`current`). **No last-grew timestamp exists.** `ShopInventory` has non-persisted `Zone string`, `MobId int`, `RoomId int` fields set at load.
- **Shop helpers:** `shops.GetShopInventory(zone string, mobId, roomId int) *ShopInventory` (`persistence.go:40`), `AllShops() []*ShopInventory` (`persistence.go:193`), `(*ShopInventory).GetStock(itemId int) *StockEntry` (`shopinventory.go:128`), `(*ShopInventory).AddStockAtRound(itemId, qty int, round uint64)` (`shopinventory.go:149`), `SaveShop(zone string, mobId, roomId int) error` (`persistence.go:129`).
- **Restock tick:** `mobs.TickMobShopRestock(mob *Mob) bool` (`internal/mobs/crafter.go:146`, non-crafter shops) and `mobs.TickMobCraft(mob) *CraftResult` (crafter shops) — BOTH fired from `internal/hooks/MobIdle_HandleIdleMobs.go` (lines ~50 and ~64). The hook layer can import `forager`, `shops`, and `mobs`; the `mobs` package CANNOT import `forager` (cycle).
- **Item component flag:** `items.GetItemSpec(itemId) *ItemSpec`; field `IsComponent bool` (`internal/items/itemspec.go:281`). `ItemSpec.Value int`, `ItemSpec.QuestToken string`.
- **Forager registry** `internal/forager/territory.go`: `AllProfiles() []*ForagerProfile`, `ProfileFor(mobId int) *ForagerProfile`, `IsForagerMob(mobTemplateId int) bool`. Chest room is `mob.StorageChestRoom` (on the **instance**, not the profile); `mob.Zone` is the instance zone.
- **Forager states** `internal/forager/state.go`: `StateResting` (zero value, at sanctuary) → `StateTravelingToTerritory` → `StateForaging` → … → `StateRecalling` → wraps. Dispatch switch `internal/behaviortree/actions_forager.go:110`. `tickForagerResting` at `:211`; its cycle-restart transition `transitionForager(ctx.MobState, forager.StateTravelingToTerritory)` is at `:237`.
- **Chest container** `internal/rooms/container.go`: `Container{ Lock gamelock.Lock; Items []items.Item; ... }`; methods `AddItem(i)`, `RemoveItem(i)`, `FindItemById(itemId) (items.Item, bool)`, `Count(itemId) int`, `HasLock()`. Found via `room.FindContainerByName("lockbox") string` then `room.Containers[key]`. Capacity = `room.StorageCapacity int` (`rooms.go:82`; **0 means default 20**).
- **Mob instance enumeration:** `mobs.GetAllMobInstanceIds() []int` (`mobs.go:662`), `mobs.GetInstance(instanceId int) *Mob` (`mobs.go:653`).
- **`forager.SellToVendor(roomId int, p *ForagerProfile, mob *mobs.Mob)`** (`internal/forager/vendor_sell.go`) — the existing free-handoff supply path; backfill mirrors its persistence/throughput pattern.
- **Round count:** `util.GetRoundCount() uint64`.
- **Config:** balance knobs live in `internal/configs/config.balance.go` as `ConfigInt`/`ConfigFloat` fields; defaults applied in a `config.balance.*.go` validation function (e.g. `config.balance.shops.go`). Access via `configs.GetBalanceConfig()`.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/actions/sell.go` (new) | Actor-pattern sell logic: `Sell(seller, opts)`, `SellOptions`, `SellResult`, per-item + sweep helpers, gold-model gate |
| `internal/actions/sell_test.go` (new) | Unit tests for the lift |
| `internal/usercommands/sell.go` (modify) | Thin to a parse+delegate wrapper |
| `internal/mobcommands/sell.go` (new) | Mob wrapper |
| `internal/mobcommands/mobcommands.go` (modify) | Register `"sell"` |
| `internal/shops/shopinventory.go` (modify) | Add `LastGrewRound` to `StockEntry`; stamp it on growth |
| `internal/shops/overstock_decay.go` (new) | `TickOverstockDecay(si, round)` |
| `internal/shops/overstock_decay_test.go` (new) | Decay tests |
| `internal/forager/chest_index.go` (new) | In-memory `zone → chest-room-set` index; `RegisterChestRoom` / `ChestRoomsForZone` |
| `internal/forager/chest_index_test.go` (new) | Index register/lookup tests |
| `internal/forager/chest_backfill.go` (new) | `BackfillVendorFromChests(vendorMob, shopInv)` + chest helpers |
| `internal/forager/chest_backfill_test.go` (new) | Backfill tests |
| `internal/behaviortree/actions_forager.go` (modify) | Chest-fullness gate in `tickForagerResting` |
| `internal/hooks/MobIdle_HandleIdleMobs.go` (modify) | Call decay + backfill per shopkeeper idle tick |
| `internal/configs/config.balance.go` + validation file + `_datafiles/config.yaml` (modify) | `ShopOverstockDecayRounds`, `ShopOverstockDecayQty`, `ChestBackpressureResumePct` |
| `internal/actions/context.md`, `internal/shops/context.md`, `internal/forager/context.md` (modify) | Document the two sell models, decay, backfill |
| `MOB_ALIVENESS_ROADMAP.md` (modify) | Flip 5.4 status on completion |

---

## Task 1: `actions.Sell` lift — types, wrappers, registration (compile-first skeleton)

Establish the new package surface and wiring before moving logic, so the rest of Task 1/2 compiles incrementally.

**Files:**
- Create: `internal/actions/sell.go`
- Create: `internal/mobcommands/sell.go`
- Modify: `internal/mobcommands/mobcommands.go`

- [ ] **Step 1: Create the type surface and a stub `Sell`**

`internal/actions/sell.go`:

```go
package actions

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// UnlimitedSell is the Quantity sentinel meaning "sell every match".
const UnlimitedSell = math.MaxInt

type SellOptions struct {
	ItemName        string // ignored when SellAllSellable
	Quantity        int    // 1, N, or UnlimitedSell
	SellAllSellable bool   // mob inventory-sweep mode (every sellable item)
	MerchantName    string // optional target merchant name; "" = first willing
}

type SellStopReason int

const (
	SellStopSoldAll       SellStopReason = iota // ran out of matching items (normal)
	SellStopNoItem                              // seller never had the item
	SellStopNoMerchant                          // no willing merchant in room
	SellStopMerchantBroke                       // merchant ran out of gold (player path only)
	SellStopRejected                            // merchant declined the item type
)

type SellResult struct {
	Sold         int
	TotalGold    int
	Reason       SellStopReason
	LastItemName string
}

// Sell is the shared seller entry point for players and mobs. The seller is
// abstracted via Actor; the merchant is a shopkeeper mob resolved from the
// seller's room. Player sells draw down shop gold; mob sells credit the seller
// but leave shop gold intact (see internal/shops/context.md "two sell models").
//
// NOTE: forager.SellToVendor is a different path — a free supply handoff, not a
// sale. See internal/forager/vendor_sell.go.
func Sell(seller Actor, opts SellOptions) SellResult {
	// Implemented in Step (Task 1, later steps).
	return SellResult{Reason: SellStopNoItem}
}

// silence unused import until logic lands
var _ = items.Item{}
```

- [ ] **Step 2: Add the mob wrapper**

`internal/mobcommands/sell.go`:

```go
package mobcommands

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Sell is the mob-side entry into actions.Sell. Mobs don't read text, so an
// empty request is a silent no-op. "sell all" (no item name) maps to the
// inventory-sweep mode; the planners (wealth-gold, upgrade-gear) rely on this.
func Sell(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if rest == "" {
		return true, nil
	}
	actor := &actions.MobActor{Mob: mob, Room: room}

	lower := strings.ToLower(strings.TrimSpace(rest))
	if lower == "all" {
		actions.Sell(actor, actions.SellOptions{SellAllSellable: true})
		return true, nil
	}

	// "sell N <item>" / "sell all <item>" / "sell <item>"
	opts := actions.SellOptions{Quantity: 1, ItemName: rest}
	parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
	if len(parts) == 2 {
		if strings.ToLower(parts[0]) == "all" {
			opts.Quantity = actions.UnlimitedSell
			opts.ItemName = parts[1]
		} else if n, err := strconv.Atoi(parts[0]); err == nil && n >= 1 {
			opts.Quantity = n
			opts.ItemName = parts[1]
		}
	}
	actions.Sell(actor, opts)
	return true, nil
}
```

- [ ] **Step 3: Register the mob command**

In `internal/mobcommands/mobcommands.go`, add to the `mobCommands` map literal, next to `"buy"`:

```go
		"sell":           {Sell, false},
```

- [ ] **Step 4: Verify it builds**

Run: `go build ./internal/actions/... ./internal/mobcommands/...`
Expected: builds clean (Sell is a stub).

- [ ] **Step 5: Commit**

```bash
git add internal/actions/sell.go internal/mobcommands/sell.go internal/mobcommands/mobcommands.go
git commit -m "feat(5.4): actions.Sell type surface + mob wrapper + registration"
```

---

## Task 2: `actions.Sell` lift — move the sale logic with the gold-model gate

Move the seller-side logic out of `usercommands/sell.go` into `actions.Sell`, abstracting the seller behind `Actor` and gating shop-gold drain on `seller.IsPlayer()`.

**Files:**
- Modify: `internal/actions/sell.go`
- Modify: `internal/usercommands/sell.go` (thin to wrapper)
- Test: `internal/actions/sell_test.go`

- [ ] **Step 1: Write failing tests for the gold model + sweep**

`internal/actions/sell_test.go` — use a fake Actor like `internal/actions/consider_test.go`'s `fakeActor` (copy its shape: implements all Actor methods, holds a `*characters.Character` and `*rooms.Room`, `IsPlayer()` returns a struct field). Add tests:

```go
func TestSell_MobSale_DoesNotDrainShopGold(t *testing.T) {
	// Arrange: a mob seller with one sellable item in a room containing a
	// shopkeeper mob whose ShopInventory has Gold=0 and a stock entry for the
	// item below MaxStock.
	// Act: actions.Sell(mobSeller, SellOptions{SellAllSellable: true})
	// Assert: result.Sold == 1; seller gold increased; shopInv.Gold unchanged
	//         (still 0); result.Reason != SellStopMerchantBroke.
}

func TestSell_PlayerSale_DrainsShopGold(t *testing.T) {
	// Same setup, player seller, shop Gold large.
	// Assert: shopInv.Gold decreased by the sale value.
}

func TestSell_SellAllSellable_SkipsQuestAndComponentAndZeroValue(t *testing.T) {
	// Mob seller carrying: a quest item (QuestToken set), an is_component
	// material, a Value==0 item, and one normal sellable item.
	// Assert: only the normal item sells (Sold == 1).
}

func TestSell_NoMerchant(t *testing.T) {
	// Room has no merchant. Assert Reason == SellStopNoMerchant, Sold == 0.
}
```

> **Live-data caveat (match 2.1/5.3 precedent):** item specs / shop pricing need loaded data files that `go test` does not provide. For branches that require real pricing (e.g. exact gold amounts), assert on direction (increased/unchanged) using a hand-built `ShopInventory` + `StockEntry`, not absolute prices. Deep price paths are covered by the in-game smoke. Keep these unit tests to the gold-gate direction, the skip-filters, and the no-merchant branch.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/actions/ -run TestSell -v`
Expected: FAIL (stub returns `SellStopNoItem`).

- [ ] **Step 3: Move `merchantSay` and the per-item sale into `actions`**

In `internal/actions/sell.go`, add `merchantSay` (verbatim from `usercommands/sell.go`, it already takes `room, mob, line` and uses `actions.Say` — drop the `actions.` qualifier since it's now in-package) and a `sellOneToMerchant` adapted from `trySellOne`:

```go
// sellOneToMerchant sells a single matching item from seller to the given
// merchant. Mirrors the player trySellOne, with two changes:
//   - seller-side access via Actor (GetCharacter / Gold / RemoveItem).
//   - shop-gold drain + merchant-broke check apply only to player sellers
//     (decision 2: NPC selling never bankrupts a shop).
func sellOneToMerchant(seller Actor, itemName string, room *rooms.Room,
	mob *mobs.Mob, shopInv *shops.ShopInventory) (soldValue int, res SellStopReason) {

	char := seller.GetCharacter()
	item, found := sellFindItemInChar(char, itemName)
	if !found {
		return 0, SellStopNoItem
	}
	itemSpec := item.GetSpec()
	if itemSpec.ItemId < 1 {
		return 0, SellStopRejected
	}
	if itemSpec.QuestToken != "" {
		if seller.IsPlayer() {
			seller.SendText(messaging.CategorySystem, "Quest items cannot be sold!")
		}
		return 0, SellStopRejected
	}

	char.CancelBuffsWithFlag(buffs.Hidden)
	if item.IsSpecial() {
		merchantSay(room, mob, "I'm afraid I don't buy those.")
		return 0, SellStopRejected
	}

	var sellValue int
	var buyReason string
	if shopInv != nil {
		cfg := shops.PricingConfigFromBalance()
		wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
		offer := shops.EvaluateBuyRules(item, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
		sellValue = offer.Price
		buyReason = offer.Reason
		if sellValue > 0 {
			barterSkill := char.GetSkillLevel(skills.Bartering)
			if barterSkill > 0 {
				bonus := float64(barterSkill) / 50.0 * 0.15
				if bonus > 0.15 {
					bonus = 0.15
				}
				sellValue = shops.ApplyBarterBuyBonus(sellValue, bonus)
			}
		}
	} else {
		sellValue = mob.GetSellPrice(item)
	}

	if sellValue <= 0 {
		merchantSay(room, mob, "I'm not interested in that.")
		return 0, SellStopRejected
	}

	// Gold-model gate: only players are constrained by — and draw down —
	// the merchant's gold. Mob sales mint the seller's payout.
	if seller.IsPlayer() {
		merchantGold := mob.Character.Gold
		if shopInv != nil {
			merchantGold = shopInv.Gold
		}
		if merchantGold < sellValue {
			merchantSay(room, mob, "I can't afford that right now.")
			return 0, SellStopMerchantBroke
		}
		if shopInv != nil {
			shopInv.Gold -= sellValue
		} else {
			mob.Character.Gold -= sellValue
		}
	}

	char.Gold += sellValue
	char.RemoveItem(item)

	if seller.IsPlayer() {
		events.AddToQueue(events.ItemOwnership{UserId: seller.GetUserId(), Item: item, Gained: false})
		events.AddToQueue(events.EquipmentChange{UserId: seller.GetUserId(), GoldChange: sellValue})
	}

	// Stock update (merchant side, unchanged from trySellOne).
	if shopInv != nil {
		shopInv.BuysCount++
		if buyReason == "gear_upgrade" {
			newItem := items.New(item.ItemId)
			if newItem.ItemId > 0 {
				returnedItems, wore, _ := mob.Character.Wear(newItem)
				if wore {
					for _, old := range returnedItems {
						if old.ItemId > 0 {
							shopInv.AddStockAtRound(old.ItemId, 1, util.GetRoundCount())
						}
					}
					room.SendTextVisual(messaging.CategoryLoot,
						fmt.Sprintf(`<ansi fg="mobname">%s</ansi> examines the <ansi fg="itemname">%s</ansi> and puts it on.`, mob.Character.Name, newItem.DisplayName()),
						seller.GetUserId(),
					)
				} else {
					shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
				}
			} else {
				shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
			}
		} else {
			shopInv.AddStockAtRound(item.ItemId, 1, util.GetRoundCount())
		}
		if err := shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
			mudlog.Error("SELL", "msg", "SaveShop failed", "error", err)
		}
	} else {
		mob.Character.Shop.StockItem(item.ItemId)
	}

	// Progression.
	seller.OnSkillUse(string(skills.Bartering))
	mob.Character.OnStatUse("charisma", 0)

	return sellValue, SellStopSoldAll
}

// sellFindItemInChar searches backpack → potions → components for a match.
func sellFindItemInChar(char *characters.Character, name string) (items.Item, bool) {
	item, found := char.FindInBackpack(name)
	if !found {
		item, found = char.FindInPotions(name)
	}
	if !found {
		item, found = char.FindInComponents(name)
	}
	return item, found
}
```

Add the imports this needs: `fmt`, `github.com/GoMudEngine/GoMud/internal/{buffs,characters,events,items,messaging,mobs,mudlog,rooms,shops,skills,util}`. Remove the temporary `var _ = items.Item{}` line.

- [ ] **Step 4: Implement merchant resolution + the two `Sell` modes**

Replace the stub `Sell` body:

```go
func Sell(seller Actor, opts SellOptions) SellResult {
	room := seller.GetRoom()
	if room == nil {
		return SellResult{Reason: SellStopNoMerchant}
	}

	if opts.SellAllSellable {
		return sellSweep(seller, room)
	}

	if opts.Quantity < 1 {
		opts.Quantity = 1
	}
	return sellNamed(seller, room, opts.ItemName, opts.Quantity)
}

// resolveMerchant returns the first merchant in the room willing to buy probe,
// plus its shop inventory. Returns nil mob if none willing.
func resolveMerchant(room *rooms.Room, probe items.Item) (*mobs.Mob, *shops.ShopInventory) {
	for _, mobId := range room.GetMobs(rooms.FindMerchant) {
		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}
		shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
		var probeValue int
		if shopInv != nil {
			cfg := shops.PricingConfigFromBalance()
			wornItems := mob.Character.Equipment.GetAllItemsWithEmptySlots()
			offer := shops.EvaluateBuyRules(probe, shopInv, mob.CrafterSkill, mob.BuysGeneral, cfg, wornItems)
			probeValue = offer.Price
		} else {
			probeValue = mob.GetSellPrice(probe)
		}
		if probeValue > 0 {
			return mob, shopInv
		}
	}
	return nil, nil
}

// sellNamed sells up to quantity copies of one item by name.
func sellNamed(seller Actor, room *rooms.Room, itemName string, quantity int) SellResult {
	char := seller.GetCharacter()
	probe, found := sellFindItemInChar(char, itemName)
	if !found {
		if seller.IsPlayer() {
			seller.SendText(messaging.CategorySystem, "You don't have that item.")
		}
		return SellResult{Reason: SellStopNoItem}
	}
	mob, shopInv := resolveMerchant(room, probe)
	if mob == nil {
		return SellResult{Reason: SellStopNoMerchant}
	}
	var out SellResult
	out.Reason = SellStopSoldAll
	for out.Sold < quantity {
		value, res := sellOneToMerchant(seller, itemName, room, mob, shopInv)
		if res != SellStopSoldAll {
			out.Reason = res
			break
		}
		out.Sold++
		out.TotalGold += value
		out.LastItemName = probe.GetSpec().Name
	}
	return out
}

// sellSweep (mob-only mode) offers every sellable inventory item once,
// skipping rejected ones. Excludes quest items, zero-value items, and crafting
// materials (the mob keeps its own components).
func sellSweep(seller Actor, room *rooms.Room) SellResult {
	char := seller.GetCharacter()
	var out SellResult
	out.Reason = SellStopSoldAll
	// Snapshot to avoid mutation-during-iteration as items are removed.
	snapshot := append([]items.Item{}, char.Items...)
	soldAny := false
	for _, itm := range snapshot {
		spec := itm.GetSpec()
		if spec.ItemId < 1 || spec.QuestToken != "" || spec.Value <= 0 || spec.IsComponent {
			continue
		}
		mob, shopInv := resolveMerchant(room, itm)
		if mob == nil {
			continue
		}
		value, res := sellOneToMerchant(seller, itm.Name(), room, mob, shopInv)
		if res == SellStopSoldAll {
			soldAny = true
			out.Sold++
			out.TotalGold += value
			out.LastItemName = spec.Name
		}
	}
	if !soldAny {
		out.Reason = SellStopRejected
	}
	return out
}
```

- [ ] **Step 5: Run the Task-2 tests**

Run: `go test ./internal/actions/ -run TestSell -v`
Expected: PASS.

- [ ] **Step 6: Thin the player wrapper**

Rewrite `internal/usercommands/sell.go` to keep all player-facing parsing + messaging, delegating the sale to `actions.Sell` and rendering from `SellResult`. Keep the existing quantity/`all.name` parsing block verbatim (it builds `quantity` + `itemName`), then:

```go
func Sell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if rest == "" {
		user.SendText(messaging.CategorySystem, "What would you like to sell?")
		return true, nil
	}

	// [KEEP the existing parsing block that yields `quantity int` and
	//  `itemName string`, including the bare-"sell all" rejection and the
	//  "all." diku prefix handling — unchanged from the current file.]

	actor := &actions.UserActor{User: user, Room: room}
	res := actions.Sell(actor, actions.SellOptions{ItemName: itemName, Quantity: quantity})

	switch res.Reason {
	case actions.SellStopNoItem:
		// actions.Sell already sent "You don't have that item." for players.
		return true, nil
	case actions.SellStopNoMerchant:
		user.SendText(messaging.CategorySystem, "There's no merchant here.")
		return true, nil
	}
	if res.Sold == 0 {
		// A merchant message (refusal) was already spoken synchronously.
		return true, nil
	}

	// Confirmation messaging (preserve the existing single vs multi format).
	displayName := res.LastItemName
	if res.Sold == 1 {
		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold your <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, displayName, res.TotalGold))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You sell a <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, displayName, res.TotalGold))
		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> sells a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, displayName), user.UserId)
	} else {
		pluralName := displayName + "s"
		user.EventLog.Add(`shop`, fmt.Sprintf(`Sold %d <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>`, res.Sold, pluralName, res.TotalGold))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You sell %d <ansi fg="itemname">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, res.Sold, pluralName, res.TotalGold))
		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> sells %d <ansi fg="itemname">%s</ansi>.`, user.Character.Name, res.Sold, pluralName), user.UserId)
	}
	if res.Reason == actions.SellStopMerchantBroke {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Sold %d before the merchant ran out of gold.</ansi>`, res.Sold))
	}
	return true, nil
}
```

Delete the now-unused `trySellOne`, `sellResult` const block, `sellFindItem`, and `merchantSay` from `usercommands/sell.go` (they moved to `actions`). Keep the file's package + imports trimmed to what remains.

- [ ] **Step 7: Verify the whole tree builds and existing sell tests pass**

Run: `go build ./... && go test ./internal/usercommands/ ./internal/actions/ ./internal/mobcommands/`
Expected: builds clean; tests PASS. If `usercommands/sell_test.go` referenced the deleted helpers, update it to drive the public `Sell` wrapper instead.

- [ ] **Step 8: Commit**

```bash
git add internal/actions/sell.go internal/actions/sell_test.go internal/usercommands/sell.go
git commit -m "feat(5.4): lift sell into actions.Sell; mob sales never drain shop gold"
```

---

## Task 3: Overstock decay

Add a per-entry "last grew" stamp and a decay sweep that drains unsold non-material overstock down to its restock baseline.

**Files:**
- Modify: `internal/shops/shopinventory.go`
- Create: `internal/shops/overstock_decay.go`
- Test: `internal/shops/overstock_decay_test.go`
- Modify: config files (Step 1)

- [ ] **Step 1: Add config knobs**

In `internal/configs/config.balance.go`, beside `ShopMaxStockMultiplier`, add:

```go
	ShopOverstockDecayRounds ConfigInt `yaml:"ShopOverstockDecayRounds,omitempty"` // Rounds an over-baseline stock entry must sit un-grown before one unit decays (default 21600 ≈ several in-game days)
	ShopOverstockDecayQty    ConfigInt `yaml:"ShopOverstockDecayQty,omitempty"`    // Units removed per decay fire (default 1)
```

In the balance defaults function (the same file that defaults `ShopAbundanceThreshold`, `internal/configs/config.balance.shops.go`):

```go
	if b.ShopOverstockDecayRounds <= 0 {
		b.ShopOverstockDecayRounds = 21600
	}
	if b.ShopOverstockDecayQty <= 0 {
		b.ShopOverstockDecayQty = 1
	}
```

Add the same two keys with those defaults to the `Balance:` block in `_datafiles/config.yaml` (match the surrounding `Shop*` entry style).

- [ ] **Step 2: Add `LastGrewRound` to `StockEntry` and stamp it on growth**

In `internal/shops/shopinventory.go`, add to `StockEntry`:

```go
	LastGrewRound uint64 `yaml:"last_grew_round,omitempty"` // Round Current last increased; drives overstock decay grace period.
```

In `AddStockAtRound` (where `Current` increases), set `entry.LastGrewRound = round`. In `RestockTier` and `Restock` (wherever `e.Current += add`), set `e.LastGrewRound = util.GetRoundCount()` on the same lines. (Search the file for `Current +=` / `.Current++` and stamp each growth site.)

- [ ] **Step 3: Write failing decay tests**

`internal/shops/overstock_decay_test.go`:

```go
package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverstockDecay_DrainsNonMaterialAboveBaseline(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 2, MaxStock: 10, Current: 6, LastGrewRound: 0},
	}}
	// itemId 100 must resolve to a non-component spec in test data; if specs
	// aren't loaded, inject via the isComponentFn seam (see Step 4).
	TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, 5, si.Stock[0].Current, "one unit should decay")
}

func TestOverstockDecay_SkipsComponents(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 200, RestockQty: 0, MaxStock: 10, Current: 8, LastGrewRound: 0},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return true }, 21600, 1)
	assert.Equal(t, 8, si.Stock[0].Current, "components never decay")
}

func TestOverstockDecay_RespectsBaselineFloor(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 5, MaxStock: 10, Current: 5, LastGrewRound: 0},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, 5, si.Stock[0].Current, "at baseline → no decay")
}

func TestOverstockDecay_GracePeriodNotElapsed(t *testing.T) {
	si := &ShopInventory{Stock: []StockEntry{
		{ItemId: 100, RestockQty: 2, MaxStock: 10, Current: 6, LastGrewRound: 99000},
	}}
	TickOverstockDecayWith(si, 100000, func(int) bool { return false }, 21600, 1)
	assert.Equal(t, 6, si.Stock[0].Current, "1000 < 21600 grace → no decay")
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/shops/ -run TestOverstockDecay -v`
Expected: FAIL (functions undefined).

- [ ] **Step 5: Implement decay**

`internal/shops/overstock_decay.go`:

```go
package shops

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// TickOverstockDecay drains unsold non-material overstock. For each stock entry
// whose Current exceeds its restock baseline (RestockQty) and is not a crafting
// material, if at least decayRounds have elapsed since it last grew, remove
// decayQty units (never below the baseline) and re-stamp LastGrewRound so decay
// paces out. Crafting materials (is_component) are never decayed.
//
// Baseline = RestockQty: NPC-dumped/backfilled items (RestockQty 0) drain fully
// to 0 when unsold; staples drain only to the level natural restock maintains.
func TickOverstockDecay(si *ShopInventory, round uint64) {
	b := configs.GetBalanceConfig()
	TickOverstockDecayWith(si, round, isComponentItem, uint64(b.ShopOverstockDecayRounds), int(b.ShopOverstockDecayQty))
}

// TickOverstockDecayWith is the testable core; isComponent + thresholds are
// injected so unit tests need no loaded item specs.
func TickOverstockDecayWith(si *ShopInventory, round uint64, isComponent func(itemId int) bool, decayRounds uint64, decayQty int) {
	if si == nil || decayRounds == 0 || decayQty <= 0 {
		return
	}
	for i := range si.Stock {
		e := &si.Stock[i]
		baseline := e.RestockQty
		if e.Current <= baseline {
			continue
		}
		if isComponent(e.ItemId) {
			continue
		}
		if e.LastGrewRound != 0 && round-e.LastGrewRound < decayRounds {
			continue
		}
		drop := decayQty
		if e.Current-drop < baseline {
			drop = e.Current - baseline
		}
		e.Current -= drop
		e.LastGrewRound = round // pace subsequent decays
	}
}

func isComponentItem(itemId int) bool {
	spec := items.GetItemSpec(itemId)
	return spec != nil && spec.IsComponent
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/shops/ -run TestOverstockDecay -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/shops/shopinventory.go internal/shops/overstock_decay.go internal/shops/overstock_decay_test.go internal/configs/config.balance.go internal/configs/config.balance.shops.go _datafiles/config.yaml
git commit -m "feat(5.4): time-based overstock decay (excludes crafting materials)"
```

---

## Task 4: Forager chest index + vendor backfill

Build a `zone → chest-rooms` index (self-populated from the live foragers, so YAML stays the single source of truth — no static-registry duplication, no all-instance scan), then aggregate forager-chest contents and top off the vendor's neediest stock gaps (free handoff, no gold).

**Files:**
- Create: `internal/forager/chest_index.go`
- Test: `internal/forager/chest_index_test.go`
- Create: `internal/forager/chest_backfill.go`
- Test: `internal/forager/chest_backfill_test.go`
- Modify: `internal/behaviortree/actions_forager.go` (register the chest on storing)

- [ ] **Step 1: Write the chest index + test**

`internal/forager/chest_index.go`:

```go
package forager

import "sync"

// chestIndex maps a zone to the set of forager storage-lockbox room IDs known
// in that zone. Self-populated by RegisterChestRoom as foragers reach their
// storing state, so the YAML-authored storage_chest_room stays the single
// source of truth (no duplication into the static profiles registry, no
// per-tick all-instance scan). Chest rooms are fixed for a server's lifetime,
// so the set only grows.
var (
	chestIndexMu sync.RWMutex
	chestIndex   = map[string]map[int]bool{}
)

// RegisterChestRoom records that zone has a forager lockbox in chestRoom.
// Idempotent. No-op for zero values.
func RegisterChestRoom(zone string, chestRoom int) {
	if zone == "" || chestRoom == 0 {
		return
	}
	chestIndexMu.Lock()
	defer chestIndexMu.Unlock()
	set := chestIndex[zone]
	if set == nil {
		set = map[int]bool{}
		chestIndex[zone] = set
	}
	set[chestRoom] = true
}

// ChestRoomsForZone returns the chest room IDs registered for zone (stable
// order by room id for determinism).
func ChestRoomsForZone(zone string) []int {
	chestIndexMu.RLock()
	defer chestIndexMu.RUnlock()
	set := chestIndex[zone]
	if len(set) == 0 {
		return nil
	}
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}
```

Add `"sort"` to the import block. `internal/forager/chest_index_test.go`:

```go
package forager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChestIndex_RegisterAndLookup(t *testing.T) {
	RegisterChestRoom("stillwater", 4198)
	RegisterChestRoom("stillwater", 4198) // idempotent
	RegisterChestRoom("stillwater", 4199)
	RegisterChestRoom("", 5)              // ignored
	RegisterChestRoom("ironwind", 0)      // ignored
	assert.Equal(t, []int{4198, 4199}, ChestRoomsForZone("stillwater"))
	assert.Nil(t, ChestRoomsForZone("nowhere"))
}
```

> Test isolation: the index is package global. If other tests in the package register rooms, scope assertions to a unique zone name per test (e.g. `"test-zone-A"`).

- [ ] **Step 2: Register the chest when a forager reaches storing**

In `internal/behaviortree/actions_forager.go`, in `tickForagerStoring`, right after the `mob.StorageChestRoom == 0` guard returns (i.e. once we know the forager HAS a chest), register it:

```go
	// 5.4: make this forager's lockbox discoverable to the vendor backfill.
	forager.RegisterChestRoom(mob.Zone, mob.StorageChestRoom)
```

(Place it immediately after the `if mob.StorageChestRoom == 0 { ... }` block at `actions_forager.go:~392`. `behaviortree` already imports `forager`.)

Run: `go test ./internal/forager/ -run TestChestIndex -v` → PASS.
Commit checkpoint:
```bash
git add internal/forager/chest_index.go internal/forager/chest_index_test.go internal/behaviortree/actions_forager.go
git commit -m "feat(5.4): forager chest index (zone -> lockbox rooms), self-populated on storing"
```

- [ ] **Step 3: Write a failing test for gap ordering + caps**

`internal/forager/chest_backfill_test.go` — the pure ranking/transfer core is testable without loaded data by injecting the chest pool. Test `selectBackfillTransfers`:

```go
package forager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestSelectBackfillTransfers_NeediestGapFirst(t *testing.T) {
	si := &shops.ShopInventory{Stock: []shops.StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 9}, // gap 1
		{ItemId: 20, MaxStock: 10, Current: 2}, // gap 8 (neediest)
	}}
	pool := map[int]int{10: 5, 20: 5} // itemId -> available in chests
	got := selectBackfillTransfers(si, pool)
	// Neediest (20) filled first up to its gap (8) but pool has only 5 → 5;
	// then 10 filled up to gap 1.
	assert.Equal(t, 5, got[20])
	assert.Equal(t, 1, got[10])
}

func TestSelectBackfillTransfers_AllToppedOff(t *testing.T) {
	si := &shops.ShopInventory{Stock: []shops.StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 10},
	}}
	pool := map[int]int{10: 5}
	got := selectBackfillTransfers(si, pool)
	assert.Empty(t, got, "no gaps → nothing transferred")
}

func TestSelectBackfillTransfers_OnlyStockedItems(t *testing.T) {
	si := &shops.ShopInventory{Stock: []shops.StockEntry{
		{ItemId: 10, MaxStock: 10, Current: 5},
	}}
	pool := map[int]int{99: 5} // vendor doesn't stock 99
	got := selectBackfillTransfers(si, pool)
	assert.Empty(t, got, "vendor only pulls items it already stocks")
}
```

- [ ] **Step 4: Run to verify it fails**

Run: `go test ./internal/forager/ -run TestSelectBackfillTransfers -v`
Expected: FAIL (undefined).

- [ ] **Step 5: Implement the backfill**

`internal/forager/chest_backfill.go`:

```go
package forager

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// selectBackfillTransfers decides how many of each item to pull from the
// aggregate chest pool into the vendor, neediest stock gap first, capped by
// each entry's MaxStock and by pool availability. Only items the vendor already
// stocks (has a StockEntry for, below MaxStock) are eligible. Pure + testable.
func selectBackfillTransfers(si *shops.ShopInventory, pool map[int]int) map[int]int {
	type gap struct {
		itemId int
		gap    int
	}
	var gaps []gap
	for i := range si.Stock {
		e := &si.Stock[i]
		g := e.MaxStock - e.Current
		if g > 0 && pool[e.ItemId] > 0 {
			gaps = append(gaps, gap{e.ItemId, g})
		}
	}
	// Neediest gap first; itemId tiebreak for determinism.
	sort.Slice(gaps, func(a, b int) bool {
		if gaps[a].gap != gaps[b].gap {
			return gaps[a].gap > gaps[b].gap
		}
		return gaps[a].itemId < gaps[b].itemId
	})
	remaining := map[int]int{}
	for id, n := range pool {
		remaining[id] = n
	}
	out := map[int]int{}
	for _, g := range gaps {
		take := g.gap
		if take > remaining[g.itemId] {
			take = remaining[g.itemId]
		}
		if take > 0 {
			out[g.itemId] = take
			remaining[g.itemId] -= take
		}
	}
	return out
}

// chestPoolForZone aggregates item counts across the forager lockboxes
// registered for the given zone (via the chest index — no instance scan).
// Returns the pool (itemId -> count) plus the chest room ids so the transfer
// step can remove from the right container.
func chestPoolForZone(zone string) (pool map[int]int, chestRooms []int) {
	pool = map[int]int{}
	for _, chestRoom := range ChestRoomsForZone(zone) {
		room := rooms.LoadRoom(chestRoom)
		if room == nil {
			continue
		}
		key := room.FindContainerByName("lockbox")
		if key == "" {
			continue
		}
		c := room.Containers[key]
		empty := true
		for _, it := range c.Items {
			pool[it.ItemId]++
			empty = false
		}
		if !empty {
			chestRooms = append(chestRooms, chestRoom)
		}
	}
	return pool, chestRooms
}

// BackfillVendorFromChests tops off vendorMob's shop from forager chests in its
// zone. Free supply handoff — no gold. Mirrors SellToVendor's persistence.
func BackfillVendorFromChests(vendorMob *mobs.Mob, shopInv *shops.ShopInventory) {
	if vendorMob == nil || shopInv == nil {
		return
	}
	pool, chestRooms := chestPoolForZone(vendorMob.Zone)
	if len(pool) == 0 {
		return
	}
	transfers := selectBackfillTransfers(shopInv, pool)
	if len(transfers) == 0 {
		return
	}

	mutated := false
	for itemId, want := range transfers {
		moved := 0
		for _, chestRoomId := range chestRooms {
			if moved >= want {
				break
			}
			room := rooms.LoadRoom(chestRoomId)
			if room == nil {
				continue
			}
			key := room.FindContainerByName("lockbox")
			if key == "" {
				continue
			}
			c := room.Containers[key]
			for moved < want {
				it, ok := c.FindItemById(itemId)
				if !ok {
					break
				}
				c.RemoveItem(it)
				entry := shopInv.GetStock(itemId)
				if entry == nil || entry.Current >= entry.MaxStock {
					// Shouldn't happen (selectBackfillTransfers capped it), but
					// put the item back rather than vaporize it.
					c.AddItem(it)
					break
				}
				entry.Current++
				moved++
				mutated = true
			}
			room.Containers[key] = c
		}
	}

	if mutated {
		if err := shops.SaveShop(vendorMob.Zone, int(vendorMob.MobId), vendorMob.HomeRoomId); err != nil {
			mudlog.Error("forager.BackfillVendorFromChests", "vendor", vendorMob.Character.Name, "error", err)
		}
	}
}
```

> Note on `room.Containers[key] = c`: `Container` is a value type in the map; mutating `c` requires writing it back. Confirm whether `room.Containers` stores `Container` or `*Container` — if pointer, drop the write-backs. (Check `rooms.Room.Containers` field type; `actTryStoreExcess` reads `container := room.Containers[key]` then calls `container.Lock.IsLocked()`, suggesting value or pointer — verify and adjust.)

- [ ] **Step 6: Run tests**

Run: `go test ./internal/forager/ -run TestSelectBackfillTransfers -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/forager/chest_backfill.go internal/forager/chest_backfill_test.go
git commit -m "feat(5.4): forager-chest aggregate backfill into neediest vendor gaps"
```

---

## Task 5: Wire decay + backfill into the shopkeeper restock tick

Run market participation **only when a restock actually fired** this tick (the spec's "per-vendor restock tick"), not on every idle tick — restocks fire on a slow per-tier cadence, so the chest enumeration + decay stay infrequent.

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go`

- [ ] **Step 1: Capture whether a restock fired, then run market participation**

In `internal/hooks/MobIdle_HandleIdleMobs.go`, the two restock paths are: the non-crafter branch `if mobs.TickMobShopRestock(mob) { ... }` (~line 50) and the crafter branch `if result := mobs.TickMobCraft(mob); result != nil { ... }` (~line 64). Capture a `restocked` flag from each:

- In the non-crafter branch, set a `restocked := false` before the `if`, and inside the `if mobs.TickMobShopRestock(mob)` true-branch set `restocked = true`. (Restructure to `didRestock := mobs.TickMobShopRestock(mob); if didRestock { ...emote...; restocked = true }`.)
- In the crafter branch, set `restocked = true` when `result != nil && result.Restocked`.

Then, after both branches (~line 90), add:

```go
	// 5.4 NPC market participation: on a restock tick, drain stale non-material
	// overstock and top the shop off from forager chests in this zone. Covers
	// both crafter and non-crafter shopkeepers. Gated on restocked so the chest
	// enumeration runs only on the slow restock cadence, not every idle tick.
	if restocked {
		if shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId); shopInv != nil {
			shops.TickOverstockDecay(shopInv, util.GetRoundCount())
			forager.BackfillVendorFromChests(mob, shopInv)
			if err := shops.SaveShop(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
				mudlog.Error("MobIdle.market", "error", err)
			}
		}
	}
```

Ensure imports include `github.com/GoMudEngine/GoMud/internal/{forager,shops}` (and `util`, `mudlog` if not already present). `BackfillVendorFromChests` already saves on mutation; this single `SaveShop` also persists decay changes — one save after both is fine.

> **Scope note:** the non-crafter branch is skipped in caravan-served zones (`IsCaravanServedZone`). In those zones a non-crafter vendor restocks only on caravan visits, so its market-participation also only runs then — acceptable (the caravan is that zone's supply line). Crafter shops always tick. If forager backfill is later wanted in caravan zones independent of caravan cadence, revisit here.

- [ ] **Step 2: Build + boot smoke**

Run: `go build ./...`
Then nuke instance saves and boot (per CLAUDE.md SOP):
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Start the server; confirm it reaches `Server Ready` with no panic and shops/mobs load.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go
git commit -m "feat(5.4): run overstock decay + chest backfill on shopkeeper idle tick"
```

---

## Task 6: Forager chest-full rest back-pressure

Keep a forager resting (don't start a new gather cycle) while its chest is full; resume once the backfill drains it to ≤ resume %.

**Files:**
- Modify: `internal/behaviortree/actions_forager.go`
- Modify: config (resume %) — reuse Step 1 pattern

- [ ] **Step 1: Add the resume-% knob**

In `internal/configs/config.balance.go` (near the forager knobs like `ForagerRestCarryThreshold`):

```go
	ChestBackpressureResumePct ConfigFloat `yaml:"ChestBackpressureResumePct,omitempty"` // Chest fill fraction at/below which a rested forager resumes gathering (default 0.9)
```

Default in the forager/balance defaults function:

```go
	if b.ChestBackpressureResumePct <= 0 {
		b.ChestBackpressureResumePct = 0.9
	}
```

Add the key to `_datafiles/config.yaml` `Balance:` block.

- [ ] **Step 2: Add a chest-fill helper**

In `internal/behaviortree/actions_forager.go` (or a small new helper near `carryRatio`):

```go
// chestFillRatio returns the fill fraction (0..1) of the forager's storage
// lockbox, or 0 if it has no chest / the chest can't be loaded. Capacity is the
// chest room's StorageCapacity (default 20 when unset).
func chestFillRatio(mob *mobs.Mob) float64 {
	if mob.StorageChestRoom == 0 {
		return 0
	}
	room := rooms.LoadRoom(mob.StorageChestRoom)
	if room == nil {
		return 0
	}
	key := room.FindContainerByName("lockbox")
	if key == "" {
		return 0
	}
	c := room.Containers[key]
	capacity := room.StorageCapacity
	if capacity <= 0 {
		capacity = 20
	}
	return float64(len(c.Items)) / float64(capacity)
}
```

- [ ] **Step 3: Gate the cycle restart**

In `tickForagerResting` (`actions_forager.go:~225`), immediately before
`transitionForager(ctx.MobState, forager.StateTravelingToTerritory)` (line ~237), insert:

```go
		// 5.4 back-pressure: don't start a new gather cycle while the storage
		// chest is full. The vendor restock backfill drains it over time; once
		// at/below the resume fraction, resume. Prevents foraging into a void.
		resumePct := float64(configs.GetBalanceConfig().ChestBackpressureResumePct)
		if chestFillRatio(mob) > resumePct {
			return Failure // stay resting; legacy idle fires flavor emotes
		}
```

(`Failure` here matches the existing "still resting" return at the end of the function.)

- [ ] **Step 4: Build + test the package**

Run: `go build ./... && go test ./internal/behaviortree/ ./internal/forager/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/actions_forager.go internal/configs/config.balance.go internal/configs/config.balance.shops.go _datafiles/config.yaml
git commit -m "feat(5.4): forager rests while storage chest is full (drain-to-resume)"
```

---

## Task 7: Docs, full verification, roadmap

**Files:**
- Modify: `internal/actions/context.md`, `internal/shops/context.md`, `internal/forager/context.md`
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Document the two sell models + new mechanics**

- `internal/actions/context.md`: note `actions.Sell` and the sale-vs-supply split (mob sales don't drain shop gold; `forager.SellToVendor`/backfill are free handoffs).
- `internal/shops/context.md`: document `LastGrewRound` + `TickOverstockDecay` (baseline = RestockQty, excludes `is_component`).
- `internal/forager/context.md`: document `BackfillVendorFromChests` + the chest-full rest back-pressure and that the chest "overflow cache" now drains.

- [ ] **Step 2: Full build + test sweep**

Run: `go build ./... && go test ./...`
Expected: all packages PASS. Record any pre-existing-failing tests separately (do not fix unrelated breakage here).

- [ ] **Step 3: Boot smoke (clean instances)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server; confirm `Server Ready`, no panic, mob `sell` command registered (grep boot log / `mobs.LoadDataFiles` counts unaffected).

- [ ] **Step 4: Update the roadmap**

In `MOB_ALIVENESS_ROADMAP.md`: flip chunk 5.4 Status to `Done (2026-06-02)` in both the progress tracker table and the 5.4 mini-brief; add a Shipped paragraph summarizing the three components + the deferred proactive-offload goal (gated on mob corpse-salvage yield). Update the roll-up count.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/context.md internal/shops/context.md internal/forager/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(5.4): context + roadmap for NPC market participation"
```

---

## In-game smoke checklist (deferred to user, per 2.8/2.9/2.10/5.3 precedent)

- [ ] A `wealth-gold` thief with stolen goods at a vendor now actually sells (no `looks a little confused` emote); its gold rises and the items appear in shop stock.
- [ ] An `upgrade-gear` mob with carried gold + an upgrade in stock completes buy→gearup.
- [ ] Confirm a mob sale does NOT reduce the shop's gold (sell to a near-broke shop as a mob, then as a player — only the player drains it).
- [ ] Forager chest fills to capacity → forager stays resting (stops gathering); buy the stocked goods from the zone vendor → on its next restock tick the chest drains → forager resumes gathering.
- [ ] Leave a pile of unsold non-material stock at a vendor → it slowly shrinks over several in-game days; a stocked crafting material at the same vendor does NOT shrink.

---

## Self-review notes (gaps flagged for the implementer)

- **`room.Containers` element type** (value vs pointer) is unconfirmed — Task 4 Step 3 and Task 6 Step 2 read `room.Containers[key]`. Verify the field type in `internal/rooms/rooms.go`; if value-typed, the Task-4 transfer must write the container back into the map (shown) and Task-6 read-only is fine; if pointer-typed, drop the write-backs.
- **`RestockTier` vs `Restock` growth sites** (Task 3 Step 2): stamp `LastGrewRound` at every `Current` increase. Confirm both method bodies in `shopinventory.go` / wherever `RestockTier` lives.
- **`SaveShop` of forager chest rooms:** the backfill mutates `room.Containers`; if container contents persist via room instance saves (which prod wipes), the drain is per-instance-lifetime only — acceptable and consistent with `SellToVendor` (same limitation). No extra chest persistence added.
- **Chest index is lazily populated** (Task 4): a forager's chest only enters the index once that forager reaches its `storing` state at least once. On a freshly booted server, backfill pulls nothing until foragers have run a gather→deliver→store cycle — which is correct (an unvisited chest is empty anyway). The in-game smoke must let a forager complete a storing cycle before expecting backfill, and the vendor must share the chest room's registered zone (`mob.Zone` at storing time). Confirm Tova's chest zone matches her vendor rooms' zone during smoke; if they differ, index by the vendor-facing zone instead.
