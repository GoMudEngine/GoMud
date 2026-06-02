# Cooking Supply Chain — Forager-Salvaged Meat + Cooks as Crafters (Design)

**Status:** Draft — awaiting user spec review
**Date:** 2026-06-02
**Related:** mob-aliveness roadmap (Phase 6 polish / living-economy); follow-on to
5.4 (overstock decay + market participation) and the deferred sell-surplus goal.
First of a two-part economy-fix split — this is the **cooking** half; the
**enchanting** half (decayed potions → enchanter) is a separate later chunk.
Addresses [[project_store_restock_considered_fix]] for the cooking vendor type.

---

## Goal

Make the cooking economy self-sustaining and alive: town cooks **produce** their
cooked meals from raw ingredients (instead of meals magically arriving on a
supply cart), and **hunters/foragers supply the meat** by salvaging the prey
they kill. The loop: a forager hunts game → salvages the corpse for raw meat →
delivers it to the town cook on its existing route → the cook (now a crafter)
turns meat + staples into grilled meat, stew, rations, etc. → players buy the
meals. Shop stock reflects NPC activity, not an invisible cart.

## Why now

The survey of cooking vendors found two structural problems behind "cooking
stores don't restock well":

1. **Cooks aren't crafters.** Tov Brann (336, Stillwater), food vendor (103,
   Thornwall), and tavern cook Brynn (248, Thornwall) carry
   `craft_support: cooking` but lack `crafter: true`. Their cooked meals
   (grilled meat 30022, hearty stew 30023, herbal tea 30024, trail rations
   30021, …) have **no producer** — they only refill via the generic supply
   cart, which is the band-aid this chunk replaces.
2. **Raw meat has no living source.** `raw-meat` (40014) is needed by the core
   cooking recipes but is cart-only today; corpse salvage yields just
   leather/sinew/cloth (`internal/crafting/corpse_salvage.go`), no meat — even
   though foragers already hunt and salvage prey.

The forager supply chain (hunt → salvage → deliver, chunk 3.8/5.4) already runs
and already visits all three cooks' rooms (Tova → 4102; Halix → 464 and 481).
The only genuinely new code is making the cooks crafters; the rest is yield-table
and bucket data.

## Background — what already exists (reused, not rebuilt)

- **Corpse salvage mechanic** (`actions.Salvage`, corpse mode; chunk 2.9): actor-
  pattern, player + mob. `crafting.LookupCorpseSalvage(mobGroups) []SalvageReturn`
  drives yields from a static `corpseSalvageTable` (first matching group wins).
  Current table: `animal` → leather-strip×2 + sinew; `humanoid` → cloth-strip×2 +
  leather-strip.
- **Forager loop** (`_datafiles/world/dogmud/behaviors/archetypes/forager.yaml`)
  already runs `try_salvage` before `try_forage` each foraging tick, and foragers
  hunt prey via `PreyWhitelist`. Salvage returns land in the forager's satchel.
- **Forager → vendor free-handoff** (`forager.SellToVendor`, chunk 3.8): for each
  satchel item, if `economy.BucketFor(itemId)` ∈ the forager's `Buckets` AND the
  vendor has a stock entry for it below `MaxStock`, the item transfers into the
  vendor's stock (no gold). Routes already cover the cooks:
  - Tova (371): `Buckets: [stillwater, base, overlap]`, VendorRooms includes
    **4102** (Tov Brann 336).
  - Halix (372): `Buckets: [base, overlap]`, VendorRooms includes **464** (food
    vendor 103) and **481** (tavern cook Brynn 248).
- **Crafter system** (`mobs.TickMobCraft`, `shops.EvaluateCraftOptions`): a mob
  with `crafter: true` + `crafterskill` + `crafterrecipeids` +
  `crafterrestockmaterials` restocks its ingredient materials per tier, then
  crafts a recipe when output < MaxStock, all ingredients are present (minus a
  reserve), and the craft is profitable. Output is added to shop stock.
  **Reference template: apothecary Ilsa (338)** — `crafter: true`,
  `crafterskill: alchemy`, recipe + material id lists, `behavior_archetype`
  unchanged (`noncombat_shopkeeper`).
- **Cooking recipes** exist in `_datafiles/world/dogmud/recipes/cooking/`
  (grilled-meat, trail-rations, hearty-stew, herbal-tea, stillwater-lake-chowder,
  antidote-broth, energy-bread, spiced-wine).
- **Economy buckets** (`internal/economy/buckets.go`): `raw-meat (40014) → base`;
  `wild-hare-meat (40064) → fernway`; `shadowcap (40063) → fernway`;
  `blood-moss (40066) → fernway`.

## Design decisions (locked during brainstorming)

1. **Forager-driven only; combat looters dropped.** The combat-looter sell path
   was considered and dropped: raw-meat is `is_component` (value 1), so 5.4's
   `sell-surplus` sweep skips it and selling value-1 meat isn't worth a looter's
   trip — and bumping meat value would tax new player cooks grinding the skill.
   Foragers supply the bulk of the meat via the free-handoff, untouched by the
   component/value rules. The deferred sell-surplus goal stays deferred until a
   higher-value surplus source exists.

2. **Cooks become crafters; cooked meals flip to crafter-produced.** Mark 336,
   103, 248 as `crafter: true` + `crafterskill: cooking` with cooking recipe ids
   + ingredient `crafterrestockmaterials`, mirroring Ilsa. Their cooked-meal
   stock entries change from `RestockQty: 5` (cart band-aid) to `RestockQty: 0`
   (crafter-produced only), so the cook is the real producer. Raw INGREDIENT
   entries keep their cart `RestockQty` as the baseline supply floor.

3. **Salvage yields meat, kept cheap.** Add `raw-meat` to the `animal` corpse
   group (value unchanged at 1 to keep cooking cheap to grind). Add
   `wild-hare-meat` for small-game/prey corpses (see decision 4). Existing
   leather/sinew/cloth returns are preserved.

4. **wild-hare-meat re-bucketed to `overlap` so foragers can deliver it.**
   `wild-hare-meat` (40064) is currently `fernway` bucket — only Kessa carries
   it, and Kessa delivers via the caravan, not vendor rooms. Re-bucket it to
   `overlap` so Tova + Halix (who both hunt hare/rat prey and carry the `overlap`
   bucket) deliver it directly to the cooks. Map the meat by prey group:
   small-game/rodent prey corpses → `wild-hare-meat`; generic `animal` →
   `raw-meat`. Content cleanup: marsh rat (367) is mis-tagged to carry
   `wild-hare-meat`; its salvage should yield generic `raw-meat`, not hare meat.

5. **Fernway forageables deferred.** `shadowcap` (40063) and `blood-moss` (40066)
   are cooking ingredients in the `fernway` bucket that aren't in any forage-yield
   table — but routing them to the cooks needs the Kessa→caravan mesh, and the
   Fernway caravan's `deliveries_by_tier` is empty. That's a separate
   caravan-supply problem. They are **cart-supplied today (RestockQty 5, not
   starved)**, so v1 leaves them on the cart and logs the caravan gap as a
   follow-up. (Same for any other `fernway`-bucket cooking input.)

## Architecture / components

### Component 1 — Salvage yields meat (`internal/crafting/corpse_salvage.go`)

Extend `corpseSalvageTable`. Because `LookupCorpseSalvage` returns the **first**
table entry whose group appears in the mob's `groups`, order matters: put the
more specific small-game/rodent entry **before** the generic `animal` entry so
hares/rats match it first.

```go
var corpseSalvageTable = []corpseSalvageEntry{
    {Group: "rodent", Returns: []items.SalvageReturn{ // small game → hare meat
        {ItemTag: "wild-hare-meat", Quantity: 1},
        {ItemTag: "leather-strip", Quantity: 1},
    }},
    {Group: "animal", Returns: []items.SalvageReturn{ // generic game → raw meat
        {ItemTag: "raw-meat", Quantity: 1},
        {ItemTag: "leather-strip", Quantity: 2},
        {ItemTag: "sinew", Quantity: 1},
    }},
    {Group: "humanoid", Returns: []items.SalvageReturn{ // unchanged
        {ItemTag: "cloth-strip", Quantity: 2},
        {ItemTag: "leather-strip", Quantity: 1},
    }},
}
```

- The exact `Group` key for small game is pinned at implementation: confirm the
  prey mobs' `groups` (wild hare 360 = `[animal, rodent, prey]`; marsh rat 367;
  the hare-meat carriers). Use whichever specific tag (`rodent`/`small-game`)
  those prey share and the generic `animal` lacks. Verify by reading the prey
  mob YAMLs. Marsh rat (367) should map to `raw-meat` (generic), not hare meat —
  if its groups would match the small-game entry, give rats raw-meat instead
  (adjust the tag or rat's groups).
- Item tags (`raw-meat`, `wild-hare-meat`, `leather-strip`, `sinew`) must match
  existing `component_tag`s (they do: 40014/40064/40002/40068).
- Update `internal/crafting/corpse_salvage_test.go` for the new yields.

### Component 2 — wild-hare-meat re-bucket (`internal/economy/buckets.go`)

Change `40064: "fernway"` → `40064: "overlap"`. This lets Tova (overlap) and
Halix (overlap) deliver salvaged hare meat to the cooks. Confirm nothing else
keys off wild-hare-meat being `fernway` (caravan cargo, forager throughput
metrics) — grep for `40064` / `wild-hare-meat` and adjust if needed.

### Component 3 — Cooks become crafters (mob YAML)

For each of **336** (Stillwater), **103**, **248** (Thornwall), edit the mob YAML
to add (mirroring Ilsa 338):

```yaml
crafter: true
crafterskill: cooking
crafterrecipeids:
  - <the cooking recipes this cook should produce, matching its cooked-meal stock>
crafterrestockmaterials:
  - <ingredient item ids the cook needs in stock to craft, e.g. 40014 raw-meat,
     40017 salt-pouch, 40015 wild-vegetables, 40016 water-flask, …>
```

- `crafterrecipeids` per cook = the cooking recipes whose outputs are in that
  cook's current shop list (e.g. 103 makes grilled-meat 30022, hearty-stew 30023,
  herbal-tea 30024, trail-rations 30021; 336/248 per their stock). Read each
  cook's shop file + mob YAML to enumerate.
- Preserve each cook's existing `behavior_archetype`, groups, idle commands,
  description, gold, etc. — additive change only.
- `crafterskill: cooking` — confirm a `cooking` skill exists and the crafter
  decision path accepts it (Ilsa uses `alchemy`; verify cooking is a valid
  crafter skill string).

### Component 4 — Cooked-meal stock flips to crafter-produced (shop files)

In each cook's shop file (`_datafiles/world/dogmud/shops/{zone}/{mobid}-room{room}.yaml`):
- Set the **cooked-meal** stock entries (30021–30024 and any other meal outputs)
  to `RestockQty: 0` (crafter-produced; cart no longer band-aids them).
- Leave **raw-ingredient** entries (raw-meat 40014, salt 40017, veg 40015, water
  40016, clam 40058, etc.) at their existing `RestockQty: 5` baseline — these are
  the cook's crafting feedstock and the supply floor.
- **SOP caveat:** shop files are persistent living-economy state, NOT instance
  saves — they are **not** wiped by the smoke-test instance cleanup. Editing them
  directly changes live state; the boot/smoke must confirm the crafter actually
  produces meals so the `RestockQty: 0` meals don't sit empty.

## Data flow

```
forager idle → hunt prey (PreyWhitelist) → kill → corpse in territory
  → try_salvage (corpse mode) → LookupCorpseSalvage(prey.groups)
      → rodent/small-game → wild-hare-meat ; animal → raw-meat   (into satchel)
  → forager travels delivery route → SellToVendor at cook's room
      → raw-meat (base) / wild-hare-meat (overlap) ∈ forager Buckets
        AND cook stocks the item → transfer into cook's ingredient stock (no gold)
cook idle tick → TickMobCraft → restock materials → EvaluateCraftOptions
  → ingredients present + meal < MaxStock + profitable → craft
      → consume ingredients, AddStockAtRound(meal)   (meal stock refills)
player → buy meal from cook
```

## Testing

- **Unit:** `corpse_salvage_test.go` — `LookupCorpseSalvage` returns raw-meat for
  a generic `animal` group, wild-hare-meat for the small-game/rodent group, and
  the small-game entry wins over `animal` when a mob has both (order). Humanoid
  unchanged. `buckets_test` (if present) — 40064 now `overlap`.
- **Boot smoke (clean instances):** server boots past data-file load without
  panic; the three cooks load as crafters (no crafter-config validation panic);
  cooking recipe ids resolve.
- **In-game smoke (deferred to user, per precedent):** confirm a forager salvages
  a prey kill and gains raw-meat / hare-meat; confirm it delivers meat to a cook
  (cook's raw-meat stock rises after a forager visit); confirm the cook crafts a
  meal (cooked-meal stock rises from 0 over time); confirm players can buy the
  meal. Watch that `RestockQty: 0` meals don't stay empty (ingredient supply
  keeps up). Sanity-check new player cooking grind isn't more expensive (meat
  value unchanged).

## File touch list (anticipated; finalized in the plan)

- **Modify:** `internal/crafting/corpse_salvage.go` (+ test) — meat yields
- **Modify:** `internal/economy/buckets.go` (+ test if present) — 40064 → overlap
- **Modify:** `_datafiles/world/dogmud/mobs/stillwater/336-*.yaml`,
  `_datafiles/world/dogmud/mobs/thornwall_city/103-*.yaml`,
  `.../248-*.yaml` — crafter fields
- **Modify:** the three cooks' shop files under
  `_datafiles/world/dogmud/shops/{stillwater,thornwall_city}/` — cooked-meal
  `RestockQty: 0`
- **Maybe modify:** marsh rat (367) groups / salvage mapping (content cleanup)
- **Context docs:** `internal/crafting/context.md` (corpse salvage yields),
  forager/economy context as needed
- **Roadmap/memory:** note the cooking half done; log the deferred fernway
  caravan-supply gap

## Out of scope (explicit)

- **Combat-looter salvage + sell** (the deferred sell-surplus goal) — dropped
  (decision 1); revisit with a higher-value surplus source.
- **Fernway-bucket forageables (shadowcap, blood-moss) reaching cooks** — needs
  the Kessa→caravan mesh + the empty Fernway `deliveries_by_tier`; deferred
  (decision 5). They stay cart-supplied. Logged as a caravan-supply follow-up.
- **Enchanting supply (decayed potions → enchanter)** — the second half of the
  economy-fix split; separate chunk.
- **General-store restock** — also flagged in [[project_store_restock_considered_fix]];
  not this chunk.
- **Spoiled-potion item, new trade-goods, value rebalancing** — none needed here.
