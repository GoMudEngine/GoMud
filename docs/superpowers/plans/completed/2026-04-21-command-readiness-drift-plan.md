# Command Readiness Drift Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `actions.CommandIsReady` the single source of truth for combat-command gating, backed by drift-detection tests. Close the universal IsCrafting hole across all 7 combat commands in the process.

**Architecture:** Each `Execute*` function adds a consistent early-return gate structure (IsCrafting universal, rally/warcry AlreadyActive). `CommandIsReady` is expanded to mirror these gates under an Actor signature. A new table-driven drift-detection test enforces agreement between the two.

**Tech Stack:** Go 1.21+, existing `testify` / `actions.Actor` interface / `characters.Character.HasBuff`.

**Related spec:** `docs/superpowers/specs/2026-04-21-command-readiness-drift-design.md`
**Branch:** `feature/command-readiness-drift` (created; spec committed as `3ece5aba`).

---

## File Structure

**Modified Go files:**
- `internal/actions/combat_rally.go` — flip IsCrafting universal, add AlreadyActive
- `internal/actions/combat_warcry.go` — same pattern
- `internal/actions/combat_bash.go` — add IsCrafting gate + Crafting field to BashResult
- `internal/actions/combat_trip.go` — same
- `internal/actions/combat_grapple.go` — same
- `internal/actions/combat_kick.go` — same
- `internal/actions/combat_taunt.go` — same
- `internal/actions/command_readiness.go` — Actor signature + new gates
- `internal/actions/command_readiness_test.go` — migrate to Actor, add new-gate tests
- `internal/behaviortree/actions_mob.go` — update `actCommandBestOf` caller
- `internal/usercommands/bash.go` — IsCrafting pre-reject + map Crafting result
- `internal/usercommands/trip.go` — same
- `internal/usercommands/grapple.go` — same
- `internal/usercommands/kick.go` — same
- `internal/usercommands/taunt.go` — same
- `internal/usercommands/rally.go` — map new AlreadyActive result to message
- `internal/usercommands/warcry.go` — same

**New Go files:**
- `internal/actions/command_readiness_drift_test.go` — table-driven drift detection

**Modified content:**
- `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml` — remove decorator-invert blocks for rally/warcry

---

## Task 1: Update ExecuteRally + ExecuteWarcry (universal IsCrafting + AlreadyActive)

**Files:**
- Modify: `internal/actions/combat_rally.go`
- Modify: `internal/actions/combat_warcry.go`
- Modify: `internal/usercommands/rally.go` (handle new `AlreadyActive` branch)
- Modify: `internal/usercommands/warcry.go` (same)

Flip IsCrafting from `actor.IsPlayer() && char.IsCrafting()` to universal `char.IsCrafting()`. Add a new `AlreadyActive` early-return before the cooldown check for actors that already have the buff.

- [ ] **Step 1: Modify `combat_rally.go`**

Replace the current IsCrafting check at line 28-30:

```go
	// IsCrafting applies to players only; mobs never craft.
	if actor.IsPlayer() && char.IsCrafting() {
		return RallyResult{Crafting: true}
	}
```

With (updated comment + universal check):

```go
	// IsCrafting applies universally — mobs can craft too (future
	// crafter archetype) and should not interrupt their craft to
	// rally.
	if char.IsCrafting() {
		return RallyResult{Crafting: true}
	}
```

Immediately after that block, ADD the new AlreadyActive early-return:

```go
	// Skip if the rally buff is already active on this actor —
	// re-casting would just burn the cooldown for no new effect.
	if char.HasBuff(80) {
		return RallyResult{AlreadyActive: true}
	}
```

Add `AlreadyActive bool` to the RallyResult struct definition (also update the docstring on the struct):

```go
type RallyResult struct {
	Executed      bool    // true if the rally actually applied
	OnCooldown    bool    // blocked by shared special-move cooldown
	Crafting      bool    // blocked because the actor is mid-craft
	AlreadyActive bool    // blocked because the rally buff is already on this actor
	Bonus         float64 // mitigation bonus the condition carries (0.05..0.20)
	Duration      int     // condition duration in rounds
}
```

- [ ] **Step 2: Modify `combat_warcry.go`**

Same pattern as Step 1 but with `HasBuff(79)`. Flip the IsCrafting gate to universal, add the AlreadyActive check for buff 79, add `AlreadyActive bool` to `WarcryResult`.

- [ ] **Step 3: Update `usercommands/rally.go`**

After the `result.OnCooldown` handling, add a new branch for `result.AlreadyActive`:

```go
	if result.AlreadyActive {
		user.SendText("You're already rallied — save it for when it matters.")
		return true, nil
	}
```

Place it BETWEEN the Crafting check and the OnCooldown check (same level; the order between those three doesn't matter for correctness, but alphabetically / thematically: Crafting → AlreadyActive → OnCooldown is fine).

- [ ] **Step 4: Update `usercommands/warcry.go`**

Same pattern. Message: `"Your warcry still echoes — you can't shout it louder."` (or similar short flavor text consistent with the existing warcry tone).

- [ ] **Step 5: Build + test**

Run: `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/`
Expected: clean build, all tests pass. The existing rally/warcry tests don't exercise the new fields but should still pass because the old code paths still fire.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/combat_rally.go internal/actions/combat_warcry.go internal/usercommands/rally.go internal/usercommands/warcry.go
git commit -m "$(cat <<'EOF'
refactor(actions): rally/warcry universal IsCrafting + AlreadyActive

Flip rally/warcry IsCrafting gate from player-only to universal
(mobs can craft too, future crafter archetype). Add AlreadyActive
early-return when the rally/warcry buff is already on the actor
so the btree doesn't burn the cooldown re-casting. User commands
map the new AlreadyActive result to an appropriate message.

Part of the command-readiness-drift refactor — the parallel
CommandIsReady update lands in a later commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add IsCrafting gate to bash/trip/grapple/kick/taunt Execute* functions

**Files:**
- Modify: `internal/actions/combat_bash.go`
- Modify: `internal/actions/combat_trip.go`
- Modify: `internal/actions/combat_grapple.go`
- Modify: `internal/actions/combat_kick.go`
- Modify: `internal/actions/combat_taunt.go`

Add a consistent IsCrafting gate at the top of each Execute* function. Each adds a `Crafting bool` field to its result struct.

- [ ] **Step 1: Read each Execute* function briefly**

Run: `for f in internal/actions/combat_bash.go internal/actions/combat_trip.go internal/actions/combat_grapple.go internal/actions/combat_kick.go internal/actions/combat_taunt.go; do echo "=== $f ==="; head -40 "$f"; done`

Verify each has a `<X>Result` struct at the top and an early-return pattern. Confirm none have an existing `Crafting bool` field (they shouldn't — only rally/warcry do today).

- [ ] **Step 2: Modify each Execute* — consistent pattern**

For each of the 5 files, add `Crafting bool` to the result struct (grouped with other gate-flags like OnCooldown). Then add a gate at the very top of the Execute function, BEFORE any other check:

```go
func ExecuteBash(actor Actor) BashResult {
	char := actor.GetCharacter()

	// Don't interrupt a craft to swing a shield.
	if char.IsCrafting() {
		return BashResult{Crafting: true}
	}

	// ... rest of existing function
}
```

The comment can be thematic per command ("Don't interrupt a craft to swing a shield" for bash, "...to trip someone" for trip, "...to taunt an enemy" for taunt, etc.). Keep it short; the consistency is more important than the wit.

**Important:** some Execute* functions resolve target and check aggro BEFORE anything else (e.g., `ExecuteBash` currently has `if char.Aggro == nil` as its first check). The IsCrafting gate should come BEFORE the aggro check — reject the crafting state earliest.

- [ ] **Step 3: Build + test**

Run: `go build ./... && go test ./internal/actions/`
Expected: clean build, existing tests pass. The new `Crafting` field is unexercised by current tests but doesn't break anything.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/combat_bash.go internal/actions/combat_trip.go internal/actions/combat_grapple.go internal/actions/combat_kick.go internal/actions/combat_taunt.go
git commit -m "$(cat <<'EOF'
feat(actions): universal IsCrafting gate on bash/trip/grapple/kick/taunt

Add an IsCrafting early-return to the five combat Execute*
functions that previously didn't check — a player or mob mid-craft
can no longer bash/trip/grapple/kick/taunt their way out of the
craft state. Each adds a Crafting bool to its result struct so
command-wrapper display logic can map it to a user-friendly error.

Part of the command-readiness-drift refactor + partial fulfillment
of project_active_command_crafting_audit.md (combat commands now
covered; cast/mutations/eat/drink still out of scope).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add IsCrafting pre-reject to 5 user commands for nicer UX

**Files:**
- Modify: `internal/usercommands/bash.go`
- Modify: `internal/usercommands/trip.go`
- Modify: `internal/usercommands/grapple.go`
- Modify: `internal/usercommands/kick.go`
- Modify: `internal/usercommands/taunt.go`

Each of these commands now has an IsCrafting gate inside `ExecuteX`, but the user commands do pre-Execute target-resolution and aggro-setting work. Reject the craft state at the top so we don't run that setup and then reject.

- [ ] **Step 1: Add pre-reject block at the top of each user command**

For each user command, add this block right after the function signature, BEFORE any target resolution or aggro setup:

```go
	if user.Character.IsCrafting() {
		user.SendText(`<ansi fg="red">You can't bash while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
```

Substitute the verb per command:
- bash: "bash"
- trip: "trip someone"
- grapple: "grapple"
- kick: "kick"
- taunt: "taunt"

Pattern matches the existing rally/warcry user-command IsCrafting handler (the one added when ExecuteRally was extracted in an earlier session).

- [ ] **Step 2: Map the Execute* `Crafting: true` result too**

After each user command's call to `ExecuteX`, add a branch for `result.Crafting`. This is the safety-net for the edge case where target resolution succeeds but the actor somehow entered the crafting state between (unlikely but cheap to handle):

```go
	if bashResult.Crafting {
		// Should have been caught by the pre-reject above; safety net.
		user.SendText(`<ansi fg="red">You can't bash while focused on your work. Finish or be interrupted first.</ansi>`)
		return true, nil
	}
```

Place it alongside the other result-flag branches (NoShield, OnCooldown, etc.). This is defensive — the pre-reject at the top should always catch it first.

- [ ] **Step 3: Build + test**

Run: `go build ./... && go test ./internal/usercommands/`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/bash.go internal/usercommands/trip.go internal/usercommands/grapple.go internal/usercommands/kick.go internal/usercommands/taunt.go
git commit -m "$(cat <<'EOF'
feat(usercommands): IsCrafting pre-reject on combat commands

Five combat user commands now reject with a friendly message
when the player is mid-craft, before running target-resolution
and aggro-setup. The Execute* layer still rejects as a safety
net.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Expand CommandIsReady (Actor signature + new gates) + migrate tests + update caller

**Files:**
- Modify: `internal/actions/command_readiness.go`
- Modify: `internal/actions/command_readiness_test.go`
- Modify: `internal/behaviortree/actions_mob.go` (update `actCommandBestOf` caller)

- [ ] **Step 1: Replace `command_readiness.go`**

Full new file content:

```go
package actions

// CommandIsReady returns true iff the named mob command would actually
// execute its effect right now. Mirrors the early-return gates in each
// Execute* function so the behavior tree's command_best_of action can
// self-gate and fall through cleanly.
//
// Takes an Actor (not a *mobs.Mob) so player-side callers can self-gate
// too if needed in the future. Actor also gives us GetCharacter() for
// the universal state checks.
//
// Unknown command names return false. This lets a behavior tree safely
// include commands that don't exist yet without firing a spurious
// Success.
//
// SYNC POINT: if this function gains a new gate, update the
// corresponding Execute* function. Drift is caught by
// TestCommandReadinessDrift in command_readiness_drift_test.go.
func CommandIsReady(actor Actor, cmd string) bool {
	if actor == nil {
		return false
	}
	char := actor.GetCharacter()
	if char == nil {
		return false
	}

	// Universal gates (apply to every command).
	if char.IsCrafting() {
		return false
	}
	if char.GetCooldown("special-move") > 0 {
		return false
	}

	switch cmd {
	case "taunt":
		return char.Aggro != nil

	case "rally":
		return !char.HasBuff(80)

	case "warcry":
		return !char.HasBuff(79)

	case "trip":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.CombatPosition.IsGroundPosition()

	case "bash":
		if char.Aggro == nil {
			return false
		}
		return char.HasShield()

	case "grapple":
		if char.Aggro == nil {
			return false
		}
		target := ResolveAggroTarget(char.Aggro)
		if !target.Found {
			return false
		}
		return !target.Char.CombatPosition.IsGrapplePosition()

	case "kick":
		return char.Aggro != nil
	}

	return false
}
```

Key changes vs. old file:
- Parameter is `actor Actor` (was `mob *mobs.Mob`)
- Imports: remove `mobs`, no new imports needed (Actor is in-package)
- New universal `IsCrafting` check
- Rally/warcry branches gain `!HasBuff(buffId)` checks
- Nil-actor guard + nil-character guard instead of nil-mob

- [ ] **Step 2: Migrate `command_readiness_test.go` to Actor-based API**

Every test currently does `CommandIsReady(m, "cmd")` where `m` is `*mobs.Mob`. Replace each call site with `CommandIsReady(&MobActor{Mob: m, Room: nil}, "cmd")`.

Also add new test cases for the new gates:

```go
// ─── IsCrafting (universal) ────────────────────────────────────────────────

func TestCommandIsReady_IsCrafting_BlocksEveryCommand(t *testing.T) {
	for _, cmd := range []string{"taunt", "rally", "warcry", "trip", "bash", "grapple", "kick"} {
		t.Run(cmd, func(t *testing.T) {
			m := newTestMob(t, func(m *mobs.Mob) {
				// Simulate crafting state. See internal/characters for the
				// exact IsCrafting trigger; if there's a direct setter use
				// it, otherwise set the underlying field.
				m.Character.StartCrafting("test", 1)
			})
			actor := &MobActor{Mob: m, Room: nil}
			assert.False(t, CommandIsReady(actor, cmd),
				"crafting mob should NOT be ready for %s", cmd)
		})
	}
}

// ─── Rally AlreadyActive ───────────────────────────────────────────────────

func TestCommandIsReady_Rally_BuffAlreadyActive_False(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.AddBuff(80, false)
	})
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "rally"))
}

func TestCommandIsReady_Rally_BuffNotActive_True(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "rally"))
}

// ─── Warcry AlreadyActive ──────────────────────────────────────────────────

func TestCommandIsReady_Warcry_BuffAlreadyActive_False(t *testing.T) {
	m := newTestMob(t, func(m *mobs.Mob) {
		m.Character.AddBuff(79, false)
	})
	actor := &MobActor{Mob: m, Room: nil}
	assert.False(t, CommandIsReady(actor, "warcry"))
}

func TestCommandIsReady_Warcry_BuffNotActive_True(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	assert.True(t, CommandIsReady(actor, "warcry"))
}

// ─── Nil actor ─────────────────────────────────────────────────────────────

func TestCommandIsReady_NilActor(t *testing.T) {
	assert.False(t, CommandIsReady(nil, "taunt"))
}
```

**Note on `StartCrafting`:** check `internal/characters/crafting.go` for the correct method to set a character into crafting state. If it's something other than `StartCrafting`, adjust. If it requires arguments (recipe name, rounds remaining), pass plausible test values. If no public method exists and it only sets an internal field, use the exact field name (e.g., `m.Character.CraftingState = &characters.CraftingState{...}`).

Delete the old `TestCommandIsReady_NilMob` test — `CommandIsReady_NilActor` replaces it.

- [ ] **Step 3: Update `actCommandBestOf` caller**

In `internal/behaviortree/actions_mob.go`, find the call:

```go
		if actions.CommandIsReady(mob, cmd) {
			mob.Command(cmd)
			return Success
		}
```

Replace with:

```go
		if actions.CommandIsReady(&actions.MobActor{Mob: mob, Room: rooms.LoadRoom(ctx.RoomId)}, cmd) {
			mob.Command(cmd)
			return Success
		}
```

The Room is available via `ctx.RoomId`. If `rooms` is not already imported in this file, add the import. Actor-constructors accept nil Room if the CommandIsReady path doesn't need it — but CommandIsReady calls `actor.GetCharacter()` not `actor.GetRoom()` so a nil Room is fine for this call site. Simpler alternative:

```go
		if actions.CommandIsReady(&actions.MobActor{Mob: mob}, cmd) {
```

(Leaving Room as its zero value, nil.) Use this simpler form unless the MobActor constructor rejects nil Room.

**Check before choosing:** run `grep -n "type MobActor\|func.*MobActor.*Room" internal/actions/actor_mob.go` and read the fields. If Room is accessed unconditionally in any method CommandIsReady calls transitively, use the LoadRoom form.

- [ ] **Step 4: Build + test**

Run: `go build ./... && go test ./internal/actions/ ./internal/behaviortree/`
Expected: clean. All migrated tests should pass. New gate tests should also pass (IsCrafting blocks, rally/warcry gate on buff).

- [ ] **Step 5: Commit**

```bash
git add internal/actions/command_readiness.go internal/actions/command_readiness_test.go internal/behaviortree/actions_mob.go
git commit -m "$(cat <<'EOF'
refactor(actions): CommandIsReady Actor signature + new gates

- Signature: *mobs.Mob → Actor (aligns with Execute* pattern;
  enables future player-side self-gating).
- Universal IsCrafting gate at the top (matches the Execute*
  changes in the parent commit).
- Rally/warcry now gate on !HasBuff(80)/(79) so the btree
  doesn't burn a cooldown re-casting an active buff.
- Tests migrated to Actor-based calls; new cases for each
  new gate.
- actCommandBestOf caller updated to wrap the mob in
  MobActor.

Drift-detection test lands in a follow-up commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Simplify tank_taunter archetype YAML

**Files:**
- Modify: `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`

Now that `CommandIsReady` gates rally/warcry on `!HasBuff(...)`, the YAML's `decorator invert + mob_has_buff` blocks are redundant. Remove them.

- [ ] **Step 1: Find the rally block**

The tree currently has this structure for the rally branch:

```yaml
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
```

Replace it with:

```yaml
    - type: action
      do: command_best_of
      cmds: [rally]
```

- [ ] **Step 2: Find the warcry block**

Same structure with `buff_id: 79` and `cmds: [warcry]`. Same simplification.

- [ ] **Step 3: Update the header docstring**

In the top-of-file comment block, update the description lines for steps 4 and 5 (rally and warcry). Current:

```yaml
#   4. Rally — mitigation buff, skip if already active (buff id 80)
#   5. Warcry — damage buff, skip if already active (buff id 79)
```

New (drop the "skip if already active" parenthetical since it's now implicit via CommandIsReady):

```yaml
#   4. Rally — mitigation buff (CommandIsReady skips if already active)
#   5. Warcry — damage buff (CommandIsReady skips if already active)
```

- [ ] **Step 4: Verify**

Run `grep -n "mob_has_buff" _datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`
Expected: no output (both occurrences removed).

Run `go build ./... && go test ./...`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml
git commit -m "$(cat <<'EOF'
chore(archetypes): drop redundant duplicate-buff decorators

CommandIsReady now gates rally on !HasBuff(80) and warcry on
!HasBuff(79) internally, so the tree's decorator invert +
mob_has_buff wrappers are redundant. Remove both blocks for a
cleaner tree and to document the new single-source-of-truth
convention.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Drift-detection test

**Files:**
- Create: `internal/actions/command_readiness_drift_test.go`

Table-driven per-command test asserting `CommandIsReady` and `Execute*` agree on whether the command would fire for a given actor state.

- [ ] **Step 1: Write the test file**

Create `internal/actions/command_readiness_drift_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// driftCase describes one point in the (command × gate) matrix.
type driftCase struct {
	name       string
	cmd        string
	mutate     func(*mobs.Mob)
	wantReady  bool   // CommandIsReady should return this
	wantReason string // Execute*-side: expected "not ready" flag name
	// If wantReady=true, wantReason is ignored (Execute* actually runs).
}

// TestCommandReadinessDrift asserts CommandIsReady and each Execute*
// agree on readiness for a shared actor state. When they diverge, the
// btree's command_best_of can issue a command the command itself
// silently rejects — exactly the class of bug that surfaced in T7
// smoke testing of the tank_taunter archetype.
//
// To add a new command or gate: add a row below. The test engine does
// the rest.
//
// SYNC POINT: when adding a new gate to CommandIsReady or an
// Execute*, add the corresponding drift row here.
func TestCommandReadinessDrift(t *testing.T) {
	cases := []driftCase{
		// ─── taunt ────────────────────────────────────────────────
		{"taunt_ready", "taunt",
			func(m *mobs.Mob) { m.Character.SetAggro(1, 0, characters.DefaultAttack) },
			true, ""},
		{"taunt_crafting", "taunt",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.StartCrafting("test", 1)
			},
			false, "Crafting"},
		{"taunt_cooldown", "taunt",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"taunt_no_aggro", "taunt",
			func(m *mobs.Mob) { /* no aggro */ },
			false, "NoTarget"},

		// ─── rally ────────────────────────────────────────────────
		{"rally_ready", "rally",
			nil,
			true, ""},
		{"rally_crafting", "rally",
			func(m *mobs.Mob) { m.Character.StartCrafting("test", 1) },
			false, "Crafting"},
		{"rally_cooldown", "rally",
			func(m *mobs.Mob) { m.Character.Cooldowns = characters.Cooldowns{"special-move": 3} },
			false, "OnCooldown"},
		{"rally_already_active", "rally",
			func(m *mobs.Mob) { m.Character.AddBuff(80, false) },
			false, "AlreadyActive"},

		// ─── warcry ───────────────────────────────────────────────
		{"warcry_ready", "warcry", nil, true, ""},
		{"warcry_crafting", "warcry",
			func(m *mobs.Mob) { m.Character.StartCrafting("test", 1) },
			false, "Crafting"},
		{"warcry_cooldown", "warcry",
			func(m *mobs.Mob) { m.Character.Cooldowns = characters.Cooldowns{"special-move": 3} },
			false, "OnCooldown"},
		{"warcry_already_active", "warcry",
			func(m *mobs.Mob) { m.Character.AddBuff(79, false) },
			false, "AlreadyActive"},

		// ─── trip ─────────────────────────────────────────────────
		// Trip ready-case needs a standing target registered via
		// SetInstanceForTest; see inline setup.
		{"trip_crafting", "trip",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.StartCrafting("test", 1)
			},
			false, "Crafting"},
		{"trip_cooldown", "trip",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"trip_no_aggro", "trip",
			func(m *mobs.Mob) { /* no aggro */ },
			false, "NoTarget"},

		// ─── bash ─────────────────────────────────────────────────
		{"bash_crafting", "bash",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.StartCrafting("test", 1)
			},
			false, "Crafting"},
		{"bash_no_shield", "bash",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 1 // human, no naturalbash, no shield
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
			},
			false, "NoShield"},
		{"bash_cooldown", "bash",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},

		// ─── grapple ──────────────────────────────────────────────
		{"grapple_crafting", "grapple",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.StartCrafting("test", 1)
			},
			false, "Crafting"},
		{"grapple_cooldown", "grapple",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"grapple_no_aggro", "grapple",
			func(m *mobs.Mob) { /* no aggro */ },
			false, "NoTarget"},

		// ─── kick ─────────────────────────────────────────────────
		{"kick_ready", "kick",
			func(m *mobs.Mob) { m.Character.SetAggro(1, 0, characters.DefaultAttack) },
			true, ""},
		{"kick_crafting", "kick",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.StartCrafting("test", 1)
			},
			false, "Crafting"},
		{"kick_cooldown", "kick",
			func(m *mobs.Mob) {
				m.Character.SetAggro(1, 0, characters.DefaultAttack)
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"kick_no_aggro", "kick",
			func(m *mobs.Mob) { /* no aggro */ },
			false, "NoTarget"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mob := newTestMob(t, tc.mutate)
			actor := &MobActor{Mob: mob, Room: nil}

			gotReady := CommandIsReady(actor, tc.cmd)
			assert.Equal(t, tc.wantReady, gotReady,
				"CommandIsReady(%s) for case %q", tc.cmd, tc.name)

			if tc.wantReady {
				// Don't run Execute* for happy-path — some have target-
				// resolution or other side effects that require more
				// setup. The !wantReady path is where drift matters.
				return
			}

			// Not-ready: run the Execute* and assert the specific flag.
			gotFlag := runExecuteAndReadFlag(tc.cmd, actor, tc.wantReason)
			assert.True(t, gotFlag,
				"Execute%s for case %q did not return %s=true", tc.cmd, tc.name, tc.wantReason)
		})
	}
}

// runExecuteAndReadFlag dispatches to the Execute* matching cmd and
// returns whether the named result field is true. Returns false if
// the command or flag isn't recognised.
func runExecuteAndReadFlag(cmd string, actor Actor, flag string) bool {
	switch cmd {
	case "taunt":
		r := ExecuteTaunt(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "rally":
		r := ExecuteRally(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "AlreadyActive":
			return r.AlreadyActive
		}
	case "warcry":
		r := ExecuteWarcry(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "AlreadyActive":
			return r.AlreadyActive
		}
	case "trip":
		r := ExecuteTrip(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "bash":
		r := ExecuteBash(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoShield":
			return r.NoShield
		}
	case "grapple":
		r := ExecuteGrapple(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "kick":
		r := ExecuteKick(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	}
	return false
}
```

**Important investigation before running:**

1. **`StartCrafting` method.** Verify the correct method to enter crafting state. Check `internal/characters/crafting.go` for the public API. If it's `StartCraft`, adjust. If it needs different args, adjust the test mutators accordingly.

2. **Execute* result field names.** The switch in `runExecuteAndReadFlag` references fields like `r.Crafting`, `r.NoTarget`, `r.NoShield`. Verify these exist on each Execute* return type after Tasks 1 and 2 land. For instance, ExecuteTaunt's TauntResult has `NoTarget bool` (not `NoAggro`) — the test expects that. If any field name differs (e.g., an Execute* uses `Executed: false` + no specific flag for a miss), adjust that case's `wantReason` to match, OR omit that case if there's no specific flag to assert (the happy-path still confirms the positive side of drift).

3. **`NoTarget` for trip/grapple/kick.** Check that ExecuteTrip / ExecuteGrapple / ExecuteKick have a `NoTarget` flag. If they use a different name, adjust. If they silently return `Executed: false` with no specific "why" flag, use `Executed` and assert it's false in a modified helper.

- [ ] **Step 2: Run the test**

Run: `go test -run 'TestCommandReadinessDrift' ./internal/actions/ -v`
Expected: all cases PASS. If any fail, that's drift — investigate whether the failure is real drift (fix the code) or a test-expectation mismatch (fix the test row).

- [ ] **Step 3: Verify full test suite still green**

Run: `go test ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/command_readiness_drift_test.go
git commit -m "$(cat <<'EOF'
test(actions): drift-detection test for CommandIsReady vs Execute*

Table-driven, ~28 cases covering each of the 7 combat commands
× each failure gate (crafting, cooldown, no aggro, no shield,
already-active for rally/warcry). For every case, asserts that
CommandIsReady and Execute* agree on readiness.

This is the contract test between the btree's self-gating path
and each command's execution-time gates. Future Execute* or
CommandIsReady changes that add a gate must add a drift row so
the agreement is preserved.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Manual smoke test

Over to the user. Start the server and validate:

- [ ] **Smoke 1: Crafting blocks combat**

Start a craft (any gather + recipe). Try `bash <target>` / `trip <target>` / `grapple <target>` / `kick <target>` / `taunt <target>` — each should reject with a friendly "focused on your work" message. Try `rally` and `warcry` — should also reject.

- [ ] **Smoke 2: Tank companion rally/warcry cadence**

Summon a magma elemental (tank_taunter). Enter combat. Watch the combat log across ~10 rounds. Rally should fire once early, warcry once, and then taunts/knockdowns dominate the rotation. Rally should NOT fire a second time while the rally buff is still active (CommandIsReady skips it, btree falls through).

- [ ] **Smoke 3: Tank rhythm feels right**

Same magma elemental fight. General vibe check — does the archetype feel competent vs last session? No regressions in taunt aggro-pull, bonus-damage kick on prone, knockdown cascade?

Report pass/fail per scenario.

---

## Task 8: Finalize + merge

- [ ] **Step 1: Full test suite**

Run: `go build ./... && go test ./...`
Expected: all green.

- [ ] **Step 2: Update MEMORY.md**

In `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`:

(a) Remove the "Command readiness drift refactor" entry from `## Next Up`.

(b) Add to `## Completed (2026-04-21)` (or if the calendar has rolled over, start a new `## Completed (<today>)` section):

```markdown
- **Command readiness drift refactor** — made actions.CommandIsReady the single source of truth for combat-command gating, backed by a table-driven drift-detection test. Signature flipped from `*mobs.Mob` → `Actor`. New universal IsCrafting gate added to all 7 combat Execute* functions (5 new: bash/trip/grapple/kick/taunt; 2 flipped from player-only to universal: rally/warcry). New AlreadyActive early-return for rally/warcry when the buff is already on the actor. Tank_taunter archetype YAML simplified (removed decorator-invert mob_has_buff blocks — CommandIsReady now authoritative). ~28-case drift test at internal/actions/command_readiness_drift_test.go asserts CommandIsReady and each Execute* agree on readiness for every (command, gate) pair. Partial fulfillment of project_active_command_crafting_audit.md (combat commands now covered; cast/mutations/eat/drink still future). Design in `docs/superpowers/specs/2026-04-21-command-readiness-drift-design.md`, plan in `docs/superpowers/plans/2026-04-21-command-readiness-drift-plan.md`.
```

(c) Confirm `project_respawn_aggro_death_loop.md` is still the top entry under `## Bugs to Fix` (it was added earlier this session). If it's there, no change. That's the next work to tackle after this merges.

- [ ] **Step 3: Prompt user about merge**

Per `github_guide.md`: feature branches merge into `development` with `--no-ff`. Ask the user before merging — do NOT merge autonomously.

Template prompt:

> "Branch `feature/command-readiness-drift` is N commits ready for `--no-ff` merge into development. Want to merge now and delete the branch, or hold?"

After the user confirms, run:

```bash
git checkout development
git merge --no-ff feature/command-readiness-drift -m "..."
git branch -d feature/command-readiness-drift
```

Commit message summarizes the refactor + references spec and follow-up memory.

---

## Self-Review

**Spec coverage:**
- §"CommandIsReady signature change" → Task 4 Step 1, 3. ✓
- §"Expanded CommandIsReady gates (IsCrafting universal + rally/warcry HasBuff)" → Task 4 Step 1-2. ✓
- §"Universal IsCrafting gate in all seven Execute* functions" → Tasks 1-2. ✓
- §"New AlreadyActive early-return in ExecuteRally / ExecuteWarcry" → Task 1. ✓
- §"User-command early IsCrafting reject" → Task 3. ✓
- §"Archetype YAML simplification" → Task 5. ✓
- §"Drift-detection test" → Task 6. ✓

**Placeholder scan:**
- No "TBD", "TODO", "similar to Task N".
- Task 6's StartCrafting caveat is an INVESTIGATION step with specific grep, not a placeholder — it gives the implementer a concrete action to take if the assumption is wrong.
- Task 4 Step 3's "simpler alternative" is a clear branch with a check-before-choosing guard.

**Type consistency:**
- `CommandIsReady(actor Actor, cmd string) bool` consistent across Tasks 4, 6, 8.
- `Crafting bool`, `AlreadyActive bool` result-struct fields consistent in Tasks 1, 2, 6.
- `MobActor{Mob: m, Room: nil}` pattern used consistently for test constructions.
- Buff ids: 80 (rally), 79 (warcry) consistent throughout.

No issues.
