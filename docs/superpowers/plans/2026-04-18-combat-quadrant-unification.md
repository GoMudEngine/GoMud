# Combat Quadrant Unification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close four parity gaps between combat quadrants (PvP/PvM/MvP/MvM) with a small test-locked `fix:` commit, then collapse the four parallel `handle{P,M}Vs{P,M}` functions into a single `handleCombatRound(attacker, defender actions.Actor, ...)` driven by the Actor interface. Behavior is unchanged from end-of-Stage-1 through end-of-Stage-2 — Stage 2 is a pure code reorganization that makes future parity gaps structurally impossible (any new combat logic in the unified handler applies to all four quadrants by default; quadrant divergence requires explicit `IsPlayer()` gating + reason comment).

**Architecture:** 3 commits on `feature/combat-quadrant-unification` off `development`.

1. `fix(combat): close four parity gaps before quadrant unification` — four small bug fixes + 4 new tests in a new file `internal/hooks/NewRound_DoCombat_parity_test.go`. Behavior DOES change (the ConditionShield double-dip reduction goes away for player defenders; the three MvM callbacks start firing).
2. `refactor(combat): introduce handleCombatRound + phase helpers` — ADD-ONLY. New `handleCombatRound` and eight phase helpers land alongside the existing four quadrant handlers. Dispatchers still call the old handlers. All tests pass unchanged.
3. `refactor(combat): switch dispatchers to handleCombatRound; remove quadrant handlers` — flip two dispatcher call sites in `NewRound_DoCombat.go`, delete the four `handle{P,M}Vs{P,M}` functions and their per-quadrant helpers, add one structural routing test. All tests pass unchanged from the end of Stage 1.

The 2a/2b split is bisect-friendly: if a regression appears after Stage 2, 2a's ADD-ONLY nature means the regression can only be in 2b (the dispatcher flip + deletion). If 2a's tests pass but 2b's fail, the divergence is in the dispatcher wiring or a deleted helper that still has a caller elsewhere.

**Tech Stack:** Go 1.25. No new dependencies. Verification via `go build ./...`, `go vet ./...`, `go test ./...` after every commit.

**Spec:** `docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md`

**Branch:** `feature/combat-quadrant-unification` off `development`.

---

## Scope Policy

This work is **scoped by the spec**. Dispositions are locked:

| Area | Disposition |
|------|-------------|
| Four Stage 1 parity gaps | **REAL FIX** in commit 1 (one commit, 4 tests in a new file). |
| `handleCombatRound` + 8 phase helpers | **REAL REFACTOR** in commit 2 (ADD-ONLY; old handlers untouched). |
| Dispatcher flip + quadrant-handler deletion | **REAL REFACTOR** in commit 3 (net line count drops; one new structural test). |
| `Actor` interface extension | **OUT OF SCOPE.** No new methods on `Actor`. Use `actor.GetCharacter()` or type assertion if a phase needs something not on the interface. |
| Combat math / balance | **OUT OF SCOPE.** No changes to `internal/combat/`. |
| Behavior-tree double-emit (`combat_start`) | **OUT OF SCOPE.** Tracked in spec risk #9; revisit only if tests reveal an issue. |
| Companion-assist routing simplification | **OUT OF SCOPE.** Preserved as-is inside Phase 7. |

Do NOT touch anything else. If you want to "improve" something beyond these dispositions, STOP and re-read the spec.

**Carryover scope-creep policy** (from prior passes):

- **Clear bug** surfaced during execution (unambiguous defect, e.g. `AttackResult.Hit` is set in one quadrant but not another for an edge case) → preceding `fix:` commit on the same branch BEFORE the commit that would otherwise demonstrate the issue. Log to `project_pvm_mvp_parity_gaps.md` under a `## Surfaced During Unification Execution` heading.
- **Ambiguous case** (test result unexpected but unclear whether production code or test is wrong) → pause and ask the user. Log to `project_pvm_mvp_parity_gaps.md` under `## Pending Decision`.
- **Dead code spotted incidentally** → `chore:` removal commit, separate from the feature commits. Same memory-file convention.

---

## CRITICAL — Read These Before Every Commit

**Working-tree noise that MUST NEVER be staged** (carried over from 1.5–1.8 and rooms-pass):

- `.claude/settings.local.json` — dirty, ignore
- `internal/usercommands/_datafiles/feedback/bugs.txt` — dirty, ignore
- `internal/usercommands/_datafiles/feedback/suggestions.txt` — dirty, ignore
- `Screenshot 2026-04-17 084513.png` — untracked, ignore

Every `git add` in this plan **enumerates explicit file paths**. NEVER use `git add .` or `git add -A`. If `git status` after a commit shows anything unexpected staged, run `git restore --staged <path>` before proceeding.

**Memory files live OUTSIDE the repo** at:

`C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\`

Edit them directly with Edit/Write. They are NOT tracked by git. NEVER `git add` them. If `git status` shows a `project_*.md` or `feedback_*.md` staged, you have accidentally created in-repo copies — delete those copies and edit the real files at the path above.

**Actor interface must not change in this pass.** `internal/actions/actor.go` is frozen. If a phase needs something not on `Actor`, use `actor.GetCharacter()` to reach the underlying `*characters.Character`, or type-assert (`if u, ok := atk.(*actions.UserActor); ok { ... }`) when you need `*users.UserRecord` directly (e.g., for `user.PartyId`). Both patterns are acceptable per the spec; pick whichever reads more cleanly per call site.

**Behavior is unchanged from end of Stage 1 through end of Stage 2.** Any test that passed at the end of Stage 1 MUST pass at the end of Stage 2. If it doesn't, the Stage 2 refactor broke something — STOP and bisect.

**Commit format:** conventional commits (`fix:`, `refactor:`, `test:`, `docs:`), `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>` trailer, heredoc format per CLAUDE.md + MEMORY.md rules.

**Pre-existing baseline noise to ignore at every task boundary:**

- `.claude/settings.local.json`, `feedback/*.txt`, `Screenshot*.png` working-tree dirt (above).
- `TestRoom_AddTemporaryExit/duplicate_name_rejected` may or may not be passing on `development` at the time Task 0 runs. If it is still pre-existing-FAILING, that failure is OUT OF SCOPE for this pass (per `project_rooms_package_audit_needed.md`). If the rooms-pass has already landed, it will be passing. Document whichever baseline you see in Task 0.

---

## Task 0: Create feature branch + capture baseline

**Files:** none.

**Estimated size:** 0 files changed. Branch creation only.

### Commands

- [ ] **Step 1: Verify you're on `development` and mostly clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: on `development`. Working tree dirty only with the documented noise above. If anything else is dirty, investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/combat-quadrant-unification
```

Expected: `Switched to a new branch 'feature/combat-quadrant-unification'`.

- [ ] **Step 3: Baseline verification — capture the pass/fail snapshot**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... 2>&1 | tee /tmp/combat_unification_baseline.txt
```

Expected: clean build, clean vet. Tests: most pass. **Record any FAIL lines in your execution notes** so they are not later blamed on this work. Likely candidates (depending on which passes have landed):

- `TestRoom_AddTemporaryExit/duplicate_name_rejected` (rooms-package pass) — may or may not be fixed yet.
- Anything else unexpected → STOP and investigate. The baseline must be stable before refactoring.

From this point on, **every commit's test sweep is judged against this baseline.** If a test fails after a commit that was passing in baseline, the commit regressed something.

---

## Task 1: `fix(combat)`: close four parity gaps before quadrant unification

One commit. Four inline fixes in `internal/hooks/NewRound_DoCombat_helpers.go` plus one new test file `internal/hooks/NewRound_DoCombat_parity_test.go` with four tests. Also commits the spec file and this plan file (they're part of this work).

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (4 edits: Gap 1, Gap 2, Gap 3 near lines 1354–1364; Gap 4 at lines 1171–1179)
- Create: `internal/hooks/NewRound_DoCombat_parity_test.go` (new file, 4 tests)
- Add: `docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md` (spec file, already on disk, being committed for the first time)
- Add: `docs/superpowers/plans/2026-04-18-combat-quadrant-unification.md` (this plan, already on disk, being committed for the first time)

**Estimated commit size:** ~60 lines production change + ~250 lines new test file + spec + plan. Medium commit. If it balloons past ~400 production-code lines of change, something is wrong — STOP and re-read the spec.

**Complexity:** Medium. Gaps 1–3 are additive; Gap 4 is a deletion. Test setup for crit-forcing / stat-gain-forcing requires seeding deterministic dice or mock state.

### Discovery

- [ ] **Step 1: Re-read the four target sites**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
# Gap 4 — the ConditionShield block to delete
sed -n '1170,1182p' internal/hooks/NewRound_DoCombat_helpers.go
# Gaps 1-3 — MvM attacker callbacks + defender crit callback
sed -n '1350,1380p' internal/hooks/NewRound_DoCombat_helpers.go
# MvP pattern to mirror for Gap 2 (MvP attacker crit callbacks)
sed -n '287,295p' internal/hooks/NewRound_DoCombat_helpers.go
```

Confirm:
- Gap 4 block at lines 1171–1179 looks like the "Stage 11.4: Minor Shield…" block, guarded by `if roundResult.Hit && defUser.Character.HasCondition(characters.ConditionShield)`.
- Gap 1 target: `handleMobVsMob` at lines 1196–1379; insertion point is AFTER the attack resolves (after line 1311 `combat.RecordAttack(...)` is fine, or immediately before `handleOffhandBreakMobDef` at line 1352 — use **right after `combat.RecordAttack` call at line 1311** to stay symmetric with MvP's ordering).
- Gap 2 target: `handleMobVsMob` attacker crit callbacks — currently lines 1357–1364 fire `OnSkillUse` on hits but not `OnCriticalSuccess`/`OnCriticalFailure`. Add a mirror of MvP's pattern.
- Gap 3 target: `handleMobVsMob` attacker stat gain — currently lines 1355–1356 call `OnStatUse` but discard the return value; need to capture it and emit the MvP-style room message.
- MvP's crit pattern to mirror: check `handleMobVsPlayer` body. Per spec "NewRound_DoCombat_helpers.go:287–295" is the reference — but in the actual current file that range is inside a spell handler; the real MvP crit-callback pattern lives inside `handleMobVsPlayer`. Find it with the grep below.

- [ ] **Step 2: Locate the live MvP attacker-crit pattern**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "OnCriticalSuccess\|OnCriticalFailure" internal/hooks/NewRound_DoCombat_helpers.go internal/hooks/NewRound_DoCombat_resolution.go
```

Expected: matches inside `handlePlayerVsPlayer` (lines 1033–1039 area) and inside `handleMvPProgression` or `handlePvMProgressionAndAggro` in `NewRound_DoCombat_resolution.go`. Use whichever MvP helper contains the per-weapon-hit crit callbacks; that is the template for Gap 2.

- [ ] **Step 3: Locate the MvP stat-gain-room-message pattern for Gap 3**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "MobStatGainMessages\|OnStatUse" internal/hooks/NewRound_DoCombat_helpers.go internal/hooks/NewRound_DoCombat_resolution.go internal/characters/*.go
```

Expected: a `characters.MobStatGainMessages` map/slice plus a usage site in MvP's progression helper where `if mob.Character.OnStatUse("strength", 0) { /* emit room msg */ }`. Copy that call shape verbatim for Gap 3.

- [ ] **Step 4: Confirm the `ConditionShield` double-dip claim by reading `characters/combat.go:161` and `:200`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '155,210p' internal/characters/combat.go
```

Expected: confirms the spec's claim that `ConditionShield` is already added into `GetStandardArmorReduction` (line ~161) and `GetPhysicalMitigation` (line ~200). If it is NOT there, the Gap 4 deletion will cause a real damage-reduction regression — STOP and ask the user.

### Implementation

- [ ] **Step 5: Gap 4 — delete the inline `ConditionShield` block in `handleMobVsPlayer`**

Target: `internal/hooks/NewRound_DoCombat_helpers.go` lines 1171–1179.

Delete this block entirely:

```go
	// Stage 11.4: Minor Shield reduces physical weapon damage (MvP-only —
	// ConditionShield lives on player defenders).
	if roundResult.Hit && defUser.Character.HasCondition(characters.ConditionShield) {
		reduction := int(defUser.Character.GetConditionMagnitude(characters.ConditionShield)) / 2
		if roundResult.DamageToTarget > reduction+1 {
			roundResult.DamageToTarget -= reduction
			roundResult.DamageToTargetReduction += reduction
		}
	}
```

After deletion, `applyCombatDamageBonuses_MvP(&roundResult, mob, defUser, mobRoom)` (currently line 1181) moves up to the slot immediately after `restore()` / `combat.AttackMobVsPlayer`.

**Load-bearing note for the Gap 4 test:** the test `TestMvP_ConditionShieldAppliedOnceNotDoubleDipped` must be written with this sequence:

1. Before any code change, run a throwaway script or test that mirrors MvP's roll path with a known `ConditionShield` magnitude, a fixed RNG seed, and stubbed attack — capture the resulting `defUser.Character.Health` delta. This is the **current (buggy) value that includes the double-dip**.
2. Then delete the inline block.
3. Write the test to assert the damage delta equals `mitigation-layer-reduction only` — i.e., the **smaller** reduction that excludes the inline `/2` double-dip.

Alternatively (cleaner): compute the expected new value from first principles — baseline `DamageToTarget` minus `GetStandardArmorReduction()` alone, with `ConditionShield` counted once through that path. Snapshotting the old buggy value is a belt-and-suspenders check; the test-against-computed-value is the real assertion. Pick whichever the implementer can ground in the test helpers that already exist.

- [ ] **Step 6: Gap 1 — MvM defender `OnCritReceived` callback**

Target: `internal/hooks/NewRound_DoCombat_helpers.go` `handleMobVsMob`. Insert AFTER `combat.RecordAttack(...)` at line 1311 and BEFORE the `for _, buffId := range roundResult.BuffSource` loop at line 1313:

```go
	// Parity: MvM defender OnCritReceived (PvM/MvP/PvP already fire; Gap 1).
	if roundResult.Hit && roundResult.Crit {
		defMob.Character.OnCritReceived("physical", 0)
	}
```

`userId=0` because mobs have no user. `"physical"` matches MvP/PvM's argument.

- [ ] **Step 7: Gap 2 — MvM attacker crit callbacks**

Target: `internal/hooks/NewRound_DoCombat_helpers.go` `handleMobVsMob` at lines 1357–1364. The current block is:

```go
	for _, wh := range roundResult.WeaponHits {
		if wh.Hit {
			mob.Character.OnSkillUse(wh.SkillTag, 0)
		}
	}
	if len(roundResult.WeaponHits) == 0 && roundResult.Hit {
		mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
	}
```

Replace with (mirroring PvP's per-weapon crit pattern):

```go
	for _, wh := range roundResult.WeaponHits {
		if wh.Hit {
			mob.Character.OnSkillUse(wh.SkillTag, 0)
			if wh.Crit {
				mob.Character.OnCriticalSuccess(wh.SkillTag, 0)
			}
		} else if wh.Fumble {
			mob.Character.OnCriticalFailure(wh.SkillTag, 0)
		}
	}
	if len(roundResult.WeaponHits) == 0 && roundResult.Hit {
		mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
	}
```

(If `OnCriticalSuccess` / `OnCriticalFailure` in this codebase only take `skillName string` rather than `(skillName, userId)`, drop the `, 0` — confirm by reading `internal/characters/` in Step 4's grep. PvP's call site at line 1035 is the authoritative signature.)

- [ ] **Step 8: Gap 3 — MvM attacker stat-gain room messages**

Target: `internal/hooks/NewRound_DoCombat_helpers.go` `handleMobVsMob` at lines 1355–1356. The current block is:

```go
	// Stage 38.3: Mob attacker progression (skip room messages for MvM)
	mob.Character.OnStatUse("strength", 0)
	mob.Character.OnStatUse("dexterity", 0)
```

The comment "skip room messages for MvM" is the bug — MvP emits them, so should MvM. Replace with MvP's pattern (copy from wherever MvP does this; likely `handleMvPProgression` in `NewRound_DoCombat_resolution.go`):

```go
	// Parity: MvM attacker stat-gain room messages (MvP already emits; Gap 3).
	if mob.Character.OnStatUse("strength", 0) {
		if msg := characters.MobStatGainMessages(mob.Character.Name, "strength"); msg != `` {
			sendVisualRoomText(mobRoom, msg)
		}
	}
	if mob.Character.OnStatUse("dexterity", 0) {
		if msg := characters.MobStatGainMessages(mob.Character.Name, "dexterity"); msg != `` {
			sendVisualRoomText(mobRoom, msg)
		}
	}
```

(Actual `MobStatGainMessages` call signature may differ — use whatever MvP uses. The code structure is `if OnStatUse returns true { emit MobStatGainMessages room text }`. `sendVisualRoomText(mobRoom, msg)` is already the pattern used throughout this file.)

- [ ] **Step 9: Create `internal/hooks/NewRound_DoCombat_parity_test.go`**

New file. Four tests, all in package `hooks`, matching the existing test style in `hooks_test.go` (testify `assert`, `seedAllRegistries()` helper, deterministic RNG via `dice.SeedDeterministic` or equivalent — check `hooks_test.go` for the exact seeding pattern).

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// TestMvM_DefenderReceivesOnCritReceived locks Gap 1: on an MvM crit hit,
// the defender mob's OnCritReceived callback fires. PvM/MvP/PvP already
// fire this; MvM was missing it before this commit.
func TestMvM_DefenderReceivesOnCritReceived(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Pair two mobs in the same room. Seed RNG / force weapons / boost
	// crit chance so the roll guarantees a crit-hit. (Use whatever
	// deterministic setup other MvM tests in hooks_test.go use — see the
	// TestMobRoundTick_* family for the pattern.)

	atkMob := /* attacker mob, crit-forced */
	defMob := /* defender mob, low defense */

	atkMob.Character.Aggro = &characters.Aggro{
		MobInstanceId: defMob.InstanceId,
		Type:          characters.DefaultAttack,
	}

	beforeCrits := defMob.Character.OnCritReceivedCount() // or whatever counter is exposed

	evt := events.NewRound{RoundNumber: 1}
	handleMobCombat(evt)

	afterCrits := defMob.Character.OnCritReceivedCount()
	assert.Greater(t, afterCrits, beforeCrits, "defender mob did not receive OnCritReceived on MvM crit hit")
}

// TestMvM_AttackerCritCallbacksFire locks Gap 2: on a crit hit, the MvM
// attacker calls OnCriticalSuccess; on a fumble, OnCriticalFailure.
func TestMvM_AttackerCritCallbacksFire(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	atkMob := /* attacker mob, crit-forced */
	defMob := /* defender mob */
	atkMob.Character.Aggro = &characters.Aggro{
		MobInstanceId: defMob.InstanceId,
		Type:          characters.DefaultAttack,
	}

	beforeHits := atkMob.Character.GetCriticalSuccessCount()
	beforeMisses := atkMob.Character.GetCriticalFailureCount()

	// Force a crit-hit.
	evt := events.NewRound{RoundNumber: 1}
	handleMobCombat(evt)

	assert.Greater(t, atkMob.Character.GetCriticalSuccessCount(), beforeHits,
		"MvM attacker OnCriticalSuccess did not fire on crit hit")

	// Separate sub-case: force a crit-fumble.
	// (Re-seed RNG / reset aggro / run again.)
	// assert OnCriticalFailure incremented.
}

// TestMvM_AttackerStatGainEmitsRoomMessage locks Gap 3: when OnStatUse
// returns true, MvM emits the same room message MvP emits.
func TestMvM_AttackerStatGainEmitsRoomMessage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	atkMob := /* attacker mob seeded so OnStatUse returns true — e.g.,
		stat training threshold / mocked progression so "gain" fires */
	defMob := /* defender mob */
	atkMob.Character.Aggro = &characters.Aggro{
		MobInstanceId: defMob.InstanceId,
		Type:          characters.DefaultAttack,
	}

	room := rooms.LoadRoom(atkMob.Character.RoomId)
	recorder := /* install a room-text recorder per hooks_test.go pattern */

	evt := events.NewRound{RoundNumber: 1}
	handleMobCombat(evt)

	// Assert that one of the MobStatGainMessages text fragments appeared
	// in the recorder. Use a substring match on a known-stable phrase
	// from characters.MobStatGainMessages.
	assert.Contains(t, recorder.JoinedText(), "feels stronger" /* or whatever */,
		"MvM did not emit stat-gain room message on strength gain")
}

// TestMvP_ConditionShieldAppliedOnceNotDoubleDipped locks Gap 4: player
// defender damage reduction under ConditionShield equals the mitigation-
// layer amount only (not mitigation + inline /2 double-dip).
func TestMvP_ConditionShieldAppliedOnceNotDoubleDipped(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Set up: mob attacker with a deterministic weapon damage roll.
	// Player defender with ConditionShield at known magnitude, no other
	// mitigation buffs.

	user := /* player defender */
	mob  := /* mob attacker */
	user.Character.AddCondition(characters.ConditionShield, 10, 20.0, "test")
	mob.Character.Aggro = &characters.Aggro{UserId: user.UserId, Type: characters.DefaultAttack}

	healthBefore := user.Character.Health

	evt := events.NewRound{RoundNumber: 1}
	handleMobCombat(evt)

	damage := healthBefore - user.Character.Health

	// Expected: base roll minus mitigation (which already includes the
	// ConditionShield magnitude). Derive from characters/combat.go.
	expectedDamage := /* computed: baseRoll - GetStandardArmorReduction */

	assert.Equal(t, expectedDamage, damage,
		"ConditionShield was double-dipped: damage should equal mitigation-only amount")
}
```

**Pseudocode-y placeholders above are intentional:** the implementer must ground them in the real mob/user seeding helpers that already exist in `hooks_test.go` (look for `seedAllRegistries`, `mobs.GetInstance(100)`, deterministic dice patterns, and any room-text recorder already in use). Do NOT invent a new mocking framework — copy from the existing tests.

**Test ordering:** all four tests are independent; each cleans up via `defer cleanup()`. No shared state between tests.

### Verify

- [ ] **Step 10: Build + vet + scoped test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/... -run "TestMvM_|TestMvP_Condition" -v
```

Expected: four new tests pass.

- [ ] **Step 11: Full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: only the baseline failures noted in Task 0 Step 3 remain. No new regressions. **If any test that passed in baseline now fails, STOP** — the Stage 1 behavior changes broke something.

Likely regression sources if something fails:
- A test that asserts MvP damage-to-player with `ConditionShield` applied — may need its expected-damage value updated (the double-dip is gone).
- A test that asserts `OnCritReceived` or `OnCriticalSuccess` counts across MvM — may now see a higher count than before (by design; update the expected value or look for assertion inequality bugs).

### Commit

- [ ] **Step 12: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/hooks/NewRound_DoCombat_helpers.go \
  internal/hooks/NewRound_DoCombat_parity_test.go \
  docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md \
  docs/superpowers/plans/2026-04-18-combat-quadrant-unification.md
git commit -m "$(cat <<'EOF'
fix(combat): close four parity gaps before quadrant unification

Three MvM-side callback gaps + one legacy MvP double-dip:
  - MvM defender: OnCritReceived on crit hits (PvM/MvP/PvP already fire)
  - MvM attacker: OnCriticalSuccess/OnCriticalFailure on crit rolls
  - MvM attacker: stat-gain room messages on OnStatUse true
  - MvP: delete the inline ConditionShield damage reduction — it is
    already counted by the mitigation layer (characters/combat.go:161,
    200); the inline block was a Stage 11.4 leftover that silently
    double-counted the reduction for player defenders only.

These were noticed during combat-quadrant unification scoping and are
pre-fixed here so the unification commit can be a pure refactor.

Four new tests in internal/hooks/NewRound_DoCombat_parity_test.go lock
the new behavior:
  - TestMvM_DefenderReceivesOnCritReceived
  - TestMvM_AttackerCritCallbacksFire
  - TestMvM_AttackerStatGainEmitsRoomMessage
  - TestMvP_ConditionShieldAppliedOnceNotDoubleDipped

Also adds the approved design spec and the implementation plan to
docs/superpowers/ for future reference.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

Stage ONLY the four enumerated paths. Do NOT stage working-tree noise. Do NOT stage any memory files (they live outside the repo).

### Spec-Compliance Checklist (Task 1)

Reviewer uses this to confirm this commit matches the spec:

- [ ] Gap 1 fix matches the spec exactly: `defMob.Character.OnCritReceived("physical", 0)` inside `handleMobVsMob`, gated on `roundResult.Hit && roundResult.Crit`.
- [ ] Gap 2 fix matches MvP's pattern (spec says "Skill name should be derived the same way MvP derives it").
- [ ] Gap 3 fix uses `characters.MobStatGainMessages` (or equivalent) and emits via `sendVisualRoomText` on the attacker's room.
- [ ] Gap 4 deletion: the nine-line block at 1171–1179 is gone, and the call-through to `applyCombatDamageBonuses_MvP` is now the first thing after the moon-mods restore.
- [ ] `characters/combat.go:161` and `:200` both still include `ConditionShield` in the mitigation layer (confirmed in Step 4 above — the deletion would be unsafe if they didn't).
- [ ] Four new tests in the new file `internal/hooks/NewRound_DoCombat_parity_test.go`. No other test files modified.
- [ ] Spec file and plan file both committed.

### Code-Quality Checklist (Task 1)

Standard quality checks, distinct from spec compliance:

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches baseline (no new failures).
- [ ] No dead code left behind (the Gap 4 block is fully removed, not commented out).
- [ ] Comments on the new callbacks reference "parity" and "Gap N" so future readers can grep.
- [ ] Commit message follows conventional-commits format and uses heredoc with the Co-Authored-By trailer.

### Rollback Plan (Task 1)

If Stage 1 breaks something subtle, `git revert` this commit and reassess. The four gaps are independent, so if only one fix is suspect:

```bash
git revert HEAD                # undo the whole commit
# re-apply three of the four fixes minus the suspect one, re-test
```

---

## Task 2: `refactor(combat)`: introduce `handleCombatRound` + phase helpers (Stage 2a, ADD-ONLY)

Pure addition. New `handleCombatRound` function and eight phase helpers land in the `hooks` package. The existing four `handle{P,M}vs{P,M}` functions remain untouched. Dispatchers still call the old handlers. **Net diff is +lines only** — this commit adds no test failures and changes zero behavior. After this commit the new code is parallel and unused.

**Files:**
- Create: `internal/hooks/NewRound_DoCombat_unified.go` (new file containing `handleCombatRound` and the eight phase helpers; splitting into a new file keeps the add-only diff reviewable and avoids conflicting with the existing four-handler structure).
- Modify (optional): `internal/hooks/NewRound_DoCombat_helpers.go` and/or `internal/hooks/NewRound_DoCombat_resolution.go` — allowed ONLY if a helper needs to be promoted to an exported/unexported shared signature (e.g., if `processDefenderProgression` already works and stays put, no change). Prefer zero changes here; if a change is unavoidable, keep it to import-visibility or signature.

**Estimated commit size:** ~600–900 lines added in `NewRound_DoCombat_unified.go`. If you go over 1200 lines in one file, split into two (e.g., `NewRound_DoCombat_unified.go` for phases 0–4 and `NewRound_DoCombat_unified_resolution.go` for phases 5–8). If a single phase helper is over ~150 lines, that is punching out of weight class — STOP and reconsider the decomposition.

**Complexity:** High. This is the meat of the refactor. The quadrant routing rule is `atk.IsPlayer()` / `def.IsPlayer()` checks at the leaf site; no `Quadrant` enum is introduced. Party-assist is special-cased on `def.GetCharacter().PartyId` (per spec), not on `def.IsPlayer()`.

### Discovery

- [ ] **Step 1: Re-read each quadrant handler to build the phase-merge mental map**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
# PvP (152 LOC)
sed -n '913,1065p' internal/hooks/NewRound_DoCombat_helpers.go
# PvM (38 LOC main + resolution helpers)
sed -n '1068,1106p' internal/hooks/NewRound_DoCombat_helpers.go
# MvP (86 LOC main + resolution helpers)
sed -n '1107,1193p' internal/hooks/NewRound_DoCombat_helpers.go
# MvM (184 LOC)
sed -n '1195,1379p' internal/hooks/NewRound_DoCombat_helpers.go
# Existing MvP/PvM resolution helpers
sed -n '1,530p' internal/hooks/NewRound_DoCombat_resolution.go
```

You are merging four linear procedures. Build a table in your head of which lines each quadrant owns for each of the eight phases, then write the phase helper that covers all four. Use `atk.IsPlayer()` / `def.IsPlayer()` to gate the divergences listed in the spec's "Intentional Divergences" section.

- [ ] **Step 2: Confirm `actions.Actor` is importable from `internal/hooks`**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "internal/actions" internal/hooks/*.go | head -5
```

Expected: at least one existing import of `actions` from `hooks`. If not, add `"github.com/GoMudEngine/GoMud/internal/actions"` to `NewRound_DoCombat_unified.go`'s imports and verify `actions.NewUserActor` / `actions.NewMobActor` (or equivalent constructors) exist — check `internal/actions/actor_user.go` and `actor_mob.go`.

### Implementation

- [ ] **Step 3: Create `internal/hooks/NewRound_DoCombat_unified.go`**

The file skeleton:

```go
package hooks

import (
	// stdlib
	"fmt"
	"math"

	// internal
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// handleCombatRound is the unified combat-round handler that replaces the
// four quadrant handlers. Routes per-phase via attacker/defender IsPlayer()
// checks at the leaf of each phase; no Quadrant enum is introduced.
//
// Behavior is IDENTICAL to the old quadrant handlers at parity (see
// Stage 1 fixes in NewRound_DoCombat_parity_test.go). Any new combat
// logic added here applies to all four quadrants by default; quadrant-
// specific logic must be explicitly gated on atk.IsPlayer() /
// def.IsPlayer() (or atk.GetCharacter() inspection) AND commented with
// the reason for divergence.
func handleCombatRound(
	atk actions.Actor,
	def actions.Actor,
	evt events.NewRound,
	moonMod float64,
	affectedPlayerIds *[]int,
	affectedMobInstanceIds *[]int,
) {
	// Phase 0: resolve the defender from atk.Aggro; validate alive/in-room.
	if !resolveCombatTarget(atk, def) {
		return
	}

	// Phase 1: already unified — use the existing handleCombatWaitRound.
	if phase1WaitRound(atk, def) {
		return
	}

	// Phase 2: roll the attack.
	res := rollCombatAttack(atk, def, moonMod)

	// Phase 3: damage bonuses (Adrenaline, ConvictionSurge, return, lifesteal).
	applyCombatDamageBonuses(atk, def, &res)

	// Combat analytics (shared across all four quadrants — not a phase,
	// but also not divergent, keep inline).
	recordCombatAnalytics(atk, def, res, evt.RoundNumber)

	// Phase 4: crit effects + messaging dispatch.
	dispatchCritAndMessaging(atk, def, &res, evt.RoundNumber)

	// Phase 5: progression (attacker stats/skills/crit callbacks, defender
	// OnCritReceived + dodge/parry/block via processDefenderProgression).
	applyCombatProgression(atk, def, &res)

	// Phase 6: defender behavior trigger (mob_hurt).
	fireDefenderBehaviorTrigger(atk, def, res)

	// Phase 7: aggro + companion/party assist.
	handleAggroAndAssist(atk, def, res)

	// Phase 8: round resolution (death handling, retarget messaging).
	resolveCombatRound(atk, def, &res, affectedPlayerIds, affectedMobInstanceIds)
}
```

Then the eight phase helpers below. Each sketch names which quadrants' logic it merges and which divergences are explicitly gated.

- [ ] **Step 4: Write Phase 0 — `resolveCombatTarget`**

```go
// resolveCombatTarget looks up the defender from atk's Aggro and
// validates it's alive and reachable. Returns true if combat should
// proceed, false to skip this round.
//
// Merges: the first ~15 lines of each of handlePlayerVsPlayer /
// handlePlayerVsMob / handleMobVsPlayer / handleMobVsMob that resolve
// defUser/defMob from user.Character.Aggro and bail on nil/room-mismatch.
func resolveCombatTarget(atk, def actions.Actor) bool {
	// The caller has already resolved `def` from atk.Aggro — this helper
	// just sanity-checks room membership and liveness.
	if def == nil || def.GetCharacter() == nil {
		atk.GetCharacter().EndAggro()
		return false
	}
	if def.GetCharacter().Health < 1 {
		atk.GetCharacter().EndAggro()
		return false
	}
	// Cross-room attacks are only legal for PvP chase-through-exit.
	if atk.GetCharacter().RoomId != def.GetCharacter().RoomId {
		if atk.IsPlayer() && def.IsPlayer() {
			// PvP-specific: if Aggro.ExitName resolves to defender's room,
			// allow the attack. (See original handlePlayerVsPlayer lines 922-932.)
			aggro := atk.GetCharacter().Aggro
			if aggro != nil && aggro.ExitName != `` {
				if uRoom := atk.GetRoom(); uRoom != nil {
					if _, exitRoomId := uRoom.FindExitByName(aggro.ExitName); exitRoomId == def.GetCharacter().RoomId {
						return true
					}
				}
			}
		}
		atk.GetCharacter().EndAggro()
		return false
	}
	return true
}
```

Divergences gated: cross-room chase is PvP-only (original handler code only checks `exitRoomId` in PvP). Other quadrants simply bail on room mismatch.

- [ ] **Step 5: Write Phase 1 — `phase1WaitRound`**

Already unified via `handleCombatWaitRound` (spec says "already unified"). Phase 1 is a thin adapter:

```go
// phase1WaitRound delegates to the pre-existing handleCombatWaitRound
// which already handles all four quadrants. Returns true if the round
// was consumed by waiting.
func phase1WaitRound(atk, def actions.Actor) bool {
	atkRoom := atk.GetRoom()
	defRoom := def.GetRoom()
	// Map Actor → (concrete pointer, ActorType) for the existing helper's
	// signature. (Inspect combat_shared_helpers.go for handleCombatWaitRound's
	// actual signature; this sketch assumes it already takes Actor. If it
	// doesn't, unwrap via type assertions.)
	return handleCombatWaitRound(
		atk.GetCharacter(), def.GetCharacter(),
		actorType(atk), actorType(def),
		userRecordOrNil(atk), userRecordOrNil(def),
		atkRoom, defRoom,
		def.GetUserId(),
	)
}
```

`actorType(a)` returns `combat.User` or `combat.Mob` based on `a.IsPlayer()`. `userRecordOrNil(a)` returns the `*users.UserRecord` when `a.IsPlayer()` else `nil`. Both small helpers live in this file.

- [ ] **Step 6: Write Phase 2 — `rollCombatAttack`**

```go
// rollCombatAttack runs the moon-mod-wrapped attack roll and returns the
// AttackResult. Polymorphic over combat.Attack{P,M}vs{P,M}.
//
// Merges: the 4 typed Attack* calls at:
//   - handlePlayerVsPlayer line 991 (AttackPlayerVsPlayer)
//   - handlePlayerVsMob    line 1098 (AttackPlayerVsMob)
//   - handleMobVsPlayer    line 1168 (AttackMobVsPlayer)
//   - handleMobVsMob       line 1245 (AttackMobVsMob)
func rollCombatAttack(atk, def actions.Actor, moonMod float64) combat.AttackResult {
	restore := applyMoonMods(atk.GetCharacter(), moonMod)
	defer restore()

	switch {
	case atk.IsPlayer() && def.IsPlayer():
		return combat.AttackPlayerVsPlayer(asUser(atk), asUser(def))
	case atk.IsPlayer() && !def.IsPlayer():
		return combat.AttackPlayerVsMob(asUser(atk), asMob(def))
	case !atk.IsPlayer() && def.IsPlayer():
		return combat.AttackMobVsPlayer(asMob(atk), asUser(def))
	default:
		return combat.AttackMobVsMob(asMob(atk), asMob(def))
	}
}
```

`asUser(a)` / `asMob(a)` are small unchecked type-asserts (safe because `IsPlayer()` already gated the branch). Put them at the bottom of the file.

- [ ] **Step 7: Write Phase 3 — `applyCombatDamageBonuses`**

Merges `applyCombatDamageBonuses_PvM` and `applyCombatDamageBonuses_MvP` (both in `NewRound_DoCombat_resolution.go`) plus the inline PvP Conviction-Surge block (helpers.go:993–1001) and the MvM inline blocks (helpers.go:1247–1304). All four quadrants do the same sequence: Conviction Surge → Adrenaline Surge → Return damage → Lifesteal.

```go
// applyCombatDamageBonuses applies all damage-layer buffs/debuffs that
// fire after the attack roll: Conviction Surge (DamageBonus flag),
// Adrenaline Surge (mutation), return damage (species + equipment),
// and lifesteal.
//
// Merges the 2 existing helpers (applyCombatDamageBonuses_PvM / _MvP in
// NewRound_DoCombat_resolution.go) and the 2 inline copies (inside
// handlePlayerVsPlayer lines 993-1001 and handleMobVsMob lines 1247-1304).
// All four quadrants do the same sequence; no divergence except actor
// type for health mutation.
func applyCombatDamageBonuses(atk, def actions.Actor, res *combat.AttackResult) {
	if !res.Hit || res.DamageToTarget <= 0 {
		return
	}
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()

	// Conviction Surge: +15% damage on hit when DamageBonus buff flag set.
	if atkChar.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(res.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defChar.Health -= bonusDmg
		res.DamageToTarget += bonusDmg
	}

	// Adrenaline Surge (mutation).
	if mutations.IsAdrenalSurgeActive(atkChar.Mutations, atkChar.Health, atkChar.HealthMax.Value) {
		if surgeBonus := mutations.GetAdrenalSurgeBonus(atkChar.Mutations); surgeBonus > 0 {
			bonusDmg := int(math.Round(float64(res.DamageToTarget) * surgeBonus))
			if bonusDmg < 1 {
				bonusDmg = 1
			}
			defChar.Health -= bonusDmg
			res.DamageToTarget += bonusDmg
		}
	}

	// Return damage (species + equipment).
	returnPct := defChar.StatMod("return_damage")
	if sp := species.GetSpecies(defChar.SpeciesId); sp != nil {
		returnPct += sp.ReturnDamage
	}
	if returnPct > 0 {
		returnDmg := int(float64(res.DamageToTarget) * float64(returnPct) / 100.0)
		if returnDmg > 0 {
			atkChar.Health -= returnDmg
			// Room text for return damage uses quadrant-flavored naming
			// (username vs mobname). Gate on IsPlayer() at each side.
			emitReturnDamageText(atk, def, returnDmg)
		}
	}

	// Lifesteal.
	if lifestealPct := atkChar.StatMod("lifesteal_pct"); lifestealPct > 0 {
		healAmt := int(float64(res.DamageToTarget) * float64(lifestealPct) / 100.0)
		if healAmt > 0 {
			atkChar.Health += healAmt
			if atkChar.Health > atkChar.HealthMax.Value {
				atkChar.Health = atkChar.HealthMax.Value
			}
		}
	}
}
```

`emitReturnDamageText` is a tiny internal helper that picks `<ansi fg="username">` vs `<ansi fg="mobname">` tags based on `IsPlayer()` on each side and sends to the attacker's room via `sendVisualRoomText`. Attacker-excludes-self when attacker is a player (so the attacker doesn't see their own name in the broadcast they also receive as 2nd-person text). Copy the exact text format from whichever current handler reads most cleanly (MvM at helpers.go:1285 is the generic pattern; PvM excludes the player-attacker's userId — spec gap #1 already fixed).

- [ ] **Step 8: Write Phase 4 — `dispatchCritAndMessaging`**

Merges `applyCritEffects` + `dispatchCritEffectsPvM` + `dispatchCritEffectsPvP` + inline MvM messaging at helpers.go:1319–1328 + `dispatchCombatMessages` + `handleMvPCritAndMessaging` + `handlePvMCritAndMessaging`.

```go
// dispatchCritAndMessaging applies crit effects and routes combat
// messages to the correct recipients per quadrant.
//
// Intentional divergences (see spec "Intentional Divergences"):
//   - Attacker text (MessagesToSource): only if atk.IsPlayer()
//   - Defender text (MessagesToTarget): only if def.IsPlayer()
//   - Room broadcasts: always, with text-receiving combatants excluded.
func dispatchCritAndMessaging(atk, def actions.Actor, res *combat.AttackResult, roundNumber uint64) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	atkRoom := atk.GetRoom()
	defRoom := def.GetRoom()

	// Darkness replacement — applies only when at least one side is a
	// player with vision considerations. Mob-on-mob doesn't replace text
	// (mobs have no connection); gate accordingly.
	if atk.IsPlayer() || def.IsPlayer() {
		srcCanSee := canSeeInRoom(atkChar, atkRoom)
		tgtCanSee := canSeeInRoom(defChar, defRoom)
		replaceDarknessMessages(res, srcCanSee, tgtCanSee)
	}

	// Crit effects (status inflict / extra damage / buffs).
	critResult := applyCritEffects(atkChar, defChar, *res, atkRoom)

	// Crit message routing (see spec Intentional Divergences #1).
	if critResult.AttackerMsg != `` && atk.IsPlayer() {
		atk.SendText(critResult.AttackerMsg)
	}
	if critResult.DefenderMsg != `` && def.IsPlayer() {
		def.SendText(critResult.DefenderMsg)
	}
	if critResult.RoomMsg != `` && atkRoom != nil {
		excludes := playerExcludeIds(atk, def)
		atkRoom.SendText(critResult.RoomMsg, excludes...)
	}

	// Buffs dispatch (AddBuff uses the Actor so PvP/PvM/MvP/MvM all work).
	for _, buffId := range res.BuffSource {
		atk.AddBuff(buffId, `combat`)
	}
	for _, buffId := range res.BuffTarget {
		def.AddBuff(buffId, `combat`)
	}

	// Direct messages (MessagesToSource / MessagesToTarget).
	if atk.IsPlayer() {
		for _, msg := range res.MessagesToSource {
			atk.SendText(msg)
		}
	}
	if def.IsPlayer() {
		for _, msg := range res.MessagesToTarget {
			def.SendText(msg)
		}
	}

	// Room broadcasts with excludes.
	excludes := playerExcludeIds(atk, def)
	for _, msg := range res.MessagesToSourceRoom {
		sendVisualRoomText(atkRoom, msg, excludes...)
	}
	for _, msg := range res.MessagesToTargetRoom {
		sendVisualRoomText(defRoom, msg, excludes...)
	}
	sendDarkRoomCombatFallback(atkRoom, excludes...)
	if defRoom != atkRoom {
		sendDarkRoomCombatFallback(defRoom, excludes...)
	}
}
```

`playerExcludeIds(atk, def)` returns `[]int{atk.GetUserId(), def.GetUserId()}` filtered to drop 0s. It lives at the bottom of the file.

- [ ] **Step 9: Write Phase 5 — `applyCombatProgression`**

Merges the four inline progression blocks (PvP helpers.go:1019–1046, MvM helpers.go:1353–1367, and the `handleMvPProgression` / `handlePvMProgressionAndAggro` resolution helpers).

```go
// applyCombatProgression runs attacker-side stat/skill/crit progression
// plus defender-side OnCritReceived and dodge/parry/block progression.
//
// Intentional divergences:
//   - TrackPlayerDamage only when both are players (#6).
//   - Stat-gain room messages use MobStatGainMessages for mob actors
//     and player-flavor text for player actors (#8).
//   - userId argument to OnSkillUse / OnStatUse / OnCritReceived is
//     actor.GetUserId() (0 for mobs).
func applyCombatProgression(atk, def actions.Actor, res *combat.AttackResult) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	atkUid := atk.GetUserId()
	defUid := def.GetUserId()
	atkRoom := atk.GetRoom()

	// Defender PvP-only damage tracking.
	if res.Hit && atk.IsPlayer() && def.IsPlayer() {
		defChar.TrackPlayerDamage(atkUid, res.DamageToTarget)
	}

	// Defender OnCritReceived (applies to all four quadrants as of Gap 1).
	if res.Hit && res.Crit {
		defChar.OnCritReceived("physical", defUid)
	}

	// Offhand break (quadrant-flavored — user/mob variants).
	if def.IsPlayer() {
		handleOffhandBreakUserDef(*res, asUser(def), def.GetRoom())
	} else {
		handleOffhandBreakMobDef(*res, asMob(def))
	}

	// Attacker stat progression with optional room-message emission.
	if atkChar.OnStatUse("strength", atkUid) {
		emitStatGainRoomMessage(atk, "strength", atkRoom)
	}
	if atkChar.OnStatUse("dexterity", atkUid) {
		emitStatGainRoomMessage(atk, "dexterity", atkRoom)
	}

	// Attacker per-weapon skill + crit callbacks.
	for _, wh := range res.WeaponHits {
		if wh.Hit {
			atkChar.OnSkillUse(wh.SkillTag, atkUid)
			if wh.Crit {
				atkChar.OnCriticalSuccess(wh.SkillTag, atkUid)
			}
		} else if wh.Fumble {
			atkChar.OnCriticalFailure(wh.SkillTag, atkUid)
		}
	}
	if len(res.WeaponHits) == 0 && res.Hit {
		atkChar.OnSkillUse(string(skills.UnarmedCombat), atkUid)
	}

	// Defender dodge/parry/block.
	processDefenderProgression(defChar, defUid, *res)

	// Player-defender concentration break.
	if def.IsPlayer() {
		handlePlayerConcentrationBreak(asUser(def), *res, def.GetRoom())
	}
}

// emitStatGainRoomMessage picks mob-flavor vs player-flavor stat-gain
// room text by actor.IsPlayer(). Divergence #8.
func emitStatGainRoomMessage(actor actions.Actor, statName string, room *rooms.Room) {
	if room == nil {
		return
	}
	if actor.IsPlayer() {
		// Player stat-gain text: the player's own "You feel stronger" is
		// sent via OnStatUse internally; no room broadcast for players
		// historically (confirm vs current MvP/PvP code in discovery).
		return
	}
	if msg := characters.MobStatGainMessages(actor.GetName(), statName); msg != `` {
		sendVisualRoomText(room, msg)
	}
}
```

Confirm in Discovery-step whether players get a room broadcast for stat gain — current MvP behavior. If they do, add the player branch; if not, leave as shown. The spec says "MvP/MvM use mob-flavor text via `MobStatGainMessages`; PvM/PvP use player-flavor text" — so both branches need content. Grep the current code for player-side stat-gain text and copy it.

- [ ] **Step 10: Write Phase 6 — `fireDefenderBehaviorTrigger`**

```go
// fireDefenderBehaviorTrigger fires the mob_hurt behavior tree when the
// defender is a mob. No-op for player defenders (players have no
// behavior tree).
//
// Merges: PvM helpers.go:??? (inside handlePvMProgressionAndAggro) and
// MvM helpers.go:1330-1338. Divergence #2 in spec — behavior tree only
// fires for mob defenders.
func fireDefenderBehaviorTrigger(atk, def actions.Actor, res combat.AttackResult) {
	if def.IsPlayer() {
		return // Players have no behavior tree.
	}
	if !res.Hit {
		return
	}
	defMob := asMob(def)
	behaviortree.TryMobBehavior(defMob.InstanceId, behaviortree.EventContext{
		EventType: "mob_hurt",
		MobId:     atk.GetMobInstanceId(), // 0 when attacker is player.
		RoomId:    defMob.Character.RoomId,
	})
}
```

- [ ] **Step 11: Write Phase 7 — `handleAggroAndAssist`**

This is the most divergent phase. Merges PvP line 1010–1020, PvM aggro flow in `handlePvMProgressionAndAggro`, MvP lines 1136–1145, and MvM lines 1340–1352.

```go
// handleAggroAndAssist manages aggro propagation (defender attacks back)
// and companion/party auto-assist.
//
// Intentional divergences (spec):
//   - Mob-aggro-on-attack with exit-walk: only when atk is player (#4).
//     When atk is mob, set Aggro directly (already in same room).
//   - Hostility groups (mobs.MakeHostile) only when atk is player and
//     def is mob (#3). Charisma-scaled duration is player-side only.
//   - Party auto-assist fires when def is in a party. Players can form
//     parties; mobs cannot today. Gate on def.GetCharacter().PartyId
//     (or equivalent) — NOT on def.IsPlayer() (#5). When mob parties
//     become real, this routes through the same code path without
//     changes.
//   - Aggro.Type == Flee is player-only; never check it on a mob
//     attacker (#7).
func handleAggroAndAssist(atk, def actions.Actor, res combat.AttackResult) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	atkRoom := atk.GetRoom()

	// Defender retaliation aggro.
	if defChar.Aggro == nil && defChar.Health > 0 {
		switch {
		case atk.IsPlayer() && !def.IsPlayer():
			// PvM: mob defender sets aggro on player attacker, optionally
			// with exit-walk if attacker entered via a named exit, plus
			// mobs.MakeHostile with charisma-scaled duration.
			defMob := asMob(def)
			defMob.PreventIdle = true
			defChar.Aggro = &characters.Aggro{
				UserId: atk.GetUserId(),
				Type:   characters.DefaultAttack,
			}
			// Hostility group assignment (divergence #3).
			// Copy the charisma-scaled call from the current PvM handler.
			// ...
		case !atk.IsPlayer() && def.IsPlayer():
			// MvP: player defender gets reciprocal aggro on the mob. Skip
			// if player is dead/downed (stale aggro in Shadow Realm guard).
			if defChar.Health > 0 {
				defChar.SetAggro(0, atk.GetMobInstanceId(), characters.DefaultAttack)
			}
			// Party auto-assist (#5) — gated on PartyId.
			if defChar.PartyId != 0 {
				handlePartyAutoAttack(asMob(atk), asUser(def))
			}
		case !atk.IsPlayer() && !def.IsPlayer():
			// MvM: defender mob gets aggro and issues its own attack command.
			defMob := asMob(def)
			defMob.PreventIdle = true
			defChar.Aggro = &characters.Aggro{
				Type: characters.DefaultAttack,
			}
			defMob.Command(fmt.Sprintf("attack #%d", atk.GetMobInstanceId()))
		case atk.IsPlayer() && def.IsPlayer():
			// PvP: mutual aggro is already set elsewhere; nothing to do here.
		}
	}

	// Companion-owner / charmed-mob assist routing (divergence #6).
	switch {
	case atk.IsPlayer() && !def.IsPlayer():
		// PvM: charmed mobs of the player attacker are NOT the case;
		// charmed mobs of the defending mob (if it's charmed) trigger
		// owner assist. Copy from current PvM code.
		handleCompanionOwnerAssist(asMob(def), fmt.Sprintf("@%d", atk.GetUserId()))
	case !atk.IsPlayer() && def.IsPlayer():
		// MvP: charmed mobs of the defending player assist.
		if atkRoom != nil {
			handleCharmedMobAssist(atkRoom, def.GetUserId(), fmt.Sprintf("#%d", atk.GetMobInstanceId()))
		}
	case !atk.IsPlayer() && !def.IsPlayer():
		// MvM: if defending mob is charmed, its owner assists.
		handleCompanionOwnerAssist(asMob(def), fmt.Sprintf("#%d", atk.GetMobInstanceId()))
	case atk.IsPlayer() && def.IsPlayer():
		// PvP: player attacker's charmed mobs assist against the defender.
		if atkRoom != nil {
			handleCharmedMobAssist(atkRoom, def.GetUserId(), fmt.Sprintf("@%d", atk.GetUserId()))
		}
	}

	_ = atkChar // keep if unused after full impl; remove when unneeded.
}
```

The four-way switch is load-bearing because each quadrant has a different aggro/assist flow. Keep the switch explicit rather than trying to collapse; per spec risk #1, grouping divergent logic into named branches with comments is preferred over single-line `IsPlayer()` checks that balloon.

- [ ] **Step 12: Write Phase 8 — `resolveCombatRound`**

Merges the final "death / retarget" blocks: PvP helpers.go:1048–1065, MvP `handleMvPRoundResolution`, PvM `handlePvMRoundResolution`, MvM helpers.go:1369–1378.

```go
// resolveCombatRound handles end-of-round death processing and retarget
// messaging.
//
// Intentional divergence:
//   - Retarget-on-death "You turn your attention to..." message only when
//     the survivor is a player (#9 in spec).
func resolveCombatRound(
	atk, def actions.Actor,
	res *combat.AttackResult,
	affectedPlayerIds *[]int,
	affectedMobInstanceIds *[]int,
) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()

	if atkChar.Health <= 0 || defChar.Health <= 0 {
		defChar.EndAggro()
		if atkChar.Health > 0 {
			// Survivor retargets.
			if atk.IsPlayer() {
				RetargetOrEnd(atkChar, atk.GetRoom(), atk.GetUserId(), 0)
				// Retarget messaging is player-only.
				// (Copy the "You turn your attention to..." block from
				// NewRound_DoCombat.go:94-104 or wherever it currently
				// lives — it's called from the outer dispatcher today.)
			} else {
				RetargetOrEnd(atkChar, atk.GetRoom(), 0, atk.GetMobInstanceId())
			}
		} else {
			atkChar.EndAggro()
		}
	} else if !atk.IsPlayer() {
		// Mob attackers refresh their aggro at end-of-round (MvM/MvP do this).
		if def.IsPlayer() {
			atkChar.SetAggro(def.GetUserId(), 0, characters.DefaultAttack)
		} else {
			atkChar.SetAggro(0, def.GetMobInstanceId(), characters.DefaultAttack)
		}
	}

	// Populate affected-id slices for the outer affectedPlayerIds /
	// affectedMobInstanceIds lists (used by handleAffected to process
	// post-round deaths).
	if atk.IsPlayer() {
		*affectedPlayerIds = append(*affectedPlayerIds, atk.GetUserId())
	} else {
		*affectedMobInstanceIds = append(*affectedMobInstanceIds, atk.GetMobInstanceId())
	}
	if def.IsPlayer() {
		*affectedPlayerIds = append(*affectedPlayerIds, def.GetUserId())
	} else {
		*affectedMobInstanceIds = append(*affectedMobInstanceIds, def.GetMobInstanceId())
	}
}
```

- [ ] **Step 13: Add the small helpers at the bottom of the file**

```go
// asUser unwraps an Actor known to be a player into the concrete
// *users.UserRecord. Panics if misused — only call after IsPlayer() gate.
func asUser(a actions.Actor) *users.UserRecord {
	if u, ok := a.(*actions.UserActor); ok {
		return u.UserRecord // or u.User() — whichever field the adapter exposes
	}
	panic("asUser called on non-player Actor")
}

// asMob is the dual of asUser.
func asMob(a actions.Actor) *mobs.Mob {
	if m, ok := a.(*actions.MobActor); ok {
		return m.Mob
	}
	panic("asMob called on non-mob Actor")
}

// actorType returns the combat.ActorType for an Actor.
func actorType(a actions.Actor) combat.ActorType {
	if a.IsPlayer() {
		return combat.User
	}
	return combat.Mob
}

func userRecordOrNil(a actions.Actor) *users.UserRecord {
	if a.IsPlayer() {
		return asUser(a)
	}
	return nil
}

func playerExcludeIds(atk, def actions.Actor) []int {
	excludes := []int{}
	if id := atk.GetUserId(); id != 0 {
		excludes = append(excludes, id)
	}
	if id := def.GetUserId(); id != 0 {
		excludes = append(excludes, id)
	}
	return excludes
}

// recordCombatAnalytics wraps combat.RecordAttack once for all four
// quadrants.
func recordCombatAnalytics(atk, def actions.Actor, res combat.AttackResult, roundNumber uint64) {
	atkType := "unarmed"
	if atk.GetCharacter().Equipment.Weapon.ItemId > 0 {
		atkType = "weapon"
	}
	combat.RecordAttack(res, actorType(atk), actorType(def), atkType,
		atk.GetCharacter(), def.GetCharacter(), roundNumber)
}
```

Confirm `actions.UserActor` / `actions.MobActor` field names in `internal/actions/actor_user.go` and `actor_mob.go`; adjust the field access accordingly.

### Verify

- [ ] **Step 14: Build + vet (no tests yet — the new code is unreachable)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
```

Expected: clean. The new `handleCombatRound` and its helpers compile but are not called from anywhere. If the build fails, an import is wrong or a signature mismatch — fix in place.

- [ ] **Step 15: Full test sweep**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./...
```

Expected: identical result to end-of-Task-1. **No test should behave differently** — the old handlers are still the only ones being called. If anything changes, something you added has side effects at init time (e.g., a global map mutation) that should not happen.

### Commit

- [ ] **Step 16: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/NewRound_DoCombat_unified.go
git commit -m "$(cat <<'EOF'
refactor(combat): introduce handleCombatRound + phase helpers

Stage 2a of combat-quadrant unification (see spec
docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md).
Pure ADD-ONLY commit. New handleCombatRound(atk, def actions.Actor, ...)
and eight phase helpers land alongside the existing four quadrant
handlers (handlePlayerVsPlayer / handlePlayerVsMob / handleMobVsPlayer /
handleMobVsMob). Dispatchers still call the old handlers; the new code
is parallel and unused at end of this commit.

Phases:
  0. resolveCombatTarget  — defender lookup + alive/in-room validation
  1. phase1WaitRound      — delegates to existing handleCombatWaitRound
  2. rollCombatAttack     — polymorphic combat.Attack{P,M}vs{P,M} wrapper
  3. applyCombatDamageBonuses — Conviction/Adrenaline/return/lifesteal
  4. dispatchCritAndMessaging — crit effects + quadrant-gated text routing
  5. applyCombatProgression   — attacker stats/skills/crits; defender D/P/B
  6. fireDefenderBehaviorTrigger — mob_hurt (no-op for player defenders)
  7. handleAggroAndAssist  — defender retaliation + companion/party assist
  8. resolveCombatRound    — death handling + retarget messaging

Routing strategy: atk.IsPlayer() / def.IsPlayer() checks at the leaf of
each phase. No Quadrant enum. Party assist uses
def.GetCharacter().PartyId (not def.IsPlayer()) so future mob parties
route through the same path without changes.

Behavior is unchanged. All tests pass unchanged from end of Stage 1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

Stage ONLY `internal/hooks/NewRound_DoCombat_unified.go`. Do NOT stage working-tree noise. Do NOT stage memory files.

### Spec-Compliance Checklist (Task 2)

- [ ] `handleCombatRound(atk, def actions.Actor, ...)` exists and takes `Actor`, not concrete types.
- [ ] Eight phase helpers exist with the names in the spec table (`resolveCombatTarget`, `phase1WaitRound` / re-uses `handleCombatWaitRound`, `rollCombatAttack`, `applyCombatDamageBonuses`, `dispatchCritAndMessaging`, `applyCombatProgression`, `fireDefenderBehaviorTrigger`, `handleAggroAndAssist`, `resolveCombatRound`).
- [ ] Routing uses `IsPlayer()` checks. No `Quadrant` enum type introduced anywhere in the file.
- [ ] Party assist gate is `def.GetCharacter().PartyId` (or equivalent), NOT `def.IsPlayer()`.
- [ ] All 10 intentional divergences in the spec are preserved with explicit gate + reason comment.
- [ ] Actor interface at `internal/actions/actor.go` is unchanged (no new methods).
- [ ] Old handlers `handlePlayerVsPlayer` / `handlePlayerVsMob` / `handleMobVsPlayer` / `handleMobVsMob` are UNTOUCHED.

### Code-Quality Checklist (Task 2)

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Full `go test ./...` matches end-of-Task-1 results.
- [ ] No phase helper exceeds ~150 lines; if one does, split it per spec risk #1.
- [ ] Each intentional divergence has a `// Divergence #N:` or similar comment explaining WHY the branch exists.
- [ ] Type assertions (`asUser`, `asMob`) are gated behind `IsPlayer()` checks at every call site.
- [ ] No `interface{}` / `any` usage — stays within the `Actor` interface surface.

### Rollback Plan (Task 2)

If this commit breaks the build or some test regresses unexpectedly:

```bash
git revert HEAD
```

Since the commit is ADD-ONLY, revert is equivalent to deleting the new file. Nothing else is affected.

If a regression appears ONLY in Stage 2b (after Task 3's dispatcher flip), you know the bug is in the dispatcher flip or the deletion — not in Task 2. Revert Task 3, keep Task 2 in place, re-approach the flip.

---

## Task 3: `refactor(combat)`: switch dispatchers, delete old handlers (Stage 2b)

Flip the two dispatcher sites in `NewRound_DoCombat.go` (lines 124–129 and 271–278 per spec scoping pass) to call `handleCombatRound(atk, def)` unconditionally. Delete the four `handle{P,M}Vs{P,M}` functions and their per-quadrant helpers. Add one structural routing test. Full test suite passes unchanged.

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go` (flip two dispatcher blocks)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (delete `handlePlayerVsPlayer`, `handlePlayerVsMob`, `handleMobVsPlayer`, `handleMobVsMob`)
- Modify: `internal/hooks/NewRound_DoCombat_resolution.go` (delete per-quadrant helpers listed below — verify each has no other callers via grep before deleting)
- Create: `internal/hooks/NewRound_DoCombat_routing_test.go` (new file, one `TestHandleCombatRound_AllQuadrantsRouteCorrectly` test)

**Estimated commit size:** net NEGATIVE line delta (~-400 to -600 lines from deletions, +100 from the one new test). If net is positive, too much was moved instead of deleted — investigate.

**Complexity:** Medium. Mechanically straightforward but requires careful grep to confirm no stale callers of deleted helpers.

### Discovery

- [ ] **Step 1: Confirm the two dispatcher sites**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
sed -n '118,135p' internal/hooks/NewRound_DoCombat.go
sed -n '265,285p' internal/hooks/NewRound_DoCombat.go
```

Confirm:
- Lines 122–130 area: two branches, `if user.Character.Aggro.UserId > 0 { handlePlayerVsPlayer(...) }` and `if user.Character.Aggro.MobInstanceId > 0 { handlePlayerVsMob(...) }`.
- Lines 270–278 area: two branches, `if mob.Character.Aggro.UserId > 0 { handleMobVsPlayer(...) }` and `if mob.Character.Aggro.MobInstanceId > 0 { handleMobVsMob(...) }`.

- [ ] **Step 2: Grep-confirm each per-quadrant helper has no callers outside the four handlers**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
for fn in applyCombatDamageBonuses_PvM applyCombatDamageBonuses_MvP \
          handlePvMCritAndMessaging handleMvPCritAndMessaging \
          handlePvMProgressionAndAggro handleMvPProgression \
          handlePvMRoundResolution handleMvPRoundResolution \
          dispatchCritEffectsPvM dispatchCritEffectsPvP \
          dispatchCombatMessages; do
  echo "=== $fn ==="
  grep -n "$fn" internal/hooks/*.go
done
```

Expected: for each helper, the only callers are inside the four `handle{P,M}Vs{P,M}` functions that are about to be deleted. If a helper has an OTHER caller, it stays (note which one and why).

**Heuristic from the scoping pass:**
- `dispatchCombatMessages`, `dispatchCritEffectsPvP`, `dispatchCritEffectsPvM` — likely PvP/MvP internal only, will be deleted. **Verify.**
- `applyCombatDamageBonuses_PvM/_MvP` — merged into Phase 3, delete if no external caller.
- `handle{P,M}vs{P,M}CritAndMessaging` — merged into Phase 4, delete.
- `handle{P,M}vs{P,M}ProgressionAndAggro` / `handle{P,M}vs{P,M}Progression` — merged into Phase 5 + 7, delete.
- `handle{P,M}vs{P,M}RoundResolution` — merged into Phase 8, delete.
- Shared helpers that are NOT per-quadrant (e.g., `handleCombatWaitRound`, `applyCritEffects`, `processDefenderProgression`, `handleOffhandBreakUserDef`, `handleOffhandBreakMobDef`, `handlePlayerConcentrationBreak`, `handleCompanionOwnerAssist`, `handleCharmedMobAssist`, `handlePartyAutoAttack`) — **KEEP**. They are still called from inside the new phase helpers.

### Implementation

- [ ] **Step 3: Flip the player dispatcher (lines 122–130 area)**

Replace the two-branch block:

```go
		// PvP combat
		if user.Character.Aggro != nil && user.Character.Aggro.UserId > 0 {
			handlePlayerVsPlayer(user, uRoom, evt, &affectedPlayerIds)
		}

		// PvM combat
		if user.Character.Aggro != nil && user.Character.Aggro.MobInstanceId > 0 {
			handlePlayerVsMob(user, uRoom, evt, moonMod, &affectedPlayerIds, &affectedMobInstanceIds)
		}
```

with:

```go
		// Unified combat dispatch (replaces PvP/PvM branch).
		if user.Character.Aggro != nil {
			var def actions.Actor
			if user.Character.Aggro.UserId > 0 {
				if defUser := users.GetByUserId(user.Character.Aggro.UserId); defUser != nil {
					def = actions.NewUserActor(defUser)
				}
			} else if user.Character.Aggro.MobInstanceId > 0 {
				if defMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); defMob != nil {
					def = actions.NewMobActor(defMob)
				}
			}
			if def != nil {
				atk := actions.NewUserActor(user)
				handleCombatRound(atk, def, evt, moonMod, &affectedPlayerIds, &affectedMobInstanceIds)
			}
		}
```

(Constructor names `NewUserActor` / `NewMobActor` per `internal/actions/` — adjust to the actual names.)

- [ ] **Step 4: Flip the mob dispatcher (lines 270–278 area)**

Replace the MvP/MvM two-branch block symmetrically with an `actions.Actor`-based dispatch:

```go
		// Unified combat dispatch (replaces MvP/MvM branch).
		if mob.Character.Aggro != nil {
			var def actions.Actor
			if mob.Character.Aggro.UserId > 0 {
				if defUser := users.GetByUserId(mob.Character.Aggro.UserId); defUser != nil {
					def = actions.NewUserActor(defUser)
				}
			} else if mob.Character.Aggro.MobInstanceId > 0 {
				if defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); defMob != nil {
					def = actions.NewMobActor(defMob)
				}
			}
			if def != nil {
				atk := actions.NewMobActor(mob)
				handleCombatRound(atk, def, evt, 0 /* moonMod — mob-side already has moonMod in scope; pass it */, &affectedPlayerIds, &affectedMobInstanceIds)
			}
		}
```

(Use the `moonMod` variable already in scope at line 140 of `handleMobCombat`.)

- [ ] **Step 5: Delete `handlePlayerVsPlayer`, `handlePlayerVsMob`, `handleMobVsPlayer`, `handleMobVsMob`**

From `internal/hooks/NewRound_DoCombat_helpers.go`, delete the four function definitions. Approximate line ranges (post-Task-1 edits may have shifted these by ~5 lines):

- `handlePlayerVsPlayer`: lines ~912–1066
- `handlePlayerVsMob`: lines ~1068–1105
- `handleMobVsPlayer`: lines ~1107–1193 (note: Gap 4's 9 lines are already gone, so this is a bit shorter than the spec's 86-LOC figure)
- `handleMobVsMob`: lines ~1195–1379 (Gap 1/2/3 additions made this slightly longer than the spec's 184-LOC figure)

Delete each function whole. Do NOT leave commented-out code behind.

- [ ] **Step 6: Delete confirmed-unused per-quadrant helpers**

From `internal/hooks/NewRound_DoCombat_resolution.go`, delete helpers confirmed unused in Step 2:

- `applyCombatDamageBonuses_PvM` (~line 33)
- `applyCombatDamageBonuses_MvP` (~line 104)
- `handleMvPCritAndMessaging` (~line 223)
- `handleMvPProgression` (~line 266)
- `handleMvPRoundResolution` (~line 309)
- `handlePvMCritAndMessaging` (~line 393)
- `handlePvMProgressionAndAggro` (~line 434)
- `handlePvMRoundResolution` (~line 513)

From `internal/hooks/NewRound_DoCombat_helpers.go`, delete:

- `dispatchCritEffectsPvP` (~line 566) — only caller was PvP, now gone
- `dispatchCritEffectsPvM` (~line 579) — only caller was `handlePvMCritAndMessaging`, now gone
- `dispatchCombatMessages` (~line 703) — only caller was PvP, now gone

**Only delete helpers that Step 2's grep confirmed have zero remaining callers.** If a helper still has a caller, it stays. Err on the side of KEEPING a helper — dead code removal can be a follow-up `chore:` commit if needed.

- [ ] **Step 7: Add the structural routing test**

New file `internal/hooks/NewRound_DoCombat_routing_test.go`. One test with four sub-cases (UU / UM / MU / MM):

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

// TestHandleCombatRound_AllQuadrantsRouteCorrectly feeds each of the four
// quadrant pairs (UU, UM, MU, MM) through handleCombatRound with
// deterministic dice and asserts:
//   (a) the right callbacks fire on the right side,
//   (b) the right text recipients receive messages,
//   (c) the right behavior triggers fire (mob_hurt only for mob
//       defenders).
func TestHandleCombatRound_AllQuadrantsRouteCorrectly(t *testing.T) {
	type quadrant struct {
		name            string
		atkIsPlayer     bool
		defIsPlayer     bool
		expectAtkText   bool // attacker receives MessagesToSource text
		expectDefText   bool // defender receives MessagesToTarget text
		expectMobHurt   bool // defender mob_hurt behavior fires
		expectPartyTrk  bool // TrackPlayerDamage only on UU
	}
	cases := []quadrant{
		{"UU", true, true, true, true, false, true},
		{"UM", true, false, true, false, true, false},
		{"MU", false, true, false, true, false, false},
		{"MM", false, false, false, false, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := seedAllRegistries()
			defer cleanup()

			// Build atk/def Actors per tc.
			atk, def := /* construct attacker and defender per flags */, /* ... */

			// Force the attack roll to hit (seed RNG / buff dice).
			var (
				affPlayers []int
				affMobs    []int
			)
			evt := events.NewRound{RoundNumber: 1}
			handleCombatRound(atk, def, evt, 0, &affPlayers, &affMobs)

			// Assertion scaffolding — replace with real recorders.
			if tc.expectAtkText {
				assert.NotEmpty(t, atkTextRecorder(atk), "atk should have received text")
			} else {
				assert.Empty(t, atkTextRecorder(atk), "atk should NOT have received text")
			}
			if tc.expectDefText {
				assert.NotEmpty(t, defTextRecorder(def), "def should have received text")
			} else {
				assert.Empty(t, defTextRecorder(def), "def should NOT have received text")
			}
			if tc.expectMobHurt {
				assert.True(t, mobHurtFired(def), "mob_hurt should have fired on mob defender")
			} else {
				assert.False(t, mobHurtFired(def), "mob_hurt must NOT fire on player defender")
			}
			// (Add expectPartyTrk assertion if PartyId wiring is available
			// in the test harness; otherwise note in a comment and defer.)
		})
	}
}
```

**REQUIRED — drive through `handleCombatRound` end-to-end:** The test
MUST invoke `handleCombatRound(atk, def, evt, ...)` as the entry
point. Calling phase helpers (`dispatchCritAndMessaging`,
`applyCombatProgression`, etc.) directly is **NOT acceptable** — that
defeats the whole point of the structural routing test. This is the
integration coverage Stage 1 deferred to Stage 2 (per the user's
agreed plan), and it is the only test in the suite that proves
the unified handler routes correctly across all four quadrants.

If `hooks_test.go` lacks the necessary harness — a deterministic dice
facility, a text-capture recorder per actor, a way to fixture two
combat-ready actors — **build it as part of this task**. Acceptable
patterns:

- Deterministic dice: introduce a test-only seed/swap on `dice.Rand`
  (or whatever the package exposes), or buff stats so attack rolls
  cannot miss. Either is fine; do NOT skip the dice problem by
  asserting only on side effects that don't require a hit.
- Text capture per actor: register an `events.Message` listener that
  routes to per-recipient slices keyed by `Aggro.UserId` /
  `Aggro.MobInstanceId`, or wrap `actor.SendText` / `actor.SendRoomText`
  via a test-scope spy. Either works; pick whichever is least
  invasive.
- Mob-vs-mob fixture: `mobs.GetInstance(100)` and `mobs.GetInstance(101)`
  both seeded to the same `RoomId`. Set Aggro on the attacker.

If genuinely impossible to build (e.g., a dependency you can't seed
from the test side), STOP and escalate to the user. Do NOT fall back
to direct method invocation — that re-creates the Stage 1 weakness.

### Verify

- [ ] **Step 8: Build + vet**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
```

Expected: clean. Common failure modes:
- An unused import after deletion (`mobs` or `behaviortree`) — remove it.
- A deleted helper that actually had a caller — put it back and grep more carefully.
- `actions.NewUserActor`/`NewMobActor` signature wrong — check `internal/actions/actor_user.go`.

- [ ] **Step 9: Scoped test — routing**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/hooks/... -run "TestHandleCombatRound_AllQuadrantsRouteCorrectly" -v
```

Expected: four sub-cases pass (UU, UM, MU, MM).

- [ ] **Step 10: Full test sweep — THIS IS THE CRITICAL ONE**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./... -count=1
```

Expected: identical results to end-of-Task-1 and end-of-Task-2 baselines. **Every test that passed at end of Task 1 must pass here.** If any regress, the refactor broke something — bisect:

1. Check `git diff` to see which helpers got deleted.
2. Is there a subtle behavioral difference in one of the phase helpers vs the old code? Likely culprits: Phase 4 message routing (quadrant text ordering), Phase 7 aggro flow (exit-walk vs direct aggro), Phase 8 retarget messaging.
3. If bisecting is hard, `git revert` Task 3 and keep Task 2 — then re-approach Task 3 with smaller surgical edits.

- [ ] **Step 11: Bench check (optional)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go test ./internal/hooks/... -bench=BenchmarkMobRoundTick -benchtime=3s
```

Expected: within ±10% of pre-refactor numbers. Interface dispatch adds minor overhead; per spec's Non-Goals "we accept any modest CPU cost in exchange for the parity guarantee." If the slowdown is >20%, something is allocation-heavy in a phase helper (likely Phase 4 message assembly or Phase 7 aggro).

### Commit

- [ ] **Step 12: Stage and commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add \
  internal/hooks/NewRound_DoCombat.go \
  internal/hooks/NewRound_DoCombat_helpers.go \
  internal/hooks/NewRound_DoCombat_resolution.go \
  internal/hooks/NewRound_DoCombat_routing_test.go
git commit -m "$(cat <<'EOF'
refactor(combat): switch dispatchers to handleCombatRound; remove quadrant handlers

Stage 2b of combat-quadrant unification. The two dispatchers
(handlePlayerCombat at NewRound_DoCombat.go:124-129 and handleMobCombat
at :271-278) now call handleCombatRound(atk, def) unconditionally after
constructing the appropriate Actor wrappers.

Removed:
  - handlePlayerVsPlayer, handlePlayerVsMob, handleMobVsPlayer,
    handleMobVsMob (the four quadrant handlers)
  - applyCombatDamageBonuses_PvM, applyCombatDamageBonuses_MvP
  - handlePvMCritAndMessaging, handleMvPCritAndMessaging
  - handlePvMProgressionAndAggro, handleMvPProgression
  - handlePvMRoundResolution, handleMvPRoundResolution
  - dispatchCritEffectsPvM, dispatchCritEffectsPvP, dispatchCombatMessages
    (verified via grep to have zero remaining callers after the handler
    deletions)

Kept (still called from inside the unified phase helpers):
  - handleCombatWaitRound (already unified pre-Stage-2)
  - applyCritEffects, processDefenderProgression
  - handleOffhandBreakUserDef, handleOffhandBreakMobDef
  - handlePlayerConcentrationBreak
  - handleCompanionOwnerAssist, handleCharmedMobAssist
  - handlePartyAutoAttack

New structural test:
TestHandleCombatRound_AllQuadrantsRouteCorrectly feeds each quadrant
pair (UU, UM, MU, MM) through handleCombatRound with deterministic
dice and asserts correct callbacks, text recipients, and behavior
triggers.

Behavior is unchanged from end of Stage 1. Full test suite passes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
git status
```

Stage ONLY the four enumerated paths. Do NOT stage working-tree noise or memory files.

### Spec-Compliance Checklist (Task 3)

- [ ] Both dispatcher sites in `NewRound_DoCombat.go` call `handleCombatRound` unconditionally (one for player attackers, one for mob attackers).
- [ ] The four quadrant handler functions are fully deleted (not commented out).
- [ ] Per-quadrant helpers (the 11 named in Step 6) are deleted after grep-verifying zero remaining callers.
- [ ] Shared non-quadrant helpers (the 9 named in "Kept" above) are unchanged and still called from phase helpers.
- [ ] `TestHandleCombatRound_AllQuadrantsRouteCorrectly` exercises all four quadrant pairs.
- [ ] `go test ./...` matches end-of-Task-1 baseline.

### Code-Quality Checklist (Task 3)

- [ ] Net line delta is NEGATIVE (more deleted than added).
- [ ] No commented-out code.
- [ ] No unused imports (vet catches these).
- [ ] The new routing test file matches the style of existing `internal/hooks/*_test.go` (testify, `seedAllRegistries`).
- [ ] Dispatcher flip preserves existing guards: `user.Character.Aggro != nil`, `defUser != nil`, `defMob != nil`, etc.
- [ ] `moonMod` is correctly threaded into both dispatch sites (player side uses the local `moonMod`; mob side too).

### Rollback Plan (Task 3)

If end-of-Task-3 tests regress:

1. `git revert HEAD` — Task 2's ADD-ONLY handleCombatRound stays but is unused; old handlers come back.
2. The revert restores the pre-Task-3 state exactly.
3. Diagnose the regression against the working Task-2 code, then re-approach Task 3.

If the regression is subtle (one quadrant's text is off-by-one character, or a callback fires at a different count), the failing test name usually points to the phase:
- Text routing fail → Phase 4.
- Callback count fail → Phase 5 or Gap 1/2/3 (but those were locked by Task 1's tests).
- Aggro/assist fail → Phase 7.
- Behavior-tree miss → Phase 6.

---

## Task 4: Memory updates

Memory-file-only task. **NO git operations.** All three files live outside the repo.

**Files:**
- Update directly (NOT via git): `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_pvm_mvp_parity_gaps.md`
- Create directly (NOT via git): `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\feedback_combat_logic_goes_in_handleCombatRound.md`
- Update directly (NOT via git): `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`

**Complexity:** Trivial.

### Steps

- [ ] **Step 1: Append four Stage 1 fixes to `project_pvm_mvp_parity_gaps.md` Resolved section**

Append entries like:

```markdown
- Stage 1 of combat-quadrant-unification / `<COMMIT_HASH_TASK1>`:
  MvM defender `OnCritReceived("physical", 0)` now fires on crit hits
  (Gap 1). PvM/MvP/PvP already fired this callback.
- Stage 1 / `<COMMIT_HASH_TASK1>`: MvM attacker `OnCriticalSuccess` /
  `OnCriticalFailure` now fire per-weapon-hit on crit/fumble (Gap 2),
  mirroring PvP/MvP/PvM's pattern. Skill name is derived from
  `wh.SkillTag`.
- Stage 1 / `<COMMIT_HASH_TASK1>`: MvM attacker stat-gain room messages
  now emit via `characters.MobStatGainMessages` when `OnStatUse` returns
  true (Gap 3). MvP already emitted; the "skip room messages for MvM"
  comment was the bug.
- Stage 1 / `<COMMIT_HASH_TASK1>`: MvP inline `ConditionShield / 2`
  damage reduction deleted (Gap 4). The condition was already counted
  by the mitigation layer (`characters/combat.go:161, 200`); the inline
  block was a Stage 11.4 leftover that silently double-counted the
  reduction for player defenders only. Locked by
  `TestMvP_ConditionShieldAppliedOnceNotDoubleDipped`.
```

Replace `<COMMIT_HASH_TASK1>` with the actual short SHA from `git rev-parse --short HEAD~2` (Task 1 is three commits back from the end of Task 3).

- [ ] **Step 2: Create `feedback_combat_logic_goes_in_handleCombatRound.md`**

```markdown
---
name: combat logic goes inside handleCombatRound or its phase helpers
description: Design rule established by the 2026-04-18 combat-quadrant unification pass — future combat-round logic lands inside handleCombatRound, and quadrant divergence requires explicit IsPlayer() gating with a reason comment
type: feedback
originSessionId: <FILL_IN_FROM_CURRENT_SESSION>
---
# Feedback: combat logic goes inside handleCombatRound

**Rule (established 2026-04-18):**

All combat-round logic that applies to a round of physical combat between
two Actors lives inside `handleCombatRound` (`internal/hooks/NewRound_DoCombat_unified.go`)
or one of its eight phase helpers (`resolveCombatTarget`, `phase1WaitRound`,
`rollCombatAttack`, `applyCombatDamageBonuses`, `dispatchCritAndMessaging`,
`applyCombatProgression`, `fireDefenderBehaviorTrigger`,
`handleAggroAndAssist`, `resolveCombatRound`).

**Why:**

Before Stage 2 of this pass, combat had four parallel quadrant handlers
(PvP/PvM/MvP/MvM). Every refactor since April surfaced a new parity gap
because new logic always landed in only one or two of the four. The
unified handler makes parity structural: by default, new logic applies
to all four quadrants.

**When quadrant divergence is correct (e.g., text routing, behavior
trees only on mobs, party assist, hostility groups):**

Gate the divergent block on `atk.IsPlayer()` / `def.IsPlayer()` at the
leaf site — NOT at the top of the phase helper — AND add a one-line
comment explaining the reason. Example:

```go
// Divergence: TrackPlayerDamage only fires when both sides are players
// (PvP analytics). Mobs have no per-player damage ledger.
if res.Hit && atk.IsPlayer() && def.IsPlayer() {
    defChar.TrackPlayerDamage(atkUid, res.DamageToTarget)
}
```

**When to use type assertion vs `GetCharacter()`:**

- Prefer `actor.GetCharacter()` when the divergent path only needs
  `*characters.Character` methods.
- Use `actor.(*actions.UserActor)` (unchecked — gated behind IsPlayer())
  only when you need `*users.UserRecord`-specific fields like
  `user.PartyId` or `user.Character.ConnectionId`.

**When to add a new method to the Actor interface:**

Only when at least TWO phase helpers need the same functionality AND
the method can be implemented reasonably for both `UserActor` and
`MobActor`. Otherwise, `GetCharacter()` + helper function is sufficient.
The Actor interface is frozen at 14 methods as of 2026-04-18 — treat
additions as a design decision, not a convenience.

**Red flags that the rule is being violated:**

- A phase helper grows a second four-way switch (`case isPlayerPlayer:`,
  etc.). That's a `Quadrant` enum in disguise and the spec rejected it.
- A divergence comment is missing on an `IsPlayer()` branch.
- New logic is being added to `handlePlayerCombat` or `handleMobCombat`
  (the dispatchers in `NewRound_DoCombat.go`) rather than inside
  `handleCombatRound`. The dispatchers are meant to be shallow.

**See also:**

- `project_pvm_mvp_parity_gaps.md` — historical parity gap log
- Spec: `docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md`
- Plan: `docs/superpowers/plans/2026-04-18-combat-quadrant-unification.md`
```

Fill `<FILL_IN_FROM_CURRENT_SESSION>` with the executor's current session id.

- [ ] **Step 3: Wire the new feedback memory into `MEMORY.md`**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`, find the Feedback section (there is a list of `feedback_*.md` files). Add a line entry pointing to the new file. Check existing entries for the exact bullet format — they typically read like:

```markdown
- feedback_combat_logic_goes_in_handleCombatRound — combat logic lives in handleCombatRound; quadrant divergence requires IsPlayer() gating + reason comment (2026-04-18)
```

Match the existing style (checkmark / bullet / dash) used by other feedback entries in the file.

- [ ] **Step 4: Add a completion entry under "Completed (2026-04-18)" in MEMORY.md**

MEMORY.md has "Completed (YYYY-MM-DD)" sections (see lines ~27–110 of that file for existing ones). Add (or append to the existing 2026-04-18 section if one already exists from the rooms-pass):

```markdown
## Completed (2026-04-18)
- Combat quadrant unification — collapsed four parallel handlers into
  handleCombatRound(atk, def Actor); pre-fixed 4 parity gaps (3 MvM
  callbacks + 1 MvP ConditionShield double-dip); 8 phase helpers
  route quadrant divergence via IsPlayer() checks; new feedback memory
  captures the "logic goes in handleCombatRound" rule.
```

### Verify

- [ ] **Step 5: Confirm all three memory files are edited (NOT in git)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
```

Expected: the three memory files do NOT appear anywhere in `git status` output. If they do, you've accidentally created in-repo copies — delete them and edit the real files at the external path.

```bash
ls "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/" | grep -E "parity|handleCombatRound|MEMORY"
```

Expected: three files present and recently modified.

### Spec-Compliance Checklist (Task 4)

- [ ] Four Stage 1 fixes appended to `project_pvm_mvp_parity_gaps.md` Resolved section with real commit hashes.
- [ ] `feedback_combat_logic_goes_in_handleCombatRound.md` exists, captures the rule, and is the correct `type: feedback` metadata.
- [ ] MEMORY.md has a Feedback-list entry for the new memory file.
- [ ] MEMORY.md has a "Completed (2026-04-18)" entry summarizing this work.
- [ ] Zero memory files appear in `git status`.

### Rollback Plan (Task 4)

Memory-file edits are independent of the code commits. If an edit is wrong, edit again. No git revert needed.

---

## Task 5: Final verify + merge

Final sanity sweep + smoke test + merge into `development`. Do NOT push.

### Steps

- [ ] **Step 1: Confirm branch shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline feature/combat-quadrant-unification ^development
```

Expected: 3 commits (newest first):
1. `refactor(combat): switch dispatchers to handleCombatRound; remove quadrant handlers`
2. `refactor(combat): introduce handleCombatRound + phase helpers`
3. `fix(combat): close four parity gaps before quadrant unification`

If a scope-creep `fix:` or `chore:` precursor was added during execution, expect more. Document in merge message.

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/combat-quadrant-unification
```

Expected files changed:
- `internal/hooks/NewRound_DoCombat.go` — modified (dispatcher flips in commit 3)
- `internal/hooks/NewRound_DoCombat_helpers.go` — modified (Gaps 1-4 in commit 1; handler deletions in commit 3)
- `internal/hooks/NewRound_DoCombat_resolution.go` — modified (per-quadrant helper deletions in commit 3)
- `internal/hooks/NewRound_DoCombat_unified.go` — NEW (commit 2)
- `internal/hooks/NewRound_DoCombat_parity_test.go` — NEW (commit 1)
- `internal/hooks/NewRound_DoCombat_routing_test.go` — NEW (commit 3)
- `docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md` — NEW (commit 1)
- `docs/superpowers/plans/2026-04-18-combat-quadrant-unification.md` — NEW (commit 1)

**NO other files.** If you see `.claude/settings.local.json`, `feedback/*.txt`, `Screenshot*.png`, or any memory file, STOP — you accidentally staged noise. `git reset HEAD~N` to the offending commit and restage.

Net line count should be: Task 1 modest positive (test file ~250 lines + 4 small Gap fixes - 9 line deletion); Task 2 large positive (~600-900 lines added in new file); Task 3 large negative (~-400 to -600 from handler deletions + ~100 from routing test). Overall probably net positive by ~400-600 lines, dominated by Task 2's new file and the new test files.

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./... -count=1
```

Expected: clean, matching the Task 0 baseline (minus any pre-existing failures that were documented). `-count=1` defeats stale cache.

- [ ] **Step 4: Manual smoke test — one scenario per quadrant**

Boot a local server. Run the four short scenarios below. Each should produce correct text, callbacks, and behavior triggers. These are sanity checks; the unit tests already lock the contract.

1. **PvM (player attacks mob):** `attack goblin` at a training mob. Confirm attacker sees 2nd-person text, room broadcasts exclude the attacker, mob retaliates with aggro, `mob_hurt` behavior fires if the goblin has a `mob_hurt` tree (check logs).
2. **MvP (mob attacks player):** let an aggressive mob initiate combat. Confirm defender player sees 2nd-person text, `ConditionShield` (if on the player) reduces damage ONCE (not twice; this is the Gap 4 fix — if the player takes noticeably more damage than before Stage 1, that's expected), retarget message appears only on the player side when the mob dies.
3. **PvP (player attacks player):** two test accounts in same room, `attack player2`. Confirm mutual 2nd-person text, `TrackPlayerDamage` records, `OnCritReceived` on crit.
4. **MvM (mob attacks mob):** two hostile mobs in same room. Let them auto-aggro (companion + wild mob, or two aggressive mobs forced into conflict). Confirm `OnCritReceived` fires on defender mob on crits (Gap 1), `OnCriticalSuccess` fires on attacker mob on crits (Gap 2), stat-gain room message appears when attacker gains (Gap 3).

If any scenario reads wrong, STOP — the unit tests missed something.

- [ ] **Step 5: Confirm memory files still live outside the repo**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/combat-quadrant-unification | grep -E "\.claude|memory/project|memory/feedback"
```

Expected: zero matches. If anything matches, memory files leaked into the repo — `git reset` and fix.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff feature/combat-quadrant-unification -m "$(cat <<'EOF'
merge: combat quadrant unification

Collapse the four parallel combat-round handlers (PvP/PvM/MvP/MvM) into
a single handleCombatRound(atk, def actions.Actor, ...) driven by the
Actor interface. Makes future parity gaps structurally impossible —
any new combat logic added to the unified handler applies to all four
quadrants by default; quadrant divergence requires explicit IsPlayer()
gating + reason comment.

Stage 1 (commit 1): pre-fix four parity gaps noticed during scoping.
  - Gap 1: MvM defender OnCritReceived now fires on crit hits.
  - Gap 2: MvM attacker OnCriticalSuccess/OnCriticalFailure fire on
    crit rolls.
  - Gap 3: MvM attacker stat-gain room messages emit via
    MobStatGainMessages.
  - Gap 4: MvP inline ConditionShield damage reduction deleted — the
    condition was already counted in the mitigation layer, so the
    inline block was a silent double-dip for player defenders only.
  Four new tests in NewRound_DoCombat_parity_test.go.

Stage 2a (commit 2): introduce handleCombatRound + 8 phase helpers in
a new file (NewRound_DoCombat_unified.go) alongside the existing four
handlers. Pure ADD-ONLY. Behavior unchanged.

Stage 2b (commit 3): flip both dispatchers (handlePlayerCombat,
handleMobCombat) to call handleCombatRound unconditionally; delete
the four handle{P,M}Vs{P,M} functions and their per-quadrant helpers.
One new structural test (TestHandleCombatRound_AllQuadrantsRouteCorrectly)
exercises all four quadrant pairs. Behavior unchanged from end of
Stage 1. Full test suite passes.

Intentional divergences preserved with reason comments: crit message
routing (attacker text player-only, defender text player-only, room
broadcast always), mob_hurt behavior tree (mob defenders only),
hostility groups (player-attacker-on-mob-defender only), mob-aggro
exit-walk (player attackers only), party auto-assist (gated on
PartyId, not IsPlayer), TrackPlayerDamage (PvP only), Aggro.Type ==
Flee checks (player attackers only), stat-gain room message flavor
(MobStatGainMessages for mobs, player-flavor for players), retarget
"You turn your attention to..." (player survivors only).

Actor interface at internal/actions/actor.go is unchanged (14 methods,
no additions).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If a scope-creep `fix:` or `chore:` precursor was added during execution, mention it in the merge body (one bullet per extra commit).

**Do NOT push to origin.** User reviews merge locally and pushes when ready.

---

## Done

After merge, the four combat handlers are gone, replaced by a single unified `handleCombatRound`. Every future combat-round change applies to all four quadrants by default; quadrant divergence is now explicit and commented. The `project_pvm_mvp_parity_gaps.md` Resolved section has four new entries; the Open Gaps section is (and stays) empty until someone finds a new one. A new feedback memory (`feedback_combat_logic_goes_in_handleCombatRound.md`) tells future executors where combat logic belongs.

---

### Critical Files for Implementation

- `C:\Users\Calabe Davis\workspace\DOGMud\internal\hooks\NewRound_DoCombat_helpers.go` (1379 lines — contains all four quadrant handlers and most shared helpers; Task 1 edits four sites here, Task 3 deletes four functions here)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\hooks\NewRound_DoCombat.go` (408 lines — the two dispatchers at lines 124–129 and 271–278 that Task 3 flips)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\hooks\NewRound_DoCombat_resolution.go` (530 lines — the per-quadrant helpers `applyCombatDamageBonuses_{PvM,MvP}`, `handle{P,M}vs{P,M}CritAndMessaging`, `handle{P,M}vs{P,M}Progression*`, `handle{P,M}vs{P,M}RoundResolution` that Task 3 deletes)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\actions\actor.go` (62 lines — FROZEN in this pass; Actor interface reference for Task 2's phase helpers)
- `C:\Users\Calabe Davis\workspace\DOGMud\internal\hooks\NewRound_DoCombat_unified.go` (NEW in Task 2 — contains `handleCombatRound` and the eight phase helpers; single most important file Task 2 creates)