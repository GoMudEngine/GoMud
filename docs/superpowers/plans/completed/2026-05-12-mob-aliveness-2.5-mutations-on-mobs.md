# Mob Aliveness 2.5 — Mutations on Mobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Body-plan gate for mutations (species declare `body_parts`, mutations declare `requires_body_parts`) + intrinsic mutations on species that stack additively at character init. Full migration of 35 existing species, 4 new elemental species (sand/storm/ice/smoke), 17 mutation YAMLs, and 9 mob YAMLs.

**Architecture:** `Species` struct gains `BodyParts []string` and `IntrinsicMutations map[string]int`. `MutationSpec.RequiresArms bool` becomes `RequiresBodyParts []string`. `Character.ApplyIntrinsicMutations(species)` merges species intrinsics additively at spawn/creation. The gating filter applies at three sites: the random-roll pool (`GetWeightedPool`), the curated `SpawnMutations` path on mob YAMLs (latent bug fix), and mid-game grants. Validation panics at boot for unknown tags / unknown mutation ids.

**Tech Stack:** Go 1.21+, existing `internal/species`, `internal/mutations`, `internal/characters`, `internal/mobs` packages, existing actor/character abstractions.

**Spec:** `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/species/species.go` | MODIFY | Add `BodyParts []string`, `IntrinsicMutations map[string]int` fields; add `HasBodyPart`, `HasAllBodyParts` helpers; add boot-time validation. |
| `internal/species/species_test.go` | MODIFY | Tests for new fields + helpers. |
| `internal/mutations/mutations.go` | MODIFY | Add `RequiresBodyParts []string`, remove `RequiresArms`. Add `CanApplyTo(*species.Species)` helper. Update `GetWeightedPool` signature. Add boot-time validation. |
| `internal/mutations/mutations_test.go` | MODIFY | Tests for new field + `CanApplyTo` + `GetWeightedPool` filter. |
| `internal/characters/character.go` (or new `intrinsic.go`) | MODIFY | `ApplyIntrinsicMutations(*species.Species)` helper, cap-aware additive merge. |
| `internal/characters/intrinsic_test.go` | NEW | Tests for `ApplyIntrinsicMutations`. |
| `internal/mobs/mobs.go` | MODIFY | (a) Update `GetWeightedPool` call site to pass species; (b) gate curated `SpawnMutations` path; (c) call `ApplyIntrinsicMutations` post-resolution. |
| `internal/mobs/mobs_test.go` | MODIFY | Spawn integration tests (canine never gets extra-arms; curated path skips with warning; intrinsic stacking). |
| `internal/users/users.go` or wherever character init happens | MODIFY | Call `ApplyIntrinsicMutations` on player creation (no-op for humans today, structural plumbing). |
| Mid-game mutation grant site (TBD — likely buff application or `drink` for mutation potions) | MODIFY | Add body-part gate before granting. |
| `_datafiles/world/dogmud/mutations/extra-arms.yaml` | MODIFY | `requires_arms: true` → `requires_body_parts: [arms]` |
| `_datafiles/world/dogmud/mutations/elongated-limbs.yaml` | MODIFY | Add `requires_body_parts: [arms]` (was implicit via RequiresArms? verify; spec says explicit now) |
| `_datafiles/world/dogmud/mutations/clawed-hands.yaml` | MODIFY | `requires_arms: true` → `requires_body_parts: [hands]` |
| Plus 14 more mutation YAMLs (see Task 8 table) | MODIFY | Add `requires_body_parts:` per the catalog. |
| `_datafiles/world/dogmud/species/{1..40}-*.yaml` (35 existing) | MODIFY | Add `body_parts:` and `intrinsic_mutations:` per migration table. |
| `_datafiles/world/dogmud/species/{NN}-sand_elemental.yaml` | NEW | New species. ID via `python tools/id_inventory.py --type species`. |
| `_datafiles/world/dogmud/species/{NN}-storm_elemental.yaml` | NEW | New species. |
| `_datafiles/world/dogmud/species/{NN}-ice_elemental.yaml` | NEW | New species. |
| `_datafiles/world/dogmud/species/{NN}-smoke_elemental.yaml` | NEW | New species. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/318-sand_elemental.yaml` | MODIFY | `speciesid` → sand. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/319-storm_elemental.yaml` | MODIFY | `speciesid` → storm. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/320-elemental_king.yaml` | MODIFY | Keep speciesid 40, add mob-YAML `mutations: { large: 1 }`. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml` | MODIFY | `speciesid` → ice, REMOVE existing `mutations: { incorporeal: 4 }`. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/322-elemental_prince.yaml` | MODIFY | `speciesid` → smoke. |
| Mob YAMLs using species 32/33/38/39 with redundant `mutations: { incorporeal: 4 }` | MODIFY | Remove the redundant overrides. (Find via grep.) |
| `internal/species/context.md` | MODIFY | Document `body_parts` + `intrinsic_mutations` + helpers. |
| `internal/mutations/context.md` | MODIFY | Document new schema, gating pipeline, intrinsic stacking, RequiresArms removal. |
| `internal/characters/context.md` | MODIFY | Document `ApplyIntrinsicMutations` + init pathway. |
| `_datafiles/world/dogmud/templates/help/mutations.template` | MODIFY | Document body-parts gating in player-facing help. |
| `_datafiles/world/dogmud/templates/help/species.template` | MODIFY | Document `body_parts` and `intrinsic_mutations`. |
| Per-mutation help templates (`extra-arms.template`, etc., for mutations gaining body-part requirements) | MODIFY | Add "Requires: <body part>" line. |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark chunk 2.5 Done, roll-up 12/41 → 13/41. |

---

## Task 1: `Species.BodyParts` + `IntrinsicMutations` + helpers

**Files:**
- Modify: `internal/species/species.go`
- Modify: `internal/species/species_test.go`

Adds the two new fields to the `Species` struct, two helper methods (`HasBodyPart`, `HasAllBodyParts`), and the canonical body-part tag whitelist. Validation against the whitelist is added in a later task once consumer sites are ready (avoids breaking the boot during partial migration).

- [ ] **Step 1: Add the canonical body-part tag list + struct fields**

Open `internal/species/species.go`. Find the `Species` struct (around line 32 per the recon). Add two new fields at the bottom of the field list. Also add the canonical tag list and helper methods at the bottom of the file.

```go
// Add to imports if not present:
// (no new imports needed — string slices and maps only)

// CanonicalBodyParts is the exhaustive set of body-part tags that
// can appear in Species.BodyParts and MutationSpec.RequiresBodyParts.
// Boot-time validation rejects any value not in this set.
var CanonicalBodyParts = []string{
    "arms",  // explicit grasping limbs distinct from legs
    "hands", // fingered manipulators on the arms
    "legs",  // distinct locomotion limbs
    "eyes",  // visual organs
    "mouth", // biting/vocal apparatus
    "skin",  // surface coverage
    "tail",  // anatomical tail
}

// IsCanonicalBodyPart reports whether the given tag is in the
// canonical set.
func IsCanonicalBodyPart(tag string) bool {
    for _, t := range CanonicalBodyParts {
        if t == tag {
            return true
        }
    }
    return false
}
```

In the `Species` struct, add (preserve indentation/yaml tags consistent with surrounding fields):

```go
    // BodyParts is the set of canonical body-part tags this species
    // has. Empty (nil) means "no field declared in YAML" — treated
    // as fail-open (every mutation passes the gate). Explicit empty
    // slice means "no body parts" (incorporeal-style — gates every
    // body-part-requiring mutation). Validated at boot.
    BodyParts []string `yaml:"body_parts,omitempty"`

    // IntrinsicMutations maps mutation id → baseline rank. Merged
    // additively into Character.Mutations at character init.
    // Cap-aware via the mutation's max rank.
    IntrinsicMutations map[string]int `yaml:"intrinsic_mutations,omitempty"`
```

Append the helper methods to the file:

```go
// HasBodyPart returns true if this species declares the given
// canonical body-part tag, OR if BodyParts is nil (fail-open for
// un-migrated species). An explicit empty slice means "no body
// parts" and returns false for every tag.
func (s *Species) HasBodyPart(part string) bool {
    if s == nil {
        return true // defensive fail-open
    }
    if s.BodyParts == nil {
        return true // un-migrated species fail-open
    }
    for _, p := range s.BodyParts {
        if p == part {
            return true
        }
    }
    return false
}

// HasAllBodyParts returns true if this species has every part in
// the requirements list. Empty requirements list always returns
// true (body-agnostic mutations).
func (s *Species) HasAllBodyParts(required []string) bool {
    for _, part := range required {
        if !s.HasBodyPart(part) {
            return false
        }
    }
    return true
}
```

- [ ] **Step 2: Run any existing species tests to confirm no regression**

Run: `go test ./internal/species/ -v`
Expected: existing tests still pass; nothing breaks because the new fields are optional.

- [ ] **Step 3: Add unit tests for new fields + helpers in `species_test.go`**

```go
func TestHasBodyPart_NilBodyParts_FailOpen(t *testing.T) {
    s := &Species{Name: "test", BodyParts: nil}
    if !s.HasBodyPart("arms") {
        t.Error("nil BodyParts should fail-open (return true)")
    }
}

func TestHasBodyPart_EmptySlice_GatesEverything(t *testing.T) {
    s := &Species{Name: "test", BodyParts: []string{}}
    if s.HasBodyPart("arms") {
        t.Error("explicit empty BodyParts should return false (incorporeal-style)")
    }
}

func TestHasBodyPart_PresentTag(t *testing.T) {
    s := &Species{Name: "test", BodyParts: []string{"arms", "legs"}}
    if !s.HasBodyPart("arms") {
        t.Error("present tag should return true")
    }
    if !s.HasBodyPart("legs") {
        t.Error("present tag should return true")
    }
}

func TestHasBodyPart_AbsentTag(t *testing.T) {
    s := &Species{Name: "test", BodyParts: []string{"arms", "legs"}}
    if s.HasBodyPart("tail") {
        t.Error("absent tag should return false")
    }
}

func TestHasAllBodyParts_EmptyRequirements(t *testing.T) {
    s := &Species{BodyParts: []string{"arms"}}
    if !s.HasAllBodyParts(nil) {
        t.Error("empty requirements should always return true")
    }
    if !s.HasAllBodyParts([]string{}) {
        t.Error("empty requirements should always return true")
    }
}

func TestHasAllBodyParts_AllPresent(t *testing.T) {
    s := &Species{BodyParts: []string{"arms", "hands", "legs"}}
    if !s.HasAllBodyParts([]string{"arms", "hands"}) {
        t.Error("all required parts present should return true")
    }
}

func TestHasAllBodyParts_SomeMissing(t *testing.T) {
    s := &Species{BodyParts: []string{"arms", "legs"}}
    if s.HasAllBodyParts([]string{"arms", "hands"}) {
        t.Error("missing required part should return false")
    }
}

func TestIsCanonicalBodyPart(t *testing.T) {
    valid := []string{"arms", "hands", "legs", "eyes", "mouth", "skin", "tail"}
    for _, v := range valid {
        if !IsCanonicalBodyPart(v) {
            t.Errorf("%q should be canonical", v)
        }
    }
    invalid := []string{"wings", "horns", "fins", "tentacle", ""}
    for _, v := range invalid {
        if IsCanonicalBodyPart(v) {
            t.Errorf("%q should NOT be canonical", v)
        }
    }
}
```

- [ ] **Step 4: Run the new tests**

Run: `go test ./internal/species/ -run 'HasBodyPart|HasAllBodyParts|IsCanonicalBodyPart' -v`
Expected: all PASS.

- [ ] **Step 5: Full build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/species/species.go internal/species/species_test.go
git commit -m "feat(species): add BodyParts + IntrinsicMutations fields with helpers"
```

---

## Task 2: `MutationSpec.RequiresBodyParts` (remove `RequiresArms`)

**Files:**
- Modify: `internal/mutations/mutations.go`
- Modify: `internal/mutations/mutations_test.go`
- Modify: `_datafiles/world/dogmud/mutations/extra-arms.yaml`
- Modify: `_datafiles/world/dogmud/mutations/elongated-limbs.yaml`
- Modify: `_datafiles/world/dogmud/mutations/clawed-hands.yaml`

Atomic field migration: add `RequiresBodyParts`, remove `RequiresArms`, update the YAMLs that used the old field, and update the consumer site in `GetWeightedPool`. This is the "old field gone forever" version — no transitional state.

- [ ] **Step 1: Add the new field; remove the old field**

In `internal/mutations/mutations.go`, find the `MutationSpec` struct (around line 47 per the recon). Replace:

```go
    RequiresArms bool `yaml:"requires_arms,omitempty"`
```

With:

```go
    // RequiresBodyParts lists canonical body-part tags from
    // species.CanonicalBodyParts. Empty/nil = body-agnostic.
    // Validated at boot against the canonical set.
    RequiresBodyParts []string `yaml:"requires_body_parts,omitempty"`
```

(If there are OTHER readers of `RequiresArms` beyond `GetWeightedPool`, those need to be updated too. Find them via `grep -rn RequiresArms internal/`. As of the recon, only one reader existed at `mutations.go:233`.)

- [ ] **Step 2: Add `CanApplyTo` helper to `mutations.go`**

Append:

```go
import (
    // Make sure species is imported.
    "github.com/GoMudEngine/GoMud/internal/species"
)

// CanApplyTo reports whether this mutation's body-part requirements
// are satisfied by the given species. Empty requirements pass for
// any embodied or unembodied species.
func (s *MutationSpec) CanApplyTo(sp *species.Species) bool {
    if sp == nil {
        return true // defensive fail-open
    }
    return sp.HasAllBodyParts(s.RequiresBodyParts)
}
```

(Be mindful of import cycles — if `internal/species` imports `internal/mutations`, this fails. Per the recon, species does NOT import mutations, so the dependency arrow is `mutations → species`. Verify by grepping species/*.go for "mutations" — should be none.)

- [ ] **Step 3: Update `GetWeightedPool` signature and body**

Current signature (per recon at `mutations.go:210`):

```go
func GetWeightedPool(current map[string]int, disabledSlots []string) []*MutationSpec {
    // ... at line 220, checks if RequiresArms by inspecting DisabledSlots
}
```

New signature:

```go
func GetWeightedPool(current map[string]int, sp *species.Species) []*MutationSpec {
    var pool []*MutationSpec
    for _, spec := range allMutationSpecs {
        if _, owned := current[spec.MutationId]; owned {
            continue
        }
        // Existing conflict check (unchanged from current code)
        if conflictsWithExisting(spec, current) {
            continue
        }
        // NEW: body-parts gate
        if !spec.CanApplyTo(sp) {
            continue
        }
        // Existing rarity weighting (unchanged)
        weight := 11 - spec.Rarity
        if weight < 1 {
            weight = 1
        }
        for i := 0; i < weight; i++ {
            pool = append(pool, spec)
        }
    }
    return pool
}
```

Replace the existing body verbatim — the variable name `disabledSlots` and the `RequiresArms`/`hasArms` check go away entirely.

- [ ] **Step 4: Migrate the 3 mutation YAMLs that used `requires_arms`**

In `_datafiles/world/dogmud/mutations/extra-arms.yaml`, find:

```yaml
requires_arms: true
```

Replace with:

```yaml
requires_body_parts: [arms]
```

In `_datafiles/world/dogmud/mutations/elongated-limbs.yaml`, the spec text says it conflicts with extra-arms; verify it currently uses `requires_arms: true`. If yes, replace with `requires_body_parts: [arms]`. If no, just add `requires_body_parts: [arms]` as a new field. (The conflict relationship is independent — `conflicts:` stays as-is.)

In `_datafiles/world/dogmud/mutations/clawed-hands.yaml`, find any `requires_arms:` line. Replace with:

```yaml
requires_body_parts: [hands]
```

If the file does NOT currently have `requires_arms`, just add the new field.

- [ ] **Step 5: Update the call site in `internal/mobs/mobs.go`**

The recon showed `mobs.go:566–569` calls `GetWeightedPool(mob.Character.Mutations, specDisabledSlots)`. Update to:

```go
pool := mutations.GetWeightedPool(mob.Character.Mutations, sp)
```

Where `sp` is `species.GetSpecies(mob.Character.SpeciesId)` — typically already loaded above. Confirm by reading the surrounding 10 lines.

- [ ] **Step 6: Add tests for `CanApplyTo` and `GetWeightedPool` filter**

Append to `internal/mutations/mutations_test.go`:

```go
func TestCanApplyTo_NoRequirements(t *testing.T) {
    spec := &MutationSpec{MutationId: "test", RequiresBodyParts: nil}
    sp := &species.Species{BodyParts: []string{}} // incorporeal
    if !spec.CanApplyTo(sp) {
        t.Error("body-agnostic mutation should apply to any species")
    }
}

func TestCanApplyTo_RequirementMet(t *testing.T) {
    spec := &MutationSpec{MutationId: "test", RequiresBodyParts: []string{"arms"}}
    sp := &species.Species{BodyParts: []string{"arms", "legs"}}
    if !spec.CanApplyTo(sp) {
        t.Error("met requirement should pass")
    }
}

func TestCanApplyTo_RequirementMissing(t *testing.T) {
    spec := &MutationSpec{MutationId: "test", RequiresBodyParts: []string{"arms"}}
    sp := &species.Species{BodyParts: []string{"legs"}}
    if spec.CanApplyTo(sp) {
        t.Error("missing requirement should fail")
    }
}

func TestGetWeightedPool_FiltersByBodyParts(t *testing.T) {
    // Synthetic state: pretend allMutationSpecs has one extra-arms
    // (requires arms) and one tough-skin (requires skin). Wolf-like
    // species has only [legs, eyes, mouth, skin, tail].
    //
    // This test relies on the real loaded catalog. Use existing
    // mutations: extra-arms (arms) + tough-skin (skin). Verify
    // pool for a body-parts: [legs, eyes, mouth, skin, tail]
    // species includes tough-skin but excludes extra-arms.
    sp := &species.Species{
        SpeciesId: 999,
        BodyParts: []string{"legs", "eyes", "mouth", "skin", "tail"},
    }
    current := map[string]int{}
    pool := GetWeightedPool(current, sp)
    seen := map[string]bool{}
    for _, spec := range pool {
        seen[spec.MutationId] = true
    }
    if seen["extra-arms"] {
        t.Error("pool should EXCLUDE extra-arms for species without arms")
    }
    if !seen["tough-skin"] {
        t.Error("pool should INCLUDE tough-skin for species with skin")
    }
}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./internal/mutations/ -run 'CanApplyTo|GetWeightedPool' -v`
Expected: all PASS.

- [ ] **Step 8: Full build**

Run: `go build ./...`
Expected: clean. If any package fails because of the removed `RequiresArms`, grep for the symbol and update those sites — should only be `mobs.go`.

- [ ] **Step 9: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/mutations_test.go internal/mobs/mobs.go _datafiles/world/dogmud/mutations/extra-arms.yaml _datafiles/world/dogmud/mutations/elongated-limbs.yaml _datafiles/world/dogmud/mutations/clawed-hands.yaml
git commit -m "refactor(mutations): replace RequiresArms with RequiresBodyParts"
```

---

## Task 3: `Character.ApplyIntrinsicMutations` helper + tests

**Files:**
- Create: `internal/characters/intrinsic.go`
- Create: `internal/characters/intrinsic_test.go`

A new helper for character init. Lives in its own file to keep `character.go` from growing further.

- [ ] **Step 1: Create `internal/characters/intrinsic.go`**

```go
package characters

import (
    "github.com/GoMudEngine/GoMud/internal/mutations"
    "github.com/GoMudEngine/GoMud/internal/species"
)

// ApplyIntrinsicMutations merges the species's intrinsic mutations
// additively into the character's Mutations map. Cap-aware: each
// combined rank is clamped to the mutation's max rank if declared
// (default cap = 4, matching the chunk-2.2a convention for ranked
// mutations).
//
// Called from mob spawn AND player creation after all other
// mutation logic (curated SpawnMutations from YAML + random
// roll + persistent acquired). No-op if species is nil or has
// no intrinsic_mutations.
func (c *Character) ApplyIntrinsicMutations(sp *species.Species) {
    if sp == nil || len(sp.IntrinsicMutations) == 0 {
        return
    }
    if c.Mutations == nil {
        c.Mutations = make(map[string]int)
    }
    for id, intrinsicRank := range sp.IntrinsicMutations {
        cap := 4 // default cap
        if spec := mutations.GetSpec(id); spec != nil && spec.MaxRank > 0 {
            cap = spec.MaxRank
        }
        combined := c.Mutations[id] + intrinsicRank
        if combined > cap {
            combined = cap
        }
        c.Mutations[id] = combined
    }
}
```

If `mutations.GetSpec(id)` doesn't exist with that name, find the equivalent loader (likely `mutations.LookupMutation(id)` or similar via the existing catalog access pattern). The function should return `*MutationSpec` or nil.

If `MaxRank` doesn't exist as a field on `MutationSpec`, default the cap to 4 unconditionally (which is the chunk-2.2a convention; verify in `internal/mutations/mutations.go` schema).

- [ ] **Step 2: Create `internal/characters/intrinsic_test.go`**

```go
package characters

import (
    "testing"

    "github.com/GoMudEngine/GoMud/internal/species"
)

func TestApplyIntrinsicMutations_NilSpecies(t *testing.T) {
    c := &Character{}
    c.ApplyIntrinsicMutations(nil)
    if len(c.Mutations) != 0 {
        t.Errorf("nil species should leave Mutations empty, got %d entries", len(c.Mutations))
    }
}

func TestApplyIntrinsicMutations_EmptyIntrinsic(t *testing.T) {
    c := &Character{}
    sp := &species.Species{IntrinsicMutations: nil}
    c.ApplyIntrinsicMutations(sp)
    if len(c.Mutations) != 0 {
        t.Errorf("empty intrinsic should leave Mutations empty, got %d entries", len(c.Mutations))
    }
}

func TestApplyIntrinsicMutations_AddsToEmpty(t *testing.T) {
    c := &Character{}
    sp := &species.Species{IntrinsicMutations: map[string]int{"tail": 1, "keen-eyes": 1}}
    c.ApplyIntrinsicMutations(sp)
    if c.Mutations["tail"] != 1 {
        t.Errorf("tail should be 1, got %d", c.Mutations["tail"])
    }
    if c.Mutations["keen-eyes"] != 1 {
        t.Errorf("keen-eyes should be 1, got %d", c.Mutations["keen-eyes"])
    }
}

func TestApplyIntrinsicMutations_StacksAdditively(t *testing.T) {
    c := &Character{Mutations: map[string]int{"tail": 1}}
    sp := &species.Species{IntrinsicMutations: map[string]int{"tail": 1}}
    c.ApplyIntrinsicMutations(sp)
    if c.Mutations["tail"] != 2 {
        t.Errorf("tail should stack to 2, got %d", c.Mutations["tail"])
    }
}

func TestApplyIntrinsicMutations_ClampsToCap(t *testing.T) {
    c := &Character{Mutations: map[string]int{"tail": 4}}
    sp := &species.Species{IntrinsicMutations: map[string]int{"tail": 2}}
    c.ApplyIntrinsicMutations(sp)
    if c.Mutations["tail"] != 4 {
        t.Errorf("tail should clamp at 4, got %d", c.Mutations["tail"])
    }
}
```

- [ ] **Step 3: Run the tests**

Run: `go test ./internal/characters/ -run TestApplyIntrinsicMutations -v`
Expected: all PASS.

- [ ] **Step 4: Full build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/intrinsic.go internal/characters/intrinsic_test.go
git commit -m "feat(characters): ApplyIntrinsicMutations helper for species → character merge"
```

---

## Task 4: Mob spawn pipeline — gate curated path + call intrinsic

**Files:**
- Modify: `internal/mobs/mobs.go`
- Modify: `internal/mobs/mobs_test.go`

The recon noted `mobs.go:554-558` is the curated `SpawnMutations` path with no checks. Fix that, and call `ApplyIntrinsicMutations` after the random-roll path. Per the spec § "Data flow / Mob spawn":

1. Curated SpawnMutations (with new body-parts gate + warning log on rejection)
2. Random roll via GetWeightedPool (already updated in Task 2)
3. ApplyIntrinsicMutations (new)

- [ ] **Step 1: Update the curated SpawnMutations loop**

Locate the existing block around `mobs.go:554-558` (find via `grep -n "SpawnMutations" internal/mobs/mobs.go`). Currently it iterates and assigns unconditionally:

```go
// OLD (paraphrased — confirm against actual code):
for id, rank := range mob.SpawnMutations {
    mob.Character.Mutations[id] = rank
}
```

Replace with:

```go
sp := species.GetSpecies(mob.Character.SpeciesId)
for id, rank := range mob.SpawnMutations {
    spec := mutations.GetSpec(id)
    if spec == nil {
        mudlog.Warn("MobSpawn",
            "msg", "unknown mutation id in SpawnMutations",
            "mobId", mob.MobId, "mutation", id)
        continue
    }
    if !spec.CanApplyTo(sp) {
        mudlog.Warn("MobSpawn",
            "msg", "mutation requirements not met by species",
            "mobId", mob.MobId,
            "mutation", id,
            "species", sp.Name,
            "requires", spec.RequiresBodyParts,
            "species_body_parts", sp.BodyParts)
        continue
    }
    mob.Character.Mutations[id] = rank
}
```

Imports needed if not already: `mutations`, `species`, `mudlog`.

- [ ] **Step 2: Call `ApplyIntrinsicMutations` after the random-roll path**

The random-roll path is at `mobs.go:561-574` per the recon. After the roll completes (whether it landed or not), insert a call to merge intrinsics. The right place is AFTER the random roll's `mob.Character.Mutations[picked] = 1` assignment AND inside the same overall init function.

```go
// After the random roll:
mob.Character.ApplyIntrinsicMutations(sp)
```

(If `sp` isn't already a local variable at that point, declare it earlier as in Step 1 — re-resolve via `species.GetSpecies(mob.Character.SpeciesId)` and pass the same pointer.)

- [ ] **Step 3: Add spawn integration test — canine never rolls extra-arms**

Append to `internal/mobs/mobs_test.go`:

```go
func TestMobSpawn_CanineNeverGetsExtraArms(t *testing.T) {
    // Synthetic: create a canine-like species (no arms) and a mob
    // template with mutationchance: 100. Spawn N times; none
    // should ever roll extra-arms.
    //
    // This test requires the real catalog. Use the existing canine
    // species (id 2) after the migration in Task 7. To make this
    // test runnable BEFORE Task 7's YAML migration completes, set
    // BodyParts directly on a constructed species and seed it via
    // species.SeedSpeciesForTest (if such helper exists; if not,
    // construct an inline *species.Species pointer and pass to
    // GetWeightedPool directly without going through GetSpecies).
    sp := &species.Species{
        SpeciesId: 2,
        Name:      "canine",
        BodyParts: []string{"legs", "eyes", "mouth", "skin", "tail"},
    }
    current := map[string]int{}
    seen := map[string]int{}
    for i := 0; i < 200; i++ {
        pool := mutations.GetWeightedPool(current, sp)
        if len(pool) == 0 {
            continue
        }
        picked := pool[util.Rand(len(pool))]
        seen[picked.MutationId]++
    }
    if seen["extra-arms"] > 0 {
        t.Errorf("canine should NEVER roll extra-arms, got %d times", seen["extra-arms"])
    }
    if seen["elongated-limbs"] > 0 {
        t.Errorf("canine should NEVER roll elongated-limbs, got %d times", seen["elongated-limbs"])
    }
    if seen["clawed-hands"] > 0 {
        t.Errorf("canine should NEVER roll clawed-hands, got %d times", seen["clawed-hands"])
    }
}

func TestMobSpawn_IntrinsicStackingWithRandomRoll(t *testing.T) {
    // Setup: a species with intrinsic tail: 1. After ApplyIntrinsic
    // is called, Character.Mutations[tail] = 1. If we then
    // simulate a random roll of tail (assign rank 1 BEFORE the
    // intrinsic call), the final should be 2 (intrinsic stacked).
    sp := &species.Species{
        IntrinsicMutations: map[string]int{"tail": 1},
    }
    c := &characters.Character{Mutations: map[string]int{"tail": 1}}
    c.ApplyIntrinsicMutations(sp)
    if c.Mutations["tail"] != 2 {
        t.Errorf("expected tail rank 2 (1 acquired + 1 intrinsic), got %d", c.Mutations["tail"])
    }
}
```

Add imports as needed: `mutations`, `species`, `characters`, `util`.

(If the `util.Rand` symbol isn't in scope from `internal/util`, find the equivalent for random index.)

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/mobs/ -run 'TestMobSpawn_CanineNeverGets|TestMobSpawn_IntrinsicStacking' -v`
Expected: PASS. If the canine test depends on the migrated species file, it might need to be skipped until Task 7 lands — use `t.Skip("requires species migration from Task 7")` and document. Better: use a synthetic `*species.Species` inline (as the test code does) so it doesn't depend on YAML state.

- [ ] **Step 5: Full build + full test suite**

Run: `go build ./... && go test ./...`
Expected: clean build, no FAILs.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "feat(mobs): gate SpawnMutations + apply intrinsic on spawn"
```

---

## Task 5: Player + mid-game mutation acquisition path

**Files:**
- Modify: `internal/users/users.go` or wherever player character creation finalizes
- Modify: the mid-game mutation grant path (likely a buff applier or mutation potion handler)

The player path is structural — humans currently have no intrinsic, so this is a no-op for human players. Plumbing it in now means future-proofing.

- [ ] **Step 1: Find the player character-creation finalization site**

Run: `grep -rn "Character.Mutations = " internal/users/ internal/hooks/ internal/usercommands/ 2>/dev/null | head -10`

Look for the site where a freshly-created character's mutations are initialized. Likely in `internal/users/` or a character-creation hook. The right insertion point is AFTER any random mutation assignment for new players (if any) AND BEFORE the first character save.

- [ ] **Step 2: Add `ApplyIntrinsicMutations` call at the player-creation site**

Once you've located the site, add:

```go
if sp := species.GetSpecies(user.Character.SpeciesId); sp != nil {
    user.Character.ApplyIntrinsicMutations(sp)
}
```

- [ ] **Step 3: Find the mid-game mutation grant site**

This is where mutation potions, quest grants, admin commands assign new mutations. Run:

```
grep -rn "Character.Mutations\[" internal/ 2>/dev/null | grep -v "_test.go" | head -20
```

The most likely candidate is the `drink` handler that processes mutation-potion buffs, OR a dedicated `mutations.GrantTo(char, id, rank)` helper. If a centralized helper exists, modify IT. If grants happen inline, find the canonical inline path and gate THERE.

- [ ] **Step 4: Add the body-part gate at the grant site**

```go
// Find the spot where a mutation id is about to be added to char.Mutations
// and add this guard before assignment:
sp := species.GetSpecies(char.SpeciesId)
spec := mutations.GetSpec(mutationId)
if spec != nil && !spec.CanApplyTo(sp) {
    // Player-facing rejection
    if userId > 0 {
        if u := users.GetByUserId(userId); u != nil {
            u.SendText("Your body cannot integrate this mutation.")
        }
    }
    return // or continue, depending on call-site flow
}
char.Mutations[mutationId] = newRank
```

The exact integration depends on the call site — adapt as needed. If you can't find a single canonical mid-game grant site, document it as DONE_WITH_CONCERNS and note that mid-game grants are currently unrestricted on player species. The chunk's primary functionality (mob spawn + body-parts gating) is unaffected.

- [ ] **Step 5: Build + test**

Run: `go build ./... && go test ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add <files-touched>
git commit -m "feat(characters): wire intrinsic + body-parts gate into player + mid-game paths"
```

---

## Task 6: Boot-time validation for body-parts tags + intrinsic refs

**Files:**
- Modify: `internal/species/species.go` (add validation entry point)
- Modify: `internal/mutations/mutations.go` (add validation entry point)
- Modify: the data-load orchestration (likely `main.go` or a `validate.go` somewhere)

Per the spec § "Validation rules": panic at boot for unknown body-part tags, unknown mutation ids in intrinsic_mutations, etc.

- [ ] **Step 1: Add validation in `internal/species/species.go`**

```go
// ValidateBodyPartTags scans all loaded species and panics on any
// unknown body-part tag or unknown intrinsic mutation id. Called
// from main after species + mutations are loaded.
func ValidateBodyPartTags(mutationIdExists func(id string) bool) {
    for _, sp := range allSpecies {
        for _, tag := range sp.BodyParts {
            if !IsCanonicalBodyPart(tag) {
                panic(fmt.Sprintf(
                    "species %q (id %d): unknown body_part tag %q (canonical: %v)",
                    sp.Name, sp.SpeciesId, tag, CanonicalBodyParts))
            }
        }
        for id := range sp.IntrinsicMutations {
            if !mutationIdExists(id) {
                panic(fmt.Sprintf(
                    "species %q (id %d): unknown mutation id in intrinsic_mutations: %q",
                    sp.Name, sp.SpeciesId, id))
            }
        }
    }
}
```

Add the `fmt` import if not present. `allSpecies` is the existing in-memory species map — confirm the exact symbol via the existing `GetSpecies` accessor.

- [ ] **Step 2: Add validation in `internal/mutations/mutations.go`**

```go
// ValidateBodyPartTags scans all loaded mutation specs and panics
// on any unknown body-part tag in RequiresBodyParts.
func ValidateBodyPartTags() {
    for _, spec := range allMutationSpecs {
        for _, tag := range spec.RequiresBodyParts {
            if !species.IsCanonicalBodyPart(tag) {
                panic(fmt.Sprintf(
                    "mutation %q: unknown requires_body_parts tag %q (canonical: %v)",
                    spec.MutationId, tag, species.CanonicalBodyParts))
            }
        }
    }
}
```

- [ ] **Step 3: Wire the validation calls at boot**

Find the data-load orchestration. Likely candidates: `main.go`, a `loader.go`, or `internal/configs/`. Run:

```
grep -rn "mutations.LoadDataFiles\|species.LoadDataFiles" internal/ main.go *.go 2>/dev/null | head -10
```

Find where both packages' loaders are called. Insert AFTER both have completed:

```go
// Cross-reference validation: body-part tags and intrinsic
// mutation references must be coherent.
mutations.ValidateBodyPartTags()
species.ValidateBodyPartTags(mutations.HasSpec)
```

Where `mutations.HasSpec(id) bool` is a helper that returns true if a mutation with that id exists in the loaded catalog. Add it if it doesn't exist:

```go
// In internal/mutations/mutations.go:
func HasSpec(id string) bool {
    _, ok := allMutationSpecs[id]
    return ok
}
```

- [ ] **Step 4: Boot the server to verify validation passes (no panics with current YAMLs)**

Run: `go run . > /tmp/boot.log 2>&1 &` in the background; wait ~15 seconds; check `tail -20 /tmp/boot.log` for "Server Ready" and absence of panic lines. Kill: `pkill -f 'go run'` (or `ps -ef | grep -E '(go run|dogmud)' | grep -v grep | awk '{print $2}' | xargs -r kill -9`).

Expected: clean boot. No species YAML has BodyParts yet (Task 7 hasn't landed), so validation has nothing to check on the species side. The mutation YAML side has 3 migrated entries from Task 2 — validation should pass them.

- [ ] **Step 5: Commit**

```bash
git add internal/species/species.go internal/mutations/mutations.go main.go
git commit -m "feat(species,mutations): boot-time validation of body-parts tags"
```

(Adjust the file list to match what you actually touched.)

---

## Task 7: Migrate existing 35 species YAMLs

**Files:**
- Modify: 35 YAML files in `_datafiles/world/dogmud/species/`

Add `body_parts:` and (optionally) `intrinsic_mutations:` to each species YAML per the spec migration table. Single big mechanical task. Following table is the source of truth — apply row-by-row.

**SKIP:** 19-dummy.yaml and 20-orb.yaml (test/system species per the spec).

- [ ] **Step 1: Apply all 35 species edits per the table below**

For each species YAML, add the `body_parts:` line at top level (sibling of `stats:`, `damage:`, etc.). If `intrinsic_mutations:` is non-empty for that row, add it too.

| File | Add `body_parts:` | Add `intrinsic_mutations:` (if non-empty) |
|---|---|---|
| `1-human.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | (none) |
| `2-canine.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1 }` |
| `3-bear.yaml` | `[arms, legs, eyes, mouth, skin]` | `{ thick-hide: 1 }` |
| `4-troll.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | `{ regenerative-tissue: 1 }` |
| `5-goblin.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | (none) |
| `6-boar.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, thick-hide: 1 }` |
| `7-deer.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, keen-eyes: 1 }` |
| `8-serpent.yaml` | `[eyes, mouth, skin]` | (none) |
| `9-raptor.yaml` | `[legs, eyes, mouth, skin]` | `{ keen-eyes: 1 }` |
| `10-rodent.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, small: 1 }` |
| `11-feline.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, night-vision: 1 }` |
| `12-insectoid.yaml` | `[legs, eyes, mouth, skin]` | (none) |
| `13-fish.yaml` | `[eyes, mouth, skin]` | `{ cold-blooded: 1 }` |
| `14-carnivorous_plant.yaml` | `[mouth, skin]` | `{ photosynthetic-skin: 1 }` |
| `15-fungal_colony.yaml` | `[skin]` | `{ photosynthetic-skin: 1 }` |
| `16-slime.yaml` | `[skin]` | `{ regenerative-tissue: 1 }` |
| `17-arachnid.yaml` | `[legs, eyes, mouth, skin]` | `{ tremorsense: 1 }` |
| `18-worm.yaml` | `[mouth, skin]` | `{ tremorsense: 1 }` |
| `21-reptile.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, cold-blooded: 1 }` |
| `22-bat.yaml` | `[legs, eyes, mouth, skin]` | `{ night-vision: 1, tremorsense: 1 }` |
| `23-aberration.yaml` | `[]` (explicit empty — no body-part assumptions) | (none) |
| `24-mustelid.yaml` | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1 }` |
| `30-skeleton.yaml` | `[arms, hands, legs, eyes, mouth]` (no skin) | `{ cold-blooded: 1, hollow-bones: 1 }` |
| `31-zombie.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | `{ cold-blooded: 1 }` |
| `32-wraith.yaml` | `[]` | `{ incorporeal: 4 }` |
| `33-spectre.yaml` | `[]` | `{ incorporeal: 4 }` |
| `34-vampire.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | `{ regenerative-tissue: 1, night-vision: 1 }` |
| `35-flesh_golem.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | `{ iron-constitution: 1 }` |
| `36-water_elemental.yaml` | `[skin]` | `{ regenerative-tissue: 1 }` |
| `37-earth_elemental.yaml` | `[arms, legs, skin]` | `{ thick-hide: 1, iron-constitution: 1 }` |
| `38-air_elemental.yaml` | `[]` | `{ incorporeal: 4 }` |
| `39-fire_elemental.yaml` | `[]` | `{ incorporeal: 4 }` |
| `40-magma_elemental.yaml` | `[skin]` | `{ thick-hide: 1 }` |
| `99-ascended.yaml` | `[arms, hands, legs, eyes, mouth, skin]` | `{ magical-resistance: 1 }` |
| `0-ghostly_spirit.yaml` | `[]` | `{ incorporeal: 4 }` |

YAML format for each insertion (preserve surrounding indentation; these are at the top level, not nested under `character:` or `stats:`):

```yaml
body_parts: [arms, hands, legs, eyes, mouth, skin]
intrinsic_mutations:
  tail: 1
  night-vision: 1
```

(Maps use the block form; lists can use the inline `[a, b, c]` form for readability.)

- [ ] **Step 2: Boot the server to verify all species parse + validate**

Run: `go run . > /tmp/boot.log 2>&1 &`; wait ~15 seconds; check `tail -30 /tmp/boot.log`.

Expected: server reaches Server Ready. Watch for any panic line containing "unknown body_part tag" or "unknown mutation id in intrinsic_mutations" — both indicate a typo to fix.

Kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/species/
git commit -m "feat(species): migrate 35 species YAMLs to body_parts + intrinsic_mutations"
```

---

## Task 8: Migrate remaining 14 mutation YAMLs (add `requires_body_parts`)

**Files:**
- Modify: 14 mutation YAMLs in `_datafiles/world/dogmud/mutations/`

The 3 mutations using `requires_arms` were migrated in Task 2. This task adds `requires_body_parts:` to the other 14 mutations that need body-part gating per the spec § "Mutation requirements" table.

- [ ] **Step 1: Add `requires_body_parts:` to each YAML below**

For each YAML, add the line at top level (alongside `mutationid:`, `rarity:`, etc.):

| File | Add line |
|---|---|
| `extra-legs.yaml` | `requires_body_parts: [legs]` |
| `keen-eyes.yaml` | `requires_body_parts: [eyes]` |
| `night-vision.yaml` | `requires_body_parts: [eyes]` |
| `infrared-vision.yaml` | `requires_body_parts: [eyes]` |
| `toxic-bite.yaml` | `requires_body_parts: [mouth]` |
| `sonic-shout.yaml` | `requires_body_parts: [mouth]` |
| `blinding-spit.yaml` | `requires_body_parts: [mouth]` |
| `tough-skin.yaml` | `requires_body_parts: [skin]` |
| `thick-hide.yaml` | `requires_body_parts: [skin]` |
| `camo-skin.yaml` | `requires_body_parts: [skin]` |
| `chameleon-skin.yaml` | `requires_body_parts: [skin]` |
| `bioluminescence.yaml` | `requires_body_parts: [skin]` |
| `blinding-flash.yaml` | `requires_body_parts: [skin]` |
| `photosynthetic-skin.yaml` | `requires_body_parts: [skin]` |

- [ ] **Step 2: Boot the server to verify validation passes**

Run: `go run . > /tmp/boot.log 2>&1 &`; wait ~15 seconds; check the log.

Expected: clean boot. Validation passes.

Kill the server.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/mutations/
git commit -m "feat(mutations): add requires_body_parts to 14 anatomically-gated mutations"
```

---

## Task 9: Create 4 new elemental species YAMLs (sand/storm/ice/smoke)

**Files:**
- Create: `_datafiles/world/dogmud/species/{NN}-sand_elemental.yaml`
- Create: `_datafiles/world/dogmud/species/{NN}-storm_elemental.yaml`
- Create: `_datafiles/world/dogmud/species/{NN}-ice_elemental.yaml`
- Create: `_datafiles/world/dogmud/species/{NN}-smoke_elemental.yaml`

ID allocation: run `python tools/id_inventory.py --type species` first to find the next free IDs. Per the existing range (top is 40 + 99), expect 41/42/43/44 as the next four.

Stat scaling notes (humans = 100 baseline): sand is sub-human total (~455, like water), storm is mid-tier (~550, faster but fragile), ice is boss-tier (~610, the queen), smoke is boss-tier (~570, the prince).

- [ ] **Step 1: Allocate IDs**

Run: `python tools/id_inventory.py --type species`
Take note of the next-free ID. Allocate four sequential IDs (e.g., 41-44).

- [ ] **Step 2: Create `41-sand_elemental.yaml`** (adjust ID per allocation)

```yaml
speciesid: 41
name: Sand Elemental
description: A swirling column of grit and sun-warmed sand, the sand
  elemental is what happens when desert wind and parched earth refuse
  to settle. It tears at flesh with abrasive whirls and chokes lungs
  with sweeping clouds that sting the eyes blind.
size: medium
unarmedname: abrasive whirl
selectable: false
tameable: false
angrycommands:
  - emote tightens its column, sand grains hissing like cicadas.
  - emote sends a stinging spray of grit ahead of itself.
stats:
  strength:
    base: 80
  dexterity:
    base: 130
  perception:
    base: 100
  vitality:
    base: 90
  willpower:
    base: 50
  charisma:
    base: 5
damage:
  basedamage: 3
  variance: 1
disabledslots: ['weapon', 'offhand', 'head', 'neck', 'back',
  'belt', 'gloves', 'ring', 'legs', 'feet']
body_parts: [skin]
intrinsic_mutations:
  incorporeal: 2
  blinding-spit: 1
```

- [ ] **Step 3: Create `42-storm_elemental.yaml`** (adjust ID)

```yaml
speciesid: 42
name: Storm Elemental
description: A roaring vortex of charged wind and crawling lightning,
  the storm elemental is the swiftest and most ferocious of the
  air-tribe elementals. It strikes with branching arcs that leap from
  one target to another and tears the breath from anyone who stands
  too long in its path.
size: large
unarmedname: arcing strike
selectable: false
tameable: false
grapple_immune: true
return_damage: 15
angrycommands:
  - emote spins faster, lightning crackling along its outer edge.
  - emote unleashes a deafening peal of thunder from its core.
stats:
  strength:
    base: 30
  dexterity:
    base: 220
  perception:
    base: 180
  vitality:
    base: 25
  willpower:
    base: 90
  charisma:
    base: 5
damage:
  basedamage: 3
  variance: 2
disabledslots: ['weapon', 'offhand', 'head', 'neck', 'back', 'body',
  'belt', 'gloves', 'ring', 'legs', 'feet']
body_parts: []
intrinsic_mutations:
  incorporeal: 4
  hasted: 1
```

- [ ] **Step 4: Create `43-ice_elemental.yaml`** (adjust ID)

```yaml
speciesid: 43
name: Ice Elemental
description: A slender humanoid figure of translucent crystal and
  flowing meltwater, the ice elemental moves with alien grace. Cold
  radiates from her presence in a way that aches in the bones, and
  her crystalline limbs strike with the brittle, splintering force
  of a glacier calving.
size: medium
unarmedname: crystal strike
selectable: false
tameable: false
angrycommands:
  - emote shifts her stance, frost spreading across the floor in
    a spreading ring.
  - emote draws breath, and the air around her crystallizes into
    razor flakes.
stats:
  strength:
    base: 110
  dexterity:
    base: 90
  perception:
    base: 100
  vitality:
    base: 180
  willpower:
    base: 120
  charisma:
    base: 30
damage:
  basedamage: 4
  variance: 2
disabledslots: ['weapon', 'offhand', 'head', 'neck', 'back',
  'belt', 'gloves', 'ring', 'legs', 'feet']
body_parts: [arms, legs, skin]
intrinsic_mutations:
  cold-blooded: 1
  magical-resistance: 1
```

- [ ] **Step 5: Create `44-smoke_elemental.yaml`** (adjust ID)

```yaml
speciesid: 44
name: Smoke Elemental
description: A lithe, shifting shape of smoke and flickering embers,
  the smoke elemental seems to exist slightly out of phase with
  reality. It moves too quickly for the eye to track cleanly and
  strikes from where it isn't, leaving stinging trails of cinder
  in its wake.
size: medium
unarmedname: ember lash
selectable: false
tameable: false
grapple_immune: true
return_damage: 20
angrycommands:
  - emote dissolves into a thicker plume, embers wheeling in
    aggressive spirals.
  - emote condenses briefly into solid form before scattering
    into smoke again.
stats:
  strength:
    base: 50
  dexterity:
    base: 200
  perception:
    base: 150
  vitality:
    base: 40
  willpower:
    base: 100
  charisma:
    base: 30
damage:
  basedamage: 4
  variance: 2
disabledslots: ['weapon', 'offhand', 'head', 'neck', 'back', 'body',
  'belt', 'gloves', 'ring', 'legs', 'feet']
body_parts: []
intrinsic_mutations:
  incorporeal: 4
  hasted: 1
  fast-reflexes: 1
```

- [ ] **Step 6: Boot the server to verify the 4 new species load**

Run: `go run . > /tmp/boot.log 2>&1 &`; wait ~15 seconds; check for the species load count incrementing by 4 (`grep -E "species.LoadDataFiles" /tmp/boot.log` should show the new count).

Kill the server.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/species/{41,42,43,44}-*.yaml
git commit -m "feat(species): add sand, storm, ice, smoke elemental species"
```

---

## Task 10: Mob YAML migration for planar_oasis + cleanup of redundant `mutations: { incorporeal: 4 }`

**Files:**
- Modify: 5 mob YAMLs in `_datafiles/world/dogmud/mobs/instance_planar_oasis/`
- Modify: Other mob YAMLs with redundant `mutations: { incorporeal: 4 }` overrides (find via grep)

- [ ] **Step 1: Repoint the 5 planar_oasis mob speciesids**

| File | Change |
|---|---|
| `318-sand_elemental.yaml` | `speciesid: 37` → `speciesid: 41` (sand) |
| `319-storm_elemental.yaml` | `speciesid: 37` → `speciesid: 42` (storm) |
| `320-elemental_king.yaml` | Keep `speciesid: 40` (magma); ADD `mutations:` block with `large: 1` at the same level as `character:` etc. |
| `321-elemental_queen.yaml` | `speciesid: 37` → `speciesid: 43` (ice); REMOVE the existing `mutations:` block (incorporeal: 4) entirely |
| `322-elemental_prince.yaml` | `speciesid: 37` → `speciesid: 44` (smoke) |

For king (320), the mutations block to add looks like:

```yaml
mutations:
  large: 1
```

(Insert it at the top level alongside `routine:`, `statpool:`, etc. — NOT inside `character:`.)

- [ ] **Step 2: Find and clean redundant `mutations: { incorporeal: 4 }` from other mobs**

Run:

```
grep -rln "incorporeal: 4" _datafiles/world/dogmud/mobs/ 2>/dev/null
```

Expected files (per chunk-2.2a's known set): wraith mobs, spectre mobs, fire elemental mobs, air elemental mobs. The queen was already removed in Step 1 above.

For each mob YAML that uses one of the species 32/33/38/39 (wraith/spectre/air/fire) AND has a `mutations: { incorporeal: 4 }` block, REMOVE the redundant mutations block. The species now provides incorporeal intrinsically.

Be careful: if a mob YAML has `mutations:` block with OTHER mutations besides incorporeal, only remove the incorporeal line, keep the others. If incorporeal is the only entry, remove the whole `mutations:` block.

- [ ] **Step 3: Boot the server + spot-check**

Run: `go run . > /tmp/boot.log 2>&1 &`; wait ~15 seconds; check the log.

Expected: clean boot. The mob load count is unchanged from before.

Kill the server.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "feat(mobs): repoint planar_oasis elementals + clean redundant incorporeal overrides"
```

---

## Task 11: Documentation — context.md files

**Files:**
- Modify: `internal/species/context.md`
- Modify: `internal/mutations/context.md`
- Modify: `internal/characters/context.md`

- [ ] **Step 1: Update `internal/species/context.md`**

Find the section documenting the `Species` struct fields. Add entries for `BodyParts` and `IntrinsicMutations`. Include:

- What each field represents
- The canonical body-parts vocabulary (the 7 tags)
- Fail-open semantics for absent `BodyParts`
- The two helper methods (`HasBodyPart`, `HasAllBodyParts`)
- Boot-time validation behavior (panic on unknown tag)
- Pointer to the chunk-2.5 spec for the full design

Example section to add:

```markdown
### Body Plan & Intrinsic Mutations (chunk 2.5)

Species declare anatomy via `BodyParts []string` and natural traits via
`IntrinsicMutations map[string]int`. The seven canonical body-part tags
are `arms`, `hands`, `legs`, `eyes`, `mouth`, `skin`, `tail` — see
`CanonicalBodyParts` in `species.go`.

- `BodyParts: nil` (absent in YAML) → fail-open. Every mutation passes
  the body-parts gate. Use for un-migrated species; legacy behavior.
- `BodyParts: []` (explicit empty in YAML) → no body parts. Every
  body-part-requiring mutation is gated. Use for incorporeal species.
- `BodyParts: [arms, hands, ...]` → declared anatomy. Mutations whose
  `RequiresBodyParts` is a subset of this list pass the gate.

`IntrinsicMutations` is merged additively into `Character.Mutations` at
character init via `Character.ApplyIntrinsicMutations(species)`. Caps
respected.

Boot-time validation (`ValidateBodyPartTags`) panics on unknown tags
or unknown mutation ids in intrinsic_mutations.

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`
```

- [ ] **Step 2: Update `internal/mutations/context.md`**

Add a section on the new gating mechanism + intrinsic stacking:

```markdown
### Body-Plan Gating (chunk 2.5)

`MutationSpec.RequiresBodyParts []string` lists canonical body-part tags
required for the mutation to apply. Empty = body-agnostic. The legacy
`RequiresArms bool` field has been REMOVED — migration replaced it with
`RequiresBodyParts: [arms]`.

Three gating sites:

1. **Random-roll pool:** `GetWeightedPool(current, species)` filters
   out mutations whose body-parts requirements aren't met by the
   species.
2. **Curated SpawnMutations path:** `mobs.go` checks each entry against
   the species; logs a warning + skips on mismatch.
3. **Mid-game grants:** The mutation potion / quest / admin path
   checks via `MutationSpec.CanApplyTo(*species.Species)` and rejects
   with player-facing flavor text on mismatch.

Boot-time validation panics on unknown body-part tags in any
mutation YAML.

### Intrinsic Mutation Stacking

Species's `IntrinsicMutations` map merges additively with acquired
mutations at character init via `Character.ApplyIntrinsicMutations`.
Cap-aware: combined rank clamped to the mutation's max rank
(default 4).

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`
```

- [ ] **Step 3: Update `internal/characters/context.md`**

Add a section on `ApplyIntrinsicMutations`:

```markdown
### Intrinsic Mutations (chunk 2.5)

`Character.ApplyIntrinsicMutations(species *species.Species)` merges
the species's intrinsic mutations additively into `Character.Mutations`.
No-op on nil species or empty intrinsic map. Cap-aware via the
mutation's max rank (default 4 for chunk-2.2a-compatible mutations).

Called once at character init AFTER all other mutation logic:
1. Curated SpawnMutations from mob YAML (or none for players)
2. Random-roll mutation acquisition (mobs only)
3. Persistent acquired mutations from save file (players only)
4. `ApplyIntrinsicMutations(species)` — this call

Stacks ADDITIVELY: a wolf species with `intrinsic_mutations: { tail: 1 }`
that also rolls `tail` rank 1 ends up with effective rank 2 in
`Character.Mutations`.

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`
```

- [ ] **Step 4: Commit**

```bash
git add internal/species/context.md internal/mutations/context.md internal/characters/context.md
git commit -m "docs: document body-plan gating and intrinsic mutations (chunk 2.5)"
```

---

## Task 12: Helpfile updates

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/mutations.template`
- Modify: `_datafiles/world/dogmud/templates/help/species.template`
- Modify: per-mutation help templates for the 17 affected mutations (only those that already have a per-mutation template — not all 17 may)

- [ ] **Step 1: Update the main mutations helpfile**

Open `_datafiles/world/dogmud/templates/help/mutations.template`. Add a paragraph (placed wherever fits the existing structure — likely after the "what is a mutation" intro and before the catalog listing) explaining body-plan gating:

```
Not every mutation is available to every species. A mutation may
REQUIRE certain anatomy — extra arms needs arms to grow from, sharp
claws need hands to grow on, keen eyes need eyes to sharpen. If your
species lacks the necessary body part, the mutation will refuse to
take hold. The flavor text will tell you when a body cannot integrate
a mutation.

Conversely, some species have INTRINSIC mutations — natural anatomy
that grants the mechanical effect of a mutation without needing to
roll for it. A wolf has an intrinsic tail mutation; a vampire has
intrinsic regenerative tissue. Acquired mutations stack additively
on top of intrinsic ones.
```

- [ ] **Step 2: Update the species helpfile**

Open `_datafiles/world/dogmud/templates/help/species.template`. Add a paragraph on body_parts and intrinsic_mutations (player-facing — keep it lore-flavored, not schema-heavy):

```
Each species has a distinct body plan that determines what mutations
can take hold. A wolf cannot grow extra arms; a snake cannot have
extra legs; an ice elemental's crystalline form cannot grow skin
that camouflages. Some species also have natural traits that
behave like baseline mutations — a wolf's tail, a vampire's
regenerative tissue, an ascended being's resistance to magic. These
intrinsic traits stack with any mutations the creature acquires.
```

- [ ] **Step 3: Update per-mutation templates that gain body-part requirements**

For each mutation that gains `requires_body_parts:`, find its help template (in `_datafiles/world/dogmud/templates/help/`) — the filename matches the mutation id (e.g., `extra-arms.template`, `tail.template`, `incorporeal.template`).

If a template exists for the mutation, add a "Requires:" line near the top, after the description. Example for `extra-arms.template`:

```
Requires: arms (a species with explicit grasping limbs)
```

For `clawed-hands.template`:

```
Requires: hands (fingered manipulators)
```

For `keen-eyes.template`, `night-vision.template`, `infrared-vision.template`:

```
Requires: eyes
```

For mutations with multiple requirements, list all: `Requires: arms, hands`.

If a per-mutation template does NOT exist for a given mutation, skip it (no template to update). The main `mutations.template` covers the general case.

- [ ] **Step 4: Boot the server + spot-check help renders**

Run: `go run . > /tmp/boot.log 2>&1 &`; wait ~15s. (Helpfiles are loaded at boot; if any has a template syntax error, watch for parse warnings.)

Kill the server.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/
git commit -m "docs(help): document body-plan gating in mutation + species helpfiles"
```

---

## Task 13: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Flip 2.5 to Done in the progress tracker**

Locate the progress-tracker table. Find the row:

```markdown
| 2.5 | Tactical | Mutations on mobs | L | — | Not started |
```

Change to:

```markdown
| 2.5 | Tactical | Mutations on mobs | L | — | Done |
```

- [ ] **Step 2: Update the roll-up line**

```markdown
**Roll-up:** 12 / 41 done • 0 in progress • 29 not started.
```

Change to:

```markdown
**Roll-up:** 13 / 41 done • 0 in progress • 28 not started.
```

- [ ] **Step 3: Update the 2.5 mini-brief**

Locate the chunk 2.5 mini-brief in the body of the document. Replace its `Status:` line with `**Status:** Done (2026-05-12) • **Size:** L` and append a Shipped section:

```markdown
- **Shipped:** Body-plan gating model — `Species.BodyParts []string` from a canonical seven-tag set (`arms, hands, legs, eyes, mouth, skin, tail`); `MutationSpec.RequiresBodyParts []string` replaces the old `RequiresArms bool`. Three gating sites updated: random-roll pool (`GetWeightedPool`), curated `SpawnMutations` path (latent bug fix — was applying unconditionally), and mid-game mutation grants. `Character.ApplyIntrinsicMutations(species)` merges species intrinsics additively into the character's mutation map at init time, cap-aware. Migration covered all 35 existing species + 4 new elemental species (sand, storm, ice, smoke). 17 mutation YAMLs gained `requires_body_parts:` declarations. 5 mob YAMLs in `instance_planar_oasis/` repointed: king kept on magma + added `mutations: { large: 1 }` override, queen moved to new ice species (dropping her chunk-2.2a `incorporeal: 4` override since her crystal/water form is corporeal), prince moved to new smoke species. Redundant `mutations: { incorporeal: 4 }` overrides on wraith/spectre/fire/air mobs cleaned up — incorporeal is now intrinsic on the species. Boot-time validation panics on unknown body-part tags or unknown mutation ids in intrinsic_mutations. Helpfiles updated to document body-plan gating in player-facing terms. Spec at `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`, plan at `docs/superpowers/plans/2026-05-12-mob-aliveness-2.5-mutations-on-mobs.md`.
```

- [ ] **Step 4: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 2.5 (mutations on mobs) as Done"
```

---

## Task 14: Smoke validation

**Files:**
- None modified — pure validation.

Per the kill-test-mud-servers SOP, do NOT just trust `go build`. Spin the server up and exercise the behaviors.

- [ ] **Step 1: Full build + full test**

Run: `go build ./...`
Expected: clean.

Run: `go test ./...`
Expected: no FAILs.

- [ ] **Step 2: Boot the server and watch the load lines**

Run: `go run . > /tmp/chunk25_smoke.log 2>&1 &`. Wait ~20 seconds.

Run: `grep -E "(species|mutations|behaviors).LoadDataFiles|Server Ready|panic" /tmp/chunk25_smoke.log`

Expected:
- `species.LoadDataFiles()` increments by 4 vs. pre-chunk baseline (4 new species)
- `mutations.LoadDataFiles()` count unchanged (no new mutations)
- `Server Ready` line appears
- No `panic:` lines

- [ ] **Step 3: Spot-check via admin commands** (or AI tester)

If you have an admin character or `/test-mud` access:

- **Wolf body-parts gate:** spawn a steppe wolf (mob 205) with `mutationchance: 100`. Inspect via `mob <inst> show mutations`. Confirm NO extra-arms entry across 10+ spawns. Confirm `tail: 1` (intrinsic) or `tail: 2` (intrinsic + acquired) shows in some.

- **Curated-skip warning:** temporarily add `mutations: { extra-arms: 1 }` to a wolf YAML (e.g., 205-steppe_wolf.yaml). Re-boot. Spawn. Verify warning in log:
  `"mutation requirements not met by species" mobId=205 mutation=extra-arms species=canine requires=[arms] species_body_parts=[legs eyes mouth skin tail]`
  Verify wolf does NOT have extra-arms equipped slots. REVERT the YAML change.

- **Intrinsic stacking:** spawn a wolf and admin-grant `tail` mutation. Confirm `Character.Mutations[tail] = 2`. Test that tailsweep trip-reskin fires in combat (drop the wolf in a room with a target, observe the trip animation text).

- **Elemental queen (corporeal):** spawn elemental queen (mob 321). Confirm via `mob <inst> show` that her species is now Ice Elemental (id 43) AND her mutation list has NO `incorporeal` entry. Confirm direct damage works on her — should NOT see the gear-effectiveness scaling that incorporeal applies.

- **Elemental prince (incorporeal-intrinsic):** spawn prince (mob 322). Confirm species is Smoke Elemental (id 44) AND mutation list shows `incorporeal: 4` (intrinsic from species). Give her a sword via admin; weapon damage contribution should be near-zero (gear-effectiveness = 0 at rank 4).

- **Sand elemental:** spawn sand elemental (mob 318). Confirm species is Sand Elemental (id 41) AND mutation list shows `incorporeal: 2` and `blinding-spit: 1`. Damage contribution from gear should be at ~50% (rank 2).

- **Storm elemental:** spawn storm elemental (mob 319). Confirm species is Storm Elemental (id 42), mutations show `incorporeal: 4` and `hasted: 1`.

- **Player human path:** as an existing or new human character, run `status` and check the prompt — mutation list should be unchanged from pre-chunk state (humans have no intrinsic mutations declared, so the player path is a no-op).

- [ ] **Step 4: Kill the test server cleanly**

```
ps -ef 2>/dev/null | grep -E '(go run|dogmud)' | grep -v grep | awk '{print $2}' | xargs -r kill -9
```

Verify with: `ps -ef | grep -E '(go run|dogmud)' | grep -v grep || echo "no servers"`

- [ ] **Step 5: Run the AI tester goal file (optional but recommended)**

If you write a goal file (`tools/testing/goals/chunk-2-5-mutations-smoke.yaml`), dispatch `/test-mud local feature-tester chunk-2-5-mutations-smoke.yaml`. Otherwise the manual spot-checks above are sufficient.

- [ ] **Step 6: Final commit log review**

```bash
git log --oneline -20
```

Expected: a clean sequence of small commits ending in the roadmap update.

---

## Out of scope (per spec)

- Body-type axis (living/constructed/elemental/undead)
- Size-based mutation gating
- Sentience-based gating
- Player species variety (humans only today)
- Mutation REMOVAL pipeline
- In-game body-parts inspection UI
- Visual / description authoring for the 4 new species' lore beyond minimal placeholders
