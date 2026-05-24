# Combat State — Chunk 4c: Position × Weapon Utility (Reach Model) Design

> **Side quest from mob aliveness chunk 2.7.** Sub-chunk 4c of the
> combat-state-machines redesign (master spec:
> `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`).
> Third of six sub-chunks in the rich-grapple expansion (master-spec
> section "3a. Position rich-grapple expansion"):
>
>   - 4a (shipped) — Position FSM scaffold.
>   - 4b (shipped) — Control-axis mechanics + cutover.
>   - **4c (this spec)** — Weapon utility by position via a single
>     `reach` field on ItemSpec + position-radius curve. Long weapons
>     degrade in grapples; daggers stay useful; sword-in-mount narrates
>     as bludgeoning the target with the pommel instead of slashing.
>   - 4d — Submission system rework.
>   - 4e — Third-party interaction asymmetries.
>   - 4f — Balance pass + flavor text + full-stack smoke.
>
> **Aliveness paused for the duration** of chunks 1-6.

## Goal

Make weapon choice in grapples matter. After chunk 4b shipped, the
Position FSM drives 14 distinct geometric states and per-round control
drift, but combat damage in those states is currently weapon-agnostic
— a polearm in mount swings for the same numbers as a dagger in mount.
That's wrong both fictionally (you can't lever a spear with a guy on
top of you) and mechanically (no incentive to carry a backup short
weapon, no skill expression in weapon choice).

4c introduces a single `reach` field (meters) on `ItemSpec` and a
position-radius curve. The combat pipeline multiplies in a reach
utility factor: weapons whose reach exceeds the position's effective
radius are penalized; weapons whose reach fits the radius pay no
penalty. The attack-message vocabulary swaps to a bludgeoning set
when the penalty fires, so the player sees "you slam the iron
sword's pommel into their ribs" instead of "you slash awkwardly with
the iron sword" — fiction tracks math.

End state: tactical weapon-swapping in grapples becomes a real
choice. Carrying a dagger as an offhand becomes a viable response
to a grappler. Two-handed reach weapons (spear, halberd, pike,
greatsword, quarterstaff) become liabilities once a clinch lands.
Natural attacks (fist, claws, bite) keep their effectiveness in
ground-grapple because their reach is short by definition.

## Non-goals (4c)

- **Damage-type swap in the mitigation pipeline.** A sword swung
  as a pommel-bludgeon still applies physical damage against the
  physical mitigation channel. The bludgeoning angle is narrative
  only (attack-message vocabulary). Routing sword damage through
  the Crushing armor profile is future balance work, not 4c.
- **Reach affects defense.** Only the attack side reads reach in 4c.
  A defender parrying with a long weapon in a clinch isn't penalized
  by this system. (Real BJJ: it should be — but defense scoring is
  already a tangle in `combat_helpers.go`; tackle in 4f or later.)
- **Weapon weight cross-reference.** Weight stays focused on
  encumbrance + swing speed. A spear isn't bad in mount because
  it's heavy; it's bad because it's long. Two axes of penalty would
  double-count.
- **Per-grapple-state reach overrides.** All 11 grapple states use
  one of two radii (standing-grapple, ground-grapple). Mount and
  HalfGuard share the same effective radius for reach purposes.
  Position-specific weapon quirks (e.g., dagger in BackGround gets
  a bonus because exposed neck) are 4f flavor passes.
- **Stand × downed positioning.** When attacker is Standing and
  target is Prone/Supine, no reach penalty applies — attacker can
  swing freely from above. Only triggers when attacker `IsGrappling()`.
- **Reach as physical range for ranged combat.** A bow's reach in
  this model is its melee-fallback reach (used as a club). The
  shooting-range concept lives in a different field (`Range` or
  similar) and is unrelated to 4c.
- **Compound reach (arm length + weapon haft).** Future Reach refinement
  per the user's chunk-4c kickoff conversation. 4c uses weapon reach
  only; arm length is a perfect-information gap that adds noise without
  affecting balance materially at MUD-fidelity resolution.
- **Player-facing reach display.** The `examine` / `identify`
  commands gain reach in 4c (it's a stat), but no UI element
  surfaces "reach: dangerous in grapple" warnings. Player learns
  by feel + helpfile.

## Architecture

The work is small relative to 4a/4b. One new field on `ItemSpec`, one
new helper in the items package (default reach by subtype), one new
combat-helper function that computes the multiplier, two integration
points in `combat/grapple.go` / `combat/skill_moves.go` / wherever the
attack pipeline calls `CalcRawDamage`, and one tweak to the
`GetAttackMessage` call site to swap subtype for narration.

### Files

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/items/itemspec.go` | MODIFY | Add `Reach float64 \`yaml:"reach,omitempty"\`` field to `ItemSpec` |
| `internal/items/reach.go` | NEW | `DefaultReachForSubtype(ItemSubType) float64` lookup; `ResolveReach(spec *ItemSpec) float64` (returns explicit reach if set, else default-by-subtype, else 0) |
| `internal/items/reach_test.go` | NEW | Unit tests for default lookup + override resolution + zero-value handling |
| `internal/combat/reach.go` | NEW | `PositionReachRadius(s position.State) float64` curve; `ReachUtility(weaponReach, posRadius float64) float64` formula; `ShouldBludgeon(weaponReach, posRadius float64) bool` predicate |
| `internal/combat/reach_test.go` | NEW | Unit tests for radius lookup + utility curve + bludgeon threshold |
| `internal/combat/damage_pipeline.go` | MODIFY | Add `CalcReachAdjustedItemMult(weapon items.Item, attacker *characters.Character) float64` convenience helper that wraps `weaponSpec.DamageMultiplier * ReachUtility(...)`. Single place to call from each combat site. |
| `internal/combat/grapple.go` | MODIFY | (No direct change — grapple roll modifiers stay reach-agnostic. Grapple is about position transitions, not damage.) |
| `internal/combat/skill_moves.go` | MODIFY | `ExecuteSkillMove` passes the reach-adjusted item mult to `CalcRawDamage` |
| `internal/hooks/NewRound_DoCombat_helpers.go` | MODIFY | Per-swing damage calc uses `CalcReachAdjustedItemMult` |
| `internal/items/attack_messages.go` | MODIFY | New `GetAttackMessageBludgeon(originalSubType ItemSubType, pctDamage int) AttackOptions` that returns Crushing-pattern messages but with weapon-name interpolation respecting the original weapon (so "you slam the iron sword's pommel" not "you slam the iron sword"). Or simpler: just call `GetAttackMessage(Crushing, pctDamage)` and let the message templates handle it via `{weapon}` token. |
| `internal/hooks/NewRound_DoCombat_helpers.go` | MODIFY (again) | Message-selection site swaps subtype when `ShouldBludgeon` returns true |
| `internal/items/context.md` | MODIFY | Document reach field, default-by-subtype table, author conventions |
| `internal/combat/context.md` | MODIFY | Document reach utility curve, integration with damage pipeline, bludgeon narration |
| `_datafiles/world/dogmud/items/**/*.yaml` | MODIFY | Add `reach` to any item that should override its subtype default (initial pass: leave all to default, override later as balance feedback comes in) |
| `internal/configs/balance.go` | MODIFY | Add `ReachStandingGrappleRadius` (default 0.5), `ReachGroundGrappleRadius` (default 0.3), `ReachUtilityFloor` (default 0.15) |

No new state machine, no FSM changes, no new btree primitives. 4c is
the cheap chunk between the heavyweight 4b cutover and the
heavyweight 4d submission rework.

## Reach taxonomy

The single source of truth lives in `internal/items/reach.go` as the
`DefaultReachForSubtype` map. Values are weapon-only reach in meters;
arm length is intentionally excluded (chunk-4c-kickoff decision).
Authors override per-item only for outliers (e.g., a particularly
short shortsword, a ceremonial dagger with an oversized hilt).

### Natural attacks
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Fist | 0.1 | Punch length |
| Claws | 0.15 | Fingers extended |
| Bite | 0.15 | Head-mounted, neck-length |
| Sting | 0.2 | Abdomen/tail-tip |
| Slam | 0.3 | Bull rush, shoulder-check |
| Gore | 0.4 | Horn-tip from skull base |
| Whipping (natural — e.g., tail mutation) | 1.0 | Tail length |

### Light melee
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Dagger | 0.3 | Hilt to tip |
| Knuckles / brass | 0.1 | Treats as enhanced fist |
| Hatchet (handle ≤ 0.5m) | 0.4 | Throwing-axe size |

### Medium melee
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Shortsword | 0.7 | Roman gladius family |
| Mace | 0.8 | One-handed |
| Axe (one-handed) | 0.9 | Battle-axe handle |
| Sword (longsword, one-handed) | 1.0 | Classic arming sword |
| Whip (entanglement) | 0.5 | Operational radius wrapped, not extended |
| Flail (one-handed) | 1.0 | Haft + chain + head |
| Rapier | 1.1 | Long thin blade |

### Heavy / two-handed melee
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Battleaxe (two-handed) | 1.2 | |
| Greatsword | 1.5 | |
| Quarterstaff | 1.5 | |
| War hammer (two-handed) | 1.4 | |
| Spear | 2.0 | |
| Trident | 2.0 | |
| Glaive | 2.2 | Polearm with blade |
| Naginata | 2.3 | |
| Halberd | 2.5 | Polearm with axe head |
| Scythe (weaponized) | 1.8 | |
| Pike | 3.0+ | Formation weapon; effectively unusable in grapple |

### Exotic
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Kusarigama | 0.6 (kama end) / 2.0 (chain) | Use 0.6 — kama end is the operational reach in close quarters |
| Javelin (melee) | 1.7 | Used as short spear when held |
| Throwing knife | 0.3 | Same as dagger |
| Blowgun | 0.4 | Used as a striking rod in melee |
| Sling | 0.2 | Hand-only operational reach when not winding up |

### Ranged (melee-fallback reach)
Used when the ranged weapon must function as a melee club (grappled
target, no room to draw/aim). Shooting-range is a separate field.
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Shortbow | 1.0 | Used as a stave |
| Longbow | 1.5 | Used as a stave |
| Crossbow | 0.7 | Compact heavy |
| Heavy crossbow | 0.9 | Bigger frame |

### Caster
| Subtype | Reach (m) | Notes |
|---------|----------|-------|
| Wand | 0.4 | Foot-long focus |
| Sceptre | 0.6 | Larger ornamented focus |
| Staff (caster) | 1.5 | Quarterstaff equivalent in close quarters |

### Authoring guidance (for context.md)

Authors creating new weapon items SHOULD:
1. Leave `reach` empty in YAML for normal items of a known subtype
   (the engine applies the default-by-subtype value).
2. Set `reach: <meters>` only when the item is unusual (e.g., "this
   particular dagger has a 0.5m blade — set reach: 0.5").
3. Use meters, not abstract units. Real-world references help
   balancing.
4. For wholly new subtypes, add a row to `DefaultReachForSubtype`
   in `internal/items/reach.go` and update the context.md table.

## Position radius curve

Two effective radii in 4c — kept simple deliberately. Per-state
refinement is 4f flavor work.

| Position class | Radius (m) | States |
|----------------|-----------|--------|
| Not grappling | ∞ (no penalty) | Standing, Prone, Supine, Turtle (defensive curl, opponent not engaging at grapple-radius) |
| Standing grapple | 0.5 | Clinch, BackStanding |
| Ground grapple | 0.3 | Mount, SideControl, KneeOnBelly, NorthSouth, Crucifix, BackGround, HalfGuard, Guard |

**Standing alone is unbounded** — a fighter standing upright at conversational
distance is not "in a grapple" so reach plays normally regardless of
target position. A standing attacker over a downed (Prone/Supine)
target also pays no reach penalty — they have room to swing.

**Turtle is unbounded too** — it's a defensive ball, not a pair
position. The Turtle player isn't grappling; if someone WAS grappling
them they'd be in BackGround or Crucifix.

The radius lookup lives in `combat.PositionReachRadius(s position.State) float64`.
Returns 0 (sentinel "no penalty") for non-grapple states.

Configurable via balance config:
- `ReachStandingGrappleRadius` (default `0.5`)
- `ReachGroundGrappleRadius` (default `0.3`)

## Utility formula

```
func ReachUtility(weaponReach, posRadius float64) float64 {
    if posRadius == 0 { return 1.0 } // not grappling — full damage
    if weaponReach <= posRadius { return 1.0 } // weapon fits — no penalty
    util := posRadius / weaponReach
    if util < ReachUtilityFloor { return ReachUtilityFloor }
    return util
}
```

Worked examples (defaults: standing-grapple radius 0.5, ground-grapple
radius 0.3, floor 0.15):

| Weapon | Reach | Standing-grapple | Ground-grapple |
|--------|-------|-------------------|-----------------|
| Fist | 0.1 | 1.00 | 1.00 |
| Dagger | 0.3 | 1.00 | 1.00 |
| Sword | 1.0 | 0.50 | 0.30 |
| Greatsword | 1.5 | 0.33 | 0.20 |
| Spear | 2.0 | 0.25 | 0.15 (floored) |
| Pike | 3.0 | 0.17 | 0.15 (floored) |
| Wand | 0.4 | 1.00 | 0.75 |
| Caster-staff | 1.5 | 0.33 | 0.20 |

Floor at 0.15 keeps the weapon from doing literal zero damage — you
can always at least poke or thrust the pommel for chip damage. Tunable
if smoke says "pike in mount should be 0.05".

## Damage pipeline integration

Existing pipeline (recap):
```
rawDmg = CalcRawDamage(stat, skillRank, itemMult, channel)
finalDmg = ApplyMitigation(rawDmg, mitPct, mitCap)
displayed = dice.RollStat(finalDmg)
```

4c change: `itemMult` becomes reach-adjusted at every site that
currently passes a weapon's `damage_multiplier`. The convenience
helper:

```go
// CalcReachAdjustedItemMult returns the weapon's damage_multiplier
// scaled by ReachUtility for the attacker's current Position. Use
// this everywhere CalcRawDamage's itemMult argument was previously
// fed weaponSpec.DamageMultiplier.
//
// For natural-attack contexts (unarmed, mob fist/claws/bite), the
// caller passes the natural-attack pseudo-spec; reach defaults to
// the subtype's natural-attack reach (fist=0.1, etc.).
func CalcReachAdjustedItemMult(
    weapon items.Item,
    attacker *characters.Character,
) float64 {
    spec := weapon.GetSpec()
    baseMult := spec.DamageMultiplier
    reach := items.ResolveReach(spec)
    posRadius := PositionReachRadius(attacker.Position.State())
    return baseMult * ReachUtility(reach, posRadius)
}
```

Sites to update:
- `combat/skill_moves.go:ExecuteSkillMove` — uses `p.DamagePercent`
  directly today; needs to wrap if attacker has a weapon. (Skill
  moves like grapple/trip/bash are stat-driven and weapon-agnostic;
  may not need reach adjustment at all. Decide per-move during
  implementation — kick already routes through subtype, so likely
  yes; grapple is force, likely no.)
- `internal/hooks/NewRound_DoCombat_helpers.go` — main per-swing
  damage path. Replace `weaponSpec.DamageMultiplier` reads with
  `CalcReachAdjustedItemMult(weapon, attacker)` calls. This is the
  hot path; expect ~3-5 sites.
- `internal/hooks/spell_resolution.go` — spells aren't reach-gated
  (4c non-goal), so no change. (The caster-weapon `spell_damage_multiplier`
  stays unaffected; staves in mount are bad for melee but the spell
  still casts normally. Future 4f could revisit if "casting in mount"
  becomes a balance issue.)

## Attack-message vocabulary swap (bludgeon narration)

When `ReachUtility < 1.0` AND original subtype is bladed/piercing,
the message-selection site swaps to the Crushing pattern. Message
templates already use `{weapon}` and `{target}` tokens, so a sword
narrated as Crushing renders correctly: "You slam the iron sword's
pommel against the bandit's ribs" — the template just needs to be
written that way for the Crushing set.

Predicate:
```go
func ShouldBludgeon(weaponReach, posRadius float64) bool {
    return posRadius > 0 && weaponReach > posRadius
}
```

Site:
```go
// Before:
msg := items.GetAttackMessage(weaponSpec.Subtype, pctDmg)

// After:
displaySubtype := weaponSpec.Subtype
if combat.ShouldBludgeon(items.ResolveReach(weaponSpec), combat.PositionReachRadius(attacker.Position.State())) {
    // Long weapon in close quarters narrates as a club/bludgeon
    // strike regardless of bladed/piercing classification.
    if weaponSpec.Subtype == items.Slashing || weaponSpec.Subtype == items.Cleaving ||
        weaponSpec.Subtype == items.Stabbing || weaponSpec.Subtype == items.Shooting {
        displaySubtype = items.Crushing
    }
}
msg := items.GetAttackMessage(displaySubtype, pctDmg)
```

Naturally-blunt subtypes (Slam, Gore, Whipping, Fist) keep their
existing vocabulary even if `ShouldBludgeon` fires — they're already
blunt.

Caster weapons (Wand, Sceptre, Staff) when used for melee already
read as bludgeoning-flavored in their existing messages; no special
handling needed (the swap is for blades).

## YAML migration plan

Phase 1 (in 4c, mechanical): no YAML changes required. Every existing
weapon inherits the default reach for its subtype via `ResolveReach`.
Combat behavior changes immediately — daggers stay punchy in grapples,
swords degrade, polearms become bludgeons.

Phase 2 (post-4c, balance feedback): per-item overrides for outliers.
Add `reach: <meters>` to specific YAMLs as smoke surfaces "this
particular dagger should be 0.5m" cases.

The audit script `tools/id_inventory.py` (or a new sibling) can
optionally scan for weapons with `reach: 0` that should probably
have a default — but the engine already handles zero by falling
through to the subtype default, so this is purely a hygiene check
for hand-authored values being intentional.

## Behavior matrix preview

Authored in `internal/combat/reach_test.go` and integration tests.
PB-201 through PB-220 (sub-chunk codes follow 4b's PB-001-080).

| ID | Scenario | Expected |
|----|----------|----------|
| PB-201 | Fist in mount | mult = 1.00 |
| PB-202 | Dagger in mount | mult = 1.00 |
| PB-203 | Sword in mount | mult = 0.30 |
| PB-204 | Spear in mount | mult = 0.15 (floored) |
| PB-205 | Sword standing (no grapple) | mult = 1.00 |
| PB-206 | Sword vs prone target (attacker standing) | mult = 1.00 (attacker not grappling) |
| PB-207 | Sword in clinch | mult = 0.50 |
| PB-208 | Wand in clinch | mult = 1.00 |
| PB-209 | Wand in mount | mult = 0.75 |
| PB-210 | Caster-staff in mount | mult = 0.20 |
| PB-211 | Greatsword in HalfGuard (bottom) | mult = 0.20 |
| PB-212 | Pike in clinch | mult = 0.17 |
| PB-213 | Item with explicit reach=0.5 (overrides subtype default) | reach resolves to 0.5, not subtype value |
| PB-214 | Item with reach=0 in YAML (omitted) | falls through to subtype default |
| PB-215 | Bludgeon narration: sword in mount | message vocabulary = Crushing |
| PB-216 | No bludgeon narration: dagger in mount | message vocabulary = Stabbing (original) |
| PB-217 | No bludgeon narration: fist in mount | message vocabulary = Fist (original; already blunt) |
| PB-218 | Caster-spell damage: staff in mount | spell damage unaffected (4c non-goal) |
| PB-219 | Reach utility floor: pike in mount = 0.15, not 0.10 | hits floor |
| PB-220 | Turtle position: no penalty (unbounded) | mult = 1.00 even with greatsword |

## Sunset list (4c)

Nothing sunset in 4c. All deletions belonged to 4b. 4c adds; it
doesn't remove.

## Persistence

`Reach` lives on `ItemSpec`, which is template-loaded from YAML at
boot. Per-instance items don't persist reach (it's not a runtime
mutable field). Saved character/mob YAMLs don't need migration —
just bump the engine and weapons start respecting position-radius
penalties.

If 4d/4e adds runtime per-item reach modifiers (e.g., a "broken
hilt" debuff that shortens a weapon's effective reach), that goes
on `Item.RuntimeData` or similar, separate from spec-default reach.
Out of 4c scope.

## Open questions / risks

1. **Should grapple-rolls themselves read reach?** (E.g., a fighter
   carrying a polearm has a harder time initiating a clinch because
   the polearm is in the way.) Probably yes for realism but 4c
   defers — reach only affects damage in 4c. 4d's submission rework
   is the natural place to revisit grapple-roll modifiers; reach
   could enter there.
2. **Two-handed reach when offhand is empty.** If a character is
   wielding a one-handed sword in a clinch but their offhand is
   free, real-world they'd probably drop the sword and grapple
   bare-handed. We don't model "drop weapon" in MUDs typically.
   Solution: just accept the penalty. Player can manually `remove
   sword` and grapple unarmed if they want. Document in helpfile.
3. **Two-weapon fighters.** If wielding dagger offhand + sword
   main-hand, the player would want to swing the dagger in mount,
   not the sword. Current swing scheduling alternates main + offhand
   per round. Per-swing reach evaluation handles this correctly —
   each swing reads the weapon being swung. No special handling.
4. **Mob natural-attack reach.** Mobs without weapons fall through
   to the unarmed combat path. Need a clean way to map mob-species
   natural-attack subtype (claws, bite, gore, etc.) to a reach value
   for the multiplier calc. `ResolveReach` should accept a "natural
   attack subtype" lookup path that doesn't require a real ItemSpec
   — likely a `ResolveNaturalReach(subtype ItemSubType) float64`
   sibling helper that just hits `DefaultReachForSubtype`.
5. **Performance.** Per-swing reach computation is two map lookups
   + a float division. Trivial. No caching needed.
6. **Reach floor calibration.** The 0.15 floor (pike in mount does
   15% damage) might be too generous — a pike in mount realistically
   does zero. Smoke testing decides. Tunable via balance config.
7. **Player visibility.** Without an in-game way to see reach
   values, players will discover the penalty by feel. Acceptable
   for first release; a `weaponstats <weapon>` admin command or
   reach-in-identify enhancement can land later.
8. **Compound reach (arm length + weapon).** User flagged this in
   chunk-4c kickoff as "problem for later." 4c uses weapon-only
   reach; arm length introduces species/size variance (Salamandri
   vs Skeleton vs Lumix) without affecting balance at MUD-fidelity
   resolution. Future spec if it becomes load-bearing.
9. **Defense-side reach.** A defender in mount with a polearm
   parry-rolling against a dagger attacker is currently unpenalized.
   Realism says they should be. Defense scoring is gnarly
   (`combat/combat_helpers.go` best-of-all defense) and adding
   reach to it risks regressions. 4c keeps it offense-side only;
   4f flavor pass can revisit if combat feels asymmetric.

## Resumption criteria (chunk 4c done when)

- `Reach` field exists on `ItemSpec`, loaded from YAML, defaults to
  zero (sentinel for "use subtype default").
- `items.DefaultReachForSubtype` and `items.ResolveReach` shipped;
  `ResolveReach` correctly falls through subtype default for zero
  reaches and uses explicit value when set.
- `combat.PositionReachRadius`, `combat.ReachUtility`, and
  `combat.ShouldBludgeon` shipped with the radius curve and
  utility formula above.
- `combat.CalcReachAdjustedItemMult` shipped and used at every
  per-swing damage site (NewRound_DoCombat_helpers, kick variants
  inside skill_moves where appropriate).
- Attack-message vocabulary swap fires for bladed weapons in
  grapple positions (`ShouldBludgeon → true`).
- Reach values authored in `internal/items/context.md` reference
  table; weapon authoring SOP documented.
- Balance knobs (`ReachStandingGrappleRadius`,
  `ReachGroundGrappleRadius`, `ReachUtilityFloor`) live in
  `internal/configs/balance.go`.
- Behavior Matrix PB-201 through PB-220 PASS or SKIP per plan.
- Chunks 0-4b regression clean.
- Server boots cleanly past data-file loading.

## Out-of-scope / future followup candidates

- Defense-side reach penalty in grapples (long parry weapons should
  be hard to use defensively from mount).
- Per-grapple-state radius overrides (e.g., BackGround tighter than
  Mount).
- Compound reach (arm length + weapon haft) for species-level reach
  variance.
- Reach affecting grapple-entry rolls (long-weapon wielder harder
  to clinch).
- Player-facing reach UI (identify integration, encumbrance-style
  prompt token).
- Per-weapon override values for the 200ish existing weapons (post-
  smoke balance).
- Casting-while-grappled penalties (caster-staff in mount degrades
  spell damage too? Currently only melee).
- "Drop weapon to grapple" mechanic — auto or manual.
- Pommel-strike / bash with reach > radius as an explicit skill
  move (instead of just damage degradation).
