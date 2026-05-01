# Material Region Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Audit all 67 materials (61 existing + 6 new Fernway forest mats) into regional supply buckets, wire ~12 mid/high-tier recipe edits to demand the new mats, reshape the 17 caravan-served vendor inventories into mirrored same-craft pairs, and produce a durable mat-audit matrix doc that Stages 3.0c, 3.1, 3.0e, and 3.4 all consume.

**Architecture:** Pure content + documentation work. Six new mat YAMLs at IDs 40062-40067. Recipe ingredient additions per the new-mat-to-recipe-home table from the spec. Vendor shop YAML edits per the same-craft pair pattern. No engine code changes.

**Tech Stack:** YAML data files (mob/recipe/material), Go (only for verification: `go build ./...` and the existing test suite).

---

## Decisions locked at plan time (from spec + scout)

**New mat IDs:** 40062 (oak bark), 40063 (shadowcap mushroom), 40064 (wild hare meat), 40065 (beeswax), 40066 (blood-moss), 40067 (pine pitch). Range 40062-40067 confirmed unused.

**Audit matrix doc location:** `docs/economy/mat-audit-matrix.md` (new file).

**Recipe insertion candidates** (chosen during scout — implementer follows these unless reading the recipe shows a structural problem):

| New mat | School A recipe | School B recipe |
|---|---|---|
| oak bark (40062) | alchemy: `fire-resistance-draught.yaml` | enchanting: `rootbind.yaml` |
| shadowcap mushroom (40063) | cooking: `herbal-tea.yaml` | alchemy: `mindshield-elixir.yaml` |
| wild hare meat (40064) | cooking: `hearty-stew.yaml` | alchemy: `essence-of-growth.yaml` |
| beeswax (40065) | alchemy: `clarity-tonic.yaml` | (tailoring deferred to 3.0e) |
| blood-moss (40066) | alchemy: `battle-trance.yaml` | cooking: `stillwater-lake-chowder.yaml` |
| pine pitch (40067) | alchemy: `mutagen-brew.yaml` | blacksmithing: `steel-ingot.yaml` |

**Vendor pair table** (caravan-served, from Stage 2 spec):

| Stillwater | Thornwall | Pair theme |
|---|---|---|
| 337 Smith Brindle | 97 Blacksmith Kerra | metal/blacksmith |
| 338 Apothecary Ilsa | 98 Apothecary Voss | alchemy |
| 340 Pearl-carver Kess | 108 Jeweler Tess (+ 109 Enchanter Vael for chrysalis stuff) | gem/enchant |
| 339 Weaver Edda | 113 Weaver Maren | tailoring (mostly defer to 3.0e) |
| 333 Innkeeper Sigrid + 348 Miller Bram | 248 Tavern Cook Brynn | food/inn |
| 336 Fishmonger Tov Brann | 103 Food Vendor | food (different sub-pair) |
| 341 Storekeeper Wulf | 104 Fence Dealer Siv + 273 Whisper | general goods (loose pair) |

**Material YAML template** (pattern from 40057-lake_mint.yaml):

```yaml
itemid: <id>
name: <name>
namesimple: <single-word version for shop list>
description: <2-4 sentence flavor>
type: object
subtype: mundane
component_tag: <kebab-case unique tag>
weight: <0.05-0.5 typical for forageables>
value: <30-80 typical for mid-tier mats>
is_component: true
```

**Recipe edit pattern:** add a new ingredient entry to the `ingredients:` list. Don't replace existing ingredients unless explicitly noted.

**Vendor shop entry pattern** (matches existing convention in Stillwater/Thornwall vendor YAMLs):

```yaml
- itemid: <id>
  quantity: 0
  quantitymax: 0
  price: <typical mid-tier 5-30g for new mats>
```

`quantity: 0`, `quantitymax: 0` is the engine convention meaning "use sensible defaults" (engine seeds with reasonable initial stock). Per spec, real supply comes from foragers/caravan post-3.1/3.4, so seed values aren't critical.

---

## File structure overview

| Layer | File | Purpose | Task |
|---|---|---|---|
| Audit doc | `docs/economy/mat-audit-matrix.md` | Durable classification for all 67 mats | T1 |
| New mats | `_datafiles/world/dogmud/items/materials-40000/40062-oak_bark.yaml` | Oak bark item | T2 |
| New mats | `..../40063-shadowcap_mushroom.yaml` | Shadowcap mushroom item | T3 |
| New mats | `..../40064-wild_hare_meat.yaml` | Wild hare meat item | T4 |
| New mats | `..../40065-beeswax.yaml` | Beeswax item | T5 |
| New mats | `..../40066-blood_moss.yaml` | Blood-moss item | T6 |
| New mats | `..../40067-pine_pitch.yaml` | Pine pitch item | T7 |
| Recipe edits | (per the demand table above, 11 recipe files) | Wire new mats as ingredients | T2-T7 (one task per mat) |
| Vendor edits | `_datafiles/world/dogmud/mobs/{stillwater,thornwall_city}/*.yaml` (17 vendors total) | Inventory slot setup | T8-T13 (paired by craft) |
| Verification | (no new files) | Boot test + manual checks | T14 |
| Patch notes | `PATCH_NOTES.md` | Stage 3.0b dev-only entry | T15 |

---

### Task 1: Audit matrix document

**Files:**
- Create: `docs/economy/mat-audit-matrix.md`

This task produces the durable classification artifact. Future stages (3.0c, 3.1, 3.0e, 3.4) consume this doc as the source of truth for which mats belong where. The matrix's classifications inform every later task in this plan (which vendor stocks what, etc.).

- [ ] **Step 1: Create the directory if missing.**

```bash
mkdir -p docs/economy
```

- [ ] **Step 2: Read each of the 61 existing mat YAMLs** to gather name + description hints. The full ID/name list is in the scout summary (top of this plan); descriptions are in `_datafiles/world/dogmud/items/materials-40000/{id}-*.yaml`.

For mats whose region isn't immediately obvious, classify by:
- Stillwater unique: lake/marsh/fishing/freshwater themes
- Thornwall unique: chrysalis/refined-metal/jewelcraft/enchanting themes
- Fernway unique: forest/herbal/wild-game themes (the 6 new mats only)
- Mid-tier overlap: generic crafting feedstock that crosses 2 regions
- Base: universal feedstock with no regional flavor (iron ingot, water flask, glass vial, salt pouch, thread spool, bone needle, raw meat, wild vegetables, wooden plank)
- Defer to 3.0e: cloth/leather/cord/sinew (40002 leather strip, 40007 cloth strip, 40012 thread spool*, 40013 bone needle*, 40055 cattail down, 40052 drowned-hunter hide). *thread spool / bone needle may be base instead of defer; flag in matrix.
- Quest/specialty (out of audit): 40031-40037 (spirit fetish, windstone, etc.), 40060-40061 (Elgar's items), 40034 (strongbox key), 40036 (bribe ledger), 40041 (creased letter) — these are quest items, not raw mats; flag and exclude.

- [ ] **Step 3: Write the audit matrix doc** with the structure below. Fill in every row. Include the 6 new Fernway mats at the bottom (they get full classification too).

```markdown
# Material Region Audit Matrix

> **Purpose:** Durable classification of all caravan-relevant raw
> materials into regional supply buckets. Consumed by Stages 3.0c
> (south expansion zone build), 3.1 (forager NPCs), 3.0e (corpse
> salvage), and 3.4 (real item transfer).
>
> Classification per spec `docs/superpowers/specs/2026-04-28-mat-region-split-design.md`.

## Bucket definitions

- **Stillwater** — Lake/marsh/fishing themed; native at Stillwater foragers
- **Thornwall** — Chrysalis/refined-metal/jewelcraft themed; in-shop crafted at Thornwall workshops
- **Fernway** — Forest/herbal/wild-game themed; foraged in The Fernway, distributed by caravan to both towns
- **Base** — Universal crafting feedstock with no regional flavor; available everywhere
- **Mid-tier overlap** — Mats that fit two of three regions; available at vendors of either region
- **Defer to 3.0e** — Cloth/leather/cord/sinew mats; classification done now, vendor wiring deferred until corpse salvage lands
- **Quest/specialty** — Quest items, not raw mats; not part of the supply pipeline

## Audit table

| ID | Name | Bucket | Native source | Notes |
|---|---|---|---|---|
| 40001 | iron ingot | Base | universal | Smithing feedstock |
| 40002 | leather strip | Defer to 3.0e | (TBD by 3.0e) | Currently stocked at multiple vendors; reorganize when corpse salvage lands |
| 40003 | wooden plank | Base | universal | |
| 40004 | healer's root | Mid-tier overlap | Thornwall + Fernway alchemy | Currently stocked at Thornwall apothecary only; expand to Stillwater apothecary |
| 40005 | bitter thistle | Mid-tier overlap | Thornwall + Fernway alchemy | |
| 40006 | glass vial | Base | universal | Alchemy infrastructure |
| 40007 | cloth strip | Defer to 3.0e | (TBD by 3.0e) | |
| 40008 | spore sac | Mid-tier overlap | Fernway + Labyrinth | |
| 40009 | dustwalk herb | Mid-tier overlap | Dustwalk Road + Fernway | |
| 40010 | Chrysalis Core | Thornwall | in-shop (Vael) | |
| 40011 | Hive Fragment | Thornwall | in-shop (Vael) | |
| 40012 | thread spool | Base | universal | (or Defer to 3.0e — flag during audit) |
| 40013 | bone needle | Base | universal | (or Defer to 3.0e — flag during audit) |
| 40014 | raw meat | Base | universal | |
| 40015 | wild vegetables | Base | universal | |
| 40016 | water flask | Base | universal | |
| 40017 | salt pouch | Base | universal | |
| 40018 | steel ingot | Thornwall | in-shop (Kerra) | Refined metal tier |
| 40019 | chain link | Base | universal | Jewelcraft + smithing |
| 40020 | coal dust | Mid-tier overlap | Thornwall-leaning | Smithing fuel; could become Thornwall-unique if smelter mats split further later |
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
| 40031-40037 | (assorted quest items) | Quest/specialty | n/a | Out of audit |
| 40038 | oil lantern | Base | universal | |
| 40039 | freight crate | Base | universal | (caravan use; flag if relevant) |
| 40040 | forest herbs | Fernway | foraged in Fernway | Already aligned |
| 40041 | creased letter | Quest/specialty | n/a | Out of audit |
| 40042 | herbalism recipe page | Quest/specialty | n/a | Out of audit |
| 40043 | clay flask | Base | universal | Alchemy bottle |
| 40044 | sealed phial | Base | universal | Alchemy bottle |
| 40045 | crystalline decanter | Base | universal | Alchemy bottle |
| 40046 | moonpetal | Mid-tier overlap | (audit during impl) | |
| 40047 | veilbloom petal | Mid-tier overlap | (audit during impl) | |
| 40048 | serpent venom sac | Mid-tier overlap | (audit during impl) | |
| 40049 | ironbark shaving | Mid-tier overlap | (audit during impl) | |
| 40050 | putrid residue | Mid-tier overlap | (audit during impl) | |
| 40051 | skitter-shrimp shell | Stillwater | foraged in Stillwater area | Lake-themed |
| 40052 | drowned-hunter hide | Defer to 3.0e | n/a until salvage | Stillwater-themed but cloth/leather adjacent |
| 40053 | Stillwater black pearl | Stillwater | foraged in Stillwater area | |
| 40054 | leviathan-tooth trophy | Stillwater | quest reward (out of audit) | |
| 40055 | cattail down | Defer to 3.0e | n/a until salvage | Stillwater-themed but fiber adjacent |
| 40056 | marsh willow bark | Stillwater | foraged in Stillwater area | |
| 40057 | lake mint | Stillwater | foraged in Stillwater area | |
| 40058 | freshwater clam | Stillwater | foraged in Stillwater area | |
| 40059 | lake-iron nodule | Stillwater | foraged in Stillwater area | |
| 40060 | Elgar's carved kingfisher | Quest/specialty | n/a | Out of audit |
| 40061 | Elgar's last journal entry | Quest/specialty | n/a | Out of audit |
| **40062** | **oak bark** | **Fernway** | **foraged in Fernway** | **NEW; alchemy + enchanting demand** |
| **40063** | **shadowcap mushroom** | **Fernway** | **foraged in Fernway** | **NEW; cooking + alchemy demand** |
| **40064** | **wild hare meat** | **Fernway** | **foraged in Fernway** | **NEW; cooking + alchemy demand** |
| **40065** | **beeswax** | **Fernway** | **foraged in Fernway** | **NEW; alchemy demand (tailoring deferred to 3.0e)** |
| **40066** | **blood-moss** | **Fernway** | **foraged in Fernway** | **NEW; alchemy + cooking demand** |
| **40067** | **pine pitch** | **Fernway** | **foraged in Fernway** | **NEW; alchemy + blacksmithing demand** |

## Vendor pair pattern

Each Stillwater vendor has a same-craft Thornwall counterpart. Both
stock identical mat slot lists (filled by different supply pipelines).
See spec for the full pair table.

## Caravan flow (post-3.1/3.4)

- **Northbound (Thornwall → Stillwater):** Thornwall-unique chrysalis/refined-metal mats
- **Southbound (Stillwater → Thornwall):** Stillwater-unique lake/marsh mats
- **Fernway → both towns:** Fernway-unique forest mats picked up by caravan in The Fernway zone

Foragers + the 3.4 real item transfer fill these flows; until those land, vendor seed stock is the only supply.
```

- [ ] **Step 4: Commit.**

```bash
git add docs/economy/mat-audit-matrix.md
git commit -m "docs(economy): add material region audit matrix (Stage 3.0b)

Durable classification of all 67 caravan-relevant raw materials into
regional supply buckets per spec
docs/superpowers/specs/2026-04-28-mat-region-split-design.md.

This doc is the source of truth that Stages 3.0c (south expansion zone
build), 3.1 (forager NPCs), 3.0e (corpse salvage), and 3.4 (real item
transfer) consume. Mat-to-region assignments determined here drive all
downstream supply-pipeline work.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Oak bark (40062) + recipe wiring

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40062-oak_bark.yaml`
- Modify: `_datafiles/world/dogmud/recipes/alchemy/fire-resistance-draught.yaml`
- Modify: `_datafiles/world/dogmud/recipes/enchanting/rootbind.yaml`

- [ ] **Step 1: Create the oak bark mat YAML.**

Write `_datafiles/world/dogmud/items/materials-40000/40062-oak_bark.yaml`:

```yaml
itemid: 40062
name: oak bark
namesimple: bark
description: A rough strip of bark peeled from a mature oak in
  the Fernway, the inner side dark with tannins. Alchemists steep
  it for astringents that draw out heat and toxins; enchanters
  grind it to a coarse powder and mix it into binding washes
  before painting wards. The smell is dry and faintly bitter,
  with a vegetal sharpness that lingers on the fingers.
type: object
subtype: mundane
component_tag: oak-bark
weight: 0.1
value: 35
is_component: true
```

- [ ] **Step 2: Verify the file loads cleanly.**

Run: `go build ./...` (no output on success).

- [ ] **Step 3: Wire oak bark into Fire Resistance Draught.**

Read `_datafiles/world/dogmud/recipes/alchemy/fire-resistance-draught.yaml`. The `ingredients:` list currently has dustwalk-herb (2x), healers-root (1x), bottle (1x). Add a new ingredient entry for oak-bark:

```yaml
ingredients:
  - item_tag: dustwalk-herb
    quantity: 2
  - item_tag: healers-root
    quantity: 1
  - item_tag: oak-bark
    quantity: 1
  - item_tag: bottle
    quantity: 1
```

- [ ] **Step 4: Wire oak bark into Rootbind.**

Read `_datafiles/world/dogmud/recipes/enchanting/rootbind.yaml`. The `ingredients:` list currently has binding-paste (2x), healers-root (1x). Add oak-bark:

```yaml
ingredients:
  - item_tag: binding-paste
    quantity: 2
  - item_tag: healers-root
    quantity: 1
  - item_tag: oak-bark
    quantity: 1
```

- [ ] **Step 5: Verify both recipes load and the build is clean.**

Run: `go build ./...` (no output on success).

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40062-oak_bark.yaml _datafiles/world/dogmud/recipes/alchemy/fire-resistance-draught.yaml _datafiles/world/dogmud/recipes/enchanting/rootbind.yaml
git commit -m "feat(items,recipes): add oak bark (40062) + wire alchemy + enchanting demand

New Fernway forest mat. Tannin-bearing oak bark for astringent
alchemy + ward-binding enchanting. Wired into Fire Resistance Draught
(alchemy, skill 20) and Rootbind (enchanting, skill 5) per Stage 3.0b
spec demand-coverage table.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Shadowcap mushroom (40063) + recipe wiring

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40063-shadowcap_mushroom.yaml`
- Modify: `_datafiles/world/dogmud/recipes/cooking/herbal-tea.yaml`
- Modify: `_datafiles/world/dogmud/recipes/alchemy/mindshield-elixir.yaml`

- [ ] **Step 1: Create the shadowcap mushroom mat YAML.**

Write `_datafiles/world/dogmud/items/materials-40000/40063-shadowcap_mushroom.yaml`:

```yaml
itemid: 40063
name: shadowcap mushroom
namesimple: shadowcap
description: A dark, broad-capped mushroom that grows in the
  deep shade of the Fernway's older oaks, the gills underneath
  almost black. Cooks chop them into stews for an earthy
  richness that improves the longer it simmers; alchemists dry
  the caps and grind them into tinctures said to sharpen the
  vision in low light. The smell raw is musty and forest-floor.
type: object
subtype: mundane
component_tag: shadowcap
weight: 0.1
value: 40
is_component: true
```

- [ ] **Step 2: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 3: Wire shadowcap into Herbal Tea.**

Read `_datafiles/world/dogmud/recipes/cooking/herbal-tea.yaml`. Add shadowcap to the `ingredients:` list:

```yaml
ingredients:
  - item_tag: healers-root
    quantity: 1
  - item_tag: shadowcap
    quantity: 1
  - item_tag: water-flask
    quantity: 1
```

- [ ] **Step 4: Wire shadowcap into Mindshield Elixir.**

Read `_datafiles/world/dogmud/recipes/alchemy/mindshield-elixir.yaml`. Add shadowcap to the `ingredients:` list:

```yaml
ingredients:
  - item_tag: chrysalis-core
    quantity: 1
  - item_tag: moonpetal
    quantity: 1
  - item_tag: shadowcap
    quantity: 1
  - item_tag: bottle
    quantity: 1
```

- [ ] **Step 5: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40063-shadowcap_mushroom.yaml _datafiles/world/dogmud/recipes/cooking/herbal-tea.yaml _datafiles/world/dogmud/recipes/alchemy/mindshield-elixir.yaml
git commit -m "feat(items,recipes): add shadowcap mushroom (40063) + wire cooking + alchemy demand

New Fernway forest mat. Dark forest fungus for vision-sharpening
tinctures and earthy cooking. Wired into Herbal Tea (cooking) and
Mindshield Elixir (alchemy, skill 10) per Stage 3.0b spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Wild hare meat (40064) + recipe wiring

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40064-wild_hare_meat.yaml`
- Modify: `_datafiles/world/dogmud/recipes/cooking/hearty-stew.yaml`
- Modify: `_datafiles/world/dogmud/recipes/alchemy/essence-of-growth.yaml`

- [ ] **Step 1: Create the wild hare meat mat YAML.**

Write `_datafiles/world/dogmud/items/materials-40000/40064-wild_hare_meat.yaml`:

```yaml
itemid: 40064
name: wild hare meat
namesimple: hare
description: Lean, dark meat from a hare snared in the Fernway
  underbrush, properly bled and dressed. Cooks consider it
  superior to penned rabbit for its depth of flavor; alchemists
  render the fat into a clean carrier base for salves and
  ointments where domesticated lard is too soft. The lean cuts
  toughen if cooked too fast — long simmers reward patience.
type: object
subtype: mundane
component_tag: wild-hare-meat
weight: 0.4
value: 50
is_component: true
```

- [ ] **Step 2: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 3: Wire wild hare into Hearty Stew.**

Read `_datafiles/world/dogmud/recipes/cooking/hearty-stew.yaml`. Add wild-hare-meat as an ingredient (alongside the existing raw-meat — the recipe gets a richer flavor profile):

```yaml
ingredients:
  - item_tag: raw-meat
    quantity: 1
  - item_tag: wild-hare-meat
    quantity: 1
  - item_tag: wild-vegetables
    quantity: 1
  - item_tag: water-flask
    quantity: 1
```

- [ ] **Step 4: Wire wild hare into Essence of Growth.**

Read `_datafiles/world/dogmud/recipes/alchemy/essence-of-growth.yaml`. Add wild-hare-meat to the `ingredients:` list as a rendered-fat carrier base:

```yaml
ingredients:
  - item_tag: moonpetal
    quantity: 2
  - item_tag: chrysalis-core
    quantity: 1
  - item_tag: healers-root
    quantity: 2
  - item_tag: wild-hare-meat
    quantity: 1
  - item_tag: bottle
    quantity: 1
```

- [ ] **Step 5: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40064-wild_hare_meat.yaml _datafiles/world/dogmud/recipes/cooking/hearty-stew.yaml _datafiles/world/dogmud/recipes/alchemy/essence-of-growth.yaml
git commit -m "feat(items,recipes): add wild hare meat (40064) + wire cooking + alchemy demand

New Fernway forest mat. Game protein for richer cooking and rendered-
fat alchemy carriers. Wired into Hearty Stew (cooking) and Essence of
Growth (alchemy, skill 28) per Stage 3.0b spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Beeswax (40065) + recipe wiring

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40065-beeswax.yaml`
- Modify: `_datafiles/world/dogmud/recipes/alchemy/clarity-tonic.yaml`

(Tailoring recipe wiring deferred until 3.0e cloth lands.)

- [ ] **Step 1: Create the beeswax mat YAML.**

Write `_datafiles/world/dogmud/items/materials-40000/40065-beeswax.yaml`:

```yaml
itemid: 40065
name: beeswax
namesimple: wax
description: A pale yellow chunk of comb wax rendered down from
  a Fernway hollow-tree hive, still faintly honey-scented.
  Alchemists use it to seal potion bottles against air and
  spoilage; tailors melt it into cloth weave to shed water.
  Soft enough to thumb but tough enough to hold a seal under
  hard travel.
type: object
subtype: mundane
component_tag: beeswax
weight: 0.1
value: 30
is_component: true
```

- [ ] **Step 2: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 3: Wire beeswax into Clarity Tonic.**

Read `_datafiles/world/dogmud/recipes/alchemy/clarity-tonic.yaml`. Add beeswax as a sealant component:

```yaml
ingredients:
  - item_tag: chrysalis-core
    quantity: 1
  - item_tag: bitter-thistle
    quantity: 1
  - item_tag: beeswax
    quantity: 1
  - item_tag: bottle
    quantity: 1
```

- [ ] **Step 4: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40065-beeswax.yaml _datafiles/world/dogmud/recipes/alchemy/clarity-tonic.yaml
git commit -m "feat(items,recipes): add beeswax (40065) + wire alchemy demand

New Fernway forest mat. Hive wax for potion-bottle sealing and (post-
3.0e) cloth waterproofing. Wired into Clarity Tonic (alchemy, skill 15)
per Stage 3.0b spec. Tailoring recipe wiring deferred until 3.0e
corpse-salvage lands and cloth supply is settled.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Blood-moss (40066) + recipe wiring

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40066-blood_moss.yaml`
- Modify: `_datafiles/world/dogmud/recipes/alchemy/battle-trance.yaml`
- Modify: `_datafiles/world/dogmud/recipes/cooking/stillwater-lake-chowder.yaml`

- [ ] **Step 1: Create the blood-moss mat YAML.**

Write `_datafiles/world/dogmud/items/materials-40000/40066-blood_moss.yaml`:

```yaml
itemid: 40066
name: blood-moss
namesimple: moss
description: A clump of dark red moss that grows on the north
  faces of Fernway boulders, where the air stays damp. Pressed
  to a wound it stops bleeding fast; alchemists fold it into
  combat tonics for the same reason. Cooks who know it use
  small pinches to thicken stews — the moss releases a savory
  weight that holds up under a long simmer. The taste raw is
  flat and earthen, almost like wet stone.
type: object
subtype: mundane
component_tag: blood-moss
weight: 0.05
value: 45
is_component: true
```

- [ ] **Step 2: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 3: Wire blood-moss into Battle Trance.**

Read `_datafiles/world/dogmud/recipes/alchemy/battle-trance.yaml`. Add blood-moss as a clotting agent:

```yaml
ingredients:
  - item_tag: ironbark
    quantity: 1
  - item_tag: chrysalis-core
    quantity: 1
  - item_tag: moonpetal
    quantity: 1
  - item_tag: blood-moss
    quantity: 1
  - item_tag: bottle
    quantity: 1
```

- [ ] **Step 4: Wire blood-moss into Stillwater Lake Chowder.**

Read `_datafiles/world/dogmud/recipes/cooking/stillwater-lake-chowder.yaml`. Add blood-moss as a savory thickener:

```yaml
ingredients:
  - item_tag: freshwater-clam
    quantity: 3
  - item_tag: wild-vegetables
    quantity: 1
  - item_tag: lake-mint
    quantity: 1
  - item_tag: blood-moss
    quantity: 1
  - item_tag: water-flask
    quantity: 1
```

- [ ] **Step 5: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40066-blood_moss.yaml _datafiles/world/dogmud/recipes/alchemy/battle-trance.yaml _datafiles/world/dogmud/recipes/cooking/stillwater-lake-chowder.yaml
git commit -m "feat(items,recipes): add blood-moss (40066) + wire alchemy + cooking demand

New Fernway forest mat. Clotting moss for combat tonics and a savory
thickener for thick stews. Wired into Battle Trance (alchemy, skill 25)
and Stillwater Lake Chowder (cooking, skill 12) per Stage 3.0b spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Pine pitch (40067) + recipe wiring

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40067-pine_pitch.yaml`
- Modify: `_datafiles/world/dogmud/recipes/alchemy/mutagen-brew.yaml`
- Modify: `_datafiles/world/dogmud/recipes/blacksmithing/steel-ingot.yaml`

- [ ] **Step 1: Create the pine pitch mat YAML.**

Write `_datafiles/world/dogmud/items/materials-40000/40067-pine_pitch.yaml`:

```yaml
itemid: 40067
name: pine pitch
namesimple: pitch
description: A black, sticky lump of resin tapped from
  Fernway pines and cooled into a workable mass. Smiths smear
  it on iron to keep rust off through a wet season; alchemists
  use it as a sticky base when compound brews need something
  to hold their layers together. Warm enough to soften in the
  hand, sharp enough to scent a whole shop when worked.
type: object
subtype: mundane
component_tag: pine-pitch
weight: 0.2
value: 35
is_component: true
```

- [ ] **Step 2: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 3: Wire pine pitch into Mutagen Brew.**

Read `_datafiles/world/dogmud/recipes/alchemy/mutagen-brew.yaml`. Add pine-pitch as a sticky compound base:

```yaml
ingredients:
  - item_tag: veilbloom
    quantity: 1
  - item_tag: chrysalis-core
    quantity: 2
  - item_tag: hive-fragment
    quantity: 1
  - item_tag: pine-pitch
    quantity: 1
  - item_tag: bottle
    quantity: 1
```

- [ ] **Step 4: Wire pine pitch into Steel Ingot.**

Read `_datafiles/world/dogmud/recipes/blacksmithing/steel-ingot.yaml`. Add pine-pitch as a rust-prevention coating:

```yaml
ingredients:
  - item_tag: iron-ingot
    quantity: 3
  - item_tag: coal-dust
    quantity: 1
  - item_tag: pine-pitch
    quantity: 1
```

- [ ] **Step 5: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40067-pine_pitch.yaml _datafiles/world/dogmud/recipes/alchemy/mutagen-brew.yaml _datafiles/world/dogmud/recipes/blacksmithing/steel-ingot.yaml
git commit -m "feat(items,recipes): add pine pitch (40067) + wire alchemy + blacksmithing demand

New Fernway forest mat. Pine resin for compound alchemy bases and
rust-prevention smithing coatings. Wired into Mutagen Brew (alchemy,
skill 35) and Steel Ingot (blacksmithing, skill 4) per Stage 3.0b spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Vendor inventory — alchemy pair (Ilsa 338 + Voss 98)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml`

Both apothecaries get the same inventory profile (mirrored slots), per the spec's pair pattern. Ilsa's native source = Stillwater foragers (lake mint, marsh-willow); caravan-fed slots = chrysalis catalysts (from Voss) + Fernway alchemy mats (from caravan-Fernway pickup). Voss mirrors.

- [ ] **Step 1: Read current Ilsa shop entries.**

Run: read `_datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml`. Note current `shop:` block.

Current per scout: `30036, 30037, 30060, 40006, 40043, 40056, 40057`

- [ ] **Step 2: Read current Voss shop entries.**

Run: read `_datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml`. Note current `shop:` block.

Current per scout: `40004, 40005, 40006, 40007, 40009, 40043, 40010`

- [ ] **Step 3: Update Ilsa's `shop:` block to the mirrored apothecary inventory.**

Replace the existing `shop:` block in `_datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml` with:

```yaml
  shop:
    # Stillwater unique (forager-fed)
    - itemid: 40056    # marsh willow bark
      quantity: 0
      quantitymax: 0
      price: 12
    - itemid: 40057    # lake mint
      quantity: 0
      quantitymax: 0
      price: 10
    # Thornwall chrysalis (caravan-fed from Voss)
    - itemid: 40010    # Chrysalis Core
      quantity: 0
      quantitymax: 0
      price: 80
    - itemid: 40029    # mutation catalyst
      quantity: 0
      quantitymax: 0
      price: 60
    # Fernway forest (caravan-fed from Fernway forager pickup)
    - itemid: 40062    # oak bark
      quantity: 0
      quantitymax: 0
      price: 35
    - itemid: 40063    # shadowcap mushroom
      quantity: 0
      quantitymax: 0
      price: 40
    - itemid: 40065    # beeswax
      quantity: 0
      quantitymax: 0
      price: 30
    - itemid: 40066    # blood-moss
      quantity: 0
      quantitymax: 0
      price: 45
    # Mid-tier overlap alchemy
    - itemid: 40004    # healer's root
      quantity: 0
      quantitymax: 0
      price: 5
    - itemid: 40005    # bitter thistle
      quantity: 0
      quantitymax: 0
      price: 5
    # Base alchemy infrastructure
    - itemid: 40006    # glass vial
      quantity: 0
      quantitymax: 0
      price: 1
    - itemid: 40043    # clay flask
      quantity: 0
      quantitymax: 0
      price: 1
    # Existing Stillwater-specific potions stay (30036, 30037, 30060)
    - itemid: 30036
      quantity: 0
      quantitymax: 0
      price: 8
    - itemid: 30037
      quantity: 0
      quantitymax: 0
      price: 8
    - itemid: 30060
      quantity: 0
      quantitymax: 0
      price: 12
```

- [ ] **Step 4: Update Voss's `shop:` block to mirror.**

Replace the existing `shop:` block in `_datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml` with:

```yaml
  shop:
    # Thornwall chrysalis (in-shop crafted)
    - itemid: 40010    # Chrysalis Core
      quantity: 0
      quantitymax: 0
      price: 80
    - itemid: 40029    # mutation catalyst
      quantity: 0
      quantitymax: 0
      price: 60
    # Stillwater unique (caravan-fed from Ilsa)
    - itemid: 40056    # marsh willow bark
      quantity: 0
      quantitymax: 0
      price: 18
    - itemid: 40057    # lake mint
      quantity: 0
      quantitymax: 0
      price: 15
    # Fernway forest (caravan-fed from Fernway forager pickup)
    - itemid: 40062    # oak bark
      quantity: 0
      quantitymax: 0
      price: 35
    - itemid: 40063    # shadowcap mushroom
      quantity: 0
      quantitymax: 0
      price: 40
    - itemid: 40065    # beeswax
      quantity: 0
      quantitymax: 0
      price: 30
    - itemid: 40066    # blood-moss
      quantity: 0
      quantitymax: 0
      price: 45
    # Mid-tier overlap alchemy
    - itemid: 40004    # healer's root
      quantity: 0
      quantitymax: 0
      price: 5
    - itemid: 40005    # bitter thistle
      quantity: 0
      quantitymax: 0
      price: 5
    - itemid: 40009    # dustwalk herb
      quantity: 0
      quantitymax: 0
      price: 8
    # Base alchemy infrastructure
    - itemid: 40006    # glass vial
      quantity: 0
      quantitymax: 0
      price: 1
    - itemid: 40043    # clay flask
      quantity: 0
      quantitymax: 0
      price: 1
```

Note: Voss had cloth strip (40007) — DROPPED here per spec (defer to 3.0e). Voss had healer's root (40004) — kept since it's mid-tier overlap and Voss has the alchemy specialty. Stillwater pricing for Stillwater-unique mats is cheaper at Ilsa (10/12) than at Voss (15/18) — reflects the caravan markup; the existing scarcity multiplier will further differentiate during runtime.

- [ ] **Step 5: Delete any stale instance saves for Ilsa or Voss** (per CLAUDE.md SOP).

```bash
rm -f _datafiles/world/dogmud/shops/stillwater/338-apothecary_ilsa-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/98-apothecary_voss-room*.yaml
```

If those files don't exist, no-op.

- [ ] **Step 6: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/338-apothecary_ilsa.yaml _datafiles/world/dogmud/mobs/thornwall_city/98-apothecary_voss.yaml
git commit -m "content(vendors): mirror alchemy pair Ilsa + Voss inventories per Stage 3.0b

Both apothecaries now stock the mirrored slot set: Stillwater-unique
(lake mint, marsh-willow), Thornwall-unique (Chrysalis Core, mutation
catalyst), Fernway-unique (oak bark, shadowcap, beeswax, blood-moss),
mid-tier overlap (healer's root, bitter thistle, +dustwalk at Voss),
and base infrastructure (glass vial, clay flask). Cloth strip dropped
from Voss pending 3.0e corpse salvage.

Same-region mats are cheaper at the native vendor (lake mint 10g at
Ilsa, 15g at Voss) reflecting caravan markup. Existing scarcity
multiplier handles further price dynamics at runtime.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Vendor inventory — blacksmith pair (Brindle 337 + Kerra 97)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml`

Smith pair stocks: Stillwater unique (lake-iron), Thornwall unique (steel ingot, refined wires), Fernway (pine pitch — smithing use), base (iron ingot, wooden plank, chain link, coal dust).

- [ ] **Step 1: Read current shop entries for both vendors.**

Run: read both files. Current per scout:
- Brindle (337): 40001, 40059, 40002, 32
- Kerra (97): 40001, 40020, 40002, 40003

(40002 leather strip = defer to 3.0e; 32 is a non-mat item like a tool — leave Brindle's 32 in.)

- [ ] **Step 2: Update Brindle's `shop:` block.**

Replace the existing `shop:` block in `_datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml` with:

```yaml
  shop:
    # Stillwater unique (forager-fed)
    - itemid: 40059    # lake-iron nodule
      quantity: 0
      quantitymax: 0
      price: 8
    # Thornwall refined (caravan-fed from Kerra)
    - itemid: 40018    # steel ingot
      quantity: 0
      quantitymax: 0
      price: 30
    # Fernway forest (caravan-fed)
    - itemid: 40067    # pine pitch
      quantity: 0
      quantitymax: 0
      price: 35
    # Base smithing
    - itemid: 40001    # iron ingot
      quantity: 0
      quantitymax: 0
      price: 3
    - itemid: 40003    # wooden plank
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40019    # chain link
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 40020    # coal dust
      quantity: 0
      quantitymax: 0
      price: 4
    # Existing tool item
    - itemid: 32
      quantity: 0
      quantitymax: 0
      price: 0
```

Note: leather strip (40002) DROPPED — defer to 3.0e.

- [ ] **Step 3: Update Kerra's `shop:` block.**

Replace the existing `shop:` block in `_datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml` with:

```yaml
  shop:
    # Thornwall refined (in-shop crafted)
    - itemid: 40018    # steel ingot
      quantity: 0
      quantitymax: 0
      price: 25
    # Stillwater unique (caravan-fed from Brindle)
    - itemid: 40059    # lake-iron nodule
      quantity: 0
      quantitymax: 0
      price: 14
    # Fernway forest (caravan-fed)
    - itemid: 40067    # pine pitch
      quantity: 0
      quantitymax: 0
      price: 35
    # Base smithing
    - itemid: 40001    # iron ingot
      quantity: 0
      quantitymax: 0
      price: 3
    - itemid: 40003    # wooden plank
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40019    # chain link
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 40020    # coal dust
      quantity: 0
      quantitymax: 0
      price: 4
```

Note: leather strip (40002) DROPPED — defer to 3.0e.

- [ ] **Step 4: Delete stale instance saves.**

```bash
rm -f _datafiles/world/dogmud/shops/stillwater/337-smith_brindle-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/97-blacksmith_kerra-room*.yaml
```

- [ ] **Step 5: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml _datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml
git commit -m "content(vendors): mirror blacksmith pair Brindle + Kerra inventories per Stage 3.0b

Both smiths now stock: Stillwater-unique (lake-iron), Thornwall-unique
(steel ingot), Fernway-unique (pine pitch for rust prevention), and
base smithing mats. Leather strip dropped pending 3.0e corpse salvage.
Stillwater-region mat (lake-iron) priced cheaper at Brindle (8g) vs
caravan-markup at Kerra (14g); same for steel ingot reverse direction.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Vendor inventory — gem/enchant trio (Kess 340 + Tess 108 + Vael 109)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/340-pearl_carver_kess.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/108-jeweler_tess.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/109-enchanter_vael.yaml`

Three-vendor pair — Pearl-carver Kess (Stillwater pearl-focused) ↔ Jeweler Tess (Thornwall gem/wire-focused) + Enchanter Vael (Thornwall chrysalis-focused). Kess stocks pearl + caravan-fed Thornwall jewelcraft mats. Tess and Vael stock their natives + caravan-fed Stillwater pearl + caravan-fed Fernway oak bark (enchanting use).

- [ ] **Step 1: Read current shop entries.**

Per scout:
- Kess (340): 40022, 40019, 40024, 40053
- Tess (108): 40021, 40022, 40023, 40026
- Vael (109): 40028

- [ ] **Step 2: Update Kess's `shop:` block** in `_datafiles/world/dogmud/mobs/stillwater/340-pearl_carver_kess.yaml`:

```yaml
  shop:
    # Stillwater unique (forager-fed)
    - itemid: 40053    # Stillwater black pearl
      quantity: 0
      quantitymax: 0
      price: 400
    # Thornwall refined jewelcraft (caravan-fed from Tess)
    - itemid: 40021    # copper wire
      quantity: 0
      quantitymax: 0
      price: 12
    - itemid: 40022    # silver wire
      quantity: 0
      quantitymax: 0
      price: 25
    - itemid: 40023    # gold wire
      quantity: 0
      quantitymax: 0
      price: 60
    - itemid: 40024    # polished stone
      quantity: 0
      quantitymax: 0
      price: 8
    - itemid: 40025    # raw gem
      quantity: 0
      quantitymax: 0
      price: 40
    - itemid: 40026    # gem dust
      quantity: 0
      quantitymax: 0
      price: 6
    # Thornwall chrysalis (caravan-fed from Vael)
    - itemid: 40030    # chrysalis setting
      quantity: 0
      quantitymax: 0
      price: 90
    # Base jewelcraft
    - itemid: 40019    # chain link
      quantity: 0
      quantitymax: 0
      price: 4
```

- [ ] **Step 3: Update Tess's `shop:` block** in `_datafiles/world/dogmud/mobs/thornwall_city/108-jeweler_tess.yaml`:

```yaml
  shop:
    # Thornwall refined (in-shop crafted)
    - itemid: 40021    # copper wire
      quantity: 0
      quantitymax: 0
      price: 10
    - itemid: 40022    # silver wire
      quantity: 0
      quantitymax: 0
      price: 22
    - itemid: 40023    # gold wire
      quantity: 0
      quantitymax: 0
      price: 55
    - itemid: 40024    # polished stone
      quantity: 0
      quantitymax: 0
      price: 7
    - itemid: 40025    # raw gem
      quantity: 0
      quantitymax: 0
      price: 35
    - itemid: 40026    # gem dust
      quantity: 0
      quantitymax: 0
      price: 5
    # Stillwater unique (caravan-fed from Kess)
    - itemid: 40053    # Stillwater black pearl
      quantity: 0
      quantitymax: 0
      price: 600
    # Thornwall chrysalis (in-shop / Vael overlap)
    - itemid: 40030    # chrysalis setting
      quantity: 0
      quantitymax: 0
      price: 80
    # Base jewelcraft
    - itemid: 40019    # chain link
      quantity: 0
      quantitymax: 0
      price: 4
```

- [ ] **Step 4: Update Vael's `shop:` block** in `_datafiles/world/dogmud/mobs/thornwall_city/109-enchanter_vael.yaml`:

```yaml
  shop:
    # Thornwall chrysalis (in-shop crafted) — Vael's specialty
    - itemid: 40010    # Chrysalis Core
      quantity: 0
      quantitymax: 0
      price: 75
    - itemid: 40027    # chrysalis shard
      quantity: 0
      quantitymax: 0
      price: 20
    - itemid: 40028    # binding paste
      quantity: 0
      quantitymax: 0
      price: 25
    - itemid: 40029    # mutation catalyst
      quantity: 0
      quantitymax: 0
      price: 55
    - itemid: 40030    # chrysalis setting
      quantity: 0
      quantitymax: 0
      price: 75
    - itemid: 40011    # Hive Fragment
      quantity: 0
      quantitymax: 0
      price: 30
    # Fernway forest (caravan-fed) — oak bark for ward inscriptions
    - itemid: 40062    # oak bark
      quantity: 0
      quantitymax: 0
      price: 35
    # Stillwater unique (caravan-fed from Kess) — pearl for high-tier enchants
    - itemid: 40053    # Stillwater black pearl
      quantity: 0
      quantitymax: 0
      price: 600
```

Note: Vael previously only stocked binding-paste; now stocks the full chrysalis line + cross-region pearl + Fernway oak bark.

- [ ] **Step 5: Delete stale instance saves for all three.**

```bash
rm -f _datafiles/world/dogmud/shops/stillwater/340-pearl_carver_kess-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/108-jeweler_tess-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/109-enchanter_vael-room*.yaml
```

- [ ] **Step 6: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/340-pearl_carver_kess.yaml _datafiles/world/dogmud/mobs/thornwall_city/108-jeweler_tess.yaml _datafiles/world/dogmud/mobs/thornwall_city/109-enchanter_vael.yaml
git commit -m "content(vendors): mirror gem/enchant trio Kess + Tess + Vael per Stage 3.0b

Pearl-carver Kess gains Thornwall refined jewelcraft slots (wires,
polished stone, raw gem, gem dust) caravan-fed from Tess. Jeweler
Tess gains Stillwater pearl slot caravan-fed from Kess. Enchanter
Vael's inventory expanded from binding-paste-only to full chrysalis
line + Fernway oak bark (ward inscriptions) + Stillwater pearl (high-
tier enchants). Stillwater-unique mats marked up at Thornwall side;
Thornwall-unique mats marked up at Stillwater side.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Vendor inventory — food/inn group (Sigrid 333 + Bram 348 + Tov Brann 336 + Brynn 248 + Food Vendor 103)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/333-innkeeper_sigrid.yaml`
- Modify: `_datafiles/world/dogmud/mobs/stillwater/336-fishmonger_tov_brann.yaml`
- Modify: `_datafiles/world/dogmud/mobs/stillwater/348-miller_bram.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/103-food_vendor.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/248-tavern_cook_brynn.yaml`

Food/inn vendors stock cooking-relevant mats. Stillwater natives = freshwater clam, hunter-eel hide (food), wild vegetables, salt; Thornwall natives = raw meat, wild vegetables, salt, water flask; Fernway = wild hare meat, shadowcap, blood-moss (cooking).

For brevity, this task batches the 5 vendors with concise edits.

- [ ] **Step 1: Read current shop entries.**

Per scout:
- Sigrid (333): 30061, 40016, 30021
- Tov Brann (336): 40058, 40014
- Bram (348): 40015
- Food Vendor (103): 30022, 30023, 30024, 30021
- Brynn (248): 40014, 40017, 40015, 40016, 40004

- [ ] **Step 2: Update Sigrid (innkeeper)** — keeps existing inn-specific items + adds Stillwater + Fernway food slots.

Replace `shop:` in `_datafiles/world/dogmud/mobs/stillwater/333-innkeeper_sigrid.yaml`:

```yaml
  shop:
    # Stillwater unique cooking
    - itemid: 40058    # freshwater clam
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 40057    # lake mint
      quantity: 0
      quantitymax: 0
      price: 10
    # Fernway cooking (caravan-fed)
    - itemid: 40063    # shadowcap mushroom
      quantity: 0
      quantitymax: 0
      price: 40
    - itemid: 40064    # wild hare meat
      quantity: 0
      quantitymax: 0
      price: 50
    # Base cooking
    - itemid: 40014    # raw meat
      quantity: 0
      quantitymax: 0
      price: 3
    - itemid: 40015    # wild vegetables
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40016    # water flask
      quantity: 0
      quantitymax: 0
      price: 1
    - itemid: 40017    # salt pouch
      quantity: 0
      quantitymax: 0
      price: 1
    # Existing inn items
    - itemid: 30061
      quantity: 0
      quantitymax: 0
      price: 8
    - itemid: 30021
      quantity: 0
      quantitymax: 0
      price: 4
```

- [ ] **Step 3: Update Tov Brann (fishmonger)** — Stillwater fish specialist.

Replace `shop:` in `_datafiles/world/dogmud/mobs/stillwater/336-fishmonger_tov_brann.yaml`:

```yaml
  shop:
    # Stillwater unique fish
    - itemid: 40058    # freshwater clam
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 40051    # skitter-shrimp shell
      quantity: 0
      quantitymax: 0
      price: 8
    # Base meat
    - itemid: 40014    # raw meat (covers fish too)
      quantity: 0
      quantitymax: 0
      price: 3
    - itemid: 40017    # salt pouch
      quantity: 0
      quantitymax: 0
      price: 1
```

- [ ] **Step 4: Update Bram (miller)** — Stillwater-local grain/produce specialist (no Thornwall pair per spec).

Replace `shop:` in `_datafiles/world/dogmud/mobs/stillwater/348-miller_bram.yaml`:

```yaml
  shop:
    # Base cooking
    - itemid: 40015    # wild vegetables
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40017    # salt pouch
      quantity: 0
      quantitymax: 0
      price: 1
    # Fernway cooking (caravan-fed)
    - itemid: 40066    # blood-moss
      quantity: 0
      quantitymax: 0
      price: 45
```

- [ ] **Step 5: Update Food Vendor (Thornwall)** — keeps existing food items + adds cross-region cooking slots.

Replace `shop:` in `_datafiles/world/dogmud/mobs/thornwall_city/103-food_vendor.yaml`:

```yaml
  shop:
    # Existing prepared foods
    - itemid: 30022
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 30023
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 30024
      quantity: 0
      quantitymax: 0
      price: 4
    - itemid: 30021
      quantity: 0
      quantitymax: 0
      price: 4
    # Stillwater unique (caravan-fed from Tov Brann / Sigrid)
    - itemid: 40058    # freshwater clam
      quantity: 0
      quantitymax: 0
      price: 8
    # Fernway cooking (caravan-fed)
    - itemid: 40063    # shadowcap mushroom
      quantity: 0
      quantitymax: 0
      price: 40
    - itemid: 40064    # wild hare meat
      quantity: 0
      quantitymax: 0
      price: 50
    # Base cooking
    - itemid: 40014    # raw meat
      quantity: 0
      quantitymax: 0
      price: 3
    - itemid: 40015    # wild vegetables
      quantity: 0
      quantitymax: 0
      price: 2
```

- [ ] **Step 6: Update Brynn (tavern cook)** — keeps existing kitchen items + cross-region.

Replace `shop:` in `_datafiles/world/dogmud/mobs/thornwall_city/248-tavern_cook_brynn.yaml`:

```yaml
  shop:
    # Stillwater unique cooking (caravan-fed)
    - itemid: 40057    # lake mint
      quantity: 0
      quantitymax: 0
      price: 15
    - itemid: 40058    # freshwater clam
      quantity: 0
      quantitymax: 0
      price: 8
    # Fernway cooking (caravan-fed)
    - itemid: 40063    # shadowcap mushroom
      quantity: 0
      quantitymax: 0
      price: 40
    - itemid: 40064    # wild hare meat
      quantity: 0
      quantitymax: 0
      price: 50
    - itemid: 40066    # blood-moss
      quantity: 0
      quantitymax: 0
      price: 45
    # Base cooking
    - itemid: 40004    # healer's root
      quantity: 0
      quantitymax: 0
      price: 5
    - itemid: 40014    # raw meat
      quantity: 0
      quantitymax: 0
      price: 3
    - itemid: 40015    # wild vegetables
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40016    # water flask
      quantity: 0
      quantitymax: 0
      price: 1
    - itemid: 40017    # salt pouch
      quantity: 0
      quantitymax: 0
      price: 1
```

- [ ] **Step 7: Delete stale instance saves for all five.**

```bash
rm -f _datafiles/world/dogmud/shops/stillwater/{333,336,348}-*-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/{103,248}-*-room*.yaml
```

- [ ] **Step 8: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 9: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/{333,336,348}-*.yaml _datafiles/world/dogmud/mobs/thornwall_city/{103,248}-*.yaml
git commit -m "content(vendors): wire food/inn group inventories per Stage 3.0b

Five food/inn vendors (Sigrid, Tov Brann, Bram in Stillwater; Food
Vendor, Brynn in Thornwall) now stock the cooking mat regional split.
Stillwater natives (clam, lake mint, skitter shrimp shell) flow to
Thornwall via caravan; Fernway natives (shadowcap, wild hare, blood-
moss) flow to both towns. Bram is Stillwater-local (no Thornwall miller
pair per spec).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Vendor inventory — general goods (Wulf 341 + Siv 104 + Whisper 273)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/341-storekeeper_wulf.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/104-fence_dealer_siv.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/273-whisper.yaml`

General goods vendors stock base mats + tools + lantern oil. Loose pair — Stillwater Storekeeper Wulf ↔ Thornwall Fence Dealer Siv (and Whisper for shadier wares). Mostly leave existing inventories alone; just add base-mat slots that should be available at general goods stores.

- [ ] **Step 1: Read current shop entries.**

Per scout:
- Wulf (341): 32, 40038, 40016, 40015, 40017, 30021
- Siv (104): 32, 33, 36 (no mats — stolen-goods specialist; minimal expansion)
- Whisper (273): 33, 34, 36 (info/contraband — minimal expansion)

- [ ] **Step 2: Update Wulf** — already has water flask + wild vegetables + salt + lantern + tools. Add a couple base mats and lake-iron (Stillwater general-goods cross-sell).

Replace `shop:` in `_datafiles/world/dogmud/mobs/stillwater/341-storekeeper_wulf.yaml`:

```yaml
  shop:
    # Existing tools/lantern/inn items
    - itemid: 32
      quantity: 0
      quantitymax: 0
      price: 0
    - itemid: 40038    # oil lantern
      quantity: 0
      quantitymax: 0
      price: 5
    - itemid: 30021
      quantity: 0
      quantitymax: 0
      price: 4
    # Base mats (general goods)
    - itemid: 40015    # wild vegetables
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40016    # water flask
      quantity: 0
      quantitymax: 0
      price: 1
    - itemid: 40017    # salt pouch
      quantity: 0
      quantitymax: 0
      price: 1
    - itemid: 40003    # wooden plank
      quantity: 0
      quantitymax: 0
      price: 2
    # Stillwater unique (forager-fed; cross-sell from Brindle's surplus)
    - itemid: 40059    # lake-iron nodule
      quantity: 0
      quantitymax: 0
      price: 10
```

- [ ] **Step 3: Update Siv (fence dealer)** — minimal change. Siv deals in stolen goods, not regional mats. Leave existing inventory; add base wood plank + lantern oil for "general goods".

Replace `shop:` in `_datafiles/world/dogmud/mobs/thornwall_city/104-fence_dealer_siv.yaml`:

```yaml
  shop:
    # Existing fenced/stolen items
    - itemid: 32
      quantity: 0
      quantitymax: 0
      price: 0
    - itemid: 33
      quantity: 0
      quantitymax: 0
      price: 0
    - itemid: 36
      quantity: 0
      quantitymax: 0
      price: 0
    # Base general goods
    - itemid: 40003    # wooden plank
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40038    # oil lantern
      quantity: 0
      quantitymax: 0
      price: 5
```

- [ ] **Step 4: Update Whisper** — info broker, leave existing inventory alone. Just add lantern oil.

Replace `shop:` in `_datafiles/world/dogmud/mobs/thornwall_city/273-whisper.yaml`:

```yaml
  shop:
    # Existing info/contraband items
    - itemid: 33
      quantity: 0
      quantitymax: 0
      price: 0
    - itemid: 34
      quantity: 0
      quantitymax: 0
      price: 0
    - itemid: 36
      quantity: 0
      quantitymax: 0
      price: 0
    # Base
    - itemid: 40038    # oil lantern
      quantity: 0
      quantitymax: 0
      price: 5
```

- [ ] **Step 5: Delete stale instance saves.**

```bash
rm -f _datafiles/world/dogmud/shops/stillwater/341-storekeeper_wulf-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/104-fence_dealer_siv-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/273-whisper-room*.yaml
```

- [ ] **Step 6: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 7: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/341-storekeeper_wulf.yaml _datafiles/world/dogmud/mobs/thornwall_city/104-fence_dealer_siv.yaml _datafiles/world/dogmud/mobs/thornwall_city/273-whisper.yaml
git commit -m "content(vendors): wire general goods trio (Wulf, Siv, Whisper) per Stage 3.0b

Storekeeper Wulf adds base mats (vegetables, water, salt, wooden plank)
and Stillwater-unique lake-iron cross-sell. Siv (fence) and Whisper
(info broker) get minimal base additions (wood, lantern); their
specialty inventories stay intact. These are loose pair members — the
caravan doesn't move regional mats through them in v1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Vendor inventory — weaver pair (Edda 339 + Maren 113)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/stillwater/339-weaver_edda.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/113-weaver_maren.yaml`

Weavers handle cloth/leather/cord — most of which are deferred to 3.0e. This task is minimal: drop the cloth/leather slots that 3.0e will reorganize, and add the Fernway beeswax slot (which weavers will use for waterproofing post-3.0e).

- [ ] **Step 1: Read current shop entries.**

Per scout:
- Edda (339): 40007 (cloth strip — defer), 40012 (thread spool), 40013 (bone needle), 40055 (cattail down — defer), 40002 (leather strip — defer)
- Maren (113): 40007 (defer), 40012, 40002 (defer), 40013, 30033

- [ ] **Step 2: Update Edda's `shop:` block** — drop deferred cloth/leather/cattail, keep base sewing tools, add beeswax.

Replace `shop:` in `_datafiles/world/dogmud/mobs/stillwater/339-weaver_edda.yaml`:

```yaml
  shop:
    # Base sewing tools
    - itemid: 40012    # thread spool
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40013    # bone needle
      quantity: 0
      quantitymax: 0
      price: 1
    # Fernway beeswax (waterproofing — wiring deferred to 3.0e but
    # the slot exists so foragers/caravan can fill it)
    - itemid: 40065    # beeswax
      quantity: 0
      quantitymax: 0
      price: 30
```

- [ ] **Step 3: Update Maren's `shop:` block** — same pattern as Edda + keep her existing 30033 specialty item.

Replace `shop:` in `_datafiles/world/dogmud/mobs/thornwall_city/113-weaver_maren.yaml`:

```yaml
  shop:
    # Base sewing tools
    - itemid: 40012    # thread spool
      quantity: 0
      quantitymax: 0
      price: 2
    - itemid: 40013    # bone needle
      quantity: 0
      quantitymax: 0
      price: 1
    # Fernway beeswax
    - itemid: 40065    # beeswax
      quantity: 0
      quantitymax: 0
      price: 30
    # Existing specialty
    - itemid: 30033
      quantity: 0
      quantitymax: 0
      price: 0
```

Note: cloth strip, leather strip, cattail down all DROPPED for now. 3.0e reorganizes the cloth/leather supply pipeline and re-adds these slots with proper sourcing.

- [ ] **Step 4: Delete stale instance saves.**

```bash
rm -f _datafiles/world/dogmud/shops/stillwater/339-weaver_edda-room*.yaml
rm -f _datafiles/world/dogmud/shops/thornwall_city/113-weaver_maren-room*.yaml
```

- [ ] **Step 5: Verify build clean.**

Run: `go build ./...`

- [ ] **Step 6: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/stillwater/339-weaver_edda.yaml _datafiles/world/dogmud/mobs/thornwall_city/113-weaver_maren.yaml
git commit -m "content(vendors): minimal weaver pair update (Edda + Maren) per Stage 3.0b

Weavers' cloth/leather/cord/cattail slots deferred to 3.0e (corpse
salvage). Both weavers retain base sewing tools (thread spool, bone
needle) and gain Fernway beeswax slot (waterproof treatment recipe
deferred to 3.0e but the slot exists for foragers/caravan to fill).
Maren's existing specialty item (30033) preserved.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 14: Verification — server boot + recipe sanity

**Files:** none — verification only.

This task confirms the full Stage 3.0b shipment loads cleanly and recipes resolve correctly. No commits unless issues found.

- [ ] **Step 1: Full build sweep.**

Run: `go build ./...`
Expected: no output (clean build).

- [ ] **Step 2: Full test suite.**

Run: `go test ./...`
Expected: all tests pass. Pay special attention to any item-load or recipe-load tests (e.g., `TestHelpFileCompleteness_Recipes` if applicable to mat recipes — confirm no missing-help-file warnings for recipe edits).

- [ ] **Step 3: Boot the server locally** per the Pre-Push SOP in CLAUDE.md.

Watch stdout for:
- `mobs.LoadDataFiles() loadedCount=N` — N should match prior count (no mob YAMLs added, just edited)
- `items.LoadDataFiles() loadedCount=N` — N should be +6 vs prior count (the 6 new Fernway mats)
- `crafting.LoadRecipeFiles() loadedCount=97` — should still be 97 (no recipes added, just edited)
- No panic during data file loading

If a panic fires citing a specific YAML file, fix and retry.

- [ ] **Step 4: Recipe ingredient sanity.**

Run a quick grep to confirm every new mat appears in at least 2 recipes:

```bash
grep -l "oak-bark" _datafiles/world/dogmud/recipes/**/*.yaml | wc -l   # Expected: 2
grep -l "shadowcap" _datafiles/world/dogmud/recipes/**/*.yaml | wc -l   # Expected: 2
grep -l "wild-hare-meat" _datafiles/world/dogmud/recipes/**/*.yaml | wc -l  # Expected: 2
grep -l "beeswax" _datafiles/world/dogmud/recipes/**/*.yaml | wc -l     # Expected: 1 (alchemy; tailoring deferred)
grep -l "blood-moss" _datafiles/world/dogmud/recipes/**/*.yaml | wc -l   # Expected: 2
grep -l "pine-pitch" _datafiles/world/dogmud/recipes/**/*.yaml | wc -l   # Expected: 2
```

- [ ] **Step 5: Vendor slot sanity (manual, in-game).**

Connect to the local server. Visit a sample of vendors:
- Apothecary Ilsa (Stillwater) — confirm `list` shows mirrored alchemy slots including new Fernway mats
- Apothecary Voss (Thornwall) — same mirrored set, different prices for cross-region mats
- Smith Brindle / Blacksmith Kerra — mirrored smith slots, no leather strip
- Pearl-carver Kess / Jeweler Tess — mirrored gem slots, with Stillwater pearl available at both
- Enchanter Vael — full chrysalis line + oak bark + Stillwater pearl
- Weaver Edda / Maren — minimal slots (sewing tools + beeswax), no cloth strip / leather strip / cattail down

If any vendor's `list` shows unexpected items, debug:
- Check if a stale instance save survived in `_datafiles/world/dogmud/shops/<zone>/`
- Check if the YAML edit didn't take effect (rebuild + reboot)

- [ ] **Step 6: No commit needed if everything passes.**

If issues found, file follow-up commits per the appropriate task above.

---

### Task 15: PATCH_NOTES + final commit

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Read the top of `PATCH_NOTES.md`** to confirm format and find insertion point (above the most recent entry).

- [ ] **Step 2: Add Stage 3.0b entry.**

Insert at the top, above the most recent entry:

```markdown
## 2026-04-XX — Stage 3.0b: Material Region Split (dev only)

**Note:** This is a dev-only landing. The full economy stack (Stages
3.0b through 3.4) sits unmerged on the `development` branch and ships
to prod (`master`) as a coherent update once Stage 3.4 lands.

- Added 6 new Fernway forest materials: oak bark (40062), shadowcap
  mushroom (40063), wild hare meat (40064), beeswax (40065),
  blood-moss (40066), pine pitch (40067). Each consumed in 2-3
  mid/high-tier recipes spanning at least 2 craft schools, giving
  forager-gathered Fernway mats real demand once Stage 3.1 ships.
- Audit matrix at `docs/economy/mat-audit-matrix.md` classifies all 67
  raw materials into regional supply buckets (Stillwater, Thornwall,
  Fernway, base, mid-tier overlap, deferred-to-3.0e). This is the
  durable artifact that subsequent stages (foragers, corpse salvage,
  real item transfer) consume.
- Reshaped vendor inventories across the 17 caravan-served vendors
  into mirrored same-craft pairs. Same-craft Stillwater + Thornwall
  vendors now stock the same mat types, with regional pricing
  asymmetry reflecting the caravan markup. Cloth/leather slots
  dropped pending Stage 3.0e (corpse salvage).
- ~12 mid/high-tier recipes updated to wire demand for the new
  Fernway mats. No new recipes invented; existing recipe corpus
  expanded with one new ingredient slot each.

(Today's date when this lands.)
```

- [ ] **Step 3: Verify build/tests still pass.**

Run: `go build ./...` and `go test ./...`
Expected: clean.

- [ ] **Step 4: Commit.**

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): add Stage 3.0b material region split entry

Stage 3.0b dev-only landing. Documents the 6 new Fernway mats, the
audit matrix doc, vendor inventory reshaping, and the ~12 recipe
demand-wiring edits. Notes that the full economy stack sits unmerged
on development pending Stage 3.4.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-review checklist

**Spec coverage** (run after plan written, before starting implementation):
- ✅ Goal — audit, invent 6 Fernway mats, recipe wiring, vendor reshape, audit matrix doc → covered by T1-T15
- ✅ Three regions — Stillwater + Thornwall + Fernway → reflected in audit matrix (T1) and vendor pairs (T8-T13)
- ✅ 6 new mats — oak bark, shadowcap, wild hare, beeswax, blood-moss, pine pitch → T2-T7 (one task each)
- ✅ Recipe wiring (2-3 across 2 schools per mat) → T2-T7 (each task wires its mat into 1-2 recipes)
- ✅ Vendor inventory pair pattern → T8-T13 (alchemy, blacksmith, gem/enchant, food/inn, general goods, weaver)
- ✅ Cloth/leather classification in matrix, wiring deferred → T1 audit matrix flags them; T13 weaver task explicitly drops cloth slots
- ✅ Audit matrix doc → T1 produces it
- ✅ Out-of-scope items (3.0c zone build, 3.0d fold spells, 3.0e salvage, 3.1 foragers, 3.4 transfer, pricing tuning) → not in any task
- ✅ Verification — boot test + recipe sanity → T14
- ✅ PATCH_NOTES → T15

**Placeholder scan:**
- "Today's date when this lands" in T15 PATCH_NOTES — explicit, expected; implementer fills with the actual date
- "audit during impl" / "flag during audit" in T1 audit matrix template — these are LEGITIMATE deferrals (the implementer reads each YAML during T1 to make calls); not unspecified scope, just "use judgment based on what you read"
- "(audit during implementation)" in mid-tier overlap rows of T1 template — same as above; the 5-6 mats listed (40046-40050) need their region classified by reading their descriptions, which is in scope for T1
- "(no Thornwall miller)" — accurate factual note, not a placeholder
- All other steps have concrete file paths, complete YAML/code blocks, and exact commands

**Type consistency:**
- `component_tag` values match between mat YAMLs and recipe `item_tag` references: oak-bark, shadowcap, wild-hare-meat, beeswax, blood-moss, pine-pitch — verified consistent across T2-T7 and the recipe insertion table at the top
- Item IDs 40062-40067 used consistently across audit matrix (T1), mat YAMLs (T2-T7), and vendor shop entries (T8-T13)
- Vendor mob IDs (337, 338, etc.) match the spec's pair table and the scout report
