# Fix: Beast moves never fire (btree bypasses ChooseSpecialMove) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Steps use `- [ ]` checkboxes.

**Goal:** Make the Phase-3/4 beast special moves (rake/maul/pounce/gore/throttle/hamstring/drain) actually fire in live combat by having the behavior-tree combat round delegate special-move selection to the existing aiprofile-weighted `ChooseSpecialMove`.

**Root cause (confirmed):** `NewRound_DoCombat.go:313` runs the btree first and `continue`s on success, skipping `handleMobAIDecision`→`ChooseSpecialMove`. The btree archetypes' `mob_combat_round` cascades (`predator.yaml`, `generic_fighter.yaml`) `command_best_of` only `[bash,trip,kick,grapple]` — no beast moves. So beast moves (wired only into `aiProfiles`/`ChooseSpecialMove`) are unreachable for any mob with a `behavior_archetype`.

**Approach (user-approved):** A delegation btree action `try_special_move` that calls the aiprofile-weighted selection and issues the result, **filtered to beast moves** so humanoid mobs keep their existing tactical cascade untouched. Reuses all Phase-3/4 weighting; no per-archetype btree files, no mob reassignment.

**Verified facts:**
- `internal/behaviortree` already imports `internal/combat` (`actions_combat.go`); `combat` does NOT import `behaviortree` (no cycle).
- Existing pattern: `actCommandBestOf` (`actions_mob.go`) → `actions.CommandIsReady` + `mob.Command(cmd)`.
- `ChooseSpecialMove` (`internal/combat/ai.go`) starts with `if util.Rand(100) >= mob.SpecialMoveChance { return "" }`, then builds weighted `moveScores` and returns the best.
- Beast moves are anatomy/identity-gated in `CanUse*` (incl. the Phase-4 hands rule), so a humanoid's `ChooseSpecialMove` never returns a beast move.
- `pure_caster.yaml` `mob_combat_round` = a cascade of `cast_best_in_category` then fall-through; `predator.yaml`/`generic_fighter.yaml` cascades start with a cast-interrupt `[bash,trip]` branch.

---

## Task 1: Split the SpecialMoveChance roll out of ChooseSpecialMove

So the btree (not a second random roll) controls when a special is attempted.

**Files:** `internal/combat/ai.go`, `internal/combat/ai_test.go`.

- [ ] **Step 1: Failing test.** Add `TestSelectSpecialMove_NoChanceRoll`: a mob with `SpecialMoveChance = 0` (would always fail the roll) still gets a move from `SelectSpecialMove` (the forced selector), while `ChooseSpecialMove` returns "" for it. Seed a fanged predator + target as in `TestChooseSpecialMove_PredatorProfile_NeverGrapple`.
- [ ] **Step 2: Refactor.** Extract the body of `ChooseSpecialMove` AFTER the `SpecialMoveChance` gate into:
```go
// SelectSpecialMove returns the best aiprofile-weighted special move for the
// mob vs target ("" if none viable). It does NOT roll SpecialMoveChance — the
// caller decides whether to attempt a special (the btree's combat cascade, or
// ChooseSpecialMove's own roll).
func SelectSpecialMove(mob *mobs.Mob, target *characters.Character) string { /* moved body */ }
```
Then `ChooseSpecialMove` becomes:
```go
func ChooseSpecialMove(mob *mobs.Mob, target *characters.Character) string {
	if util.Rand(100) >= mob.SpecialMoveChance {
		return ""
	}
	return SelectSpecialMove(mob, target)
}
```
- [ ] **Step 3:** Test passes; `go test ./internal/combat/` green; `go build ./...` clean.
- [ ] **Step 4: Commit** `refactor(combat): extract SelectSpecialMove (weighted selection, no chance roll)`.

(End every commit with `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.)

---

## Task 2: `try_special_move` btree action (beast-move filtered)

**Files:** `internal/behaviortree/actions_mob.go` (+ wherever actions are registered), `internal/behaviortree/actions_mob_test.go`.

- [ ] **Step 1: Failing test** (follow `TestActCommandBestOf_*` harness in `actions_test.go`): a fanged, no-hands beast mob (seed species, set aggro to a target, no special-move cooldown) → `LookupAction("try_special_move")(params, ctx)` returns `Success` AND issued a beast move (assert via the input-capture harness the existing btree tests use, or that `mob`'s aggro round was consumed). A humanoid mob (has hands → no beast move viable) → returns `Failure` (so the tactical cascade still runs). An out-of-combat mob → `Failure`.
- [ ] **Step 2: Implement** `actTrySpecialMove` (mirror `actCommandBestOf`):
```go
var beastMoves = map[string]bool{
	"rake": true, "maul": true, "pounce": true, "gore": true,
	"throttle": true, "hamstring": true, "drain": true,
}

func actTrySpecialMove(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Aggro == nil {
		return Failure
	}
	target := resolveAggroTargetChar(mob) // see note
	if target == nil {
		return Failure
	}
	move := combat.SelectSpecialMove(mob, target)
	if move == "" || !beastMoves[move] {
		return Failure // let the tactical cascade handle humanoid specials
	}
	mob.Command(move)
	return Success
}
```
Resolve the aggro target's `*characters.Character` the same way `handleMobAIDecision` does (UserId → `users.GetByUserId`; MobInstanceId → `mobs.GetInstance`) — read that block and reuse it (add a small local helper or inline). Register `try_special_move` in the action registry next to `command_best_of`.
- [ ] **Step 3:** Tests pass; `go test ./internal/behaviortree/` green; `go build ./...` clean.
- [ ] **Step 4: Commit** `feat(behaviortree): try_special_move action delegating to SelectSpecialMove (beast-move filtered)`.

---

## Task 3: Wire `try_special_move` into the btree archetypes

**Files:** `_datafiles/world/dogmud/behaviors/archetypes/predator.yaml`, `generic_fighter.yaml`, `pure_caster.yaml`.

- [ ] **Step 1:** In `predator.yaml` and `generic_fighter.yaml`, add as the FIRST child of the `mob_combat_round` selector (before the cast-interrupt `[bash,trip]` branch):
```yaml
        - type: action
          do: try_special_move
```
Rationale: a beast prefers its weighted beast move (incl. `throttle`, which `ScoreThrottle` boosts vs casting targets — covering the cast-interrupt case for beasts). Humanoids get `Failure` here and fall through to the unchanged tactical cascade.
- [ ] **Step 2:** In `pure_caster.yaml`, add the same `try_special_move` action as the LAST child of the `mob_combat_round` selector (after all `cast_best_in_category` attempts), so casters (wraith/spectre) drain only when no spell is castable — matching the "occasional drain" design.
- [ ] **Step 3:** `go build ./...` clean; the btree YAML loads (boot test in Task 4). Commit `feat(content): delegate beast special-move selection from predator/generic_fighter/pure_caster btrees`.

---

## Task 4: Verify + boot + smoke + merge

- [ ] **Step 1:** `go build ./... && go test ./...` green.
- [ ] **Step 2: Boot test** (instance-save wipe, then boot) → `Server Ready`, no panic, btrees load.
- [ ] **Step 3: In-game smoke** — reuse the Thornwall-placed P2smoke tester + `tools/playtest/goals/phase4-beast-checks.yaml`. THIS TIME the moves must actually fire: confirm a Steppe Wolf **mauls/throttles/pounces** (not just bites/trips), a boar **gores**, a summoned wraith/spectre **drains** occasionally, a goblin/skeleton still does NOT beast-move (hands rule), and basic attacks + humanoid tactics are unchanged. Write a report.
- [ ] **Step 4: Merge to local master** (no push).

---

## Self-review notes (controller)
- The **beast-move filter** in `actTrySpecialMove` is the crux: it ensures only beasts are affected; humanoid mobs return `Failure` and keep their deliberate tactical cascade (cast-interrupt → kick-prone → bash → grapple-lone → trip) exactly as before. No humanoid-combat regression.
- Frequency: the shared `special-move` cooldown still caps one special per `SpecialMoveChance`-independent cooldown window; `try_special_move` at the top of the cascade means that one special is the weighted beast move when ready, else the cascade falls through.
- This activates the (currently inert) Phase-3/4 `aiprofile` assignments + the `skirmisher`/`serpent`/`caster`-`drain` weighting with zero data churn.
