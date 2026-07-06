# Crash Site Boss Mechanics — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build the counterplay-rich Warden-Prime and Core Guardian encounters from
`docs/superpowers/specs/2026-07-06-crashsite-boss-mechanics-design.md`.

**Architecture:** Reuse existing engine primitives (fold-casting telegraph,
`InterruptTargetCast`, mob-grapple, mob-heal, `spawn_mob`, `TransportCompanions`
room-move, `BehaviorState` counters); add ~5 small Go hooks + config; author two boss
behavior trees, three add mobs, two boss abilities (fold-cast spells), and one airlock
room.

**Tech Stack:** Go (engine hooks in `internal/`), YAML (mobs, behaviors, spells, rooms,
config), the behavior-tree action/condition registry.

**Precedent to model:** `_datafiles/world/dogmud/behaviors/marches_spur_road/275-old_edrin.yaml`
(multi-phase boss: summons named adds, `state_equals`/`set_state` gates, `cooldown`
decorator, `send_room_text` narration).

**Branch:** `feature/crashsite-boss-mechanics` off `master`.

---

## Chunk A — Engine glue (TDD). All in `internal/`; each task independently testable.

### Task A1: Disruptor allowlists (config)

**Files:** Modify `internal/configs/config.balance.go` (+ its default-set file
`config.balance.misc.go`), `_datafiles/config.yaml`.

- [ ] Add two `GamePlay` (or `Balance`) config fields — general, not crash-site-specific:
  `BossInterruptItemIds []int` and `BossInterruptSpellIds []string`. Defaults empty.
- [ ] Populate in `_datafiles/config.yaml`: item-ids = `[<flashbang item id>]` (look it
  up: the flashbang recipe output — grep `_datafiles/world/dogmud/recipes` for `flashbang`
  → its `output.item_id`); spell-ids = `[neural-stun, sensory-overload, kinetic-shove]`.
- [ ] Helper accessors (e.g. `IsBossInterruptItem(id int) bool`,
  `IsBossInterruptSpell(spellId string) bool`) with a small unit test.

- [ ] **Commit:** `feat(combat): boss-interrupt disruptor allowlists (config)`

### Task A2: Interrupt a mob's cast with a thrown disruptor

**Files:** Modify `internal/usercommands/throw.go` (hit-resolution ~:193-200);
Test: `internal/usercommands/throw_test.go` (or the nearest existing throw test).

- [ ] **Test first:** a mob in a casting state (`mob.Character.Activity` casting) that is
  hit by a thrown item whose id ∈ `BossInterruptItemIds` has its cast interrupted (assert
  `InterruptTargetCast` effect — casting state cleared). A thrown non-disruptor item does
  NOT interrupt. Study `internal/actions/cast_interrupt.go:14` `InterruptTargetCast(target
  *characters.Character, by state.ActorRef)` for the call shape and how "is casting" is
  queried (`Activity.IsCasting()`).
- [ ] Implement: after a successful thrown-item hit on a mob, if
  `configs...IsBossInterruptItem(thrownItemId)` and the mob is casting, call
  `actions.InterruptTargetCast(&mob.Character, throwerRef)`.
- [ ] Run tests green; `go build ./...`.
- [ ] **Commit:** `feat(combat): thrown disruptors interrupt a casting mob`

### Task A3: Interrupt a mob's cast with a specific disruption spell

**Files:** Modify `internal/hooks/spell_resolution.go` (`resolveAgainstMob` ~:249);
Test: the nearest spell-resolution test file.

- [ ] **Test first:** casting mob + a player casting a spell whose id ∈
  `BossInterruptSpellIds` → cast interrupted; a non-allowlisted spell → not.
- [ ] Implement: in `resolveAgainstMob`, after the effect applies, if
  `IsBossInterruptSpell(spellId)` and the mob is casting, `InterruptTargetCast`.
- [ ] Tests green; build clean.
- [ ] **Commit:** `feat(combat): disruption spells interrupt a casting mob`

### Task A4: Area life-drain (drain all players in room, heal the mob)

**Files:** Modify `internal/actions/combat_drain.go` (add `ExecuteDrainArea` alongside
`ExecuteDrain`); a new `internal/mobcommands/` entry OR extend `drain.go` to accept an
`area` argument; Test: `internal/actions/combat_drain_test.go`.

- [ ] **Test first:** a mob with N players in room; `ExecuteDrainArea` damages every
  player and heals the mob by the aggregate fraction (mirror the single-target `ExecuteDrain`
  math + the `HarmArea` loop pattern in spell resolution). Assert per-player damage + mob heal.
- [ ] Implement `ExecuteDrainArea(mob, room, ...)` looping `room.GetPlayers()`.
- [ ] Expose to the btree: a `drain_area` btree action (see Task A6 for how btree actions
  are registered) OR reuse the `cast` path if the drain is authored as a spell (decide in
  Chunk C — prefer authoring the drain as a **fold-cast spell** whose effect calls the area
  drain, so it telegraphs for free).
- [ ] Tests green; build clean.
- [ ] **Commit:** `feat(combat): area life-drain (ExecuteDrainArea)`

### Task A5: Push companions out of a room (gear-safe), + `sweep_companions` btree action

**Files:** Modify `internal/hooks/companion_follow.go` (add `PushCompanionsToRoom`
alongside `TransportCompanions` ~:31); `internal/behaviortree/actions_mob.go` +
`actions.go` (register a `sweep_companions` action); Test:
`internal/hooks/companion_follow_test.go`.

- [ ] **Test first:** an owner with live companions in room X; `PushCompanionsToRoom(owner,
  destRoomId)` moves every companion mob instance to destRoom (via the same
  `RemoveMob`/`AddMob` + `RoomId` reassignment `TransportCompanions` uses), **without
  destroying them** — assert the companion mob instances still exist, their `Equipment`/`Items`
  intact, now in destRoom. Interrupt any in-progress companion cast (mirror
  `TransportCompanions`).
- [ ] Implement `PushCompanionsToRoom`. For each player in the boss room, push *their*
  companions to the airlock room id (the action reads the airlock destination from a param).
- [ ] Register a btree action `sweep_companions` (param: `dest_room` id) that, for every
  player in the acting mob's room, calls `PushCompanionsToRoom`. Model registration on the
  existing `spawn_mob`/`summon_companion` actions (`actions_mob.go`, `actions.go`).
- [ ] Tests green; build clean.
- [ ] **Commit:** `feat(companions): PushCompanionsToRoom + sweep_companions btree action`

### Task A6: `target` passthrough on the btree `cast` action (Repair Frame heals boss by name)

**Files:** Modify `internal/behaviortree/actions_combat.go` (`actCast` ~:64).

- [ ] Add an optional `target` param to the `cast` btree action so it emits
  `mob.Command("cast " + spell + " " + target)`. The engine's mob `HelpSingle` targeting
  already resolves a named mob in the room (`internal/actions/cast.go:189`,
  `spell_resolution.go:1043`) — this just lets the add name the boss.
- [ ] A small unit test if the btree action layer is unit-testable; else verify via Chunk B.
- [ ] Build clean.
- [ ] **Commit:** `feat(btree): cast action accepts a target name (mob-to-mob heal)`

### Task A7: Killing the grappler releases its grapple (verify → build if needed)

**Files:** Investigate `internal/combat/grapple.go`, the grapple state teardown, and mob
death handling (`internal/hooks/MobDeath*` / wherever a mob's combat state is cleaned up).

- [ ] **Verify:** when a mob that is grappling a player dies, is the player's grapple/control
  state released automatically? (Check whether grapple state is torn down on the grappler's
  death.) This is the load-bearing counterplay for the Grapnel Warden (§10.2 of the spec:
  killing the grappler MUST free the ally).
- [ ] If NOT automatic: add a hook on mob death that, if the dying mob was grappling/controlling
  a player, releases that player's grapple state. TDD if a seam exists.
- [ ] Build + relevant tests green.
- [ ] **Commit:** `fix(combat): releasing a player's grapple when the grappler dies` (or a
  note in the plan if already automatic).

### Task A8: Chunk-A verification

- [ ] `go build ./...` clean; `go test ./internal/... -count=1` green (note the pre-existing
  grapple flake if it surfaces).

---

## Chunk B — The three add mobs (content)

**IDs:** run `python tools/id_inventory.py --zone crash_site_interior` first; allocate 3
new mob ids in that zone's band. **Filename SOP:** `{id}-{ConvertForFilename(name)}.yaml`.

### Task B1: Repair Frame add

**Files:** Create `_datafiles/world/dogmud/mobs/crash_site_interior/<id>-repair_frame.yaml`
+ its behavior `_datafiles/world/dogmud/behaviors/crash_site_interior/<id>-*.yaml`.

- [ ] Low-statpool construct (species 37, `archetype: fighting`), `hostile: true`. Btree:
  each round (or on a short `cooldown`), `cast <heal-spell> <boss-name>` — a `HelpSingle`
  heal targeting the boss by name (uses Task A6). Pick/author a mob-castable heal spell that
  restores a meaningful boss HP fraction (tuning knob). Loud-ish room text so players notice
  the heal.
- [ ] Boot-smoke (instance wipe + boot; mobs load +1; no panic; kill/verify/clean).
- [ ] **Commit:** `content(crashsite): Repair Frame add (heals the boss)`

### Task B2: Grapnel Warden add

**Files:** Create the mob + behavior YAMLs.

- [ ] Low-statpool construct. Btree: on engage, **grapple a player** (the mob-grapple path
  — model on how mobs initiate grapple; `internal/mobcommands/grapple.go`), with **loud
  `send_room_text`** naming it as the cause ("The Grapnel Warden seizes {target} in a
  crushing lock — break it down to free them!"). Full control-lock per spec §10.2; killing
  it releases the grapple (Task A7).
- [ ] Boot-smoke.
- [ ] **Commit:** `content(crashsite): Grapnel Warden add (locks down a player)`

### Task B3: Hull Sweeper add

**Files:** Create the mob + behavior YAMLs.

- [ ] Low-statpool construct. Btree: on its action, `sweep_companions dest_room: <airlock
  id>` (Task A5) with loud room text ("The Hull Sweeper's field lashes out — your conjured
  allies are hurled from the chamber!"). The airlock room id comes from Chunk C (Task C4);
  until then, use a placeholder to be filled when the airlock exists.
- [ ] Boot-smoke.
- [ ] **Commit:** `content(crashsite): Hull Sweeper add (sweeps companions to the airlock)`

---

## Chunk C — Warden-Prime (the teaching fight) + boss abilities

### Task C1: The core-discharge ability (fold-cast spell)

**Files:** Create `_datafiles/world/dogmud/spells/<slug>.yaml` (e.g. `core-discharge`).

- [ ] A mob-castable party-wide harm spell with **`BaseFolds ≥ 2`** so it auto-telegraphs
  (multi-round windup + per-round in-progress room text; see
  `internal/hooks/NewRound_DoCombat_helpers.go:394` `handleMobFoldCasting`). `HarmArea`-style
  (hits all players in room). Windup + release room text per spec §4.3. Damage = a tuning knob
  (the wipe threat).
- [ ] Boot-smoke (spells load).
- [ ] **Commit:** `content(crashsite): core-discharge boss ability (telegraphed AoE)`

### Task C2: Warden-Prime behavior tree

**Files:** Create `_datafiles/world/dogmud/behaviors/crash_site_interior/9561-warden_prime.yaml`;
update the mob YAML `9561-warden_prime.yaml` to reference it / drop `aiprofile: brute`.

- [ ] Btree (model on Old Edrin): on a **fixed round timer** (`round_mod`), cast
  `core-discharge` (telegraphed by its folds). At an HP threshold (`mob_health_below`), a
  one-shot (`state_equals`/`set_state`) `spawn_mob`/`summon_companion` of the **Repair Frame**
  (Task B1). No drain, no grapple, no sweep — the teaching version.
- [ ] Boot-smoke (behaviors load; no panic).
- [ ] **Commit:** `content(crashsite): Warden-Prime btree (teaching: discharge + repair add)`

### Task C3: Harness-verify the Warden-Prime mechanic

- [ ] Using the 3-Meirok harness rig (accounts quester4/5/6 → Vael/Ryn/Doss; playbook in
  `tools/playtest/reports/2026-07-05-crashsite*`): enter #22, reach Warden-Prime, and confirm
  LIVE: the discharge telegraphs; a **flashbang throw** cancels it; a **neural-stun** cancels
  it; a **melee bash does NOT**; the Repair Frame spawns + heals + can be killed to stop the
  heal. Report the observed behavior. (Wipe stale instance saves first; drive movement via the
  leader.)

---

## Chunk D — The Core Guardian (full fight) + airlock room

### Task D1: The airlock room

**Files:** Create `_datafiles/world/dogmud/rooms/crash_site_interior/<id>.yaml`; wire an exit
so it sits **on the natural approach path** to the Core Guardian's room (6395) — adjacent, in
the travel direction (spec §10.1). Non-hostile, no spawns. Run `cartcheck`/boot to confirm
map consistency (or `non_cartesian` already covers the zone).

- [ ] Boot-smoke (rooms load; zone consistency OK).
- [ ] Backfill the Hull Sweeper's `dest_room` (Task B3) with this room id.
- [ ] **Commit:** `content(crashsite): airlock room for swept companions`

### Task D2: The core-drain ability (fold-cast spell → area drain + heal + charge)

**Files:** Create `_datafiles/world/dogmud/spells/<slug>.yaml` (e.g. `core-drain`).

- [ ] A mob-castable **`BaseFolds ≥ 2`** telegraphed ability whose effect performs the **area
  life-drain** (Task A4 `ExecuteDrainArea`) — damages all players + heals the Guardian. Windup
  + resolve room text per spec §4.2. The **Core Charge increment** on resolution is driven by
  the btree (Task D3), not the spell.
- [ ] Boot-smoke.
- [ ] **Commit:** `content(crashsite): core-drain boss ability (telegraphed area drain)`

### Task D3: The Core Guardian behavior tree

**Files:** Create `_datafiles/world/dogmud/behaviors/crash_site_interior/9562-the_core_guardian.yaml`;
update the mob YAML.

- [ ] Btree orchestrating the §4 loop:
  - **Core Charge** = a `BehaviorState` counter (`set_state`/`increment_state`/
    `state_greater_than`).
  - **Drain cadence:** on a `round_mod`/`cooldown` timer, cast `core-drain` (telegraphed);
    after it resolves, `increment_state` Core Charge. (Detect resolution via the fold-cast
    completion — reference `handleMobFoldCasting`; if btree can't observe completion directly,
    increment on the same timer one cycle later — decide at build time and document.)
  - **Discharge:** when `state_greater_than` Core Charge ≥ threshold, cast `core-discharge`
    (Task C1); on resolution, reset Core Charge to 0.
  - **Interrupt handling:** an interrupted drain or discharge (Chunk A) cancels the cast; the
    btree must reset Core Charge on an interrupted discharge (decisive interrupts, spec §10.4).
    Confirm the interrupt path leaves the mob in a state the btree reads as "no pending release"
    and resets charge (may need a small `set_state` on the cast-cancel path — flag at build).
  - **Adds:** at HP thresholds / timers, `spawn_mob` the Repair Frame, Grapnel Warden, and —
    gated on companions being present — the Hull Sweeper (Chunk B).
- [ ] Boot-smoke (behaviors load; no panic).
- [ ] **Commit:** `content(crashsite): Core Guardian btree (charge/drain/discharge + 3 adds)`

### Task D4: Harness-verify the full Core Guardian fight

- [ ] 3-Meirok rig: reach the Core Guardian under **natural attrition** (no teleport), confirm
  LIVE every mechanic fires and is counterable: drain telegraphs + is interruptible; charge
  builds + discharge threatens; interrupts are decisive (charge resets); Repair Frame heals +
  is killable; Grapnel Warden locks a player loudly + dies to focus-fire to free them; Hull
  Sweeper relocates companions to the airlock **with gear intact** + they can be recovered.
  Record the fight as a party experiences it (rounds, downs, interrupt-tool use, win/wipe).

---

## Chunk E — Calibration

### Task E1: Tune to the razor's-edge target

- [ ] With all mechanics live, run the 3-Meirok party at the **450g buy-in** and tune the §9
  knobs (add cadence/statpools, Repair heal rate, grapple duration, Sweeper trigger, drain
  magnitude, Core Charge threshold, discharge damage, boss statpool coefficient) so #22 lands
  the target: **a close wipe or a down-to-the-wire win.** Iterate: adjust knobs, re-run,
  measure. Document the final values + the run evidence in a report under
  `tools/playtest/reports/`.
- [ ] Final full-suite + boot-smoke; update the spec's §9 with the tuned values.
- [ ] **Commit:** `content(crashsite): boss mechanics calibration pass (tuned to target)`

---

## Notes / build-time decisions to resolve
- **Fold-cast completion → btree state:** confirm how the btree observes a mob fold-cast
  finishing (to increment/reset Core Charge). If not directly observable, drive the counter on
  the same timer with a one-cycle offset and document.
- **Interrupted-cast → charge reset:** verify the cast-cancel path (`TriggerCastCancel`) can be
  observed by the btree or set a state flag on it; the discharge reset (spec §10.4) depends on it.
- **Repair Frame heal spell:** reuse an existing mob-castable heal if one fits; else author a
  minimal one.
- **Grapnel loudness:** ensure the grapple room text is unmissable and clearly attributes the
  lock to the Warden (the whole counterplay depends on players reading it).
