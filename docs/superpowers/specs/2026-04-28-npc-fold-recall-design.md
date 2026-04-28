# NPC Fold-Recall — Design (Stage 3.0d of Caravan/Economy Effort)

**Date:** 2026-04-28
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Make `fold-anchor` and `fold-recall` work for NPCs the way they work
for players, so future forager NPCs (Stage 3.1) can use fold-recall
to travel back to their sell point quickly and to retreat when
injured. Wire Edrin (existing test hermit) for smoke verification and
opt the caravan crew in as wipe insurance.

## Multi-stage context

Stage 3.0d of the multi-stage caravan/economy effort. Earlier stages:
1 (NPC parties), 2 (caravan), 3.0b (mat region split), 3.0e (corpse
salvage) — all sit unmerged on the `development` branch. Per user
direction, nothing ships to prod (`master`) until the entire economy
stack lands.

Stage 3.0d is independent of 3.0a (west-of-Stillwater zone build),
3.0c (Fernway south expansion), 3.0e (corpse salvage), and 3.1
(forager NPCs). It pairs naturally with 3.1 — foragers are the
primary consumer of NPC fold-recall — but ships standalone.

**Pillar reminder:** mob/player parity when possible. The design
generalizes the existing player-only resolvers rather than building a
parallel mob-side path.

## Architecture

Three pieces, tight scope:

1. **Actor-shaped resolvers.** Refactor `resolveFoldAnchor`,
   `validateFoldRecall`, `resolveFoldRecall` to take
   `actions.Actor` instead of `*users.UserRecord`. Anchor lookup goes
   through `actor.GetCharacter().GetMiscData(...)`. Room broadcast
   uses `actor.SendRoomText(...)`. Self-feedback uses
   `actor.SendText(...)` (a no-op for mobs — fine, no one is
   listening). The teleport branches inside the resolver: players
   route through `rooms.MoveToRoom(userId, anchor)`; mobs route
   through `oldRoom.RemoveMob → newRoom.AddMob → mob.Character.RoomId
   = anchor`. One function, one source of truth for the spell
   semantics.

2. **YAML pre-stamped anchor.** Add an optional `fold_anchor_room:
   <roomId>` field to the Mob struct. On `NewMobByIdFresh` (mob
   spawn path), if the field is set, stamp it into
   `mob.Character.MiscData["fold-anchor-room"]`. After that, runtime
   is identical to a player who already cast `fold-anchor`. Mobs
   can also cast `fold-anchor` at runtime (free for parity), but the
   common case is YAML pre-stamp.

3. **Mob-cast wiring.** `resolveMobSpell`
   (`internal/hooks/spell_resolution.go:844`) gets two new cases for
   `fold-anchor` / `fold-recall`. Each wraps the casting mob in a
   `MobActor` and calls the shared resolver. The player path
   (`internal/hooks/spell_resolution.go:200-216`) gets the same
   treatment with a `UserActor` wrapper — same dispatcher entry
   points, same downstream resolver.

## Components & Files

| Action | File | Purpose |
|---|---|---|
| MODIFY | `internal/hooks/spell_foldanchor.go` | `resolveFoldAnchor(actor)` instead of `(user)` |
| MODIFY | `internal/hooks/spell_foldrecall.go` | `validateFoldRecall(actor)` + `resolveFoldRecall(actor)`. New helper `teleportActor(actor, toRoomId)` branches on `actor.IsPlayer()` |
| MODIFY | `internal/hooks/spell_resolution.go` | Player path wraps `user` in UserActor; `resolveMobSpell` adds the same two cases wrapping mob in MobActor |
| MODIFY | `internal/mobs/mobs.go` | Add `FoldAnchorRoom int \`yaml:"fold_anchor_room,omitempty"\`` to the Mob struct |
| MODIFY | `internal/mobs/mobs.go` (`NewMobByIdFresh` at ~line 331) | After `Character.MiscData` is initialized at spawn, if `FoldAnchorRoom > 0`, set `MiscData["fold-anchor-room"] = FoldAnchorRoom` |
| MODIFY | `internal/hooks/spell_foldanchor_test.go` | Adapt existing player tests to actor signature; add a mob-actor test |
| MODIFY | `internal/hooks/spell_foldrecall_test.go` | Adapt existing player tests; add mob-actor cases (anchor missing, anchor at current room, successful teleport) |
| MODIFY | `_datafiles/world/dogmud/mobs/marches_spur_road/275-old_edrin.yaml` | Add `fold_anchor_room: 4037` (the Cluttered Back Room, 1-west of Edrin's spawn at 4036). Add `spellbook: fold-recall: 30` and a tactics line `trigger: health_below:30 → action: cast fold-recall, priority: 13` (above the existing `flee` at 25 so recall fires first) |
| MODIFY | `_datafiles/world/dogmud/mobs/thornwall_city/357-ketil.yaml`, `358-marta.yaml`, `359-lars.yaml` | Each gets `fold_anchor_room: 465` (Market Square, Center — the Thornwall caravan depot), `spellbook: fold-recall: 20`, and a panic-recall tactics line at `health_below:30` priority above their existing flee/heal triggers |
| MODIFY | `docs/schemas/mob.md` | Document the new `fold_anchor_room` field |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0d dev-only entry |

## Data Flow

**Spawn → anchor pre-stamp:**

```
mob spawns (NewMobByIdFresh)
  → mob.Character.MiscData = make(map[string]any)
  → if mob.FoldAnchorRoom > 0:
       mob.Character.MiscData["fold-anchor-room"] = mob.FoldAnchorRoom
```

**Recall trigger → teleport (panic recall path):**

```
combat round tick
  → mobai.tactics evaluates triggers
  → health_below:30 matches → action="cast fold-recall"
  → tactics dispatcher emits "cast fold-recall" through the mob spell pipeline
  → resolveMobSpell sees spellId == "fold-recall"
    → wraps mob in MobActor
    → validateFoldRecall(actor) checks anchor exists + not current room
    → resolveFoldRecall(actor):
        actor.GetCharacter().EndAggro()
        SendRoomText("X folds through the Veil and vanishes!", excludeSelf=true)
        teleportActor(actor, anchorRoom):
          oldRoom.RemoveMob(instId)
          newRoom.AddMob(instId)
          mob.Character.RoomId = anchorRoom
        SendRoomText("X folds through the Veil and appears!", excludeSelf=true) on new room
```

## Edge cases

| Case | Behavior |
|---|---|
| Mob has `fold_anchor_room` pointing to a non-existent room | `validateFoldRecall` returns false (anchorRoom > 0 passes, but new-room load fails inside `teleportActor`). Cast fails silently for the mob. No log emission added in 3.0d — the resolvers don't currently use a logger; if observability proves necessary, add a `mudlog.Warn` line in a follow-up. |
| Mob's anchor IS the current room | Existing `validateFoldRecall` check returns false (silent for mob). No teleport. Recall doesn't burn this round; tactics retry on next tick (or fall through to lower-priority `flee`). |
| Anchor room is in an instanced zone with `allow_recall: false` | Existing `validateFoldRecall` already gates this on the *current* room. The anchor-room side isn't checked today (player parity). 3.0d preserves that — anchor-set side validation isn't added. |
| Mob is in a party (caravan), leader fold-recalls | Leader vanishes from current room. Followers still in current room — `party_follow_leader` is exit-event driven, teleport doesn't fire it. Followers stay put and continue tactics evaluation independently (each will recall on their own panic threshold). Acceptable — matches "individual recall" decision. |
| Mob fold-recalls while charmed by a player companion | EndAggro clears combat state. Charm buff persists. Mob teleports cross-zone; player loses sight of their companion. Mitigation: only mobs we wire `fold-recall` to in 3.0d are non-charmable (Edrin and caravan crew are `charm_immune: true`). Future foragers should be `charm_immune` too. |
| Fold-recall during a multi-round cast that gets interrupted | Existing cast machinery handles this — interruption clears `CraftingState` / `CastingState`. Spell never fires. |
| YAML field set, but anchor room is later removed from world data | Mob still spawns. MiscData stamped at spawn-time value. Cast fails at validate (room not loadable). Player-parity behavior. |

## Out of scope (explicitly)

- **Forager NPC wiring.** That's Stage 3.1.
- **Logistic recall trigger** (e.g., `inventory_full → cast
  fold-recall`). That's Stage 3.1's responsibility to define and wire.
- **Group-aware recall.** Caravan leader recalling does NOT pull
  followers along — followers each have their own anchor and their
  own panic trigger. Simpler, matches player parity (party members
  recall individually too).
- **Cooldowns.** Recall is gated by spell components / casting cost
  like every other mob spell. No special cooldown layer added in 3.0d.
- **Player-visible anchor manipulation.** Players see the shimmer text
  on cast (because `SendRoomText` fires), but they can't interact
  with another actor's anchor.
- **Anchor-side validation** (e.g., refusing to set anchor in a
  combat zone or instanced room). Not done for players today; not
  added here either.

## Testing strategy

**Phase 1 — unit tests:**
- `resolveFoldAnchor` and `resolveFoldRecall` with both UserActor and
  MobActor inputs.
- Anchor stamping (YAML pre-stamp + runtime cast both produce the
  same MiscData state).
- Validation gates (no anchor, anchor at current room, instanced
  zone block).
- Teleport mutation of room state (mob removed from origin, added to
  destination, RoomId updated).

**Phase 2 — boot test:**
- `go build ./...` clean.
- Server boots without panic.
- Mobs loadedCount unchanged (modifying existing YAMLs).

**Phase 3 — in-game smoke test:**
1. Walk to Edrin's room. `look` confirms him present.
2. Attack Edrin. Drop his HP below 30%.
3. Watch the room broadcast: "Old Edrin folds through the Veil and
   vanishes!"
4. `look` confirms Edrin is no longer in the room.
5. Walk 1-west to the back room (his pre-stamped anchor).
6. `look` confirms Edrin is now there. He may try to heal / re-buff
   per his existing tactics — that's fine, just confirms he's alive
   and post-recall.
7. (Optional) attack the caravan at the bandit camp; if their HP
   drops low enough, verify they recall to the Thornwall depot.

## Implementation order (preview for the plan stage)

Approximate ordering. Plan task 0 will refine.

1. Add `FoldAnchorRoom` field to Mob struct + YAML schema doc (small)
2. Stamp MiscData at `NewMobByIdFresh` (small) + tests
3. Refactor `resolveFoldAnchor` to actor signature + adapt player tests + add mob test (small-medium)
4. Refactor `validateFoldRecall` + `resolveFoldRecall` + new `teleportActor` helper + tests (medium — branching teleport path)
5. Wire mob-cast dispatcher in `resolveMobSpell` for both spells (small)
6. Wire Edrin YAML (small) + caravan crew YAMLs (small)
7. PATCH_NOTES + verification + smoke test (small + manual)

~7 tasks total. Comparable to 3.0e in size.
