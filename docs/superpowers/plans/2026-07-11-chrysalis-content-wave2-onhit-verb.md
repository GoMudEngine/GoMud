# Chrysalis Content — Wave 2: on-hit-buff hook + verb-enhancement

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Establish two reusable mechanic recipes — (P2) a data-driven **on-hit-buff** system (an attacker's mutation applies a buff to the defender on a landed hit) and (P3) **verb-enhancement** (a mutation reshapes an existing command) — each proven with one exemplar: **Venom Glands** (on-hit venom, reusing buff 39) and **Raptor Legs** (enhances `kick`).

**Architecture:** The on-hit-buff mechanism is a new mutation effect `type: on_hit_buff` whose `value` is a buff id; a helper collects them and the existing post-hit hook `applyCombatDamageBonuses` (which already reads attacker mutations and touches the defender) applies them. Verb-enhancement mirrors the established `trip`→tailsweep pattern: `kick` checks for the mutation and boosts its params. No new buff is authored (Venom reuses buff 39).

**Tech Stack:** Go, `internal/mutations`, `internal/hooks` (combat round), `internal/actions` (kick), YAML mutation + help data, testify.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (Wave 2 of §9). Builds on the Wave 1 recipe (merged).

**Scope — deferred to Wave 2b (same on-hit recipe, each authors a NEW buff):** Rending Claws (bleed buff), Grasping Tendrils (root buff), Evil Eye (accuracy/defense-down curse buff). Wave 2 proves the recipe with the zero-new-buff exemplar (Venom Glands → buff 39) so the mechanism lands before the buff-authoring fan-out.

**Reminder (from Wave 1):** every new mutation YAML needs a matching `_datafiles/world/dogmud/templates/help/{id}.template` or `TestHelpFileCompleteness_Mutations` fails; and every effect type a mutation uses needs a `DescribeEffect` case or the `mutations` command shows a blank benefit line.

---

## File Structure

**Modify:**
- `internal/mutations/mutations.go` — `GetOnHitBuffs` helper
- `internal/mutations/describe.go` — `on_hit_buff` describe case
- `internal/hooks/NewRound_DoCombat_unified.go` — apply on-hit buffs in `applyCombatDamageBonuses`
- `internal/actions/combat_kick.go` — Raptor Legs kick enhancement

**Create:**
- `_datafiles/world/dogmud/mutations/venom-glands.yaml` + `.../help/venom-glands.template`
- `_datafiles/world/dogmud/mutations/raptor-legs.yaml` + `.../help/raptor-legs.template`
- Tests: `internal/mutations/onhit_test.go`, `internal/actions/combat_kick_raptor_test.go`

---

## Phase 1 — On-hit-buff mechanism (P2 recipe)

### Task 1: `GetOnHitBuffs` helper + describe case

**Files:** Modify `internal/mutations/mutations.go`, `internal/mutations/describe.go`; Test `internal/mutations/onhit_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/onhit_test.go
package mutations

import "testing"

func TestGetOnHitBuffs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"venom-glands": {MutationId: "venom-glands", Name: "Venom Glands", Rarity: 7,
			Pros: []MutationEffect{{Type: "on_hit_buff", Value: 39}}},
		"plain": {MutationId: "plain", Name: "Plain", Rarity: 2,
			Pros: []MutationEffect{{Type: "stat_flat", Target: "strength", Value: 5}}},
	})
	defer cleanup()

	got := GetOnHitBuffs(map[string]int{"venom-glands": 1, "plain": 1})
	if len(got) != 1 || got[0] != 39 {
		t.Fatalf("GetOnHitBuffs = %v, want [39]", got)
	}
	if len(GetOnHitBuffs(map[string]int{})) != 0 {
		t.Fatal("no mutations → no on-hit buffs")
	}
}

func TestDescribeEffect_OnHitBuff(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "on_hit_buff", Value: 39}) == "" {
		t.Fatal("on_hit_buff must have a non-empty description")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mutations/ -run "TestGetOnHitBuffs|TestDescribeEffect_OnHitBuff" -v`
Expected: FAIL — `GetOnHitBuffs` undefined; describe returns "".

- [ ] **Step 3: Implement**

In `internal/mutations/mutations.go`, near the other `Get*` helpers:

```go
// GetOnHitBuffs returns the buff ids that owned mutations apply to a struck
// target on a landed melee/natural hit (effect type "on_hit_buff", Value = buff id).
func GetOnHitBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "on_hit_buff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}
```

In `internal/mutations/describe.go`, add a case to the `DescribeEffect` switch:

```go
	case "on_hit_buff":
		return "Your natural strikes leave a debilitating affliction in the wound."
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mutations/ -run "TestGetOnHitBuffs|TestDescribeEffect_OnHitBuff" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/describe.go internal/mutations/onhit_test.go
git commit -m "feat(mutations): on_hit_buff effect type — GetOnHitBuffs + describe"
```

---

### Task 2: Apply on-hit buffs in the post-hit combat hook

**Files:** Modify `internal/hooks/NewRound_DoCombat_unified.go` (`applyCombatDamageBonuses`, ~line 321)

- [ ] **Step 1: Add the application**

`applyCombatDamageBonuses` already early-returns unless `res.Hit && res.DamageToTarget > 0`, and has `atkChar`/`defChar`. After the `defChar := def.GetCharacter()` line (~326), add:

```go
	// Mutation graph: on-hit-buff mutations (Venom Glands, …) afflict the
	// struck defender. Buff specs are keyed by id; Character.AddBuff is nil-safe
	// and no-ops on an unknown id.
	for _, buffId := range mutations.GetOnHitBuffs(atkChar.Mutations) {
		_ = defChar.AddBuff(buffId, false)
	}
```

(`mutations` is already imported — it's used a few lines below for Adrenaline Surge.)

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 3: Run the hooks suite (no regression)**

Run: `go test ./internal/hooks/...`
Expected: PASS. (The wiring is a thin, guarded call; correctness of buff application is verified by the boot/manual smoke in Task 6 — unit-testing it requires loaded buff specs + Actor construction, which the boot smoke covers end-to-end.)

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_unified.go
git commit -m "feat(hooks): on-hit-buff mutations afflict the struck defender"
```

---

## Phase 2 — Verb-enhancement (P3 recipe)

### Task 3: Raptor Legs enhances `kick`

**Files:** Modify `internal/actions/combat_kick.go`; Test `internal/actions/combat_kick_raptor_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/actions/combat_kick_raptor_test.go
package actions

import "testing"

// raptorLegsKickBonus is the pure helper Task 3 extracts; the test pins its
// behavior so the kick path's Raptor Legs branch is verified without a full
// combat harness.
func TestRaptorLegsKickBonus(t *testing.T) {
	dmg, kd := raptorLegsKickBonus(map[string]int{}, 0.80, 20)
	if dmg != 0.80 || kd != 20 {
		t.Fatalf("no mutation → unchanged, got dmg=%v kd=%d", dmg, kd)
	}
	dmg2, kd2 := raptorLegsKickBonus(map[string]int{"raptor-legs": 1}, 0.80, 20)
	if !(dmg2 > 0.80 && kd2 > 20) {
		t.Fatalf("raptor-legs should raise kick damage + knockdown, got dmg=%v kd=%d", dmg2, kd2)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestRaptorLegsKickBonus -v`
Expected: FAIL — `raptorLegsKickBonus` undefined.

- [ ] **Step 3: Implement**

In `internal/actions/combat_kick.go`, add the pure helper (near the top-level funcs):

```go
// raptorLegsKickBonus boosts standard-kick damage and knockdown when the
// attacker has the Raptor Legs mutation (digitigrade, talon-clawed legs).
// Returns the (possibly) adjusted damagePercent and knockdownChance.
func raptorLegsKickBonus(owned map[string]int, damagePercent float64, knockdownChance int) (float64, int) {
	if _, ok := owned["raptor-legs"]; ok {
		damagePercent += 0.20
		knockdownChance += 15
	}
	return damagePercent, knockdownChance
}
```

Then in `ExecuteKick`, after the variant determination block (after the Knee `if` at ~line 116, before `combat.ExecuteSkillMove`), apply it to the standard kick only:

```go
	// Raptor Legs mutation: talon-clawed legs make a plain kick bite far harder.
	if variant == KickStandard {
		damagePercent, knockdownChance = raptorLegsKickBonus(char.Mutations, damagePercent, knockdownChance)
	}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/actions/ -run TestRaptorLegsKickBonus -v`
Expected: PASS

- [ ] **Step 5: Build + actions suite**

Run: `go build ./... && go test ./internal/actions/...`
Expected: clean + PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/actions/combat_kick.go internal/actions/combat_kick_raptor_test.go
git commit -m "feat(actions): Raptor Legs mutation enhances the kick command"
```

---

## Phase 3 — Content

### Task 4: `venom-glands.yaml` + help

**Files:** Create the YAML and help template.

- [ ] **Step 1: Write `_datafiles/world/dogmud/mutations/venom-glands.yaml`**

```yaml
mutationid: venom-glands
name: Venom Glands
description: |
  Glands swell behind your teeth and beneath your nails, weeping a potent
  neurotoxin. Anything you strike with tooth or claw is envenomed — the
  poison does its slow, certain work long after the blow lands.
rarity: 7
clusters: [ravener, stalker]
pole: body
requires_body_parts: [mouth]
visual: Their teeth and nails glisten with a faint, oily sheen that smells of bitter almonds.
pros:
  - type: on_hit_buff
    value: 39
```

- [ ] **Step 2: Write `_datafiles/world/dogmud/templates/help/venom-glands.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="yellow">Venom Glands</ansi> mutation

Glands behind your teeth and beneath your nails weep a potent
neurotoxin. Anything you strike with tooth or claw is envenomed.

<ansi fg="yellow">Type:</ansi>     Passive
<ansi fg="yellow">Rarity:</ansi>   Very Rare

<ansi fg="yellow">Benefits:</ansi>
  Your natural strikes leave a debilitating venom in the wound that
  keeps harming the target after the blow lands

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mutations</ansi>
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/venom-glands.yaml _datafiles/world/dogmud/templates/help/venom-glands.template
git commit -m "content(mutations): venom-glands (Ravener/Stalker on-hit venom bridge)"
```

### Task 5: `raptor-legs.yaml` + help

**Files:** Create the YAML and help template.

- [ ] **Step 1: Write `_datafiles/world/dogmud/mutations/raptor-legs.yaml`**

(The kick enhancement lives in code; the YAML carries a small describable Dexterity passive so it is not a blank-effect mutation and reads as an agile-legs change.)

```yaml
mutationid: raptor-legs
name: Raptor Legs
description: |
  Your legs re-tension into digitigrade, talon-clawed limbs — built to run
  prey down and open it up. You are quicker on your feet, and a kick from
  you lands like a falling axe.
rarity: 4
clusters: [ravener]
pole: body
requires_body_parts: [legs]
visual: Their legs bend the wrong way at the ankle, ending in splayed, black-taloned feet.
pros:
  - type: stat_multiplier
    target: dexterity
    value: 0.05
```

- [ ] **Step 2: Write `_datafiles/world/dogmud/templates/help/raptor-legs.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="yellow">Raptor Legs</ansi> mutation

Your legs re-tension into digitigrade, talon-clawed limbs, built to
run prey down and open it up.

<ansi fg="yellow">Type:</ansi>     Passive
<ansi fg="yellow">Rarity:</ansi>   Uncommon

<ansi fg="yellow">Benefits:</ansi>
  Heightens your Dexterity
  Your <ansi fg="command">kick</ansi> lands far harder and is more likely to knock a
  foe from their feet

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help mutations</ansi>
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/raptor-legs.yaml _datafiles/world/dogmud/templates/help/raptor-legs.template
git commit -m "content(mutations): raptor-legs (Ravener kick-enhance keystone)"
```

---

## Phase 4 — Full verification

### Task 6: Build, suites, boot smoke, help completeness

- [ ] **Step 1: Build + affected suites**

Run: `go build ./... && go test ./internal/mutations/... ./internal/hooks/... ./internal/actions/... ./internal/devtools/...`
Expected: build clean, all PASS (devtools help-completeness passes with the two new help files).

- [ ] **Step 2: Boot smoke (real YAML + on-hit wiring load)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|panic:"
```
Expected: loads (count = 47 + 2 = 49), no panic. Ctrl-C after world load.

- [ ] **Step 3: Manual smoke (recommended, records the P2/P3 behavior end-to-end)**

With an admin/test character: grant `venom-glands`, attack a mob, confirm a Venom DoT lands on hit; grant `raptor-legs`, `kick` a mob, confirm harder hits/knockdowns. (Drift-driven acquisition is validated in the Wave 6 playtest.)

- [ ] **Step 4: Commit** (only if Step 1–2 required a fix)

---

## Self-Review (completed during authoring)

- **Spec coverage:** Wave 2 (§9) P2 on-hit-buff + P3 verb-enhance, each with one exemplar. Rending Claws / Grasping Tendrils / Evil Eye explicitly deferred to Wave 2b (they need new bleed/root/curse buffs — a buff-authoring fan-out that shouldn't gate the mechanism).
- **Placeholder scan:** none — every code, YAML, and help step is complete. Magnitudes (on_hit via buff 39's own tuning; kick +0.20/+15; dex 0.05) are first-pass/tunable per the spec's deferred-magnitude note.
- **Type consistency:** `GetOnHitBuffs(owned map[string]int) []int` and the `on_hit_buff` effect string are consistent across helper, YAML, describe, and hook; `raptorLegsKickBonus(owned, dmg, kd) (float64, int)` consistent between helper, call site, and test. Both new mutations have help files (completeness) and describable effects (Venom via the new `on_hit_buff` case; Raptor Legs via `stat_multiplier`).

## Follow-on

- **Wave 2b:** Rending Claws (author a bleed DoT buff → on_hit_buff), Grasping Tendrils (author a root/immobilize buff), Evil Eye (author an accuracy/defense-down curse buff) — all reuse the Task 1/2 mechanism; each adds one buff YAML + mutation + help + a `DescribeEffect` for its buff flavor if distinct.
- Waves 3–6 per the content spec §9.
