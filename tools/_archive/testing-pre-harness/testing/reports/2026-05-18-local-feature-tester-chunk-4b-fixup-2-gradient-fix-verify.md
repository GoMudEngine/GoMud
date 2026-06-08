# Gradient Fix Verification Smoke — 2026-05-18

**Branch:** feature/mob-aliveness-1.3-crimes  
**Fix commit:** 54899f3b (lazy SetSelf in pair.go + Position_GrappleTick.go)  
**Session duration:** ~10 minutes  
**Mob:** Smuggler Enforcer (mobid 246)  
**Positions visited:** Clinch → B.Gnd → Mount → Guard → SC → Mount (multiple cycles)

---

## 1. Did gradient lines fire?

**YES — gradient lines are now firing prolifically.**

In the prior smoke (chunk-4b-fixup-2), ZERO of 36 templates fired in ~200 rounds. After the fix, gradient messages fired on every position transition and on ambient ticks within positions.

### Confirmed gradient messages observed:

**Transition / boundary messages:**
- "A scramble, a shift — and you're on the wrong end of the pin." (entry/Clinch boundary)
- "smuggler enforcer finds the leverage you missed. The position flips and suddenly they're the one in control." (upper_boundary_down — Controlling → Neutral)
- "You held it a half-second too long. smuggler enforcer reverses the position and pins you down." (upper_boundary_down variant)
- "Hooks dig under your hips and lift; you cartwheel forward into their guard." (Mount → Guard transition)
- "You plant a foot on the hip, push them sideways, and follow them to the top — side control." (Guard → SC)
- "Their near elbow frames high; you undercut it, knee-slide across the torso, and ride into mount." (SC → Mount)
- "A snap-down sets it up — you sprawl over the shoulder and slide into mount before they can recover." (Clinch → Mount)
- "You sense the turn coming and time it: as they face down you spin to the back, threading both legs around their hips." (SC → B.Gnd)
- "smuggler enforcer kicks off you and pops to their feet — the grapple breaks." (break message)

**Ambient / state messages (within a position):**
- "You and smuggler enforcer grind in the clinch — wrist fights and head-position grunts." (Clinch ambient)
- "Shoulders pressed together, breath ragged — you wrestle for the underhook." (Clinch ambient)
- "Pressure stays on. smuggler enforcer can't generate the leverage to move you." (ground hold ambient)
- "You ride high mount, knees driving into smuggler enforcer's biceps — their arms can't lift to defend." (Mount ambient)
- "You posture down and ride out smuggler enforcer's squirming." (Mount ambient)
- "Their arms tire from defending their face. You sit heavy and let the strikes through." (Mount ambient)
- "You settle your weight and let smuggler enforcer burn cardio trying to shift you." (Mount ambient)
- "You ride high in mount and rain elbows down." (Mount ambient)
- "You shift your weight onto smuggler enforcer's sternum and unload — short, vicious elbows from the top." (Mount ambient)

**Total distinct gradient lines observed: 19+** (session was only ~10 minutes; many more exist in the library)

---

## 2. Did position advance?

**YES — position advanced actively throughout the session.**

Positions progressed fluidly: Clinch → B.Gnd → Mount → Guard → SC → Mount across multiple grapple engagements. The position FSM is working correctly. Rounds to advance varied (typically 2-4 rounds per transition based on contested ticks).

---

## 3. combatstats Grapple Controller %

**Grapple Controller: 0.0%** (75 events tracked)  
**Non-Controller: 49.3%** (75 events)

The Grapple Controller hit rate is still showing 0.0%. This is a separate tracking issue from gradient firing — the position FSM and gradient messages are working correctly. The 0% Grapple Controller stat likely reflects a combatstats recording bug (controller hits aren't being attributed to the Controller bucket) rather than a grapple control problem. The gradient messages confirm the ControlLevel state machine IS transitioning correctly. This warrants a follow-up look at the combatstats tracking code but is NOT blocking.

---

## 4. Any panics or unexpected behavior?

**No panics observed.** Server ran cleanly throughout.

One minor observation: grapple breaks and re-initiations worked correctly. The mob occasionally escaped the grapple and re-engaged in melee, which required re-grappling. Grapple re-initiation was seamless.

---

## Summary

The fix at commit 54899f3b fully resolves the gradient message silence bug. The root cause (Character.Control machine had `self=ActorRef{}` because `NewMachine()` leaves it zero → boundary callbacks fired with zero ref → `characterFromRef` returned nil → `emitGradientMessage` early-returned silently) is confirmed resolved. Gradient flavor templates now fire on every position transition and ambient tick.

**Action item:** Investigate why `combatstats position` Grapple Controller % remains 0.0% — this appears to be a hit-attribution tracking bug separate from the gradient fix, but worth a dedicated look.
