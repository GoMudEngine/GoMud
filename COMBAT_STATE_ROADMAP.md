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
| 1 | Awareness | Not started | Visible / Concealing / Hidden / Revealing |
| 2 | Life | Not started | Alive / Dead / Respawning |
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
