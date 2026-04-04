# Living Economy — Stage 1

**Date:** 2026-04-04
**Goal:** Replace infinite static shops with a living economy where NPC
merchants have finite stock, dynamic pricing driven by supply, autonomous
crafting driven by profit decisions, and rule-based buying behavior. Add
non-combatant protection for shop NPCs.

---

## 1. Finite Shop Inventories

### Current State

Shops are defined as a `Shop []ShopItem` slice on `Character`. Each
`ShopItem` has a `Quantity`, `QuantityMax` (0 = unlimited), and
`RestockRate`. Buying calls `Shop.Destock()`, selling calls
`Shop.StockItem()`. Stock does NOT persist across restarts — shops
reset to template on spawn.

### New Model

**Shop inventory becomes persistent zone state**, decoupled from mob
lifecycle. A `ShopInventory` struct stored per shop-mob (keyed by
mob template ID + room ID) that survives restarts, deaths, and
respawns.

**Fields per stocked item:**

```yaml
shop:
  stock:
    - item_id: 40001        # iron ore
      restock_qty: 8        # supply cart brings this many
      max_stock: 15         # hard cap on accumulation
      current: 10           # actual current stock (persisted)
    - item_id: 10005        # iron dagger (finished good)
      restock_qty: 0        # NOT restocked — NPC crafted
      max_stock: 8          # NPC stops crafting past this
      current: 3
```

Items with `restock_qty: 0` are finished goods produced by the NPC's
craft system, not by the supply cart.

**Persistence:** Shop inventory saves to a new directory
`_datafiles/world/dogmud/shops/` keyed by zone + mob ID + room ID.
Loaded at startup, saved on change.

**Migration:** Existing `Shop []ShopItem` definitions on mobs become
the initial template. On first load with no persisted shop file, the
system seeds `current` = `restock_qty` for materials and a reasonable
starting stock for finished goods.

### What Changes

- `buy` command reads from `ShopInventory` instead of `Shop` slice
- `sell` command writes to `ShopInventory`
- `Shop` slice on Character becomes the template/seed only
- New `ShopInventory` type in a new package or in `internal/mobs/`

---

## 2. Dynamic Pricing

### Formula

```
sellPrice = baseValue × scarcityMult(current, restockQty)
buyPrice  = baseValue × ShopBuyRatio × scarcityMult(current, restockQty)
```

**Config knobs:**
- `ShopBuyRatio` (default 0.50) — base buy/sell spread. At baseline
  stock, NPC sells for 100% of base value and buys at 50%.
- `ShopPriceFloor` (default 0.25) — minimum scarcity multiplier
  (when massively overstocked)
- `ShopPriceCeiling` (default 5.0) — maximum scarcity multiplier
  (when nearly/completely out)

### Scarcity Multiplier Curve

The curve is normalized by `restock_qty` — rare items (restock 1)
spike faster than common items (restock 10).

```
ratio = current / restock_qty

if ratio >= abundanceThreshold:
    mult = PriceFloor  (0.25)
elif ratio <= 0:
    mult = PriceCeiling (5.0)
else:
    mult = interpolate on a smooth curve from PriceCeiling to PriceFloor
```

The interpolation uses an inverse curve so prices rise sharply as
stock approaches zero:

```
mult = PriceFloor + (PriceCeiling - PriceFloor) × (1 - ratio/abundanceThreshold)^2
```

Where `abundanceThreshold` (default 3.0) means stock at 3× restock
quantity is considered fully abundant. Config: `ShopAbundanceThreshold`.

**Examples (restock_qty=5, defaults):**

| Current Stock | Ratio | Mult  | Sell (base=10g) | Buy (base=10g) |
|---------------|-------|-------|-----------------|-----------------|
| 0             | 0.0   | 5.00x | 50g             | 25g             |
| 1             | 0.2   | 4.15x | 41g             | 21g             |
| 3             | 0.6   | 2.85x | 29g             | 14g             |
| 5 (baseline)  | 1.0   | 1.90x | 19g             | 10g             |
| 10            | 2.0   | 0.73x | 7g              | 4g              |
| 15+           | 3.0+  | 0.25x | 3g              | 1g              |

### Bartering Skill Modifier

The player's bartering skill applies as a modifier on top of dynamic
pricing:

```
finalSellPrice = sellPrice × (1 - barterDiscount)
finalBuyPrice  = buyPrice × (1 + barterBonus)
```

Where `barterDiscount` and `barterBonus` scale with skill rank via
the existing `SkillMultiplier` curve. Config knobs:
`BarterMaxDiscount` (default 0.15 = 15% off sell prices at max skill),
`BarterMaxBonus` (default 0.15 = 15% more on buy prices at max skill).

---

## 3. Timer-Based Material Restocking

### Mechanism

Each shop has a `restock_interval` (in game ticks). When the interval
elapses:

1. For each item with `restock_qty > 0`:
   - Add `min(restock_qty, max_stock - current)` to stock
   - If stock was already at or above `max_stock`, add nothing
2. Fire a restock emote to the room (flavor text)
3. Save updated shop inventory to disk

### Restock Configuration

```yaml
shop:
  restock_interval: 1440    # ticks between deliveries
  restock_emote: >-
    A weathered supply cart rumbles to a stop outside the smithy.
    The driver unloads crates of raw materials before moving on.
```

If `restock_emote` is empty, use a generic default:
"A delivery arrives for {mobname}."

### No Gold Cost

The NPC does not pay for restocked materials. The supply cart is an
abstracted wholesale arrangement. This prevents death spirals where
broke NPCs can't restock. Future work: real supplier NPCs with
economic relationships.

### Integration Point

Restock check runs inside the existing `MobIdle_HandleIdleMobs` hook,
alongside the existing `TickMobCraft` call. The restock fires first
(materials arrive), then the craft decision runs (NPC considers
crafting with new materials).

---

## 4. NPC Craft Decisions

### Current State

`TickMobCraft()` in `internal/mobs/crafter.go` already handles
autonomous NPC crafting with material restocking. The new system
replaces the craft selection logic with profit-aware decisions.

### Decision Priority

On each idle tick (after restock), the NPC evaluates crafts in this
order:

**Priority 1: Self-gear upgrade**

For each known recipe that produces an equippable item:
- Would this item be better than what I'm currently wearing in
  that slot?
- Do I have the materials (respecting reserve)?
- If yes: craft it, equip on completion.

This means NPCs naturally gear up over time. A blacksmith who has
been crafting for weeks will be wearing impressive gear — emergent
world flavor.

**Priority 2: Most profitable stock item**

For each known recipe where output `restock_qty == 0` (finished good):
- `materialCost` = sum of `baseValue × scarcityMult` for each
  ingredient at current stock levels (opportunity cost)
- `productValue` = `baseValue × scarcityMult` for the output item
  at current stock level
- Skip if `current >= max_stock` for that product
- Skip if any ingredient would drop to or below `ShopMaterialReserve`
- Skip if `productValue <= materialCost` (not profitable)

Craft the option with the highest `productValue - materialCost`.

**Priority 3: Profitable salvage**

When overstocked on finished goods but low on materials, NPCs break
items down for parts using the existing salvage system.

For each finished good in stock where `current > 1`:
- `itemValue` = `baseValue × scarcityMult` at current stock level
- `salvageValue` = sum of expected material returns, each valued at
  `baseValue × scarcityMult` at that material's current stock level
- Skip if `salvageValue <= itemValue` (item worth more intact)

Salvage the option with the highest `salvageValue - itemValue`.

Uses the same `salvage_returns` data on ItemSpec or recipe-based
ingredient recovery from the existing salvage system. NPC salvage
skill progresses naturally through use like any other skill.

### Reserve Threshold

NPCs hold back the last N of any material so it's still available
for players to buy (at a high scarcity premium).

Config: `ShopMaterialReserve` (default 1).

### Craft Frequency

One craft decision per idle tick. Uses the existing multi-round
craft system (`TimeRounds`). The NPC only evaluates a new craft
when not already crafting.

### Recipe Knowledge

- `shop.starting_recipes` — list of recipe IDs the NPC knows at spawn
- NPCs also discover recipes through normal crafting progression
  (use-based, same as players). Over time they learn more.
- `shop.craft_skill` — which skill governs their crafting (matches
  existing `CrafterSkill` field)

---

## 5. Rule-Based Buy Behavior

### Current State

`sell.go` uses `mob.GetSellPrice()` which checks if the item type
or subtype matches existing shop stock, then applies a scaling factor
(max 25% of item value). Any merchant will buy anything.

### New Model

NPCs evaluate buy offers based on composable rules tied to their
role. Rules are checked in priority order; the first matching rule
determines the offer.

### Buy Rules

**Rule 1: Gear Upgrade**

If the offered item is equipment AND it's better than what the NPC
is wearing in that slot AND the NPC has enough gold:
- Offer up to `baseValue × ShopBuyRatio × scarcityMult` (generous,
  they want it)
- On purchase: equip immediately

"Better" is determined by a simple item power comparison: sum of
stat modifiers + damage multiplier + mitigation values. A new utility
function `items.ComparePower(a, b)` returns which item is stronger
for the relevant slot.

**Rule 2: Craft Materials**

If the NPC has a `craft_skill` AND the item has a `component_tag`
matching ingredients used in that skill's recipes:
- Offer `baseValue × ShopBuyRatio × scarcityMult` at current stock
- Diminishing returns: as stock grows, scarcityMult drops toward
  `PriceFloor` (0.25x)
- Won't buy past `max_stock` for that item

**Rule 3: Potions (Non-Expired)**

If the NPC has a `craft_skill` of alchemy OR is flagged as a
general merchant AND the item is a potion:
- Check aging phase — reject if Declining or Spoiled
- Offer `baseValue × ShopBuyRatio × scarcityMult`
- Same diminishing returns as craft materials

**Rule 4: General Goods**

If the NPC is flagged as a general merchant (`shop.buys_general`):
- Buy anything not covered above at `baseValue × 0.25` (steep
  discount, they're a junk dealer)
- Still won't buy quest items

### What NPCs Won't Buy

- Quest items (`is_quest_item` flag)
- Spoiled or declining-phase potions
- Items they already have at `max_stock`
- Items they can't afford (gold check)
- Items with no matching buy rule (specialist shops)

### NPC Gold

Merchants start with a `shop.starting_gold` value (seed for first
boot only) and accumulate gold through sales. Gold persists across
restarts in the shop save file. If they run out, they can't buy
from players ("I can't afford that right now" — already implemented).

**Gold reserve:** NPCs won't spend below `starting_gold ×
ShopGoldReserveRatio` on gear purchases. Materials buying can dip
lower since that's core business. This prevents an NPC from blowing
all their gold on a nice helmet and being unable to buy materials.

Config: `ShopGoldReserveRatio` (default 0.50).

---

## 6. Non-Combatant Protection

### Mechanism

A `non_combatant` boolean flag on mob YAML. When true:

- `attack` command: "You can't attack {name}."
- `bash`/`kick`/`trip`/`grapple`/`taunt`: same rejection
- `shoot`: same rejection
- `steal`/`pickpocket`: "You can't steal from {name}."
- `cast` (harm spells targeting the mob): "You can't target {name}
  with a harmful spell."
- Mob never sets aggro, never enters combat loop
- AoE harm spells skip non-combatant mobs in target resolution
- Companions don't auto-target non-combatant mobs

### Data

```yaml
non_combatant: true
```

All existing shop mobs in Thornwall and other towns get this flag.

---

## 7. Shop Data Model

### YAML Definition (mob template)

```yaml
shop:
  restock_interval: 1440
  restock_emote: >-
    A supply cart arrives with fresh materials for the smithy.
  starting_gold: 500
  buys_general: false
  craft_skill: blacksmithing
  starting_recipes:
    - iron-dagger
    - iron-sword
    - iron-helm
  stock:
    - item_id: 40001
      restock_qty: 8
      max_stock: 15
    - item_id: 40002
      restock_qty: 6
      max_stock: 12
    - item_id: 10005
      restock_qty: 0
      max_stock: 8
```

### Persisted Shop State

Saved to `_datafiles/world/dogmud/shops/{zone}/{mobid}-room{roomid}.yaml`:

```yaml
last_restock: 145200
gold: 487
inventory:
  40001: 10
  40002: 7
  10005: 3
```

Simple map of item_id → current count, plus gold and restock timer.

---

## 8. Migration Plan

### Phase 1: Infrastructure

- New `ShopInventory` type with persistence
- Dynamic pricing functions
- Non-combatant flag and all combat/theft checks
- Restock timer integration into `MobIdle`

### Phase 2: Buy/Sell Rework

- Rewrite `buy.go` to read from `ShopInventory` with dynamic pricing
- Rewrite `sell.go` to use rule-based buy behavior
- NPC gold tracking

### Phase 3: Craft Decision Rework

- Replace `TickMobCraft` selection logic with profit-aware decisions
- Self-gear priority
- Reserve threshold respect
- Integration with `ShopInventory` for stock tracking

### Phase 4: Data Migration

- Convert existing merchant mobs to new `shop:` YAML format
- Add `non_combatant: true` to all town merchants
- Seed initial shop inventory files
- Populate `starting_recipes` for crafter NPCs

---

## 9. Config Knobs Summary

All under `Balance` in `config.yaml`:

| Knob | Default | Description |
|------|---------|-------------|
| `ShopBuyRatio` | 0.50 | Base buy/sell spread (NPC buys at 50% of sell) |
| `ShopPriceFloor` | 0.25 | Min scarcity multiplier (way overstocked) |
| `ShopPriceCeiling` | 5.0 | Max scarcity multiplier (out of stock) |
| `ShopAbundanceThreshold` | 3.0 | Stock/restock ratio considered fully abundant |
| `ShopMaterialReserve` | 1 | NPCs hold back this many of each material |
| `BarterMaxDiscount` | 0.15 | Max sell price discount at max bartering skill |
| `BarterMaxBonus` | 0.15 | Max buy price bonus at max bartering skill |
| `ShopGoldReserveRatio` | 0.50 | NPC won't spend below this % of starting gold on gear |

---

## 10. Documentation Updates

The implementation plan must include updates to:

- `internal/mobs/context.md` — new shop persistence model, shop
  directory, crafter decision logic, non-combatant flag
- `internal/items/context.md` — `ComparePower` utility, salvage
  integration with shop NPCs
- `internal/characters/context.md` — shop inventory decoupled from
  Character.Items, gold reserve behavior
- Help templates — update `buy`, `sell`, `craft`, `salvage` help
  files to reflect dynamic pricing and NPC buy rules
- `CLAUDE.md` — add shop persistence notes (shops/ directory
  separate from rooms.instances/ and mobs.instances/, not cleaned
  by instance save SOP)

---

## 11. Follow-Up Work (Not in Scope)

- **World material sources** — ore nodes, foraging spots, mob loot
  tables with crafting materials. The shop economy assumes materials
  enter the game via restocking, player crafting, mob drops, and
  future gathering systems. Expanding world sources is separate.
- **Real supplier NPCs** — replace flavor-text supply carts with
  actual NPC caravans that transport goods between towns.
- **Idle complaint text** — NPCs grumble about low stock or
  overstock ("Running low on iron ore..." / "I've got daggers
  coming out my ears..."). Data is there, just needs emote hooks.
- **Disposition modifiers** — NPC attitudes adjust prices and
  willingness to trade. Builds on top of the pricing system.
- **Thornwall bank** — separate spec for gold/item storage.
