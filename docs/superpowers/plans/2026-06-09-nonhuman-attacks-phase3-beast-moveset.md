# Non-human Attacks — Phase 3: Beast Moveset (rake / maul / pounce / gore / drain / throttle) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the expanded beast special-move set — `rake`, `maul`, `pounce`, `gore`, `drain`, and `throttle` — each with full player↔mob command parity, anatomy/`natural_attack`-identity gating, AI selection, complete combat-message files, helpfiles, and tests.

**Architecture:** Each move is a shared `actions.Execute*` action (resolved through `combat.ExecuteSkillMove`) wrapped by a player handler (`internal/usercommands/`) and a mob handler (`internal/mobcommands/`), gated by `combat.CanUse*` + `Score*` in `internal/combat/ai.go`, mirrored in `actions.CommandIsReady`, and weighted into AI profiles. Gating predicates read the species' `natural_attack` identity (fanged=`bite`, clawed=`claws`, horned=`gore`) plus body parts and a new `LifeDrain` species flag — centralized in small helper predicates so the rules have one source of truth.

**Tech Stack:** Go; `internal/combat`, `internal/actions`, `internal/usercommands`, `internal/mobcommands`, `internal/species`, `internal/characters`, `internal/buffs`; YAML data (`combat-messages/`, species, help templates, `keywords.yaml`).

**Spec:** `docs/superpowers/specs/2026-06-09-nonhuman-attacks-and-beast-moveset-design.md` (Layer 2b beast moveset). Phase 1 (natural-attack messaging) and Phase 2 (anatomy gating of human moves + hamstring-into-AI + bite retirement) are DONE and on local master.

**Decisions locked with the user (2026-06-09):**
1. **Full player↔mob parity** for all six new moves (each gets a player handler + helpfile + keywords + parity-list entry, per spec §B), not mob-only.
2. **Vampire `drain`** replaces the retired bite special: it applies a bleeding debuff to the target and heals the vampire (lifesteal). It is gated on a **new species `LifeDrain` bool flag** (set on species 34 vampire), NOT on `natural_attack` — the vampire stays a weapon-using humanoid (its basic attacks remain `claws`; `body_parts` unchanged).

**Verified facts (2026-06-09, against the codebase):**
- `combat.ExecuteSkillMove(combat.SkillMoveParams{...}) combat.SkillMoveResult{Hit,Damage,KnockedDown,TargetMaxHP}` — `internal/combat/skill_moves.go`. Params include `Attacker,Defender,AttackStat,AttackSkill,DefenseStat,DefenseSkill,DamagePercent,KnockdownChance,SkillRank,DamageStat,MitigationMultiplier,KnockdownToSupine`.
- `actions.ExecuteHamstring` (`internal/actions/combat_hamstring.go`) is the canonical Execute\* template: cooldown via `char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown))`, `ResolveAggroTarget(char.Aggro)`, `ExecuteSkillMove`, on-hit `target.Char.AddCondition(characters.ConditionBleeding, dur, magnitude, "source")`, `combat.RecordSpecialMove(...)`, `char.Aggro.RoundsWaiting = 1`, `actor.OnSkillUse(string(skills.UnarmedCombat))`.
- Bleed: `Character.AddCondition(characters.ConditionBleeding, durationRounds, magnitudeFloat, source)`.
- Heal: `Character.Heal(hp int) int` (`internal/characters/resources.go:180`) — adds HP, clamps to `HealthMax.Value`, returns applied amount.
- Player special-move handler pattern: `internal/usercommands/kick.go` — `func Kick(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)`; resolves/creates a target + `SetAggro` if not already in combat, calls `actions.ExecuteKick(&actions.UserActor{User: user, Room: room})`, then formats messages. Registered in `internal/usercommands/usercommands.go` `userCommands` map.
- Mob special-move handler pattern: `internal/mobcommands/kick.go` — `func Kick(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error)`; silently returns if `!mob.Character.IsInCombat()`, calls `actions.ExecuteKick(&actions.MobActor{Mob: mob, Room: room})`, formats darkness-aware messages. Registered in `internal/mobcommands/mobcommands.go` `mobCommands` map.
- `actions.CommandIsReady` cases in `internal/actions/command_readiness.go`; parity test `supported` slice at `internal/mobcommands/command_parity_test.go:25`.
- AI: `combat.ChooseSpecialMove` builds `moveScores` via `CanUse*`, weights by `aiProfiles` (map in `ai.go:13`), dispatches `mob.Command(name,0)`. `hamstring` already wired (Phase 2).
- Species struct (`internal/species/species.go`): has `NaturalAttack items.ItemSubType`, `NaturalBash bool`, `GrappleImmune bool`, `BodyParts []string`. ADD `LifeDrain bool`.
- Combat-message subtype constants (`internal/items/itemspec.go`): `Bite="bite"`, `Claws="claws"`, `Gore="gore"`, `Slam="slam"`, `Sting="sting"`, `Unarmed`.
- Message-file format: `_datafiles/world/dogmud/combat-messages/<subtype>.yaml`, `optionid:` + `options:` with intensities `prepare/wait/miss/weak/normal/heavy/critical/fumble`, each → `together`/`separate` → `toattacker`/`todefender`/`toroom` → `beginner`/`expert`/`master` lists. `gore.yaml` exists and is the structural template. **The loader (`internal/items/attack_messages.go`) PANICS at boot unless every intensity × tier × audience is present** — new files must be complete.
- Buff flags in `internal/buffs/buffspec.go`; NO silence/cast-block flag exists yet (shout checks `user.Muted`, an admin field). `throttle` must add one.
- Cast entry points: player `internal/usercommands/skill.cast.go`; resolution `internal/hooks/spell_resolution.go`. Shout: `internal/usercommands/shout.go`.

---

## File Structure

**New code files (per move, ×6):**
- `internal/actions/combat_<move>.go` — `Execute<Move>` + `<Move>Result`.
- `internal/usercommands/<move>.go` — player handler.
- `internal/mobcommands/<move>.go` — mob handler.

**Modified code:**
- `internal/species/species.go` — add `LifeDrain bool`; load-time validation helper.
- `internal/combat/ai.go` — `CanUse<Move>`/`Score<Move>` (×6); shared identity predicates; new AI profiles; weight tables; wire into `ChooseSpecialMove`.
- `internal/actions/command_readiness.go` — a `CommandIsReady` case per move.
- `internal/usercommands/usercommands.go`, `internal/mobcommands/mobcommands.go` — register each handler.
- `internal/mobcommands/command_parity_test.go` — add move names to `supported`.
- `internal/actions/cast_interrupt.go` (NEW small shared helper) — `InterruptTargetCast` reusing the engine's existing cast-cancel (`activity.TriggerCastCancel` transition + conviction refund), called by throttle.

**Data/content:**
- `_datafiles/world/dogmud/combat-messages/{maul,pounce,throttle,drain}.yaml` — NEW, full matrix. `rake` reuses `claws.yaml`; `gore.yaml` exists.
- `_datafiles/world/dogmud/species/*.yaml` — add `lifedrain: true` to vampire (34); add `horns` body part to horned (gore) species.
- `_datafiles/world/dogmud/mobs/summons/304-vampire.yaml` — add `drain` to `combatcommands` (replacing the retired bite).
- `_datafiles/world/dogmud/templates/help/{rake,maul,pounce,gore,drain,throttle}.template` — NEW helpfiles.
- `_datafiles/world/dogmud/keywords.yaml` — register each command topic.
- Beast AI profile assignment on beast mob YAML (Task 8).

**Docs:** `internal/combat/context.md`, `internal/actions/context.md`, `internal/mobcommands/context.md`, `internal/usercommands/context.md` (Task 9).

**Move complexity order (easiest → hardest), used as task order:** rake → maul → pounce → gore → drain → throttle.

---

## Task 1: Species `LifeDrain` flag + shared gating predicates + validation

**Files:**
- Modify: `internal/species/species.go`
- Modify: `internal/combat/ai.go`
- Test: `internal/species/species_test.go`, `internal/combat/ai_test.go`

- [ ] **Step 1: Failing test for the species flag + predicates**

In `internal/combat/ai_test.go`:
```go
func TestBeastIdentityPredicates(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		8001: {SpeciesId: 8001, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
		8002: {SpeciesId: 8002, Name: "feline", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Claws},
		8003: {SpeciesId: 8003, Name: "boar", BodyParts: []string{"legs", "mouth", "horns"}, NaturalAttack: items.Gore},
		8004: {SpeciesId: 8004, Name: "vampire", BodyParts: []string{"arms", "hands", "legs"}, NaturalAttack: items.Claws, LifeDrain: true},
	})
	defer cleanup()
	wolf := &characters.Character{SpeciesId: 8001}
	feline := &characters.Character{SpeciesId: 8002}
	boar := &characters.Character{SpeciesId: 8003}
	vamp := &characters.Character{SpeciesId: 8004}

	if !speciesIsFanged(wolf) || speciesIsFanged(feline) {
		t.Error("fanged predicate wrong")
	}
	if !speciesIsClawed(feline) || speciesIsClawed(wolf) {
		t.Error("clawed predicate wrong")
	}
	if !speciesIsHorned(boar) || speciesIsHorned(wolf) {
		t.Error("horned predicate wrong")
	}
	if !speciesHasLifeDrain(vamp) || speciesHasLifeDrain(wolf) {
		t.Error("lifedrain predicate wrong")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`LifeDrain` field + predicates undefined).
Run: `go test ./internal/combat/ -run TestBeastIdentityPredicates -v`

- [ ] **Step 3: Add the species field.** In `internal/species/species.go`, after the `NaturalBash` line:
```go
	NaturalBash      bool             `yaml:"naturalbash,omitempty"`    // Can bash without a shield (elementals, golems)
	LifeDrain        bool             `yaml:"lifedrain,omitempty"`      // Drains life on its `drain` special (vampires, parasites)
```

- [ ] **Step 4: Add the shared predicates.** In `internal/combat/ai.go`, near the top of the viability section (after the `import` block; `items` and `species` are already imported as of Phase 2):
```go
// --- Beast-identity predicates (single source of truth for beast-move gating) ---

func speciesNaturalAttack(char *characters.Character) items.ItemSubType {
	if sp := species.GetSpecies(char.SpeciesId); sp != nil {
		return sp.NaturalAttack
	}
	return ""
}
func speciesIsFanged(char *characters.Character) bool { return speciesNaturalAttack(char) == items.Bite }
func speciesIsClawed(char *characters.Character) bool { return speciesNaturalAttack(char) == items.Claws }
func speciesIsHorned(char *characters.Character) bool { return speciesNaturalAttack(char) == items.Gore }
func speciesHasLifeDrain(char *characters.Character) bool {
	sp := species.GetSpecies(char.SpeciesId)
	return sp != nil && sp.LifeDrain
}
```

- [ ] **Step 5: Run — expect PASS.** `go test ./internal/combat/ -run TestBeastIdentityPredicates -v`; `go build ./...`.

- [ ] **Step 6: Load-time validation for `LifeDrain` + gore→horns.** In `internal/species/species.go`'s existing `LoadDataFiles()` validation loop (the same loop Phase 1 added `validateNaturalAttack` to), add: if `s.NaturalAttack == items.Gore` and `"horns"` is not in `s.BodyParts`, `panic` with a clear message (a horned creature must declare `horns` so `gore` is anatomically valid). Write a focused test `TestSpecies_GoreRequiresHorns` using `SeedSpeciesForTest` is not enough (validation runs in `LoadDataFiles`); instead unit-test a small `validateGoreHasHorns(s *Species) error` helper directly (return error; the loop panics on non-nil). TDD: test the helper returns error when gore+no horns, nil otherwise.

- [ ] **Step 7: Commit.**
```bash
git add internal/species/species.go internal/combat/ai.go internal/species/species_test.go internal/combat/ai_test.go
git commit -m "feat(species): LifeDrain flag + beast-identity predicates + gore-needs-horns validation"
```
(End every commit with: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`)

---

## Task 2: `rake` — clawed dmg + bleed (establishes the full-parity wiring pattern)

`rake` is the simplest move and reuses `claws.yaml` (no new message file), so it establishes the 8-point wiring pattern cleanly. Mechanic: a clawing rake — `ExecuteSkillMove` damage (reuse `TripDamagePercent`, Strength) + a short bleed on hit. Gate: `speciesIsClawed` (+ no extra body part needed — clawed implies the limb).

**Files:**
- Create: `internal/actions/combat_rake.go`, `internal/usercommands/rake.go`, `internal/mobcommands/rake.go`
- Modify: `internal/combat/ai.go`, `internal/actions/command_readiness.go`, `internal/usercommands/usercommands.go`, `internal/mobcommands/mobcommands.go`, `internal/mobcommands/command_parity_test.go`
- Data: `_datafiles/world/dogmud/templates/help/rake.template`, `_datafiles/world/dogmud/keywords.yaml`
- Test: `internal/actions/combat_rake_test.go`, `internal/combat/ai_test.go`

- [ ] **Step 1: Failing test — `ExecuteRake` applies bleed on hit.**
Mirror `internal/actions/combat_hamstring_test.go` if present (else write a fresh test using a seeded clawed attacker + a target, asserting `res.Executed`, and that on a forced hit the target gains `ConditionBleeding`). If the existing action tests can't force a hit deterministically, assert the structural contract (`NoTarget` when `Aggro==nil`; `OnCooldown` when cooldown set) plus `res.Executed==true` with a target — matching how `combat_hamstring_test.go`/`combat_kick_test.go` test their actions. Inspect those test files first and follow their exact harness.

- [ ] **Step 2: Run — expect FAIL** (`ExecuteRake` undefined).

- [ ] **Step 3: Implement `internal/actions/combat_rake.go`** — copy `combat_hamstring.go` verbatim, rename `Hamstring`→`Rake`/`HamstringResult`→`RakeResult`, keep the same `ExecuteSkillMove` params (Dexterity attack, `TripDamagePercent`, Strength damage, `KnockdownChance:0`), on hit apply `target.Char.AddCondition(characters.ConditionBleeding, 4, magnitude, "rake")` where `magnitude = max(2, char.Stats.Strength.ValueAdj/12)`, and `RecordSpecialMove(..., "rake", ...)`. Keep `BleedDmg` field.

- [ ] **Step 4: Add `CanUseRake` + `ScoreRake` to `internal/combat/ai.go`:**
```go
func CanUseRake(char *characters.Character) bool {
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	return speciesIsClawed(char)
}

func ScoreRake(mob *mobs.Mob, target *characters.Character) int {
	score := 45
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 40 {
		score += 15
	}
	if score < 0 {
		score = 0
	}
	return score
}
```
Wire into `ChooseSpecialMove` (alongside the other `CanUse*` checks):
```go
	if CanUseRake(&mob.Character) {
		moveScores["rake"] = ScoreRake(mob, target)
	}
```

- [ ] **Step 5: `CommandIsReady` case** in `internal/actions/command_readiness.go`:
```go
	case "rake":
		if char.Aggro == nil {
			return false
		}
		return char.HasBodyPart("legs") || char.HasBodyPart("arms") // clawed creatures strike with a clawed limb
```
(Identity gating — clawed — lives in `CanUseRake`; readiness only confirms combat + that there is a striking limb, mirroring how the other cases stay light. If the test mob seeding makes this awkward, gate readiness simply on `char.Aggro != nil`; the AI `CanUseRake` is the real gate. Match whatever the existing cases do — read them first.)

- [ ] **Step 6: Player handler `internal/usercommands/rake.go`** — copy `usercommands/kick.go`, rename to `Rake`, delegate to `actions.ExecuteRake`, and write rake-flavored hit/miss messages using `combat.GetDamageDescription` (NO raw numbers; describe the bleed as "raking wounds that won't stop weeping" rather than a number). Keep the combat-initiation/target-resolution block verbatim.

- [ ] **Step 7: Mob handler `internal/mobcommands/rake.go`** — copy `mobcommands/kick.go`, rename to `Rake`, delegate to `actions.ExecuteRake`, format darkness-aware messages.

- [ ] **Step 8: Register both handlers.** Add `"rake": Rake,` to `usercommands.go` `userCommands` and `"rake": {Rake, false},` to `mobcommands.go` `mobCommands` (match the existing map literal's alignment/format).

- [ ] **Step 9: Parity.** Add `"rake"` to the `supported` slice in `command_parity_test.go`.

- [ ] **Step 10: Helpfile + keywords.** Create `templates/help/rake.template` following `templates/help/trip.template`'s shape (usage, what it does — descriptive, no numbers, gating note "natural claws required"). Add a `rake` entry under the combat command topics in `_datafiles/world/dogmud/keywords.yaml` (find how `kick`/`trip` are registered there and mirror it).

- [ ] **Step 11: Run — package tests + parity + build.**
Run: `go test ./internal/actions/ ./internal/combat/ ./internal/mobcommands/ ./internal/usercommands/` → green; `go build ./...` → clean.

- [ ] **Step 12: Commit.**
```bash
git add internal/actions/combat_rake.go internal/usercommands/rake.go internal/mobcommands/rake.go internal/combat/ai.go internal/actions/command_readiness.go internal/usercommands/usercommands.go internal/mobcommands/mobcommands.go internal/mobcommands/command_parity_test.go internal/actions/combat_rake_test.go _datafiles/world/dogmud/templates/help/rake.template _datafiles/world/dogmud/keywords.yaml
git commit -m "feat(combat): add rake beast move (clawed dmg+bleed) with full player/mob parity"
```

---

## Task 3: `maul` — fanged savage flurry (dmg + bleed) + NEW `maul.yaml`

Same 8-point wiring as Task 2. Mechanic: higher damage than rake + a stronger bleed; gate `speciesIsFanged`. NEW message file required.

**Files:** like Task 2 with `Maul`/`maul`, plus Create `_datafiles/world/dogmud/combat-messages/maul.yaml`.

- [ ] **Step 1: Author `maul.yaml` (COMPLETE matrix).** Copy `combat-messages/gore.yaml` as the structural template and rewrite the prose for a fanged savage flurry (tearing, savaging, worrying flesh). It MUST define `optionid: maul` and every intensity `prepare/wait/miss/weak/normal/heavy/critical/fumble` under `together` (and `separate` if `gore.yaml` has it) → `toattacker/todefender/toroom` → `beginner/expert/master`. **critical, fumble, and all three skill tiers are mandatory — the loader panics otherwise.** Verify by listing `gore.yaml`'s keys and matching them exactly.

- [ ] **Step 2: `ExecuteMaul`** — copy `combat_hamstring.go`→`combat_maul.go`, rename, use `KickDamagePercent` (higher than trip) for `DamagePercent`, bleed `AddCondition(ConditionBleeding, 5, max(3, Str/8), "maul")`, `RecordSpecialMove(...,"maul",...)`.

- [ ] **Step 3: `CanUseMaul`/`ScoreMaul`** — like rake but `return speciesIsFanged(char)`; `ScoreMaul` base 55, +15 high skill, +10 if `target.Health` < 50% (finisher). Wire into `ChooseSpecialMove`.

- [ ] **Step 4–10: full parity** — `command_readiness.go` `case "maul"`, player handler `usercommands/maul.go`, mob handler `mobcommands/maul.go`, register both maps, add `"maul"` to `supported`, helpfile `maul.template`, keywords entry. (Same pattern as Task 2; write maul-flavored messages.)

- [ ] **Step 11: Verify the message file loads.** Build, then boot-check is in Task 10; here at minimum run `go test ./internal/items/` (message-load tests) + the package tests + `go build ./...`.

- [ ] **Step 12: Commit** (`feat(combat): add maul beast move (fanged flurry dmg+bleed) + maul.yaml + parity`). Include `maul.yaml`.

---

## Task 4: `pounce` — leap opener (knockdown + bonus dmg) + NEW `pounce.yaml`

Mechanic: a leaping opener — knockdown + bonus damage; only viable when NOT already grappling and the attacker is on the offensive (opening rounds). Gate: `legs` + (fanged OR clawed) — a quadruped predator.

**Files:** like Task 3 with `Pounce`/`pounce` + `combat-messages/pounce.yaml`.

- [ ] **Step 1: Author `pounce.yaml`** (complete matrix, leaping/lunging/bearing-down prose), per the Task 3 Step 1 completeness rule.

- [ ] **Step 2: `ExecutePounce`** — copy the hamstring/kick action; use `BashDamagePercent` + `KnockdownChance: cfg.BashKnockdownChance` (it knocks down), `KnockdownToSupine: true` (driven backward), NO bleed. `RecordSpecialMove(...,"pounce",...)`.

- [ ] **Step 3: `CanUsePounce`/`ScorePounce`:**
```go
func CanUsePounce(char *characters.Character) bool {
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	if char.IsGrappling() {
		return false // can't leap from a clinch
	}
	if !char.HasBodyPart("legs") {
		return false
	}
	return speciesIsFanged(char) || speciesIsClawed(char)
}
func ScorePounce(mob *mobs.Mob, target *characters.Character) int {
	score := 50
	if !target.IsOnFloor() {
		score += 20 // pounce wants a standing target to knock down
	} else {
		score -= 40
	}
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 40 {
		score += 10
	}
	if score < 0 {
		score = 0
	}
	return score
}
```
Wire into `ChooseSpecialMove`.

- [ ] **Step 4–10: full parity** (`case "pounce"` in readiness — gate on `char.Aggro != nil && char.HasBodyPart("legs")`; player + mob handlers; register; `supported`; helpfile; keywords).

- [ ] **Step 11–12: tests + build + commit** (`feat(combat): add pounce beast opener (knockdown) + pounce.yaml + parity`).

---

## Task 5: `gore` — horned charge (dmg + knockback); `gore.yaml` exists

Mechanic: a charging horn strike — damage + knockdown (knockback). `gore.yaml` already exists. Gate: `speciesIsHorned` (which now requires the `horns` body part, enforced by Task 1 validation). Also add `horns` to the horned species' `body_parts` (data).

**Files:** like prior, with `Gore`/`gore`; Data: add `horns` to species whose `natural_attack: gore`.

- [ ] **Step 1: Data — add `horns` body part.** Find every species file with `natural_attack: gore` (`grep -rl "natural_attack: gore" _datafiles/world/dogmud/species/`) and add `"horns"` to its `body_parts` list. (Task 1's validation will panic at boot if any horned species lacks it — this step prevents that.)

- [ ] **Step 2: `ExecuteGore`** — copy action; `KickDamagePercent` damage + `KnockdownChance` (charge knockback) + `KnockdownToSupine: true`; no bleed; `RecordSpecialMove(...,"gore",...)`.

- [ ] **Step 3: `CanUseGore`/`ScoreGore`** — `return speciesIsHorned(char)` (cooldown first); `ScoreGore` base 50, +20 standing target, −40 if already on floor.

- [ ] **Step 4–10: full parity** (readiness `case "gore"`; handlers; register; `supported`; helpfile `gore.template`; keywords). Note: `gore.yaml` already complete — no new message file.

- [ ] **Step 11–12: tests + build + commit** (`feat(combat): add gore beast move (horned charge+knockback) + horns body part + parity`). Include the species YAML edits.

---

## Task 6: `drain` — vampire lifesteal (bleed target + heal self) + NEW `drain.yaml`

Mechanic (user spec): on a successful `drain`, apply a bleeding debuff to the target AND heal the attacker proportional to the damage dealt (lifesteal). Gate: `speciesHasLifeDrain` (the new flag), NOT `natural_attack`. The vampire keeps weapons + `claws` basic attack. Full parity (player handler exists but is lifedrain-gated, so normal players can't use it — consistent with the user's full-parity choice).

**Files:** like prior, with `Drain`/`drain` + `combat-messages/drain.yaml`; Data: `lifedrain: true` on species 34, `drain` added to vampire mob `combatcommands`.

- [ ] **Step 1: Author `drain.yaml`** (complete matrix — draining, leeching, sapping vitality prose), per the completeness rule.

- [ ] **Step 2: Config knob.** Add `DrainHealRatio float64` to the Balance config (`internal/configs/config.balance.go`) with default `0.75` (heal = 75% of damage dealt), following how an existing float knob like `SalvageMinChance` is declared + defaulted. Document it as "fraction of `drain` damage the attacker heals."

- [ ] **Step 3: Failing test — `ExecuteDrain` heals attacker + bleeds target on hit.**
Seed a `LifeDrain` attacker with reduced current HP; after a forced hit, assert (a) target has `ConditionBleeding`, (b) attacker's `Health` increased (clamped to max). Follow the existing action-test harness; if a forced hit isn't possible, assert the structural contract + that the heal helper computes `floor(damage * DrainHealRatio)` via a small pure helper you unit-test directly.

- [ ] **Step 4: `ExecuteDrain`** (`internal/actions/combat_drain.go`) — copy the hamstring action; `ExecuteSkillMove` with Willpower or Strength attack stat (use `char.Stats.Strength.ValueAdj` for consistency with the physical pipeline; rhetoric/charisma is for taunts), `TripDamagePercent` damage. On hit:
```go
	// Bleed the victim.
	target.Char.AddCondition(characters.ConditionBleeding, 4, float64(max(2, char.Stats.Strength.ValueAdj/12)), "drain")
	// Lifesteal: heal the attacker for a fraction of damage dealt.
	healAmt := int(float64(result.Damage) * configs.GetBalanceConfig().DrainHealRatio)
	if healAmt < 1 {
		healAmt = 1
	}
	healed := char.Heal(healAmt)
```
Return `healed` in a `DrainResult.Healed int` field for the caller's messaging. `RecordSpecialMove(...,"drain",...)`. (Define a local `max` if the package lacks one, or inline the comparison.)

- [ ] **Step 5: `CanUseDrain`/`ScoreDrain`:**
```go
func CanUseDrain(char *characters.Character) bool {
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	return speciesHasLifeDrain(char)
}
func ScoreDrain(mob *mobs.Mob, target *characters.Character) int {
	score := 50
	// Want it more when the attacker is hurt (lifesteal matters most then).
	hpPct := float64(mob.Character.Health) * 100.0 / float64(mob.Character.HealthMax.Value)
	if hpPct < 60 {
		score += 25
	}
	if score < 0 {
		score = 0
	}
	return score
}
```
Wire into `ChooseSpecialMove`.

- [ ] **Step 6–11: full parity** — readiness `case "drain"` (gate `char.Aggro != nil`; identity gate is in `CanUseDrain`), player + mob handlers (player handler describes the lifesteal: "you feel <target>'s vitality flow into you" via `combat.GetHealDescription(healed, maxHP)` — NO raw numbers), register both maps, add `"drain"` to `supported`, helpfile `drain.template`, keywords entry.

- [ ] **Step 12: Data.** Add `lifedrain: true` to `_datafiles/world/dogmud/species/34-vampire.yaml`. In `_datafiles/world/dogmud/mobs/summons/304-vampire.yaml`, add `drain` to `combatcommands` (it currently has only `''` after Phase 2 removed `bite`):
```yaml
combatcommands:
  - drain
  - ''
```

- [ ] **Step 13: tests + build + commit** (`feat(combat): add vampire drain move (lifesteal: bleed+heal) gated on species LifeDrain + parity`). Include species/mob YAML + config + drain.yaml.

---

## Task 7: `throttle` — fanged choke (cast-interrupt + stamina/health DoT) + NEW `throttle.yaml`

Mechanic (per user, 2026-06-09 — simplified from the spec's "silence status"): the fanged finisher — clamp the throat. On a hit it (a) deals immediate damage, (b) applies a **stamina + health damage-over-time** effect, and (c) when the victim is mid-cast, has a **fairly high chance to interrupt their spellcast** using the engine's EXISTING cast-interruption mechanism (the `activity.TriggerCastCancel` transition + conviction refund — the same path `cancel.go` and the damage-driven `checkConcentrationBreak` use). **No new "silenced" status, no new buff flag, no edits to the cast/shout commands.** Gate: `speciesIsFanged`. Best set up after a knockdown (pounce).

**Files:** Create `internal/actions/combat_throttle.go`, `internal/actions/cast_interrupt.go`, `internal/usercommands/throttle.go`, `internal/mobcommands/throttle.go`, `_datafiles/world/dogmud/combat-messages/throttle.yaml`, `_datafiles/world/dogmud/buffs/<id>-throttled.yaml`, `templates/help/throttle.template`; Modify `internal/combat/ai.go`, `internal/actions/command_readiness.go`, both command maps, `command_parity_test.go`, `internal/configs/config.balance.go`, `keywords.yaml`.

- [ ] **Step 1: Shared cast-interrupt helper `internal/actions/cast_interrupt.go`.** The cast-cancel-with-refund pattern (`refund := unspent/2; char.Conviction += refund; clamp; Activity.TransitionToFree({Trigger: activity.TriggerCastCancel, Actor})`) is currently duplicated in `usercommands/cancel.go`, `mobcommands/cancel.go`, `behaviortree/actions_combat.go`, and `usercommands/skill.cast.go`. Add ONE reusable helper (do NOT refactor the existing 4 sites now — out of scope; just add the shared function for throttle to call):
```go
// InterruptTargetCast cancels target's in-progress spellcast using the engine's
// standard cast-cancel path (conviction refund + activity TriggerCastCancel).
// Returns true if the target was casting and the cast was interrupted.
func InterruptTargetCast(target *characters.Character, byActor state.ActorRef) bool { ... }
```
Read `usercommands/cancel.go` (the refund math) + `behaviortree/actions_combat.go:230-250` (the transition) and reproduce the exact refund + transition. TDD: a casting character → `InterruptTargetCast` returns true and leaves `IsCasting()` false; a non-casting character → returns false, no-op.

- [ ] **Step 2: Stamina-DoT "Throttled" buff data file.** Run `python tools/id_inventory.py --type buffs` for the next free id; create `_datafiles/world/dogmud/buffs/<id>-throttled.yaml` following the tick-buff schema (see `internal/buffs/buffspec.go` `tick_pool`/`tick_percent`/`tick_min` + `nausea.yaml` for shape):
```yaml
buffid: <id>
name: Throttled
description: A crushing grip on your throat — you can barely breathe.
triggerrate: 1 round
triggercount: 3
tick_pool: stamina
tick_percent: -0.06   # drains ~6% max stamina per round (negative = damage)
tick_min: 2
```
(Health-over-time is handled by `ConditionBleeding` in Step 3, the established health-DoT used by hamstring/maul/rake — no second buff needed. NO `flags:` — this is a pure DoT, not a silence.)

- [ ] **Step 3: Config knob.** Add `ThrottleInterruptChance float64` to `internal/configs/config.balance.go`, default `0.75` ("chance throttle interrupts a casting victim"), following the existing float-knob declaration+default pattern.

- [ ] **Step 4: Failing test — `ExecuteThrottle` bleeds + applies the Throttled buff + interrupts a casting target.** Seed a fanged attacker + a target; on a forced hit assert the target gains `ConditionBleeding` and the Throttled buff; with the target mid-cast assert (deterministically, by setting `ThrottleInterruptChance: 1.0` via the config-override test helper) that the target's cast was interrupted (`IsCasting()` false). Follow the existing action-test harness (`combat_hamstring_test.go`).

- [ ] **Step 5: `ExecuteThrottle`** (`internal/actions/combat_throttle.go`) — copy `combat_hamstring.go`, rename, then on hit:
```go
	// Immediate bite damage already applied by ExecuteSkillMove.
	// Health-over-time: bleed (reuse the established condition).
	target.Char.AddCondition(characters.ConditionBleeding, 3, float64(max(2, char.Stats.Strength.ValueAdj/10)), "throttle")
	// Stamina-over-time: the Throttled DoT buff.
	target.Char.AddBuff(throttledBuffId, false) // confirm AddBuff signature/arg via a real call site
	// Cast interrupt: fairly high chance, using the engine's existing mechanism.
	interrupted := false
	if target.Char.IsCasting() && dice.RollStat(... ) // OR a simple util.Rand(100) < int(cfg.ThrottleInterruptChance*100)
	{
		interrupted = InterruptTargetCast(target.Char, actorRef(actor))
	}
```
Confirm the `AddBuff` signature against an existing caller (grep `AddBuff(` in actions/hooks). Use the project's standard roll (`util.Rand(100) < int(cfg.ThrottleInterruptChance*100)`) for the interrupt chance — this is a flat config probability, not a stat-opposed roll, matching the user's "fairly high chance" intent. Return `ThrottleResult{Executed, MoveResult, InterruptedCast bool, BleedDmg int}`. `RecordSpecialMove(...,"throttle",...)`.

- [ ] **Step 6: `CanUseThrottle`/`ScoreThrottle`** in `ai.go` — `return speciesIsFanged(char)` (cooldown first). `ScoreThrottle` base 50, +30 if `target.IsCasting()` (the engine already values interrupting casters — see `condTargetIsCasting`), +20 if `target.IsOnFloor()` (set up by pounce). Wire into `ChooseSpecialMove`.

- [ ] **Step 7: Author `throttle.yaml`** (complete matrix — clamping the windpipe, crushing the throat, the victim gasping/unable to make a sound; per the completeness rule). NOTE the messaging may *describe* the victim being unable to cry out, but there is no mechanical silence — only the DoT + cast-interrupt.

- [ ] **Step 8–12: full parity** — readiness `case "throttle"` (gate `char.Aggro != nil`; identity gate in `CanUseThrottle`), player + mob handlers (messaging conveys the choke + the DoT + "their spell collapses as they fight for air" when `InterruptedCast`), register both maps, add `"throttle"` to `supported`, helpfile `throttle.template` (describe: damage + lingering choke that saps stamina/health + interrupts spells), keywords entry.

- [ ] **Step 13: tests + build + commit** (`feat(combat): add throttle fanged choke (cast-interrupt + stamina/health DoT) + throttle.yaml + parity`). Include the throttled buff yaml + config knob.

---

## Task 8: AI profiles — beast profiles + weight the new moves + assign to mobs

**Files:** Modify `internal/combat/ai.go` (`aiProfiles`); Data: beast mob YAML `aiprofile`/profile assignment.

- [ ] **Step 1: Add beast profiles to `aiProfiles`** in `ai.go`:
```go
	"predator": { // fanged hunters: wolves, etc.
		"pounce":    40,
		"maul":      35,
		"throttle":  30,
		"hamstring": 25,
		"kick":      10,
		"trip":      10,
	},
	"ambush_predator": { // felines: open with pounce, finish with rake
		"pounce": 45,
		"rake":   40,
		"hamstring": 20,
		"trip":   10,
	},
	"brute": { // bears/boars: maul/gore, less finesse
		"maul":    40,
		"gore":    40,
		"pounce":  25,
		"bash":    10,
	},
```
Also add `rake`/`maul`/`pounce`/`gore`/`drain`/`throttle` weights into the generic `default`/`aggressive` profiles where sensible (so a beast on a generic profile still uses its anatomy-permitted moves). Body-part/identity gating filters automatically, so listing a move a given mob can't do is harmless.

- [ ] **Step 2: Failing test — profile selection.** In `ai_test.go`, seed a fanged predator mob on the `predator` profile and assert `ChooseSpecialMove` returns one of its permitted beast moves (and NEVER `grapple`); seed a clawed mob and assert it can pick `rake` but never `maul` (fanged-only). Use the existing `ChooseSpecialMove` test harness (set `SpecialMoveChance` high to force a pick).

- [ ] **Step 3: Implement** (the profile additions from Step 1) and make the test pass.

- [ ] **Step 4: Data — assign beast profiles.** Assign `predator`/`ambush_predator`/`brute` to the beast mobs where the default profile isn't apt (wolves→predator, felines→ambush_predator, bears/boars→brute). Find the mob `aiprofile` field convention (`grep -rn "aiprofile\|AIProfile" _datafiles/world/dogmud/mobs/ internal/mobs/`) and set it on representative beast mobs. Scope: a curated pass over the obvious predators, not all ~100 mobs (note in the commit which zones/mobs were assigned and that the rest fall back to default + anatomy gating).

- [ ] **Step 5: tests + build + commit** (`feat(combat): beast AI profiles (predator/ambush_predator/brute) + weight beast moves + assign to mobs`).

---

## Task 9: context.md updates

**Files:** `internal/combat/context.md`, `internal/actions/context.md`, `internal/mobcommands/context.md`, `internal/usercommands/context.md`.

- [ ] **Step 1:** In `internal/combat/context.md`, extend the Phase-2 "Anatomy-Gated Special Moves" section with a Phase-3 "Beast Moveset" subsection: list rake/maul/pounce/gore/drain/throttle, their gates (clawed / fanged / quadruped-predator / horned / LifeDrain / fanged), the beast-identity predicates as the single source of truth, the new AI profiles, and that throttle adds the `Silenced` flag (blocks shout/cast).

- [ ] **Step 2:** `internal/actions/context.md` — note the new `Execute*` actions + their effects (bleed/knockdown/lifesteal/silence-hold). `internal/mobcommands/context.md` + `internal/usercommands/context.md` — note the new both-usable commands and their gating.

- [ ] **Step 3: Commit** (`docs: document Phase-3 beast moveset across combat/actions/commands context.md`).

---

## Task 10: Validation + full verification + boot + smoke

- [ ] **Step 1: Full build + targeted tests.**
Run: `go build ./... && go test ./internal/combat/ ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/species/ ./internal/items/ ./internal/buffs/` → all green (incl. parity + message-load tests).

- [ ] **Step 2: Full suite.** `go test ./...` → green (watch for cross-package breaks like the Phase-2 behaviortree case; fix any test that assumed a no-identity mob could use a now-gated move).

- [ ] **Step 3: Boot test (message files + species validation are load-time panics).**
Run: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`, then build + boot, watch for `Server Ready` with NO panic. Confirm `itemspec.LoadDataFiles()` reports the new `attackMessageCount` (maul/pounce/throttle/drain loaded), `species.LoadDataFiles()` clean (gore→horns validation passed), and CommandParity logs no unexpected gaps for the new commands. Kill server; clear ports.

- [ ] **Step 4: In-game smoke** (reuse/extend `tools/playtest/goals/phase2-beast-combat.yaml`): fight a fanged predator and confirm it pounces / mauls / throttles / hamstrings and bites (basic) but never grapples; a clawed beast rakes; a horned beast gores; the vampire drains (and heals); a humanoid still grapples/bashes; `throttle` blocks the victim's shout/cast. Confirm new help topics resolve (`help rake`, etc.). Write a report to `tools/playtest/reports/`.

- [ ] **Step 5: Merge to local `master`** (no push — for the bundle). Phase 4 (data audit: per-mob overrides, broader profile assignment, rarity/dropchance) is a separate plan.

---

## Self-review notes (controller)

- **Spec coverage (Layer 2b moveset):** rake (T2), maul (T3), pounce (T4), gore (T5), drain (T6 — the user's vampire replacement), throttle (T7); AI profiles/selection (T8); context.md (T9); validation + verify (T10). Hamstring + bite-retirement already shipped in Phase 2.
- **User decisions honored:** full player↔mob parity per move (handler + helpfile + keywords + `supported` entry in every move task); drain gated on a new species `LifeDrain` flag (T1/T6), vampire stays a weapon-using humanoid (no `natural_attack` change), drain = bleed + lifesteal heal.
- **Cross-cutting wiring (spec §A–D), enumerated per task:** §A message-file completeness — every NEW file (maul/pounce/throttle/drain) authors the full intensity×tier×audience matrix or the loader panics (T3/4/6/7 step 1; T10 boot is the guard); rake reuses `claws.yaml`, gore uses existing `gore.yaml`. §B full parity — each move touches both command maps + both handlers + `CommandIsReady` + `supported` + helpfile + keywords. §C context.md (T9). §D helpfiles + keywords (each move task).
- **Single source of truth:** beast-identity predicates (`speciesIsFanged/Clawed/Horned`, `speciesHasLifeDrain`) added once in T1, reused by every `CanUse*` — mirrors the spec's "keep gating predicates in one place."
- **Conventions:** all moves route damage through `ExecuteSkillMove` (unified pipeline), use config-knobbed percentages/multipliers (no flat values), and emit only descriptive player text (`GetDamageDescription`/`GetHealDescription`, no raw numbers).
- **`throttle` (T7) simplified per user (2026-06-09):** no silence status / new buff flag / cast-shout system edits. It reuses the engine's EXISTING cast-interrupt (`activity.TriggerCastCancel` + conviction refund, via a new shared `actions.InterruptTargetCast` helper) with a flat high chance (`ThrottleInterruptChance` 0.75), plus `ConditionBleeding` (health DoT, reused) + a small stamina-DoT "Throttled" tick buff. Much lower risk than the original silence-flag design.
- **Message-file authoring is large** (~300–500 lines each for maul/pounce/throttle/drain). Each is its own step copying `gore.yaml`'s structure; completeness is load-guaranteed by the boot test.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-09-nonhuman-attacks-phase3-beast-moveset.md`. Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task (one move at a time), spec + quality review between tasks, fast iteration. Sequence content-creating tasks (message files, helpfiles) to avoid ID collisions; code tasks can pipeline.
2. **Inline Execution** — execute tasks in this session with checkpoints.

Which approach?
