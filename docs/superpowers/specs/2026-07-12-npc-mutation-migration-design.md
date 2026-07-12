# NPC Mutation Migration — Legacy-Pool Nuke + Mob Content Cleanup

**Date:** 2026-07-12
**Status:** Design approved (disposition + archetype-shift decisions locked)
**Ships in:** the `0.14.0` clean-break release, alongside the player migration
(`docs/superpowers/specs/2026-07-11-mutation-migration-design.md`).

---

## 1. Purpose & the reframe

The player migration (spec above, decisions locked, grounded in the 34 real prod
accounts in `_archive/prod-users/users/`) covers players. This substage covers
**everything else** — and the key finding from exploring the code is that it is
*not* a parallel-system migration:

> **Mob mutation acquisition is already on the new graph.**
> `tickMobMutationAcquisition` (`internal/hooks/NewRound_MobRoundTick.go`) already
> calls `mutations.GetGraphPool` with `ClusterAffinity` / `EffectiveAffinity` /
> `RollDeepening` / battle-frenzy — the exact machinery players use. There is no
> old mob *system* to migrate off of.

What remains is **legacy content cleanup**, most of which is not mob-specific.
The core problem it fixes:

> **The legacy pool leak.** `affinityFor` (`internal/mutations/affinity.go`)
> returns `math.MaxFloat64` for any zero-cluster mutation, so a zero-cluster
> mutation is *always* eligible in `GetGraphPool` / the bloom pool. **All 44
> untagged mutations are therefore always-offered** to both players and mobs —
> but only ~10 are intended Center enablers. The other ~34 are retired-41
> leftovers silently polluting every drift/bloom/tutorial grant.

**Goal:** collapse the acquisition pool to exactly the designed graph — 9
clusters + 9 bridges + Chrysifier + ~10 Center enablers — by deleting every
legacy leftover, and clean up the mob/behavior content that referenced them.

## 2. Disposition decision (locked)

**Nuke all legacy mutations** — every zero-cluster mutation except the ~10
intended Center enablers. Rationale (chosen over "keep a curated few"):

- **Actives are DOA.** The 6 wired active-ability mutations share the
  special-move cooldown with real special attacks / shouts / spells, so they are
  strictly dominated — a player would never spend the cooldown on them. Delete
  all 6.
- **No passive clears the bar.** Every legacy passive is either redundant with a
  better new keystone (mitigation/regen/stealth/stat-tradeoff) or generic filler.
  The two least-weak (`adrenaline-surge`'s unique `conditional_damage_low_hp`
  mechanic; `magical-resistance`'s raw 25% ward) are still not dopamine hits —
  the first is absorbed by Ravener's Blood Frenzy, the second is "just a number."
  Keeping any muddies an otherwise clean graph.

### 2.1 Survivors (the ~10 intended Center enablers — KEEP, untouched)
`hollow-bones`, `prehensile-tail`, `keen-senses`, `rapid-healing`, `thick-coat`,
`tremorsense`, `precognition`, `spiracle-lungs`, `winged-flight`, `tail`.

### 2.2 Delete set (~34 legacy mutations)
All zero-cluster mutations NOT in 2.1. Authoritative derivation: walk
`_datafiles/world/dogmud/mutations/*.yaml`, delete every file with **no
`clusters:` field** whose id is not in the 2.1 survivor list. Known members:
`adrenaline-surge`, `bioluminescence`, `blinding-flash`, `blinding-spit`,
`brazen-resolve`, `camo-skin`, `clawed-hands`, `cold-blooded`, `elongated-limbs`,
`extra-legs`, `fast-reflexes`, `hasted`, `healing-gel`, `heightened-senses`,
`incorporeal`, `infrared-vision`, `iron-constitution`, `keen-eyes`, `large`,
`magical-resistance`, `night-vision`, `pacifism-aura`, `pheromone-glands`,
`photosynthetic-skin`, `psychic-resistance`, `rapid-metabolism`,
`regenerative-tissue`, `sixth-sense`, `skilled`, `small`, `sonic-shout`,
`talented`, `tough-skin`, `toxic-bite`.

Each deletion removes **both** the `mutations/<id>.yaml` and its
`templates/help/<id>.template` (helpfile-completeness test stays green because it
checks one template per *surviving* YAML).

## 3. Ripple cleanup (the reason this is its own substage)

Deleting the set dangles several references. All must be resolved in the same
release or the boot validators / build break.

### 3.1 Wired active-ability commands (Go)
6 legacy actives have command code that must be removed together:
`blinding_flash`, `blinding_spit`, `healing_gel`, `pacifism_aura`, `sonic_shout`,
`toxic_bite`.
- Delete `internal/actions/mutation_<name>.go` (and any `internal/usercommands/`
  wrapper).
- Remove their dispatch entries in `internal/behaviortree/actions_mutation.go`
  and `internal/mobcommands/mobcommands.go` (mobs could invoke these).
- Delete any **dedicated buff** owned only by a deleted active (audit each
  active's buff id; delete orphans, leave shared buffs).

### 3.2 `adrenaline-surge` Go consumer
`internal/mutations/mutations.go` has `IsAdrenalSurgeActive` (hardcodes the id)
plus the `conditional_damage_low_hp` combat path. Remove the id-named helper and
its combat consumer. The generic `conditional_damage_low_hp` effect-type plumbing
(reader + `describe.go` case) may be **left inert** — harmless, no owner — or
removed; implementer's choice, default leave-inert.

### 3.3 Dangling `conflicts:` references in survivors
Surviving mutations that list a deleted id in `conflicts:` must have that entry
scrubbed. Known: `extra-arms` (→ `clawed-hands`, `elongated-limbs`),
`chameleon-skin` (→ `bioluminescence`), `tail` (→ `small`). Sweep all surviving
YAMLs for `conflicts:` entries pointing at the delete set.

### 3.4 Mob `spawnmutations` repoint
38 mob templates carry `spawnmutations:`; repoint every reference to a deleted id
to its surviving equivalent (or drop the entry when there is no meaningful
equivalent). Repoint map:

| Deleted id (refs) | Repoint to | Note |
|---|---|---|
| `large` (×6) | `titan-growth` | Colossus size node |
| `fast-reflexes` (×4) | `keen-senses` | Center reflex/dodge |
| `keen-eyes` (×3) | `keen-senses` | Center perception |
| `iron-constitution` (×2) | `titan-growth` | HP |
| `psychic-resistance` (×2) | *drop* | no clean equivalent |
| `magical-resistance` (×2) | *drop* | no clean equivalent |
| `tough-skin` (×1) | `thick-hide` | Ironhide armor |
| `sixth-sense` (×1) | `keen-senses` | dodge |
| `regenerative-tissue` (×1) | `rapid-healing` | Center regen |
| `camo-skin` (×1) | `padded-soles` | Stalker stealth |
| `night-vision` / `cold-blooded` / `rapid-metabolism` (×1 each) | *drop* | trivial |
| `chameleon-skin` (×1) | `padded-soles` | it is now the Stalker **apex** (r9, prereqs) — inappropriate as a random cat's spawn; a stealth-entry node fits better |

Surviving referenced ids (`thick-hide`, `dense-muscles`) are left as-is.

### 3.5 Archetype-shift — RE-BASE on clusters (locked decision)
Preserve the feature's intent (a mob visibly re-archetypes as it mutates) but
drive the pull from the **new graph** instead of the now-empty `archetype_pull`
table.

- **Drop** the `archetype_pull` YAML field from `MutationSpec` and delete
  `ValidateArchetypePulls` / `validateArchetypePulls` (nothing to validate).
- **Replace** `strongestArchetypePull` with a cluster/pole-derived pull: the
  rarest owned *graph* mutation determines the pull via a `cluster → archetype`
  map. Keep the existing FROM-set protection, TO-whitelist, per-mob-tree shadow,
  and flavor lines unchanged.
- Proposed `cluster → archetype` map (TO-whitelist targets):

  | Cluster (pole) | Archetype pull |
  |---|---|
  | colossus, ironhide (body) | `tank_taunter` |
  | ravener (body) | `predator` |
  | stalker (body) | `ambusher` |
  | ethereal (belief) | `pure_caster` |
  | manifester, zealot (belief) | `defensive_caster` |
  | weaver, trickster (hybrid) | *(none)* |
  | chrysifier, Center / zero-cluster | *(none)* |

  A mob whose rarest owned mutation maps to *(none)* does not shift — same
  "no pull" path as today.

### 3.6 Verification / no-orphan sweep
- `grant_mutation` (quest-30 `cleric_hadwen`, plus any others) takes no explicit
  id → it draws from the pool; it simply inherits the cleaner pool. Verify the
  tutorial still grants a designed mutation and room-5200's "has any mutation"
  lore gate still fires.
- `grep` Go for any *other* hardcoded deleted id (beyond `adrenaline-surge` and
  the 6 actives) and resolve. `species.go`'s `large`/`small` are the `Size`
  enum — unrelated, leave them.
- Confirm no size / special logic keys off the `large` / `small` *mutations*.

## 4. Ordering & release

This substage ships **in `0.14.0` with the player migration**, as one clean
break: the player migration wipes players' old mutations; this nuke removes those
mutations from the world and the acquisition pool so nothing can re-acquire them.
Do the content deletion + Go cleanup + archetype re-base, then run the player
migration on top. Boot-clean (graph validator, helpfile-completeness, archetype
re-base) is the gate.

## 5. Testing

- **Boot smoke** (authoritative): nuke instance saves, `go run .`, expect
  `Server Ready` with no panic — catches dangling conflicts, missing help
  templates, unresolved spawnmutations, and the archetype re-base validator.
- **Full suite** `go test ./...` — expect 87 packages ok. Update/relocate the
  archetype-shift tests (`archetype_shift_test.go`) to the cluster-derived pull;
  delete tests for removed actives.
- **Pool census test** (new, small): assert `GetGraphPool` / the bloom pool for a
  fresh character contains *only* designed ids (no member of the delete set) —
  locks the leak closed and prevents regressions.
- **Repoint audit**: after the spawnmutations sweep, re-grep mob templates for
  any delete-set id → expect none.

## 6. Out of scope

- The player migration itself (its own locked spec).
- Per-rank magnitude balance (6e balance pass).
- A future cluster-driven title system and any new mob archetypes.
