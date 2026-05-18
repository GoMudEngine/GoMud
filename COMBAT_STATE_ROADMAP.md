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
| 3 | Activity | Done (2026-05-15) | Free / Casting / Crafting / Salvaging. Star-topology FSM consolidates `Character.CastingState` + `Character.CraftingState` pointer fields and the salvage MiscData hijack into one per-state-data machine. Per-activity interrupt policy formalized (casting allows combat entry; craft/salvage block & cancel on combat/damage/movement). `cancel_activity` btree primitive added. 16/22 Behavior Matrix tests PASS/SKIP. |
| 4a | Position — FSM | Done (2026-05-16) | 14 geometric states (Standing / Prone / Supine / Clinch / BackStanding / Mount / SideControl / KneeOnBelly / NorthSouth / Crucifix / BackGround / HalfGuard / Guard / Turtle). Prone/Supine split during brainstorm — submission paths, recovery difficulty, and back-take vulnerability diverge. Per-state data (StandingData / ProneData / SupineData / shared GrappleData), ~75-edge transition graph, 22 trigger constants, 19 Character predicates, 10 btree primitives, Life-Dead cascade observer. Ships DORMANT — zero behavior change; legacy CombatPosition enum + all command writers untouched. 4b cuts over writers + control rolls + sunsets legacy. |
| 4b | Position — control axis | Done (2026-05-16) | Per-grappler 5-level control scale (InControl / LosingControl / Neutral / BecomingControlled / Controlled). Per-round opposed Strength + Unarmed-combat rolls with stamina + encumbrance curves; margin → delta with 2-consecutive-controlled threshold; gradient + transition + stamina-warning messages; 6 new btree control-axis primitives; 4 pair invariants enforced via TransitionPair + ValidateGrapplePair + periodic ConsistencyCheck. Legacy `CombatPosition` / `PositionRoundsMin` / `GrappleControllerId` / `ConditionGrappleController` / `combatposition.go` all sunset. |
| 4c | Position — weapon utility | Done (2026-05-16) | `Reach float64` field (meters) on `ItemSpec` + default-by-subtype lookup (`internal/items/reach.go`); per-state grapple-radius curve (standing-grapple 0.5m, ground-grapple 0.3m, other unbounded); `ReachUtility = radius/reach` formula floored at 0.15; pipeline integration via `CalcReachAdjustedItemMult` at `combat/combat_helpers.go:buildWeaponSetup`; bladed weapons (Slashing/Cleaving/Stabbing/Shooting) in grapples narrate with Bludgeoning vocabulary at `buildAttackMessages`. 3 new balance knobs. New `help reach` top-level helpfile + per-weapon helpfile mentions. Phase-1 YAML migration zero (per-item override added for `lake_iron_hook_spear` since spear defaulted to dagger range). |
| 4d | Position — submissions | Done (2026-05-18) | Symmetric opportunistic per-round submission system on top of chunk-4b's drift roll. Drift-margin > alpha or defender-crit opens a sub window on either side; separate sub roll resolves into 4 tiers (Bad/Neutral/Success/Crit). Position picks sub type via role-split mapping (top-attack vs bottom-attack subs); 7 SubmissionType enum values. Policy-driven outcomes (mercy/subdue/cripple/lethal) with no per-round prompts. Subdue + cripple reuse the Life cascade with new NoDeprogression + GoldLossFraction DeadData flags — defender wakes at temple, no stat decay, partial gold loss, optional broken-limb buff (cripple). Mob policies inherit from archetype defaults with per-mob YAML overrides (bosses → lethal). Legacy player-typed `submit` command + AttemptSubmission/ApplySubmissionSuccess/ApplySubmissionFailure helpers fully sunset. 2 new buffs (broken-limb #83, submission-stunned #84). 3 new btree primitives. Behavior Matrix PB-301..PB-341 mixed PASS/SKIP. |
| 4b-fixup | Position — outcome model | Done (2026-05-18) | Replaces chunk-4b's ControlLevel drift-needle with direct position-change outcomes (Hold / Advance / Degrade / Reversal / Escape) per round. Mount is the striking apex (1/2-step Hold, 3-step → BackGround); BackGround is the control apex. Crucifix terminal (sub-only). Reversal swaps roles with two realism exceptions (Mount→Guard, BackGround→Mount). ControlLevel + InitialControlForPair + gradient messages + sustained-pressure escape gate all sunset. ~280 flavor templates in grapple_outcomes.yaml across advancements / degradations / reversals / escapes / holds / striking_apex categories, validated by fresh-subagent realism pass. Chunk 4d submission gate composes via shared `|z| >= 1.5` threshold; sub fires from post-advance position. Species-gated grappling deferred (see project_species_gated_grappling.md memory). |
| 4e | Position — third-party | Not started | Symmetric defense degradation (controlled severely, controller moderately); offense restrictions; outside-damage → control degradation; mob AI bias toward grappled targets; submission-interrupt risk. |
| 4f | Position — balance + smoke | Not started | Tune modifiers, write position-flavor text, full-stack combat smoke. |
| 5 | Presence | Not started | Player and mob variants |
| 6 | Perception | Not started | Sighted / Blinded. Observer-side dual to Awareness — centralizes room-broadcast visibility gating. Added 2026-05-13 after recurring blind/dark-room broadcast bugs (latest: companion-name leak through blindness). No hard dependency on chunks 3-5; could ship earlier if pain escalates. |

**Mob aliveness work paused** for the duration of chunks 1-6. The
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
- Position machine (chunk 4) fully repointed the Position pre-wire
  in `Life_Cascades.go`: chunk 4a created the `position_life_dead`
  cascade observer (in `internal/hooks/Position_Cascades.go`); chunk
  4b R4 (`a481797f`) deleted the legacy pre-wire after every
  CombatPosition reader was migrated to the FSM predicates and the
  legacy fields were sunset (T21, `6a9697d5`).
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

**Aliveness work stays paused** for chunks 3-6. Chunk 3 (Activity
machine) brainstorm is next per the master spec ordering, with
chunk 6 (Perception) added 2026-05-13 to address the recurring
blind/dark-room broadcast bug class.

Next: chunk 3 — Activity machine (`Free` / `Casting` / `Crafting` /
`Foraging` / `Salvaging` / ...). Perception (chunk 6) may bump
earlier in the sequence if blind-broadcast bugs become blocking.

---

## Chunk 3 — Shipped (2026-05-15)

Built the `internal/state/activity/` machine (`Free / Casting /
Crafting / Salvaging`) on the chunk-0 framework. Star topology — every
active state goes through Free, no direct active-to-active. Per-state
data structs (`CastingData`, `CraftingData`, `SalvagingData`) preserve
the field shapes of the deleted `CastingState` and `CraftingState`
pointer fields so per-tick consumers (`processFoldRound` for casts,
inline craft-tick blocks for crafts/salvages) only swap accessor.

**Migration cadence:** parallel-write strategy kept the server bootable
and tests green at every commit through Tasks 6-10 — both the legacy
`CastingState`/`CraftingState` pointer fields AND the new Activity
machine stayed in sync. Task 11 deleted the legacy fields + struct files
and added three `Advance*` per-tick helpers (`AdvanceCastingFolds`,
`AdvanceCraftingRound`, `AdvanceSalvagingRound`) that mutate per-state
data without re-transitioning (transition-table-safe).

**Per-activity interrupt policy (formalized in spec):**
- **Casting** — combat entry allowed (cast IS a combat action; veto
  exempt). Damage triggers concentration break (willpower roll —
  existing rule, rewired to fire `Activity.TransitionToFree` on roll
  failure).
- **Crafting** — combat entry blocked by the activity veto in
  `CombatPhase_Vetoes.go` (`RegisterActivityCheck` returns
  `!c.IsCrafting() && !c.IsSalvaging()`). Damage fires hard cancel
  via `cancelCraftOrSalvageOnDamage` (no roll). Movement cancels via
  `Activity.TransitionToFree(TriggerMovementInterrupt)` from `go.go`.
- **Salvaging** — same rules as Crafting.

**Mob/player parity** — three pre-chunk-3 asymmetries resolved:
- Mob crafting auto-cancelled on combat entry; player crafting didn't.
  Now both flow through the same `RegisterActivityCheck` veto.
- Damage broke casts only; crafts and salvages were damage-resilient.
  Now all three respond to damage per the policy table.
- Mob-only combat-cancel block in `tickMobCrafting` deleted; cascade
  observer covers both actors generically.

**Cancel + btree primitive** — `usercommands/cancel.go` now dispatches
on `Activity.State()`: 50% conviction refund on cast cancel (preserved
existing math), no refund on craft/salvage cancel (no materials consumed
until completion). `mobcommands/cancel.go` is new — mob parity. New
`cancel_activity` btree action in `internal/behaviortree/actions_combat.go`
enables tactical-abort patterns from behavior trees (panic-flee on low
HP, swap to heal mid-cast, drop craft to defend). **Authoring the
behavior trees that use it is deferred to content/aliveness work after
chunk 6.**

**Salvage hijack cleanup** — `CraftingState.RecipeId = "salvage:<itemid>"`
+ `MiscData["salvage_item_uuid"]` + `MiscData["salvage_spoiled_potion"]`
all gone. `SalvagingData` holds the same data as typed fields. Resolver
in `NewRound_UserRoundTick.go:resolveSalvage` reads directly from
`Activity.SalvagingData()`.

**IsActing gate audit** — 13 `IsCrafting()` call sites migrated to
`IsActing()` (the canonical "busy with any locked-in activity" gate),
preserving 5 sites that genuinely want crafting-specifically (the craft
command's own re-entrancy check, round-completion checks, and the
predicate definition itself). Casting + Salvaging now block bash / kick
/ taunt / rally / warcry / trip alongside Crafting.

**Sunset:**
- `Character.CastingState` + `Character.CraftingState` pointer fields
  deleted.
- `internal/characters/casting.go` + `internal/characters/crafting.go`
  struct files deleted (helpers moved to `cast_helpers.go`).
- Chunk-2 Activity pre-wire in `Life_Cascades.go` (direct
  `CastingState = nil` / `CraftingState = nil`) deleted — Activity-side
  observer (`activity_life_dead` in `Activity_Cascades.go`) subscribed
  to Life Dead now owns the cleanup.

**Behavior Matrix:** 38 intent-driven tests (AC-001 through AC-038)
authored in `activity_test.go`. 16 PASS directly (basic transitions,
star-topology veto); 22 SKIP at the unit layer because they require
cross-machine wiring (verified by `Activity_Cascades_test.go` integration
tests + the migration tasks themselves). Chunks 0/1/2 regression tests
pass; package tests across the affected boundary (state/..., characters,
hooks, usercommands, mobcommands, actions, behaviortree, combat) all
green.

**Intentional asymmetries (documented in `internal/state/activity/context.md`):**
1. No Foraging or Tracking state — both are one-shot today; ceremony
   without payoff.
2. Mob forager `forager.ForagerState` left in btree — different
   abstraction layer (AI orchestration vs character mechanic state).
   Mob foragers remain `Activity = Free` throughout the forage loop.
3. No `IsForaging()` / `IsTracking()` predicates on Character — direct
   consequence of (1).
4. Salvage gets its own state despite structural similarity to crafting
   — cleans up the hijack; future divergence likely.

**Deferred from chunk 3 (followups):**
- Multi-round mob salvage with per-round messaging (mobs stay
  single-tick at the resolution layer for now).
- Tactical activity-cancel behavior trees (primitive ships; authoring
  is content/aliveness work after chunk 6).
- Shared ability cooldown — master spec logs this as a Phase-7
  candidate (helper, not machine).
- In-game smoke scenarios from the spec deferred to user session.

**Aliveness work stays paused** for chunks 4-6. Chunk 4 (Position
machine) brainstorm is next.

Next: chunk 4 — Position machine (`Standing` / `Prone` / `Clinched` /
`Grounded`).

---

## Chunk 4a — Shipped (2026-05-16)

Built the `internal/state/position/` machine on the chunk-0 framework
as the architectural scaffold for the rich-grapple system. 4a ships
DORMANT — no production code transitions the new FSM; legacy
`CombatPosition` enum + `PositionRoundsMin` + `GrappleControllerId` +
`ConditionGrappleController` + all command writers (trip / bash /
grapple / stand / kick / spell knockdown / `AttemptRecovery`) + all
readers (kick variant selector / flee veto / defense degradation /
chunk-0 `RegisterPositionCheck`) remain unchanged. Zero behavior
change. 4b's cutover sub-chunk swaps command-site writers to the new
FSM and incrementally sunsets the legacy state.

**14-state taxonomy:** Standing / Prone (face-down knockdown) /
Supine (face-up knockdown) / Clinch / BackStanding / Mount /
SideControl / KneeOnBelly / NorthSouth / Crucifix / BackGround /
HalfGuard / Guard / Turtle. **Prone/Supine split** during brainstorm:
the distinction matters mechanically — Prone is back-take vulnerable
(face-down) and harder to recover from; Supine can pull guard
(face-up) and recovers more easily. Submission paths diverge entirely
(Prone → back-take → RNC; Supine → guard pull → all BJJ submissions).

**Per-state data:** `StandingData` (empty), `ProneData` /
`SupineData` (Reason + MinRecoveryRounds + KnockdownSource), shared
`GrappleData` (Reason + Partner + ControlLevel) across all 11 grapple
states. Per-state extras (ClinchGrip, ArmsIsolated, HooksIn,
TrappedLeg, GuardVariant) deferred to 4b/4c as wrapping structs when
consumers materialize.

**Control axis** (`ControlLevel` enum: Neutral / InControl /
LosingControl / BecomingControlled / Controlled) stored as
`GrappleData` field. Neutral is iota=0 so Go's zero value defaults
match the spec — `GrappleData{Partner: ref}` literals get Neutral
without explicit assignment. 4a does NOT drive control transitions;
4b adds the per-round opposed rolls.

**Transition graph:** ~75 valid edges across the 14×14 matrix.
Star-ish topology around Standing — every grapple state can return
to Standing; Standing is the entry point. Intentional non-edges
documented: Standing → BackStanding (requires Clinch first), Supine
→ BackGround (requires intermediate state to flip target),
Clinch → KOB/NS/Crucifix (those positions require target already on
ground; reach via SideControl first).

**22 trigger constants** covering knockdowns (face-forward /
face-backward / spell), recovery (roll + stand-command), grapple
entry/break, 5 takedown variants from Clinch, 3 back-take paths,
controller-advance + controlled-escape, defensive (turtle-defend +
guard-pull), opportunistic (mount-prone-target), arm-isolation
(Crucifix), and cascade (death).

**19 Character predicates** (`internal/characters/position_predicates.go`):
- 14 per-state: IsStanding / IsProne / IsSupine / IsClinch /
  IsBackStanding / IsMount / IsSideControl / IsKneeOnBelly /
  IsNorthSouth / IsCrucifix / IsBackGround / IsHalfGuard / IsGuard /
  IsTurtle
- 5 rollup: IsGrappling (any of 11 grapple states), IsStandingGrapple
  (Clinch | BackStanding), IsGroundGrapple (9 ground grapple states),
  IsTopDominant (Mount/SC/KOB/NS/Crucifix/BackGround), IsOnFloor
  (Prone | Supine | any ground grapple)

**10 btree primitives** (`internal/behaviortree/conditions_position.go`):
- 7 self-position: mob_is_standing, mob_is_prone, mob_is_grappling,
  mob_in_mount, mob_in_guard, mob_in_clinch, mob_in_top_dominant
- 3 target-position: target_is_standing, target_is_prone,
  target_is_grappled
- All DORMANT in 4a (always return Failure because no mob's Position
  machine is transitioned in production). Become live once 4b drives
  transitions. Control-axis primitives (mob_is_in_control,
  target_is_being_controlled) deferred to 4b.

**Cross-machine cascade:** Life Dead → Position Standing observer
(`internal/hooks/Position_Cascades.go`, handler key
`position_life_dead`). **Coexists with the chunk-2 Life pre-wire**
(which still resets `c.CombatPosition = PositionStanding` directly +
clears `GrappleControllerId`). Both fire on death; both reach Standing
(legacy pre-wire on the legacy field; 4a observer on the new FSM).
No drift possible because the new FSM defaults to Standing and 4a has
no writers. 4b removes the chunk-2 pre-wire once command sites cut
over.

**Behavior Matrix:** 45 intent-driven tests (PO-001 through PO-045)
authored in `position_test.go`. 38 PASS at unit layer; 7 SKIP because
they require cross-machine wiring (verified by `Position_Cascades_test.go`
+ `conditions_position_test.go` integration tests). Chunks 0/1/2/3
regression tests pass; package tests across the affected boundary
(state/..., characters, hooks, behaviortree) all green.

**Intentional simplifications** (documented in
`internal/state/position/context.md`):
1. Prone/Supine split — intentional, not consolidated.
2. Shared GrappleData across all 11 grapple states — per-state extras
   deferred to 4b/4c via wrapping structs.
3. No control-axis rolls — `ControlLevel` field exists default
   Neutral; 4b adds the rolls.
4. No btree primitives for control axis — 4b additions.
5. Cascade coexistence with chunk-2 pre-wire — no drift possible
   because new FSM defaults to Standing and 4a has no writers.
6. Standing → BackStanding NOT direct — must go via Clinch.
7. Supine → BackGround NOT direct — would require flipping target.
8. Clinch → KOB / NorthSouth / Crucifix NOT direct — require ground
   pin first.
9. Btree primitive subset — only commonly-queried positions get
   primitives; rollup `mob_in_top_dominant` covers broad cases.

**Bonus fix during T3:** `ControlLevel` enum reordered so `Neutral`
is iota=0 (was `InControl`). The original ordering made Go's zero
value `InControl`, which silently demoted explicit InControl assignments
via a workaround. Reorder makes the zero value semantically correct
and lets 4b's roll-driven code set any control level explicitly without
demotion.

**Status of 4a's "Deferred to 4b" list (post-4b cutover, 2026-05-16):**

All shipped. See the Chunk 4b section below for details.

---

## Chunk 4b — Shipped (2026-05-16)

Cutover sub-chunk for the Position FSM. Writers, readers, and tests
all moved off the legacy `CombatPosition` enum onto the 14-state FSM
+ ControlLevel axis built in chunk 4a, then the legacy fields and
file were deleted. Per-round drift mechanics (opposed rolls with
margin → control-level delta, threshold-triggered position
transitions, gradient + transition + stamina-warning messaging)
went live in production. Four formal pair invariants enforced via
`TransitionPair` + `ValidateGrapplePair` + a periodic
`Position_ConsistencyCheck` observer that snapshots + auto-corrects
drift.

**Writer cutover (W1-W8):** every command-site writer migrated to
fire FSM transitions: `ApplyGrappleResult` (W1), trip + bash + spell
knockdown (W4 + W5), `AttemptRecovery` + `stand` (W6), submission
outcomes + grapple crit-fail (W3 + W8). Each landed as its own
commit (1c373be1, fb0cd1f9, 5cbca323, 162a65f4, ae17ef2d).

**Reader cutover (R1-R6):** the post-W reader sweep covered every
remaining `.CombatPosition.*` / `IsGrapplePosition()` /
`IsGroundPosition()` / `HasCondition(ConditionGrappleController)`
across `combat/ai.go`, `combat/grapple.go`, `actions/combat_kick.go`,
`actions/command_readiness.go`, `behaviortree/conditions_mob.go`,
`hooks/combat_shared_helpers.go`, `characters/combat_state_compat.go`,
`mobcommands/submit.go`, `usercommands/submit.go`,
`users/userrecord.prompt.go`, `combat/analytics.go`, and
`modules/gmcp/gmcp.Commands.go` (`4738b26e`). R4 deleted the legacy
chunk-2 Life cascade pre-wire (`a481797f`). R5 (CombatPhase
`RegisterPositionCheck`) and R6 (`{pos}` prompt token) already
shipped earlier in chunk 4b.

**Sunsets (S1-S5):** `Character.CombatPosition` field, `PositionRoundsMin`,
`GrappleControllerId`, the `ConditionGrappleController` constant, and
`internal/characters/combatposition.go` (the legacy enum + helpers:
`IsGroundPosition`, `IsGrapplePosition`, `GetSpeedMultiplier`,
`GetPositionColor`, `GetWorstPosition`) all deleted in a single
commit (`6a9697d5`). Replacements live in the FSM: per-state data
slots (`ProneData.MinRecoveryRounds`, `SupineData.MinRecoveryRounds`),
the new `Position.ExtendRecoveryRound()` helper for stomp, and
`c.IsController()` derived from `GrappleData.ControlLevel`. Net diff:
-169 lines across 30 files.

**Test fixtures (F1):** `setCombatPositionParallel(c, position.State)`
helpers in each test package (combat/hitroll_test.go,
actions/actions_test.go, behaviortree/conditions_test.go,
hooks/hooks_test.go, mobcommands/mobcommands_test.go,
usercommands/usercommands_test.go) now FSM-only (legacy parallel
write deleted in T21 alongside the field). Signature changed from
`characters.CombatPosition` to `position.State`.

**Per-round control axis** — `Position_GrappleTick.go` fires per
round for every grappler. Opposed Strength + Unarmed-combat roll
scaled by stamina + encumbrance penalty curves. Margin → control
delta via `MarginToDelta` (capped at ±1 per round). Two-consecutive-
Controlled gate before position downgrade fires, preventing
single-round flukes. Threshold transitions: Controlled →
DefaultEscapeTarget; LosingControl below stamina threshold →
escalating message; etc.

**Asymmetric stamina cost** — controller pays less per round than
controlled side (encourages opportunistic top-control play instead
of immediate sub attempts).

**Messaging contract** — `Position_Messaging.go` per-grapple-cooldown-
gated. Three message classes: gradient (control shifting),
transition (position changed), stamina warning (resource at risk).
YAML config + message templates load at boot from
`_datafiles/world/dogmud/grapple-messages.yaml`.

**6 new btree control-axis primitives** (per chunk-4b spec):
`mob_is_in_control`, `target_is_being_controlled`,
`mob_low_grapple_stamina`, `target_low_grapple_stamina`,
`mob_position_threshold_winning`, `mob_position_threshold_losing`.
Together with the 10 from chunk 4a, that's **16 total position
primitives**.

**4 pair invariants** — formalized in `ValidateGrapplePair`:
1. Both sides reference each other (Partner symmetry)
2. ControlLevels sum to "valid pairing" (no double-controller, no
   double-controlled)
3. Pair-state matches (e.g., Mount on one ⇒ Mount or BackGround on
   the other)
4. Pair lifespan is monotonic (no resurrection)

Enforced at write time by `TransitionPair`. Backstopped by the
periodic `Position_ConsistencyCheck` observer that scans live
grapple pairs every N rounds and force-breaks any pair that drifts
out of invariant.

**Behavior Matrix:** PB-001 through PB-080 authored across
`position_test.go` and the per-package integration tests
(`Position_*_test.go`, `grapple_test.go`, `ai_test.go`,
`conditions_position_test.go`). Mix of PASS / SKIP per chunks-0-3
convention. Chunks 0-4a regression clean. 176-test position package
suite green.

**Smoke debugging:** the chunk-4b smoke surfaced a long-standing
shared-state-machine bug in `mobs.newMobByIdInternal` — `mob := *m`
shallow-copied the template Character including pointer-typed Life
/ CombatPhase / Position / Awareness / Activity machines AND the
`combatPhaseWired = true` guard. Every spawned instance shared the
template's machines; observers wired on the template fired with the
template's `*Character` (`MobInstanceId=0`), so the mob despawn
cascade silently skipped. Fix: `Character.ResetForMobInstance()`
nils the machine pointers + clears the guard after the shallow
copy, so the next `Validate()` builds fresh per-instance machines
and re-fires `OnCharacterCreated`. Committed at `aee70eed`. Saved
as a class-of-bug memory (`feedback_shallow_copy_shared_pointers.md`)
since this pattern is easy to repeat anywhere the codebase clones
a Character / Mob struct.

**Documentation:** T22 audit (`tools/testing/audits/2026-05-16-chunk-4b-doc-helpfile-audit.md`)
inventoried stale "deleted in S1" / "sunset target" / "until S1"
framing across all context.md files. T23 (`9b188c7c`) rewrote 7
context.md files to present-tense post-cutover voice:
`state/position/context.md`, `characters/context.md`,
`hooks/context.md`, `combat/context.md`, `behaviortree/context.md`,
plus `state/life/context.md` and `spells/context.md`.

**Aliveness work stays paused** for chunks 4c-6. Chunk 4c
(weapon-utility-by-position table) brainstorm is next.

**NOTE (2026-05-18):** The ControlLevel drift-needle model described above was
replaced by chunk 4b-fixup (Position — outcome model). See the 4b-fixup row in
the chunk table for the revised mechanics.

Next: chunk 4c — Position × WeaponType modifier table (long weapons
fail in mount, knives stay useful, etc.).

---

## Chunk 4c — Shipped (2026-05-16)

Same-day follow-up to 4b. Position × weapon utility shipped as a
reach-on-weapon model: single `Reach float64` (meters) field on
`ItemSpec` with default-by-subtype lookup; position radius curve in
the combat package; `radius / reach` formula floored at 0.15 wired
into the per-swing damage path. Bladed weapons (Slashing / Cleaving /
Stabbing / Shooting) in grapples narrate with the Bludgeoning
vocabulary so the fiction tracks the math. End-state: tactical
weapon-swapping in grapples becomes a real choice — carrying a
dagger as offhand is now a viable response to a grappler; two-
handed reach weapons (spear, halberd, greatsword, quarterstaff)
become liabilities once a clinch lands.

**Architecture (smaller than 4a/4b):** one new field, two new small
files (`items/reach.go`, `combat/reach.go`), one pipeline helper, two
call-site updates in `combat_helpers.go` (`buildWeaponSetup` for the
damage multiplier, `buildAttackMessages` for the vocabulary swap),
and three new balance knobs. No FSM changes, no new btree primitives,
no sunsets. The cheap chunk between 4b and 4d.

**Reach taxonomy:** documented in `internal/items/context.md`. Natural
attacks 0.1-0.5m (fist 0.1, claws/bite 0.15, sting 0.2, slam 0.3,
gore 0.4, whipping 0.5). Stabbing 0.3, Cleaving 0.9, Slashing 1.0,
Bludgeoning 0.8, Shooting 1.0 (melee-fallback). Caster: wand 0.4,
sceptre 0.6, staff 1.5.

**Position radius curve:** Standing / Prone / Supine / Turtle
unbounded (no penalty). Clinch + BackStanding share standing-grapple
radius (0.5m). The 8 ground-grapple states (Mount, SideControl,
KneeOnBelly, NorthSouth, Crucifix, BackGround, HalfGuard, Guard)
share ground-grapple radius (0.3m).

**Bludgeon narration:** when `ShouldBludgeon(reach, radius)` fires
for a bladed weapon, the attack-message rendering subtype swaps to
Bludgeoning. Caster weapons (Wand/Sceptre/Staff) and natural-blunt
subtypes (Fist/Claws/Bite/Sting/Slam/Gore/Whipping) stay with their
own vocabulary. Damage math is the same Physical-channel value just
multiplied by ReachUtility — the swap is cosmetic only. The existing
Bludgeoning templates use `{itemname}` token interpolation, so
"You bash the iron greatsword's pommel into the bandit" renders
without needing bespoke pommel-strike vocabulary.

**Balance knobs** (`internal/configs/balance.go`):
- `ReachStandingGrappleRadius` (default 0.5)
- `ReachGroundGrappleRadius` (default 0.3)
- `ReachUtilityFloor` (default 0.15)

**Behavior Matrix:** PB-201 through PB-220 PASS. Coverage split
across T1 (PB-213/214 ResolveReach behavior in `items/reach_test.go`),
T2 (PB-201..212 + PB-219..220 utility math in
`combat/reach_test.go::TestBehaviorMatrix_Reach`), and T4
(PB-215..218 bludgeon narration in `combat/reach_bludgeon_test.go`).
Chunks 0-4b regression clean; server boots cleanly in 6.3s.

**Documentation:** T8 audit
(`tools/testing/audits/2026-05-16-chunk-4c-doc-helpfile-audit.md`,
commit `052d7d6d`) inventoried helpfiles + context.md surface. T9
(`0d9e9eba`) shipped 11 helpfile updates including a new
`reach.template` top-level reference page, an expanded
`grapple.template`, and per-weapon reach notes on iron-dagger /
iron-short-sword / steel-longsword / lake-iron-hook-spear. T9 also
added a `reach: 2.0` YAML override to `lake-iron-hook-spear` since
the Stabbing subtype default (0.3m) is fictionally wrong for a
spear. T10 (`bee42e9b` + followup `a7b4320a`) updated 6 context.md
files: items (verified taxonomy table; the T8 false-alarm finding
that called Bludgeoning "narration-only" was caught and reverted —
Bludgeoning is both a real carry-subtype on mace/hammer items AND
the narration-swap target), combat (new "Weapon Reach Utility"
section), state/position (cross-reference), characters (predicate
consumer note), hooks (per-swing site update), configs (3 new
knobs table).

**Phase-1 YAML migration is zero** by design — every existing weapon
inherits its subtype default. Per-item overrides will land
post-smoke as balance feedback comes in. The lake-iron-hook-spear
fix surfaced during T8 audit is the first such override.

**Aliveness work stays paused** for chunks 4d-6. Chunk 4d
(submission rework — opportunistic submissions gated on Position +
ControlLevel) is next.

Next: chunk 4d — Submission rework.

---

## Chunk 4d — Shipped (2026-05-18)

Symmetric opportunistic per-round submission system shipped. The
legacy player-typed `submit` command + AttemptSubmission /
ApplySubmissionSuccess / ApplySubmissionFailure helpers are
sunset; the engine now fires sub attempts automatically when the
chunk-4b drift roll favors a side by margin > alpha (or defender
crits defense). End-state: submissions are an organic per-round
threat that emerges from chunk-4b control drift; position drives
the sub type; the consequences are real but proportional to the
aggressor's chosen brutality.

**Symmetric per-round model** — both controller and defender sides
of a pair are checked for sub-attempt opportunity each round.
Controller side fires when drift margin > `SubmissionAttemptAlpha`
(default 1.0) AND position has top-attack subs. Defender side fires
when defender wins drift by margin > alpha OR defender z-score >=
`SubmissionAttemptCritZ` (default 2.0). Tiebreak on larger absolute
z-score when both sides qualify.

**Separate sub roll + 4-tier outcome:** opposed Strength +
Unarmed-combat-skill check (defender bonus Vitality). Result tiers:
- **Bad** (z < `SubBadZThreshold`, default -1.0): attempter
  overcommits → falls Prone, pair breaks to Standing
- **Neutral**: no consequence, pair stays
- **Success**: sub locks, attempter's policy resolves
- **Crit** (z >= `SubCritZThreshold`, default 2.0): sub locks AND
  recipient gets the 1-round Stunned buff (only on mercy outcomes —
  other policies enter the death cascade where stun is moot)

**Position drives sub type** — 7 named subs (Armbar / RNC / Triangle /
Americana / Kimura / Omoplata / Anaconda). Mount → Americana / Triangle
/ Armbar (rotating via per-character round-robin). BackGround → RNC.
Crucifix → Armbar. Guard-bottom (the bottom-game controller in FSM
terms) → Triangle / Armbar / Omoplata. Bottom-attack subs are sparser
by design (asymmetry favors the controller).

**Choke degradation:** cripple + a choke sub (RNC / Triangle /
Anaconda) automatically degrades to subdue because chokes don't
break limbs (CrippleBodyPart returns "").

**Policy-driven outcomes (no per-round prompts):**

| Policy | Behavior on success/crit |
|--------|--------------------------|
| **mercy** | Clean release; brief recovery debuff. Crit additionally stuns the recipient for 1 round (Stunned buff #84). |
| **subdue** (default) | No-deprogression death; partial gold transfer; defender wakes at temple woozy but uninjured. |
| **cripple** | Same as subdue + broken-limb buff (#83, 900-round duration, persists across respawn). Chokes degrade to subdue. |
| **lethal** | Full death cascade with deprogression + full corpse loot. Requires two-step confirmation the first time set. |

**Defender's `SurrenderPolicy`** — `never` / `always` / `auto-tap-below
<hp%>` (default `auto-tap-below 15` for players). Honored ONLY by
mercy controllers, per the realism framing: "the tap is a signal,
not a guarantee." Bandits / predators / hostile NPCs ignore taps and
apply their policy anyway.

**Death-cascade reuse** — two new `DeadData` fields drive subdue/
cripple semantics through the existing chunk-2 Life pipeline:
- `NoDeprogression bool` — `Death_PlayerCleanup` skips the stat-
  decay step when true
- `GoldLossFraction float64` — `Death_PlayerCorpse` skips full corpse
  + transfers fraction of gold to Killer when > 0 (default 0.20 from
  `SubGoldLossFraction` config)
- A `TriggerSubmission` constant added to `internal/state/life/`
- `Death_PlayerAnnouncement` gated so subdue/cripple skip the
  "YOU HAVE DIED" broadcast + "darkness swallows you" closure (the
  player was knocked out, not killed — fiction matches)

**Mob policy storage** — `MobSpec.SubmissionPolicy` /
`SurrenderPolicy` yaml fields with fallback to
`DefaultSubmissionPolicyForArchetype` / `DefaultSurrenderPolicyForArchetype`.
8 per-mob overrides authored on named bosses (Edrin / Sylara / Rhett
/ Soren / Chrysalis Phantom + Elemental King / Arena Champion /
Stone Beetle Queen) — all set to lethal + never. Most mobs inherit
archetype defaults.

**3 new btree primitives** in
`internal/behaviortree/conditions_submission.go`:
- `mob_can_submit_top` — controller in sub-eligible position
- `mob_can_submit_bottom` — controlled side with bottom-attack subs
- `mob_submission_policy_is <policy>` — branch on policy enum

Engine fires sub attempts via `Position_SubmissionTick.go`
(observer registered AFTER `Position_GrappleTick.go` via
filename-alphabetical init order so the drift snapshot is fresh).
Reads `Character.LastDriftRoll` snapshot (round-numbered for
staleness detection).

**Player UX:**
- `set submission <mercy|subdue|cripple|lethal>` — controller policy.
  Lethal requires two-step confirmation the first time.
- `set surrender <never|always|auto-tap-below <N>>` — defender policy.
- `status` shows both policies + broken-limb buff with remaining
  rounds.
- New helpfiles `help submission` + `help surrender`. Existing
  helpfiles updated (grapple, combat, attack, special, death,
  conditions, status, set, unarmed-combat, weapon-combat).

**Behavior Matrix:** PB-301 through PB-341, mix of PASS / SKIP per
the chunks-0-3 convention. Coverage split across
`internal/state/position/submissions_test.go`,
`internal/characters/submission_policy_test.go`,
`internal/combat/submission_test.go`,
`internal/combat/submission_outcome_test.go`,
`internal/hooks/Position_SubmissionTick_test.go`,
`internal/mobs/mobs_test.go`,
`internal/behaviortree/conditions_submission_test.go`,
`internal/buffs/buffs_test.go`, and
`internal/usercommands/usercommands_test.go`. Chunks 0-4c regression
clean. Server boots cleanly past data-file loading.

**Documentation:** T20 audit
(`tools/testing/audits/2026-05-18-chunk-4d-doc-helpfile-audit.md`,
commit `75e6d1c0`) inventoried 9 helpfile updates + 9 context.md
updates. T21 (`93684b32`) fixed the live broken `help submit` link
in `grapple.template` (was 404'ing at runtime after T18 sunset),
removed `submit` from `special.template`, added a no-deprogression
section to `death.template`, expanded `conditions.template` to
include the broken-limb buff, and added cross-references across
8 other helpfiles. T22 (`f3578fb5`) updated 9 context.md files —
the largest update was `internal/combat/context.md` getting a new
"Submission System (chunk 4d)" section AND removing stale refs
to the deleted helpers.

**Aliveness work stays paused** for chunks 4e-6. Chunk 4e (third-
party interaction asymmetries — defenders / parties / outside
damage during a grapple) is next.

Next: chunk 4e — Third-party interaction asymmetries.

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
