# DOGMud — Combat State Machines Roadmap

> Living document. Tracks the 6-chunk combat-state-machine redesign
> that replaces the overloaded `Character.Aggro` field with six
> orthogonal state machines + one flag.
>
> **Master spec:**
> `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`

## Why this effort exists

The combat-state surface of the engine has been fixed multiple times
(targeting >= 4x, flee >= 3x, aggro lifecycle >= 3x) without the root
cause being addressed: `Character.Aggro` was overloaded with three
meanings (current target, in-combat flag, fleeing sentinel), and
combat-adjacent state was scattered across 5+ stores with no unified
model.

This effort collapses that surface to one canonical framework:

- Six orthogonal state machines, each owning one concern
- One `NonCombatant` flag replacing the old `AutoAggro`/`Hostile` booleans
- Mob/player parity by construction — all six machines live on every
  `Character`, player and mob alike
- Intent-driven TDD: a Behavior Matrix drives RED-phase tests before
  each chunk's implementation starts
- Hard cutover within each chunk: old fields deleted at chunk end
  (chunk 0 defers field deletion only because ~200 reads remain)

---

## Progress

| Chunk | Title | Status | Branch / Notes |
|-------|-------|--------|----------------|
| 0 | Framework + Combat Phase | Done (2026-05-13) | `feature/mob-aliveness-1.3-crimes`. Framework package + Combat Phase machine; compat wrappers preserve `Aggro` API. Full field deletion deferred. |
| 1 | Awareness | Done (2026-05-15) | Visible / Concealing / Hidden / Revealing. FSM port + Hidden mechanic refresh. 33 Behavior Matrix tests (29 PASS + 4 SKIP). |
| 2 | Life | Done (2026-05-13) | Alive / Dead / Respawning. 252-line `mobcommands/suicide.go` + ~290-line `usercommands/suicide.go` consolidated into thin handlers + Life cascade + 14 observer files. Permadeath + extra-lives sunset. Auto-look after respawn teleport. 12 Behavior Matrix tests PASS + 15 SKIP (integration-deferred). |
| 3 | Activity | Not started | Free / Casting / Crafting / Foraging / Salvaging / ... |
| 4 | Position | Not started | Standing / Prone / Clinched / Grounded |
| 5 | Presence | Not started | Player and mob variants |

**Mob aliveness work paused** for the duration of chunks 1-5. The
aliveness substrate (memory, disposition, factions, schedules) is a
consumer of the state machines; building it before the substrate is
stable would create churn. Resumes after chunk 5.

**Chunk 2.7 (mob skullduggery suite)** Task 19 (smoke scenario 8)
unblocks when chunk 0 smoke confirms the thief-archetype regression is
fixed. The `SoftTarget` slot on `EvalContext` is already shipped.

---

## Chunk 0 — Shipped (2026-05-13)

Built the `internal/state/` framework (`Machine[S]`, transitions,
vetoes/cascades/observers, scheduled transitions) co-developed
with the first consumer (`internal/state/combatphase/`). Migrated
~90 Aggro readers across `usercommands/`, `hooks/`,
`behaviortree/`, `combat/`, `mobcommands/`. Migrated writers via
centralized dual-write in `SetAggro`/`EndAggro` compat wrappers
(`internal/characters/combat_state_compat.go`). Round driver
dispatches via Combat Phase state. Wired btree transition events
(`mob_engaging`/`mob_engaged`/`mob_disengaging`/`mob_combat_ended`).
Wired companion auto-assist via `SubscribeAttackersChange`.
Sunset `internal/hooks/aggro_helpers.go` (functions moved to
`combat_retarget.go` for the few remaining DoCombat-internal
callers). Introduced `Character.NonCombatant` flag + `Mob.AutoAggro`
field (legacy `Hostile` private-bridged for YAML backward compat).

**Marquee fix:** the chunk-2.7 thief-archetype bug is structurally
impossible. `EvalContext.SoftTarget` slot enables non-combat target
picking without triggering Combat Phase transitions. `target_random_player_in_room`
stashes there; `try_steal`/`try_plant`/`try_shadow` consume it via
`resolveSkullduggeryTarget`. Behavior Matrix tests CP-026 and CP-027
encode this structural property.

**Behavior Matrix complete:** 32 intent-driven tests (CP-001 through
CP-036, with some numbering reshuffles) cover entry/exit, vetoes,
multi-attacker tracking, surprise attack semantics, death cascades,
and per-state tick dispatch. Every test maps to an intent row, not
a parity-with-old-code row.

**Deferred from chunk 0 (followups):**
- `Character.Aggro` field NOT removed — 200+ direct reads remain
  across the codebase; preserved as compat surface via wrappers.
  Field deletion scheduled for a post-chunks-1-5 cleanup pass.
- `internal/hooks/NewRound_DoCombat_unified.go` NOT deleted —
  Stage 2b commit (`3aaa19cc`, pre-chunk-0) had already activated
  it as production code. Preserved as live dispatch.
- `combat_retarget.go` (the former `aggro_helpers.go`) — kept as
  the DoCombat-internal retarget/validate logic. Future cleanup
  may fold into Combat Phase cascades.

**Aliveness work paused** for chunks 1-5. Chunk 2.7 Task 19
(roadmap closeout for the skullduggery suite) remains pending
until user-driven smoke confirms scenario 8 (thief regression).

Next: chunk 1 — Awareness machine (`Visible` / `Concealing` /
`Hidden` / `Revealing`).

---

## Chunk 1 — Shipped (2026-05-15)

Built the `internal/state/awareness/` machine
(`Visible/Concealing/Hidden/Revealing`) on the chunk-0 framework.
Subscribed to Combat Phase's `OnEndOfRoundIfSurprise` callback to
close the chunk-0 surprise handshake at end of first combat
round. Replaces buff-#9-as-state-of-truth with Awareness state;
buff #9 stays as the side-effect carrier with the cascade in
`internal/hooks/Awareness_Cascades.go` keeping it mirrored.

**Marquee mechanic refresh** bundled with the FSM port:
- **No duration** on Hidden — persists until explicitly broken
  (combat, detection roll, light state change, logout, noisy
  action). Buff #9 YAML stripped of `triggerrate`/`triggercount`.
- **Stamina cost for hidden movement** — default 3.0× multiplier,
  stacks multiplicatively with encumbrance. Replaces a
  pre-existing hardcoded 1.5× in `GetMovementStaminaCost`.
- **Light-conditional sneak score** — sneaker-side 4-way
  conditional: baseline (dark/dark), 0.9× (dark sneaker, lit
  room), 0.85× (lit sneaker, lit room), 0.5× (lit sneaker, dark
  room — beacon in darkness). `CalcSneakScore(char, effectiveLit)`
  + `CalcSneakScoreVsObserver(sneaker, observer, room)` helper.
  NightVision observer treats sneaker as in a lit room for that
  observer's roll.
- **Noisy actions** break stealth via `TriggerNoisyAction`:
  `say`, `shout`, `rally`, `warcry`, `taunt`. Direct-target
  `whisper` stays quiet (confirmed during migration —
  no broadcast-form variant existed).
- **Logout safety valve** — players logging out while Hidden are
  forced through Revealing → Visible synchronously, so observers
  see the leave broadcast and reconnects start Visible.
- **Activity veto pre-wire** — `Visible → Concealing` blocked when
  the character is mid-cast or mid-craft. Chunk-3 will repoint
  the callback to the proper Activity machine.

**Sunset:**
- Buff #20 (`very_hidden`) deleted as dead content.
- ~49 `HasBuffFlag(buffs.Hidden)` / `HasFlagFromAnySource(buffs.Hidden)`
  readers migrated to `Character.IsHidden()` across 31 files.
- 8 explicit `CancelBuffsWithFlag(buffs.Hidden)` writers migrated
  to `Awareness.TransitionToRevealing(...)`.
- Sneak action's direct `AddBuff(9, ...)` replaced by Awareness
  state transitions; the buff-mirror cascade handles the buff.
- Zombie `aggro.go` / `aggro_helpers.go` files left behind from
  chunk-0 Task 18 finally cleaned up (the implementer who did
  Task 18 said `git rm` but files were still on disk).
- `internal/buffs/buffspec.go` `Validate()` now allows no-trigger
  buffs (needed for the no-duration Hidden buff).

**Behavior Matrix complete:** 33 intent-driven tests (AW-001
through AW-033) authored in awareness_test.go. 29 pass directly;
4 (AW-024-027, the CalcSneakScore truth-table rows) implemented
in `internal/actions/skill_helpers_test.go` (or sneak_test.go)
where the function lives — AW-024/025 implemented, AW-026/027
skipped because EmitsLight=true requires buff/equipment setup
beyond unit-test scope (covered by future in-game smoke).

**Deferred from chunk 1 (followups):**
- `internal/hooks/Awareness_LightChange.go` scaffolding shipped
  (event listeners wired); full re-roll body deferred — existing
  room-entry detection in `internal/hooks/go.go` continues to
  handle the primary case. Future expansion adds re-rolls for
  light-source equipment changes within a room.
- AW-026/027 unit tests skipped (integration-only).
- The 10 in-game smoke scenarios from the spec deferred to user
  session.

**Aliveness work stays paused** for chunks 2-5. Chunk 2 (Life
machine) brainstorm is next.

Next: chunk 2 — Life machine (`Alive` / `Dead` / `Respawning`).

---

## Chunk 2 — Shipped (2026-05-13)

Built the `internal/state/life/` machine (`Alive / Dead / Respawning`)
on the chunk-0 framework. Consolidated scattered death-cleanup logic
into a Life cascade + per-concern observer files. `usercommands/suicide.go`
shrank from ~290 lines to ~50 (thin handler chaining
`TransitionToDead → TransitionToRespawning → TransitionToAlive`).
`mobcommands/suicide.go` shrank from 252 lines to ~60 (thin handler;
mobs stay at `Dead`, observers handle the rest). Same-tick observer
firing for both player and mob death paths.

**Cascade + observer architecture:**
- `Life_Cascades.go` — cross-machine cleanup on `Alive → Dead` (Combat
  Phase → Idle, Awareness → Visible, casting/crafting nil, position
  Standing, grapple cleared, non-permanent buffs canceled, conditions
  cleared); on `Dead → Respawning` (resource refill to 5% of max,
  NoAggroTarget grace buff #81, clear PlayerDamage, CharacterVitalsChanged
  event).
- **Player death observers:** `Death_PlayerCleanup` (stat decay + skill
  rust + KD + party notify), `Death_PlayerAnnouncement` (room +
  global broadcasts, events.PlayerDeath queue, worldevents PvE emit,
  weakened/darkness text, instance ejection), `Death_PlayerCorpse`
  (corpse creation in death room).
- **Mob death observers:** `Death_MobLoot` (carried/equipped item drop,
  gold, corpse), `Death_AlivenessSubstrate` (fires events.MobDeath),
  `Death_MobInstanceCleanup` (DeleteMobInstance + DestroyInstance +
  CleanupMobSpawns + RemoveMob), `Death_MobBroadcast` (room "X has
  died" + Guide tempdata + worldevents.MobKilledByPlayer),
  `Death_MobBehaviorTree` (mob_die btree event), `Death_MobKillCredit`
  (EndAggro + KD.AddMobKill + OnFirstMobKill + party credit),
  `Death_MobCharmCleanup` (TrackRecentDeath + RemoveCharm).
- **Cross-cutting:** `Death_InboundAggroCleanup` (clears mobs and
  companions targeting the dying actor; fires for both player AND mob
  deaths).
- **Respawn observers:** `Respawn_PlayerTeleport` (`rooms.MoveToRoom`
  to `ResolveRespawnRoom` destination + belt-and-suspenders EndAggro),
  `Respawn_PlayerAutoLook` (fires `u.Command("look")` so the new room
  renders without manual command — UX fix, parallel fold-recall fix
  logged as followup).

**Character API additions:**
- `Life *life.Machine` field + `IsAlive()` / `IsDead()` / `IsRespawning()`
  predicates.
- `Die(killer, trigger)` helper in `die.go` — chains the appropriate
  transitions (mobs stay Dead; players chain Dead → Respawning →
  Alive same-tick). Callers pre-check ReviveOnDeath, dedupe, Shadow
  Realm.
- `ResolveRespawnRoom()` reads `home` setting → looks up
  `HomeLocations` (exported map in `respawn_home.go`) → falls back to
  `default` (room 0).
- `MobInstanceId` non-persisted field added (mirrors `Mob.InstanceId`)
  for cheap mob-actor gating in Life observers without a full instance
  scan.

**Combat-driven death migration:** the four production sites that
detect health-zero (`NewRound_DoCombat.go` sweep + handleAffected,
`NewRound_AutoHeal.go` player catch-all, `NewRound_MobRoundTick.go`
DoT/idle, `Buff_ApplyBuffs.go` buff-tick) now call `c.Die()` directly
instead of queueing `user.Command("suicide")` or `mob.Command("suicide")`.
Observers fire same-tick.

**Sunset:**
- Permadeath system removed entirely. `Character.ExtraLives` field +
  `Death.PermaDeath` / `LivesStart` / `LivesMax` / `PricePerLife`
  config knobs deleted. `{{ permadeath }}` template helper removed.
  Status template + about helpfile cleaned. `events.PlayerDeath.Permanent`
  field kept for upstream parity but always queued false. Scripting
  docs (`FUNCTIONS_ACTORS.md` `GiveExtraLife()`, `SCRIPTING_ITEMS.md`
  example) updated.
- ReviveOnDeath buff preserved (separate one-shot mechanic). Stat
  decay + skill rust preserved as normal-death penalties.

**Behavior Matrix complete:** 27 intent-driven tests (LI-001 through
LI-027) authored in `life_test.go`. 12 pass directly (LI-001 through
LI-007, LI-017-019); 15 SKIP because they require hook integration
(LI-008-016, LI-020-027 — verified by the hook observer files +
in-game smoke). Chunk 0 + chunk 1 regression tests pass; package
tests across state/life, state/combatphase, state/awareness,
characters, hooks, usercommands, mobcommands all green.

**Deferred from chunk 2 (followups):**
- Activity machine (chunk 3) will repoint the Activity pre-wire in
  `Life_Cascades.go` (currently clears `CastingState`/`CraftingState`
  directly) to the proper Activity machine query.
- Position machine (chunk 4) will repoint the Position pre-wire in
  `Life_Cascades.go` (currently clears `CombatPosition` /
  `GrappleControllerId` directly).
- Auto-look after fold-recall teleport — separate memory entry
  `project_auto_look_after_room_change.md` covers this parallel UX
  fix.
- Chunk 1 sneak-end-message cosmetic bug
  (`project_chunk1_sneak_end_message_bug.md`) — addressing during a
  later cleanup pass.
- 10 in-game smoke scenarios from the spec (player suicide flow,
  combat death, mob death + loot, mid-cast death, hidden death,
  grappled death, stat decay verification, multi-killer mob kill,
  permadeath path gone, chunk 0/1 regression) deferred to user
  session.

**Aliveness work stays paused** for chunks 3-5. Chunk 3 (Activity
machine) brainstorm is next.

Next: chunk 3 — Activity machine (`Free` / `Casting` / `Crafting` /
`Foraging` / `Salvaging` / ...).

---

## Architectural principles

- **Six machines, one flag.** Each machine owns exactly one concern.
  `NonCombatant` (the flag) replaces the overloaded `AutoAggro`/`Hostile`
  booleans from the legacy system.
- **Veto + Cascade hooks.** Every machine exposes `BeforeTransition`
  (veto, blocks the transition) and `AfterTransition` (cascade, fires
  after state change). Observers fire last and are read-only.
- **Synchronous transitions.** No goroutine or channel crossing inside
  the framework. The engine's single-threaded round loop ensures all
  state changes are serialized.
- **Single global scheduler.** `RoundScheduler.Tick()` is called once
  per round by the round driver; all scheduled transitions across all
  machines fire from this one tick.
- **Import-cycle-safe wiring.** Characters cannot import hooks. All
  veto/cascade wiring goes through `characters.OnCharacterCreated`
  callbacks registered by the hooks package at init time.

---

## Per-chunk design artifacts

| Chunk | Spec | Plan |
|-------|------|------|
| 0 | `docs/superpowers/specs/2026-05-13-state-chunk-0-framework-and-combat-phase-design.md` | `docs/superpowers/plans/2026-05-13-state-chunk-0-framework-and-combat-phase.md` |
| 1-5 | TBD (spec before each chunk picks up) | TBD |

---

## See also

- Master spec:
  `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`
- Framework package docs: `internal/state/context.md`
- Combat Phase package docs: `internal/state/combatphase/context.md`
- Characters integration: `internal/characters/context.md`
  (section: "Combat Phase Machine Integration (chunk 0)")
- Hooks integration: `internal/hooks/context.md`
  (section: "Combat State Machine Integration (chunk 0)")
- BTree SoftTarget fix: `internal/behaviortree/context.md`
  (section: "EvalContext.SoftTarget (chunk 2.7 fix)")
- Mob aliveness roadmap (paused): `MOB_ALIVENESS_ROADMAP.md`
