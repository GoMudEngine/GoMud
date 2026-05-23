# Mob Aliveness 2.10 Followups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close out chunk 2.10's deferred work by shipping 5 followups in one bundled chunk: charge.go cleanup, surprise-attack unification, try_any_active_mutation btree action, mutation damage pipeline routing, and the forager locked-chest workflow.

**Architecture:** Refactors first (mechanical, low risk), new mechanisms next (build on existing patterns), feature work last (forager-chest workflow with new content). All on one feature branch.

**Tech Stack:** Go 1.24, existing `internal/actions/` Actor abstraction, existing `internal/behaviortree/actions_*.go` registry pattern, unified damage pipeline (`internal/combat/damage_pipeline.go`), `internal/gamelock/` for lock state, YAML content.

**Spec:** `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-followups-design.md`

**Branch:** `feature/mob-aliveness-2.10-followups` (already created; spec committed as `31049971`).

---

## Stage map

| Stage | Item | Tasks |
|---|---|---|
| 1 | Item 1: charge.go dup | T1 |
| 2 | Item 2: Surprise-attack unification | T2 |
| 3 | Item 3: try_any_active_mutation | T3 |
| 4 | Item 4: Mutation damage pipeline (sonic-shout) | T4 |
| 4 | Item 4: Mutation damage pipeline (toxic-bite) | T5 |
| 5 | Item 5a: Mob lock/unlock verbs | T6 |
| 5 | Item 5b: try_store_excess btree primitive | T7 |
| 5 | Item 5c: Tova's dwelling room | T8 |
| 5 | Item 5d: Keys + mob inventory updates | T9 |
| 5 | Item 5e: State machine StateStoring + archetype | T10 |
| 6 | Closeout: memory writes | T11 |
| 6 | Closeout: PATCH_NOTES + roadmap | T12 |

13 sequential tasks. Subagent-driven execution recommended.

---

## Task 1: charge.go trip-math duplication cleanup

**Files:**
- Modify: `internal/mobcommands/charge.go`
- Possibly modify: `internal/actions/combat_trip.go` (only if signature extension needed)
- Test: `internal/mobcommands/charge_test.go` (existing; add charge-decoration test)

- [ ] **Step 1: Read both files side-by-side**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
cat internal/mobcommands/charge.go
cat internal/actions/combat_trip.go
```

Identify the trip-resolution lines in `charge.go` that duplicate `actions.ExecuteTrip` (knockdown roll, prone application, damage-on-fall). Note the charge-specific lines that stay (distance approach narration, post-impact aggro behavior).

If `actions.ExecuteTrip`'s signature can't accommodate charge's needs (e.g., charge wants different stamina cost or knockdown threshold), determine what extension is needed. If extension required, expand `actions.ExecuteTrip`'s `Opts` struct with new fields; default behavior stays unchanged.

- [ ] **Step 2: Write a charge-decoration test (TDD for the refactor)**

Add to `internal/mobcommands/charge_test.go` (or create the file if it doesn't exist):

```go
package mobcommands

import (
	"testing"
)

func TestCharge_Decoration_FiresOnKnockdown(t *testing.T) {
	// Set up attacker mob with adequate stats to make knockdown likely.
	// Set up victim mob in same room.
	// Force the knockdown roll to succeed (e.g., set attacker.Strength
	// very high and victim Vitality very low).
	// Run Charge.
	// Assert: victim has Prone condition AND the charge-specific
	// approach/impact emote fired (verify via room broadcast capture).
}
```

Adapt to whatever helpers exist in `internal/mobcommands/mobcommands_test.go`. The intent is to lock in the charge-decoration's presence so the refactor preserves it.

- [ ] **Step 3: Run test, expect PASS or FAIL based on current state**

```bash
go test ./internal/mobcommands/ -run TestCharge_Decoration -v
```

Document the baseline. If current `charge.go` does emit the decoration, the test passes immediately (this becomes a regression test for the refactor). If it doesn't, the test FAILs first and we'll make it pass during refactor.

- [ ] **Step 4: Refactor charge.go to delegate to actions.ExecuteTrip**

In `internal/mobcommands/charge.go`, identify the trip-resolution block. Replace with a call:

```go
result := actions.ExecuteTrip(
    actions.NewMobActorInRoom(mob, room),
    target,            // however target is currently resolved
    actions.ExecuteTripOpts{
        // any charge-specific overrides
    },
)
```

Keep the charge-specific lines outside the call: distance-approach emote BEFORE the call, post-impact aggro update AFTER the call. Match what the existing charge.go did, minus the trip math.

Run grep to find the actual `ExecuteTrip` signature:

```bash
grep -n "func ExecuteTrip" internal/actions/combat_trip.go
```

Adapt the call to match.

- [ ] **Step 5: Run tests**

```bash
go test ./internal/mobcommands/ ./internal/actions/ -v 2>&1 | tail -10
```

Expected: PASS. Charge tests + trip tests both clean.

- [ ] **Step 6: Build check**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/mobcommands/charge.go internal/mobcommands/charge_test.go
# Also add internal/actions/combat_trip.go if it was extended
git commit -m "$(cat <<'EOF'
refactor(combat): charge.go delegates trip math to actions.ExecuteTrip

mobcommands/charge.go was reimplementing the trip-resolution math
(knockdown roll, prone application) instead of calling the shared
actions.ExecuteTrip helper. The duplication is now consolidated:
charge keeps only its decoration (approach narration, post-impact
aggro behavior) and delegates the trip mechanic to the actions
package.

Resolves project_charge_trip_math_duplication.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Surprise-attack unification refactor

**Files:**
- Create: `internal/actions/surprise_attack.go`
- Create: `internal/actions/surprise_attack_test.go`
- Modify: `internal/usercommands/attack.go` (find `executeSurpriseAttack` helper, replace with action call)
- Modify: `internal/mobcommands/attack.go:64` area (replace inline branch with action call)

- [ ] **Step 1: Read both existing implementations**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "executeSurpriseAttack\|surprise" internal/usercommands/attack.go | head -20
grep -n "surprise" internal/mobcommands/attack.go | head -20
```

Read the player helper function and the mob inline branch. Note the exact shared logic: hidden-state check, special-move cooldown try, per-weapon iteration, per-weapon attack roll, Awareness_Cascades signal.

- [ ] **Step 2: Write the failing test**

Create `internal/actions/surprise_attack_test.go`:

```go
package actions

import (
	"testing"
)

func TestSurpriseAttack_NotHidden(t *testing.T) {
	attacker := newTestMobBare(t)
	// Do NOT set hidden condition.
	attacker.Character.Stamina = 100
	room := newTestRoomBare(t)
	victim := newTestMobInRoom(t, room)
	actor := NewMobActorInRoom(attacker, room)
	target := NewMobActorInRoom(victim, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: target})
	if res.Triggered {
		t.Fatal("expected Triggered=false when attacker is not hidden")
	}
	if res.BlockReason != "not-hidden" {
		t.Errorf("expected BlockReason=not-hidden, got %s", res.BlockReason)
	}
}

func TestSurpriseAttack_NoTarget(t *testing.T) {
	attacker := newTestMobBare(t)
	// Set hidden condition (use whatever the codebase's hidden mechanism is)
	// attacker.Character.AddCondition(characters.ConditionHidden, ...)
	attacker.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(attacker, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: nil})
	if res.Triggered {
		t.Fatal("expected Triggered=false without target")
	}
	if res.BlockReason != "no-target" {
		t.Errorf("expected BlockReason=no-target, got %s", res.BlockReason)
	}
}

func TestSurpriseAttack_Success_FiresPerWeapon(t *testing.T) {
	attacker := newTestMobBare(t)
	// Set hidden condition.
	attacker.Character.Stamina = 100
	// Equip 2 weapons (main + offhand) on attacker
	// ... whatever the helpers allow

	room := newTestRoomBare(t)
	victim := newTestMobInRoom(t, room)
	actor := NewMobActorInRoom(attacker, room)
	target := NewMobActorInRoom(victim, room)

	res := SurpriseAttack(actor, SurpriseAttackOpts{Target: target})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if res.StrikeCount < 1 {
		t.Errorf("expected StrikeCount >= 1, got %d", res.StrikeCount)
	}
}
```

The "hidden condition" mechanism varies by codebase — check the existing player code to see how it gates. Common pattern: `characters.ConditionHidden` or a `Hidden` buff. Use whatever the existing `executeSurpriseAttack` checks.

If `newTestMobInRoom` doesn't exist in `actions_test.go`, model on chunk 2.10's B2 test helpers — local helper in the test file is acceptable per chunk-2.10 precedent.

- [ ] **Step 3: Run test, expect failure**

```bash
go test ./internal/actions/ -run TestSurpriseAttack -v
```

Expected: FAIL with "undefined: SurpriseAttack, SurpriseAttackOpts".

- [ ] **Step 4: Implement `internal/actions/surprise_attack.go`**

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	// other imports as needed
)

// SurpriseAttackOpts parameterizes a surprise-attack trigger attempt.
type SurpriseAttackOpts struct {
	Target Actor
}

// SurpriseAttackResult is the structured outcome.
// On Triggered=false, BlockReason is one of:
//   "not-hidden", "on-cooldown", "no-target", "no-character"
// On Triggered=true, StrikeCount counts the per-weapon strikes that landed.
type SurpriseAttackResult struct {
	Triggered   bool
	StrikeCount int
	BlockReason string
}

// SurpriseAttack fires the surprise-attack mechanic. Requires the
// attacker to be in a hidden state and have a free special-move
// cooldown. On success, iterates each equipped weapon and rolls an
// attack against the target. The Awareness_Cascades hook handles the
// Hidden → Revealing transition at round end.
func SurpriseAttack(actor Actor, opts SurpriseAttackOpts) SurpriseAttackResult {
	char := actor.GetCharacter()
	if char == nil {
		return SurpriseAttackResult{BlockReason: "no-character"}
	}

	// Hidden gate — port from existing executeSurpriseAttack
	if !char.HasCondition(characters.ConditionHidden) {  // verify the constant name
		return SurpriseAttackResult{BlockReason: "not-hidden"}
	}

	if opts.Target == nil {
		return SurpriseAttackResult{BlockReason: "no-target"}
	}

	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", "5 rounds") {
		// Adjust the cooldown duration to match what the existing code uses
		return SurpriseAttackResult{BlockReason: "on-cooldown"}
	}

	// Per-weapon iteration — port from existing code
	strikeCount := 0
	// for each equipped weapon (main + offhand + any extra arms):
	//   roll attack against opts.Target.GetCharacter()
	//   if hit: apply damage via the unified pipeline or whatever
	//     the existing executeSurpriseAttack does
	//   strikeCount++
	// (port the existing logic verbatim; this is a lift, not a redesign)

	// Awareness_Cascades signaling — the hook handles Hidden → Revealing
	// at end of round. The action doesn't need to do anything special
	// here; the hidden state stays in place during this round so the
	// surprise attack resolves correctly.

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			"You strike from concealment!")
	}

	_ = cfg // remove if cfg not used; otherwise reference what's needed

	return SurpriseAttackResult{Triggered: true, StrikeCount: strikeCount}
}
```

**Critical:** Port the per-weapon iteration verbatim from `executeSurpriseAttack`. Don't rewrite the attack-roll logic — copy it. The lift goal is to consolidate, not redesign.

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/actions/ -run TestSurpriseAttack -v
```

- [ ] **Step 6: Rewrite the player wrapper**

In `internal/usercommands/attack.go`, find the `executeSurpriseAttack` helper. Replace its body with a call to `actions.SurpriseAttack`:

```go
// Old: ~50-100 lines of executeSurpriseAttack implementation
// New: collapse to a thin call-through.
//
// Find the callsite (probably in the main Attack function's
// "if hidden then surprise" branch). Replace direct call to
// executeSurpriseAttack(...) with:

actor := actions.NewUserActorInRoom(user, room)
target := actions.NewMobActorInRoom(targetMob, room)
// OR: actions.ResolveEngagedTarget(actor, room) — use whichever
//     the wrapper has access to. The attack flow already has the
//     target resolved by the time this branch fires.
_ = actions.SurpriseAttack(actor, actions.SurpriseAttackOpts{Target: target})
```

Delete the now-orphaned `executeSurpriseAttack` function from `internal/usercommands/attack.go`.

- [ ] **Step 7: Rewrite the mob wrapper**

In `internal/mobcommands/attack.go:64` area, find the inline surprise-attack branch (the comment "Hidden mobs get a surprise attack on their first strike" plus the logic below it). Replace with a call-through:

```go
actor := actions.NewMobActorInRoom(mob, room)
target := actions.NewMobActorInRoom(targetMob, room)  // or similar
_ = actions.SurpriseAttack(actor, actions.SurpriseAttackOpts{Target: target})
```

- [ ] **Step 8: Run full test suite**

```bash
go build ./...
go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ 2>&1 | tail -10
```

Expected: PASS, no regressions.

- [ ] **Step 9: Commit**

```bash
git add internal/actions/surprise_attack.go \
        internal/actions/surprise_attack_test.go \
        internal/usercommands/attack.go \
        internal/mobcommands/attack.go
git commit -m "$(cat <<'EOF'
refactor(actions): lift surprise_attack into actions package

Player and mob paths previously implemented the per-weapon
surprise-attack logic independently. The mechanics were identical
(hidden state + free special-move cooldown -> per-weapon attack roll)
but living in two places, prone to drift on future tweaks.

Consolidated as actions.SurpriseAttack with both wrappers (user +
mob) collapsing to thin call-throughs. Mechanics preserved verbatim.

Awareness_Cascades hook coordination unchanged - the state-transition
policy stays where it is.

Resolves project_surprise_attack_unification.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `try_any_active_mutation` btree action

**Files:**
- Modify: `internal/behaviortree/actions_mutation.go` (extend with new action func + helpers)
- Modify: `internal/behaviortree/actions.go` (register `try_any_active_mutation`)
- Modify: `internal/behaviortree/actions_mutation_test.go` (add new tests)
- Modify: `internal/behaviortree/context.md` (document the new action)

- [ ] **Step 1: Read the existing `actions_mutation.go`**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
cat internal/behaviortree/actions_mutation.go
```

Note the existing `mutationTriggers` map (post-Issue-2 fix: contains only the self/AoE mutations — `blinding-flash`, `healing-gel`, `pacifism-aura`, `sonic-shout`). The new action reuses this same map.

- [ ] **Step 2: Inspect mutation YAML files for the `rarity` field**

```bash
ls _datafiles/world/dogmud/mutations/
cat _datafiles/world/dogmud/mutations/blinding-flash.yaml | head -20
```

Confirm the `rarity` field exists on each mutation YAML and find the loader/registry that exposes it at runtime (likely `internal/mutations/`):

```bash
grep -rn "rarity\|Rarity" internal/mutations/ | head -10
```

You need a function or accessor that returns the rarity given a mutation key — something like `mutations.RarityOf(key)` or reading from `mutations.RegistryGet(key).Rarity`. Find the right API.

- [ ] **Step 3: Write failing tests**

Add to `internal/behaviortree/actions_mutation_test.go`:

```go
func TestTryAnyActiveMutation_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_any_active_mutation"]; !ok {
		t.Fatal("try_any_active_mutation not registered in actionRegistry")
	}
}

func TestTryAnyActiveMutation_NoEligibleMutations(t *testing.T) {
	// Mob with no mutations -> Failure
	mob := newTestMobInst(t)
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	res := actTryAnyActiveMutation(map[string]any{}, ctx)
	if res != Failure {
		t.Errorf("expected Failure for mob with no mutations, got %v", res)
	}
}

func TestTryAnyActiveMutation_OnlySingleTargetMutations(t *testing.T) {
	// Mob has only blinding-spit + toxic-bite (single-target, excluded)
	mob := newTestMobInst(t)
	mob.Character.Mutations = map[string]int{
		"blinding-spit": 1,
		"toxic-bite":    1,
	}
	mob.Character.Stamina = 100
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	res := actTryAnyActiveMutation(map[string]any{}, ctx)
	if res != Failure {
		t.Errorf("expected Failure when no self/AoE mutations available, got %v", res)
	}
}

func TestTryAnyActiveMutation_RaresFirstOrdering(t *testing.T) {
	// Mob has blinding-flash (assume rarity 5) + healing-gel (assume rarity 3)
	// Expect blinding-flash tried first because higher rarity.
	// Since both will likely block (no combat or whatever), expect Failure
	// after both attempted - but the test checks the ORDER, which we verify
	// by checking which mutation's preamble was consulted first.
	//
	// One way: mock or intercept the mutationTriggers map.
	// Another way: set the mob's stamina to enable ONLY ONE of the
	// mutations to fire (e.g., stamina 10 lets healing-gel fire at 8
	// but not blinding-flash at 14). If healing-gel is the one that
	// fires, ordering was right (rarer one tried first, failed on
	// stamina, fell through to common one). If blinding-flash is the
	// one that fires, ordering was wrong.
	//
	// Easier: introspection-via-side-effect. Add a test-only helper
	// that records the order mutationTriggers was called. Implement
	// in the production code if needed.
	t.Skip("rarity-ordering test requires either side-effect introspection " +
		"or a fully-instantiated mutation registry; cover during smoke test")
}
```

The rarity-ordering test is genuinely hard to write at the unit level without infrastructure to introspect dispatch order. Document the skip; smoke-test will exercise it.

- [ ] **Step 4: Run tests, expect failure**

```bash
go test ./internal/behaviortree/ -run TryAnyActiveMutation -v
```

Expected: FAIL with "undefined: actTryAnyActiveMutation".

- [ ] **Step 5: Implement the new action**

Extend `internal/behaviortree/actions_mutation.go`:

```go
// Add after the existing actTryMutationActive function:

import (
	"sort"
	// ...existing imports
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// actTryAnyActiveMutation fires the rarest available self/AoE mutation
// the mob has, falling through to less rare ones if the rarer ones can't
// fire (cooldown / stamina / etc.). Returns Failure if no candidate fires.
//
// Single-target mutations (blinding-spit, toxic-bite) are excluded:
// they require target resolution that this action doesn't perform.
// They'll be dispatchable via a future try_mutation_active_at_target
// primitive once that lands.
//
// Determinism: candidates are sorted by rarity (descending), with
// alphabetical key tiebreak. This makes ordering reproducible
// regardless of Go map iteration order.
func actTryAnyActiveMutation(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}

	// Enumerate mob's mutations that are in the dispatch map.
	candidates := []string{}
	for key := range mob.Character.Mutations {
		if _, ok := mutationTriggers[key]; ok {
			candidates = append(candidates, key)
		}
	}
	if len(candidates) == 0 {
		return Failure
	}

	// Sort: rarity descending, alphabetical tiebreak.
	sort.SliceStable(candidates, func(i, j int) bool {
		ri := mutations.RarityOf(candidates[i])  // adapt to actual API
		rj := mutations.RarityOf(candidates[j])
		if ri != rj {
			return ri > rj  // descending
		}
		return candidates[i] < candidates[j]  // alphabetical tiebreak
	})

	actor := actions.NewMobActorInRoom(mob, room)
	for _, key := range candidates {
		trigger := mutationTriggers[key]
		res := trigger(actor, actions.MutationOpts{})
		if res.Triggered {
			return Success
		}
	}
	return Failure
}
```

**Adapt `mutations.RarityOf` to the actual API found in Step 2.** Likely names:
- `mutations.RarityOf(key string) int`
- `mutations.GetRarity(key string) int`
- Or read from a registry: `mutations.GetSpec(key).Rarity`

If no public accessor exists, add one as a leaf helper in `internal/mutations/`:

```go
// internal/mutations/rarity.go
func RarityOf(key string) int {
	spec, ok := mutationsByKey[key]  // adapt to actual storage
	if !ok {
		return 0
	}
	return spec.Rarity
}
```

- [ ] **Step 6: Register the action**

In `internal/behaviortree/actions.go`, add to the init block (near `try_mutation_active`):

```go
actionRegistry["try_any_active_mutation"] = actTryAnyActiveMutation
```

- [ ] **Step 7: Run tests, expect PASS**

```bash
go test ./internal/behaviortree/ -v 2>&1 | tail -10
go build ./...
```

Expected: PASS.

- [ ] **Step 8: Document in context.md**

Append to `internal/behaviortree/context.md`'s action-primitive table:

```markdown
| `try_any_active_mutation` | none | Enumerates the mob's current self/AoE mutations at tick time, sorted by rarity (descending), and fires the first one that successfully triggers. Single-target mutations (`blinding-spit`, `toxic-bite`) are excluded. Coexists with `try_mutation_active`: this one is for autonomous "use whatever you have" archetypes; the other is for explicit-keys curated archetypes. |
```

Also add to the instant-action table:

```markdown
| `try_any_active_mutation` | No |
```

- [ ] **Step 9: Commit**

```bash
git add internal/behaviortree/actions_mutation.go \
        internal/behaviortree/actions_mutation_test.go \
        internal/behaviortree/actions.go \
        internal/behaviortree/context.md
# Also add internal/mutations/rarity.go if you created a helper
git commit -m "$(cat <<'EOF'
feat(btree): try_any_active_mutation dynamic-dispatch action

Enumerates a mob's current self/AoE mutations at tick time, sorted by
rarity descending (alphabetical tiebreak), and fires the first one
that successfully triggers. Single-target mutations excluded - they
need a target-resolving variant that's deferred.

Coexists with try_mutation_active (chunk 2.10's explicit-keys version):
authors choose between curated lists and autonomous dispatch per node.

Solves the runtime-evolved mutation problem - mobs/companions that
evolve a new active mutation at runtime will use it autonomously
without manual archetype edits.

Resolves project_mutation_active_runtime_evolution_btree.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Mutation damage pipeline routing — sonic-shout

**Files:**
- Modify: `internal/actions/mutation_sonic_shout.go`
- Modify: `internal/actions/mutation_sonic_shout_test.go` (update existing tests; add mitigation-applied test)

- [ ] **Step 1: Read the current sonic-shout implementation**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
cat internal/actions/mutation_sonic_shout.go
```

Find the damage formula (currently `int(char.Stats.Willpower.ValueAdj) * 8 / 100` per the chunk 2.10 audit). Note where it's used to apply damage to victim mobs in the AoE loop.

- [ ] **Step 2: Inspect the unified damage pipeline API**

```bash
grep -n "func CalcRawDamage\|func ApplyMitigation\|ChannelConviction\|ChannelPhysical" internal/combat/damage_pipeline.go
grep -n "func.*GetConvictionMitigation\|func.*GetPhysicalMitigation" internal/characters/
grep -n "ConvictionMitigationCap\|PhysicalMitigationCap" internal/configs/
```

Confirm:
- `combat.CalcRawDamage(stat, skillRank, itemMult, channel)` signature
- `combat.ApplyMitigation(raw, mitigation%, cap)` signature
- `character.GetConvictionMitigation()` method exists
- `cfg.ConvictionMitigationCap` config knob exists
- `combat.ChannelConviction` constant exists

If any of these don't exist with the expected name, adapt the implementation to whatever DOES exist (read CLAUDE.md's "Unified Damage & Mitigation Pipeline (Stage 34)" section for the canonical names).

- [ ] **Step 3: Update the failing test**

In `internal/actions/mutation_sonic_shout_test.go`, find the test that exercises damage application (or add one). Verify it now expects pipeline-mitigated output:

```go
func TestTriggerSonicShout_MitigationAppliesViaPipeline(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"sonic-shout": 1}
	attacker.Character.Stamina = 100
	attacker.Character.Stats.Willpower.Value = 200

	room := newTestRoomBare(t)
	mitigatedVictim := newTestMobInRoom(t, room)
	// Set mitigatedVictim with high conviction-mitigation gear
	// (cheating: directly set the mitigation field if possible, or
	// equip a high-mitigation item)
	mitigatedVictim.Character.Stats.Charisma.Value = 200  // for any conviction-related calcs

	// Put attacker in combat:
	// attacker.Character.SetAggro(... however)

	actor := NewMobActorInRoom(attacker, room)
	res := TriggerSonicShout(actor, MutationOpts{})

	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// Damage assertion: don't pin to exact numbers (pipeline produces
	// variance via dice.RollStat), but assert the victim DID take some
	// damage AND that a hypothetical zero-mitigation victim would have
	// taken MORE damage. The second half is harder to test directly;
	// instead assert that:
	// - mitigated victim's health DROPPED (some damage applied)
	// - the damage was less than the raw input (mitigation was applied)
}
```

Adapt the test to whatever the actions_test infrastructure permits. If exact damage assertion is too brittle, just assert "victim health decreased" + "stun condition applied" + "Triggered=true."

- [ ] **Step 4: Run test, expect FAIL or unchanged (depending on current behavior)**

```bash
go test ./internal/actions/ -run TestTriggerSonicShout -v
```

Document the baseline.

- [ ] **Step 5: Refactor sonic-shout damage to use the pipeline**

In `internal/actions/mutation_sonic_shout.go`, find the AoE damage loop. Replace the raw arithmetic with the pipeline call:

```go
// OLD:
// dmg := int(char.Stats.Willpower.ValueAdj) * 8 / 100

// NEW:
cfg := configs.GetBalanceConfig()
raw := combat.CalcRawDamage(
    float64(char.Stats.Willpower.ValueAdj),
    float64(char.GetSkillLevel(skills.UnarmedCombat)),
    1.0,                       // no item multiplier
    combat.ChannelConviction,  // routes through conviction_mitigation
)
mitigated := combat.ApplyMitigation(raw,
    victim.Character.GetConvictionMitigation(),
    cfg.ConvictionMitigationCap,
)
dmg := dice.RollStat(mitigated).Value
if dmg < 0 {
    dmg = 0
}
```

Apply this inside the per-victim loop so each victim gets their own mitigation calculation. The Willpower input is the attacker's; the mitigation is per-victim.

Update the godoc NOTE comment from the chunk-2.10 lift — replace "pre-existing damage-pipeline bypass preserved verbatim" with "damage routed through combat.CalcRawDamage + ApplyMitigation via Conviction channel; defenders' conviction_mitigation gates incoming sonic damage."

Damage messages should still use `combat.GetDamageDescription` (already in place from chunk 2.10).

- [ ] **Step 6: Run tests, expect PASS**

```bash
go test ./internal/actions/ -run TestTriggerSonicShout -v
go test ./internal/actions/ -v 2>&1 | tail -5
go build ./...
```

Expected: PASS, no regressions.

- [ ] **Step 7: Commit**

```bash
git add internal/actions/mutation_sonic_shout.go \
        internal/actions/mutation_sonic_shout_test.go
git commit -m "$(cat <<'EOF'
refactor(actions): route mutation_sonic_shout damage through pipeline

Damage previously computed as raw int(Wil * 0.08); now flows through
combat.CalcRawDamage(Wil, UnarmedCombat rank, 1.0, ChannelConviction)
+ ApplyMitigation against the victim's conviction_mitigation +
dice.RollStat for variance.

Willpower stays as the input stat (preserves the "willful shout"
character) but defense now routes through the Conviction channel,
which matches the disruptive/mental-impact theme of the mutation
better than physical or magical mitigation would.

Behavioral change: magnitudes shift versus the prior raw arithmetic.
High-conviction-mitigation targets take less; zero-mitigation targets
take roughly the pipeline's ChannelScale-derived value. Patch notes
flag this for player awareness.

Partial resolution of project_mutation_damage_pipeline_bypass (poison
DoT magnitude pipeline still pending in next task).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Mutation damage pipeline routing — toxic-bite

**Files:**
- Modify: `internal/actions/mutation_toxic_bite.go`
- Modify: `internal/actions/mutation_toxic_bite_test.go`

- [ ] **Step 1: Read the current toxic-bite implementation**

```bash
cat internal/actions/mutation_toxic_bite.go
```

Find the bite damage formula (currently `int(char.Stats.Strength.ValueAdj) * 6 / 100`) AND the poison magnitude formula (currently `float64(char.Stats.Vitality.ValueAdj) * 0.04`).

- [ ] **Step 2: Update the failing test**

In `internal/actions/mutation_toxic_bite_test.go`, add or update a test that verifies bite damage flows through the physical pipeline:

```go
func TestTriggerToxicBite_BiteDamageViaPipeline(t *testing.T) {
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"toxic-bite": 1}
	attacker.Character.Stamina = 100
	attacker.Character.Stats.Strength.Value = 200

	room := newTestRoomBare(t)
	victim := newTestMobInRoom(t, room)

	// Combat setup
	// ...

	actor := NewMobActorInRoom(attacker, room)
	target := NewMobActorInRoom(victim, room)
	res := TriggerToxicBite(actor, MutationOpts{TargetActor: target})

	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	// Assert: victim took some damage (health dropped) + poisoned condition
	// applied. Don't pin exact damage.
}
```

- [ ] **Step 3: Run test, expect FAIL or unchanged**

```bash
go test ./internal/actions/ -run TestTriggerToxicBite -v
```

- [ ] **Step 4: Refactor toxic-bite damage to use the pipeline**

In `internal/actions/mutation_toxic_bite.go`, find the bite damage block. Replace:

```go
// OLD:
// biteDamage := int(char.Stats.Strength.ValueAdj) * 6 / 100

// NEW:
cfg := configs.GetBalanceConfig()
raw := combat.CalcRawDamage(
    float64(char.Stats.Strength.ValueAdj),
    float64(char.GetSkillLevel(skills.UnarmedCombat)),
    1.0,
    combat.ChannelPhysical,
)
mitigated := combat.ApplyMitigation(raw,
    target.Character.GetPhysicalMitigation(),  // adapt to actual API
    cfg.PhysicalMitigationCap,
)
biteDamage := dice.RollStat(mitigated).Value
if biteDamage < 1 {
    biteDamage = 1
}
```

**Important — DO NOT touch the poison DoT magnitude.** The `float64(char.Stats.Vitality.ValueAdj) * 0.04` calculation for poison stays raw. DoT magnitudes work differently; a future followup will address them. Add or update the godoc NOTE to reflect this:

```go
// NOTE: bite damage now routed through combat.CalcRawDamage + Physical
// channel. Poison DoT magnitude still uses raw Vit*0.04 — pipeline
// routing for over-time damage is a separate followup
// (project_poison_dot_magnitude_pipeline).
```

Damage description message should use `combat.GetDamageDescription` if it's not already (no-hard-numbers rule).

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/actions/ -run TestTriggerToxicBite -v
go test ./internal/actions/ -v 2>&1 | tail -5
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/actions/mutation_toxic_bite.go \
        internal/actions/mutation_toxic_bite_test.go
git commit -m "$(cat <<'EOF'
refactor(actions): route mutation_toxic_bite damage through pipeline

Bite damage previously computed as raw int(Str * 0.06); now flows
through combat.CalcRawDamage(Str, UnarmedCombat rank, 1.0,
ChannelPhysical) + ApplyMitigation against the victim's
physical_mitigation + dice.RollStat.

Poison DoT magnitude (Vit * 0.04) intentionally left as raw
arithmetic - DoT magnitudes work differently from instant damage
and need a separate pipeline variant. Logged as
project_poison_dot_magnitude_pipeline.

Behavioral change: bite damage magnitudes shift versus the prior raw
arithmetic; high-physical-mitigation targets take less. Patch notes
flag this.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Mob `lock` and `unlock` verbs

**Files:**
- Possibly create: `internal/actions/lock.go` and `internal/actions/unlock.go` (if lift path chosen)
- Possibly create: `internal/actions/lock_test.go` and `internal/actions/unlock_test.go`
- Create: `internal/mobcommands/lock.go`
- Create: `internal/mobcommands/unlock.go`
- Create: `internal/mobcommands/lock_test.go`
- Create: `internal/mobcommands/unlock_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `"lock"` and `"unlock"`)
- Possibly modify: `internal/usercommands/lock.go` and `unlock.go` (if lift path chosen — collapse to thin wrappers)

- [ ] **Step 1: Read the player implementations**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
cat internal/usercommands/lock.go
cat internal/usercommands/unlock.go
```

Identify the core logic:
- Target resolution (container by name in current room)
- Key-matching against carrier's inventory
- Lock state mutation (SetLocked / SetUnlocked on the container)
- Player-facing messages

Identify player-specific bits:
- Keyring storage (`SetKey(key-<lockid>, <itemid>)`) — this is for remembering which key opens which lock
- "Add to keyring" messaging

Mobs don't have a keyring concept — they just check inventory each time.

- [ ] **Step 2: Decide lift vs standalone**

If the core logic (target resolution + key matching + state mutation + messaging) can be cleanly separated from the keyring bookkeeping, lift to `actions/`. If the keyring intertwines with the matching logic, standalone mob versions are cleaner.

**Decision rule:** look at how many lines are dedicated to keyring management vs the lock mechanic. If keyring is <20% of the file, lift. If it's >40%, standalone mob versions.

If choosing the **standalone path**: skip Step 3 (actions/ lift) and go to Step 6 (mob implementation).

If choosing the **lift path**: continue with Step 3.

- [ ] **Step 3 (lift path only): Write failing tests for actions.Lock / actions.Unlock**

Create `internal/actions/lock_test.go`:

```go
package actions

import (
	"testing"
)

func TestLock_NoTarget(t *testing.T) {
	mob := newTestMobBare(t)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := Lock(actor, LockOpts{ContainerName: ""})
	if res.Succeeded {
		t.Fatal("expected failure with empty container name")
	}
	if res.BlockReason != "no-target" {
		t.Errorf("expected BlockReason=no-target, got %s", res.BlockReason)
	}
}

// Add: TestLock_AlreadyLocked, TestLock_NoKey, TestLock_Success_PutsToInventory
// Mirror for Unlock.
```

The container infrastructure setup is non-trivial for unit tests. If too painful, fall back to: only basic API-shape tests + integration verified via the mob smoke test.

- [ ] **Step 4 (lift path only): Run test, expect failure**

- [ ] **Step 5 (lift path only): Implement actions.Lock / actions.Unlock**

Create `internal/actions/lock.go`:

```go
package actions

type LockOpts struct {
	ContainerName string  // noun for the container in the room
}

type LockResult struct {
	Succeeded   bool
	BlockReason string  // "no-target", "no-key", "already-locked", "no-room"
}

func Lock(actor Actor, opts LockOpts) LockResult {
	// Port the player implementation, IsPlayer-gating any user-only
	// text (which is most of it). Drop the keyring bookkeeping for
	// the lift; the wrapper layer adds it back for users.

	// 1. Resolve container in actor.GetRoom() by name
	// 2. Check current lock state
	// 3. Check actor's inventory for matching key
	// 4. SetLocked on the container
	// 5. Emit message (IsPlayer-gated user text; room broadcast)
	// 6. Return LockResult
}
```

Similar shape for `internal/actions/unlock.go` (returns `UnlockResult`).

- [ ] **Step 6: Implement mob wrappers**

Whether lifted or standalone, the mob wrappers go at:

`internal/mobcommands/lock.go`:

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Lock fires the lock action for a mob actor. Thin wrapper over
// actions.Lock (lift path) OR direct implementation (standalone path).
//
// LIFT PATH version:
func Lock(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.Lock(actor, actions.LockOpts{ContainerName: rest})
	return true, nil
}
```

If standalone path: inline the logic (target resolution, key check, state mutation) in `internal/mobcommands/lock.go` directly. ~80-100 LoC.

`internal/mobcommands/unlock.go` follows the same pattern.

- [ ] **Step 7: Smoke tests for the mob wrappers**

Create `internal/mobcommands/lock_test.go`:

```go
package mobcommands

import "testing"

func TestLock_RoutesProperly(t *testing.T) {
	mob := newTestMob(t)
	room := newTestRoom(t)

	handled, err := Lock("lockbox", mob, room)
	if !handled || err != nil {
		t.Fatalf("expected (true, nil), got (%v, %v)", handled, err)
	}
}

func TestLock_CommandRegistered(t *testing.T) {
	if _, ok := mobCommands["lock"]; !ok {
		t.Fatal("lock not registered in mobCommands")
	}
}
```

Mirror for unlock.

- [ ] **Step 8: Register in mobcommands.go**

In `internal/mobcommands/mobcommands.go`, add (alphabetically near `look`):

```go
"lock":   {Lock, false},
"unlock": {Unlock, false},
```

Sort alphabetically — `lock` goes between `kick` and `look`; `unlock` likely near the end before `wander`.

- [ ] **Step 9: Build + test**

```bash
go build ./...
go test ./internal/actions/ ./internal/mobcommands/ ./internal/usercommands/ 2>&1 | tail -10
```

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/mobcommands/lock.go internal/mobcommands/unlock.go \
        internal/mobcommands/lock_test.go internal/mobcommands/unlock_test.go \
        internal/mobcommands/mobcommands.go
# Add the actions/ files if lift path was taken:
# internal/actions/lock.go internal/actions/unlock.go
# internal/actions/lock_test.go internal/actions/unlock_test.go
# internal/usercommands/lock.go internal/usercommands/unlock.go
git commit -m "$(cat <<'EOF'
feat(mobcommands): lock and unlock mob verbs

Adds the mob-side counterparts to the player lock/unlock commands.
Mobs use these to interact with their own locked containers
(forager chests workflow). Key-matching uses the same
key.UniqueId == lock.UniqueId model; mob inventory checked for a
matching key item.

[NOTE: include in commit message whether lift or standalone path was
taken, and the rationale.]

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: `try_store_excess` btree primitive

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` (or create `actions_forager_storage.go` if file is crowded)
- Modify: `internal/behaviortree/actions.go` (register `try_store_excess`)
- Add: tests in appropriate behaviortree test file
- Modify: `internal/behaviortree/context.md`

- [ ] **Step 1: Read the existing forager btree action shape**

```bash
cat internal/behaviortree/actions_forager.go | head -120
cat internal/behaviortree/actions_forager_verbs.go
```

Note how existing forager actions issue `mob.Command(...)` calls (e.g., `mob.Command(fmt.Sprintf("pathto %d", target))`). The new primitive uses the same pattern.

- [ ] **Step 2: Write the failing test**

Add to an appropriate test file (e.g., `internal/behaviortree/actions_forager_test.go`):

```go
func TestTryStoreExcess_NoItems(t *testing.T) {
	// Mob with empty satchel -> Failure
	mob := newTestMobInst(t)
	mob.Character.Items = nil
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	res := actTryStoreExcess(map[string]any{"chest_room": 100}, ctx)
	if res != Failure {
		t.Errorf("expected Failure with empty satchel, got %v", res)
	}
}

func TestTryStoreExcess_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_store_excess"]; !ok {
		t.Fatal("try_store_excess not registered in actionRegistry")
	}
}
```

Multi-tick state machine tests for the full pathto→unlock→put→lock flow are awkward in unit tests; cover the basic shape here and rely on the in-game smoke test for end-to-end verification.

- [ ] **Step 3: Run test, expect failure**

```bash
go test ./internal/behaviortree/ -run TryStoreExcess -v
```

- [ ] **Step 4: Implement `try_store_excess`**

Add to `internal/behaviortree/actions_forager.go` (or new file):

```go
// actTryStoreExcess walks the forager through a chest deposit cycle:
// pathto chest room -> unlock -> put items -> lock. Multi-tick:
// each tick advances one step, leveraging btree re-tick for the
// remainder.
//
// Required param:
//   chest_room (int) — the room where the lockbox lives
//
// Algorithm:
//   1. If satchel is empty: Failure (nothing to deposit).
//   2. If not in chest_room: issue pathto; Success.
//   3. If chest is locked: issue unlock; Success.
//   4. Iterate satchel items, issue "put <item> in lockbox" per item.
//      Engine handles capacity gracefully — unsuccessful puts no-op,
//      items remain in satchel for next cycle.
//   5. Issue lock; Success.
//
// The state machine transitions out of StateStoring back to
// StateRecalling on the next tick after lock succeeds. Skipping
// directly to Recalling happens via a watchdog if any single step
// repeats too long.
func actTryStoreExcess(params map[string]any, ctx *EvalContext) Result {
	chestRoom := getIntParam(params, "chest_room")
	if chestRoom == 0 {
		mudlog.Error("try_store_excess",
			"error", "missing required `chest_room` parameter",
			"instance_id", ctx.InstanceId)
		return Failure
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}

	// 1. Items check
	if len(mob.Character.Items) == 0 {
		return Failure
	}

	// 2. Room check
	if mob.Character.RoomId != chestRoom {
		mob.Command(fmt.Sprintf("pathto %d", chestRoom))
		return Success
	}

	// 3. Lock state check
	room := rooms.LoadRoom(chestRoom)
	if room == nil {
		return Failure
	}

	// Inspect the room's containers map for the chest. The container
	// noun is "lockbox" by convention (verify against the existing
	// Halix/Kessa room YAMLs). If the noun varies, parameterize via
	// an optional `container_name` param.
	container, ok := room.Containers["lockbox"]
	if !ok {
		mudlog.Warn("try_store_excess",
			"warn", "chest_room has no lockbox container",
			"chest_room", chestRoom, "instance_id", ctx.InstanceId)
		return Failure
	}

	if container.Lock.IsLocked() {  // adapt to actual API
		mob.Command("unlock lockbox")
		return Success
	}

	// 4. Deposit
	for _, item := range mob.Character.Items {
		mob.Command(fmt.Sprintf("put %s in lockbox", item.Name()))
	}

	// 5. Lock
	mob.Command("lock lockbox")
	return Success
}
```

**API adaptations to verify:**
- `room.Containers["lockbox"]` — verify the actual field name on `*rooms.Room`
- `container.Lock.IsLocked()` — verify the actual lock-state-check API in `internal/gamelock/`
- `item.Name()` — verify the actual method (might be `item.GetSpec().Name` or `item.NameSimple()`)

- [ ] **Step 5: Register the action**

In `internal/behaviortree/actions.go`:

```go
actionRegistry["try_store_excess"] = actTryStoreExcess
```

- [ ] **Step 6: Document in context.md**

Append to `internal/behaviortree/context.md`'s action-primitive table:

```markdown
| `try_store_excess` | `chest_room` (int, required) | Forager chest-deposit workflow. Multi-tick: pathto chest room → unlock → put items → lock. Skips itself if satchel is empty. Engine handles chest-full gracefully — failed puts no-op and items stay in satchel for next cycle. |
```

And the instant-action table:

```markdown
| `try_store_excess` | No |
```

(`No` here means it's not a delayed-action — but it IS multi-tick because it issues `mob.Command` calls that resolve over rounds.)

- [ ] **Step 7: Build + test**

```bash
go build ./...
go test ./internal/behaviortree/ -v 2>&1 | tail -5
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/behaviortree/actions_forager.go \
        internal/behaviortree/actions.go \
        internal/behaviortree/context.md
# Plus the test file modifications
git commit -m "$(cat <<'EOF'
feat(btree): try_store_excess primitive for forager chest deposits

Multi-tick workflow: pathto chest room -> unlock -> put items -> lock.
Skips itself when satchel is empty. Chest-full handling is graceful
(failed puts no-op; items stay in satchel for next cycle).

Required param: chest_room (the room where the forager's lockbox
lives). The forager state machine's new StateStoring branch (next
task) consults the mob's storage_chest_room field for this value.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Tova's new dwelling room

**Files:**
- Create: `_datafiles/world/dogmud/rooms/stillwater_marsh/<NEW_ROOM_ID>.yaml`
- Modify: `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml` (change `fold_anchor_room`)
- Modify: an existing marsh room to add an exit into Tova's new dwelling (for adjacency)

- [ ] **Step 1: Allocate a free room ID in the marsh zone**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
python tools/id_inventory.py --zone stillwater_marsh --type rooms
```

Take the "next free ID" value. Use it as `<NEW_ROOM_ID>` in subsequent steps.

- [ ] **Step 2: Coord-collision scan (required per feedback_zone_coord_planning)**

List all occupied coords in the marsh zone AND geographically-adjacent zones:

```bash
grep -A 3 "^coord:" _datafiles/world/dogmud/rooms/stillwater_marsh/*.yaml \
                    _datafiles/world/dogmud/rooms/stillwater/*.yaml | \
    grep -E "x:|y:|z:" | head -100
```

Build a list of `(x, y, z)` triples that are occupied. Find Tova's existing wander territory rooms (you'll need to identify these from her forager profile or current anchor room neighbors) and locate a free coordinate adjacent to one of them.

Pick the coordinate. Verify it does NOT appear in the occupied list. If it does, pick a different free adjacent coordinate.

- [ ] **Step 3: Pick the room Tova's new dwelling connects to**

Tova's current `fold_anchor_room` is 4123 (Temple of Stillwater, public). The new dwelling should be in the marsh, near her wander territory — NOT inside the temple. Pick an existing marsh room as the entry point.

```bash
grep -l "spawninfo:" _datafiles/world/dogmud/rooms/stillwater_marsh/*.yaml | head -5
```

Read 1-2 candidate rooms to verify they're geographically near Tova's actual foraging area (the marsh, not the town).

Pick one. Note its current exits to find a free cardinal direction for the new exit.

- [ ] **Step 4: Write Tova's new dwelling room YAML**

Create `_datafiles/world/dogmud/rooms/stillwater_marsh/<NEW_ROOM_ID>.yaml`:

```yaml
roomid: <NEW_ROOM_ID>
zone: Stillwater Marsh
title: Tova's Reedwoven Hut
description: >
  A small <ansi fg="itemname">stilted hut</ansi> woven from
  reed-bundles and lashed cane, built on a low platform above the
  marsh shallows. The inside is dim and dry, with a rolled
  <ansi fg="itemname">bedding pallet</ansi> against one wall, a
  small <ansi fg="itemname">drying-rack</ansi> hung with bundles
  of marsh herbs and a half-cleaned fish-skin, and a tin
  <ansi fg="itemname">kettle</ansi> on a small cold stove. A
  patient quiet pervades the place — the hush of someone who has
  lived alone in the reeds long enough that the marsh's own quiet
  has soaked in. An ironbound
  <ansi fg="itemname">lockbox</ansi> sits tucked beneath the
  bedding pallet, hand-cut hardwood with a brass hasp worn smooth
  from use.
biome: marsh
coord:
  x: <FREE_X>
  y: <FREE_Y>
  z: 0
exits:
  <DIRECTION_BACK_TO_ENTRY_ROOM>:
    roomid: <ENTRY_ROOM_ID>
nouns:
  stilted hut: |
    A small hut of woven reed-bundles and lashed cane, built on
    a low platform above the marsh shallows. The roof is thatched
    with the same reed; the floor is dry pine plank set on cross
    timbers above the waterline.
  bedding pallet: |
    A rolled pallet of canvas-wrapped marsh-down, with a folded
    wool blanket at the foot. Beneath the pallet, an ironbound
    lockbox is tucked away out of casual sight.
  drying-rack: |
    A simple frame of hazel rods strung across one wall, hung with
    bundles of marsh herbs (sweetflag, calamus, marshmallow root),
    a half-cleaned fish-skin in slow patient curing, and a coil of
    reed-cord for the next bundle of cuttings.
  kettle: |
    A small tin kettle on a cold flatstone stove. Tova boils her
    tea here when she comes home; the stove is banked rather than
    extinguished so a small handful of dry reed will catch it
    quickly again.
  lockbox: |
    A small ironbound lockbox tucked beneath the bedding pallet —
    Tova the marsh forager's, hand-cut hardwood with a brass hasp
    and a lock kept well-oiled but worn smooth from patient use.
    She keeps the overflow of her cycles here.
mutators:
- mutatorid: sanctuary
idlemessages:
- a reed sways and clicks softly against the platform
- ''
- water laps quietly at the stilts below
- ''
- a marsh-bird calls once, far off, and answers itself
- ''
- a coal settles in the flatstone stove with a quiet tick
spawninfo:
- mobid: 371        # Tova, Marsh Forager
containers:
  lockbox:
    lock:
      difficulty: 10
      relockinterval: 24 hours
      rotationseed: 1
    items: []
```

Replace `<NEW_ROOM_ID>`, `<FREE_X>`, `<FREE_Y>`, `<DIRECTION_BACK_TO_ENTRY_ROOM>`, `<ENTRY_ROOM_ID>` with the values from Steps 1-3.

- [ ] **Step 5: Add the reverse exit on the entry room**

Modify the room you picked in Step 3 (the entry room). Add an exit pointing at the new dwelling:

```yaml
exits:
  ...existing exits...
  <DIRECTION_INTO_HUT>:
    roomid: <NEW_ROOM_ID>
```

Pick a direction that's free (not already used as an exit on the entry room).

**Check for instance saves:** per the CLAUDE.md "Room Instance Saves" SOP, if `_datafiles/world/dogmud/rooms.instances/stillwater_marsh/<ENTRY_ROOM_ID>.yaml` exists, delete it so the engine loads the fresh template with the new exit:

```bash
ls _datafiles/world/dogmud/rooms.instances/stillwater_marsh/ 2>/dev/null | head -5
# If <ENTRY_ROOM_ID>.yaml exists:
rm _datafiles/world/dogmud/rooms.instances/stillwater_marsh/<ENTRY_ROOM_ID>.yaml
```

- [ ] **Step 6: Update Tova's mob YAML**

Modify `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml`:

```yaml
# Change:
fold_anchor_room: 4123
# To:
fold_anchor_room: <NEW_ROOM_ID>
```

This makes Tova's home the new private hut instead of the public temple.

**Note:** the `storage_chest_room` field will be added in Task 9 (alongside the key updates). Don't add it here.

- [ ] **Step 7: Boot test**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build .
# Run with a short test: boot the server, wait for "all data loaded"
# log lines, then SIGINT. Verify no panic.
timeout 10 go run . 2>&1 | grep -E "loadedCount|panic|fatal" | head -20
```

Expected: no panic. Tova's mob loads cleanly with the new anchor room.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/rooms/stillwater_marsh/<NEW_ROOM_ID>.yaml \
        _datafiles/world/dogmud/rooms/stillwater_marsh/<ENTRY_ROOM_ID>.yaml \
        _datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml
git commit -m "$(cat <<'EOF'
feat(content): Tova's reedwoven hut + lockbox

Tova now has a private dwelling in Stillwater Marsh: a small stilted
hut adjacent to <ENTRY_ROOM_NAME>, containing a bedding pallet,
drying-rack, and ironbound lockbox. fold_anchor_room moves from 4123
(Temple of Stillwater, public) to the new hut.

Lockbox follows the same setup as Halix and Kessa's chests
(difficulty 10, 24-hour relock interval, rotationseed 1).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Keys + mob inventory + storage_chest_room field

**Files:**
- Possibly modify: `_datafiles/world/dogmud/items/keys-*/` (allocate IDs)
- Create: 3 new key item YAMLs (one per forager)
- Modify: `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml`
- Modify: `_datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml`
- Possibly modify: the 3 chest-room YAMLs (Tova's new hut + Halix's 3040 + Kessa's 4197) to add per-chest key identifiers

- [ ] **Step 1: Inspect the existing key item structure**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
find _datafiles/world/dogmud/items/ -name "*key*" | head -5
cat _datafiles/world/dogmud/items/<sample-key>.yaml
```

Note the YAML shape for keys: itemid, name, description, weight, the `keyid` or equivalent matching field.

- [ ] **Step 2: Allocate 3 free item IDs for the keys**

```bash
python tools/id_inventory.py --type items
```

Use 3 sequential free IDs in the keys range (or whatever range keys live in; check the Item Filepath Gotcha note in CLAUDE.md).

- [ ] **Step 3: Inspect how chest-key matching works**

Look at the existing chests (Halix's 3040, Kessa's 4197) and determine how the chest's lock identifies the matching key:

```bash
grep -B 2 -A 5 "rotationseed\|keyid\|lockid" \
    _datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml \
    _datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml
```

If chests already have a `lockid` or similar identifier, each key's YAML needs a `keyid` matching that. If not, the matching is via `rotationseed` or item-id-equivalence — read `internal/gamelock/` or the unlock.go code path to determine the exact mechanism.

If chests don't already have unique identifiers but need them, add a unique `lockid` (e.g., `lockid: forager-tova`, `forager-halix`, `forager-kessa`) to each chest's `containers.lockbox.lock` block.

- [ ] **Step 4: Write the 3 key item YAMLs**

Path will be something like `_datafiles/world/dogmud/items/keys-30000/<ID>-<name>.yaml` (verify the actual key-item directory layout):

```yaml
itemid: <KEY_ID_TOVA>
name: Tova's Lockbox Key
description: >
  A small brass key on a leather thong, worn smooth from years of
  patient use. The bit is hand-cut to a single careful pattern.
itemtype: key
keyid: forager-tova   # adapt to the actual matching field
weight: 1
```

Similar files for Halix (`forager-halix`) and Kessa (`forager-kessa`).

- [ ] **Step 5: Update the chest room YAMLs to declare the lock identifier**

For each of the 3 forager rooms (Tova's new hut, Halix's 3040, Kessa's 4197), if the chest doesn't already have a `lockid`, add one matching the corresponding key:

```yaml
containers:
  lockbox:
    lock:
      difficulty: 10
      relockinterval: 24 hours
      rotationseed: 1
      lockid: forager-tova   # add this line; corresponds to the key
    items: []
```

If `lockid` is the wrong field name, use whatever the gamelock package expects.

- [ ] **Step 6: Add keys to each forager's mob YAML**

Modify `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml`:

```yaml
  # Add to equipment (or to a new inventory/items field per the
  # actual mob YAML schema):
  equipment:
    weapon:
      itemid: 10033
    belt:
      itemid: 20059
  inventory:
  - itemid: <KEY_ID_TOVA>
```

If `inventory` isn't the right field, use whatever the existing patterns show (check other mobs that carry items).

Add `storage_chest_room: <NEW_ROOM_ID>` (the room ID from Task 8) to Tova's YAML.

Similar updates for Halix (`storage_chest_room: 3040`, add Halix's key) and Kessa (`storage_chest_room: 4197`, add Kessa's key).

- [ ] **Step 7: Boot test**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build .
timeout 10 go run . 2>&1 | grep -E "loadedCount|panic|fatal" | head -20
```

Expected: clean load. Items + mobs + rooms all parse without panics.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/items/keys-*/ \
        _datafiles/world/dogmud/rooms/stillwater_marsh/<TOVA_NEW_ROOM>.yaml \
        _datafiles/world/dogmud/rooms/ironwind_steppe/3040.yaml \
        _datafiles/world/dogmud/rooms/the_fernway_south/4197.yaml \
        _datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml \
        _datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml \
        _datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml
git commit -m "$(cat <<'EOF'
feat(content): forager chest keys + storage_chest_room field

Three new key items (one per forager), matched to each forager's
existing lockbox via lockid. Mobs Tova/Halix/Kessa gain the
storage_chest_room field pointing at the room containing their
lockbox, and carry the matching key in inventory.

Halix and Kessa's existing rooms (3040, 4197) gain a lockid on
the chest. Tova's new hut gets the same setup.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: StateStoring + forager archetype wiring

**Files:**
- Modify: `internal/forager/state.go` (add `StateStoring`)
- Modify: `internal/forager/state_test.go` (add transition tests)
- Modify: `internal/behaviortree/actions_forager.go` (handle StateStoring in the state-switch)
- Modify: `_datafiles/world/dogmud/behaviors/<forager-archetype>.yaml` (add StateStoring branch)

- [ ] **Step 1: Read the existing state enum + transitions**

```bash
cat internal/forager/state.go
cat internal/forager/state_test.go
```

Note the existing states (`StateResting`, `StateTraveling`, `StateForaging`, `StateDelivering`, `StateRecalling`) and the allowed-transitions map.

- [ ] **Step 2: Write failing transition tests**

In `internal/forager/state_test.go`, add tests for the new transitions:

```go
func TestStateTransition_DeliveringToStoring(t *testing.T) {
	cases := []struct {
		from, to State
	}{
		{StateDelivering, StateStoring},
		{StateStoring, StateRecalling},
		// Existing transition still valid:
		{StateDelivering, StateRecalling},
	}
	for _, c := range cases {
		if !c.from.CanTransitionTo(c.to) {
			t.Errorf("expected %s -> %s to be allowed", c.from, c.to)
		}
	}
}

func TestStateStoring_Stringer(t *testing.T) {
	if got := StateStoring.String(); got != "storing" {
		t.Errorf("StateStoring.String() = %q, want \"storing\"", got)
	}
}
```

Adapt to the actual transition-validation API in `state.go` (might be `CanTransitionTo`, or a transitions map keyed by from-state).

- [ ] **Step 3: Run tests, expect failure**

```bash
go test ./internal/forager/ -run StateTransition_DeliveringToStoring -v
```

Expected: FAIL (StateStoring undefined OR transition rejected).

- [ ] **Step 4: Add `StateStoring` + transitions**

In `internal/forager/state.go`, add the new state to the enum and update the transition table:

```go
const (
	StateResting State = iota
	StateTraveling
	StateForaging
	StateTravelingToDropoff  // verify this exists; adapt if not
	StateDelivering
	StateStoring             // NEW
	StateRecalling
)

var stateNames = map[State]string{
	StateResting:            "resting",
	StateTraveling:          "traveling",
	StateForaging:           "foraging",
	StateTravelingToDropoff: "traveling-to-dropoff",
	StateDelivering:         "delivering",
	StateStoring:            "storing",  // NEW
	StateRecalling:          "recalling",
}

var allowedTransitions = map[State][]State{
	// ...existing transitions...
	StateDelivering: {StateStoring, StateRecalling},  // NEW: add StateStoring
	StateStoring:    {StateRecalling},                // NEW state's only exit
	// ...existing transitions...
}
```

Adapt to the actual file shape (the constants/maps might be in different places).

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/forager/ -v 2>&1 | tail -10
```

- [ ] **Step 6: Wire StateStoring into the btree action dispatcher**

In `internal/behaviortree/actions_forager.go`, find the state-switch in `actForagerStep` (or wherever each state's tick handler is dispatched). Add a case for StateStoring:

```go
case forager.StateStoring:
    return tickForagerStoring(profile, mob, ctx)
```

Implement `tickForagerStoring`:

```go
func tickForagerStoring(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	// Read storage_chest_room from mob's YAML-loaded fields.
	// If not set, transition straight to Recalling (forager has no chest).
	chestRoom := getForagerStorageChestRoom(mob)  // helper to add
	if chestRoom == 0 {
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}

	// If satchel is empty, transition to Recalling.
	if len(mob.Character.Items) == 0 {
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}

	// Delegate to try_store_excess. After it returns, check if the
	// satchel is now empty AND we're in the chest room AND the chest
	// is locked again — those are the signals that the workflow is
	// complete and we should transition to Recalling.
	res := actTryStoreExcess(map[string]any{"chest_room": chestRoom}, ctx)

	// Transition out when deposit-complete heuristic fires.
	// Heuristic: satchel empty (or stuck), and chest locked, and
	// mob in chest_room. Use a turn-count watchdog to prevent
	// infinite loops if the workflow stalls.
	storingTurns := getIntFromState(ctx.MobState, "forager_storing_turns")
	storingTurns++
	ctx.MobState.Set("forager_storing_turns", strconv.Itoa(storingTurns))

	if storingTurns >= 20 || len(mob.Character.Items) == 0 {
		ctx.MobState.Set("forager_storing_turns", "0")
		transitionForager(ctx.MobState, forager.StateRecalling)
	}

	return res
}

func getForagerStorageChestRoom(mob *mobs.Mob) int {
	// Read from the mob's YAML-loaded extra fields. The exact API
	// depends on how the mobs package exposes custom fields — verify
	// against existing patterns (e.g., how fold_anchor_room is read).
	return mob.StorageChestRoom  // adapt to actual field name
}
```

The `storage_chest_room` YAML field needs to be added to the `Mob` struct in `internal/mobs/` if not already there. Check:

```bash
grep -n "FoldAnchorRoom\|fold_anchor_room\|StorageChestRoom\|storage_chest_room" internal/mobs/mobs.go
```

If `FoldAnchorRoom` exists, add `StorageChestRoom` with the same pattern.

- [ ] **Step 7: Update the forager archetype YAML**

Find the forager archetype YAML:

```bash
ls _datafiles/world/dogmud/behaviors/ | grep -i forager
```

Read the existing archetype YAML. Find the branch that handles the Delivering→Recalling transition. Replace or extend so that:
- If StateStoring is active, dispatch to `try_store_excess`.
- The Delivering branch's exit transition picks StateStoring (instead of going directly to Recalling) if the mob has `storage_chest_room` set and items in inventory.

The exact YAML edit depends on the archetype's existing shape. Read it first to understand the structure.

- [ ] **Step 8: Boot test**

```bash
go build ./...
timeout 10 go run . 2>&1 | grep -E "loadedCount|panic|fatal" | head -20
```

Expected: clean load, no panics.

- [ ] **Step 9: Commit**

```bash
git add internal/forager/state.go internal/forager/state_test.go \
        internal/behaviortree/actions_forager.go \
        internal/mobs/mobs.go \
        _datafiles/world/dogmud/behaviors/<forager-archetype>.yaml
git commit -m "$(cat <<'EOF'
feat(forager): StateStoring + try_store_excess archetype wiring

New forager state StateStoring inserted between StateDelivering and
StateRecalling. Foragers with storage_chest_room set route through
Storing after a vendor trip if they still have unsold items, executing
the unlock -> put -> lock workflow at their chest before recalling
home. Foragers without storage_chest_room skip Storing entirely.

20-round watchdog in tickForagerStoring prevents infinite loops if the
workflow stalls (e.g., chest persistently full or mob can't path to
chest room).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Memory writes

**Files (outside git, in user's memory directory):**
- `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\`

- [ ] **Step 1: Mark 5 resolved followups as Done**

For each of these files, update the frontmatter `description:` and the body to note resolution + commit hash. Also update the corresponding MEMORY.md row's "Priority" column to **Done**.

- `project_charge_trip_math_duplication.md` — resolved by Task 1's commit
- `project_surprise_attack_unification.md` — resolved by Task 2's commit
- `project_mutation_active_runtime_evolution_btree.md` — resolved by Task 3's commit (Path B + rarity-descending shipped)
- `project_mutation_damage_pipeline_bypass.md` — PARTIALLY resolved (Tasks 4 + 5 routed sonic-shout + toxic-bite bite damage; toxic-bite poison DoT magnitude pipeline routing remains as a new followup, see Step 2)
- `project_forager_locked_chest_workflow.md` — resolved by Tasks 6-10

For each: edit the file to add a "**RESOLVED 2026-05-23** via commit \<sha\>" line at the top of the body. Update MEMORY.md to show **Done** in the Priority column.

- [ ] **Step 2: Create 3 new followup memory entries**

Each follows the chunk 2.10 Stage F format (see existing project_*.md files in the memory dir for examples).

**File 1: `project_vendor_backfill_from_forager_chests.md`** (CRITICAL — surfaced by Section 4 of the spec)

```markdown
---
name: vendor-backfill-from-forager-chests
description: Forager chests (chunk 2.10-followups) are one-way drains without a chest -> vendor flow mechanism; design space sketched for the followup chunk
metadata:
  type: project
---

Chunk 2.10-followups added the forager locked-chest workflow as an
"overflow cache" — foragers deposit unsold items into a private
chest after each vendor trip. The chest design assumed the goods
would eventually flow back into vendor inventory to backfill gaps,
but no mechanism was built. Without that, chests are one-way
drains and the "overflow" promise is fiction.

**Why:** Surfaced during chunk 2.10-followups brainstorming
(2026-05-23). User flagged this gap explicitly before the spec
landed. Scope was too large to bundle into the same chunk; logged
as a deferred followup.

**How to apply:** Pick ONE of the four design directions below. All
four are viable; choice depends on which feels most in-character
for the world.

1. **Forager withdraws on next vendor trip.** Before walking to the
   vendor town, the forager unlocks their chest, takes whatever
   the vendor will buy (queried via existing pricing rules), and
   adds it to the satchel for that trip. Item flow: chest -> satchel
   -> vendor. Pros: same NPC handles both ends. Cons: forager has
   to make decisions about what to pull from chest based on vendor
   demand — adds AI complexity.

2. **Courier NPC pattern.** A new NPC archetype (delivery boy /
   merchant's apprentice / etc.) periodically walks the forager
   route, unlocks each forager's chest with a shared courier key,
   moves a load to the vendor. Pros: cleanest separation of
   concerns; the forager just gathers. Cons: needs a new NPC
   archetype + content authoring.

3. **Vendor restock querying nearby chests.** When a vendor's
   restock tick fires, query the world for nearby forager chests
   and pull items from them as part of the restock budget. Pros:
   no new NPCs; works without forager AI changes. Cons: vendor
   "knows" about chest locations magically; feels less grounded.

4. **Scheduled bulk transfer.** A time-of-day hook (Phase 3 work)
   fires at a configured hour and moves chest -> vendor for all
   foragers in batch. Pros: emergent feel; matches "weekly market
   day" worldbuilding. Cons: needs Phase 3.1 game-time hook to land
   first.

Recommendation order: option 1 first (smallest surface area,
preserves forager-centric model), option 4 last (needs Phase 3.1).

Related: [[project_forager_locked_chest_workflow]] (parent, now
resolved). When this is picked up, the forager chest workflow's
"deposit-only" v1 limitation goes away.
```

MEMORY.md row (Features & Content table):

```
| 2026-05-23 | High | [Vendor backfill from forager chests](project_vendor_backfill_from_forager_chests.md) | Chests need a chest -> vendor flow mechanism to fulfill their "overflow cache" promise; four design directions sketched |
```

**File 2: `project_poison_dot_magnitude_pipeline.md`**

```markdown
---
name: poison-dot-magnitude-pipeline
description: toxic-bite's poison DoT magnitude still uses raw Vit*0.04; future cleanup for over-time damage pipeline routing
metadata:
  type: project
---

Chunk 2.10-followups Task 5 routed toxic-bite's instant bite damage
through combat.CalcRawDamage (Physical channel) + ApplyMitigation,
but the poison DoT magnitude (`float64(Vit.ValueAdj) * 0.04`) was
intentionally left as raw arithmetic. DoT magnitudes work differently
from instant damage — they apply per-tick over a duration rather than
as a single resolved hit — and the existing pipeline doesn't have a
DoT-specific variant.

**Why:** Surfaced during chunk 2.10-followups implementation
(2026-05-23). The chunk's goal was to route instant damage through
the pipeline; DoTs need a separate design pass for tick semantics.

**How to apply:**
- Design a DoT-aware variant of CalcRawDamage that accounts for
  duration: e.g., `CalcDoTMagnitude(stat, skillRank, channel, durationRounds)`
  with the total damage roughly equal to a single instant hit but
  spread across rounds.
- OR: keep DoT magnitude as raw but apply mitigation at apply-time
  (per-tick) via the existing channel mitigation getters.
- Touches all DoT mechanics in the codebase (poison, burn, bleed if
  any). Not just toxic-bite.

Related: [[project_mutation_damage_pipeline_bypass]] (parent).
```

MEMORY.md row (Loose Followups table):

```
| 2026-05-23 | Low | [Poison DoT magnitude pipeline](project_poison_dot_magnitude_pipeline.md) | toxic-bite's poison DoT still uses raw Vit*0.04; future cleanup for over-time damage routing |
```

**File 3: `project_single_target_mutation_btree_dispatch.md`**

```markdown
---
name: single-target-mutation-btree-dispatch
description: blinding-spit and toxic-bite still aren't dispatchable from try_mutation_active or try_any_active_mutation; need target-resolving primitive
metadata:
  type: project
---

Chunk 2.10 introduced `try_mutation_active` and 2.10-followups added
`try_any_active_mutation`. Both intentionally exclude the two
single-target mutations (blinding-spit, toxic-bite) because neither
primitive resolves mob targets. A future primitive
`try_mutation_active_at_target` (or similar) needs to:

1. Resolve a target Actor (via the mob's current Aggro target, or
   via a `target` parameter in the YAML node)
2. Dispatch to actions.TriggerBlindingSpit or
   actions.TriggerToxicBite with the resolved target

**Why:** Surfaced during chunk 2.10 design (2026-05-23). The
robustness contract from chunk 2.10 made it explicit that single-
target mutations require target resolution that isn't in scope for
the dispatch primitives. Coverage for these mutations remains gated
on this followup.

**How to apply:**
- Mirror the existing `try_mutation_active` shape, but add target
  resolution. Either:
  - **Implicit**: target = mob's current Aggro target. Simple,
    matches the existing mob-wrapper resolution pattern in chunk
    2.10's blinding-spit / toxic-bite wrappers.
  - **Explicit**: target = a YAML param. More authoring control.
- Recommendation: implicit-first. Authors who want explicit control
  can compose with other btree primitives.

Related: [[project_mutation_active_runtime_evolution_btree]] (parent,
now resolved).
```

MEMORY.md row (Loose Followups table):

```
| 2026-05-23 | Medium | [Single-target mutation btree dispatch](project_single_target_mutation_btree_dispatch.md) | blinding-spit and toxic-bite need a target-resolving primitive (`try_mutation_active_at_target`) — currently undispatchable from any btree primitive |
```

- [ ] **Step 3: Verify all MEMORY.md edits**

Read MEMORY.md and confirm:
- The 5 resolved entries' Priority column is now **Done**
- The 3 new entries appear in the appropriate tables, sorted by date (newest at the bottom of the date cluster)
- All `[[name]]` links resolve to existing memory files

- [ ] **Step 4: Commit (if memory dir is git-tracked, otherwise skip)**

```bash
cd "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory"
git status 2>&1 | head -2
```

If the directory IS a git repo, commit. If not, the changes are local-only (per the chunk 2.10 precedent).

---

## Task 12: PATCH_NOTES + roadmap closeout

**Files:**
- Modify: `PATCH_NOTES.md`
- Modify: `MOB_ALIVENESS_ROADMAP.md` (append "Followup chunk shipped" note to 2.10's Shipped paragraph)

- [ ] **Step 1: Read recent PATCH_NOTES entries for format reference**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
head -100 PATCH_NOTES.md
```

Match the player-narrative + internal-cleanup format used by chunks 2.9 and 2.10.

- [ ] **Step 2: Add PATCH_NOTES entry**

At the top of `PATCH_NOTES.md` (above the chunk 2.10 entry), add:

```markdown
## 2026-05-XX — Mob Aliveness 2.10 Followups

**Foragers stash their overflow.** Tova, Halix, and Kessa now bring
their unsold goods home to a locked chest at the end of each vendor
trip — unlocking, depositing, and re-locking before settling in to
rest. Tova gained a new private reedwoven hut deep in Stillwater
Marsh where her lockbox lives; Halix and Kessa use the chests they
already had in the steppe alcove and fernway camp respectively. The
chests are real locks: players with picklock skill or a stolen key
can break in, but doing so is theft and the foragers don't forget.

**Companions and mobs use any mutation they've evolved.** A new
behavior-tree primitive lets mobs autonomously fire whichever active
mutation they currently have, preferring rarer mutations first. This
matters for companions: a companion who evolves a new active mutation
during play will start using it in combat without requiring their
archetype to be hand-edited.

**Sonic shout and toxic bite hit harder against the right defenses.**
Both mutations previously dealt raw stat-derived damage that ignored
target armor. They now flow through the unified damage pipeline:
- Sonic shout (Willpower-driven) is gated by Conviction mitigation —
  resistance comes from mental resilience, not physical armor.
- Toxic bite's bite damage (Strength-driven) is gated by Physical
  mitigation — armor matters now.
- Toxic bite's poison DoT magnitude is unchanged (DoT pipeline
  routing is a separate followup).

Net effect: damage magnitudes shift in both directions depending on
the target's mitigation. High-mitigation targets take less; bare
targets take roughly what they took before.

**Internal cleanup.**
- Surprise attack lifted to `internal/actions/surprise_attack.go` —
  player and mob paths now share one implementation.
- `mobcommands/charge.go` delegates trip resolution to
  `actions.ExecuteTrip` instead of reimplementing the math.
- New `lock` and `unlock` mob verbs registered (lifted to
  `internal/actions/` OR added standalone — see commit).
- New `try_any_active_mutation` and `try_store_excess` btree primitives.
- New forager state `StateStoring` inserted between Delivering and
  Recalling for foragers with `storage_chest_room` configured.

**Known limitation flagged for future work:**
- **Forager chests are deposit-only right now.** Items go in but
  don't come back out — there's no chest-to-vendor flow mechanism
  yet. Without that, chests accumulate indefinitely and don't actually
  backfill vendor inventory the way the "overflow cache" design
  promised. A followup chunk will add one of four sketched solutions
  (forager-withdraws-on-next-vendor-trip is the leading candidate).

**Deferred to followup chunks:**
- Vendor backfill from forager chests (the missing other half of the
  overflow design — critical, high priority).
- Poison DoT magnitude pipeline routing.
- Single-target mutation btree dispatch (`try_mutation_active_at_target`).
```

- [ ] **Step 3: Update MOB_ALIVENESS_ROADMAP.md**

Find chunk 2.10's `**Shipped:**` paragraph. Append a "Followup chunk shipped" line:

```markdown
**Followup chunk shipped 2026-05-XX** (commit <merge-sha>): 5 followups
bundled — charge.go trip-math dedup, surprise-attack unification, new
`try_any_active_mutation` btree action (rarity-descending), mutation
damage pipeline routing for sonic-shout (Conviction channel) and
toxic-bite bite damage (Physical channel), forager locked-chest
workflow (new Tova dwelling, lock/unlock mob verbs, `try_store_excess`
btree primitive, `StateStoring` forager state). Plan at
`docs/superpowers/plans/2026-05-23-mob-aliveness-2.10-followups.md`.
```

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(2.10-followups): PATCH_NOTES + roadmap closeout

Player-narrative summary of the 5 followups bundle: forager chest
workflow, autonomous mutation dispatch, mutation damage pipeline
routing, surprise-attack unification, charge.go cleanup.

Roadmap chunk 2.10's Shipped paragraph extended with the followup
chunk note.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Manual smoke checklist (user-driven, post-merge)

Per chunk 2.9/2.10 precedent, in-game smoke testing is deferred to the user. Checklist:

- [ ] Server boots cleanly past data-file loading (no panics, no startup warnings beyond the 3 expected for `lock`/`unlock`/`throw` which should NOW be silent since `lock` and `unlock` are added)
- [ ] Trigger a charge attack in-game — verify approach narration fires AND knockdown still applies on success
- [ ] Spawn a hidden mob, attack it from range — verify surprise-strike still fires per weapon
- [ ] Spawn a test mob with 2 active mutations (e.g., `blinding-flash` rarity 5 + `healing-gel` rarity 3), attach a btree node `try_any_active_mutation` — verify blinding-flash fires first (rare-first ordering)
- [ ] Trigger a sonic-shout in combat — verify damage feels proportionate (no clearly broken-low or broken-high magnitude) AND conviction-mitigation-bearing target takes less
- [ ] Trigger toxic-bite — same: damage feels proportionate, physical-mitigation target takes less, poison condition still applies
- [ ] Watch Tova complete a full daily cycle — forage → vendor → if she has unsold items, walk to her new hut, unlock the lockbox, deposit items, lock it again, then return to anchor → rest. Verify the lockbox now has items in it after a few cycles
- [ ] Optionally: pick the lock on Tova's chest as a player — verify the items are recoverable

If smoke surfaces issues, fix and follow up before promoting to prod.

---

## Self-review checklist

**Spec coverage:**
- [x] Item 1 (charge.go dup) — Task 1
- [x] Item 2 (Surprise-attack unification) — Task 2
- [x] Item 3 (try_any_active_mutation) — Task 3
- [x] Item 4 (Mutation damage pipeline) — Tasks 4 + 5
- [x] Item 5a (Mob lock/unlock verbs) — Task 6
- [x] Item 5b (try_store_excess) — Task 7
- [x] Item 5c (Tova's dwelling + coord scan) — Task 8 (Step 2 = scan)
- [x] Item 5d (Keys + storage_chest_room) — Task 9
- [x] Item 5e (StateStoring + archetype) — Task 10
- [x] Cross-cutting: memory writes — Task 11 (5 resolved + 3 created, incl. vendor backfill)
- [x] Cross-cutting: PATCH_NOTES + roadmap — Task 12
- [x] Cross-cutting: testing strategy — embedded per task
- [x] Cross-cutting: smoke checklist — at end of plan
- [x] Out-of-scope items appear nowhere in tasks (correctly excluded)

**Placeholder scan:**
- `<NEW_ROOM_ID>`, `<KEY_ID_*>`, `<FREE_X>`, etc. are parameterizations resolved at implementation time via `id_inventory.py` and the coord scan — appropriate placeholders, not plan failures
- No "TBD" / "TODO" / "fill in details" / "Similar to Task N" without repeated code
- The lift-vs-standalone decision in Task 6 explicitly defers to runtime judgment with a decision rule — appropriate non-determinism

**Type consistency:**
- `actions.SurpriseAttackOpts` / `Result` used consistently in Task 2
- `actions.LockOpts` / `LockResult` introduced in Task 6 (lift path); standalone path explicitly notes the alternative
- `actTryAnyActiveMutation` referenced consistently in Task 3
- `actTryStoreExcess` referenced consistently in Task 7
- `StateStoring` named consistently across Tasks 7, 9, 10
- `storage_chest_room` (snake_case) used in YAML; `StorageChestRoom` (PascalCase) used in Go — matches existing patterns

Issues found inline: none.

---

## Execution handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-23-mob-aliveness-2.10-followups.md`.** Two execution options:

**1. Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration. Especially well-suited here because the 12 tasks are well-isolated and the spec has detailed enough per-item designs that each subagent gets full context upfront.

**2. Inline Execution** — execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
