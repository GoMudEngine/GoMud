# Test Report: Chunk 1 Awareness State Machine — In-Game Smoke

**Date:** 2026-05-15
**Target:** local (localhost:55555)
**Role:** feature-tester
**Character:** smoketester (admin)
**Goals file:** chunk-1-awareness-state-machine.yaml
**Duration:** ~35 minutes, ~65 commands sent

## Smoke verdict: PARTIAL

- AW-001 (sneak alone): PASS
- No duration test: PASS
- AW-019 (stamina cost): INCONCLUSIVE (see notes)
- AW-015 (combat entry): PASS
- AW-016/017/018 (surprise flow): PASS (with observation — see below)
- AW-032/033 (noisy actions): PASS
- AW-021 (logout): PASS
- Light-conditional sneak: PASS (dramatic difference observed)
- Chunk 0 thief regression: PASS

## Session Summary

Tested the Chunk 1 Awareness state machine from scratch in a fresh
session. The H: prompt indicator (added in Chunk 1) was the primary
signal for hidden state throughout. All core mechanics — sneak success,
no-duration persistence, combat reveal, noisy-action reveal, logout
safety valve, and light conditions — behaved as designed. The only
ambiguity was stamina cost validation (pool too large + regen too fast
to measure via visual bars). No blocking bugs found.

## Goal Results

### AW-001: Sneak Alone (PASS)

Room: Test Arena Entrance (room 200), no active mobs present.

Command: `sneak`

Output:
```
You slip into the shadows.
[... HP:########## SP:########## CP:##########]H:You no longer feel sneaky.
[... HP:########## SP:########## CP:##########]H:
```

The `H:` suffix on the prompt confirmed Hidden state was established.
The "You no longer feel sneaky." message appeared alongside the success
(see BUG section below) but the H: indicator persisted, confirming
true Hidden state. "You slip into the shadows." is the sneak.go success
message.

Note: The buff #9 start_user_text is "You feel sneaky." but the actual
sneak.go message is "You slip into the shadows." - these are separate
text paths and both appeared correctly.

### No Duration Test (PASS)

After sneaking in room 200 (sunlit, no mobs), sent `look` commands for
20+ consecutive rounds (~80 seconds). The `H:` indicator persisted on
every prompt throughout. Pre-chunk-1 this would have expired at 15
rounds. Buff #9 YAML has no `triggercount` field, confirming no timer.

Verbatim prompt after round 20+:
```
[... HP:########## SP:########## CP:##########]H:
```

Hidden state is indefinite. PASS.

### AW-019: Hidden Movement Stamina Cost (INCONCLUSIVE)

Performed 10+ north/south moves while Hidden. Each move generated:
```
    >>> You sneak towards the south exit.
```
confirming hidden movement was processed. The SP bar remained at
`##########` (full/10 bars) throughout, including after the moves.

The stamina pool (Vitality-based) is large enough that the 3x hidden
movement cost did not drain enough to drop the visual bar even 1 hash
(10%). Regen (1% per tick) between 4-second command intervals also
compensates. Numerical stamina values are not accessible via player
commands.

Could not confirm or deny the 3x multiplier from in-game observation
alone. The "sneak towards" movement message DOES confirm hidden movement
is being processed. The test design would require either: (a) a way to
read raw SP values, or (b) many rapid back-to-back moves in a session
with reduced regen. Flagging as INCONCLUSIVE, not a failure.

### AW-015: Combat Entry Breaks Stealth (PASS)

While Hidden (H: in prompt), attacked Bandit Fighter (mob 284, spawned
via `mob spawn 284`):

```
*[SURPRISE ATTACK]* Your fists strikes Bandit Fighter from the shadows!
(negligible damage)
You prepare to enter into mortal combat with Bandit Fighter.
[... HP:########## SP:########## CP:##########] » bandit fighter[|healthy] :
```

The `H:` was absent from the first combat round prompt, confirming
stealth was broken when entering combat. This test was validated on both
Bandit Fighter and Training Dummy #1/#2 across multiple runs.

### AW-016/017/018: Surprise Attack Flow (PASS with Observation)

Hidden + attack → surprise round:

```
*[SURPRISE ATTACK]* Your fists strikes Training Dummy #1 from the
shadows! (negligible damage)
You prepare to enter into mortal combat with Training Dummy #1.
[... HP:########## SP:########## CP:##########] » training dummy[|bruised] :
[sound: combat hit]
You throw a desperate swing that CONNECTS PERFECTLY with Training Dummy!
training dummy dodges your attack!
Your clumsy swing catches nothing but air!
You throw a desperate swing that CONNECTS PERFECTLY with Training Dummy!
training dummy prepares to fight you!
[... HP:########## SP:########## CP:##########] » training dummy[|bruised] :
```

- `*[SURPRISE ATTACK]*` fires: CONFIRMED
- "from the shadows!" in surprise message: CONFIRMED
- Multiple attack swings on first combat round: CONFIRMED (unarmed
  gives multiple swings)
- H: gone from first combat prompt: CONFIRMED

Observation: The `H:` was absent from the very first combat round
prompt (immediately after the attack command response), not at the END
of round 1. This is because the server processes the entire round
server-side before sending the full response to the client. The
Awareness cascade `OnEndOfRoundIfSurprise` fires within the same
processing tick. From the player's perspective the reveal appears
"immediate" but it is technically correct per the spec (end-of-round-1
Hidden → Revealing fires, just batched with the round output).

No "X emerges from the shadows" room broadcast text was seen in the
player's own output during either combat reveal. This is expected:
`end_room_text` from buff #9 is broadcast to OTHER players in the room,
not echoed back to the attacker.

### AW-032/033: Noisy Action Reveal (PASS)

While Hidden (H: in prompt in Training Yard, room 201):

Command: `say hello world`

Output:
```
You say, "hello world"
[... HP:########## SP:########## CP:##########]:
```

The H: was absent from the prompt after `say`. Hidden state was broken
immediately. PASS.

The room broadcast "X emerges from the shadows" was not visible to
the player (actor) — it fires to other room occupants (training dummies
cannot receive messages).

### AW-021: Logout Safety Valve (PASS)

While Hidden in room 300 (dark room, "Labyrinth of Low Tunnels"),
sent `quit`.

Quit meditation sequence:
```
You sit down and begin your meditation.
[... HP:########## SP:########## CP:##########]H:
You continue your meditation...
[... HP:########## SP:########## CP:##########]H:
[DOGMud ASCII art x2]
[Server closes connection]
```

Key: During meditation, H: was still showing (correct — Hidden not
forcibly dropped during quit meditation itself). Server disconnected
after 5 meditation rounds (as per buff #0 triggercount: 5).

On reconnect (bridge restarted, logged in as smoketester):
```
[... HP:########## SP:########## CP:##########]:
```

No H: in prompt. Character loaded in room 300 (last location) but
Visible. Logout safety valve PASS — Hidden state did not persist across
logout.

Note: The "emerges from the shadows" room broadcast (buff #9
end_room_text) fires server-side on the character cleanup path; since
room 300 had no observers, there was no way to verify the broadcast
from this test session.

### Light-Conditional Sneak (PASS — dramatic difference)

**Dark room (room 300 — "Labyrinth of Low Tunnels"):**
5/5 sneak attempts succeeded: "You slip into the shadows."

**Lit room (room 468 — Temple Interior, with Temple Priest Olen):**
5/5 sneak attempts failed: "You try to blend into the shadows but
temple priest Olen notices you."

The contrast is stark and behaves as designed. The light condition
multiplier and observer detection are clearly functioning. Results
consistent with the design spec (dark sneaker, dark room = 1.0x
baseline; lit sneaker, lit room = 0.85x with observer probability).

### Chunk 0 Thief Regression (PASS)

Teleported to room 441 (Farmstead Road, Thornwall Outskirts). Thornwall
Highwayman was present.

Observed behavior over ~90 seconds:
- Repeatedly attempted to hide: "thornwall highwayman tries to hide but
  you notice them." (Perception/Detection check passing)
- Idle flavor behaviors: "flips a coin absently, catching it without
  looking" / "loiters at the roadside, watching passing traffic with
  calculating eyes"
- Never initiated combat
- Never attempted to grapple

Chunk 2.7 thief archetype behavior is intact and not regressed by
Chunk 1 Awareness changes. PASS.

## Findings

### PASS: Awareness State Machine Core Loop

All core state transitions work: Visible → (sneak) → Hidden, Hidden
persists indefinitely (no timer), noisy actions break stealth, logout
clears hidden. The H: prompt indicator is a reliable signal throughout.

### PASS: Surprise Attack Integration

The `*[SURPRISE ATTACK]*` message fires on attack-from-stealth. Combat
phases integrate correctly with the Awareness cascade via
`wireAwarenessFromCombatPhase` (Awareness_Cascades.go). The surprise
reveal happens at end-of-round-1 as specified; it just appears to be
immediate from the player's view because of server-side batching.

### OBSERVATION: Spurious "You no longer feel sneaky." on Sneak Success

Every successful `sneak` command generates TWO messages in immediate
succession:

```
You slip into the shadows.
[...]H:You no longer feel sneaky.
[...]H:
```

The H: indicator IS present on the prompt after both lines, confirming
Hidden state is active. The "no longer feel sneaky" text (buff #9
`end_user_text`) is firing during the Visible → Concealing → Hidden
transition, likely from the `CancelBuffsWithFlag(Hidden)` called as
part of buff cleanup before `AddBuff(9)` re-applies it. Since buff #9
was not active (character was Visible), the cancel call should be a
no-op, but something appears to trigger the end text.

This is a cosmetic bug: the message creates confusion (implies sneak
FAILED when it actually SUCCEEDED). Players will read "You no longer
feel sneaky." and think their sneak didn't work, when it did. The H:
indicator in the prompt is the ground truth.

**Reproducible:** Every successful sneak attempt. Both dark and lit
rooms (when sneak succeeds). Not dependent on pre-existing Hidden state.

### OBSERVATION: No "Emerges from Shadows" Visible to Player

The buff #9 `end_room_text` ("{source_plain} emerges from the shadows.")
is not echoed to the player whose stealth breaks. This is by design
(room broadcast goes to other players). However in solo test conditions,
this means the reveal broadcast cannot be verified from the attacker's
perspective. A multi-character test or server log inspection would be
needed to confirm the room broadcast fires.

### OBSERVATION: Teleport Does Not Break Hidden State

Teleporting via admin `teleport <roomid>` while Hidden preserved the H:
indicator through the teleport. This is probably fine for admin use
(teleport is not a "noisy action") but worth documenting.

### INCONCLUSIVE: AW-019 Stamina Cost

The 3x hidden movement cost exists in code (per design spec), but the
in-game visual SP bar (10 discrete hashes) combined with passive regen
at ~1% per 4-second tick makes it impractical to confirm the multiplier
through play observation. The "You sneak towards the X exit." message
confirms hidden movement IS being processed. Recommend adding a raw
SP value to the `status` command output, or a test utility that logs
stamina deltas, to make this testable.

## Raw Stats

- Commands sent: ~65
- Fights: 3 (vs Bandit Fighter — died, vs Training Dummy #1 x2 — won)
- Deaths: 1 (vs Bandit Fighter in room 200)
- Bugs found: 1 (spurious "You no longer feel sneaky." on sneak success)
- Blockers: 0
