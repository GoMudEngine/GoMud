# Mob Aliveness 2.7 — Mob Skullduggery Suite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Lift the four remaining skullduggery verbs (`steal`, `plant`, `defuse`, `shadow`) into the `internal/actions/` package alongside the existing `Sneak` so both players and mobs share one code path. Add five btree action primitives (`try_steal`, `try_sneak`, `try_plant`, `try_defuse`, `try_shadow`), three state-query conditions (`mob_is_hidden`, `target_is_hidden`, `target_has_gold`), one helper target-picker (`target_random_player_in_room`), one new archetype (`thief`), and flip the Thornwall highwayman to demo the chunk. Sunset three pieces of legacy code that the consolidation makes redundant.

**Architecture:** Each verb lifts into `internal/actions/<verb>.go` exposing `<Verb>(actor Actor, opts <Verb>Options) <Verb>Result`. Player wrappers in `internal/usercommands/skill.skullduggery.*.go` collapse to ~25 lines (parse CLI, call action, format result). Mob wrappers in `internal/mobcommands/<verb>.go` are similar (~20 lines). Btree primitives live in two new files (`actions_skullduggery.go`, `conditions_skullduggery.go`). Existing `actions.ExecuteSneak` renames to `actions.Sneak` for naming parity with `Buy`/`Consider`.

**Tech Stack:** Go 1.21+, existing `internal/actions` actor abstraction (chunks 2.1, 2.4), existing `internal/dice.OpposedRollStat`, existing `internal/buffs` Hidden flag (buff 9), existing crime/knowledge substrate (chunks 1.3, 1.4).

**Spec:** `docs/superpowers/specs/2026-05-13-mob-aliveness-2.7-skullduggery-suite-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/actions/sneak.go` | RENAME | `ExecuteSneak` → `Sneak`; signature unchanged |
| `internal/actions/sneak_test.go` | MODIFY | Update test calls to new name |
| `internal/actions/steal.go` | NEW | `Steal(actor, opts) StealResult` lifted from usercommands |
| `internal/actions/steal_test.go` | NEW | Unit tests (success/fail/cooldown/no-observer) |
| `internal/actions/plant.go` | NEW | `Plant(actor, opts) PlantResult` |
| `internal/actions/plant_test.go` | NEW | Unit tests |
| `internal/actions/defuse.go` | NEW | `Defuse(actor, opts) DefuseResult` |
| `internal/actions/defuse_test.go` | NEW | Unit tests |
| `internal/actions/shadow.go` | NEW | `Shadow(actor, opts) ShadowResult` |
| `internal/actions/shadow_test.go` | NEW | Unit tests |
| `internal/actions/context.md` | MODIFY | Document new actions |
| `internal/usercommands/skill.skullduggery.steal.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.plant.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.defuse.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.shadow.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.skullduggery.sneak.go` | TOUCH | Repoint to `actions.Sneak` |
| `internal/usercommands/stealth_detection.go` | DELETE | Redundant cross-package shim |
| `internal/usercommands/usercommands_test.go` | MODIFY | Remove pickpocket placeholder |
| `internal/mobcommands/steal.go` | NEW | Thin mob wrapper |
| `internal/mobcommands/plant.go` | NEW | Thin mob wrapper |
| `internal/mobcommands/defuse.go` | NEW | Thin mob wrapper |
| `internal/mobcommands/shadow.go` | NEW | Thin mob wrapper |
| `internal/mobcommands/sneak.go` | TOUCH | Repoint to `actions.Sneak` |
| `internal/mobcommands/mobcommands.go` | MODIFY | Register four new commands |
| `internal/mobcommands/mobcommands_test.go` | MODIFY | Add to parity list |
| `internal/behaviortree/actions_skullduggery.go` | NEW | Five `try_<verb>` actions |
| `internal/behaviortree/actions_target.go` | MODIFY | Add `actTargetRandomPlayerInRoom` |
| `internal/behaviortree/conditions_skullduggery.go` | NEW | Three state-query conditions |
| `internal/behaviortree/actions.go` | MODIFY | Register six new actions |
| `internal/behaviortree/conditions.go` | MODIFY | Register three new conditions |
| `internal/behaviortree/context.md` | MODIFY | Document new primitives |
| `internal/hooks/go.go` | MODIFY | Collapse triple-removal at lines 445-447 |
| `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml` | NEW | Thief archetype tree |
| `_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml` | MODIFY | `behavior_archetype: thief` |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.7 Done, roll-up 15/41 |

If `internal/behaviortree/actions_target.go` doesn't exist as a file, add `actTargetRandomPlayerInRoom` into the file that already houses `actTargetWeakestMobInRoom` (chunk 2.4 lives in `actions_combat.go`).

---

## Task 1: Rename `ExecuteSneak` → `Sneak`

**Files:**
- Modify: `internal/actions/sneak.go`
- Modify: `internal/actions/sneak_test.go`
- Modify: `internal/usercommands/skill.skullduggery.sneak.go`
- Modify: `internal/mobcommands/sneak.go`

Mechanical rename for naming parity with `Buy`, `Consider`. No behavior change.

- [ ] **Step 1: Rename the function**

In `internal/actions/sneak.go` line 44, change:
```go
func ExecuteSneak(actor Actor) SneakResult {
```
to:
```go
func Sneak(actor Actor) SneakResult {
```

- [ ] **Step 2: Update all callers**

Run `grep -rn "actions.ExecuteSneak\|ExecuteSneak(" --include="*.go" .` and update every match to `actions.Sneak`/`Sneak(`. Confirmed callers from survey:
- `internal/usercommands/skill.skullduggery.sneak.go` (around line 79)
- `internal/mobcommands/sneak.go` (around line 9)
- `internal/actions/sneak_test.go` (all test function bodies)

- [ ] **Step 3: Run sneak tests to verify rename is clean**

```bash
go test ./internal/actions/ -run TestSneak -v
```

Expected: PASS — all existing sneak tests still pass after the rename.

- [ ] **Step 4: Build verification**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/sneak.go internal/actions/sneak_test.go internal/usercommands/skill.skullduggery.sneak.go internal/mobcommands/sneak.go
git commit -m "refactor(actions): rename ExecuteSneak to Sneak for naming parity"
```

---

## Task 2: Lift `actions.Steal`

**Files:**
- Create: `internal/actions/steal.go`
- Create: `internal/actions/steal_test.go`

This is the largest lift (~280 LoC from `internal/usercommands/skill.skullduggery.steal.go`). The source file has three functions:
- `Steal(...)` — entry: arg parsing, cooldown gate, dispatch to mob/container path.
- `stealFromMob(mobInstanceId, attackerScore, rank, room, user)` — mob-target path.
- `stealFromContainer(containerName, attackerScore, rank, room, user)` — container path.

The lift maps these to:
- `actions.Steal(actor, opts) StealResult` — entry; replaces arg parsing with structured opts.
- `actions.stealFromMob(actor, mobInstanceId, attackerScore, rank) StealResult` — unexported helper.
- `actions.stealFromContainer(actor, containerName, attackerScore, rank) StealResult` — unexported helper.

User-specific calls that must be translated:
- `user.SendText(...)` → `actor.SendText(...)`.
- `user.Character` → `actor.GetCharacter()`.
- `user.UserId` → `actor.GetUserId()` (returns 0 for mobs).
- `user.Character.Aggro` → `actor.GetCharacter().Aggro`.
- `user.Character.TryCooldown(...)` → `actor.GetCharacter().TryCooldown(...)`.
- `user.Character.OnSkillUse(...)` → `actor.OnSkillUse(...)`.
- Quest engine notifications (`quests.NotifyCommand`) — gate on `actor.IsPlayer()` per chunk 2.1 precedent.
- Existing `crimes.RecordTheft(...)`, `factions.BumpRep(...)`, `knowledge.RecordWitnessedCrime(...)` — keep as-is; substrate is actor-agnostic.

- [ ] **Step 1: Write the StealOptions / StealResult types + the public signature stub**

Create `internal/actions/steal.go`:

```go
package actions

// StealOptions parameterizes a theft attempt.
// Exactly one of TargetMobInstanceId / TargetUserId / ContainerNoun
// must be set. ItemNoun narrows the steal to a specific item;
// when empty, the action defaults to gold-or-random-item per the
// existing player-side logic.
type StealOptions struct {
	TargetMobInstanceId int
	TargetUserId        int
	ContainerNoun       string
	ItemNoun            string
}

// StealResult is the structured outcome of a steal attempt.
type StealResult struct {
	Succeeded     bool   // skill check passed and transfer happened
	Detected      bool   // detection roll fired and the defender noticed
	StoleGold     int    // gold transferred (0 if item-only or failed)
	StoleItemId   int    // item id transferred (0 if gold-only or failed)
	StoleItemName string // for messaging
	DefenderName  string // who/what was robbed
	OnCooldown    bool   // attempt was blocked by skullduggery cooldown
	Reason        string // when Succeeded==false and !OnCooldown, why
}

// Steal runs a skullduggery theft attempt from actor against the
// resolved target. Both UserActor and MobActor are supported. On
// success the transfer happens before return. The cooldown is set
// on the actor's command map whenever a roll actually happened.
func Steal(actor Actor, opts StealOptions) StealResult {
	// IMPLEMENTATION FOLLOWS IN STEP 3
	return StealResult{Reason: "not yet implemented"}
}
```

- [ ] **Step 2: Write failing tests in `internal/actions/steal_test.go`**

Mirror the pattern in `internal/actions/sneak_test.go` for room/user/mob construction. Tests:

```go
package actions

import (
	"testing"

	// + whatever fixtures sneak_test.go uses
)

// TestSteal_MobTarget_GoldSuccess verifies the success path:
// attacker outclasses defender, gold transfers from mob to player.
func TestSteal_MobTarget_GoldSuccess(t *testing.T) {
	// Setup: room with a player and a mob, mob has 50 gold,
	// player has dex 80 + skullduggery rank 8, mob has perception 5.
	// Roll math should heavily favor the attacker.
	// Expect: result.Succeeded == true, result.StoleGold > 0,
	// mob.Character.Gold reduced, player.Character.Gold increased.
}

// TestSteal_OnCooldown verifies a second attempt within the
// cooldown window returns OnCooldown: true without rolling.
func TestSteal_OnCooldown(t *testing.T) {
	// Setup: player invokes Steal once successfully, then again.
	// Expect: second result.OnCooldown == true,
	// and the mob's gold is unchanged from the first call.
}

// TestSteal_NoTarget verifies missing target returns a clean
// failure with a Reason explaining "no target".
func TestSteal_NoTarget(t *testing.T) {
	// Expect: Succeeded == false, OnCooldown == false,
	// Reason mentions "no target" or "not found".
}

// TestSteal_MobOnPlayer verifies the mob-side path: a mob with
// high skullduggery successfully pickpockets a player. The
// detection roll independently determines whether the player
// sees a message — but the transfer still happens.
func TestSteal_MobOnPlayer(t *testing.T) {
	// Setup: MobActor with high skullduggery, UserActor target
	// with mid perception.
	// Force the detection roll outcome via dice seeding if
	// supported, or accept either Detected value but verify
	// transfer happened.
}
```

Run the tests to verify they fail (`Steal` returns the stub):
```bash
go test ./internal/actions/ -run TestSteal -v
```
Expected: FAIL with "not yet implemented" / "result.Succeeded = false, want true".

- [ ] **Step 3: Implement `Steal` by lifting from `usercommands/skill.skullduggery.steal.go`**

Read the existing file end-to-end. Port:

1. **Arg parsing** stays in the wrapper; the action takes structured `StealOptions`. So delete the `args := util.SplitButRespectQuotes(...)` block and the "Steal from whom?" branch — those go to the caller.

2. **Combat gate** stays:
   ```go
   if actor.GetCharacter().Aggro != nil {
       return StealResult{Reason: "in combat"}
   }
   ```

3. **Under-attack gate** stays:
   ```go
   if actor.GetRoom().AreMobsAttacking(actor.GetUserId()) {
       return StealResult{Reason: "under attack"}
   }
   ```
   For mobs `GetUserId()` returns 0; `AreMobsAttacking(0)` returns false, so this branch is a no-op for mob actors — desired.

4. **Cooldown gate:**
   ```go
   cfg := configs.GetBalanceConfig()
   cooldownKey := skills.Skullduggery.String("steal")
   if !actor.GetCharacter().TryCooldown(cooldownKey,
       fmt.Sprintf("%d real seconds", int(cfg.StealCooldown))) {
       return StealResult{OnCooldown: true,
           Reason: fmt.Sprintf("%d rounds remaining",
               actor.GetCharacter().GetCooldown(cooldownKey))}
   }
   ```

5. **Attacker score:** preserve the existing math verbatim. Replace `user.Character.Stats.Dexterity.ValueAdj` with `actor.GetCharacter().Stats.Dexterity.ValueAdj`. Hidden bonus check: `actor.GetCharacter().HasBuffFlag(buffs.Hidden)`.

6. **Skill-rank gate** inside the steal-from-mob path:
   ```go
   if skillLevel < 2 {
       return StealResult{Reason: "not advanced enough"}
   }
   ```
   The wrapper translates `Reason` to the player-facing message.

7. **Dispatch** to the mob-target or container-target helper based on opts.

8. **`stealFromMob` lift:** preserves the opposed roll, target validation (non-combatant rebuff, immune-mob rebuff), gold-vs-item logic, detection by witnesses, crime recording, faction rep, knowledge witnessing. Replace `user.SendText(...)` with `actor.SendText(...)` and add `actor.GetRoom().SendText(observerUserIds, ...)` for observer broadcasts (see source for exact patterns). Quest engine: gate behind `if actor.IsPlayer() { quests.NotifyCommand(...) }`.

9. **`stealFromContainer` lift:** same translation pattern. Container theft has no faction victim — crime recording only fires when a faction-witness observer is present (preserved logic in source).

10. **Mob-on-player detection** (new behavior, architectural must #3 in spec): when the actor is a mob and the target is a player, run an independent Per+Search-vs-Dex+Skullduggery roll on the victim. On detection-win, emit `target.SendText(...)` via the actor abstraction — the action does not currently know about a `target` Actor directly; convert by:

    ```go
    if !actor.IsPlayer() && targetUser != nil {
        searchScore := CalcSearchScore(&targetUser.Character)
        sneakScore := CalcSneakScore(actor.GetCharacter())
        detected, _, _, _ := dice.OpposedRollStat(searchScore, sneakScore)
        if detected {
            targetUser.SendText(fmt.Sprintf(
                `<ansi fg="mobname">%s</ansi> lifts <ansi fg="gold">%d gold</ansi> from your pocket!`,
                actor.GetName(), result.StoleGold))
            result.Detected = true
        }
    }
    ```

    The `CalcSearchScore`/`CalcSneakScore` helpers already exist in `internal/actions/skill_helpers.go` (per survey).

11. **Cooldown is set when a roll happened** — the existing `TryCooldown` call sets it at gate-time. That's the canonical behavior; preserve.

Run the tests:
```bash
go test ./internal/actions/ -run TestSteal -v
```
Expected: PASS.

- [ ] **Step 4: Run the full actions package tests**

```bash
go test ./internal/actions/ -v
```
Expected: all existing tests still pass plus the new `TestSteal_*` tests.

- [ ] **Step 5: Build verification**

```bash
go build ./...
```
Expected: clean build. (The existing `usercommands/skill.skullduggery.steal.go` still compiles — we haven't rewritten the wrapper yet, but no symbols collide.)

- [ ] **Step 6: Commit**

```bash
git add internal/actions/steal.go internal/actions/steal_test.go
git commit -m "feat(actions): lift Steal into actions package for player/mob parity"
```

---

## Task 3: Rewrite `usercommands` steal wrapper

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.steal.go`

Collapse the file from 387 LoC to ~50 LoC. The wrapper parses CLI args into `StealOptions`, calls `actions.Steal`, and formats the result for player-facing output.

- [ ] **Step 1: Rewrite the file**

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Steal(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(strings.ToLower(rest))
	if len(args) == 0 {
		user.SendText("Steal from whom?")
		return true, nil
	}

	opts := parseStealArgs(args, room)

	actor := &actions.UserActor{User: user, Room: room}
	result := actions.Steal(actor, opts)

	formatStealResultForPlayer(user, result, args)
	return true, nil
}

// parseStealArgs maps CLI args to StealOptions. Args are
// already lowercased. Accepts `steal <mob>`, `steal from <mob>`,
// `steal gold from <mob>`, `steal <item> from <mob/container>`.
func parseStealArgs(args []string, room *rooms.Room) actions.StealOptions {
	// implementation — replicate the existing arg-parsing logic
	// from the pre-lift Steal func lines 54-101. Should resolve
	// the target by name (mob or container) and set the
	// appropriate field on StealOptions. Returns zero-value
	// opts if target not found; the action returns a clean
	// "no target" Reason that the formatter handles.
	return actions.StealOptions{}
}

// formatStealResultForPlayer emits the player-facing text based
// on the structured result. Mirrors the messaging from the
// pre-lift function (success, cooldown, "not advanced enough",
// "in combat", "you don't see them here").
func formatStealResultForPlayer(user *users.UserRecord, result actions.StealResult, args []string) {
	switch {
	case result.OnCooldown:
		user.SendText(fmt.Sprintf("You need to wait %s before you can do that again.", result.Reason))
	case result.Reason == "in combat":
		user.SendText("You can't do that while in combat!")
	case result.Reason == "under attack":
		user.SendText("You can't do that while you are under attack!")
	case result.Reason == "not advanced enough":
		user.SendText("You're not skilled enough at skullduggery to attempt that.")
	case !result.Succeeded:
		// Generic miss / failure case — preserve pre-lift wording.
		user.SendText("Your attempt fails.")
	case result.StoleGold > 0:
		user.SendText(fmt.Sprintf(`You lift <ansi fg="gold">%d gold</ansi> from %s.`,
			result.StoleGold, result.DefenderName))
	case result.StoleItemId > 0:
		user.SendText(fmt.Sprintf("You lift the %s from %s.",
			result.StoleItemName, result.DefenderName))
	}
}
```

**Note for the implementer:** the exact arg-parsing logic in `parseStealArgs` must match the pre-lift surface. Open the original `Steal` function (lines 54-101 of the pre-lift file from the working tree before this task) and port the noun-resolution into `parseStealArgs`. Use `mobs.FindMobInRoom(...)` and `rooms.GetContainer(...)` calls verbatim. Keep "from" prepositional handling.

- [ ] **Step 2: Build verification**

```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 3: Manual integration smoke**

Boot the server (`go run main.go` or use the test target), log in as a character with skullduggery >= 2, and run:
```
steal mob
```
Verify it still produces a recognizable response (success or "your attempt fails"). The exact text is preserved from the pre-lift implementation.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/skill.skullduggery.steal.go
git commit -m "refactor(usercommands): thin Steal wrapper around actions.Steal"
```

---

## Task 4: Lift `actions.Plant`

**Files:**
- Create: `internal/actions/plant.go`
- Create: `internal/actions/plant_test.go`

Same shape as Task 2. The source file `internal/usercommands/skill.skullduggery.plant.go` (283 LoC) has:
- `Plant(...)` — entry.
- `plantOnMob(...)` — mob-target path.
- `plantInContainer(...)` — container-target path.

Plant shares the skullduggery cooldown key (`skills.Skullduggery.String("steal")`) with Steal — preserve.

- [ ] **Step 1: Create `internal/actions/plant.go` with type + signature stubs**

```go
package actions

type PlantOptions struct {
	TargetMobInstanceId int
	TargetUserId        int
	ContainerNoun       string
	ItemNoun            string // required
}

type PlantResult struct {
	Succeeded     bool
	Detected      bool
	PlantedItemId int
	DefenderName  string
	OnCooldown    bool
	Reason        string
}

func Plant(actor Actor, opts PlantOptions) PlantResult {
	return PlantResult{Reason: "not yet implemented"}
}
```

- [ ] **Step 2: Write failing tests in `internal/actions/plant_test.go`**

Mirror Task 2's test shape. Tests:
- `TestPlant_MobTarget_Success` — player plants an item on a mob, opposed roll heavily favors attacker, item transfers.
- `TestPlant_RequiresItem` — `PlantOptions.ItemNoun` empty → returns `Reason: "no item"`, no roll.
- `TestPlant_SharesStealCooldown` — invoking Plant after a successful Steal returns `OnCooldown: true`.

Run them:
```bash
go test ./internal/actions/ -run TestPlant -v
```
Expected: FAIL.

- [ ] **Step 3: Implement Plant by lifting from `usercommands/skill.skullduggery.plant.go`**

Apply the same translation pattern as Task 2:
- Arg parsing stays in the wrapper.
- Combat gate, under-attack gate, cooldown gate all use `actor.GetCharacter()`.
- Cooldown key: `skills.Skullduggery.String("steal")` — same key as Steal (intentional).
- `plantOnMob` / `plantInContainer` become unexported `plantOnMob(actor, ...)` / `plantInContainer(actor, ...)`.
- Crime recording (theft on faction-witness) preserved.
- Quest notifications behind `if actor.IsPlayer()` gate.
- Mob-on-player detection roll: same pattern as Task 2 step 3, point 10.

Run tests:
```bash
go test ./internal/actions/ -run TestPlant -v
```
Expected: PASS.

- [ ] **Step 4: Build verification**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/plant.go internal/actions/plant_test.go
git commit -m "feat(actions): lift Plant into actions package for player/mob parity"
```

---

## Task 5: Rewrite `usercommands` plant wrapper

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.plant.go`

Mirror Task 3's pattern. Wrapper parses args into `PlantOptions`, calls `actions.Plant`, formats result. Plant's CLI is `plant <item> on <mob>` or `plant <item> in <container>`.

- [ ] **Step 1: Rewrite the wrapper**

Apply the Task 3 pattern. Functions: `parseplantArgs(args, user, room) actions.PlantOptions` and `formatPlantResultForPlayer(user, result)`. The arg parser must look up the item in the user's backpack (`user.Character.GetItemFromBackpack(args[0])` or similar — verify against the original source) and pass either the item id or a noun via opts. **Decision:** because the action operates on `actor.GetCharacter().Backpack`, pass the item noun (not the resolved item) — the action re-resolves it inside, so mob actors and player actors share the path.

- [ ] **Step 2: Build verification**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/skill.skullduggery.plant.go
git commit -m "refactor(usercommands): thin Plant wrapper around actions.Plant"
```

---

## Task 6: Lift `actions.Defuse`

**Files:**
- Create: `internal/actions/defuse.go`
- Create: `internal/actions/defuse_test.go`

Source: `internal/usercommands/skill.skullduggery.defuse.go` (~192 LoC, single function). Per+Skullduggery vs trap difficulty × 10. Consumes a disarm kit if present.

- [ ] **Step 1: Create `internal/actions/defuse.go` with stubs**

```go
package actions

type DefuseOptions struct {
	TrapNoun string
}

type DefuseResult struct {
	Succeeded      bool
	TrapName       string
	KitConsumed    bool
	KitBonusUsed   int
	TriggeredTraps []int
	Reason         string
}

func Defuse(actor Actor, opts DefuseOptions) DefuseResult {
	return DefuseResult{Reason: "not yet implemented"}
}
```

- [ ] **Step 2: Write failing tests**

`internal/actions/defuse_test.go`:
- `TestDefuse_Success` — high-skill actor, low-difficulty trap, expect `Succeeded: true`, trap removed from room's `TrapBuffIds`.
- `TestDefuse_FailureTriggers` — low-skill actor, expect `Succeeded: false`, `TriggeredTraps` non-empty, buffs applied to actor.
- `TestDefuse_KitBonus` — actor with disarm kit in backpack, expect `KitConsumed: true`, kit gone, bonus applied to roll.
- `TestDefuse_NoTrap` — empty `TrapNoun` and no traps in room, expect `Reason: "no trap"`, no roll.

Run:
```bash
go test ./internal/actions/ -run TestDefuse -v
```
Expected: FAIL.

- [ ] **Step 3: Implement Defuse by lifting**

Port the existing function. Translations:
- `user.SendText` → `actor.SendText`.
- `user.Character.Backpack.FindItem(...)` → `actor.GetCharacter().Backpack.FindItem(...)`.
- Skill progression `OnSkillUse("skullduggery")` via `actor.OnSkillUse(...)`.
- Trap iteration uses `room.TrapBuffIds` / `room.RemoveTrapBuff(...)` — leave as-is.
- Buff application on failure: `actor.GetCharacter().AddBuff(...)` via the event queue (existing path).
- Quest notification gated by `actor.IsPlayer()`.

Run:
```bash
go test ./internal/actions/ -run TestDefuse -v
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/defuse.go internal/actions/defuse_test.go
git commit -m "feat(actions): lift Defuse into actions package for player/mob parity"
```

---

## Task 7: Rewrite `usercommands` defuse wrapper

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.defuse.go`

Mirror Task 3. Wrapper parses `defuse <trap_noun>`, calls `actions.Defuse`, formats result.

- [ ] **Step 1: Rewrite the wrapper**

Apply the Task 3 pattern. Format the result: success message, failure message, "no trap here" message, kit-consumed flavor text.

- [ ] **Step 2: Build & commit**

```bash
go build ./...
git add internal/usercommands/skill.skullduggery.defuse.go
git commit -m "refactor(usercommands): thin Defuse wrapper around actions.Defuse"
```

---

## Task 8: Lift `actions.Shadow`

**Files:**
- Create: `internal/actions/shadow.go`
- Create: `internal/actions/shadow_test.go`

Source: `internal/usercommands/skill.skullduggery.shadow.go` (~176 LoC). Requires hidden state. Stores target in misc-data and auto-follows on target move (existing event-listener wiring; preserve).

- [ ] **Step 1: Create `internal/actions/shadow.go` with stubs**

```go
package actions

type ShadowOptions struct {
	TargetMobInstanceId int
	TargetUserId        int
}

type ShadowResult struct {
	Succeeded  bool
	Detected   bool
	TargetName string
	OnCooldown bool
	Reason     string
}

func Shadow(actor Actor, opts ShadowOptions) ShadowResult {
	return ShadowResult{Reason: "not yet implemented"}
}
```

- [ ] **Step 2: Write failing tests**

`internal/actions/shadow_test.go`:
- `TestShadow_RequiresHidden` — actor without buff 9 → `Reason: "not hidden"`, no roll, no cooldown.
- `TestShadow_Success` — actor hidden, target acquired, misc-data `shadow-target-user` set.
- `TestShadow_Cooldown` — second invocation within `ShadowCooldown` returns `OnCooldown: true`.

Run:
```bash
go test ./internal/actions/ -run TestShadow -v
```
Expected: FAIL.

- [ ] **Step 3: Implement Shadow by lifting**

Port the function. Translations as in Tasks 2/4/6. The `cfg.ShadowCooldown` value is used directly. Detection roll (target wins on `target_per > attacker_dex`) preserved — the message-to-target path goes through `targetUser.SendText` when target is a player.

- [ ] **Step 4: Commit**

```bash
git add internal/actions/shadow.go internal/actions/shadow_test.go
git commit -m "feat(actions): lift Shadow into actions package for player/mob parity"
```

---

## Task 9: Rewrite `usercommands` shadow wrapper

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.shadow.go`

- [ ] **Step 1: Rewrite using Task 3 pattern.**

- [ ] **Step 2: Build & commit**

```bash
go build ./...
git add internal/usercommands/skill.skullduggery.shadow.go
git commit -m "refactor(usercommands): thin Shadow wrapper around actions.Shadow"
```

---

## Task 10: Mob wrappers + registration

**Files:**
- Create: `internal/mobcommands/steal.go`
- Create: `internal/mobcommands/plant.go`
- Create: `internal/mobcommands/defuse.go`
- Create: `internal/mobcommands/shadow.go`
- Modify: `internal/mobcommands/mobcommands.go`
- Modify: `internal/mobcommands/mobcommands_test.go`

Each mob wrapper is ~20 LoC and mirrors `internal/mobcommands/consider.go` (created in chunk 2.4) / `internal/mobcommands/sneak.go`.

- [ ] **Step 1: Create `internal/mobcommands/steal.go`**

```go
package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Steal(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	opts := parseMobStealArgs(strings.ToLower(rest), mob, room)
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Steal(actor, opts)
	return true, nil
}

// parseMobStealArgs resolves a player or mob target by name in
// the current room. Mob calls typically don't pass through CLI;
// the btree primitive `try_steal` invokes Steal directly with
// pre-resolved opts (Task 14). This wrapper exists for parity
// with the player command surface and admin testing.
func parseMobStealArgs(rest string, mob *mobs.Mob, room *rooms.Room) actions.StealOptions {
	if rest == "" {
		return actions.StealOptions{}
	}
	args := util.SplitButRespectQuotes(rest)
	target := args[len(args)-1]
	if userId, _ := room.FindByName(target); userId > 0 {
		return actions.StealOptions{TargetUserId: userId}
	}
	if mobId, _ := room.FindMobByName(target); mobId > 0 {
		return actions.StealOptions{TargetMobInstanceId: mobId}
	}
	return actions.StealOptions{ContainerNoun: target}
}
```

If `room.FindByName` / `room.FindMobByName` aren't the exact API on the room type, substitute the actual lookup helpers (search the codebase for `FindByName` in `internal/rooms/`).

- [ ] **Step 2: Create the other three mob wrappers using the same pattern**

`internal/mobcommands/plant.go` — parses `plant <item> on <target>`, returns `actions.PlantOptions`.

`internal/mobcommands/defuse.go` — parses `defuse <trap>`, returns `actions.DefuseOptions{TrapNoun: rest}`.

`internal/mobcommands/shadow.go` — parses `shadow <target>`, returns `actions.ShadowOptions`.

- [ ] **Step 3: Register in `internal/mobcommands/mobcommands.go`**

Open the file and find the `mobCommands` map declaration (similar to where `consider` was added in chunk 2.4 — search for `"consider":`). Add four entries:

```go
"steal":  Steal,
"plant":  Plant,
"defuse": Defuse,
"shadow": Shadow,
```

Keep alphabetical order if the existing map maintains it.

- [ ] **Step 4: Add to parity test**

`internal/mobcommands/mobcommands_test.go` — find the expected-commands list (created by chunk 2.4 patterns) and add the four new commands.

- [ ] **Step 5: Run mob command tests**

```bash
go test ./internal/mobcommands/ -v
```
Expected: PASS (or update parity test to match if it fails on the registration list).

- [ ] **Step 6: Build verification**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/mobcommands/steal.go internal/mobcommands/plant.go internal/mobcommands/defuse.go internal/mobcommands/shadow.go internal/mobcommands/mobcommands.go internal/mobcommands/mobcommands_test.go
git commit -m "feat(mobcommands): add steal/plant/defuse/shadow mob wrappers"
```

---

## Task 11: Btree action `target_random_player_in_room`

**Files:**
- Modify: `internal/behaviortree/actions_combat.go` (location of `target_weakest_mob_in_room`; same file is the natural home for the new picker)
- Modify: `internal/behaviortree/actions.go` (registration)
- Modify: `internal/behaviortree/actions_combat_test.go` (test)

Sister to `target_weakest_mob_in_room` from chunk 2.4. Walks `room.GetPlayers()`, picks at random, sets the mob's Aggro to that player. Returns Success when a player is picked, Failure when the room is empty of players.

- [ ] **Step 1: Write failing test**

Add to `internal/behaviortree/actions_combat_test.go`:

```go
func TestActTargetRandomPlayerInRoom_PicksAPlayer(t *testing.T) {
	// Setup: room with two players + a mob. Run the action.
	// Expect: Result is Success, mob.Character.Aggro.UserId is
	// one of the two players' UserId values.
}

func TestActTargetRandomPlayerInRoom_EmptyRoom(t *testing.T) {
	// Setup: room with no players, only the mob.
	// Expect: Result is Failure, mob.Character.Aggro is unchanged.
}
```

Run:
```bash
go test ./internal/behaviortree/ -run TestActTargetRandomPlayerInRoom -v
```
Expected: FAIL ("action not registered" or compile error since func doesn't exist).

- [ ] **Step 2: Implement the action**

In `internal/behaviortree/actions_combat.go` (or wherever `actTargetWeakestMobInRoom` lives), add:

```go
// actTargetRandomPlayerInRoom picks a random player in the
// caller's current room and sets them as the caller's aggro
// target. Returns Failure when no players are present.
func actTargetRandomPlayerInRoom(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	playerIds := room.GetPlayers()
	if len(playerIds) == 0 {
		return Failure
	}
	idx := util.Rand(len(playerIds))
	pickedId := playerIds[idx]
	mob.Character.Aggro = &characters.Aggro{UserId: pickedId}
	return Success
}
```

Use existing imports as templates (look at `actTargetWeakestMobInRoom`). `util.Rand` is the codebase's standard RNG.

- [ ] **Step 3: Register in `actions.go` init()**

```go
actionRegistry["target_random_player_in_room"] = actTargetRandomPlayerInRoom
```

- [ ] **Step 4: Verify tests pass**

```bash
go test ./internal/behaviortree/ -run TestActTargetRandomPlayerInRoom -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/actions_combat.go internal/behaviortree/actions.go internal/behaviortree/actions_combat_test.go
git commit -m "feat(btree): add target_random_player_in_room action"
```

---

## Task 12: Btree conditions (`mob_is_hidden`, `target_is_hidden`, `target_has_gold`)

**Files:**
- Create: `internal/behaviortree/conditions_skullduggery.go`
- Create: `internal/behaviortree/conditions_skullduggery_test.go`
- Modify: `internal/behaviortree/conditions.go`

- [ ] **Step 1: Write failing tests**

`internal/behaviortree/conditions_skullduggery_test.go`:

```go
package behaviortree

import "testing"

func TestCondMobIsHidden_TrueWhenBuffPresent(t *testing.T) {
	// Setup: mob with buff 9 applied. Run condition.
	// Expect: Result == Success.
}

func TestCondMobIsHidden_FalseWhenNoBuff(t *testing.T) {
	// Setup: mob without buff 9. Run condition.
	// Expect: Result == Failure.
}

func TestCondTargetIsHidden_TrueWhenTargetBuffPresent(t *testing.T) {
	// Setup: mob with aggro target (player) carrying buff 9.
	// Expect: Result == Success.
}

func TestCondTargetIsHidden_FalseWhenNoTarget(t *testing.T) {
	// Setup: mob with no aggro target.
	// Expect: Result == Failure.
}

func TestCondTargetHasGold_TrueAboveThreshold(t *testing.T) {
	// Setup: mob with aggro player target who has 50 gold.
	// Params: min=10.
	// Expect: Success.
}

func TestCondTargetHasGold_FalseBelowThreshold(t *testing.T) {
	// Setup: target player with 5 gold, params min=10.
	// Expect: Failure.
}
```

Run:
```bash
go test ./internal/behaviortree/ -run "TestCond(MobIsHidden|TargetIsHidden|TargetHasGold)" -v
```
Expected: FAIL (compile error — conditions don't exist).

- [ ] **Step 2: Implement the conditions in `conditions_skullduggery.go`**

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// condMobIsHidden returns Success when the calling mob carries the
// Hidden buff (buff 9).
func condMobIsHidden(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.HasBuffFlag(buffs.Hidden) {
		return Success
	}
	return Failure
}

// condTargetIsHidden returns Success when the resolved target
// (event user → aggro user → aggro mob) carries the Hidden buff.
func condTargetIsHidden(params map[string]any, ctx *EvalContext) Result {
	user, mob := resolveTargetForCondition(ctx)
	if user != nil && user.Character.HasBuffFlag(buffs.Hidden) {
		return Success
	}
	if mob != nil && mob.Character.HasBuffFlag(buffs.Hidden) {
		return Success
	}
	return Failure
}

// condTargetHasGold returns Success when the resolved player
// target has at least `min` gold. Mob targets always Failure
// (mob gold is merchant currency, not steal-able).
func condTargetHasGold(params map[string]any, ctx *EvalContext) Result {
	min := getIntParam(params, "min", 1)
	user, _ := resolveTargetForCondition(ctx)
	if user == nil {
		return Failure
	}
	if user.Character.Gold >= min {
		return Success
	}
	return Failure
}

// resolveTargetForCondition picks a target with the standard
// fallback chain: ctx.Event.UserId → mob.Aggro.UserId →
// mob.Aggro.MobInstanceId. Returns the resolved user or mob.
func resolveTargetForCondition(ctx *EvalContext) (*users.UserRecord, *mobs.Mob) {
	if ctx.Event.UserId > 0 {
		if u := users.GetByUserId(ctx.Event.UserId); u != nil {
			return u, nil
		}
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Aggro == nil {
		return nil, nil
	}
	if mob.Character.Aggro.UserId > 0 {
		return users.GetByUserId(mob.Character.Aggro.UserId), nil
	}
	if mob.Character.Aggro.MobInstanceId > 0 {
		return nil, mobs.GetInstance(mob.Character.Aggro.MobInstanceId)
	}
	return nil, nil
}
```

If `resolveTargetForCondition` already exists in the package (chunk 2.4 added `resolveTargetPower` which uses similar logic), import its target-resolution helper instead of re-implementing. Check `conditions_combat.go` for a reusable helper.

- [ ] **Step 3: Register in `conditions.go` init()**

```go
conditionRegistry["mob_is_hidden"] = condMobIsHidden
conditionRegistry["target_is_hidden"] = condTargetIsHidden
conditionRegistry["target_has_gold"] = condTargetHasGold
```

- [ ] **Step 4: Verify tests pass**

```bash
go test ./internal/behaviortree/ -run "TestCond(MobIsHidden|TargetIsHidden|TargetHasGold)" -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/conditions_skullduggery.go internal/behaviortree/conditions_skullduggery_test.go internal/behaviortree/conditions.go
git commit -m "feat(btree): add skullduggery state-query conditions"
```

---

## Task 13: Btree actions (five `try_<verb>` primitives)

**Files:**
- Create: `internal/behaviortree/actions_skullduggery.go`
- Create: `internal/behaviortree/actions_skullduggery_test.go`
- Modify: `internal/behaviortree/actions.go`

Five thin btree actions that resolve the mob's target and call into the corresponding `actions.<Verb>`. Map result `Succeeded` → btree `Success`, otherwise `Failure`.

- [ ] **Step 1: Write failing tests**

`internal/behaviortree/actions_skullduggery_test.go`:

```go
package behaviortree

import "testing"

func TestActTrySneak_SuccessWhenNoObservers(t *testing.T) {
	// Setup: mob alone in a room. Run try_sneak.
	// Expect: Success, mob has buff 9 after.
}

func TestActTrySteal_SuccessWithAggroTarget(t *testing.T) {
	// Setup: mob with aggro target (high-gold player), high
	// skullduggery, target low perception. Run try_steal.
	// Expect: Success, target gold reduced.
}

func TestActTrySteal_FailureNoTarget(t *testing.T) {
	// Setup: mob with no aggro and no event target.
	// Expect: Failure.
}

func TestActTryPlant_RequiresItemTag(t *testing.T) {
	// Setup: mob with aggro target but params missing item_tag.
	// Expect: Failure (Plant returns "no item" reason).
}

func TestActTryShadow_FailureWhenNotHidden(t *testing.T) {
	// Setup: mob not hidden, aggro target present.
	// Expect: Failure.
}

func TestActTryDefuse_FailureNoTraps(t *testing.T) {
	// Setup: room with no traps.
	// Expect: Failure.
}
```

Run:
```bash
go test ./internal/behaviortree/ -run TestActTry -v
```
Expected: FAIL (compile error).

- [ ] **Step 2: Implement the actions in `actions_skullduggery.go`**

```go
package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// actTrySneak invokes actions.Sneak via MobActor.
func actTrySneak(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Sneak(actor)
	if result.Success || result.AlreadyHidden {
		return Success
	}
	return Failure
}

// actTrySteal resolves a target via Event.UserId → Aggro and
// runs actions.Steal. Returns Success when the theft succeeded.
func actTrySteal(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	opts := actions.StealOptions{}
	if ctx.Event.UserId > 0 {
		opts.TargetUserId = ctx.Event.UserId
	} else if mob.Character.Aggro != nil {
		if mob.Character.Aggro.UserId > 0 {
			opts.TargetUserId = mob.Character.Aggro.UserId
		} else if mob.Character.Aggro.MobInstanceId > 0 {
			opts.TargetMobInstanceId = mob.Character.Aggro.MobInstanceId
		}
	}
	if opts.TargetUserId == 0 && opts.TargetMobInstanceId == 0 {
		return Failure
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Steal(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actTryPlant — same target resolution as actTrySteal. Reads
// item_tag param to populate PlantOptions.ItemNoun. Returns
// Failure if item_tag missing.
func actTryPlant(params map[string]any, ctx *EvalContext) Result {
	itemTag := getStringParam(params, "item_tag", "")
	if itemTag == "" {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	opts := actions.PlantOptions{ItemNoun: itemTag}
	// Same target resolution chain as actTrySteal:
	if ctx.Event.UserId > 0 {
		opts.TargetUserId = ctx.Event.UserId
	} else if mob.Character.Aggro != nil {
		if mob.Character.Aggro.UserId > 0 {
			opts.TargetUserId = mob.Character.Aggro.UserId
		} else if mob.Character.Aggro.MobInstanceId > 0 {
			opts.TargetMobInstanceId = mob.Character.Aggro.MobInstanceId
		}
	}
	if opts.TargetUserId == 0 && opts.TargetMobInstanceId == 0 {
		return Failure
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Plant(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actTryShadow — same target resolution. Requires the mob to
// already be hidden.
func actTryShadow(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	opts := actions.ShadowOptions{}
	if ctx.Event.UserId > 0 {
		opts.TargetUserId = ctx.Event.UserId
	} else if mob.Character.Aggro != nil {
		if mob.Character.Aggro.UserId > 0 {
			opts.TargetUserId = mob.Character.Aggro.UserId
		} else if mob.Character.Aggro.MobInstanceId > 0 {
			opts.TargetMobInstanceId = mob.Character.Aggro.MobInstanceId
		}
	}
	if opts.TargetUserId == 0 && opts.TargetMobInstanceId == 0 {
		return Failure
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Shadow(actor, opts)
	if result.Succeeded {
		return Success
	}
	return Failure
}

// actTryDefuse picks the first trap in the room and attempts to
// defuse it. Empty room → Failure.
func actTryDefuse(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	if len(room.TrapBuffIds) == 0 {
		return Failure
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Defuse(actor, actions.DefuseOptions{})
	if result.Succeeded {
		return Success
	}
	return Failure
}
```

The four-way duplicate target-resolution block is intentional — extracting a helper has marginal payoff for ~12 lines × 3, and keeps each action self-contained for the reader. If the codebase already has a `resolveAggroTarget(...)` helper (search `internal/behaviortree/`), use it.

- [ ] **Step 3: Register in `actions.go` init()**

```go
actionRegistry["try_sneak"] = actTrySneak
actionRegistry["try_steal"] = actTrySteal
actionRegistry["try_plant"] = actTryPlant
actionRegistry["try_shadow"] = actTryShadow
actionRegistry["try_defuse"] = actTryDefuse
```

- [ ] **Step 4: Verify tests pass**

```bash
go test ./internal/behaviortree/ -run TestActTry -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/actions_skullduggery.go internal/behaviortree/actions_skullduggery_test.go internal/behaviortree/actions.go
git commit -m "feat(btree): add try_<verb> skullduggery action primitives"
```

---

## Task 14: `thief` archetype YAML

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml`

Modeled on `predator.yaml` (chunk 2.4). Selector with priority-ordered children: shared panic-flee → power-overmatch combat → self-defense → steal-and-flee → re-stealth → fallback wander.

- [ ] **Step 1: Write the archetype**

```yaml
# thief archetype
#
# Sneak-first opportunist. Hides on idle, pickpockets passing
# players, flees on detection or aggression. Engages directly
# only when target is vastly outclassed (power ratio > 1.5).
#
# Spec: docs/superpowers/specs/2026-05-13-mob-aliveness-2.7-skullduggery-suite-design.md

tree:
  type: selector
  children:
    # Panic-flee at critical HP (shared with other archetypes
    # since chunk 2.6).
    - type: sequence
      event: mob_hurt
      children:
        - type: condition
          check: mob_health_below
          percent: 25
        - type: action
          do: flee

    # Power-overmatch combat opportunism: if a player walks in
    # who is 1.5x weaker, drop stealth and engage directly.
    - type: sequence
      event: player_enter
      children:
        - type: condition
          check: target_power_ratio_above
          value: 1.5
        - type: action
          do: attack

    # Self-defense: if attacked, fight back instead of fleeing
    # immediately. (Panic-flee branch above still handles the
    # critical-HP case.)
    - type: action
      event: mob_attacked
      do: attack

    # Core steal-and-flee loop: when idle and hidden, pick a
    # player with at least 5 gold, steal, and flee.
    - type: sequence
      event: mob_idle
      children:
        - type: condition
          check: mob_is_hidden
        - type: action
          do: target_random_player_in_room
        - type: condition
          check: target_has_gold
          min: 5
        - type: action
          do: try_steal
        - type: action
          do: flee

    # Re-stealth when uncovered.
    - type: sequence
      event: mob_idle
      children:
        - type: decorator
          mod: invert
          child:
            type: condition
            check: mob_is_hidden
        - type: action
          do: try_sneak

    # Fallback: idle wander (engine default if no branch fires).
```

The `attack` action is the existing standard btree action; `flee` is also existing. Verify the inversion syntax matches the predator/lookout pattern (chunk 2.4/2.6 conventions).

- [ ] **Step 2: Boot the server to verify archetype loads**

```bash
go run main.go
```

Watch the boot output for `behaviors.LoadDataFiles() loadedCount=...` without panic. Then kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/thief.yaml
git commit -m "feat(behaviors): add thief archetype for sneak-and-steal mobs"
```

---

## Task 15: Flip the Thornwall highwayman

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml`

Single-line change: `behavior_archetype: generic_fighter` → `behavior_archetype: thief`. The highwayman already has `skullduggery: 8` (well above the rank-2 floor) and stat training that favors stealth (dex 8, per 5).

- [ ] **Step 1: Apply the change**

Edit `_datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml` line 3:

```yaml
behavior_archetype: thief
```

Optional cleanup: the `sneak` entry in `idlecommands` (line 12) becomes redundant — the archetype handles it. Leave the entry for backward compat (it'll just be a duplicate sneak attempt that succeeds or already-hidden).

- [ ] **Step 2: Check rooms.instances**

Per CLAUDE.md SOP, check for stale instance saves that might override the template:
```bash
ls "_datafiles/world/dogmud/rooms.instances/thornwall_outskirts/" 2>/dev/null
```
Mob YAML changes don't trigger instance saves — they're loaded fresh on respawn. But verify no mob instance save for mob 90 exists:
```bash
find _datafiles/world/dogmud/mobs.instances/ -name "*highwayman*" 2>/dev/null
```
If present, delete (they're gitignored anyway).

- [ ] **Step 3: Boot the server and verify clean load**

```bash
go run main.go
```

Expected: `mobs.LoadDataFiles()` succeeds without panic. Kill the server.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_outskirts/90-thornwall_highwayman.yaml
git commit -m "feat(mobs): flip Thornwall highwayman to thief archetype"
```

---

## Task 16: Legacy sunset

**Files:**
- Delete: `internal/usercommands/stealth_detection.go`
- Modify: `internal/usercommands/usercommands_test.go`
- Modify: `internal/hooks/go.go`

Three small cleanups now that the consolidation has landed.

- [ ] **Step 1: Verify no consumers remain for `stealth_detection.go`**

```bash
grep -rn "usercommands.calcSneakScore\|calcSneakScore" --include="*.go" .
```

After Tasks 1-9, all callers should use `actions.CalcSneakScore` directly. If any residual references remain, fix them before deleting the file.

- [ ] **Step 2: Delete `internal/usercommands/stealth_detection.go`**

```bash
git rm internal/usercommands/stealth_detection.go
```

- [ ] **Step 3: Remove pickpocket placeholder from tests**

Open `internal/usercommands/usercommands_test.go` and find line ~7142 (the empty pickpocket test). Delete the function (and any surrounding registration entries).

- [ ] **Step 4: Collapse triple-removal in `hooks/go.go`**

Open `internal/hooks/go.go` at line 445. The current pattern looks roughly like:

```go
mob.Character.CancelBuffsWithFlag(buffs.Hidden)
mob.Character.RemovePermaBuff(9)
mob.Character.Buffs.RemoveBuff(9)
```

Collapse to:

```go
mob.Character.CancelBuffsWithFlag(buffs.Hidden)
```

The `cancel-on-combat` flag handling in the buff engine is the authoritative removal path. The explicit `RemovePermaBuff` / `RemoveBuff` are belt-and-suspenders that mask the canonical event flow.

- [ ] **Step 5: Run package tests to verify no regression**

```bash
go test ./internal/usercommands/ ./internal/hooks/ -v
```
Expected: all pass.

- [ ] **Step 6: Build verification**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add -A internal/usercommands/ internal/hooks/go.go
git commit -m "chore: sunset legacy skullduggery shim, pickpocket placeholder, triple-removal pattern"
```

---

## Task 17: Documentation updates

**Files:**
- Modify: `internal/actions/context.md`
- Modify: `internal/behaviortree/context.md`

Per the roadmap's `context.md` SOP — package-level docs must stay current.

- [ ] **Step 1: Update `internal/actions/context.md`**

Add a section "Skullduggery actions" listing the five entries:

```markdown
### Skullduggery actions (chunk 2.7)

- `Sneak(actor) SneakResult` — apply hidden buff after opposed roll
  against observers. Already lifted in chunk 2.4 prep; renamed
  from `ExecuteSneak` to `Sneak` in chunk 2.7.
- `Steal(actor, opts) StealResult` — pickpocket a mob or rob a
  room container. Mob-on-player path runs an extra detection
  roll on the victim to gate the "you got pickpocketed" message.
- `Plant(actor, opts) PlantResult` — slip an item from
  backpack onto a mob or into a container. Shares the
  skullduggery cooldown key with Steal.
- `Defuse(actor, opts) DefuseResult` — disarm a room trap.
  Optional disarm-kit consumption from backpack.
- `Shadow(actor, opts) ShadowResult` — follow a target while
  hidden. Stores target in misc-data for follow-on auto-go.

All five expose StealOptions/PlantOptions/etc. for caller-side
target structuring (CLI args parsed in usercommands/mobcommands
wrappers; btree primitives populate from EvalContext.Event +
mob.Character.Aggro).
```

- [ ] **Step 2: Update `internal/behaviortree/context.md`**

Add to the Actions section:
```markdown
- `try_sneak` — invoke `actions.Sneak`. Success when the mob
  enters or is already in the hidden state.
- `try_steal` — invoke `actions.Steal` against aggro target.
- `try_plant item_tag: "<noun>"` — invoke `actions.Plant`
  against aggro target with named item from inventory.
- `try_shadow` — invoke `actions.Shadow` against aggro target.
  Requires the mob already hidden.
- `try_defuse` — invoke `actions.Defuse` on the first trap in
  the current room.
- `target_random_player_in_room` — sister to
  `target_weakest_mob_in_room`. Picks a random player in the
  current room and sets them as Aggro.
```

Add to the Conditions section:
```markdown
- `mob_is_hidden` — true when self carries Hidden buff (id 9).
- `target_is_hidden` — true when resolved target carries Hidden buff.
- `target_has_gold min: <N>` — true when resolved player target
  has >= N gold. Mob targets always Failure.
```

- [ ] **Step 3: Commit**

```bash
git add internal/actions/context.md internal/behaviortree/context.md
git commit -m "docs(context): chunk 2.7 — skullduggery actions + btree primitives"
```

---

## Task 18: Build, full test, and smoke validation

**Files:** (none new — verification only)

- [ ] **Step 1: Build the whole project**

```bash
go build ./...
```
Expected: clean.

- [ ] **Step 2: Run all package tests**

```bash
go test ./...
```
Expected: all packages pass.

- [ ] **Step 3: Boot the server locally and verify data files load cleanly**

Per the CLAUDE.md Pre-Push SOP — `go build` only checks compilation, not data file integrity. Boot the server and confirm:

```bash
go run main.go
```

Watch for these log lines (or equivalent) without panic:
- `mobs.LoadDataFiles() loadedCount=...`
- `quests.LoadDataFiles() loadedCount=...`
- `behaviors.LoadDataFiles() loadedCount=...`
- `items.LoadDataFiles() loadedCount=...`

After confirming clean boot, kill the server.

- [ ] **Step 4: In-game smoke per spec section "Testing & smoke validation"**

Boot the server, log in as a test character, and run through the four smoke scenarios from the spec:

1. **Stealth-only loop** (mid-Per character): walk to Thornwall Outskirts, enter the highwayman's room, wait 2-3 rounds, leave. Re-enter. Verify gold disappears between visits OR detection message fires.

2. **Detection path** (high-Per character): same flow, expect detection message text: `<ansi fg="mobname">thornwall highwayman</ansi> lifts <ansi fg="gold">N gold</ansi> from your pocket!`

3. **Power-overmatch override** (high-level character): walk in vastly stronger than the mob. Verify it switches to combat-aggressive — drops stealth, fires `attack`, fights normally.

4. **Self-defense** (any character): attack the highwayman. Verify it fights back (does not flee on first hit; panic-flee at <25% HP still applies).

Record observed behavior in commit notes. If any scenario fails, the issue is most likely either the archetype tree wiring or a target-resolution bug in a btree primitive — both reachable from the in-game state and not by unit tests.

- [ ] **Step 5: Kill all running test servers**

Per the kill-test-servers SOP after a smoke session:
```bash
# Windows
taskkill /F /IM dogmud.exe 2>/dev/null; taskkill /F /IM "go run" 2>/dev/null
# Or list and kill manually
```

- [ ] **Step 6: Commit smoke notes (optional)**

If smoke surfaces a fix worth committing, do so in this task. Otherwise skip.

---

## Task 19: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Mark 2.7 Done in the progress tracker table (around line 92)**

Change:
```markdown
| 2.7 | Tactical | Mob skullduggery suite | M | — | Not started |
```
to:
```markdown
| 2.7 | Tactical | Mob skullduggery suite | M | — | Done |
```

- [ ] **Step 2: Update the roll-up line (around line 120)**

Change:
```markdown
**Roll-up:** 14 / 41 done • 0 in progress • 27 not started.
```
to:
```markdown
**Roll-up:** 15 / 41 done • 0 in progress • 26 not started.
```

- [ ] **Step 3: Update the 2.7 mini-brief (around line 360)**

Change the `**Status:** Not started` line to `**Status:** Done (2026-05-13)`. Add a `- **Shipped:**` paragraph summarizing the work — model after how 2.4/2.6 wrote theirs (sentence on what landed, file pointers, spec/plan links, any known followups).

- [ ] **Step 4: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 2.7 (mob skullduggery suite) as Done"
```

---

## Spec coverage check

| Spec section | Tasks |
|--------------|-------|
| Actions-package consolidation (5 verbs) | Tasks 1, 2, 4, 6, 8 |
| Player wrappers thin | Tasks 3, 5, 7, 9 |
| Mob wrappers + registration | Task 10 |
| Btree action primitives (5 `try_*`) | Task 13 |
| Btree conditions (3 state queries) | Task 12 |
| Helper: target_random_player_in_room | Task 11 |
| thief archetype | Task 14 |
| Test mob (Thornwall highwayman) | Task 15 |
| Legacy sunset (3 items) | Task 16 |
| Docs (actions + btree context.md) | Task 17 |
| Build/test/smoke | Task 18 |
| Roadmap update | Task 19 |

All architectural musts from the spec are covered. Mob-on-player detection roll is implemented inside `actions.Steal` and `actions.Plant` (Tasks 2 step 3 point 10, Task 4 step 3). Symmetric-with-player-side detection is preserved.

## Known followup (not in this chunk)

- Dual-path sneak detection in `hooks/go.go:81-86` — buff flag AND misc-data `sneaking`. Refactor risk; logged as a separate followup ticket post-chunk.
- Per spec out-of-scope: pickpocket as a separate verb, hide as a separate verb, additional thief-archetype mob flips, btree consumers for defuse/shadow, mob-on-player crime/opinion ledger.
