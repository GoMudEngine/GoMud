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
