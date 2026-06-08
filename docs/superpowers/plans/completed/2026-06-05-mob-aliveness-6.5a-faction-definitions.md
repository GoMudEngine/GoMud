# Mob Aliveness 6.5a — Faction Definitions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author 8 new world factions on the 1.2/1.3 substrate (bandits, ironwind_tribe, dustwalk_caravans, road_wardens, shopkeepers, ashwick_villagers, watchers_crossing, bloodline_agents), wire the unified law-bloc-vs-outlaw ally/enemy graph, correct the warren miscategorization, and tag member mobs — so the 6.5 content batches tag against a settled roster.

**Architecture:** Pure YAML data authoring. Faction definitions live at `_datafiles/world/dogmud/factions/{faction_id}.yaml`; membership is the `faction_id` string appearing in a mob's `groups:` list. The faction loader (`internal/factions/registry.go`) **panics at boot on any unresolved ally/enemy reference**, so the boot test is the validation gate — there is no bespoke Go unit test for faction content (matches the existing codebase pattern; the loader IS the test harness).

**Tech Stack:** YAML data files, the `internal/factions` loader.

**Spec:** `docs/superpowers/specs/2026-06-05-mob-aliveness-6.5a-faction-definitions-design.md`

---

## Reference: the faction schema (verified against `internal/factions/types.go`)

```yaml
faction_id: <slug>            # must match the filename base
display_name: "<Display>"
description: |
  <in-character description, ~80-char wrapped>
default_rep: <int>           # player's starting reputation
allies: [<faction_id>, ...]  # both-sided edges (loader does NOT auto-mirror)
enemies: [<faction_id>, ...] # loader PANICS if any ref is an unknown faction
# guard/justice factions only:
# holding_cell_room: <roomId>
# release_room: <roomId>
```

## Reference: the graph (complete, internally consistent)

**Law bloc (9 factions, mutually allied clique):** thornwall_guards,
stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens,
dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing.
Each lists the **other eight** in `allies:`.

- **Enforcers** (thornwall_guards, stillwater_guards, road_wardens):
  `enemies: [bandits, ironwind_tribe]`.
- **Non-enforcers** (the other six law-bloc): `enemies: []` (they don't initiate;
  the outlaw side declares them as enemies so bandits/goblins are hostile to them).

**Outlaw cluster:** bandits + ironwind_tribe, allied to each other, each with
`enemies:` = all 9 law-bloc factions.

**Neutral:** bloodline_agents (`allies: []`, `enemies: []`).

**warren correction:** remove the `warren ↔ thornwall_guards` enemy edge entirely;
warren becomes `allies: []`, `enemies: []`, `default_rep: -25` unchanged.

---

## Task 1: Create the 8 new faction definition YAMLs

**Files:**
- Create: `_datafiles/world/dogmud/factions/road_wardens.yaml`
- Create: `_datafiles/world/dogmud/factions/dustwalk_caravans.yaml`
- Create: `_datafiles/world/dogmud/factions/shopkeepers.yaml`
- Create: `_datafiles/world/dogmud/factions/ashwick_villagers.yaml`
- Create: `_datafiles/world/dogmud/factions/watchers_crossing.yaml`
- Create: `_datafiles/world/dogmud/factions/bandits.yaml`
- Create: `_datafiles/world/dogmud/factions/ironwind_tribe.yaml`
- Create: `_datafiles/world/dogmud/factions/bloodline_agents.yaml`

- [ ] **Step 1: Write `road_wardens.yaml`**

```yaml
faction_id: road_wardens
display_name: "Road Wardens"
description: |
  Sworn keepers of the trade roads between the towns. They escort
  caravans, patrol the waystations, and run down the bandits that
  prey on travelers. Allied with the town guards and the merchant
  trains they protect.
default_rep: 0
allies: [thornwall_guards, stillwater_guards, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
enemies: [bandits, ironwind_tribe]
```

- [ ] **Step 2: Write `dustwalk_caravans.yaml`**

```yaml
faction_id: dustwalk_caravans
display_name: "Dustwalk Caravans"
description: |
  The merchant trains that haul goods along the Dustwalk Road and
  beyond. Wary of strangers but quick to befriend those who help
  keep the roads safe. They lean on the road wardens for protection
  and trade with every town.
default_rep: 10
allies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, shopkeepers, ashwick_villagers, watchers_crossing]
enemies: []
```

- [ ] **Step 3: Write `shopkeepers.yaml`**

```yaml
faction_id: shopkeepers
display_name: "Merchants' Concord"
description: |
  An informal chamber of commerce binding the towns' shopkeepers,
  crafters, and traders. They look out for one another -- cheat or
  rob one merchant and word travels fast through the Concord.
default_rep: 0
allies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, ashwick_villagers, watchers_crossing]
enemies: []
```

- [ ] **Step 4: Write `ashwick_villagers.yaml`**

```yaml
faction_id: ashwick_villagers
display_name: "Ashwick Villagers"
description: |
  The farmers, herbalists, and faithful of the little village of
  Ashwick. They tend their fields and chapel and look out for their
  own. They keep no guards and rely on the wardens and town watches
  when trouble finds them.
default_rep: 0
allies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, watchers_crossing]
enemies: []
```

- [ ] **Step 5: Write `watchers_crossing.yaml`**

```yaml
faction_id: watchers_crossing
display_name: "Watcher's Crossing"
description: |
  The innkeeper, traders, and toll-keeper of the waystation at
  Watcher's Crossing. A small but vital stop on the trade roads,
  bound to the wider web of guards, wardens, and merchants.
default_rep: 0
allies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers]
enemies: []
```

- [ ] **Step 6: Write `bandits.yaml`**

```yaml
faction_id: bandits
display_name: "Road Bandits"
description: |
  Brigands and highwaymen who prey on the trade roads and lonely
  stretches of wilderness. They answer to no law and rob whoever
  they can. Hunted by the wardens and guards; loosely thrown in
  with the steppe tribes when it suits them.
default_rep: -35
allies: [ironwind_tribe]
enemies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
```

- [ ] **Step 7: Write `ironwind_tribe.yaml`**

```yaml
faction_id: ironwind_tribe
display_name: "Ironwind Tribe"
description: |
  The goblin tribe of the Ironwind Steppe, led by their shaman.
  Fiercely territorial and hostile to the settled folk who push
  into the grasslands. They share no love for civilization and
  raid alongside the bandits when their paths cross.
default_rep: -25
allies: [bandits]
enemies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
```

- [ ] **Step 8: Write `bloodline_agents.yaml`**

```yaml
faction_id: bloodline_agents
display_name: "Bloodline Agents"
description: |
  Shadowy operatives moving quietly through the world on errands
  whose purpose few can guess. For now they keep to themselves --
  but those who watch closely sense a larger design taking shape.
default_rep: 0
allies: []
enemies: []
```

- [ ] **Step 9: Commit**

```bash
git add _datafiles/world/dogmud/factions/road_wardens.yaml _datafiles/world/dogmud/factions/dustwalk_caravans.yaml _datafiles/world/dogmud/factions/shopkeepers.yaml _datafiles/world/dogmud/factions/ashwick_villagers.yaml _datafiles/world/dogmud/factions/watchers_crossing.yaml _datafiles/world/dogmud/factions/bandits.yaml _datafiles/world/dogmud/factions/ironwind_tribe.yaml _datafiles/world/dogmud/factions/bloodline_agents.yaml
git commit -m "feat(factions): 6.5a — add 8 world factions (definitions + graph)"
```

---

## Task 2: Update the 5 existing faction YAMLs (law-bloc clique + warren correction)

**Files:**
- Modify: `_datafiles/world/dogmud/factions/thornwall_guards.yaml`
- Modify: `_datafiles/world/dogmud/factions/stillwater_guards.yaml`
- Modify: `_datafiles/world/dogmud/factions/thornwall_citizens.yaml`
- Modify: `_datafiles/world/dogmud/factions/stillwater_citizens.yaml`
- Modify: `_datafiles/world/dogmud/factions/warren.yaml`

Edit ONLY the `allies:` and `enemies:` lines in each. Preserve every other field
(description, default_rep, and the guards' `holding_cell_room`/`release_room`).

- [ ] **Step 1: `thornwall_guards.yaml`** — set:

```yaml
allies: [stillwater_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
enemies: [bandits, ironwind_tribe]
```
(Was `allies: [thornwall_citizens]`, `enemies: [warren]` — warren removed, outlaws added, clique allies added. Keep `holding_cell_room`/`release_room`.)

- [ ] **Step 2: `stillwater_guards.yaml`** — set:

```yaml
allies: [thornwall_guards, road_wardens, thornwall_citizens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
enemies: [bandits, ironwind_tribe]
```
(Was `allies: [stillwater_citizens]`, `enemies: []`. Keep any existing `holding_cell_room`/`release_room`.)

- [ ] **Step 3: `thornwall_citizens.yaml`** — set:

```yaml
allies: [thornwall_guards, stillwater_guards, road_wardens, stillwater_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
enemies: []
```
(Was `allies: [thornwall_guards]`.)

- [ ] **Step 4: `stillwater_citizens.yaml`** — set:

```yaml
allies: [thornwall_guards, stillwater_guards, road_wardens, thornwall_citizens, dustwalk_caravans, shopkeepers, ashwick_villagers, watchers_crossing]
enemies: []
```
(Was `allies: [stillwater_guards]`.)

- [ ] **Step 5: `warren.yaml`** — set (the correction):

```yaml
allies: []
enemies: []
```
(Was `enemies: [thornwall_guards]` — removed. `default_rep: -25` and description unchanged. Optionally tweak the description to read as insular/mistrusted rather than hostile, but that's flavor, not required.)

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/factions/thornwall_guards.yaml _datafiles/world/dogmud/factions/stillwater_guards.yaml _datafiles/world/dogmud/factions/thornwall_citizens.yaml _datafiles/world/dogmud/factions/stillwater_citizens.yaml _datafiles/world/dogmud/factions/warren.yaml
git commit -m "feat(factions): 6.5a — wire law-bloc clique + remove warren guard-enemy edge"
```

---

## Task 3: Boot-test the faction graph (validation checkpoint)

The faction loader panics on any unresolved ally/enemy reference. A clean boot
proves the 13-faction graph is internally consistent. No members tagged yet.

- [ ] **Step 1: Build**

Run: `go build ./...`
Expected: clean (no code changed, but confirms tree builds).

- [ ] **Step 2: Wipe instance saves (smoke SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 3: Boot and confirm no faction panic**

Build a binary and run it (e.g. `go build -o GoMud.exe . && ./GoMud.exe`), or
`make run`. Watch startup for the faction load and confirm **no panic** of the
form `factions: faction "X" lists unknown ally/enemy "Y"`. Confirm the server
reaches "Server Ready".
Expected: clean boot past data-file load. If it panics on an unknown ally/enemy,
a `faction_id` is misspelled in an allies/enemies list — fix and re-boot.

- [ ] **Step 4: (optional) confirm faction count via admin**

Connect as an admin (`smoketester`) and run `faction list`. Expect all 13
factions listed (5 existing + 8 new). Run `faction show bandits` and
`faction show road_wardens` to eyeball the graph.

No commit (verification only; Tasks 1–2 already committed).

---

## Task 4: Tag bounded-faction member mobs

Each mob already has a `groups:` YAML list (e.g. `- humanoid`). **Append** the
`faction_id` as a new list item (existing non-faction tags like `bandit`,
`merchant`, `warden` are ignored by `FactionsForMob`; only registered faction_ids
count). Pattern, e.g. for `dustwalk_road/80-dustwalk_bandit.yaml`:

```yaml
# before
groups:
  - bandit
  - humanoid
# after
groups:
  - bandit
  - humanoid
  - bandits
```

- [ ] **Step 1: Tag `bandits` members** — append `- bandits` to `groups:` in each:
  - `_datafiles/world/dogmud/mobs/dustwalk_road/80-dustwalk_bandit.yaml`
  - `_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml`
  - `_datafiles/world/dogmud/mobs/thornwall_city/105-thornwall_thug.yaml`
  - `_datafiles/world/dogmud/mobs/marches_spur_road/253-road_bandit.yaml`
  - `_datafiles/world/dogmud/mobs/marches_spur_road/254-bandit_leader.yaml`
  - `_datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml`
  - `_datafiles/world/dogmud/mobs/north_road/284-bandit_fighter.yaml`
  - `_datafiles/world/dogmud/mobs/north_road/285-bandit_caster.yaml`
  - `_datafiles/world/dogmud/mobs/north_road/286-soren.yaml`

- [ ] **Step 2: Tag `ironwind_tribe` members** — append `- ironwind_tribe`:
  - `_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml`
  - `_datafiles/world/dogmud/mobs/ironwind_steppe/218-goblin_scrapper.yaml`
  - `_datafiles/world/dogmud/mobs/ironwind_steppe/219-goblin_shaman.yaml`
  - `_datafiles/world/dogmud/mobs/ironwind_steppe/222-goblin_sentry.yaml`
  - (Do NOT tag `13-loot_goblin` (special) or `68-cave_goblin_guard` (sanctum area).)

- [ ] **Step 3: Tag `road_wardens` members** — append `- road_wardens`:
  - `_datafiles/world/dogmud/mobs/dustwalk_road/83-road_warden_tessara.yaml`
  - (Do NOT tag `241-windwarden_sylara` — quest boss. If a per-zone scan finds
    other escort/warden road mobs, tag them and note them in the report.)

- [ ] **Step 4: Tag `dustwalk_caravans` members** — append `- dustwalk_caravans`:
  - `_datafiles/world/dogmud/mobs/north_road/281-caravan_master.yaml`
  - Then scan for caravan crew: `grep -rl "caravan" _datafiles/world/dogmud/mobs/*/` and for any mob whose `groups:` actually contains a `caravan` crew tag (NOT just a description mention — verify the `groups:` block), append `- dustwalk_caravans`. List any tagged in the report.

- [ ] **Step 5: Tag `bloodline_agents` member** — append `- bloodline_agents`:
  - `_datafiles/world/dogmud/mobs/north_road/287-bloodline_agent.yaml`

- [ ] **Step 6: Tag `ashwick_villagers` members** — append `- ashwick_villagers`:
  - `_datafiles/world/dogmud/mobs/ashwick/259-delia.yaml`
  - `_datafiles/world/dogmud/mobs/ashwick/260-deacon_ferris.yaml`
  - `_datafiles/world/dogmud/mobs/ashwick/261-farmer_hesta.yaml`
  - `_datafiles/world/dogmud/mobs/ashwick/262-the_forager.yaml`

- [ ] **Step 7: Tag `watchers_crossing` members** — append `- watchers_crossing`:
  - `_datafiles/world/dogmud/mobs/watchers_crossing/84-innkeeper_tolva.yaml`
  - `_datafiles/world/dogmud/mobs/watchers_crossing/86-toll_collector_harn.yaml`
  - `_datafiles/world/dogmud/mobs/watchers_crossing/85-merchant_brecca.yaml` — append BOTH `- watchers_crossing` AND `- shopkeepers` (dual membership).
  - `_datafiles/world/dogmud/mobs/watchers_crossing/88-traveling_merchant.yaml` — append BOTH `- watchers_crossing` AND `- shopkeepers`.

- [ ] **Step 8: thornwall_outskirts light tags** (not a town — faction-tag only):
  - `_datafiles/world/dogmud/mobs/thornwall_outskirts/89-farmer_dorn.yaml` — append `- thornwall_citizens`.
  - `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml` — append `- thornwall_guards`.
  - (`90-thornwall_highwayman` already tagged `bandits` in Step 1; `91-crop_pest` is wildlife — skip.)

- [ ] **Step 9: Commit**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "feat(factions): 6.5a — tag bounded-faction member mobs"
```

---

## Task 5: Tag shopkeepers (scan-driven, world-wide)

Shopkeepers = every mob that owns a shop, EXCEPT in `sanctum_basin` (excluded —
newbie-area rework supersedes it). Town merchants get **dual membership** (their
existing town-citizen tag stays; append `shopkeepers`).

- [ ] **Step 1: Enumerate shop-owning mobs**

Run:
```bash
grep -rl "^  shop:" _datafiles/world/dogmud/mobs/ | grep -v "/sanctum_basin/"
```
(Shop-bearing mobs declare a `shop:` block — e.g. `337-smith_brindle.yaml` has
`behavior_archetype: noncombat_shopkeeper` + a `shop:` block. Adjust the grep to
the actual indentation if needed; cross-check with
`grep -rl "noncombat_shopkeeper" _datafiles/world/dogmud/mobs/ | grep -v sanctum_basin`.)
Known anchors that MUST appear: stillwater `336`–`341`, `north_road/278-haral`,
the Thornwall city merchants, and watchers `85`/`88` (already dual-tagged in
Task 4 Step 7 — do not double-add).

- [ ] **Step 2: Append `- shopkeepers` to each shop-owner's `groups:` list**

For every file from Step 1 (minus the two watchers merchants already done),
append `- shopkeepers` to its `groups:` list, preserving existing groups
(including any `thornwall_citizens`/`stillwater_citizens` tag — dual membership is
intended). Skip non-shop "merchant-flavored" mobs that have no `shop:` block
(e.g. `thornwall_city/101-street_performer` is a performer, not a shop — exclude).

- [ ] **Step 3: Report the final shopkeeper list**

In your report, list every mob tagged `shopkeepers` (path + id) so the reviewer
can verify completeness and that no sanctum mob slipped in.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "feat(factions): 6.5a — tag shop-owning mobs into shopkeepers (world-wide, ex-sanctum)"
```

---

## Task 6: Boot-test + faction smoke (members live)

- [ ] **Step 1: Build + full test**

Run: `go build ./... && go test ./...`
Expected: build clean; all packages pass (no Go changed, but confirms nothing
broke).

- [ ] **Step 2: Wipe instances + boot**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot the server; confirm clean load past data files with **no panic** (faction
ref-integrity + mob load).

- [ ] **Step 3: In-game faction smoke** (admin `smoketester`)

- `faction list` → 13 factions.
- `faction show bandits` → enemies = the 9 law-bloc factions; allies = [ironwind_tribe].
- Locate a bandit mob (e.g. dustwalk_road) and confirm `FactionsForMob` resolves
  it: attacking it should apply a `bandits` rep change (and, per the law-bloc
  graph, ripple to allied factions). Confirm a road_warden mob reads as
  `road_wardens` and treats the player per the bandits/ironwind enemy edges.
- Confirm warren members no longer read as enemies of thornwall_guards.

(In-game smoke MAY be deferred to user verification per the chunk 2.8/2.9
precedent; the boot-test panic-gate is the hard requirement. Note in the report
which smoke steps were run vs deferred.)

No commit (verification only).

---

## Task 7: Update context.md + roadmap

**Files:**
- Modify: `internal/factions/context.md`
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Update `internal/factions/context.md`**

If it documents the authored faction list / count / graph, update it to reflect
13 factions and the law-bloc-vs-outlaw structure (+ the warren correction). If it
only documents the code (not the content), add a one-line note that world faction
content lives in `_datafiles/world/dogmud/factions/` and was expanded in 6.5a.

- [ ] **Step 2: Update `MOB_ALIVENESS_ROADMAP.md`**

In the Progress tracker table, set the 6.5a row Status to `Done (2026-06-05)`
(add a 6.5a row if one isn't present yet). In the 6.5a mini-brief, set
`**Status:** Done` and add a `- **Shipped:**` bullet summarizing: 8 new factions,
the law-bloc/outlaw graph, warren correction, member tagging (bounded + scan-driven
shopkeepers), boot-validated. Re-tally the roll-up line.

- [ ] **Step 3: Commit**

```bash
git add internal/factions/context.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs: 6.5a faction definitions — context.md + roadmap"
```

---

## Self-review notes

**Spec coverage:**
- Spec "New factions (8)" → Task 1 (all 8 with exact content).
- Spec "law bloc clique" + "outlaw cluster" + "warren correction" → Tasks 1–2
  (graph authored both-sided; warren edge removed).
- Spec "member tagging" anchors → Task 4 (bounded) + Task 5 (shopkeepers scan).
- Spec "thornwall_outskirts light treatment" → Task 4 Step 8.
- Spec "validation" (boot panic gate + faction smoke) → Tasks 3 and 6.
- Spec "files touched: context.md" + roadmap maintenance → Task 7.

**No-placeholder check:** every faction YAML is given in full; every bounded
member has an exact path; the only scan-driven step (shopkeepers, Task 5) provides
the exact grep + inclusion rule + exclusions + a report-back requirement, because
world-wide shop enumeration is genuinely data-discovered (the spec anticipated
this).

**Graph consistency check:** every `faction_id` referenced in any `allies:`/
`enemies:` list above is one of the 13 (5 existing + 8 new) — so the loader's
panic-on-unknown-ref will pass. Law-bloc allies lists each contain exactly the
other 8 law-bloc ids; outlaw enemies lists each contain exactly the 9 law-bloc ids.

**TDD note:** faction content is validated by the boot-time loader (panic on bad
ref), not Go unit tests — consistent with the codebase (no existing faction-content
tests). Tasks 3 and 6 are the test gates.
