# Mob Aliveness 3.3 — Sleeping & Wake States Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sleeping becomes a bidirectional mechanic. NPCs visibly asleep during authored sleep segments. Players can sleep anywhere. 5× regen while sleeping. Entire-first-round auto-crits for attackers. Wake on damage, failed steal, shout, light entering, or `stand`. Schedule executor recognizes `activity: sleeping` with a grace cooldown.

**Architecture:** Hangs off the existing buff system — `Sleeping` becomes a state-query `buffs.Flag` so any code path can check `HasBuffFlag(buffs.Sleeping)`. New `actions.Sleep(actor)` actor function powers both player and mob `sleep` commands plus the schedule executor's segment-entry hook. Wake triggers (4) all converge on a single `mobs.OnSleeperWoken(actor)` helper that stamps `schedule_wake_round` MiscData so the schedule executor's grace cooldown can suppress re-sleep. First-hit-crit uses a `forceCrit bool` parameter on the damage pipeline; the round dispatcher snapshots sleeping victims at start-of-round so all attackers in the round share the payoff before `cancel-on-damage` cancels the buff.

**Tech Stack:** Go 1.24. Files touched in `internal/buffs/`, `internal/actions/`, `internal/usercommands/`, `internal/mobcommands/`, `internal/hooks/`, `internal/characters/`, `internal/combat/`, `internal/rooms/`, `internal/mobs/`, `internal/configs/`. Content YAMLs in `_datafiles/world/default/buffs/`, `_datafiles/world/dogmud/schedules/thornwall_city/`. Helpfile templates in `_datafiles/world/dogmud/templates/help/`.

**Spec:** `docs/superpowers/specs/2026-05-25-mob-aliveness-3.3-sleeping-wake-states-design.md`

**Branch:** `feature/mob-aliveness-3.3-sleeping-wake-states` (already created; spec committed as `2a69d6de`).

---

## Stage map

| Stage | Task | Description |
|---|---|---|
| 1 | T1 | `buffs.Sleeping` flag + buff YAML update + `cancel-on-damage` flag constant |
| 2 | T2 | `OnSleeperWoken` helper + `cancel-on-damage` damage-pipeline wiring |
| 3 | T3 | `actions.Sleep` actor function (TDD) |
| 4 | T4 | `sleep` user command + `sleep` mob command |
| 5 | T5 | `stand` extension (player + mob) + tests |
| 6 | T6 | Config knobs: `SleepRegenMultiplier` + `ScheduleWakeGraceRounds` |
| 7 | T7 | Schedule loader: recognize `activity: sleeping` |
| 8 | T8 | Schedule executor: `WantsSleep` / `WantsWake` with grace cooldown |
| 9 | T9 | Room rendering: `(asleep)` suffix on `VisibleMobs` / `VisiblePlayers` |
| 10 | T10 | Wake trigger: failed steal |
| 11 | T11 | Wake trigger: shout in room |
| 12 | T12 | Wake trigger: light source on room entry |
| 13 | T13 | Damage pipeline: `forceCrit` parameter |
| 14 | T14 | Round dispatcher (`handleCombatRound`): snapshot sleeping victims + use `forceCrit` |
| 15 | T15 | Regen boost: `SleepRegenMultiplier` applied to HP/SP/CP |
| 16 | T16 | Pilot retrofit: 3 schedule YAMLs |
| 17 | T17 | Documentation pass |
| 18 | T18 | Smoketester goal file + manual smoke + roadmap closeout |

18 tasks. Subagent dispatch order matters — T1 first (foundation), then T2-T7 in roughly parallel-safe groups, T8-T16 in dependency order, T17-T18 closeout. See Self-Review for dependency check.

---

## Task 1: `buffs.Sleeping` flag + buff YAML + `cancel-on-damage` constant

**Files:**
- Modify: `internal/buffs/buffspec.go` (add two `Flag` constants)
- Modify: `_datafiles/world/default/buffs/15-sleeping.yaml` (add `sleeping` flag + `cancel-on-damage` flag, bump `triggercount`)

- [ ] **Step 1: Read the existing flag declarations**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '30,80p' internal/buffs/buffspec.go
```

Confirm the existing pattern for `Flag` constants (`Hidden`, `NightVision`, `CancelIfCombat`, `CancelOnAction`). The new constants `Sleeping` and `CancelOnDamage` follow the same shape.

- [ ] **Step 2: Add the two new flag constants**

In `internal/buffs/buffspec.go`, find the block of `Flag` constants. Add `CancelOnDamage` adjacent to the existing `CancelIfCombat` / `CancelOnAction`:

```go
CancelIfCombat Flag = `cancel-on-combat`
CancelOnAction Flag = `cancel-on-action`
CancelOnDamage Flag = `cancel-on-damage` // chunk 3.3: cancels on any damage event
```

Add `Sleeping` in the state-query group near `Hidden`, `NightVision`:

```go
Hidden         Flag = `hidden`
// ... other state flags ...
NightVision    Flag = `nightvision`
Sleeping       Flag = `sleeping` // chunk 3.3: bearer is asleep
```

Match the local alignment style of the existing constants (the file uses gofmt-aligned ` Flag = ` columns).

- [ ] **Step 3: Update the buff YAML**

Replace the contents of `_datafiles/world/default/buffs/15-sleeping.yaml` with:

```yaml
buffid: 15
name: Sleeping
description: You are getting much needed rest.
triggerrate: 1 round
triggercount: 100000
flags:
- sleeping
- cancel-on-action
- cancel-on-combat
- cancel-on-damage
```

`triggercount: 100000` is the effectively-infinite duration sentinel — duration is driven by cancel-on-* flags and explicit removal, not buff tick expiration.

- [ ] **Step 4: Build to confirm the new constants are valid Go**

```bash
go build ./...
```

Expected: clean. No code consumes the new constants yet — this task just declares them.

- [ ] **Step 5: Commit**

```bash
git add internal/buffs/buffspec.go _datafiles/world/default/buffs/15-sleeping.yaml
git commit -m "$(cat <<'EOF'
feat(buffs): Sleeping state-query flag + cancel-on-damage flag

Adds the buffs.Sleeping Flag constant so other packages can
HasBuffFlag(buffs.Sleeping). Adds cancel-on-damage flag for the
damage-pipeline wake trigger (T2 wires it). Buff YAML updated to
include both flags and bump triggercount to effectively-infinite
(cancel-on-* drives duration now).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Use `git add -f` if the buffs YAML gets caught by the gitignore quirk (the `dogmud` pattern). Default buffs are under `_datafiles/world/default/`, not `_datafiles/world/dogmud/`, so they should not need the force flag — but check.

---

## Task 2: `OnSleeperWoken` helper + `cancel-on-damage` damage-pipeline wiring

**Files:**
- Create: `internal/mobs/sleeper.go` (`OnSleeperWoken` helper)
- Create: `internal/mobs/sleeper_test.go` (helper tests)
- Modify: `internal/combat/damage_pipeline.go` OR wherever damage is applied to a character (investigate; the existing `cancel-on-combat` wiring is in `internal/characters/buffs.go:88` and is triggered by combat-entry, not damage-event — see Step 1)

- [ ] **Step 1: Investigate where damage is APPLIED (not just calculated)**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -rn "Health -= \|Health = " internal/combat internal/hooks --include="*.go" | grep -v "_test.go" | head -10
grep -rn "Character.Damage\|ApplyDamage\|applyDamage" internal --include="*.go" | head -10
```

You're looking for the central site where damage subtracts from `character.Health`. There may be several call sites; the goal is to find the chokepoint (or wire each one). Likely candidates: `internal/combat/damage_pipeline.go` (the `ApplyMitigation` function returns a value — the caller subtracts), or a higher-level applier.

If there's no single chokepoint, the cleanest fix is to add a helper `combat.ApplyDamageToCharacter(target *characters.Character, dmg int)` that handles the subtraction AND fires the cancel-on-damage flag, then migrate call sites. But that's a larger task. For T2 scope, identify the 2-3 main damage application sites (combat round, spell damage, DoT tick if exists) and wire each.

If the investigation surfaces a clear chokepoint, use it. Otherwise document the sites you found and wire them individually.

- [ ] **Step 2: Write the failing test for `OnSleeperWoken`**

Create `internal/mobs/sleeper_test.go`:

```go
package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestOnSleeperWoken_StampsWakeRoundOnMob(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	mob := &Mob{}
	mob.Character.IsMob = true

	OnSleeperWoken(&mob.Character)

	got := mob.Character.GetMiscData("schedule_wake_round")
	if got == nil {
		t.Fatalf("expected schedule_wake_round to be stamped, got nil")
	}
	if got.(int) != 1000 {
		t.Errorf("expected stamp value 1000, got %v", got)
	}
}

func TestOnSleeperWoken_NoOpForPlayers(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	// Players don't have schedule grace; OnSleeperWoken should not stamp.
	var c struct{ MiscData map[string]any } // placeholder — use the real characters.Character
	_ = c

	// Build a real player character via the standard helper.
	// If no helper exists in package mobs, inline-build:
	playerChar := newPlayerCharForTest()

	OnSleeperWoken(playerChar)

	if playerChar.GetMiscData("schedule_wake_round") != nil {
		t.Errorf("expected no stamp for player, got value")
	}
}

// newPlayerCharForTest is a tiny helper. If a richer helper already exists
// in the test files, use that instead.
func newPlayerCharForTest() *characters.Character {
	c := &characters.Character{}
	c.IsMob = false
	return c
}
```

Note: the test for players assumes `OnSleeperWoken` is a no-op for non-mob characters (per spec: "For players, it's a no-op (no schedule grace concept)"). If you want a stronger test, also verify the mob path doesn't fire when `mob.ScheduleId == ""` — the stamp is harmless even then, but cleaner to skip the MiscData write when no schedule is in play.

For the spec semantic ("Stamp the wake round only on scheduled mobs") add the guard explicitly in the implementation (Step 4) and update the test.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/mobs/ -run TestOnSleeperWoken -v
```

Expected: compilation error — `OnSleeperWoken` not defined.

- [ ] **Step 4: Implement `OnSleeperWoken`**

Create `internal/mobs/sleeper.go`:

```go
package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// OnSleeperWoken is the central wake-event hook. Called by every code
// path that cancels the Sleeping buff on a character outside of natural
// segment-end (damage, failed steal, shout, light-on-entry, stand).
//
// For scheduled mobs, it stamps `schedule_wake_round` in MiscData so
// the schedule executor's grace cooldown (ScheduleWakeGraceRounds)
// can suppress re-sleep for N rounds after the wake event.
//
// For players and non-scheduled mobs, it is a no-op (no schedule
// grace concept to enforce).
func OnSleeperWoken(c *characters.Character) {
	if c == nil || !c.IsMob {
		return
	}
	// Look up the mob instance to check if it has a schedule.
	mob := GetInstance(c.MobInstanceId)
	if mob == nil || mob.ScheduleId == "" {
		return
	}
	c.SetMiscData("schedule_wake_round", int(util.GetRoundCount()))
}
```

- [ ] **Step 5: Adjust test to match the schedule-gated implementation**

Update `TestOnSleeperWoken_StampsWakeRoundOnMob` to construct a real registered mob with a `ScheduleId`. The simplest pattern is to inline a schedule registration via the test helpers from T1 (`RegisterScheduleForTest`):

```go
func TestOnSleeperWoken_StampsWakeRoundOnMob(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	// Register a fixture schedule so the cross-check passes.
	RegisterScheduleForTest(&Schedule{
		Id: "test_sched",
		Segments: []ScheduleSegment{{Start: 0, End: 24, TargetRoom: 1, IdleCommands: []string{"x"}}},
	})
	defer UnregisterScheduleForTest("test_sched")

	// Build and register a mob with the schedule id.
	mob := &Mob{InstanceId: 9999, ScheduleId: "test_sched"}
	mob.Character.IsMob = true
	mob.Character.MobInstanceId = 9999
	registerMobInstanceForTest(mob) // see Step 5a below — use any existing equivalent
	defer unregisterMobInstanceForTest(9999)

	OnSleeperWoken(&mob.Character)

	got := mob.Character.GetMiscData("schedule_wake_round")
	if got == nil || got.(int) != 1000 {
		t.Errorf("expected schedule_wake_round=1000, got %v", got)
	}
}

func TestOnSleeperWoken_NoOpForUnscheduledMob(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	mob := &Mob{InstanceId: 9998, ScheduleId: ""}
	mob.Character.IsMob = true
	mob.Character.MobInstanceId = 9998
	registerMobInstanceForTest(mob)
	defer unregisterMobInstanceForTest(9998)

	OnSleeperWoken(&mob.Character)

	if mob.Character.GetMiscData("schedule_wake_round") != nil {
		t.Errorf("expected no stamp for unscheduled mob")
	}
}

func TestOnSleeperWoken_NoOpForPlayer(t *testing.T) {
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	var playerChar characters.Character
	playerChar.IsMob = false

	OnSleeperWoken(&playerChar)
	// No assertion beyond not panicking — players have no MiscData expectation.
}
```

- [ ] **Step 5a: Identify or add a mob-instance test helper**

```bash
grep -n "mobInstances\[" internal/mobs/mobs.go | head -5
grep -n "func.*ForTest\|func.*Test" internal/mobs/*.go | head -10
```

If `registerMobInstanceForTest` doesn't already exist, add minimal helpers in `internal/mobs/test_helpers_test.go` (or create that file). The simplest version directly pokes `mobInstances` under the mutex:

```go
func registerMobInstanceForTest(m *Mob) {
	mobInstancesMu.Lock()
	defer mobInstancesMu.Unlock()
	mobInstances[m.InstanceId] = m
}

func unregisterMobInstanceForTest(id int) {
	mobInstancesMu.Lock()
	defer mobInstancesMu.Unlock()
	delete(mobInstances, id)
}
```

These are test-only helpers — keep them in a `_test.go` file so they don't compile into the production binary.

- [ ] **Step 6: Wire `cancel-on-damage` flag into damage application**

Per the investigation in Step 1, modify the damage application chokepoint(s). The pattern is:

```go
// After damage is applied to target.Character.Health:
if target.HasBuffFlag(buffs.Sleeping) {
    // The Sleeping flag is also a cancel-on-damage flag carrier,
    // so the CancelBuffsWithFlag(CancelOnDamage) below will fire
    // the wake-event hook before clearing the buff.
    mobs.OnSleeperWoken(target)
}
target.CancelBuffsWithFlag(buffs.CancelOnDamage)
```

The order matters: read `HasBuffFlag(buffs.Sleeping)` BEFORE calling `CancelBuffsWithFlag`, because the cancel removes the buff and the subsequent flag query would return false.

If the damage path applies through multiple call sites (e.g., melee, spell, DoT), wire each. Document any sites you deliberately skip with a comment.

If `internal/combat/damage_pipeline.go` has no `ApplyDamageToCharacter`-style chokepoint and damage application is scattered, file this as DONE_WITH_CONCERNS noting which sites you wired and which you skipped. The smoke test in T18 will catch missed sites.

- [ ] **Step 7: Run tests and build**

```bash
go test ./internal/mobs/ -v -run TestOnSleeperWoken
go build ./...
```

Expected: green.

- [ ] **Step 8: Commit**

```bash
git add internal/mobs/sleeper.go internal/mobs/sleeper_test.go internal/mobs/test_helpers_test.go internal/combat/damage_pipeline.go
git commit -m "$(cat <<'EOF'
feat(mobs): OnSleeperWoken helper + cancel-on-damage wiring

OnSleeperWoken stamps schedule_wake_round MiscData for scheduled
mobs so the schedule executor's grace cooldown suppresses
re-sleep after a wake event. No-op for players and unscheduled
mobs.

Damage application path now reads HasBuffFlag(Sleeping) BEFORE
cancelling cancel-on-damage buffs, then fires OnSleeperWoken
on the victim. Order matters: cancel removes the buff so the
post-cancel flag query would return false.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Adjust the file list to include exactly which files the investigation in Step 1 led you to modify.

---

## Task 3: `actions.Sleep` actor function

**Files:**
- Create: `internal/actions/sleep.go`
- Create: `internal/actions/sleep_test.go`

- [ ] **Step 1: Read existing actor functions for the pattern**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
ls internal/actions/ | head -20
sed -n '1,60p' internal/actions/forage.go
sed -n '1,40p' internal/actions/actor.go
```

Confirm the `Actor` interface shape and how existing actor functions handle the player-vs-mob branch (the `Actor` abstraction means most code doesn't need to branch; only user-visible messaging does).

- [ ] **Step 2: Write the failing test**

Create `internal/actions/sleep_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
)

func TestSleep_BlocksInCombat(t *testing.T) {
	actor := newTestUserActor(t)
	actor.Character().Aggro = &combat.Aggro{} // any non-nil Aggro

	res := Sleep(actor, SleepOptions{})

	if res != Failure {
		t.Errorf("expected Failure when in combat, got %v", res)
	}
	if actor.Character().HasBuffFlag(buffs.Sleeping) {
		t.Errorf("expected Sleeping not applied during combat")
	}
}

func TestSleep_AppliesBuffOnSuccess(t *testing.T) {
	actor := newTestUserActor(t)

	res := Sleep(actor, SleepOptions{})

	if res != Success {
		t.Fatalf("expected Success, got %v", res)
	}
	if !actor.Character().HasBuffFlag(buffs.Sleeping) {
		t.Errorf("expected Sleeping flag applied")
	}
}

func TestSleep_IdempotentWhenAlreadySleeping(t *testing.T) {
	actor := newTestUserActor(t)
	Sleep(actor, SleepOptions{}) // first call applies

	// Second call should be a no-op success — no re-apply, no re-emit.
	res := Sleep(actor, SleepOptions{})

	if res != Success {
		t.Errorf("expected idempotent Success on re-sleep, got %v", res)
	}
}
```

`newTestUserActor` is a helper — check if one exists in `internal/actions/actions_test.go`; if not, add a minimal version that returns an actor wrapping a fresh `users.UserRecord` with a basic `Character`.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/actions/ -run TestSleep_ -v
```

Expected: compilation error — `Sleep` and `SleepOptions` not defined.

- [ ] **Step 4: Implement `internal/actions/sleep.go`**

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// SleepOptions is reserved for future authoring knobs (bed-item bonus,
// custom emote prose, etc.). Empty for chunk 3.3.
type SleepOptions struct{}

// Sleep applies the Sleeping buff (id 15) to actor's character. Used by
// the player sleep command, the mob sleep command, and the schedule
// executor (via mob.Command("sleep")).
//
// Fails when the actor is in combat / has Aggro. User actors receive a
// player-visible "You can't sleep right now." message; mob actors get no
// player-visible message (the schedule executor retries silently).
//
// Idempotent: if the actor is already sleeping, returns Success without
// re-applying the buff or re-emitting the room message.
func Sleep(actor Actor, opts SleepOptions) Result {
	c := actor.Character()
	if c == nil {
		return Failure
	}

	// Idempotent re-sleep.
	if c.HasBuffFlag(buffs.Sleeping) {
		return Success
	}

	// Combat gate.
	if c.Aggro != nil || c.IsInCombat() {
		actor.SendText(messaging.CategorySystem, "You can't sleep right now.")
		return Failure
	}

	// Apply buff 15.
	addBuffToCharacter(c, 15)

	// Visual room message.
	actor.SendRoomVisual(messaging.CategoryMobEmote,
		actor.NameForOthers()+" lies down to sleep.")

	return Success
}

// addBuffToCharacter is a small wrapper around the existing buff-add API.
// If a public helper already exists in characters/buffs, use that instead.
func addBuffToCharacter(c *characters.Character, buffId int) {
	c.AddBuff(buffId, 0, "") // adjust args to match the real signature
}
```

The actual `Character.AddBuff` signature lives in `internal/characters/buffs.go` — check it and use the right arg shape. If the existing API requires a source / origin parameter, pass appropriate defaults (e.g., `"self"`).

Similarly, `actor.SendText(...)`, `actor.SendRoomVisual(...)`, and `actor.NameForOthers()` are placeholders for the actual `Actor` interface methods. Match the names used by other actor functions like `actions.Forage` or `actions.Steal`.

- [ ] **Step 5: Run the test, confirm pass**

```bash
go test ./internal/actions/ -run TestSleep_ -v
```

Expected: all three tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/sleep.go internal/actions/sleep_test.go
git commit -m "$(cat <<'EOF'
feat(actions): Sleep actor function

Powers both player sleep command and mob sleep command. Applies
buff 15 (Sleeping) idempotently. Combat-gated: user actors see
"You can't sleep right now."; mob actors fail silently (schedule
executor retries on next idle tick after combat ends).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: `sleep` user command + `sleep` mob command

**Files:**
- Create: `internal/usercommands/sleep.go`
- Create: `internal/mobcommands/sleep.go`
- Modify: `internal/usercommands/usercommands.go` (register the new user command)
- Modify: `internal/mobcommands/mobcommands.go` (register the new mob command — check actual file)

- [ ] **Step 1: Read the existing command-registration patterns**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,50p' internal/usercommands/stand.go
grep -n "Stand\b" internal/usercommands/usercommands.go | head -5
grep -n "Stand\b" internal/mobcommands/*.go | head -10
```

The `stand` command is the closest functional precedent. Use it as the template for `sleep`.

- [ ] **Step 2: Implement the user command**

Create `internal/usercommands/sleep.go`:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Sleep is the player-facing entry point for the sleep verb. Delegates
// to actions.Sleep which applies the Sleeping buff (chunk 3.3).
func Sleep(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	userActor := actions.NewUserActor(user, room) // adjust to the real constructor name
	actions.Sleep(userActor, actions.SleepOptions{})
	return true, nil
}
```

Check the real actor-construction call by looking at how another command (e.g. `Forage` or `Steal`) builds an actor.

- [ ] **Step 3: Register the user command**

Find the user-command registration block (likely in `internal/usercommands/usercommands.go`). Add:

```go
"sleep":  {Sleep, true, false}, // adjust the tuple shape to match the existing registry struct
```

Run `grep -n "stand" internal/usercommands/usercommands.go` to confirm the exact registration form. Match it for `sleep`.

- [ ] **Step 4: Implement the mob command**

Create `internal/mobcommands/sleep.go`:

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Sleep is the mob-side entry point for the sleep verb. Called by the
// schedule executor via mob.Command("sleep"). Delegates to actions.Sleep
// which applies the Sleeping buff (chunk 3.3).
func Sleep(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	mobActor := actions.NewMobActor(mob, room) // adjust to the real constructor name
	actions.Sleep(mobActor, actions.SleepOptions{})
	return true, nil
}
```

- [ ] **Step 5: Register the mob command**

Find the mob-command registration block. Add the entry analogous to step 3:

```go
"sleep":  {Sleep, true}, // adjust to match
```

- [ ] **Step 6: Build and confirm registration**

```bash
go build ./...
```

If any registration syntax doesn't match the existing pattern, fix it. The compiler will reject misregistered commands.

- [ ] **Step 7: Commit**

```bash
git add internal/usercommands/sleep.go internal/usercommands/usercommands.go internal/mobcommands/sleep.go internal/mobcommands/mobcommands.go
git commit -m "$(cat <<'EOF'
feat(commands): sleep user and mob commands

Both delegate to actions.Sleep. The user command is the player
verb; the mob command is used by the chunk 3.3 schedule executor
via mob.Command("sleep") when entering an activity: sleeping
segment.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Adjust the file list to match the actual registration files modified.

---

## Task 5: `stand` extension (player + mob) + tests

**Files:**
- Modify: `internal/usercommands/stand.go` (insert Sleeping-cancel branch at top)
- Modify: `internal/mobcommands/stand.go` (if exists; same shape)
- Create or append: `internal/usercommands/usercommands_test.go` (add Stand+Sleeping test)

- [ ] **Step 1: Re-read the player stand handler**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '16,70p' internal/usercommands/stand.go
```

The existing handler bails early if `!IsProne() && !IsSupine()` ("You're already standing."). The Sleeping cancel needs to fire BEFORE that early-bail check — otherwise a standing-but-sleeping player would hit the bail and never wake.

Check if a mob equivalent exists:

```bash
ls internal/mobcommands/stand.go 2>/dev/null || echo "no mob stand command"
```

If there's no mob stand command, the schedule executor's wake path is the direct `CancelBuffsWithFlag` call (T8 covers it). If there is one, mirror the player change.

- [ ] **Step 2: Write the failing test**

Append to `internal/usercommands/usercommands_test.go`:

```go
func TestStand_CancelsSleeping(t *testing.T) {
	user := newTestUser(t) // use whatever test helper exists in this package
	user.Character.AddBuff(15, 0, "") // apply Sleeping (adjust to real AddBuff sig)
	if !user.Character.HasBuffFlag(buffs.Sleeping) {
		t.Fatalf("test setup failed: Sleeping should be applied")
	}

	room := newTestRoom(t)
	Stand("", user, room, 0)

	if user.Character.HasBuffFlag(buffs.Sleeping) {
		t.Errorf("expected Sleeping to be cancelled by stand")
	}
}
```

If `newTestUser` / `newTestRoom` don't exist in the package, look for similar helpers and adapt. If the test scaffolding would require a big build-out, file the test as `t.Skip("integration covered in T18 smoke")` instead.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/usercommands/ -run TestStand_CancelsSleeping -v
```

Expected: fail (Sleeping is not cancelled by the current stand handler).

- [ ] **Step 4: Implement the extension**

In `internal/usercommands/stand.go`, AT THE VERY TOP of the `Stand` function (before the IsProne/IsSupine bail), add:

```go
func Stand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Chunk 3.3: stand wakes a sleeping player. Cancel BEFORE the
	// "already standing" bail so a standing-but-sleeping player can
	// still wake via stand.
	if user.Character.HasBuffFlag(buffs.Sleeping) {
		user.Character.CancelBuffsWithFlag(buffs.Sleeping)
		mobs.OnSleeperWoken(&user.Character)
		// Fall through — the existing "you stand up" / "you're already
		// standing" message covers wake narration via the user's stand
		// attempt context.
	}

	// Chunk 4b W7: gate on the new Position FSM (Prone or Supine).
	if !user.Character.IsProne() && !user.Character.IsSupine() {
		user.SendText(messaging.CategorySystem, "You're already standing.")
		return true, nil
	}
	// ... rest unchanged ...
```

Add `buffs` and `mobs` imports if not present.

If a mob stand command exists (`internal/mobcommands/stand.go`), apply the same shape — `if mob.Character.HasBuffFlag(buffs.Sleeping) { CancelBuffsWithFlag + OnSleeperWoken }`.

- [ ] **Step 5: Run the test, confirm pass**

```bash
go test ./internal/usercommands/ -run TestStand_CancelsSleeping -v
go build ./...
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/stand.go internal/mobcommands/stand.go internal/usercommands/usercommands_test.go
git commit -m "$(cat <<'EOF'
feat(commands): stand cancels Sleeping (player + mob)

Stand now cancels the Sleeping buff before the position-state
bail, so a sleeping (regardless of position) actor wakes when
they stand. OnSleeperWoken fires so the schedule grace cooldown
applies for scheduled NPCs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Config knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (add two `ConfigFloat`/`ConfigInt` fields)
- Modify: `internal/configs/config.balance.mobs.go` (set defaults)

- [ ] **Step 1: Read the existing knob pattern**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "ScheduleMaxPathRetries" internal/configs/config.balance.go internal/configs/config.balance.mobs.go
```

The chunk 3.2 knob `ScheduleMaxPathRetries` is the perfect template. Look at how it's declared and defaulted.

- [ ] **Step 2: Add the two new fields**

In `internal/configs/config.balance.go`, find the MOB SCHEDULES section (where `ScheduleMaxPathRetries` lives). Add:

```go
// SleepRegenMultiplier multiplies HP/SP/CP per-round percentage regen
// when the bearer has the Sleeping flag. Default 5.0 — sleep is the
// dominant recovery mechanic. Chunk 3.3.
SleepRegenMultiplier ConfigFloat `yaml:"SleepRegenMultiplier"`

// ScheduleWakeGraceRounds is the cooldown (in rounds) during which a
// scheduled mob will not re-sleep after a forced wake. Prevents the
// schedule executor from immediately re-applying Sleeping when the
// player interacts with a sleeping NPC. Default 50 (~200 sec real-time
// at default tick rate). Chunk 3.3.
ScheduleWakeGraceRounds ConfigInt `yaml:"ScheduleWakeGraceRounds"`
```

If `ConfigFloat` isn't the right wrapper type for floats in the local convention, check what other floats use (e.g., `BarterMaxDiscount`, `KickDamagePercent`, etc.) and match.

- [ ] **Step 3: Set the defaults**

In `internal/configs/config.balance.mobs.go` (or wherever `ScheduleMaxPathRetries` defaults are set), add:

```go
if b.SleepRegenMultiplier <= 0 {
	b.SleepRegenMultiplier = 5.0
}
if b.ScheduleWakeGraceRounds < 1 {
	b.ScheduleWakeGraceRounds = 50
}
```

Match the comparison style of nearby defaults (some use `< 1`, some use `<= 0`).

- [ ] **Step 4: Build**

```bash
go build ./...
go test ./internal/configs/ -v
```

Expected: clean build, configs tests still green.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "$(cat <<'EOF'
feat(configs): SleepRegenMultiplier + ScheduleWakeGraceRounds

Two new chunk 3.3 knobs. SleepRegenMultiplier (default 5.0)
multiplies HP/SP/CP per-round regen while bearer has Sleeping.
ScheduleWakeGraceRounds (default 50) prevents scheduled mobs
from immediately re-sleeping after a forced wake event.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Schedule loader recognizes `activity: sleeping`

**Files:**
- Modify: `internal/mobs/schedule_loader.go` (extend the activity warning whitelist)
- Modify: `internal/mobs/schedule_loader_test.go` (test that `sleeping` doesn't warn)

- [ ] **Step 1: Read the activity warning block**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "unknown activity" internal/mobs/schedule_loader.go
```

Locate the warning loop in `validateScheduleStandalone` or wherever the activity values are checked.

- [ ] **Step 2: Write the failing test**

Append to `internal/mobs/schedule_loader_test.go`:

```go
func TestValidateSchedule_AcceptsSleepingActivity(t *testing.T) {
	s := &Schedule{
		Id: "sleeper_test",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 1, Activity: "sleeping",
			 IdleCommands: []string{"emote snores."}},
		},
	}
	// Should validate cleanly. We can't assert on log output without a
	// capture, so just confirm no error is returned.
	if err := validateScheduleStandalone(s); err != nil {
		t.Errorf("expected sleeping activity to validate, got %v", err)
	}
}
```

This test relies on the validator NOT panicking for `sleeping`. Since `validateScheduleStandalone` returns an error rather than panicking for warnings, the test asserts on `err == nil`. The warning emission goes to logs but isn't returned.

- [ ] **Step 3: Run the test (likely already passes — it's a regression guard)**

```bash
go test ./internal/mobs/ -run TestValidateSchedule_AcceptsSleepingActivity -v
```

If the test passes already, the activity-value check doesn't warn on unknown values as an error — it just logs. The implementation change in Step 4 simply updates the recognized vocabulary so the log noise stops for `sleeping`.

- [ ] **Step 4: Extend the recognized vocabulary**

In `internal/mobs/schedule_loader.go`, find the activity check (likely something like `if seg.Activity != "" && seg.Activity != "craft" { mudlog.Warn(...) }`). Add `sleeping` to the recognized list:

```go
if seg.Activity != "" &&
	seg.Activity != "craft" &&
	seg.Activity != "sleeping" {
	mudlog.Warn("schedule", "id", s.Id, "segment", i,
		"msg", "unknown activity value", "value", seg.Activity)
}
```

- [ ] **Step 5: Run all schedule loader tests**

```bash
go test ./internal/mobs/ -v
go build ./...
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/schedule_loader.go internal/mobs/schedule_loader_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): schedule loader recognizes activity: sleeping

Extends the activity warning whitelist so the pilot retrofit
in T16 doesn't trigger boot-time warning noise.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Schedule executor: `WantsSleep` / `WantsWake` with grace cooldown

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs_schedule.go` (add WantsSleep/WantsWake to schedulePlan; populate in scheduleTickPlan; apply in applySchedulePlan)
- Modify: `internal/mobs/schedule.go` (add `SegmentByStart` helper for the WantsWake lookup)
- Modify: `internal/hooks/NewRound_IdleMobs_schedule_test.go` (tests for both new fields + grace cooldown)

- [ ] **Step 1: Re-read the existing executor**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
cat internal/hooks/NewRound_IdleMobs_schedule.go | head -130
```

- [ ] **Step 2: Add `SegmentByStart` helper**

In `internal/mobs/schedule.go`, append:

```go
// SegmentByStart returns the segment with matching Start hour, or nil.
// Used by the schedule executor to look up the prior segment's activity
// for transition detection.
func (s *Schedule) SegmentByStart(startHour int) *ScheduleSegment {
	if s == nil {
		return nil
	}
	for i := range s.Segments {
		if s.Segments[i].Start == startHour {
			return &s.Segments[i]
		}
	}
	return nil
}
```

- [ ] **Step 3: Write the failing tests**

Append to `internal/hooks/NewRound_IdleMobs_schedule_test.go`:

```go
func TestScheduleTick_WantsSleep_AtTargetDuringSleepSegment(t *testing.T) {
	registerSleepyScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234 // at the sleep target

	plan := scheduleTickPlan(mob, 23 /* sleeping segment */)
	if !plan.HasSchedule {
		t.Fatalf("expected HasSchedule=true")
	}
	if !plan.WantsSleep {
		t.Errorf("expected WantsSleep=true at target during sleep segment, got %+v", plan)
	}
}

func TestScheduleTick_WantsSleep_FalseWhenAwayFromTarget(t *testing.T) {
	registerSleepyScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 5678 // wrong room

	plan := scheduleTickPlan(mob, 23)
	if plan.WantsSleep {
		t.Errorf("expected WantsSleep=false away from target, got %+v", plan)
	}
}

func TestScheduleTick_WantsSleep_FalseDuringGraceCooldown(t *testing.T) {
	registerSleepyScheduleForTest(t)
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234
	mob.Character.SetMiscData("schedule_wake_round", 980) // 20 rounds ago

	plan := scheduleTickPlan(mob, 23)
	// Default grace cooldown is 50 rounds; 20 elapsed is within grace.
	if plan.WantsSleep {
		t.Errorf("expected WantsSleep=false during grace cooldown, got %+v", plan)
	}
}

func TestScheduleTick_WantsSleep_TrueAfterGraceCooldown(t *testing.T) {
	registerSleepyScheduleForTest(t)
	util.SetRoundCountForTest(1100)
	defer util.SetRoundCountForTest(0)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234
	mob.Character.SetMiscData("schedule_wake_round", 1000) // 100 rounds ago, > 50

	plan := scheduleTickPlan(mob, 23)
	if !plan.WantsSleep {
		t.Errorf("expected WantsSleep=true after grace cooldown, got %+v", plan)
	}
}

func TestScheduleTick_WantsWake_OnExitFromSleepSegment(t *testing.T) {
	registerSleepyScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234
	mob.Character.SetMiscData("schedule_last_seg_start", 22) // last tick: sleep segment

	plan := scheduleTickPlan(mob, 6 /* new segment: awake */)
	if !plan.WantsWake {
		t.Errorf("expected WantsWake=true on exit from sleep segment, got %+v", plan)
	}
}

func registerSleepyScheduleForTest(t *testing.T) {
	t.Helper()
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "sleepy_test",
		Segments: []mobs.ScheduleSegment{
			{Start: 6, End: 22, TargetRoom: 1234, Activity: "",
			 IdleCommands: []string{"emote wakes."}},
			{Start: 22, End: 6, TargetRoom: 1234, Activity: "sleeping",
			 IdleCommands: []string{"emote snores."}},
		},
	})
	t.Cleanup(func() { mobs.UnregisterScheduleForTest("sleepy_test") })
}
```

- [ ] **Step 4: Run the tests, confirm fail**

```bash
go test ./internal/hooks/ -run TestScheduleTick_Wants -v
```

Expected: compile errors — `plan.WantsSleep` and `plan.WantsWake` not defined.

- [ ] **Step 5: Extend `schedulePlan`**

In `internal/hooks/NewRound_IdleMobs_schedule.go`, add two fields to the struct:

```go
type schedulePlan struct {
	// ... existing fields ...

	// WantsSleep: current segment is activity: sleeping AND mob is at
	// segment target_room AND not within ScheduleWakeGraceRounds of a
	// recent wake event. Idempotent per-tick check; safe to fire even
	// if mob is already sleeping (applySchedulePlan checks).
	WantsSleep bool

	// WantsWake: transitioning OUT of an activity: sleeping segment
	// (the prior segment had activity sleeping; this tick's segment
	// does not). Triggers explicit CancelBuffsWithFlag(Sleeping).
	WantsWake bool
}
```

- [ ] **Step 6: Populate them in `scheduleTickPlan`**

Inside `scheduleTickPlan`, after the existing transition detection:

```go
// Detect wake transition (prior segment was activity: sleeping; new is not).
if plan.SegmentChanged {
	if prior := s.SegmentByStart(lastSegStart); prior != nil &&
		prior.Activity == "sleeping" && seg.Activity != "sleeping" {
		plan.WantsWake = true
	}
}

// Detect sleep desire (current segment is activity: sleeping; at target;
// past grace cooldown).
if seg.Activity == "sleeping" && mob.Character.RoomId == seg.TargetRoom {
	lastWoken := getMiscDataInt(&mob.Character, "schedule_wake_round")
	grace := int(configs.GetBalanceConfig().ScheduleWakeGraceRounds)
	if lastWoken == 0 || int(util.GetRoundCount())-lastWoken >= grace {
		plan.WantsSleep = true
	}
}
```

- [ ] **Step 7: Apply them in `applySchedulePlan`**

In `applySchedulePlan`, after the existing SegmentChanged block but BEFORE the pathing logic (wake first, then path, then sleep):

```go
// Chunk 3.3: wake first to clear stale sleep from a prior sleep segment.
if plan.WantsWake {
	mob.Character.CancelBuffsWithFlag(buffs.Sleeping)
}

// ... existing pathing / WantsPath / WantsHomeFallback logic ...

// Chunk 3.3: sleep last — only if at target and not on grace cooldown.
if plan.WantsSleep && !mob.Character.HasBuffFlag(buffs.Sleeping) {
	mob.Command("sleep")
}
```

The `!HasBuffFlag(Sleeping)` guard makes the apply idempotent — if the mob is already sleeping (from a previous tick), no re-issue.

Add `buffs` import if not present.

- [ ] **Step 8: Run all hooks tests**

```bash
go test ./internal/hooks/ -v
go build ./...
```

Expected: green.

- [ ] **Step 9: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs_schedule.go internal/hooks/NewRound_IdleMobs_schedule_test.go internal/mobs/schedule.go
git commit -m "$(cat <<'EOF'
feat(hooks): schedule executor recognizes activity: sleeping

Adds WantsSleep / WantsWake to schedulePlan. Sleep fires when
the mob is at the segment target and outside the
ScheduleWakeGraceRounds cooldown. Wake fires on transitions OUT
of a sleeping segment. Both go through the existing executor
plumbing; sleep delegates to mob.Command("sleep") which routes
to actions.Sleep.

Adds Schedule.SegmentByStart helper for prior-segment lookup
during wake detection.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Room rendering — `(asleep)` suffix

**Files:**
- Modify: `internal/rooms/roomdetails.go` (append `(asleep)` to VisiblePlayers / VisibleMobs entries when bearer has Sleeping)

- [ ] **Step 1: Re-read the existing suffix patterns (AFK precedent)**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '260,350p' internal/rooms/roomdetails.go
```

The AFK suffix at line ~270 is the precedent:

```go
playerEntry += ` <ansi fg="8">(AFK: ` + d.Message + `)</ansi>`
```

The sleeping suffix follows the same shape, but appended via a separate `if` after the AFK block.

- [ ] **Step 2: Add the player-side suffix**

In `internal/rooms/roomdetails.go`, find the VisiblePlayers loop (around lines 260-280). After the existing AFK suffix block but before `details.VisiblePlayers = append(...)`:

```go
// Chunk 3.3: sleeping suffix
if player.Character != nil && player.Character.HasBuffFlag(buffs.Sleeping) {
	playerEntry += ` <ansi fg="8">(asleep)</ansi>`
}

details.VisiblePlayers = append(details.VisiblePlayers, playerEntry)
```

Add `buffs` import if not present.

- [ ] **Step 3: Add the mob-side suffix**

In the VisibleMobs loop (around line 340), after `mobName := mob.Character.GetMobName(...)` and any other name decoration but before `details.VisibleMobs = append(...)`:

```go
mobNameStr := mobName.String()

// Chunk 3.3: sleeping suffix
if mob.Character.HasBuffFlag(buffs.Sleeping) {
	mobNameStr += ` <ansi fg="8">(asleep)</ansi>`
}

if mob.Character.IsCharmed() {
	visibleFriendlyMobs = append(visibleFriendlyMobs, mobNameStr)
} else {
	details.VisibleMobs = append(details.VisibleMobs, mobNameStr)
}
```

Note: the existing code uses `mobName.String()` inline twice (charmed vs not). Refactor to a single `mobNameStr` local first, then append `(asleep)`, then branch on charmed. Otherwise the suffix would have to be duplicated in both branches.

- [ ] **Step 4: Build and verify no test regressions**

```bash
go build ./...
go test ./internal/rooms/ -v
```

There may not be a direct unit test for this rendering — it's hard to test without a full room/player fixture. Manual smoke at T18 verifies visually.

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/roomdetails.go
git commit -m "$(cat <<'EOF'
feat(rooms): (asleep) suffix on sleeping occupants in look output

Mirrors the existing (AFK) suffix pattern. Players and mobs with
the Sleeping flag get a dim "(asleep)" appended to their name in
the "Also here:" line of the room render.

Schedule idle-commands (emote snores, etc.) continue firing on
top — the suffix is a static marker for players who don't catch
the idle-tick timing.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Wake trigger — failed steal

**Files:**
- Modify: `internal/actions/steal.go` (failure path — cancel Sleeping on victim if present, call OnSleeperWoken)

- [ ] **Step 1: Find the steal-fail branch**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "fail\|caught\|Failure" internal/actions/steal.go | head -10
```

Identify the code path that fires when a steal/pickpocket roll fails. There may be multiple failure paths (one for "caught while sneaking", one for "no items to steal", etc.) — wire all paths that represent a "noisy failure" (sneaking caught = wake; nothing to steal = quiet).

- [ ] **Step 2: Add the wake hook**

At each failure path that represents a noisy fail, after the failure messaging:

```go
// Chunk 3.3: failed theft wakes a sleeping victim.
if victim.HasBuffFlag(buffs.Sleeping) {
	victim.CancelBuffsWithFlag(buffs.Sleeping)
	mobs.OnSleeperWoken(victim)
}
```

`victim` here is the `*characters.Character` of the steal target. Adjust the variable name to match the local context.

Add `buffs` and `mobs` imports if not present.

- [ ] **Step 3: Build**

```bash
go build ./...
go test ./internal/actions/ -v
```

Expected: clean. Existing steal tests should still pass — the new code only fires when the victim is sleeping, which existing tests don't set up.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/steal.go
git commit -m "$(cat <<'EOF'
feat(actions): failed steal wakes sleeping victim

Successful steals stay silent (player's stealth reward). Failed
or caught steals cancel the victim's Sleeping buff and fire
OnSleeperWoken so the schedule grace cooldown applies.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Wake trigger — shout in room

**Files:**
- Modify: `internal/usercommands/shout.go` (cancel Sleeping on all in-room occupants after broadcast)
- Modify: `internal/mobcommands/shout.go` (same; if mob shout exists)

- [ ] **Step 1: Find the shout command**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
ls internal/usercommands/shout.go internal/mobcommands/shout.go 2>&1
sed -n '1,80p' internal/usercommands/shout.go
```

Identify where the shout broadcasts to room/zone occupants. The wake hook fires for SAME-ROOM listeners only (per spec — no adjacent-room propagation).

- [ ] **Step 2: Add the wake hook after broadcast**

After the room broadcast logic, iterate room occupants and wake sleepers:

```go
// Chunk 3.3: shout wakes sleepers in the same room.
for _, otherUserId := range room.GetPlayers() {
	if otherUserId == user.UserId {
		continue
	}
	if other := users.GetByUserId(otherUserId); other != nil {
		if other.Character.HasBuffFlag(buffs.Sleeping) {
			other.Character.CancelBuffsWithFlag(buffs.Sleeping)
			mobs.OnSleeperWoken(&other.Character)
		}
	}
}
for _, mobInstanceId := range room.GetMobs() {
	if m := mobs.GetInstance(mobInstanceId); m != nil {
		if m.Character.HasBuffFlag(buffs.Sleeping) {
			m.Character.CancelBuffsWithFlag(buffs.Sleeping)
			mobs.OnSleeperWoken(&m.Character)
		}
	}
}
```

Adjust the iteration helpers (`room.GetPlayers()`, `room.GetMobs()`) to the actual API.

If a mob shout command exists (`internal/mobcommands/shout.go`), mirror the change there.

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/shout.go internal/mobcommands/shout.go
git commit -m "$(cat <<'EOF'
feat(commands): shout wakes sleepers in the same room

After the shout broadcasts, iterate room occupants and cancel
Sleeping on any sleeper. Same-room only — adjacent-room sound
propagation is out of scope for chunk 3.3.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Wake trigger — light source on room entry

**Files:**
- Modify: `internal/usercommands/go.go` (post-arrival hook)

- [ ] **Step 1: Find the post-arrival hook**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "MoveToRoom\|moveToRoom\|entered\|Enter" internal/usercommands/go.go | head -10
```

Locate where the player arrives in the destination room (after `MoveToRoom` returns / after the room-enter event fires). The wake hook fires there.

- [ ] **Step 2: Determine the "carries light" check**

```bash
grep -rn "Illumination\|carries.*light\|HasLitItem" internal --include="*.go" | head -10
```

Check what API exists for detecting "this actor brings light into the room." Likely candidates:
- A buff flag like `buffs.Illumination` or `buffs.Lit`
- A method `user.Character.HasLitItem()` or `user.Character.CarriesLight()`
- An iteration through equipped/carried items checking a `Lit` field on ItemSpec

If no clean API exists, the simplest approximation for 3.3 is to check for the `Illumination` buff and any item with a `light` keyword tag — leave the deeper "every torch type" check for a future polish pass.

For this task, use the cleanest available signal. Document what you used in the commit message.

- [ ] **Step 3: Add the wake hook**

After successful arrival in the destination room:

```go
// Chunk 3.3: a player arriving with a light source wakes any sleepers
// in the destination room. False positives possible if the room was
// already lit — acceptable for chunk 3.3 scope (most NPC sleep rooms
// are dim/dark indoors).
arrivingLightSource := user.Character.HasBuffFlag(buffs.Illumination) ||
	playerCarriesLitItem(user) // helper — investigate the real check
if arrivingLightSource {
	for _, mobInstanceId := range destRoom.GetMobs() {
		if m := mobs.GetInstance(mobInstanceId); m != nil &&
			m.Character.HasBuffFlag(buffs.Sleeping) {
			m.Character.CancelBuffsWithFlag(buffs.Sleeping)
			mobs.OnSleeperWoken(&m.Character)
		}
	}
	for _, otherUserId := range destRoom.GetPlayers() {
		if otherUserId == user.UserId {
			continue
		}
		if other := users.GetByUserId(otherUserId); other != nil &&
			other.Character.HasBuffFlag(buffs.Sleeping) {
			other.Character.CancelBuffsWithFlag(buffs.Sleeping)
			mobs.OnSleeperWoken(&other.Character)
		}
	}
}
```

Replace `playerCarriesLitItem(user)` with the actual check from Step 2.

- [ ] **Step 4: Build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/go.go
git commit -m "$(cat <<'EOF'
feat(commands): light source on room entry wakes sleepers

When a player arrives in a room carrying a light source (lit item
or Illumination buff), iterate the destination room's occupants
and cancel Sleeping on any sleeper. False positives possible when
the room was already lit; acceptable for chunk 3.3 scope.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Damage pipeline — `forceCrit` parameter

**Files:**
- Modify: `internal/combat/damage_pipeline.go` (add `forceCrit bool` parameter to the crit-resolving entry point)
- Modify: `internal/combat/damage_pipeline_test.go` (test that forceCrit bypasses the Z-score check)

- [ ] **Step 1: Find the crit-resolution path**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "ZScore\|CritZScore\|isCrit\|IsCritical" internal/combat/damage_pipeline.go internal/combat/*.go | head -10
sed -n '60,110p' internal/combat/damage_pipeline.go
```

Identify the function that takes raw damage and produces final damage + crit flag. The `forceCrit bool` parameter goes there.

If the existing crit decision lives in `dice.OpposedRollStat` or similar, the change is at that boundary instead — pass through.

- [ ] **Step 2: Write the failing test**

Append to `internal/combat/damage_pipeline_test.go`:

```go
func TestDamageWithForceCrit_BypassesZScoreCheck(t *testing.T) {
	// Without forceCrit, a non-extreme roll should not crit.
	dmg, isCrit := computeFinalDamageWithForceCrit(/* args matching the
		signature you create */, false /* forceCrit */)
	if isCrit {
		t.Errorf("expected non-crit without forceCrit, got crit")
	}

	// With forceCrit, the same input should crit.
	dmg2, isCrit2 := computeFinalDamageWithForceCrit(/* same args */, true)
	if !isCrit2 {
		t.Errorf("expected crit with forceCrit=true")
	}
	if dmg2 <= dmg {
		t.Errorf("expected forced crit damage > non-crit damage; got %d vs %d", dmg2, dmg)
	}
}
```

The exact signature depends on where the crit check lives. Adapt to the real API.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/combat/ -run TestDamageWithForceCrit -v
```

Expected: compile error or test fails (forceCrit parameter doesn't exist yet).

- [ ] **Step 4: Add the `forceCrit` parameter**

In the crit-resolution function (likely the variance roll → crit-decision boundary), add a new `forceCrit bool` parameter. When `forceCrit=true`, bypass the Z-score check and treat the result as a crit:

```go
// Existing logic snipped for illustration; adjust to actual code.
func computeFinalDamageWithForceCrit(/* existing args */, forceCrit bool) (int, bool) {
	zScore := /* existing Z-score calc */
	isCrit := zScore >= configs.GetBalanceConfig().CritZScoreThreshold
	if forceCrit {
		isCrit = true
		// Use a high zScore so any downstream "crit magnitude" logic
		// treats this as a clear crit. Pick threshold + 0.5 so it
		// reads as "definite crit" not "borderline."
		if zScore < float64(configs.GetBalanceConfig().CritZScoreThreshold)+0.5 {
			zScore = float64(configs.GetBalanceConfig().CritZScoreThreshold) + 0.5
		}
	}
	// ... rest unchanged ...
	return finalDmg, isCrit
}
```

Callers that don't care pass `false`. T14 wires the `true` case from the round dispatcher.

- [ ] **Step 5: Run the test**

```bash
go test ./internal/combat/ -v
go build ./...
```

Expected: green build, new test passes, existing combat tests unaffected (all existing callers pass `forceCrit=false`).

- [ ] **Step 6: Commit**

```bash
git add internal/combat/damage_pipeline.go internal/combat/damage_pipeline_test.go
git commit -m "$(cat <<'EOF'
feat(combat): forceCrit parameter on damage pipeline

Adds a forceCrit bool parameter to the crit-resolution path.
When true, bypasses the Z-score threshold check and treats the
result as a clear crit. The Z-score is bumped to threshold+0.5
so downstream crit-magnitude logic sees a confident crit, not a
borderline one.

T14 wires this from the round dispatcher's sleeping-victim
snapshot. Other future first-hit-crit triggers (surprise attack,
backstab) can use the same parameter without further pipeline
changes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Round dispatcher — snapshot sleeping victims, pass `forceCrit`

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go` (snapshot sleeping victims at start-of-round, pass to handleCombatRound or whatever consumes the snapshot)

- [ ] **Step 1: Find handleCombatRound and the round-victim iteration**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "handleCombatRound" internal/hooks/NewRound_DoCombat.go
sed -n '140,170p' internal/hooks/NewRound_DoCombat.go
sed -n '270,320p' internal/hooks/NewRound_DoCombat.go
```

Identify where the round iterates over combat pairs (attacker, defender). The snapshot needs to happen ONCE per round, BEFORE any damage events resolve.

- [ ] **Step 2: Add the snapshot at start-of-round**

In the appropriate `NewRound_DoCombat` function, at the top of the round-resolution loop (before any iteration over combat pairs):

```go
// Chunk 3.3: snapshot sleeping victims at start-of-round so the
// cancel-on-damage flag that fires mid-round doesn't blunt later
// attackers' crit payoff in the same round. Other future first-hit-crit
// triggers (surprise attack, backstab, etc.) can add parallel snapshot
// checks at this same site.
sleepingVictims := map[int]bool{} // keyed by character's identifier
for _, /* each defender in this round */ {
	if def.Character.HasBuffFlag(buffs.Sleeping) {
		sleepingVictims[def.Character.GetIdentifier()] = true
	}
}
```

The exact iteration and identifier scheme depends on the existing code. Goal: for each defender that will receive a damage event in this round, mark them in the snapshot.

- [ ] **Step 3: Pass `forceCrit` through `handleCombatRound`**

`handleCombatRound` already takes attacker / defender args. Add a `forceCrit bool` parameter (or look up from the snapshot inside if the function has access). The damage call inside `handleCombatRound` then passes `forceCrit` into the pipeline function T13 modified.

```go
forceCrit := sleepingVictims[def.Character.GetIdentifier()]
handleCombatRound(atk, def, evt, moonMod, &cfg, &affectedPlayerIds, &affectedMobInstanceIds, forceCrit)
```

Inside `handleCombatRound`, pass `forceCrit` to whatever damage function it calls (the one T13 modified).

- [ ] **Step 4: Build and run combat tests**

```bash
go build ./...
go test ./internal/hooks/ -v
go test ./internal/combat/ -v
```

Expected: green. Existing tests pass `forceCrit=false` (the default behavior).

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat.go
git commit -m "$(cat <<'EOF'
feat(hooks): first-hit-crit snapshot for sleeping victims

handleCombatRound now snapshots victims with the Sleeping flag at
start-of-round. All damage events in the round against snapshotted
victims force-crit, even after cancel-on-damage clears the buff
mid-round. Sleep becomes a high-stakes tactical opportunity.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Regen boost — `SleepRegenMultiplier`

**Files:**
- Modify: `internal/characters/resources.go` (HealthPerRound, StaminaPerRound, ConvictionPerRound — apply multiplier when bearer has Sleeping)
- Modify: `internal/characters/resources_test.go` (test the boost applies / doesn't apply)

- [ ] **Step 1: Re-read the regen functions**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '190,240p' internal/characters/resources.go
```

- [ ] **Step 2: Write the failing tests**

Append to `internal/characters/resources_test.go` (if the file doesn't exist, create it):

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
)

func TestHealthPerRound_SleepMultiplier(t *testing.T) {
	c := &Character{}
	c.HealthMax = ConfigCappedInt{Value: 1000}
	// Without Sleeping, base regen at default 1% = 10.
	base := c.HealthPerRound()
	if base != 10 {
		t.Fatalf("setup check: expected base=10, got %d", base)
	}

	// Apply Sleeping; expect 5× multiplier.
	c.AddBuff(15, 0, "") // adjust to real AddBuff sig
	boosted := c.HealthPerRound()
	if boosted != 50 {
		t.Errorf("expected boosted=50 (5× base), got %d", boosted)
	}
}

func TestStaminaPerRound_SleepMultiplier(t *testing.T) {
	c := &Character{}
	c.StaminaMax = ConfigCappedInt{Value: 1000}
	base := c.StaminaPerRound()

	c.AddBuff(15, 0, "")
	boosted := c.StaminaPerRound()
	if boosted != base*5 {
		t.Errorf("expected boosted = 5× base (%d), got %d", base*5, boosted)
	}
}

func TestConvictionPerRound_SleepMultiplier(t *testing.T) {
	c := &Character{}
	c.ConvictionMax = ConfigCappedInt{Value: 1000}
	base := c.ConvictionPerRound()

	c.AddBuff(15, 0, "")
	boosted := c.ConvictionPerRound()
	if boosted != base*5 {
		t.Errorf("expected boosted = 5× base (%d), got %d", base*5, boosted)
	}
}
```

If `ConfigCappedInt` isn't the right type for the pool fields, check `internal/characters/character.go` for the actual type. The `AddBuff` signature also needs to match.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/characters/ -run TestHealthPerRound_Sleep -v
```

Expected: fail (multiplier not applied yet).

- [ ] **Step 4: Apply the multiplier in all three regen functions**

In `internal/characters/resources.go`, modify `HealthPerRound`, `StaminaPerRound`, and `ConvictionPerRound`. For each, add the multiplier just before the final `return`:

```go
func (c *Character) HealthPerRound() int {
	b := configs.GetBalanceConfig()
	pct := float64(b.PlayerHealthRegenPct)
	if c.IsMob {
		pct = float64(b.MobHealthRegenPct)
	}
	pct += float64(c.StatMod(string(statmods.HealthRecovery))) / 100.0
	base := int(pct * float64(c.HealthMax.Value))
	if base < 1 {
		base = 1
	}

	// Chunk 3.3: 5× regen while sleeping.
	if c.HasBuffFlag(buffs.Sleeping) {
		mult := float64(b.SleepRegenMultiplier)
		if mult > 0 {
			base = int(float64(base) * mult)
		}
	}

	return base
}
```

Repeat the same multiplier block at the bottom of `StaminaPerRound` and `ConvictionPerRound`.

For `StaminaPerRound`, the multiplier should compose with the existing mutation-multiplier — apply the sleep multiplier AFTER the mutation one:

```go
// Apply stamina_regen_multiplier mutations (existing)
if mult := mutations.GetStaminaRegenMultiplier(c.Mutations); mult != 0 {
	base = int(float64(base) * (1.0 + mult))
	if base < 1 {
		base = 1
	}
}

// Chunk 3.3: sleep multiplier composes on top.
if c.HasBuffFlag(buffs.Sleeping) {
	smult := float64(configs.GetBalanceConfig().SleepRegenMultiplier)
	if smult > 0 {
		base = int(float64(base) * smult)
	}
}

return base
```

Add `buffs` import.

- [ ] **Step 5: Run the tests, confirm pass**

```bash
go test ./internal/characters/ -v
go build ./...
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/characters/resources.go internal/characters/resources_test.go
git commit -m "$(cat <<'EOF'
feat(characters): SleepRegenMultiplier for HP/SP/CP regen

All three per-round regen functions now multiply by
SleepRegenMultiplier (default 5.0) when the bearer has the
Sleeping flag. For stamina, composes on top of the existing
stamina_regen_multiplier mutation modifier.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: Pilot retrofit — Thornwall sleep segments

**Files:**
- Modify: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_smith.yaml`
- Modify: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_tavern_keeper.yaml`
- Modify: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_temple_priest.yaml`

- [ ] **Step 1: Edit the smith schedule**

In `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_smith.yaml`, find the 22-6 (loft) segment. Change `activity: ""` to `activity: sleeping`. The 6-9 (waking) segment stays `activity: ""`.

- [ ] **Step 2: Edit the tavern keeper schedule**

In `thornwall_tavern_keeper.yaml`, find the 22-6 (quarters) segment. Change `activity: ""` to `activity: sleeping`.

- [ ] **Step 3: Edit the temple priest schedule**

In `thornwall_temple_priest.yaml`, find the 22-4 (chamber) segment. Change `activity: ""` to `activity: sleeping`. The 4-6 (rise) and 10-12 (rest) chamber segments stay non-sleeping (those are waking states with sleepy flavor).

- [ ] **Step 4: Confirm clean build (validator path)**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build ./...
```

The validator will recognize `sleeping` (per T7); no warning noise.

- [ ] **Step 5: Boot smoke**

Run the server briefly. Confirm `mobs.LoadSchedules() loadedCount=3` logs cleanly. No "unknown activity value" warnings.

You can ask the user to run the server if you can't boot it yourself.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/schedules/thornwall_city/thornwall_smith.yaml _datafiles/world/dogmud/schedules/thornwall_city/thornwall_tavern_keeper.yaml _datafiles/world/dogmud/schedules/thornwall_city/thornwall_temple_priest.yaml
git commit -m "$(cat <<'EOF'
feat(content): Thornwall pilot sleep segments use activity: sleeping

Kerra's 22-6 loft, Marek's 22-6 quarters, and Olen's 22-4
chamber segments now declare activity: sleeping. The schedule
executor will apply the Sleeping buff and cancel it on segment
exit. Other chamber/loft segments stay non-sleeping (those are
waking states with sleepy flavor, not actual sleep).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Use `git add -f` if the dogmud gitignore catches these files.

---

## Task 17: Documentation pass

**Files:**
- Modify: `docs/schemas/schedule.md` (add `sleeping` to activity values)
- Modify: `internal/buffs/context.md` (Sleeping Flag + cancel-on-damage flag)
- Modify: `internal/actions/context.md` (Sleep actor function)
- Modify: `internal/combat/context.md` (first-hit-crit on sleepers + snapshot pattern note)
- Modify: `internal/configs/context.md` (SleepRegenMultiplier + ScheduleWakeGraceRounds)
- Modify: `CLAUDE.md` (Sleep Mechanics subsection)
- Create: `_datafiles/world/dogmud/templates/help/sleep.template`
- Modify: `_datafiles/world/dogmud/templates/help/stand.template` (or default if no dogmud override exists)

- [ ] **Step 1: Update `docs/schemas/schedule.md`**

Read the file first. Find the activity vocabulary section. Add an entry for `sleeping`:

```markdown
- `sleeping` — When the segment is active and the mob is at the
  segment's `target_room`, the schedule executor applies the
  Sleeping buff. The buff cancels on segment exit and on any
  wake trigger (damage, failed steal, shout in room, light source
  entering, the `stand` command). Sleeping characters receive 5×
  regen on all pools and any attacker's first round of attacks
  auto-crits. After a forced wake during a sleep segment, the
  schedule executor will not re-sleep the mob for
  `ScheduleWakeGraceRounds` rounds (default 50).
```

- [ ] **Step 2: Update `internal/buffs/context.md`**

Add a row to the state-query flag table:

```markdown
| `Sleeping` | "sleeping" | Bearer is asleep — gates regen boost, first-hit-crit, room rendering. Chunk 3.3. |
```

Add a row to the cancel-flag table:

```markdown
| `CancelOnDamage` | "cancel-on-damage" | Buff cancels when any damage is applied to bearer. Wired in damage pipeline. Chunk 3.3. |
```

- [ ] **Step 3: Update `internal/actions/context.md`**

Add a row to the actor-function table:

```markdown
| `Sleep(actor, opts)` | Applies buff 15 (Sleeping) to actor. Combat-gated. Idempotent. Used by sleep user command, sleep mob command, and schedule executor. Chunk 3.3. |
```

- [ ] **Step 4: Update `internal/combat/context.md`**

Add to the crit-resolution section:

```markdown
### First-hit crit on sleepers (chunk 3.3)

The damage pipeline accepts a `forceCrit bool` parameter that
bypasses the Z-score threshold. `handleCombatRound` snapshots
victims with the Sleeping flag at start-of-round and passes
`forceCrit=true` for damage events against snapshotted victims.
All attackers in the same round share the crit payoff before
`cancel-on-damage` clears the buff mid-round. Subsequent rounds
are normal.

Other future first-hit-crit triggers (surprise attack, backstab)
can add parallel snapshot checks at the same start-of-round site.
```

- [ ] **Step 5: Update `internal/configs/context.md`**

Add two rows to the balance knobs table:

```markdown
| `SleepRegenMultiplier` | 5.0 | HP/SP/CP per-round regen multiplier when bearer has Sleeping. Chunk 3.3. |
| `ScheduleWakeGraceRounds` | 50 | After a forced wake during a sleep segment, suppress re-sleep for N rounds (~200 sec real-time at default tick rate). Chunk 3.3. |
```

- [ ] **Step 6: Update `CLAUDE.md`**

Append a subsection (placement: near the "Sleep" / "NPC Schedules" subsection if any, or at the bottom of "Project Context"):

```markdown
## Sleep Mechanics

Players and NPCs can `sleep` (the verb — no slash). Sleepers gain
5× HP/SP/CP regen but the entire first round of attacks against
them auto-crits. Wake triggers: any damage, failed steal,
shout-in-room, light source entering room, the `stand` command,
or schedule segment end for scheduled mobs. Scheduled NPCs sleep
during segments with `activity: sleeping` (see
`docs/schemas/schedule.md`); a grace cooldown
(`ScheduleWakeGraceRounds`, default 50) prevents immediate
re-sleep after a wake event. Use `actions.Sleep(actor, opts)` for
the actor-parity entry point. State queryable via
`HasBuffFlag(buffs.Sleeping)`.
```

- [ ] **Step 7: Create the sleep helpfile**

Create `_datafiles/world/dogmud/templates/help/sleep.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">sleep</ansi>

The <ansi fg="command">sleep</ansi> command lies you down to rest. You can sleep in
any room — there is no safe-room gate.

While asleep, all your pools (health, stamina, conviction)
regenerate <ansi fg="white-bold">five times faster</ansi> than normal.

But sleep is dangerous: the entire first round of attacks
against a sleeper <ansi fg="red">automatically critical-hits</ansi>. Sleeping in
hostile territory is an invitation to assassination.

You wake when:
  - You take any damage.
  - Someone fails a steal or pickpocket against you.
  - Someone shouts in your room.
  - Someone enters your room carrying a light source.
  - You use <ansi fg="command">stand</ansi>.

NPCs follow daily routines and may be asleep at night. See
<ansi fg="command">help time</ansi>.
```

- [ ] **Step 8: Update the stand helpfile**

Find `_datafiles/world/dogmud/templates/help/stand.template` (if it exists) or `_datafiles/world/default/templates/help/stand.template`. Append:

```
If you are sleeping, <ansi fg="command">stand</ansi> will wake you up.
```

- [ ] **Step 9: Build and commit**

```bash
go build ./...
git add docs/schemas/schedule.md internal/buffs/context.md internal/actions/context.md internal/combat/context.md internal/configs/context.md CLAUDE.md _datafiles/world/dogmud/templates/help/sleep.template _datafiles/world/dogmud/templates/help/stand.template _datafiles/world/default/templates/help/stand.template 2>/dev/null
git commit -m "$(cat <<'EOF'
docs: sleep mechanics across context.md / CLAUDE.md / helpfiles

Schedule schema gains the sleeping activity entry. context.md
updates for buffs/actions/combat/configs. CLAUDE.md gains a
Sleep Mechanics subsection. New sleep helpfile; stand helpfile
appended with the wake-up note.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Use `git add -f` for any dogmud-path files that hit the gitignore quirk.

---

## Task 18: Smoketester goal file + manual smoke + roadmap closeout

**Files:**
- Create: `tools/testing/goals/3.3-sleep-mechanics.yaml`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Move: spec and plan to `completed/` subdirectories

- [ ] **Step 1: Author the smoketester goal file**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
cat tools/testing/goals/3.2-schedule-observation.yaml | head -30
```

Mirror the schema. Create `tools/testing/goals/3.3-sleep-mechanics.yaml`:

```yaml
description: "Exercise chunk 3.3 sleep mechanics: player sleep + regen, sleeper crit, NPC sleep cycle, wake triggers."

goals:
  - "Sleep as the test character via the `sleep` command; confirm the room emote fires."
  - "Wait 5 rounds. Compare HP/SP/CP regen to baseline — expect ~5× normal rate."
  - "Take any action; confirm Sleeping cancels (cancel-on-action)."
  - "Sleep again. Use admin `mob spawn` to summon a hostile mob, attack it, and confirm the first round of attacks against you (if the mob retaliates) crits — or, conversely, attack a sleeping NPC and confirm your first round crits."
  - "Use `time set 23` and look at Blacksmith Kerra in her loft (room 5101). Confirm the `(asleep)` suffix in the 'Also here:' line."
  - "Successfully pickpocket Kerra at 23 — Kerra stays asleep."
  - "Fail a pickpocket on Kerra at 23 — Kerra wakes."
  - "Wait ScheduleWakeGraceRounds (~50 rounds). Confirm Kerra re-sleeps automatically."
  - "Enter Marek's tavern at 23 carrying a lit torch. Confirm Marek wakes (he should be in his quarters at 23 — adjust by going `up` from tavern to quarters)."

pass_criteria:
  - "sleep command applies the Sleeping buff and 5× regen visibly accelerates pools"
  - "first-round attacks against sleepers crit"
  - "Scheduled NPC sleep segments render with (asleep) suffix"
  - "All four wake triggers function (damage, failed steal, shout, light)"
  - "Grace cooldown prevents immediate re-sleep after a wake event"

notes:
  - "If admin time-set is not available, observe Kerra at a natural 22-6 hour by waiting real-time."
  - "If the test character isn't admin-flagged, skip the admin-spawn steps and rely on natural mob encounters."
```

Adapt the schema if the actual goal-file format differs.

- [ ] **Step 2: Update the roadmap**

Edit `MOB_ALIVENESS_ROADMAP.md`. Find the chunk 3.3 progress-tracker row:

```markdown
| 3.3 | Routine | Sleeping / wake states | S | 3.1 | Not started |
```

Change to:

```markdown
| 3.3 | Routine | Sleeping / wake states | M | 3.1 | Done |
```

(Size updated from S → M to reflect the expanded scope with player parity / regen / first-hit-crit.)

Find the chunk 3.3 detailed section. Append:

```markdown
- **Shipped:** Sleeping is a queryable state-flag (`buffs.Sleeping`)
  applied via `actions.Sleep(actor)` from both player `sleep` and
  mob `sleep` commands, and from the schedule executor's
  `activity: sleeping` segment hook. Sleepers gain 5× HP/SP/CP
  regen (`SleepRegenMultiplier`, default 5.0). Attackers in the
  first round against a sleeper auto-crit via a start-of-round
  victim snapshot + `forceCrit bool` on the damage pipeline.
  Wake triggers: damage (new `cancel-on-damage` flag), failed
  steal, shout-in-room, light source on room entry, `stand`
  command. Schedule executor honors a grace cooldown
  (`ScheduleWakeGraceRounds`, default 50) after a forced wake.
  Room render appends `(asleep)` to occupant names. Three
  Thornwall pilots retrofit. Spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.3-sleeping-wake-states-design.md`,
  plan at
  `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.3-sleeping-wake-states.md`.
```

If chunk 5.1 (Town Justice) appears in the roadmap as a future chunk, append a note on it:

```markdown
- **3.3 leaves available for 5.1:** Town Justice may wish to scale
  faction response by victim state at crime time. The data is
  queryable live (`victim.Character.HasBuffFlag(buffs.Sleeping)`)
  at the moment `crimes.Record(...)` is called; no Crime-schema
  change is required up front.
```

- [ ] **Step 3: Move spec and plan to `completed/`**

```bash
git mv docs/superpowers/specs/2026-05-25-mob-aliveness-3.3-sleeping-wake-states-design.md docs/superpowers/specs/completed/
git mv docs/superpowers/plans/2026-05-25-mob-aliveness-3.3-sleeping-wake-states.md docs/superpowers/plans/completed/
```

- [ ] **Step 4: Final verification**

```bash
go build ./...
go test ./...
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add tools/testing/goals/3.3-sleep-mechanics.yaml MOB_ALIVENESS_ROADMAP.md docs/superpowers/specs/completed/ docs/superpowers/plans/completed/
git commit -m "$(cat <<'EOF'
chore(roadmap): mark 3.3 sleeping/wake states Done

Sizing bumped S → M to reflect player parity, regen boost, and
first-hit-crit additions. Smoketester goal file authored at
tools/testing/goals/3.3-sleep-mechanics.yaml. Spec + plan
moved to completed/.

Note: Town Justice (chunk 5.1) can scale faction response by
victim state at crime-record time via a live HasBuffFlag query;
no Crime-schema change required up front.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 6: Pre-push checklist (only when ready to push to prod)**

Per CLAUDE.md SOP. Do NOT push without:

1. Boot the server locally — confirm clean startup past `mobs.LoadDataFiles()`.
2. Set `Logging.LogToFile: false` in `_datafiles/config.yaml`.
3. Update `PATCH_NOTES.md` with the chunk summary.
4. Smoketester session has either passed or any anomalies are documented in the PATCH_NOTES entry.

---

## Self-Review

**Spec coverage check:**

- ✅ `buffs.Sleeping` flag constant + buff YAML update → T1
- ✅ `cancel-on-damage` flag constant + damage-pipeline wiring → T1 + T2
- ✅ `OnSleeperWoken` helper → T2
- ✅ `actions.Sleep` actor function → T3
- ✅ `sleep` user command + mob command → T4
- ✅ `stand` extension → T5
- ✅ Config knobs (`SleepRegenMultiplier`, `ScheduleWakeGraceRounds`) → T6
- ✅ Schedule loader recognizes `activity: sleeping` → T7
- ✅ Schedule executor `WantsSleep` / `WantsWake` with grace cooldown → T8
- ✅ Room rendering `(asleep)` suffix → T9
- ✅ Wake trigger: failed steal → T10
- ✅ Wake trigger: shout in room → T11
- ✅ Wake trigger: light source on room entry → T12
- ✅ `forceCrit` parameter on damage pipeline → T13
- ✅ Round-dispatcher snapshot + force-crit on sleeping victims → T14
- ✅ Regen boost (5× HP/SP/CP) → T15
- ✅ Pilot retrofit (3 schedule YAMLs) → T16
- ✅ Documentation pass (schema doc, context.md, CLAUDE.md, helpfiles) → T17
- ✅ Smoketester goal file + roadmap closeout → T18

**Dependency check:**

- T1 (foundation) blocks T2 (uses CancelOnDamage), T3 (uses Sleeping), T5 (uses Sleeping), T9, T10, T11, T12, T14, T15.
- T2 (OnSleeperWoken) blocks T5, T10, T11, T12.
- T3 (actions.Sleep) blocks T4, T8.
- T4 (sleep mob command) blocks T8 (schedule executor calls it).
- T6 (config knobs) blocks T8 (grace cooldown), T15 (regen multiplier).
- T7 (loader recognizes sleeping) blocks T16 (pilot uses activity: sleeping).
- T13 (forceCrit param) blocks T14 (uses param).
- T8 (schedule executor) + T16 (pilot) together enable end-to-end NPC sleep.

Safe parallel groups (after their dependencies land):
- {T9, T15} once T1 lands
- {T10, T11, T12} once T1 + T2 land
- {T13} independent
- {T14} once T13 lands

Sequential dispatch is safest for content-fragile tasks (T8, T16) since botched intermediate state could break boot. The above ordering avoids that.

**Placeholder scan:**

Searched for "TBD", "TODO", "implement later", "appropriate error handling" — none found. Where the plan requires investigation (e.g., T2 step 1 finding the damage application chokepoint, T12 step 2 finding the light-source check), the steps are explicit about what to investigate and what fallback to take. No silent placeholders.

**Type consistency check:**

- `buffs.Sleeping` (Flag constant) used consistently from T1 onward.
- `buffs.CancelOnDamage` (Flag constant) used consistently in T2 and T17.
- `mobs.OnSleeperWoken(c *characters.Character)` signature used identically in T2, T5, T10, T11, T12.
- `actions.Sleep(actor Actor, opts SleepOptions) Result` signature used identically in T3, T4, T8 (via mob command).
- MiscData keys: `schedule_wake_round` used consistently in T2 and T8.
- Config knob field names: `SleepRegenMultiplier` and `ScheduleWakeGraceRounds` used consistently in T6, T8, T15.
- `schedulePlan.WantsSleep` and `WantsWake` defined in T8, no usage elsewhere — consistent.
- `forceCrit bool` parameter introduced in T13, consumed in T14 — consistent.

Plan is internally consistent and ready for execution.
