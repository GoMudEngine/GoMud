# Chrysalis Content Wave 5 — Actives + Flight Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the mutation graph's two rare actives (Venom Coat, Cocoon) and the Generalist apex Winged Flight, building the P4 (active-ability, REUSE) and P9 (flight, NEW/bespoke) primitives.

**Architecture:** Venom Coat + Cocoon reuse the established mutation-active pattern (`flag: active-ability` YAML + `TriggerX(actor, opts)` in `internal/actions/mutation_*.go` calling `mutationPreamble` + a buff, wrapped by thin user/mob commands registered in the command tables). Winged Flight is an always-on passive transformation: a `flying` mutation flag read by three bespoke hooks — a single flight edge on the melee opposed roll (flyer beats the earthbound both offensively and defensively), a move-stamina reduction, and a flee-stamina reduction.

**Tech Stack:** Go. Packages: `internal/actions`, `internal/usercommands`, `internal/mobcommands`, `internal/mutations`, `internal/combat`, `internal/configs`, plus mutation/buff/help YAML + templates.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (§5 items: Venom Coat line 102, Cocoon line 171, Winged Flight line 203; primitives P4 line 217, P9 line 222).

**Design decisions (user-approved 2026-07-11):** Flight is **always-on passive** (not toggled); **no terrain restriction** (applies everywhere for MVP); Cocoon has **no attack-lock** (mitigation + aggro-drop only). Per-rank magnitude curves are **deferred to the Wave 6 playtest** (spec §7) — this wave ships a fixed effect per active and tunable config edges for flight; deepening-per-rank is a follow-on.

---

## Verified context (codegraph/read-confirmed — do not re-discover)

- **Active pattern** (`internal/actions/mutation_healing_gel.go`, `mutation_helpers.go`): `mutationPreamble(actor, mutationKey, combatRequired, staminaCost) preambleResult` runs 4 gates (ownership, combat-required, shared `special-move` cooldown via `char.Cooldowns.Try("special-move", …)`, stamina) and deducts stamina only on full success. `TriggerHealingGel(actor Actor, opts MutationOpts) MutationResult` is the template for a self-buff active (combatRequired=false).
- **Command wrappers**: `internal/usercommands/mutation_healing_gel.go` → `HealingGel(rest, user, room, flags)` = `actions.TriggerHealingGel(actions.NewUserActorInRoom(user, room), actions.MutationOpts{})`; `internal/mobcommands/mutation_healing_gel.go` → mob variant via `NewMobActorInRoom`.
- **Command registration**: `internal/usercommands/usercommands.go` map, e.g. `` `healing-gel`: {HealingGel, false, true, false} ``; a parallel table exists in `internal/mobcommands/`.
- **Buff YAML** (`_datafiles/world/dogmud/buffs/101-emboldened.yaml`): `buffid`, `name`, `description`, `triggerrate` (e.g. `1 round`), `triggercount`, `statmods:` (keys incl. `strength`, `willpower`, `physical_mitigation`, `magical_mitigation`, `conviction_mitigation`), `start_user_text`, `end_user_text`. Next free buff ids: **103, 104** (100=blood_frenzy, 101=emboldened, 102=disrupted).
- **Mutation flags**: `mutations.GetMutationFlags(owned)` / `HasMutationFlag(owned, flag)` read `type: flag` effects (Target = flag name). `internal/mutations/describe.go` `flagPhrase` renders flags for the `mutations` command — needs a `flying` case. `DescribeEffect` renders effect types.
- **Prereq validator** (`internal/mutations/graph.go:38`): **panics at boot if a `prerequisites` id does not exist.** `hollow-bones.yaml` and `tail.yaml` both exist → Winged Flight prereqs resolve.
- **Flee** (`internal/usercommands/flee.go`): flat `fleeStaminaCost = 10` deducted via `user.Character.DeductStamina`. A parallel `internal/mobcommands/flee.go` exists (carries the NoFlee gate).
- **Help completeness**: `TestHelpFileCompleteness_Mutations` requires a `_datafiles/world/dogmud/templates/help/{mutationid}.template` for every mutation, or `go test ./...` fails.
- **Existing mutation files**: `venom-glands.yaml` exists (Wave 2 on-hit) — Venom **Coat** is a NEW, distinct file. `venom-coat.yaml`, `cocoon.yaml`, `winged-flight.yaml` do not exist.

---

## File Structure

- `internal/actions/mutation_venom_coat.go` — `TriggerVenomCoat` (self-buff active).
- `internal/actions/mutation_cocoon.go` — `TriggerCocoon` (self-buff active + aggro-drop).
- `internal/usercommands/mutation_venom_coat.go`, `internal/mobcommands/mutation_venom_coat.go` — wrappers.
- `internal/usercommands/mutation_cocoon.go`, `internal/mobcommands/mutation_cocoon.go` — wrappers.
- `internal/usercommands/usercommands.go` + the mobcommands table — register `venom-coat`, `cocoon`.
- `internal/mutations/flight.go` — `IsFlying(owned)` helper.
- `internal/mutations/describe.go` — `flying` flagPhrase case.
- `internal/combat/flight.go` — `FlightOpposedEdge(attacker, defender)` combat modifier + wiring into the melee opposed roll.
- `internal/usercommands/go.go`, `internal/usercommands/flee.go` — flight move/flee stamina hooks.
- `internal/configs/config.balance.go` + `.combat.go` — flight config knobs.
- Buffs: `_datafiles/world/dogmud/buffs/103-venom_coat.yaml`, `104-cocoon.yaml`.
- Mutations: `_datafiles/world/dogmud/mutations/venom-coat.yaml`, `cocoon.yaml`, `winged-flight.yaml`.
- Help: `templates/help/venom-coat.template`, `cocoon.template`, `winged-flight.template`.

---

### Task 1: Venom Coat buff (103)

**Files:**
- Create: `_datafiles/world/dogmud/buffs/103-venom_coat.yaml`

- [ ] **Step 1: Author the buff.** Weapon-slicked-in-venom: an outgoing damage edge + a to-hit edge for a duration. Number-free player text. Use statmods that flow through existing consumers (`strength` lifts physical damage + hit rolls via the Strength-based melee pipeline).

```yaml
buffid: 103
name: Venom Coat
description: Your weapons run slick with your own venom, each strike carrying a burning, debilitating edge.
triggerrate: 1 round
triggercount: 15
statmods:
  strength: 12
start_user_text: You flex, and a slick sheen of venom weeps across your weapons. Every blow will bite deeper now.
end_user_text: The last of the venom dries and flakes away from your weapons.
```

- [ ] **Step 2: Boot smoke** (buffs validate at load). Nuke instance saves, boot, confirm `buffSpec.LoadDataFiles()` loads without panic, quit. (Full boot deferred to Task 12; a quick load check is enough here.)

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/buffs/103-venom_coat.yaml
git commit -m "content(mutations): Venom Coat buff (103)"
```

---

### Task 2: Venom Coat action + commands + mutation + help

**Files:**
- Create: `internal/actions/mutation_venom_coat.go`
- Create: `internal/actions/mutation_venom_coat_test.go`
- Create: `internal/usercommands/mutation_venom_coat.go`, `internal/mobcommands/mutation_venom_coat.go`
- Modify: `internal/usercommands/usercommands.go` + the mobcommands registration table
- Create: `_datafiles/world/dogmud/mutations/venom-coat.yaml`, `templates/help/venom-coat.template`

- [ ] **Step 1: Write the failing test** (`mutation_venom_coat_test.go`) — mirror `mutation_healing_gel_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/stretchr/testify/assert"
)

func TestTriggerVenomCoat_NoMutation(t *testing.T) {
	actor, _ := newTestPlayerActor(t) // use the same helper the healing-gel test uses
	res := TriggerVenomCoat(actor, MutationOpts{})
	assert.Equal(t, "no-mutation", res.BlockReason)
}

func TestTriggerVenomCoat_AppliesBuff(t *testing.T) {
	actor, char := newTestPlayerActor(t)
	char.Mutations = map[string]int{"venom-coat": 1}
	char.Stamina = 100
	res := TriggerVenomCoat(actor, MutationOpts{})
	assert.True(t, res.Triggered)
	assert.True(t, char.HasBuff(103))
	_ = buffs.Buff{}
}
```

> Use whatever player-actor test helper `mutation_healing_gel_test.go` uses (read it for the exact constructor/`HasBuff` API); match it rather than inventing one.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestTriggerVenomCoat`
Expected: FAIL — undefined `TriggerVenomCoat`.

- [ ] **Step 3: Implement the action** (`mutation_venom_coat.go`) — template from `mutation_healing_gel.go`, `combatRequired=false`, stamina cost 8, applies buff 103 via the actor buff path:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// TriggerVenomCoat fires the venom-coat mutation for any Actor. Self-buff:
// slicks the actor's weapons in venom (buff 103) for a burst of extra bite.
// Gates: owns "venom-coat" + shared special-move cooldown + 8 stamina.
// Combat not required (prep move).
func TriggerVenomCoat(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "venom-coat", false, 8)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}
	actor.AddBuff(103, "venom-coat")
	if actor.IsPlayer() {
		actor.SendText(messaging.CategoryMutation,
			`<ansi fg="green-bold">You flex, and venom weeps slick across your weapons.</ansi>`)
	}
	if room := actor.GetRoom(); room != nil {
		room.SendTextVisual(messaging.CategoryMutation,
			`<ansi fg="green"><ansi fg="username">`+actor.GetName()+`</ansi>'s weapons glisten with a sudden venomous sheen.</ansi>`,
			actor.GetUserId())
	}
	actor.OnSkillUse(string(skills.WeaponCombat))
	return MutationResult{Triggered: true, AffectedCount: 1}
}
```

> Confirm the `Actor` interface exposes `AddBuff(id int, source string)` — the healing-gel path uses `char.Heal`, but other actives (blinding-flash/sonic-shout) apply buffs; read one that applies a buff to the actor and match its exact call (it may be `actor.GetCharacter().AddBuff` routed through the actor wrapper, or an `actor.AddBuff`). Use the event-queue-safe wrapper, not a raw `Character.AddBuff` (per the Wave 2 gotcha).

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/actions/ -run TestTriggerVenomCoat`
Expected: PASS.

- [ ] **Step 5: Command wrappers + registration.** Create user + mob wrappers (copy `mutation_healing_gel.go` in both dirs, rename to `VenomCoat`/`TriggerVenomCoat`). Register in `usercommands.go` (`` `venom-coat`: {VenomCoat, false, true, false} ``) and the mobcommands table (match the `healing-gel` entry there).

- [ ] **Step 6: Mutation YAML** (`venom-coat.yaml`) — Ravener, rarity 7, `flag: active-ability`. (Cluster/pole tags land in Wave 6; leave untagged now so it stays inert-but-usable.)

```yaml
mutationid: venom-coat
name: Venom Coat
description: On command, your venom glands flush your skin and weapons with a slick, burning toxin. For a short while every strike you land carries its bite — but working the venom up costs you.
rarity: 7
visual: A faint, oily green film clings to the edges of their weapons and knuckles.
pros:
  - type: flag
    target: active-ability
    value: 1
```

- [ ] **Step 7: Help template** (`templates/help/venom-coat.template`) — 80-col, describe the `venom-coat` command, the shared-cooldown cost, no hard numbers.

- [ ] **Step 8: Build + focused test**

Run: `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ -run 'VenomCoat|Command'`
Expected: clean.

- [ ] **Step 9: Commit**

```bash
git add internal/actions/mutation_venom_coat*.go internal/usercommands/mutation_venom_coat.go internal/usercommands/usercommands.go internal/mobcommands/mutation_venom_coat.go internal/mobcommands/*.go _datafiles/world/dogmud/mutations/venom-coat.yaml _datafiles/world/dogmud/templates/help/venom-coat.template
git commit -m "feat(mutations): Venom Coat active (P4)"
```

---

### Task 3: Cocoon buff (104)

**Files:**
- Create: `_datafiles/world/dogmud/buffs/104-cocoon.yaml`

- [ ] **Step 1: Author the buff** — near-invulnerable encasement: very high mitigation across all three channels, short duration.

```yaml
buffid: 104
name: Cocoon
description: You have sealed yourself inside a hardened chrysalis shell, and the blows of the world barely reach you.
triggerrate: 1 round
triggercount: 3
statmods:
  physical_mitigation: 85
  magical_mitigation: 85
  conviction_mitigation: 85
start_user_text: You fold inward and a hard, translucent shell snaps shut around you. The world's blows dull to distant taps.
end_user_text: Your shell splits and sloughs away, and the world rushes back in.
```

- [ ] **Step 2: Boot-load check** (as Task 1 Step 2). **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/buffs/104-cocoon.yaml
git commit -m "content(mutations): Cocoon buff (104)"
```

---

### Task 4: Cocoon action + commands + mutation + help

**Files:** mirror Task 2 for `cocoon` (action, user/mob wrappers, registration, YAML, help, test). Cocoon differs from Venom Coat in two ways: it applies buff 104 **and drops aggro**.

- [ ] **Step 1: Failing test** (`mutation_cocoon_test.go`) — assert buff 104 applied, and that a room mob previously aggro'd on the actor has its aggro cleared. Model the aggro-drop assertion on however combat/charm tests set + read mob aggro (`SetAggro` / `IsAggro` / `EndAggro`).

- [ ] **Step 2: Verify it fails** — `go test ./internal/actions/ -run TestTriggerCocoon`.

- [ ] **Step 3: Implement `TriggerCocoon`** — `mutationPreamble(actor, "cocoon", false, 10)`, apply buff 104, then the bespoke aggro-drop: iterate the actor's room mobs and `EndAggro()` any whose current target is the actor (the "vanish from threat"). No attack-lock (design decision).

```go
func TriggerCocoon(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "cocoon", false, 10)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}
	actor.AddBuff(104, "cocoon")
	// Drop aggro: every mob in the room currently fixed on the actor loses it.
	dropped := 0
	if room := actor.GetRoom(); room != nil {
		for _, mobId := range room.GetMobs() { // confirm the room mob-id accessor name
			m := mobs.GetInstance(mobId)
			if m == nil {
				continue
			}
			if m.Character.IsInCombat() && m.Character.CurrentCombatTarget().UserId == actor.GetUserId() {
				m.Character.EndAggro()
				dropped++
			}
		}
	}
	if actor.IsPlayer() {
		actor.SendText(messaging.CategoryMutation,
			`<ansi fg="cyan-bold">You fold inward; a hard shell snaps shut and the fight loses sight of you.</ansi>`)
	}
	return MutationResult{Triggered: true, AffectedCount: dropped}
}
```

> Confirm the room→mob-ids accessor (`GetMobs`) and `CurrentCombatTarget().UserId` field names against a file that already iterates room mobs for aggro (the companion_summon.go aggro-clear loop is a good reference). Match exact names.

- [ ] **Step 4: Verify it passes.** **Step 5:** user/mob wrappers + register `cocoon`. **Step 6:** `cocoon.yaml` — bridge (Ethereal⇄Weaver), rarity 8, `flag: active-ability`. **Step 7:** help template. **Step 8:** `go build ./... && go test ./internal/actions/ …`.

- [ ] **Step 9: Commit** `feat(mutations): Cocoon active — near-invuln + aggro-drop (P4)`.

---

### Task 5: Flight flag helper + flying descriptor

**Files:**
- Create: `internal/mutations/flight.go`, `internal/mutations/flight_test.go`
- Modify: `internal/mutations/describe.go` (flagPhrase `flying` case)

- [ ] **Step 1: Failing test** (`flight_test.go`):

```go
package mutations

import "testing"

func TestIsFlying(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"winged-flight": {MutationId: "winged-flight", Name: "Winged Flight", Rarity: 8,
			Pros: []MutationEffect{{Type: "flag", Target: "flying"}}},
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3},
	})
	defer cleanup()
	if IsFlying(map[string]int{"claws": 2}) {
		t.Fatal("no flight mutation -> not flying")
	}
	if !IsFlying(map[string]int{"winged-flight": 1}) {
		t.Fatal("winged-flight -> flying")
	}
}
```

- [ ] **Step 2: Verify fails** — `go test ./internal/mutations/ -run TestIsFlying`.

- [ ] **Step 3: Implement** (`flight.go`):

```go
package mutations

// IsFlying reports whether any owned mutation grants the "flying" flag
// (the Winged Flight transformation). Read by combat/flee/movement hooks.
func IsFlying(owned map[string]int) bool {
	return HasMutationFlag(owned, "flying")
}
```

Add to `describe.go` `flagPhrase` (before the closing default):

```go
	case "flying":
		return "You take to the air on wings — swift over ground, hard for the earthbound to touch, and free to break away at will."
```

- [ ] **Step 4: Verify passes** — `go test ./internal/mutations/ -run 'TestIsFlying|TestDescribeEffect'`.

- [ ] **Step 5: Commit** `feat(mutations): flying flag helper + descriptor`.

---

### Task 6: Flight config knobs

**Files:**
- Modify: `internal/configs/config.balance.combat.go` (struct fields near other combat knobs) + its defaults path
- Test: `internal/configs/…_test.go`

- [ ] **Step 1: Failing test** asserting defaults: `FlightOpposedEdge` (default 25 — a to-hit/dodge edge on the opposed roll vs the earthbound), `FlightMoveStaminaMult` (default 0.5 — half move-stamina while flying), `FlightFleeStaminaMult` (default 0.5 — half flee cost). Match the `ConfigInt`/`ConfigFloat` types of neighbors.

- [ ] **Step 2: verify fails. Step 3:** add the three fields + `<= 0` / `< 1` default guards (mirror the Task-2 companion-knob pattern from the prior wave). **Step 4: verify passes. Step 5: commit** `feat(mutations): flight config knobs`.

---

### Task 7: Flight combat edge (opposed roll)

**Files:**
- Create: `internal/combat/flight.go`, `internal/combat/flight_test.go`
- Modify: the melee attacker-vs-defender opposed-roll site in `internal/combat/combat_helpers.go` (locate the function that has BOTH `sourceChar`/`targetChar` — `resolveDefenseOutcome` at :701 and its caller build the scores; read in-context to find where the attacker score / defense score are finalized before the roll).

- [ ] **Step 1: Failing test** (`flight_test.go`) for a pure helper:

```go
package combat

import "testing"

func TestFlightOpposedEdge(t *testing.T) {
	// attacker flying, defender grounded -> positive edge (attacker advantage)
	if e := flightEdge(true, false, 25); e != 25 {
		t.Fatalf("flyer vs grounded = 25, got %d", e)
	}
	// defender flying, attacker grounded -> negative edge (defender advantage)
	if e := flightEdge(false, true, 25); e != -25 {
		t.Fatalf("grounded vs flyer = -25, got %d", e)
	}
	// both flying (or both grounded) -> cancels
	if e := flightEdge(true, true, 25); e != 0 {
		t.Fatalf("flyer vs flyer = 0, got %d", e)
	}
	if e := flightEdge(false, false, 25); e != 0 {
		t.Fatalf("grounded vs grounded = 0, got %d", e)
	}
}
```

- [ ] **Step 2: verify fails.**

- [ ] **Step 3: Implement the pure helper** (`internal/combat/flight.go`):

```go
package combat

// flightEdge returns the signed edge applied to the attacker's side of a melee
// opposed roll from the flight mismatch: a flyer beats the earthbound both when
// attacking (strike from angle) and when defending (dodge earthbound); when both
// or neither fly, it cancels.
func flightEdge(attackerFlying, defenderFlying bool, edge int) int {
	if attackerFlying && !defenderFlying {
		return edge
	}
	if defenderFlying && !attackerFlying {
		return -edge
	}
	return 0
}
```

- [ ] **Step 4: verify passes** — `go test ./internal/combat/ -run TestFlightOpposedEdge`.

- [ ] **Step 5: Wire into the melee opposed roll.** Read the attack-resolution function in `combat_helpers.go` that computes the attacker score vs the best defense score for a MELEE swing (both `*characters.Character` in scope). Add, at the point the scores are finalized:

```go
if edge := flightEdge(mutations.IsFlying(attacker.Mutations), mutations.IsFlying(defender.Mutations),
	int(configs.GetBalanceConfig().FlightOpposedEdge)); edge != 0 {
	attackerScore += float64(edge) // apply to the attacker's side of the opposed roll
}
```

Use the actual attacker/defender char variable names and score variable at that site. Guard: apply only to melee/physical swings (not spell/ranged) unless the site is already melee-specific. Add `mutations` + `configs` imports if absent.

- [ ] **Step 6: Build + combat tests** — `go build ./... && go test ./internal/combat/`. Expected: clean (no existing combat test regressions).

- [ ] **Step 7: Commit** `feat(mutations): Winged Flight combat edge vs the earthbound (P9)`.

---

### Task 8: Flight movement + flee hooks

**Files:**
- Modify: `internal/usercommands/go.go` (move-stamina cost site), `internal/usercommands/flee.go` (fleeStaminaCost)

- [ ] **Step 1: Flee hook.** In `flee.go`, before the `const fleeStaminaCost = 10` deduction, scale the cost by `FlightFleeStaminaMult` when the character is flying:

```go
fleeStaminaCost := 10
if mutations.IsFlying(user.Character.Mutations) {
	fleeStaminaCost = int(float64(fleeStaminaCost) * float64(configs.GetBalanceConfig().FlightFleeStaminaMult))
	if fleeStaminaCost < 1 {
		fleeStaminaCost = 1
	}
}
```

Change the `const` to a `:=`, add `mutations` + `configs` imports.

- [ ] **Step 2: Move-stamina hook.** In `go.go`, locate the movement stamina-cost computation (the encumbrance move-stamina multiplier per CLAUDE.md). Scale the final move stamina cost by `FlightMoveStaminaMult` when flying (flyers glide over terrain). Read the exact variable in-context and apply the multiplier just before the stamina is deducted.

- [ ] **Step 3: Build + tests** — `go build ./... && go test ./internal/usercommands/`. If there's no unit test seam for these (they need a room/move harness), rely on the build + the Task 12 boot/smoke and note it.

- [ ] **Step 4: Commit** `feat(mutations): Winged Flight movement + flee benefits (P9)`.

---

### Task 9: Winged Flight mutation + help

**Files:**
- Create: `_datafiles/world/dogmud/mutations/winged-flight.yaml`, `templates/help/winged-flight.template`

- [ ] **Step 1: Author the mutation** — Generalist apex, rarity 8, `flag: flying`, **prereq hollow-bones + tail**. It is a transformation: bundle a modest passive alongside the flag (the flight hooks carry the real punch). No pole tag (Generalist → no pole choke).

```yaml
mutationid: winged-flight
name: Winged Flight
description: Your hollow bones and balancing tail finish what they started — great membranous wings unfurl from your back, and the ground releases its claim on you. You move above the fight, striking down at the earthbound and slipping away on the wind, though your lightened frame bruises easily.
rarity: 8
visual: Vast, translucent wings arch from their shoulders, stirring the air with every movement.
prerequisites:
  - mutationid: hollow-bones
    level: 1
  - mutationid: tail
    level: 1
pros:
  - type: flag
    target: flying
    value: 1
```

> Confirm the exact `prerequisites` YAML shape against an existing mutation that uses it (or `internal/mutations/graph.go` `MutationPrereq` struct tags). If none in data yet, match the struct's yaml tags precisely — a wrong key is a silent no-op that would make Flight ungated.

- [ ] **Step 2: Help template** (`winged-flight.template`) — 80-col; describe the four benefits in feel terms (no numbers), and that it requires hollow bones + a tail.

- [ ] **Step 3: Boot smoke** — nuke instances, boot, confirm the prereq validator passes (no `prerequisite … does not exist` panic) and `mutators.LoadDataFiles()` loads clean. Quit.

- [ ] **Step 4: Commit** `content(mutations): Winged Flight apex (Generalist, prereq hollow-bones+tail)`.

---

### Task 10: Help-completeness + describe coverage

**Files:** verification task.

- [ ] **Step 1:** Run `go test ./... -run 'TestHelpFileCompleteness_Mutations|TestDescribeEffect'`. Expected: PASS (Tasks 2/4/9 authored the three help files; Task 5 added the `flying` phrase). If it flags a missing template or an effect type with no `DescribeEffect` case, add it and re-run.

- [ ] **Step 2: Commit** any fixes: `test(mutations): wave 5 help/describe coverage`.

---

### Task 11: Full suite

- [ ] Run `go test ./...`. Expected: exit 0. Fix any regression (watch `internal/actions`, `internal/combat`, `internal/usercommands`, `internal/mobcommands`, `internal/mutations`, `internal/configs`). Commit if fixes needed.

---

### Task 12: Patch notes + full boot smoke

**Files:** `PATCH_NOTES.md`

- [ ] **Step 1:** Add a dated entry (player terms, no numbers): a venom-coating strike-buff and a defensive chrysalis that shrugs off blows and drops you from a fight; and, for those who go light and grow a tail, true flight — faster travel, an edge over earthbound foes, and an easy exit.
- [ ] **Step 2:** `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`, then boot `go run .`; confirm clean load through `mutators.LoadDataFiles()` / `buffSpec.LoadDataFiles()`, mapper errors=0, Server Ready, no panic. Quit.
- [ ] **Step 3: Commit** `docs(mutations): patch notes for Wave 5 actives + flight`.

---

## Out of scope (follow-on)

- **Per-rank deepening curves** (Venom Coat duration/strength per rank, Cocoon mitigation/duration per rank, Flight edge scaling) — deferred to the Wave 6 playtest (spec §7). This wave ships a fixed effect per active + tunable flight config edges.
- **Cluster/pole tags + full prereq spines** on venom-coat/cocoon — Wave 6 authoring.
- **Flight terrain restriction** (grounded indoors/underground) — deliberately deferred (user decision); the flight hooks are flag-gated so a biome check can be added later in one place.
- **Mob AI use of the actives** (btree triggering Venom Coat/Cocoon) — the mob command wrappers exist for parity, but wiring them into behavior trees is separate.
