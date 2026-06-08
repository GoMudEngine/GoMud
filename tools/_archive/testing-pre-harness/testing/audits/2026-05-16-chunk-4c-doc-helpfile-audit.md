# Chunk 4c documentation + helpfile audit

Produced 2026-05-16 to feed Task 9 (helpfile updates) and Task 10
(context.md updates) of the chunk-4c Position × Weapon Utility plan.

Scope: identify helpfiles and context.md files that must add, update,
or cross-reference the new reach mechanic introduced by commits:

- `bbb325b6` — T1: Reach field on ItemSpec + default-by-subtype lookup
- `d7347ba3` — T2: position radius curve + ReachUtility + ShouldBludgeon
- `95e6d992` — T3: CalcReachAdjustedItemMult wired into per-swing damage
- `f814e238` — T4: Bludgeon narration (Bludgeoning subtype swap for
  bladed weapons in grapples)

Survey date: 2026-05-16. Branch: `feature/mob-aliveness-1.3-crimes`.

---

## Status table (chunk 4c tasks through T8)

| Task | Status | Commit |
|------|--------|--------|
| T1 — Reach field + default lookup | Done | `bbb325b6` |
| T2 — Position radius curve + utility | Done | `d7347ba3` |
| T3 — Damage pipeline integration | Done | `95e6d992` |
| T4 — Bludgeon narration | Done | `f814e238` |
| T5 — Behavior Matrix PB-201..PB-220 | Done | (in T2/T3/T4 tests) |
| T6 — Build/test/smoke | Done | (per prior session) |
| T7 — Reserved (skipped by design) | N/A | — |
| T8 — Doc audit (this file) | In progress | — |
| T9 — Helpfile updates | Pending | — |
| T10 — context.md updates | Pending | — |
| T11 — Roadmap closeout | Pending | — |

---

## Files reviewed — summary table

### Helpfiles

| File | Keyword match | Verdict |
|------|--------------|---------|
| `attack.template` | grapple (Defense System section) | UPDATE |
| `combat.template` | grapple (Special Moves section) | UPDATE |
| `grapple.template` | grapple (entire file) | UPDATE |
| `unarmed-combat.template` | grapple (Progression section) | UPDATE |
| `weapon-combat.template` | — (relevant by type) | UPDATE |
| `kick.template` | grapple (Knee variant) | UPDATE |
| `submit.template` | grapple (entire file) | KEEP-AS-IS |
| `stand.template` | — (not grapple-specific) | KEEP-AS-IS |
| `bash.template` | — (not grapple-specific) | KEEP-AS-IS |
| `trip.template` | — (not grapple-specific) | KEEP-AS-IS |
| `identify.template` | — (spell description) | KEEP-AS-IS |
| `equip.template` | — (slot management) | KEEP-AS-IS |
| `knee.template` | grapple (alias description) | KEEP-AS-IS |
| `iron-dagger.template` | reach keyword found | UPDATE |
| `iron-short-sword.template` | reach ("without sacrificing reach") | UPDATE |
| `steel-longsword.template` | reach ("demand reach, balance") | UPDATE |
| `lake-iron-hook-spear.template` | — (no reach keyword; long weapon) | UPDATE |

### Context.md files

| File | Verdict | Scope |
|------|---------|-------|
| `internal/items/context.md` | KEEP-AS-IS (T1 already added reach section) | Verify completeness |
| `internal/combat/context.md` | UPDATE | New "Weapon reach utility" section |
| `internal/state/position/context.md` | UPDATE | Cross-reference to reach utility |
| `internal/characters/context.md` | UPDATE | One-line note on position predicates as reach consumers |
| `internal/hooks/context.md` | UPDATE | Mention CalcReachAdjustedItemMult at per-swing site |
| `internal/configs/context.md` | UPDATE | Three new balance knobs |

---

## Per-helpfile findings

### `attack.template` — UPDATE

**Why:** The Defense System section lists the three defenses but the
Attack Mechanics section has no note about weapon choice in grapples.
Players who read this file after being surprised by "my greatsword does
nothing in mount" will find no explanation here.

**Suggested copy** — add a new "Weapon Reach in Grapples" subsection
after "Stamina and Penalties" (before "Target Switching"). Match
existing `━━━` header style and 80-char wrap:

```
<ansi fg="yellow">━━━ Weapon Reach in Grapples ━━━</ansi>

Your weapon's effective reach matters when you're locked in close
combat. Long weapons — spears, greatswords, polearms — become
awkward when an opponent has clinched you. There's simply not
enough space to swing them properly, so they strike with reduced
force (a pommel jab, a hilt-check). Short weapons — daggers, fists,
claws — fit comfortably in any grapple and deal full damage.

The tighter the grapple, the worse long weapons fare:
  - <ansi fg="stat">Standing grapple (Clinch):</ansi> Medium-length weapons still
    work; polearms struggle.
  - <ansi fg="stat">Ground grapple (Mount, Guard):</ansi> Even swords are
    awkward. Daggers and fists stay dangerous.

See <ansi fg="command">help grapple</ansi> for grapple positions.
```

Also add `help grapple` to the See Also line.

---

### `combat.template` — UPDATE

**Why:** The Special Combat Moves and Unarmed Combat sections both
touch grapple adjacently. No mention of reach in the weapon-choice
context. Players reading `help combat` for a tactical overview
deserve a brief pointer.

**Suggested copy** — add one sentence to the Unarmed Combat block
after "No Penalty: Natural weapons ignore dual-wield penalty":

```
  - <ansi fg="stat">Grapple Advantage:</ansi> Fists and claws fit
    comfortably in any grapple — long weapons lose effectiveness
    in close quarters. See <ansi fg="command">help grapple</ansi>.
```

Keep it tight — this is an overview file, not the detailed explainer.

---

### `grapple.template` — UPDATE

**Why:** This is the primary explainer for grapple. Currently 15 lines
covering initiation, the restraint effects, and the mechanic unlock.
Nothing about how the grapple state affects the combatants' own weapon
damage. This is the highest-priority update — players will `help grapple`
first when confused.

**Suggested copy** — replace the current body with an expanded version:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">grapple</ansi>

The <ansi fg="command">grapple</ansi> command attempts to lock your
opponent in a grappling hold. Grappled foes are restrained: they
cannot flee, their attacks and defenses are penalised, and your
<ansi fg="command">knee</ansi> special becomes available.

<ansi fg="yellow">━━━ Grapple Positions ━━━</ansi>

A grapple isn't a single state — it develops through positions as
control shifts. Standing grapples (Clinch, BackStanding) are closer
quarters than a normal fight. Ground grapples (Mount, Guard, Side
Control, etc.) are tighter still.

<ansi fg="yellow">━━━ Weapon Reach in Grapples ━━━</ansi>

Long weapons become a liability when you're grappling. In a clinch
or mount, a sword or spear can't be swung freely — the haft catches
your own body. The weapon's pommel or hilt still lands for reduced
damage, and your combat narration shifts to reflect it.

  - <ansi fg="stat">Short weapons (daggers, fists, claws):</ansi>
    Remain fully effective in any grapple.
  - <ansi fg="stat">Medium weapons (swords, axes):</ansi>
    Penalised in ground grapples; partial penalty in clinch.
  - <ansi fg="stat">Long weapons (spears, polearms, staves):</ansi>
    Severely penalised in all grapple positions.

Carrying a dagger in your offhand is a sound counter to grapplers —
it will swing at full damage while a main-hand sword is hampered.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">grapple <target></ansi>   Attempt to grapple a target in combat.

Success is an opposed roll of Strength vs. the target's Strength
and Dexterity. Grapple-immune creatures (e.g. elementals, oozes)
cannot be grappled.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help attack</ansi>,
  <ansi fg="command">help kick</ansi>, <ansi fg="command">help trip</ansi>,
  <ansi fg="command">help submit</ansi>
```

---

### `unarmed-combat.template` — UPDATE

**Why:** The file already mentions grapple in the Progression section
("Using kick, trip, and grapple"). Adding one sentence about the
grapple advantage of natural weapons is the obvious extension.

**Suggested copy** — add to the Attacks section after "No Dual-Wield
Penalty":

```
  - <ansi fg="stat">Grapple-Friendly:</ansi> Fists and claws have very
    short reach. They stay fully effective in clinches and ground
    grapples where long weapons become awkward.
```

---

### `weapon-combat.template` — UPDATE

**Why:** Currently 8 lines with zero mention of reach or grapple.
Weapon-combat governs swords, axes, maces — exactly the weapons most
affected by the reach mechanic. This file is where an armed player
goes to understand their skill.

**Suggested copy** — append a new section before the See Also line:

```
<ansi fg="yellow">━━━ Reach and Grappling ━━━</ansi>

Melee weapons have a reach that affects how well they perform in
close-quarters grapples. Long weapons (swords, axes, staves) are
penalised when you're locked in a clinch or ground grapple — there
isn't room to swing them freely. Short weapons (daggers) remain
fully effective. See <ansi fg="command">help grapple</ansi> for details.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help skills</ansi>,
  <ansi fg="command">help equip</ansi>, <ansi fg="command">help grapple</ansi>
```

---

### `kick.template` — UPDATE

**Why:** The Knee variant description says "when you have control in a
clinch or ground grapple, your kick becomes a close-range knee strike."
This is exactly the grapple context where weapon reach penalises swings.
A note here saves the player the inference step.

**Suggested copy** — add one line to the Knee section:

```
<ansi fg="yellow">━━━ Knee (grapple, in control) ━━━</ansi>

  When you have control in a clinch or ground grapple, your
  kick becomes a close-range knee strike. Strong damage that
  works where kicks can't — and where long weapons struggle.
```

---

### `submit.template` — KEEP-AS-IS

Accurate as written. Describes outcomes of the submit attempt (yield,
resist, failure). Reach does not affect submission rolls in 4c.
Update only if 4d adds reach-gated submission windows.

---

### `stand.template` — KEEP-AS-IS

Describes recovery from prone (not grapple positions). Reach has no
interaction with the stand/prone path. Keep.

---

### `bash.template` — KEEP-AS-IS

Bash is force-driven (shield hit → knockdown). Reach-agnostic by
design in 4c. Keep.

---

### `trip.template` — KEEP-AS-IS

Trip is force-driven (leg sweep → knockdown). Reach-agnostic by
design in 4c. Keep.

---

### `identify.template` — KEEP-AS-IS

The identify spell reveals item properties. In 4c, `identify` does
not surface reach as a displayed stat (per the design spec: "The
examine / identify commands gain reach in 4c (it's a stat), but no
UI element surfaces 'reach: dangerous in grapple' warnings — player
learns by feel + helpfile."). Keep as-is. Post-smoke: if player
confusion warrants it, T9's next iteration can add reach to the
identify display and update this file.

---

### `equip.template` — KEEP-AS-IS

Covers slot management and arm-pair mechanics. No reach content
needed at this scope. Keep.

---

### `knee.template` — KEEP-AS-IS

This is a one-line alias stub redirecting to `help kick`. The detail
lives in kick.template where it will be updated. Keep the alias.

---

## Per-weapon helpfile findings

These four helpfiles exist in `_datafiles/world/dogmud/templates/help/`
and are the representative crafted-weapon templates T9 should update.
All are currently blacksmithing-recipe explainers with no reach content.

### `iron-dagger.template` — UPDATE

**Current:** "A short iron blade with a leather-wrapped grip — quick
to draw and reliable for close-quarters fighting."

**Reach value (Stabbing default):** 0.30 m — fits inside ground-grapple
radius (0.30 m); no penalty in any grapple.

**Suggested reach line** — append after the description paragraph:

```
<ansi fg="cyan">Reach:</ansi> Very short — stays fully effective in any grapple.
A reliable offhand choice against grapplers.
```

Note: the existing description already says "close-quarters" which
reads as a hint — the explicit reach note makes it actionable. No
numbers needed; use descriptive language per SOP.

---

### `iron-short-sword.template` — UPDATE

**Current description:** "A broad, compact iron blade with a leather
grip — a dependable sidearm for warriors who value reach without
sacrificing speed."

Note: the word "reach" here is used colloquially (blade length). After
4c, the word has an engine meaning. Leaving it unchanged is acceptable
(it's still true — the short sword does have more blade reach than a
dagger). However, a mechanical note clarifies which side of the penalty
the short sword falls on.

**Reach value (Slashing default):** 1.00 m — exceeds ground-grapple
radius (0.30 m); penalty applies. Below standing-grapple radius (0.50 m)?
No — 1.00 m > 0.50 m, so clinch also penalises it.

**Suggested reach line** — append after the description paragraph:

```
<ansi fg="cyan">Reach:</ansi> Medium — penalised in close-quarters grapples.
Consider a dagger offhand if grapplers are common.
```

---

### `steel-longsword.template` — UPDATE

**Current description:** "A keen steel blade with a leather-wrapped
hilt — the weapon of choice for seasoned warriors who demand reach,
balance, and edge."

Same colloquial "reach" usage as the short sword — no conflict, but
worth adding the mechanical note.

**Reach value (Slashing default):** 1.00 m — same as short sword in
the Slashing family. Penalised in both clinch and ground grapples.

**Suggested reach line** — append after the description paragraph:

```
<ansi fg="cyan">Reach:</ansi> Long — becomes awkward in grapples. The hilt
still connects for reduced damage, but consider drawing a dagger
when the fight goes to the ground.
```

---

### `lake-iron-hook-spear.template` — UPDATE

**Current description:** "A long-shafted spear of refined lake-iron
with a back-curving hook welded to the socket — the smith's
adaptation of a fisher's gaff for a fighter's hand. The hook catches
a parrying weapon and drags it offline."

**Reach value:** The Stabbing default is 0.30 m (dagger family) — but
this is a spear! This is a known gap in the subtype taxonomy: the
engine groups all Stabbing weapons at 0.30 m by default, which is
wrong for a spear. The spear's actual reach should be ~2.00 m per the
design spec taxonomy.

**T9 action:** Add `reach: 2.0` to the spear's YAML AND add the
helpfile note. This is the first concrete example of a per-item YAML
override that the spec anticipated ("post-smoke balance feedback").
The hook-spear is the only Stabbing-subtype weapon whose reach clearly
diverges from the dagger default.

**Suggested reach line:**

```
<ansi fg="cyan">Reach:</ansi> Very long — this is a full spear. In a grapple
it becomes nearly useless as a conventional weapon; the shaft is
too long to control. You will be swinging the butt end.
Consider a backup short blade.
```

---

## Per-context.md findings

### `internal/items/context.md` — KEEP-AS-IS (T1 complete)

T1 (`bbb325b6`) already added the "Weapon Reach (chunk 4c)" section at
the bottom of this file. The section includes:

- Overview paragraph (what reach means, link to spec)
- Default-by-subtype table (all 15 subtypes with reach values)
- Authoring guidance (5 numbered rules)
- Consumer cross-reference to `internal/combat/reach.go`

**Completeness check:**
- All in-engine subtypes present: yes (Fist through Staff, plus
  Bludgeoning at 0.80 m).
- Authoring SOP complete: yes (empty = use default, explicit for
  outliers, meters not abstract, new subtypes add a case + update table,
  arm length out of scope).
- Cross-reference to combat package: yes.
- Cross-reference to spec doc: yes.
- One gap: the table lists `Bludgeoning` at 0.80 m but the code in
  `internal/items/reach.go` only defines Stabbing/Slashing/Cleaving/
  Shooting/Fist/Claws/Bite/Sting/Slam/Gore/Whipping/Wand/Sceptre/Staff.
  `Bludgeoning` subtype is NOT in the switch statement (it's a combat-
  message subtype used for the narration swap, not a weapon carry-type
  that authors set on item specs). The table entry is technically
  correct (a mace-type item would author `subtype: bludgeoning` and
  the engine would fall through to 0.0 sentinel — which means no
  penalty, which is correct for a mace family). But the table implies
  a default reach of 0.80 m when actually the code returns 0 for
  `Bludgeoning`. **This is a documentation error introduced in T1 that
  T10 must correct.** The fix: remove Bludgeoning from the table, or
  add a note "Bludgeoning is a combat-message vocabulary, not a
  carry-subtype. Mace/hammer items should be authored as generic or
  use a future Bludgeoning carry-subtype if added." The reach.go switch
  should also add a `case Bludgeoning: return 0.80` if the intent is
  that authors can set `subtype: bludgeoning` on maces.

**T10 action:** Correct the Bludgeoning row or add a clarifying note.
This is the only inaccuracy found; all other entries are correct.

---

### `internal/combat/context.md` — UPDATE

**Why:** This is the primary context file for the combat package. T3
and T4 added `CalcReachAdjustedItemMult`, the per-swing site update,
and the bludgeon narration swap — none of which are documented here.
The existing file has grapple references (grapple positions, control
axis, penalty scoring) but nothing about how weapon reach interacts
with the per-swing damage path.

**T10 scope:** Add a new "Weapon Reach Utility (chunk 4c)" section
after the existing "Position FSM and Grapple States" content. Suggested
location: between the per-round combat flow description and the
combat calculations section. Cover:

1. One-paragraph overview: what the reach system does.
2. Function signatures and summary:
   - `PositionReachRadius(s position.State) float64` — returns 0
     (no penalty) for non-grapple states, `ReachStandingGrappleRadius`
     for Clinch/BackStanding, `ReachGroundGrappleRadius` for all ground
     grapple states.
   - `ReachUtility(weaponReach, posRadius float64) float64` — returns
     `min(1.0, posRadius/weaponReach)` floored at `ReachUtilityFloor`.
   - `ShouldBludgeon(weaponReach, posRadius float64) bool` — true when
     weapon exceeds grapple radius; drives the narration swap.
   - `CalcReachAdjustedItemMult(weapon Item, attacker *Character) float64`
     — pipeline integration helper used at every per-swing damage site.
3. Bludgeon narration rules:
   - When `ShouldBludgeon` fires, the attack-message subtype swaps to
     `Bludgeoning` for bladed/ranged weapons (Slashing, Cleaving,
     Stabbing, Shooting).
   - Exempt: natural-blunt subtypes (Fist, Slam, Gore, Whipping, Claws,
     Bite, Sting) and caster subtypes (Wand, Sceptre, Staff).
   - Damage math unchanged; narration only.
4. Balance knobs (from `internal/configs/balance.go`):
   - `ReachStandingGrappleRadius` (default 0.5 m)
   - `ReachGroundGrappleRadius` (default 0.3 m)
   - `ReachUtilityFloor` (default 0.15)
5. Implementation note: natural-attack paths (mob unarmed, claws, bite)
   use `items.ResolveNaturalReach(subtype)` directly, not
   `CalcReachAdjustedItemMult`.

---

### `internal/state/position/context.md` — UPDATE

**Why:** The "Next chunks" paragraph in the Status section currently
reads: "4c — weapon/combat integration (position-based weapon
availability, attack-variant selection, ground-striking modifiers)."
4c has shipped; that description is also not quite right (4c is reach
penalty, not weapon availability or variant selection). Update to
reflect what actually shipped.

**T10 action:** Change the "4c" entry in the Next Chunks paragraph:

```
**4c shipped:** Weapon reach utility. `internal/combat/reach.go`
reads `State()` to compute a damage multiplier (position-radius
curve: standing-grapple 0.5 m, ground-grapple 0.3 m). Long weapons
degrade in grapples; short weapons stay effective. Bladed weapons
narrate as Bludgeoning when `ShouldBludgeon` fires. See
`internal/combat/context.md` for the integration.
```

Also update the "Next chunks" sequence to reflect 4c done and the
4d–4f remainder.

---

### `internal/characters/context.md` — UPDATE

**Why:** Line 676 already mentions grapple speed multipliers
(Clinch/BackStanding 0.6, ground grapples 0.3). This is adjacent to
the reach utility — position predicates are now reach-utility consumers.
A one-line forward reference helps future contributors find the link.

**T10 action:** Under the `IsGrappling()` / `IsGroundGrapple()` /
position predicates section, add:

```
  Position predicates also drive the chunk-4c reach utility
  (internal/combat/reach.go): `IsGrappling()` + `State()` determine
  the grapple radius, which scales weapon damage per swing.
```

---

### `internal/hooks/context.md` — UPDATE

**Why:** The file documents the combat-round flow in detail. The
per-swing damage path in `NewRound_DoCombat_helpers.go` now calls
`CalcReachAdjustedItemMult` instead of reading `weaponSpec.DamageMultiplier`
directly, and the attack-message selection site now invokes
`ShouldBludgeon` before calling `GetAttackMessage`. Neither change is
documented.

**T10 action:** In the section describing `NewRound_DoCombat_helpers.go`
(or the per-swing flow), add:

```
- **Reach-adjusted damage (chunk 4c):** Per-swing damage now calls
  `combat.CalcReachAdjustedItemMult(weapon, attacker)` instead of
  reading `weaponSpec.DamageMultiplier` directly. Long weapons take
  a multiplicative penalty in grapple positions.
- **Bludgeon narration (chunk 4c):** Attack-message selection calls
  `combat.ShouldBludgeon(reach, radius)` before `items.GetAttackMessage`.
  When true, bladed weapon subtypes swap to `Bludgeoning` vocabulary
  so fiction tracks math.
```

---

### `internal/configs/context.md` — UPDATE

**Why:** Three new balance knobs were added in T2 with no corresponding
entry in the config context. This file documents Balance section fields.

**T10 action:** Add a "Weapon Reach (chunk 4c)" subsection under the
Balance section:

```
#### Weapon Reach (chunk 4c)

| Knob | Default | Effect |
|------|---------|--------|
| `ReachStandingGrappleRadius` | 0.5 | Effective radius (m) for Clinch / BackStanding grapple states. Weapons longer than this are penalised. |
| `ReachGroundGrappleRadius` | 0.3 | Effective radius (m) for ground grapple states (Mount, Guard, etc.). Tighter than standing. |
| `ReachUtilityFloor` | 0.15 | Minimum damage multiplier from the reach curve. Prevents long weapons from doing literal zero. |
```

---

## New player-facing surface

### Standalone `reach.template` helpfile — RECOMMENDED

**Verdict: Yes, author a dedicated `reach.template`.**

**Reasoning:** The grapple.template update above is the primary
discovery path ("look up grapple, learn about reach"). But players who
already know about reach and want the full reference — all the position
categories, what counts as short vs. long, whether daggers and fists
are always safe — benefit from a standalone `help reach` target.

The plan spec (T9 scope) explicitly lists this as a possibility. Given
that 4c is the first chunk where weapon choice becomes a meaningful
tactical variable, a dedicated help page sets the precedent for
weapon-property help and gives admin tools a landing target.

Suggested structure for `reach.template` (full draft for T9):

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">reach</ansi>

Weapons in DOGMud have a reach — how much physical space they need
to swing at full effectiveness. Reach matters when you're grappling:
long weapons don't have room to operate, and deal reduced damage as
pommel or hilt strikes instead.

<ansi fg="yellow">━━━ Reach and Grapple Position ━━━</ansi>

The penalty depends on how close the grapple is:

  <ansi fg="stat">Not grappling:</ansi>
    No penalty. Any weapon swings freely.

  <ansi fg="stat">Standing grapple (Clinch, BackStanding):</ansi>
    Moderate constraint. Short weapons (daggers) are fine. Medium
    weapons (swords) are somewhat penalised. Very long weapons
    (polearms) are strongly penalised.

  <ansi fg="stat">Ground grapple (Mount, Guard, Side Control, etc.):</ansi>
    Tight constraint. Only short weapons (daggers, fists, claws)
    stay at full effectiveness. Swords are significantly penalised.
    Spears and polearms are nearly useless.

<ansi fg="yellow">━━━ Weapon Families ━━━</ansi>

  <ansi fg="green">Short (no grapple penalty):</ansi>
    Daggers, fists, claws, bite, knuckles

  <ansi fg="yellow">Medium (partial penalty in clinch, stronger in ground):</ansi>
    Shortswords, axes, maces, swords, wands, sceptres

  <ansi fg="red">Long (strongly penalised in all grapple positions):</ansi>
    Greatswords, polearms, spears, staves, bows used as clubs

<ansi fg="yellow">━━━ Narration ━━━</ansi>

When a long bladed weapon is penalised in a grapple, its combat
narration shifts to reflect the reality: "you slam the iron sword's
pommel against the bandit's ribs" rather than slashing narration.
The damage is reduced; the fiction tracks it.

<ansi fg="yellow">━━━ Tactical Notes ━━━</ansi>

  - Carrying a dagger in your offhand is a sound counter to
    grapplers. Each weapon is evaluated independently per swing,
    so the dagger swings at full damage in mount while your
    main-hand sword is hampered.

  - Two-weapon fighters with dagger offhand naturally adapt: the
    dagger carries the damage load once the grapple lands.

  - You can manually remove a long weapon and fight unarmed. This
    is slower to set up than an offhand dagger but an option.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help grapple</ansi>,
  <ansi fg="command">help unarmed-combat</ansi>,
  <ansi fg="command">help weapon-combat</ansi>
```

---

### `identify` integration — OUT OF SCOPE (4c)

**Verdict: Do not add reach to identify output in T9.**

The design spec is explicit: "The examine / identify commands gain reach
in 4c (it's a stat), but no UI element surfaces 'reach: dangerous in
grapple' warnings. Player learns by feel + helpfile." Confirm this
decision is still correct post-T4:

- The identify.template describes how to use the spell, not what
  properties it reveals.
- No change to identify output was made in T1-T4.
- The `identify.template` KEEP-AS-IS verdict above is consistent with
  the spec.

Post-smoke action: if players repeatedly ask "why does my sword suck in
grapples" despite the grapple/reach helpfiles existing, add reach to the
identify output and update this file. Track as a 4f candidate.

---

### Bludgeon narration spot-check

**Concern:** Do the Bludgeoning combat-message templates read cleanly
when `{itemname}` is "iron sword" instead of "mace"?

**Finding (read `bludgeoning.yaml`):**

Spot-checking a Normal hit (toattacker, beginner):
- "You bash {target} with your {itemname}!" → "You bash the bandit
  with your iron sword!" — reads fine. Generic bash verb, not
  mace-specific.
- "Your {itemname} pounds into {target}!" → "Your iron sword pounds
  into the bandit!" — reads fine. Pounds is weapon-agnostic.
- "You pivot and smash through {target}'s guard!" — no itemname here;
  reads fine.

Spot-checking a Critical hit (toattacker, beginner):
- "Your {itemname} CRITICALLY SMASHES {target}!" → "Your iron sword
  CRITICALLY SMASHES the bandit!" — reads fine.
- "Your {itemname} CRUSHES {target}!" → "Your iron sword CRUSHES the
  bandit!" — reads fine.

Spot-checking a Weak hit (toattacker, expert):
- "You step forward and tap {target} lightly with your {itemname}." →
  "You step forward and tap the bandit lightly with your iron sword."
  — reads fine; "tap lightly" is generic enough.

**Conclusion:** The Bludgeoning templates use generic bash/pound/smash
verbs paired with `{itemname}` token interpolation. They do NOT say
"you swing your mace" or "you grip the handle of your club" — they say
"you bash with your {itemname}" which works cleanly with any weapon
name. No bespoke pommel-strike vocabulary needed. The narration swap
is validated as fiction-clean.

**One mild concern:** Some master-tier messages say "your weapon
crushes through" (no {itemname}). These are weapon-agnostic and read
fine. The miss messages say "your calculated strike misses as {target}
evades" — also weapon-agnostic. No confusion surfaced.

**No changes needed to `bludgeoning.yaml`.**

---

## Per-weapon helpfile inventory (T9 checklist)

| Template | Reach value | Verdict | Suggested 1-line copy |
|----------|------------|---------|----------------------|
| `iron-dagger.template` | 0.30 m (Stabbing default) | UPDATE | "Reach: Very short — fully effective in any grapple." |
| `iron-short-sword.template` | 1.00 m (Slashing default) | UPDATE | "Reach: Medium — penalised in grapples. Pair with a dagger offhand." |
| `steel-longsword.template` | 1.00 m (Slashing default) | UPDATE | "Reach: Long — awkward in grapples. The pommel still lands." |
| `lake-iron-hook-spear.template` | 2.00 m (per-item override needed) | UPDATE + YAML override | "Reach: Very long — nearly unusable in a grapple. Keep a backup." |

**Note on spear YAML:** `lake-iron-hook-spear` is typed as Stabbing
(piercing tip design). The Stabbing default reach is 0.30 m — correct
for a dagger, wrong for a spear. T9 must add `reach: 2.0` to the item
YAML to override the subtype default. This is the first per-item reach
override in the codebase and validates the outlier-override authoring
path from the spec.

**Caster weapon templates:** No dedicated helpfiles found for wand,
sceptre, or staff as distinct weapon items. If these templates are
added in future content, they should note: "Caster weapons deal reduced
melee damage in grapples (smaller reach multiplier) but spell damage
is unaffected — spells still cast normally from a grapple."

---

## Summary counts

| Category | Files reviewed | Updates needed | Keep-as-is |
|----------|---------------|----------------|------------|
| Combat/grapple helpfiles | 13 | 6 | 7 |
| Per-weapon helpfiles | 4 | 4 | 0 |
| New helpfile recommended | — | 1 (reach.template) | — |
| context.md files | 6 | 5 | 1 (items, already done) |

**Total files T9 must touch:** 10 (6 helpfiles + 4 weapon templates +
`reach.template` new file).

**Total files T10 must touch:** 5 (combat, state/position, characters,
hooks, configs context.md).

**YAML change required in T9:** `lake-iron-hook-spear` item YAML needs
`reach: 2.0` to correct the Stabbing-default mismatch.

---

## Surprising findings

1. **`iron-short-sword.template` already uses "reach" colloquially.**
   The phrase "warriors who value reach without sacrificing speed" uses
   reach as "blade length" — the design intent of the description. After
   4c, `reach` has an engine meaning. The existing text remains correct
   (the short sword does have more reach than a dagger) but the addition
   of the mechanical note makes the dual usage explicit. No conflict;
   just a coincidental vocabulary overlap.

2. **Stabbing default (0.30 m) makes `lake-iron-hook-spear` stealth-
   correct under the grapple system.** As currently authored (Stabbing
   subtype), the hook-spear gets the dagger's reach value and takes no
   grapple penalty. This is wrong for the fiction — a spear is not a
   dagger. This is the highest-priority YAML override catch from the
   audit. The spec anticipated these cases ("per-item overrides land
   post-smoke") but this one was predictable from the taxonomy alone:
   any item named "spear" and typed as Stabbing will be under-penalised.

3. **Bludgeoning subtype documentation error in T1 items/context.md.**
   The reach table lists `Bludgeoning` at 0.80 m, but `Bludgeoning` is
   a combat-message vocabulary subtype, not a weapon carry-subtype that
   authors set in item YAML. The `DefaultReachForSubtype` switch has no
   `case Bludgeoning`. The table implies authors can set
   `subtype: bludgeoning` on maces and get 0.80 m — but they'd actually
   get 0.0 (the default case). T10 must fix this before the confusion
   propagates to item authors.

4. **No existing per-weapon helpfile for caster weapons.** Wands,
   sceptres, and staves exist in the engine (Wand/Sceptre/Staff subtypes)
   but no `wand.template`, `sceptre.template`, or `staff.template` help
   files exist in the help directory. If these items are craftable or
   lootable, players have no dedicated help. Out of T9 scope (these items
   need to exist first), but logged.

5. **Bludgeoning templates are weapon-agnostic.** Every hit message uses
   verbs like "bash," "pound," "smash," "crush" without assuming a
   specific weapon shape. The `{itemname}` token is reliably present in
   the non-generic messages. No bespoke "iron sword used as a bludgeon"
   vocabulary is needed — the existing Bludgeoning set reads cleanly
   with any weapon name.

6. **Defense is NOT penalised for reach in 4c (by design).** The audit
   confirms no helpfile mentions that a defender parrying with a polearm
   in mount gets no reach penalty on their parry roll. This is a known
   4c non-goal (per the spec). If 4f adds defense-side reach, the
   `defense.template` and `attack.template` will need updates at that
   time.

---

## Post-audit notes for T9 execution

Voice conventions (match existing helpfiles):
- No hard stat numbers in player-facing text for damage amounts.
- Reach values (meters) ARE appropriate for per-weapon notes — they are
  mechanical specs the player can reason about, similar to how the bash
  helpfile says "50% of your normal attack damage." One-line reach notes
  in weapon helpfiles should use descriptive terms ("very short," "long")
  not raw numbers, per project SOP. Exception: per-item YAML is allowed
  to use `reach: 2.0` because it's authoring data, not player text.
- 80-char line wrap throughout.
- Use the `━━━` section-header style (yellow) and `<ansi fg="stat">` for
  stat names, consistent with existing combat helpfiles (see
  `attack.template` and `kick.template` for reference).
- `<ansi fg="command">` for command names, `<ansi fg="skill">` for skill
  names. See existing files for examples.

For `reach.template` (new file): model the overall structure on
`grapple.template` (short sections, tactical notes, See Also) rather
than the detailed `attack.template` (which is longer and more
mechanically exhaustive). The reach helpfile is a reference page, not
a tutorial.

For context.md updates (T10): use present-tense "fully shipped" voice
consistent with the chunk-4b pattern. Reference commit SHAs where
applicable (`bbb325b6`, `d7347ba3`, `95e6d992`, `f814e238`).
