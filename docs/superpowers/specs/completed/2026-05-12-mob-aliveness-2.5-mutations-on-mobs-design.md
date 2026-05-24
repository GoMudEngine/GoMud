# Mob Aliveness 2.5 — Mutations on Mobs (Body-Plan Gating + Intrinsic Mutations)

> **Phase 2 tactical (fifth chunk).** Close the mob-mutation parity
> gap left from chunk 2.2a. Introduce a body-plan gating model
> (`body_parts` on species × `requires_body_parts` on mutations) so
> mutations only apply to species whose anatomy supports them.
> Add `intrinsic_mutations` to species so natural anatomy (a wolf's
> tail, a vampire's regenerative tissue) is expressed in the same
> system as acquired mutations and stacks additively. Full migration
> pass on all 35 existing species + 4 new elemental species (sand,
> storm, ice, smoke).

## Goal

A wolf can never roll `extra-arms` because wolves don't have arms.
A wolf that already has a tail gets the tail mutation's mechanical
effects (tail slot, tailsweep trip-reskin) intrinsically, without
needing to roll it. A wolf that DOES roll `tail` gets *another*
tail (rank 2 effects). Stone golems can still ooze healing-gel
because that's an interesting flavor choice we're not gating
against.

Concretely: replace the single `RequiresArms bool` gate on
mutations with a generalized `requires_body_parts []string`
mechanism, declare what body parts each species has, and let
species declare natural anatomy as `intrinsic_mutations` that
merge additively with acquired mutations at character init.

## Architectural musts

Brainstorming refined the framing:

1. **Body-part vocabulary is a fixed canonical set of seven tags.**
   `arms`, `hands`, `legs`, `eyes`, `mouth`, `skin`, `tail`. Authors
   pick from this set — boot-time validation panics on unknown
   tags. The set is exhaustive enough for the existing 42-mutation
   catalog (17 mutations declare a requirement; the other 25 are
   body-agnostic — skin and mouth families each have multiple
   members, eyes has three, arms has two).

2. **One gating axis in v1: body-parts.** Body-type (living vs.
   constructed vs. elemental), size-based gating, and sentience
   gating are explicitly out of scope. Existing `conflicts:` lists
   on mutation YAMLs continue to do their job in parallel. Size
   mutations (`large`/`small`) are not gated by current species
   size — that's a known weakness flagged as a follow-up.

3. **Species declare `body_parts`; absence = fail-open.** A species
   YAML without a `body_parts:` field is treated as "has all
   seven" — every mutation passes the gate. This makes the
   migration safe and gradual; species that haven't been tagged
   yet keep working as today. The migration pass in this chunk
   tags ALL 35 existing species + 4 new ones, but the fail-open
   default is the long-term semantics for any future-added species
   that gets forgotten.

4. **Mutations declare `requires_body_parts` (a list).** A
   mutation with an empty/missing list is body-agnostic. Most
   mutations declare nothing; the 13 anatomically-specific ones
   declare one or two tags. Validation: unknown tag → boot-time
   panic.

5. **Migration: `RequiresArms bool` → `RequiresBodyParts [arms]`.**
   The existing single-bool gate becomes a special case of the
   new list. WITHIN THIS CHUNK: the `MutationSpec.RequiresArms`
   Go field is removed, all consumers updated to read
   `RequiresBodyParts`, and any mutation YAML that previously
   set `requires_arms: true` is rewritten to
   `requires_body_parts: [arms]`. The proxy logic that read
   `Species.DisabledSlots` for the arms check is replaced by
   reading `Species.BodyParts` directly. Net: one Go field
   removed, one Go field added, N YAMLs rewritten — all
   landed together so no transitional state exists post-chunk.

6. **`intrinsic_mutations` on species; additive stacking.**
   Species can declare `intrinsic_mutations: { id: rank }` — a
   map from mutation id to baseline rank. At character init
   (mob spawn or player creation), after all acquired mutations
   are resolved, iterate the species's intrinsic map and ADD
   each rank to `Character.Mutations[id]`. Cap-aware — if a
   mutation declares a max rank, the sum is clamped.

7. **One unified character init pathway.** Both mob spawn and
   player creation share `Character.applyIntrinsicMutations(species)`
   helper. Player path is structurally the same as mob path
   even though humans (the only player-selectable species)
   currently declare no intrinsic mutations — keeps the code
   uniform and supports any future hypothetical "play a wolf"
   feature without re-plumbing.

8. **Latent bug fix: SpawnMutations curated list now gates.**
   The current `mobs.go:554-558` path applies mob-YAML-declared
   `mutations:` unconditionally — no body-parts check, no
   conflicts check, no cap check. The new behavior: apply each
   entry through the same gate the random-roll path uses, log a
   warning on rejection, skip the entry. This means a wolf YAML
   declaring `mutations: { extra-arms: 2 }` will now silently
   refuse to grant extra-arms and log: "mob 205: cannot grant
   extra-arms (requires body_part: arms, species canine has
   none)."

9. **Mid-game mutation grants gate too.** Mutation potions,
   quest rewards, admin commands all flow through the same
   gate. If a player drinks an extra-arms mutation potion as a
   human, it works (humans have arms). If a hypothetical
   "wolf-form" player drinks it, the potion fizzles with
   in-fiction flavor text ("Your body cannot integrate this
   mutation"). For v1 the player gate is structural-only since
   only humans are selectable.

10. **Validation panics at boot.** Mutation YAML with unknown
    body-part tag → panic. Species YAML with unknown body-part
    tag → panic. Species YAML referencing unknown mutation id
    in `intrinsic_mutations` → panic. Per the project's data-load
    convention, fail-loud at boot beats subtle runtime
    misbehavior.

11. **Elementals expanded: 4 new species.** sand, storm, ice,
    smoke. Each models a distinct elemental flavor; existing
    earth/fire/air/water/magma stay as-is. The 5 instance-zone
    elementals (sand 318, storm 319, king 320, queen 321,
    prince 322) get repointed:
    - King → existing magma (40) + mob-YAML `mutations: { large: 1 }` override for the towering form
    - Queen → new ice (NEW), drops the chunk-2.2a `incorporeal: 4` mob override (her crystal/water body is corporeal per description)
    - Prince → new smoke (NEW), keeps incorporeal as intrinsic
    - Sand → new sand (NEW), incorporeal rank 2 (lower than air/storm — particulate has some substance)
    - Storm → new storm (NEW), incorporeal rank 4

12. **Chunk 2.2a cleanup falls out.** The 5 mob YAMLs currently
    tagged with `mutations: { incorporeal: 4 }` (wraith, spectre,
    fire elemental, air elemental, elemental queen) lose those
    redundant declarations — incorporeal is now intrinsic on
    the species. Queen specifically *loses* the override
    entirely (her species changes to ice, which is NOT
    incorporeal).

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/species/species.go` | MODIFY | Add `BodyParts []string` and `IntrinsicMutations map[string]int` fields. Add boot-time validation. |
| `internal/mutations/mutations.go` | MODIFY | `MutationSpec.RequiresArms` → `RequiresBodyParts []string`. Update `GetWeightedPool` filter signature to take `species.BodyParts` instead of `disabledSlots`. Add boot-time validation. |
| `internal/characters/character.go` (or new `intrinsic.go`) | MODIFY | New helper `applyIntrinsicMutations(species *species.Species)` that merges intrinsic ranks additively, cap-aware. Called from both mob spawn and player creation. |
| `internal/mobs/mobs.go` | MODIFY | (a) Replace the `disabledSlots` arg to `GetWeightedPool` with `species.BodyParts`. (b) Add body-part gate to the curated `SpawnMutations` path. (c) Call `applyIntrinsicMutations` after curated + roll. |
| `internal/users/users.go` (or character-creation site) | MODIFY | Call `applyIntrinsicMutations` on player creation. (No-op for humans today; future-proofs the player path.) |
| `internal/usercommands/mutate.go` or similar (mid-game grant) | MODIFY | Add body-part gate before granting. Reject with flavor text on mismatch. (Confirm the exact entry point during implementation — could also live in a buff-application hook.) |
| `_datafiles/world/dogmud/species/*.yaml` (existing 35) | MODIFY | Add `body_parts:` to each. Add `intrinsic_mutations:` where appropriate. |
| `_datafiles/world/dogmud/species/NN-sand_elemental.yaml` | NEW | Sand elemental species. ID TBD via `python tools/id_inventory.py --type species`. |
| `_datafiles/world/dogmud/species/NN-storm_elemental.yaml` | NEW | Storm elemental species. |
| `_datafiles/world/dogmud/species/NN-ice_elemental.yaml` | NEW | Ice elemental species. |
| `_datafiles/world/dogmud/species/NN-smoke_elemental.yaml` | NEW | Smoke elemental species. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/318-sand_elemental.yaml` | MODIFY | `speciesid` → sand. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/319-storm_elemental.yaml` | MODIFY | `speciesid` → storm. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/320-elemental_king.yaml` | MODIFY | Keep `speciesid: 40` (magma); add `mutations: { large: 1 }` override. |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml` | MODIFY | `speciesid` → ice. Remove `mutations: { incorporeal: 4 }` (queen is corporeal per description). |
| `_datafiles/world/dogmud/mobs/instance_planar_oasis/322-elemental_prince.yaml` | MODIFY | `speciesid` → smoke. |
| Other mob YAMLs currently tagged `mutations: { incorporeal: 4 }` (wraith, spectre, fire elemental, air elemental) | MODIFY | Remove redundant `mutations:` block — incorporeal is now intrinsic on the species. |
| `_datafiles/world/dogmud/mutations/*.yaml` | MODIFY | 13 mutations gain `requires_body_parts:`. See table in §"Mutation requirements." |
| `internal/species/context.md` | MODIFY | Document new fields. |
| `internal/mutations/context.md` | MODIFY | Document new schema, gating pipeline, intrinsic stacking, and the `RequiresArms` removal. |
| `internal/characters/context.md` | MODIFY | Document `applyIntrinsicMutations` and the init pathway. |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.5 Done, roll-up 12/41 → 13/41. |

## Body-part vocabulary

Seven canonical tags, exhaustive:

| Tag | Meaning | Example species WITH it | Example species WITHOUT it |
|---|---|---|---|
| `arms` | Explicit grasping limbs distinct from legs | human, troll, goblin, vampire, zombie, skeleton, bear, flesh_golem, earth_elemental | canine, feline, serpent, slime, fish, raptor |
| `hands` | Fingered manipulators on the arms | human, troll, goblin, vampire, zombie, skeleton, flesh_golem | earth_elemental (chunky fists, not fingers), bear |
| `legs` | Distinct locomotion limbs | most species | serpent, worm, slime, fish, fungal_colony, ghostly_spirit, wraith, spectre, air/fire/storm/smoke_elemental |
| `eyes` | Visual organs | most species | slime, fungal_colony, worm, ghostly_spirit, wraith, spectre, air/fire/storm/sand/smoke_elemental |
| `mouth` | Biting/vocal apparatus | most species | slime (debatable), fungal_colony, ghostly_spirit, all incorporeal elementals |
| `skin` | Surface coverage | most embodied species; bones-only skeleton lacks | skeleton, ghostly_spirit, wraith, spectre, air/fire/storm/smoke_elemental |
| `tail` | Anatomical tail (additive — mutation can still add a tail slot even if natural) | canine, feline, rodent, reptile, mustelid, vampire (some lore), insectoid (scorpion-style) | human, troll, goblin, skeleton, zombie, slime, etc. |

The `tail` tag is included for completeness but no mutation
*requires* tail in the catalog. It's recorded on species for
future symmetry (e.g., a "barbed-tail" mutation that requires
having one).

## Mutation requirements (xref against the 42-catalog)

The full set; only the 13 below declare a requirement.

| Mutation | `requires_body_parts` |
|---|---|
| extra-arms | `[arms]` |
| elongated-limbs | `[arms]` |
| clawed-hands | `[hands]` |
| extra-legs | `[legs]` |
| keen-eyes | `[eyes]` |
| night-vision | `[eyes]` |
| infrared-vision | `[eyes]` |
| toxic-bite | `[mouth]` |
| sonic-shout | `[mouth]` |
| blinding-spit | `[mouth]` |
| tough-skin | `[skin]` |
| thick-hide | `[skin]` |
| camo-skin | `[skin]` |
| chameleon-skin | `[skin]` |
| bioluminescence | `[skin]` |
| blinding-flash | `[skin]` |
| photosynthetic-skin | `[skin]` |

**Total: 17 mutations declare a body-part requirement.**

The remaining 25 mutations (adrenaline-surge, brazen-resolve,
cold-blooded, dense-muscles, fast-reflexes, hasted, healing-gel,
heightened-senses, hollow-bones, incorporeal, iron-constitution,
large, magical-resistance, pacifism-aura, pheromone-glands,
psychic-resistance, rapid-metabolism, regenerative-tissue,
sixth-sense, skilled, small, tail, talented, tremorsense, plus
one more from the catalog) declare no requirements. They apply
to any embodied creature.

## Species migration table (35 existing + 4 new)

### Humanoid (4)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 1 | human | `[arms, hands, legs, eyes, mouth, skin]` | `{}` |
| 4 | troll | `[arms, hands, legs, eyes, mouth, skin]` | `{ regenerative-tissue: 1 }` |
| 5 | goblin | `[arms, hands, legs, eyes, mouth, skin]` | `{}` |
| 99 | ascended | `[arms, hands, legs, eyes, mouth, skin]` | `{ magical-resistance: 1 }` |

### Quadruped / four-legged (9)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 2 | canine | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1 }` |
| 3 | bear | `[arms, legs, eyes, mouth, skin]` | `{ thick-hide: 1 }` (paws ≠ hands; bear arms count for wrestling but lack fine manipulation) |
| 6 | boar | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, thick-hide: 1 }` |
| 7 | deer | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, keen-eyes: 1 }` |
| 9 | raptor | `[legs, eyes, mouth, skin]` | `{ keen-eyes: 1 }` (wings ≠ arms — birds don't wield) |
| 11 | feline | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, night-vision: 1 }` |
| 22 | bat | `[legs, eyes, mouth, skin]` | `{ night-vision: 1, tremorsense: 1 }` (echolocation flavored as tremorsense) |
| 24 | mustelid | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1 }` |
| 10 | rodent | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, small: 1 }` |

### Limbless or near-limbless (3)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 8 | serpent | `[eyes, mouth, skin]` | `{}` (toxic-bite is per-mob — not all serpents are venomous) |
| 18 | worm | `[mouth, skin]` | `{ tremorsense: 1 }` |
| 21 | reptile | `[legs, eyes, mouth, skin, tail]` | `{ tail: 1, cold-blooded: 1 }` |

### Arthropod (2)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 12 | insectoid | `[legs, eyes, mouth, skin]` | `{}` (too varied for a baseline; per-mob) |
| 17 | arachnid | `[legs, eyes, mouth, skin]` | `{ tremorsense: 1 }` (toxic-bite per-mob) |

### Aquatic (1)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 13 | fish | `[eyes, mouth, skin]` | `{ cold-blooded: 1 }` |

### Plant / fungal / amorphous (3)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 14 | carnivorous_plant | `[mouth, skin]` | `{ photosynthetic-skin: 1 }` |
| 15 | fungal_colony | `[skin]` | `{ photosynthetic-skin: 1 }` |
| 16 | slime | `[skin]` | `{ regenerative-tissue: 1 }` |

### Aberration (1)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 23 | aberration | `[]` (open — per-mob via mob YAML) | `{}` |

### Ethereal (1)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 0 | ghostly_spirit | `[]` | `{ incorporeal: 4 }` |

### Undead (6)
| ID | Species | `body_parts` | `intrinsic_mutations` |
|---|---|---|---|
| 30 | skeleton | `[arms, hands, legs, eyes, mouth]` (no skin) | `{ cold-blooded: 1, hollow-bones: 1 }` |
| 31 | zombie | `[arms, hands, legs, eyes, mouth, skin]` | `{ cold-blooded: 1 }` |
| 32 | wraith | `[]` | `{ incorporeal: 4 }` |
| 33 | spectre | `[]` | `{ incorporeal: 4 }` |
| 34 | vampire | `[arms, hands, legs, eyes, mouth, skin]` | `{ regenerative-tissue: 1, night-vision: 1 }` |
| 35 | flesh_golem | `[arms, hands, legs, eyes, mouth, skin]` | `{ iron-constitution: 1 }` |

### Elementals (5 existing + 4 new = 9)
| ID | Species | `body_parts` | `intrinsic_mutations` | Status |
|---|---|---|---|---|
| 36 | water_elemental | `[skin]` | `{ regenerative-tissue: 1 }` | existing — body-tag |
| 37 | earth_elemental | `[arms, legs, skin]` | `{ thick-hide: 1, iron-constitution: 1 }` | existing — body-tag |
| 38 | air_elemental | `[]` | `{ incorporeal: 4 }` | existing — body-tag |
| 39 | fire_elemental | `[]` | `{ incorporeal: 4 }` | existing — body-tag |
| 40 | magma_elemental | `[skin]` | `{ thick-hide: 1 }` | existing — body-tag |
| NEW | sand_elemental | `[skin]` | `{ incorporeal: 2, blinding-spit: 1 }` | NEW |
| NEW | storm_elemental | `[]` | `{ incorporeal: 4, hasted: 1 }` | NEW |
| NEW | ice_elemental | `[arms, legs, skin]` | `{ cold-blooded: 1, magical-resistance: 1 }` | NEW (humanoid form, corporeal) |
| NEW | smoke_elemental | `[]` | `{ incorporeal: 4, hasted: 1, fast-reflexes: 1 }` | NEW (phase-shifting) |

**Test/system species (skip migration):** 19-dummy, 20-orb. These keep no `body_parts` field, fall through to fail-open (every mutation passes the gate). Acceptable since they aren't real combatants.

## Mob YAML changes (instance_planar_oasis)

| Mob | Old `speciesid` | New `speciesid` | mob-YAML `mutations:` change |
|---|---|---|---|
| 318-sand_elemental | 37 (earth) | NEW sand | remove (intrinsic via species) |
| 319-storm_elemental | 37 (earth) | NEW storm | remove (intrinsic via species) |
| 320-elemental_king | 37 (earth) | 40 (magma) | ADD `mutations: { large: 1 }` for towering stature |
| 321-elemental_queen | 37 (earth) | NEW ice | REMOVE existing `mutations: { incorporeal: 4 }` (corporeal per description) |
| 322-elemental_prince | 37 (earth) | NEW smoke | remove (intrinsic via species) |

**Other mobs with redundant `mutations: { incorporeal: 4 }`:**
- wraith (any mob using species 32) — REMOVE redundant override
- spectre (species 33) — REMOVE redundant override
- fire elemental mobs using species 39 — REMOVE
- air elemental mobs using species 38 — REMOVE

The chunk-2.2a-era authoring put incorporeal on the mob YAML
because the intrinsic-mutations mechanism didn't exist yet.
Now it does; cleanup is mechanical.

## Public API

### Species struct extension

```go
// internal/species/species.go

type Species struct {
    // ... existing fields ...

    // BodyParts is the set of canonical body-part tags this
    // species has. Used by the mutation gating pipeline.
    // Empty/missing means "no body" (incorporeal-style) — gates
    // every body-part-requiring mutation. Absence of the field
    // entirely in YAML is treated as fail-open (all seven implicit)
    // for backward compat with un-migrated species.
    BodyParts []string `yaml:"body_parts,omitempty"`

    // IntrinsicMutations maps mutation id → baseline rank.
    // Merged additively into Character.Mutations at character init.
    // Cap-aware via the mutation's max rank.
    IntrinsicMutations map[string]int `yaml:"intrinsic_mutations,omitempty"`
}

// HasBodyPart returns true if this species declares the given
// canonical body-part tag, OR if BodyParts is absent (fail-open).
func (s *Species) HasBodyPart(part string) bool {
    if s == nil {
        return true // fail-open for missing species (defensive)
    }
    if s.BodyParts == nil {
        return true // fail-open for un-migrated species
    }
    for _, p := range s.BodyParts {
        if p == part {
            return true
        }
    }
    return false
}

// HasAllBodyParts returns true if this species has every part
// in the requirements list (empty list always returns true).
func (s *Species) HasAllBodyParts(required []string) bool {
    for _, part := range required {
        if !s.HasBodyPart(part) {
            return false
        }
    }
    return true
}
```

### MutationSpec change

```go
// internal/mutations/mutations.go

type MutationSpec struct {
    // ... existing fields ...

    // RequiresBodyParts lists canonical body-part tags the
    // mutation needs. Empty/missing = body-agnostic. Validated
    // at boot against the canonical seven-tag set.
    RequiresBodyParts []string `yaml:"requires_body_parts,omitempty"`

    // RequiresArms is REMOVED. Migrate via:
    //   RequiresArms: true → RequiresBodyParts: [arms]
}

// CanApplyTo returns true if the mutation's body-part
// requirements are met by the given species.
func (s *MutationSpec) CanApplyTo(sp *species.Species) bool {
    return sp.HasAllBodyParts(s.RequiresBodyParts)
}
```

### Pool filter signature change

```go
// Before:
//   func GetWeightedPool(current map[string]int, disabledSlots []string) []*MutationSpec
//
// After:
func GetWeightedPool(current map[string]int, sp *species.Species) []*MutationSpec {
    var pool []*MutationSpec
    for _, spec := range allMutationSpecs {
        if _, owned := current[spec.MutationId]; owned {
            continue
        }
        if conflictsWithExisting(spec, current) {
            continue
        }
        if !spec.CanApplyTo(sp) {
            continue
        }
        // weight by rarity (existing math)
        for i := 0; i < 11-spec.Rarity; i++ {
            pool = append(pool, spec)
        }
    }
    return pool
}
```

### Character init helper

```go
// internal/characters/character.go (or new intrinsic.go)

// ApplyIntrinsicMutations merges the species's intrinsic
// mutations additively into the character's mutation map.
// Cap-aware: clamps each combined rank to the mutation's max
// rank if declared. Called from mob spawn AND player creation
// after all other mutation logic (curated SpawnMutations,
// random roll, persistent acquired).
//
// Logs a debug line per intrinsic applied. No-op if species is
// nil or has no intrinsic_mutations.
func (c *Character) ApplyIntrinsicMutations(sp *species.Species) {
    if sp == nil || len(sp.IntrinsicMutations) == 0 {
        return
    }
    if c.Mutations == nil {
        c.Mutations = make(map[string]int)
    }
    for id, intrinsicRank := range sp.IntrinsicMutations {
        spec := mutations.GetSpec(id) // returns nil if not loaded
        cap := 4 // default cap matches existing chunk-2.2a convention
        if spec != nil && spec.MaxRank > 0 {
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

## Data flow

### Mob spawn (full sequence)

1. `NewMobByIdFresh(mobId)` clones the template, copies species
2. **Curated `SpawnMutations` from mob YAML** (the `mutations:` block):
   - For each entry `(id, rank)`: check `species.HasAllBodyParts(spec.RequiresBodyParts)`. If false, log warning + skip. Else assign rank to `Character.Mutations[id]`.
3. **Random roll** (probability `mutationchance%`):
   - Build `pool := mutations.GetWeightedPool(Character.Mutations, species)` — filtered by body-parts + conflicts + already-owned
   - If pool non-empty: `mutations.RollAcquisition(pool)` picks one, assign rank 1
4. **Intrinsic merge:** `Character.ApplyIntrinsicMutations(species)` adds species's intrinsic ranks additively, cap-aware
5. Recompute derived stats (already in the existing pipeline)

### Player character creation

1. Player selects species (humanoid-only in current world; structurally any selectable species)
2. Allocate initial stats (existing flow)
3. Mutation selection UI (if applicable — players may choose starting mutations via character creation or earn them later)
4. `Character.ApplyIntrinsicMutations(species)` runs at the same insertion point as mob spawn's step 4
5. Save player file with effective `Mutations` map (intrinsic + acquired baked in)

### Mid-game mutation acquisition

When a player gains a mutation post-creation (potion drink, quest reward, admin grant):

1. Look up `mutations.GetSpec(id)`
2. Check `spec.CanApplyTo(player.species)`. If false:
   - Player-facing flavor text: `"Your body cannot integrate this mutation."`
   - Buff/potion is still consumed (no refund mechanic — author's choice; if we want refund-on-reject, that's a future tuning knob)
3. If true: add rank to `Character.Mutations[id]`, clamp to max
4. Recompute derived stats

### Species change (rare future case)

The chunk does NOT implement species-change for players — there's no in-game mechanism today. But the design is forward-compat: if a player's species ever changes, the simplest behavior is to discard ALL mutations and re-run `ApplyIntrinsicMutations` for the new species. Document but don't implement.

## Validation rules (boot-time panics)

1. Every species YAML's `body_parts:` entries must be in the
   canonical seven-tag set. Unknown tag → panic with file path
   + offending value.
2. Every species YAML's `intrinsic_mutations:` keys must
   reference loaded mutation ids. Unknown id → panic.
3. Every mutation YAML's `requires_body_parts:` entries must
   be in the canonical set. Unknown tag → panic.
4. The legacy `requires_arms: true` field is REMOVED entirely
   from the Go struct AND from all mutation YAMLs in this chunk.
   Any post-chunk YAML still using it would fail to parse (Go
   doesn't have the field). No transitional safety check is
   needed — the migration is atomic.

## Testing

| Test file | Cases |
|---|---|
| `internal/species/species_test.go` | `HasBodyPart` returns true for present tag, false for absent. `HasBodyPart` returns true on nil species (fail-open). `HasAllBodyParts` returns true on empty requirement list. `HasAllBodyParts` returns false when any required tag is absent. |
| `internal/mutations/mutations_test.go` | `CanApplyTo` returns true when species has all required parts. `CanApplyTo` returns false when species lacks any required part. `CanApplyTo` returns true for body-agnostic mutation. `GetWeightedPool` excludes mutations whose body-part requirements aren't met. |
| `internal/characters/character_test.go` (or new `intrinsic_test.go`) | `ApplyIntrinsicMutations` adds intrinsic to empty `Mutations` map. `ApplyIntrinsicMutations` stacks additively when already-acquired. `ApplyIntrinsicMutations` clamps at max rank. `ApplyIntrinsicMutations` no-ops on nil species. |
| `internal/mobs/mobs_test.go` | Spawn integration: canine spawn N times, none roll extra-arms. Wolf spawn with `mutations: { extra-arms: 1 }` curated entry — warning logged, skip applied. Wolf spawn with `intrinsic_mutations: { tail: 1 }` and `mutationchance` rolling tail → effective rank 2 (additive). |
| `internal/mutations/validation_test.go` (new) | YAML with unknown body-part tag → panic at load. Species with unknown intrinsic mutation id → panic. |

## Smoke test

After unit tests pass:

1. `go build ./...` clean
2. `go test ./...` no FAILs
3. Boot server, watch for clean species/mutation load. The new
   species YAMLs must load without panic. `species.LoadDataFiles
   () loadedCount=` increments by 4.
4. Spot-check via admin:
   - **Wolf body-parts gate:** spawn N=20 steppe wolves with
     `mutationchance: 100`. Confirm NONE have extra-arms via
     `mob <id> show mutations`. Confirm some have tail at
     rank 2 (intrinsic 1 + rolled 1).
   - **Curated-skip warning:** add a temporary
     `mutations: { extra-arms: 1 }` entry to a wolf YAML.
     Spawn. Verify warning in log: "mob 205: cannot grant
     extra-arms (species canine lacks body_part: arms)".
     Verify wolf does NOT have extra-arms equipped slots.
     Revert the YAML.
   - **Intrinsic stacking:** spawn a single wolf and capture
     its mutations via admin. Should show `tail: 1`. Use
     admin to add a tail mutation manually
     (`mutations grant tail`). Should now show `tail: 2`.
     Confirm tailsweep/trip-reskin works in combat (drop in
     same room, trip it).
   - **Elemental queen (formerly incorporeal):** spawn the
     elemental queen (mob 321). Confirm she is now corporeal
     (regular damage works on her, no `incorporeal` in her
     mutation list).
   - **Prince incorporeal:** spawn the elemental prince
     (mob 322). Confirm `incorporeal: 4` is in her mutation
     list (via intrinsic from smoke_elemental species).
     Confirm gear-effectiveness multiplier kicks in
     (test by giving her a sword via admin — damage
     contribution should be near-zero).
   - **Player path no-op:** create a new test character.
     Confirm `mutations` list is empty (humans have no
     intrinsic mutations declared).
   - **Drink potion gate:** as a human player, drink an
     extra-arms mutation potion. Should work (humans have
     arms). Document the player-facing line for the future
     non-human-player case ("Your body cannot integrate this
     mutation") — but cannot test without a non-human player,
     so this is structural-only.
5. Run the AI tester goal file at
   `tools/testing/goals/chunk-2-5-mutation-parity-smoke.yaml`
   (authored alongside the chunk).
6. Kill test servers per the kill-test-mud-servers SOP.

## Out of scope / deferred

- **Size-based gating** (`large` mutation on already-Large
  species, `small` on Small species). Acknowledged weakness.
  No clean axis for v1; `Size` enum on Species is used for
  combat/inventory math, not for mutation eligibility. Tuning
  pass.
- **Body-type axis** (living vs. constructed vs. elemental
  vs. undead) for finer mutation gating like "no healing-gel
  on stone golems." Explicitly punted — golems can still ooze
  healing-gel because it's content-acceptable.
- **Sentience axis** for "no `skilled` on mindless creatures."
  Niche; deferred to a content pass.
- **Combat archetype × mutation rules.** Non-combat archetypes
  with combat mutations are harmless today (mutation effects
  sit idle). Not gated.
- **Player species variety.** Only humans are
  `Selectable: true`. The body-parts gating on the player path
  is structural — there's no in-game case where it activates
  today. Future "play a wolf" feature inherits the
  infrastructure for free.
- **Mutation REMOVAL.** Once a player has a mutation, removing
  it (purge spell, antidote, etc.) is not designed here. Save
  state stores combined effective rank; removing intrinsic vs.
  acquired separately is not supported and not needed today.
- **In-game body-parts inspection.** No player-facing UI to see
  "your species has these body parts" — the information is on
  the species YAML / help files. Future polish if needed.
- **Cross-mutation interactions beyond `conflicts:` and
  `requires_body_parts:`.** E.g., "tail mutation needs `legs`
  to anchor" — not modeled. Conflicts cover the common cases.
- **The 4 new elemental species' visual / description
  authoring.** This spec specifies mechanics. The species
  YAMLs' `description:`, `unarmedname:`, etc. fields need
  written prose during implementation. Author-time concern.
- **Hand-vs-paw distinction for predatory beasts** — wolves
  and cats have claws-on-paws, but we model this via species
  `Damage`/`NaturalBash`, NOT via the `clawed-hands` mutation.
  Documented intent; no code change.

## Roadmap touchpoints

This chunk:

- Closes chunk **2.5** on `MOB_ALIVENESS_ROADMAP.md`. Roll-up
  moves from 12/41 → 13/41.
- Builds on chunk **2.2a** (Incorporeal mutation), folding the
  per-mob YAML `mutations: { incorporeal: 4 }` overrides into
  species intrinsic declarations. Net deletion of 5 mob-YAML
  overrides.
- Closes the chunk-2.4 spec's "Out of scope" note about
  *"Mutations on mobs (Companion Phase 5)"*.
- Provides a substrate (`Species.BodyParts`,
  `Character.ApplyIntrinsicMutations`) that future chunks can
  build on:
  - Chunk **5.3** (equipment-aware shopping) — itemvalue scoring
    already reads `mutations.GearEffectivenessMultiplier`,
    which intrinsic stacking now feeds.
  - Chunk **6.5** (broader content rollout) — new mobs get a
    rich mutation surface without per-mob `mutations:` block
    authoring.
