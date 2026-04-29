# Material Region Split — Design (Stage 3.0b of Caravan/Economy Effort)

**Date:** 2026-04-28
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Audit the 61 existing raw materials in the codebase, invent ~6 new
forest/herbal mats for The Fernway region, classify all mats into
regional supply buckets, update ~12-15 mid/high-tier recipes to wire
demand for the new mats, and reshape the 17 caravan-served vendor
inventories so each region has a distinct supply profile. The audit
matrix becomes the durable reference document that Stages 3.0c, 3.1,
3.0e, and 3.4 all consume.

## Multi-stage context

This spec is **Stage 3.0b** of the multi-stage caravan/economy effort.
Stages 1 (NPC parties) and 2 (basic caravan) shipped 2026-04-27 — both
sit unmerged on `development`. Stage 3 itself decomposes into:

- **3.0b** (THIS SPEC) — region-split mat audit + vendor reassignment + recipe demand wiring
- **3.0a** — new wilderness zone west of Stillwater (~30-50 rooms, wildlife, foragables)
- **3.0c** — south expansion of The Fernway (forager territory)
- **3.0d** — fold-anchor / fold-recall as NPC-castable spells (combat retreat, NOT supply pipeline)
- **3.0e** — cloth/leather/cord/sinew via corpse salvage (extends salvage system)
- **3.1** — forager NPCs (wire `forage` for NPCs + routines + caravan slowdown + caravan route extension to include The Fernway)

Per user direction, nothing ships to prod (`master`) until the entire
economy stack (Stages 3.0b through 3.4) lands on `development`.

## Worldbuilding

The world has three foraging regions, each with distinct material
identity:

| Region | Theme | Geographic anchor |
|---|---|---|
| **Stillwater** | Lake / marsh / fishing / fine craft | Stillwater town + adjacent + new west zone (3.0a) |
| **Thornwall** | Chrysalis / enchanting / refined metals / urban craft | Thornwall City + Outskirts (in-shop crafted, not foraged) |
| **The Fernway** | Forest / herbal / wild-game / alchemy | Existing Fernway zone + south expansion (3.0c) |

The Fernway forager stays in-zone; the caravan stops at The Fernway as
part of its route, picks up forest mats, and distributes them to BOTH
towns symmetrically. The Stillwater↔Thornwall flow remains asymmetric
(lake mats south, chrysalis/refined mats north), with Fernway mats as
the third axis.

**Bandit-blocks-route justification** (from Stages 1+2): the bandit
pack on North Road (room 4052) prevents lone foragers from making the
Stillwater↔Thornwall trip with bulk goods. The party-strong caravan is
the only safe way mats cross between towns. Foragers personally use
fold-recall (Stage 3.0d) for combat retreat but never haul cargo
through the bandits.

## The audit matrix

Every existing material (61) and new mat (6) gets classified into
region buckets, native sources, and vendor slot assignments. The matrix
is the durable artifact this spec produces. Stages 3.0c, 3.1, 3.0e,
and 3.4 all consume it.

**Bucket definitions:**

| Bucket | Definition | Stocked at... |
|---|---|---|
| **Base** | Universal crafting feedstock with no regional flavor | Every region's appropriate vendor |
| **Stillwater unique** | Lake/marsh/fishing-themed | Stillwater vendors (forager-fed); Thornwall vendors (caravan-fed) |
| **Thornwall unique** | Chrysalis/enchanting/refined-metal | Thornwall vendors (in-shop crafted); Stillwater vendors (caravan-fed) |
| **Fernway unique** | Forest/herbal/wild-game | Both town vendors (caravan-fed from Fernway forager) |
| **Mid-tier overlap** | Mats that fit two of three regions | Per pairing |
| **Defer to 3.0e** | Cloth/leather/cord/sinew — implementation deferred but classified now | Skip vendor wiring in 3.0b; classification remains for 3.0e to use |

**Stillwater unique (existing):** lake-iron nodule (40059), Stillwater
black pearl (40053), freshwater clam (40058), cattail down, marsh-willow
bark, lake mint, hunter-eel hide. (~7-10 mats; exact list confirmed
during implementation.)

**Thornwall unique (existing):** Chrysalis Core (40010), chrysalis
shard (40027), binding paste (40028), mutation catalyst (40029),
chrysalis setting (40030), Hive Fragment (40011), steel ingot (40018),
copper wire (40021), silver wire (40022), gold wire (40023), polished
stone (40024), raw gem (40025), gem dust (40026). (~10-13 mats.)

**Fernway unique (NEW — this spec invents):**

| # | Mat | Theme | Recipe homes (2-3 across schools) |
|---|---|---|---|
| 1 | **oak bark** | tannins, ward paint | alchemy (tannin tonic — astringent) + enchanting (binding wash for inscriptions) |
| 2 | **shadowcap mushroom** | dim-forest fungus | cooking (mushroom-and-game stew variant) + alchemy (vision tincture) |
| 3 | **wild hare meat** | game protein | cooking (game stew — better than raw-meat stew) + alchemy (rendered fat for salves) |
| 4 | **beeswax** | sealant/wax | alchemy (potion-bottle sealant) + tailoring (waterproof treatment — *recipe wiring deferred until 3.0e cloth lands*) |
| 5 | **blood-moss** | clotting agent | alchemy (clotting salve) + cooking (savory binder for thick stews) |
| 6 | **pine pitch** | resin/glue | alchemy (sticky base for compound potions) + blacksmithing (rust-prevention coat on tools) |

**Base mats (universal):** iron ingot (40001), wooden plank (40003),
glass vial (40006), water flask (40016), salt pouch (40017), thread
spool (40012), bone needle (40013), raw meat (40014), wild vegetables
(40015). (~9 mats.)

**Mid-tier overlap (existing):** dustwalk herb (40009 — Dustwalk Road,
overlap with Fernway alchemy), healer's root (40004 — common alchemy,
overlap likely), bitter thistle (40005), spore sac (40008 — Labyrinth/
Endless Trashheap, may overlap with Fernway), chain link (40019 —
universal), coal dust (40020 — Thornwall-leaning but appears in
multiple recipes). (~6-8 mats; exact assignment during implementation.)

**Cloth / leather / cord / sinew (defer wiring to 3.0e, classify now):**

| Mat | ID | Region designation (from 3.0b classification) | Notes |
|---|---|---|---|
| leather strip | 40002 | Mid-tier overlap (animal hide is universal) | 3.0e wires drop-from-corpse |
| cloth strip | 40007 | Mid-tier overlap | 3.0e decides drop source vs forager-gathered fiber |
| (any others surface during audit) | — | — | — |

The full matrix gets fleshed out as part of Task 0 in the implementation
plan — the implementer reads each of the 61 existing mat YAMLs and
slots them into the table above. The spec lists the buckets and seed
classifications; the matrix doc is built during implementation.

## Vendor inventory model

**Same-craft vendor pairs across the two towns:**

| Stillwater vendor | Thornwall vendor | Caravan transfers (post-3.4) |
|---|---|---|
| Smith Brindle (337) | Blacksmith Kerra (97) | lake-iron, pine pitch ↔ steel ingot, refined wires |
| Apothecary Ilsa (338) | Apothecary Voss (98) | lake mint, marsh-willow ↔ chrysalis catalyst, mutation catalyst (plus Fernway mats — oak bark, shadowcap, blood-moss, beeswax — flow to BOTH from caravan picking up at Fernway) |
| Pearl-carver Kess (340) | Jeweler Tess (108) | stillwater pearl ↔ chrysalis setting, gem dust, polished stone |
| Weaver Edda (339) | Weaver Maren (113) | cattail down ↔ (chrysalis-cloth?) [3.0e to confirm] |
| Storekeeper Wulf (341) | Food Vendor (103) / Fence Dealer Siv (104) | general goods (audit during implementation) |
| Innkeeper Sigrid (333) | Tavern cook Brynn (248) | freshwater clam, hunter-eel ↔ Thornwall meat |
| Miller Bram (348) | (no Thornwall miller) | grain stays Stillwater-local |
| Fishmonger Tov Brann (336) | Food vendor / Tavern cook | clam, eel ↔ Thornwall meat |

**Each vendor's inventory slots = both halves of their pair + Fernway
slots (where craft-appropriate) + base mats.**

For example, Apothecary Ilsa (Stillwater) stocks slots for:
- Native (Stillwater forager-fed): lake mint, marsh-willow bark
- Caravan-fed (from Voss): chrysalis catalyst, mutation catalyst
- Caravan-fed (from Fernway forager, picked up by caravan in-zone): oak bark, shadowcap mushroom, blood-moss, beeswax
- Base: water flask, glass vial

Apothecary Voss (Thornwall) mirrors:
- Native (in-shop crafted): chrysalis catalyst, mutation catalyst
- Caravan-fed (from Ilsa): lake mint, marsh-willow bark
- Caravan-fed (from Fernway forager, picked up by caravan in-zone): oak bark, shadowcap mushroom, blood-moss, beeswax
- Base: water flask, glass vial

Both apothecaries end up with the same total slot count and identical
mat types — just different supply pipelines fill them. The Fernway
forager meets the caravan in The Fernway zone (rather than returning
to either town), so Fernway mats reach both towns symmetrically via
the caravan. **This same pattern applies to every vendor pair.**

**Vendor inventory edits:**

1. Add Fernway mat slots to all alchemy/cooking/blacksmith/jeweler/enchanter vendors as appropriate
2. Add cross-region mat slots so each vendor pair has mirrored inventories
3. Drop misplaced slots (e.g., chrysalis stuff currently on Stillwater vendors moves to Thornwall vendors only)
4. Drop cloth/leather slots (3.0e will reorganize them)
5. Set RestockQty/MaxStock to seed values (1-3) — real supply comes from foragers + caravan post-3.1/3.4

## Recipe demand wiring

**Coverage target:** each new Fernway mat appears in 2-3 mid/high-tier
recipes spanning at least 2 craft schools. ~12-15 recipe edits total.

**Per the new-mat table above:**

- **oak bark** → 1 alchemy recipe + 1 enchanting recipe
- **shadowcap mushroom** → 1 cooking recipe + 1 alchemy recipe
- **wild hare meat** → 1 cooking recipe + 1 alchemy recipe
- **beeswax** → 1 alchemy recipe (+1 tailoring deferred to 3.0e)
- **blood-moss** → 1 alchemy recipe + 1 cooking recipe
- **pine pitch** → 1 alchemy recipe + 1 blacksmithing recipe

That's ~12 recipe edits. Implementation may bump 1-2 mats up to a third
recipe if a clean home exists (durability spread).

**Recipe edit pattern:** find an existing mid/high-tier recipe whose
ingredient list could plausibly include the new mat (e.g., a clotting
salve recipe currently using only generic herbs — add blood-moss as a
required ingredient), and swap or extend the ingredient slot. Don't
invent new recipes — work within the existing 97-recipe corpus.

**Recipes touching cloth/leather/cord:** skip in 3.0b. The existing
recipes using leather strip / cloth strip stay unchanged for now;
3.0e will revisit when corpse salvage lands.

## Files affected

| Action | File | Purpose |
|---|---|---|
| CREATE | `_datafiles/world/dogmud/items/materials-40000/{N}-oak_bark.yaml` | New Fernway alchemy/enchanting mat |
| CREATE | `..../{N+1}-shadowcap_mushroom.yaml` | New Fernway cooking/alchemy mat |
| CREATE | `..../{N+2}-wild_hare_meat.yaml` | New Fernway cooking primary |
| CREATE | `..../{N+3}-beeswax.yaml` | New Fernway alchemy + tailoring (tailoring deferred) |
| CREATE | `..../{N+4}-blood_moss.yaml` | New Fernway alchemy/cooking |
| CREATE | `..../{N+5}-pine_pitch.yaml` | New Fernway alchemy/blacksmithing |
| MODIFY | `_datafiles/world/dogmud/recipes/alchemy/*.yaml` (~6-8 edits) | Wire new mats as ingredients |
| MODIFY | `_datafiles/world/dogmud/recipes/cooking/*.yaml` (~3-4 edits) | Wire wild hare, shadowcap, blood-moss |
| MODIFY | `_datafiles/world/dogmud/recipes/enchanting/*.yaml` (~1-2 edits) | Wire oak bark |
| MODIFY | `_datafiles/world/dogmud/recipes/blacksmithing/*.yaml` (~1-2 edits) | Wire pine pitch |
| MODIFY | `_datafiles/world/dogmud/mobs/stillwater/*.yaml` (8 vendors) | Inventory slot setup per pair pattern |
| MODIFY | `_datafiles/world/dogmud/mobs/thornwall_city/*.yaml` (9 vendors) | Mirrored inventory slot setup |
| CREATE | `docs/economy/mat-audit-matrix.md` (or similar location) | The durable audit matrix doc; consumed by 3.0c, 3.1, 3.0e, 3.4 |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0b dev-only entry (note: economy stack still unmerged) |

Item ID range for new mats: scout for next free range above 40060
during plan task 0. Likely 40061-40066 unless a gap exists.

## Verification

**Phase 1 — boot test:**
- Server boots without panic. New mat YAMLs parse cleanly via
  `mobs.LoadDataFiles()` / item load path.

**Phase 2 — recipe-references-mats audit:**
- Every new Fernway mat appears in at least 2 recipes across at least
  2 craft schools (per the design table).
- Every recipe ingredient still resolves to an existing mat ID (no
  typos introduced by the recipe edits).

**Phase 3 — vendor inventory smoke test (manual, in-game):**
- Visit each of the 17 caravan-served vendors; confirm inventory shows
  the expected slot mix (Stillwater unique + Fernway + cross-region +
  base, per the pair pattern).
- Confirm misplaced slots are gone (e.g., no chrysalis core at
  Stillwater Smith Brindle).
- Confirm cloth/leather slots are unchanged from pre-3.0b state (3.0e
  will revisit).

**Phase 4 — backward compat smoke test:**
- Existing recipes still craft successfully (no broken ingredient
  references from the recipe edits).
- Non-served-zone vendors (Sanctum Basin, Watchers Crossing, etc.)
  unchanged.

## Out of scope (explicitly)

- **Cloth/leather/cord/sinew vendor slots + recipe wiring** — defer to
  Stage 3.0e (corpse salvage). Classification in the audit matrix IS
  in scope; vendor/recipe wiring is not.
- **Caravan route redesign to include The Fernway as a stop** — Stage
  3.1 implementation concern. The vendor inventories in 3.0b assume
  the route will eventually include Fernway, but the actual route
  change ships with foragers.
- **Forager NPC creation** — Stage 3.1.
- **Pricing tuning** — existing scarcity multiplier (`ShopAbundanceThreshold`,
  `ScarcityMultiplier`) handles regional supply pressure automatically.
  No manual price edits in 3.0b.
- **New South Crossroads / Fernway zone build (south expansion)** —
  Stage 3.0c.
- **Non-caravan-served vendors** (Sanctum Basin, Watchers Crossing,
  Ashwick if any) — stay as-is. Regional mats only flow through the
  caravan-served network in v1.
- **Any new crafts or recipe schools** — strictly recipe edits within
  the existing 6 schools.
- **Caravan slowdown tuning** — Stage 3.1 will slow the caravan
  cadence to make foragers matter; 3.0b leaves the cadence at the
  current Stage 2 default (`CaravanDepotDwellRounds: 360`).

## Open implementation questions (for the plan stage)

These are detail-level decisions made during planning, not
brainstorming-level decisions:

- Exact item IDs for the 6 new Fernway mats (scout for next free range
  above 40060).
- Exact recipe edits per craft school — implementation reads each
  recipe and picks the cleanest demand insertion point.
- Final mat-to-vendor pairing for vendors without an obvious
  same-craft counterpart (Storekeeper Wulf ↔ Food Vendor / Fence
  Dealer Siv; Miller Bram has no Thornwall pair).
- Audit matrix doc location — `docs/economy/mat-audit-matrix.md` is
  the working assumption; could land in `docs/superpowers/specs/` if
  preferred for traceability.
- Whether cooking has enough mid/high-tier recipes to absorb 3 of the
  6 new mats (shadowcap, wild hare, blood-moss). If not, the recipe
  count for cooking may shrink and the alchemy count may grow.
