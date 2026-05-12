# internal/species — Package Context

## Purpose

Implements the species system for DOGMud. Species define character ancestry,
base stats, stat modifiers, and intrinsic mutations. Unlike the upstream GoMud
race system, species are a full custom system tailored to DOGMud's mutation and
progression mechanics.

All players in DOGMud are Human (a species choice at character creation).
Mobs can be any species defined in the world data.

---

## Key Concepts

### Species Definition (SpeciesInfo struct)

A `SpeciesInfo` struct loaded from YAML (`_datafiles/world/dogmud/species/`)
contains:

- `specieid` — unique identifier (e.g., "human", "wolf", "spirit")
- `name` — display name ("Human", "Gray Wolf", "Spirit Elemental")
- `description` — flavor text shown when examining a character or mob
- `base_stats` — per-stat baseline (Strength, Dexterity, Perception, Vitality,
  Willpower, Charisma). Applied additively to a random stat pool during
  character creation.
- `stat_modifiers` — optional equipment-independent stat adjustments
  (e.g., "wolves get +10 Perception")
- `body_parts` — canonical anatomy tags (chunk 2.5)
- `intrinsic_mutations` — natural traits (chunk 2.5)

### Registry

```go
species.LoadSpeciesFiles()      // called once from main.go at startup
species.GetSpecies(id)          // look up a spec by id
species.GetAll()                // the full map
```

---

## Body Plan & Intrinsic Mutations (chunk 2.5)

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
respected (default MutationMaxRank = 4).

Boot-time validation (`ValidateBodyPartTags`) panics on unknown tags
or unknown mutation ids in intrinsic_mutations.

Helpers: `HasBodyPart(tag)`, `HasAllBodyParts(required)`,
`IsCanonicalBodyPart(tag)`.

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.5-mutations-on-mobs-design.md`

---

## Files in This Package

| File | Purpose |
|------|---------|
| `species.go` | Structs, registry, loader, body-part validation |
| `species_test.go` | Unit tests for body-part logic |
| `test_helpers.go` | Test utilities |
| `context.md` | This file — package overview for Claude Code |

---

## Integration Points

Species are consulted at:
1. **Character creation** (`modules/newchar.go`) — seed base stats, apply body-plan
   gating during random mutation acquisition, apply intrinsic mutations
2. **Mob spawn** (`mobs.go`) — apply intrinsic mutations + curated spawn mutations
   to fresh mob instances
3. **Mutation acquisition** (`internal/mutations/mutations.go`,
   `GetWeightedPool()`) — filter candidates by body-part compatibility

---

## Stage Roadmap

- **chunk 2.5** (in progress) — body-plan gating, intrinsic mutations
