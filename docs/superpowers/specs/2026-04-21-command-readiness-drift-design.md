# Command Readiness Drift Refactor — Design

**Date:** 2026-04-21
**Status:** Approved
**Related memory:** `project_command_readiness_drift.md` (user's next-session commitment from the archetypes work)
**Companion specs:** `2026-04-21-tank-and-generic-archetypes-design.md` (the landing that surfaced this follow-up)

## Goal

Make `actions.CommandIsReady` the single source of truth for combat-command gating, with drift-detection tests ensuring it stays in sync with each `Execute*` function's own early-return gates. Close the universal `IsCrafting` hole across all seven combat commands in the process.

## Problem statement

After the 2026-04-21 archetype work, we have two parallel code paths doing nearly-identical gate checks:

1. **`actions.CommandIsReady(mob, cmd)`** — non-mutating peek used by the btree's `command_best_of` action for self-gating.
2. **Each `Execute*` function** — rich return struct (e.g., `BashResult{NoShield, OnCooldown, ...}`) used by player/mob command wrappers to drive display text.

The gates themselves are duplicated. If a future `Execute*` gains a new gate (e.g., a new condition for `grapple`), `CommandIsReady` silently diverges — the btree will issue the command, the Execute* will reject it, and the tree's selector thinks Success was achieved. The same class of bug already bit us once this session (taunt not registered as a mob command, caught only by play-testing).

Additional inconsistencies this refactor closes:

- **`IsCrafting` gate exists on `rally` and `warcry` only**, and is player-only (`actor.IsPlayer() && char.IsCrafting()`). Mobs mid-craft (future crafter archetype) aren't blocked from combat moves, and players mid-craft can bash/trip/grapple/kick/taunt their way out of their craft state today.
- **Duplicate-buff skip for rally/warcry** is implemented in the archetype YAML via `decorator invert + mob_has_buff`. Works, but the tree author has to remember it — and a future archetype that forgets the decorator will spam rally on every cooldown cycle.

Out-of-scope (confirmed during brainstorm):

- **Resource-cost gates.** Investigation showed no `Execute*` gates on stamina/conviction today — commands scale damage via `combat.ResourceMultiplier` but always execute. No real "resource-starved no-op" to prevent.
- **Broader active-command crafting audit** (cast / mutations / eat / drink) — stays tracked in `project_active_command_crafting_audit.md` as separate future work.

## Design

### Rule

> **`CommandIsReady(actor, cmd)` returns true iff the named command's `Execute*` would proceed past its early-return gates.** Drift-detection tests prove the two stay consistent.

### Changes

#### 1. `CommandIsReady` signature change

```go
// Before
func CommandIsReady(mob *mobs.Mob, cmd string) bool

// After
func CommandIsReady(actor Actor, cmd string) bool
```

Aligns with the rest of the `actions` package (every `Execute*` takes `Actor`). Opens future player-side self-gating. One call-site change in `actCommandBestOf` (wrap mob in `MobActor`).

#### 2. Expanded `CommandIsReady` gates

Current switch stays structurally similar. New universal top-level checks BEFORE the per-command switch:

- `if actor.GetCharacter().IsCrafting() { return false }` — universal.
- Shared special-move cooldown — unchanged.

Per-command additions:

- **`rally`**: also check `!char.HasBuff(80)` (skip if rally already active).
- **`warcry`**: also check `!char.HasBuff(79)` (skip if warcry already active).

Existing gates unchanged: aggro for commands that need a target; target position for trip/grapple; shield (via `HasShield()`) for bash.

#### 3. Universal `IsCrafting` gate in all seven `Execute*` functions

- **`ExecuteBash`**, **`ExecuteTrip`**, **`ExecuteGrapple`**, **`ExecuteKick`**, **`ExecuteTaunt`** — **add** a `Crafting bool` field to their result struct and gate at top.
- **`ExecuteRally`**, **`ExecuteWarcry`** — **flip** existing `actor.IsPlayer() && char.IsCrafting()` to just `char.IsCrafting()`. Result struct `Crafting` field already exists.

All seven now start with:

```go
char := actor.GetCharacter()
if char.IsCrafting() {
    return XResult{Crafting: true}
}
```

The comment on rally/warcry's existing gate (`"IsCrafting applies to players only; mobs never craft"`) is updated to reflect the new policy.

#### 4. New `AlreadyActive` early-return in `ExecuteRally` / `ExecuteWarcry`

Before the cooldown check:

```go
if char.HasBuff(80) {  // 79 for warcry
    return RallyResult{AlreadyActive: true}
}
```

Add `AlreadyActive bool` to `RallyResult` / `WarcryResult`. User commands map it to an appropriate message (or silently no-op — `"You're already rallied."` is fine).

#### 5. User-command early IsCrafting reject

Each of the five user commands that newly-gain the IsCrafting gate adds a pre-Execute* check to produce a nicer error before target resolution runs:

```go
// internal/usercommands/bash.go (and trip, grapple, kick, taunt)
if user.Character.IsCrafting() {
    user.SendText(`<ansi fg="red">You can't bash while focused on your work. Finish or be interrupted first.</ansi>`)
    return true, nil
}
```

Pattern matches the existing rally/warcry user-command error path. Text substituted per command.

#### 6. Archetype YAML simplification

In `_datafiles/world/dogmud/behaviors/archetypes/tank_taunter.yaml`, remove the two `decorator invert + mob_has_buff` blocks around the rally and warcry branches. The tree becomes:

```yaml
# Before
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

# After
- type: action
  do: command_best_of
  cmds: [rally]
```

Same for warcry (buff 79). CommandIsReady is now authoritative.

### Drift-detection test

New file `internal/actions/command_readiness_drift_test.go`. Table-driven, fresh actor per case. Per the brainstorm, ~35-40 cases total covering:

**Per-command happy path** (7 cases):
- Fresh actor, aggro set where required, no cooldown, not crafting, no conflicting buffs → both `CommandIsReady` and `Execute*` say "ready" (Execute* actually runs; assert no gate-failure flag on its result).

**Per-command per-gate failure** (~28 cases):
- Each command × each gate that applies to it:
  - IsCrafting → both say not-ready (Execute* returns `Crafting: true`)
  - Cooldown → both say not-ready (Execute* returns `OnCooldown: true`)
  - No aggro (for commands that need it) → both say not-ready (Execute* returns its no-aggro equivalent; for most this is silent Executed:false)
  - Target in wrong position (trip/grapple) → both say not-ready
  - No shield (bash without NaturalBash) → both say not-ready (Execute* returns `NoShield: true`)
  - Rally buff already active → both say not-ready (Execute* returns `AlreadyActive: true`)
  - Warcry buff already active → same

Test shape (illustrative):

```go
type driftCase struct {
    name        string
    cmd         string
    mutate      func(*mobs.Mob) // state setup
    wantReady   bool
    wantReason  string          // Execute*-side: which flag should be true
}

func TestCommandReadinessDrift(t *testing.T) {
    cases := []driftCase{
        {"bash_ready", "bash", func(m *mobs.Mob) { /* naturalbash species, aggro */ }, true, ""},
        {"bash_crafting", "bash", func(m *mobs.Mob) { /* ...set IsCrafting */ }, false, "Crafting"},
        {"bash_noshield", "bash", func(m *mobs.Mob) { /* ...human species, no shield */ }, false, "NoShield"},
        // ... ~4-5 cases per command
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            mob := newTestMob(t, tc.mutate)
            actor := &actions.MobActor{Mob: mob, Room: nil}

            gotReady := actions.CommandIsReady(actor, tc.cmd)
            require.Equal(t, tc.wantReady, gotReady,
                "CommandIsReady disagreed with expectation for %s", tc.name)

            // Now run Execute* and verify the failure reason (when not ready).
            // Each case knows its command, so we dispatch via a small switch or map:
            if !tc.wantReady {
                result := runExecute(tc.cmd, actor)
                assert.True(t, getFlag(result, tc.wantReason),
                    "Execute* did not return %s: true for %s", tc.wantReason, tc.name)
            }
        })
    }
}
```

Helper `runExecute(cmd, actor)` is a small dispatch function mapping command name to the relevant `Execute*` call. Helper `getFlag(result, name)` uses reflection (or a small switch) to pull the named bool field from the returned struct.

State mutators (`newTestMob` with cfg func) already exist from T3.

### Architecture diagram

```
┌─────────────────────────────────────────────┐
│            Actor (player or mob)            │
└──────────────┬──────────────────────────────┘
               │
       ┌───────┴──────────┐
       ▼                  ▼
  CommandIsReady      Execute*  (ExecuteBash, ExecuteTaunt, ...)
  (peek, bool)        (mutate, rich result)
       │                  │
       └──── drift test ──┘
       (both agree for the same actor state)
```

`Execute*` remains the authoritative mutator. `CommandIsReady` remains the non-mutating peek. The drift test is the contract between them.

## Edge cases accepted

- **Execute* and CommandIsReady are still two files of gate logic.** The refactor doesn't unify them (that would require larger API surgery per the Option A we rejected). The drift test is the guardrail, not code-level consolidation.
- **User-command wrappers still duplicate the IsCrafting check** (once at the top for early error, once inside Execute* as safety net). Acceptable — the user-facing error message differs per command, and having the safety net inside Execute* means mob commands and future callers are also protected without each one needing its own reject.
- **`project_active_command_crafting_audit.md` still open** for cast / mutations / eat / drink. This refactor closes the seven combat commands, not the broader audit.

## Testing

### Unit tests (Go)

- **`command_readiness_drift_test.go`** — ~35-40 table-driven drift cases (per above).
- **Existing `TestCommandIsReadyNamesAreMobCommands`** — still passes (names unchanged, now takes Actor).
- **Existing `TestCommandIsReady_*` in `command_readiness_test.go`** — migrate to Actor-based call, add cases for new gates (IsCrafting, rally-buff-active, warcry-buff-active).

### Smoke tests (server)

1. Log in, start crafting, try `bash <someone>` — should reject with "focused on your work" text (was: worked fine before).
2. Summon a magma elemental (tank_taunter), enter combat, confirm rally fires once then doesn't re-fire until the rally buff expires (was: re-fired every ready round in absence of YAML gate — now gated by CommandIsReady).
3. Same test for warcry.
4. Confirm tank archetype combat rhythm still feels right (rally once, warcry once, taunts, knockdowns in rotation).

## Out of scope

- **Consolidating gate logic into single-source `BashReadiness(actor)` per-command helpers.** (Option A from brainstorm.) Drift test is sufficient; this would be a larger refactor that doesn't improve observable behavior.
- **Resource-cost gating.** Not a real gap today.
- **Active-command audit for cast / mutations / eat / drink.** Separate project.
- **Removing the IsCrafting check from ExecuteRally/ExecuteWarcry** (instead of flipping player-only to universal). The gate stays in both the universal-gated Execute* and the CommandIsReady peek — redundant but matches the rest of the seven-command pattern.
