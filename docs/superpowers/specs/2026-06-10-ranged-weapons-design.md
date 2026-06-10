# Ranged Weapons System — Design Spec

**Date:** 2026-06-10
**Status:** Approved pending user review (design-only deliverable; implementation scheduled separately)

## Goal

Give DOGMud a complete ranged-combat system with a distinct identity: the
**loaded-weapon model**. Firing is an immediate, deliberate, hard-hitting
action; reloading consumes the shared special-move cooldown. Hand crossbows
and primitive pistols ride in the offhand for hybrid builds; bows and
arbalests anchor dedicated archer builds governed by a revived
`ranged-combat` skill and the Perception stat. Mobs get full archer AI
(fire, reload, kite, hide-to-cover).

## Decisions (settled during brainstorm)

| Question | Decision |
|---|---|
| Core identity | Loaded-weapon model: instant fire, reload on the shared special-move cooldown, loaded state persists out of combat. NOT a positioning/battlefield sim |
| Offhand hybrids | Yes — one-handed ranged weapons (hand crossbow, primitive pistol) equip in the offhand beside a melee main |
| Ammunition | Ammo bundles: one inventory item per ammo type using the existing `Uses` mechanic; reload decrements by one; wrong ammo type doesn't fit |
| Skill & stat | Un-retire `ranged-combat` as the 10th skill; Perception governs both hit and damage |
| Mob usage | Full archer AI: fire, reload, kite to adjacent rooms, hide while reloading |
| Balance posture | Per-shot damage must be LARGE — see Balance Math. Ballpark melee parity, discounted for cross-room safety |
| Crime integration | Cross-room shots MUST flow through aggro/witness/crime/guard-justice systems — explicit test requirement |

## What already exists (build on, don't duplicate)

- `shoot <target> <direction>`: cross-room target resolution + remote aggro
  (`internal/actions/combat_shoot.go`, `Character.SetAggroRemote`,
  `characters.Shooting` aggro type) — actor-parity, charm/non-combatant
  guards, sneak interplay.
- `shooting` weapon subtype; one item (Training Bow 10004, 2h,
  damage_multiplier 0.30, grapplemodifier 0.2).
- `CategoryHitRanged` messaging category (colors wired).
- `throw` (grenades/throwables) already uses the special-move cooldown —
  the shared-cooldown precedent.
- `ranged-combat` skill is RETIRED: stripped by `validate.go:286` with
  tests asserting the strip — un-retiring reverses this deliberately.
- Shooting weapons currently train `weapon-combat` via
  `CombatSkillTagForItem` (characters/skills.go:253).

## Core loop

1. **Fire** — `shoot <target>` (same room) or `shoot <target> <direction>`
   (adjacent room). Requires a loaded ranged weapon (main or offhand).
   Resolves IMMEDIATELY as a single opposed-roll attack (not queued into
   the auto-attack round). The weapon becomes unloaded.
2. **Reload** — `reload` consumes the shared special-move cooldown
   (`Balance.SpecialMoveCooldown`, currently 4 rounds, same pool as
   kick/trip/bash/throw) and one ammo from a matching bundle in
   inventory. Fails cleanly with a message if no matching ammo.
3. **Loaded persistence** — loaded state survives logout/zone
   changes/combat end. The classic opener: walk in loaded, first shot
   free, every later shot costs special-move economy.
4. **Unloaded melee fallback** — a main-hand ranged weapon still
   auto-attacks each round as weak melee (existing low damage_multiplier
   models clubbing someone with a bow). An offhand ranged weapon
   contributes nothing to melee rounds.

## Balance math (the 4-round reality)

A dedicated 2h-ranged user's cycle is: fire (round 1) → reload occupies
the special-move cooldown (~4 rounds) → fire again ≈ **one shot per ~5
rounds**, while giving up ALL special moves. A melee fighter in those same
5 rounds lands up to 4 swings/round × weapon multiplier AND keeps
kick/trip/bash.

Reference numbers (stat 100, rank 0, itemMult 1.0): melee raw ≈ 30/swing ×
~2 average swings/round × 5 rounds ≈ **300 raw per cycle**. Therefore:

- **Per-shot raw target:** a 2h ranged shot should land roughly **60–75%
  of the equivalent melee cycle** (≈ 180–220 raw at baseline) — huge per
  hit, discounted from full parity because the shooter can fire from an
  adjacent room (safety), open every fight pre-loaded (free alpha), and
  retains weapon-slot flexibility. One-handers land roughly half that
  (hybrid builds also keep a full melee main-hand).
- **Mechanism:** ranged `damage_multiplier` values are simply LARGE
  (e.g. arbalest ~7.0, hunting bow ~5.5, hand crossbow ~3.0, pistol
  ~3.5 — pistol > hand crossbow, louder/cruder), flowing through the
  standard pipeline unchanged. A global `Balance.RangedShotScale`
  (default 1.0) multiplies all ranged shots as the single tuning knob.
- **Offhand note:** the offhand hand-crossbow is NOT free extra DPS — its
  reload competes with the same special-move cooldown the melee main
  would spend on kick/trip/bash. It's a trade, not a stack.
- Final numbers are playtest-tuned; the spec commits to the TARGET RATIO
  (ballpark melee parity, 60–75% cycle damage for safety discount), not
  the literal constants.

## Resolution

- **Attack roll:** `dice.OpposedRollStat(attacker Perception-based score,
  defender defense score)` with the ranged-combat skill folded in exactly
  as melee folds weapon skills. Z-score ≥2 crit / ≤-2 fumble as
  everywhere (fumble flavor: misfire/string snap — no self-damage v1).
- **Defenses: dodge and block apply; parry does NOT.** You can't parry a
  bolt; a shield can stop one. Best-of-all defense resolution simply
  excludes parry for ranged attacks. `MinDefenseChance` floor applies.
- **Damage:** `combat.CalcRawDamage(Perception, rangedRank,
  weapon.DamageMultiplier × RangedShotScale, Physical)` → physical
  mitigation → `dice.RollStat` variance. `GetDamageDescription` tiers in
  messages (no hard numbers).
- **Cross-room shots:** same resolution. Visibility gates apply (you must
  be able to see into the target room — darkness blocks targeting).
  Message routing: shooter sees the shot; target room sees impact (and
  direction it came from); shooter's room sees the loosing. All via
  `CategoryHitRanged` (verbosity gate: ranged hits are already in the
  suppressible-at-light table; the fire action is deliberate, consider
  exempting own-fire from light suppression at implementation time).
- **Progression:** `OnSkillUse(ranged-combat)` + `OnStatUse(perception)`
  per fire; crit callbacks as melee.

## Weapons & ammo

ItemSpec additions:
- `ammo_tag: arrows | bolts | shot` on ranged weapons.
- Ammo bundles: regular items with `Uses: N` and a matching
  `ammo_tag` field (e.g. "quiver of arrows", Uses 20). Reload consumes
  one Use; empty bundles are consumed/destroyed.
- Loaded state: a runtime bool on the item INSTANCE (like Uses), persisted
  in saves.
- Min-Strength wield requirement on heavy bows/arbalest (flavor +
  build gating).

V1 content (~6 weapons + 3 bundles, vendor-stocked; IDs via
`id_inventory.py`):

| Weapon | Hands | Ammo | Role |
|---|---|---|---|
| Sling | 1 | shot | newbie/cheap, lowest mult |
| Hand crossbow | 1 | bolts | offhand hybrid |
| Primitive pistol | 1 | shot | offhand hybrid, loudest, slightly stronger |
| Training bow (exists) | 2 | arrows | starter, mult raised to fit the new scale |
| Hunting bow | 2 | arrows | mainline archer |
| Arbalest | 2 | bolts | heaviest hit, min-Str gate |

## Skill revival: ranged-combat

- Remove from the dead-skill strip list (`validate.go:286`) and its
  strip-assertion tests; register as the 10th skill (soft cap 50, standard
  `OnSkillUse` probabilistic progression; progression multiplier tuned
  alongside salvage's 2.0 precedent).
- `CombatSkillTagForItem` returns `ranged-combat` for `shooting`-subtype
  weapons (affects the weak unloaded-melee swings too — acceptable:
  clubbing with your bow is still "knowing your weapon").
- Helpfile + `skills` display + CLAUDE.md damage-pipeline table update
  (the "9 skills" references become 10).
- Migration: existing characters simply start at rank 0 (the old stripped
  ranks are long gone — acceptable, the skill was dead).

## Archer AI (full)

Behavior-tree additions following the beast-move/special-move delegation
pattern (predator/generic_fighter precedent):

- **`try_fire`** — loaded weapon + valid target (same room, or remote
  target via existing mob shoot parity) → fire.
- **`try_reload`** — unloaded + not melee-engaged → reload (mob reloads
  obey the same special-move cooldown bookkeeping).
- **`keep_distance`** — melee-engaged + healthy → withdraw through an
  exit (existing flee/pathing plumbing), then cross-room fire back
  through the exit via remote aggro. This is the kiting loop, symmetric
  with player kiting.
- **"Take cover" = the hidden mechanic, not a cover sim** — archers with
  skullduggery hide while reloading (existing hidden-mob rules + the
  existing Perception+Search vs Dex+Skullduggery detection on room entry
  apply). No new cover system.
- New `archer` btree archetype wired like the beast-move delegation;
  goal-weights so the 4.2 planner doesn't fight the kiting behavior.
- Content: 2–3 archer mobs (e.g. Thornwall crossbowman guard, a marsh
  hunter) with ranged loadouts + ammo.

## Crime, witnesses, and justice (explicit requirement)

Shooting a mob from an adjacent room MUST behave like an assault, not a
free action. Required behaviors, each with a test:

1. **Target retaliation:** the shot mob aggros the shooter and paths
   toward them (damage → `TrackPlayerDamage` + `mob_hurt` btree event —
   verify these fire on the cross-room damage path, not just same-room).
2. **Witness classification:** witnesses in the TARGET's room process the
   assault through the existing witness seeders (guard→report,
   noncombatant→alarm, combatant→revenge). The attacker's identity
   carries across the room boundary when the shot is visible/attributable;
   shooting from hiding interacts with the existing sneak rules
   (unattributed assault = the existing unknown-assailant handling).
3. **Guard justice:** guards who witness (or receive a report of) a
   cross-room shooting attempt to track down and warn/arrest the shooter
   through the existing town-justice flow — the shooter being in the next
   room must not exempt them.
4. **Crime recording:** whatever `recordAssaultCrime`-style bookkeeping
   the melee attack path performs must also fire on the shoot path.

The live smoke for implementation must include: shoot a guard's protected
NPC from the next room → guard responds; shoot a civilian with witnesses →
report/alarm fires; shoot from hiding → unattributed path.

## Player UX

- `shoot <target>` / `shoot <target> <direction>` (existing command,
  extended to same-room + loaded checks). `fire` aliases to `shoot`.
- `reload` — new command (player) + mob parity action.
- `look`/inventory show loaded state ("(loaded)" tag on the item line);
  examining a ranged weapon reports its ammo type.
- Helpfiles: `shoot` (update), `reload` (new), `ranged-combat` skill.
- Trying to shoot while unloaded → clear message naming the reload
  command; reload without ammo → names the ammo type needed.

## Out of scope (deferred)

- Range bands / cover simulation / battlefield mini-map.
- Fletching/gunsmithing crafting recipes (ammo is vendor-bought v1; a
  crafting follow-up pairs naturally with the existing salvage and
  materials economy).
- Volley / aimed-shot / called-shot special moves.
- Weather/visibility accuracy modifiers (tempting with weather now live;
  balance later).
- Mounted archery; ammo recovery from corpses.

## Implementation sequencing sketch (for the future plan)

1. Item plumbing: ammo_tag, loaded state, ammo bundles, reload command.
2. Fire resolution path (same-room first, then unify cross-room shoot).
3. Skill revival + Perception governance.
4. Balance pass to the target ratio (RangedShotScale knob).
5. Crime/witness/justice integration + tests (the explicit requirement).
6. Archer AI btree + archetype + mobs.
7. Content (weapons/ammo/vendors) + helpfiles + live smoke.
