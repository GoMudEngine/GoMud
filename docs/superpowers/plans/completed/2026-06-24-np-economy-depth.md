# New Plymouth — Economy Depth Pass Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 3 vendors + ~6 new flavor items to the New Plymouth capital — a Common
general-goods seller (mob 9390 @ room 5604), a Temple offerings seller (mob 9391 @
room 5903), and converting the existing Flower Seller (mob 9326) into a real
micro-vendor — to fix the thin-vendor gap found in the 2026-06-24 capital playtest.

**Architecture:** Pure content (YAML data files): ~6 new `type: object` items, 2 new
shopkeeper mobs + dialogue + light schedules, and an edit to one existing mob. No new
faction, no Go change, no new rooms. Validated by server boot +
`ValidateShopMobTags` + schedule validators + a live buy/sell smoke test. The
controller (main loop) drives all shell — subagents Write/Edit YAML only.

**Tech Stack:** GoMud engine (run locally), DOGMud world YAML (`docs/schemas/`),
`tools/id_inventory.py`, the `/playtest` harness (or manual telnet) for the smoke.

**Spec:** `docs/superpowers/specs/2026-06-24-np-economy-depth-design.md`.

---

## Conventions for every task (READ FIRST)

- **Branch:** `feature/np-economy-depth` (from `master` in Task 0). Commit per stage;
  trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Controller drives all shell** (subagents Write/Edit YAML only).
- **Boot test = verification:**
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . > /tmp/econ_boot.log 2>&1 &
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath|panic:|ValidateShopMob" /tmp/econ_boot.log   # expect none
  grep -nE "items.LoadDataFiles|mobs.LoadDataFiles|LoadSchedules|Server Ready" /tmp/econ_boot.log
  ```
  NOT bare `panic` (gotcha #8). Kill after (`taskkill //F //IM go.exe //T`). Do NOT
  wipe `shops/` (living-economy state).
- **Authoring gotchas:** (1) `": "` in a prose value → `" — "`. (2) Title-Case mob
  `name:` (`AssertCanonical` panics). (3) faction via `groups:` (existing factions
  only: np_commonfolk, temple_np). (4) shop-bearing mobs need `craft_support:`
  (`general`) or `ValidateShopMobTags` PANICS. (5) item filename =
  `{itemid}-{ConvertForFilename(name)}.yaml` in the correct type-folder. (6) dialogue
  `text` first-person, `hints` narrator 2nd-person, triggers discoverable, NO quest
  fields. (7) names collision-checked vs the world roster.
- **Reference files** (READ before authoring):
  - Item: `_datafiles/world/dogmud/items/materials-40000/40077-tallow_candle.yaml`
    (a `type: object, subtype: mundane` item — the exact template for the new goods).
  - Shopkeeper mob + shop block:
    `_datafiles/world/dogmud/mobs/new_plymouth_crafting/9333-master_halvard.yaml`
    (`craft_support:` is TOP-LEVEL; the `shop:` list is nested UNDER `character:`,
    a sibling of `stats:`/`gold:`).
  - Shopkeeper dialogue:
    `_datafiles/world/dogmud/dialogue/new_plymouth_crafting/9333.yaml`.
  - Light 2-segment schedule:
    `_datafiles/world/dogmud/schedules/new_plymouth_docks/np_docks_marn.yaml`.
  - Flower Seller (to edit):
    `_datafiles/world/dogmud/mobs/new_plymouth_common/9326-flower_seller.yaml`.

---

## ID allocation (pre-assigned)

| Kind | Range | Assignment |
|------|-------|------------|
| Items | **40102–40107** (6) | see Item Manifest |
| Mobs | **9390** (Common general), **9391** (Temple offerings) | + EDIT existing 9326 |
| Dialogue | by mobid (9390, 9391; light add to 9326 if absent) | |
| Faction | none created (membership only) | np_commonfolk / temple_np |

---

## Item Manifest (6 — all `type: object`, `subtype: mundane`, in `items/materials-40000/`)

Each item: `itemid`, `name` (Title-Case), `namesimple` (lowercase short target word),
a 2–4 line `description:` (no `": "`), `weight`, `value`, and
`vendor_categories:\n- general` (so general-craft_support vendors can stock it). Model
exactly on `40077-tallow_candle.yaml`.

| itemid | name | namesimple | value | weight | sold by |
|--------|------|-----------|-------|--------|---------|
| 40102 | Coil of Rope | rope | 3 | 1.0 | Common general (9390) |
| 40103 | Waterskin | waterskin | 2 | 0.5 | Common general (9390) |
| 40104 | Cake of Tallow Soap | soap | 1 | 0.2 | Common general (9390) |
| 40105 | Tinderbox | tinderbox | 2 | 0.3 | Common general (9390) |
| 40106 | Votive Candle | votive | 1 | 0.2 | Temple offerings (9391) |
| 40107 | Cut Flowers | flowers | 1 | 0.2 | Flower Seller (9326) |

(Filenames: `40102-coil_of_rope.yaml`, `40103-waterskin.yaml`,
`40104-cake_of_tallow_soap.yaml`, `40105-tinderbox.yaml`, `40106-votive_candle.yaml`,
`40107-cut_flowers.yaml` — all under `items/materials-40000/`.)

---

## Mob Manifest

| Mob | Name (suggested — collision-check + rename if taken) | Room | `groups` | `craft_support` | Shop stock (itemids) |
|-----|------|------|----------|-----------------|----------------------|
| 9390 | Mardle the Sundries-Seller | 5604 (Common Market) | humanoid, np_commonfolk | general | 40102, 40103, 40104, 40105, 40077 (Tallow Candle), 40076 (Sack of Flour) |
| 9391 | Almoner Hale | 5903 (Temple Gate Plaza) | humanoid, temple_np | general | 40106 (Votive Candle), 27 (Temple Incense), 40077 (Tallow Candle), 20089 (Folk Charm) |
| 9326 (edit) | Flower Seller (keep) | its current spawn | humanoid, np_commonfolk (unchanged) | general (ADD) | 40107 (Cut Flowers) |

Both new mobs: `behavior_archetype: noncombat_shopkeeper`, `non_combatant: true`,
`charm_immune: true`, `hostile: false`, `maxwander: 0`, a `statpool` (~28–32), a
`character:` block with name + a mutation-woven `description:` + modest `gold:` +
a couple of `stats:`, and the `shop:` list nested under `character:`.

---

## Task 0: Branch + ID sanity

- [ ] **Step 1: Branch** — `git checkout master && git checkout -b feature/np-economy-depth`.
- [ ] **Step 2: Confirm IDs free** — `python tools/id_inventory.py --type items | grep "next free"` (expect ≥ 40102); `--type mobs` next-free ≥ 9390.
- [ ] **Step 3: Verify reuse item IDs exist + are sellable** — confirm `27` (Temple Incense), `20089` (Folk Charm), `40076` (Sack of Flour), `40077` (Tallow Candle) each resolve to a real item file (`find _datafiles/world/dogmud/items -name "27-*.yaml" -o -name "20089-*.yaml" ...`). If any reuse item has `vendor_categories` that EXCLUDES general and that blocks stocking, note it (the new general items all get `general`; reuse items already stock fine via the shop list — the shop `- itemid:` list is authoritative for what a vendor sells).
- [ ] **Step 4: Baseline boot** — boot-test recipe; clean. Kill server.

## Task 1: Stage A — the 6 new flavor items

**Files:** Create `items/materials-40000/40102-coil_of_rope.yaml` … `40107-cut_flowers.yaml`.

- [ ] **Step 1:** Dispatch a content subagent (haiku or sonnet — mechanical) with the Item Manifest + the `40077-tallow_candle.yaml` template. Author all 6 items: `type: object`, `subtype: mundane`, the manifest `value`/`weight`, `namesimple`, `vendor_categories:\n- general`, and a 2–4 line evocative `description:` each (everyday New Plymouth flavor; the Votive Candle reads as a temple offering; Cut Flowers as a Carter's Rise posy). No `": "` in any value.
- [ ] **Step 2: Verify filenames + folder** — all 6 in `items/materials-40000/`, names match `ConvertForFilename`.
- [ ] **Step 3: Boot-verify** — `items.LoadDataFiles` count rises by 6; no `did not end in Filepath` panic.
- [ ] **Step 4: Commit** — `feat(np-economy): 6 flavor goods (40102-40107) — rope, waterskin, soap, tinderbox, votive, flowers`.

## Task 2: Stage B — Common general-goods vendor (9390)

**Files:** Create `mobs/new_plymouth_common/9390-*.yaml` + `dialogue/new_plymouth_common/9390.yaml`; Modify a Common Market room to spawn it.

- [ ] **Step 1: Pick + collision-check the name** — controller runs `grep -rho "name: .*" _datafiles/world/dogmud/mobs/ | sort -u | grep -i mardle` (and the chosen name); if "Mardle the Sundries-Seller" collides, pick another Title-Case name. Hand the final name to the subagent.
- [ ] **Step 2:** Dispatch a content subagent (sonnet) with the 9390 manifest row + Halvard reference. Author the mob: `behavior_archetype: noncombat_shopkeeper`, `non_combatant: true`, `charm_immune: true`, `hostile: false`, `maxwander: 0`, `groups: [humanoid, np_commonfolk]`, `craft_support: general` (TOP-LEVEL), a `character:` block (name, a 4–6 line description weaving in a **mild mundane mutation** — e.g. an unfailing memory for every face and price, or weather-reading joints — modest `gold:`, a `stats:` pair), and the `shop:` list **nested under `character:`**: itemids 40102, 40103, 40104, 40105, 40077, 40076. Dialogue (`9390.yaml`, model on `9333.yaml`): ≥2 discoverable first-person topics — (1) the wares/sundries, (2) the Common quarter/everyday life; optional (3) where the goods come from. NO quest fields.
- [ ] **Step 3: Add spawn** — edit `rooms/new_plymouth_common/5604.yaml` to add `spawninfo:` with `- mobid: 9390` `respawnrate: "10 real minutes"` (append; preserve existing content).
- [ ] **Step 4: Boot-verify** — `mobs.LoadDataFiles` +1; `ValidateShopMobTags` passes (craft_support general present); dialogue loads; spawn loads.
- [ ] **Step 5: Commit** — `feat(np-economy): Common general-goods vendor (9390) at the Common Market`.

## Task 3: Stage B — Temple offerings vendor (9391)

**Files:** Create `mobs/new_plymouth_temple/9391-*.yaml` + `dialogue/new_plymouth_temple/9391.yaml`; Modify the Temple Gate Plaza room to spawn it.

- [ ] **Step 1: Pick + collision-check the name** — as Task 2 (suggested "Almoner Hale"; verify not taken; NOTE "Quill" is already used by the Old Quarter lamplighter — do not reuse).
- [ ] **Step 2:** Dispatch a content subagent (sonnet) with the 9391 manifest row + Halvard reference. Author the mob: shopkeeper fields as Task 2 but `groups: [humanoid, temple_np]`, `craft_support: general`, a `character:` block with a 4–6 line description weaving in a **quiet devotional-flavored mutation** (e.g. hands that never tire of folding offerings, eyes that catch the altar-light), and the `shop:` list nested under `character:`: itemids 40106, 27, 40077, 20089. Dialogue (`9391.yaml`): ≥2 discoverable first-person topics — (1) the offerings / what to bring to the altar (incense, a votive, a charm), (2) the temple/the gate; optional (3) the devotional why. NO quest fields.
- [ ] **Step 3: Add spawn** — edit `rooms/new_plymouth_temple/5903.yaml` to add `spawninfo:` `- mobid: 9391` `respawnrate: "10 real minutes"`.
- [ ] **Step 4: Boot-verify** — `mobs.LoadDataFiles` +1; `ValidateShopMobTags` passes; dialogue + spawn load.
- [ ] **Step 5: Commit** — `feat(np-economy): Temple offerings vendor (9391) at the Temple Gate Plaza`.

## Task 4: Stage C — convert the Flower Seller (9326) into a vendor

**Files:** Modify `mobs/new_plymouth_common/9326-flower_seller.yaml`; (optional) Create `dialogue/new_plymouth_common/9326.yaml` if none exists.

- [ ] **Step 1: Edit the mob** — make three precise edits, preserving all existing content (idlecommands, description, stats):
  - Change `behavior_archetype: noncombat_passive` → `behavior_archetype: noncombat_shopkeeper`.
  - Add a TOP-LEVEL line `craft_support: general` (e.g. right after the `behavior_archetype` line).
  - Add a `shop:` list **nested under `character:`** (as a sibling of `stats:`), containing `- itemid: 40107`.
- [ ] **Step 2: Light shop dialogue (optional)** — if `dialogue/new_plymouth_common/9326.yaml` does not exist, create a SHORT one: a first-person flower-seller voice (the blooms, the morning sell-out — canon: "sells out before midday"), ≥1 discoverable topic (e.g. `flowers`, `blooms`). Keep it brief (micro-vendor). NO quest fields. If a dialogue already exists, leave it.
- [ ] **Step 3: Boot-verify** — `ValidateShopMobTags` passes for 9326 (now a shop mob with craft_support); no panic; the Flower Seller still spawns at its existing location.
- [ ] **Step 4: Commit** — `feat(np-economy): convert the Flower Seller (9326) into a cut-flowers micro-vendor`.

## Task 5: Stage D — light schedules for the two new vendors

**Files:** Create `schedules/new_plymouth_common/np_common_sundries.yaml` + `schedules/new_plymouth_temple/np_temple_offerings.yaml`; Modify mobs 9390 + 9391 to add `schedule_id:`.

- [ ] **Step 1:** Author two light 2-segment schedules (model on `np_docks_marn.yaml`; 24h coverage, no gaps, all `target_room`s within the vendor's district + pathto-reachable):
  - `np_common_sundries` (9390): day at the stall (target_room 5604, activity ""), night `activity: sleeping` (target_room 5604 or a nearby Common interior — use 5604 to stay trivially reachable). Two segments covering 0–24.
  - `np_temple_offerings` (9391): day at the gate (target_room 5903, activity ""), night `activity: sleeping` (target_room 5903). Two segments covering 0–24.
  Use NPC-voice idlecommands per segment (selling by day; quiet/closed at night).
- [ ] **Step 2:** Add `schedule_id: np_common_sundries` to mob 9390 and `schedule_id: np_temple_offerings` to mob 9391 (top-level, near `zone:`).
- [ ] **Step 3: Boot-verify** — `LoadSchedules` +2; no coverage-gap/unreachable panic.
- [ ] **Step 4: Commit** — `feat(np-economy): light day-stall schedules for the two new vendors`.

## Task 6: Stage E — buy/sell smoke test + merge

- [ ] **Step 1: Boot the server** (background) and connect via the playtest harness (`tools/playtest/.run` bridge, target local `smoketester`) OR confirm via the persisted shop-state files that the three vendors registered: `ls _datafiles/world/dogmud/shops/new_plymouth_common/ _datafiles/world/dogmud/shops/new_plymouth_temple/` should now include the 9390/9391/9326 room-keyed files.
- [ ] **Step 2: Live smoke (if driving the agent):** at each vendor — `list` (wares show with prices), `buy <item>` (gold decrements, item received), `sell <item>` (gold increments). Confirm dynamic pricing renders. Cover one item per vendor (e.g. `buy rope` at 9390, `buy votive` at 9391, `buy flowers` at 9326).
- [ ] **Step 3: Triage** — fix any blocking issue inline (`fix(np-economy): …`); log cosmetics. Kill server + harness.
- [ ] **Step 4: Final boot test** — clean (no panics, `ValidateShopMobTags` + schedules pass).
- [ ] **Step 5: Merge** — `git checkout master && git merge --no-ff feature/np-economy-depth -m "Merge: New Plymouth economy-depth pass (3 vendors + flavor goods)"`. **Do NOT push** (push stays HELD per user policy).
- [ ] **Step 6: Update memory** — note the economy-depth pass done + merged; NEXT enrichment axis = quests (per the synthesis ordering).

---

## Self-Review (completed during planning)

- **Spec coverage:** §0 scope (3 vendors, reuse+new items, money-sink deferred) →
  Tasks 1–5; §1 IDs/locations → Task 0 + manifests; §2 the 3 vendors → Tasks 2/3/4;
  §3 new items → Task 1; §4 conventions (groups/shop/craft_support/dynamic pricing/
  schedules/dialogue) → Tasks 2–5; §5 staging A–E → Tasks 1–6; §6 DoD → Task 6.
  Legendary-BIS + money-sink are explicitly OUT (deferred) — no task, correct.
- **Placeholder scan:** item bodies + mob bodies are subagent-authored from concrete
  manifests (named items with values, named itemids in shop lists); names are
  collision-checked at build (the established district pattern). No TBD/TODO.
- **Consistency:** items 40102–40107 used identically in the Item Manifest and the
  shop lists (9390 stocks 40102-40105+40077+40076; 9391 stocks 40106+27+40077+20089;
  9326 stocks 40107). Mobs 9390/9391 + edit 9326. craft_support `general` on all
  three shop mobs. Factions np_commonfolk/temple_np (existing). `shop:` nested under
  `character:` (verified in Halvard). No new faction, no Go change, no new rooms.
