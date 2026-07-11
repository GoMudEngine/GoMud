# New Plymouth — Merchant District Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the New Plymouth Merchant Quarter (`new_plymouth_merchant`, 25
rooms 5800–5824): the Central Square + 7 anchors (incl. Horst as a watched,
untouchable menace) + ambient, dialogue, the `bloodline_domestic` faction (+ the
`cooperage_circle` enemy backfill), anchor schedules, a supply-runner extension to
the Square, and Falk's Bloom Trail link — a complete, boot-clean, harness-tested
content district.

**Architecture:** Content (YAML data files) authored by content subagents within a
pre-allocated ID block, validated by server boot + `cartcheck` + schedule/patrol
validators + a harness playtest. The supply extension is data-only (extend the
existing `np_docks_runner_circuit` patrol YAML + import manifest from Plan
`2026-06-22-np-supply-runner.md`). The controller (main loop) drives all shell —
boot tests, `cartcheck`, commits — since subagents are shell-denied in this env.

**Tech Stack:** GoMud engine (run locally to boot-test), DOGMud world YAML
(`docs/schemas/`), `tools/id_inventory.py`, the `cartcheck` admin command, the
`/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-06-22-np-merchant-district-design.md`.

---

## Conventions for every task (READ FIRST — identical discipline to the Crafting build)

- **Branch:** `feature/np-merchant-district` (from `master` in Task 0). Commit per
  stage; trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **The controller drives all shell** (subagents are shell-denied here): authoring
  subagents only Write/Edit YAML; the controller runs boot/cartcheck/commit.
- **Boot test = the verification** (YAML errors panic at *startup*, not build):
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . > /tmp/np_boot.log 2>&1 &   # boots, loads data, idles
  # after the world-load lines:
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath" /tmp/np_boot.log   # expect none
  grep -nE "ValidateZoneConsistency|loadAllRoomZones|mobs.LoadDataFiles|factions.LoadAllDefiniti|LoadSchedules|LoadPatrols" /tmp/np_boot.log
  ```
  Do NOT grep bare `panic` (matches `MapConsistencyEnforce value=panic`, gotcha #8).
  Kill the server after (`taskkill //F //IM go.exe //T`).
- **Coordinates — solve against the LIVE frame.** The nominal city-wide regions are
  wrong; the Crafting/Common/Merchant junction is tight. Before Stage A, the
  controller dumps the real occupied cells (`grep coord:` across built NP zones) and
  assigns exact collision-free coords for each room, handing them to the room
  subagent. `cartcheck`/`ValidateZoneConsistency errors=0` is the gate.
- **Authoring gotchas (panic / render-breaking):** (1) `": "` inside a prose value
  breaks the parser → use `" — "`. (2) faction `allies/enemies` referencing a
  faction not built at all PANICS (forward-ref) — but a MUTUAL ref between two
  factions both built in this load is fine. (3) mob `name:` Title-Case
  (`casing.AssertCanonical`). (4) faction membership via `groups:`, not `faction:`.
  (5) no exits to unbuilt rooms (prose-stub instead). (6) shop mobs need a valid
  top-level `craft_support:` (cooking/general/blacksmithing/alchemy/tailoring/
  jewelcrafting/enchanting) or `ValidateShopMobTags` panics. (7) noun ansi MUST be
  `<ansi fg="itemname">noun</ansi>` — never `fg="<noun>"`; post-grep
  `fg="[^"]* [^"]*"` → expect zero. (8) interiors use compass/`up`/`down` only,
  never `enter`/`leave`.
- **Reference files to copy patterns from** (already-built, high quality):
  rooms `_datafiles/world/dogmud/rooms/new_plymouth_crafting/5702.yaml` (hub) +
  `5708.yaml` (lore); mob+dialogue `…/mobs/new_plymouth_crafting/9333-master_halvard.yaml`
  + `…/dialogue/new_plymouth_crafting/9333.yaml`; faction
  `…/factions/np_dockfolk.yaml`; schedule
  `…/schedules/new_plymouth_crafting/np_crafting_halvard.yaml`; patrol
  `…/patrols/new_plymouth/np_docks_runner_circuit.yaml`.

---

## ID allocation (pre-assigned — do not deviate)

| Kind | Range | Assignment |
|------|-------|------------|
| Rooms | **5800–5824** | see Room Manifest |
| Mobs | **9344–9355** | 7 anchors (9344–9350) + 5 ambient (9351–9355) |
| Dialogue | by mobid | one file per speaking mob |
| Faction | `bloodline_domestic` (create) + `cooperage_circle` (backfill) | |
| Zone folder | `new_plymouth_merchant` | `ConvertForFilename("New Plymouth Merchant")` |
| Items | 40106+ only if a new prop is genuinely needed | reuse existing where possible |
| Quests | none | |

---

## Room Manifest (25 — clusters + purpose + exits; coords solved at Stage A dispatch)

Entry: Crafting **5704** (The Long Market — East End) prose-stubs east → wire
**5704 `east` → the Merchant entry room** + reciprocal. The Central Square is the
hub; the Long Market runs E–W through it, the Processional N–S.

**Cluster 1 — the Central Square + arteries (6):**
| ID | Name | Purpose & exits |
|----|------|-----------------|
| 5800 | The Long Market — Merchant Gate | Western approach; entry from Crafting 5704 (`west`→5704). `east`→5801. |
| 5801 | The Central Square | The great market hub — the artery crossing; Dobb's circuit passes through; market crier + Square guard. Branches to all clusters. |
| 5802 | The Central Square — Market Stalls | The retail sprawl; ambient vendors/customers. Off 5801. |
| 5803 | The Processional — North End | Civic axis north. Prose-stub: continues north to the Noble Quarter (NO exit). Off 5801. |
| 5804 | The Processional — South Steps | Civic axis south. Prose-stub: descends south toward the Common Quarter (NO exit). Off 5801. |
| 5805 | The Long Market — East Reach | East side of the Square. Prose-stub: continues east toward the Temple (NO exit). Off 5801. |

**Cluster 2 — the financial row (4):**
| ID | Name | Purpose & exits |
|----|------|-----------------|
| 5806 | The Exchange | **Goss (9346)** — the counting house/moneylender. `up`→5807. |
| 5807 | The Counting Room | Goss's room above; ledgers, a strongbox. `down`→5806. |
| 5808 | Falk's Auction House | **Falk (9345)** — the block; high-end goods; the property-records **Bloom Trail link** noun. `up`→5809. |
| 5809 | Above the Auction House | Falk's room. `down`→5808. |

**Cluster 3 — the arms row (4):**
| ID | Name | Purpose & exits |
|----|------|-----------------|
| 5810 | Dame Ostry's Armoury | **Ostry (9347)** — weapon dealer (Extra-Arm mutation); blades from Halvard. A Dobb supply stop. `up`→5811. |
| 5811 | Above Ostry's | Ostry's room. `down`→5810. |
| 5812 | Brun's Plate-Works | **Brun (9348)** — armorer; plate from warehouse. A Dobb supply stop. `up`→5813. |
| 5813 | Above the Plate-Works | Brun's room. `down`→5812. |

**Cluster 4 — the Gilt Threshold (3):**
| ID | Name | Purpose & exits |
|----|------|-----------------|
| 5814 | The Gilt Threshold | **Madam Sephe (9350)** — the high inn common room. `up`→5816. |
| 5815 | The Gilt Threshold — Private Parlor | The room where Horst takes quiet meetings; **the overhear node** (a discoverable noun/scene — you catch a fragment of bloodline business). Off 5814. |
| 5816 | The Gilt Threshold — Lodging | Upstairs rooms. `down`→5814. |

**Cluster 5 — the bloodline apparatus (4):**
| ID | Name | Purpose & exits |
|----|------|-----------------|
| 5817 | The Permit Office | **Clerk Vell (9349)** — bloodline permits; the *felt* reach (tribute notices, a queue). `up`→5818 (or a back office). |
| 5818 | The Permit Office — Records | Vell's back records room; bloodline paperwork. `down`→5817. |
| 5819 | The Watched Lane | A guarded residential lane leading to Horst's house; a liveried watcher. `east`→5820. Off the Square. |
| 5820 | Horst's Rented House — Gate | **Horst (9344)** stands here (or in the Square). The door is locked/guarded — PROSE-STUB the interior (NO exit inward; you cannot enter/confront him). `west`→5819. |

**Cluster 6 — connective / civic / ambient (4):**
| ID | Name | Purpose & exits |
|----|------|-----------------|
| 5821 | The Square Fountain | A civic water node off 5801; midday-convergence; ambient. |
| 5822 | Coin Alley | A back alley behind the Exchange; a moneylender's clerk; ambient. |
| 5823 | The Porters' Dock | Goods-handling yard off the market; a porter; ambient. |
| 5824 | The Gilded Walk | A wealthy promenade; a well-dressed customer; ambient; connects the inn/financial clusters. |

(The subagent wires intra-cluster + cluster-to-Square exits per the controller's
exact coords; above-shop rooms use `up`/`down` stacks.)

---

## Mob Manifest (12 — 9344–9355)

| Mob | Name | Work room | Mutation | `craft_support` | `groups` | Notes |
|-----|------|-----------|----------|-----------------|----------|-------|
| 9344 | Horst | 5820 / the Square | none visible (uninfected) | — (**NO shop**, `non_combatant: true`) | humanoid, bloodline_domestic | watched/untouchable menace; runs agents; dialogue is guarded/cold, reveals nothing actionable |
| 9345 | Falk the Auctioneer | 5808 | never forgets a face or a price | `general` | humanoid | fences high-end goods; the **Bloom Trail link** (property records) |
| 9346 | Goss the Moneylender | 5806 | weighs gold true by touch | `general` | humanoid | holds Common-folk debts (cross-district tension) |
| 9347 | Dame Ostry | 5810 | an Extra Arm (visible) | `blacksmithing` | humanoid | weapon dealer; blades from Halvard; Dobb supply stop |
| 9348 | Brun the Armorer | 5812 | plated, callused skin | `blacksmithing` | humanoid | armorer; plate from warehouse; Dobb supply stop |
| 9349 | Clerk Vell | 5817 | a seal-shaped birthmark on his palm | `general` (minor permit/seal shop) | humanoid, bloodline_domestic | Horst's contact; collects tribute; the felt reach |
| 9350 | Madam Sephe | 5814 | a subtle charming glamour | `general` | humanoid | the Gilt Threshold; hosts Horst's meetings; the overhear hook |
| 9351 | A Market Crier | 5801/5802 | — | — | humanoid | ambient |
| 9352 | A Square Guard | 5801 | — | — | humanoid | ambient (`non_combatant`) |
| 9353 | A Moneylender's Clerk | 5822 | — | — | humanoid | ambient |
| 9354 | A Porter | 5823 | — | — | humanoid | ambient |
| 9355 | A Well-Dressed Customer | 5824 | — | — | humanoid | ambient |

**Shop vendors:** Falk (5808), Goss (5806), Ostry (5810), Brun (5812), Vell
(5817, minor), Sephe (5814). **Ostry + Brun additionally stock the raw materials
Dobb delivers** (steel ingot 40018, iron 40001, leather strip 40002, chain link
40019 — all already in the runner manifest) alongside their finished goods. Horst
has **no shop**.

---

## Task 0: Branch + ID sanity + coordinate groundwork

- [ ] **Step 1: Branch**
```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git checkout master && git checkout -b feature/np-merchant-district
```
- [ ] **Step 2: Confirm IDs free**
```bash
python tools/id_inventory.py --type rooms | grep -E "merchant|next free"   # expect rooms next-free 5725 → 5800-5824 clear
python tools/id_inventory.py --type mobs  | grep -E "next free"            # expect ≥ 9344
```
- [ ] **Step 3: Dump the live coordinate frame** (controller solves Stage A/B coords from this)
```bash
python3 - <<'PY'
import re,glob
for z in ["new_plymouth_docks","new_plymouth_common","new_plymouth_crafting","new_plymouth_merchant"]:
  cells=[]
  for f in glob.glob(f"_datafiles/world/dogmud/rooms/{z}/*.yaml"):
    t=open(f,encoding="utf-8").read()
    rid=re.search(r"roomid:\s*(\d+)",t); x=re.search(r"x:\s*(-?\d+)",t); y=re.search(r"y:\s*(-?\d+)",t); zc=re.search(r"z:\s*(-?\d+)",t)
    if rid and x and y: cells.append((int(rid.group(1)),int(x.group(1)),int(y.group(1)),int(zc.group(1)) if zc else 0))
  print(z, sorted((c[1],c[2],c[3],c[0]) for c in cells))
PY
```
Expected: prints occupied (x,y,z,roomid) per zone. Solve 5800–5824 into free cells east of Crafting / NE of Common (the Square as the hub), keeping the 5704→east seam a 1-step exit. Record the coord assignment for Stages A/B.
- [ ] **Step 4: Baseline boot** — boot-test recipe; expect clean (`ValidateZoneConsistency errors=0`). Kill server.

## Task 1: Stage A rooms — Central Square + arteries + the Crafting entry seam

**Files:** Create `rooms/new_plymouth_merchant/5800.yaml`…`5805.yaml` + `zone-config.yaml`; Modify `rooms/new_plymouth_crafting/5704.yaml` (add `east`→5800).

- [ ] **Step 1:** Dispatch a `/new-room`-style content subagent (sonnet) with the Cluster 1 manifest rows, the **exact coords** the controller solved, the Conventions block, and the reference files. Author 5800–5805 + `zone-config.yaml` (name `New Plymouth Merchant`, region `Windward Marches`, roomid 5800). Honor the prose-stubs (5803 N, 5804 S, 5805 E — no exits). The Central Square (5801) is the rich hub.
- [ ] **Step 2:** Edit `new_plymouth_crafting/5704.yaml` — add `east:` → roomid 5800 (reciprocal of 5800's `west`). Keep 5704's existing `west`→5703 + its prose intact.
- [ ] **Step 3: ansi leak check** — `grep -rnE 'fg="[^"]* [^"]*"' _datafiles/world/dogmud/rooms/new_plymouth_merchant/` → expect none.
- [ ] **Step 4: Boot-verify + cartcheck** — boot-test recipe; expect no panic, `ValidateZoneConsistency errors=0`. Re-solve coords on any collision.
- [ ] **Step 5: Commit** — `feat(np-merchant): Stage A rooms (5800-5805) — Central Square, arteries, Crafting entry`.

## Task 2: Stage B rooms — the four clusters

**Files:** Create `rooms/new_plymouth_merchant/5806.yaml`…`5824.yaml` (19 rooms).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with Clusters 2–6 manifest rows + the controller's exact coords + Conventions. Above-shop `up`/`down` stacks; the Gilt Threshold parlor (5815) carries the **overhear-Horst** scene noun; Falk's auction house (5808) carries the **property-records** noun (the Bloom link surface — author it intriguing/unresolved); Horst's house gate (5820) prose-stubs the locked interior (NO inward exit).
- [ ] **Step 2: ansi leak check** (same) — expect none.
- [ ] **Step 3: Boot-verify + cartcheck** — expect clean.
- [ ] **Step 4: Commit** — `feat(np-merchant): Stage B rooms (5806-5824) — financial/arms rows, inn, bloodline apparatus`.

## Task 3: The `bloodline_domestic` faction + `cooperage_circle` backfill

**Files:** Create `factions/bloodline_domestic.yaml`; Modify `factions/cooperage_circle.yaml`.

- [ ] **Step 1: Author `bloodline_domestic.yaml`** (match `np_dockfolk.yaml`'s field shape):
```yaml
faction_id: bloodline_domestic
display_name: "The Bloodline Domestic"
description: |
  The domestic apparatus of the bloodline that rules from beyond the city — its
  handlers, permit-clerks, and quiet agents who keep New Plymouth compliant and
  paying. They wear no uniform and need none. Their reach is felt in tribute
  rounds, stamped permits, and meetings that happen behind the right doors. To
  the working quarters they are the weight of a hand that is never quite removed.
default_rep: -10
allies: []          # temple_np ally is a FORWARD-REF (not built yet) — wire in the Temple build
enemies:
  - cooperage_circle
```
- [ ] **Step 2: Backfill `cooperage_circle.yaml`** — change `enemies: []` to:
```yaml
enemies:
  - bloodline_domestic
```
(Both factions load this build, so the mutual reference resolves — gotcha #2 only bites on a faction not built at all.)
- [ ] **Step 3: Boot-verify** — boot-test recipe; `factions.LoadAllDefinitions loadedCount` increments by 1 (→18); no faction panic.
- [ ] **Step 4: Commit** — `feat(np-merchant): bloodline_domestic faction + cooperage_circle enemy backfill`.

## Task 4: Stage C — the 7 anchors (9344–9350) + dialogue

**Files:** Create `mobs/new_plymouth_merchant/9344-*.yaml`…`9350-*.yaml` + `dialogue/new_plymouth_merchant/9344.yaml`…`9350.yaml`.

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Mob Manifest rows 9344–9350 + the dialogue SOPs + reference files. Each anchor: Title-Case name, `ConvertForFilename` filename, mutation woven into the description, `non_combatant: true`/`hostile: false`/`maxwander: 0`, `groups:` per manifest (bloodline_domestic on Horst 9344 + Vell 9349), shop block for vendors (Ostry/Brun also stock the raw materials 40018/40001/40002/40019 + a couple finished goods — reuse existing weapon/armor item IDs, verifying each exists in `_datafiles/world/dogmud/items/` before adding), ≥3 discoverable first-person dialogue topics, **no quests**. **Horst (9344): NO shop, NO craft_support; dialogue cold and guarded — he reveals nothing actionable, deflects, and radiates untouchable menace.** **Sephe (9350): a dialogue topic that lets the player learn she hosts Horst's quiet meetings (the overhear hook, discoverable).**
- [ ] **Step 2: Verify filenames** match `<id>-<ConvertForFilename(name)>.yaml`.
- [ ] **Step 3: Boot-verify** — boot-test recipe; `mobs.LoadDataFiles` +7; `ValidateShopMobTags` passes (no panic); no `AssertCanonical` panic.
- [ ] **Step 4: Commit** — `feat(np-merchant): the 7 anchors + dialogue (9344-9350)`.

## Task 5: Stage C — ambient mobs (9351–9355) + room spawns

**Files:** Create `mobs/new_plymouth_merchant/9351-*.yaml`…`9355-*.yaml`; Modify the room YAMLs to add `spawninfo:`.

- [ ] **Step 1:** Author 5 ambient `non_combatant` townsfolk (manifest 9351–9355; `behavior_archetype: noncombat_passive`, no shop/craft_support/dialogue), per the Crafting ambient pattern.
- [ ] **Step 2: Add `spawninfo:`** to each work/ambient room placing the anchors (9344→5820 or 5801, 9345→5808, 9346→5806, 9347→5810, 9348→5812, 9349→5817, 9350→5814) and ambient (9351→5802, 9352→5801, 9353→5822, 9354→5823, 9355→5824), `respawnrate: "10 real minutes"` (match `new_plymouth_docks/5501.yaml`).
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +5; confirm spawns load.
- [ ] **Step 4: Verify spawn coverage** — `grep "mobid: 93XX"` per room.
- [ ] **Step 5: Commit** — `feat(np-merchant): ambient residents (9351-9355) + room spawns`.

## Task 6: Stage D — Bloom Trail link (Falk) + the overhear scene (Sephe)

**Files:** Modify Falk's dialogue (9345) + room 5808 nouns; Sephe's dialogue (9350) + room 5815 nouns.

- [ ] **Step 1:** Author Falk's **property-records** breadcrumb: a discoverable dialogue topic (trigger words surfaced in the root/hints — avoid short stems that prefix-shadow, per the Crafting `buyer`/`buy` lesson) where Falk lets slip an address discrepancy — a Noble-quarter delivery-house that changed hands quietly / a canal-district property that shouldn't be occupied — pointing toward the **Noble delivery-house / Old Quarter**. Add a `property-roll`/`ledger` noun in 5808. **Seeds, does not resolve.**
- [ ] **Step 2:** Ensure the Gilt Threshold parlor (5815) overhear scene + Sephe's hint are coherent (the player can learn Horst meets here, without it resolving anything).
- [ ] **Step 3: Boot-verify** — no panic; dialogue loads.
- [ ] **Step 4: Commit** — `feat(np-merchant): Bloom Trail link (Falk's property records) + the overhear scene`.

## Task 7: Stage D — anchor schedules

**Files:** Create `schedules/new_plymouth_merchant/np_merchant_*.yaml`; Modify each anchor mob to add `schedule_id:`.

- [ ] **Step 1:** Author 24h-contiguous schedules (validators panic on gaps/unreachable; routing compass/`up`/`down` only). Beats: **market hours** (anchors at their work rooms by day, the Square busy at midday — converge on 5801/5821); **the Gilt Threshold fills** in the evening (Sephe + a merchant or two); **Clerk Vell's tribute rounds** (office 5817 → the Square → back); **Horst** moves furtively between his house (5820) and the inn parlor (5815). Every `target_room` must be reachable within the built rooms. Reference `np_crafting_halvard.yaml` for shape.
- [ ] **Step 2:** Add `schedule_id:` to each anchor mob (9344–9350).
- [ ] **Step 3: Boot-verify** — `LoadSchedules` +7; no coverage-gap/unreachable panic.
- [ ] **Step 4: In-game spot-check** — a couple of anchors move/sleep per segment.
- [ ] **Step 5: Commit** — `feat(np-merchant): anchor schedules — market hours, the Gilt Threshold, Vell's rounds`.

## Task 8: Stage D — extend Dobb's supply circuit through the Central Square

**Files:** Modify `patrols/new_plymouth/np_docks_runner_circuit.yaml`; possibly `internal/caravan/import_circuits.go` (only if Ostry/Brun stock a material not already in `ImportItems`).

- [ ] **Step 1:** Append waypoints to the patrol YAML so Dobb continues from the last Crafting vendor **east into the Central Square (5801) and on to Ostry (5810) + Brun (5812)**, then the strict loop returns to the depot. Use `arrival_event: np_runner_vendor` at Ostry/Brun (deliver), and the Square (5801) as a transit/visible stop (`arrival_event: ""` or `np_runner_vendor` if a Square material-vendor exists). Every new waypoint must be pathto-reachable (the Crafting→Merchant 5704→5800 seam must be clean). Keep the depot wp0 **zero-dwell** (Plan-2 gotcha #1).
- [ ] **Step 2:** Confirm Ostry/Brun's deliverable materials are all in `import_circuits.go`'s `ImportItems` + a covered bucket (40018 thornwall, 40001/40019 base, 40002 overlap — all already present). If they stock a material NOT covered, add it to `ImportItems` (and its bucket to `DeliveryBuckets`); else **no Go change**. Do NOT add Merchant to `CaravanServedZones` (finished goods stay ticker-served — gotcha #2 starvation rule; the runner is additive here).
- [ ] **Step 3: Boot-verify** — patrol loads with the new waypoints; no patrol-validator panic; `go build ./...` clean (if Go touched, `go test ./internal/caravan/...`).
- [ ] **Step 4: Smoke-test** (parked-player method, Plan-2 gotcha #3): boot, connect, teleport to the depot to activate Dobb, deplete an Ostry material, confirm the runner reaches the Square + tops up (watch the shop file `current`).
- [ ] **Step 5: Commit** — `feat(np-merchant): extend Dobb's circuit through the Central Square to Ostry/Brun`.

## Task 9: District harness playtest

- [ ] **Step 1:** `/playtest local feature-tester` — drive from the Crafting→Merchant seam: visit each anchor, `ask <npc> about <topic>` (≥1 each), buy from a vendor, try (and fail) to enter Horst's house, find Falk's Bloom breadcrumb + Sephe's overhear, observe an anchor on schedule + Dobb in the Square.
- [ ] **Step 2:** Triage findings; fix blocking/cosmetic inline (`fix(np-merchant): …`); log deferred polish in memory.
- [ ] **Step 3: Final boot test** — confirm clean.

## Task 10: Merge to master (hold push)

- [ ] **Step 1:** `git checkout master && git merge --no-ff feature/np-merchant-district -m "Merge: New Plymouth Merchant Quarter (district 4/7)"`.
- [ ] **Step 2:** Update `project_new_plymouth_build.md` + `MEMORY.md`: Merchant done + merged; `bloodline_domestic` now exists (Temple build wires the `temple_np` ally edge); NEXT = District 5 Temple. **Do NOT push.**

---

## Self-Review (completed during planning)

- **Spec coverage:** §1 zone/IDs/coords → Task 0 + manifests; §2 layout/entry/stubs →
  Tasks 1–2; §3 anchors (incl. watched Horst) → Task 4; §4 factions (create +
  backfill) → Task 3; §5 supply extension → Task 8; §6 Bloom link → Task 6; §7
  schedules → Task 7; §8 staging A–E → Tasks 1–9; §9 DoD → Task 9–10.
- **Placeholder scan:** room/mob bodies are authored by content subagents from the
  manifests + controller-solved coords (the project's established content pattern,
  not a placeholder); the faction YAMLs + patrol-extension are given as literal
  content. Item-ID reuse for Ostry/Brun finished goods is gated on
  "verify it exists" (no invented IDs). No TBD/TODO.
- **Consistency:** rooms 5800–5824, mobs 9344–9355, dialogue-by-mobid, faction
  `bloodline_domestic`/`cooperage_circle`, and the Crafting 5704→east seam are used
  identically across the manifests and tasks. Supply extension reuses the existing
  `np_docks_runner_circuit` + `ImportItems` (no new bucket needed).
