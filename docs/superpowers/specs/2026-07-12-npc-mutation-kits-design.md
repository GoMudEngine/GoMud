# NPC Mutation Kits — Detailed NPC Migration (companion to the player migration)

**Date:** 2026-07-12
**Status:** Design approved (breadth, depth, boss-scope, rank-handling locked)
**Builds on:** `2026-07-12-npc-mutation-migration-design.md` (the legacy-pool nuke +
minimal single-mutation repoints). This design **supersedes those minimal
intrinsic repoints** with curated, coherent kits.
**Ships in / coupled to:** the `0.14.0` clean break with the player migration
(`2026-07-11-mutation-migration-design.md`); ranks are tuned in the **6e balance
pass** (see §6).

---

## 1. Purpose

The legacy-pool nuke left monstrous creatures with at most a single repointed
mutation (or nothing) — a wraith is no longer meaningfully "ethereal," an
elemental is just `grapple_immune`. This migration gives the world's creatures a
**coherent identity in the new mutation graph**: appropriate species carry a
curated kit that reads as "this creature IS a Colossus / an Ethereal being / a
Stalker," and the fights that matter (bosses) carry bespoke depth on top.

It is the NPC-side companion to the player migration: the player migration
classifies each veteran's history into cluster mutations; this classifies each
*creature's nature* into a cluster kit.

## 2. Approach — data-only, native mechanism

No new engine code. Two existing, boot-validated mechanisms carry everything:

- **Species base kit** → `intrinsic_mutations` on the species YAML
  (`_datafiles/world/dogmud/species/`). `Character.ApplyIntrinsicMutations`
  merges the whole map at ranks into every creature of that species; the
  cross-reference validator (`species.go`) panics at boot on an unknown id.
- **Boss bespoke overlay** → `spawnmutations` on the specific mob YAML, which
  merges *on top of* the species base for that one creature.

Rejected: a named-kit-template indirection layer and code-driven assignment —
both over-engineer ~20 species and hide content in code, against the engine's
data-driven grain.

## 3. Principles

1. **One primary cluster per species.** Each monstrous species anchors to a
   single cluster; a **single thematic splash** (a Center/bridge node, or one
   flavor rider such as an elemental's `reflect-skin-*`) is allowed on top. This
   makes the in-combat
   **acquisition** (`GetGraphPool` drift) and **archetype-shift**
   (cluster→archetype map, 2026-07-12) deepen the creature *coherently* — an
   Ethereal wraith that mutates mid-fight drifts further into Ethereal and shifts
   toward `pure_caster`.
2. **Depth by species tier.** Elite/rare species carry keystones up to their
   apex; common species carry a single keystone; bosses add depth via their own
   `spawnmutations`. A species kit applies uniformly to every creature of that
   species, so this per-species tiering is what keeps trash mobs reasonable.
3. **Species fields persist alongside the kit.** `grapple_immune`, `lifedrain`,
   `body_parts: []`, `buffids`, `naturalarmor`, etc. are untouched — the kit adds
   graph identity, it does not replace the mechanical species traits.
4. **Humans are the neutral baseline.** `human` and `goblin` (and other plain
   humanoids) carry **no** kit — they are the zero point the graph is measured
   against.

## 4. The species → cluster-kit mapping

Ranks below follow the **provisional convention** (§6): common keystone = 1,
elite keystone = 1–2, apex = 1. Marked `(elite)` species go deep; others shallow.

### 4.1 Undead
| Species | Tier | Cluster | Kit (`mutation: rank`) |
|---|---|---|---|
| ghostly_spirit | common | Ethereal | `ether-gland: 1` |
| skeleton | common | Center | `hollow-bones: 1` *(already present — keep)* |
| zombie | common | Ironhide | `regrowth: 1` |
| wraith | **elite** | Ethereal | `ether-gland: 2`, `second-sight: 1`, `kinetic-backlash: 1`, `discorporation: 1` |
| spectre | **elite** | Ethereal | `ether-gland: 2`, `second-sight: 1`, `discorporation: 1` |
| vampire | **elite** | Stalker | `padded-soles: 1`, `compound-eyes: 1`, `venom-glands: 1` *(+ species lifedrain)* |
| flesh_golem | mid | Colossus | `dense-muscles: 1`, `titan-growth: 1` |

### 4.2 Elementals
All keep `grapple_immune`; most anchor Ethereal ("untouchable") with an
element-flavored rider; the earthy ones anchor a body cluster.
| Species | Cluster | Kit |
|---|---|---|
| water | Ironhide | `regrowth: 1` |
| earth | Colossus | `dense-muscles: 1`, `titan-growth: 1`, `thick-hide: 1` |
| air | Ethereal | `ether-gland: 1`, `second-sight: 1` |
| fire | Ethereal | `ether-gland: 1`, `kinetic-backlash: 1` |
| magma | Ironhide | `thick-hide: 1`, `chitin-plating: 1` |
| sand | Stalker | `padded-soles: 1`, `veiling-musk: 1` |
| storm | Ethereal | `ether-gland: 1`, `reflect-skin-voltaic: 1` |
| ice | Ironhide | `thick-hide: 1`, `reflect-skin-frostbite: 1` |
| smoke | Ethereal | `second-sight: 1`, `veiling-musk: 1` |

### 4.3 Overtly-magical / monstrous
| Species | Tier | Cluster | Kit |
|---|---|---|---|
| aberration | elite | Trickster | `evil-eye: 1`, `corvid-brain: 1` *(+ `translucent-body: 1` on boss instances)* |
| ascended | **elite** | Zealot | `commanding-presence: 1`, `zealous-conviction: 1`, `radiant-avatar: 1` |
| slime | mid | Ironhide | `regrowth: 1` |
| fungal_colony | mid | Weaver | `sticky-secretion: 1`, `dissonance-organ: 1` |
| carnivorous_plant | mid | Weaver | `grasping-tendrils: 1` |
| orb | common | Ethereal | `ether-gland: 1` |
| troll | mid | Ironhide | `thick-hide: 1`, `regrowth: 1` |

### 4.4 Mundane beasts — light 1–2 flavor touch
| Species | Kit | Species | Kit |
|---|---|---|---|
| feline | `padded-soles: 1` | canine | `rending-claws: 1` |
| bear | `dense-muscles: 1` | boar | `dense-muscles: 1` |
| raptor | `raptor-legs: 1` | deer | `keen-senses: 1` |
| serpent | `venom-glands: 1` | arachnid | `silk-glands: 1` |
| worm | `tremorsense: 1` *(already present — keep)* | bat | `keen-senses: 1` |
| mustelid | `padded-soles: 1` | insectoid | `compound-eyes: 1` |
| rodent | *(none — keep `tail`)* | fish | *(none)* |

### 4.5 Bare (baseline)
`human` and `goblin` (plain humanoids) and `dummy` (test/training species) carry
**no** kit — they are the neutral zero point. (`orb` is treated as a minimal
magical construct in §4.3, not bare.)

## 5. The named-boss bespoke layer

### 5.1 Policy
Each curated boss adds `spawnmutations` **on top of** its species base —
typically its cluster's **apex + one supporting keystone**. Signature/endgame
bosses may splash a second cluster or carry two apexes. The overlay is authored
per-mob so a boss reads as a deliberate, formidable version of its kind.

### 5.2 Selection criteria (~40–60 mobs)
A mob qualifies for a bespoke kit if it is any of:
- one of the existing 33 `spawnmutations` mobs (upgrade from the repointed single
  mutation to a full kit);
- an **endgame** boss (the #20/#21 encounter bosses in
  `docs/ENDGAME_COMBAT_TUNING.md`). NOTE: "Meirok" in that doc is the *player
  character* used as the difficulty yardstick, NOT a boss — do not give it a kit;
- a **Confluence** climax boss (#17, Q73/Q74);
- a **quest-line** boss (e.g. the Q34 bandit captain);
- a **zone apex/alpha/sentinel/keeper** (proper-noun uniques like
  `the_pass_apex`, `the_reach_alpha`, `the_sentinel`, `the_threshold_keeper`,
  `warden_prime`, `the_core_guardian`, `elemental_king`, `stone_beetle_queen`,
  `the_foldweaver`, `the_old_white`).

The implementation plan enumerates the full list world-wide (grep proper-noun
mob names + the known boss files); this spec fixes the criteria and the kit
policy so enumeration is mechanical.

### 5.3 Examples (illustrative — full list in the plan)
| Boss | Species | Bespoke overlay (`spawnmutations`) |
|---|---|---|
| `the_foldweaver` | arachnid/aberration | Weaver apex: `grasping-tendrils`, `paralytic-field` |
| `elemental_king` | elemental | cross-element: `titan-growth`, `discorporation` |
| `warden_prime` / `the_core_guardian` | flesh_golem/construct | Colossus: `titan-growth`, `colossus-form` |
| `the_pass_apex` / `the_reach_alpha` | feline/beast | Ravener/Stalker: `rending-claws`, `apex-predator` |
| #20/#21 encounter bosses | (per lore) | signature apex kit per `ENDGAME_COMBAT_TUNING.md` |

## 6. Ranks are PROVISIONAL — coupled to the 6e balance pass

This design fixes kit **composition** (which mutations), not numeric power. The
per-mutation **rank ceilings** and **per-rank magnitudes** are undefined until
the 6e balance pass (content-spec §7), which tunes players and NPCs together.

- Author all kits at the **provisional rank convention**: common keystone `1`,
  elite keystone `1–2`, apex `1`.
- When 6e lands, revisit **only the rank numbers** on these kits — identity does
  not change. Because ranks live in a single map field per species/boss, the
  retune is a cheap mechanical sweep, not re-authoring.
- Do **not** block this migration on 6e; ship provisional ranks, tune later.

## 7. Integration with the completed minimal migration

- **Replaces** the single-mutation `intrinsic_mutations` repoints from
  `2026-07-12-npc-mutation-migration` (e.g. `flesh_golem: titan-growth` →
  the full Colossus base `dense-muscles + titan-growth`). Species that were
  emptied (wraith/spectre/elementals) get their kits back.
- **Keeps** the mob `spawnmutations` repoints where those mobs are *not* curated
  bosses; upgrades them where they are.
- The **guard test** `TestNoLegacyZeroClusterMutations` and the boot validators
  (intrinsic id cross-reference, graph prereq/cluster) remain the safety net —
  every kit id must be a live graph mutation.
- **Restores** cluster membership so the re-based archetype-shift produces
  sensible drift (Ethereal undead → `pure_caster`, Colossus golems →
  `tank_taunter`, etc.).

## 8. Testing

- **Boot smoke** (authoritative): nuke instance saves, `go run .`, expect
  `Server Ready` with no panic — exercises the intrinsic-id validator, graph
  cluster/prereq checks, and mob/species loaders against every new kit.
- **Kit-coherence test** (new, filesystem-walk in `internal/devtools`, mirrors
  the legacy-pool guard): for each species with `intrinsic_mutations`, assert
  every id **exists as a live mutation YAML** (catches typos + any deleted-legacy
  regression), and that the kit is **anchored** — at least one member belongs to
  the species' declared primary cluster. This enforces the anchoring principle
  while permitting the single thematic splash (§3.1) — it does not require every
  member to share the primary cluster.
- **Repoint audit:** re-grep species intrinsics + boss spawnmutations for any
  deleted-legacy id → expect none (regression guard vs the nuke).
- Full suite `go test ./...` → 87 packages ok.

## 9. Phasing (for the implementation plan)

- **Phase 1 — species base kits** (§4): the bulk; auto-covers every creature.
  Ships + boots green on its own.
- **Phase 2 — boss bespoke overlays** (§5): enumerate the curated set, author
  `spawnmutations` overlays.
- **Phase 3 — kit-coherence test + boot + PATCH_NOTES** (§8).

Each phase is independently testable and committable.

## 10. Out of scope

- Per-rank magnitudes / rank ceilings (6e balance pass — §6).
- The player migration itself (its own spec).
- New mutations or new clusters (the graph is complete).
- Humanoid/baseline species (§4.5) — intentionally bare.
