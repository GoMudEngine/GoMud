# perception — Package Documentation

## Overview

The `internal/state/perception` package is the seventh consumer of the
`internal/state` framework — completing the combat-state-machines arc
(chunks 0-6). It defines a two-state FSM (`Sighted | Blinded`) that
gates "does this character's eyes work?" semantics.

**Status (shipped DORMANT, 2026-05-19):**

The machine transitions correctly via existing buff/condition lifecycle
hooks (Buff 3 Blinded, Buff 77 Flashbang Blindness, ConditionBlinded).
NO CONSUMER reads the state yet. The future centralized messaging
framework chunk wires this primitive into broadcast gating, infrared
"red shapes" rendering, look-command blocking, color coding by event
category, and centralized line wrapping. See the
`messaging-framework-chunk` project memory for the full scope of that
successor work.

The dormant ship follows the chunk-4a precedent (Position FSM shipped
DORMANT before chunk 4b wired writers + readers).

---

## State enum

| State | Semantics |
|---|---|
| Sighted | Default — eyes work |
| Blinded | Any active blind source (Buff 3, Buff 77, or ConditionBlinded) |

Two states. No transient states. No state-data structs.

---

## Transition table

```
Sighted → {Blinded}
Blinded → {Sighted}
```

Re-entry (Sighted→Sighted, Blinded→Blinded) is NOT in the table.
Callers must check current state before firing transitions — the
inline guards in `Character.AddBuff` / `AddCondition` / etc. handle
this.

---

## Trigger sources

| Source | File | Hook |
|---|---|---|
| Buff 3 (Blinded) | `_datafiles/world/dogmud/buffs/3-blinded.yaml` | `Character.AddBuff` / `RemoveBuff` |
| Buff 77 (Flashbang Blindness) | `_datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml` | `Character.AddBuff` / `RemoveBuff` |
| ConditionBlinded | `internal/characters/conditions.go` | `Character.AddCondition` / `RemoveCondition` |

Detection is by buff ID (not by flag) because the existing buff YAMLs
don't carry a blindness-specific flag — adding one would require data
file edits. Buff IDs `BuffIdBlinded = 3` and `BuffIdFlashbangBlindness
= 77` are constants in `transitions.go`.

---

## Helper: `HasAnyBlindSource`

`Character.HasAnyBlindSource()` (in `internal/characters/sight.go`)
returns true if any of the three sources is currently active. Used
by expire-paths to decide whether to fire Blinded→Sighted when one
of multiple overlapping sources clears.

Important implementation detail: the buff checks use
`Buffs.TriggersLeft(id) > 0` rather than `Buffs.HasBuff(id)`. The
buff system marks a buff expired (TriggersLeft=0) on RemoveBuff but
defers map-entry pruning to the next game-tick. HasBuff checks map
membership only and returns true for expired-but-not-yet-pruned buffs,
which would break the overlap guard. TriggersLeft > 0 returns false
immediately after RemoveBuff — correct semantic.

---

## Construction

`NewMachine()` returns a Machine in Sighted state. Same constructor
for both player and mob — no per-actor polymorphism (unlike chunk 5
Presence).

`Character.Perception` field initialized at four sites:

1. `characters.New()` — player default.
2. `Character.Validate()` — nil-guard for YAML-loaded characters.
3. `mobs.Mob.Validate()` — unconditional overwrite.
4. `Character.ResetForMobInstance()` — reset to nil so fresh mob
   instances get their own machine.

---

## Integration points (current and future)

| When | Where |
|---|---|
| **Now (dormant)** | Transitions fire correctly; no consumer reads state. |
| **Future (messaging framework chunk)** | `room.SendTextVisual` consults `Perception.State()`; `look` command blocked when Blinded; infrared anonymizer transforms visual broadcasts. See `messaging-framework-chunk` project memory. |

---

## Testing

- `perception_test.go` — Behavior Matrix unit tests (PE-001 through
  PE-009): pure-FSM coverage.
- `integration_test.go` — real-Character integration (PE-INT-001
  through PE-INT-007): overlap + single-source paths via `AddBuff`
  / `AddCondition` / etc.

No smoke pass — chunk 6 ships dormant; no player-visible behavior
to verify in-game.
