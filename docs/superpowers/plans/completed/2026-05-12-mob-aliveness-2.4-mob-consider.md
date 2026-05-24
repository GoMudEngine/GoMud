# Mob Aliveness 2.4 — Mob `consider` + Threat-Aware Behaviors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate `consider` via the actor pattern (like chunk 2.1's `actions.Buy`) so players and mobs share one code path, then add two behavior-tree primitives (`target_power_ratio_above`/`_below` condition + `target_weakest_mob_in_room` action) and demo wiring on the lookout archetype + a new predator archetype for ironwind steppe wolves. No `combat.PowerScore` math changes — gear already flows through `ValueAdj`/`Get*Mitigation` pipes; the audit deliverable is documentation.

**Architecture:** `internal/actions/consider.go` holds the shared `Consider(actor, target) ConsiderResult` function with text emission via `actor.SendText` (no-op for `MobActor`). Player wrapper (`internal/usercommands/consider.go`) and mob wrapper (`internal/mobcommands/consider.go`) both collapse to ~15 lines. Btree condition lives in new `internal/behaviortree/conditions_combat.go`; btree action joins the existing `actions_combat.go`. `MobIdle_HandleIdleMobs.go` already fires `mob_idle` events into the btree before legacy idle logic, so the predator archetype's leading `mob_idle` branch naturally preempts wander/lookfortrouble.

**Tech Stack:** Go 1.21+, existing `internal/actions` actor abstraction (chunk 2.1), existing `internal/combat.PowerScore`, existing `mob.HatesMob` predicate for faction/pack awareness.

**Spec:** `docs/superpowers/specs/2026-05-12-mob-aliveness-2.4-mob-consider-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/actions/consider.go` | NEW | `Consider(actor, target) ConsiderResult` shared compute + text |
| `internal/actions/consider_test.go` | NEW | Unit tests for the shared function |
| `internal/usercommands/consider.go` | REWRITE | Thin wrapper delegating to `actions.Consider` |
| `internal/mobcommands/consider.go` | NEW | Thin mob wrapper delegating to `actions.Consider` |
| `internal/mobcommands/mobcommands.go` | MODIFY | Register `consider` in the `mobCommands` map |
| `internal/mobcommands/mobcommands_test.go` | MODIFY | Add `consider` to the parity expected-commands list |
| `internal/behaviortree/conditions_combat.go` | NEW | `condTargetPowerRatioAbove`, `condTargetPowerRatioBelow`, `resolveTargetPower` |
| `internal/behaviortree/conditions.go` | MODIFY | Register both new conditions in `init()` |
| `internal/behaviortree/conditions_combat_test.go` | NEW | Unit tests for both conditions |
| `internal/behaviortree/actions_combat.go` | MODIFY | Add `actTargetWeakestMobInRoom` |
| `internal/behaviortree/actions.go` | MODIFY | Register `target_weakest_mob_in_room` in `init()` |
| `internal/behaviortree/actions_combat_test.go` | MODIFY | Add test cases for the new action |
| `internal/behaviortree/context.md` | MODIFY | Document the new condition + action |
| `internal/combat/context.md` | MODIFY | Append "Power Scoring & Gear Contribution" section |
| `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml` | MODIFY | Add leading `player_enter` branch |
| `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml` | NEW | New archetype copying generic_fighter + leading mob_idle predation |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/205-steppe_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/206-young_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/223-scarred_wolf.yaml` | MODIFY | `behavior_archetype: predator` |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.4 Done, roll-up 11/41 → 12/41 |

---

## Task 1: `actions.Consider` shared function + tests

**Files:**
- Create: `internal/actions/consider.go`
- Create: `internal/actions/consider_test.go`

The shared compute + text emission. `MobActor.SendText` is already a no-op, so the same code path runs silently for mobs.

- [ ] **Step 1: Create `internal/actions/consider.go`**

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
)

// ConsiderResult is the structured output of an actor's consider
// action. Ratio is self power divided by target power; values > 1
// mean the actor outclasses the target.
type ConsiderResult struct {
	Ratio          float64 // 0 if target power is 0 (degenerate)
	SelfPower      float64
	TargetPower    float64
	TargetName     string
	TargetIsPlayer bool
}

// Consider computes a power-ratio assessment of target from
// actor's POV. For UserActor: also formats a colored prediction
// string and calls actor.SendText(...). For MobActor: SendText
// is a no-op (existing actor abstraction), so the math runs
// silently. Triggers OnStatUse("perception") on the actor.
func Consider(actor Actor, target Actor) ConsiderResult {
	actor.OnStatUse("perception")

	selfChar := actor.GetCharacter()
	targetChar := target.GetCharacter()

	selfPower := combat.PowerScore(*selfChar)
	targetPower := combat.PowerScore(*targetChar)

	result := ConsiderResult{
		SelfPower:      selfPower,
		TargetPower:    targetPower,
		TargetName:     target.GetName(),
		TargetIsPlayer: target.IsPlayer(),
	}
	if targetPower > 0 {
		result.Ratio = selfPower / targetPower
	}

	// Format and emit prediction text. UserActor delivers to the
	// player connection; MobActor.SendText is a no-op.
	considerType := "mob"
	if result.TargetIsPlayer {
		considerType = "user"
	}
	actor.SendText(fmt.Sprintf(
		`You consider <ansi fg="%sname">%s</ansi>...`,
		considerType, result.TargetName))
	actor.SendText(fmt.Sprintf(
		`Your instincts tell you: %s`, predictionFor(result.Ratio)))

	return result
}

// predictionFor maps a power ratio to the canonical prediction
// text + color. Ratio = 0 (degenerate target) is treated as
// "will not survive" — preserved verbatim from the pre-refactor
// usercommands.Consider behavior.
func predictionFor(ratio float64) string {
	switch {
	case ratio > 4:
		return `<ansi fg="blue-bold">They pose no threat to you</ansi>`
	case ratio > 3:
		return `<ansi fg="green">You hold a clear advantage</ansi>`
	case ratio > 2:
		return `<ansi fg="green">The odds favor you</ansi>`
	case ratio > 1:
		return `<ansi fg="yellow">An even contest — tread carefully</ansi>`
	case ratio > 0.5:
		return `<ansi fg="red-bold">They have the upper hand</ansi>`
	case ratio > 0:
		return `<ansi fg="red-bold">You are severely outmatched</ansi>`
	default:
		return `<ansi fg="red-bold">You will not survive this fight</ansi>`
	}
}
```

- [ ] **Step 2: Create `internal/actions/consider_test.go`**

```go
package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// fakeActor is a minimal Actor implementation for unit tests. It
// records SendText messages and tracks OnStatUse/OnSkillUse calls.
type fakeActor struct {
	char     *characters.Character
	name     string
	isPlayer bool
	sent     []string
	statUses map[string]int
}

// newFakeActor builds a fakeActor with the given ValueAdj for
// Strength/Dexterity and a HealthMax for durability. Setting
// ValueAdj directly skips the Training → Value → ValueAdj
// recalculation pipeline and matches how character_test.go
// constructs characters for combat math testing.
func newFakeActor(name string, statAdj, healthMax int, isPlayer bool) *fakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Strength.ValueAdj = statAdj
	c.Stats.Dexterity.ValueAdj = statAdj
	c.Stats.Perception.ValueAdj = statAdj
	c.Stats.Vitality.ValueAdj = statAdj
	c.Stats.Willpower.ValueAdj = statAdj
	c.Stats.Charisma.ValueAdj = statAdj
	c.HealthMax.Value = healthMax
	c.StaminaMax.Value = healthMax
	c.ConvictionMax.Value = healthMax / 2
	return &fakeActor{
		char:     c,
		name:     name,
		isPlayer: isPlayer,
		statUses: map[string]int{},
	}
}

func (a *fakeActor) GetCharacter() *characters.Character        { return a.char }
func (a *fakeActor) GetRoom() *rooms.Room                       { return nil }
func (a *fakeActor) SendText(msg string)                        { a.sent = append(a.sent, msg) }
func (a *fakeActor) SendRoomText(msg string, _ bool)            {}
func (a *fakeActor) SendRoomCommunication(msg string, _ bool)   {}
func (a *fakeActor) GetName() string                            { return a.name }
func (a *fakeActor) IsPlayer() bool                             { return a.isPlayer }
func (a *fakeActor) GetUserId() int                             { return 0 }
func (a *fakeActor) GetMobInstanceId() int                      { return 0 }
func (a *fakeActor) AddBuff(buffId int, source string)          {}
func (a *fakeActor) OnSkillUse(skillName string) bool {
	a.statUses[skillName]++
	return false
}
func (a *fakeActor) OnStatUse(statName string) bool {
	a.statUses[statName]++
	return false
}
func (a *fakeActor) OnCriticalSuccess(skillName string) {}
func (a *fakeActor) OnCriticalFailure(skillName string) {}

func TestConsider_OnStatUsePerception(t *testing.T) {
	self := newFakeActor("self", 100, 100, true)
	target := newFakeActor("target", 100, 100, true)

	Consider(self, target)
	if self.statUses["perception"] != 1 {
		t.Errorf("expected perception OnStatUse called once, got %d",
			self.statUses["perception"])
	}
}

func TestConsider_TextEmittedForPlayer(t *testing.T) {
	self := newFakeActor("hero", 100, 100, true)
	target := newFakeActor("orc", 100, 100, false)

	Consider(self, target)
	if len(self.sent) != 2 {
		t.Fatalf("expected 2 text lines emitted, got %d", len(self.sent))
	}
	if !strings.Contains(self.sent[0], "orc") {
		t.Errorf("first line should name the target, got %q", self.sent[0])
	}
	if !strings.Contains(self.sent[1], "instincts tell you") {
		t.Errorf("second line should be the prediction, got %q", self.sent[1])
	}
}

func TestConsider_ZeroTargetPower(t *testing.T) {
	self := newFakeActor("hero", 100, 100, true)
	// Construct a target with truly zero PowerScore: all ValueAdj=0,
	// no health, no skills, no mutations.
	target := &fakeActor{
		char:     &characters.Character{Name: "Ghost", Buffs: buffs.New()},
		name:     "Ghost",
		isPlayer: false,
		statUses: map[string]int{},
	}

	r := Consider(self, target)
	if r.TargetPower != 0 {
		// If this fails, default-construct of Character introduces
		// non-zero terms (e.g., default weapon offense). Document
		// the actual TargetPower and adjust the test expectation.
		t.Logf("TargetPower=%f (expected 0); see PowerScore default-weapon path", r.TargetPower)
	}
	if r.TargetPower == 0 && r.Ratio != 0 {
		t.Errorf("expected Ratio=0 when TargetPower=0, got %f", r.Ratio)
	}
}

func TestConsider_PredictionRatioBands(t *testing.T) {
	cases := []struct {
		ratio  float64
		expect string
	}{
		{5.0, "pose no threat"},
		{3.5, "clear advantage"},
		{2.5, "odds favor you"},
		{1.5, "even contest"},
		{0.75, "upper hand"},
		{0.25, "severely outmatched"},
		{0.0, "will not survive"},
	}
	for _, c := range cases {
		got := predictionFor(c.ratio)
		if !strings.Contains(got, c.expect) {
			t.Errorf("ratio=%v: expected %q in %q", c.ratio, c.expect, got)
		}
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `go test ./internal/actions/ -run TestConsider -v`
Expected: All four tests PASS. If the `Actor` interface in this package has different methods than `fakeActor` implements, fix the test by adding the missing methods — they're all listed in `internal/actions/actor.go`.

- [ ] **Step 4: Build the full module to catch downstream breaks**

Run: `go build ./...`
Expected: Clean build. Nothing consumes `actions.Consider` yet so no other packages should react.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/consider.go internal/actions/consider_test.go
git commit -m "feat(actions): add Consider(actor, target) shared compute"
```

---

## Task 2: Player wrapper rewrite

**Files:**
- Modify: `internal/usercommands/consider.go`

The player wrapper collapses to ~15 lines. Existing UX preserved verbatim because the text formatting moved into `actions.Consider`.

- [ ] **Step 1: Replace the body of `internal/usercommands/consider.go`**

Current file is 86 lines with the math inline. New body:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Consider(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(rest)
	if len(args) == 0 {
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, args[0],
		actions.ResolveTargetOptions{ExcludeUserId: user.UserId})
	if err != nil {
		// Pre-migration silently no-oped on no-match. On
		// ErrTargetVanished (stale mob ID) the original code DID
		// message "You don't see them here." — preserve that.
		if err == actions.ErrTargetVanished {
			user.SendText("You don't see them here.")
		}
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	actions.Consider(actor, target)
	return true, nil
}
```

- [ ] **Step 2: Build and run user-command tests**

Run: `go build ./...`
Expected: Clean. (If `combat` or `util` imports go red, double-check the trimmed import list — only `actions`, `events`, `rooms`, `users`, `util` are needed.)

Run: `go test ./internal/usercommands/ -run Consider -v`
Expected: Existing usercommand tests pass. No new tests added in this task — `actions.Consider` tests in Task 1 cover the math; player UX preservation is smoke-validated at the end.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/consider.go
git commit -m "refactor(usercommands): collapse consider to actions.Consider wrapper"
```

---

## Task 3: Mob wrapper + registration + parity test

**Files:**
- Create: `internal/mobcommands/consider.go`
- Modify: `internal/mobcommands/mobcommands.go`
- Modify: `internal/mobcommands/mobcommands_test.go`

- [ ] **Step 1: Create `internal/mobcommands/consider.go`**

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Consider is the mob-side entry into actions.Consider. The mob
// has no list-fall-through, so empty rest is a silent no-op.
// MobActor.SendText is a no-op, so the math runs but no text
// is emitted.
func Consider(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if rest == "" {
		return true, nil
	}
	target, err := actions.ResolveTargetActor(room, rest,
		actions.ResolveTargetOptions{ExcludeMobInstanceId: mob.InstanceId})
	if err != nil {
		return true, nil
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Consider(actor, target)
	return true, nil
}
```

- [ ] **Step 2: Register `consider` in `internal/mobcommands/mobcommands.go`**

Open the file and locate the `mobCommands` map (around line 25). Add a new entry, keeping the map in alphabetical order:

```go
		"consume":        {Consume, false},
		"consider":       {Consider, false},
		"converse":       {Converse, false},
```

(Place `consider` between `consume` and `converse` to match the existing alphabetical pattern.)

- [ ] **Step 3: Add `consider` to the mobcommand parity expected list**

Open `internal/mobcommands/mobcommands_test.go` and locate the test that enumerates expected commands. Add `"consider"` to the expected list in the matching alphabetical slot. (If the test reads the map via `GetAllMobCommands()` and compares against a literal slice, the new command must appear in that slice.)

If the test uses a different style (e.g., sorted set comparison), follow the existing convention.

- [ ] **Step 4: Run the parity test**

Run: `go test ./internal/mobcommands/ -run Parity -v`
Expected: PASS. If FAIL: the test's expected list is out of sync — update it to include `consider`.

- [ ] **Step 5: Build the module**

Run: `go build ./...`
Expected: Clean.

- [ ] **Step 6: Commit**

```bash
git add internal/mobcommands/consider.go internal/mobcommands/mobcommands.go internal/mobcommands/mobcommands_test.go
git commit -m "feat(mobcommands): add consider command wrapping actions.Consider"
```

---

## Task 4: Btree condition `target_power_ratio_above` / `_below`

**Files:**
- Create: `internal/behaviortree/conditions_combat.go`
- Create: `internal/behaviortree/conditions_combat_test.go`
- Modify: `internal/behaviortree/conditions.go`

Btree condition that compares self power to target power. Target resolution: `Event.UserId` first, then `mob.Character.Aggro.UserId`, then `mob.Character.Aggro.MobInstanceId`.

- [ ] **Step 1: Create `internal/behaviortree/conditions_combat.go`**

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// condTargetPowerRatioAbove returns Success when
// self_power / target_power > value.
func condTargetPowerRatioAbove(params map[string]any, ctx *EvalContext) Result {
	return targetPowerRatioCompare(params, ctx, true)
}

// condTargetPowerRatioBelow returns Success when
// self_power / target_power < value.
func condTargetPowerRatioBelow(params map[string]any, ctx *EvalContext) Result {
	return targetPowerRatioCompare(params, ctx, false)
}

// targetPowerRatioCompare implements the shared comparison body
// for the two power-ratio conditions. above=true means "ratio
// strictly greater than value"; above=false means "ratio strictly
// less than value". Missing/zero value → Failure (caller config
// error).
//
// Degenerate target power (<= 0) is treated as "infinitely
// weaker": above-comparison Succeeds, below-comparison Fails.
func targetPowerRatioCompare(params map[string]any, ctx *EvalContext, above bool) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	threshold := getFloatParam(params, "value", 0)
	if threshold == 0 {
		return Failure
	}

	targetPower, ok := resolveTargetPower(mob, ctx)
	if !ok {
		return Failure
	}

	selfPower := combat.PowerScore(mob.Character)
	if targetPower <= 0 {
		if above {
			return Success
		}
		return Failure
	}
	ratio := selfPower / targetPower
	if above && ratio > threshold {
		return Success
	}
	if !above && ratio < threshold {
		return Success
	}
	return Failure
}

// resolveTargetPower returns the PowerScore of the contextual
// target, with fallback chain:
//   1. ctx.Event.UserId → player
//   2. mob.Character.Aggro.UserId → player
//   3. mob.Character.Aggro.MobInstanceId → mob
//
// Returns (0, false) when no target resolvable.
func resolveTargetPower(mob *mobs.Mob, ctx *EvalContext) (float64, bool) {
	if ctx.Event.UserId > 0 {
		if u := users.GetByUserId(ctx.Event.UserId); u != nil {
			return combat.PowerScore(*u.Character), true
		}
	}
	if mob.Character.Aggro != nil {
		if mob.Character.Aggro.UserId > 0 {
			if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
				return combat.PowerScore(*u.Character), true
			}
		}
		if mob.Character.Aggro.MobInstanceId > 0 {
			if m := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); m != nil {
				return combat.PowerScore(m.Character), true
			}
		}
	}
	return 0, false
}
```

- [ ] **Step 2: Register both conditions in `internal/behaviortree/conditions.go`**

Locate the `init()` function (around line 9). Add two lines alongside the existing registrations (alphabetical-ish; place near the other `target_*` entries):

```go
	conditionRegistry["target_power_ratio_above"] = condTargetPowerRatioAbove
	conditionRegistry["target_power_ratio_below"] = condTargetPowerRatioBelow
```

- [ ] **Step 3: Create `internal/behaviortree/conditions_combat_test.go`**

The behaviortree package already provides three test helpers in `test_helpers_test.go`: `seedTestMob(t, templateId, instanceId, homeRoomId, name)`, `seedTestUser(t, userId, username, charName, roomId)`, and `seedTestRoom(t, roomId, zone)`. Each returns a cleanup function — `defer` it.

Reference pattern (see `conditions_test.go:361-385` for `TestCondMobInRoom_Hit`): seed room → seed mob → mutate fields on the registered instance to set stats → build `EvalContext{InstanceId: ..., RoomId: ..., Event: EventContext{UserId: ...}}` → call the condition function directly.

Concrete test file:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestTargetPowerRatioAbove_MissingValue(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := condTargetPowerRatioAbove(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure on missing value param, got %v", r)
	}
}

func TestTargetPowerRatioAbove_NoTargetResolvable(t *testing.T) {
	cleanMob := seedTestMob(t, 5, 105, 1, "TestMob")
	defer cleanMob()

	// No Event.UserId, no Aggro — nothing to compare against.
	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Failure {
		t.Errorf("expected Failure with no resolvable target, got %v", r)
	}
}

func TestTargetPowerRatioAbove_StrongMobVsWeakPlayer(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "StrongMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "weakling", "Weakling", 1)
	defer cleanUser()

	// Pump mob stats + durability to dominate the user's NewTestUser
	// defaults (ValueAdj=100, HealthMax=100).
	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 5000
	mob.Character.StaminaMax.Value = 5000
	mob.Character.ConvictionMax.Value = 2500
	mob.Character.Stats.Strength.ValueAdj = 500
	mob.Character.Stats.Dexterity.ValueAdj = 500
	mob.Character.Stats.Willpower.ValueAdj = 500
	mob.Character.Stats.Charisma.ValueAdj = 500

	ctx := &EvalContext{
		InstanceId: 105, RoomId: 1,
		Event: EventContext{UserId: 42},
	}
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Success {
		t.Errorf("expected Success (strong mob > weak player), got %v", r)
	}
	if r := condTargetPowerRatioBelow(map[string]any{"value": 1.0}, ctx); r != Failure {
		t.Errorf("expected Failure for below-1.0 when self stronger, got %v", r)
	}
}

func TestTargetPowerRatioBelow_WeakMobVsStrongPlayer(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "WeakMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "champion", "Champion", 1)
	defer cleanUser()

	// User already has ValueAdj=100 + HealthMax=100 from NewTestUser.
	// Mob is fresh (no stats, no health). User dominates.
	ctx := &EvalContext{
		InstanceId: 105, RoomId: 1,
		Event: EventContext{UserId: 42},
	}
	if r := condTargetPowerRatioBelow(map[string]any{"value": 1.0}, ctx); r != Success {
		t.Errorf("expected Success (weak mob < strong player), got %v", r)
	}
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Failure {
		t.Errorf("expected Failure for above-1.0 when self weaker, got %v", r)
	}
}

func TestTargetPowerRatio_AggroFallback(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "StrongMob")
	defer cleanMob()
	cleanUser := seedTestUser(t, 42, "weakling", "Weakling", 1)
	defer cleanUser()

	mob := mobs.GetInstance(105)
	mob.Character.HealthMax.Value = 5000
	mob.Character.Stats.Strength.ValueAdj = 500
	// Aggro on user 42 — no Event.UserId set; condition must fall back.
	mob.Character.Aggro = &characters.Aggro{UserId: 42}

	ctx := &EvalContext{InstanceId: 105, RoomId: 1} // no Event.UserId
	if r := condTargetPowerRatioAbove(map[string]any{"value": 1.0}, ctx); r != Success {
		t.Errorf("expected Success via Aggro fallback, got %v", r)
	}
}
```

(The PowerScore math is sensitive — large multipliers above keep the strong/weak direction unambiguous regardless of stat curve tuning. If a test result comes back unexpectedly opposite, add `t.Logf("self=%f target=%f", combat.PowerScore(mob.Character), combat.PowerScore(*u.Character))` to confirm the direction and bump the seeded values accordingly.)

- [ ] **Step 4: Run the test file**

Run: `go test ./internal/behaviortree/ -run TargetPowerRatio -v`
Expected: At minimum, the two `_NoTarget` / `_ZeroValue` cases PASS. The richer cases require the real test helper — flesh them out before completing the task.

- [ ] **Step 5: Build the module**

Run: `go build ./...`
Expected: Clean.

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/conditions_combat.go internal/behaviortree/conditions_combat_test.go internal/behaviortree/conditions.go
git commit -m "feat(behaviortree): add target_power_ratio_above/below conditions"
```

---

## Task 5: Btree action `target_weakest_mob_in_room`

**Files:**
- Modify: `internal/behaviortree/actions_combat.go`
- Modify: `internal/behaviortree/actions.go`
- Modify: `internal/behaviortree/actions_combat_test.go`

- [ ] **Step 1: Add `actTargetWeakestMobInRoom` to `internal/behaviortree/actions_combat.go`**

Append the function to the existing file. Required imports may need to be added at the top: `combat`, `mobs`, `rooms`, and `characters` (the last is already present in this file).

```go
// actTargetWeakestMobInRoom scans room.GetMobs(), computes
// PowerScore(target) / PowerScore(self) for each candidate that
// passes mob.HatesMob, picks the lowest ratio strictly below the
// ratio_below ceiling (default 1.0), and sets it as Aggro.
// Returns Success on a successful target pick, Failure otherwise.
//
// Skips: self, dead mobs, non-combatant mobs, mobs the caller's
// HatesMob returns false for, and (if caller is itself charmed)
// fellow companions of the same owner.
//
// Players are NOT scanned — predation is a mob-vs-mob action.
// Player aggression continues through the standard hostile-mob
// attack chain.
func actTargetWeakestMobInRoom(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.IsNonCombatant() {
		return Failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}

	selfPower := combat.PowerScore(mob.Character)
	if selfPower <= 0 {
		return Failure
	}
	// ratio_below defaults to 1.0 (engage anyone strictly weaker).
	ceiling := getFloatParam(params, "ratio_below", 1.0)

	callerCharmedBy := mob.Character.GetCharmedUserId()
	var bestId int
	bestRatio := ceiling
	for _, otherId := range room.GetMobs() {
		if otherId == mob.InstanceId {
			continue
		}
		other := mobs.GetInstance(otherId)
		if other == nil || other.IsNonCombatant() {
			continue
		}
		if other.Character.Health <= 0 {
			continue
		}
		// Companion-allegiance skip: if caller is itself charmed,
		// skip fellow companions of the same owner. A wild caller
		// can still prey on a player's companion if HatesMob says
		// so — companions of an enemy are still enemies.
		if callerCharmedBy > 0 && other.Character.IsCharmed(callerCharmedBy) {
			continue
		}
		if !mob.HatesMob(other) {
			continue
		}
		targetPower := combat.PowerScore(other.Character)
		if targetPower <= 0 {
			continue
		}
		ratio := targetPower / selfPower
		if ratio < bestRatio {
			bestRatio = ratio
			bestId = otherId
		}
	}
	if bestId == 0 {
		return Failure
	}
	mob.Character.SetAggro(0, bestId, characters.DefaultAttack)
	return Success
}
```

Verify the import block at the top of `actions_combat.go` includes:
- `"github.com/GoMudEngine/GoMud/internal/combat"`
- `"github.com/GoMudEngine/GoMud/internal/mobs"`
- `"github.com/GoMudEngine/GoMud/internal/rooms"`
- `"github.com/GoMudEngine/GoMud/internal/characters"`

(`characters` is likely already present for `actAttack`'s `SurpriseAttack`/`DefaultAttack` references.)

- [ ] **Step 2: Register the action in `internal/behaviortree/actions.go`**

Locate the `init()` function (around line 26) and add the registration alongside existing combat-action entries:

```go
	actionRegistry["target_weakest_mob_in_room"] = actTargetWeakestMobInRoom
```

- [ ] **Step 3: Add test cases to `internal/behaviortree/actions_combat_test.go`**

**Important seeding nuance:** `seedTestMob` from `test_helpers_test.go` calls `mobs.SeedMobsForTest(...)`, which REPLACES the entire mob registry each call. Calling `seedTestMob` twice in one test wipes out the first mob. For multi-mob tests, seed directly via `mobs.SeedMobsForTest` with all needed entries in one call. The helper below wraps that pattern. Add it at the top of the new test functions in `actions_combat_test.go`:

```go
// seedTwoMobs registers two mob templates + two instances in one
// SeedMobsForTest call, avoiding the single-mob limitation of
// seedTestMob. Returns a single cleanup function.
func seedTwoMobs(t *testing.T, roomId int,
	template1, instance1 int, name1 string,
	template2, instance2 int, name2 string,
) func() {
	t.Helper()
	specs := map[int]*mobs.Mob{
		template1: {MobId: mobs.MobId(template1), Character: characters.Character{
			Name: name1, RoomId: roomId, Buffs: buffs.New(),
		}},
		template2: {MobId: mobs.MobId(template2), Character: characters.Character{
			Name: name2, RoomId: roomId, Buffs: buffs.New(),
		}},
	}
	instances := map[int]*mobs.Mob{
		instance1: {MobId: mobs.MobId(template1), InstanceId: instance1, HomeRoomId: roomId,
			Character: characters.Character{Name: name1, RoomId: roomId, Buffs: buffs.New()}},
		instance2: {MobId: mobs.MobId(template2), InstanceId: instance2, HomeRoomId: roomId,
			Character: characters.Character{Name: name2, RoomId: roomId, Buffs: buffs.New()}},
	}
	return mobs.SeedMobsForTest(specs, instances)
}
```

The test imports for this file likely need: `mobs`, `rooms`, `characters`, `buffs` — check and add as needed.

Now append the test functions:

```go
func TestTargetWeakestMobInRoom_EmptyRoom(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Wolf")
	defer cleanMob()
	// Room exists but the wolf is the only mob in it.

	wolf := mobs.GetInstance(105)
	wolf.Character.HealthMax.Value = 1000
	wolf.Character.Stats.Strength.ValueAdj = 200

	room := rooms.LoadRoom(1)
	room.AddMob(105)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure with no other mobs, got %v", r)
	}
}

func TestTargetWeakestMobInRoom_HatedWeakerMob_Success(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 10, 110, "Rat")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	rat := mobs.GetInstance(110)

	// Pump wolf, leave rat at zero defaults so PowerScore(rat) < PowerScore(wolf).
	wolf.Character.HealthMax.Value = 1000
	wolf.Character.Stats.Strength.ValueAdj = 200
	// Wire the hates list + groups so wolf.HatesMob(rat) returns true.
	wolf.Hates = []string{"rodent"}
	rat.Groups = []string{"rodent"}

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(110)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Success {
		t.Errorf("expected Success picking the rat, got %v", r)
	}
	if wolf.Character.Aggro == nil || wolf.Character.Aggro.MobInstanceId != 110 {
		t.Errorf("expected Aggro set to rat (110), got %+v", wolf.Character.Aggro)
	}
}

func TestTargetWeakestMobInRoom_HatedButStronger_Failure(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 12, 112, "Bear")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	bear := mobs.GetInstance(112)

	// Bear stronger than wolf — wolf hates bears but won't engage.
	bear.Character.HealthMax.Value = 5000
	bear.Character.Stats.Strength.ValueAdj = 500
	wolf.Hates = []string{"ursine"}
	bear.Groups = []string{"ursine"}

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(112)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure (target is stronger), got %v", r)
	}
	if wolf.Character.Aggro != nil {
		t.Errorf("expected no Aggro set, got %+v", wolf.Character.Aggro)
	}
}

func TestTargetWeakestMobInRoom_NotHated_Failure(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	// Same template (5) for both — HatesMob returns false on same MobId.
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 5, 106, "OtherWolf")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	wolf.Character.HealthMax.Value = 1000

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(106)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure (same template, HatesMob false), got %v", r)
	}
}

func TestTargetWeakestMobInRoom_DeadMobSkipped(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 10, 110, "DeadRat")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	rat := mobs.GetInstance(110)

	wolf.Character.HealthMax.Value = 1000
	wolf.Hates = []string{"rodent"}
	rat.Groups = []string{"rodent"}
	rat.Character.Health = 0 // already dead — skip

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(110)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	if r := actTargetWeakestMobInRoom(map[string]any{}, ctx); r != Failure {
		t.Errorf("expected Failure (only candidate is dead), got %v", r)
	}
}

func TestTargetWeakestMobInRoom_RatioBelowCap(t *testing.T) {
	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMobs := seedTwoMobs(t, 1, 5, 105, "Wolf", 10, 110, "Rat")
	defer cleanMobs()

	wolf := mobs.GetInstance(105)
	rat := mobs.GetInstance(110)

	// Make the wolf only slightly stronger than the rat — ratio ~0.9.
	// A ratio_below: 0.5 ceiling should reject the engagement.
	wolf.Character.HealthMax.Value = 1100
	rat.Character.HealthMax.Value = 1000
	wolf.Hates = []string{"rodent"}
	rat.Groups = []string{"rodent"}

	room := rooms.LoadRoom(1)
	room.AddMob(105)
	room.AddMob(110)

	ctx := &EvalContext{InstanceId: 105, RoomId: 1}
	r := actTargetWeakestMobInRoom(map[string]any{"ratio_below": 0.5}, ctx)
	if r != Failure {
		t.Errorf("expected Failure (target ratio above 0.5 ceiling), got %v", r)
	}
}
```

- [ ] **Step 4: Run the test file**

Run: `go test ./internal/behaviortree/ -run TargetWeakest -v`
Expected: All non-skipped cases PASS.

- [ ] **Step 5: Build the module**

Run: `go build ./...`
Expected: Clean.

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/actions_combat.go internal/behaviortree/actions.go internal/behaviortree/actions_combat_test.go
git commit -m "feat(behaviortree): add target_weakest_mob_in_room action"
```

---

## Task 6: Document new btree primitives in context.md

**Files:**
- Modify: `internal/behaviortree/context.md`

- [ ] **Step 1: Append condition entry to the Condition Reference section**

Find the "Mob State" or "Environment" subsection of the Condition Reference table and add a new subsection (or extend an existing one). Insert below the existing `multiple_enemies` row:

```markdown
### Combat Assessment

| Condition | Params | Description |
|-----------|--------|-------------|
| `target_power_ratio_above` | `value` (float) | True when self_power / target_power > value. Target resolution: `Event.UserId` → `Aggro.UserId` → `Aggro.MobInstanceId`. Returns Failure if no target resolvable or value missing/zero. |
| `target_power_ratio_below` | `value` (float) | Mirror of `_above`: true when ratio < value. |
```

- [ ] **Step 2: Append action entry to the Action Reference section**

Find the Action Reference section (around line 209 per current file) and add a new subsection:

```markdown
### Combat Targeting — instant

| Action | Params | Description |
|--------|--------|-------------|
| `target_weakest_mob_in_room` | `ratio_below` (float, default 1.0) | Scans `room.GetMobs()`, picks the mob with the lowest power ratio relative to self that the caller's `HatesMob` returns true for, sets it as Aggro. Skips self, dead, non-combatant, same-owner companions, and mobs the caller doesn't hate. Returns Success on a pick, Failure otherwise. Players are NOT scanned. |
```

- [ ] **Step 3: Run any docs-related sanity check**

(No automated check exists for context.md; visually verify the new entries appear in the right sections.)

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/context.md
git commit -m "docs(behaviortree): document target_power_ratio + target_weakest_mob_in_room"
```

---

## Task 7: PowerScore audit — document gear contribution

**Files:**
- Modify: `internal/combat/context.md`

- [ ] **Step 1: Append "Power Scoring & Gear Contribution" section**

Append the following section to the end of `internal/combat/context.md` (or place it logically near other PowerScore content if a natural insertion point exists):

```markdown
## Power Scoring & Gear Contribution

`combat.PowerScore(char)` combines six terms: Offense, Defense,
Durability, Skills, Mutations, and KD ratio. Equipment
contribution flows through the standard pipes; there is no
separate "gear quality" axis.

| PowerScore term | Equipment field(s) that feed it |
|---|---|
| Offense (physAtk per-swing) | weapon `DamageMultiplier`, `SpeedMultiplier`; offhand + ExtraArm weapons |
| Offense (magAtk caster) | equipped weapon `SpellDamageMultiplier` |
| Offense (any stat-derived) | equipment `StatMods` → `Stats.X.ValueAdj` |
| Defense (mitigation) | equipment `PhysicalMitigation` / `MagicalMitigation` / `ConvictionMitigation` summed by `char.Get*Mitigation()` |
| Defense (avoidance) | equipment-driven dodge/parry/block via `char.GetDefenseScore(...)` |
| Durability | `char.HealthMax.Value` / `StaminaMax.Value` / `ConvictionMax.Value` — all reflect equipment stat boosts |
| Skills | not gear-driven |
| Mutations | not gear-driven |
| KD ratio | not gear-driven |

A player swapping a steel sword for an iron one will see
PowerScore drop because (a) the weapon's `DamageMultiplier`
changes (physAtk) and (b) any stat-mod difference flows through
`ValueAdj` into multiple terms. The Incorporeal mutation (chunk
2.2a) further scales gear contributions via
`mutations.GearEffectivenessMultiplier` — an ethereal wraith's
PowerScore reflects gear at the rank-determined fraction.

Consumers: `actions.Consider` (player + mob `consider`),
behavior tree conditions `target_power_ratio_above` and
`target_power_ratio_below`, behavior tree action
`target_weakest_mob_in_room`.
```

- [ ] **Step 2: Commit**

```bash
git add internal/combat/context.md
git commit -m "docs(combat): document gear contribution path in PowerScore"
```

---

## Task 8: Demo wiring — lookout archetype

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml`

- [ ] **Step 1: Insert the leading `player_enter` branch**

Open the file. Current `tree:` block has three children — `packmate_hurt`, `mob_hurt`, `heard_callforhelp`. Insert a new sequence ABOVE those, as the first child of the top-level `selector`:

```yaml
tree:
  type: selector
  children:
    # NEW: ambush only if I outmatch the entering player.
    # target_power_ratio_above: self_power / target_power > value.
    # value=1.0 means self is strictly stronger than target.
    - type: sequence
      event: player_enter
      children:
        - type: condition
          check: target_power_ratio_above
          value: 1.0
        - type: action
          do: attack

    # Existing branches below — leave untouched.
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: attack

    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: attack

    - type: action
      event: heard_callforhelp
      do: go_to_caller_room
```

- [ ] **Step 2: Boot the server briefly to verify YAML parse**

Run: `go run main.go` (run in background or with a short timeout — boot just past data-file load, then kill)
Expected: `behaviors.LoadDataFiles() loadedCount=...` increments without parse error. Watch for any `panic: yaml ...` lines.

Kill the server. (If using PowerShell: `Get-Process dogmud,go -ErrorAction SilentlyContinue | Stop-Process -Force`. If bash: `pkill -f 'go run' && pkill -f dogmud`.)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/lookout.yaml
git commit -m "feat(behaviors): lookout ambushes only when stronger than entering player"
```

---

## Task 9: Demo wiring — predator archetype + wolf YAMLs

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/205-steppe_wolf.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/206-young_wolf.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/223-scarred_wolf.yaml`

- [ ] **Step 1: Create `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml`**

```yaml
# predator archetype
#
# Opportunistic hunter. On idle ticks, scans the room for a
# weaker mob it would normally prey on (via mob.HatesMob —
# uses the YAML `hates:` list and same-species skip) and
# engages. Otherwise inherits the full generic_fighter combat
# behavior — packmate response, callforhelp navigation, and
# the per-round combat cascade (interrupt casts, kick prone/
# clinched, bash, grapple lone targets, trip).
#
# Faction/pack awareness comes from `hates:`; wolves with
# `hates: [boar, rodent]` will prey on boars and rats but
# never fellow canines. A mob with an empty `hates:` list
# effectively never opportunistically engages — the
# `target_weakest_mob_in_room` action will Failure-out.
#
# Example users: steppe wolves, alpha wolf, scarred wolf,
# young wolf. Distinguished from `generic_fighter` by the
# leading mob_idle predation branch.
#
# Spec: docs/superpowers/specs/2026-05-12-mob-aliveness-2.4-mob-consider-design.md

tree:
  type: selector
  children:
    # NEW: opportunistic predation on idle ticks. Selector
    # short-circuits on Success; if a weaker hated mob is in
    # the room, Aggro is set and the next tick's
    # mob_combat_round fires the cascade below.
    - type: action
      event: mob_idle
      do: target_weakest_mob_in_room
      ratio_below: 0.85

    # ── Below: verbatim copy of generic_fighter.yaml ─────

    # packmate_hurt: engage the attacker.
    - type: action
      event: packmate_hurt
      do: attack

    # heard_callforhelp: navigate toward an adjacent caller.
    - type: action
      event: heard_callforhelp
      do: go_to_caller_room

    # mob_combat_round: original combat cascade.
    - type: selector
      event: mob_combat_round
      children:
        - type: sequence
          children:
            - type: condition
              check: target_is_casting
            - type: action
              do: command_best_of
              cmds: [bash, trip]

        - type: sequence
          children:
            - type: condition
              check: target_not_standing
            - type: action
              do: command_best_of
              cmds: [kick]

        - type: action
          do: command_best_of
          cmds: [bash]

        - type: sequence
          children:
            - type: decorator
              mod: invert
              child:
                type: condition
                check: multiple_enemies
            - type: action
              do: command_best_of
              cmds: [grapple]

        - type: action
          do: command_best_of
          cmds: [trip]
```

- [ ] **Step 2: Flip four wolf YAMLs to use the predator archetype**

For each of:
- `_datafiles/world/dogmud/mobs/ironwind_steppe/205-steppe_wolf.yaml`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/206-young_wolf.yaml`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml`
- `_datafiles/world/dogmud/mobs/ironwind_steppe/223-scarred_wolf.yaml`

Locate the line `behavior_archetype: generic_fighter` (typically near the top, around line 3). Replace it with:

```yaml
behavior_archetype: predator
```

No other changes — `groups:`, `hates:`, `stats:`, etc. stay untouched. The existing `hates:` list (typically `[boar, rodent]` for wolves) drives predation targeting automatically.

- [ ] **Step 3: Boot the server briefly to verify YAML parse**

Run: `go run main.go` (background or short timeout)
Expected: clean boot past data-file load. `behaviors.LoadDataFiles() loadedCount=...` increments by 1 (the new predator archetype). `mobs.LoadDataFiles() loadedCount=...` unchanged. Watch for parse errors.

Kill the server.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/predator.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/205-steppe_wolf.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/206-young_wolf.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/215-alpha_wolf.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/223-scarred_wolf.yaml
git commit -m "feat(behaviors): add predator archetype, wire to ironwind wolves"
```

---

## Task 10: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Flip 2.4 to Done in the progress tracker**

Locate the progress-tracker table (around line 76). Find the row:

```markdown
| 2.4 | Tactical | Mob `appraise` / `assess` | S | 2.2 | Not started |
```

Change to:

```markdown
| 2.4 | Tactical | Mob `consider` + threat-aware behaviors | S | 2.2 | Done |
```

(Title changes to match the reframed scope: predation + lookout, not appraise/assess.)

- [ ] **Step 2: Update the roll-up line**

Below the table, locate:

```markdown
**Roll-up:** 11 / 41 done • 0 in progress • 30 not started.
```

Change to:

```markdown
**Roll-up:** 12 / 41 done • 0 in progress • 29 not started.
```

- [ ] **Step 3: Update the 2.4 mini-brief**

Locate the chunk 2.4 mini-brief (search for `### 2.4 Mob \`appraise\` / \`assess\``). Replace it with:

```markdown
### 2.4 Mob `consider` + threat-aware behaviors
**Status:** Done (2026-05-12) • **Size:** S

- **Goal:** Mobs size up combat threats the same way players do; reactive (lookout-only-ambushes-weaker-players) and opportunistic (predator-engages-weaker-prey) behaviors unlocked.
- **In:** Actor-pattern consolidation of `consider` into `actions.Consider(actor, target) ConsiderResult` shared by player + mob wrappers (`internal/usercommands/consider.go` thinned, `internal/mobcommands/consider.go` added). Two btree primitives: `target_power_ratio_above`/`_below` condition (in new `conditions_combat.go`) and `target_weakest_mob_in_room` action (added to `actions_combat.go`). Demo wiring: lookout archetype gains `player_enter`→ambush-if-stronger branch; new `predator` archetype copies generic_fighter and adds a leading `mob_idle` predation branch; four ironwind wolf YAMLs flip to predator.
- **Out:** Player gear-coveting (players don't drop gear so no use case); `appraise` mobcommand (player command is obsoleted by identify spell); `combat.PowerScore` math changes (audit confirmed gear is already reflected through `ValueAdj`/`Get*Mitigation` pipes; the audit deliverable is a documentation section in `internal/combat/context.md`).
- **Depends on:** 2.2 (item-comparison primitive contributed conceptually but PowerScore-based assessment uses existing combat infrastructure).
- **Why:** Reactive lookouts that don't suicide-ambush strong players. Opportunistic predators that go after weaker prey. Foundation for chunk 2.6 (tactics-cast preemption — power-ratio gating offensive vs. defensive cast selection) and 5.2 (bounty hunting — bounty hunters need to assess wanted targets).
- **Shipped:** `internal/actions/consider.go` — `Consider(actor, target) ConsiderResult` with prediction text emission via `actor.SendText` (MobActor no-op preserves silent compute path). Player + mob wrappers each ~15 lines. Btree primitives in `conditions_combat.go` and `actions_combat.go` (new function alongside existing entries). Target resolution chain: `Event.UserId` → `Aggro.UserId` → `Aggro.MobInstanceId`. `mob.HatesMob(other)` predicate gates predation — covers faction/pack-awareness without coupling to 1.2 substrate. Lookout `player_enter` branch with `target_power_ratio_above: 1.0` ambush gate. Predator archetype `ratio_below: 0.85` predation ceiling. PowerScore audit section added to `internal/combat/context.md`. Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.4-mob-consider-design.md`, plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.4-mob-consider.md`.
```

- [ ] **Step 4: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 2.4 (mob consider) as Done"
```

---

## Task 11: Smoke validation

**Files:**
- None modified — pure validation.

Per the SOP, do NOT just trust `go build`. Spin the server up locally and exercise the behaviors.

- [ ] **Step 1: Full build + test**

Run: `go build ./...`
Expected: Clean.

Run: `go test ./...`
Expected: No FAILs. If existing tests touch the renamed/moved consider math, fix any expectations that read the math directly (none should — text output preserved).

- [ ] **Step 2: Boot the server and watch the load lines**

Run: `go run main.go` (in a separate terminal, or in the background)
Expected: server reaches the post-load steady state. Look for:
- `behaviors.LoadDataFiles() loadedCount=...` (increments by 1 vs. pre-chunk baseline)
- `mobs.LoadDataFiles() loadedCount=...` (unchanged)
- No panics; no YAML parse errors.

- [ ] **Step 3: Spot-check the player consider parity**

Connect to the server as a test character. Find a target.

Run (in-game): `consider <some weak npc>`
Expected: same text format as before — `"You consider <ansi fg="mobname">...</ansi>..."` followed by `"Your instincts tell you: <prediction>"`. Confirm prediction text matches one of the six bands.

Run (in-game): `consider <some strong npc>`
Expected: a different prediction band (e.g., "outmatched" or "upper hand"). Visually confirm the colored output renders.

- [ ] **Step 4: Spot-check mob consider silence**

As an admin, find a mob's instance id and run:

In-game: `mob <mobid> consider <some target name>`
Expected: no panic, no text leakage to the room, no broadcast. (The command runs, `actions.Consider` computes silently because `MobActor.SendText` is a no-op.)

- [ ] **Step 5: Spot-check lookout ambush — weak player**

Locate or spawn the bandit lookout (mob 283). As a low-stat throwaway character (or admin-debuffed), walk into the lookout's room.

Expected: lookout attacks. Room broadcast fires through the existing `attack` action ("X attacks Y" or similar). Combat starts.

- [ ] **Step 6: Spot-check lookout stay-hidden — strong player**

As an admin-buffed character with elevated stats (or a high-level test character), walk into the lookout's room.

Expected: no aggression. Lookout remains hidden. No room broadcast.

- [ ] **Step 7: Spot-check wolf predation**

Spawn a steppe wolf (mob 205) and a rat (or other mob in the wolf's `hates:` list) in the same room. Ensure the rat is weaker than the wolf.

Wait for the wolf's next idle tick (typically a few seconds; check `Balance.IdleCheckInterval` config).

Expected: wolf sets Aggro on the rat. Verify via `mob <wolfid> show aggro` or by watching combat begin. The mob_idle predation branch fires, action picks the rat (lowest ratio, hated, weaker), sets Aggro. The next combat round fires `mob_combat_round`, the cascade picks an attack.

- [ ] **Step 8: Spot-check wolf same-species skip**

Spawn two steppe wolves in the same room with no other prey present.

Expected: neither sets Aggro on the other. Both remain idle. `mob.HatesMob(otherWolf)` returns false (same species), so the predation action fails and the wolves stay idle.

- [ ] **Step 9: Spot-check wolf same-owner companion skip**

Charm a wolf (e.g., via admin `mob <wolfid> charm <playerid>`). Spawn a charmed rat (companion of the same player) in the room.

Expected: wolf does NOT engage the rat. The `callerCharmedBy > 0 && other.IsCharmed(callerCharmedBy)` guard skips fellow companions.

- [ ] **Step 10: Kill the test server cleanly**

Stop the running server. (Bash: `pkill -f 'go run'`. PowerShell: `Get-Process dogmud,go -ErrorAction SilentlyContinue | Stop-Process -Force`.) Per the kill-test-mud-servers SOP, clean up any leftover `dogmud*.exe` / `go run` processes.

- [ ] **Step 11: Final commit (if anything was missed)**

If any of the smoke checks revealed a missing fix and an additional commit landed, that's fine. Otherwise no extra commit is needed — Task 10's commit closes the chunk.

```bash
git log --oneline -15   # verify the chunk's commit chain
```

Expected: a clean sequence of small commits ending in the roadmap update. Roughly:
- `feat(actions): add Consider(actor, target) shared compute`
- `refactor(usercommands): collapse consider to actions.Consider wrapper`
- `feat(mobcommands): add consider command wrapping actions.Consider`
- `feat(behaviortree): add target_power_ratio_above/below conditions`
- `feat(behaviortree): add target_weakest_mob_in_room action`
- `docs(behaviortree): document target_power_ratio + target_weakest_mob_in_room`
- `docs(combat): document gear contribution path in PowerScore`
- `feat(behaviors): lookout ambushes only when stronger than entering player`
- `feat(behaviors): add predator archetype, wire to ironwind wolves`
- `docs(roadmap): mark chunk 2.4 (mob consider) as Done`

---

## Out of scope (per spec)

- PowerScore math changes (audit-only documentation)
- Faction-substrate-based predation gating (1.2 `factions` package coupling)
- Per-mob predation cooldown
- Players as predation targets
- Scan radius beyond current room
- Memoizing PowerScore
- Full archetype audit / pass beyond lookout + predator
- Player-vs-player power-ratio gating
- `appraise` command deprecation
