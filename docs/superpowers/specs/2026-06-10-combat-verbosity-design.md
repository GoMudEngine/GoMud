# Combat Text Verbosity — Design Spec

**Date:** 2026-06-10
**Status:** Approved pending user review

## Goal

Let players reduce combat text spam with a per-user `combatverbosity`
setting — `full` (current behavior, default), `medium` (hits only), `light`
(one compact tally per round) — without losing the information they can't
afford to miss. The worst offender, other people's fights in the same room,
is reduced one step further than the player's own fights.

## Decisions (settled during brainstorm)

| Question | Decision |
|---|---|
| Filter scope | Tiered by involvement: chosen level applies to your own fights; spectated fights render one step lower (full→medium, medium→light, light→light) |
| Always-shown floor | Deaths & kill blows; status-changing moves (knockdowns, grapple transitions, stuns, disarms); every hit that damages you personally — all in full prose at any level |
| Light-mode format | Compact tally, one line per attacker→defender pair per round: "You strike the marsh wolf twice (serious wounds); it fails to land a blow." No raw numbers — wound tier via `GetDamageDescription` |
| Architecture | Approach A: gate at the combat round's message drain + per-round aggregator. Messaging pipeline untouched |
| Phase-1 scope | Auto-attack round narration only. Special moves, taunts, and spells keep current always-shown behavior |

## Architecture

### Where the gate lives

Combat narration is already collected per-audience on
`combat.AttackResult` (`internal/combat/attackresult.go`):
`MessagesToSource`, `MessagesToTarget`, `MessagesToSourceRoom`,
`MessagesToTargetRoom`, `MessagesToRoomOld` — each a `[]TaggedMessage`
(`messaging.Category` + text). The round hook
(`internal/hooks/NewRound_DoCombat_unified.go` and helpers) drains these to
`user.SendText` / `room.SendText`. That drain is the single seam:

- It knows the recipient's role: source messages go to the attacker, target
  messages to the defender, room messages to spectators.
- Each line's `Category` already classifies it: `CategoryHit*` = landed
  hits; `CategoryDodge` / `CategoryParry` / `CategoryBlock` = avoided
  swings (misses).
- The `AttackResult` itself carries the data the aggregator needs:
  `SwingEvents` (per-swing hit/crit/damage), `DamageToTarget`,
  `Hit`, actor identities from the surrounding hook code.

The exact drain function name(s) are confirmed during plan-writing; the
seam is the point where `MessagesTo*` lists are iterated and sent.

### Effective verbosity per recipient

```
effective(viewer, fight) =
    viewer.CombatVerbosity                  if viewer is the attacker or defender
    oneStepLower(viewer.CombatVerbosity)    otherwise (spectator)

oneStepLower: full→medium, medium→light, light→light
```

Mobs have no verbosity (mob-directed messages unaffected). AI-flagged
accounts default to `full` like everyone else (playtest harness relies on
full text).

### Per-line gate

For each TaggedMessage being drained to a player recipient:

| Effective level | Hit lines (CategoryHit*) | Miss/defense lines (Dodge/Parry/Block) |
|---|---|---|
| full | pass | pass |
| medium | pass | suppress |
| light | suppress → aggregate | suppress → aggregate |

**Floor overrides (checked before the gate, any level):**
- Lines representing damage to the viewer personally (i.e. hit lines in
  `MessagesToTarget` when the viewer is the defender) always pass in full.
- Death/kill-blow messages always pass (they are emitted outside the
  per-swing drain; the gate simply must not touch them).
- Status-changing move messages (kick/trip/bash/grapple/stun/disarm
  categories) always pass — phase-1 scope keeps the gate off those
  categories entirely.

Categories not explicitly gated (everything that isn't `CategoryHit*` or
`CategoryDodge/Parry/Block`) always pass. The gate is an allowlist of
suppressible categories, so new combat text is verbose-by-default (safe).

### Light-mode aggregator

A small per-round accumulator keyed by (viewerUserId, attackerName,
defenderName):

- Counts swings that hit and swings that missed per direction, tracks the
  worst single-hit damage tier (`GetDamageDescription(maxHit, defenderMaxHP)`).
- Filled at the drain point when a line is suppressed-to-aggregate; the
  AttackResult's `SwingEvents` provide accurate counts (don't parse text).
- Flushed at the end of the combat phase of the round (after all fights in
  the room are processed, same hook), emitting one compact tally line per
  pair to each subscribed viewer via the normal `SendText` path with a
  combat category (rendering/color preserved).
- Cleared every round. No cross-round state; a viewer leaving the room
  mid-round simply gets no flush (accumulator entries are per-viewer).

Tally templates (no raw numbers, 80-char wrapped):
- Outgoing: `You strike the marsh wolf twice (serious wounds); it fails to
  land a blow.`
- Spectated: `Velk batters the bog shambler (serious wounds); it claws Velk
  (light wounds).`
- Whiff round: `You trade swings with the marsh wolf; neither side draws
  blood.`
- Singular/plural handled ("once", "twice", "three times", "again and
  again" for 4+).
- Incoming damage half of your own tally compresses to counts/whiffs since
  each landed incoming hit already showed in full (floor rule).

### Setting storage & UX

- `UserRecord.CombatVerbosity string` (`yaml:"combatverbosity,omitempty"`,
  "" = full), beside `LineWidth` — same persistence pattern.
- Set via the existing settings command: `set combatverbosity
  full|medium|light` (exact wiring matches how `set linewidth` is
  implemented; discovered during plan-writing). Invalid values list the
  options.
- `help combatverbosity` helpfile; mention in the `set` helpfile if one
  exists.
- Web client: no special handling phase 1 — it receives the same reduced
  text. (A dashboard toggle could call the same command later.)

## Phase-1 scope guard

In scope: melee/auto-attack round narration (the `MessagesTo*` drain).
Out of scope (always shown, unchanged): special moves (kick/trip/bash),
taunts, rally/warcry, spells, combat initiation/flee text, death messages,
progression messages. A later pass can extend the gate if these prove
spammy.

## Testing

- Unit tests on the gate: category × role × level matrix (table-driven).
- Unit tests on the aggregator: counts from SwingEvents, worst-tier
  selection, singular/plural phrasing, whiff rounds, multi-fight rooms,
  spectator vs participant keys.
- Hook-level test: a simulated round with one participant at each level +
  a spectator, asserting exactly which lines each receives (the hooks
  package has existing combat-round test scaffolding to build on).
- Live smoke: a fight at each level + a spectator window, verifying floor
  rules (incoming damage always full, deaths always shown).

## Out of scope / future

- Extending the gate to spells/special moves/taunts.
- Narrative-style summaries (compact tally chosen).
- Per-channel verbosity (e.g. different setting for party members'
  fights) — spectator tiering covers the need for now.
- GMCP/web structured combat feed.
