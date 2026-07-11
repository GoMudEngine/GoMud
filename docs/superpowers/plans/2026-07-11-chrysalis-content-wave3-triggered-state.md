# Chrysalis Content — Wave 3: triggered-state recipe (Blood Frenzy)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Establish the **triggered-state** mechanic (P5) — a mutation drives a combat *state* that turns on/off from a condition each round — proven with **Blood Frenzy** (Ravener): when badly wounded you enter a frenzy that hits harder but cannot retreat.

**Architecture:** A triggered state is a short-duration **buff** applied/refreshed by a per-round check. Blood Frenzy reuses the existing `DamageBonus` buff-flag (already worth +15% on-hit damage in `applyCombatDamageBonuses`) for the upside and adds one new `frenzied` buff-flag for the drawback (a `flee` gate, mirroring the existing `NoMovement` gate). The trigger is a small helper called from the existing user + mob round ticks. The mutation itself carries a describable `flag` (`battle-frenzy`) that both drives the trigger and renders in the `mutations` command.

**Tech Stack:** Go — `internal/buffs`, `internal/usercommands` (flee), `internal/hooks` (round ticks), `internal/mutations` (describe); YAML buff + mutation + help data; testify.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (Wave 3 of §9). Builds on Waves 1–2 (merged).

**Scope — this wave is P5 only.** Blood Frenzy's richer effects (faster attacks, taunt/fear immunity, lifesteal, terror-on-kill) are deepening/bespoke work deferred to the balance pass — the exemplar proves the state mechanic with a clean upside (damage) + drawback (no flee).

**Transformation-apex (P6) note — NO new framework needed, deferred to Wave 6 authoring.** An apex is simply a high-rarity, always-on **passive-bundle** mutation: strong existing-type effects (stat/health/armor/damage multipliers) + its cluster's `pole` tag (so pole-depth drives the opposition choke, already in the engine) + a `prerequisites` spine on its cluster's cores. No engine framework is required for that. Only the *bespoke* apex effects (devour execute, room cleave, full incorporeal miss, swarm spawn, party aura, flight) need custom code — those ride with Waves 4–5 + the per-cluster authoring in Wave 6, once each cluster's core keystones exist to satisfy the prereq spines.

**Wave 1–2 reminders:** every mutation needs a `templates/help/{id}.template` (completeness test) and a `DescribeEffect`/`flagPhrase` entry per effect it uses, or `go test ./...` fails / the `mutations` command shows a blank line.

---

## File Structure

**Modify:**
- `internal/buffs/buffspec.go` — add the `Frenzied` flag constant
- `internal/usercommands/flee.go` — gate flee on the frenzy flag
- `internal/mutations/describe.go` — `flagPhrase("battle-frenzy")` case
- `internal/hooks/NewRound_UserRoundTick.go` + `NewRound_MobRoundTick.go` — call the frenzy tick

**Create:**
- `internal/hooks/mutation_frenzy.go` — `tickBloodFrenzy(actor)` trigger + test `mutation_frenzy_test.go`
- `_datafiles/world/dogmud/buffs/{id}-frenzied.yaml` — the Frenzied state buff
- `_datafiles/world/dogmud/mutations/blood-frenzy.yaml` + `.../help/blood-frenzy.template`

---

## Phase 1 — The frenzy state & its gate

### Task 1: `Frenzied` buff flag + flee gate

**Files:** Modify `internal/buffs/buffspec.go`, `internal/usercommands/flee.go`; Test `internal/usercommands/flee_frenzy_test.go`

- [ ] **Step 1: Add the flag constant**

In `internal/buffs/buffspec.go`, alongside `NoMovement Flag = ` and `Sleeping Flag = ` (~line 38–59):

```go
	Frenzied       Flag = `frenzied` // blood-frenzy state: cannot flee
```

- [ ] **Step 2: Write the failing test**

```go
// internal/usercommands/flee_frenzy_test.go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
)

// A frenzied bearer cannot flee. Uses the same flag-driven gate as NoMovement.
func TestFleeBlockedWhileFrenzied(t *testing.T) {
	// buffs.Frenzied must exist and be distinct from NoMovement.
	if buffs.Frenzied == buffs.NoMovement || buffs.Frenzied == "" {
		t.Fatalf("buffs.Frenzied must be a distinct non-empty flag, got %q", buffs.Frenzied)
	}
}
```

> The behavioral gate is verified by the boot/manual smoke (Flee needs a full UserRecord/room harness); this test pins the flag's existence + distinctness, and Step 4 wires the gate identically to the proven `NoMovement` path.

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/usercommands/ -run TestFleeBlockedWhileFrenzied -v`
Expected: FAIL — `buffs.Frenzied` undefined.

- [ ] **Step 4: Add the flee gate**

In `internal/usercommands/flee.go`, after the existing `NoMovement` block (~line 21):

```go
	// Blood Frenzy: a bearer mid-rage cannot make themselves retreat.
	if user.Character.HasBuffFlag(buffs.Frenzied) {
		user.SendText(messaging.CategorySystem, `The red haze won't let you turn your back — you can only fight.`)
		return true, nil
	}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/usercommands/ -run TestFleeBlockedWhileFrenzied -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/buffs/buffspec.go internal/usercommands/flee.go internal/usercommands/flee_frenzy_test.go
git commit -m "feat(buffs): Frenzied flag + flee gate for the blood-frenzy state"
```

---

### Task 2: The Frenzied state buff

**Files:** Create `_datafiles/world/dogmud/buffs/{id}-frenzied.yaml`

First choose the id: run `python tools/id_inventory.py --type buffs` and use the next free buff id. This plan writes **90** — replace if taken.

- [ ] **Step 1: Write the buff (model the schema on an existing flag+text buff, e.g. `39-venom.yaml` / `15-sleeping.yaml`)**

```yaml
buffid: 90
name: Blood Frenzy
description: A red haze — you hit like a beast and cannot make yourself retreat.
triggerrate: 1 round
triggercount: 2
flags:
  - damage-bonus
  - frenzied
start_user_text: The world goes red. Something in you comes off its chain.
start_room_text: "{source}'s eyes go wide and bloodshot, and a low snarl builds in their throat."
end_user_text: The red haze recedes, and your limbs sag.
```

> `damage-bonus` reuses the existing +15% on-hit consumer in `applyCombatDamageBonuses`; `frenzied` drives the flee gate. `triggercount: 2` gives it a 2-round life so it lapses on its own once the trigger stops refreshing it. Confirm the exact flag strings for `buffs.DamageBonus` and `buffs.Frenzied` in `internal/buffs/buffspec.go` and match them here.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/buffs/90-frenzied.yaml
git commit -m "content(buffs): Blood Frenzy state buff (damage-bonus + frenzied)"
```

---

## Phase 2 — The trigger

### Task 3: `tickBloodFrenzy` + describe case

**Files:** Create `internal/hooks/mutation_frenzy.go`, `internal/hooks/mutation_frenzy_test.go`; Modify `internal/mutations/describe.go`

- [ ] **Step 1: Add the mutation flag describe case**

In `internal/mutations/describe.go`, inside `flagPhrase`'s switch:

```go
	case "battle-frenzy":
		return "When badly wounded you fly into a battle frenzy — you hit harder, but cannot retreat."
```

- [ ] **Step 2: Write the failing test (the pure trigger predicate)**

```go
// internal/hooks/mutation_frenzy_test.go
package hooks

import "testing"

func TestShouldFrenzy(t *testing.T) {
	// owns the flag + below half HP → frenzy
	if !shouldFrenzy(true, 40, 100) {
		t.Fatal("flag + <50% HP should frenzy")
	}
	// healthy → no
	if shouldFrenzy(true, 80, 100) {
		t.Fatal(">50% HP should not frenzy")
	}
	// no flag → no
	if shouldFrenzy(false, 10, 100) {
		t.Fatal("without the flag, never frenzy")
	}
	// guard: zero max HP → no divide, no frenzy
	if shouldFrenzy(true, 0, 0) {
		t.Fatal("zero max HP must not frenzy")
	}
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestShouldFrenzy -v`
Expected: FAIL — `shouldFrenzy` undefined.

- [ ] **Step 4: Implement the trigger**

In `internal/hooks/mutation_frenzy.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// frenziedBuffId is the Blood Frenzy state buff (see 90-frenzied.yaml).
const frenziedBuffId = 90

// shouldFrenzy reports whether a bearer of the battle-frenzy mutation flag
// should be in the frenzy state: below half of max health.
func shouldFrenzy(hasFlag bool, health, maxHealth int) bool {
	if !hasFlag || maxHealth <= 0 {
		return false
	}
	return health*2 < maxHealth
}

// tickBloodFrenzy maintains the Blood Frenzy state each round for an actor that
// carries the battle-frenzy mutation flag. Applied via the actor buff wrapper
// (start text + GMCP), refreshed while wounded; the buff's own 2-round life
// lets it lapse once the actor recovers past half health.
func tickBloodFrenzy(actor actions.Actor) {
	char := actor.GetCharacter()
	if char == nil || !char.IsInCombat() {
		return
	}
	has := mutations.HasMutationFlag(char.Mutations, "battle-frenzy")
	if shouldFrenzy(has, char.Health, char.HealthMax.Value) {
		actor.AddBuff(frenziedBuffId, "blood-frenzy")
	}
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/hooks/ -run TestShouldFrenzy -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/mutation_frenzy.go internal/hooks/mutation_frenzy_test.go internal/mutations/describe.go
git commit -m "feat(hooks): blood-frenzy trigger (tickBloodFrenzy) + describe"
```

---

### Task 4: Wire the trigger into the round ticks

**Files:** Modify `internal/hooks/NewRound_UserRoundTick.go`, `internal/hooks/NewRound_MobRoundTick.go`

- [ ] **Step 1: User tick**

In `NewRound_UserRoundTick.go`, in the per-player loop (near the other per-round combat effects, alongside the mutation-progress block), add — using the same `actions.NewUserActor(user)`-style actor wrapper the file already uses for actor calls (grep the file for how it builds an `actions.Actor` from `user`; mirror it):

```go
	tickBloodFrenzy(actions.NewUserActor(user))
```

- [ ] **Step 2: Mob tick**

In `NewRound_MobRoundTick.go`, in the active per-mob section (alongside `tickMobMutationAcquisition(mob, &mb)`), add:

```go
	tickBloodFrenzy(actions.NewMobActor(mob))
```

> Confirm the exact actor constructors (`actions.NewUserActor` / `actions.NewMobActor` or the local equivalents) by grepping how these two files already wrap user/mob into `actions.Actor` for other calls; use whatever they use.

- [ ] **Step 3: Build + hooks suite**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean + PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go internal/hooks/NewRound_MobRoundTick.go
git commit -m "feat(hooks): drive blood-frenzy from the user + mob round ticks"
```

---

## Phase 3 — Content

### Task 5: `blood-frenzy.yaml` + help

**Files:** Create the mutation YAML + help template.

- [ ] **Step 1: Write `_datafiles/world/dogmud/mutations/blood-frenzy.yaml`**

```yaml
mutationid: blood-frenzy
name: Blood Frenzy
description: |
  Predatory adrenal glands flood you the moment a fight turns against you.
  Wounded, you come off the chain — you hit like something cornered and rabid,
  and you could not turn your back on prey if you tried.
rarity: 6
clusters: [ravener]
pole: body
visual: A network of dark, engorged glands stands out along their throat and shoulders.
pros:
  - type: flag
    target: battle-frenzy
```

- [ ] **Step 2: Write `_datafiles/world/dogmud/templates/help/blood-frenzy.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="yellow">Blood Frenzy</ansi> mutation

Predatory adrenal glands flood you the moment a fight turns against
you. Wounded, you come off the chain.

<ansi fg="yellow">Type:</ansi>     Passive (triggered)
<ansi fg="yellow">Rarity:</ansi>   Rare

<ansi fg="yellow">Benefits:</ansi>
  When you drop below half health in a fight you enter a frenzy and
  your blows land noticeably harder

<ansi fg="yellow">Drawbacks:</ansi>
  While frenzied you cannot flee — you can only fight

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mutations</ansi>
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/blood-frenzy.yaml _datafiles/world/dogmud/templates/help/blood-frenzy.template
git commit -m "content(mutations): blood-frenzy (Ravener triggered-state keystone)"
```

---

## Phase 4 — Verification

### Task 6: Build, suites, boot smoke, manual smoke

- [ ] **Step 1: Build + affected suites**

Run: `go build ./... && go test ./internal/buffs/... ./internal/usercommands/... ./internal/hooks/... ./internal/mutations/... ./internal/devtools/...`
Expected: build clean, all PASS (devtools help-completeness passes with the new help file).

- [ ] **Step 2: Full suite** (catches cross-package completeness like Wave 1's help-file gap)

Run: `go test ./...`
Expected: 0 failing packages.

- [ ] **Step 3: Boot smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|buffs.*Load|panic:"
```
Expected: mutations + buffs load (mutation count = 49 + 1 = 50), no panic. Ctrl-C after world load.

- [ ] **Step 4: Manual smoke (records the state end-to-end)**

Grant `blood-frenzy` to a test character, fight until below half HP: confirm the "The world goes red…" start text fires, `flee` is refused with the rage message, and hits land harder; then win/heal and confirm the frenzy lapses after a couple of rounds.

- [ ] **Step 5: Commit** (only if Step 1–3 required a fix)

---

## Self-Review (completed during authoring)

- **Spec coverage:** Wave 3 P5 (triggered state) via Blood Frenzy, end-to-end (flag → buff → round-tick trigger → flee gate → content). P6 (transformation-apex) is captured as a note: it needs no new framework (passive-bundle + pole + prereq), so real apex authoring rides with Wave 6 once cores exist; bespoke apex effects ride with Waves 4–5. Richer frenzy effects deferred to the balance pass.
- **Placeholder scan:** buff id 90 is flagged as a to-confirm-free placeholder; two spots (the DamageBonus flag string; the actor-constructor names) are explicitly "grep the file and match" because they're local conventions — every other step carries complete code.
- **Type consistency:** `shouldFrenzy(hasFlag bool, health, maxHealth int) bool` and `tickBloodFrenzy(actor actions.Actor)` consistent across impl, test, and call sites; `frenzied`/`battle-frenzy` flag strings consistent across buff YAML, flag constant, mutation YAML, describe, and gate; buff applied via the actor wrapper per the Wave 2 lesson (start text + GMCP).

## Follow-on

- **Wave 4:** auras (P7 — Commanding Presence, Dissonance Organ) + companion extensions (P8 — Brood Sac, Hive Mind).
- **Wave 5:** the two actives (Venom Coat, Cocoon) + Winged Flight.
- **Wave 6:** full per-cluster authoring (all cores + apexes, with prereq spines now satisfiable), migration/re-bloom, `archetype_pull`→cluster re-curation, and the balance pass (including Blood Frenzy's deepening effects: faster attacks, taunt/fear immunity, lifesteal, terror).
