# The Chrysalis Graph — Mutation Rework Design

**Date:** 2026-07-10
**Status:** Approved (engine spec; content is a follow-on)
**Origin:** Queued brainstorm `project_mutation_graph_redesign`, unblocked after the
mutation-archetype-shift feature shipped (`docs/superpowers/specs/2026-07-10-mutation-archetype-shift-design.md`).
Re-curates that feature's provisional `archetype_pull` table.

---

## 1. Why

Today's 41 mutations are a flat pile of small stat tweaks (`+10 dexterity`, `−5%
strength`). Only the rarest (incorporeal, extra-arms) actually change how you
play; the rest are forgettable riders. Acquisition is a flat weighted random
roll with no structure, so a caster-leaning mob can roll straight into a brute
body plan.

This rework makes three moves:

1. **Mutations become keystones.** Each is a permanent *body change* that is a
   power spike or a whole new avenue of play — a dopamine hit on land *and* on
   deepen. Fewer, bigger, more impactful. The old rare-mutation bar becomes the
   baseline bar for every mutation.
2. **Mutations form a graph.** Playstyle clusters, an opposition axis, shared
   bridge nodes, and prerequisite spines. Builds emerge from play; identities
   are coherent; over-commitment has a cost.
3. **Belief steers form.** Canon: *"the organism responds to belief and
   self-concept — what you believe you are shapes what the Chrysalis makes you"*
   (`novel/STORY_BIBLE.md`); *"your mutation will become what you believe it is"*
   (`world.md`). Mechanically: **what you keep doing** drifts your form; crafted
   **phials** (belief made physical) let you re-commit deliberately.

### Design goals / player appeal

Distinct playstyle fantasies that appeal to different players — the smasher, the
feral brawler, the assassin, the unkillable wall, the summoner, the caster, the
controller, the trickster, the party-leader, and the broad generalist — each
reachable, each feeling like a real transformation.

---

## 2. Decisions (from the brainstorm)

| Question | Decision |
|---|---|
| Player agency model | **Emergent drift** — what you DO steers your form. No menu. |
| Drift signal | **Decaying action-affinity + owned-mutation gravity**, stacked. |
| Mutation shape | **Keystones** = permanent body changes (passive / verb-enhancer / metabolic state / transformation). Never a bespoke "move." |
| Size | Fewer, bigger. ~3–4 keystones per cluster; each escalates on deepen (levels 1–4). |
| Structure | A **graph**: 9 clusters in a ring + a central Generalist. |
| Opposition | Deep **Body** shrinks the Conviction pool; deep **Belief** degrades body-weapon/gear effectiveness. Gradual per step; near-total only at the very top. Via **pool degradation**, not cost inflation. |
| Gating | **Cluster affinity** steers drift; **prerequisite spines** gate what can land. |
| Player legibility | **Fully opaque in-game** (no cluster names, no meters) but **heavily hinted in helpfiles** (the shape + a few worked prereq examples). |
| Phials | Alchemy-crafted, per-cluster; a final flavoring step commits the direction. Drinking **strips other mutations + re-blooms onto that cluster**. Unflavored → **Generalist**. |
| Migration | **Clean break + free re-bloom** — retire the old 41, wipe ownership, one-time free re-bloom seeded from existing skills. |
| Acquisition cadence | Unchanged — combat-tick acquisition (mobs mutate in combat). |

---

## 3. The graph

Nine clusters form a **ring**; the **Generalist** sits at the center. Clusters
are **design-side scaffolding only — their names are never shown to players.**
Mutations have player-facing names; clusters do not.

```
                 Colossus ─Carapace─ Ironhide ─Rallying Aura─ Zealot
                /                                                   \
          Extra Arms                                            Beast Bond
            /                        ◇ UNIVERSAL                      \
       Ravener                       / Generalist                   Manifester
            \                     apex: Winged Flight                  /
       Venom Glands✦             (Hollow Bones + Tail)          Spirit Tether
              \                                                     /
            Stalker ─Umbral Skin─ Trickster ─Silk Glands─ Weaver ─Cocoon✦─ Ethereal
```

Ring order: **Colossus–Ironhide–Zealot–Manifester–Ethereal–Weaver–Trickster–Stalker–Ravener**
(and back to Colossus). Each cluster has exactly **two bridge mutations**, one
shared with each neighbor. Two bridges are **apex** (heavy, rare shared
keystones): **Venom Glands** (Ravener⇄Stalker) and **Cocoon** (Weaver⇄Ethereal).

### Poles & opposition

- **Body pole** (Colossus, Ironhide, Ravener, Stalker — the left/upper arc):
  physical mutations. Going deep shrinks the **Conviction pool** (`ConvictionMax`),
  which chokes spells, conviction-attacks, and manifestation together (all three
  are belief-manifestations that draw on Conviction / Charisma).
- **Belief pole** (Ethereal, Manifester — the right arc; Zealot is social/Cha,
  belief-adjacent): magic/psychic/social mutations. Going deep degrades
  **body-weapon and gear effectiveness** (generalizing incorporeal's existing
  `gear_effectiveness_loss`).
- **Hybrids** (Trickster, Weaver — center-bottom): deliberately stay *partial* on
  both poles — masters of neither extreme, so they never trigger a hard penalty.
  They are the connective tissue between the poles.

Opposed clusters sit **across** the ring; neighbors sit **along** it. The
opposition is what stops "caster → brick-shithouse" wandering and makes
commitment meaningful.

### Cluster identities

Each is a distinct playstyle fantasy. Stats/skills are the **soil** — they scale
the keystones, they do not define the identity.

| Cluster | Fantasy | Primary stat(s) / skill | Provisional apex |
|---|---|---|---|
| **Ravener** ✓ | feral unarmed martial — claws, fangs, frenzy | Str/Dex · unarmed-combat | Apex Form (always-on beast) |
| **Colossus** | size, smashing might, extra limbs, weapons | Str · weapon-combat | Titanic Frame (gigantism) |
| **Stalker** | stealth assassin — venom, ambush, evasion | Dex/Per · skullduggery | Perfect Predator (nightform) |
| **Ironhide** | unkillable body — carapace, plating, regen-of-form | Vit | Living Carapace |
| **Ethereal** | incorporeal + psychic caster | Wil/Per · spellcasting | Discorporation (full incorporeal) |
| **Manifester** | pure companions / summoning | Cha/Wil · manifestation | Brood / Second Body |
| **Zealot** | party buffs, rally, presence, control | Cha · rhetoric | Radiant Presence |
| **Trickster** (hybrid) | rogue/caster — hexes, phase, misdirection | Dex/Wil | *(lighter cluster — TBD)* |
| **Weaver** (hybrid) | control — webs, stunning shouts, disrupt | Wil/Per | Web Lord / Paralytic Field |
| **Generalist** (center) | breadth, adaptability | neutral enablers | **Winged Flight** |

> Apex names above are provisional; the content spec finalizes them.

---

## 4. Mutation shape

Every mutation is a **permanent body change**. Where a mutation adds offense, it
**enhances an existing verb** (`kick`, `punch`, `bite`) rather than inventing a
bespoke command. Keystone types:

- **Passive body** — a standing change (Rending Claws → strikes cause bleed).
- **Verb-enhancer** — reshapes an existing action (Raptor Legs → `kick` rakes/hamstrings).
- **Metabolic state** — a triggered physiological state (Blood Frenzy).
- **Transformation** — an always-on capstone (Apex Form, Winged Flight).

**Dopamine on land AND on deepen.** Levels 1–4 escalate the *ability*, not a
decimal (Pounce-style bleed → armor-shred → corpse-erupt). This preserves the
existing 4-level deepening machinery.

### Prerequisite spines

A second kind of graph edge, distinct from cluster membership. **Cluster
membership steers drift; prerequisites gate what can actually land.** A mutation
may require owning other mutations (at a min level) first.

- Winged Flight requires **Hollow Bones + Prehensile Tail**.
- Apex Form (Ravener) requires **Rending Claws + Raptor Legs**.

Prerequisites also produce **emergent exclusions**: Hollow Bones is the
antithesis of Colossus/Ironhide (mass, density, plating), so heavy body
specialists physically cannot fly — no hand-written rule needed. Prereqs are
authored per-mutation and validated at boot (unknown prereq id → panic, same
convention as `archetype_pull` / schedule validators).

### Worked example — Ravener (the template)

| Keystone | Type | On land | On deepen |
|---|---|---|---|
| **Rending Claws** | passive body (hands) | unarmed strikes open a stacking bleed | more stacks · armor-shred · corpse-erupt splash |
| **Raptor Legs** | verb-enhancer (legs) | `kick` hits hard + raking hamstring | knockdown · one-step lunge · free stomp on downed |
| **Blood Frenzy** | metabolic state | on foe-bleed or self <50% HP: faster attacks, immune to taunt/fear, cannot retreat | lifesteal · kills extend · terror radius |
| **Apex Form** ★ | transformation (always-on) | permanent body-weapon power, terror aura, devour execute; *progressively chokes casting* | devour heals more · terror widens · damage climbs |

*Prereq spine:* Rending Claws + Raptor Legs → Apex Form.

### Worked example — Generalist (center)

| Keystone | Type | Notes |
|---|---|---|
| **Hollow Bones** | passive body (skeleton) | lighter/faster, cheaper movement, frailer; flight root |
| **Prehensile Tail** | passive body (tail slot) | balance, free grasp, aerial control; flight root |
| **Quickened Nerves & Keen Senses** | passive body (neural) | dodge/initiative/spot-hidden — safe always-useful enablers |
| **Winged Flight** ★ | transformation | fly across rooms, escape melee, attack from angle, dodge earthbound; requires Hollow Bones + Tail |

The Generalist **never pays a pole penalty** (it never goes deep on a pole); its
payoff for breadth is an apex the specialists cannot reach.

---

## 5. Drift & acquisition engine

Replaces `mutations.GetWeightedPool`.

### Cluster affinity

For each cluster, `affinity = decayingActionSignal + ownedGravity`.

- **Action signal** — a per-cluster accumulator incremented on cluster-relevant
  actions, decaying over time (recent behavior weighted). Signal map:

  | Signal source | Cluster(s) |
  |---|---|
  | weapon-combat / unarmed-combat use | Colossus, Ravener |
  | skullduggery / ranged-combat use | Stalker |
  | spellcasting use | Ethereal |
  | rhetoric use | Zealot |
  | manifestation use | Manifester |
  | damage absorbed (tanked) | Ironhide |
  | (hybrids Trickster/Weaver drift from *mixed* signals — see content spec) |

- **Owned gravity** — each owned mutation contributes affinity to its cluster(s),
  so a build snowballs into itself. Bridge/shared mutations contribute to *both*
  their clusters.

### Rollability gate

A candidate mutation is eligible only when **all** hold:

1. Cluster **affinity ≥ the mutation's depth threshold** (rarer/deeper keystones
   demand more affinity — this replaces the flat `calcRarityBonus`).
2. **Prerequisites owned** at required level.
3. **No conflict** with owned mutations (existing `HasConflict`).
4. **Body-part requirements** met for the species (existing `CanApplyTo`).

Eligible candidates are weighted by `base(rarity) × affinity` and rolled on the
existing combat-tick acquisition path. Deepening an owned mutation vs. acquiring
a new one keeps the existing split.

### Tunable knobs (config, not pinned here)

Decay rate, affinity-per-signal, depth-threshold curve, owned-gravity weight,
and acquisition rate are all config. Per the rework intent, the **mutation rate
will likely be turned up** post-rework (`MobMutationRate` for mobs — clamped to
(0,1.0]; use `MutationBaseProgress` / `MutationProgressScale` / `MutationMaxCount`
for aggressive tuning — plus the player-side equivalents).

---

## 6. Opposition / pool-decay

`poleDepth(pole)` = Σ `rarity × level` over owned mutations belonging to that pole.

- **Body poleDepth → Conviction pool decay.** `ConvictionMax` is scaled down by a
  curve of Body poleDepth. Gradual per step; **near-total only at the extreme**
  (e.g. maxed Apex Form + maxed Extra Arms). Chokes spells, taunt, and summons
  together.
- **Belief poleDepth → body-weapon/gear decay.** Extends the existing
  `gear_effectiveness_loss` / body-weapon effectiveness to scale with Belief
  poleDepth, not just incorporeal's rank.

Hybrids and Generalists keep low poleDepth on both sides → no meaningful penalty.
The curve is a config knob; the shape is "flat until you commit, then steepening,
asymptotic near the apex."

---

## 7. Phials — deliberate re-spec

Phials are the **one visible, deliberate lever** in an opaque system, crafted
through the existing alchemy/potion framework (~9 flavors, one per cluster).

- The recipe ends with a **flavoring step**: adding a cluster-themed reagent
  commits the phial to that direction.
- **Drinking a flavored phial strips the character's other mutations and
  re-blooms them onto that cluster's path** (grants the entry keystone + sets
  strong affinity so subsequent play deepens it).
- **Skipping the flavoring step** yields a neutral phial that **leans the drinker
  into the Generalist** — grants **Hollow Bones**, rooting the flight spine.
- Non-trivial cost (reagents + toxicity), matching world.md's "removing
  mutations is expensive/ritual." A deliberate re-spec, not a casual toggle.

Phials operate over the single cap-aware `Character.Mutations` map, so they
compose with the other levers.

### Three re-mutation levers (all compose)

| Lever | Axis | Effect |
|---|---|---|
| **Phial** | direction | strip + re-bloom onto a chosen cluster (or Generalist) |
| **Moon-crash potion** | tier | clear-all + reroll with a higher rarity floor (`GetWeightedPoolWithFloor`) |
| **Bloom** | depth | cheap, dangerous deepening of owned mutations |

---

## 8. Player legibility

- **In-game:** opaque. No cluster names, no affinity meters. Mutations announce
  themselves (name + flavor) on land/deepen; the *system* is not exposed.
- **Helpfiles:** teach the shape — playstyle shapes belief and form over time;
  actions steer which mutations come; prerequisites gate the big ones — with a
  few concrete worked examples (Hollow Bones → Winged Flight). A player who reads
  and plays can work out the mechanics; the game never spells them out directly.

---

## 9. Mob integration

- Mobs **spawn with a seed cluster affinity** derived from their behavior
  archetype, so they drift *within* their nature (a predator deepens predator,
  not caster).
- The provisional `archetype_pull` table is **re-curated into cluster tags** —
  each mutation's cluster membership drives the existing
  `behaviortree.ReevaluateArchetypeShift` (cluster → archetype). Rarest-wins
  becomes vestigial once pulls are consistent per cluster.
- The existing **FROM-set protection** stands: bosses, leaders, authored
  specialists, and per-mob-btree mobs never shift/drift archetype.
- Mobs mutate in combat on the existing tick (`tickMobMutationAcquisition`).

---

## 10. Migration (clean break)

1. **Retire the old 41 mutations.** New mutation set replaces them.
2. **Wipe existing mutation ownership** on all characters (players + mob
   instances). Instance saves are not deployed to prod, so mob state is moot;
   player saves get the wipe.
3. **One-time free re-bloom** on next login: seed initial cluster affinity from
   the character's existing **skill levels** (a veteran swordfighter re-blooms
   body-side), then let natural acquisition proceed. Framed as an in-world event
   ("the moons stir").
4. **Sweep every reference to old mutation IDs**: items (e.g. Seething Prism's
   worn mutation tick), mob spawn `buffids`, the moon-crash potion pool, any
   spell/quest/dialogue that named a specific mutation, and the archetype-shift
   `archetype_pull` fields. Boot-time validators must catch dangling references.

---

## 11. Scope & follow-on

**This spec covers the engine** plus Ravener and Generalist as fully-worked
examples and one-line identity/apex sketches for the other eight clusters.

**Deferred to a follow-on content spec/plan:** authoring all 9 clusters ×
~3–4 keystones (balanced effects, prereq spines, bridge nodes, apex
transformations), the ~9 phial recipes + reagents, helpfile copy, the migration
mapping sweep, and the balance pass. That content work is too large to author
blind before the engine rules are real, and it will shape (and be shaped by) the
config knobs above.

## 12. Out of scope

- Per-cluster balance tuning (deferred to post-build playtest, per SOP).
- New spell/ability content beyond what keystones enhance.
- Reworking the alchemy framework itself (phials reuse it as-is).
- Any change to the 4-level deepening or the combat-tick cadence.
