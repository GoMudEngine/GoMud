# Wave 6a — Ironhide Cluster Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the Ironhide cluster (the retaliation tank) — including the headline new mechanic, **Reflect Skin**, whose four flavors are chosen by the **moon phase at the moment of acquisition**.

**Architecture:** Ironhide reuses two already-built passives (`natural_armor`, `health_regen_multiplier`) for Thick Hide / Regrowth / Chitin Plating. The new mechanic is a **reflect** — a `reflect_damage` mutation effect that extends the existing untyped `return_damage` combat path. The four Reflect Skin variants conflict each other and each carries a `moon_flavor` bucket (1–4); the bloom candidate pool filters Reflect-Skin variants down to the one matching the **current moon bucket**, so which flavor you get depends on the sky when the Chrysalis reshapes you. Per the banked design: **no elemental damage-type system** — all four reflect the same % of damage; the flavor is cosmetic + a light rider (heavy riders — burn DoT / slow / shock-chain — and control-immunity + Resonant Larynx's shout-stacking are deferred, noted).

**Tech Stack:** Go. Packages: `internal/mutations`, `internal/gametime`, `internal/characters` (bloom), `internal/hooks` (combat return-damage), plus mutation/help YAML.

**Spec:** content-spec §3 Ironhide; banked moon-phase decision (memory: Reflect Skin ×4 = moon-phase-at-acquisition, flavor+rider not type).

---

## Verified context (do not re-discover)

- **Existing reflect path** (`internal/hooks/NewRound_DoCombat_unified.go:359`): `returnPct := defChar.StatMod("return_damage"); if sp := species...; returnPct += sp.ReturnDamage; if returnPct > 0 { returnDmg = DamageToTarget * returnPct/100; atkChar.Health -= returnDmg; emitReturnDamageText(...) }`. Add a mutation term here.
- **Mutation effect readers:** `sumEffects(owned, type, target)` + `GetX`; flags via `HasMutationFlag`; `DescribeEffect`/`flagPhrase` need a case per new type. `GetNaturalArmor` (natural_armor) + `GetHealthRegenMultiplier` (health_regen_multiplier) already exist and are consumed.
- **MutationSpec** (`internal/mutations/mutations.go`): has `Rarity`, `Conflicts []string` (yaml `conflicts`, used by legacy mutations; `HasConflict` checks both directions), `Clusters`, `Pole`. **Add a `MoonFlavor int` field** (yaml `moon_flavor,omitempty`) — 0 = always eligible, 1–4 = only offered under that moon bucket.
- **Bloom grant site** (`internal/characters/bloom_mutation.go:73` `BloomSeedNewMutation`): builds a `candidates` slice, skipping owned / `HasConflict` / `!CanApplyTo`. **Add the moon filter here** (skip `spec.MoonFlavor != 0 && spec.MoonFlavor != currentBucket`).
- **Moon API** (`internal/gametime/moonphase.go`): `GetEyePhase()`, `GetWandererPhase()`, `GetSwiftmoonPhase()` → each 0.0 (new) … 1.0 (full). Use two moons' fullness for a 1–4 bucket.
- **Boot:** `ValidateGraph` panics on unknown cluster (`ironhide` IS in KnownClusters) / missing prereq / (add: unknown is fine for MoonFlavor). Help template per mutation required (`TestHelpFileCompleteness_Mutations`).

---

## File Structure
- `internal/mutations/mutations.go` — add `MoonFlavor` field to MutationSpec.
- `internal/mutations/ironhide.go` (+ `_test.go`) — `GetReflectDamage` reader.
- `internal/mutations/describe.go` — `reflect_damage` case.
- `internal/gametime/moonphase.go` (+ `_test.go`) — `CurrentMoonFlavorBucket() int` (1–4).
- `internal/characters/bloom_mutation.go` (+ `_test.go`) — moon filter in `BloomSeedNewMutation`.
- `internal/hooks/NewRound_DoCombat_unified.go` — add the mutation reflect term.
- `_datafiles/world/dogmud/mutations/` — thick-hide, reflect-skin-barbed/molten/frostbite/voltaic, regrowth, living-carapace, chitin-plating (+ help templates).

---

### Task 1: `MoonFlavor` field + moon-bucket helper

**Files:** `internal/mutations/mutations.go`; `internal/gametime/moonphase.go`, `_test.go`

- [ ] **Step 1:** Add to `MutationSpec` (near `Conflicts`):

```go
	// MoonFlavor gates a mutation to a moon "bucket" (1–4): the bloom pool only
	// offers it when the current moon bucket matches. 0 = always eligible. Used
	// by the Reflect Skin family so the moon at acquisition picks the flavor.
	MoonFlavor int `yaml:"moon_flavor,omitempty"`
```

- [ ] **Step 2: Failing test** for `CurrentMoonFlavorBucket` (gametime) — returns 1–4 deterministically from the moon phases. Since moon phase depends on round count (which advances), test the pure mapping helper instead:

```go
func TestMoonFlavorBucketMapping(t *testing.T) {
	// bucket = 1 + (eyeFull?2:0) + (wanderFull?1:0)
	if got := moonFlavorBucket(0.2, 0.2); got != 1 { t.Fatalf("new/new -> 1, got %d", got) }
	if got := moonFlavorBucket(0.2, 0.9); got != 2 { t.Fatalf("new/full -> 2, got %d", got) }
	if got := moonFlavorBucket(0.9, 0.2); got != 3 { t.Fatalf("full/new -> 3, got %d", got) }
	if got := moonFlavorBucket(0.9, 0.9); got != 4 { t.Fatalf("full/full -> 4, got %d", got) }
}
```

- [ ] **Step 3: Implement** (`moonphase.go`):

```go
// moonFlavorBucket maps two moons' fullness (0..1, >=0.5 = "full") to a 1–4
// bucket. Pure for testability.
func moonFlavorBucket(eye, wander float64) int {
	b := 1
	if eye >= 0.5 { b += 2 }
	if wander >= 0.5 { b += 1 }
	return b
}

// CurrentMoonFlavorBucket returns the current 1–4 moon bucket (from the Eye and
// Wanderer), used to pick which Reflect Skin flavor the Chrysalis grants.
func CurrentMoonFlavorBucket() int {
	return moonFlavorBucket(GetEyePhase(), GetWandererPhase())
}
```

- [ ] **Step 4: verify passes. Step 5: commit** `feat(ironhide): MoonFlavor field + moon-bucket helper`.

---

### Task 2: Reflect-damage effect reader + combat consumer

**Files:** `internal/mutations/ironhide.go`, `_test.go`; `internal/mutations/describe.go`; `internal/hooks/NewRound_DoCombat_unified.go`

- [ ] **Step 1: Failing test** — seed a mutation with `reflect_damage: 20`; assert `GetReflectDamage(owned)` returns 20.

- [ ] **Step 2-3: Implement** (`ironhide.go`):

```go
package mutations

// GetReflectDamage returns the net percent of incoming damage an owner reflects
// back at melee attackers (Reflect Skin / Living Carapace). Feeds the combat
// return-damage path alongside species/equipment return_damage.
func GetReflectDamage(owned map[string]int) float64 {
	return sumEffects(owned, "reflect_damage", "")
}
```

Add `DescribeEffect` case: `case "reflect_damage": return "A share of the harm done to you lashes back into whoever struck you."`

Wire into `NewRound_DoCombat_unified.go` (after the species term):

```go
	returnPct := defChar.StatMod("return_damage")
	if sp := species.GetSpecies(defChar.SpeciesId); sp != nil {
		returnPct += sp.ReturnDamage
	}
	returnPct += int(mutations.GetReflectDamage(defChar.Mutations)) // Ironhide Reflect Skin
```

> Confirm `returnPct` is an `int` (species return is int); `GetReflectDamage` is float — cast. Add the `mutations` import if absent.

- [ ] **Step 4: build + test** (`./internal/mutations/`, `./internal/hooks/`). **Step 5: commit** `feat(ironhide): reflect_damage effect + combat consumer`.

---

### Task 3: Moon filter in the bloom pool

**Files:** `internal/characters/bloom_mutation.go`, `_test.go`

- [ ] **Step 1:** In `BloomSeedNewMutation`'s candidate loop, after the conflict/CanApplyTo checks, add:

```go
		// Moon-gated flavor (Reflect Skin): only the variant matching the
		// current moon bucket is eligible right now.
		if spec.MoonFlavor != 0 && spec.MoonFlavor != gametime.CurrentMoonFlavorBucket() {
			continue
		}
```

> Add the `gametime` import. Check for an import cycle (characters → gametime); if gametime imports characters this won't compile — if so, pass the bucket in or read it via an injected function. Verify before committing.

- [ ] **Step 2: Test** — seed two moon-gated specs (MoonFlavor 1 and 2) + a normal one; with the current bucket stubbed/known, assert `BloomSeedNewMutation` never returns the non-matching flavor. (If the bucket can't be stubbed without a clock, test the filter predicate in isolation.)

- [ ] **Step 3: commit** `feat(ironhide): bloom offers only the moon-matching Reflect Skin flavor`.

---

### Task 4: Reflect Skin ×4 variants

**Files:** `_datafiles/world/dogmud/mutations/reflect-skin-{barbed,molten,frostbite,voltaic}.yaml` + 4 help templates

- [ ] Author all four: `clusters: [ironhide]`, `pole: body`, rarity 5, each `moon_flavor: 1|2|3|4`, each `conflicts: [<the other three>]`, each `reflect_damage: N` + a light flavor rider via existing effect types (MVP — heavy riders deferred):
  - **Barbed** (moon_flavor 1): highest pure reflect (e.g. `reflect_damage: 25`).
  - **Molten** (2): `reflect_damage: 18` + a small `magical_mitigation` statmod-flavored ward ("runs hot, shrugs cold").
  - **Frostbite** (3): `reflect_damage: 18` + a small `natural_armor` ("frost-rimed hide").
  - **Voltaic** (4): `reflect_damage: 18` + a small `dodge_modifier` ("charged reflexes").
- [ ] 4 help templates (80-col, number-free; each names its flavor + that the moon decides which you get). **Boot smoke** (conflicts + moon_flavor load; no panic). **Commit** `content(ironhide): Reflect Skin — four moon-chosen flavors`.

> Riders are first-pass differentiators using built effect types; the spec's burn-DoT / cold-slow / shock-chain riders are deferred to a polish pass.

---

### Task 5: Thick Hide + Regrowth + Chitin Plating — YAML

**Files:** `_datafiles/world/dogmud/mutations/{thick-hide,regrowth,chitin-plating}.yaml` + help

- [ ] **Thick Hide** (entry, rarity 3) — `clusters: [ironhide]`, `pole: body`, `natural_armor: N`.
- [ ] **Regrowth** (core, rarity 6, prereq Thick Hide) — `clusters: [ironhide]`, `pole: body`, `health_regen_multiplier: N` (heal mid-fight). Cleanse ("shrug off crippling") deferred.
- [ ] **Chitin Plating** (bridge, rarity 6) — `clusters: [colossus, ironhide]`, `pole: body`, `natural_armor: N` + a `stat_flat strength` bump; note lower-mobility flavor (a `dodge_modifier` penalty) in text.
- [ ] Help templates. **Commit** `content(ironhide): Thick Hide + Regrowth + Chitin Plating bridge`.

---

### Task 6: Living Carapace apex

**Files:** `_datafiles/world/dogmud/mutations/living-carapace.yaml` + help

- [ ] Apex — `clusters: [ironhide]`, `pole: body`, rarity 9, prereq `thick-hide` + `regrowth`. Bundle: big `natural_armor` + a large `reflect_damage` (the "amplifies your Reflect Skin" — MVP: stacks additional reflect on top of the variant) + `aggro_magnet` (pulls aggro). **Immovable/control-immunity deferred** (shared new mechanic with Ossified Frame — build in a later mechanics chunk; note in the help as "coming").
- [ ] Help template. **Boot smoke** (apex prereqs). **Commit** `content(ironhide): Living Carapace apex`.

---

### Task 7: Coverage + suite + patch notes + boot

- [ ] `go test ./... -run 'TestHelpFileCompleteness_Mutations|TestDescribeEffect'` — fix gaps (8 new mutations → 8 templates).
- [ ] `go test ./...` — exit 0.
- [ ] `PATCH_NOTES.md` — dated, player-facing: the Ironhide path — a hardened hide, a retaliating skin whose *nature is set by the moons the night it wakes*, flesh that knits mid-fight, and a plated apex that punishes anything that strikes it.
- [ ] Nuke instances, full boot smoke (clean load, mapper errors=0, Server Ready, no panic).
- [ ] **Commit** `docs(ironhide): patch notes` + any fixes.

---

## Out of scope / follow-on
- **Heavy Reflect-Skin riders** (Molten burn-DoT, Frostbite slow, Voltaic shock-chain) — polish pass; MVP differentiates by reflect% + a light built-effect rider.
- **Control-immunity / "immovable"** (Living Carapace + Colossus's Ossified Frame) — a shared new mechanic, its own chunk.
- **Resonant Larynx** bridge (⇄Zealot) — needs the shout-stacking mechanic; deferred to the Zealot/shout chunk.
- **Regrowth cleanse** ("shrug off crippling") — deferred.
- **Per-rank magnitudes** — Wave-6 playtest.
