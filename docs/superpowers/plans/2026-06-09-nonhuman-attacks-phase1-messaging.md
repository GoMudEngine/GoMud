# Non-human Attacks — Phase 1: Natural-Attack Messaging — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make non-human creatures' unarmed BASIC attacks narrate with anatomy-appropriate verbs (a wolf bites/claws instead of "punches"), by mapping each species to a combat-message subtype.

**Architecture:** Add a `NaturalAttack` subtype field to the species spec; `combat.buildWeaponSetup` uses it (instead of the hardcoded `items.Unarmed`) when no weapon is equipped, routing basic-attack messages to the existing `bite`/`claws`/`slam`/`gore`/`sting` message files. A load-time validator keeps `natural_attack` values valid. Data: tag the non-human species.

**Tech Stack:** Go; existing `internal/species`, `internal/combat`, `internal/items` packages; YAML species + combat-message data files.

**Spec:** `docs/superpowers/specs/2026-06-09-nonhuman-attacks-and-beast-moveset-design.md` (this plan implements **Layer 1** only).

**Phase sequence (each its own plan):** **Phase 1 — natural-attack messaging (THIS PLAN)** → Phase 2 — anatomy gating of human technique moves + wake `hamstring` + retire `bite` special → Phase 3 — new beast moves (`throttle`, `pounce`, `maul`, `rake`, `gore`) → Phase 4 — AI profiles + per-mob override + full data audit. Phase 1 is independently ship-able.

---

## File Structure

- `internal/species/species.go` — MODIFY: add `NaturalAttack` field (+ existing loader gets validated in Task 3).
- `internal/combat/combat_helpers.go` — MODIFY: `buildWeaponSetup` uses the species natural attack.
- `internal/species/species.go` or a small new validator — MODIFY/ADD: load-time check that `natural_attack` is a known subtype.
- `_datafiles/world/dogmud/species/*.yaml` — MODIFY (data): add `natural_attack:` to non-human species.
- Tests alongside each (`internal/species/species_test.go`, `internal/combat/combat_helpers_test.go`).

---

## Task 1: Add `NaturalAttack` field to the species spec

**Files:**
- Modify: `internal/species/species.go` (Species struct, ~line 32-53)
- Test: `internal/species/species_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/species/species_test.go` (create with `package species` + imports `testing`, `gopkg.in/yaml.v3`, and the items package if not present):

```go
func TestSpecies_NaturalAttackUnmarshal(t *testing.T) {
	var s Species
	if err := yaml.Unmarshal([]byte("speciesid: 99\nname: testcanine\nnatural_attack: bite\n"), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.NaturalAttack != items.Bite {
		t.Errorf("NaturalAttack = %q, want %q", s.NaturalAttack, items.Bite)
	}
}
```

(Confirm the project's YAML lib import path — `gopkg.in/yaml.v3` per the repo; match what other `_test.go` files in this package use.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/species/ -run TestSpecies_NaturalAttackUnmarshal -v`
Expected: compile error — `s.NaturalAttack` undefined.

- [ ] **Step 3: Add the field**

In `internal/species/species.go`, inside the `Species` struct (next to `UnarmedName`), add:

```go
	// NaturalAttack is the combat-message subtype an unarmed member of this
	// species uses for BASIC attacks (e.g. items.Bite, items.Claws). Empty =>
	// humanoid default (Unarmed -> generic). Must be a known items.ItemSubType
	// with a loaded combat-message file (validated at load).
	NaturalAttack items.ItemSubType `yaml:"natural_attack,omitempty"`
```

(`internal/items` is already imported in this file — `Damage items.Damage` exists.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/species/ -run TestSpecies_NaturalAttackUnmarshal -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/species/species.go internal/species/species_test.go
git commit -m "feat(species): add NaturalAttack subtype field"
```
End the commit message with:
`Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## Task 2: `buildWeaponSetup` uses the species natural attack for unarmed

**Files:**
- Modify: `internal/combat/combat_helpers.go` (`buildWeaponSetup`, ~line 245-256)
- Test: `internal/combat/combat_helpers_test.go`

- [ ] **Step 1: Write the failing test**

First confirm how `combat` tests seed species: search for an existing species seeder (e.g. `species.SeedSpeciesForTest`, or how other combat tests set `species` data). Run:
`grep -rn "SeedSpeciesForTest\|allSpecies\s*=\|species.GetSpecies" internal/species/ internal/combat/ | grep -i test`
Use the discovered seeding mechanism in the test below (the assertion is the point). Add to `internal/combat/combat_helpers_test.go`:

```go
func TestBuildWeaponSetup_UsesSpeciesNaturalAttack(t *testing.T) {
	// Seed a fanged species (id 9001) and a humanoid (id 9002).
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		9001: {SpeciesId: 9001, Name: "testwolf", UnarmedName: "jaws", NaturalAttack: items.Bite, DamageMultiplier: 1.0},
		9002: {SpeciesId: 9002, Name: "testhuman", UnarmedName: "fists", DamageMultiplier: 1.0},
	})
	defer cleanup()

	wolf := &characters.Character{SpeciesId: 9001}
	human := &characters.Character{SpeciesId: 9002}
	noWeapon := items.Item{}

	wsWolf := buildWeaponSetup(wolf, wolf, noWeapon, 0, 1)
	if wsWolf.weaponSubType != items.Bite {
		t.Errorf("unarmed wolf subtype = %q, want %q", wsWolf.weaponSubType, items.Bite)
	}

	wsHuman := buildWeaponSetup(human, human, noWeapon, 0, 1)
	if wsHuman.weaponSubType != items.Unarmed {
		t.Errorf("unarmed human subtype = %q, want %q", wsHuman.weaponSubType, items.Unarmed)
	}
}
```

If no `SeedSpeciesForTest` exists, ADD one to `internal/species` mirroring `mobs.SeedMobsForTest` (save/replace `allSpecies`, return restore func) as a tiny prerequisite step, then use it. (This is reusable test infra, not production behavior.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/combat/ -run TestBuildWeaponSetup_UsesSpeciesNaturalAttack -v`
Expected: FAIL — unarmed wolf subtype is `unarmed`, want `bite` (current code hardcodes `items.Unarmed`).

- [ ] **Step 3: Wire the species natural attack into the unarmed default**

In `internal/combat/combat_helpers.go`, in `buildWeaponSetup`, the `ws` struct is built with `weaponSubType: items.Unarmed`. Immediately AFTER the `ws := weaponSetup{...}` literal and BEFORE the `if weapon.ItemId > 0` block, add:

```go
	// Non-human basic attacks render through the species' natural-attack
	// subtype (bite/claws/slam/...) instead of generic. A real equipped
	// weapon overrides this below. Humanoids leave NaturalAttack empty and
	// stay on Unarmed -> generic.
	if raceInfo != nil && raceInfo.NaturalAttack != "" {
		ws.weaponSubType = raceInfo.NaturalAttack
	}
```

(`raceInfo` is already `species.GetSpecies(sourceChar.SpeciesId)` at the top of the function. The weapon branch sets `ws.weaponSubType = itemSpec.Subtype`, so an equipped weapon still wins. The natural-reach calc in the `else` branch reads `ws.weaponSubType`, which is now correctly the natural-attack subtype.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/combat/ -run TestBuildWeaponSetup_UsesSpeciesNaturalAttack -v`
Expected: PASS. Then `go test ./internal/combat/` (whole package) — expect green.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/combat_helpers.go internal/combat/combat_helpers_test.go internal/species/
git commit -m "feat(combat): unarmed basic attacks use species NaturalAttack subtype"
```
(End with the Co-Authored-By line.)

---

## Task 3: Load-time validation of `natural_attack`

**Files:**
- Modify: `internal/species/species.go` (`LoadDataFiles`, ~line 184-197)
- Test: `internal/species/species_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/species/species_test.go`:

```go
func TestValidateNaturalAttack(t *testing.T) {
	// Known subtype with a message file: OK.
	if err := validateNaturalAttack(&Species{SpeciesId: 1, Name: "ok", NaturalAttack: items.Bite}); err != nil {
		t.Errorf("expected nil for known subtype, got %v", err)
	}
	// Empty: OK (humanoid default).
	if err := validateNaturalAttack(&Species{SpeciesId: 2, Name: "empty"}); err != nil {
		t.Errorf("expected nil for empty, got %v", err)
	}
	// Unknown subtype: error.
	if err := validateNaturalAttack(&Species{SpeciesId: 3, Name: "bad", NaturalAttack: items.ItemSubType("notarealsubtype")}); err == nil {
		t.Error("expected error for unknown subtype")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/species/ -run TestValidateNaturalAttack -v`
Expected: compile error — `validateNaturalAttack` undefined.

- [ ] **Step 3: Implement the validator + call it from LoadDataFiles**

First confirm the set of valid natural-attack subtypes. Use the existing `items` constants that have combat-message files (`bite`,`claws`,`slam`,`gore`,`sting`; allow `unarmed`/empty). Add to `internal/species/species.go`:

```go
// validNaturalAttacks is the set of subtypes a species may declare for its
// basic unarmed attacks. Each must have a loaded combat-message file of the
// same name in _datafiles/world/dogmud/combat-messages/.
var validNaturalAttacks = map[items.ItemSubType]struct{}{
	items.Bite:  {},
	items.Claws: {},
	items.Slam:  {},
	items.Gore:  {},
	items.Sting: {},
}

func validateNaturalAttack(s *Species) error {
	if s.NaturalAttack == "" || s.NaturalAttack == items.Unarmed {
		return nil
	}
	if _, ok := validNaturalAttacks[s.NaturalAttack]; !ok {
		return fmt.Errorf("species %d (%s): unknown natural_attack %q", s.SpeciesId, s.Name, s.NaturalAttack)
	}
	return nil
}
```

In `LoadDataFiles`, after `allSpecies = tmpSpecies`, add (mirrors the house panic-validator style):

```go
	for _, s := range allSpecies {
		if err := validateNaturalAttack(s); err != nil {
			panic(err)
		}
	}
```

Add `"fmt"` to imports if not present.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/species/ -run TestValidateNaturalAttack -v`
Expected: PASS. Then `go build ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/species/species.go internal/species/species_test.go
git commit -m "feat(species): validate natural_attack is a known subtype at load (panic)"
```
(End with the Co-Authored-By line.)

---

## Task 4: Tag non-human species with `natural_attack` (data)

**Files:**
- Modify: `_datafiles/world/dogmud/species/*.yaml` (non-human species)

- [ ] **Step 1: Inventory species + decide each subtype**

List species and their `unarmedname` to choose a subtype:
`grep -rH "name:\|unarmedname:\|body_parts:" _datafiles/world/dogmud/species/*.yaml`

Map each NON-human/non-humanoid species to one of `bite` / `claws` / `slam` / `gore` / `sting` using `unarmedname` as the guide. Suggested mapping (adjust to actual roster):
- jaws/fangs/teeth → `bite` (canine, serpent, vampire-if-beast, …)
- claws/talons/fangs-and-claws → `claws` (feline, bear, raptor, bat, …)
- hooves/fists-of-stone/mandible-less slam → `slam` (deer hooves?, elementals, golems, oozes)
- mandibles/stinger → `sting` (insectoid, arachnid, scorpion)
- horns/antlers/tusks → `gore` (only if such species exist)
Leave humanoids (human, and any humanoid NPC species using "fists") WITHOUT `natural_attack` (they stay generic).

- [ ] **Step 2: Add `natural_attack:` to each chosen species file**

For each non-human species YAML, add a line (next to `unarmedname:`), e.g. in `2-canine.yaml`:
```yaml
natural_attack: bite
```
Use a targeted line insert; do not reformat other content. Every value MUST be one of the validated subtypes (Task 3 will panic otherwise).

- [ ] **Step 3: Verify the data loads**

Run: `go build ./...` then boot: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` and `go run .` (background). Expect `species.LoadDataFiles()` logged with the same count and NO `unknown natural_attack` panic. Then `Server Ready`. Kill the server + clear ports 33333/44444/55555.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/species/
git commit -m "content(species): tag non-human species with natural_attack subtypes"
```
(End with the Co-Authored-By line.)

---

## Task 5: Update context.md docs

**Files:**
- Modify: `internal/species/context.md`, `internal/combat/context.md`, `internal/items/context.md`

- [ ] **Step 1: Document the natural-attack field + wiring**

Update each touched package's `context.md` (established DOGMud norm) to reflect this change — short, factual additions:
- `internal/species/context.md`: add the `NaturalAttack` field to the documented `Species` shape — the unarmed basic-attack message subtype (empty = humanoid/generic), one of `bite`/`claws`/`slam`/`gore`/`sting`, validated at load.
- `internal/combat/context.md`: note that `buildWeaponSetup` resolves the unarmed (no-weapon) attack subtype from the attacker's species `NaturalAttack` (falling back to `Unarmed`→generic); an equipped weapon's subtype still overrides.
- `internal/items/context.md`: note that the `bite`/`claws`/`slam`/`gore`/`sting` `ItemSubType`s are now used by mob natural attacks (not only hypothetical weapons), so their combat-message files are live for basic attacks.

Match each file's existing tone/section structure; don't restructure.

- [ ] **Step 2: Verify build (docs only, no code)**

Run: `go build ./...`
Expected: clean (docs changes don't affect build; this just confirms nothing else drifted).

- [ ] **Step 3: Commit**

```bash
git add internal/species/context.md internal/combat/context.md internal/items/context.md
git commit -m "docs(context): natural-attack subtype field + buildWeaponSetup wiring"
```
(End with the Co-Authored-By line.)

---

## Task 6: Verification + smoke

- [ ] **Step 1: Full build + targeted tests**

Run: `go build ./... && go test ./internal/species/ ./internal/combat/ ./internal/items/`
Expected: all green.

- [ ] **Step 2: Confirm reused message files are complete (crit/fumble + skill tiers)**

The combat-message loader validates that every message file defines all 8
intensities (`prepare, wait, miss, weak, normal, heavy, critical, fumble`) ×
`beginner`/`expert`/`master`. The five reused subtypes (`bite`/`claws`/`slam`/
`gore`/`sting`) are already complete, so a clean boot (no `missing option[...]`
panic from `items/attack_messages.go`) is the proof — no new message authoring in
Phase 1. (New message files are a Phase-3 concern; see the spec's cross-cutting
section A.) Confirm the boot log shows the combat-messages loaded with no
validation error.

- [ ] **Step 3: In-game smoke**

Boot (`go run .`), connect, and fight a non-human mob (e.g. a wolf). Confirm its BASIC attacks now read as bite/claws verbiage (from `bite.yaml`/`claws.yaml`), NOT "punch"/"fist", across hit intensities including a crit/fumble if observed. Confirm a humanoid NPC still reads as generic/fists. Kill the server + clear ports afterward.

- [ ] **Step 4: (No push — local only)**

This phase merges into the running local feature work for a later bundle push. Stop here; Phase 2 is a separate plan.

---

## Self-review notes (controller)

- **Spec coverage (Layer 1 only):** species `natural_attack` field (T1), `buildWeaponSetup` wiring (T2), validation (T3), species tagging (T4), context.md docs (T5), verification + message-completeness + smoke (T6). Per-mob override is deferred to Phase 4.
- **Cross-cutting requirements that apply to Phase 1:** message-file completeness (covered — reused files are loader-validated complete; T6 Step 2) and context.md updates (T5). Parity-registry wiring and helpfiles do NOT apply — Phase 1 adds no command (it's basic-attack messaging via a species data field); those are enumerated in the spec for Phases 2–3.
- **No new message files** needed in Phase 1 (bite/claws/slam/gore/sting all exist, loaded, and validated-complete).
- **Boundary:** this phase does not touch move selection/gating (Phase 2) or add moves (Phase 3).
- **Discovery steps** (T2 species seeder, T4 roster mapping) are verify-then-act, not placeholders.
