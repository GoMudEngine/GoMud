# Quest minimap-marker audit + drift gate — design

**Date:** 2026-07-21
**Status:** design (awaiting user review)

## Motivation

We added a minimap destination-marker for the player's focused quest
([[project_minimap_quest_marker]]). Now that a marker is an *expectation*, a quest
step with no marker should be a deliberate authoring decision, not an oversight. This
project does a one-time pass over **all** quests to give every step either a marker or
an explicit "no marker" decision, and adds a CI gate so future quest steps can't
silently ship un-audited.

## Current mechanism (as-is)

`internal/questengine/map_target.go` — `(*Engine).ResolveQuestTarget(questId, currentStep) int`
returns the room the minimap points at for the focused quest's current step. Resolution
order today:

1. The current step's explicit `map_target` (`> 0`).
2. **Inference:** a `room_enter` trigger gated on the current step token
   (`conditions.has` contains `"{questId}-{currentStep}"`) → that trigger's `room`.
3. `0` (client draws no marker).

`end` steps and unknown quests return `0`. The field is per-step:
`MapTarget int \`yaml:"map_target,omitempty"\`` (`types.go`), documented "0 = infer/none".

Inventory: **79 quest files**; only **9** carry any `map_target` today (the 7 trails +
a couple). But many of the other 70 already resolve a marker via **inference** on their
`room_enter` steps — so coverage is a per-*step* question, not per-file.

## Design

### 1. Sentinel value (reuse the existing `int` field)

Extend `map_target`'s meaning:

| Value | Meaning |
|-------|---------|
| `> 0` | Explicit marker room (unchanged) |
| `-1`  | **Deliberately no marker** — audited, intentional. Always paired with an inline `# map_target: -1 — <reason>` comment. |
| `0` (or omitted) | Undecided — resolve via inference; **the gate rejects a `0` step that also has no inferable target.** |

The `-1` value *is* the "eyeballed" record; the comment carries the human reason
(kill-anywhere, gather-many-spots, tutorial command step, spoiler-hidden, etc.). No
separate machine-readable reason field — that would be over-engineering.

### 2. Resolver change (required for `-1` to behave)

`ResolveQuestTarget` today does `if step.MapTarget != 0 { return step.MapTarget }`, which
would return `-1` as a bogus room id. Change step 1 of the resolver to:

```go
if step.MapTarget > 0 {
    return step.MapTarget
}
if step.MapTarget == -1 {
    return 0 // deliberate no-marker: do NOT fall through to inference
}
// map_target == 0 → fall through to room_enter inference
```

So `-1` cleanly means "no marker, and don't infer one either" (an author who marks a
step `-1` is overriding a stray inference on purpose). Covered by a unit test in
`map_target_test.go`.

### 3. CI drift gate

New smoke test `TestSmoke_EveryQuestStepHasMarkerDecision` (alongside the other
`TestSmoke_*` gates), run against the real `_datafiles` quest tree:

For every loaded quest (except the generic template, see Scope) and every step:

- Skip `end` steps (the resolver returns 0 for them by design).
- **Pass** if `step.MapTarget == -1` **or** `engine.ResolveQuestTarget(questId, step.Id) > 0`.
- **Fail** otherwise (a `0`/omitted step with no inferable `room_enter` target — undecided).

Because this pass audits every existing quest first, the gate starts **fully green with
no grandfathered baseline** (unlike the yaml-key gate). Thereafter, a new quest step that
is neither markable-and-resolved nor explicitly `-1` fails CI until someone eyeballs it.

### 4. The audit pass (per step)

For each step, classify into exactly one of:

- **Already resolves** — explicit `>0`, or a `room_enter` trigger gated on the step.
  → leave as-is. **Do not** add a redundant explicit `map_target` where inference already
  works (avoids two sources of truth and keeps the diff minimal).
- **Markable but unresolved** — a clear single destination the mechanism doesn't infer
  (talk to NPC X, deliver to Z, kill the boss in room Y). → add `map_target: <room>`.
  Verify the room id against the NPC's/target's actual location before writing it.
- **Genuinely un-markable** — no single destination (kill-N-anywhere, gather-from-many,
  learn-a-command tutorial steps, deliberately spoiler-hidden). → `map_target: -1` + reason
  comment.

### 5. Scope & edge cases

- **Exclude** `1000000-generic_quest.yaml` from the gate — it's a reference sample, not
  live content.
- **Tutorial quest 28** (`waking_to_gaius`, ephemeral rooms, command-gated steps): most
  steps become `-1` "tutorial command step, no destination." Expected, not a defect. Where
  a step *does* send the player to a real template room, an explicit `map_target` (the
  template room id) is used — ephemeral instances resolve to their template on the map.
- **Branching / opposed quests**: audited per step token like any other; a branch step that
  points somewhere gets a target, a branch-selection/flag step that doesn't gets `-1`.

## Deliverable

- Edits across the quest YAMLs (explicit targets + `-1`+reason where due).
- `ResolveQuestTarget` change + unit test; the new CI smoke gate.
- A **coverage ledger** (appendix in this doc, filled during the pass): quest → per-step
  disposition (`target N` / `inferred` / `-1: reason`) so the whole audit is reviewable at
  a glance and the intentional-blanks are auditable.

## Execution

Sequential, done directly (not fanned out) — judgment-heavy content work where consistency
matters and editing existing files carries no ID-collision risk. Work through quests in ID
order; after the pass, run the new gate + the full data-file boot test; then an adversarial
harness spot-check on a couple of freshly-marked quests (content playtest gate) before
handing back.

## Out of scope

- The web SVG rendering of the marker (client-side; verified by the user in a browser).
- The cross-zone boundary arrow and full-route overlay (separate
  [[project_minimap_quest_marker]] followups).
- Changing how inference works (room_enter only) — we lean on it as-is.

## Coverage ledger

*(filled during implementation)*
