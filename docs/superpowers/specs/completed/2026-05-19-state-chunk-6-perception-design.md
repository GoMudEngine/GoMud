# State Chunk 6 — Perception (Design)

**Status:** Draft 2026-05-19 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes`
**Predecessor chunks:** 5 (Presence, closed 2026-05-19), 4a-4f, 3, 2, 1, 0
**Successor chunks:** Centralized Messaging Framework (new, future) — consumes the Perception FSM that this chunk lands.
**Master spec:** `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md` §7

---

## 1. Problem Statement

The master combat-state-machines spec lists Perception as the sixth and final FSM (`Sighted | Blinded`) gating room broadcasts so blind / dark-room observers don't receive visual narration. During brainstorming the scope expanded — the user identified that the BROADER messaging problem (color-coded combat text by category, infrared anonymized rendering, centralized line-wrapping, the 228-broadcast-site audit, the headline companion-name leak) is bigger than a single chunk and deserves its own chunk dedicated to a centralized messaging framework.

Chunk 6 therefore lands ONLY the Perception state machine. It ships **dormant**, following the chunk 4a precedent (Position FSM shipped before chunk 4b wired it up). The buff and condition observers transition the machine correctly; no consumer reads the state yet. The future centralized messaging framework chunk will consume this primitive.

---

## 2. Design Goals

1. **Land the Perception FSM as a primitive.** Sighted / Blinded states, transition table, observer/veto wrappers — same shape as the other five state machines.
2. **Drive transitions from existing blind sources** (Buff 3, Buff 77, ConditionBlinded). Buff/condition lifecycle hooks fire the transitions; no new gameplay surface exposed to players.
3. **Zero player-visible behavior change.** Chunk 6 ships dormant. The dormant ship lets the FSM bake in the buff lifecycle while the messaging framework brews.
4. **Mob/player parity.** Same machine, same constructor, same observer wiring for both actor types.
5. **Preserve existing infrastructure.** `SendTextVisual`, `canSeeInRoom`, `sendRoomTextDarknessAware`, `ConditionBlinded` dodge penalty all unchanged.

---

## 3. State List and Transition Table

### 3.1 Two-state union enum

```go
package perception

type State int

const (
    Sighted State = iota  // default — eyes work
    Blinded               // any blind source active
)
```

Smallest of the six state machines. No transient states, no state-data structs.

### 3.2 Transition table

```go
var transitions = state.TransitionTable[State]{
    Sighted: {Blinded},
    Blinded: {Sighted},
}
```

### 3.3 Trigger constants

```go
const (
    TriggerBuffApplied      = "buff_applied"
    TriggerBuffExpired      = "buff_expired"
    TriggerConditionAdded   = "condition_added"
    TriggerConditionRemoved = "condition_removed"
)
```

### 3.4 Constructor + initial state

```go
func NewMachine() *perception.Machine
```

Returns Sighted. Same constructor for player and mob (no per-actor polymorphism, unlike chunk 5).

Persistence: NOT serialized (`yaml:"-"`). On load, characters with active blind buffs/conditions get the correct state via the buff/condition observer firing during `Validate()`.

---

## 4. Transition Rules

Observer-driven, fired by existing buff/condition lifecycle hooks:

| Event | Action |
|---|---|
| Buff 3 (Blinded) applied | `TransitionTo(Blinded, TriggerBuffApplied)` |
| Buff 77 (Flashbang Blindness) applied | `TransitionTo(Blinded, TriggerBuffApplied)` |
| Buff 3 or 77 expired/removed | If `!c.HasAnyBlindSource()` → `TransitionTo(Sighted, TriggerBuffExpired)` |
| `ConditionBlinded` added | `TransitionTo(Blinded, TriggerConditionAdded)` |
| `ConditionBlinded` removed | If `!c.HasAnyBlindSource()` → `TransitionTo(Sighted, TriggerConditionRemoved)` |

The `HasAnyBlindSource()` guard prevents flicker when a player has multiple overlapping blind sources and one expires.

### 4.1 `HasAnyBlindSource()` helper

```go
// HasAnyBlindSource returns true if any active blind source is currently
// affecting this character. Used by Perception expire observers to decide
// whether to transition back to Sighted when one of multiple overlapping
// sources clears.
func (c *Character) HasAnyBlindSource() bool {
    if c.HasFlagFromAnySource(buffs.Blinded) {
        return true
    }
    if c.HasFlagFromAnySource(buffs.FlashbangBlindness) {
        return true
    }
    if c.HasCondition(ConditionBlinded) {
        return true
    }
    return false
}
```

If `buffs.FlashbangBlindness` flag constant doesn't exist (only Buff 77 in YAML), it will be added as a 2-line addition to `internal/buffs/buffspec.go` during T1 implementation.

---

## 5. Per-Character Integration

### 5.1 `Character.Perception` field

```go
// Perception is the canonical state machine for "do this character's eyes
// work?" — Sighted / Blinded. Ships dormant in chunk 6: the machine
// transitions correctly via buff/condition observers but no consumer
// reads the state yet. The future centralized messaging framework chunk
// wires it into broadcast gating, infrared rendering, look-command
// blocking, etc. See internal/state/perception/context.md.
Perception                 *perception.Machine            `yaml:"-"`
```

### 5.2 Initialization (4 sites)

Same pattern as Presence (chunk 5):

1. `characters.New()` — sets `c.Perception = perception.NewMachine()`.
2. `characters.Character.Validate()` — nil-guard installing the default machine for YAML-loaded characters.
3. `mobs.Mob.Validate()` — unconditional overwrite after `Character.Validate()`. Same constructor as the player path, but the unconditional overwrite matches the Presence pattern for consistency.
4. `Character.ResetForMobInstance()` — sets `c.Perception = nil` so freshly shallow-copied mob instances get their own machine via `Validate()`.

### 5.3 Buff observer wiring

The exact wiring depends on whether the buff system already publishes lifecycle events. T1 implementation will:

- Check `internal/buffs/` for existing `OnApply` / `OnExpire` / `OnRemove` hooks.
- If hooks exist: subscribe a Perception router that dispatches blind buffs (3, 77) to the character's Perception machine.
- If hooks don't exist: hook directly in `Character.AddBuff()` / `Character.RemoveBuff()` (less clean but workable).

### 5.4 Condition observer wiring

`ConditionBlinded` lifecycle lives in `Character.AddCondition()` / `Character.RemoveCondition()` in `internal/characters/conditions.go`. Add direct calls there:

```go
// In AddCondition, when conditionType == ConditionBlinded:
if c.Perception != nil && c.Perception.State() == perception.Sighted {
    _ = c.Perception.TransitionTo(perception.Blinded,
        state.TransitionReason{Trigger: perception.TriggerConditionAdded})
}

// In RemoveCondition, when conditionType == ConditionBlinded:
if c.Perception != nil && c.Perception.State() == perception.Blinded && !c.HasAnyBlindSource() {
    _ = c.Perception.TransitionTo(perception.Sighted,
        state.TransitionReason{Trigger: perception.TriggerConditionRemoved})
}
```

No observer indirection needed — the condition lifecycle is already inside the characters package.

---

## 6. Behavior Matrix (Intent-Driven Tests)

PR = Perception. Every row becomes a RED-phase test. PE-001 through PE-011.

| ID | From | Trigger | Conditions | To | Notes |
|---|---|---|---|---|---|
| PE-001 | (init) | NewMachine | — | Sighted | Default |
| PE-002 | Sighted | Buff 3 applied | — | Blinded | Blindness spell / debuff |
| PE-003 | Sighted | Buff 77 applied | — | Blinded | Flashbang |
| PE-004 | Sighted | ConditionBlinded added | — | Blinded | Eye-damage mutation |
| PE-005 | Blinded | Buff 3 expired | no other blind source | Sighted | Recovery |
| PE-006 | Blinded | Buff 77 expired | no other blind source | Sighted | Recovery |
| PE-007 | Blinded | ConditionBlinded removed | no other blind source | Sighted | Recovery |
| PE-008 | Blinded | Buff 3 expired | Buff 77 still active | Blinded | Overlap → stay |
| PE-009 | Blinded | Buff 3 expired | ConditionBlinded still active | Blinded | Overlap → stay |
| PE-010 | Sighted | Buff 3 applied while already Sighted | — | Blinded | Idempotent-style entry |
| PE-011 | Blinded | Buff 3 applied a second time | — | Blinded | No-op; table rejects Blinded→Blinded |

---

## 7. Testing Strategy

### 7.1 Unit tests (new)

- `internal/state/perception/perception_test.go` — table-driven, one row per PE-NNN matrix ID. Pure-machine; uses a stubbed `HasAnyBlindSource` predicate so unit tests don't pull in real Character / buff state.

### 7.2 Integration tests (new)

- `internal/state/perception/integration_test.go` — exercises real Character + buff + condition lifecycle:
  - PE-INT-001: Apply Buff 3 → machine in Blinded.
  - PE-INT-002: Apply Buff 3, then ConditionBlinded, then expire Buff 3 → still Blinded.
  - PE-INT-003: Apply Buff 3, then ConditionBlinded, then remove condition, then expire Buff 3 → Sighted.
  - PE-INT-004: Re-applying Buff 3 while already Blinded → no transition (no log spam, no error).

### 7.3 No smoke pass

Chunk 6 ships **dormant**. There's no player-visible behavior change to smoke. The centralized messaging framework chunk (future) will run the AI smoke pass.

---

## 8. New Artifacts

| Path | Responsibility |
|---|---|
| `internal/state/perception/perception.go` | State enum, Machine wrapper, NewMachine, observer/veto wrappers |
| `internal/state/perception/transitions.go` | Transition table + trigger constants |
| `internal/state/perception/perception_test.go` | PE-001 through PE-011 unit tests |
| `internal/state/perception/integration_test.go` | Real Character integration tests for overlap and single-source paths |
| `internal/state/perception/context.md` | Package documentation |
| `internal/buffs/perception_observer.go` (or hook into existing buff lifecycle) | Buff lifecycle observer routing blind buffs to Perception |

## 9. Modified Files

| Path | Change |
|---|---|
| `internal/characters/character.go` | Add `Perception *perception.Machine` field; init in `New()`; reset in `ResetForMobInstance()` |
| `internal/characters/validate.go` | Nil-guard installs player-default machine |
| `internal/mobs/mobs.go` | Unconditional overwrite in `Validate()` (consistency with Presence pattern) |
| `internal/characters/conditions.go` | Fire Perception transitions on `AddCondition(ConditionBlinded)` and `RemoveCondition(ConditionBlinded)` |
| `internal/characters/sight.go` (new or extend existing) | `HasAnyBlindSource()` helper method |
| `internal/buffs/buffspec.go` | Add `FlashbangBlindness Flag` constant if not already present |
| `internal/state/context.md` | Cross-link perception as the sixth consumer |
| `internal/characters/context.md` | Note the new Perception field |
| `COMBAT_STATE_ROADMAP.md` | Mark chunk 6 as Done (dormant) |
| `PATCH_NOTES.md` | Brief entry noting the FSM ships dormant |

---

## 10. Sunset List

**None.** Chunk 6 ships dormant.

Existing infrastructure stays put. All consolidation / deletion / cutover moves to the future Messaging Framework chunk's scope:

- `room.SendTextVisual()` — unchanged.
- `actions/actor_mob.sendRoomTextDarknessAware()` — unchanged.
- `combat.canSeeInRoom()` + duplicate in `hooks/NewRound_DoCombat_helpers.go` — unchanged.
- `ConditionBlinded` dodge penalty — unchanged.
- `buffs.NightVision` flag + handling — unchanged.

---

## 11. Out of Scope (Explicit) — All Deferred to the Future Messaging Framework Chunk

- **Broadcast cutover** — the 228 `SendText*` call sites audit + visual/audio classification.
- **Composite predicates** (`CanSee`, `CanSeeClearly`, `CanSeeShapes`).
- **InfraredVision flag and anonymized "red shapes" rendering** for infrared observers.
- **Magical darkness** as a room-level effect or observer debuff that transitions to Blinded.
- **`SendTextVisual` extension** to consult Perception state.
- **`look` / `look <target>` blindness gating** ("You can't see anything — your eyes are useless.").
- **Color coding** of combat narration by event category (defense / hit / grapple / strike / counter / special).
- **Centralized line-wrapping** at the broadcast layer.
- **Companion-name-leak bug fix** — the headline bug class from the master spec that motivated chunk 6.

A separate project memory captures the full scope of the future Messaging Framework chunk so we don't lose the design context.

---

## 12. Risks / Open Questions

- **`buffs.FlashbangBlindness` constant.** Buff 77 exists in YAML; the code-side flag constant may not. T1 implementation verifies; if absent, adds a 2-line entry to `internal/buffs/buffspec.go`.
- **`buffs.Blinded` flag.** Buff 3 may not have a `Blinded` flag set (different from buff ID). T1 implementation verifies; if the flag is missing, drive the observer by buff ID instead of flag, OR add the flag (depending on the buff system's convention).
- **Buff lifecycle observability.** If the buff system doesn't already publish `OnApply` / `OnExpire` / `OnRemove` events, the Perception observer hooks directly in `Character.AddBuff` / `Character.RemoveBuff`. Both paths work; the choice is cosmetic.
- **Re-entry to Blinded while already Blinded.** Table rejects `Blinded→Blinded` (per PE-011). The observer must check current state before firing to avoid `ErrInvalidTransition` log noise. One-liner in each observer.

---

## 13. Success Criteria

1. `internal/state/perception/` package exists with the union enum + machine wrapper.
2. `Character.Perception` field populated for both players and mobs at construction time.
3. Buff 3 applied / expired fires correct Perception transition; verified by integration test.
4. Buff 77 applied / expired fires correct Perception transition; verified by integration test.
5. `ConditionBlinded` add / remove fires correct Perception transition; verified by integration test.
6. Overlap test: applying both Buff 3 and ConditionBlinded → only one Sighted→Blinded transition fires; removing one source while the other is still active stays Blinded; verified by integration test.
7. Behavior Matrix rows PE-001 through PE-011 all pass as RED-phase tests.
8. `go build ./...` clean; `go test ./...` green.
9. Server boots cleanly past data-file loading.
10. **No player-visible behavior change.** Smoke verification is unnecessary — this is a primitive ship.

---

## 14. Implementation Order (Preview for writing-plans)

1. New `internal/state/perception/` package + Behavior Matrix unit tests (RED → GREEN).
2. Wire `Character.Perception` field + per-actor constructors.
3. `HasAnyBlindSource()` helper method on Character.
4. `ConditionBlinded` add/remove integration in `characters/conditions.go`.
5. Buff observer integration (`buffs.Blinded` + `buffs.FlashbangBlindness`). Includes adding `FlashbangBlindness` flag constant if missing.
6. Integration tests (overlap + single-source cases).
7. Context.md sweep + roadmap close-out + patch notes.

Estimated 5-7 small tasks. No smoke, no field deletions, no large cutover.

---

## 15. Successor Chunk (Captured Separately)

The deferred messaging-framework work is captured as a project memory at `~/.claude/projects/.../memory/project_messaging_framework_chunk.md`. That memory carries the full scope (broadcast cutover, infrared anonymizer, color coding, line wrapping, look-command gating, companion-name leak fix) and serves as the seed for a future brainstorming session when we're ready to land it.
