# Chrysalis Graph — Cluster Content Design

**Date:** 2026-07-11
**Status:** Approved (design); implementation is a multi-wave follow-on
**Depends on:** the engine (merged `7d7afa57a`) —
`docs/superpowers/specs/2026-07-10-chrysalis-mutation-graph-design.md` and its
plan. This spec fills that engine with real content: every cluster's keystone
roster, the shared bridges, the Center hub, and the **mechanic primitives** the
keystones require.

---

## 1. What this is

The engine ships graph-aware acquisition, cluster affinity, prerequisite gating,
and the Body↔Belief pole opposition — but is **inert** because no mutation carries
`clusters`/`pole`/`prerequisites` yet. This spec defines that content: **9
clusters × 4 own-keystones + 9 shared bridges + a Center hub of ~9 enablers**.

The old 41 mutations are retired (clean-break migration, per the engine spec).
This roster replaces them.

### Design principles (from the design sessions)

1. **A mutation is a body change, never a skill/command.** Offense comes from
   *passive properties*, *on-hit effects*, or *verb-enhancements* of existing
   actions — never a new button, unless it's a rare active (below).
2. **Actives are rare and powerful.** Any mutation-granted active command shares
   the ONE special-move cooldown with all shouts and spellcasting (via
   `actions.mutationPreamble`), so they must be few, high-rarity, and clearly
   worth the slot. The entire roster has **only two** actives (Venom Coat,
   Cocoon).
3. **Mostly passive; few triggered states.** Triggered states are reserved for
   genuinely distinct moments (Blood Frenzy). No redundant "worse-than-apex"
   states.
4. **Weird, biological body-horror.** Chitin, spores, venom, reflective/elemental
   skin, translucency, compound eyes, brood sacs — the Chrysalis is a native
   organism; mutations look like it.
5. **Pole opposition is the cost of depth.** Deep Body degrades the Conviction
   pool (chokes spells/taunt/summons); deep Belief degrades gear/body-weapon
   effectiveness. Hybrids and the Generalist stay shallow on both → no penalty.

---

## 2. The ring & poles

Ring order (each cluster shares one bridge with each ring-neighbor):

```
Colossus ─ Ironhide ─ Zealot ─ Manifester ─ Ethereal ─ Weaver ─ Trickster ─ Stalker ─ Ravener ─(Colossus)
 body       body      belief    belief       belief    hybrid    hybrid      body      body
```

- **Body pole** (`pole: body`): Ravener, Colossus, Ironhide, Stalker.
- **Belief pole** (`pole: belief`): Ethereal, Manifester, Zealot (Conviction-based).
- **Hybrids** (`pole: ""`): Trickster, Weaver — deliberately shallow on both.
- **Center** (`pole: ""`, `clusters: [generalist]`): neutral enablers.

---

## 3. Cluster rosters

Each keystone lists: **primitive** (implementation category, see §6), **rarity**,
and effect. Apex ★ is an always-on transformation gated by a prereq spine.

### Ravener — feral unarmed martial (Body · Str/Dex)  *(locked in engine design)*
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Rending Claws | on-hit buff | 3 | unarmed strikes open a stacking bleed |
| Raptor Legs | verb-enhance (`kick`) | 4 | kicks hit hard + raking hamstring; knockdown at depth |
| Blood Frenzy | triggered state | 6 | on foe-bleed / self <50% HP: faster attacks, immune to taunt/fear, can't retreat; lifesteal + terror at depth |
| **Apex Form ★** | transformation | 9 | always-on predator: body-weapon power, terror aura, devour execute — *prereq: Rending Claws + Raptor Legs* |

Bridges: **Extra Arms** (⇄Colossus), **Venom Glands** (⇄Stalker).

### Colossus — size & smashing might (Body · Str)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Dense Muscle | passive | 3 | Strength + carry power surge |
| Ossified Frame | passive (control-immune) | 5 | immune to trip, knockback, grapple, shove |
| Titan Growth | passive (size) | 7 | huge HP + reach, slower & easier to hit *(conflicts Hollow Bones)* |
| **Colossus Form ★** | transformation | 9 | always-on giant: blows cleave adjacent, hurl foes — *prereq: Dense Muscle + Titan Growth* |

Bridges: **Extra Arms** (⇄Ravener), **Chitin Plating** (⇄Ironhide).

### Ironhide — retaliation tank (Body · Vit)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Thick Hide | passive (armor) | 3 | flat natural armor |
| Reflect Skin ⚑ | passive (reflect) | 5 | **pick one flavor, they conflict:** Barbed (physical) / Molten (burn, resist cold) / Frostbite (cold + slow) / Voltaic (shock + chain) |
| Regrowth | passive (regen) | 6 | heal mid-fight; shrug off crippling |
| **Living Carapace ★** | transformation | 9 | plated shell: huge armor, **amplifies your Reflect Skin**, pulls aggro, near-immovable — *prereq: Thick Hide + Regrowth* |

Bridges: **Chitin Plating** (⇄Colossus), **Resonant Larynx** (⇄Zealot).
Bench alt-armor (conflicting variants, author if desired): Bark Skin (armor, fire-weak), Segmented Body (crit-resist), Caustic Blood (acid on wound), Spore Bloom (poison haze).

### Stalker — stealth & venom (Body · Dex/Per)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Padded Soles | passive (sneak) | 3 | strong hide/sneak bonus (silent movement) |
| Compound Eyes | passive (sensory) | 4 | see hidden; cannot be ambushed/flanked; harsh light dazzles |
| Venom Coat | **active** | 7 | slick weapons in venom: +damage/hit for ~15 rounds (longer per rank). Shared cooldown. |
| **Chameleon Skin ★** | transformation | 9 | near-invisible in shadow; first strike each fight crits; execute the unaware — *prereq: Padded Soles + Compound Eyes* |

Bridges: **Venom Glands** (⇄Ravener), **Veiling Musk** (⇄Trickster).

### Ethereal — incorporeal & psychic (Belief · Wil/Per)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Ether Gland | passive (spell power) | 4 | spellcasting power & efficiency rise markedly |
| Second Sight | passive (sensory) | 5 | perceive intent early + see hidden; dodge/parry bonus, never surprised |
| Kinetic Backlash | passive (reflect + knockdown) | 7 | when struck, a telekinetic recoil with high chance to knock the attacker flat |
| **Discorporation ★** | transformation | 10 | all but bodiless: physical attacks mostly miss, gear ornamental, will-damage surges — *prereq: Ether Gland + Second Sight* |

Bridges: **Spirit Tether** (⇄Manifester), **Cocoon** (⇄Weaver).

### Manifester — brood & summoning (Belief · Cha/Wil)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Brood Sac | companion (passive) | 4 | host & birth a bonded spawn — always have a companion creature |
| Symbiotic Bond | companion (passive) | 5 | your regen/buffs bleed into companions; they hit harder |
| Hive Mind | companion (passive) | 6 | extra companion slot + coordinated action |
| **Brood Mother ★** | transformation + companion | 9 | continuously spawn & sustain a swarm; fallen spawn replaced — *prereq: Brood Sac + Hive Mind* |

Bridges: **Spirit Tether** (⇄Ethereal), **Beast Bond** (⇄Zealot).

### Zealot — voice, presence & party command (Belief · Cha)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Commanding Presence | aura | 4 | standing aura: allies in room gain hit + morale |
| Booming Lungs | passive (shout amp) | 5 | your shouts/taunts/rallies reach the whole room and land harder |
| Zealous Conviction | passive | 5 | heavier Conviction-channel damage + stickier taunts; resist demoralize |
| **Radiant Avatar ★** | transformation + aura | 9 | constant party-wide buff aura, immune to fear, impossible to ignore — *prereq: Commanding Presence + Zealous Conviction* |

Bridges: **Resonant Larynx** (⇄Ironhide), **Beast Bond** (⇄Manifester).

### Trickster — illusion-caster / rogue (Hybrid · Dex/Wil)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Quicksilver Nerves | passive | 4 | nimble Dexterity + unusually fast casting (the dabbler) |
| Evil Eye | on-target debuff | 5 | a baleful eye — foes you fight suffer sagging accuracy & defenses |
| Corvid Brain | passive (spell power) | 6 | illusion & mind (Mental-school) magic lands harder and costs less |
| **Translucent Body ★** | transformation | 8 | glassy see-through flesh deepening toward near-invisibility — foes can barely target you — *prereq: Quicksilver Nerves + Corvid Brain* |

Bridges: **Veiling Musk** (⇄Stalker), **Silk Glands** (⇄Weaver).

### Weaver — control, web & disrupt (Hybrid · Wil/Per)
| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Sticky Secretion | passive (retaliation) | 4 | adhesive skin: attackers get stuck — slowed, swings drag |
| Dissonance Organ | aura (anti-cast) | 5 | disruptive resonance: enemies near you may fumble spells & shouts |
| Grasping Tendrils | on-hit (root) | 6 | barbed tendrils erupt from your strikes and root the target |
| **Paralytic Field ★** | transformation + aura | 9 | constant field of silk & dissonance: nearby enemies slowed, snared, and struggle to cast — *prereq: Sticky Secretion + Grasping Tendrils* |

Bridges: **Silk Glands** (⇄Trickster), **Cocoon** (⇄Ethereal).

---

## 4. The 9 bridges (shared nodes)

Each belongs to **both** neighbors (`clusters: [a, b]`), so it draws affinity
from either and stitches the ring together.

| Bridge | Clusters | Primitive | Rarity | Pole | Effect |
|---|---|---|---|---|---|
| Extra Arms | colossus, ravener | passive (slots + strikes) | 9 | body | extra weapon/unarmed strikes; more equip slots (existing) |
| Chitin Plating | colossus, ironhide | passive (armor) | 6 | body | heavy exoskeleton armor + Strength, lower mobility |
| Venom Glands ✦ | ravener, stalker | on-hit buff | 7 | body | any natural strike/ambush stacks neurotoxin → slow → paralysis |
| Veiling Musk | stalker, trickster | passive (detection) | 4 | "" | masking pheromones — passively much harder to detect |
| Silk Glands | trickster, weaver | passive (snare/escape) | 4 | "" | produce silk: passively snare adjacent foes; escape-lines |
| Cocoon ✦ | ethereal, weaver | **active** | 8 | belief | encase self: near-invulnerable + drop aggro (vanish from threat) |
| Spirit Tether | ethereal, manifester | companion (passive) | 6 | belief | a bonded-spirit organ channels/strengthens a companion |
| Beast Bond | manifester, zealot | companion (passive) | 6 | belief | extend buffs & commands to your companions |
| Resonant Larynx | ironhide, zealot | passive (shout stacking) | 6 | "" | layered vocal cords: loose multiple shout-effects (rally+war cry+taunt) in one action |

✦ = apex bridge (heavy, rare).

---

## 5. The Center — Generalist hub

`clusters: [generalist]`, `pole: ""` (no opposition penalty). The on-ramp
everyone builds in; deep here = the flight generalist.

| Keystone | Primitive | Rarity | Effect |
|---|---|---|---|
| Hollow Bones | passive | 2 | light, fast, frailer *(conflicts Titan Growth; flight prereq)* |
| Prehensile Tail | passive (tail slot) | 3 | balance + free grasp; aerial control *(flight prereq)* |
| Keen Senses | passive | 3 | dodge, initiative, spotting |
| Rapid Healing | passive (regen) | 3 | strong out-of-combat recovery |
| Thick Coat | passive | 3 | fur/insulation — resist cold & harsh environments |
| Tremorsense | passive (sensory) | 4 | detect hidden & burrowed foes |
| Precognition | passive | 4 | half-second early — bonus dodge/parry |
| Spiracle Lungs | passive | 4 | deep stamina, gas/poison immunity, long underwater breath |
| **Winged Flight ★** | transformation | 8 | fly: cross rooms, escape melee, strike from angle, dodge earthbound — *prereq: Hollow Bones + Prehensile Tail* |

---

## 6. Mechanic-primitive inventory (implementation categories)

Every keystone maps to one of these. This is the **build backbone** — the waves
build the primitives, then author YAML against them.

| # | Primitive | Reuse vs new | Keystones using it |
|---|---|---|---|
| P1 | **Passive property/flag effects** — extend the mutation effect-type system + consumers (control-immunity, reflect-damage typed, see-hidden/anti-ambush, spell-power, crit-resist, detection/stealth, shout-amp, shout-stacking, spell-fumble-aura source) | mostly NEW effect types + consumer code | the majority (Dense Muscle, Ossified Frame, Thick Hide, Reflect Skin×4, Ether Gland, Second Sight, Kinetic Backlash, Compound Eyes, Corvid Brain, Booming Lungs, Resonant Larynx, Chitin, Veiling Musk, Silk Glands, all Center enablers, …) |
| P2 | **On-hit buff application** — combat hook: attacker's mutation applies a buff to the target (bleed/venom-stack/root/curse) | NEW combat hook, data-driven | Rending Claws, Venom Glands, Grasping Tendrils, Evil Eye |
| P3 | **Verb-enhancement** — a command checks the mutation and alters its effect | REUSE pattern (kick variants, tailsweep) | Raptor Legs (`kick`) |
| P4 | **Active-ability command** — new command via `mutationPreamble` (shared cooldown) | REUSE pattern | Venom Coat, Cocoon |
| P5 | **Triggered-state buff** — condition → state with combat effects | NEW state + hooks | Blood Frenzy |
| P6 | **Transformation apex** — always-on buff bundle + progressive pole choke; some bespoke effects (cleave, execute, incorporeal-miss, swarm, party-aura) | framework NEW; pole-choke reuses engine | all 9 apexes + Winged Flight |
| P7 | **Aura** — room-scan applying ally-buff / enemy-debuff each tick | NEW aura system | Commanding Presence, Dissonance Organ, Radiant Avatar, Paralytic Field |
| P8 | **Companion/brood** — persistent companions, slots, respawn | EXTEND existing companion/manifestation system | Brood Sac, Symbiotic Bond, Hive Mind, Brood Mother, Spirit Tether, Beast Bond |
| P9 | **Flight** — positioning/dodge/escape mechanic | NEW, bespoke | Winged Flight |

---

## 7. Conventions

- **Rarity → depth threshold.** Rarity drives the engine's affinity gate
  (`depthThreshold = rarity × MutationAffinityPerRarity`). Entry ≈ 2–4, core ≈
  5–7, apex ≈ 8–10. Bridges 4–9.
- **Prereq spines.** Each apex requires its two core keystones (listed per
  cluster). Flight requires Hollow Bones + Prehensile Tail. Authored via the
  engine's `prerequisites` field; boot-validated.
- **Pole tags** per §2/§4. Apexes carry their cluster's pole so they dominate
  pole-depth (deep commitment = the opposition bites).
- **Conflicts.** Reflect Skin's four flavors conflict each other; Titan Growth ⟷
  Hollow Bones; Colossus/Ironhide heavy-body vs the incorporeal/translucent set
  (cross-pole opposition largely handles this, plus explicit `conflicts` for the
  obvious physical/ethereal contradictions).
- **No hard numbers in player text**; descriptive language only (per repo SOP).
- **Naming/filenames** follow `ConvertForFilename`; new effect types documented
  in `docs/schemas/`.
- **Per-rank magnitudes are NOT yet defined (deferred).** This spec fixes each
  mutation's *identity, primitive, rarity, and prereqs* — but **not** the numeric
  strength of ranks 1–4 (how much bleed, how much armor, how large each reflect,
  how long Venom Coat lasts per rank, the exact deepening curve). Those values are
  authored during implementation and tuned in the post-build playtest (Wave 6 /
  §10). Deepening should always escalate the *effect*, not just a decimal — a
  design rule the numbers must honor.
- **Drift-signal coverage (engine addition).** The engine's `skillClusters` map
  currently covers Colossus/Ravener/Stalker/Ethereal/Zealot/Manifester. **Ironhide**
  (damage-absorbed signal), **Trickster**, and **Weaver** (mixed Dex/Wil/Per) need
  signal entries added, or those clusters are only reachable via phials — not
  emergent drift. Fold this small `internal/mutations/graph.go` + combat-hook
  addition into Wave 1.

---

## 8. Roster size

~**54 keystones** (9 clusters × 4 own + 9 bridges + 9 Center) + 3 extra Reflect
variants ≈ **57 mutation files**. This is *more files* than the retired 41, but
every one is a build-defining keystone (not a stat tweak), and a single build
only acquires ~6–10 of them (gated by affinity + prereqs + pole). "Fewer" was
always about **per-build density and impact**, not total catalog size.

---

## 9. Implementation decomposition (waves → separate plans)

This is a large program; it will be **multiple plans**, not one. Suggested waves
(each its own plan, each independently testable):

1. **Wave 1 — Passive effect-types (P1).** Extend the mutation effect system with
   the new passive types + their combat/perception consumers. Unlocks the bulk of
   the roster. Ship with a few real passive mutations tagged (e.g. Colossus,
   Center) to make drift observable.
2. **Wave 2 — On-hit + verb-enhance (P2, P3).** The combat on-hit-buff hook and
   the `kick` enhancement; author Ravener + Stalker + Weaver on-hit content.
3. **Wave 3 — States + transformations (P5, P6).** Blood Frenzy + the
   transformation-apex framework (composing passives + pole-choke + bespoke
   effects); author all apexes.
4. **Wave 4 — Auras + companions (P7, P8).** The aura tick system + companion
   extensions; author Zealot, Weaver auras, Manifester brood.
5. **Wave 5 — Actives + Flight (P4, P9).** Venom Coat, Cocoon, Winged Flight.
6. **Wave 6 — Full authoring + migration + smoke.** Complete every YAML with
   `clusters`/`pole`/`prerequisites`/conflicts, re-curate `archetype_pull` → cluster
   tags for mobs, run the clean-break migration + free re-bloom, helpfiles, and a
   full playtest/balance pass.

Balance tuning (rarities, thresholds, effect magnitudes) is deferred to the
post-build playtest per the repo SOP.

## 10. Out of scope

- Engine changes already shipped (affinity, gating, pole-decay) — reused, not
  re-specified.
- Final numeric balance (post-build playtest).
- Underwater zones (Spiracle Lungs' breath-holding is forward-compatible only).
- Phials + moon-crash + Bloom interplay (separate follow-on per the engine spec).
