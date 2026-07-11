# Newbie Chunk 1 — Phase M Implementation Plan (Mobs + Items)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the eight Pothole Coulee hub NPCs (9100–9107) with visible-mutation descriptions and idle texture, spawn them into their rooms, stock Trader Onna's shop with existing starter goods, and stop at the Phase M review gate.

**Architecture:** Pure YAML: 8 mob files + spawninfo additions on 8 host rooms + manifest-check extension. NO dialogue (Phase D), NO new items, NO schedules (chunk 9). All NPCs non-combatant, statically placed (maxwander 0).

**Working directory (ALL tasks):** `C:/Users/Calabe Davis/workspace/DOGMud/.claude/worktrees/feature+newbie-area` (isolated worktree, branch worktree-feature+newbie-area). Never `git add -A`.

---

## Verified facts (do not re-derive)

- **Mob YAML shape** (models — READ BOTH): `_datafiles/world/dogmud/mobs/ashwick/259-delia.yaml` (townsfolk: behavior_archetype, statpool, hostile false, charm_immune, maxwander 0, groups, idlecommands with interleaved empty strings for pacing, activitylevel, character{name, description, speciesid}) and `_datafiles/world/dogmud/mobs/stillwater/337-smith_brindle.yaml` (shopkeeper: `non_combatant: true`, `behavior_archetype: noncombat_shopkeeper`, `character.shop:` item list with itemid/quantity/quantitymax/price).
- **Filename:** `{mobid}-{ConvertForFilename(name)}.yaml` in `_datafiles/world/dogmud/mobs/pothole_coulee/`.
- **Spawning:** `spawninfo:` list on the ROOM yaml (`- mobid: N`, bare entry = default behavior) — ashwick townsfolk use bare entries.
- **Dialogue is keyed by MOB ID** (`dialogue/<zone>/<mobid>.yaml`) — NOT by the sub-spec's d-400 numbers. No dialogue field exists on mob YAML; Phase D creates `dialogue/pothole_coulee/9100.yaml`–`9107.yaml`. Nothing dialogue-related lands in Phase M.
- **Sub-spec manifest (the design of record):** `docs/superpowers/specs/2026-06-12-newbie-chunk1-hub-subspec.md` §4 — ids, names, roles, rooms, mutation flavors.
- `schedule_id` / `knows_facts` / `relationships`: chunk-9 polish — OMIT in Phase M.
- Existing starter-goods item ids for Onna's stock: verify live ids by reading the Sanctum Basin store mob (find it: grep `shop:` in mobs/sanctum_basin/) and the ranged items 10038 (Sling) / 30064 (Pouch of Shot). Choose ~8-12 stock entries: torches/rations/waterskin-tier basics + 1-2 cheapest melee weapons + the sling + pouch of shot (quiet Spoke-G preview). NO new item specs.
- **No-numbers rule, ≤80-char source lines, scablands voice, mutation-is-normal** all apply to mob descriptions exactly as to rooms.
- speciesid: 1 (human — matches Delia/townsfolk convention; all hub NPCs are Opened humans).

## NPC manifest (sub-spec §4, authoritative)

| Id | File name | Name | Room (spawninfo host) | Role / behavior_archetype | Visible mutation (description must carry it naturally) |
|---|---|---|---|---|---|
| 9100 | 9100-cleric_hadwen.yaml | Cleric Hadwen | 5200 | rite-keeper; `noncombat_questgiver` | Pale chitin plates along the jaw |
| 9101 | 9101-innkeep_tally.yaml | Innkeep Tally | 5205 | innkeeper; `noncombat_questgiver` | Gill-frills at the collar |
| 9102 | 9102-sala_the_mender.yaml | Sala the Mender | 5209 | healer; `noncombat_questgiver` | A third, slow-blinking eye |
| 9103 | 9103-ledger_keeper_croup.yaml | Ledger-Keeper Croup | 5208 | banker; `noncombat_questgiver` | Knuckles of polished horn |
| 9104 | 9104-trader_onna.yaml | Trader Onna | 5207 | shopkeeper; `noncombat_shopkeeper` + `character.shop:` | Skin that shifts like reed-shadow |
| 9105 | 9105-granny_wicker.yaml | Granny Wicker | 5210 | folk healer; `noncombat_questgiver` | Willow-green hair that moves on its own |
| 9106 | 9106-crier_toke.yaml | Crier Toke | 5203 | crier; `noncombat_questgiver` | A second mouth at the throat (the loud one) |
| 9107 | 9107-warden_esk.yaml | Warden Esk | 5215 | portal warden; `noncombat_questgiver` | Eyes like settled embers |

All eight: `hostile: false`, `non_combatant: true`, `charm_immune: true`,
`maxwander: 0`, `statpool:` 36–60 (small, townsfolk-tier; Esk and Hadwen
at the higher end — they guard things), `groups: [humanoid, coulee_folk]`,
`activitylevel:` ~10–14, `idlecommands:` 3–4 emotes with interleaved empty
strings (each NPC's idles must express their role AND their mutation at
least once — e.g. Toke's second mouth murmuring along, Sala's third eye
opening; keep them quiet-craft, not slapstick).

NOTE on behavior_archetype: verify `noncombat_questgiver` exists (Delia
uses it) and `noncombat_shopkeeper` (Brindle uses it). If a plain
`noncombat_villager`-style archetype exists and fits the non-questgiver
NPCs better TODAY (quests come in Phase D), prefer the questgiver
archetype anyway IF it idles harmlessly without dialogue — verify by boot
+ watching one NPC; report. Description lengths: 4–7 sentences, person-
first (their work, their manner), mutation woven in matter-of-factly, no
numbers.

---

### Task 0: Service NPCs — Hadwen, Tally, Sala, Croup (+ room spawns)

**Files:**
- Create: `_datafiles/world/dogmud/mobs/pothole_coulee/9100-cleric_hadwen.yaml`, `9101-innkeep_tally.yaml`, `9102-sala_the_mender.yaml`, `9103-ledger_keeper_croup.yaml`
- Modify: rooms `5200.yaml`, `5205.yaml`, `5209.yaml`, `5208.yaml` (add `spawninfo:` with the bare mobid entry; touch NOTHING else in the room files)

- [ ] **Step 1:** Author the four mob YAMLs per the manifest + binding rules. Personality anchors: Hadwen — patient, unhurried, the man who has explained the same miracle a thousand times and still means it; Tally — brisk warmth, runs the room without seeming to; Sala — quiet competence, speaks like someone taking a pulse; Croup — dry, precise, secretly kind.
- [ ] **Step 2:** Add spawninfo to the four rooms. Boot check (wipe instances → boot ~40s → `mobs.LoadDataFiles loadedCount=234` (230+4), zero panics → verify via log or AI port that the NPCs stand in their rooms → kill → re-wipe).
- [ ] **Step 3:** Commit: `git commit -m "content(newbie-c1): service NPCs — Hadwen, Tally, Sala, Croup"`

### Task 1: Trade + character NPCs — Onna (shop), Wicker, Toke, Esk (+ spawns)

**Files:**
- Create: `9104-trader_onna.yaml`, `9105-granny_wicker.yaml`, `9106-crier_toke.yaml`, `9107-warden_esk.yaml`
- Modify: rooms `5207.yaml`, `5210.yaml`, `5203.yaml`, `5215.yaml` (spawninfo only)

- [ ] **Step 1:** Author the four mob YAMLs. Personality anchors: Onna — sharp-eyed, fair, remembers what you bought last time; Wicker — the old way wearing a kind face, speaks in remedies and rhymes; Toke — both mouths busy, news before you ask; Esk — still, watchful, the calm of someone guarding a door that doesn't need guarding.
- [ ] **Step 2:** Onna's `character.shop:` stock per the verified-facts note (mirror Brindle's entry shape exactly: itemid/quantity/quantitymax/price; price = item value). READ the Sanctum Basin store mob first for the starter-goods id set.
- [ ] **Step 3:** Spawninfo on the four rooms; boot check (`loadedCount=238`); kill; re-wipe.
- [ ] **Step 4:** Commit: `git commit -m "content(newbie-c1): trade + character NPCs — Onna (stocked shop), Wicker, Toke, Esk"`

### Task 2: Manifest-check extension + audits + review artifact

**Files:**
- Modify: `tools/newbie_manifest_check.py` (add NPC assertions)
- Create: `docs/superpowers/specs/newbie-c1-phase-m-walkthrough.txt`

- [ ] **Step 1:** Extend the manifest checker: assert the 8 mob files exist with correct mobid/name/zone/non_combatant/charm_immune/maxwander-0/groups; assert each host room's spawninfo contains its NPC's mobid; assert Onna has a non-empty shop list; assert no mob description contains digits. Run → ALL PASS.
- [ ] **Step 2:** No-numbers grep over the mob dir; `python tools/coord_inventory.py` (still 0 — no coords changed, cheap confidence).
- [ ] **Step 3:** Review artifact via AI port (teleport-per-room method, smoketester admin): for each of the 8 NPC rooms — `look` (room now shows the NPC), `look <npc name>` (the description), and for Onna a `list` (the shop inventory). Capture 2-3 minutes of idle-command output in 2 busy rooms (square + inn) so the user sees the idle texture. ANSI-strip → `docs/superpowers/specs/newbie-c1-phase-m-walkthrough.txt` with per-NPC headers. Kill server; re-wipe instances; no strays.
- [ ] **Step 4:** Commit: `git commit -m "content(newbie-c1): phase-M audits + NPC review walkthrough"`
- [ ] **Step 5: STOP — Phase M review gate.** Do NOT begin dialogue/quests (Phase D is run INLINE by the controller per user direction).

---

## Self-review notes
- Sub-spec §4 fully covered (8 NPCs, mutations, rooms, non_combatant, shop). §8 acceptance: boot items exercised per task; dialogue-dependent items (3,4,7) are Phase D. Schedules/facts/relationships explicitly omitted (chunk 9).
- The d-400 numbering in sub-spec §4 is superseded by the mobid-keyed dialogue convention (recorded above; Phase D uses 9100-9107.yaml).
- Personality anchors are Phase D seeds — dialogue must match these voices.
