# New Plymouth — Crafting District (content) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the New Plymouth Crafting Quarter (`new_plymouth_crafting`, 25
rooms 5700–5724): 7 anchor artisans + ambient, dialogue, the `cooperage_circle`
faction, the Bloom Trail middle breadcrumb, and anchor schedules — a complete,
boot-clean, harness-playtested content district.

**Architecture:** Pure content (YAML data files) authored by content subagents
(`/new-room`, `/new-mob`) within a pre-allocated ID block, validated by server
boot + `cartcheck` + schedule validators + a harness playtest. The supply-runner
*engine* is a separate plan (`2026-06-22-np-supply-runner.md`); this plan only
ensures vendors pre-declare deliverable `StockEntry`s so they are delivery-ready.

**Tech Stack:** GoMud engine (Go server, run locally to boot-test), DOGMud world
YAML schema (`docs/schemas/`), `tools/id_inventory.py`, the `cartcheck` admin
command, the `/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-06-22-np-crafting-district-design.md`.

---

## Conventions for every task (READ FIRST)

- **Branch:** work on `feature/np-crafting-district` (create from `master` in
  Task 0). Commit per stage with `feat(np-crafting): …` conventional messages,
  trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Boot test = the verification.** YAML errors panic at *startup*, not build.
  The boot-test recipe (used verbatim in every "verify" step):
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . 2>&1 | tee /tmp/np_boot.log &   # server boots, loads data, then idles
  # wait for the world-load lines, then:
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath" /tmp/np_boot.log
  grep -nE "ValidateZoneConsistency|rooms\.LoadDataFiles|mobs\.LoadDataFiles" /tmp/np_boot.log
  ```
  Do **NOT** grep for bare `panic` (matches the config line
  `MapConsistencyEnforce value=panic` — gotcha #8). Kill the server after.
- **`cartcheck`** runs in-game (`cartcheck new_plymouth_crafting`) and at boot via
  `ValidateZoneConsistency` (mode `warn`); a clean district reports `errors=0`.
- **Authoring gotchas (panic / render-breaking):**
  1. YAML prose containing `": "` inside a noun/desc value breaks the parser → use
     `" — "` instead of `": "`.
  2. Faction `allies`/`enemies` forward-refs PANIC — only reference factions that
     already exist.
  3. Mob `name:` must be Title-Case (`casing.AssertCanonical` panics otherwise).
  4. Faction membership is via mob `groups:`, **not** a `faction:` field.
  5. Exits to not-yet-built rooms dangle — author deferred cross-district links as
     **prose only** (no `exits:` entry).
  6. Every shop-bearing mob needs a valid top-level `craft_support:` tag (one of
     `cooking`/`general`/`blacksmithing`/`alchemy`/`tailoring`/`jewelcrafting`/
     `enchanting`) or `ValidateShopMobTags` PANICS.
  7. Noun-highlight ansi MUST be `<ansi fg="itemname">noun</ansi>` — never
     `fg="<noun>"`. After authoring rooms, run
     `grep -rE 'fg="[^"]* [^"]*"' _datafiles/world/dogmud/rooms/new_plymouth_crafting/`
     — expect **zero** matches.
  8. Interiors use **compass/vertical exits only** — never `enter`/`leave` (not
     movement verbs, not `pathto`-traversable). Above-shop rooms = `up`; cellars =
     `down`.
- **Subagents author YAML → ID collisions if parallel.** This plan pre-assigns
  every ID, so a subagent uses only its listed IDs. Dispatch room-authoring
  subagents **sequentially** within a stage (or hand each a disjoint ID list);
  never let two pick "next free ID" at once.

---

## ID allocation (pre-assigned — do not deviate)

| Kind | Range | Assignment |
|------|-------|------------|
| Rooms | **5700–5724** | see Room Manifest |
| Mobs | **9332–9343** | 7 anchors (9332–9338) + 5 ambient (9339–9343) |
| Dialogue | **9326–9332** | one per anchor (Toby included) |
| Items | **40102–40105** | district-flavor goods + Edvar's map (only if needed) |
| Faction | `cooperage_circle` | new slug |
| Zone folder | `new_plymouth_crafting` | `ConvertForFilename("New Plymouth Crafting")` |

---

## Room Manifest (25 rooms — the authoritative layout)

Coordinate region: **x −24 … −8, y 100 … 114, z 0**. The **Long Market** is the
E–W spine (one y-row). Entry from the Docks is a **long** exit (the footbridge)
from Docks room **5523** (x −18 / y 86). The room-authoring subagent assigns exact
non-overlapping coords inside the region, keeping the Long Market on a single
y-row and the listed adjacencies, and runs `cartcheck` until `errors=0`.

**Cluster 1 — The Long Market (E–W spine):**
| ID | Name | Purpose & key exits |
|----|------|---------------------|
| 5700 | Bridge End | Entry plaza where the footbridge lands. `south` → **5523 (long exit, footbridge)**; `north` → 5702. |
| 5701 | The Long Market — West End | West stretch. `east` → 5702. *(Prose-stub: the street continues west toward the docks — no exit.)* |
| 5702 | The Long Market — the Crossing | The hub. `west`→5701, `east`→5703, `south`→5700, `north`→5705 (Inkwalk), plus spurs to clusters (`down`/compass to 5709 forge-row & 5715 tailor-row as the subagent lays out). |
| 5703 | The Long Market — Artisan Stalls | Runner-delivered general stalls (ambient vendor). `west`→5702, `east`→5704. |
| 5704 | The Long Market — East End | `west`→5703. *(Prose-stub: continues east to the Central Square / Merchant — no exit.)* |

**Cluster 2 — The Inkwalk (lore corner, N lane off the Crossing):**
| ID | Name | Purpose & key exits |
|----|------|---------------------|
| 5705 | The Inkwalk | Quiet book-lane. `south`→5702, `east`→5706 (Orin's stall), `west`→5708 (Edvar's). |
| 5706 | Orin's Bookstall | **Orin (9332)** works here. `west`→5705, `up`→5707. |
| 5707 | Above the Bookstall | Orin's room. `down`→5706. |
| 5708 | Edvar's Shuttered Shop | The boarded cartographer's; the Edvar/Gritta lore node (a display-case noun of pre-Founding fragments). `east`→5705. |

**Cluster 3 — The hot trades (forge/lab/kiln):**
| ID | Name | Purpose & key exits |
|----|------|---------------------|
| 5709 | The Forge Yard | **Halvard (9333)** works. Connects to the Crossing; `up`→5710. |
| 5710 | Above the Forge | Halvard's room. `down`→5709. |
| 5711 | Vesna's Alchemy Lab | **Vesna (9334)** works (the Bloom Trail middle node). `up`→5712. |
| 5712 | Above the Lab | Vesna's room. `down`→5711. |
| 5713 | Edda's Glass Kiln | **Edda Glass (9335)** works. |
| 5714 | The Kiln Complex | The dead-kiln expansion stub; an ambient glass-apprentice (9343). Connects to 5713. |

**Cluster 4 — The soft trades / Common seam:**
| ID | Name | Purpose & key exits |
|----|------|---------------------|
| 5715 | Nessa's Tailor Shop | **Nessa (9336)** works. `up`→5716. |
| 5716 | Above the Tailor | Nessa's room. `down`→5715. |
| 5717 | Corwin's Tannery | **Corwin (9337)** works. Prose acknowledges the adjacent Common tannery streets (Common seam — **no exit**, deferred per spec §2). |
| 5718 | The Tanning Yard | Ambient (the smell of honest work). Connects to 5717. |

**Cluster 5 — The abandoned cooperage (lore + OQ stub):**
| ID | Name | Purpose & key exits |
|----|------|---------------------|
| 5719 | The Abandoned Cooperage — Front | **Toby (9338)** tends it; door kept shut. |
| 5720 | The Cooperage Floor | Toby's cot; the `cooperage_circle` memory (Asha references). `up`/compass→5719. |
| 5721 | The Boarded Cellar-Mouth | A boarded stair down toward the canal. **Prose-stub** the `down`→Old Quarter (no exit — OQ unbuilt). Connects to 5720. |

**Cluster 6 — Connective / civic / ambient:**
| ID | Name | Purpose & key exits |
|----|------|---------------------|
| 5722 | Craftwell Square | The district well — civic midday-convergence node. On/near the Long Market. |
| 5723 | Tinpot Alley | A cut-through back alley; ambient porter (9341). |
| 5724 | The Porter's Yard | Goods-marshalling yard near the stalls; ambient. |

---

## Mob Manifest (12 mobs — 9332–9343)

Each anchor: Title-Case `name`, the unique mutation woven into `description`,
`non_combatant: true`, `hostile: false`, `maxwander: 0`, `groups: [humanoid, …]`,
shop-bearers carry a `craft_support:` tag, and a `homeroom`/spawn in the listed
work room. Dialogue file id in parentheses.

| Mob | Name | Work room | Mutation | `craft_support` | `groups` | Notes |
|-----|------|-----------|----------|-----------------|----------|-------|
| 9332 | Orin the Bookseller (dlg 9326) | 5706 | ink-stained eyes that read in the dark | `general` | humanoid, cooperage_circle | sells maps/books; the Edvar/Gritta lore web |
| 9333 | Master Halvard (dlg 9327) | 5709 | skin that won't scorch | `blacksmithing` | humanoid | brother of Marda (Common); supplies Ostry (later) |
| 9334 | Vesna (dlg 9328) | 5711 | iridescent oil-sheen skin | `alchemy` | humanoid | supplies Ysolde (Common); **Bloom Trail middle** |
| 9335 | Edda Glass (dlg 9329) | 5713 | heat-scarred forearms, glass-clear fingernails | `general` | humanoid, cooperage_circle | knew Asha; the kiln |
| 9336 | Nessa the Tailor (dlg 9330) | 5715 | uncannily nimble fingers | `tailoring` | humanoid | supplies Aurel (Noble, later) |
| 9337 | Corwin the Tanner (dlg 9331) | 5717 | leather-tough skin | `tailoring` | humanoid | Common-seam prose |
| 9338 | Toby the Cooper's Lad (dlg 9332) | 5719 | steam-reddened permanent hands | — (no shop) | humanoid, cooperage_circle | knows the basement; cooperage-lore key |
| 9339 | Smith's Apprentice | 5709 | — | — | humanoid | ambient, `non_combatant` |
| 9340 | A Lab Assistant | 5711 | — | — | humanoid | ambient |
| 9341 | A Crafting-Quarter Porter | 5723 | — | — | humanoid | ambient |
| 9342 | A Street-Sweep | 5722 | — | — | humanoid | ambient |
| 9343 | A Glass-Apprentice | 5714 | — | — | humanoid | ambient |

**Shop vendors (delivery-ready for Plan 2):** Orin (5706), Halvard (5709), Vesna
(5711), Edda (5713), Nessa (5715), Corwin (5717), and an ambient general-stall
vendor at 5703. Each shop must **pre-declare** the deliverable feedstock items it
sells as `StockEntry` (see Task 8).

---

## Task 0: Branch + ID sanity check

**Files:** none (git + verification only).

- [ ] **Step 1: Create the feature branch**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git checkout master && git pull --ff-only 2>/dev/null; git checkout -b feature/np-crafting-district
```

- [ ] **Step 2: Confirm the pre-assigned IDs are still free**

```bash
python tools/id_inventory.py --type rooms | grep -E "new_plymouth|next free"
python tools/id_inventory.py --type mobs  | grep -E "new_plymouth|next free"
```

Expected: rooms next-free ≥ 5625 (so 5700–5724 is clear); mobs next-free ≥ 9332
(so 9332–9343 is clear). If anything overlaps, STOP and re-allocate before
authoring.

- [ ] **Step 3: Baseline boot (prove master is clean before we add anything)**

Run the boot-test recipe (Conventions). Expected: zero `ERROR:.*PANIC` /
`fatal error:`; `ValidateZoneConsistency` reports `errors=0`. Kill the server.

---

## Task 1: Stage A rooms — the Long Market + the Inkwalk + the footbridge entry

**Files:**
- Create: `_datafiles/world/dogmud/rooms/new_plymouth_crafting/5700.yaml` …
  `5708.yaml` (9 rooms: 5700–5708)
- Modify: `_datafiles/world/dogmud/rooms/new_plymouth_docks/5523.yaml` (add the
  reciprocal `north` long exit → 5700)

- [ ] **Step 1: Author rooms 5700–5708**

Dispatch a `/new-room`-style content subagent (model: sonnet) with the Cluster 1
+ Cluster 2 manifest rows above, the Conventions block, and these constraints:
- Zone display name `New Plymouth Crafting`, folder `new_plymouth_crafting`.
- Assign exact coords in x −24…−8 / y 100…114 / z 0; Long Market on one y-row;
  5700 at the south column under Docks 5523's x so the footbridge is a clean N–S
  long exit.
- 5700 `south` exit → 5523 with the long-exit form (the footbridge); set the exit
  so `ValidateZoneConsistency` treats it as a long connector (proportional length).
- Honor every deferred prose-stub (5701 west, 5704 east) — **no** exit entries.
- 80-col prose wrap; biome-appropriate; `fg="itemname"` noun ansi only.

- [ ] **Step 2: Wire the Docks side of the footbridge**

Edit `new_plymouth_docks/5523.yaml` — add under `exits:` a `north:` entry whose
`roomid: 5700` (reciprocal of 5700's `south`). The existing prose already
describes "a low stone footbridge to the north", so no prose change is needed.
Mark it a long exit to match 5700's side.

- [ ] **Step 3: Noun-ansi leak check**

```bash
grep -rE 'fg="[^"]* [^"]*"' _datafiles/world/dogmud/rooms/new_plymouth_crafting/
```
Expected: no output.

- [ ] **Step 4: Boot-verify + cartcheck**

Run the boot-test recipe. Expected: no panics; `ValidateZoneConsistency errors=0`
(the new zone loads). If `cartcheck` reports a collision or a non-reciprocal exit,
the room subagent re-assigns coords until clean.

- [ ] **Step 5: Update the coordinate map + commit**

Add the Crafting Stage-A rooms to `docs/coordinate_map.md`. Then:
```bash
git add _datafiles/world/dogmud/rooms/new_plymouth_crafting/57{0,1,2,3,4,5,6,7,8}*.yaml \
        _datafiles/world/dogmud/rooms/new_plymouth_docks/5523.yaml docs/coordinate_map.md
git commit -m "feat(np-crafting): Stage A rooms (5700-5708) — Long Market, Inkwalk, footbridge entry"
```

---

## Task 2: Stage B rooms — hot trades, soft trades, cooperage, connective

**Files:**
- Create: `_datafiles/world/dogmud/rooms/new_plymouth_crafting/5709.yaml` …
  `5724.yaml` (16 rooms: 5709–5724)

- [ ] **Step 1: Author rooms 5709–5724**

Dispatch a content subagent (model: sonnet) with Clusters 3–6 manifest rows + the
Conventions block. Constraints:
- Connect each cluster to the Long Market hub (5702) and stalls (5703) via
  compass/`up`/`down` exits; keep coords non-overlapping in the region.
- Above-shop rooms use `up`/`down` stacks (5710↔5709, 5712↔5711, 5716↔5715).
- 5721 The Boarded Cellar-Mouth: **prose-stub** the `down`→Old Quarter — no exit.
- 5717 Corwin's Tannery: prose acknowledges the adjacent Common tannery streets —
  **no** exit (deferred Common seam).
- 5708 Edvar's Shuttered Shop and 5719/5720 cooperage: author the lore nouns
  (display-case fragments; cooper's tools; an Asha reference) — these are the
  cooperage_circle / Edvar lore surface.

- [ ] **Step 2: Noun-ansi leak check** (same command as Task 1 Step 3) — expect no output.

- [ ] **Step 3: Boot-verify + cartcheck** — boot-test recipe; expect no panics,
  `ValidateZoneConsistency errors=0`. Re-assign coords on any collision.

- [ ] **Step 4: Update coordinate map + commit**

```bash
git add _datafiles/world/dogmud/rooms/new_plymouth_crafting/57{09,1,2}*.yaml docs/coordinate_map.md
git commit -m "feat(np-crafting): Stage B rooms (5709-5724) — hot/soft trades, cooperage, civic"
```

---

## Task 3: The `cooperage_circle` faction

**Files:**
- Create: `_datafiles/world/dogmud/factions/cooperage_circle.yaml`

- [ ] **Step 1: Author the faction**

```yaml
# cooperage_circle.yaml
name: cooperage_circle
displayname: "the Cooperage Circle"
description: "Remaining sympathizers of the departed dissidents — keepers of pre-Founding lore."
defaultreputation: 5
# NO allies/enemies — bloodline_domestic does not exist yet (gotcha #2).
# The enemy edge is added by the Merchant/Noble build that creates that faction.
```
(Match the exact field names used by existing faction YAMLs — verify against
`_datafiles/world/dogmud/factions/np_dockfolk.yaml` before writing; copy its shape.)

- [ ] **Step 2: Boot-verify the faction loads**

Boot-test recipe. Expected: no panic; grep the log for the faction-load count line
and confirm it incremented (e.g. `factions … count=16` → now one higher).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/factions/cooperage_circle.yaml
git commit -m "feat(np-crafting): activate the cooperage_circle faction"
```

---

## Task 4: Stage C — the 7 anchors (mobs 9332–9338) + dialogue

**Files:**
- Create: `_datafiles/world/dogmud/mobs/new_plymouth_crafting/9332-*.yaml` …
  `9338-*.yaml`
- Create: `_datafiles/world/dogmud/dialogue/.../9326..9332.yaml` (per the loader's
  dialogue path convention — verify with an existing NP dialogue file)

- [ ] **Step 1: Author the 7 anchors (sequential, one subagent, model: sonnet)**

Hand the Mob Manifest rows 9332–9338 + Conventions. Each mob:
- Title-Case `name`; mutation in `description`; `non_combatant: true`,
  `hostile: false`, `maxwander: 0`; `groups:` per the manifest (cooperage_circle
  members: Orin 9332, Edda 9335, Toby 9338).
- Shop-bearers (all but Toby) carry `craft_support:` per the manifest and a
  `shop:` block of trade-appropriate items (reuse existing item IDs — iron/steel
  for Halvard, reagents/potions for Vesna, glass/general for Edda, cloth/thread
  for Nessa, leather for Corwin, books/maps for Orin). **Filename** must match
  `ConvertForFilename(name)` (e.g. `9333-master_halvard.yaml`).
- Each anchor references its dialogue file; each dialogue has ≥3 discoverable
  topics with `triggers`, NPC text in **first person**, hints in narrator voice
  (SOP). Quest-free this build — no `grantsQuest`.

- [ ] **Step 2: Verify mob filenames match `Filepath()`**

```bash
ls _datafiles/world/dogmud/mobs/new_plymouth_crafting/
```
Confirm each filename is `<id>-<convertforfilename(name)>.yaml` (a mismatch panics
at boot — gotcha re: `Filepath()`).

- [ ] **Step 3: Boot-verify** — boot-test recipe. Expect no panic;
  `mobs.LoadDataFiles()` count incremented by 7; `ValidateShopMobTags` passes
  (no panic — every shop mob has a `craft_support:`).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/new_plymouth_crafting/933*.yaml _datafiles/world/dogmud/dialogue/
git commit -m "feat(np-crafting): the 7 anchor artisans + dialogue (9332-9338)"
```

---

## Task 5: Stage C — ambient mobs (9339–9343) + room spawns

**Files:**
- Create: `mobs/new_plymouth_crafting/9339-*.yaml` … `9343-*.yaml`
- Modify: the room YAMLs to add `spawninfo:` for anchors + ambient in their rooms

- [ ] **Step 1: Author 5 ambient mobs** (manifest rows 9339–9343) — simple
  `non_combatant` townsfolk with 2–3 `idlecommands` each, no shop, no dialogue.

- [ ] **Step 2: Add spawn entries**

For each work/ambient room, add the mob to the room's `spawninfo:` so it spawns on
load (match the spawn shape used in Docks/Common rooms — verify one existing NP
room with a `spawninfo:` block first). Anchors spawn in their work rooms; ambient
in their listed rooms.

- [ ] **Step 3: Boot-verify** — boot-test recipe; mobs count +5; confirm (via a
  quick in-game `look` or the spawn log) that anchors appear in their rooms.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/new_plymouth_crafting/934*.yaml \
        _datafiles/world/dogmud/rooms/new_plymouth_crafting/
git commit -m "feat(np-crafting): ambient residents (9339-9343) + room spawns"
```

---

## Task 6: Stage D — the Bloom Trail middle (Vesna)

**Files:**
- Modify: Vesna's dialogue (9328) + room 5711 (Vesna's lab) nouns

- [ ] **Step 1: Author the breadcrumb (content-only — no mechanic)**

In Vesna's dialogue add a topic (discoverable trigger, e.g. `buyer`/`order`) where
she — first person — frets about a standing order from a buyer who "pays too well
and asks no questions," and that her oil-of-something keeps going *down the canal*.
Add a room noun in 5711 (a `ledger`/`standing-order slip`) that points the player
onward toward the Old Quarter / the canal. **Seeds, does not resolve** — no quest,
no item transfer; the climax (Deren, Old Quarter) is a later build. Keep it
consistent with the Docks breadcrumbs (Marn's back room, the Bloom-addled wanderer)
and the city-wide Bloom Trail web.

- [ ] **Step 2: Boot-verify** — boot-test recipe; no panic; dialogue still loads.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/dialogue/ _datafiles/world/dogmud/rooms/new_plymouth_crafting/5711.yaml
git commit -m "feat(np-crafting): Bloom Trail middle — Vesna's uneasy standing order"
```

---

## Task 7: Stage D — anchor schedules

**Files:**
- Create: `_datafiles/world/dogmud/schedules/new_plymouth_crafting/np_crafting_*.yaml`
  (one per scheduled anchor)
- Modify: each anchor mob YAML to add `schedule_id:`

- [ ] **Step 1: Author schedules**

One `schedule_id` per anchor, covering all 24h (validators PANIC on coverage gaps
/ unreachable targets). Pattern: home(`up`-room) → work(`activity: working`) →
**midday meal** (Craftwell Square 5722 or the cookshop) → evening (social) →
sleep(`activity: sleeping`). Route **only** via compass/`up`/`down` exits (gotcha
#8) — every `pathto` target must be reachable.

**Signature beat — Halvard's midday meal:** schedule Halvard (9333) to walk to the
**Common cookshop** at midday to eat with his sister Marda, then back. The full
path Crafting → footbridge `south` to Docks 5523 → … → Common cookshop must be
`pathto`-traversable. **If** the cross-district route is not fully reachable
(validator fails), fall back to a Craftwell-Square local meal node for Halvard and
keep the sibling bond in dialogue — do **not** block the build on it. Reference
the docks/common schedules under `schedules/new_plymouth_docks|common/` for the
exact YAML shape.

- [ ] **Step 2: Add `schedule_id:` to each anchor mob** (e.g. Halvard →
  `schedule_id: np_crafting_halvard`).

- [ ] **Step 3: Boot-verify the schedule validators**

Boot-test recipe; additionally grep:
```bash
grep -nE "LoadSchedules|schedule.*coverage|schedule.*unreachable|unresolved schedule" /tmp/np_boot.log
```
Expected: `LoadSchedules` count incremented by the number of new schedules; **no**
coverage-gap / unreachable / unresolved panics.

- [ ] **Step 4: In-game spot-check** — boot, jump to a couple of anchor rooms at
  different game-hours (or read the executor log), confirm anchors move/sleep per
  segment. Confirm Halvard's midday route fires (or the documented fallback).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/schedules/new_plymouth_crafting/ \
        _datafiles/world/dogmud/mobs/new_plymouth_crafting/933*.yaml
git commit -m "feat(np-crafting): anchor schedules — the quarter's daily routines"
```

---

## Task 8: Vendor `StockEntry` pre-declaration (delivery-ready for Plan 2)

**Files:**
- Modify: the 7 shop mobs' shop blocks (Orin 9332, Halvard 9333, Vesna 9334, Edda
  9335, Nessa 9336, Corwin 9337, the 5703 general stall) — or wherever shop stock
  is declared for these vendors.

- [ ] **Step 1: Pre-declare deliverable feedstock as StockEntries**

For each Crafting vendor, ensure its shop declares the **base/overlap feedstock
items** Dobb will deliver (Plan 2 §5.3) as stock entries with `Current: 0`,
`MaxStock: <small buffer, e.g. 8>`, `RestockQty: 0`. Map by trade: Halvard → iron
ingot 40001 / steel ingot 40018 / coal dust 40020; Vesna → glass vial 40006 /
healer's root 40004 / bitter thistle 40005; Nessa → thread spool 40012 / cloth
strip 40007; Corwin → leather strip 40002; Edda → glass vial 40006; the 5703 stall
→ a couple of base items. (`VisitVendorsInRoom` silently skips items with no
existing entry; `RestockQty:0` prices sanely thanks to the shipped
`DefaultPricingBaselineQty` fix.) Verify the exact StockEntry YAML shape against an
existing Docks vendor's persisted shop / mob shop block first.

- [ ] **Step 2: Boot-verify** — boot-test recipe; no panic; the vendors load with
  the new (empty) stock entries. (They will read as out-of-stock until Plan 2's
  runner delivers — that's expected.)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mobs/new_plymouth_crafting/933*.yaml
git commit -m "feat(np-crafting): vendors pre-declare deliverable stock (runner-ready)"
```

---

## Task 9: District harness playtest

**Files:** none (produces a report under `tools/playtest/reports/`, gitignored).

- [ ] **Step 1: Run a feature-tester playtest**

```bash
/playtest local feature-tester
```
Drive the tester through the Crafting Quarter from the Docks footbridge: visit each
anchor, `ask <npc> about <topic>` for ≥1 topic each, buy from a vendor, read the
Inkwalk/cooperage lore nouns, follow Vesna's Bloom breadcrumb, and observe an
anchor on schedule (moving or asleep). Capture the report path.

- [ ] **Step 2: Triage findings**

Log any defects. Fix blocking/cosmetic issues inline (commit as
`fix(np-crafting): …`); record genuinely-deferred polish in the build memory.

- [ ] **Step 3: Final boot test** — boot-test recipe one more time; confirm clean.

---

## Task 10: Merge to master (hold push)

- [ ] **Step 1: Merge `--no-ff`**

```bash
git checkout master
git merge --no-ff feature/np-crafting-district -m "Merge: New Plymouth Crafting Quarter (district 3/7 — content)"
```

- [ ] **Step 2: Update build memory**

Update `project_new_plymouth_build.md` + `MEMORY.md` STATUS: Crafting district
content COMPLETE & merged; NEXT = Plan 2 (the supply runner engine), then district
4 = Merchant. **Do NOT push** (push policy: hold until the whole capital is built).

---

## Self-Review (completed during planning)

- **Spec coverage:** §1 zone/IDs → Task 0 + manifests; §2 layout/footbridge/stubs
  → Tasks 1–2; §3 anchors → Task 4; §4 faction → Task 3; §6 Bloom middle → Task 6;
  §7 schedules → Task 7; §8 Plan-1 staging A–D → Tasks 1–9; vendor delivery-readiness
  (§5.3.6) → Task 8. The supply *engine* (§5) is intentionally Plan 2 — not here.
- **Placeholder scan:** every task names exact files/IDs and the verification
  command; subagent-authored YAML steps give the manifest + constraints rather than
  literal 25× file bodies (the project's established content-authoring pattern),
  which is the correct granularity for `/new-room`/`/new-mob` work — not a placeholder.
- **Consistency:** room IDs 5700–5724, mob IDs 9332–9343, dialogue 9326–9332, and
  faction `cooperage_circle` are used identically across the manifests and tasks.
