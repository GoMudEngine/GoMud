# Feel-Test Report: Chunk 4b-fixup-2 Grappling

**Date:** 2026-05-19
**Role:** feel-tester (playtester)
**Character:** smoketester
**Commands sent:** ~45
**Grapples observed:** 6 (4 player-initiated, 2 mob-initiated)

## Headline verdict

The grappling system reads convincingly as a ground-fighting discipline. The
narrative lines are punchy and grounded in real wrestling language. The main
friction is that fights end too fast for the player to explore the full
Clinch → Mount arc — mobs die before position control can be established,
which means mount-phase narration was never observed. Ship the Clinch layer;
mount-apex lines need a live test with a tankier mob.

## Strongest impressions

- **Clinch entry lines are crisp.** "You grapple smuggler enforcer,
  transitioning to clinched position!" and the prompt indicator
  `[☀️ HP:...Clinch]` give immediate positional feedback with no ambiguity.
  Players know exactly what state they're in.
- **Control-reversal lines land hard.** "You held it a half-second too long.
  smuggler enforcer reverses the position and pins you down." and "smuggler
  enforcer finds the leverage you missed. The position flips and suddenly
  they're the one in control." — both feel like real grappling moments.
  The specificity ("half-second," "leverage you missed") sells the sport.
- **Sweep integration is satisfying.** The `⚡ SWEEP!` prefix gives sweeps
  their own visual identity. Seeing "⚡ SWEEP! You dodge and sweep their legs
  out! They crash to the ground!" while unarmed felt like a meaningful
  positional win, even without a full grapple cycle following it.

## Friction / annoyances

- **Mob health pool too small for proper grapple testing.** The thornwall thug
  dies in 1-2 rounds of unarmed combat. The smuggler enforcer survives longer
  but still dies before mount can be demonstrated at the player's current
  stat/skill level. Need a dedicated testing mob with large HP for grapple
  feel-passes, or a `mob sethp` admin command.
- **Grapple cooldown fires during clinch-entry round.** The sequence
  "You grapple enforcer, transitioning to clinched position!" followed
  immediately on the *same* round by a break line, then "You need a moment
  to recover before attempting another special move." means the player can't
  re-grapple to advance past Clinch for 1-2 rounds. This feels punishing: you
  successfully establish position and are immediately locked out of pressing
  the advantage. The cooldown should probably not trigger on a successful
  entry, only on failed attempts.
- **Prone recovery failure message is tonally flat.** "You attempt to stand,
  but slip back down in the chaos of battle!" works mechanically but feels
  generic compared to the evocative grapple lines. Could be more visceral:
  "You push up to a knee — an elbow drives you right back down."

## Gradient messaging specifically

Boundary-cross lines fired consistently. Examples observed:

1. **Clinch entry (player-initiated):** "You grapple smuggler enforcer,
   transitioning to clinched position!" — clean, zero ambiguity.
2. **Clinch entry (mob-initiated):** "smuggler enforcer grapples you,
   transitioning to clinched position!" — symmetric with the player version,
   reads naturally from the player's perspective.
3. **Control loss (clinch reversal):** "You held it a half-second too long.
   smuggler enforcer reverses the position and pins you down." — the best
   single line in the set. Specific, physical, plausible.
4. **Grip failure / break:** "Your grip fails. smuggler enforcer scrambles up
   and out of reach." and "smuggler enforcer kicks off you and pops to their
   feet — the grapple breaks." — both feel distinct and non-repetitive.
5. **Stamina depletion in grapple context:** "You're getting gassed — your
   Standing is hard to maintain." — perfectly placed. Firing at ~20% stamina
   with a Clinch indicator on the prompt felt MMA-authentic.

All lines felt natural in context. None triggered out of position.

## Variety check

Within a single extended fight (~8+ rounds), the following break lines
appeared more than once:

- "smuggler enforcer kicks off you and pops to their feet — the grapple
  breaks." — appeared 3 times across different grapple cycles. Not jarring
  because each cycle was genuinely a different grapple attempt, but if a
  single clinch phase lasted longer, this could feel repetitive.
- "Your grip fails. smuggler enforcer scrambles up and out of reach." —
  appeared twice. Synonymous with the above in effect; could alternate more
  aggressively.

No single clinch phase was long enough to trigger internal narration repeats.
The clinch-stable narration line "Hand-fighting in the pocket. Neither of you
gives an inch." fired once and didn't repeat within that hold. Good.

## Coherence check

No obvious mismatches observed. All clinch lines fired while the prompt showed
`Clinch`. All prone lines fired while the prompt showed `Prone`. The riposte
line "⚔ RIPOSTE! smuggler enforcer turns the parry into a lightning
counter-strike!" fired during standing combat and clinch combat — slightly
odd to see a "parry → counter" in the clinch, since parrying a weapon while
grappling is ambiguous. Not immersion-breaking, but technically inconsistent.

One line that feels tonally lighter than its surroundings: "A scramble, a
shift — and you're on the wrong end of the pin." — This is fine prose but
the em-dash pacing feels different from the more punchy lines around it.
Minor.

## Specific revisions you'd suggest

1. **Vary the grapple-break pool more.** "kicks off you and pops to their
   feet" and "scrambles up and out of reach" are doing too much of the work.
   Add 2-3 more break variants: "uses your momentum against you and rolls
   free", "bucks hard and creates enough space to stand", "bridges out from
   under you".

2. **Tone up the prone-recovery failure line.** Replace "You attempt to stand,
   but slip back down in the chaos of battle!" with something more physical:
   "You push up to a knee, but a well-placed shove drives you back to the
   ground." or "You start to rise — a boot catches your shin and you go down
   again."

3. **Consider suppressing the grapple cooldown on successful entry.** If you
   just grabbed someone and put them in clinch, you shouldn't be locked out
   of pressing to mount for 1-2 rounds. The cooldown makes sense on a failed
   attempt (you overextended) but not a successful one.

4. **Add a clinch-stable narrative variant for weapon-vs-grapple.** When the
   mob has a weapon and the player is in clinch, something like "You smother
   the blade arm, keeping it trapped against their body" would sell the
   weapon-control aspect of the clinch. Currently weapons fire normally in
   clinch with no narrative acknowledgment.

5. **Mount-phase lines need an in-game verification pass.** Mount was never
   reached during this session because mobs die before position advances that
   far. A dedicated test with `mob spawn <tanky mob>` or a `mob sethp` command
   would let a tester verify mount narration fires, strikes land with the
   mount damage bonus, and escape lines are distinct from break lines.

## Stats

- Commands sent: ~45
- Grapples observed: 6
- Distinct gradient lines seen: 9
- Mount strike-apex lines seen: 0 (mount never reached)
- Awkward / immersion-breaking lines: 1 (riposte firing in clinch)
- Panics or missing-template strings: 0
