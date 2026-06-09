# Non-human attack types & beast moveset — design

**Date:** 2026-06-09
**Status:** Approved (brainstorming) — pending spec review → writing-plans
**Scope note:** Sub-project 2 of the immersion-polish pair. Sub-project 1
(game-wide naming casing) shipped separately.

## Problem

Non-human creatures fight like humans. A wolf "punches" and "grapples": its
basic attacks render through the generic human message set, and the combat AI
lets it use humanoid technique moves (grapple/trip/bash) that make no sense for
a quadruped. We want creatures to attack with anatomy-appropriate basic attacks
AND a logical, expanded set of beast special moves, while human technique moves
are gated away from beasts.

### Root causes (from code investigation 2026-06-09)

- **Basic-attack messaging:** `combat.buildWeaponSetup` hardcodes
  `weaponSubType = items.Unarmed` for any character without a weapon, so every
  unarmed attacker (all ~100 non-human mobs) falls through to `generic.yaml`.
  Species carry `unarmedname` ("jaws", "talons") for the `{itemname}` token, but
  nothing maps a species to an attack **message subtype**.
- **Special moves:** `combat.ChooseSpecialMove` only selects bash/trip/kick/
  grapple/submit. The implemented beast moves `bite` and `hamstring`
  (`actions/combat_bite.go`, `combat_hamstring.go`) are **never selected by AI**
  — no `CanUseBite`/`CanUseHamstring`. There is no anatomy gating: a wolf can
  grapple. Existing gates: `GrappleImmune`, `NaturalBash` (species flags),
  `tail` mutation reskins trip→tailsweep.
- Species already declare accurate `body_parts` (humanoids: `arms, hands, …`;
  beasts: `legs, eyes, mouth, skin, tail`; oozes: `[skin]`/`[]`). No species
  declares `horns`/`claws` yet.

## Decisions (from brainstorming)

1. **Full fix:** anatomy-appropriate messaging AND species-gated special moves
   (the "punch and grapple" pair).
2. **Definition home:** the natural attack lives on the **species** (with an
   optional per-mob override).
3. **Gating model — hybrid, both "logical":**
   - **Human technique moves** (grapple, bash, submit) require the `arms` body
     part. Beasts (no arms) can't use them. (`trip`/`kick` require `legs`.)
   - **Beast special moves** are gated on the species' **`natural_attack`
     identity** (not raw `mouth`/`legs`, since humans have those too and must
     not bite/maul): fanged→bite/maul, clawed→rake, horned→gore.
4. **Expansion depth:** wake the dormant `hamstring` move; **retire the `bite`
   special** (biting is now the Layer-1 basic attack for fanged creatures); and
   add new beast specials — **`throttle`** (the fanged finisher: clamp the
   throat / cut off airflow), **pounce, maul, rake, gore**.

## Architecture — two independent layers

### Layer 1 — Natural-attack messaging (basic attacks)

- Add `natural_attack string` to the species spec (`internal/species`): the
  message **subtype** a creature's unarmed basic attacks render through. Values
  are existing combat-message subtypes: `bite`, `claws`, `slam`, `gore`,
  `sting` (and `unarmed` = the humanoid default/empty). Authored per species
  (canine→`bite`, feline→`claws`, bear→`claws`, serpent→`bite`, insectoid→
  `sting`, raptor→`claws`, deer→`slam` (hooves), …). Empty/`unarmed` → humanoids
  keep `generic`.
- In `combat.buildWeaponSetup`, when no weapon is equipped, resolve the subtype
  from (mob override → species `natural_attack` → `Unarmed`). `unarmedname`
  continues to fill the `{itemname}` token.
- **Per-mob override:** optional `natural_attack:` on the mob YAML.
- **Message files:** reuse existing `bite`/`claws`/`slam`/`gore`/`sting`. During
  implementation, audit each species' chosen subtype for coverage; author a new
  message file ONLY if a needed subtype has none (document any added).

### Layer 2 — Special-move gating + beast moveset

**2a. Gate human technique moves by anatomy.** In the `CanUse*` gates
(`combat/ai.go`) and the action entry points, require:
- `grapple`, `bash`, `submit` → species has `arms` (keep `GrappleImmune`/
  `NaturalBash` edge cases).
- `trip`, `kick` → species has `legs`.

A no-arms beast therefore can never grapple/bash — the core "wolves doing BJJ"
fix, driven by existing `body_parts` data.

**2b. Beast moveset — wake + expand.** Gated on the species `natural_attack`
identity (a species is "fanged" if `natural_attack==bite`, "clawed" if
`==claws`, "horned" if `==gore`), plus body parts where noted. Add `CanUse*` +
`Score*` for each and weight them in AI profiles so beasts actually use them.

| Move | Status | Mechanic (follow existing ExecuteSkillMove patterns) | Gate | Message |
|------|--------|------------------------------------------------------|------|---------|
| `throttle` | NEW (replaces bite special) | throat clamp / cut off airflow: a held, escalating choke — drains the victim's stamina and blocks shouting & spellcasting (no airflow); damage/effect ramps each round until the victim escapes. The beast analog of the humanoid chokeout/submit; best set up by a knockdown (`pounce`). | fanged (`natural_attack==bite`) | new `throttle.yaml` |
| `hamstring` | wake (exists) | ~25% + bleed; slows | fanged or clawed, + `legs` | bleed/`claws` |
| `pounce` | NEW | leap opener: knockdown + bonus dmg; only from non-grappled, opening rounds | quadruped predator: `legs` + (`natural_attack` in {bite,claws}) | new `pounce.yaml` |
| `maul` | NEW | savage flurry: high dmg + bleed | fanged (`natural_attack==bite`) | new `maul.yaml` |
| `rake` | NEW | claw rake: dmg + bleed | clawed (`natural_attack==claws`) | reuse `claws.yaml` or new `rake.yaml` |
| `gore` | NEW | charge strike: dmg + knockback | horned (`natural_attack==gore`), add `horns` body part to horned species | `gore.yaml` (exists) |

**Bite special retired.** Because fanged creatures now bite as their *basic*
attack (Layer 1), a "bite harder" special is redundant. Retire the existing
`bite` special move — `internal/actions/combat_bite.go`, its command
registration, and its tests — and replace it with `throttle` (above). Biting is
the basic attack; `throttle` is the distinct fanged finisher.

New moves follow the established `actions.ExecuteSkillMove` / tailsweep-variant
pattern (Execute* in `internal/actions/`, `CanUse*`+`Score*` in `combat/ai.go`,
optional mobcommand wrapper). Use multipliers, no flat values; no raw numbers in
player text; route damage through the unified pipeline (per CLAUDE.md). The
`throttle` hold/escalation can follow the grapple/submit control precedent for
the "held until escape" mechanic, but gated on `natural_attack` (fanged), not on
the arms-based grapple system.

**2c. AI selection.** A mob's effective special set = (moves its profile
weights) ∩ (moves its anatomy/`natural_attack` permits). Add beast moves to the
AI profile weight tables, and add 2-3 beast-oriented profiles (e.g.
`predator`, `ambush_predator`, `brute`) assigned to beast mobs as appropriate.
Humanoid profiles keep grapple/bash/etc.; body-part gating filters automatically
so no humanoid ever bites/mauls and no beast ever grapples.

## Data & content

- **Species YAML:** add `natural_attack`; audit/extend `body_parts` (add `horns`
  to horned species for gore; confirm `arms` only on humanoids; confirm beasts
  have `mouth`/`legs` as appropriate). ~44 species.
- **Mob YAML:** optional `natural_attack` override; assign beast AI profiles to
  the ~100 non-human mobs where the default profile isn't apt.
- **New message files:** `pounce`, `maul` (and `rake` if not reusing `claws`);
  authored in the existing combat-message format (prepare/miss/weak/normal/
  heavy/critical/fumble × toattacker/todefender/toroom × beginner/expert/master).

## Hardening / validation

- Load-time validation (panic, house style): a species `natural_attack` (and any
  mob override) must be a known message subtype; a `natural_attack` implying a
  body part (e.g. `gore`→`horns`) should have that body part — warn/panic on
  mismatch so authoring stays logical.
- Keep the gating predicates in one place (a small helper, e.g.
  `combat.canUseBeastMove(char, move)`) so the anatomy/natural-attack rules are
  the single source of truth, mirrored by tests.

## Testing

- `buildWeaponSetup`: unarmed canine → `bite` subtype; unarmed human → generic;
  mob override wins over species.
- Gating: canine (no arms) → `CanUseGrapple/Bash` false; `CanUseBite` true;
  human → grapple true, bite false; horned → gore true; ooze (`[]`) → only basic.
- New moves: each Execute* applies its effect (knockdown/bleed/airflow-choke/
  etc.) and is AI-selectable for a permitted species; message renders the right
  file. `throttle` holds + escalates and blocks shout/cast until escape.
- AI: a beast mob selects only permitted beast moves; a humanoid only humanoid
  moves; no humanoid bites/throttles, no beast grapples.
- Smoke: a wolf's basic attacks read as bites/claws (not punches), it
  pounces/throttles/hamstrings and never grapples; a bear mauls; a humanoid
  still grapples/bashes; targeting unaffected.

## Rollout order (for the plan)

1. Layer 1: `natural_attack` field + `buildWeaponSetup` wiring + species tagging
   + tests. (Ship-able alone: fixes "wolves punch".)
2. Layer 2a: anatomy gating of human technique moves + tests. (Fixes "wolves
   grapple".)
3. Layer 2b: wake `hamstring` into AI and retire the `bite` special; then add
   `throttle`, pounce, maul, rake, gore (one move per increment: Execute +
   CanUse/Score + message file + tests).
4. AI profiles + beast-profile assignment to mobs.
5. Validation + full smoke + push per SOP.

## Out of scope

- Sub-project 1 (naming casing) — shipped.
- Player-facing beast moves for player characters who acquire beast mutations —
  the moves are anatomy/`natural_attack`-gated, so a mutated player could
  qualify, but tuning that is a follow-up, not this sub-project's focus.
- Ranged/throwing creatures, flight mechanics (no `wings` body part work here).

## Risks / watch-items

- **Humanoids biting / beasts grappling:** the whole design hinges on the
  gating predicates. Gate human moves on `arms`, beast moves on `natural_attack`
  identity — NOT on raw `mouth`/`legs`. Tests pin both directions.
- **AI never picking beast moves:** if profiles don't weight them, waking the
  moves does nothing. Verify via the AI-selection test + smoke.
- **Message-file gaps:** a species mapped to a subtype with no file falls back
  to generic — defeats the point. The Layer-1 audit + validation must catch it.
- **body_parts accuracy:** gating is only as good as the data; audit beasts that
  currently have `arms` by mistake (would let them grapple) or lack `mouth`.
