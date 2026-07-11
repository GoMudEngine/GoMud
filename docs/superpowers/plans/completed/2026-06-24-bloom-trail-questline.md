# The Bloom Trail Questline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Bloom Trail two-quest chain — Q66 The Addict's Plight (a named
addict → Ysolde detox → the call to find the source) and Q67 The Bloom Trail (the
multi-district expose with an opt-in undercover dosing beat, bittersweet ending) —
using the placed breadcrumb NPCs + the shipped Bloom mechanic.

**Architecture:** Pure content (quest YAML + a new mob + evidence items + dialogue
additions). No Go, no new rooms. The quest engine drives everything via `triggers`
(room_interact / item_give / quest_granted events + has/missing conditions + grant/
give_item/send_text/npc_say actions) and quest flags. Validated by server boot (quests
panic at startup on undeclared flags / bad triggers / missing end-token exclusions) +
a harness playtest. Controller drives all shell; content subagents Write/Edit YAML.

**Tech Stack:** GoMud quest engine, DOGMud world YAML (`docs/schemas/dialogue.md`,
quest examples), `tools/id_inventory.py`, the `/playtest` harness.

**Spec:** `docs/superpowers/specs/2026-06-24-bloom-trail-questline-design.md`.

---

## Conventions for every task (READ FIRST)

- **Branch:** `feature/bloom-trail-questline` (from `master` in Task 0). Commit per
  task; trailer `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Controller drives all shell** (subagents Write/Edit YAML only).
- **Boot test = verification** (quests/dialogue/flags panic at startup):
  ```bash
  rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
  go run . > /tmp/bt_boot.log 2>&1 &
  grep -nE "ERROR:.*PANIC|fatal error:|undeclared flag|quest|grantsQuest" /tmp/bt_boot.log | grep -iE "panic|error|undeclared|warn"
  grep -nE "quests.LoadDataFiles|Server Ready" /tmp/bt_boot.log
  ```
  NOT bare `panic` (gotcha #8). Kill after (`taskkill //F //IM go.exe //T`).
- **THE quest template** (read it — the whole pattern lives here):
  `_datafiles/world/dogmud/quests/63-dock_rat.yaml` — steps (id/description/hint),
  `rewards` (gold/rep_faction/rep_amount/playermessage/roommessage), and the
  `triggers` block: `room_interact`(room+noun) / `item_give`(mob+item) /
  `quest_granted`(quest_token) events; `conditions: {has:[], missing:[]}`; `actions:`
  `grant`/`give_item`/`send_text`/`room_text`/`npc_say`. The **auto-complete** pattern
  (report→end) is a `quest_granted` trigger on the report token → `grant: end`.
- **Flags** (`_datafiles/world/dogmud/quests/11-the_windwardens_dilemma.yaml`):
  ```yaml
  flags:
    - key: <flagname>
      values: [a, b]
      description: "..."
  ```
  Flag key convention `"{questId}-{flagname}"`. Dialogue: `setsQuestFlag: {key, value}`,
  `questFlagRequired: {key: value}`. Quest engine: `has_flag`/`missing_flag` conditions,
  `set_flag` action. **Undeclared flag refs PANIC at boot.**
- **Dialogue grant** (`grantsQuest` in a tree node/pattern): MUST include the quest's
  **end token** in `questExcluded` (e.g. `grantsQuest: "66-start"` →
  `questExcluded: ["66-start", "66-end"]`), and `"quest"`+`"task"` in the node's
  `triggers`/`keywords`. `grantsQuest` does NOT fire `quest_granted` — so report/auto
  steps use **`item_give` triggers** or the `grant` action (which DOES fire
  quest_granted). Prefer `questRequired`/`questFlagRequired` over `requires`.
- **give.go gotcha:** a given item transfers to the mob BEFORE handlers fire; the Dock
  Constable KEEPING the case-file is correct (no return_item needed).
- **Reward rep:** the block grants ONE positive `rep_faction`/`rep_amount`. For the
  bloom_trade-DOWN (§reward), VERIFY at build whether `rep_amount: -N` is honored OR a
  trigger `rep`/`faction` action exists; if neither, ship dockfolk-up + narrative
  disruption and note bloom_trade-down deferred (do NOT block on it).
- **Authoring gotchas:** Title-Case mob `name:` (collision-check vs roster); no `": "`
  in YAML values (→ `" — "`); noun ansi `fg="itemname"`; only verified item ids;
  dialogue 1st-person / hints narrator / discoverable triggers.

---

## ID allocation (pre-assigned)

| Kind | ID | Assignment |
|------|-----|------------|
| Quests | **66** The Addict's Plight, **67** The Bloom Trail | 67 gated on 66-end |
| Mob | **9392** | the named addict (anchor) |
| Items | **40110** spent wax wrapper (evidence), **40111** the case-file | quest props |
| Quest flags | `66-addict-fate` [saved, lost] · `67-entry` [undercover, evidence] | declared in the quest YAML |
| Factions | bloom_trade / np_dockfolk / np_commonfolk (existing) | rep shifts |
| Dialogue | edits to 9305/9345/9369/9379/9380/9323/9316 + new 9392 | the trail nodes |

---

## Task 0: Branch + ID sanity
- [ ] **Step 1:** `git checkout master && git checkout -b feature/bloom-trail-questline`.
- [ ] **Step 2:** Confirm IDs free — `python tools/id_inventory.py --type mobs | grep "next free"` (≥9392); `--type items` (≥40110); quests: `ls _datafiles/world/dogmud/quests/ | sed 's/-.*//' | sort -n | tail` (66/67 free).
- [ ] **Step 3: Baseline boot** — clean (quests load, Server Ready, no panic). Kill.

## Task 1: The named addict (mob 9392) + dialogue + spawn + the Q66 hook
**Files:** Create `mobs/new_plymouth_docks/9392-<name>.yaml` + `dialogue/new_plymouth_docks/9392.yaml`; Modify a Docks waterfront room to add the spawn.
- [ ] **Step 1: Pick + collision-check the name** — controller greps the mob roster (`grep -rho "name: .*" _datafiles/world/dogmud/mobs/ | sort -u`) for the chosen name (e.g. "Teels" / "Hask"); pick a collision-free Title-Case name. Hand it to the subagent.
- [ ] **Step 2:** Dispatch a content subagent (sonnet). Author mob 9392: `non_combatant: true`, `hostile: false`, `maxwander: 0` (or 1 for a small drift), `groups: [humanoid, np_dockfolk]` (or np_commonfolk), a `character:` block with the name + a description weaving in a **Bloom-exposure mutation** (faint copper veining beginning — echoes Deren/Vane), modest stats. A few craving/lucid `idlecommands:`. Then dialogue (`9392.yaml`, model on a quest-granting dialogue + the Marn 9305 register): in a lucid moment the addict pleads for help; a node with **`grantsQuest: "66-start"`**, `triggers` including `"quest"`,`"task"`,`"help"`,`"bloom"`,`"sick"`, `hints` (narrator), and **`questExcluded: ["66-start", "66-end"]`**. The addict vaguely names "a clean shop — a draper" (→ Marn) as where the Bloom comes from (a soft Q67 foreshadow). NO givesItem here.
- [ ] **Step 3: Spawn** — edit an accessible **Docks waterfront room** (recommend the Cookshop/Dock Street area, e.g. 5510 or 5516 — controller picks a built, early-reachable room with existing spawninfo) to add `- mobid: 9392`, `respawnrate: "10 real minutes"`.
- [ ] **Step 4: Boot-verify** — `mobs.LoadDataFiles` +1; dialogue loads; the `grantsQuest` warns/loads clean (66 not yet defined → the quest is authored next task; if the loader panics on a grantsQuest referencing an undefined quest, author Task 2's quest FIRST then this — note order). 
- [ ] **Step 5: Commit** — `feat(bloom-trail): the named addict (9392) + Q66 hook dialogue + spawn`.

> **Order note:** if Step 4 panics because quest 66 isn't defined yet, swap Tasks 1↔2 (define the quest YAML first). The controller resolves at boot.

## Task 2: Quest 66 (The Addict's Plight) + Ysolde Q66 dialogue
**Files:** Create `quests/66-the_addicts_plight.yaml`; Modify `dialogue/new_plymouth_common/9323.yaml` (Ysolde).
- [ ] **Step 1:** Dispatch a content subagent. Author quest 66 (model on 63-dock_rat.yaml):
  - `flags:` declare `66-addict-fate` values `[saved, lost]`.
  - `steps:` `start` (helping the addict), `escort` (get them to Ysolde — implemented as the item_give below), `detox` (Ysolde treats + warns of relapse-without-source), `end`.
  - `triggers:`
    - `item_give` to Ysolde (mob 9323) of item **40110** (the addict's wax-wrapper — the addict's dialogue Step-2 OR an in-room pickup gives the player 40110; simplest: the addict's grantsQuest node ALSO `givesItem: 40110`, so the player carries the wrapper to Ysolde) with `conditions {has:[66-start], missing:[66-escort]}` → `grant: 66-escort` + `npc_say` (Ysolde recognizes the Bloom) + `grant: 66-detox`.
    - `quest_granted` on `66-detox` → `grant: 66-end` (auto-complete) + the relapse-without-source line bridging to Q67.
  - `rewards:` `gold: 25`, `rep_faction: np_commonfolk`, `rep_amount: 10`, `playermessage` (the addict stabilized; you learned the detox exists; the source must be stopped), `roommessage`.
  - Set `66-addict-fate=saved` on completion (via `set_flag` action on the detox/end trigger).
- [ ] **Step 2: Ysolde dialogue** — add a Q66 node: when the player has `66-start`/`66-escort`, she treats the addict + dispenses understanding of the detox (the Purge 40109 is already `givesItem` via her detox node from the mechanic build — reuse it) + delivers the relapse-without-source line. `questRequired`/`questFlagRequired` gated; preserve her existing detox/lore dialogue.
- [ ] **Step 3:** Update the addict's dialogue (Task 1) so the grant node also `givesItem: 40110` (the wax wrapper to carry to Ysolde). VERIFY 40110 exists (Task 3 authors it — or author 40110 here first; controller sequences).
- [ ] **Step 4: Boot-verify** — quest 66 loads (flags declared, no undeclared-flag panic); the grant/item_give/quest_granted chain is well-formed.
- [ ] **Step 5: Commit** — `feat(bloom-trail): Quest 66 The Addict's Plight + Ysolde detox bridge`.

## Task 3: Evidence items (40110, 40111) + Quest 67 (The Bloom Trail)
**Files:** Create `items/materials-40000/40110-spent_wax_wrapper.yaml`, `40111-the_case_file.yaml`; Create `quests/67-the_bloom_trail.yaml`.
- [ ] **Step 1: Items** — author 40110 (a spent wax wrapper — the copper-flower Bloom trace; `type: object`, `subtype: mundane`, `not_salable: true`, low value, no vendor_categories needed since not_salable) and 40111 (the case-file — the assembled evidence; same shape). Evocative descriptions; no `": "`.
- [ ] **Step 2: Quest 67** — author `67-the_bloom_trail.yaml` (model on 63):
  - `flags:` declare `67-entry` values `[undercover, evidence]`.
  - `steps:` `start` (find the source), `front` (Marn's back room), `lintel` (Falk → the address), `delivery` (Wenna), `source` (215 crime scene + Deren), `witness` (Quill), `report` (Constable), `end`.
  - The `start` grant is on **Ysolde's** dialogue (or the addict's), gated `questRequired: 66-end`, `questExcluded: ["67-start","67-end"]`, `"quest"`/`"task"` triggers — authored in Task 4's dialogue, but the quest YAML defines the steps/flags/rewards/triggers here.
  - `triggers:` the step chain via `item_give`/`quest_granted`/`room_interact` (the bulk of the dialogue-driven advances are authored as dialogue `grantsQuest`-style nodes in Task 4; the quest YAML holds the auto-complete `quest_granted` chain report→end + the `report` item_give of 40111 to the Constable 9316 → grant report + npc_say + set the bittersweet beat).
  - `rewards:` `gold: 75`, `rep_faction: np_dockfolk`, `rep_amount: 20`, `playermessage` (the trade disrupted, Deren shielded by the bloodline office — bittersweet; the addict saved), `roommessage`. **bloom_trade-down:** attempt `rep_amount: -N` on a second mechanism if the engine supports it (controller verifies at boot/playtest); else narrative-only (note it).
- [ ] **Step 3: Boot-verify** — items +2; quest 67 loads (flags declared); 67 gating on 66-end resolves.
- [ ] **Step 4: Commit** — `feat(bloom-trail): evidence items (40110-40111) + Quest 67 The Bloom Trail`.

## Task 4: The trail dialogue additions (the step nodes + the entry branch)
**Files:** Modify dialogue: `9305` (Marn, Docks), `9345` (Falk, Merchant), `9369` (Wenna, Noble), `9379` (Deren, OQ), `9380` (Quill, OQ), `9323` (Ysolde — the Q67 grant), `9316` (Dock Constable). Possibly room_interact triggers in the quest YAML for in-room evidence.
- [ ] **Step 1:** Dispatch a content subagent with the full step map (spec §4) + the Dock Rat trigger pattern. Wire each trail NPC's quest-aware node (gated by `questRequired`/`questFlagRequired`, preserving existing lore dialogue):
  - **Ysolde (9323):** `grantsQuest: "67-start"` after `66-end` (the source-hunt call). `questExcluded: ["67-start","67-end"]` + quest/task triggers.
  - **Marn (9305):** the back-room gate + **the branch**. Two nodes:
    - undercover: if the player doses Bloom (or chooses the dose option) → `setsQuestFlag: {key: "67-entry", value: "undercover"}` + advance `front` (the dose itself is the mechanic — Marn's node accepts a player who's visibly used; gate on the Bloom Communion buff (90) being active OR a dialogue choice that instructs the player to dose). KEEP the mechanic-coupling simple: a dialogue option "show him you're a user" that checks for buff 90 / recent dose, else points them to where to score (the wafer).
    - evidence/alt: present the wax-wrapper (40110) / a skullduggery-or-search option → `setsQuestFlag: {key:"67-entry", value:"evidence"}` + advance `front`.
    Both paths grant the `front` step token. (VERIFY the cleanest in-engine check for "has dosed" — a buff-active condition; if dialogue can't check buffs, use a pure dialogue-choice branch where one option is flavored as dosing.)
  - **Falk (9345):** the existing `property/lintel` pointer → advance `lintel` (the 215 address). `questRequired` the `front` token.
  - **Wenna (9369):** the delivery-house slip → advance `delivery`. Gated on `lintel`.
  - **Deren (9379):** quest-aware confrontation (he's untouchable; the player arrives with the case half-built) → advance `source`; the crime-scene room nouns (6028, built) supply the physical evidence (optionally a `room_interact` trigger gives 40111 the case-file here, or the case-file is assembled at `witness`/given by the Constable flow).
  - **Quill (9380):** the traffic-testimony → advance `witness` + (give 40111 the assembled case-file if not already). Gated on `source`.
  - **Dock Constable (9316):** receives 40111 (item_give, authored in the Q67 YAML Task 3) → the bittersweet report + disruption.
- [ ] **Step 2: ansi/colon-space checks** on edited dialogue.
- [ ] **Step 3: Boot-verify** — all dialogue loads; the grant/flag chain is well-formed (no undeclared-flag/missing-exclusion panic/warn).
- [ ] **Step 4: Commit** — `feat(bloom-trail): trail dialogue nodes + the undercover/evidence entry branch`.

## Task 5: The addict's-fate closure (saved path)
**Files:** Modify the addict (9392) dialogue + (if needed) the Q67 `end` flow.
- [ ] **Step 1:** Add the addict's closing beat: once `67-end` (or `66-addict-fate=saved`), the addict — encountered again — is stabilized/recovering (a hopeful node, gated `questFlagRequired: {"66-addict-fate": "saved"}` or `questRequired: 67-end`). The one life the player could touch. (The `lost` darker branch is a STRETCH — ship saved-only; only add `lost` if trivial, per spec §6.)
- [ ] **Step 2: Boot-verify** — clean.
- [ ] **Step 3: Commit** — `feat(bloom-trail): the addict's closing beat (saved)`.

## Task 6: Harness playtest + merge (hold push)
- [ ] **Step 1: `/playtest local feature-tester`** — drive the full chain: find the addict → get Q66 → wax wrapper to Ysolde → detox → Q67 → Marn (test BOTH entry paths: the undercover dose [verify the high/crash + Ysolde detox out] and the evidence alt) → Falk → Wenna → 215 Lintel/Deren (confirm untouchable) → Quill → report to the Constable → bittersweet end → the addict's saved closure. (smoketester is admin — can inject a wafer / use `teleport` to reach districts fast.)
- [ ] **Step 2: Triage** — fix blocking inline (`fix(bloom-trail): …`); log cosmetics. Confirm rep shifts applied; confirm no re-grant (end-token exclusions work).
- [ ] **Step 3: Final boot test** — clean (quests 66/67 load, flags declared, errors=0).
- [ ] **Step 4: Merge** — `git checkout master && git merge --no-ff feature/bloom-trail-questline -m "Merge: The Bloom Trail questline (Q66 + Q67)"`. **Do NOT push** (HELD).
- [ ] **Step 5: Update memory** — Bloom Trail questline done+merged; the marquee questline ships the Bloom mechanic into a quest. NEXT enrichment = a quest per remaining quest-less district (Gallery Cipher / Pre-Founding Web / Cooperage Circle / Horst) per the synthesis.

---

## Self-Review (completed during planning)

- **Spec coverage:** §1 IDs/flags → Task 0 + manifests; §2 the addict → Task 1; §3 Q66
  → Task 2; §4 Q67 + trail + entry branch → Tasks 3 (quest/items) + 4 (dialogue nodes);
  §5 dialogue additions → Task 4; §6 addict fate → Task 5; §7 staging A–F → Tasks 1–6;
  §8 DoD → Task 6.
- **Placeholder scan:** quest/dialogue bodies are subagent-authored from the concrete
  step map + the Dock Rat trigger template (literal pattern shown); the addict name +
  the rep-down mechanism + the buff-check for the undercover gate are flagged
  "verify/pick at build" (the established content pattern), not TBD. The `lost` branch
  is explicitly a defer-able stretch. No TODO.
- **Consistency:** quests 66/67, mob 9392, items 40110/40111, flags `66-addict-fate`/
  `67-entry`, the trail NPC ids (9305/9345/9369/9379/9380/9323/9316), the
  step-token chain, and the Dock-Rat trigger grammar used identically across tasks.
  67 gated on 66-end throughout. No new faction, no Go, no new rooms.
- **Risk notes:** (1) grantsQuest-references-undefined-quest ordering (Task 1↔2 swap
  note). (2) the undercover gate's "has dosed" check — buff-active condition may or may
  not be expressible in dialogue; the fallback is a pure dialogue-choice branch.
  (3) bloom_trade-down rep mechanism — verify or defer. All three carry explicit
  build-time resolution notes.
