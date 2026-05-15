# Combat State Machines — Chunk 1: Awareness

> **Second chunk of the combat-state-machines redesign**
> (master spec: `2026-05-13-combat-state-machines-design.md`).
> Builds the second machine — `Visible | Concealing | Hidden |
> Revealing` — on the `internal/state/` framework. Bundles a
> long-standing Hidden mechanic refresh (no duration, stamina
> cost for movement, light-conditional sneak score) with the
> FSM port. Replaces buff-#9-as-state-of-truth with Awareness
> state, keeping buff #9 as the effect carrier. Sunsets the
> unused buff #20 (very_hidden).

## Goal

Today, "is this character hidden?" is answered by checking buff
#9 via `HasBuffFlag(buffs.Hidden)`. The buff carries a hardcoded
15-round duration, a `cancel-on-combat` flag, and stat-mod
payloads — three concerns conflated into one expiring buff.

This chunk:

- **Makes Awareness state explicit** as a state machine
  (`Visible/Concealing/Hidden/Revealing`) with documented
  transitions, vetoes, and cascade handlers.
- **Removes Hidden's arbitrary duration.** Stealth persists
  until something explicitly breaks it (a detection roll lost,
  combat entered, light change, logout) — never via a silent
  timer expiry.
- **Adds a stamina cost for moving while Hidden** (default
  `3.0×` multiplier on movement stamina), stacking
  multiplicatively with encumbrance. Stealth becomes a
  meaningful tactical resource trade, not a free travel mode.
- **Models light-conditional sneak scoring** via a four-way
  conditional in `CalcSneakScore`:
  - Dark sneaker, dark room: baseline
  - Dark sneaker, lit room: 0.9× (alert observers)
  - Lit sneaker, lit room: 0.85× (blends in)
  - Lit sneaker, dark room: 0.5× (beacon in darkness)
- **Closes the surprise-attack handshake from chunk 0.** The
  Awareness machine subscribes to Combat Phase's
  `OnEndOfRoundIfSurprise` callback to fire Hidden → Revealing
  at end of the first combat round (after all weapon swings).
- **Sunsets buff #20 (`very_hidden`)** — dead content with no
  production consumers.

## Non-goals

- **Per-observer detection state.** Stealth remains global — a
  character is either Hidden (all observers must roll to detect)
  or Visible. Per-observer awareness is a much bigger redesign
  out of scope for this chunk.
- **Multi-round concealment.** `Concealing` state slot exists
  for future expansion, but the sneak roll resolves
  synchronously today and stays synchronous.
- **Sneak skill mechanics rework.** The underlying Dexterity +
  skullduggery roll math is unchanged. Only the modifier system
  (light) and the lifecycle (no duration, no auto-expiry) are
  changing.
- **`very_hidden` tiering.** Survey showed buff #20 is dead
  content with no game-content consumer. Delete; do not
  resurrect as a tier system.
- **Buff system redesign.** Buff #9 stays as the effect carrier.
  This chunk does not collapse the buff into the machine.

## Architectural musts

Confirmed during brainstorm:

1. **Awareness machine OWNS the state answer.** Other systems
   query `Character.IsHidden()` (Awareness predicate); buff #9
   becomes the cascade-driven effect carrier. Stat-mod payloads
   stay on the buff. Awareness transitions add/remove the buff
   via cascade handlers.

2. **No duration.** Buff #9 YAML drops `triggerrate` and
   `triggercount`. The buff persists indefinitely until an
   Awareness transition removes it via cascade. The
   `cancel-on-combat` flag stays for the legacy
   `CancelBuffsWithFlag` paths that haven't migrated yet, but
   the canonical removal path is the Awareness machine
   transitioning Hidden → Revealing → Visible.

3. **Hidden movement costs stamina.** New config
   `Balance.HiddenMoveStaminaMultiplier` (default `3.0`). Stacks
   multiplicatively with the existing encumbrance multiplier in
   `internal/usercommands/go.go`. Stamina exhaustion does NOT
   auto-break stealth — the player decides whether to risk a
   detection failure on the next movement attempt.

4. **Light-conditional sneak score replaces flat
   `× 0.5 if EmitsLight`.** `CalcSneakScore` gains a `room`
   parameter and the four-way conditional from the brainstorm.
   Sneaker-side only (per the discussion that found observer-
   and-sneaker stacking produced wrong-direction penalties).
   Config knobs for each branch in `Balance`.

5. **Event-driven detection rolls — no periodic ticks.** Rolls
   fire on specific triggers (entry, movement, search/look,
   light change). No round-tick scanning of "who's hidden."
   This matches the "no duration / lasts until discovered"
   design intent.

6. **Combat Phase × Awareness handshake closes the surprise
   loop from chunk 0.** Chunk 0 left `OnEndOfRoundIfSurprise`
   exposed; chunk 1 subscribes. SurpriseAttack reasons preserve
   Hidden through Engaging and Engaged; the end-of-first-round
   cascade fires Hidden → Revealing. Supports dual/triple/
   Extra-Arms weapon configurations (every swing in the
   surprise round gets the bonus).

7. **Logout safety valve.** Players logging out while Hidden
   are forced through Hidden → Revealing → Visible synchronously
   before session cleanup. Buff #9 removed; room broadcast
   fires. Mob despawn doesn't need a separate cascade — the
   instance destruction takes the state with it.

8. **Hard cutover within chunk, no compat wrappers.** Hidden's
   surface is small (~30 readers, ~15 writers). Direct
   migration. Buff #9 stays as the side effect; if any caller
   accidentally still queries `HasBuffFlag(buffs.Hidden)` after
   the migration, the buff still mirrors Awareness state via
   cascade, so they get a correct answer with a slow path.

9. **Activity-machine pre-wire for Concealing veto.** Activity
   machine lands in chunk 3, but the veto rule "Activity ≠ Free
   blocks Visible → Concealing" is needed today. Pre-wire via
   existing `CastingState != nil || CraftingState != nil`
   check, same pattern as chunk-0 Combat Phase vetoes. When
   chunk 3 lands, the callback gets repointed to the real
   Activity machine.

10. **Aggression is NOT a per-command category.** Two mechanical
    trigger paths break stealth:
    - **Combat Phase entry** cascades to Awareness (covers
      attack, taunt, offensive cast, grapple init, etc.).
    - **Detection roll failed** explicitly transitions
      Awareness → Revealing (covers movement detection, observer
      search, light state change, skullduggery skill failures).

    No global aggression flag, no command-category registry, no
    "physical action" event. New stealth-breakers in the future
    are added as explicit `TransitionToRevealing` calls in
    their command handlers.

## Awareness machine design

### States

```go
package awareness

type State int

const (
    Visible State = iota
    Concealing
    Hidden
    Revealing
)
```

### Per-state data

```go
// VisibleData is empty — the default state has no per-state data.
type VisibleData struct{}

// ConcealingData captures the in-flight sneak attempt.
// Today's synchronous resolve means this is set briefly during
// the transition handler then immediately cleared. Future
// multi-round concealment would populate RoundsUntil similar to
// Combat Phase EngagingData.
type ConcealingData struct {
    RoundsUntil int // 0 in chunk-1 implementation (synchronous)
}

// HiddenData carries hidden-state metadata that doesn't belong
// on the buff. Today empty; reserved for future light-source
// tracking or per-observer awareness lists.
type HiddenData struct{}

// RevealingData captures the in-flight reveal cascade. Set
// during the cascade run, cleared at the same-tick transition
// to Visible. Reason carries context for cascade subscribers
// (why is this character being revealed?).
type RevealingData struct {
    Reason state.TransitionReason
}
```

### Transition table

```go
var validTransitions = state.TransitionTable[State]{
    Visible:    {Concealing},
    Concealing: {Hidden, Visible},  // success vs failure
    Hidden:     {Revealing, Visible}, // Revealing for cascade; Visible for direct force (logout, death)
    Revealing:  {Visible},
}
```

### Trigger reasons

```go
const (
    TriggerSneakCommand          = "sneak_command"
    TriggerSneakSuccess          = "sneak_success"
    TriggerSneakFailed           = "sneak_failed"
    TriggerCombatEntered         = "combat_entered"
    TriggerSurpriseRoundEnd      = "surprise_round_end"
    TriggerMovementDetected      = "movement_detected"
    TriggerObserverSearch        = "observer_search"
    TriggerLightChange           = "light_change"
    TriggerSkullduggeryFailed    = "skullduggery_failed"
    TriggerNoisyAction           = "noisy_action"  // say/shout/whisper/rally/warcry/taunt
    TriggerLogout                = "logout_safety_valve"
    TriggerDeath                 = "death_cascade"
    TriggerForceVisible          = "force_visible"
)
```

## Behavior Matrix

Each row drives a RED-phase test. ID prefix `AW-` = Awareness.

### Basic entry/exit (sneak resolve)

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-001 | Visible | `sneak` command | No observers in room | → Concealing → Hidden (synchronous); cascade applies buff #9; room broadcast: `"X disappears into the shadows."` | Free sneak when alone; standard mechanic. |
| AW-002 | Visible | `sneak` command | Observers present; sneak roll fails against at least one observer | → Concealing → Visible; that observer sees `"X tries to hide but you notice them."` | Failed sneak is observable; gives the observing player real signal. |
| AW-003 | Visible | `sneak` command | Observers present; sneak roll succeeds against all observers | → Concealing → Hidden; cascade applies buff #9 | Standard stealth success. |

### Detection on room entry / observer presence

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-004 | Hidden | Hidden actor moves into a room with observers | per-observer detection roll fires; all sneak rolls won | Hidden persists | Observer didn't beat the sneak score; stealth holds. |
| AW-005 | Hidden | Observer enters room with hidden actor | per-observer detection roll | If any roll lost by sneaker → Hidden → Revealing | New observer's perception challenges the existing stealth. |
| AW-006 | Hidden | Either entry trigger; detection roll won by observer | (forced) | → Revealing → Visible same-tick; room broadcast `"X emerges from the shadows."` (observer-side: detection-specific message) | Standard stealth break. |
| AW-007 | Hidden | Either entry trigger; detection roll won by sneaker | (forced) | Hidden persists | Sneak score beat perception; stealth holds. |

### Movement detection

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-008 | Hidden | Hidden actor moves (room change) | per-observer detection roll fires at destination | If any roll fails → Hidden → Revealing → Visible | Movement creates noise/visibility; per-observer roll. |
| AW-009 | Hidden | Hidden actor moves (room change) | sneak rolls won against all observers | Hidden persists in new room | Quiet movement maintains stealth. |

### Observer-initiated search

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-010 | Hidden | Observer runs `look <actor>` against a hidden actor | detection roll fires for that one observer | If roll lost → Hidden → Revealing → Visible | Targeted look is an active detection attempt. |
| AW-011 | Hidden | Observer runs `search` or `scan` | per-hidden-actor detection rolls fire for that observer | Each hidden actor whose sneak loses → Revealing → Visible (independent decisions) | General-area search probes all hidden actors. |

### Light state changes

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-012 | Hidden | Sneaker's emission state changes (equips/removes torch; casts/cancels light spell; gains/loses glowing mutation) | re-roll fires for all observers using new modifiers | If any roll lost → Hidden → Revealing → Visible | Becoming a light source is a state change that observers can react to. |
| AW-013 | Hidden | Room light state changes (someone enters with only light; someone leaves with only light) | re-roll fires for all hidden actors in the room | Same outcome per actor | Environmental light change re-evaluates all stealth. |
| AW-014 | Hidden | Observer has NightVision; both are in a dark room | (per-observer detection roll) | The sneak score for THIS observer's roll uses the lit-room modifier (`0.9×` if sneaker not lit, `0.85×` if sneaker is lit). The room "is dark" globally but "is effectively lit from this observer's POV" via NightVision. | NightVision logically equates to lit-room conditions for the observer's perception; treat the sneaker as if they were in a lit room when computing against this observer specifically. |

### Combat-entry cascade

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-015 | Hidden | Combat Phase Idle → Engaging | Trigger.reason == TriggerAttackCommand (non-surprise) | → Revealing → Visible same-tick via cascade | Standard "attack breaks stealth." |
| AW-016 | Hidden | Combat Phase Idle → Engaging | Trigger.reason == TriggerSurpriseAttack | NO cascade — Hidden persists through Engaging | Surprise round preserves stealth setup. |
| AW-017 | Hidden | Combat Phase Engaging → Engaged with EngagedData.SurpriseLeft = true | (cascade chain) | Hidden persists; combat round can fire surprise-bonus damage on all weapon swings | Surprise round fully resolves with bonus before reveal. |
| AW-018 | Hidden | Combat Phase OnEndOfRoundIfSurprise callback fires | (after surprise round resolves) | → Revealing → Visible | Reveal at end of round, after all dual/triple/Extra-Arms swings consumed surprise. |

### Stamina cost for movement

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-019 | Hidden | Player movement step | (no encumbrance) | Movement stamina cost = `BaseCost × HiddenMoveStaminaMultiplier` (default 3.0) | Stealth costs stamina; tunable. |
| AW-020 | Hidden | Player movement step | Over carry capacity | Cost = `BaseCost × EncumbranceMod × HiddenMoveStaminaMultiplier` | Multiplicative stacking; heavy stealth is brutally expensive (5×3 = 15× cost). |

### Logout safety valve

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-021 | Hidden | Player logout / disconnect | (forced) | → Revealing → Visible synchronously before session cleanup; buff #9 removed; room broadcast fires | No leftover Hidden state on relogin; observers see the leave. |
| AW-022 | Hidden | Mob despawn | (instance destroyed) | State goes with the instance; no separate cascade needed | Mob instance lifetime owns the Awareness machine. |

### Vetoes

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-023 | Visible | `sneak` command | `CastingState != nil || CraftingState != nil` (Activity ≠ Free pre-wire) | Veto, error returned, no transition | Can't sneak mid-cast or mid-craft. |

### Light-conditional sneak score

| ID | EmitsLight | Room Lit | sneakMod | Rationale |
|----|------------|----------|----------|-----------|
| AW-024 | false | false | 1.0 | Baseline — best stealth conditions. |
| AW-025 | false | true | 0.9 | Alert observers in lit space; modest hit. |
| AW-026 | true | false | 0.5 | Beacon in darkness — worst case. |
| AW-027 | true | true | 0.85 | Blends in among other lit things; small hit. |

### Persistence

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-028 | (n/a) | NewMachine() | (construction) | State() == Visible | Sensible default at construction. |
| AW-029 | (any) | Server restart | (boot) | All characters boot to Visible; no Awareness state persists | Match chunk-0 Combat Phase pattern; the world awakens uncommitted. |

### Revealing state semantics

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-030 | Hidden | Any Hidden → Revealing transition | (cascade fires) | Room broadcast text fires; buff #9 removed; cascade subscribers notified (Combat Phase, aliveness substrate) | Revealing is the atomic "reveal happens now" state where all observer-facing effects fire. |
| AW-031 | Revealing | (cascade complete) | (forced) | → Visible same-tick | Revealing is conceptual; no multi-round duration. |

### Noisy action stealth break

| ID | Start | Trigger | Conditions | Outcome | Rationale |
|----|-------|---------|------------|---------|-----------|
| AW-032 | Hidden | `say` / `shout` / `whisper` (room-broadcast variants only) | (forced) | → Revealing → Visible | Speaking aloud reveals position; you cannot maintain stealth while talking. |
| AW-033 | Hidden | `rally` / `warcry` / `taunt` | (forced) | → Revealing → Visible | These are loud broadcast verbs. `taunt` enters Combat Phase too (cascade fires double); `rally`/`warcry` may not enter combat directly but still break stealth via explicit reveal. |

## Hidden mechanic redesign details

### No duration

`_datafiles/world/dogmud/buffs/9-hidden.yaml` changes:

```yaml
# BEFORE:
buffid: 9
name: Hidden
description: You're very sneaky.
secret: false
triggerrate: 1 round        # <-- DROP
triggercount: 15            # <-- DROP
flags:
  - hidden
  - cancel-on-combat
start_user_text: "You feel sneaky."
start_room_text: "{source_plain} disappears into the shadows."
end_user_text: "You no longer feel sneaky."
end_room_text: "{source_plain} emerges from the shadows."

# AFTER (chunk 1):
buffid: 9
name: Hidden
description: You're very sneaky.
secret: false
# No trigger fields — buff has no automatic expiry. Awareness
# machine drives addition/removal via cascade handlers.
flags:
  - hidden
  - cancel-on-combat
start_user_text: "You feel sneaky."
start_room_text: "{source_plain} disappears into the shadows."
end_user_text: "You no longer feel sneaky."
end_room_text: "{source_plain} emerges from the shadows."
```

The `cancel-on-combat` flag stays for defense-in-depth — any
caller who explicitly calls `CancelBuffsWithFlag(buffs.Hidden)`
during the migration window or by accident still cleans the
buff up. The canonical removal path post-chunk-1 is the
Awareness machine.

### Stamina cost site

`internal/usercommands/go.go` movement-cost calculation gains a
new factor:

```go
moveCost := baseRoomMoveCost
if user.Character.IsOverCapacity() {
    moveCost *= encumbranceMultiplier(user.Character)
}
if user.Character.IsHidden() {
    moveCost *= cfg.HiddenMoveStaminaMultiplier  // default 3.0
}
// existing stamina deduction
```

Place the check at the actual stamina-deduction site (grep for
existing encumbrance multiplier application).

### `CalcSneakScore` refactor

`internal/actions/skill_helpers.go`:

```go
// BEFORE:
func CalcSneakScore(char *characters.Character) float64 {
    base := float64(char.Stats.Dexterity.ValueAdj) +
        combat.SkillMultiplier(char.GetSkillLevel(skills.Skullduggery)) * 25.0 +
        mutationStealthBonus(char.Mutations)
    if char.EmitsLight() {
        base *= 0.5
    }
    return base
}

// AFTER (chunk 1):
// CalcSneakScore now takes an effectiveLit bool — the room is
// "effectively lit" from the observer's POV if either the room
// is lit OR the observer has NightVision. Callers (detection
// roll sites) compute this per-observer before calling.
func CalcSneakScore(char *characters.Character, effectiveLit bool) float64 {
    base := float64(char.Stats.Dexterity.ValueAdj) +
        combat.SkillMultiplier(char.GetSkillLevel(skills.Skullduggery)) * 25.0 +
        mutationStealthBonus(char.Mutations)

    cfg := configs.GetBalanceConfig()
    emits := char.EmitsLight()

    switch {
    case emits && !effectiveLit:
        base *= cfg.SneakModEmitsLightDarkRoom   // default 0.5
    case emits && effectiveLit:
        base *= cfg.SneakModEmitsLightLitRoom    // default 0.85
    case !emits && effectiveLit:
        base *= cfg.SneakModNoLightLitRoom       // default 0.9
        // else: baseline (no mod applied)
    }
    return base
}

// Convenience helper for the common case where the caller has
// the room and observer in hand.
func CalcSneakScoreVsObserver(sneaker, observer *characters.Character, room *rooms.Room) float64 {
    effectiveLit := room.GetVisibility() >= 1 ||
        observer.HasFlagFromAnySource(buffs.NightVision)
    return CalcSneakScore(sneaker, effectiveLit)
}
```

NightVision observer in a dark room → `effectiveLit = true` →
sneak score uses the lit-room modifier (0.9× or 0.85×) when
rolled against THIS observer. Same room, no NightVision
observer → `effectiveLit = false` → baseline (or beacon, if
sneaker emits). Per-observer evaluation is critical here; the
same sneaker may roll differently against different observers
in the same room.

Every detection-roll caller passes both observer and room.
Already accessible at each site (`internal/hooks/go.go`
room-entry detection, observer search commands, etc.).

### Light state change detection

New hook in `internal/hooks/Awareness_LightChange.go`:

```go
// Fires on:
//  - Room transitions (someone enters/leaves with only light)
//  - Sneaker emission state changes (equips/removes glowing item,
//    casts/cancels light spell, gains/loses glowing mutation)
//
// For each affected hidden actor in the room, re-roll detection
// against all observers using the new modifiers. On loss,
// transition Awareness Hidden → Revealing.
```

Wire to the existing event triggers:
- `events.RoomChange` listener (player entry/exit)
- `events.EquipmentChange` listener (item equipped/removed)
- `events.SpellCast` listener for light spells
- `events.MutationChange` listener for glow mutations

### Config defaults

Add to `Balance`:

```yaml
HiddenMoveStaminaMultiplier: 3.0
SneakModEmitsLightDarkRoom: 0.5
SneakModEmitsLightLitRoom: 0.85
SneakModNoLightLitRoom: 0.9
```

## Awareness × Combat Phase interaction

### Awareness informs Combat Phase entry

`internal/usercommands/attack.go` and `internal/mobcommands/attack.go`:

```go
// BEFORE: check buff
isHidden := user.Character.HasBuffFlag(buffs.Hidden)
trigger := combatphase.TriggerAttackCommand
if isHidden {
    trigger = combatphase.TriggerSurpriseAttack
}

// AFTER (chunk 1): check Awareness state
trigger := combatphase.TriggerAttackCommand
if user.Character.IsHidden() {
    trigger = combatphase.TriggerSurpriseAttack
}
```

### Combat Phase cascades to Awareness

New hook in `internal/hooks/Awareness_Cascades.go`:

```go
func init() {
    characters.OnCharacterCreated(wireAwarenessFromCombatPhase)
}

func wireAwarenessFromCombatPhase(c *characters.Character) {
    // Subscribe to Combat Phase Engaging transitions.
    c.CombatPhase.Inner().AfterTransition("awareness_combat_cascade",
        func(from, to combatphase.State, r state.TransitionReason) {
            if from == combatphase.Idle && to == combatphase.Engaging {
                if r.Trigger == combatphase.TriggerSurpriseAttack {
                    return // preserve Hidden for surprise round
                }
                if c.Awareness.State() == awareness.Hidden {
                    _ = c.Awareness.TransitionToRevealing(state.TransitionReason{
                        Trigger: awareness.TriggerCombatEntered,
                    })
                }
            }
        })

    // Subscribe to Combat Phase end-of-surprise-round callback.
    c.CombatPhase.OnEndOfRoundIfSurprise(func(r state.TransitionReason) {
        if c.Awareness.State() == awareness.Hidden {
            _ = c.Awareness.TransitionToRevealing(state.TransitionReason{
                Trigger: awareness.TriggerSurpriseRoundEnd,
            })
        }
    })
}
```

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/state/awareness/awareness.go` | NEW | State enum, data types, Machine wrapper |
| `internal/state/awareness/transitions.go` | NEW | Valid transitions, trigger constants |
| `internal/state/awareness/rules.go` | NEW | Transition methods, veto/cascade wiring helpers |
| `internal/state/awareness/awareness_test.go` | NEW | Behavior Matrix tests (AW-001 through AW-031) |
| `internal/state/awareness/context.md` | NEW | Package documentation |
| `internal/characters/character.go` | MODIFY | Add `Awareness *awareness.Machine` field; `IsHidden()` predicate; init in `New()`/`Validate()` |
| `internal/hooks/Awareness_Vetoes.go` | NEW | Activity pre-wire veto (Casting/Crafting blocks Concealing) |
| `internal/hooks/Awareness_Cascades.go` | NEW | Combat Phase subscriptions + buff #9 add/remove cascade |
| `internal/hooks/Awareness_LightChange.go` | NEW | Light-state-change event handler; re-roll fires |
| `internal/hooks/Logout_AwarenessCleanup.go` | NEW | Logout safety valve cascade |
| `internal/actions/skill_helpers.go` | MODIFY | `CalcSneakScore` takes room param; four-way conditional |
| `internal/actions/sneak.go` | MODIFY | Use `Awareness.TransitionToConcealing(...)` instead of direct `AddBuff(9, ...)` |
| `internal/actions/steal.go` | MODIFY | Replace `CancelBuffsWithFlag(buffs.Hidden)` with `Awareness.TransitionToRevealing(...)` |
| `internal/actions/plant.go` | MODIFY | Same |
| `internal/actions/remove_equip.go` | MODIFY | Same |
| `internal/actions/shadow.go` | MODIFY | `IsHidden()` check from Awareness, not buff |
| `internal/usercommands/go.go` | MODIFY | Stamina cost multiplier when `IsHidden()`; movement detection routes through Awareness |
| `internal/usercommands/skill.skullduggery.sneak.go` | MODIFY | Read state via `IsHidden()`; transitions via Awareness |
| `internal/mobcommands/sneak.go` | MODIFY | Same |
| ~30 reader sites | MODIFY | `HasBuffFlag(buffs.Hidden)` → `IsHidden()` |
| `internal/configs/balance.go` | MODIFY | New config fields (HiddenMoveStaminaMultiplier, three sneak mod knobs) |
| `_datafiles/config.yaml` | MODIFY | Default values for new config fields |
| `_datafiles/world/dogmud/buffs/9-hidden.yaml` | MODIFY | Drop `triggerrate` and `triggercount` |
| `_datafiles/world/default/buffs/20-very_hidden.yaml` | DELETE | Dead content, no consumers |
| `internal/characters/context.md` | MODIFY | Document Awareness field, IsHidden() predicate |
| `internal/behaviortree/context.md` | MODIFY | Note that conditions/actions reading Awareness use the new predicate |
| `internal/hooks/context.md` | MODIFY | Document the new cascade/veto/lightchange files |
| `COMBAT_STATE_ROADMAP.md` | MODIFY | Mark chunk 1 Done; update progress |

## Smoke scenarios

In-game validation. Each maps to one or more Behavior Matrix
rows.

1. **Sneak in empty room (AW-001).** Player walks into an empty
   room and runs `sneak`. Expect: room is empty, sneak succeeds
   trivially, player becomes Hidden, room broadcast text fires.

2. **Sneak with observer (AW-002, AW-003).** Player runs `sneak`
   in a room with at least one other player or hostile mob.
   Expect: observable failure message OR clean success based on
   the roll.

3. **Hidden movement stamina cost (AW-019).** Hidden player
   walks 5 rooms. Compare stamina delta to baseline (Visible
   player walks same 5 rooms). Expect: ~3× the drain.

4. **Lit room walk-through (AW-013, AW-025).** Hidden player
   walks into a lit room (e.g., a temple or torchlit corridor)
   with an observer present. Expect: detection roll uses 0.9×
   sneak; observer may detect.

5. **Torch-equip break (AW-012, AW-026).** Hidden player equips
   a torch (or casts a light spell) in a dark room with
   observers. Expect: re-roll fires; with the 0.5× modifier,
   detection is likely; player transitions to Visible.

6. **Combat-entry stealth break (AW-015).** Hidden player runs
   `attack <visible target>` (non-stealth attack). Expect:
   Combat Phase Engaging fires; Awareness cascades Revealing;
   room broadcast.

7. **Surprise attack flow (AW-016, AW-017, AW-018).** Hidden
   player runs `attack <target>` from stealth. Expect: Engaging
   with SurpriseAttack reason; surprise bonus on all swings;
   end-of-round reveal cascade. Repeat with dual-wield to
   confirm both weapons get the surprise bonus before reveal.

8. **Logout safety valve (AW-021).** Player runs `sneak`,
   becomes Hidden, then runs `quit`. Expect: room broadcast
   fires showing the player leaving stealth before the
   disconnect. On reconnect: player is Visible.

9. **Concealing veto while casting (AW-023).** Player begins
   casting a multi-round spell. Mid-cast, runs `sneak`. Expect:
   veto error returned; player remains Visible.

10. **Chunk-2.7 thief regression re-verification.** Walk a
    character past the Thornwall highwayman after chunk 1
    lands. Verify the thief archetype still works correctly
    (the smoke that closed chunk 0). No regression from the
    Awareness migration.

## Sunset list

- `_datafiles/world/default/buffs/20-very_hidden.yaml` — DELETE (dead content)
- `_datafiles/world/dogmud/buffs/9-hidden.yaml` — `triggerrate` and `triggercount` removed
- All explicit `AddBuff(9, ...)` calls migrated to `Awareness.TransitionToConcealing(...)` or removed
- All explicit `CancelBuffsWithFlag(buffs.Hidden)` calls migrated to `Awareness.TransitionToRevealing(...)` or removed
- ~30 `HasBuffFlag(buffs.Hidden)` reads migrated to `IsHidden()`
- Old flat `× 0.5 if EmitsLight` in `CalcSneakScore` replaced by the four-way conditional

## Risks / known limitations

- **Concealing today is synchronous; the state slot is mostly
  bookkeeping.** Future multi-round concealment work can extend
  `ConcealingData.RoundsUntil` and the OnRoundTick advance,
  similar to Combat Phase's Engaging implementation.
- **`Revealing` is same-tick; the state slot is mostly atomic.**
  Worth keeping the slot because cascade subscribers can hook
  the moment of revelation cleanly. Could be collapsed to a
  cascade fire without an intermediate state, but the framework
  symmetry (Visible→Concealing→Hidden mirrored by Hidden→
  Revealing→Visible) is worth the consistency.
- **`CalcSneakScore` signature change.** Every caller must
  pass the room. Some test fixtures will need updates. Caught
  by compile error; migration is mechanical.
- **Light state change handler is new infrastructure.** No
  existing event listener for "torch lit" or "room light
  changed." Need to identify the right event hooks and wire
  them carefully. The chunk's design includes scaffolding for
  the four trigger sources (RoomChange / EquipmentChange /
  SpellCast / MutationChange) but the exact integration may
  need iteration during implementation.
- **Activity machine pre-wire.** Chunk 3 will rewire this once
  the real Activity machine lands. Document the connection
  point clearly so the rewire is mechanical.
- **Stamina cost balance.** 3.0× is a starting point. Live play
  may show it's too brutal or too soft; tunable via config
  knob.

## Resolved during spec review

- **NightVision modifier.** Resolved as: NightVision observer
  in a dark room is treated for that observer's detection roll
  as if the room were lit — the sneak score uses the lit-room
  modifier (`0.9×` if sneaker not lit, `0.85×` if sneaker is
  lit). `CalcSneakScore` signature takes `effectiveLit bool`;
  the per-observer detection sites compute `effectiveLit =
  room.IsLit() || observer.HasNightVision`. AW-014 matrix row
  updated to reflect this; convenience helper
  `CalcSneakScoreVsObserver` provided.

- **`say`/`shout` and related noisy verbs.** Resolved as: all
  noisy communication and broadcast verbs (`say`, `shout`,
  `whisper` for the room-broadcast paths, `rally`, `warcry`,
  `taunt`) break stealth via explicit
  `Awareness.TransitionToRevealing(TriggerNoisyAction)` calls
  in their command handlers. Matrix rows AW-032 and AW-033
  cover this category. Some of these verbs also enter Combat
  Phase (`taunt` always; `warcry` against hostile targets);
  the cascade fires too but is idempotent against
  already-Revealing/Visible Awareness state, so double-firing
  is safe.

  `whisper` continues to NOT break stealth when used in its
  direct-target form (whispering a target by name is quiet by
  design). Only room-broadcast forms reveal.

## Roadmap impact

- Master spec
  `2026-05-13-combat-state-machines-design.md` references this
  as Chunk 1.
- On completion: chunk 1 marked Done in
  `COMBAT_STATE_ROADMAP.md`; chunk 2 (Life machine) brainstorm
  begins.
- Aliveness work stays paused per master spec.

## Resumption criteria

Awareness chunk is complete when:
1. All 31 Behavior Matrix tests pass.
2. All ~30 reader callsites migrated to `IsHidden()`.
3. All explicit AddBuff(9)/CancelBuffsWithFlag(Hidden) writers
   migrated to Awareness transitions.
4. `very_hidden` YAML deleted.
5. Buff #9 YAML has no duration.
6. `CalcSneakScore` is room-aware with the four-way conditional.
7. Stamina cost knob lands in config; movement stamina
   multiplier applied at runtime.
8. Light-change detection handler fires re-rolls.
9. Logout safety valve fires for hidden players.
10. Chunk 0 chunk-2.7 thief regression test still passes (no
    regression from Awareness migration).
11. Full server boot clean; full test suite green.
