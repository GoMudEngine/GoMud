# Test Report: Chunk 4f position-system smoke (feel-tester pass)

**Date:** 2026-05-19
**Target:** local
**Role:** feel-tester
**Character:** smoketester (admin, Awakened apprentice generalist, keen Wil)
**Goals file:** chunk-4f-position-system-smoke.yaml
**Duration:** ~6 min wall, ~13 commands

## Session Summary

Played the grapple loop naturally against two Thornwall Thugs in Back
Alley West. Goal of the pass was qualitative — assess viscerality,
variety, coherence, pacing of the chunk 4a-4f position system as a
single integrated experience. Did NOT pursue specific goal-by-goal
mechanical verification (that was the feature-tester pass).

## Findings

### IMMERSION: Grapple narration is consistently visceral and BJJ-accurate
Over two grapple sequences I observed a rich variety of position-flavor
templates that read like a knowledgeable BJJ practitioner wrote them:

- Mount-controller hold rounds: "You settle your weight and let
  thornwall thug burn cardio trying to shift you." / "You shift your
  weight onto thornwall thug's sternum and unload — short, vicious
  elbows from the top." / "You ride high mount, knees driving into
  thornwall thug's biceps — their arms can't lift to defend." /
  "You ride high in mount and rain elbows down." / "Sweat and copper.
  You ride the mount and drop knuckles like pistons." — five distinct
  templates in a single Mount stretch. Variety is genuine; not
  one-template-repeating.

- Position-change narration: "Their stance breaks. You drag them flat
  and climb on top, knees high in mount." (Clinch → Mount Advance) /
  "thornwall thug bridges hard and hip-escapes — your mount slides
  off into side control." (Mount → SC Degrade) / "thornwall thug
  explodes free — they're back on their feet before you can grab
  again." (Escape).

- Reversal: "You held it a half-second too long. thornwall thug
  reverses the position and pins you down." Visceral, properly tense.

This is one of the strongest IMMERSION wins of the entire chunk 4 arc.
The chunk 4b-fixup-era flavor authoring (~280 templates in
`grapple_outcomes.yaml`) is paying off heavily here.

### IMMERSION: Mob idle behavior reads well alongside combat
On entry to the Back Alley I caught "thornwall thug cracks knuckles
with deliberate slowness" — a mob-aliveness idle line that sets up
the fight tonally. Not directly chunk 4, but interacts well with the
grapple loop.

### IMMERSION: Skill-progression callouts are satisfying
"*** You feel your unarmed-combat skills sharpening! ***" and
"*** You feel your weapon-combat skills sharpening! ***" landed
during natural combat. Quick, no-numbers, mood-positive.

### PACING: Reversal-into-Escape in two consecutive rounds reads jarringly
In one fight I had:

> Round N: "You held it a half-second too long. thornwall thug
> reverses the position and pins you down."
> Round N+1: "thornwall thug explodes free — they're back on their
> feet before you can grab again."

A Reversal (which moves the thug into the controller role) followed
immediately by an Escape (which breaks the grapple entirely) feels
abrupt — like the thug "wins" twice in a row when they should have
been settling into their new top position. Mechanically this is
fine (the position resolver can run any outcome each round), but
narratively the transition skips a beat. Not a bug, but a CONCERN
worth logging for a future flavor pass.

### PACING: Thug fights against keen-stat character end fast
This is a balance-not-flavor observation. With keen Strength + steel
longsword, a Thornwall Thug goes from healthy to dead in 3-5 rounds.
The grapple flow rarely gets a chance to develop a long Mount
position because the thug is dead within ~3 Mount rounds. Future
playtesters at lower power level would experience this differently;
not a chunk 4 issue per se.

### CLARITY: "in control" sentence-start lowercase
> "*** in control, you press forward and pound heavily into
> Thornwall Thug! ***"

Sentence starts lowercase ("in control,..."). Minor capitalization
issue in a flavor template. Pre-existing; not chunk 4f.

### CLARITY: "a aggressive posture" — article agreement
> "While grappling on the ground, you adopt a aggressive posture
> against Thornwall Thug."

Should be "an aggressive posture". Pre-existing template issue.

### OBSERVATION: Combat narration outside grapple is also varied
Standing-combat templates I observed in just a few rounds: "Your
steel longsword TEARS THROUGH Thornwall Thug!" / "Your steel
longsword CRITICALLY SMASHES Thornwall Thug!" / "Your masterful
feint opens Thornwall Thug completely - you ANNIHILATE them!" /
"You time a perfect counter and DEVASTATE Thornwall Thug!" /
"You feint low and crack Thornwall Thug with a brutal uppercut!"
/ "Reading every move, you deliver a FLAWLESS BASH to Thornwall
Thug!" / "Your combination builds to a THUNDEROUS FINISHING
BLOW!" — variety is solid, crit/non-crit narration distinct,
intensity scales with hit quality.

### OBSERVATION: Position prompt indicator gives clean situational awareness
Prompt updates per round with `Clinch` / `Mount` / `SC` / `B.Gnd`
indicators in the HP/SP/CP bar. Easy to glance at and know your
position. Works well alongside the prose.

### OBSERVATION: No double-narration issues observed
Looked specifically for the "same outcome narrated twice in one
round" symptom flagged in goals. Did not observe it across two
full grapple sequences. The narration flow is one outcome per
round consistently.

## Raw Stats

- Commands sent: ~13
- Fights: 2 (sequential, both with Thornwall Thug 105)
- Deaths (player): 0
- Spells cast: 0 (intentionally — feel-tester focus was combat flow)
- Items used: 0
- Sub attempts observed: 0 (none fired in the two fights; not a
  bug — sub eligibility requires drift-margin > alpha, which
  doesn't fire every round)

## Final Assessment

The chunk 4 position system reads as a coherent, immersive system
from a player's perspective. The flavor authoring (chunk 4b-fixup +
chunk 4b-fixup-2) is the dominant contributor to the felt-quality
of the loop. Chunk 4f's helpfile softening (T3) reads naturally
inline with the rest of the grapple prose.

Two pre-existing minor template issues surfaced ("in control"
lowercase, "a aggressive posture" article) — neither is chunk 4f
work and both fall under the same followup as the bigger flavor
pass would. The Reversal-into-Escape sequence is the only thing
I'd flag for a future flavor-pacing pass.

The feel of the system overall is the strongest part of the chunk
4 arc. The chunk 4f changes (chance-based disruption, helpfile
softening) integrate without disrupting any of that.
