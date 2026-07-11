# Chrysalis Content — Wave 1: passive-effect recipe + observable-drift proof slice

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the reusable "new passive mutation effect" recipe (one numeric example, `spell_power`, end-to-end), and author a proof slice of tagged mutations so cluster-affinity drift becomes real across Colossus / Stalker / Ethereal / Center.

**Architecture:** Numeric passive effects need no new plumbing in `internal/mutations` — `sumEffects` already totals any effect-type string — so a new type is just a named helper wrapper + one consumer call in the relevant subsystem. Flag passives reuse the existing flag system (`GetMutationFlags`). This wave adds `spell_power` (helper + consumer in the shared spell-damage function) and authors six mutations that use `spell_power`, existing numeric types, and a reused `see-hidden` flag — all carrying `clusters`/`pole` so the merged engine's `GetGraphPool` gates them by drift.

**Tech Stack:** Go, `internal/mutations`, `internal/hooks` (spell damage), YAML mutation data files, testify.

**Spec:** `docs/superpowers/specs/2026-07-11-chrysalis-cluster-content-design.md` (this is Wave 1 of §9).

**Scope note — deferred to later waves (same recipe):** the multi-site consumers — control-immunity (trip/grapple/knockback gates), typed reflect-damage (on-hit-taken), crit-resist, detection/stealth scores, shout-amp/stacking, and the aura sources (Dissonance/Presence) — are NOT in Wave 1. Wave 1 proves the recipe with one clean single-site numeric consumer and ships observable drift. On-hit buffs, verb-enhance, states, transformations, auras, companions, actives, and flight are Waves 2–5.

**Interim coexistence:** the old 41 mutations stay in place (untagged → treated as universal/always-eligible by the engine) until the Wave 6 clean-break migration. New files use ids that do **not** collide with old ones.

---

## File Structure

**Create (mutation YAML — `_datafiles/world/dogmud/mutations/`):**
- `ether-gland.yaml` — Ethereal, `spell_power` (the new numeric type)
- `titan-growth.yaml` — Colossus, existing `health_multiplier` + `dodge_modifier`
- `compound-eyes.yaml` — Stalker, reused `flag: see-hidden` + `dodge_modifier`
- `rapid-healing.yaml` — Center (universal), existing `health_regen_multiplier`
- `precognition.yaml` — Center (universal), existing `dodge_modifier`
- `spiracle-lungs.yaml` — Center (universal), existing `stamina_regen_multiplier`

**Modify:**
- `internal/mutations/mutations.go` — add `GetSpellPowerMultiplier` helper (near the other `Get*Multiplier` helpers)
- `internal/hooks/combat_shared_helpers.go` — apply it in `calcSpellDamageForCharacter`
- Tests: `internal/mutations/mutations_test.go` (helper), `internal/hooks/*_test.go` or a new `spell_power_test.go` (consumer)

---

## Phase 1 — The `spell_power` numeric-passive recipe

### Task 1: `GetSpellPowerMultiplier` helper

**Files:**
- Modify: `internal/mutations/mutations.go`
- Test: `internal/mutations/mutations_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/mutations/mutations_test.go (append)
func TestGetSpellPowerMultiplier(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"ether-gland": {MutationId: "ether-gland", Name: "Ether Gland", Rarity: 4,
			Pros: []MutationEffect{{Type: "spell_power", Value: 0.5}}},
	})
	defer cleanup()
	// level 1 → 1.0× → 0.5
	if got := GetSpellPowerMultiplier(map[string]int{"ether-gland": 1}); got != 0.5 {
		t.Fatalf("spell_power L1 = %v, want 0.5", got)
	}
	if got := GetSpellPowerMultiplier(map[string]int{}); got != 0 {
		t.Fatalf("no mutation → 0, got %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mutations/ -run TestGetSpellPowerMultiplier -v`
Expected: FAIL — `GetSpellPowerMultiplier` undefined.

- [ ] **Step 3: Implement**

In `internal/mutations/mutations.go`, add near `GetDamageMultiplier` (~line 560):

```go
// GetSpellPowerMultiplier returns the net spell_power bonus from mutations
// (Ether Gland, Corvid Brain, …). Apply as:
//   dmg = dmg * (1.0 + GetSpellPowerMultiplier(m))
func GetSpellPowerMultiplier(owned map[string]int) float64 {
	return sumEffects(owned, "spell_power", "")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/mutations/ -run TestGetSpellPowerMultiplier -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/mutations_test.go
git commit -m "feat(mutations): GetSpellPowerMultiplier helper (spell_power effect type)"
```

---

### Task 2: Consumer — amplify caster spell damage

**Files:**
- Modify: `internal/hooks/combat_shared_helpers.go:40` (inside `calcSpellDamageForCharacter`)
- Test: `internal/hooks/spell_power_test.go` (new)

- [ ] **Step 1: Write the failing test**

```go
// internal/hooks/spell_power_test.go (new file)
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

func TestCalcSpellDamage_SpellPowerAmplifies(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"ether-gland": {MutationId: "ether-gland", Name: "Ether Gland", Rarity: 4,
			Pros: []mutations.MutationEffect{{Type: "spell_power", Value: 0.5}}},
	})
	defer cleanup()

	spell := &spells.SpellData{Name: "Test Bolt", DamageMultiplier: 1.0, TargetDefenseType: "magical"}
	caster := &characters.Character{}
	caster.Stats.Willpower.ValueAdj = 100
	caster.Conviction, caster.ConvictionMax.Value = 100, 100
	target := &characters.Character{}
	target.HealthMax.Value = 1000

	// calcSpellDamageForCharacter applies dice.RollStat variance, so average
	// over many rolls to compare means deterministically.
	avg := func(c *characters.Character) int {
		total := 0
		for i := 0; i < 300; i++ {
			total += calcSpellDamageForCharacter(spell, c, target, 0, false)
		}
		return total
	}
	base := avg(caster)
	caster.Mutations = map[string]int{"ether-gland": 1}
	boosted := avg(caster)

	if !(boosted > base) {
		t.Fatalf("spell_power should raise average spell damage: base=%d boosted=%d", base, boosted)
	}
}
```

> Note: `calcSpellDamageForCharacter` rolls `dice.RollStat` (variance), so the test averages over 300 rolls — with a 50% spell_power boost the means separate cleanly. Mirror the caster/target construction the sibling spell tests in `internal/hooks` use if they factor a helper; the fields set above are the minimum the function reads (Willpower, Conviction/Max; target mitigation defaults to 0).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks/ -run TestCalcSpellDamage_SpellPowerAmplifies -v`
Expected: FAIL — boosted == base (spell_power not applied).

- [ ] **Step 3: Implement the consumer**

In `internal/hooks/combat_shared_helpers.go`, inside `calcSpellDamageForCharacter`, immediately after the `rawDmg := combat.CalcRawDamage(...)` line (~40):

```go
		// Mutation graph: spell-power passives (Ether Gland, Corvid Brain) amplify caster output.
		if spBonus := mutations.GetSpellPowerMultiplier(caster.Mutations); spBonus != 0 {
			rawDmg *= 1.0 + spBonus
		}
```

(`mutations` is already imported in this file.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hooks/ -run TestCalcSpellDamage_SpellPowerAmplifies -v`
Expected: PASS

- [ ] **Step 5: Run the hooks suite (no regressions)**

Run: `go test ./internal/hooks/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/combat_shared_helpers.go internal/hooks/spell_power_test.go
git commit -m "feat(hooks): spell_power mutations amplify caster spell damage"
```

---

## Phase 2 — Author the proof-slice mutations

Each is a `Write` of a new YAML file. Rank-1 magnitudes are **first-pass and
tunable** (per-rank curves are a balance-wave item per the content spec). After
Phase 2, verify all load at boot in Phase 3.

### Task 3: `ether-gland.yaml` (Ethereal — the spell_power carrier)

**Files:** Create `_datafiles/world/dogmud/mutations/ether-gland.yaml`

- [ ] **Step 1: Write the file**

```yaml
mutationid: ether-gland
name: Ether Gland
description: |
  New organs bud along your spine and throat, rendering raw belief into
  force. Casting comes easier and strikes harder — the world bends a little
  further to what you will it to be.
rarity: 4
clusters: [ethereal]
pole: belief
visual: A faint violet luminescence pulses at the hollow of their throat when they concentrate.
pros:
  - type: spell_power
    value: 0.5
```

- [ ] **Step 2: Commit** (batched with the rest of Phase 2 after Task 8, or per-file; per-file is fine)

```bash
git add _datafiles/world/dogmud/mutations/ether-gland.yaml
git commit -m "content(mutations): ether-gland (Ethereal spell_power keystone)"
```

### Task 4: `titan-growth.yaml` (Colossus — existing types)

**Files:** Create `_datafiles/world/dogmud/mutations/titan-growth.yaml`

- [ ] **Step 1: Write the file**

```yaml
mutationid: titan-growth
name: Titan Growth
description: |
  Your frame swells past any natural bound — bones lengthen, mass piles on.
  You tower, and there is simply more of you to endure a fight. But a target
  this large is hard to miss and slow to duck a blow.
rarity: 7
clusters: [colossus]
pole: body
conflicts:
  - hollow-bones
visual: They loom a full head taller than they should, shoulders blotting out the light behind them.
pros:
  - type: health_multiplier
    value: 0.20
cons:
  - type: dodge_modifier
    value: -15
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mutations/titan-growth.yaml
git commit -m "content(mutations): titan-growth (Colossus size keystone)"
```

### Task 5: `compound-eyes.yaml` (Stalker — reused see-hidden flag)

**Files:** Create `_datafiles/world/dogmud/mutations/compound-eyes.yaml`

- [ ] **Step 1: Write the file**

```yaml
mutationid: compound-eyes
name: Compound Eyes
description: |
  Your eyes facet into hundreds of glinting lenses. Nothing crosses your
  field of view unseen and nothing flanks you — though a sudden bright light
  is a blaze of confusion.
rarity: 4
clusters: [stalker]
pole: body
requires_body_parts: [eyes]
visual: Their eyes are faceted and iridescent, catching light in a hundred tiny glints.
pros:
  - type: flag
    target: see-hidden
    value: 1
  - type: dodge_modifier
    value: 10
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mutations/compound-eyes.yaml
git commit -m "content(mutations): compound-eyes (Stalker see-hidden keystone)"
```

### Task 6: `rapid-healing.yaml` (Center — universal)

**Files:** Create `_datafiles/world/dogmud/mutations/rapid-healing.yaml`

- [ ] **Step 1: Write the file** (no `clusters` → universal, always eligible; `pole: ""` via omission)

```yaml
mutationid: rapid-healing
name: Rapid Healing
description: |
  Your flesh knits with unnatural speed. Rest a moment and wounds close that
  should have scarred for weeks.
rarity: 3
visual: Old cuts on their skin are already pink and closing, faster than they ought to be.
pros:
  - type: health_regen_multiplier
    value: 0.15
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mutations/rapid-healing.yaml
git commit -m "content(mutations): rapid-healing (Center universal enabler)"
```

### Task 7: `precognition.yaml` (Center — universal)

**Files:** Create `_datafiles/world/dogmud/mutations/precognition.yaml`

- [ ] **Step 1: Write the file**

```yaml
mutationid: precognition
name: Precognition
description: |
  A half-second of the future bleeds into your present. You flinch from blows
  before they are thrown.
rarity: 4
visual: Their eyes track things a moment before those things happen.
pros:
  - type: dodge_modifier
    value: 15
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mutations/precognition.yaml
git commit -m "content(mutations): precognition (Center universal enabler)"
```

### Task 8: `spiracle-lungs.yaml` (Center — universal)

**Files:** Create `_datafiles/world/dogmud/mutations/spiracle-lungs.yaml`

- [ ] **Step 1: Write the file**

```yaml
mutationid: spiracle-lungs
name: Spiracle Lungs
description: |
  Your breathing reorganizes into a lattice of insectile spiracles — tireless,
  and indifferent to bad air. You could hold a breath for the better part of an
  hour.
rarity: 4
visual: Rows of tiny breathing-slits flex along their ribs.
pros:
  - type: stamina_regen_multiplier
    value: 0.15
```

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/mutations/spiracle-lungs.yaml
git commit -m "content(mutations): spiracle-lungs (Center universal enabler)"
```

---

## Phase 3 — Boot validation & drift eligibility

### Task 9: Boot-load validation (real data through ValidateGraph)

**Files:** Test: `internal/mutations/mutations_test.go` (or a new load test) — plus the SOP boot smoke.

- [ ] **Step 1: Add a real-data load test**

```go
// internal/mutations/mutations_test.go (append)
// Guards that the shipped mutation YAML (including the new graph-tagged files)
// loads and passes ValidateGraph without panicking.
func TestRealMutationData_ValidatesGraph(t *testing.T) {
	LoadMutationFiles() // loads _datafiles/world/dogmud/mutations
	if len(GetAll()) == 0 {
		t.Skip("no data files in this environment")
	}
	ValidateGraph()          // panics on bad cluster/pole/prereq
	ValidateBodyPartTags()   // panics on bad body-part tags
	// Spot-check a new file parsed its graph fields.
	if spec := GetMutation("ether-gland"); spec == nil || len(spec.Clusters) == 0 || spec.Clusters[0] != "ethereal" {
		t.Fatalf("ether-gland did not load with clusters:[ethereal]: %+v", spec)
	}
}
```

> `LoadMutationFiles` reads from the configured data path; this test runs from the repo root where that path resolves. If the environment can't resolve data files, the `Skip` keeps CI green.

- [ ] **Step 2: Run it**

Run: `go test ./internal/mutations/ -run TestRealMutationData_ValidatesGraph -v`
Expected: PASS (loads, validates, ether-gland has clusters:[ethereal]).

- [ ] **Step 3: Commit**

```bash
git add internal/mutations/mutations_test.go
git commit -m "test(mutations): real mutation data passes ValidateGraph with new content"
```

---

### Task 10: Drift-eligibility test (proof that tags gate acquisition)

**Files:** Test: `internal/mutations/affinity_test.go` (append)

- [ ] **Step 1: Write the test**

```go
// internal/mutations/affinity_test.go (append)
// Proves the proof-slice tags actually gate acquisition by cluster affinity.
func TestGraphPool_ProofSliceGating(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"ether-gland":   {MutationId: "ether-gland", Name: "Ether Gland", Rarity: 4, Clusters: []string{"ethereal"}, Pole: "belief"},
		"titan-growth":  {MutationId: "titan-growth", Name: "Titan Growth", Rarity: 7, Clusters: []string{"colossus"}, Pole: "body"},
		"rapid-healing": {MutationId: "rapid-healing", Name: "Rapid Healing", Rarity: 3}, // universal
	})
	defer cleanup()

	// No affinity → only the universal enabler is eligible.
	cold := GetGraphPool(map[string]int{}, map[string]float64{}, nil)
	if !contains(cold, "rapid-healing") {
		t.Fatal("universal rapid-healing must always be eligible")
	}
	if contains(cold, "ether-gland") || contains(cold, "titan-growth") {
		t.Fatal("clustered mutations must be gated out with zero affinity")
	}

	// A caster (ethereal affinity ≥ rarity 4) unlocks ether-gland but not titan-growth.
	caster := GetGraphPool(map[string]int{}, map[string]float64{"ethereal": 6}, nil)
	if !contains(caster, "ether-gland") {
		t.Fatal("ethereal affinity should unlock ether-gland")
	}
	if contains(caster, "titan-growth") {
		t.Fatal("ethereal affinity must NOT unlock the colossus keystone")
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/mutations/ -run TestGraphPool_ProofSliceGating -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/mutations/affinity_test.go
git commit -m "test(mutations): proof-slice mutations gate by cluster affinity"
```

---

### Task 11: Full build + suite + boot smoke

**Files:** none (verification).

- [ ] **Step 1: Build + affected suites**

Run: `go build ./... && go test ./internal/mutations/... ./internal/hooks/...`
Expected: build clean, all PASS.

- [ ] **Step 2: SOP boot smoke** (per repo SOP — nuke instance saves, boot, watch for panics)

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -mE1 "mutations.LoadMutationFiles|panic:"
```
Expected: `mutations.LoadMutationFiles() loadedCount=…` prints (count is old-41 + 6 new = 47), **no panic**. Ctrl-C after boot reaches world load.

- [ ] **Step 3: Commit** (nothing to commit if clean; otherwise fix + commit)

---

## Self-Review (completed during authoring)

- **Spec coverage:** implements Wave 1 of §9 — the passive-effect recipe (numeric `spell_power` end-to-end) + observable-drift proof slice across Colossus/Stalker/Ethereal/Center. Explicitly defers the multi-site consumers (control-immunity, reflect, crit-resist, detection, shout, auras) and all non-passive primitives (P2–P9) to later waves, matching the spec's decomposition.
- **Placeholder scan:** none — every YAML and code step is complete. Rank-1 magnitudes are real values, flagged first-pass/tunable per the spec's deferred-magnitude note.
- **Type consistency:** `GetSpellPowerMultiplier(owned map[string]int) float64` used identically in helper, consumer, and tests; `spell_power` effect-type string consistent across YAML + helper; `see-hidden` flag reuses the existing consumer (no new code); Center enablers authored **without** `clusters` (zero-cluster = universal/always-eligible), which is the correct engine behavior for the "everyone starts here" hub — a clarification of the spec's `clusters:[generalist]` note (a `generalist`-tagged mutation would be unreachable since no signal feeds `generalist` affinity).

## Follow-on (Wave 1b and beyond)

- **Wave 1b — remaining passive types** (same recipe): control-immunity (flag + trip/grapple/knockback consumers), typed reflect-damage (on-hit-taken consumer), crit-resist, detection/stealth scores, shout-amp. Each = helper/flag + consumer + author its mutations.
- **Spec clarification to fold in:** update the content spec §5 to note Center enablers are authored zero-cluster (universal), not `clusters:[generalist]`.
- Waves 2–6 per the content spec §9.
