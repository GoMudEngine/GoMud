# Trail-Quest Per-Step `map_target` Pass — Design

**Date:** 2026-07-19
**Status:** Approved (design), pending spec review
**Relates to:** `docs/superpowers/specs/2026-07-18-minimap-quest-marker-design.md`
(the minimap quest-marker feature this pass feeds)

## Problem

The web minimap quest-marker feature (shipped 2026-07-18) marks the **focused**
quest's destination room on the leather map and draws a next-step arrow toward
it. The destination is resolved **per current step** by
`questengine.(*Engine).ResolveQuestTarget(questId, step)`, which returns:

1. the step's explicit `map_target: <roomid>`, else
2. a room inferred from a `room_enter` trigger gated on the current step token,
   else
3. `0` — no marker.

The seven Pothole Coulee newbie **trail quests** (32 First Blood, 35 First Heat,
38 First Brew, 41 First Sign, 44 First Casting, 47 First Words, 50 First Shot)
are handed to the player all at once when they finish Find Your Footing (quest
31). They advance via `command` / `mob_death` / `item_give` triggers — **none
use `room_enter`** — so path 2 never fires. Today only First Blood carries a
single `map_target` (on its `start` step). Consequently:

- The other six trails show **no marker at all** for the focused quest.
- First Blood loses its marker the moment the player advances past `start`,
  even though later steps still expect the player at a specific room.

This is the "trail quests still need per-step `map_target`s" item called out as
an open concern in the minimap-marker work and the newbie-onboarding backlog
(item #11 follow-up). The maze-like coulee layout is exactly where newcomers
most need the marker.

## Goal

Author a `map_target` on **every non-terminal step** of all seven trail quests
so the focused quest's marker is always lit and always points at the room where
the player makes progress. Content-only change (one YAML line per step); no
engine code.

## Non-Goals

- No engine/`ResolveQuestTarget` changes — the resolver already supports
  per-step `map_target`.
- No cross-zone boundary-exit arrow, no full-route overlay (both deferred in the
  minimap spec; all seven trails are single-zone within Pothole Coulee).
- No changes to quest logic, triggers, rewards, prose, or step order.
- Non-trail newbie quests (29 Two Roads, 31 Find Your Footing) already carry
  targets and are out of scope.

## Authoring Rules

1. **Objective step** (player must *do* something at a specific room) → target
   that room.
2. **Turn-in or location-agnostic step** (report to an NPC; or a `drink` /
   `help` / `reload` that works anywhere) → target the trailhead NPC's room, so
   the marker guides the player back for the hand-in.
3. **`end` step** → no `map_target`. It is the terminal/complete state; the
   quest leaves the active panel on completion, so a marker would be moot.

## Per-Quest Step Targets

NPC spawn rooms (confirmed from room `spawninfo`):
Vorn 9108→5227, Rusk 9116→5245, Birna 9128→5265, Tarn 9136→5283,
Grieve 9144→5303, Wenna 9153→5324, Iden 9161→5350.

| Quest | Step | `map_target` | Room / rationale |
|---|---|---|---|
| **32 First Blood** | start *(already set)* | 5227 | Drill Yard |
| | strike, special, consider, verbosity | 5227 | Vorn + dummy are all in 5227; keeps marker lit on the work room through the turn-in (`ask vorn report`) |
| **35 First Heat** | start | 5245 | Forge — buy at Rusk's stall + `craft iron dagger` |
| | craft | 5245 | Rusk turn-in (`ask rusk done`) |
| **38 First Brew** | start | 5265 | Birna's pool — buy + `craft healing salve` at the bench |
| | brew | 5265 | next is `drink` (anywhere) then report → point at Birna |
| | drink | 5265 | Birna turn-in (`ask birna done`) |
| **41 First Sign** | start | 5284 | Open Steppe — `track hare` + `forage` |
| | track | 5284 | forage objective is also on the steppe; Tarn (5283) is adjacent |
| | forage | 5284 | track objective is also on the steppe; Tarn (5283) is adjacent |
| **44 First Casting** | start | 5305 | Star Chamber — `cast conviction-spike mote` |
| | cast | 5303 | Observatory Hall — return to Grieve (`ask grieve done`) |
| **47 First Words** | start | 5324 | Wenna's farmstead — `ask wenna` + `help` (anywhere) |
| | help | 5324 | Wenna turn-in (`ask wenna done`) |
| **50 First Shot** | start | 5351 | Long Terrace — after ammo/`equip`/`reload`, `shoot butt` down-range |
| | reload | 5351 | still the Long Terrace shoot objective |
| | shoot | 5350 | return west to Iden (`ask iden done`) |

### Design note — the First Sign asymmetry

Casting and Shot give their post-objective step a **distinct return-to-NPC**
target because those trails genuinely send the player to a different room to
hand in (Grieve's hall below the Star Chamber; Iden's range west of the Long
Terrace). First Sign does **not**: its two objectives (`track`, `forage`) are
parallel — both gate only on `41-start`, both act on the Open Steppe (5284; sage
flat 5285 also forageable), and Tarn stands one room away at 5283. Bouncing the
marker to Tarn while an objective may still be pending on the steppe would
mislead. So all three of First Sign's steps mark the steppe work-area (5284);
the "then report to Tarn" nudge stays in the step hints, and the next-step arrow
routes the short hop to 5283 for the hand-in.

## Verification

1. **Boot test** — nuke instance saves, boot local, confirm the seven quests
   still load (`quests.LoadDataFiles() loadedCount` unchanged, no panic). YAML
   field addition only, so low risk, but a stray-indent typo would panic.
2. **GMCP smoke (automated gate the harness *can* do).** Drive a fresh char
   through (or admin-grant tokens for) each trail; on each step confirm the
   `Char.Quests` payload for the focused quest carries the expected
   `target_room` from the table above. The ASCII harness is SVG-blind and cannot
   send inbound `Char.Quests.Focus`, but it can read the outbound payload.
3. **Adversarial content playtest (SOP gate).** Per the content playtest-review
   SOP, run the bug-finder harness through the route-2 newcomer flow and read
   the output — primarily to confirm no prose/step regression and that the
   `hint`/panel focus still agrees with the intended step.
4. **Browser playtest (the visual gate — user).** The marker glyph, patina
   destination pin, and gold next-step arrow render only in the web client and
   cannot be seen by the harness. Final confirmation that each trail's marker
   lights and points correctly is the user's browser walk-through, same as the
   original minimap-marker feature.

## Files Touched

`_datafiles/world/dogmud/quests/{32-first_blood,35-first_heat,38-first_brew,
41-first_sign,44-first_casting,47-first_words,50-first_shot}.yaml` — add
`map_target:` lines per the table. ~18 line additions total, no deletions.
