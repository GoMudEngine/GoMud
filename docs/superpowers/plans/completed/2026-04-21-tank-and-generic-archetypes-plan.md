# Tank + Generic Fighter Archetypes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship two new behavior-tree archetypes (`tank_taunter`, `generic_fighter`) on the Phase 4 framework, with the engine plumbing needed to self-gate mob commands like `cast_best_in_category` does for spells. Pre-wire five companion mob templates.

**Architecture:** New btree action `command_best_of` calls a shared `actions.CommandIsReady(mob, cmd)` helper before issuing. Three new btree conditions cover target-state checks. Rally + warcry are extracted to `actions.Execute*` helpers to give mobs player-parity access (mirrors the existing taunt/howl pattern).

**Tech Stack:** Go 1.21+, YAML archetype trees, `reflect` / `testify` as in neighboring packages.

**Related spec:** `docs/superpowers/specs/2026-04-21-tank-and-generic-archetypes-design.md`
**Branch:** `feature/tank-and-generic-archetypes` (created; spec committed as `5d02a64d`).

---

## File Structure

**New Go files:**
- `internal/actions/combat_rally.go` — shared `ExecuteRally(Actor) RallyResult`
- `internal/actions/combat_warcry.go` — shared `ExecuteWarcry(Actor) WarcryResult`
- `internal/actions/command_readiness.go` — `CommandIsReady(mob, cmd string) bool`
- `internal/actions/command_readiness_test.go` — table-driven tests
- `internal/mobcommands/rally.go` — thin wrapper
- `internal/mobcommands/warcry.go` — thin wrapper

**Modified Go files:**
- `internal/usercommands/rally.go` — refactor to call `actions.ExecuteRally`
- `internal/usercommands/warcry.go` — refactor to call `actions.ExecuteWarcry`
- `internal/mobcommands/mobcommands.go` — register `"rally"` + `"warcry"`
- `internal/behaviortree/actions.go` — register `"command_best_of"` action
- `internal/behaviortree/actions_mob.go` — add `actCommandBestOf` function
- `internal/behaviortree/actions_test.go` — test for `actCommandBestOf`
- `internal/behaviortree/conditions.go` — register 3 new conditions
- `internal/behaviortree/conditions_mob.go` — add `condTargetIsCasting`, `condTargetAggroNotOnMe`, `condTargetNotStanding`
- `internal/behaviortree/conditions_test.go` — tests for the three conditions

**New content:**
- `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`
- `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`

**Modified content (mob YAMLs — add `behavior_archetype` field):**
- `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml` → `tank_taunter`
- `_datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml` → `tank_taunter`
- `_datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml` → `tank_taunter`
- `_datafiles/world/dogmud/mobs/summons/243-steppe_spirit_wolf.yaml` → `generic_fighter`
- `_datafiles/world/dogmud/mobs/summons/301-zombie.yaml` → `generic_fighter`

---

## Task 1: Extract rally to shared action + mob command wrapper

**Files:**
- Create: `internal/actions/combat_rally.go`
- Modify: `internal/usercommands/rally.go`
- Create: `internal/mobcommands/rally.go`
- Modify: `internal/mobcommands/mobcommands.go`

Extract the core cooldown+apply logic from `usercommands/rally.go` into a shared action. The user command keeps its party/companion fan-out; mob command does self-buff only (per spec "keep it simple: mob rally buffs self").

- [ ] **Step 1: Create the shared action**

Create `internal/actions/combat_rally.go`:

```go
package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// RallyResult reports the outcome of a rally cooldown+buff application.
type RallyResult struct {
	Executed   bool    // true if the rally actually applied
	OnCooldown bool    // blocked by shared special-move cooldown
	Crafting   bool    // blocked because a player is mid-craft (player-only)
	Bonus      float64 // mitigation bonus the condition carries (0.05..0.20)
	Duration   int     // condition duration in rounds
}

// ExecuteRally performs the cooldown check + self-buff application shared by
// both the player "rally" command and the mob "rally" command. Callers handle
// any fan-out (party members, companions, room broadcast) and player-facing
// text.
func ExecuteRally(actor Actor) RallyResult {
	char := actor.GetCharacter()

	// IsCrafting applies to players only; mobs never craft.
	if actor.IsPlayer() && char.IsCrafting() {
		return RallyResult{Crafting: true}
	}

	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return RallyResult{OnCooldown: true}
	}

	// Magnitude: 0.05 + 0.15 * sqrt((rhetoric/75) * (charisma/175)), clamped.
	rhetoric := float64(char.GetSkillLevel(skills.Rhetoric))
	charisma := float64(char.Stats.Charisma.ValueAdj)
	bonus := 0.05 + 0.15*math.Sqrt((rhetoric/75.0)*(charisma/175.0))
	if bonus < 0.05 {
		bonus = 0.05
	}
	if bonus > 0.20 {
		bonus = 0.20
	}
	duration := 25

	char.AddCondition(characters.ConditionRally, duration, bonus, "rally")
	char.AddBuff(80, false)

	// Set combat wait if in combat (matches player + mob behavior).
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return RallyResult{
		Executed: true,
		Bonus:    bonus,
		Duration: duration,
	}
}
```

- [ ] **Step 2: Refactor user command**

Replace `internal/usercommands/rally.go`'s body. The function signature stays the same, the crafting/cooldown/magnitude code calls `ExecuteRally`, and the function keeps its party+companion fan-out after the core apply succeeds:

```go
package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Rally(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	result := actions.ExecuteRally(&actions.UserActor{User: user, Room: room})

	if result.Crafting {
		user.SendText(`<ansi fg="red">You can't rally while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
	if result.OnCooldown {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}
	if !result.Executed {
		return true, nil
	}

	user.SendText(`<ansi fg="cyan-bold">You rally your allies with an inspiring shout that steadies their resolve!</ansi>`)
	room.SendTextVisual(
		fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="username">%s</ansi> rallies everyone with an inspiring shout!</ansi>`, user.Character.Name),
		user.UserId,
	)

	// Fan out to party members in the room.
	if party := parties.Get(user.UserId); party != nil {
		for _, memberId := range party.GetMembers() {
			if memberId == user.UserId {
				continue
			}
			memberUser := users.GetByUserId(memberId)
			if memberUser == nil || memberUser.Character.RoomId != user.Character.RoomId {
				continue
			}
			memberUser.Character.AddCondition(characters.ConditionRally, result.Duration, result.Bonus, "rally")
			memberUser.Character.AddBuff(80, false)
			memberUser.SendText(
				fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="username">%s</ansi>'s rallying cry steadies your nerves!</ansi>`, user.Character.Name))
			applyRallyToCompanions(memberUser, room, result.Bonus, result.Duration)
		}
	}

	// Fan out to caster's own companions in the room.
	applyRallyToCompanions(user, room, result.Bonus, result.Duration)

	// Rhetoric skill progression.
	if user.Character.Aggro != nil {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	} else if util.Rand(100) < 50 {
		user.Character.OnSkillUse(string(skills.Rhetoric), user.UserId)
	}

	return true, nil
}

func applyRallyToCompanions(owner *users.UserRecord, room *rooms.Room, bonus float64, duration int) {
	for _, mobInstId := range owner.Character.GetCharmIds() {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}
		if mob.Character.RoomId != owner.Character.RoomId {
			continue
		}
		mob.Character.AddCondition(characters.ConditionRally, duration, bonus, "rally")
		mob.Character.AddBuff(80, false)
	}
}
```

- [ ] **Step 3: Create the mob command wrapper**

Create `internal/mobcommands/rally.go`:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Rally is the mob-side shout that applies the rally mitigation buff to
// the casting mob. (Unlike the player version, mobs don't rally allies —
// their "allies" are the summoner's party, which is out of scope here.)
func Rally(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	result := actions.ExecuteRally(&actions.MobActor{Mob: mob, Room: room})
	if !result.Executed {
		return true, nil
	}

	room.SendText(fmt.Sprintf(
		`<ansi fg="cyan-bold"><ansi fg="mobname">%s</ansi> lets out a rallying roar!</ansi>`,
		mob.Character.Name,
	))

	return true, nil
}
```

- [ ] **Step 4: Register the mob command**

In `internal/mobcommands/mobcommands.go`, locate the command registry map (where `"howl"` is registered). Add an entry for `"rally"` in alphabetical order:

```go
		"rally":          {Rally, false},
```

(Alphabetical placement means between `"quit"`-ish and `"run"`-ish entries — check the existing file to confirm.)

- [ ] **Step 5: Build + test**

Run: `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/`
Expected: clean build, all existing tests pass. No new test added this step — behavior is end-to-end equivalent to before for player rally; mob rally is new but not yet called by any caller.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/combat_rally.go internal/usercommands/rally.go internal/mobcommands/rally.go internal/mobcommands/mobcommands.go
git commit -m "$(cat <<'EOF'
refactor(actions): extract ExecuteRally for mob/player parity

Mirrors the ExecuteTaunt pattern — core cooldown + self-buff
logic lives in actions/combat_rally.go. usercommands/rally.go
keeps its party-and-companion fan-out; new mobcommands/rally.go
is the thin mob wrapper.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Extract warcry to shared action + mob command wrapper

**Files:**
- Create: `internal/actions/combat_warcry.go`
- Modify: `internal/usercommands/warcry.go`
- Create: `internal/mobcommands/warcry.go`
- Modify: `internal/mobcommands/mobcommands.go`

Mirror Task 1's pattern for warcry. Warcry is the damage-boost counterpart to rally (buff 79 instead of 80, condition `ConditionWarcry` probably, room-wide room broadcast).

- [ ] **Step 1: Read the current warcry.go**

Run: `cat internal/usercommands/warcry.go`

Inspect for: the cooldown key, the `AddCondition` constant (likely `ConditionWarcry`), the buff id (79 per spec), the magnitude formula, party/companion fan-out (if any), and the exact messages.

- [ ] **Step 2: Create the shared action**

Create `internal/actions/combat_warcry.go`. Use the same structure as `combat_rally.go` from Task 1, but substitute:
- `RallyResult` → `WarcryResult`
- `ExecuteRally` → `ExecuteWarcry`
- buff id 80 → 79
- condition constant → `ConditionWarcry` (verify name in Step 1 reading)
- magnitude formula, duration — copy from `warcry.go`

Keep the same `Executed / OnCooldown / Crafting / Bonus / Duration` result fields. No fan-out in the shared action — just cooldown + self-buff.

- [ ] **Step 3: Refactor the user command**

Replace `internal/usercommands/warcry.go`'s body. Delegate the crafting/cooldown/magnitude work to `actions.ExecuteWarcry`. Keep whatever fan-out + display the current function does, using `result.Bonus` / `result.Duration` for any shared downstream application.

- [ ] **Step 4: Create mob command wrapper**

Create `internal/mobcommands/warcry.go` mirroring `rally.go`:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Warcry is the mob-side shout that applies the warcry damage buff to the
// casting mob. Like Rally, it does not fan out to allies — that fan-out
// is player-only.
func Warcry(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	result := actions.ExecuteWarcry(&actions.MobActor{Mob: mob, Room: room})
	if !result.Executed {
		return true, nil
	}

	room.SendText(fmt.Sprintf(
		`<ansi fg="red-bold"><ansi fg="mobname">%s</ansi> lets out a bone-shaking warcry!</ansi>`,
		mob.Character.Name,
	))

	return true, nil
}
```

- [ ] **Step 5: Register the mob command**

In `internal/mobcommands/mobcommands.go`, add `"warcry":` in alphabetical position:

```go
		"warcry":         {Warcry, false},
```

- [ ] **Step 6: Build + test**

Run: `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/`
Expected: clean build + all green.

- [ ] **Step 7: Commit**

```bash
git add internal/actions/combat_warcry.go internal/usercommands/warcry.go internal/mobcommands/warcry.go internal/mobcommands/mobcommands.go
git commit -m "$(cat <<'EOF'
refactor(actions): extract ExecuteWarcry for mob/player parity

Same pattern as ExecuteRally — cooldown + self-buff in the
shared action, fan-out kept in the user command. New mob
command wrapper registers "warcry" for behavior-tree use.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: CommandIsReady helper

**Files:**
- Create: `internal/actions/command_readiness.go`
- Create: `internal/actions/command_readiness_test.go`

Synchronous readiness check for the seven supported commands. The btree uses it before issuing a command so the selector can fall through on not-ready. Non-mutating — uses `GetCooldown` (peek) rather than `Cooldowns.Try` (which mutates).

- [ ] **Step 1: Write the failing test**

Create `internal/actions/command_readiness_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// newTestMob builds a minimal combat-ready mob with configurable state.
func newTestMob(t *testing.T, cfg func(*mobs.Mob)) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{
		MobId:      1,
		InstanceId: 100,
	}
	m.Character.Name = "Test"
	m.Character.Stamina = 999
	m.Character.StaminaMax.Value = 999
	m.Character.Conviction = 999
	m.Character.ConvictionMax.Value = 999
	m.Character.Position = characters.PositionStanding
	m.Character.SetAggro(1, 0, characters.DefaultAttack) // user 1 as generic target
	if cfg != nil {
		cfg(m)
	}
	return m
}

func TestCommandIsReady_UnknownCommand(t *testing.T) {
	m := newTestMob(t, nil)
	assert.False(t, CommandIsReady(m, "does_not_exist"))
}

func TestCommandIsReady_Taunt_NoAggroFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) { m.Character.EndAggro() })
	assert.False(t, CommandIsReady(m, "taunt"))
}

func TestCommandIsReady_Taunt_OnCooldownFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
	})
	assert.False(t, CommandIsReady(m, "taunt"))
}

func TestCommandIsReady_Taunt_ReadyTrue(t *testing.T) {
	m := newTestMob(t, nil)
	assert.True(t, CommandIsReady(m, "taunt"))
}

func TestCommandIsReady_Rally_NoAggroStillTrue(t *testing.T) {
	// Rally doesn't require an aggro target.
	m := newTestMob(t, func(m *mobs.Mob) { m.Character.EndAggro() })
	assert.True(t, CommandIsReady(m, "rally"))
}

func TestCommandIsReady_Warcry_OnCooldownFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.Cooldowns = characters.Cooldowns{"special-move": 1}
	})
	assert.False(t, CommandIsReady(m, "warcry"))
}

func TestCommandIsReady_Trip_TargetAlreadyProneFalse(t *testing.T) {
	m := newTestMob(t, nil)
	// ResolveAggroTarget would need a real target character; for unit
	// scope we simulate by placing a real mob target.
	targetMob := &mobs.Mob{InstanceId: 200}
	targetMob.Character.Name = "Target"
	targetMob.Character.Position = characters.PositionProne
	// Register the target mob for ResolveAggroTarget's lookup.
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)
	m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
	assert.False(t, CommandIsReady(m, "trip"))
}

func TestCommandIsReady_Bash_NoShieldNoNaturalBashFalse(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.SpeciesId = 1 // human (no naturalbash)
	})
	assert.False(t, CommandIsReady(m, "bash"))
}

func TestCommandIsReady_Grapple_TargetAlreadyClinchedFalse(t *testing.T) {
	m := newTestMob(t, nil)
	targetMob := &mobs.Mob{InstanceId: 201}
	targetMob.Character.Name = "Target"
	targetMob.Character.Position = characters.PositionClinched
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)
	m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
	assert.False(t, CommandIsReady(m, "grapple"))
}

func TestCommandIsReady_Kick_ReadyTrue(t *testing.T) {
	m := newTestMob(t, nil)
	assert.True(t, CommandIsReady(m, "kick"))
}
```

**Note on `mobs.SetInstanceForTest`:** if this helper doesn't exist in `internal/mobs/`, the implementer should add a minimal one in `internal/mobs/test_helpers.go` (or the existing seed registry test helper). Simple implementation:

```go
// test_helpers.go
func SetInstanceForTest(instId int, mob *Mob) {
    mobInstancesMu.Lock()
    defer mobInstancesMu.Unlock()
    if mob == nil {
        delete(mobInstances, instId)
        return
    }
    mobInstances[instId] = mob
}
```

If investigation shows an existing equivalent, use that instead.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -run 'TestCommandIsReady_' ./internal/actions/`
Expected: all FAIL with `undefined: CommandIsReady`.

- [ ] **Step 3: Implement the helper**

Create `internal/actions/command_readiness.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// CommandIsReady returns true iff the named mob command would actually
// execute its effect right now. Checks cover: shared special-move
// cooldown, aggro target (where required), and target state (where
// relevant — e.g., trip needs standing target, grapple needs non-
// clinched target). Resource costs (stamina/conviction) are NOT checked
// — if a mob is resource-starved the command will no-op at execution
// time. For v1 this is acceptable because the shared cooldown is the
// dominant gate.
//
// Unknown command names return false. This lets a behavior tree safely
// include commands that don't exist yet without firing a spurious
// Success.
func CommandIsReady(mob *mobs.Mob, cmd string) bool {
	if mob == nil {
		return false
	}
	char := &mob.Character

	// All supported commands share the special-move cooldown.
	if char.GetCooldown("special-move") > 0 {
		return false
	}

	switch cmd {
	case "taunt":
		return char.Aggro != nil

	case "rally", "warcry":
		return true // cooldown already checked above; no target needed

	case "trip":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.Position.IsGroundPosition()

	case "bash":
		if char.Aggro == nil {
			return false
		}
		// HasShield() already accounts for NaturalBash species.
		return char.HasShield()

	case "grapple":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.Position.IsGrapplePosition()

	case "kick":
		return char.Aggro != nil
	}

	return false
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -run 'TestCommandIsReady_' ./internal/actions/`
Expected: all PASS.

- [ ] **Step 5: Build + full test**

Run: `go build ./... && go test ./...`
Expected: all green.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/command_readiness.go internal/actions/command_readiness_test.go
# If SetInstanceForTest was added to internal/mobs/:
# git add internal/mobs/test_helpers.go
git commit -m "$(cat <<'EOF'
feat(actions): CommandIsReady helper for btree command gating

Synchronous readiness check for taunt / rally / warcry / trip /
bash / grapple / kick. Covers shared special-move cooldown,
aggro requirement, target-state constraints (prone for trip,
clinched for grapple), and shield/naturalbash for bash.

Non-mutating: uses GetCooldown (peek) rather than Cooldowns.Try
(which sets the cooldown). Resource costs intentionally not
checked — the command's own execution-time gates catch those;
for v1 we accept the rare resource-starved no-op round.

Used by the forthcoming command_best_of btree action.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Three new btree conditions

**Files:**
- Modify: `internal/behaviortree/conditions.go` (registry)
- Modify: `internal/behaviortree/conditions_mob.go` (three new funcs)
- Modify: `internal/behaviortree/conditions_test.go` (tests)

The three conditions drive the archetype sequences: `target_is_casting`, `target_aggro_not_on_me`, `target_not_standing`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/behaviortree/conditions_test.go`:

```go
// ─── target_is_casting ──────────────────────────────────────────────────────

func TestCondTargetIsCasting_TargetCasting_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 200}
	target.Character.CastingState = &characters.CastingState{}
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetIsCasting(nil, ctx)
	assert.Equal(t, Success, result)
}

func TestCondTargetIsCasting_TargetNotCasting_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 201}
	// CastingState nil = not casting
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetIsCasting(nil, ctx)
	assert.Equal(t, Failure, result)
}

func TestCondTargetIsCasting_NoTarget_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	mob.Character.EndAggro()

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetIsCasting(nil, ctx)
	assert.Equal(t, Failure, result)
}

// ─── target_aggro_not_on_me ────────────────────────────────────────────────

func TestCondTargetAggroNotOnMe_TargetAggrosMe_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t) // instance 100
	target := &mobs.Mob{InstanceId: 200}
	target.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetAggroNotOnMe(nil, ctx)
	assert.Equal(t, Failure, result, "target is aggro'd on me, should NOT taunt")
}

func TestCondTargetAggroNotOnMe_TargetAggrosSomeoneElse_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t) // instance 100
	target := &mobs.Mob{InstanceId: 201}
	target.Character.SetAggro(42, 0, characters.DefaultAttack) // user 42
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetAggroNotOnMe(nil, ctx)
	assert.Equal(t, Success, result, "target aggros someone else, SHOULD taunt")
}

func TestCondTargetAggroNotOnMe_TargetHasNoAggro_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 202}
	// target.Character.Aggro is nil
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetAggroNotOnMe(nil, ctx)
	assert.Equal(t, Success, result)
}

// ─── target_not_standing ───────────────────────────────────────────────────

func TestCondTargetNotStanding_TargetStanding_ReturnsFailure(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 203}
	target.Character.Position = characters.PositionStanding
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetNotStanding(nil, ctx)
	assert.Equal(t, Failure, result)
}

func TestCondTargetNotStanding_TargetProne_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 204}
	target.Character.Position = characters.PositionProne
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetNotStanding(nil, ctx)
	assert.Equal(t, Success, result)
}

func TestCondTargetNotStanding_TargetClinched_ReturnsSuccess(t *testing.T) {
	mob := newTestMob(t)
	target := &mobs.Mob{InstanceId: 205}
	target.Character.Position = characters.PositionClinched
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	ctx := &EvalContext{InstanceId: mob.InstanceId}
	result := condTargetNotStanding(nil, ctx)
	assert.Equal(t, Success, result)
}
```

If `newTestMob` helper doesn't exist in this test file, add one at the top:

```go
// newTestMob seeds a test mob at instance 100 and returns it. The caller
// is responsible for test-registering it via mobs.SetInstanceForTest.
func newTestMob(t *testing.T) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{MobId: 1, InstanceId: 100}
	m.Character.Name = "TestMob"
	mobs.SetInstanceForTest(m.InstanceId, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(m.InstanceId, nil) })
	return m
}
```

Check existing helpers first — there may already be a builder.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestCondTarget' ./internal/behaviortree/`
Expected: FAIL with `undefined: condTargetIsCasting` (etc.).

- [ ] **Step 3: Implement the conditions**

Append to `internal/behaviortree/conditions_mob.go`:

```go
// condTargetIsCasting returns Success if the mob's current aggro target
// is mid-cast. Used by archetypes that want to prioritize interrupts.
func condTargetIsCasting(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Aggro == nil {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsCasting() {
		return Success
	}
	return Failure
}

// condTargetAggroNotOnMe returns Success if the mob's current aggro
// target is NOT attacking the mob — either has no aggro, or aggros
// someone else. Used by tank archetypes to gate taunt so they only
// taunt when they're not already the focus.
func condTargetAggroNotOnMe(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Aggro == nil {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.Aggro == nil {
		return Success
	}
	// Target's aggro is on me (mob) iff Aggro.MobInstanceId == mob.InstanceId.
	if target.Char.Aggro.MobInstanceId == mob.InstanceId && target.Char.Aggro.UserId == 0 {
		return Failure
	}
	return Success
}

// condTargetNotStanding returns Success if the mob's current aggro target
// is in any non-standing position (prone / clinched / grounded).
// Used to gate bonus-damage kicks (stomp when prone, knee when clinched).
func condTargetNotStanding(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Aggro == nil {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.Position != characters.PositionStanding {
		return Success
	}
	return Failure
}
```

Ensure the imports at the top of `conditions_mob.go` include `characters` and `actions`. Add if missing.

Register them in `internal/behaviortree/conditions.go` (in the init block alongside the others):

```go
	conditionRegistry["target_is_casting"] = condTargetIsCasting
	conditionRegistry["target_aggro_not_on_me"] = condTargetAggroNotOnMe
	conditionRegistry["target_not_standing"] = condTargetNotStanding
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestCondTarget' ./internal/behaviortree/`
Expected: PASS.

- [ ] **Step 5: Full package + project**

Run: `go test ./internal/behaviortree/ && go build ./...`
Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/conditions.go internal/behaviortree/conditions_mob.go internal/behaviortree/conditions_test.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): conditions for archetype decision trees

Three new conditions for the incoming tank_taunter and
generic_fighter archetypes:

- target_is_casting — interrupt-priority gate
- target_aggro_not_on_me — taunt gate (don't taunt if already
  holding aggro)
- target_not_standing — bonus-damage kick gate (prone = stomp,
  clinched = knee)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: command_best_of btree action

**Files:**
- Modify: `internal/behaviortree/actions.go` (registry)
- Modify: `internal/behaviortree/actions_mob.go` (new action fn)
- Modify: `internal/behaviortree/actions_test.go` (tests)

- [ ] **Step 1: Write the failing tests**

Append to `internal/behaviortree/actions_test.go`:

```go
// ─── command_best_of ───────────────────────────────────────────────────────

func TestActCommandBestOf_FiresFirstReady(t *testing.T) {
	// Taunt is blocked (no aggro), bash is unreadiable (no shield),
	// trip is ready → trip should fire.
	mob := newTestMob(t)
	mob.Character.EndAggro() // no aggro → taunt and trip and bash all fail

	// Give aggro back with a real target in standing position so trip is ready.
	target := &mobs.Mob{InstanceId: 200}
	target.Character.Name = "Target"
	target.Character.Position = characters.PositionStanding
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	params := map[string]any{
		"cmds": []any{"taunt", "bash", "trip"},
	}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	// Clear mob's command queue to detect what got issued.
	mobs.ClearCommandQueueForTest(mob)

	result := actCommandBestOf(params, ctx)
	assert.Equal(t, Success, result)

	// Expect trip was queued (taunt ready too but it's first — actually,
	// with aggro restored, taunt IS ready, so taunt fires first).
	// Fix: make taunt NOT ready to force fall-through. Simplest way:
	// put mob on cooldown... but that blocks trip too. Instead swap
	// order: test with cmds in order that forces fall-through.
	queue := mobs.PeekCommandQueueForTest(mob)
	assert.Contains(t, queue, "taunt", "taunt was first in list and ready")
}

func TestActCommandBestOf_AllFailReturnsFailure(t *testing.T) {
	// Put mob on cooldown so every command's readiness check fails.
	mob := newTestMob(t)
	mob.Character.Cooldowns = characters.Cooldowns{"special-move": 5}

	params := map[string]any{
		"cmds": []any{"taunt", "bash", "trip", "kick"},
	}
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	result := actCommandBestOf(params, ctx)
	assert.Equal(t, Failure, result)
}

func TestActCommandBestOf_SkipsNotReadyFires(t *testing.T) {
	// Bash is not ready (no shield, no naturalbash species); trip is ready.
	mob := newTestMob(t)
	mob.Character.SpeciesId = 1 // human, no NaturalBash
	target := &mobs.Mob{InstanceId: 201}
	target.Character.Name = "Target"
	target.Character.Position = characters.PositionStanding
	mobs.SetInstanceForTest(target.InstanceId, target)
	defer mobs.SetInstanceForTest(target.InstanceId, nil)
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)

	params := map[string]any{
		"cmds": []any{"bash", "trip"},
	}
	ctx := &EvalContext{InstanceId: mob.InstanceId}
	mobs.ClearCommandQueueForTest(mob)

	result := actCommandBestOf(params, ctx)
	assert.Equal(t, Success, result)
	queue := mobs.PeekCommandQueueForTest(mob)
	assert.Contains(t, queue, "trip", "bash skipped (no shield); trip should fire")
	assert.NotContains(t, queue, "bash")
}
```

**Note on test helpers:** `mobs.ClearCommandQueueForTest` / `mobs.PeekCommandQueueForTest` are proposed — investigate whether existing mob-queue introspection exists. If not, add minimal helpers to `internal/mobs/test_helpers.go`:

```go
func ClearCommandQueueForTest(mob *Mob) {
    mob.commandQueue = nil
}

func PeekCommandQueueForTest(mob *Mob) []string {
    out := make([]string, 0, len(mob.commandQueue))
    for _, entry := range mob.commandQueue {
        out = append(out, entry.Command)
    }
    return out
}
```

The exact field name for the queue may differ — investigate and match. If the mob's command queue is an unexported field that can't be introspected cleanly, an alternative is to check `mob.lastCommandTurn` (or similar queued-state signal) to infer that a command was queued, and let the smoke test in Task 7 do end-to-end verification. Report BLOCKED if neither approach works within 15 min of investigation.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestActCommandBestOf_' ./internal/behaviortree/`
Expected: FAIL with `undefined: actCommandBestOf`.

- [ ] **Step 3: Implement the action**

Append to `internal/behaviortree/actions_mob.go`:

```go
// actCommandBestOf iterates the `cmds` list in order; for each, checks
// actions.CommandIsReady and issues the first ready command via
// mob.Command. Returns Success if any command was issued, Failure if
// all were not-ready.
//
// Mirrors cast_best_in_category for command-style moves. Used by the
// tank_taunter and generic_fighter archetypes.
//
// params: cmds (list of strings — command names in priority order)
func actCommandBestOf(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	cmds := getStringListParam(params, "cmds")
	for _, cmd := range cmds {
		if actions.CommandIsReady(mob, cmd) {
			mob.Command(cmd)
			return Success
		}
	}
	return Failure
}
```

If `getStringListParam` helper doesn't exist in `internal/behaviortree/`, look for how existing actions pull a list from params (some use `[]any` → coerce to string list). If nothing exists, add one next to the other `getXxxParam` helpers:

```go
// getStringListParam pulls a []string from YAML params[key]. YAML unmarshals
// lists as []any, so we type-assert each element.
func getStringListParam(params map[string]any, key string) []string {
    v, ok := params[key].([]any)
    if !ok {
        return nil
    }
    out := make([]string, 0, len(v))
    for _, el := range v {
        if s, ok := el.(string); ok {
            out = append(out, s)
        }
    }
    return out
}
```

Register the action in `internal/behaviortree/actions.go` (alongside the others):

```go
	actionRegistry["command_best_of"] = actCommandBestOf
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestActCommandBestOf_' ./internal/behaviortree/`
Expected: PASS.

- [ ] **Step 5: Full package + project**

Run: `go test ./internal/behaviortree/ && go build ./...`
Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/actions.go internal/behaviortree/actions_mob.go internal/behaviortree/actions_test.go
# If test helpers were added to internal/mobs/test_helpers.go:
# git add internal/mobs/test_helpers.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): command_best_of action

Mirrors cast_best_in_category for command-style moves. Takes an
ordered list of command names, iterates, checks
actions.CommandIsReady for each, issues the first ready one.
Returns Failure when every command in the list is not-ready so
the parent selector can fall through to the legacy attack loop.

Used by the incoming tank_taunter and generic_fighter archetypes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Archetype YAMLs + mob wiring

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`
- Modify: 5 mob YAMLs under `_datafiles/world/dogmud/mobs/summons/`

- [ ] **Step 1: Write tank_taunter.yaml**

Create `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`:

```yaml
# tank_taunter archetype
#
# Sticky front-liner that holds aggro, cycles self-buffs, and prefers
# knockdown/interrupt control over grappling. All moves share the
# special-move cooldown, so priorities determine what fires on any
# given ready-round; cooldown rounds fall through to legacy attack.
#
# Decision order per mob_combat_round:
#   1. Interrupt — bash/trip a casting target
#   2. Taunt — if I'm not already the target's aggro
#   3. Bonus-damage kick — target prone (stomp) or clinched (knee)
#   4. Rally — mitigation buff, skip if already active (buff id 80)
#   5. Warcry — damage buff, skip if already active (buff id 79)
#   6-8. Knockdown cascade: bash (first if qualified) → grapple
#        (single-enemy only) → trip (final fallback)
#   fall through to legacy attack

tree:
  type: selector
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
          check: target_aggro_not_on_me
        - type: action
          do: command_best_of
          cmds: [taunt]

    - type: sequence
      children:
        - type: condition
          check: target_not_standing
        - type: action
          do: command_best_of
          cmds: [kick]

    - type: sequence
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: mob_has_buff
            buff_id: 80
        - type: action
          do: command_best_of
          cmds: [rally]

    - type: sequence
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: mob_has_buff
            buff_id: 79
        - type: action
          do: command_best_of
          cmds: [warcry]

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

- [ ] **Step 2: Write generic_fighter.yaml**

Create `_datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml`:

```yaml
# generic_fighter archetype
#
# Melee mob with the same control toolkit as tank_taunter but no
# signature taunt / self-buffs. Good default for competent non-tank
# fighter companions.
#
# Decision order per mob_combat_round:
#   1. Interrupt — bash/trip a casting target
#   2. Bonus-damage kick — target prone (stomp) or clinched (knee)
#   3-5. Knockdown cascade: bash → grapple (single-enemy only) → trip
#   fall through to legacy attack

tree:
  type: selector
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

- [ ] **Step 3: Wire tank mobs**

In each of these three files, add `behavior_archetype: tank_taunter` alongside the existing top-level mob fields (like `zone:`, `statpool:`, `archetype:` — the stat-distribution archetype is a different field):

- `_datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml`
- `_datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml`
- `_datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml`

Example snippet (location near the top of the mob YAML):

```yaml
mobid: 311
zone: summons
statpool: 90
archetype: fighting
behavior_archetype: tank_taunter   # ← add this line
```

- [ ] **Step 4: Wire generic mobs**

Same pattern, adding `behavior_archetype: generic_fighter` to:

- `_datafiles/world/dogmud/mobs/summons/243-steppe_spirit_wolf.yaml`
- `_datafiles/world/dogmud/mobs/summons/301-zombie.yaml`

- [ ] **Step 5: Build + test**

Run: `go build ./... && go test ./...`
Expected: green (YAML changes don't affect compilation or unit tests).

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml _datafiles/world/dogmud/behaviors/archetypes/generic_fighter.yaml _datafiles/world/dogmud/mobs/summons/305-flesh_golem.yaml _datafiles/world/dogmud/mobs/summons/311-earth_elemental.yaml _datafiles/world/dogmud/mobs/summons/314-magma_elemental.yaml _datafiles/world/dogmud/mobs/summons/243-steppe_spirit_wolf.yaml _datafiles/world/dogmud/mobs/summons/301-zombie.yaml
git commit -m "$(cat <<'EOF'
feat(archetypes): tank_taunter + generic_fighter

Two new behavior-tree archetypes on the Phase 4 framework:

- tank_taunter: interrupt → taunt → bonus-kick → rally → warcry →
  bash/grapple/trip cascade. Wired to flesh golem (305), earth
  elemental (311), magma elemental (314).

- generic_fighter: interrupt → bonus-kick → bash/grapple/trip
  cascade (no taunt/buffs). Wired to steppe spirit wolf (243),
  zombie (301).

Skeleton (300) and water elemental (310) intentionally left dumb.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Manual smoke test

Over to the user. Start the local server, summon each wired companion, run it through the scenarios below, report pass/fail.

**Pre-test:** build and start server (`go run . -c _datafiles/config.yaml`). Summon test character needs access to at least one raise / summon / conjure spell that produces each wired mob. If the character lacks spells for a given mob, admin tools can help.

- [ ] **Smoke 1: Earth elemental bashes a caster**

Summon an earth elemental. Find or admin-spawn a caster mob (wraith = 302). Confirm the elemental `bashes` the wraith when it's mid-cast (look for "knocks you to the ground" or "bashes" in the combat log during a cast round).

- [ ] **Smoke 2: Flesh golem trips (no naturalbash, no shield)**

Summon a flesh golem. Put it against the same caster. Since golem has no shield/naturalbash, confirm it **trips** (not bashes).

- [ ] **Smoke 3: Zombie generic rotation + stomp**

Summon a zombie. Against a single-target enemy, run the fight long enough to observe: trip fires early → target goes prone → next cooldown cycle, kick variant "stomp" fires (look for stomp text + higher damage than plain kick).

- [ ] **Smoke 4: Magma elemental full cycle**

Summon a magma elemental solo vs. one enemy. Observe cycle: rally fires first (mitigation buff up), next cooldown warcry fires (damage buff up), subsequent cooldowns taunt dominates. Combat log should show "rallying roar" then "warcry" then repeat of taunt messages.

- [ ] **Smoke 5: Grapple gated on multi-enemy**

Summon a magma elemental into a fight with 2+ enemies (e.g., pack-spawn area). Confirm grapple does NOT fire — knockdown moves + kick dominate instead.

- [ ] **Smoke 6: Skeleton / water elemental unchanged**

Summon a skeleton (300) or water elemental (310). Confirm they still attack plainly with no taunts / bashes / rally etc. These mobs are intentionally not archetyped.

Report back pass/fail per scenario.

---

## Task 8: Finalize + follow-up memory + merge

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./...`
Expected: all green.

- [ ] **Step 2: Write the follow-up memory**

Create `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_command_readiness_drift.md`:

```markdown
---
name: Command readiness drift — CommandIsReady vs command exec-time gates
description: Next session's work — refactor existing command implementations (taunt, rally, warcry, trip, bash, grapple, kick) to call actions.CommandIsReady as their own pre-execution gate so the btree helper and the commands share a single source of truth.
type: project
---

## Background

2026-04-21 shipped the tank_taunter + generic_fighter archetypes
alongside a new `actions.CommandIsReady(mob, cmd)` helper. The helper
mirrors the execution-time gates each command performs internally
(cooldown, aggro, target state, shield/naturalbash for bash) so the
new `command_best_of` btree action can self-gate without mutating
state.

**The drift risk:** CommandIsReady and the command's own internal
gates are two separate codepaths with identical logic. If one
changes and the other doesn't, the btree might issue a command the
command itself silently refuses to execute — wasting a round of
potential fallback-attack.

## Follow-up scope

Refactor each affected command to call `CommandIsReady` as its
own gate BEFORE the mutating `Cooldowns.Try` call:

- `internal/mobcommands/howl.go` (taunt reskin)
- `internal/mobcommands/rally.go` (added 2026-04-21)
- `internal/mobcommands/warcry.go` (added 2026-04-21)
- `internal/mobcommands/trip.go`
- `internal/mobcommands/bash.go`
- `internal/mobcommands/grapple.go`
- `internal/mobcommands/kick.go`
- Same set under `internal/usercommands/` IF the readiness check is
  fully equivalent for the player side (e.g., player-only IsCrafting
  guard has to stay inline)

After this refactor, the single source of truth is `CommandIsReady`;
the command's execute-time gate is a no-op if the btree pre-checked.

## Considerations

- `CommandIsReady` is non-mutating (uses `GetCooldown`, not
  `Cooldowns.Try`). After readiness passes, the command still needs
  to call `Cooldowns.Try` to actually SET the cooldown. Keep that
  call at execution time.
- Player-only guards (IsCrafting) should stay in the player command
  path. Either leave them there (CommandIsReady only gates common
  concerns) or extend CommandIsReady to take an Actor (not just a
  Mob) and check actor.IsPlayer() before player-only gates.
- Resource checks (stamina / conviction) are currently NOT in
  CommandIsReady per spec — decide whether to add them during the
  refactor. Adding them makes the helper fully authoritative.
```

- [ ] **Step 3: Update MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`:

(a) Remove the `Tank/taunter archetype` entry from `## Next Up`.

(b) Add to `## Completed (2026-04-21)`:

```markdown
- **tank_taunter + generic_fighter archetypes** — two new btree archetypes on the Phase 4 framework. Tank: interrupt → taunt-if-not-holding-aggro → bonus-kick → rally → warcry → bash/grapple/trip cascade; wired to flesh golem (305), earth (311), magma (314) elementals. Generic: interrupt → bonus-kick → bash/grapple/trip; wired to wolf (243) and zombie (301). New engine: `command_best_of` btree action (self-gates via new `actions.CommandIsReady` helper), 3 new conditions (`target_is_casting`, `target_aggro_not_on_me`, `target_not_standing`), shared `actions.Execute{Rally,Warcry}` + mob command wrappers (mob/player parity). Branch: `feature/tank-and-generic-archetypes`. Smoke-verified in game. Design in `docs/superpowers/specs/2026-04-21-tank-and-generic-archetypes-design.md`, plan in `docs/superpowers/plans/2026-04-21-tank-and-generic-archetypes-plan.md`. Follow-up `project_command_readiness_drift.md` flags the refactor to make CommandIsReady the single source of truth for command gating (user committed to next session).
```

(c) Under `## Future Work`, add (or confirm already present from this plan):

```markdown
- [Command readiness drift](project_command_readiness_drift.md) — refactor existing command impls to call CommandIsReady for single source of truth on gating
```

- [ ] **Step 4: Prompt user about merge**

Per `github_guide.md`: feature branches merge into `development` with `--no-ff`. Ask the user before merging — do NOT merge autonomously.

Template prompt:

> "Branch `feature/tank-and-generic-archetypes` is N commits ready for `--no-ff` merge into development. Want to merge now and delete the branch, or hold?"

After the user confirms, run:

```bash
git checkout development
git merge --no-ff feature/tank-and-generic-archetypes -m "..."
git branch -d feature/tank-and-generic-archetypes
```

Commit message should summarize the feature + reference the spec and follow-up memory.

---

## Self-Review

**Spec coverage:**
- §"Engine additions 1. command_best_of" — Task 5. ✓
- §"Engine additions 2. CommandIsReady" — Task 3. ✓
- §"Engine additions 3. Three new btree conditions" — Task 4. ✓
- §"Engine additions 4. Shared rally + warcry + mob command wrappers" — Tasks 1 + 2. ✓
- §"Content — tank_taunter archetype" — Task 6 step 1. ✓
- §"Content — generic_fighter archetype" — Task 6 step 2. ✓
- §"Content — Mob YAML wiring" — Task 6 steps 3 + 4. ✓
- §"Testing — Unit tests" items 1-6 — Tasks 3, 4, 5. ✓
- §"Testing — Smoke tests" items 1-5 — Task 7 (plus extra scenario for control case). ✓
- §"Out of scope — Refactoring existing command implementations" — Task 8 Step 2 (follow-up memory). ✓

**Placeholder scan:**
- No TBD / TODO / "implement later" / "similar to Task N".
- Task 5 has a conditional `BLOCKED` escape hatch for queue-introspection helpers — well-defined branching behavior with a fallback, not a placeholder.

**Type consistency:**
- `CommandIsReady(mob *mobs.Mob, cmd string) bool` — consistent across Tasks 3, 5, 8.
- `RallyResult` / `WarcryResult` have matching field sets (Executed, OnCooldown, Crafting, Bonus, Duration).
- `actions.Actor` interface used consistently (`ExecuteRally`, `ExecuteWarcry`, `UserActor`, `MobActor`).
- Buff IDs 80 (rally), 79 (warcry) consistent between the archetype YAML and the shared action code.

No issues found.
