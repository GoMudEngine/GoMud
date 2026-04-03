# Command Unification — Substage 4: Magic System Parity

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify the magic system's duplicated code paths. Mobs already use fold-based casting — the work is merging the parallel resolution dispatchers and fold-casting handlers, and aligning the cast initiator logic.

**Architecture:** The fold accumulation engine, CastingState, CalcFoldsPerRound, concentration breaks, and shared helpers are already unified. The duplication is in three layers: (1) the cast command initiator, (2) the combat loop fold handlers, (3) the spell resolution dispatchers. We merge each layer using the Actor pattern.

**Tech Stack:** Go, testify/assert, existing spell/combat systems

---

## Current State (already shared)

| Component | Status |
|-----------|--------|
| CastingState struct | Shared on characters.Character |
| CalcFoldsPerRound | Shared function |
| NextPowerOfTwo | Shared function |
| simulateFoldRound / advanceFolds | Shared in combat_shared_helpers.go |
| checkConcentrationBreak | Shared in combat_shared_helpers.go |
| calcFoldConvictionCost | Shared in combat_shared_helpers.go |

## What Needs Unification

| Component | Player | Mob | Divergence |
|-----------|--------|-----|-----------|
| Cast initiator | skill.cast.go (initiation roll, conviction pre-check) | cast.go (no initiation roll, no conviction check) | Mob skips two checks |
| Fold handler | handlePlayerFoldCasting | handleMobFoldCasting | Nearly identical, different resolve call |
| Spell resolution | resolveSpell | resolveMobSpell | Parallel dispatchers, different target helpers |

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/actions/cast.go` | Shared cast initiation logic |
| `internal/actions/cast_test.go` | Tests for shared cast logic |
| `internal/usercommands/skill.cast.go` | Thin wrapper (modify) |
| `internal/mobcommands/cast.go` | Thin wrapper (modify) |
| `internal/hooks/spell_resolution.go` | Merge resolve dispatchers (modify) |
| `internal/hooks/NewRound_DoCombat_helpers.go` | Merge fold handlers (modify) |

---

### Task 1: Shared Cast Initiation

**Files:**
- Create: `internal/actions/cast.go`
- Modify: `internal/usercommands/skill.cast.go`
- Modify: `internal/mobcommands/cast.go`

The cast initiator has significant shared logic: spell lookup, target
resolution, fold calculation, CastingState setup, conviction cost
computation, cooldown check.

- [ ] **Step 1: Read both cast files fully**

Read `internal/usercommands/skill.cast.go` and `internal/mobcommands/cast.go`
completely. Map out the shared vs divergent logic.

- [ ] **Step 2: Create shared cast initiation**

Create `internal/actions/cast.go` with:

```go
type CastResult struct {
    SpellInfo    *spells.SpellData
    CastingState *characters.CastingState
    Initiated    bool
    // Early exit reasons
    AlreadyCasting bool
    SpellNotKnown  bool
    OnCooldown     bool
    InsufficientConviction bool
    InitiationFailed bool
    InitiationCooldown int // rounds of cooldown on failure
    NoTarget     bool
    TargetUserIds        []int
    TargetMobInstanceIds []int
}

func InitiateCast(actor Actor, spellName string, rest string) CastResult
```

InitiateCast should handle the SHARED logic:
1. Look up spell by name/alias
2. Validate spell is known (`char.HasSpell`)
3. Check not already casting (`char.CastingState != nil`)
4. Check special-move cooldown
5. Compute foldsNeeded and foldsPerRound
6. Resolve targets based on spell type
7. Compute total conviction cost
8. Create and return the CastingState (but don't set it on the character — let the wrapper do that after any wrapper-specific checks)

**What stays in user wrapper:**
- Initiation roll (`CalcInitiationChance`) — this is a player-skill-gate
  that mobs intentionally skip. Mob spellcasting decisions happen in AI,
  not in the cast command.
- Upfront conviction sufficiency check (mob skips this)
- Conviction cost multiplier from Magical Resistance mutation
- The Eye mutation perception modulation for foldsPerRound
- `onCast` spell script execution
- Skill progression event
- Player messaging

**What stays in mob wrapper:**
- First-round conviction deduction (mob pays immediately, player pays per-round in combat loop)
- `onCast` spell script execution
- Immediate aggro setting for offensive spells
- Room messaging (darkness-aware)

- [ ] **Step 3: Update user cast wrapper**

Refactor to use `actions.InitiateCast()` for shared logic, then handle
user-specific: initiation roll, conviction check, mutation modifiers,
scripts, messaging.

- [ ] **Step 4: Update mob cast wrapper**

Refactor to use `actions.InitiateCast()` for shared logic, then handle
mob-specific: first-round cost, aggro, messaging.

- [ ] **Step 5: Verify build + tests**

Run: `go build ./...`
Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ -count=1`

- [ ] **Step 6: Commit**

```bash
git commit -m "refactor: shared cast initiation with Actor pattern"
```

---

### Task 2: Merge Fold Casting Handlers

**Files:**
- Create: `internal/hooks/combat_fold_casting.go` (or add to existing shared helpers)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go`
- Modify: `internal/hooks/NewRound_DoCombat.go`

`handlePlayerFoldCasting` and `handleMobFoldCasting` are nearly
identical. Merge them into a single function that operates on
`*characters.Character` (which both players and mobs have).

- [ ] **Step 1: Read both fold handlers fully**

Read `handlePlayerFoldCasting` and `handleMobFoldCasting` in
`NewRound_DoCombat_helpers.go`. Document every difference.

Expected differences:
- Player: sends personal messages to user (`user.SendText`)
- Mob: sends room messages (`room.SendText`)
- Player: calls `resolveSpell(user, ...)`
- Mob: calls `resolveMobSpell(mob, ...)`
- Player: fires SkillUsed + StatUsed events
- Mob: no skill/stat progression events

- [ ] **Step 2: Extract shared fold processing**

Create a shared function that handles the mechanical fold processing:

```go
type FoldRoundResult struct {
    StillCasting    bool
    CastComplete    bool
    ConcentrationBroke bool
    InsufficientConviction bool
    TargetGone      bool
    FoldDelta       int
    ConvictionCost  int
    SpellData       *spells.SpellData
}

func processFoldRound(char *characters.Character) FoldRoundResult
```

This handles: prone/bleedout break, target-gone check, simulateFoldRound,
conviction cost/deduction, advanceFolds. Returns result for the caller
to handle messaging and resolution.

The caller (`handlePlayerFoldCasting` / `handleMobFoldCasting`) becomes
thin: call processFoldRound, then handle messaging and resolution dispatch
based on actor type.

- [ ] **Step 3: Update both handlers**

Both handlers call `processFoldRound`, then:
- Player handler: user messaging, `resolveSpell`, skill/stat events
- Mob handler: room messaging, `resolveMobSpell`

- [ ] **Step 4: Verify build + tests**

- [ ] **Step 5: Commit**

```bash
git commit -m "refactor: merge fold casting handlers into shared processFoldRound"
```

---

### Task 3: Merge Spell Resolution Dispatchers

**Files:**
- Modify: `internal/hooks/spell_resolution.go`

`resolveSpell` and `resolveMobSpell` are parallel dispatchers that
loop over targets and call per-target helpers. The per-target helpers
(`resolveAgainstMob`, `resolveMobSpellAgainstPlayer`, etc.) have more
divergence and should stay separate for now.

- [ ] **Step 1: Read spell_resolution.go fully**

Read both `resolveSpell` and `resolveMobSpell`. Document the shared
dispatch logic vs the divergent per-target helpers.

- [ ] **Step 2: Extract shared dispatch logic**

If the two dispatchers are structurally identical (loop over targets,
dispatch by target type), extract the loop into a shared function that
takes a callback for per-target resolution. If they diverge too much,
leave them separate and note why.

This task is exploratory — the per-target helpers have significant
differences (player resolution fires onHit scripts, mob resolution
uses different damage functions). If merging the dispatchers adds
complexity without reducing duplication, skip it and document why.

- [ ] **Step 3: Verify build + tests**

- [ ] **Step 4: Commit**

```bash
git commit -m "refactor: merge spell resolution dispatchers (or document why not)"
```

---

### Task 4: Cast Parity Tests

**Files:**
- Create: `internal/actions/cast_test.go`

- [ ] **Step 1: Write tests**

Tests for the shared InitiateCast:
1. Unknown spell → SpellNotKnown=true
2. Already casting → AlreadyCasting=true
3. On cooldown → OnCooldown=true
4. Successful initiation → CastingState populated correctly
5. FoldsNeeded is NextPowerOfTwo of BaseFolds
6. FoldsPerRound matches CalcFoldsPerRound output

Note: testing with real spells may require seeding the spell registry.
Check `internal/spells/` for a `SeedSpellsForTest` function.

- [ ] **Step 2: Run tests**

- [ ] **Step 3: Commit**

```bash
git commit -m "test: cast initiation parity tests"
```

---

### Task 5: Final Verification

- [ ] **Step 1: Full build + all tests**

Run: `go build ./...`
Run: `go test ./... -count=1 -timeout 180s`

- [ ] **Step 2: Manual smoke test**

Test in-game:
- Player casts a spell (fold accumulation, resolution)
- Mob casts a spell (fold accumulation, resolution)
- Interrupt a mob's spell with damage during casting
- Interrupt player's spell with mob damage during casting
- Spell resolution hits correct targets

- [ ] **Step 3: Commit any fixups**
