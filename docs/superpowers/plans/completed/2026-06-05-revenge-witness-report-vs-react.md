# Revenge-Witness: Report vs React Split — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop the theft + assault reactive seeders from seeding a personal `revenge-mob` goal into *every* room witness; instead classify each witness — guards report-only (5.1 enforcement handles it), noncombatants react with momentary alarm, combat-capable mobs still take revenge.

**Architecture:** A pure classifier `classifyWitnessResponse(mob) → WitnessResponse` (unit-tested) plus a thin effect dispatcher `seedWitnessResponse(mob, playerId, priority)` that both `OnTheft` and `aggressiveActionToRevenge` call for victim + each witness. Guard detection is lifted from a package-private hooks helper to an exported `mobs.IsGuardMob`. The "report" path already exists (5.1 `crimes.Record` + bounty + `RunGuardEnforcement`); we only gate the revenge seed and add a light alarm reaction.

**Tech Stack:** Go; `internal/seeders`, `internal/mobs`, `internal/goals`, `internal/rooms`.

**Spec:** `docs/superpowers/specs/2026-06-05-revenge-witness-report-vs-react-design.md`

---

## Reference: verified current code

- `seeders.OnTheft(thiefUserId int, victimMob *mobs.Mob, item items.Item)` —
  seeds revenge into victim (pri `theftVictimRevengePriority`=90) + each
  `room.GetMobs()` witness (pri `theftWitnessRevengePriority`=60).
- `seeders.aggressiveActionToRevenge(event)` — on `PlayerAttackedMob`, returns
  early if `attackedMob.AutoAggro`; seeds revenge into attacked mob (pri
  `aggressiveVictimRevengePriority`=75) + each non-`AutoAggro` witness (pri
  `aggressiveWitnessRevengePriority`=50).
- `seedRevengeGoalIfAbsent(mob *mobs.Mob, targetKind string, targetId, priority int) *goals.Goal` — dedup-and-add (in `seeders/state.go`).
- `mob.Groups []string`; `mob.IsNonCombatant() bool`; `mob.Command(inputTxt string, waitSeconds ...float64)`; `rooms.LoadRoom(id).Exits` is a map keyed by exit name.
- `isGuardMob(groups []string) bool` in `internal/hooks/NewRound_MobRoundTick.go` (checks for the `"guard"` group marker) — to be lifted.
- **Do NOT use `Character.IsGuard()`** — that's the combat-stance predicate (false friend).

---

## Task 1: Lift `isGuardMob` → `mobs.IsGuardMob`

**Files:**
- Modify: `internal/mobs/mobs.go` (add exported func)
- Test: `internal/mobs/mobs_test.go` (or a new `internal/mobs/guard_test.go`)
- Modify: `internal/hooks/NewRound_MobRoundTick.go` (use the lifted func)

- [ ] **Step 1: Write the failing test** — `internal/mobs/guard_test.go`:
```go
package mobs

import "testing"

func TestIsGuardMob(t *testing.T) {
	if !IsGuardMob([]string{"humanoid", "guard"}) {
		t.Fatal("expected true when groups contain 'guard'")
	}
	if IsGuardMob([]string{"humanoid", "thornwall_guards"}) {
		t.Fatal("faction id 'thornwall_guards' is NOT the 'guard' marker")
	}
	if IsGuardMob([]string{"humanoid"}) {
		t.Fatal("expected false without 'guard' marker")
	}
	if IsGuardMob(nil) {
		t.Fatal("expected false for nil groups")
	}
}
```

- [ ] **Step 2: Run it — fails (undefined: IsGuardMob)**
Run: `go test ./internal/mobs/ -run TestIsGuardMob -v` → FAIL (undefined).

- [ ] **Step 3: Add `IsGuardMob` to `internal/mobs/mobs.go`** (near `IsNonCombatant`):
```go
// IsGuardMob reports whether a mob's groups include the law-enforcement
// "guard" marker (5.1 town justice). Note: this is the literal "guard"
// group tag, NOT a guard faction id and NOT the combat-stance
// Character.IsGuard() predicate.
func IsGuardMob(groups []string) bool {
	for _, g := range groups {
		if g == "guard" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run it — passes**
Run: `go test ./internal/mobs/ -run TestIsGuardMob -v` → PASS.

- [ ] **Step 5: Update the hooks caller** — in `internal/hooks/NewRound_MobRoundTick.go`, replace the call `isGuardMob(mob.Groups)` (around line 116) with `mobs.IsGuardMob(mob.Groups)`, and DELETE the now-unused package-private `isGuardMob` func (around lines 473-482). Confirm `mobs` is already imported in that file (it is).
Run: `go build ./internal/hooks/` → clean (if "isGuardMob declared and not used" or "undefined", you missed the delete/replace).

- [ ] **Step 6: Commit**
```bash
go build ./... && go test ./internal/mobs/ ./internal/hooks/
git add internal/mobs/ internal/hooks/NewRound_MobRoundTick.go
git commit -m "refactor(mobs): lift isGuardMob to exported mobs.IsGuardMob"
```

---

## Task 2: Pure witness classifier

**Files:**
- Create: `internal/seeders/witness_response.go`
- Test: `internal/seeders/witness_response_test.go`

- [ ] **Step 1: Write the failing test** — `internal/seeders/witness_response_test.go`:
```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestClassifyWitnessResponse(t *testing.T) {
	guard := &mobs.Mob{Groups: []string{"humanoid", "guard"}}
	if got := classifyWitnessResponse(guard); got != ResponseReportOnly {
		t.Fatalf("guard: want ResponseReportOnly, got %v", got)
	}

	civilian := &mobs.Mob{Groups: []string{"humanoid"}}
	civilian.Character.NonCombatant = true
	if got := classifyWitnessResponse(civilian); got != ResponseAlarm {
		t.Fatalf("noncombatant: want ResponseAlarm, got %v", got)
	}

	fighter := &mobs.Mob{Groups: []string{"humanoid"}}
	if got := classifyWitnessResponse(fighter); got != ResponseRevenge {
		t.Fatalf("combat mob: want ResponseRevenge, got %v", got)
	}

	if got := classifyWitnessResponse(nil); got != ResponseReportOnly {
		t.Fatalf("nil: want ResponseReportOnly (safe), got %v", got)
	}

	// Guard takes precedence over noncombatant (a non-combatant guard still
	// reports rather than alarms).
	ncGuard := &mobs.Mob{Groups: []string{"guard"}}
	ncGuard.Character.NonCombatant = true
	if got := classifyWitnessResponse(ncGuard); got != ResponseReportOnly {
		t.Fatalf("noncombat guard: want ResponseReportOnly, got %v", got)
	}
}
```

- [ ] **Step 2: Run it — fails (undefined: classifyWitnessResponse / Response*)**
Run: `go test ./internal/seeders/ -run TestClassifyWitnessResponse -v` → FAIL.

- [ ] **Step 3: Write `internal/seeders/witness_response.go`** (classifier + enum only for now):
```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// WitnessResponse is how a mob reacts to witnessing (or being the victim of)
// a crime against another mob committed by a player.
type WitnessResponse int

const (
	// ResponseRevenge: combat-capable non-guard — seed a personal revenge goal.
	ResponseRevenge WitnessResponse = iota
	// ResponseAlarm: noncombatant civilian — momentary fright reaction; the
	// law (5.1 crime record + guard enforcement) handles the actual response.
	ResponseAlarm
	// ResponseReportOnly: guard (or nil) — seed nothing; 5.1 crime record +
	// RunGuardEnforcement enforce. A personal revenge goal would derail proper
	// enforcement.
	ResponseReportOnly
)

// classifyWitnessResponse decides how a mob should respond. Pure (no side
// effects). Guard takes precedence over the noncombatant check.
func classifyWitnessResponse(m *mobs.Mob) WitnessResponse {
	if m == nil {
		return ResponseReportOnly
	}
	if mobs.IsGuardMob(m.Groups) {
		return ResponseReportOnly
	}
	if m.IsNonCombatant() {
		return ResponseAlarm
	}
	return ResponseRevenge
}
```
(The `rooms` import is used by `alarmReaction` added in Task 3; if Go complains about an unused import in this step, omit the `rooms` import here and add it in Task 3.)

- [ ] **Step 4: Run it — passes**
Run: `go test ./internal/seeders/ -run TestClassifyWitnessResponse -v` → PASS.

- [ ] **Step 5: Commit**
```bash
go build ./... && go test ./internal/seeders/
git add internal/seeders/witness_response.go internal/seeders/witness_response_test.go
git commit -m "feat(seeders): pure witness-response classifier (guard/noncombat/combat)"
```

---

## Task 3: Effect dispatcher + alarm reaction + wire both seeders

**Files:**
- Modify: `internal/seeders/witness_response.go` (add `seedWitnessResponse` + `alarmReaction`)
- Modify: `internal/seeders/witness_of_theft_to_revenge.go`
- Modify: `internal/seeders/aggressive_action_to_revenge.go`

- [ ] **Step 1: Add the dispatcher + alarm to `witness_response.go`** (ensure the `rooms` import is present now):
```go
// seedWitnessResponse classifies the mob and performs the matching effect.
// Used for both the direct victim and each room witness (victim at the higher
// victim priority). The victim is never a noncombatant (you cannot steal from
// or attack a non_combatant mob), so the victim only ever hits the guard or
// revenge branch.
func seedWitnessResponse(m *mobs.Mob, playerId, priority int) {
	switch classifyWitnessResponse(m) {
	case ResponseRevenge:
		seedRevengeGoalIfAbsent(m, "player", playerId, priority)
	case ResponseAlarm:
		alarmReaction(m)
	case ResponseReportOnly:
		// no-op: the 5.1 crime record + RunGuardEnforcement handle it.
	}
}

// alarmReaction is a momentary fright reaction for a noncombatant witness — a
// room-visible emote plus a single step toward an exit. No persistent goal
// (deliberately avoids the survival-goal-pruned-at-full-HP behavior). The
// actual "report" is the 5.1 crime record fired by the steal/attack action.
func alarmReaction(m *mobs.Mob) {
	if m == nil {
		return
	}
	m.Command("emote recoils and cries out, then hurries for the nearest way out.")
	room := rooms.LoadRoom(m.Character.RoomId)
	if room == nil {
		return
	}
	for exitName := range room.Exits { // map order is randomized → a random exit
		m.Command(exitName)
		break
	}
}
```

- [ ] **Step 2: Wire `OnTheft`** — in `internal/seeders/witness_of_theft_to_revenge.go`, replace the victim seed and the witness-loop seed:
```go
	// Victim + witnesses route through the classifier (guard -> report-only,
	// noncombatant -> alarm, combat-capable -> revenge).
	seedWitnessResponse(victimMob, thiefUserId, theftVictimRevengePriority)

	room := rooms.LoadRoom(victimMob.Character.RoomId)
	if room == nil {
		return
	}
	for _, witnessInstId := range room.GetMobs() {
		if witnessInstId == victimMob.InstanceId {
			continue
		}
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil {
			continue
		}
		seedWitnessResponse(witness, thiefUserId, theftWitnessRevengePriority)
	}
```
(Replaces the two former `seedRevengeGoalIfAbsent(...)` calls. The `items` import may become unused — if so, drop it; `OnTheft`'s signature keeps the `item items.Item` param, so the import likely stays. Build will tell you.)

- [ ] **Step 3: Wire `aggressiveActionToRevenge`** — in `internal/seeders/aggressive_action_to_revenge.go`, replace the two `seedRevengeGoalIfAbsent(...)` calls (keep the `AutoAggro` early-return for the attacked mob and the `witness.AutoAggro` skip):
```go
	seedWitnessResponse(attackedMob, pa.UserId, aggressiveVictimRevengePriority)

	room := rooms.LoadRoom(attackedMob.Character.RoomId)
	if room == nil {
		return
	}
	for _, witnessInstId := range room.GetMobs() {
		if witnessInstId == attackedMob.InstanceId {
			continue
		}
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil || witness.AutoAggro {
			continue
		}
		seedWitnessResponse(witness, pa.UserId, aggressiveWitnessRevengePriority)
	}
```

- [ ] **Step 4: Build + test**
Run: `go build ./... && go test ./internal/seeders/`
Expected: clean; existing seeder tests still pass (the nil/zero-guard tests are unaffected; the classifier test passes). If an existing test asserted the old blanket-seed behavior, update it to the new classified behavior.

- [ ] **Step 5: Commit**
```bash
git add internal/seeders/
git commit -m "feat(seeders): route theft + assault witnesses through report/alarm/revenge classifier"
```

---

## Task 4: Build, test, boot smoke

- [ ] **Step 1: Full build + test**
Run: `go build ./... && go test ./...`
Expected: build clean; all packages pass.

- [ ] **Step 2: Boot smoke** (wipe instances first):
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Boot; confirm clean load past data files (no panic). The change is logic-only (no new data), so boot is just a sanity check.

- [ ] **Step 3: In-game behavior smoke** (admin; MAY be deferred to user)
- Steal from / attack a combat mob in a room that also holds: a guard (groups include `guard`), a noncombatant shopkeeper, and a plain combat thug.
- Expect: the guard does NOT acquire a personal revenge chase (it enforces via 5.1 — warn/arrest); the shopkeeper recoils and steps toward an exit (no pursuit); the thug acquires the revenge goal and pursues. The 5.1 crime + bounty still record as before.

No commit (verification only).

---

## Task 5: Docs + memory

**Files:**
- Modify: `internal/seeders/context.md`, `internal/mobs/context.md`
- Memory: `project_revenge_witness_report_vs_react_followup` (mark resolved)

- [ ] **Step 1: context.md notes**
- `internal/seeders/context.md`: note the witness-response classifier — theft + assault witnesses route through `classifyWitnessResponse`/`seedWitnessResponse` (guard→report-only, noncombat→alarm, combat→revenge); `friend_killed_to_revenge` (kin) is unchanged. Note the slightly-stale filename `witness_of_theft_to_revenge.go` (now also report/alarm).
- `internal/mobs/context.md`: note the new exported `IsGuardMob(groups)`.

- [ ] **Step 2: Commit docs**
```bash
git add internal/seeders/context.md internal/mobs/context.md
git commit -m "docs(context): witness-response classifier + mobs.IsGuardMob"
```

- [ ] **Step 3: Mark the followup memory resolved**
Update `project_revenge_witness_report_vs_react_followup` (topic file + the MEMORY.md index line) to RESOLVED 2026-06-05, noting: theft + assault witness seeders now classify by type; guards report-only (5.1 enforcement), noncombatants alarm, combat revenge; murder/kin left as-is. (This is a memory edit, not a repo commit.)

---

## Self-review notes

**Spec coverage:**
- Spec "gate revenge seed by witness type" → Tasks 2 (classifier) + 3 (dispatch + wiring of both theft & assault).
- Spec "lift isGuardMob to mobs.IsGuardMob (not Character.IsGuard)" → Task 1.
- Spec "noncombatant alarm reaction, no persistent goal" → Task 3 `alarmReaction`.
- Spec "victim never noncombatant; guard victim → report-only" → covered (victim routes through the same `seedWitnessResponse`; classifier handles it).
- Spec "murder/kin left as-is" → no task touches `friend_killed_to_revenge`.
- Spec testing (unit tests on the pure classifier + IsGuardMob; smoke for effects) → Tasks 1, 2, 4.

**Placeholder check:** all code given in full; the only judgment notes are the conditional import drops (build-verified). No TBDs.

**Type consistency:** `classifyWitnessResponse` / `seedWitnessResponse` / `WitnessResponse` / `Response{Revenge,Alarm,ReportOnly}` / `mobs.IsGuardMob` used consistently across Tasks 1-3. Priorities reuse the existing consts (90/60/75/50).

**TDD:** the pure classifier + `IsGuardMob` get real unit tests (Tasks 1-2). The effect dispatch + alarm (which need live mob/room/goal state) are verified by build + the boot/behavior smoke — consistent with the existing seeder tests, which explicitly defer full-seed integration to smoke.
