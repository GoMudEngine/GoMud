# Mob Aliveness 3.3 — Sleeping & Wake States (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.3 (Phase 3 — Routine layer)
**Size:** M (expanded from roadmap's "S" — see "Scope notes" below)
**Branch:** `feature/mob-aliveness-3.3-sleeping-wake-states`
**Depends on:** 3.1 (game-time hook), 3.2 (NPC schedules)

## Goal

Sleeping becomes a real bidirectional mechanic, not just NPC flavor.
NPCs visibly asleep during their authored sleep segments. Players
can sleep anywhere they choose. Sleep is queryable engine state
(`HasBuffFlag(buffs.Sleeping)`) so theft, look-rendering, combat,
and wake-triggers all hook coherently.

Sleeping carries real mechanical weight:

- **Aggressive recovery:** 5× HP/SP/CP regen while asleep.
- **Real vulnerability:** the entire first round of attacks against
  a sleeper auto-crit, then the buff cancels.
- **Multiple wake triggers:** damage, failed steal, shout in room,
  light entering room, the `stand` command, schedule segment end.

## Scope notes

The roadmap classified 3.3 as size **S** with a narrow NPC-only
focus: "NPCs visibly asleep at night; wakeable by sound, light,
attack." Brainstorming expanded the scope per user direction:

- Added **player parity**: `sleep` user command + `stand` extension
  for waking, via `actions.Sleep(actor)` actor-function wrapper
  (PvM/MvP parity discipline per `feedback_target_resolution_uses_actor`).
- Added **regen boost** (5× while sleeping) — turns sleep into a
  meaningful resource-recovery mechanic, not just flavor.
- Added **first-hit-crit assassination payoff** — turns
  combat-on-sleeper into a high-stakes tactical opportunity.

These additions move sizing to **M**. The roadmap line about
"more crime severity" for combat-on-sleeper has been explicitly
dropped — see "Out of scope" — because the consumer (chunk 5.1
Town Justice) doesn't exist yet. YAGNI cut.

## In scope

1. New `buffs.Sleeping Flag = "sleeping"` constant.
2. Update `_datafiles/world/default/buffs/15-sleeping.yaml` to
   include the `sleeping` flag, the new `cancel-on-damage` flag,
   and an effectively-infinite `triggercount` (governed by cancel
   flags, not buff-tick expiration).
3. New buff flag: `cancel-on-damage`. Cancels the buff when any
   damage is applied to the bearer. Wired in the damage pipeline.
4. `actions.Sleep(actor, options) Result` — actor-function wrapper
   in `internal/actions/sleep.go`.
5. New user command: `sleep` (in `internal/usercommands/sleep.go`)
   calling `actions.Sleep(userActor, …)`.
6. New mob command: `sleep` (in `internal/mobcommands/sleep.go`)
   calling `actions.Sleep(mobActor, …)`. Used by the schedule
   executor via `mob.Command("sleep")`.
7. Extension of the existing `stand` command (player + mob) to
   also cancel `Sleeping` if present.
8. Schedule executor extensions:
   - On entry into an `activity: sleeping` segment, issue
     `mob.Command("sleep")` once the mob is at the segment's
     `target_room`.
   - On exit from an `activity: sleeping` segment, cancel the
     Sleeping buff directly via `CancelBuffsWithFlag`.
   - Grace cooldown: after a forced wake during a sleep segment,
     suppress re-sleep for `ScheduleWakeGraceRounds` rounds
     (default 50).
9. Room rendering: mobs and players with the Sleeping flag render
   as "X is sleeping here." (mob) or "X is asleep." (player from
   another player's POV) instead of the standard "X is here."
   Schedule idle-commands continue firing on top.
10. Wake triggers — four hooks, one each:
    - **Damage:** new `cancel-on-damage` flag.
    - **Failed steal:** in steal action's failure path, cancel
      Sleeping on the victim.
    - **Shout in room:** shout dispatcher cancels Sleeping on all
      same-room listeners.
    - **Light source on room entry:** `go.go` post-arrival hook
      cancels Sleeping on room sleepers when the arriving actor
      carries a lit item or has Illumination.
11. Regen boost: `SleepRegenMultiplier` (default 5.0) applied to
    HP/SP/CP per-round percentage regen when bearer has Sleeping.
12. First-hit-crit: round dispatcher snapshots sleeping victims
    at start-of-round. Damage pipeline gains a `forceCrit bool`
    parameter. All damage events in the snapshot round against
    sleeping victims auto-crit. The `cancel-on-damage` flag
    cancels Sleeping mid-round; the snapshot ensures all
    attackers in the same round still benefit.
13. Schedule loader: `activity: sleeping` added to the recognized
    activity vocabulary (alongside `""` and `craft`).
14. Pilot retrofit: three Thornwall schedule YAMLs
    (`thornwall_smith`, `thornwall_tavern_keeper`,
    `thornwall_temple_priest`) — sleep segments gain
    `activity: sleeping`.
15. Documentation: `schedule.md` schema update, `context.md`
    updates, `CLAUDE.md` sleep summary, new `sleep.template`
    helpfile, `stand.template` append.

## Out of scope

- **Crime severity uplift for sleeper-victim attacks.** The
  roadmap mentioned this; the consumer (chunk 5.1 Town Justice)
  doesn't exist yet. Adding `was_victim_sleeping bool` to the
  Crime struct now would be speculative. When 5.1 lands, this
  can be added then — the data needed will all be queryable at
  crime-record time (`victim.Character.HasBuffFlag(Sleeping)`).
- **Generic `combat.ShouldForceCrit(...)` helper.** The damage
  pipeline gains a `forceCrit bool` parameter (the abstraction
  surface). Whether the caller sets it from sleeper-state,
  surprise-state, backstab-state, etc., is the caller's concern.
  The round-dispatcher snapshot is one line per source; not
  worth building a helper before a second concrete caller exists.
  Inline comment marks the snapshot site for future authors.
- **Adjacent-room sound propagation.** Combat or shout in a
  neighbouring room does NOT wake sleepers. Same-room only.
- **Explicit `wake <mob>` / `shake <mob>` verbs.** Use `stand`
  instead. (Acceptable to add later as aliases if content needs
  them.)
- **Wild creatures sleeping.** Schedule-driven only. Wolves,
  goblins, etc. don't sleep — that's downstream content work,
  not engine-side.
- **Charmed mob sleeping while controller is awake.** Charmed
  mobs follow their controller; no schedule semantics apply.
- **Bed/inn safe-room gate.** Player can `sleep` anywhere. The
  first-hit-crit risk is the only natural disincentive.
- **Sleep duration ceiling for players.** Players sleep
  indefinitely (until any action / damage / stand / etc.). No
  fatigue or "you can't sleep again for N minutes" gate.
- **NPCs napping (off-schedule daytime sleep).** Only segments
  marked `activity: sleeping` apply the buff. A `chamber`
  segment with "resting" idlecommands but no sleeping activity
  is just rest, not sleep.

## Architecture

### Data model

**`internal/buffs/buffspec.go`** — new state-query flag constant:
```go
Sleeping Flag = "sleeping"
```
Placed alongside `Hidden`, `NightVision`, etc.

**`internal/buffs/buffspec.go`** — new cancel-flag constant (if
the package uses constants for cancel flags too; otherwise
treated as a plain string match):
```go
CancelOnDamage Flag = "cancel-on-damage"
```

**`_datafiles/world/default/buffs/15-sleeping.yaml`** updated:

```yaml
buffid: 15
name: Sleeping
description: You are getting much needed rest.
triggerrate: 1 round
triggercount: 100000   # effectively infinite; cancel-on-* drives duration
flags:
  - sleeping           # NEW — state-query flag
  - cancel-on-action
  - cancel-on-combat
  - cancel-on-damage   # NEW — wakes on any damage event
```

### `actions.Sleep` actor function

New file `internal/actions/sleep.go`:

```go
package actions

// SleepOptions is reserved for future authoring knobs (bed-item
// bonus, custom emote prose, etc.). Empty for chunk 3.3.
type SleepOptions struct{}

// Sleep applies the Sleeping buff to actor's character. Used by
// the player sleep command, the mob sleep command, and the
// schedule executor (via mob.Command("sleep")).
//
// Fails (returns Failure with a user-facing reason) when the
// actor is in combat / has Aggro. Idempotent: if already
// sleeping, no-op success without re-applying the buff or
// re-emitting the room emote.
func Sleep(actor Actor, opts SleepOptions) Result
```

Behavior:
1. If `actor.Character().IsInCombat()` or `Aggro != nil` →
   return Failure. User actors receive `"You can't sleep right
   now."` via `actor.SendText(messaging.CategorySystem, ...)`.
   Mob actors get no player-visible message — the schedule
   executor's idempotent re-issue handles retry on the next tick
   once combat ends.
2. If `actor.Character().HasBuffFlag(buffs.Sleeping)` already →
   return Success without re-applying (idempotent).
3. Apply buff 15. Existing buff-apply path emits the per-buff
   user message ("You are getting much needed rest.").
4. Emit room visual message: `"X lies down to sleep."` to other
   occupants.
5. Return Success.

### User and mob commands

**`internal/usercommands/sleep.go`** — tiny dispatcher:

```go
func Sleep(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    userActor := actions.NewUserActor(user, room)
    actions.Sleep(userActor, actions.SleepOptions{})
    return true, nil
}
```

**`internal/mobcommands/sleep.go`** — symmetric for mobs.

Both registered in their respective command-registry init blocks.

### `stand` extension

Find the existing `stand` handler in `internal/usercommands/` and
`internal/mobcommands/`. At the top of each, after argument
parsing but before the position state machine work:

```go
if c.HasBuffFlag(buffs.Sleeping) {
    c.CancelBuffsWithFlag(buffs.Sleeping)
    mobs.OnSleeperWoken(actor)  // stamps schedule_wake_round for scheduled mobs
    // existing room message about standing covers wake narration
}
```

The existing "you stand up" room message naturally narrates
waking. No new emote needed. `OnSleeperWoken` ensures the schedule
grace cooldown applies — without it, a scheduled NPC standing up
during a sleep segment would immediately re-sleep on the next
tick.

### Schedule executor integration

Modify `internal/hooks/NewRound_IdleMobs_schedule.go`. Add two
new fields to `schedulePlan`:

```go
type schedulePlan struct {
    // ... existing fields ...
    WantsSleep bool   // current segment is activity: sleeping AND mob is at target
    WantsWake  bool   // transitioning OUT of an activity: sleeping segment
}
```

`scheduleTickPlan` populates them:

```go
// At segment-transition detection:
if seg.Start != lastSegStart {
    plan.SegmentChanged = true
    plan.NewSegmentStart = seg.Start
    plan.NewIdleCommands = seg.IdleCommands

    // Look up the prior segment via lastSegStart. If it had
    // activity: sleeping, we're transitioning OUT of sleep.
    if prior := s.SegmentByStart(lastSegStart); prior != nil &&
        prior.Activity == "sleeping" {
        plan.WantsWake = true
    }
}

// Per-tick sleep check (idempotent — see grace cooldown below):
if seg.Activity == "sleeping" &&
    mob.Character.RoomId == seg.TargetRoom &&
    !mob.Character.HasBuffFlag(buffs.Sleeping) {
    // Grace cooldown: don't re-sleep if recently woken.
    lastWoken := getMiscDataInt(&mob.Character, "schedule_wake_round", 0)
    grace := int(configs.GetBalanceConfig().ScheduleWakeGraceRounds)
    if lastWoken == 0 || int(util.GetRoundCount())-lastWoken >= grace {
        plan.WantsSleep = true
    }
}
```

`applySchedulePlan` execution order (matters):

```go
// 1. Wake first — clear stale sleep before any new path/sleep.
if plan.WantsWake {
    mob.Character.CancelBuffsWithFlag(buffs.Sleeping)
}

// 2. Existing pathing logic (clear stale path on transition,
//    queue new pathto, fallback to home after retries).

// 3. Sleep last — only if at target AND not on grace cooldown.
if plan.WantsSleep {
    mob.Command("sleep")
}
```

The grace cooldown is recorded by a separate wake-event hook
(NOT in the schedule executor itself): when any wake-trigger
fires on a scheduled mob during a sleeping segment, set
`mob.Character.SetMiscData("schedule_wake_round",
int(util.GetRoundCount()))`. The next `WantsSleep` check
compares the elapsed round count to `ScheduleWakeGraceRounds`.

Implementation note: the wake-event hook is the most awkward
piece. Cleanest is a small `OnSleeperWoken(mob)` helper in
`internal/mobs/` that callers fire after cancelling the
Sleeping buff for a wake-trigger reason. Callers: the four
wake-trigger sites (damage pipeline, failed-steal, shout
dispatcher, light-room-entry) plus the `stand` extension.

### Room rendering

The existing room template (`_datafiles/world/dogmud/templates/descriptions/who.template`)
emits a comma-separated "Also here:" list, not per-occupant
sentences. Occupant names are built in `internal/rooms/roomdetails.go`
(`VisiblePlayers` and `VisibleMobs` slices). The closest precedent
is the `(AFK)` suffix appended to player names in the same loop.

Sleeping rendering follows the same pattern — append an
`<ansi fg="8">(asleep)</ansi>` suffix to the name string when
the occupant has `buffs.Sleeping`:

```go
// In the VisiblePlayers loop, after AFK suffix:
if player.Character != nil && player.Character.HasBuffFlag(buffs.Sleeping) {
    playerEntry += ` <ansi fg="8">(asleep)</ansi>`
}

// In the VisibleMobs loop, after the mob name is built:
if mob.Character.HasBuffFlag(buffs.Sleeping) {
    nameStr += ` <ansi fg="8">(asleep)</ansi>`
}
```

Schedule idle-commands (e.g., "emote turns over with a soft snore.")
continue firing as usual — they add per-tick texture on top of the
static `(asleep)` marker that a player sees in the "Also here:" list.

### Wake-trigger hooks

| Trigger | Hook location (best-known) | Logic |
|---|---|---|
| Damage to bearer | `internal/combat/damage_pipeline.go` after damage is applied | `c.CancelBuffsWithFlag(buffs.CancelOnDamage)`; if Sleeping was in that set, fire `OnSleeperWoken(victim)` |
| Failed steal | `internal/actions/steal.go` failure branch | If `victim.HasBuffFlag(Sleeping)`, cancel and `OnSleeperWoken(victim)` |
| Shout in room | `internal/usercommands/shout.go` + mob equivalent | After broadcasting, iterate room occupants; for each with Sleeping, cancel + OnSleeperWoken |
| Light source on room entry | `internal/usercommands/go.go` post-arrival | If arriving actor has a lit item or `buffs.Illumination` flag, iterate room sleepers, cancel + OnSleeperWoken |

`OnSleeperWoken(mob)` is the central helper. For mobs, it stamps
`schedule_wake_round` MiscData. For players, it's a no-op (no
schedule grace concept).

### Combat: regen boost and first-hit-crit

**Regen boost.** New config knob `SleepRegenMultiplier ConfigFloat`
(default `5.0`) in `internal/configs/config.balance.go`. In the
per-round regen tick (`HealthPerRound` / `StaminaPerRound` /
`ConvictionPerRound`):

```go
base := floor(poolMax * pct)  // existing per-pool percentage
if base < 1 { base = 1 }      // existing floor
if c.HasBuffFlag(buffs.Sleeping) {
    return int(float64(base) * configs.GetBalanceConfig().SleepRegenMultiplier)
}
return base
```

**First-hit-crit (entire-round semantic).** In the round dispatcher
(where attacks resolve per-actor per-round — exact file TBD,
likely in `internal/combat/` or `internal/hooks/NewRound_DoCombat.go`):

```go
// Snapshot sleeping victims at the start of round resolution so
// the cancel-on-damage flag that fires mid-round doesn't blunt
// later attackers' crit payoff. Other future first-hit-crit
// triggers (surprise attack, backstab, etc.) can add parallel
// snapshot checks at this same site.
sleepingVictims := map[int]bool{}
for _, victimId := range combatVictimIdsThisRound {
    if v := resolveActor(victimId); v != nil &&
        v.Character().HasBuffFlag(buffs.Sleeping) {
        sleepingVictims[victimId] = true
    }
}

// Each damage event receives forceCrit:
forceCrit := sleepingVictims[victimId]
dmg := combat.CalcDamageWithForce(..., forceCrit)
```

The damage pipeline (`CalcRawDamage` → variance roll → mitigation)
gains a `forceCrit bool` parameter. When true, the Z-score check
is bypassed and the result is always a crit (ZScore set to e.g.
`combat.CritZScoreThreshold + 0.5` so all existing crit-flavor
branches fire normally).

After the round resolves, the `cancel-on-damage` flag has already
cancelled Sleeping on all damaged victims. Second round: no
forced crits, normal combat.

### Schedule loader validator

`validateScheduleStandalone` (or the activity warning loop in
the schedule loader) recognizes `sleeping` as a known activity:

```go
if seg.Activity != "" &&
    seg.Activity != "craft" &&
    seg.Activity != "sleeping" {
    mudlog.Warn("schedule", "id", s.Id, "segment", i,
        "msg", "unknown activity value", "value", seg.Activity)
}
```

### Configuration knobs

Added to `internal/configs/config.balance.go`:

| Knob | Default | Purpose |
|---|---|---|
| `SleepRegenMultiplier` | 5.0 | HP/SP/CP regen multiplier when bearer has Sleeping flag |
| `ScheduleWakeGraceRounds` | 50 | After a forced wake during a sleep segment, suppress re-sleep for N rounds (≈200 sec real-time at default tick rate) |

Defaults applied in `validateMobs` (or equivalent) alongside
`ScheduleMaxPathRetries`.

### Pilot retrofit

Three schedule YAMLs gain `activity: sleeping` on their sleep
segments:

| File | Segment | Change |
|---|---|---|
| `thornwall_smith.yaml` | 22-6 (loft) | `activity: ""` → `activity: sleeping` |
| `thornwall_tavern_keeper.yaml` | 22-6 (quarters) | `activity: ""` → `activity: sleeping` |
| `thornwall_temple_priest.yaml` | 22-4 (chamber) | `activity: ""` → `activity: sleeping` |

Olen's 4-6 (rising) and 10-12 (rest) segments stay non-sleeping
intentionally — those are waking states with sleepy flavor.

## Edge cases & failure modes

| Situation | Behaviour |
|---|---|
| Player sleeps in combat | `actions.Sleep` returns Failure with user-facing reason |
| Mob sleep tick fires while mob is in combat | `actions.Sleep` returns Failure silently. Schedule executor retries on next idle tick after combat ends. |
| Mob fails to reach sleep-segment target_room before segment ends | Schedule executor never fires `WantsSleep` (gated on at-target). Next segment transitions normally. No buff to clean up. |
| Wake-trigger fires when mob isn't asleep | `CancelBuffsWithFlag` is a no-op. `OnSleeperWoken` still stamps the wake round; harmless. |
| Player /sleeps, takes damage from a pre-existing DoT | `cancel-on-damage` fires on the DoT tick. Sleep ends. First-hit-crit may or may not apply depending on whether the DoT damage event hits the snapshot path — likely no, since DoT damage typically resolves outside combat-round dispatch. Acceptable. |
| Mob sleeps, server restarts mid-segment | Spawn-override places mob at segment target (existing 3.2 behavior). Buff is transient and re-applies on the next idle tick (`WantsSleep` check is idempotent). |
| Two attackers in the same round, second attacker arrives via summon AFTER snapshot | Second attacker doesn't crit (was not in the snapshot). Acceptable; the snapshot is a per-round-start photograph. |
| Multi-pool damage in one event (e.g. AoE) | The forceCrit parameter applies to whatever damage event the call processes. Each damage event is one call. |
| Sleeper is healed mid-sleep | Heals don't trigger cancel-on-damage. Buff persists. Regen 5× continues to compound with heal. Acceptable (and arguably desirable). |
| Mob's sleep segment uses target_room that the mob can't reach (e.g. zone refactor broke pathing) | Existing schedule pathto-retry-then-fallback-to-home logic applies. Mob never reaches sleep segment target; `WantsSleep` never fires; no buff. |
| Grace cooldown is too short for a scenario (50 rounds isn't enough) | Config knob `ScheduleWakeGraceRounds` lets ops re-tune. |
| Schedule executor and a wake-trigger race on the same tick | All MiscData reads/writes happen serially in IdleMobs loop. Wake-trigger sites also serial within their dispatchers. No real race; the only ambiguity is "did the wake fire before or after this tick's WantsSleep check," and the grace-cooldown handles both orderings correctly. |

## Validation summary

**Load time (panic):** unchanged from 3.2. No new schedule-side
validation needed; `activity: sleeping` just gets recognized
instead of warning.

**Load time (warn-only, dedup'd):**
- Existing: `activity:` value not in known set (now `{"", "craft",
  "sleeping"}`)
- Existing: `activity: craft` on non-crafter mob

**Runtime:** unchanged from 3.2.

## Testing

### Unit tests

| File | Coverage |
|---|---|
| `internal/actions/sleep_test.go` | `actions.Sleep` blocks in combat (Failure); applies buff 15 on success; idempotent re-sleep is no-op; emits expected room message |
| `internal/hooks/NewRound_IdleMobs_schedule_test.go` (additions) | `WantsSleep` fires only when at target_room + segment activity is sleeping + not within grace; `WantsWake` fires on transition OUT of a sleeping segment; grace cooldown blocks `WantsSleep` for `ScheduleWakeGraceRounds` after a wake event |
| `internal/combat/damage_pipeline_test.go` (additions) | `forceCrit: true` parameter bypasses Z-score check, produces crit damage; `forceCrit: false` is unchanged; crit narration fires for forced crits |
| `internal/characters/resources_test.go` (or wherever regen tests live) | HP/SP/CP per-round regen multiplied by `SleepRegenMultiplier` when bearer has Sleeping flag; multiplier=1.0 when flag absent |
| `internal/buffs/buffs_test.go` (additions) | `cancel-on-damage` flag triggers cancellation when damage applied; flag absent → no cancel |
| `internal/usercommands/stand_test.go` (additions) | `stand` cancels Sleeping if bearer has it; otherwise existing behavior preserved |
| `internal/mobs/schedule_loader_test.go` (additions) | Schedule with `activity: sleeping` loads without warning; unknown activity values still warn |

### Manual smoke pass

1. `sleep` in any room → buff applies, room emits "You lie down
   to sleep."
2. Wait 10 rounds → confirm HP/SP/CP regen ~5× normal (compare
   via status display).
3. Take any action (move, attack, look) → buff cancels (existing
   `cancel-on-action`).
4. `sleep` again, admin-spawn a hostile mob, `attack` it → first
   round damage is a crit, buff cancels; subsequent rounds normal.
5. `time set 23` and look at Kerra/Marek/Olen's home rooms →
   confirm "X is sleeping here." rendering.
6. Successfully pickpocket Kerra at 23 — sleep survives.
7. Fail a pickpocket on Kerra at 23 — sleep cancels.
8. Walk into Kerra's loft carrying a lit torch at 23 — Kerra wakes.
9. `shout something` in Marek's quarters at 23 — Marek wakes.
10. Wait `ScheduleWakeGraceRounds` rounds (≈200 sec) — Marek
    re-sleeps automatically.
11. `stand` while sleeping → buff cancels, "X stands up" message.

### Autonomous smoketester (deferred to T13-style task in plan)

A scenario goal file at
`tools/testing/goals/3.3-sleep-mechanics.yaml` exercises:
- Player sleep + regen rate check
- Player sleep + attack against the tester → first-round crit
- Watch Kerra's sleep segment cycle in/out
- Steal-fail wake → grace cooldown → re-sleep

Roughly 30-40 commands; sub-15-minute session.

## Documentation

| File | Change |
|---|---|
| `docs/schemas/schedule.md` | Add `sleeping` to recognized activity values, describe buff-apply + wake-grace + regen-boost effects |
| `internal/buffs/context.md` | New `Sleeping` Flag row; new `cancel-on-damage` flag in cancel-flag section |
| `internal/actions/context.md` | New `Sleep(actor, opts)` row in the actor-function table |
| `internal/combat/context.md` | First-hit-crit on sleepers documented in crit-resolution section; note the snapshot-at-round-start pattern |
| `internal/configs/context.md` | New rows: `SleepRegenMultiplier`, `ScheduleWakeGraceRounds` |
| `CLAUDE.md` | Append a "Sleep Mechanics" subsection: sleeper = first-round crits, 5× regen, wake triggers (damage / fail-steal / shout / light / stand), schedule integration via `activity: sleeping` |
| `_datafiles/world/dogmud/templates/help/sleep.template` (new) | Player helpfile: how to sleep, what wakes you, recovery rate, vulnerability warning |
| `_datafiles/world/default/templates/help/stand.template` or dogmud override | Append: "If you are sleeping, `stand` will wake you up." |

## Commit shape

Suggested split (each commit independently reviewable):

1. `feat(buffs): Sleeping state-query flag + cancel-on-damage flag`
2. `feat(actions): Sleep actor function + sleep user/mob commands + stand extension`
3. `feat(hooks): schedule executor recognizes activity: sleeping (with grace cooldown)`
4. `feat(combat): first-hit-crit on sleepers + SleepRegenMultiplier`
5. `feat(usercommands): wake triggers (failed steal, shout, light on room entry)`
6. `feat(content): Thornwall pilot — sleep segments mark activity: sleeping`
7. `docs: schedule/buff/combat context.md updates + CLAUDE.md + helpfiles`
8. `chore(roadmap): mark 3.3 Done`

`PATCH_NOTES.md` updated at push time per pre-push SOP.

## Roadmap closeout

`MOB_ALIVENESS_ROADMAP.md`:
- Flip chunk 3.3 status to **Done**.
- Add "Shipped:" describing the M-sized expansion (player parity,
  regen boost, first-hit-crit).
- Note that 5.1 Town Justice may want to consume sleeper-victim
  state from the live buff query (no Crime-schema change needed).

## Open questions

None — design fully scoped during brainstorming.

## References

- Roadmap: `MOB_ALIVENESS_ROADMAP.md` chunk 3.3
- Dependency: chunk 3.1 (game-time hook) and 3.2 (NPC schedules,
  see `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`)
- Existing buff: `_datafiles/world/default/buffs/15-sleeping.yaml`
- Buff flag system: `internal/buffs/buffspec.go`,
  `internal/characters/buffs.go`
- Damage pipeline: `internal/combat/damage_pipeline.go`,
  `internal/combat/damage_pipeline.md` (if exists)
- Regen system: `internal/characters/resources.go`
- Schedule executor: `internal/hooks/NewRound_IdleMobs_schedule.go`
- Actor function pattern: see `internal/actions/steal.go`,
  `internal/actions/forage.go` for precedents
- Pre-push SOP: `CLAUDE.md`
