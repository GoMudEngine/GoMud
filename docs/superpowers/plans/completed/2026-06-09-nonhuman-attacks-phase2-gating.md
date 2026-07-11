# Non-human Attacks — Phase 2: Anatomy-Gated Moves + Wake Hamstring + Retire Bite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop non-humanoid creatures using human technique moves (grapple/bash/trip/kick/submit) by gating them on anatomy (`body_parts`); wake the dormant `hamstring` beast move into AI selection; and retire the now-redundant `bite` special (biting is the Phase-1 basic attack).

**Architecture:** A `Character.HasBodyPart` helper reads the species' `body_parts`. The `CanUse*` gates in `internal/combat/ai.go` (and the grapple action entry) require the relevant part — `arms` for grapple/bash/submit (bash bypassed by `NaturalBash`), `legs` for trip/kick. `hamstring` gains `CanUseHamstring`/`ScoreHamstring` + AI-profile weight (it's already a registered mob command). The `bite` special (`actions.ExecuteBite` + `mobcommands` `bite`) is removed; `toxic-bite` (a mutation move) stays.

**Tech Stack:** Go; `internal/combat`, `internal/characters`, `internal/species`, `internal/actions`, `internal/mobcommands`.

**Spec:** `docs/superpowers/specs/2026-06-09-nonhuman-attacks-and-beast-moveset-design.md` (this plan implements **Layer 2a + the hamstring/bite parts of 2b**). Phase 3 = the new beast moves (throttle/pounce/maul/rake/gore); Phase 4 = AI profiles + per-mob override + data audit.

**Verified facts (2026-06-09):**
- `bite` & `hamstring` are MOB-ONLY commands (`internal/mobcommands/mobcommands.go:29,57`); no player handlers; not in `command_readiness.go`; not in the parity `supported` list. `toxic-bite` is separate (keep).
- `CanUseGrapple` (ai.go:184) checks cooldown + IsGrappling + `GrappleImmune` — NO body-part check. `CanUseBash` (152) requires `HasShield()` + cooldown. `CanUseTrip` (164)/`CanUseKick` (176) check cooldown (+ trip: not grappling). `CanUseSubmit` (200) requires ground-grapple + controller.
- `NaturalBash` species may lack `arms` (e.g. magma elemental `body_parts: [skin]`) — bash gating MUST let `NaturalBash` bypass the arms requirement.
- Species `BodyParts []string` (yaml `body_parts`), accessor `species.GetSpecies(id)`.

---

## File Structure

- `internal/characters/description.go` (or a small `internal/characters/anatomy.go`) — ADD `HasBodyPart`.
- `internal/combat/ai.go` — gate `CanUseGrapple/Bash/Trip/Kick/Submit`; add `CanUseHamstring`/`ScoreHamstring`; register `hamstring` in `aiProfiles`; add `hamstring` to `ChooseSpecialMove`'s viability list.
- `internal/actions/combat_grapple.go` — arms guard at the grapple action entry (defense-in-depth).
- DELETE `internal/actions/combat_bite.go`, `internal/mobcommands/bite.go`; remove the `bite` map entry in `internal/mobcommands/mobcommands.go`; remove related tests.
- `internal/combat/context.md`, `internal/actions/context.md`, `internal/mobcommands/context.md` — docs.

---

## Task 1: `Character.HasBodyPart` helper

**Files:**
- Create: `internal/characters/anatomy.go`
- Test: `internal/characters/anatomy_test.go`

- [ ] **Step 1: Write the failing test**

`internal/characters/anatomy_test.go`:

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/species"
)

func TestCharacter_HasBodyPart(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7001: {SpeciesId: 7001, Name: "armed", BodyParts: []string{"arms", "legs", "mouth"}},
		7002: {SpeciesId: 7002, Name: "noarms", BodyParts: []string{"legs", "mouth"}},
	})
	defer cleanup()

	armed := &Character{SpeciesId: 7001}
	beast := &Character{SpeciesId: 7002}

	if !armed.HasBodyPart("arms") {
		t.Error("armed should have arms")
	}
	if beast.HasBodyPart("arms") {
		t.Error("beast should NOT have arms")
	}
	if !beast.HasBodyPart("legs") {
		t.Error("beast should have legs")
	}
}
```

(Confirm `species.SeedSpeciesForTest` signature — added/used in Phase 1.)

- [ ] **Step 2: Run the test — verify it fails**

Run: `go test ./internal/characters/ -run TestCharacter_HasBodyPart -v`
Expected: `armed.HasBodyPart` undefined.

- [ ] **Step 3: Implement**

`internal/characters/anatomy.go`:

```go
package characters

import "github.com/GoMudEngine/GoMud/internal/species"

// HasBodyPart reports whether this character's species declares the given
// canonical body-part tag (e.g. "arms", "legs", "mouth"). Used to gate which
// combat moves a creature's anatomy permits (humanoid technique moves need
// "arms"; trip/kick need "legs"). Returns false if the species is unknown.
func (c *Character) HasBodyPart(part string) bool {
	sp := species.GetSpecies(c.SpeciesId)
	if sp == nil {
		return false
	}
	for _, p := range sp.BodyParts {
		if p == part {
			return true
		}
	}
	return false
}
```

(`internal/characters` already imports `species` in description.go — no cycle: species does not import characters.)

- [ ] **Step 4: Run the test — verify it passes**

Run: `go test ./internal/characters/ -run TestCharacter_HasBodyPart -v` → PASS; then `go build ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/anatomy.go internal/characters/anatomy_test.go
git commit -m "feat(characters): HasBodyPart anatomy helper"
```
(End with the Co-Authored-By line: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`)

---

## Task 2: Gate grapple on `arms` (CanUse + action entry)

**Files:**
- Modify: `internal/combat/ai.go` (`CanUseGrapple`, ~line 184)
- Modify: `internal/actions/combat_grapple.go` (the grapple-initiation guard, near the existing `GrappleImmune` check)
- Test: `internal/combat/ai_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Add to `internal/combat/ai_test.go`:

```go
func TestCanUseGrapple_RequiresArms(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7101: {SpeciesId: 7101, Name: "humanoid", BodyParts: []string{"arms", "hands", "legs"}},
		7102: {SpeciesId: 7102, Name: "wolf", BodyParts: []string{"legs", "mouth", "tail"}},
	})
	defer cleanup()

	humanoid := &characters.Character{SpeciesId: 7101, Cooldowns: characters.Cooldowns{}}
	wolf := &characters.Character{SpeciesId: 7102, Cooldowns: characters.Cooldowns{}}

	if !CanUseGrapple(humanoid) {
		t.Error("armed humanoid should be able to grapple")
	}
	if CanUseGrapple(wolf) {
		t.Error("no-arms wolf must NOT be able to grapple")
	}
}
```

(Confirm the `Cooldowns` zero-value/type — match how other ai_test or combat tests build a Character with empty cooldowns; the point is the arms gate.)

- [ ] **Step 2: Run — verify it fails**

Run: `go test ./internal/combat/ -run TestCanUseGrapple_RequiresArms -v`
Expected: FAIL — wolf currently CAN grapple (no arms check).

- [ ] **Step 3: Add the arms gate to `CanUseGrapple`**

In `internal/combat/ai.go`, in `CanUseGrapple`, add after the `GrappleImmune` check (before `return true`):

```go
	// Grappling is a humanoid technique — requires arms to seize/hold.
	if !char.HasBodyPart("arms") {
		return false
	}
```

- [ ] **Step 4: Add the same guard at the grapple action entry (defense-in-depth)**

In `internal/actions/combat_grapple.go`, find where it rejects on `GrappleImmune` (the initiation path) and add an equivalent arms check that returns the same "can't grapple" result the GrappleImmune path returns (match the existing result/return shape in that function — read it first). This blocks a btree/direct command, not just AI selection. If the function signature/return differs from a simple bool, mirror exactly what the `GrappleImmune` early-return does there.

- [ ] **Step 5: Run — verify it passes + package green**

Run: `go test ./internal/combat/ -run TestCanUseGrapple_RequiresArms -v` → PASS; then `go test ./internal/combat/ ./internal/actions/` and `go build ./...` → green.

- [ ] **Step 6: Commit**

```bash
git add internal/combat/ai.go internal/actions/combat_grapple.go internal/combat/ai_test.go
git commit -m "feat(combat): gate grapple on the arms body part (no-arms beasts can't grapple)"
```
(Co-Authored-By line.)

---

## Task 3: Gate bash on `arms` (NaturalBash bypass)

**Files:**
- Modify: `internal/combat/ai.go` (`CanUseBash`, ~line 152)
- Test: `internal/combat/ai_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCanUseBash_ArmsOrNaturalBash(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7201: {SpeciesId: 7201, Name: "armless_elemental", BodyParts: []string{"skin"}, NaturalBash: true},
		7202: {SpeciesId: 7202, Name: "wolf", BodyParts: []string{"legs", "mouth"}},
	})
	defer cleanup()

	// NaturalBash elemental: bypasses both shield AND arms requirements.
	elem := &characters.Character{SpeciesId: 7201, Cooldowns: characters.Cooldowns{}}
	if !CanUseBash(elem) {
		t.Error("NaturalBash elemental should be able to bash without arms/shield")
	}
	// Wolf: no shield, not NaturalBash -> already false (no change), but assert it.
	wolf := &characters.Character{SpeciesId: 7202, Cooldowns: characters.Cooldowns{}}
	if CanUseBash(wolf) {
		t.Error("wolf without shield/NaturalBash must not bash")
	}
}
```

(`CanUseBash` currently returns false without a shield; for the elemental case, NaturalBash must allow it. Confirm how `NaturalBash` is read — `species.GetSpecies(char.SpeciesId).NaturalBash`.)

- [ ] **Step 2: Run — verify it fails**

Run: `go test ./internal/combat/ -run TestCanUseBash_ArmsOrNaturalBash -v`
Expected: FAIL — `CanUseBash` requires a shield, so the NaturalBash elemental returns false.

- [ ] **Step 3: Rework `CanUseBash`**

Replace the body of `CanUseBash` in `internal/combat/ai.go` with shield-OR-NaturalBash + arms-OR-NaturalBash gating:

```go
func CanUseBash(char *characters.Character) bool {
	// Must not be on cooldown
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	naturalBash := false
	if sp := species.GetSpecies(char.SpeciesId); sp != nil {
		naturalBash = sp.NaturalBash
	}
	// Needs a shield to bash, unless the species bashes naturally
	// (elementals/golems).
	if !char.HasShield() && !naturalBash {
		return false
	}
	// Bash is a humanoid/arms technique — but NaturalBash species bash with
	// their body (slam) and bypass the arms requirement.
	if !char.HasBodyPart("arms") && !naturalBash {
		return false
	}
	return true
}
```

(Confirm `species` is imported in ai.go — it is, used by CanUseGrapple.)

- [ ] **Step 4: Run — verify it passes + package green**

Run: `go test ./internal/combat/ -run TestCanUseBash -v` → PASS; `go test ./internal/combat/` green.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/ai.go internal/combat/ai_test.go
git commit -m "feat(combat): gate bash on arms unless NaturalBash (elementals still bash)"
```
(Co-Authored-By line.)

---

## Task 4: Gate trip + kick on `legs`

**Files:**
- Modify: `internal/combat/ai.go` (`CanUseTrip` ~164, `CanUseKick` ~176)
- Test: `internal/combat/ai_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCanUseTripKick_RequireLegs(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7301: {SpeciesId: 7301, Name: "legged", BodyParts: []string{"legs", "mouth"}},
		7302: {SpeciesId: 7302, Name: "serpent", BodyParts: []string{"mouth", "skin"}},
	})
	defer cleanup()

	legged := &characters.Character{SpeciesId: 7301, Cooldowns: characters.Cooldowns{}}
	serpent := &characters.Character{SpeciesId: 7302, Cooldowns: characters.Cooldowns{}}

	if !CanUseTrip(legged) || !CanUseKick(legged) {
		t.Error("legged creature should trip and kick")
	}
	if CanUseTrip(serpent) {
		t.Error("legless serpent must not trip")
	}
	if CanUseKick(serpent) {
		t.Error("legless serpent must not kick")
	}
}
```

- [ ] **Step 2: Run — verify it fails**

Run: `go test ./internal/combat/ -run TestCanUseTripKick_RequireLegs -v`
Expected: FAIL — serpent can currently trip/kick.

- [ ] **Step 3: Add `legs` gates**

In `CanUseTrip`, after the existing cooldown + IsGrappling checks, before `return true`:

```go
	// Tripping sweeps the legs — requires the attacker to have legs.
	if !char.HasBodyPart("legs") {
		return false
	}
```

In `CanUseKick`, after the cooldown check, before `return true`:

```go
	// Kicking requires legs.
	if !char.HasBodyPart("legs") {
		return false
	}
```

- [ ] **Step 4: Run — verify it passes + package green**

Run: `go test ./internal/combat/ -run TestCanUseTripKick_RequireLegs -v` → PASS; `go test ./internal/combat/` green.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/ai.go internal/combat/ai_test.go
git commit -m "feat(combat): gate trip/kick on the legs body part"
```
(Co-Authored-By line.)

---

## Task 5: Gate submit on `arms`

**Files:**
- Modify: `internal/combat/ai.go` (`CanUseSubmit`, ~line 200)
- Test: `internal/combat/ai_test.go`

- [ ] **Step 1: Write the failing test**

Submit requires ground-grapple + controller; since gating grapple already prevents beasts from reaching that state, this is completeness. Test the arms gate directly:

```go
func TestCanUseSubmit_RequiresArms(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7402: {SpeciesId: 7402, Name: "wolf", BodyParts: []string{"legs", "mouth"}},
	})
	defer cleanup()
	wolf := &characters.Character{SpeciesId: 7402}
	if CanUseSubmit(wolf) {
		t.Error("no-arms wolf must not submit (even if somehow grappling)")
	}
}
```

(A no-arms wolf isn't grappling anyway, so `CanUseSubmit` already returns false via the IsGroundGrapple check; adding the arms gate makes the intent explicit and robust. If the test passes already due to the grapple-state check, KEEP the explicit arms gate anyway for clarity — note this in your report.)

- [ ] **Step 2: Run — observe result**

Run: `go test ./internal/combat/ -run TestCanUseSubmit_RequiresArms -v`
Expected: likely already PASS (wolf not in ground-grapple). That's fine — proceed to add the explicit gate so the rule is co-located with the others.

- [ ] **Step 3: Add the arms gate to `CanUseSubmit`**

After the controller check, before `return true`:

```go
	// Submission holds require arms.
	if !char.HasBodyPart("arms") {
		return false
	}
```

- [ ] **Step 4: Run + package green**

Run: `go test ./internal/combat/` → green.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/ai.go internal/combat/ai_test.go
git commit -m "feat(combat): gate submit on arms (explicit anatomy rule)"
```
(Co-Authored-By line.)

---

## Task 6: Wake `hamstring` into AI selection

**Files:**
- Modify: `internal/combat/ai.go` (add `CanUseHamstring` + `ScoreHamstring`; add to `ChooseSpecialMove`; add to `aiProfiles`)
- Test: `internal/combat/ai_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCanUseHamstring_FangedOrClawedWithLegs(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7501: {SpeciesId: 7501, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
		7502: {SpeciesId: 7502, Name: "human", BodyParts: []string{"arms", "hands", "legs", "mouth"}},
		7503: {SpeciesId: 7503, Name: "serpent", BodyParts: []string{"mouth", "skin"}, NaturalAttack: items.Bite},
	})
	defer cleanup()

	wolf := &characters.Character{SpeciesId: 7501, Cooldowns: characters.Cooldowns{}}
	human := &characters.Character{SpeciesId: 7502, Cooldowns: characters.Cooldowns{}}
	serpent := &characters.Character{SpeciesId: 7503, Cooldowns: characters.Cooldowns{}}

	if !CanUseHamstring(wolf) {
		t.Error("fanged, legged wolf should hamstring")
	}
	if CanUseHamstring(human) {
		t.Error("plain humanoid (no natural fang/claw) should not hamstring")
	}
	if CanUseHamstring(serpent) {
		t.Error("legless serpent should not hamstring (no legs to cut)")
	}
}
```

- [ ] **Step 2: Run — verify it fails**

Run: `go test ./internal/combat/ -run TestCanUseHamstring -v`
Expected: `CanUseHamstring` undefined.

- [ ] **Step 3: Implement `CanUseHamstring` + `ScoreHamstring`**

In `internal/combat/ai.go` (mirror the `CanUseKick`/`ScoreKick` shape):

```go
func CanUseHamstring(char *characters.Character) bool {
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	// Beast move: needs legs to cut, and a fanged/clawed natural attack.
	if !char.HasBodyPart("legs") {
		return false
	}
	sp := species.GetSpecies(char.SpeciesId)
	if sp == nil {
		return false
	}
	return sp.NaturalAttack == items.Bite || sp.NaturalAttack == items.Claws
}

func ScoreHamstring(mob *mobs.Mob, target *characters.Character) int {
	score := 45
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 40 {
		score += 15
	}
	// Bonus when the target is healthy and mobile — hamstring to slow them.
	targetHealthPercent := float64(target.Health) * 100.0 / float64(target.HealthMax.Value)
	if targetHealthPercent > 50 {
		score += 15
	}
	if score < 0 {
		score = 0
	}
	return score
}
```

(Add `items` import to ai.go if not present — confirm; ai.go already uses `species`, `skills`, `mobs`, `characters`.)

- [ ] **Step 4: Wire it into `ChooseSpecialMove` + profiles**

In `ChooseSpecialMove`, alongside the other `CanUse*` checks:

```go
	if CanUseHamstring(&mob.Character) {
		moveScores["hamstring"] = ScoreHamstring(mob, target)
	}
```

In `aiProfiles`, add `hamstring` weight to beast-relevant profiles (e.g. `aggressive`, `brawler`, `default`):

```go
	"aggressive": {
		"bash":      40,
		"kick":      30,
		"trip":      20,
		"grapple":   10,
		"hamstring": 35,
	},
```

(Add a `hamstring` weight to `default` and `brawler` similarly; pick weights in the 25–35 range. `hamstring` is already a registered MOB command — `mobcommands.go:57` — so `mob.Command("hamstring")` from `handleMobAIDecision` dispatches correctly.)

- [ ] **Step 5: Run — verify it passes + package green + parity**

Run: `go test ./internal/combat/ -run TestCanUseHamstring -v` → PASS; `go test ./internal/combat/ ./internal/mobcommands/` (parity test) → green; `go build ./...` clean.

- [ ] **Step 6: Commit**

```bash
git add internal/combat/ai.go internal/combat/ai_test.go
git commit -m "feat(combat): wake hamstring into mob AI selection (fanged/clawed, legged)"
```
(Co-Authored-By line.)

---

## Task 7: Retire the `bite` special

**Files:**
- Delete: `internal/actions/combat_bite.go` (+ its `_test.go` if any)
- Delete: `internal/mobcommands/bite.go`
- Modify: `internal/mobcommands/mobcommands.go` (remove the `"bite": {Bite, false},` entry at line 29)
- Modify/remove: any other reference to `actions.ExecuteBite` / the `bite` mob command

- [ ] **Step 1: Find every reference (reference-check before deleting)**

Run:
```bash
grep -rn "ExecuteBite\|BiteResult\|\"bite\"\|mobcommands.Bite\b" internal/ | grep -v "toxic-bite\|toxicbite\|ToxicBite"
grep -rn "\bbite\b" _datafiles/world/dogmud/mobs/ _datafiles/world/dogmud/behaviors/ 2>/dev/null | grep -viE "toxic-bite|bite_|natural_attack: bite"
```
The first finds Go references to the special (must all be removed/none left). The second finds DATA references to a `bite` COMMAND (e.g. a mob `combatcommands:`/`idlecommands:` or a btree action `do: bite`) — if any mob/btree invokes the `bite` command, removing it would break them. Record them; if any exist, they must be retargeted (to `hamstring`, or removed) as part of this task. (`natural_attack: bite` in species is NOT a command reference — leave it.)

- [ ] **Step 2: Write/adjust the test expectation**

If `internal/actions/combat_bite_test.go` exists, delete it (the move is gone). Confirm no other test references `ExecuteBite`/`BiteResult`. The parity test does not list `bite`, so it stays green. (If the reference-check found a test elsewhere asserting bite, update it.)

- [ ] **Step 3: Remove the special**

- Delete `internal/actions/combat_bite.go` (and its test file).
- Delete `internal/mobcommands/bite.go`.
- In `internal/mobcommands/mobcommands.go`, remove the line `"bite":           {Bite, false},`.
- Retarget any data references found in Step 1 (e.g. a mob `combatcommands: [bite]` → remove `bite` or swap to `hamstring`). Keep ALL `toxic-bite` references untouched.

- [ ] **Step 4: Build + tests + parity green**

Run: `go build ./...` (clean — no dangling `Bite`/`ExecuteBite` references), then `go test ./internal/actions/ ./internal/mobcommands/ ./internal/combat/`.
Expected: green, including `command_parity_test.go` (it never listed `bite`).

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(combat): retire the bite special (biting is now the basic attack); keep toxic-bite"
```
(Co-Authored-By line.)

---

## Task 8: Update context.md

**Files:**
- Modify: `internal/combat/context.md`, `internal/actions/context.md`, `internal/mobcommands/context.md`

- [ ] **Step 1: Document the changes**

- `internal/combat/context.md`: document anatomy gating — `CanUseGrapple/Submit` require `arms`; `CanUseBash` requires `arms` OR `NaturalBash`; `CanUseTrip/Kick` require `legs`; `CanUseHamstring` (new) requires `legs` + a fanged/clawed `NaturalAttack`. Note `Character.HasBodyPart` is the single anatomy predicate and `hamstring` is now AI-selectable.
- `internal/actions/context.md`: note the `bite` special move (`combat_bite.go`) was retired (basic-attack biting replaces it) and the grapple action entry now also requires `arms`.
- `internal/mobcommands/context.md`: note `bite` was removed from the mob command set (`toxic-bite` remains); `hamstring` is now reachable via AI selection.

Match each file's tone/structure; concise factual additions.

- [ ] **Step 2: Build (docs only)**

Run: `go build ./...` → clean.

- [ ] **Step 3: Commit**

```bash
git add internal/combat/context.md internal/actions/context.md internal/mobcommands/context.md
git commit -m "docs(context): anatomy move-gating, hamstring AI, bite-special retirement"
```
(Co-Authored-By line.)

---

## Task 9: Verification + smoke

- [ ] **Step 1: Full build + targeted tests**

Run: `go build ./... && go test ./internal/combat/ ./internal/characters/ ./internal/actions/ ./internal/mobcommands/ ./internal/species/`
Expected: all green (incl. parity test).

- [ ] **Step 2: Boot test**

Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` then `go run .` → `Server Ready`, no panic. Kill server + clear ports 33333/44444/55555.

- [ ] **Step 3: In-game smoke**

Boot, connect, and fight a no-arms beast (wolf): confirm it NEVER grapples/bashes and DOES use bite (basic) + hamstring; fight a serpent (no legs): confirm it never trips/kicks; confirm a humanoid still grapples/bashes/trips. Confirm the `bite` command is gone for mobs (the special) while basic-attack biting still narrates (Phase 1). Kill server + clear ports.

- [ ] **Step 4: (No push — local only)**

Merges into local `master` for the bundle; Phase 3 is a separate plan.

---

## Self-review notes (controller)

- **Spec coverage (Layer 2a + hamstring/bite of 2b):** anatomy helper (T1); grapple/bash/trip/kick/submit gating (T2–T5, with NaturalBash bypass for bash and legs for trip/kick/hamstring); wake hamstring into AI (T6); retire bite special (T7); context.md (T8); verify (T9).
- **Cross-cutting wiring (per spec section B/C):** retiring `bite` touches both the mob command map + handler + action + tests (T7, with a data reference-check); `hamstring` needs no new registration (already a mob command). No NEW player↔mob commands are added in Phase 2, so no new parity-`supported`/helpfile/keywords entries (those land in Phase 3). context.md updated (T8).
- **Key gotchas baked in:** `NaturalBash` bypasses the bash arms-gate (magma elemental has no arms); submit gating is mostly redundant behind the grapple gate but added for explicit co-location; `toxic-bite` is preserved when retiring `bite`.
- **Discovery steps** (T7 reference-check, T2 grapple-action return shape, Cooldowns zero-value) are verify-then-act, not placeholders.
