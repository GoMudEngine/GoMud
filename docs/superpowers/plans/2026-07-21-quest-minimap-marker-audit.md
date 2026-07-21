# Quest Minimap-Marker Audit Implementation Plan (phased, revised 2026-07-21)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline, chosen by the user) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Every quest step has a deliberate minimap-marker decision — a real forward marker or an audited `-1` no-marker — with a CI gate enforcing it; and coarse "bundled" steps are split into concrete markable steps where that adds a genuinely useful marker.

**Architecture:** `map_target int` per step: `>0` = room, `-1` = deliberate none, `0` = undecided (gate rejects). A `TestSmoke_*` gate asserts every step resolves `>0` or is `-1`. The audit is a three-way decision (target / split-then-target / `-1`), executed in phases so the safe state lands first and marker quality improves incrementally, **each phase playtested**.

**Tech Stack:** Go 1.25, `internal/questengine`, `boot_smoke_test.go`, quest YAML under `_datafiles/world/dogmud/quests/`, the `mudagent` playtest harness.

Spec: `docs/superpowers/specs/2026-07-21-quest-minimap-marker-audit-design.md`
Branch: `feature/quest-marker-audit`. Mob→room lookup table already built at `scratchpad/mob2room.txt` (611 mobs).

---

## Phase 0 — Mechanism (DONE)

- [x] Resolver `-1` sentinel (`map_target.go`) + unit test (`map_target_test.go`). Committed.
- [x] `Engine.AllQuests()` accessor (`engine.go`) — *uncommitted, lands with Phase 1's gate commit.*
- [x] `TestSmoke_EveryQuestStepHasMarkerDecision` gate (`boot_smoke_test.go`) — *uncommitted; currently RED, produced the 170-step worklist.*

---

## Phase 1 — Two-way audit → gate green (playtested)

Every one of the 170 undecided steps gets a decision. Split candidates are **deferred**
here: they get a temporary `-1` with a `# split candidate — <what to split>` reason so the
gate goes green and Phase 2+ has a worklist.

### Task 1.1: Classify + edit every undecided step (batched by quest id)

**Files:** `_datafiles/world/dogmud/quests/*.yaml` (all in the worklist).

Per quest (id order), for each undecided step read its **advancing trigger** (the trigger
whose `conditions.has` includes this step's token) and decide:

- **Concrete forward destination** — the advancing trigger references a room directly
  (`room_enter`/`room_interact` `room:`) or a single-spawn mob (`ask`/`item_give`/`mob_death`
  `mob:` → look up `mob2room.txt`), and it's a place the player must *travel to*. →
  `map_target: <room>`.
- **Directionless** — `skill_use`/`command` demo, kill/gather a common mob anywhere,
  report to the giver you started beside, tutorial command step. → `map_target: -1` with a
  one-line reason comment.
- **Bundled** — folds fetch + return or several destinations. → `map_target: -1` with a
  `# split candidate — <objectives>` comment (Phase 2+ will split it).

- [ ] **Step 1: Process quests in id-range batches** (e.g. 2–13, 14–21, 28–43, 45–63, 65–87).
  Read each quest; apply the decision to each undecided step; verify room ids against
  `mob2room.txt` / the room file.
- [ ] **Step 2: After each batch, re-run the gate** to watch the count fall:
  `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_EveryQuestStepHasMarkerDecision 2>&1 | grep -E "quest step|marker decision" | tail -3`
- [ ] **Step 3: Fill the coverage ledger** (spec appendix): one row per quest listing each
  step's disposition (`target N` / `inferred` / `-1: reason` / `-1 SPLIT: what`).
- [ ] **Step 4: Commit each batch** (`content(quests): marker audit phase 1 — <id range>`).

### Task 1.2: Land the gate + boot test

- [ ] **Step 1:** Gate green: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_EveryQuestStepHasMarkerDecision -v` → PASS.
- [ ] **Step 2:** Full boot smoke: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_ServerBootsCleanWithRealData -v` → PASS (loadedCount=79, no panic).
- [ ] **Step 3:** `gofmt -l` + `go vet ./internal/questengine/ .` clean; `go test ./internal/questengine/ -count=1` PASS.
- [ ] **Step 4:** Commit accessor + gate + ledger: `test(quests): gate every quest step on a marker decision`.

### Task 1.3: Phase-1 playtest (content gate)

- [ ] **Step 1:** Start the server; drive a local `mudagent` session (reuse the pre-position
  technique: edit a test user's `roomid`/`questprogress`, restore after). For **2–3 quests
  that got new explicit targets**, confirm `Char.Quests` `target_room` matches the room set.
- [ ] **Step 2:** For **2–3 `-1` steps**, confirm `Char.Quests` emits no `target_room` (no marker).
- [ ] **Step 3:** Record findings; fix any wrong room id; re-run the gate. Write a report to
  `tools/playtest/reports/`.

**End of Phase 1: gate green, every step decided, split-candidate worklist produced.**

---

## Phase 2..N — Split bundled steps, then mark (per batch, each playtested)

Batches are enumerated from Phase 1's split-candidate list (unknown until Phase 1 runs —
this is the repeatable template for each batch). Keep batches small (≈3–6 quests) so each
playtest is tractable. **The gate stays green throughout** (replacing a `-1` with target
steps only adds markers).

### Task 2.x: Split-and-mark one batch of quests

For each split-candidate quest in the batch:

- [ ] **Step 1: Design the split.** Identify the sub-objectives and the room each points at.
  E.g. quest 5 "start" → `find_ledger` (tollhouse room) + `return` (Tolva, mob 84 → room 423).
- [ ] **Step 2: Rewrite the steps.** Replace the bundled step with the ordered concrete
  steps (each with `id`, `description`, `hint`, `map_target: <room>`).
- [ ] **Step 3: Rewire the triggers.** Add the intermediate trigger(s) that advance
  sub-step→sub-step (e.g. an `item_gain`/`room_enter` that grants the mid token), and update
  every affected trigger's `conditions.has`/`missing` so the chain flows in order. Preserve
  the final completion trigger.
- [ ] **Step 4: Token-reference safety sweep.** Grep for the quest's tokens outside this file
  — dialogue `questRequired`/`questExcluded`/`grantsQuest`, chain-quest `chain_quest`, other
  quests' `conditions`. A renamed/removed token that another file references is a break:
  `grep -rn "{questId}-{oldStep}" _datafiles/world/dogmud/dialogue/ _datafiles/world/dogmud/quests/`
  Keep the original step ids where they're externally referenced; only *add* new intermediate
  ids. (If an external ref forces it, keep the old id as the final sub-step.)
- [ ] **Step 5: Remove the temporary `-1`** and update the ledger rows for these quests.
- [ ] **Step 6: Boot + gate green** (`DOGMUD_BOOT_SMOKE=1 go test . -run 'TestSmoke_EveryQuestStepHasMarkerDecision|TestSmoke_ServerBootsCleanWithRealData'`).
- [ ] **Step 7: Playtest each split quest END TO END** in the harness — the split changes the
  play flow (new intermediate step advance), so this is the content playtest gate, not
  optional. Confirm each new sub-step advances and its marker points at the right room.
- [ ] **Step 8: Commit the batch** (`content(quests): split bundled steps for markers — <quests>`).

**Repeat until the split-candidate list is exhausted.**

---

## Finish

After the last phase: `superpowers:finishing-a-development-branch` — verify the full affected
suites, then merge `feature/quest-marker-audit` to master `--no-ff` (not pushed; user pushes).

## Self-review

- **Spec coverage:** three-way decision (Task 1.1 + Phase 2 template), gate (Phase 0 + Task 1.2),
  scope exclusions (gate code), ledger (Task 1.3 rows), token-safety on splits (Task 2.x Step 4),
  playtest per phase (Tasks 1.3, 2.x Step 7). Covered.
- **Placeholder scan:** Phase 2+ batch list is intentionally derived from Phase 1 output (can't
  be enumerated earlier); the per-split procedure is exact. No other TBDs.
- **Type consistency:** `map_target`, `conditions.has/missing`, `mob2room.txt`, `Char.Quests
  target_room`, gate/resolver names — all match Phase 0's built code.
