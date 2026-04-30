# Material Region Audit Matrix

> **Purpose:** Durable classification of all caravan-relevant raw
> materials into regional supply buckets. Consumed by Stages 3.0c
> (south expansion zone build), 3.1 (forager NPCs), 3.0e (corpse
> salvage), and 3.4 (real item transfer).
>
> Classification per spec
> `docs/superpowers/specs/2026-04-28-mat-region-split-design.md`.
>
> **Implementation note:** This matrix is mirrored in
> `internal/economy/buckets.go`. Drift between the two is caught at
> test time by `TestBucketMap_AuditMatrixCoverage`.
>
> Audited: 2026-04-27. Auditor: Claude (Stage 3.0b).

## Bucket definitions

- **Stillwater** — Lake/marsh/fishing themed; native at Stillwater
  foragers and local vendors
- **Thornwall** — Chrysalis/refined-metal/jewelcraft themed; in-shop
  crafted at Thornwall workshops
- **Fernway** — Forest/herbal/wild-game themed; foraged in The
  Fernway, distributed by caravan to both towns
- **Base** — Universal crafting feedstock with no regional flavor;
  available everywhere
- **Mid-tier overlap** — Mats that fit two of three regions; available
  at vendors of either region
- **Defer to 3.0e** — Cloth/leather/cord/sinew mats; classification
  done now, vendor wiring deferred until corpse salvage lands
- **Quest/specialty** — Quest items or non-crafting props; not part of
  the supply pipeline

## Audit table

| ID | Name | Bucket | Native source | Notes |
|---|---|---|---|---|
| 40001 | iron ingot | Base | universal | Smithing feedstock |
| 40002 | leather strip | Mid-tier overlap | corpse-salvage sourced | Reclassified by 3.0e: dropped by salvaging animal- and humanoid-group corpses |
| 40003 | wooden plank | Base | universal | |
| 40004 | healer's root | Mid-tier overlap | Thornwall + Fernway alchemy | Currently stocked at Thornwall apothecary only; expand to Stillwater apothecary |
| 40005 | bitter thistle | Mid-tier overlap | Thornwall + Fernway alchemy | |
| 40006 | glass vial | Base | universal | Alchemy infrastructure |
| 40007 | cloth strip | Mid-tier overlap | corpse-salvage sourced | Reclassified by 3.0e: dropped by salvaging humanoid-group corpses |
| 40008 | spore sac | Mid-tier overlap | Fernway + Labyrinth | Chrysalis-cave fauna; alchemy use; not is_component (holdable prop) — flag for 3.0e reclassification if recipe demand added |
| 40009 | dustwalk herb | Mid-tier overlap | Dustwalk Road + Fernway | Grows in dry creek beds along Dustwalk Road; Thornwall proximity |
| 40010 | Chrysalis Core | Thornwall | in-shop (Vael) | |
| 40011 | Hive Fragment | Thornwall | in-shop (Vael) | |
| 40012 | thread spool | Base | universal | is_component: true; generic sewing feedstock — Base, not Defer |
| 40013 | bone needle | Base | universal | is_component: true; generic sewing tool — Base, not Defer |
| 40014 | raw meat | Base | universal | |
| 40015 | wild vegetables | Base | universal | |
| 40016 | water flask | Base | universal | |
| 40017 | salt pouch | Base | universal | |
| 40018 | steel ingot | Thornwall | in-shop (Kerra) | Refined metal tier |
| 40019 | chain link | Base | universal | Jewelcraft + smithing feedstock |
| 40020 | coal dust | Mid-tier overlap | Thornwall-leaning | Smithing flux; Thornwall smith primary, but generic enough for overlap |
| 40021 | copper wire | Thornwall | in-shop (Tess) | Refined metal tier |
| 40022 | silver wire | Thornwall | in-shop (Tess/Kess) | Refined metal tier |
| 40023 | gold wire | Thornwall | in-shop (Tess) | Refined metal tier |
| 40024 | polished stone | Thornwall | in-shop (Tess/Kess) | Jewelcraft tier |
| 40025 | raw gem | Thornwall | in-shop (Tess) | Jewelcraft tier |
| 40026 | gem dust | Thornwall | in-shop (Tess) | Jewelcraft tier |
| 40027 | chrysalis shard | Thornwall | in-shop (Vael) | |
| 40028 | binding paste | Thornwall | in-shop (Vael) | Currently only at Vael |
| 40029 | mutation catalyst | Thornwall | in-shop (Vael) | |
| 40030 | chrysalis setting | Thornwall | in-shop (Tess/Vael) | |
| 40031 | spirit fetish | Quest/specialty | n/a | Steppe-spirit quest item; component_tag set but is_component: false — not a supply-chain mat |
| 40032 | windstone sample | Quest/specialty | n/a | Geomancer specimen; value 0, not is_component — quest/exploration prop |
| 40033 | carved wolf totem | Quest/specialty | n/a | Quest prop; value 0, not is_component |
| 40034 | strongbox key | Quest/specialty | n/a | type: key; out of audit |
| 40035 | tally stick | Quest/specialty | n/a | questtoken: 14-explore; out of audit |
| 40036 | bribe ledger | Quest/specialty | n/a | Quest evidence item; out of audit |
| 40037 | guard captain's commendation | Quest/specialty | n/a | Quest reward document; Thornwall Watch seal; out of audit |
| 40038 | oil lantern | Quest/specialty | n/a | Prop item; no component_tag, no is_component — not a crafting mat |
| 40039 | freight crate | Quest/specialty | n/a | Caravan prop for NPC quest; no is_component — not a supply-chain mat |
| 40040 | forest herbs | Quest/specialty | n/a | subtype: holdable, uses: 0, no is_component — bundle prop for NPC caravan quest; not a crafting mat |
| 40041 | creased letter | Quest/specialty | n/a | Quest/lore item; out of audit |
| 40042 | herbalism recipe page | Quest/specialty | n/a | usable item that trains search skill; not a raw mat |
| 40043 | clay flask | Base | universal | Alchemy bottle tier 1 (fastest aging, 3.0x) |
| 40044 | sealed phial | Base | universal | Alchemy bottle tier 3 (0.5x aging); chrysalis-infused stopper is flavor, not supply gating |
| 40045 | crystalline decanter | Base | universal | Alchemy bottle tier 4 (0.25x aging, slowest) |
| 40046 | moonpetal | Fernway | foraged in Fernway | Pale nocturnal flower; opens under moonlight; mind/nerve alchemy; forest meadow origin fits Fernway forager |
| 40047 | veilbloom petal | Mid-tier overlap | Dustwalk Road + Thornwall | Description: "grows only on wind-scoured steppes" — steppe/Dustwalk origin; Thornwall alchemy demand |
| 40048 | serpent venom sac | Mid-tier overlap | Labyrinth + Fernway caves | Cave/dungeon creature drop; alchemy use; Labyrinth-adjacent and Fernway cave fringe |
| 40049 | ironbark shaving | Fernway | foraged in Fernway | Bark from ironbark tree; forest origin; body-hardening alchemy demand; fits Fernway forager |
| 40050 | putrid residue | Mid-tier overlap | Labyrinth + Thornwall | Organic decay product; volatile alchemy base; dungeon/Labyrinth origin; Thornwall alchemist demand |
| 40051 | skitter-shrimp shell | Stillwater | foraged in Stillwater area | Lake crustacean; Stillwater unique |
| 40052 | drowned-hunter hide | Defer to 3.0e | n/a until salvage | Stillwater-themed creature hide; cloth/leather adjacent — defer vendor wiring to 3.0e |
| 40053 | Stillwater black pearl | Stillwater | foraged in Stillwater area | Lake freshwater pearl; Stillwater unique |
| 40054 | leviathan-tooth trophy | Quest/specialty | n/a | Bounty reward item; constabulary turn-in — not a supply-chain mat |
| 40055 | cattail down | Defer to 3.0e | n/a until salvage | Stillwater marsh fiber; cloth/fiber adjacent — defer vendor wiring to 3.0e |
| 40056 | marsh willow bark | Stillwater | foraged in Stillwater area | Marsh tree bark; Stillwater unique; alchemy demand |
| 40057 | lake mint | Stillwater | foraged in Stillwater area | Freshwater herb; Stillwater unique; cooking + alchemy demand |
| 40058 | freshwater clam | Stillwater | foraged in Stillwater area | Lake bivalve; Stillwater unique; cooking demand |
| 40059 | lake-iron nodule | Stillwater | foraged in Stillwater area | Mineral deposit from lake bed; Stillwater unique; smithing demand |
| 40060 | Elgar's carved kingfisher | Quest/specialty | n/a | Quest item; out of audit |
| 40061 | Elgar's last journal entry | Quest/specialty | n/a | Quest item; out of audit |
| **40062** | **oak bark** | **Fernway** | **foraged in Fernway** | **NEW; alchemy + enchanting demand** |
| **40063** | **shadowcap mushroom** | **Fernway** | **foraged in Fernway** | **NEW; cooking + alchemy demand** |
| **40064** | **wild hare meat** | **Fernway** | **foraged in Fernway** | **NEW; cooking + alchemy demand** |
| **40065** | **beeswax** | **Fernway** | **foraged in Fernway** | **NEW; alchemy demand (tailoring deferred to 3.0e)** |
| **40066** | **blood-moss** | **Fernway** | **foraged in Fernway** | **NEW; alchemy + cooking demand** |
| **40067** | **pine pitch** | **Fernway** | **foraged in Fernway** | **NEW; alchemy + blacksmithing demand** |
| **40068** | **sinew** | **Mid-tier overlap** | **corpse-salvage sourced** | **NEW (Stage 3.0e); dropped by salvaging animal-group corpses; demand wired into tailoring + blacksmithing** |

## Bucket summary

| Bucket | Count | IDs |
|---|---|---|
| Base | 13 | 40001, 40003, 40006, 40012, 40013, 40014, 40015, 40016, 40017, 40019, 40043, 40044, 40045 |
| Stillwater | 6 | 40051, 40053, 40056, 40057, 40058, 40059 |
| Thornwall | 13 | 40010, 40011, 40018, 40021, 40022, 40023, 40024, 40025, 40026, 40027, 40028, 40029, 40030 |
| Fernway | 8 | 40046, 40049, 40062, 40063, 40064, 40065, 40066, 40067 |
| Mid-tier overlap | 11 | 40004, 40005, 40008, 40009, 40020, 40047, 40048, 40050, 40002, 40007, 40068 |
| Defer to 3.0e | 2 | 40052, 40055 |
| Quest/specialty | 15 | 40031, 40032, 40033, 40034, 40035, 40036, 40037, 40038, 40039, 40040, 40041, 40042, 40054, 40060, 40061 |

> Row count: 68 total. 40001-40067 = 67 existing mats; 40068 added by
> Stage 3.0e (corpse salvage). All 68 rows appear in the audit table
> above, including 15 quest/specialty items that are out of the supply
> pipeline.

## Vendor pair pattern

Each Stillwater vendor has a same-craft Thornwall counterpart. Both
stock identical mat slot lists (filled by different supply pipelines).
See spec for the full pair table.

## Caravan flow (post-3.1/3.4)

- **Northbound (Thornwall to Stillwater):** Thornwall-unique
  chrysalis/refined-metal mats
- **Southbound (Stillwater to Thornwall):** Stillwater-unique
  lake/marsh mats
- **Fernway to both towns:** Fernway-unique forest mats picked up by
  caravan in The Fernway zone

Foragers + the 3.4 real item transfer fill these flows; until those
land, vendor seed stock is the only supply.

## Judgment calls & re-audit flags

- **40008 spore sac** — Marked Mid-tier overlap (Fernway + Labyrinth)
  but is NOT is_component (subtype: holdable). If a recipe is added
  that consumes it, update component_tag and is_component then
  re-confirm bucket.
- **40012 thread spool / 40013 bone needle** — Retained as Base rather
  than Defer to 3.0e. Both are is_component: true and used in current
  tailoring/leatherwork recipes as generic feedstock, not
  cloth/leather drops. If they become salvage outputs in 3.0e,
  re-evaluate.
- **40019 chain link** — Listed as Base (universal smithing feedstock)
  despite being used in jewelcraft. Keep Base; it crosses both
  smithing and jewelcraft and has no regional flavor.
- **40020 coal dust** — Mid-tier overlap (Thornwall-leaning). Could be
  Base; kept as overlap because it has no recipe demand outside
  smithing and Thornwall is the smithing hub. Re-evaluate if
  Stillwater gets a forge.
- **40038 oil lantern / 40039 freight crate / 40040 forest herbs** —
  All marked Quest/specialty. None have is_component: true. They are
  props or holdable bundles used in NPC/caravan quest scripting.
  If any gain crafting recipe demand in future stages, update
  is_component and re-classify bucket.
- **40044 sealed phial / 40045 crystalline decanter** — Description
  references chrysalis-infused seals (Thornwall flavor), but kept as
  Base because they are alchemy infrastructure items sold universally.
  If a Thornwall weaver monopoly is desired, reclassify to Thornwall.
- **40046 moonpetal** — Classified Fernway (forest meadow nocturnal
  flower). No explicit "forest" language in description but nocturnal
  wildflower fits Fernway forage far better than steppe or lake.
  Flag if intended as Thornwall greenhouse stock instead.
- **40047 veilbloom petal** — Description explicitly states
  "wind-scoured steppes" — classified Mid-tier overlap
  (Dustwalk + Thornwall), NOT Fernway. If the south expansion zone
  includes steppes, re-evaluate.
- **40050 putrid residue** — Mid-tier overlap (Labyrinth + Thornwall
  alchemy). Value is 1g; clearly a low-tier junk drop. Could also
  fit Base. Kept as overlap because its only use is volatile alchemy
  (Thornwall demand) and dungeon fringe sourcing.
