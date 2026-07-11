# Non-human Attacks — Phase 4: Beast Data Audit & Polish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the beast moveset: gate beast natural-weapon moves to true beasts (the `hands` rule), add `skirmisher`/`serpent` AI profiles, assign profiles across the remaining beast mobs, audit body-part/identity data, clean up inert combatcommands, and retag deer→gore + wraith/spectre→lifedrain.

**Architecture:** One focused code refinement (beast natural-weapon moves require `!HasBodyPart("hands")`, applied at the three Phase-3 sync points), two new AI profiles, and mostly DATA edits (species + mob YAML) validated by the existing load checks + a boot test. `drain` stays `LifeDrain`-gated (exempt from `!hands`) so armed undead drain.

**Tech Stack:** Go (`internal/combat`, `internal/actions`); YAML data (species, mobs, profiles already in `ai.go`).

**Spec:** `docs/superpowers/specs/2026-06-09-phase4-beast-data-audit-design.md`. Phases 1-3 are on local master.

**Verified facts (2026-06-09):**
- Beast natural-weapon moves + their gate sites (all on master): `rake`/`maul`/`pounce`/`gore`/`throttle` (Phase 3) + `hamstring` (Phase 2). Each gates identity at 3 sync points: `combat.CanUse<Move>` (`internal/combat/ai.go`), `actions.Execute<Move>` action entry (`internal/actions/combat_<move>.go`, returns `…Result{Not<Identity>: true}`), `actions.CommandIsReady` (`internal/actions/command_readiness.go`), with drift rows in `command_readiness_drift_test.go`.
- `pounce`'s identity is the helper `combat.SpeciesIsQuadrupedPredator(char) = char.HasBodyPart("legs") && (SpeciesIsFanged(char) || SpeciesIsClawed(char))`.
- `drain` gates on `combat.SpeciesHasLifeDrain(char)` (the `LifeDrain` species flag), NOT anatomy — it must stay exempt from the `!hands` rule.
- `Character.HasBodyPart(part string) bool` exists.
- `body_parts` today: tool-humanoids have `hands` (goblin `[arms,hands,legs,…]`, skeleton `[arms,hands,legs,…]`, vampire `[arms,hands,legs,…]`); bear `[arms,legs,…]` has NO hands; other beasts have neither arms nor hands. wraith(32)/spectre(33) have `body_parts: []` + `grapple_immune: true` + empty `natural_attack`.
- `aiProfiles` map in `internal/combat/ai.go` (keys incl. caster/default/aggressive/predator/ambush_predator/brute). The `caster` profile prefers spells (`ChooseCastAction` handles spell selection separately) and carries only low special-move weights.
- Mob profile via `aiprofile:` YAML key. Beast species→mob counts: canine(2)=10, boar(6)=4, serpent(8)=4, raptor(9)=6, rodent(10)=15, feline(11)=1, reptile(21)=6, bat(22)=2, mustelid(24)=2, goblin(5)=6, deer(7)=?, wraith(32)=1, spectre(33)=1.
- gore→horns load validation already panics if a `gore` species lacks `horns` (so deer→gore MUST add horns same-change).

---

## File Structure
- `internal/combat/ai.go` — add `!hands` to the 6 beast-move gates (+ `SpeciesIsQuadrupedPredator`); add `skirmisher`/`serpent` profiles; add low `drain` weight to `caster`.
- `internal/actions/combat_{rake,maul,pounce,gore,throttle,hamstring}.go` — add the `!hands` action-entry gate.
- `internal/actions/command_readiness.go` — add `!hands` to the 6 readiness cases.
- `internal/actions/command_readiness_drift_test.go`, `internal/combat/ai_test.go` — `*_hashands` drift rows + CanUse tests.
- Species YAML: `7-deer.yaml` (gore+horns), `32-wraith.yaml`/`33-spectre.yaml` (lifedrain), plus any audit fixes.
- Mob YAML: `aiprofile:` assignments across beast mobs; inert-combatcommand removals.
- `internal/combat/context.md`, `internal/actions/context.md` — docs.

---

## Task 1: The `hands` rule — beast natural-weapon moves require `!hands`

Apply `!char.HasBodyPart("hands")` to ALL SIX beast natural-weapon moves at all three sync points. `drain` is NOT touched.

**Files:** `internal/combat/ai.go`, `internal/actions/combat_{rake,maul,pounce,gore,throttle,hamstring}.go`, `internal/actions/command_readiness.go`, tests.

- [ ] **Step 1: Failing tests (ai_test.go).** Add a test that, for each of `CanUseRake/Maul/Pounce/Gore/Throttle/Hamstring`, a species WITH `hands` returns false even when the identity matches, and a no-hands beast returns true. Example:
```go
func TestBeastMoves_RequireNoHands(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		// clawed + fanged + horned + legs, but a tool-using humanoid (has hands)
		9001: {SpeciesId: 9001, Name: "goblinoid", BodyParts: []string{"arms", "hands", "legs", "mouth", "horns"}, NaturalAttack: items.Claws},
		9002: {SpeciesId: 9002, Name: "fanged-humanoid", BodyParts: []string{"arms", "hands", "legs", "mouth"}, NaturalAttack: items.Bite},
		9003: {SpeciesId: 9003, Name: "bear", BodyParts: []string{"arms", "legs", "mouth"}, NaturalAttack: items.Claws}, // arms, NO hands
		9004: {SpeciesId: 9004, Name: "horned-beast", BodyParts: []string{"legs", "mouth", "horns"}, NaturalAttack: items.Gore},
	})
	defer cleanup()
	gob := &characters.Character{SpeciesId: 9001}
	fang := &characters.Character{SpeciesId: 9002}
	bear := &characters.Character{SpeciesId: 9003}
	horned := &characters.Character{SpeciesId: 9004}

	// hands-bearers: blocked from beast natural-weapon moves
	if CanUseRake(gob) || CanUsePounce(gob) { t.Error("hands goblinoid must not rake/pounce") }
	if CanUseMaul(fang) || CanUseThrottle(fang) || CanUseHamstring(fang) { t.Error("hands fanged-humanoid must not maul/throttle/hamstring") }
	if CanUseGore(gob) { t.Error("hands goblinoid must not gore") }
	// no-hands beasts: still eligible
	if !CanUseMaul(bear) || !CanUseRake(bear) { t.Error("handless bear should maul/rake") }
	if !CanUseGore(horned) { t.Error("handless horned beast should gore") }
}
```
Run → FAIL (hands-bearers currently pass).

- [ ] **Step 2: ai.go gates.** In each of `CanUseRake`, `CanUseMaul`, `CanUseGore`, `CanUseThrottle`, `CanUseHamstring`, add after the cooldown check:
```go
	if char.HasBodyPart("hands") {
		return false // beast natural-weapon moves are for true beasts, not tool-users
	}
```
For `pounce`, update the shared predicate instead:
```go
func SpeciesIsQuadrupedPredator(char *characters.Character) bool {
	return !char.HasBodyPart("hands") && char.HasBodyPart("legs") && (SpeciesIsFanged(char) || SpeciesIsClawed(char))
}
```
Run the Step-1 test → PASS.

- [ ] **Step 3: Action-entry gates.** In each of `ExecuteRake/Maul/Pounce/Gore/Throttle/Hamstring` (`internal/actions/combat_<move>.go`), at the existing identity-gate site (where it returns `…Result{Not<Identity>: true}`), extend the condition to also fail on hands. E.g. rake:
```go
	if char.HasBodyPart("hands") || !combat.SpeciesIsClawed(char) {
		return RakeResult{NotClawed: true}
	}
```
(Reuse the existing `Not<Identity>` result — the hands+identity-match case, e.g. a clawed vampire, is unreachable for real players and silent for mobs; no new result field needed. For pounce, the `SpeciesIsQuadrupedPredator` change already covers it — confirm `ExecutePounce` calls that predicate and needs no extra edit. `ExecuteHamstring` lives at `internal/actions/combat_hamstring.go` — add the hands check to its fanged/clawed gate.) Quote each before/after.

- [ ] **Step 4: Readiness gates.** In `command_readiness.go`, add `&& !char.HasBodyPart("hands")` to each of the `rake/maul/pounce/gore/throttle/hamstring` cases. (pounce already calls `SpeciesIsQuadrupedPredator`, now hands-aware — no extra edit, but confirm.)

- [ ] **Step 5: Drift rows.** For each of the six moves, add a `<move>_hashands` row to `command_readiness_drift_test.go`: seed a hands-bearing species whose identity matches the move (so only the hands rule blocks it), aggro set, no cooldown → expect NOT ready. Confirm `runExecuteAndReadFlag` maps the resulting `Not<Identity>` flag.

- [ ] **Step 6: Verify.** `go test ./internal/combat/ ./internal/actions/` green; confirm `drain` is unaffected (no hands check) — add/keep a test that a hands-bearing `LifeDrain` species (vampire) STILL passes `CanUseDrain`. `go build ./...` clean.

- [ ] **Step 7: Commit.**
```bash
git add internal/combat/ai.go internal/combat/ai_test.go internal/actions/combat_rake.go internal/actions/combat_maul.go internal/actions/combat_pounce.go internal/actions/combat_gore.go internal/actions/combat_throttle.go internal/actions/combat_hamstring.go internal/actions/command_readiness.go internal/actions/command_readiness_drift_test.go
git commit -m "feat(combat): beast natural-weapon moves require no hands (true-beast gate); drain stays LifeDrain-gated"
```
(End every commit with `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.)

---

## Task 2: New AI profiles — `skirmisher` + `serpent`

**Files:** `internal/combat/ai.go` (`aiProfiles`), `internal/combat/ai_test.go`.

- [ ] **Step 1: Failing selection test.** Seed a fanged legged rodent on `skirmisher` and assert (high `SpecialMoveChance`, many iterations) it favours `hamstring`/`trip` and rarely/never returns `maul` as the dominant pick; seed a fanged LEGLESS serpent on `serpent` and assert it can return `maul`/`throttle` but NEVER `pounce`/`hamstring` (anatomy gates those — legless). Read the existing `TestChooseSpecialMove_*` harness.

- [ ] **Step 2: Add the profiles** to `aiProfiles`:
```go
	"skirmisher": { // small fanged vermin (rats, insects): harry, don't maul
		"hamstring": 35,
		"trip":      30,
		"kick":      20,
		"maul":      10,
	},
	"serpent": { // legless fanged (snakes, worms): strike + constrict
		"maul":     35,
		"throttle": 35,
		// no pounce/hamstring — legless; anatomy gates them, don't weight them
	},
```
Run the test → PASS.

- [ ] **Step 3: Verify + commit.** `go test ./internal/combat/` green; `go build ./...` clean.
```bash
git add internal/combat/ai.go internal/combat/ai_test.go
git commit -m "feat(combat): add skirmisher + serpent beast AI profiles"
```

---

## Task 3: Correctness audit — body_parts / natural_attack sweep

Mostly confirmation; fix any mismatch. Produce an audit table.

**Files:** species YAML under `_datafiles/world/dogmud/species/` (fixes only); a short audit note appended to `internal/combat/context.md` or the commit body.

- [ ] **Step 1: Build the audit table.** For every beast species (any with a non-empty `natural_attack`, plus humanoid-monsters goblin/skeleton/vampire), record `natural_attack`, `body_parts`, and check the invariants:
  - `hands` present ONLY on tool-using humanoids (goblin/skeleton/vampire/human/dwarf/elf/etc.) and ABSENT on all beasts (bear/feline/canine/boar/raptor/rodent/reptile/serpent/bat/mustelid/arachnid/insectoid/worm/…). This is what makes Task 1 correct.
  - fanged/clawed species have `mouth`; `gore` species have `horns`; species used by legged moves have `legs`.
  - No non-bear quadruped beast has stray `arms`.
  - `natural_attack` thematically apt.
  Use: `for f in _datafiles/world/dogmud/species/*.yaml; do …; done` to dump name/na/body_parts. Cross-check against the invariants.

- [ ] **Step 2: Fix mismatches.** Edit only the species that violate an invariant (quote each fix). Expected to be FEW (the brainstorming survey found the data largely correct). If a beast wrongly has `hands` or `arms`, remove it; if a fanged species lacks `mouth`, add it. Do NOT retag deer/wraith here (Tasks 4/5).

- [ ] **Step 3: Verify + commit.** `go build ./...` clean; (boot validation is Task 9). Commit the audit table (in the commit body or context.md) + any fixes:
```bash
git add <changed species yaml> internal/combat/context.md
git commit -m "chore(species): beast body_parts/natural_attack correctness audit (+fixes)"
```
If NO fixes were needed, still commit the audit table to context.md so the verification is recorded.

---

## Task 4: Retag deer → gore (antlers)

**Files:** `_datafiles/world/dogmud/species/7-deer.yaml`; deer mob YAML (assign `brute`).

- [ ] **Step 1:** In `7-deer.yaml`: change `natural_attack: slam` → `natural_attack: gore`; add `horns` to `body_parts` (REQUIRED — the gore→horns load validation panics otherwise): `body_parts: [legs, eyes, mouth, skin, tail, horns]`. (Deer's `unarmedname` may be "hooves" — update to "antlers" for apt `{itemname}` in gore messages.)
- [ ] **Step 2:** Find deer mobs (`grep -rln "speciesid: 7$" _datafiles/world/dogmud/mobs/`) and set `aiprofile: brute` on each (or leave default if a deer is a non-combatant — check `non_combatant`).
- [ ] **Step 3: Verify + commit.** `go build ./...` clean (boot is Task 9). 
```bash
git add _datafiles/world/dogmud/species/7-deer.yaml <deer mob yaml>
git commit -m "feat(content): deer gore with antlers (slam->gore + horns) + brute profile"
```

---

## Task 5: Retag wraith + spectre → lifedrain (caster-preferred, occasional drain)

Wraith/spectre are SPELLCASTERS: they should prefer casting and only OCCASIONALLY drain (when buffs are already up). Implement as: lifedrain flag + `caster` profile + a LOW `drain` weight on the caster profile (so spells dominate and drain surfaces rarely — naturally more once buff-spells are already cast).

**Files:** `_datafiles/world/dogmud/species/32-wraith.yaml`, `33-spectre.yaml`; `internal/combat/ai.go` (`caster` profile); wraith/spectre mob YAML.

- [ ] **Step 1:** Add `lifedrain: true` to `32-wraith.yaml` and `33-spectre.yaml`. (They keep `body_parts: []` + `grapple_immune` — so they have no humanoid/beast natural-weapon moves; `drain` is their only special, gated on `LifeDrain`.)

- [ ] **Step 2: caster profile gets a low drain weight.** In `aiProfiles` `"caster"`, add `"drain": 15` (kept low so `ChooseCastAction`'s spell preference dominates; drain is the occasional mix-in). Confirm via reading `ChooseSpecialMove`/`GetAIProfile` + the cast-vs-special decision path that a `caster`-profile mob attempts spells first and only falls to special moves at low frequency — quote the ordering. If the engine does NOT prefer casting over special moves for casters (i.e. a low weight is NOT enough to make drain "occasional"), STOP and report the ordering so the controller can decide whether to add a buffs-up gate to `ScoreDrain`.

- [ ] **Step 3:** Set `aiprofile: caster` on the wraith + spectre mobs (`grep -rln "speciesid: 32$\|speciesid: 33$" _datafiles/world/dogmud/mobs/`). Do NOT force `drain` into their `combatcommands` (that would override the prefer-spells behavior) — let the AI weighting surface it occasionally.

- [ ] **Step 4: Test.** `CanUseDrain` true for a `LifeDrain` species; a `caster`-profile lifedrain mob's `ChooseSpecialMove` can return `drain` (when it picks a special at all). `go test ./internal/combat/` green; `go build ./...` clean.

- [ ] **Step 5: Commit.**
```bash
git add internal/combat/ai.go internal/combat/ai_test.go _datafiles/world/dogmud/species/32-wraith.yaml _datafiles/world/dogmud/species/33-spectre.yaml <wraith/spectre mob yaml>
git commit -m "feat(content): wraith/spectre lifedrain — caster-preferred with occasional drain"
```

---

## Task 6: Broad beast-mob profile assignment

Assign specialized profiles across the beast mobs of each species per the spec's mapping. Mobs already assigned in Phase 3 (15) keep theirs; this is the remaining ~60.

**Files:** mob YAML under `_datafiles/world/dogmud/mobs/`.

- [ ] **Step 1: Mapping (from the spec).**
  - canine(2), reptile(21), mustelid(24) → `predator`
  - feline(11), bat(22), raptor(9), arachnid(17) → `ambush_predator`
  - boar(6), bear(3) → `brute` (deer handled in Task 4)
  - rodent(10), insectoid(12) → `skirmisher`
  - serpent(8), worm(18) → `serpent`
  - slime(16)/elementals/fish(13)/fungal(15) and any limbless `slam` species → leave default (basic attacks only)
  - goblin(5)/skeleton(30) → leave (humanoid; `hands` rule already excludes beast moves)

- [ ] **Step 2:** For each species in the mapping, `grep -rln "speciesid: <sid>$" _datafiles/world/dogmud/mobs/` and set/replace `aiprofile:` to the mapped value on each combatant mob (skip `non_combatant: true` mobs). Match the existing `aiprofile:` line format/indent. LIST every file changed + the profile set, grouped by species.

- [ ] **Step 3: Verify + commit.** `go build ./...` clean (boot is Task 9).
```bash
git add <all changed mob yaml>
git commit -m "feat(content): assign beast AI profiles across remaining beast mobs (predator/ambush_predator/brute/skirmisher/serpent)"
```

---

## Task 7: Inert combatcommand cleanup

**Files:** mob YAML under `_datafiles/world/dogmud/mobs/`.

- [ ] **Step 1: Sweep.** For each mob with a `combatcommands` (or `angrycommands`) list, cross-check each listed combat move against the mob's species anatomy:
  - `grapple`/`bash`/`submit` on a species without `arms` → dead.
  - `trip`/`kick` on a species without `legs` → dead.
  - a beast move (`rake`/`maul`/`pounce`/`gore`/`throttle`/`hamstring`) on a species with `hands` or the wrong identity → dead.
  Script it: dump each mob's species + its combatcommands, flag mismatches. Known: `sump_dweller` (aberration `[]`) lists `bash`.
- [ ] **Step 2: Remove** each dead combat-command entry (leave non-combat emotes/`''` intact; match YAML format). LIST every removal (mob + command).
- [ ] **Step 3: Verify + commit.** `go build ./...` clean.
```bash
git add <changed mob yaml>
git commit -m "chore(content): remove inert combatcommands (anatomy-forbidden moves no-op post-gating)"
```

---

## Task 8: context.md updates

**Files:** `internal/combat/context.md`, `internal/actions/context.md`.

- [ ] **Step 1:** In `internal/combat/context.md`'s beast-moveset section, add the Phase-4 refinement: the `hands` rule (beast natural-weapon moves require `!hands`; `drain` exempt via LifeDrain); the new `skirmisher`/`serpent` profiles; the species→profile mapping; the deer-gore + wraith/spectre-lifedrain retags. In `internal/actions/context.md`, note the action-entry `!hands` gate on the six beast moves.
- [ ] **Step 2: Commit.** `docs: document Phase-4 hands rule + new profiles + retags`.

---

## Task 9: Validation + boot + smoke + merge

- [ ] **Step 1: Full build + suite.** `go build ./... && go test ./...` → green (watch for cross-package breaks; fix any test that assumed a hands-bearing species could use a beast move).
- [ ] **Step 2: Boot test** (species + mob YAML edits are load-validated; deer-gore needs horns). `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`, build + boot, watch for `Server Ready`, no panic, `species.LoadDataFiles()` clean (deer gore→horns passes), no CommandParity surprises. Kill server; clear ports.
- [ ] **Step 3: In-game smoke** (extend `tools/playtest/goals/phase2-beast-combat.yaml`): a goblin grapples but does NOT pounce/rake; a bear mauls; a rat skirmishes (hamstring/trip, not maul-heavy); a snake throttles/mauls but never pounces; a wraith mostly casts and occasionally drains. Write a report to `tools/playtest/reports/`.
- [ ] **Step 4: Merge to local master** (no push). Phase 4 closes the non-human-attacks sub-project.

---

## Self-review notes (controller)

- **Spec coverage:** Workstream A → T1 (the `hands` rule, all 6 moves × 3 sync points, drain exempt); B → T2 (profiles) + T6 (assignment); C/D audit → T3; D cleanup → T7; E retags → T4 (deer) + T5 (wraith/spectre, caster-preferred per the user's note). Docs T8, verify T9.
- **User's wraith note honored:** T5 implements "prefer spells, occasional drain" via the `caster` profile + a low (15) drain weight + an explicit STOP-if-ordering-wrong investigation step (rather than forcing drain into combatcommands).
- **Key gotchas baked in:** `drain` must stay OUT of the `!hands` gate (T1 Step 6 pins it); deer→gore REQUIRES `horns` same-change (T4 Step 1); the `hands` rule is the symmetric counterpart of Phase-2's `arms` rule and bears (arms, no hands) intentionally stay maulers.
- **Mostly data:** only T1, T2, and the T5 caster-weight are code; the rest is YAML validated by the boot test. The audit (T3) is expected to be confirmation-heavy.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-09-phase4-beast-data-audit.md`. Two execution options:
1. **Subagent-Driven (recommended)** — fresh subagent per task, spec+quality review between tasks (as Phase 3). Sequence the data tasks (T3/T4/T5/T6/T7 all touch YAML) to avoid ID/file races; the code tasks (T1/T2) can go first.
2. **Inline Execution** with checkpoints.

Which approach?
