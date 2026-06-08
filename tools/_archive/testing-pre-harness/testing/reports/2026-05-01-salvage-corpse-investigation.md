# Investigation: `Tova looks a little confused (salvage corpse)`

**Date:** 2026-05-01  
**Investigator:** Claude Code (static analysis, no code modified)

---

## TL;DR

`mob.Command("salvage corpse")` is called unconditionally on every foraging
tick inside `tickForagerForaging()` in
`internal/behaviortree/actions_forager.go` (line 212). No `salvage`
handler exists in `internal/mobcommands/` — the mob command registry covers
only player-parity verbs and does not include salvage. Every unrecognized mob
command falls through to the `emote looks a little confused (X)` fallback in
`world.go:1049`. The design intent (Stage 3.1 spec, line 183–186) was for
foragers to run "the existing Stage 3.0e salvage flow" — but that flow is
entirely implemented in user-side handlers (`usercommands/salvage.go` +
`hooks/NewRound_UserRoundTick.go`) and is unreachable from a mob.
**Recommendation: Option A** — implement a mob-side salvage-corpse handler
(or inline the corpse-salvage logic in the btree action) so the foraging tick
can actually strip hides from corpses.

---

## The Error Path

### Where `salvage corpse` is issued

**`internal/behaviortree/actions_forager.go`, line 212:**

```go
// Salvage any corpse in current room.
mob.Command("salvage corpse")
```

This sits inside `tickForagerForaging()`, which fires on every `mob_idle`
event while a forager is in the `StateForaging` state. There is no guard:
the call is unconditional regardless of whether a corpse is actually present.

### Where the dispatch fails

**`world.go`, lines 1040–1049** — the mob command dispatcher:

```go
handled, err = mobcommands.TryCommand(command, remains, mobInstanceId)
// ...
if !handled {
    if len(command) > 0 {
        mob.Command(fmt.Sprintf(`emote looks a little confused (%s %s).`, command, remains))
    }
}
```

`TryCommand("salvage", "corpse", instanceId)` returns `handled=false` because
`salvage` is not registered in the mob command registry. The fallback emote
fires, producing the player-visible message:

> `Tova looks a little confused (salvage corpse)`

The same `emote` fires every foraging tick (every ~1 round) for all three
foragers while they are in territory, so players in any forager's territory
see this message repeatedly.

---

## Existing Mob Salvage Handler?

**No.** `internal/mobcommands/` contains no `salvage.go` file and no file
references the string `salvage`. The full file list (confirmed via `ls`) has
no salvage entry. There is no alias or wrapper under another name.

The user-side salvage command lives at:

- `internal/usercommands/salvage.go` — initiates item salvage and corpse
  salvage via `CraftingState`
- `internal/hooks/NewRound_UserRoundTick.go`, lines 305–321 and 566–649 —
  resolves the multi-round activity and delivers materials to the player

Both paths depend on `users.UserRecord` and `CraftingState`, which are
player-only constructs. Mobs have no `CraftingState` equivalent, and
`mob.Command()` routes through the mob command registry exclusively —
it cannot reach user-side handlers.

---

## BTree Corpse Logic

### Tova's btree YAML

**`_datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml`:**

```yaml
tree:
  type: selector
  children:
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: attack
    - type: sequence
      event: mob_idle
      children:
        - type: action
          do: forager_step
```

The btree itself has no corpse-detection logic. All foraging behavior —
including the salvage call — lives in the `forager_step` Go action.

### The foraging state handler

**`internal/behaviortree/actions_forager.go`, lines 184–222 (`tickForagerForaging`):**

The function does the following each idle tick:

1. Increments the fatigue timer; transitions to `StateTravelingToDropoff` if
   fatigue ≥ 480 or carry ratio ≥ threshold.
2. Every `ForagerForageDwellRounds` ticks, calls `npcAttemptForage()` to
   harvest environmental items.
3. Calls `mob.Command("salvage corpse")` unconditionally (line 212).
4. Calls `npcWanderTerritoryNeighbor()` to move to a random adjacent room.
5. Returns `Failure` to let the legacy idle path (`lookfortrouble`) set aggro
   on prey.

There is **no corpse-presence check** before the salvage call. The same call
fires whether a corpse is in the room or not, meaning the confused-emote fires
on every single foraging tick even in empty rooms.

All three foragers (371-Tova, 372-Halix, 373-Kessa) use the same
`forager_step` action via their respective btrees at:

- `_datafiles/world/dogmud/behaviors/stillwater_marsh/371-tova.yaml`
- `_datafiles/world/dogmud/behaviors/ironwind_steppe/372-halix.yaml`
- `_datafiles/world/dogmud/behaviors/the_fernway_south/373-kessa.yaml`

All three will exhibit the same confused-emote bug.

---

## Skill Plumbing

Each forager mob has `salvage: 20` set in their mob YAML. Confirmed in:

- `_datafiles/world/dogmud/mobs/stillwater_marsh/371-tova.yaml`, line 53
- `_datafiles/world/dogmud/mobs/ironwind_steppe/372-halix.yaml`, line 55
- `_datafiles/world/dogmud/mobs/the_fernway_south/373-kessa.yaml`, line 55

However, **`mob.Character.Skills["salvage"]` (or
`mob.Character.GetSkillLevel(skills.Salvage)`) is never read anywhere in the
forager code path.** The skill value exists on the mob but is inert —
the `salvage corpse` command stub in `actions_forager.go` never reaches a
handler that would use it. The only code paths that read a character's salvage
skill are:

- `internal/hooks/NewRound_UserRoundTick.go` line 507 (`resolveSalvage` — user
  item salvage)
- `internal/hooks/NewRound_UserRoundTick.go` line 617 (`resolveCorpseSalvage`
  — user corpse salvage)

Both are user-only. The forager salvage skill is a half-wired stub: present
on the mob data, called from the btree, but never dispatched to a handler that
reads it.

---

## Design Intent

### Stage 3.0e spec (corpse-salvage design)

**`docs/superpowers/specs/completed/2026-04-28-corpse-salvage-design.md`**,
lines 183–186:

> "Players salvage corpses for cloth, leather, and sinew. Stage 3.1 (forager
> NPCs) and Stage 3.4 (real item transfer) will revisit whether vendor
> inventories should also stock these mats."

And lines 247–249:

> "Forager NPCs salvaging corpses for the player economy — Stage 3.1 decides
> whether foragers extend to corpse-salvage gathering"

Stage 3.0e explicitly deferred the NPC-salvage question to Stage 3.1.

### Stage 3.1 spec (forager NPCs design)

**`docs/superpowers/specs/2026-04-29-stage-3-1-foragers-design.md`**,
lines 7–9:

> "Add three forager NPCs — one per region — that gather raw materials in
> their home territory, **salvage corpses they encounter**, and feed the
> supply pipeline that 3.0b wired up."

Lines 183–187:

> "If a corpse is in the room (from wildlife killing wildlife, the forager's
> own kills, or anyone else's), forager runs the existing Stage 3.0e salvage
> flow, adding cloth/leather/sinew to inventory. Visible: *'Kessa kneels by
> the carcass and cuts strips of hide from it.'*"

Lines 134–137 (state machine diagram):

```
│  foraging              │  6-10 rounds, run shared
│  (until inventory      │  forage core; if corpse
│   carry-cap ≥ 75%      │  in room, salvage it
│   OR fatigue timer ≥   │
```

Stage 3.1 acceptance criteria (line 527):

> "3. Watch a forager engage prey (a marsh rat in the same room). Forager wins;
> corpse drops; forager salvages it; loot shows in their inventory."

**The design intent is unambiguous:** foragers should salvage corpses and add
cloth/leather/sinew to their satchel for delivery. The spec assumed "the
existing Stage 3.0e salvage flow" could be invoked from an NPC, but Stage
3.0e's flow is player-only.

---

## Recommendation: Option A — Wire Up Mob-Side Salvage

The call at line 212 of `actions_forager.go` is not scaffolding or accident —
it reflects a deliberate design feature from the Stage 3.1 spec that was
stubbed in but never completed. The mob salvage command needs to be built.

### What Option A requires

Rather than replicating the multi-round `CraftingState` machinery (which is
a player-side UX feature), the mob-side implementation should be a simpler
**single-tick direct action**:

1. **Add a mob command `salvage` in `internal/mobcommands/salvage.go`** that:
   - Checks whether any corpse exists in the current room with salvageable
     groups (`animal` or `humanoid` per `crafting.LookupCorpseSalvage`).
   - If no eligible corpse: return `handled=true` (no emote) with no effect.
   - If a corpse is found: roll the salvage yield using the mob's salvage
     skill level (`mob.Character.GetSkillLevel(skills.Salvage)`) via the
     existing `crafting.CalcSalvageChance()` + `crafting.RollSalvageReturnsFromSpec()`.
   - Add recovered items to `mob.Character.Items` via `mob.Character.StoreItem()`.
   - Remove the corpse from the room via `room.RemoveCorpse()`.
   - Emit a flavor message (per spec: *"Tova kneels by the carcass and cuts
     strips of hide from it."*) using `room.SendText()`.
   - Return `handled=true`.

2. **Add a corpse-presence guard in `tickForagerForaging()`** — only call
   `mob.Command("salvage corpse")` when `len(room.Corpses) > 0`. This
   eliminates the every-tick dispatch noise even before the handler lands.

### Why not Option B (silence the call)?

Removing the call would orphan the `salvage: 20` skill on all three forager
mobs, delete a named Stage 3.1 acceptance criterion, and leave the economy
pipeline short cloth/leather/sinew from the forager path. The design intent
is clear and the feature is partially built — it just needs the missing
mob-command handler.

### Why not Option C?

There is no alternative approach here. The code path, design intent, and data
wiring all converge on needing a mob-side salvage handler. The btree call is
correct; the handler is absent.

### Immediate mitigation (before Option A ships)

Add the corpse-presence guard to `tickForagerForaging()`:

```go
// Only attempt salvage when a corpse is actually present.
room := rooms.LoadRoom(ctx.RoomId)
if room != nil && len(room.Corpses) > 0 {
    mob.Command("salvage corpse")
}
```

This stops the every-tick confused-emote without requiring the full Option A
implementation. The command still produces a confused-emote when a corpse IS
present, but that's silent in rooms with no players — far less visible than
the current once-per-tick spam in any room.

---

## Summary of Files to Touch for Option A

| File | Change |
|------|--------|
| `internal/mobcommands/salvage.go` (new) | Implement mob salvage handler: check for eligible corpse, roll yield using mob's salvage skill, strip corpse, emit flavor message |
| `internal/behaviortree/actions_forager.go` line 211–212 | Add corpse-presence guard before `mob.Command("salvage corpse")` |
| `internal/mobcommands/mobcommands.go` | Register the new handler (pattern matches existing `RegisterCommand` calls) |
