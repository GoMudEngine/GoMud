# Mob Aliveness 2.10 — PvM/MvP/PvP/MvM Parity Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close out Phase 2 of the mob aliveness roadmap with a full
parity audit, lift the 6 `mutation_*` commands into the `actions/`
package with symmetric mob wrappers, delete the dead `selljunk` verb,
patch quick gaps inline, and surface deferred gaps for triage.

**Architecture:** Audit-first (no commits). Then a TDD actions-lift for
the 6 mutation commands mirroring the chunk 2.1 `Buy` / chunk 2.9
`Forage`/`Salvage` precedent: shared helpers in `actions/mutation_helpers.go`,
one action function per mutation, thin wrappers on both user and mob
sides, one btree action `try_mutation_active`. Quick patches and
deletions in atomic commits. Deferred-gap list surfaced as a separate
review doc for user triage before any memory entries land.

**Tech Stack:** Go 1.24, existing `internal/actions/` Actor abstraction,
existing `internal/behaviortree/actions_*.go` registry pattern, YAML
btree archetypes, Go standard testing.

**Spec:** `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md`

**Branch:** `feature/mob-aliveness-2.10-parity-audit` (already created;
spec committed as `b8b7f90e`).

---

## Stage map

| Stage | Description | Tasks |
|---|---|---|
| A | Audit walk (no Go changes) | A1, A2 (parallel) |
| B | Mutation\_\* actions-lift | B1–B8 |
| C | selljunk deletion | C1 |
| D | Quick patches (instantiated post-audit) | D-template + per-gap |
| E | Deferred-gap review doc | E1 |
| F | Memory writes (post-triage) | F1 + per-decision |
| G | Closeout | G1, G2 |

**Order:** A first (or in parallel with B), then B, then C, then D, then E
(user reviews → triage), then F, then G.

**Sessions:** likely two. Stages A–C in session 1; D–G in session 2.

---

## Stage A — Audit walk

### Task A1: Player-side audit

**Files:**
- Modify: `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md` (append "Player-side audit table" section)

**This task can be run by a research subagent.** No Go changes.

- [ ] **Step 1: Enumerate audit candidates**

Run:
```
find "internal/usercommands" -maxdepth 1 -name "*.go" \
  -not -name "*_test.go" \
  -not -name "admin.*" \
  -not -name "default.go" \
  -not -name "noop.go" \
  -not -name "enchant_slot.go" \
  -not -name "helpfile_completeness*"
```

Expected: ~95 files. List of candidate commands.

- [ ] **Step 2: For each candidate, classify**

For each command file, read the implementation and decide its verdict
against the six-bucket scheme from the spec:

| Verdict | Meaning |
|---|---|
| Equivalent | Mob has a working counterpart (verify by reading the mob file) |
| Orthogonal | One-sided by design (incl. progression-mechanic verbs like quest dialogue) |
| Never-relevant | Mob has no applicable concept |
| Gap: patch inline | ≤30 LoC lift, no new gameplay/config decision |
| Gap: delete divergent verb | Asymmetric side is dead code (verify with grep) |
| Gap: defer | Bigger / ambiguous / needs design |

Cross-reference candidate mob counterparts by both literal name match
AND by command-purpose lookup in `internal/mobcommands/`. Note that
file-name differences are common (e.g., `skill.cast` ↔ `cast`,
`skill.skullduggery.steal` ↔ `steal`).

- [ ] **Step 3: Write the table**

Append to the spec doc (in a new section titled
`## Player-side audit table`). Format:

```markdown
| Command | Mob counterpart | Verdict | Notes / file refs |
|---------|----------------|---------|-------------------|
| afk | (none) | Orthogonal | Player session state only |
| ask | converse (loose) | Orthogonal | Mob-quest acquisition is not a design goal |
| attack | attack | Equivalent | Both call through `actions.Attack` |
| ... (one row per candidate) | | | |
```

For Gap rows, include a one-line rationale of why it's a gap and what
the fix would be.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md
git commit -m "$(cat <<'EOF'
docs(2.10): player-side parity audit table

Walked the ~95 non-admin, non-meta player commands and classified each
against the six-bucket parity scheme. Gap rows note proposed direction
(patch / delete / defer).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task A2: Mob-side audit

**Files:**
- Modify: `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md` (append "Mob-side audit table" section)

**This task can run in parallel with A1.** Different file rows.

- [ ] **Step 1: Enumerate audit candidates**

Run:
```
find "internal/mobcommands" -maxdepth 1 -name "*.go" \
  -not -name "*_test.go" \
  -not -name "mobcommands.go" \
  -not -name "noop.go"
```

Expected: ~62 files. List of candidate commands.

- [ ] **Step 2: For each candidate, classify**

Same six-bucket scheme as A1. Direction reversed: "does the player side
have a counterpart, and if not, is that a gap?"

For each candidate that has no obvious player counterpart, run a grep
to check if the mob command actually has callers (btree YAML, hooks,
or AI code):

```bash
grep -rln "command-name" _datafiles/ internal/ --include="*.yaml" --include="*.go"
```

If a mob command has **zero callers** in YAML/Go (except its own
registration in `mobcommands.go` and its own test file), classify as
**Gap: delete divergent verb**. Selljunk is the case study.

- [ ] **Step 3: Write the table**

Append to spec doc (new section `## Mob-side audit table`). Same row
format as A1, with the "counterpart" column showing the player side.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md
git commit -m "$(cat <<'EOF'
docs(2.10): mob-side parity audit table

Walked the ~62 mob commands and classified each. Verified delete-verdict
candidates have zero live callers in YAML/Go before recommending deletion.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Stage B — Mutation_* actions-lift

### Task B1: Create `actions/mutation_helpers.go`

**Files:**
- Create: `internal/actions/mutation_helpers.go`
- Create: `internal/actions/mutation_helpers_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/actions/mutation_helpers_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestMutationPreamble_NoMutation(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobBare(t)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "blinding-flash", true, 14)
	if res.OK {
		t.Fatal("expected preamble to fail without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

func TestMutationPreamble_NotInCombat(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"blinding-flash": 1}
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "blinding-flash", true, 14)
	if res.OK {
		t.Fatal("expected preamble to fail when not in combat")
	}
	if res.BlockReason != "not-in-combat" {
		t.Errorf("expected BlockReason=not-in-combat, got %s", res.BlockReason)
	}
}

func TestMutationPreamble_LowStamina(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 1
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	// healing-gel doesn't require combat (combatRequired=false)
	res := mutationPreamble(actor, "healing-gel", false, 14)
	if res.OK {
		t.Fatal("expected preamble to fail with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

func TestMutationPreamble_Success(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := mutationPreamble(actor, "healing-gel", false, 14)
	if !res.OK {
		t.Fatalf("expected preamble OK, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 86 {
		t.Errorf("expected stamina deducted to 86, got %d", mob.Character.Stamina)
	}
}
```

The helpers `newTestMobBare` and `newTestRoomBare` already exist in
`internal/actions/actions_test.go` — verify before writing:

```bash
grep -n "func newTestMobBare\|func newTestRoomBare" internal/actions/actions_test.go
```

If they don't exist or don't cover what's needed, model on
`newTestMobInRoom` / similar already in that file.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/actions/ -run TestMutationPreamble -v
```

Expected: FAIL with "undefined: mutationPreamble" or similar compile error.

- [ ] **Step 3: Write `actions/mutation_helpers.go`**

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// preambleResult is the outcome of a mutation-active preamble check.
// On OK=false, BlockReason is set to one of:
// "no-mutation", "not-in-combat", "on-cooldown", "low-stamina".
type preambleResult struct {
	OK          bool
	BlockReason string
}

// mutationPreamble runs the shared "can this actor fire this mutation?"
// gates: mutation present, in combat (if required), cooldown free,
// stamina sufficient. On success, deducts staminaCost from the actor's
// character.
//
// All five "AoE/single-target" mutation actives use combatRequired=true.
// healing-gel uses combatRequired=false.
//
// Player-perspective messaging is emitted by the calling action via
// IsPlayer()-gated SendText calls; this helper only emits the silent
// gate decision in the preambleResult.
func mutationPreamble(actor Actor, mutationKey string, combatRequired bool, staminaCost int) preambleResult {
	char := actor.GetCharacter()
	if char == nil {
		return preambleResult{BlockReason: "no-character"}
	}

	if !mutations.HasMutation(char.Mutations, mutationKey) {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You don't have that ability.")
		}
		return preambleResult{BlockReason: "no-mutation"}
	}

	if combatRequired && !char.IsInCombat() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You must be in combat to use %s!", mutationKey))
		}
		return preambleResult{BlockReason: "not-in-combat"}
	}

	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				"You need a moment to recover before attempting another special move.")
		}
		return preambleResult{BlockReason: "on-cooldown"}
	}

	if char.Stamina < staminaCost {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You're too exhausted!")
		}
		return preambleResult{BlockReason: "low-stamina"}
	}
	char.Stamina -= staminaCost

	return preambleResult{OK: true}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/actions/ -run TestMutationPreamble -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Run the full actions package test suite**

```bash
go test ./internal/actions/ -v
```

Expected: PASS. No regression on existing tests.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/mutation_helpers.go internal/actions/mutation_helpers_test.go
git commit -m "$(cat <<'EOF'
feat(actions): mutation_helpers shared preamble for active mutations

New mutationPreamble helper centralises the five-step gate (mutation
presence / in-combat / cooldown / stamina) shared by the six mutation
active commands about to be lifted. Five gates emit player-perspective
text via the existing IsPlayer() gate; mob path stays silent.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B2: Lift `mutation_blinding_flash` (worked example)

**Files:**
- Create: `internal/actions/mutation_blinding_flash.go`
- Create: `internal/actions/mutation_blinding_flash_test.go`
- Modify: `internal/usercommands/mutation_blinding_flash.go` (collapse to thin wrapper)
- Create: `internal/mobcommands/mutation_blinding_flash.go`
- Create: `internal/mobcommands/mutation_blinding_flash_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `blinding_flash`)

- [ ] **Step 1: Read the existing player implementation**

```bash
cat internal/usercommands/mutation_blinding_flash.go
```

Note the exact effect: AoE blind on room mobs via opposed
`Wil + UnarmedCombat` vs `Wil + CombatSkillLevel` roll. Self-blind
afterimage with 0.7x duration. Stamina cost 14. Uses `ConditionBlinded`
with magnitude 3 round duration, 0.5x AoE / 0.7x self.

- [ ] **Step 2: Write the failing test**

Create `internal/actions/mutation_blinding_flash_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestTriggerBlindingFlash_NoMutation(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	mob.Character.SetAggro(/* synthetic combat target */ nil, "", 1, 0)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerBlindingFlash(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

func TestTriggerBlindingFlash_Success_BlindsRoomMobs(t *testing.T) {
	configs.SetTestConfig()
	attacker := newTestMobBare(t)
	attacker.Character.Mutations = map[string]int{"blinding-flash": 1}
	attacker.Character.Stamina = 100
	attacker.Character.Stats.Willpower.Value = 200 // ensure success roll

	room := newTestRoomBare(t)
	victim := newTestMobInRoom(t, room)
	victim.Character.Stats.Willpower.Value = 50 // ensure defender loses

	// Put attacker in combat so the in-combat gate passes:
	attacker.Character.SetAggro(victim, "attack", 1, 0)

	actor := NewMobActorInRoom(attacker, room)
	res := TriggerBlindingFlash(actor, MutationOpts{})

	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if res.AffectedCount < 1 {
		t.Errorf("expected AffectedCount >= 1, got %d", res.AffectedCount)
	}
	if !victim.Character.HasCondition(characters.ConditionBlinded) {
		t.Error("expected victim to have Blinded condition")
	}
	if !attacker.Character.HasCondition(characters.ConditionBlinded) {
		t.Error("expected attacker to have self-blind condition")
	}
}
```

Verify `newTestMobInRoom` exists in `actions_test.go`. If not, add it
modeled on `newTestMobBare`.

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/actions/ -run TestTriggerBlindingFlash -v
```

Expected: FAIL with "undefined: TriggerBlindingFlash, MutationOpts".

- [ ] **Step 4: Define `MutationOpts` and `MutationResult` types**

Create `internal/actions/mutation_blinding_flash.go`:

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// MutationOpts parameterizes a mutation-active trigger attempt.
// TargetActor is used only by single-target mutations (toxic-bite,
// blinding-spit). AoE mutations (blinding-flash, sonic-shout,
// pacifism-aura) ignore it. Self-only mutations (healing-gel) also
// ignore it.
type MutationOpts struct {
	TargetActor Actor
}

// MutationResult is the structured outcome of a mutation-active trigger.
// On Triggered=false, BlockReason is one of:
//   "no-character", "no-mutation", "not-in-combat", "on-cooldown",
//   "low-stamina", "no-target", "no-room".
// On Triggered=true, AffectedCount counts the number of targets that
// received the effect (1 for self/single-target, >=0 for AoE).
type MutationResult struct {
	Triggered     bool
	BlockReason   string
	AffectedCount int
}

// TriggerBlindingFlash fires the blinding-flash mutation. AoE: applies
// ConditionBlinded (magnitude 3 rounds, 0.5x AoE penalty) to each
// non-attacker mob in the room that fails an opposed Wil+UnarmedCombat
// vs Wil+CombatSkill roll. The attacker takes a milder self-blind
// (magnitude 1 round, 0.7x penalty). Stamina cost 14. Requires the
// blinding-flash mutation and in-combat status. Shares the special-move
// cooldown bucket.
func TriggerBlindingFlash(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "blinding-flash", true, 14)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	char := actor.GetCharacter()
	room := actor.GetRoom()
	if room == nil {
		return MutationResult{BlockReason: "no-room"}
	}

	attackerScore := float64(char.GetSkillLevel(skills.UnarmedCombat)) +
		float64(char.Stats.Willpower.ValueAdj)

	// Self-emit + room broadcast
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="white-bold">Blinding light erupts from your skin in a searing flash!</ansi>`)
	}
	room.SendTextVisual(messaging.CategoryMutation,
		fmt.Sprintf(`<ansi fg="white-bold">A blinding flash of light erupts from <ansi fg="username">%s</ansi>!</ansi>`,
			actor.GetName()),
		actor.GetUserId(),
	)

	// AoE blind: all mobs in room except attacker
	blindedCount := 0
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		if mobInstId == actor.GetMobInstanceId() {
			continue
		}
		victim := mobs.GetInstance(mobInstId)
		if victim == nil {
			continue
		}
		defenderScore := float64(victim.Character.Stats.Willpower.ValueAdj) +
			float64(victim.Character.GetCombatSkillLevel())
		success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)
		if success {
			victim.Character.AddCondition(characters.ConditionBlinded, 3, 0.5, "blinding-flash")
			blindedCount++
		}
	}

	if blindedCount > 0 && actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(`<ansi fg="white">Creatures are blinded by the flash!</ansi>`))
	}

	// Self-blind aftermath
	char.AddCondition(characters.ConditionBlinded, 1, 0.7, "blinding-flash self")
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="yellow">The afterimage sears your own vision briefly.</ansi>`)
	}

	actor.OnSkillUse(skills.UnarmedCombat.String())

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return MutationResult{Triggered: true, AffectedCount: blindedCount}
}
```

Note: the player command also emits a numeric count message. Per the
"no hard numbers in player-facing text" rule (CLAUDE.md), this was
incorrect in the original — replaced with a descriptive variant. No
change to game mechanics.

- [ ] **Step 5: Run tests**

```bash
go test ./internal/actions/ -run TestTriggerBlindingFlash -v
go test ./internal/actions/ -v   # full suite
```

Expected: TriggerBlindingFlash tests PASS, no regression.

- [ ] **Step 6: Rewrite the player wrapper**

Replace `internal/usercommands/mutation_blinding_flash.go`:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BlindingFlash is a thin wrapper over actions.TriggerBlindingFlash.
// All gates, scoring, effects, and messages live in the action; this
// wrapper only adapts the user record to the Actor interface.
func BlindingFlash(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	actor := actions.NewUserActorInRoom(user, room)
	_ = actions.TriggerBlindingFlash(actor, actions.MutationOpts{})
	return true, nil
}
```

Verify `NewUserActorInRoom` exists in `internal/actions/actor_user.go`.
If not, the helper name might be `NewUserActor` — check the imports of
the existing `forage.go` / `salvage.go` user wrappers and match the
pattern.

- [ ] **Step 7: Run any existing player-side tests for blinding-flash**

```bash
go test ./internal/usercommands/ -run BlindingFlash -v
```

Expected: PASS (or no matching tests, which is also OK).

- [ ] **Step 8: Create the mob wrapper**

Create `internal/mobcommands/mutation_blinding_flash.go`:

```go
package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// BlindingFlash fires the blinding-flash mutation for a mob actor. Thin
// wrapper over actions.TriggerBlindingFlash. The structured result is
// not consumed here; btree dispatch via try_mutation_active reads the
// result instead.
func BlindingFlash(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.TriggerBlindingFlash(actor, actions.MutationOpts{})
	return true, nil
}
```

- [ ] **Step 9: Create the mob-wrapper smoke test**

Create `internal/mobcommands/mutation_blinding_flash_test.go`:

```go
package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestBlindingFlash_RoutesToAction(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMob(t)             // existing helper in mobcommands_test.go
	room := newTestRoom(t)           // existing helper in mobcommands_test.go

	// Smoke: should not panic; returns true.
	handled, err := BlindingFlash("", mob, room)
	if !handled || err != nil {
		t.Fatalf("expected (true, nil), got (%v, %v)", handled, err)
	}
}
```

Verify the test helper names by reading the existing
`internal/mobcommands/mobcommands_test.go`:

```bash
grep -n "func newTestMob\|func newTestRoom" internal/mobcommands/mobcommands_test.go
```

If helper names differ, adapt accordingly.

- [ ] **Step 10: Register `blinding_flash` in mobcommands**

In `internal/mobcommands/mobcommands.go`, add inside the
`mobCommands` map (preserve alphabetical-ish ordering near `bite`):

```go
"blinding_flash": {BlindingFlash, false},
```

- [ ] **Step 11: Run mobcommands tests**

```bash
go test ./internal/mobcommands/ -run BlindingFlash -v
go test ./internal/mobcommands/ -v
```

Expected: PASS.

- [ ] **Step 12: Run the full Go build + test**

```bash
go build ./...
go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ -v
```

Expected: PASS. No regressions anywhere.

- [ ] **Step 13: Commit**

```bash
git add internal/actions/mutation_blinding_flash.go \
        internal/actions/mutation_blinding_flash_test.go \
        internal/usercommands/mutation_blinding_flash.go \
        internal/mobcommands/mutation_blinding_flash.go \
        internal/mobcommands/mutation_blinding_flash_test.go \
        internal/mobcommands/mobcommands.go
git commit -m "$(cat <<'EOF'
refactor(actions): lift mutation_blinding_flash into actions package

New TriggerBlindingFlash + MutationOpts/MutationResult types in
internal/actions/. Player wrapper collapses to ~12 lines; new mob
wrapper added and registered. Replaces the player command's raw
numeric "N creatures blinded" message with a descriptive variant
per the no-hard-numbers rule.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B3: Lift `mutation_blinding_spit`

**Files:**
- Create: `internal/actions/mutation_blinding_spit.go`
- Create: `internal/actions/mutation_blinding_spit_test.go`
- Modify: `internal/usercommands/mutation_blinding_spit.go`
- Create: `internal/mobcommands/mutation_blinding_spit.go`
- Create: `internal/mobcommands/mutation_blinding_spit_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `blinding_spit`)

**Structure mirrors B2.** Per-mutation specifics:

- **Targeting:** Single target. Uses `MutationOpts.TargetActor`; if nil,
  the action returns `BlockReason: "no-target"`.
- **Effect:** Opposed Wil+UnarmedCombat roll; on success applies
  `ConditionBlinded` magnitude 5 / multiplier 1.0.
- **Stamina cost:** 10.
- **Existing player code:** read `internal/usercommands/mutation_blinding_spit.go`
  for the exact roll formula, message strings, and any per-mutation
  follow-up effects (self-blind? no for this one).

- [ ] **Step 1: Read existing player implementation**

```bash
cat internal/usercommands/mutation_blinding_spit.go
```

- [ ] **Step 2: Write failing test** (single-target variants — no-target,
  no-mutation, success-applies-blind-on-target)

Pattern: same as B2 step 2 but with single-target assertions. Test
that omitting `TargetActor` returns `BlockReason: "no-target"` and that
providing a valid target applies the condition.

- [ ] **Step 3: Run test, expect failure**

- [ ] **Step 4: Implement `TriggerBlindingSpit` in
  `internal/actions/mutation_blinding_spit.go`**

Use `mutationPreamble(actor, "blinding-spit", true, 10)`. After preamble,
check `opts.TargetActor != nil`. Apply the effect from the player code,
using `actor.GetName()` for messaging, `actor.IsPlayer()` to gate
self-text, `room.SendTextVisual` for broadcast.

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Rewrite player wrapper** to call through. The wrapper
  must parse `rest` to resolve the target (use existing
  `actions.ResolveTargetActor` or whatever resolver the original used)
  and pass it via `MutationOpts.TargetActor`.

- [ ] **Step 7: Create mob wrapper + smoke test + register
  `blinding_spit`** in `mobcommands.go`.

- [ ] **Step 8: Run full build + tests**

- [ ] **Step 9: Commit**

```bash
git commit -m "$(cat <<'EOF'
refactor(actions): lift mutation_blinding_spit into actions package

Single-target variant of the blinding family. New TriggerBlindingSpit
in actions/, player + mob wrappers collapsed. Target resolution stays
in the wrappers; the action only consumes the resolved Actor.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B4: Lift `mutation_healing_gel`

**Files:**
- Create: `internal/actions/mutation_healing_gel.go`
- Create: `internal/actions/mutation_healing_gel_test.go`
- Modify: `internal/usercommands/mutation_healing_gel.go`
- Create: `internal/mobcommands/mutation_healing_gel.go`
- Create: `internal/mobcommands/mutation_healing_gel_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `healing_gel`)

**Structure mirrors B2.** Per-mutation specifics:

- **Targeting:** Self only.
- **Effect:** Self-heal. Heal amount derived from `floor(HealthMax * fraction)`
  per the no-flat-healing rule (CLAUDE.md regen section). Read the existing
  player code to confirm the fraction.
- **Combat gate:** `combatRequired=false` (only mutation that doesn't
  require combat). Pass `false` to `mutationPreamble`.
- **Stamina cost:** read from player code (likely ~10).
- **Existing player code:** `internal/usercommands/mutation_healing_gel.go`

- [ ] **Step 1: Read existing player implementation**

- [ ] **Step 2: Write failing test** — assert health-gain magnitude is a
  fraction of `HealthMax`, not a flat amount; assert it works out of combat.

- [ ] **Step 3: Run test, expect failure**

- [ ] **Step 4: Implement `TriggerHealingGel`**. Note the `false` arg to
  `mutationPreamble` for combat-required.

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Rewrite player wrapper**

- [ ] **Step 7: Create mob wrapper + smoke test + register `healing_gel`**

- [ ] **Step 8: Run full build + tests**

- [ ] **Step 9: Commit**

```bash
git commit -m "$(cat <<'EOF'
refactor(actions): lift mutation_healing_gel into actions package

Self-only, out-of-combat-allowed mutation. New TriggerHealingGel in
actions/. Player + mob wrappers collapsed. Heal magnitude stays as
fraction-of-max per the no-flat-heal rule.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B5: Lift `mutation_pacifism_aura`

**Files:**
- Create: `internal/actions/mutation_pacifism_aura.go`
- Create: `internal/actions/mutation_pacifism_aura_test.go`
- Modify: `internal/usercommands/mutation_pacifism_aura.go`
- Create: `internal/mobcommands/mutation_pacifism_aura.go`
- Create: `internal/mobcommands/mutation_pacifism_aura_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `pacifism_aura`)

**Structure mirrors B2.** Per-mutation specifics:

- **Targeting:** AoE on room mobs (like blinding-flash).
- **Effect:** Applies `ConditionPacified` (verify exact constant in
  `internal/characters/conditions.go`) on opposed-roll success.
- **Stamina cost:** read from player code.
- **Existing player code:** `internal/usercommands/mutation_pacifism_aura.go`

- [ ] **Step 1: Read existing player implementation**

- [ ] **Step 2: Write failing test** — AoE applies pacified to multiple
  room mobs on success.

- [ ] **Step 3: Run test, expect failure**

- [ ] **Step 4: Implement `TriggerPacifismAura`**

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Rewrite player wrapper**

- [ ] **Step 7: Create mob wrapper + smoke test + register `pacifism_aura`**

- [ ] **Step 8: Run full build + tests**

- [ ] **Step 9: Commit**

```bash
git commit -m "$(cat <<'EOF'
refactor(actions): lift mutation_pacifism_aura into actions package

AoE pacification mutation. New TriggerPacifismAura in actions/, player
+ mob wrappers collapsed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B6: Lift `mutation_sonic_shout`

**Files:**
- Create: `internal/actions/mutation_sonic_shout.go`
- Create: `internal/actions/mutation_sonic_shout_test.go`
- Modify: `internal/usercommands/mutation_sonic_shout.go`
- Create: `internal/mobcommands/mutation_sonic_shout.go`
- Create: `internal/mobcommands/mutation_sonic_shout_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `sonic_shout`)

**Structure mirrors B2.** Per-mutation specifics:

- **Targeting:** AoE on room mobs.
- **Effect:** Damage + `ConditionStunned`. Damage via
  `combat.CalcRawDamage` per the unified damage pipeline; verify the
  channel (likely magical or special) by reading the player code.
- **Stamina cost:** read from player code.
- **Existing player code:** `internal/usercommands/mutation_sonic_shout.go`

- [ ] **Step 1: Read existing player implementation**

- [ ] **Step 2: Write failing test** — AoE applies stun + damage; damage
  uses the unified pipeline (no inline magic numbers).

- [ ] **Step 3: Run test, expect failure**

- [ ] **Step 4: Implement `TriggerSonicShout`**

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Rewrite player wrapper**

- [ ] **Step 7: Create mob wrapper + smoke test + register `sonic_shout`**

- [ ] **Step 8: Run full build + tests**

- [ ] **Step 9: Commit**

```bash
git commit -m "$(cat <<'EOF'
refactor(actions): lift mutation_sonic_shout into actions package

AoE damage + stun mutation. New TriggerSonicShout in actions/, player
+ mob wrappers collapsed. Damage continues to flow through the unified
combat.CalcRawDamage pipeline.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B7: Lift `mutation_toxic_bite`

**Files:**
- Create: `internal/actions/mutation_toxic_bite.go`
- Create: `internal/actions/mutation_toxic_bite_test.go`
- Modify: `internal/usercommands/mutation_toxic_bite.go`
- Create: `internal/mobcommands/mutation_toxic_bite.go`
- Create: `internal/mobcommands/mutation_toxic_bite_test.go`
- Modify: `internal/mobcommands/mobcommands.go` (register `toxic_bite`)

**Structure mirrors B2.** Per-mutation specifics:

- **Targeting:** Single target. Uses `MutationOpts.TargetActor`.
- **Effect:** Damage + `ConditionPoisoned`. Largest of the six (169 LoC)
  — likely has additional logic around bite-skill bonus, racial
  modifiers, or armor interaction. Read carefully.
- **Stamina cost:** read from player code.
- **Existing player code:** `internal/usercommands/mutation_toxic_bite.go`

- [ ] **Step 1: Read existing player implementation carefully (largest of the six)**

- [ ] **Step 2: Write failing test** — single-target poison + damage;
  validates the bite-specific score formula matches player behavior.

- [ ] **Step 3: Run test, expect failure**

- [ ] **Step 4: Implement `TriggerToxicBite`**

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Rewrite player wrapper** — note the wrapper must still
  do target resolution.

- [ ] **Step 7: Create mob wrapper + smoke test + register `toxic_bite`**

- [ ] **Step 8: Run full build + tests**

- [ ] **Step 9: Commit**

```bash
git commit -m "$(cat <<'EOF'
refactor(actions): lift mutation_toxic_bite into actions package

Single-target damage + poison mutation. Largest of the six lifts; new
TriggerToxicBite in actions/, player + mob wrappers collapsed. Per-
mutation bite-skill bonus formula preserved.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task B8: `try_mutation_active` btree action

**Files:**
- Create: `internal/behaviortree/actions_mutation.go`
- Create: `internal/behaviortree/actions_mutation_test.go`
- Modify: `internal/behaviortree/actions.go` (register the new action)
- Modify: `internal/behaviortree/context.md` (document the action)

- [ ] **Step 1: Write the failing test**

Create `internal/behaviortree/actions_mutation_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestTryMutationActive_NoKeyOrKeys(t *testing.T) {
	configs.SetTestConfig()
	ctx := newTestContext(t)

	res := actTryMutationActive(map[string]any{}, ctx)
	if res != Failure {
		t.Errorf("expected Failure with neither key nor keys, got %v", res)
	}
}

func TestTryMutationActive_SingleKey_MobLacksMutation(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobInst(t)
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	res := actTryMutationActive(map[string]any{"key": "blinding-flash"}, ctx)
	if res != Failure {
		t.Errorf("expected Failure when mob lacks the mutation, got %v", res)
	}
}

func TestTryMutationActive_OrderedKeys_FirstAvailableWins(t *testing.T) {
	configs.SetTestConfig()
	mob := newTestMobInst(t)
	mob.Character.Mutations = map[string]int{"healing-gel": 1, "sonic-shout": 1}
	mob.Character.Stamina = 100
	ctx := &EvalContext{InstanceId: mob.InstanceId}

	res := actTryMutationActive(map[string]any{
		"keys": []any{"blinding-flash", "healing-gel", "sonic-shout"},
	}, ctx)
	if res != Success {
		t.Errorf("expected Success when healing-gel is available, got %v", res)
	}
}

// Verify the registry binding survives test setup.
func TestTryMutationActive_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_mutation_active"]; !ok {
		t.Fatal("try_mutation_active not registered in actionRegistry")
	}
}
```

Helper names (`newTestContext`, `newTestMobInst`) — verify against
existing patterns in `actions_forager_test.go` and adapt.

- [ ] **Step 2: Run tests, expect failure**

```bash
go test ./internal/behaviortree/ -run TryMutationActive -v
```

Expected: FAIL with "undefined: actTryMutationActive".

- [ ] **Step 3: Implement `actions_mutation.go`**

Create `internal/behaviortree/actions_mutation.go`:

```go
package behaviortree

// actions_mutation.go — btree action primitive for chunk 2.10
// mutation_* actives: try_mutation_active.
//
// Dispatches to the per-mutation TriggerXxx function in the actions
// package. Accepts either `key: <mutation-key>` (single) or
// `keys: [<key1>, <key2>, ...]` (ordered preference list). At least
// one of the two must be set; nodes with neither are rejected at
// call time with a log + Failure.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// mutationTriggers maps mutation key → action invocation. Add a row
// when lifting a new mutation_* command into the actions package.
//
// Single-target mutations are listed but cannot be used by mobs
// without a btree-side target resolver; v1 only wires AoE and self
// targeting via try_mutation_active. Single-target mutations
// (blinding-spit, toxic-bite) will Failure-out with "no-target" when
// invoked through this action — they need a separate primitive
// (deferred to a followup chunk).
var mutationTriggers = map[string]func(actions.Actor, actions.MutationOpts) actions.MutationResult{
	"blinding-flash": actions.TriggerBlindingFlash,
	"blinding-spit":  actions.TriggerBlindingSpit,
	"healing-gel":    actions.TriggerHealingGel,
	"pacifism-aura":  actions.TriggerPacifismAura,
	"sonic-shout":    actions.TriggerSonicShout,
	"toxic-bite":     actions.TriggerToxicBite,
}

// actTryMutationActive fires the first available mutation in the
// preference list. Success on a triggered mutation; Failure if no
// candidate fires (missing mutation, on cooldown, low stamina, no
// target, no entry in mutationTriggers).
//
// Validation: rejects nodes with neither `key` nor `keys` set. Logs a
// clear error and returns Failure so the author sees the misconfig.
func actTryMutationActive(params map[string]any, ctx *EvalContext) Result {
	keys := collectMutationKeys(params)
	if len(keys) == 0 {
		mudlog.Error("try_mutation_active",
			"error", "node missing required `key` or `keys` parameter",
			"instance_id", ctx.InstanceId)
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
	actor := actions.NewMobActorInRoom(mob, room)

	for _, key := range keys {
		trigger, ok := mutationTriggers[key]
		if !ok {
			mudlog.Warn("try_mutation_active",
				"warn", "unknown mutation key (no actions.TriggerXxx)",
				"key", key, "instance_id", ctx.InstanceId)
			continue
		}
		res := trigger(actor, actions.MutationOpts{})
		if res.Triggered {
			return Success
		}
		// BlockReason in {no-mutation, on-cooldown, low-stamina, not-in-combat,
		// no-target} — fall through to next candidate.
	}
	return Failure
}

// collectMutationKeys returns the ordered preference list from params.
// `key` (single string) and `keys` ([]any of strings) are both accepted;
// when both are set, `key` takes precedence as the first entry, then
// `keys` are appended in order. Duplicates are preserved (cheap; cooldown
// gate makes them no-ops on the second hit).
func collectMutationKeys(params map[string]any) []string {
	out := []string{}
	if single := getStringParam(params, "key"); single != "" {
		out = append(out, single)
	}
	if list := getStringListParam(params, "keys"); len(list) > 0 {
		out = append(out, list...)
	}
	return out
}
```

- [ ] **Step 4: Register the action**

In `internal/behaviortree/actions.go`, add to the init block:

```go
actionRegistry["try_mutation_active"] = actTryMutationActive
```

Place it near other tactical primitives (next to `try_forage` /
`try_salvage`).

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/behaviortree/ -run TryMutationActive -v
go test ./internal/behaviortree/ -v
```

Expected: PASS.

- [ ] **Step 6: Document in `context.md`**

Append to the `internal/behaviortree/context.md` action-primitive
table:

```markdown
| `try_mutation_active` | `key` (string, optional) OR `keys` ([]string, optional). At least one required; nodes with neither are rejected with a log + Failure. | Invoke `actions.TriggerXxx` for the first available mutation in the preference list. Success on triggered; Failure if no candidate fires. Single-target mutations (`blinding-spit`, `toxic-bite`) cannot use this action without a separate target resolver — they return Failure with "no-target". |
```

Also add to the "instant action" table (the action runs synchronously,
no round delay):

```markdown
| `try_mutation_active` | No |
```

- [ ] **Step 7: Build + full test**

```bash
go build ./...
go test ./internal/behaviortree/ ./internal/actions/ -v
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/behaviortree/actions_mutation.go \
        internal/behaviortree/actions_mutation_test.go \
        internal/behaviortree/actions.go \
        internal/behaviortree/context.md
git commit -m "$(cat <<'EOF'
feat(btree): try_mutation_active btree action

Dispatches to actions.TriggerXxx for a mob's available mutation actives,
respecting an ordered preference list. Nodes with neither `key` nor
`keys` set are rejected with a logged error + Failure. Single-target
mutations (blinding-spit, toxic-bite) Failure-out with "no-target" via
this action; a future primitive will add target resolution.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Stage C — selljunk deletion

### Task C1: Delete the dead `selljunk` mob command

**Files:**
- Delete: `internal/mobcommands/selljunk.go`
- Delete: `internal/mobcommands/mobcommands_test.go:TestSelljunk` (lines ~1005-1020)
- Modify: `internal/mobcommands/mobcommands.go` (remove registration)
- Modify: `internal/actions/divergences.go` (remove `"selljunk"` entry)

- [ ] **Step 1: Verify zero callers one more time**

```bash
grep -rn "selljunk\|Selljunk" internal/ _datafiles/ --include="*.go" --include="*.yaml"
```

Expected: only registrations, the implementation file itself, and the
test. No YAML callers, no btree callers, no hook callers. If anything
unexpected surfaces, STOP and reclassify — deletion is wrong.

- [ ] **Step 2: Delete the implementation**

```bash
rm internal/mobcommands/selljunk.go
```

- [ ] **Step 3: Remove the test**

In `internal/mobcommands/mobcommands_test.go`, find `func TestSelljunk(`
and delete the function (typically ~15-20 lines including the closing
`}`).

- [ ] **Step 4: Remove the registration**

In `internal/mobcommands/mobcommands.go`, delete the line:

```go
"selljunk":       {Selljunk, false},
```

- [ ] **Step 5: Remove the divergences entry**

In `internal/actions/divergences.go`, delete the line:

```go
"selljunk":       "mob-ai: converts inventory items to gold",
```

- [ ] **Step 6: Build + test**

```bash
go build ./...
go test ./internal/mobcommands/ ./internal/actions/ -v
```

Expected: PASS. No compile errors (the symbol `Selljunk` is gone from
all referrers).

- [ ] **Step 7: Commit**

```bash
git add internal/mobcommands/selljunk.go \
        internal/mobcommands/mobcommands_test.go \
        internal/mobcommands/mobcommands.go \
        internal/actions/divergences.go
git commit -m "$(cat <<'EOF'
chore(parity): drop dead selljunk mob command

The selljunk mob command was registered, tested, and listed as a
parity divergence, but had zero callers in YAML btrees, hooks, or any
mob AI code. Case study for the new "delete divergent verb" verdict
from chunk 2.10's audit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Stage D — Quick patches (per-gap, instantiated post-audit)

After Stages A1 and A2 produce the audit tables, iterate over the rows
classified as **Gap: patch inline** or **Gap: delete divergent verb**.
Each gap follows this per-gap micro-template.

### Per-gap template

**Files:** (vary per gap, listed in the audit row's "Notes / file refs"
column)

- [ ] **Step 1: Re-read the audit row** — confirm the proposed fix is
  still ≤30 LoC and needs no new gameplay decision. If the fix has
  grown since the audit, demote to **Gap: defer** and add to the
  deferred-gap review doc (Stage E).

- [ ] **Step 2: For "patch inline" gaps — write the failing test**
  (when behavior is testable). Skip if the change is a pure rename or
  doc-only patch.

- [ ] **Step 3: Run test, expect failure**

- [ ] **Step 4: Implement the patch**

- [ ] **Step 5: Run test, expect PASS** (or `go build ./...` + manual
  scan for doc-only patches)

- [ ] **Step 6: Commit** — use the `fix(parity):` or `chore(parity):`
  prefix per the spec's commit-shape table. One commit per gap.

```bash
git commit -m "$(cat <<'EOF'
fix(parity): <verb> — <one-line description>

<2-3 sentence explanation of what was missing and how the fix
brings parity. Reference the audit row.>

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

For **delete** gaps, follow the C1 task shape: verify zero callers,
delete file + test + registration, build + test, commit with
`chore(parity): drop dead <verb>`.

**Quick-patch budget reminder:** one commit per gap. If you find
yourself committing more than ~5-7 inline patches, surface for
mid-chunk check-in — the audit may have under-categorized "patch
inline" and we should re-triage.

---

## Stage E — Deferred-gap review doc

### Task E1: Write the deferred-gap review doc

**Files:**
- Create: `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md`

- [ ] **Step 1: Iterate over the audit tables**

For every row classified as **Gap: defer**, write one entry in the
review doc using the template from the spec:

```markdown
### <verb-name>
- **Direction:** mob-side missing | player-side missing | both-sides need design
- **Surface:** what the command does today on the present side
- **Why deferred:** ≤30 LoC didn't fit / needs new config / needs gameplay decision / ambiguous
- **Sketch of fix:** 2-3 sentence proposal
- **Proposed verdict:** patch-as-followup-chunk | memory-entry-only | wontfix | drop-the-divergent-side | needs-your-call
- **Estimated size:** S / M / L
```

Group entries by **Direction** (mob-side missing first, then player-side
missing, then both-sides). Within each group, sort by **Proposed verdict**
(needs-your-call first to surface ambiguity, then patch-as-followup-chunk,
then memory-entry-only, then wontfix).

- [ ] **Step 2: Add a per-doc header**

```markdown
# Mob Aliveness 2.10 — Deferred Parity Gaps for Review

Surfaced by the chunk 2.10 audit. The audit produced 6 verdicts; this doc
holds every row classified as **Gap: defer**. Each entry lists a proposed
verdict; user triage selects one of:

- **accept-proposed-verdict** — carry through
- **change-verdict** — adjust per user note
- **drop-entirely** — remove without memory entry
- **fix-now-anyway** — pull back into chunk 2.10 (triggers a Stage D micro-task)

After triage, the F-stage tasks write per-verdict memory entries.
```

- [ ] **Step 3: Commit the doc**

```bash
git add docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md
git commit -m "$(cat <<'EOF'
docs(2.10): deferred parity gaps for review

Surfaced every Gap-defer row from the audit tables into a single review
doc for user triage. Each entry has a proposed verdict; user decision
captured per-entry drives Stage F memory writes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 4: Surface the doc to the user**

Post to the user:

> "Deferred-gap review doc written and committed (`docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-deferred-gaps-review.md`). Please go through it and tell me one of {accept-proposed-verdict, change-verdict, drop-entirely, fix-now-anyway} for each entry. Inline annotations or in-thread responses both work."

**Wait for the user's response before proceeding to Stage F.**

---

## Stage F — Memory writes (post-triage)

### Task F1: Runtime-evolved mutations btree-dispatch followup

**Files:**
- Create: `<memory-dir>/project_mutation_active_runtime_evolution_btree.md`
- Modify: `<memory-dir>/MEMORY.md` (add table row)

This one is known up front (called out in the spec as a chunk-output
memory). Land it regardless of user triage results.

- [ ] **Step 1: Write the memory file**

Path: `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_mutation_active_runtime_evolution_btree.md`

Content:

```markdown
---
name: mutation-active-runtime-evolution-btree-dispatch
description: Runtime-evolved active mutations don't flow into a mob's btree dispatch — chunk 2.10 ships explicit-key try_mutation_active; auto-aware dispatch is the followup
metadata:
  type: project
---

Chunk 2.10 shipped the `try_mutation_active` btree action with explicit
`key` / `keys` parameters. Mob archetype authors enumerate which
mutations a mob can use; the dispatcher fires the first available one.

This breaks down when a mob (including a companion) evolves a new
active mutation at runtime that isn't listed in its archetype's
`try_mutation_active` nodes. The evolved mutation will never fire in
combat because the btree doesn't know to consider it.

**Why:** Documented at the end of chunk 2.10 brainstorming (2026-05-23).
The explicit-keys design was chosen for v1 because implicit "use any
mutation the mob has" relies on Go map iteration order
(non-deterministic) and makes priority an accident. The runtime-evolution
problem is a real but smaller-impact followup.

**How to apply:**
- Three candidate fix paths (pick one when the followup is picked up):
  1. **Loader auto-augment:** btree loader scans the mob template's
     `mutations:` field at load time and auto-appends active-trigger
     mutations not already in `try_mutation_active` lists. Catches
     template-declared mutations only; misses runtime evolution.
  2. **Dynamic dispatch action:** new `try_any_active_mutation` action
     that enumerates the mob's *current* mutations at tick time in a
     deterministic order (rarity-descending, evolution-order, or
     author-tagged priority). Author opts in per node.
  3. **Grant-time registration:** mutation-grant code writes evolved
     keys into a mob-scoped MiscData list (`evolved_active_mutations`
     or similar) the btree action reads. Author opt-in.
- Option 2 is the cleanest aliveness story (mobs autonomously try
  newly-evolved abilities) but trades off determinism in tuning. Option
  1 is the simplest. Option 3 is the most explicit but requires touching
  the mutation-grant flow.
- This followup is *not* blocking — mobs that author their `keys` list
  comprehensively at archetype-design time avoid the issue entirely.
  Worth picking up when a player-noticed gap surfaces.
- Related rule: [[feedback_companion_autonomy]] — players don't directly
  order companions, so even runtime-evolved companion mutations need
  *autonomous* dispatch to be reachable.
```

- [ ] **Step 2: Add MEMORY.md table row**

In the "Loose Followups" table, add (preserve date-ordering):

```markdown
| 2026-05-23 | Medium   | [Runtime-evolved mutations btree dispatch](project_mutation_active_runtime_evolution_btree.md) | Mobs/companions that evolve active mutations at runtime can't fire them via try_mutation_active without manual archetype edits; 3 fix paths documented |
```

- [ ] **Step 3: Commit**

```bash
# Memory files live outside the project tree; commit happens in the
# memory directory's git repo (if any). Most users keep memory in their
# Claude config dir, NOT in the project repo. Skip git add if the memory
# dir isn't a repo.
```

If the memory directory is git-tracked, commit it with:

```bash
cd "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory"
git add project_mutation_active_runtime_evolution_btree.md MEMORY.md
git commit -m "chore(memory): runtime-evolved mutations btree dispatch followup"
```

Otherwise, no git action needed — memory files are local-only.

---

### Task F-per-decision: Memory entry per triaged gap

For each user-triaged entry from the deferred-gap review doc with
verdict `patch-as-followup-chunk` or `memory-entry-only`:

- [ ] **Step 1: Write the memory file**

Filename: `project_parity_<verb-name>.md` (kebab-case the verb).

Template:

```markdown
---
name: parity-<verb-name>
description: <one-line summary from the deferred-gap entry>
metadata:
  type: project
---

Surfaced by chunk 2.10's PvM/MvP/PvP/MvM parity audit (2026-05-23).

**Direction:** <mob-side missing | player-side missing | both-sides need design>
**Surface today:** <what the command does on the present side>
**Why deferred from 2.10:** <≤30 LoC didn't fit / needs new config / needs gameplay decision / ambiguous>
**Sketch of fix:** <2-3 sentence proposal from the review doc>
**Verdict at triage:** <accepted from spec | changed by user>
**Estimated size:** <S/M/L>

**How to apply:**
- <when picking this up, what to verify / what files to touch>
- <related followups via [[name]] links>
```

- [ ] **Step 2: Add MEMORY.md table row**

In the "Loose Followups" table:

```markdown
| 2026-05-23 | <user-selected priority> | [Parity: <verb>](project_parity_<verb-name>.md) | <one-line summary> |
```

- [ ] **Step 3: Repeat for each `patch-as-followup-chunk` or `memory-entry-only` entry**

For `wontfix` or `drop-entirely` entries: no memory file needed; the
review doc already records the rationale.

For `fix-now-anyway` entries: pull back into Stage D, run the per-gap
template, then mark the review-doc entry as "fixed in 2.10 commit
\<hash\>".

- [ ] **Step 4: Commit memory writes** (if memory dir is git-tracked)

Single batch commit:

```bash
cd "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory"
git add project_parity_*.md MEMORY.md
git commit -m "chore(memory): log 2.10 deferred parity gaps"
```

---

## Stage G — Closeout

### Task G1: Update MEMORY.md Companion Phase 5 entry

**Files:**
- Modify: `<memory-dir>/MEMORY.md`

- [ ] **Step 1: Update the Companion Phase 5 row**

The current row (set during brainstorming) is:

```markdown
| —          | Medium   | Companion Phase 5 | Player-facing UI closed wontfix per [[feedback_companion_autonomy]]; mutation actives on mobs in scope for chunk 2.10 |
```

If chunk 2.10 has fully landed and Companion Phase 5 has no remaining
sub-items, change status to **Done** and rewrite the row to reflect
that. If anything remains (e.g., specific companion UX polish unrelated
to mutations), keep the row with the remaining-only summary.

- [ ] **Step 2: Commit** (if memory dir is git-tracked)

```bash
cd "C:/Users/Calabe Davis/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory"
git add MEMORY.md
git commit -m "chore(memory): mark Companion Phase 5 done post-2.10"
```

---

### Task G2: Mark chunk 2.10 Done in roadmap

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md` (project root)

- [ ] **Step 1: Update progress tracker row**

Find:

```markdown
| 2.10 | Tactical | PvM/MvP/PvP/MvM parity audit | M | 2.1–2.9 | Not started |
```

Change `Not started` → `Done`.

- [ ] **Step 2: Update chunk 2.10's mini-brief section**

Find the `### 2.10 PvM/MvP/PvP/MvM parity audit` section and update:

```markdown
### 2.10 PvM/MvP/PvP/MvM parity audit
**Status:** Done (2026-05-XX) • **Size:** M
```

(Use the actual completion date.)

- [ ] **Step 3: Append a Shipped paragraph**

Add a `- **Shipped:**` bullet at the end of the section, mirroring the
style of the other shipped chunks (2.7, 2.8, 2.9). Cover:
- Audit-table location (the design spec file)
- Mutation\_\* lift summary (6 actions, 6 mob wrappers,
  `try_mutation_active` btree action)
- `selljunk` deletion as the delete-divergent-verb case study
- Quick-patch count (one line listing the verbs patched inline)
- Deferred-gap count + review doc location
- Memory entries created
- Spec at `docs/superpowers/specs/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit-design.md`
- Plan at `docs/superpowers/plans/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit.md`
- Note: **Manual in-game smoke testing deferred to user** (per chunk
  2.9 precedent)

- [ ] **Step 4: Update the roll-up line**

Find:

```markdown
**Roll-up:** 17 / 41 done • 0 in progress • 24 not started.
```

Update to: `18 / 41 done • 0 in progress • 23 not started.`

- [ ] **Step 5: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(2.10): mark mob-aliveness 2.10 Done

Phase 2 of the mob aliveness roadmap now fully shipped.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Manual smoke test (user-driven, post-chunk)

Per the chunk 2.9 precedent, in-game smoke is deferred to the user.
Checklist for the user to run after the feature branch is merged:

- [ ] Boot the server with `go run .` and verify clean startup past
  data-file loading (no panics).
- [ ] Spawn a test mob with `mutations: { blinding-flash: 1 }` and a
  btree node `try_mutation_active: blinding-flash`.
- [ ] Attack the mob and verify the flash fires when it's in combat.
- [ ] Verify the player gets blinded (condition applied; descriptive
  message, no numbers).
- [ ] Verify other mobs in the room get blinded.
- [ ] Repeat with a mob that has only `healing-gel` — verify it heals
  itself out of combat too.
- [ ] Verify the deleted `selljunk` mob command produces "unknown
  command" if any old btree YAML still references it (should be none
  after Stages A/D).

---

## Self-review checklist (run after writing this plan, before handoff)

**Spec coverage:**
- [x] Audit walk — Stages A1, A2
- [x] Mutation_* actions-lift — Stages B1–B7
- [x] Btree `try_mutation_active` — Stage B8
- [x] Selljunk deletion — Stage C1
- [x] Quick patches — Stage D template
- [x] Deferred-gap review doc — Stage E1
- [x] Memory writes (per-gap, runtime-evolution followup) — Stages F1, F-per-decision
- [x] Roadmap update — Stages G1, G2
- [x] PATCH\_NOTES update — not in plan; this chunk merges to
  development first, PATCH\_NOTES update happens at the master-merge
  step per project SOP. If user wants this chunk pushed to prod
  directly, add a G3 task: update `PATCH_NOTES.md` with a 2026-05-23
  entry, set `Logging.LogToFile: false` in `_datafiles/config.yaml`,
  boot-test, then merge to master.

**Placeholder scan:**
- No "TBD" / "implement later" / "fill in details"
- No "Similar to Task N" without showing the code (B3–B7 do show
  per-mutation specifics; the structure-mirror reference is
  acceptable because each task's steps repeat the framework and the
  per-mutation deltas are explicit)
- All commit messages use heredoc + Co-Authored-By line

**Type consistency:**
- `MutationOpts` / `MutationResult` defined in B2, used in B3-B8
- `mutationPreamble` / `preambleResult` defined in B1, used in B2-B7
- `actTryMutationActive` / `mutationTriggers` defined in B8
- `actions.TriggerBlindingFlash` etc. — naming convention `Trigger<MutationName>`
  consistent across B2-B7

Issues fixed inline. No re-review needed.

---

## Execution handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-23-mob-aliveness-2.10-pvm-mvp-parity-audit.md`.** Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration. Particularly well-suited here because A1 and A2 are parallelizable, B2–B7 are six near-identical TDD cycles ideal for parallel subagents (after B1 lands), and D's per-gap micro-template is naturally subagent-shaped.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
