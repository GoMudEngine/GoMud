# New Plymouth — Economy Depth Pass (design)

**Status:** approved 2026-06-24 (brainstorming). First improvement pass from the
2026-06-24 capital playtest synthesis (`tools/playtest/reports/2026-06-24-capital-playtest-synthesis.md`),
**economy axis**. Adds vendor coverage + buyable flavor goods to the two thinnest
districts (Common, Temple) and converts a dead flavor stall into a real micro-vendor.

> **Context:** the capital is built (7/7), bug-clean, and prod-ready (push HELD by
> user pending enrichment). This pass is enrichment, not a blocker. The playtest
> found the economy *engine* healthy (ShopBuyRatio 0.50, dynamic pricing, buy/sell
> verified live) but vendor *distribution* thin: Common had 2 cooking-only vendors,
> Temple 0, and the Flower Seller sold nothing.

## 0. Scope (locked with user 2026-06-24)

IN: **3 vendors** + **~6 new flavor items** + their dialogue/faction/schedule wiring.
1. A **Common general-goods vendor** (new mob) in The Common Market (5604).
2. A **Temple offerings vendor** (new mob) at the Temple Gate Plaza (5903).
3. A **Flower micro-vendor** — convert the existing Flower Seller (mob 9326) into a
   real vendor.

Items policy: **reuse existing item IDs + add a few new flavor items** only where no
suitable item exists.

OUT (deferred):
- **A money-sink *mechanic*** (donations/blessings/cosmetics) — deferred per user.
- **Legendary BIS craft-items** (user's future idea, captured as
  [[project-legendary-bis-craft-items]]): pinnacle best-in-slot crafted gear gated by
  quest elements + high levels in *multiple* crafting skills + very rare drops from
  **new instance zones**. This is the real long-term gold/aspiration sink. Its own
  spec→plan→build later; NOT this pass.
- Vendors for the other districts (Crafting/Merchant/Docks already have 4–6 each;
  Noble 1 is appropriate; Old Quarter 0 by design).

## 1. IDs & locations

- **New mobs:** 9390 (Common general vendor), 9391 (Temple offerings vendor).
  (Next-free mob ID is 9390.)
- **Edited mob:** 9326 (Flower Seller, `mobs/new_plymouth_common/9326-flower_seller.yaml`)
  — add `craft_support: general` + a `shop:` block (it is already
  `groups: [humanoid, np_commonfolk]`, no shop today).
- **New items:** IDs **40102+** (next-free item ID is 40102). ~6 items (final count
  set in §3). Each lives in its type-folder per the item schema
  (`items/{type}-{range}/{subtype?}/{itemid}-{ConvertForFilename(name)}.yaml`) —
  VERIFY the folder from the schema at build (a path/`Filepath()` mismatch panics).
- **Reuse item IDs** (VERIFY each exists + its type at build by glob `*/{id}-*.yaml`):
  Temple Incense **27**, Folk Charm **20089**, Sack of Flour **40076**, Tallow Candle
  **40077**.
- **Vendor rooms (existing, no new rooms):** Common Market **5604**; Temple Gate
  Plaza **5903**; Flower Seller stays at its current spawn (Flower Market 5603 /
  Carter's Rise per its existing `spawninfo`).
- **Dialogue:** by mobid (`dialogue/new_plymouth_common/9390.yaml`,
  `dialogue/new_plymouth_temple/9391.yaml`; the Flower Seller gets a light shop
  dialogue at `dialogue/new_plymouth_common/9326.yaml` if none exists).
- **Schedules:** light day-stall schedules at `schedules/new_plymouth_common/` and
  `schedules/new_plymouth_temple/` (validators panic on gaps/unreachable).

## 2. The three vendors (life-sheets-lite)

Each new vendor follows the established NP resident pattern but lighter than a full
anchor: a **mutation** woven into the description, **faction via `groups:`**,
`craft_support: general`, a `shop:` block, **≥2 discoverable first-person dialogue
topics**, a **spawn**, and a **light 2-segment schedule** (open the stall by day,
sleep/closed at night — pattern from `schedules/new_plymouth_docks/np_docks_marn.yaml`).
All three are ordinary working folk — `non_combatant: true` is appropriate (vendors;
matches the other NP shopkeepers — verify against an existing NP shopkeeper mob, e.g.
`9333-master_halvard.yaml`, for the exact field set incl. `behavior_archetype:
noncombat_shopkeeper`).

| Mob | Name (placeholder — build picks a canon-clean name, collision-checked) | Room | Mutation (NP convention) | `groups` | Stocks |
|-----|------|------|--------------------------|----------|--------|
| 9390 | a Common general-goods seller | 5604 (Common Market) | a mild, mundane mutation (e.g. weather-reading joints, an unfailing memory for faces/prices) | humanoid, np_commonfolk | new general goods + Tallow Candle 40077 + Sack of Flour 40076 |
| 9391 | a Temple offerings seller | 5903 (Temple Gate Plaza) | a quiet, devotional-flavored mutation (e.g. hands that never tire, eyes that catch the altar-light) | humanoid, temple_np | Temple Incense 27 + Tallow Candle 40077 + Folk Charm 20089 + 1 new votive item |
| 9326 (edit) | Flower Seller (existing) | its current spawn | (keep existing description; add nothing if it lacks one, or a light touch) | humanoid, np_commonfolk (unchanged) | new cut-flower item(s) |

**Dialogue intents (first-person NPC `text`; narrator `hints`; discoverable
triggers; NO quests, NO `grantsQuest`):**
- **General seller (9390):** (1) the wares / what they stock; (2) the Common /
  everyday life of the quarter; (3) (optional) supply — where the goods come from
  (ties to the living economy without quest mechanics).
- **Offerings seller (9391):** (1) the offerings / what to bring to the altar
  (incense, a candle, a charm); (2) the temple / the gate; (3) (optional) the
  devotional why.
- **Flower Seller (9326):** a light shop voice (the blooms, the morning sell-out —
  canon: "they sell out every morning"). Keep it short; this is a micro-vendor.

## 3. New flavor items (~6 — IDs 40102+)

All: correct `type`/`subtype` per the item schema, modest `value`, in the right
type-folder, `ConvertForFilename` filename, **no `: ` (colon-space) in any value**.
General goods are plain `type: object` utility items (verify the exact object subtype
against an existing simple object item, e.g. the Tallow Candle 40077 or Sack of Flour
40076). Final list (build may merge/adjust within the spirit; keep it ~6):

**Common general goods (≈4):**
1. **Coil of Rope** — everyday utility object, low value.
2. **Waterskin** — everyday utility object, low value.
3. **Cake of Tallow Soap** — everyday utility object, low value.
4. **Tinder-and-Flint** (fire-starting kit) — everyday utility object, low value.

**Temple offerings (≈1):**
5. **Votive Candle** (or a Prayer-Token) — a devotional offering object, low value;
   a flavor money-sink at the gate (distinct from the general Tallow Candle 40077 —
   this one reads as a temple offering).

**Flower micro-vendor (≈1):**
6. **Cut Flowers** (a posy / a bunch of blooms) — a cheap flavor object, very low
   value. (Optionally 2 variants if trivial; default 1 to honor "a few new items".)

> If any of these duplicate an existing item closely enough to reuse, reuse it
> instead and drop the new one — keep new items to what genuinely doesn't exist.

## 4. Conventions (faction / dialogue / schedule / shop)

- **Faction** via `groups:` referencing EXISTING factions only (np_commonfolk,
  temple_np — both exist). No new faction.
- **Shop block** = the mob-template `shop:` list of `- itemid: N` entries (see
  `9333-master_halvard.yaml`); `RegisterMobShop` seeds the living-economy shop state
  from it. Every shop-bearing mob needs a valid `craft_support:` tag (`general` here)
  or `ValidateShopMobTags` PANICS.
- **Dynamic pricing** is automatic (the engine prices from `value` + restock +
  scarcity). Set sane `value`s; no per-shop price config needed. Because these are
  NOT in `CaravanServedZones`, the ticker restocks them normally (no supply-runner
  manifest dependency — these vendors self-restock).
- **Schedules:** light 2-segment (day at stall / night closed-or-sleeping); all
  `target_room`s within the vendor's own district, pathto-reachable. Validators panic
  on gaps/unreachable.
- **Dialogue SOPs:** NPC `text` first-person; `hints` narrator 2nd-person; triggers
  discoverable; no prefix-shadowing; no quest fields.

## 5. Build staging (feeds ONE plan; each stage boot-verified)

> Pre-smoke ritual: wipe `mobs.instances/*` + `rooms.instances/*` (NOT `shops/`);
> boot-poll `ERROR:.*PANIC`/`fatal error:`, not bare "panic" (gotcha #8).

- **Stage A — new items (40102+):** author the ~6 flavor items; boot-verify
  (`items.LoadDataFiles` count rises; no Filepath panic).
- **Stage B — the two new vendors (9390, 9391) + dialogue + shop + craft_support +
  groups + spawn:** author mobs; boot-verify (`mobs.LoadDataFiles` +2;
  `ValidateShopMobTags` passes; dialogue loads).
- **Stage C — convert the Flower Seller (9326):** add `craft_support: general` + the
  `shop:` block + light dialogue; boot-verify.
- **Stage D — light schedules** for 9390 + 9391 (+ optionally 9326 if it lacks one);
  add `schedule_id:`; boot-verify (`LoadSchedules` rises; no gap/unreachable panic).
- **Stage E — smoke test:** boot, walk to each vendor (or confirm via the living
  shop-state files `shops/new_plymouth_common|temple/...`), `list` + `buy` + `sell`
  one item per vendor; confirm dynamic pricing + the buy/sell loop. Then commit per
  stage and merge `--no-ff` onto master (push still HELD).

## 6. Definition of done

- 3 vendors live: Common Market (5604) general seller, Temple Gate Plaza (5903)
  offerings seller, Flower Seller (9326) now sells cut flowers.
- ~6 new flavor items load (correct type-folders, no Filepath/colon-space panic).
- Each new vendor: mutation in description, `groups:` faction, `craft_support:
  general`, `shop:` block, ≥2 discoverable dialogue topics, spawn, light schedule.
- `ValidateShopMobTags` + schedule validators pass; boot errors=0; `cartcheck`
  unaffected (no room changes).
- Smoke test: `list`/`buy`/`sell` works at each new vendor; dynamic pricing applies.
- Merge `--no-ff` to master; **push still HELD** (per user policy until the
  enrichment phase is further along — or user decides to push).

## 7. Honored gotchas

#1 YAML `": "` in a value → `" — "` · #2 faction refs (membership only — existing) ·
#3 Title-Case mob names (`AssertCanonical`) · #4 faction via `groups:` · #6 shop mobs
need `craft_support:` · #8 boot-poll `ERROR:.*PANIC` · item filenames =
`ConvertForFilename` in the correct type-folder (verify `Filepath()`) · only verified
reuse item IDs · names collision-checked vs the world mob roster · dialogue
first-person/discoverable, no quest fields.
