# New Plymouth — Noble District Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the New Plymouth Noble Quarter (`new_plymouth_noble`, 20 rooms
6000–6019): the bloodline elite's watched streets — admin, the gallery where the
pre-Founding cipher resolves as lore, the atelier, the gated lane + Wenna's Bloom
beat, and the Palace gate — with 7 anchors + ambient, faction membership, anchor
schedules, the Doryn↔Garrick comrade edge, and a Dobb supply branch to Aurel.

**Architecture:** Pure content (YAML data files) authored by content subagents
within a pre-allocated ID block — no new faction, no Go change. Validated by server
boot + `cartcheck` + schedule/patrol validators + a harness playtest. The
controller (main loop) drives all shell — subagents are shell-denied in this env.

**Tech Stack:** GoMud engine (run locally), DOGMud world YAML (`docs/schemas/`),
`tools/id_inventory.py`, `cartcheck`, the `/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-06-22-np-noble-district-design.md`.

---

## Conventions for every task (READ FIRST — identical discipline to prior districts)

- **Branch:** `feature/np-noble-district` (from `master` in Task 0). Commit per
  stage; trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Controller drives all shell** (subagents Write/Edit YAML only).
- **Boot test = verification** (YAML errors panic at startup):
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . > /tmp/n_boot.log 2>&1 &
  grep -nE "ERROR:.*PANIC|fatal error:|did not end in Filepath" /tmp/n_boot.log   # expect none
  grep -nE "ValidateZoneConsistency|loadAllRoomZones|mobs.LoadDataFiles|LoadSchedules|LoadPatrols|Server Ready" /tmp/n_boot.log
  ```
  NOT bare `panic` (gotcha #8). Kill after (`taskkill //F //IM go.exe //T`).
- **Coordinates — solve against the LIVE frame.** Controller dumps occupied cells
  before Stage A/B and hands exact collision-free coords. Noble climbs NORTH off
  Merchant 5803 (Processional, at −14/90); entry 5803→north→6000; Noble in the
  open ground y≥91. `cartcheck errors=0` is the gate.
- **Authoring gotchas:** (1) `": "` in a prose value → `" — "`. (2) faction
  membership via `groups:` referencing EXISTING factions only (bloodline_domestic,
  np_commonfolk, cooperage_circle all exist — no new faction). (3) Title-Case mob
  names. (4) no exits to unbuilt rooms (Palace/gated-lane/Old-Quarter = prose
  stubs). (5) shop mobs need `craft_support:` (only Aurel). (6) noun ansi
  `<ansi fg="itemname">noun</ansi>` only; post-grep `fg="[^"]* [^"]*"` → zero. (7)
  interiors compass/`up`/`down` only. (8) **sentence-leading noun tags lowercase**.
  (9) avoid short trigger stems that prefix-shadow longer discoverable words. (10)
  ONLY verified-existing item ids in Aurel's shop.
- **Reference files:** rooms `…/rooms/new_plymouth_merchant/5802.yaml` (hub) +
  `…/5820.yaml` (impassable gate, for the Palace/lane stubs); mob+dialogue
  `…/mobs/new_plymouth_merchant/9344-horst.yaml` (a no-shop bloodline anchor) +
  `…/dialogue/new_plymouth_merchant/9344.yaml`; the relationships field shape in
  `…/mobs/ashwick/259-delia.yaml`; schedule
  `…/schedules/new_plymouth_merchant/np_merchant_vell.yaml`; patrol
  `…/patrols/new_plymouth/np_docks_runner_circuit.yaml`.

---

## ID allocation (pre-assigned)

| Kind | Range | Assignment |
|------|-------|------------|
| Rooms | **6000–6019** | see Room Manifest |
| Mobs | **9368–9378** | 7 anchors (9368–9374) + 4 ambient (9375–9378) |
| Dialogue | by mobid | one per speaking mob |
| Faction | none created (membership only) | bloodline_domestic / np_commonfolk / cooperage_circle |
| Zone folder | `new_plymouth_noble` | `ConvertForFilename("New Plymouth Noble")` |
| Items / Quests | none | |

---

## Room Manifest (20 — clusters + purpose; coords solved at Stage A dispatch)

Entry: Merchant **5803 (Processional North) → north → 6000**.

**Cluster 1 — the watched streets + entry (5):** 6000 The Processional — Noble
Approach (entry from Merchant 5803; the climb into the elite quarter), 6001 The
Watched Plaza (a surveilled square; the chilly observed atmosphere; a sentry),
6002 The Noble Way (the main promenade), 6003 Guide Ferrol's Flat (**Ferrol 9370**;
the approved-history script), 6004 The Tour Steps (a viewing point; the official
narrative posted).

**Cluster 2 — admin + gallery (5):** 6005 The Administrative Office (**Caldwin
9368**; the bloodline's will, executed cleanly), 6006 The Office Antechamber (a
cold waiting room; petitioners), 6007 The Art Gallery — Hall (**the cipher
resolution** — the old paintings; the orbital symbol in the pigments), 6008 The
Gallery — Upper (**Lysha 9371** lives/works above; her color-true reading of the
pigments), 6009 The Gallery — Old Wing (the oldest, least-visited canvases; the
suppressed pre-Founding scenes).

**Cluster 3 — atelier + gated lane + servants (5):** 6010 Modiste Aurel's Atelier
(**Aurel 9373**; cloth from Nessa; a Dobb supply stop), 6011 The Estate Lane —
Gatehouse (**Skell 9372**; turns all away — the gated lane is a STUB beyond here,
prose-only, NO exit through), 6012 The Servants' Stair (the back way the staff
use), 6013 Wenna's Garret (**Wenna 9369**; the **Bloom delivery-house** breadcrumb;
terrified of Caldwin), 6014 The Service Court (a back court; deliveries; a watched
servant).

**Cluster 4 — the Palace approach (5):** 6015 The Palace Processional (the
ceremonial final stretch), 6016 The Palace Gatehouse (**Doryn 9374**; the Palace
gate is an endgame STUB — prose-only, impassable, NO exit through to the Palace),
6017 The Gatehouse Barracks (Doryn's post; the Doryn↔Garrick beat), 6018 The
Sentry Walk (a patrolled wall-walk; a sentry), 6019 The Noble Garden (a manicured
elite garden; a noble passer-by; ambient).

(The subagent wires intra-cluster + cluster-to-promenade exits per the controller's
exact coords; the Palace gate (6016) and the gated lane (6011) are impassable
prose-stubs — NO exits beyond them.)

---

## Mob Manifest (11 — 9368–9378)

| Mob | Name | Work room | Mutation | Shop / `craft_support` | `groups` |
|-----|------|-----------|----------|------------------------|----------|
| 9368 | Steward Caldwin | 6005 | none (uninfected pride) | none | humanoid, bloodline_domestic |
| 9369 | Wenna the Servant | 6013 | flinches with a faint fear-light | none | humanoid, np_commonfolk |
| 9370 | Guide Ferrol | 6003 | a rehearsed smile that never reaches the eyes | none | humanoid, bloodline_domestic |
| 9371 | Keeper Lysha | 6007 | color-true eyes (reads old pigments) | none | humanoid, cooperage_circle |
| 9372 | Porter Skell | 6011 | an immovable stance | none | humanoid, bloodline_domestic |
| 9373 | Modiste Aurel | 6010 | fingers that read fabric quality blind | `tailoring` shop: thread 40012 + cloth strip 40007 (runner materials) + 1-2 finished garments (BROWSE `_datafiles/world/dogmud/items/` for valid tailoring/clothing itemids, verify each exists) | humanoid |
| 9374 | Guard-Captain Doryn | 6016 | a parade-perfect physique | none | humanoid, bloodline_domestic |
| 9375 | A Liveried Footman | 6002 | — | — | humanoid, bloodline_domestic |
| 9376 | A Noble Passer-by | 6019 | — | — | humanoid |
| 9377 | A Watched Servant | 6014 | — | — | humanoid, np_commonfolk |
| 9378 | A Gatehouse Sentry | 6018 | — | — | humanoid, bloodline_domestic |

---

## Task 0: Branch + ID sanity + coordinate groundwork

- [ ] **Step 1: Branch** — `git checkout master && git checkout -b feature/np-noble-district`.
- [ ] **Step 2: Confirm IDs free** — `python tools/id_inventory.py --type rooms | grep -E "noble|next free"` (expect next-free 5925 → 6000–6019 clear); `--type mobs` next-free ≥ 9368.
- [ ] **Step 3: Dump the live coordinate frame** — run the per-zone occupied-cell dump over `new_plymouth_merchant/temple/crafting/common`, confirm the open ground north of Merchant (max y≈91), and solve 6000–6019 into free cells north of 5803, keeping 5803→north→6000 a 1-step exit. Record the assignment.
- [ ] **Step 4: Baseline boot** — boot-test recipe; `errors=0`. Kill server.

## Task 1: Stage A rooms — the Processional approach + watched streets

**Files:** Create `rooms/new_plymouth_noble/6000.yaml`…`6004.yaml` + `zone-config.yaml`; Modify `rooms/new_plymouth_merchant/5803.yaml` (add `north`→6000).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Cluster 1 manifest + the controller's exact coords + Conventions + reference files. Author 6000–6004 + `zone-config.yaml` (name `New Plymouth Noble`, region `Windward Marches`, roomid 6001). The watched-streets atmosphere (surveillance, the chill of the elite quarter, eyes that note who passes) is the through-line. Ferrol's flat (6003) carries his approved-history script (which the gallery will later contradict).
- [ ] **Step 2:** Edit `new_plymouth_merchant/5803.yaml` — add `north:` → roomid 6000 (reciprocal of 6000's `south`). Keep 5803's existing exits (south→5802, east→5804) + prose intact.
- [ ] **Step 3: ansi leak check** — `grep -rnE 'fg="[^"]* [^"]*"' _datafiles/world/dogmud/rooms/new_plymouth_noble/` → none.
- [ ] **Step 4: Boot-verify + cartcheck** — `errors=0`. Re-solve coords on collision.
- [ ] **Step 5: Commit** — `feat(np-noble): Stage A rooms (6000-6004) — the Processional, watched streets, Merchant entry`.

## Task 2: Stage B rooms — admin, gallery, atelier, gated lane, Palace approach

**Files:** Create `rooms/new_plymouth_noble/6005.yaml`…`6019.yaml` (15 rooms).

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with Clusters 2–4 manifest + exact coords + Conventions. The **Art Gallery (6007/6009)** is the LORE PAYOFF — author the paintings (esp. the Old Wing 6009) so a player can read the **suppressed pre-Founding history** in the old pigments (the eight-pointed/orbital symbol, an older settlement) — evocative, a real revelation that contradicts the official founding. The **Palace Gatehouse (6016)** and the **Estate Lane Gatehouse (6011)** are impassable prose-stubs (copy the Merchant 5820 technique — the gate does not open; NO exit beyond). Wenna's garret (6013) and the service court carry the watched-servant unease.
- [ ] **Step 2: ansi leak check** (same) — none.
- [ ] **Step 3: Boot-verify + cartcheck** — clean.
- [ ] **Step 4: Commit** — `feat(np-noble): Stage B rooms (6005-6019) — admin, the gallery, atelier, gated lane, Palace gate`.

## Task 3: Stage C — the 7 anchors (9368–9374) + dialogue

**Files:** Create `mobs/new_plymouth_noble/9368-*.yaml`…`9374-*.yaml` + `dialogue/new_plymouth_noble/9368.yaml`…`9374.yaml`.

- [ ] **Step 1:** Dispatch a content subagent (sonnet) with the Mob Manifest rows 9368–9374 + dialogue SOPs + reference files. Each: Title-Case name, `ConvertForFilename` filename, mutation woven in, `non_combatant:true`/`hostile:false`/`maxwander:0`, `groups:` per manifest, ≥3 discoverable first-person dialogue topics, **no quests**. Only Aurel (9373) has a shop (tailoring; thread 40012 + cloth strip 40007 + verified finished garments) + `craft_support: tailoring`; the others NO shop / NO craft_support. Key dialogue intents: **Caldwin** — cold bloodline functionary, the will executed cleanly, deflects (Horst's superior); **Wenna** — frightened, oblique; a discoverable topic lets slip the **Bloom delivery-house** (an estate where unmarked goods arrive at odd hours) — reachable but fearful, unresolved; **Ferrol** — the approved founding history, polished, which a player who's seen the gallery knows is a script; **Lysha** — the gallery's real story / the cipher RESOLUTION (she reads the old pigments; confirms the pre-Founding truth Dross/Ept/Orin pointed at); **Skell** — immovable, turns all away (the gated lane); **Aurel** — the clothier, dresses the bloodline, cloth from Nessa; **Doryn** — parade-perfect, the Palace is closed, AND a warm topic about his old pit-comrade **Garrick** in the Common (the cross-district beat).
- [ ] **Step 2: Verify filenames** match `<id>-<ConvertForFilename(name)>.yaml`.
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +7; `ValidateShopMobTags` (Aurel) + `AssertCanonical` pass.
- [ ] **Step 4: Commit** — `feat(np-noble): the 7 anchors + dialogue (9368-9374)`.

## Task 4: Stage C — ambient mobs (9375–9378) + room spawns

**Files:** Create `mobs/new_plymouth_noble/9375-*.yaml`…`9378-*.yaml`; Modify room YAMLs to add `spawninfo:`.

- [ ] **Step 1:** Author 4 ambient `non_combatant` mobs (manifest 9375–9378; `behavior_archetype: noncombat_passive`, no shop/craft_support/dialogue; `groups:` per manifest). Per the Merchant ambient pattern.
- [ ] **Step 2: Add `spawninfo:`** placing anchors (9368→6005, 9369→6013, 9370→6003, 9371→6007, 9372→6011, 9373→6010, 9374→6016) + ambient (9375→6002, 9376→6019, 9377→6014, 9378→6018), `respawnrate: "10 real minutes"`.
- [ ] **Step 3: Boot-verify** — `mobs.LoadDataFiles` +4; spawns load.
- [ ] **Step 4: Verify spawn coverage** — `grep "mobid: 93XX"` per room.
- [ ] **Step 5: Commit** — `feat(np-noble): ambient residents (9375-9378) + room spawns`.

## Task 5: Stage D — gallery-cipher lore resolution + Wenna's Bloom beat

**Files:** Modify Lysha (9371) + Ferrol (9370) dialogue + gallery room nouns (6007/6009); Modify Wenna (9369) dialogue + 6013 nouns.

- [ ] **Step 1: Gallery cipher resolution** — ensure Lysha's dialogue + the gallery paintings (6007/6009 nouns) let a player who followed Dross's Temple breadcrumb actually DECODE the pre-Founding truth: the old art shows the eight-pointed/orbital symbol + an older settlement the founding story overwrites; Lysha (color-true eyes) reads what the official narrative hides, and names that Ferrol's tour is the cover. Make the discovery reachable (surface "cipher"/"painting"/"symbol"/"founding" in root/hints; avoid prefix-shadowing). Cross-reference Dross/Orin where natural. A REAL revelation, but the full convergence (the buried city) still waits for the Old Quarter.
- [ ] **Step 2: Wenna's Bloom beat** — ensure Wenna's dialogue surfaces the **Noble delivery-house** breadcrumb (a guarded estate where unmarked goods arrive at odd hours, under Caldwin's eye) pointing toward the Old Quarter — discoverable but fearful, unresolved. Add a corroborating noun in 6013/6014 if helpful.
- [ ] **Step 3: Boot-verify** — no panic; dialogue loads.
- [ ] **Step 4: Commit** — `feat(np-noble): gallery cipher resolves as lore + Wenna's Bloom delivery-house beat`.

## Task 6: Stage D — the Doryn↔Garrick relationship edge

**Files:** Modify `mobs/new_plymouth_noble/9374-*.yaml` (Doryn) + `mobs/new_plymouth_common/9324-garrick_one_hand.yaml`.

- [ ] **Step 1: Add the relationship edge to BOTH mobs.** The schema is a top-level `relationships:` list of `{to: <mobid>, type, subtype}` (see `ashwick/259-delia.yaml`). On Doryn (9374), add:
```yaml
relationships:
  - to: 9324
    type: friend
    subtype: old_comrade
```
And on `9324-garrick_one_hand.yaml` (keep its existing content intact), add the same block pointing back:
```yaml
relationships:
  - to: 9374
    type: friend
    subtype: old_comrade
```
- [ ] **Step 2:** Confirm Doryn's dialogue (from Task 3) speaks warmly of Garrick; optionally add ONE line to Garrick's dialogue mentioning Doryn (keep the Garrick edit minimal — don't rework his tree).
- [ ] **Step 3: Boot-verify** — the relationship edges load with no panic (a relationship `to:` an existing mob 9324/9374 resolves; both exist).
- [ ] **Step 4: Commit** — `feat(np-noble): Doryn<->Garrick old-comrade relationship edge + dialogue cross-refs`.

## Task 7: Stage D — anchor schedules

**Files:** Create `schedules/new_plymouth_noble/np_noble_*.yaml`; Modify each anchor mob to add `schedule_id:`.

- [ ] **Step 1:** Author 24h-contiguous schedules (validators panic on gaps/unreachable; compass/`up`/`down` routing). Beats: Caldwin's office hours (6005); Ferrol's tour circuits (6003 flat → 6007 gallery → 6001 plaza); the gallery hours (Lysha 6007, sleep 6008); the gatehouse watches (Doryn 6016, Skell 6011); Wenna's furtive servant routine (6013 garret → errands at 6014/6002, avoiding 6005); Aurel's atelier (6010). Every `target_room` within the built Noble rooms. Reference `np_merchant_vell.yaml`.
- [ ] **Step 2:** Add `schedule_id:` to each anchor mob (9368–9374).
- [ ] **Step 3: Boot-verify** — `LoadSchedules` +7; no coverage-gap/unreachable panic.
- [ ] **Step 4: Commit** — `feat(np-noble): anchor schedules — office hours, tours, the gatehouse watches`.

## Task 8: Stage D — extend Dobb's circuit to Aurel

**Files:** Modify `patrols/new_plymouth/np_docks_runner_circuit.yaml`; confirm Aurel's StockEntries (Task 3).

- [ ] **Step 1:** Append a waypoint to the patrol so Dobb branches up the Processional to **Aurel's atelier (6010)** (e.g. after the temple gate 5903, or after the Merchant cluster — wherever keeps the strict loop pathto-valid), `arrival_event: np_runner_vendor` (delivers thread 40012 / cloth strip 40007 — both in the import manifest). The new waypoint must be pathto-reachable across the Merchant→Noble seam (5803→6000). Keep depot wp0 zero-dwell. NO `CaravanServedZones` change.
- [ ] **Step 2:** Confirm Aurel (9373) pre-declares thread 40012 + cloth strip 40007 as deliverable shop stock (from Task 3); the import manifest already covers them (no Go change).
- [ ] **Step 3: Boot-verify** — patrol loads with the new waypoint; no patrol-validator/home-fallback panic.
- [ ] **Step 4: Smoke (optional, parked-player method):** activate Dobb at the depot, confirm the extended loop reaches Aurel (low-priority given the proven mechanism + validated route).
- [ ] **Step 5: Commit** — `feat(np-noble): extend Dobb's circuit up the Processional to Modiste Aurel`.

## Task 9: District harness playtest

- [ ] **Step 1:** `/playtest local feature-tester` — drive from the Merchant→Noble seam: visit each anchor, **decode the gallery cipher** (ask Lysha + read the Old Wing paintings; confirm Dross's breadcrumb pays off), find Wenna's Bloom delivery-house beat, try (and fail) to pass the Palace gate + the gated lane, hear Doryn speak of Garrick, observe an anchor on schedule.
- [ ] **Step 2:** Triage; fix blocking/cosmetic inline (`fix(np-noble): …`); log deferred polish.
- [ ] **Step 3: Final boot test** — clean.

## Task 10: Merge to master (hold push)

- [ ] **Step 1:** `git checkout master && git merge --no-ff feature/np-noble-district -m "Merge: New Plymouth Noble Quarter (district 6/7)"`.
- [ ] **Step 2:** Update `project_new_plymouth_build.md` + `MEMORY.md`: Noble done + merged; the gallery cipher resolved as lore; NEXT = District 7 Old Quarter (the FINAL district — the Bloom Trail climax with Deren @ 215 Lintel St, z=−1, where Falk/Vesna/Wenna's Bloom beats + the pre-Founding thread all converge), then the whole-capital pre-push SOP + push. **Do NOT push.**

---

## Self-Review (completed during planning)

- **Spec coverage:** §1 zone/IDs/coords → Task 0 + manifests; §2 layout/entry/stubs
  → Tasks 1–2; §3 anchors → Task 3; §4 faction membership → Task 3 (groups); §5
  gallery cipher lore → Tasks 2 (paintings) / 3 (Lysha) / 5 (resolution); §6 Bloom
  beat + Doryn↔Garrick + Dobb-to-Aurel → Tasks 5 / 6 / 8; §7 schedules → Task 7; §8
  staging A–E → Tasks 1–9; §9 DoD → Tasks 9–10.
- **Placeholder scan:** room/mob bodies are subagent-authored from manifests +
  controller-solved coords (the established content pattern); the relationship-edge
  YAML and the patrol waypoint are literal; Aurel's finished-garment ids are gated
  on "verify exists". No TBD/TODO.
- **Consistency:** rooms 6000–6019, mobs 9368–9378, dialogue-by-mobid, Doryn 9374 ↔
  Garrick 9324, faction groups (bloodline_domestic/np_commonfolk/cooperage_circle —
  all existing), Aurel's deliverable materials (40012/40007, in-manifest) used
  identically across manifests and tasks. No new faction, no Go change.
