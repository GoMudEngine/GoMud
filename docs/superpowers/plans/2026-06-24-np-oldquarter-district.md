# New Plymouth — Old Quarter District Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the New Plymouth Old Quarter (`new_plymouth_old_quarter`, 20 rooms
6020–6039): the **z−1 buried canal city** — Lintel Street, 215 Lintel St (the Bloom
Trail climax as a discoverable crime scene), Quill's lamp-walk, and Gritta's flooded
cellar where the pre-Founding lore web closes — with 3 anchors + ~8 ambient, faction
membership, the Gritta↔Coll / Gritta↔Orin lore edges, anchor schedules, and **no
quest / no combat** (Deren is watched/untouchable). The FINAL district: completing
it triggers the **whole-capital pre-push SOP + push prod** (Task 11).

**Architecture:** Pure content (YAML data files) authored by content subagents
within a pre-allocated ID block — **no new faction, no Go change, no quest**.
Validated by server boot + `cartcheck` + schedule/relationship validators + a harness
playtest. The controller (main loop) drives all shell — subagents are shell-denied in
this env.

**Tech Stack:** GoMud engine (run locally), DOGMud world YAML (`docs/schemas/`),
`tools/id_inventory.py`, `cartcheck`, the `/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-06-24-np-oldquarter-district-design.md`.

---

## Conventions for every task (READ FIRST — identical discipline to prior districts)

- **Branch:** `feature/np-oldquarter-district` (from `master` in Task 0). Commit per
  stage; trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Controller drives all shell** (subagents Write/Edit YAML only).
- **Boot test = verification** (YAML errors panic at startup):
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . > /tmp/oq_boot.log 2>&1 &
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath" /tmp/oq_boot.log   # expect none
  grep -nE "ValidateZoneConsistency|loadAllRoomZones|mobs.LoadDataFiles|LoadSchedules|Server Ready" /tmp/oq_boot.log
  ```
  NOT bare `panic` (gotcha #8 — `MapConsistencyEnforce value=panic` is a normal config
  line). Kill after (`taskkill //F //IM go.exe //T`).
- **Coordinates — solve against the LIVE frame, z = −1 (buried).** Controller dumps
  occupied cells before Stage A/B. The ONLY existing z−1 rooms are Docks **5520
  (−20,80,z−1)** and **5521 (−19,80,z−1)** — keep all Old Quarter cells clear of
  those two. Entry: **5520 → west → 6020** (west = (−21,80,z−1), free). The quarter
  spreads **west/south**, rough footprint x[−27..−21]/y[76..84]/z−1; two interior
  structures descend to **z−2** (215's corridor+production room; Gritta's deep
  cellar/pre-Founding stonework) via vertical exits (always collision-free).
  `cartcheck errors=0` is the gate. Do NOT trust the citywide nominal region (y86+ was
  nominal; the real underdock anchor is y80).
- **Authoring gotchas:** (1) `": "` in a prose value → `" — "` (a colon-space in a
  noun value panicked the Noble boot — watch noun/desc values especially). (2) faction
  membership via `groups:` referencing EXISTING factions only (`bloom_trade`,
  `np_commonfolk` both exist — no new faction). (3) Title-Case mob names
  (`casing.AssertCanonical` panics). (4) no exits to unbuilt rooms (the cooperage
  cellar-mouth stays boarded; no OQ→elsewhere stubs). (5) IF an optional shop is built
  it needs a valid `craft_support:` tag (default = no shop). (6) noun ansi
  `<ansi fg="itemname">noun</ansi>` only; post-grep `fg="[^"]* [^"]*"` → zero. (7)
  interiors compass/`up`/`down` only — never `enter`/`leave`. (8) **sentence-leading
  noun tags lowercase**. (9) avoid short trigger stems that prefix-shadow longer
  discoverable words. (10) **VERIFY the zone folder name** from the loader's
  `Filepath()` — display name `New Plymouth Old Quarter` →
  `ConvertForFilename` → folder `new_plymouth_old_quarter` (a mismatch panics:
  `... did not end in Filepath()`).
- **Reference files:** rooms `…/rooms/new_plymouth_docks/5520.yaml` (the z−1 seam
  room) + `…/5521.yaml` (the z−1 Bloom breadcrumb, sealed dead-end — DO NOT alter);
  mob+dialogue `…/mobs/new_plymouth_merchant/9344-horst.yaml` (the watched/untouchable
  non-combatant anchor pattern — Deren mirrors this) + `…/dialogue/new_plymouth_merchant/9344.yaml`;
  the Bloom-front pattern `…/mobs/new_plymouth_docks/9305-marn_the_draper.yaml` +
  dialogue; the relationships field shape `…/mobs/ashwick/259-delia.yaml`; schedule
  `…/schedules/new_plymouth_docks/np_docks_marn.yaml` (a furtive night-aware beat).

---

## ID allocation (pre-assigned)

| Kind | Range | Assignment |
|------|-------|------------|
| Rooms | **6020–6039** | see Room Manifest |
| Mobs | **9379–9389** | 3 anchors (9379–9381) + ~8 ambient (9382–9389) |
| Dialogue | by mobid | one per speaking mob (3 anchors + Deren's doorman) |
| Faction | none created (membership only) | `bloom_trade` / `np_commonfolk` |
| Zone folder | `new_plymouth_old_quarter` | `ConvertForFilename("New Plymouth Old Quarter")` — VERIFY |
| Items / Quests | none | (optional scavenger stall reuses verified ids only — default off) |

---

## Room Manifest (20 — clusters + purpose; coords solved at Stage A/B dispatch)

Entry: Docks Underdocks **5520 → west → 6020** (z−1).

**Cluster 1 — the canal mouth + Lintel Street spine (5, z−1):** 6020 The Canal Mouth
(entry from Underdocks 5520; the bilge channel widens into a dead, still canal; the
copper-flower trace from the Pilings Haunt lingers faintly), 6021 Lower Lintel Street
(the canal-side way begins; the poorest, the water close), 6022 The Stone Footbridge
(canon — the street ducks under a pre-colonial stone footbridge here), 6023 Lintel
Street (narrows as it climbs from the water), 6024 Upper Lintel Street (toward number
215; the quieter, watched end).

**Cluster 2 — 215 Lintel Street, the Bloom climax (5; z−1 + z−2):** 6025 215 Lintel
Street — the Iron Door (**Deren's Doorman 9382**; the iron-banded door, the canon
pattern-knock; a heavy who watches but does not block), 6026 The Seven Steps (the
descent; creaking steps; z−1→z−2), 6027 The Stone Corridor (z−2; two oil lamps, low
ceiling, mineral cold), 6028 The Production Room (**z−2; the crime scene** — the drain
in the floor, the bare pallet, the collection apparatus: glass, copper tubing,
wax-sealed clay pots; the captive long gone), 6029 Deren's Ledger Room (**Deren
9379**; z−1, off the iron-door vestibule; the creaking 4th & 7th steps; the ledger of
a closed account).

**Cluster 3 — Quill's lamp-walk + the drowned courts (5, z−1):** 6030 Quill's
Lamp-Walk (the canal-side lamps Quill tends; the one warm thread in the dark), 6031
Quill's Hovel (**Quill 9380**; a single lamp, the alley window; the Bloom witness),
6032 A Drowned Court (a flooded court of the poorest; ambient), 6033 Canal-side Hovels
(the destitute residents; ambient), 6034 The Flooded Stair (a stair down into the
deep stone; z−1→z−2; toward Gritta).

**Cluster 4 — Gritta's cellar + the pre-Founding stonework, the lore climax (5,
z−2):** 6035 Gritta's Flooded Cellar (**Gritta 9381**; she works the silt; the gray
material she senses in the deep stone), 6036 The Pre-Founding Stonework (massive
lintels older than the colony; the city paved over this), 6037 The Buried Lintel
(**an original lintel carved with the orbital / eight-pointed symbol — CLOSES the
Noble gallery cipher**), 6038 The Deep Canal (a flooded dead-end; the gray material
showing at the waterline), 6039 A Silted Gallery (the pre-Founding gray material at
source; Gritta's find; the oldest dark).

(The subagent wires intra-cluster + spine exits per the controller's exact coords;
z transitions via `up`/`down` only. The cooperage cellar-mouth is NOT connected — no
exit beyond the built rooms.)

---

## Mob Manifest (11 — 9379–9389; all `non_combatant`)

| Mob | Name | Work room | Mutation | Shop | `groups` | Dialogue |
|-----|------|-----------|----------|------|----------|----------|
| 9379 | Deren | 6029 | veins glowing faint copper (Bloom exposure) | none | humanoid, bloom_trade | yes (≥3) |
| 9380 | Quill the Lamplighter | 6031 | night-adapted eyes | none | humanoid, np_commonfolk | yes (≥3) |
| 9381 | Gritta | 6035 | senses the buried gray material | none | humanoid | yes (≥3) |
| 9382 | Deren's Doorman | 6025 | an immovable stance | none | humanoid, bloom_trade | yes (short — menace, no gate) |
| 9383 | A Bloom-Addled Wanderer | 6021 | — | — | humanoid | — |
| 9384 | A Canal Beggar | 6032 | — | — | humanoid, np_commonfolk | — |
| 9385 | A Mudlark | 6038 | — | — | humanoid | — |
| 9386 | A Destitute Resident | 6033 | — | — | humanoid, np_commonfolk | — |
| 9387 | A Furtive Runner | 6023 | — | — | humanoid, bloom_trade | — |
| 9388 | A Canal-side Crone | 6033 | — | — | humanoid, np_commonfolk | — |
| 9389 | A Lamplighter's Boy | 6030 | — | — | humanoid, np_commonfolk | — |

All anchors + the doorman: `non_combatant: true`, `hostile: false`, `maxwander: 0`.
Ambient: `non_combatant: true`, `behavior_archetype: noncombat_passive`, no
shop/craft_support/dialogue.

---

## Task 0: Branch + ID sanity + coordinate groundwork

- [ ] **Step 1: Branch** — `git checkout master && git checkout -b feature/np-oldquarter-district`.
- [ ] **Step 2: Confirm IDs free** — `python tools/id_inventory.py --type rooms | grep -E "next free"` (expect next-free 6020 → 6020–6039 clear); `--type mobs` next-free ≥ 9379.
- [ ] **Step 3: Dump the live z−1 frame** — list all rooms at `z: -1` across the built NP zones (expect only Docks 5520 at −20,80 and 5521 at −19,80). Solve 6020–6039 into free z−1/z−2 cells west/south of 5520, keeping **5520→west→6020** a 1-step exit. Record the coordinate assignment (entry 6020 at −21,80,z−1; spine running south/west; the two z−2 sub-clusters under the 215 and the flooded-stair cells). Confirm none collide with 5520/5521.
- [ ] **Step 4: Baseline boot** — boot-test recipe; `errors=0`. Kill server.

## Task 1: Stage A rooms — the canal mouth + Lintel Street spine

**Files:** Create `rooms/new_plymouth_old_quarter/6020.yaml`…`6024.yaml` + `zone-config.yaml`; Modify `rooms/new_plymouth_docks/5520.yaml` (add `west`→6020 + a light prose line).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Cluster 1 manifest + the controller's exact coords + Conventions + reference files. Author 6020–6024 + `zone-config.yaml` (`name: New Plymouth Old Quarter`, `roomid: 6020`, `defaultbiome: city`, `region: Windward Marches` — matching the Docks zone-config). The through-line: a buried, water-close, lamp-lit poverty; pre-colonial stone the colony paved over; the canal dead and still. The footbridge (6022) is the canon stone footbridge ("ducks under… narrows"). Faint copper-flower trace near the canal mouth (continuity with the Pilings Haunt 5521).
- [ ] **Step 2:** Edit `new_plymouth_docks/5520.yaml` — add `west:` → roomid 6020 (reciprocal of 6020's `east`). Append ONE line to the description revealing the west canal mouth (today it ends "The passage east continues into the dark between the piles. The stair is behind you." — add that the bilge channel / a low passage also runs WEST into the old canal). Keep 5520's existing exits (up→5509, east→5521), its nouns, and its 9318 spawn intact.
- [ ] **Step 3: ansi leak check** — `grep -rnE 'fg="[^"]* [^"]*"' _datafiles/world/dogmud/rooms/new_plymouth_old_quarter/` → none.
- [ ] **Step 4: Boot-verify + cartcheck** — `errors=0`; the 5520→6020 seam loads. Re-solve coords on any collision (esp. vs 5520/5521).
- [ ] **Step 5: Commit** — `feat(np-oldquarter): Stage A rooms (6020-6024) — the canal mouth, Lintel Street, Underdocks seam`.

## Task 2: Stage B rooms — 215 Lintel St, Quill's, the drowned courts, Gritta's cellar

**Files:** Create `rooms/new_plymouth_old_quarter/6025.yaml`…`6039.yaml` (15 rooms).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with Clusters 2–4 manifest + exact coords (incl. the z−2 cells) + Conventions. Three payoff zones to author richly:
  - **215 Lintel Street (6025–6029):** the iron-banded door + doorman vestibule (6025); the seven steps (6026, z−1→z−2 `down`); the stone corridor (6027, z−2); the **production room (6028, z−2)** — author the room + nouns as a *discoverable crime scene*: the drain in the floor, the bare pallet, the collection apparatus (glass, copper tubing, wax-sealed clay pots), and the clear sense the captive is **gone** (canon — Junie went south). Deren's ledger room (6029, z−1, `up` from the steps / off the vestibule; the creaking 4th & 7th steps). This is the Bloom Trail PAYOFF — make the place *say* what happened without a quest.
  - **Quill's lamp-walk + drowned courts (6030–6034):** the warm canal lamps vs the flooded poverty; Quill's hovel (6031); the flooded stair (6034, z−1→z−2 `down`).
  - **Gritta's cellar + pre-Founding stonework (6035–6039, z−2):** the LORE climax. Author 6037 **The Buried Lintel** so the **orbital / eight-pointed symbol** carved in the original stone is a real, readable revelation — the buried city the Noble gallery cipher (Lysha vs Ferrol), Dross, Ept, and Orin all pointed at. The gray material (6038/6039) is the pre-Founding substance Gritta senses.
  - All z transitions via `up`/`down`. The cooperage cellar-mouth is NOT connected (no exit beyond the built rooms).
- [ ] **Step 2: ansi leak check** (same) — none.
- [ ] **Step 3: Boot-verify + cartcheck** — clean; all z−1/z−2 cells collision-free.
- [ ] **Step 4: Commit** — `feat(np-oldquarter): Stage B rooms (6025-6039) — 215 Lintel St, Quill's, the drowned courts, Gritta's cellar`.

## Task 3: Stage C — the 3 anchors (9379–9381) + the doorman (9382) + dialogue

**Files:** Create `mobs/new_plymouth_old_quarter/9379-*.yaml`…`9382-*.yaml` + `dialogue/new_plymouth_old_quarter/9379.yaml`…`9382.yaml`.

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Mob Manifest rows 9379–9382 + dialogue SOPs + reference files (Horst 9344 = the watched/untouchable pattern; Marn 9305 = the Bloom-front voice). Each: Title-Case name, `ConvertForFilename` filename, mutation woven into the description, `non_combatant: true`/`hostile: false`/`maxwander: 0`, `groups:` per manifest, ≥3 discoverable first-person dialogue topics, **NO quests, NO shops, NO craft_support**. Key dialogue intents:
  - **Deren (9379)** — exposed, careful, merchant-not-fighter; a man who knows he is watched. Three topics: (a) the operation — oblique, never an outright confession, but he does not pretend; (b) **Marn** — his Docks contact / the successor vacuum (cross-ref the Bloom Trail; Marn already references "his supplier"); (c) the captive — gone now, "surrendered the key," spoken like a closed account. He is `non_combatant` — a player who followed the trail confronts him verbally and is told, in effect, the bloodline office runs the production side and the watch already has eyes here: nothing is the player's to close.
  - **Quill (9380)** — night-adapted eyes; "light the lamps, see nothing, say less." Three topics: (a) the lamps / his work; (b) the **traffic** at 215 at odd hours — the Bloom witness breadcrumb, oblique and frightened-poor, a confirmation for a player who already has the address (do NOT make it a confession); (c) the quarter / the poorest.
  - **Gritta (9381)** — senses the buried gray material. Three topics: (a) the gray material in the deep stone (her mutation); (b) the pre-Founding city — the lintels older than the colony; what the bloodline's Founding story overwrites (the lore closer); (c) her fragments — she feeds finds to **Coll** (Common) and **Orin** (Crafting); name BOTH so the web is discoverable from her end.
  - **Deren's Doorman (9382)** — short dialogue/idle lines: pure menace and unease; he watches and disapproves but says he is not paid to stop anyone walking into a place that is already finished. **He does NOT block movement** (no `pathto`/exit gate; the player passes freely to the steps).
- [ ] **Step 2: Verify filenames** match `<id>-<ConvertForFilename(name)>.yaml`.
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +4; `AssertCanonical` passes; no `ValidateShopMobTags` issue (no shops).
- [ ] **Step 4: Commit** — `feat(np-oldquarter): Deren, Quill, Gritta + the doorman (9379-9382) + dialogue`.

## Task 4: Stage C — ambient mobs (9383–9389) + room spawns

**Files:** Create `mobs/new_plymouth_old_quarter/9383-*.yaml`…`9389-*.yaml`; Modify room YAMLs to add `spawninfo:`.

- [ ] **Step 1:** Author 7 ambient `non_combatant` mobs (manifest 9383–9389; `behavior_archetype: noncombat_passive`, no shop/craft_support/dialogue; `groups:` per manifest). The Bloom-addled wanderer (9383) may reuse the Docks 9317 archetype's muttered Bloom-tinged flavor (the human cost of the trade).
- [ ] **Step 2: Add `spawninfo:`** placing anchors (9379→6029, 9380→6031, 9381→6035, 9382→6025) + ambient (9383→6021, 9384→6032, 9385→6038, 9386→6033, 9387→6023, 9388→6033, 9389→6030), `respawnrate: "10 real minutes"`.
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +7; spawns load.
- [ ] **Step 4: Verify spawn coverage** — `grep "mobid: 93XX"` per room.
- [ ] **Step 5: Commit** — `feat(np-oldquarter): ambient canal-folk (9383-9389) + room spawns`.

## Task 5: Stage D — the Bloom crime-scene lore + Deren confrontation

**Files:** Modify the 215 room nouns (6025–6029, esp. the production room 6028) + Deren (9379) dialogue.

- [ ] **Step 1: The crime scene reads.** Ensure 6028's room + nouns tell the story environmentally — the drain, the pallet, the apparatus, the absence of the captive — so a player who followed the trail (Falk's Lintel St pointer / Wenna's delivery-house / Marn's supplier / the Pilings Haunt smell) reaches a real, legible payoff WITHOUT a quest. Surface discoverable noun keywords (`apparatus`, `drain`, `pallet`, `ledger`, `Bloom`) coherently; avoid prefix-shadowing.
- [ ] **Step 2: The confrontation.** Ensure Deren's dialogue closes the *narrative* trail: exposed, watched/untouchable, the captive gone, the production side run by the bloodline office — the player cannot "solve" it here (the Bloom mechanic + any quest are a deferred backfill). Make the through-line (Falk→Wenna→Marn→Deren) feel completed as story.
- [ ] **Step 3: Boot-verify** — no panic; dialogue + nouns load.
- [ ] **Step 4: Commit** — `feat(np-oldquarter): the Bloom Trail climax — 215 Lintel St crime scene + Deren confrontation (lore)`.

## Task 6: Stage D — the pre-Founding lore web closes

**Files:** Modify Gritta (9381) dialogue + the pre-Founding room nouns (6036/6037/6038/6039, esp. the Buried Lintel 6037).

- [ ] **Step 1: The buried lintel reads as the cipher's answer.** Ensure 6037's nouns render the **orbital / eight-pointed symbol** carved in the original stone as the buried city the Noble gallery cipher pointed at — a player who saw the gallery (Lysha vs Ferrol) and heard Dross/Ept/Orin can recognize it here. Surface `symbol`, `lintel`, `founding`, `stone`, `Bloom`-free lore keywords; cross-reference the gallery/Dross/Ept/Orin where natural.
- [ ] **Step 2: Gritta closes the web.** Ensure Gritta's dialogue names what she senses (the gray material), what the lintels mean (the city built over an older settlement), and that she feeds fragments to **Coll** and **Orin** — making the web discoverable from her end. A real revelation, no quest.
- [ ] **Step 3: Boot-verify** — no panic; loads.
- [ ] **Step 4: Commit** — `feat(np-oldquarter): the pre-Founding lore web closes — Gritta + the buried orbital-symbol lintel`.

## Task 7: Stage D — Gritta↔Coll / Gritta↔Orin relationship edges + Deren→Marn cross-ref

**Files:** Modify `mobs/new_plymouth_old_quarter/9381-*.yaml` (Gritta); Modify `mobs/new_plymouth_common/9320-coll_the_sweeper.yaml` + `mobs/new_plymouth_crafting/9332-orin_the_bookseller.yaml`; (optional) Modify `mobs/new_plymouth_docks/9305-marn_the_draper.yaml`.

- [ ] **Step 1: Add the lore-source relationship edges.** The schema is a top-level `relationships:` list of `{to: <mobid>, type, subtype}` (see `ashwick/259-delia.yaml`; `type: friend` is the proven value, `subtype` is freeform). On Gritta (9381) add:
```yaml
relationships:
  - to: 9320
    type: friend
    subtype: lore_source
  - to: 9332
    type: friend
    subtype: lore_source
```
And add the reciprocal block (pointing back `to: 9381`) to BOTH `9320-coll_the_sweeper.yaml` and `9332-orin_the_bookseller.yaml`, keeping their existing content intact (append a `relationships:` block, or add an entry if one already exists):
```yaml
relationships:
  - to: 9381
    type: friend
    subtype: lore_source
```
- [ ] **Step 2: Deren→Marn cross-ref.** Confirm Deren's dialogue (Task 3/5) names Marn as the Docks end; OPTIONALLY add a `relationships:` edge Deren(9379)→Marn(9305) `{type: friend, subtype: trade}` + the reciprocal on Marn — keep the Marn edit minimal (dialogue cross-ref is sufficient; the edge is optional polish).
- [ ] **Step 3: Boot-verify** — the relationship edges load with no panic (every `to:` resolves: 9320/9332/9381 all exist; 9305 if used). Watch for the relationship loader rejecting an unknown `type` — `friend` is safe.
- [ ] **Step 4: Commit** — `feat(np-oldquarter): Gritta<->Coll / Gritta<->Orin lore edges + Deren->Marn cross-ref`.

## Task 8: Stage D — anchor + pooled schedules

**Files:** Create `schedules/new_plymouth_old_quarter/np_oldquarter_*.yaml`; Modify each anchor mob to add `schedule_id:`.

- [ ] **Step 1:** Author 24h-contiguous schedules (validators PANIC on coverage gaps or unreachable `pathto`; compass/`up`/`down` routing only — all targets within the built OQ rooms). Reference `np_docks_marn.yaml` (a furtive night-aware beat). Beats:
  - **Quill (9380) — night-active lamplighter:** lights the canal lamps at dusk, walks the lamp-walk (6030) + Lintel Street through the night, sleeps shallow by day in his hovel (6031, `activity: sleeping`). Mirrors the Docks/Common night-active rhythm.
  - **Deren (9379) — furtive:** ledger room (6029) by day; stays within the 215 cluster + Lower Lintel; `activity: sleeping` overnight in a back corner of 6029. His world is small and watched.
  - **Gritta (9381):** works the deep cellar / silted gallery (6035/6039, z−2) by day; surfaces to the flooded stair (6034); sleeps in the cellar (6035). Mostly stationary in the lore cluster.
  - **Doorman (9382)** + ambient may stay schedule-less (stationary spawns) OR get a simple pooled beat — keep it minimal.
- [ ] **Step 2:** Add `schedule_id:` to the 3 anchors (9379–9381).
- [ ] **Step 3: Boot-verify** — `LoadSchedules` +3; no coverage-gap/unreachable panic; the executor places the anchors (spot-check: Quill active on the lamp-walk after dusk, Gritta in the cellar by day).
- [ ] **Step 4: Commit** — `feat(np-oldquarter): anchor schedules — Quill's night lamps, Deren furtive, Gritta in the deep`.

## Task 9: District harness playtest

- [ ] **Step 1:** `/playtest local feature-tester` — drive from the Underdocks 5520→west seam: walk Lintel Street under the footbridge; reach **215 Lintel St**, pass the doorman freely, descend the seven steps, read the **production-room crime scene**, confront **Deren** (confirm un-attackable + the trail closes as story); find **Quill's** Bloom-witness breadcrumb; descend to **Gritta's** cellar and read the **buried orbital-symbol lintel** (confirm the gallery-cipher payoff lands); observe Quill night-active.
- [ ] **Step 2:** Triage; fix blocking/cosmetic inline (`fix(np-oldquarter): …`); log deferred polish.
- [ ] **Step 3: Final boot test** — clean.

## Task 10: Merge to master (hold push)

- [ ] **Step 1:** `git checkout master && git merge --no-ff feature/np-oldquarter-district -m "Merge: New Plymouth Old Quarter (district 7/7 — the capital is complete)"`.
- [ ] **Step 2:** Update `project_new_plymouth_build.md` + `MEMORY.md`: Old Quarter done + merged; the Bloom Trail climax + pre-Founding web both resolved as lore; **the capital is 7/7 COMPLETE**; NEXT = the whole-capital pre-push SOP + push prod (Task 11). **Do NOT push yet** (Task 11 gates it).

## Task 11: THE FINALE — whole-capital pre-push SOP + push prod

> The capital is complete and the push hold is RELEASED. Execute the pre-push SOP
> (CLAUDE.md) against the WHOLE accumulated bundle (~95+ commits ahead of prod).

- [ ] **Step 1: PATCH_NOTES.md** — extend/complete the New Plymouth entry to cover the WHOLE capital (all 7 districts: Docks, Common, Crafting, Merchant, Temple, Noble, Old Quarter + the supply runner + the engine prereqs), describing the player-facing shape of the city (a living capital you arrive at by sea, the Long Market supply spine, the Processional, the watched Noble streets, the buried Old Quarter). Dated 2026-06-24.
- [ ] **Step 2: config** — confirm `Logging.LogToFile: false` in `_datafiles/config.yaml` (prod droplet disk).
- [ ] **Step 3: Full local boot test (the real gate)** — wipe instance saves, rebuild, boot, and confirm clean load past data-file loading:
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go build ./... && go run . > /tmp/final_boot.log 2>&1 &
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath" /tmp/final_boot.log   # expect none
  grep -nE "loadAllRoomZones|mobs.LoadDataFiles|quests.LoadDataFiles|LoadSchedules|LoadPatrols|ValidateZoneConsistency|Server Ready" /tmp/final_boot.log
  ```
  Confirm: all 7 NP zones load, room/mob/quest loadedCounts sane, **`ValidateZoneConsistency` errors=0** (note the mode), schedule/patrol/relationship validators clean, **no panics**. Kill server.
- [ ] **Step 4: Push** — `git push origin master`. (The USER does the droplet deploy.)
- [ ] **Step 5: Update memory** — mark the capital DONE; record prod state (new HEAD); append a datapoint to `reference_prod_perf_baseline` after the droplet restart (this is the largest content bundle yet — watch the restart time); close the New Plymouth build project.

---

## Self-Review (completed during planning)

- **Spec coverage:** §0 scope (no quest / watched-untouchable / ambient lore) →
  Tasks 3/5/6 (no quest engine touched); §1 zone/IDs/coords (z−1/z−2, folder verify)
  → Task 0 + manifests; §2 layout/seam/z-levels/cooperage-boarded → Tasks 1–2; §3
  anchors + ambient → Tasks 3–4; §4 faction membership → Tasks 3–4 (groups); §5a
  Bloom climax lore → Task 5; §5b pre-Founding web → Task 6; §6 edges (Gritta↔Coll/
  Orin, Deren→Marn, no Dobb) → Task 7; §7 schedules → Task 8; §8 staging A–F → Tasks
  1–11; §9 DoD → Tasks 9–10; §11 finale (whole-capital pre-push SOP + push) → Task 11.
- **Placeholder scan:** room/mob bodies are subagent-authored from manifests +
  controller-solved coords (the established content pattern); the relationship-edge
  YAML is literal; no shop/quest/Go work. No TBD/TODO.
- **Consistency:** rooms 6020–6039, mobs 9379–9389, dialogue-by-mobid, Gritta 9381 ↔
  Coll 9320 ↔ Orin 9332, Deren 9379 → Marn 9305, faction groups (`bloom_trade`,
  `np_commonfolk` — both existing), z−1 entry off 5520, z−2 sub-levels via vertical
  exits, used identically across manifests and tasks. No new faction, no Go change,
  no quest.
- **The finale (Task 11)** is correctly part of THIS plan — completing the Old Quarter
  is the trigger; the push hold (user policy 2026-06-20) releases only here.
