# Crash Site Boss Mechanics — Design Spec

**Date:** 2026-07-06
**Status:** Draft for review
**Author:** (design collaboration)

**Goal:** Turn the Crash Site (#22) bosses — Warden-Prime (9561) and the Core
Guardian (9562) — from featureless stat-brutes into interesting, counterplay-rich
encounters that are a genuine razor's-edge finale for a geared 3-player party,
without inflating numbers that don't bite.

---

## 1. Why: the calibration finding

A geared 3-player party (three clones of the prod character *Meirok* — skills
~50–69, full masterwork elemental set, triple drowned-claws, healer/summoner kit)
was run against #22 at a **450g buy-in** via the harness. With *competent play*
(area-healing, moving through hazard corridors without lingering) the result was
an unambiguous **faceroll**:

- Party never dropped below **77% HP** the entire run.
- Arrived at the Core Guardian at **92% HP/CP**, killed it in ~15–20 rounds, and
  *finished at the same 92%* — near-zero resources spent, zero downs.

Two root causes, neither fixable by raising statpool:

1. **The companion wall.** Each Meirok fields up to ~3 companions (2 flesh golems
   + 1 spirit wolf) → a "3-player party" is really a **~12-combatant army**. A
   single golem tanked the Core Guardian to 6% HP while no player took meaningful
   damage. Companions absorb the entire encounter.
2. **The bosses are featureless brutes** (`aiprofile: brute`, no abilities). Against
   a geared *melee* party whose physical claws bypass the zone's spell-suppression,
   a brute is a DPS race the party always wins. More statpool → a *longer* faceroll,
   not a more dangerous one; the 3150-statpool Guardian's offense never threatened
   anyone.

**Design conclusion:** the fix is *mechanics and counterplay*, plus a gear-safe way
to neutralize the companion wall — not bigger numbers.

---

## 2. Design principles

- **Counterplay over lethality.** Every threat should be something the party can
  *react to or prepare for* (a telegraph, a kill-priority, a saved tool) — decisions,
  not unavoidable damage. MUD combat is round-based text, so "counterplay" means
  interrupts, kill-priority, positioning between rooms, and resource budgeting — not
  twitch-dodging.
- **Gear is sacred.** Companions carry equipped gear. **No mechanic may destroy a
  companion** (that would delete its gear). Companion removal is *relocation*, always.
- **Reuse the zone's signature.** The Crash Site already suppresses spell-damage and
  mutation-damage (`hull_suppression`, factor 0.35). The boss design should make that
  suppression *matter as a dilemma*, not just a nerf (see §7).
- **Teach then test.** Warden-Prime is the teaching fight (one add + the core
  mechanic); the Core Guardian is the full exam.
- **Tune last.** Land the mechanics first; adjust statpool/buy-in/magnitudes only
  after the shape is proven against the harness party.

---

## 3. The companion problem — the Hull Sweeper (gear-safe)

**Decision:** the companion wall is handled by an *add with a role* — the **Hull
Sweeper** — not by a blanket always-on companion debuff. Rationale: relocation is
gear-safe by construction (a companion's gear lives on its mob instance and survives
a room move), and an add gives *counterplay* (kill it before it sweeps; re-summon or
recall after) instead of a flat nerf.

- The zone's existing **spell/mutation suppression stays as-is** (it's the zone's
  signature and drives the §7 dilemma).
- There is **no** always-on companion stat-debuff. If, after tuning, the Sweeper
  alone proves insufficient, a light companion debuff can be revisited — but it is
  **out of scope for v1**.

**Hull Sweeper behavior:** when the boss activates a Sweeper (see §4), on its action
it **pushes all of the party's summoned companions out of the boss room** into a
designated adjacent "airlock" room — alive, gear intact. The players must then fight
the boss directly for a window, kill the Sweeper, and re-summon / walk their
companions back. This removes the tank wall *temporarily and reversibly*.

---

## 4. The Core Guardian encounter (the fight loop)

The Guardian carries a **Core Charge** resource (a per-mob `BehaviorState` counter,
starts 0). The fight is a loop of *pressure → charge → release*, with the party
juggling several simultaneous demands.

### 4.1 Adds — three roles

The Guardian activates dormant construct adds at HP thresholds / on cooldown timers.
Each is a distinct new mob (new mob-ids in `crash_site_interior/`), authored light
(low statpool) — they're *jobs to manage*, not damage sponges:

| Add | Role | Counterplay |
|---|---|---|
| **Repair Frame** | Channels a repair (heal) on the Guardian each round it lives. | Peel and kill it fast, or the boss out-heals the party's DPS and the fight never ends. |
| **Grapnel Warden** | Grapples and locks down one player (that player is controlled — can't freely act — until freed). | Free the grappled member (break the grapple) or accept reduced party DPS/interrupt capacity. |
| **Hull Sweeper** | Sweeps the party's companions out of the room (§3). | Kill it before it sweeps; re-summon/recall afterward. Activated when companions are present. |

Spawn cadence is a tuning knob (§9): e.g. Repair Frame at 75%/40% HP, Grapnel Warden
on a round timer, Hull Sweeper when companion count in the room exceeds a threshold.

### 4.2 Life-drain recharge (telegraphed)

Periodically the Guardian **draws the party's vitality to feed its core**. This is a
**telegraphed cast** (a 2-round windup, so it announces round N and resolves round
N+1):

- *Windup (round N):* room text — e.g. "The Core Guardian's chest-cavity irises open;
  filaments of cold light reach toward you. It will feed on the room next round."
- *Resolve (round N+1):* a **party-wide drain** — damages every player in the room and
  **heals the Guardian + increments Core Charge**.

**Counterplay (a resource-budget decision):** the drain windup is an interruptible cast
(§6). The party can spend a *constrained interrupt* (grenade / disruption spell) to
cancel it — denying the charge and the heal — **or save that interrupt for the
discharge (§4.3)**. They rarely have enough interrupts for both. That tension is the
point.

### 4.3 Core discharge (telegraphed, constrained interrupt)

When **Core Charge reaches its threshold** (e.g. 3), the Guardian begins charging its
signature attack — a **multi-round fold-cast**, which the engine already renders as a
telegraph with per-round in-progress messaging:

- *Windup:* "Its core flares white-hot and the air screams — full discharge imminent."
- *Resolve (if not interrupted):* a **massive party-wide discharge** — a potential
  wipe if the party is depleted. Then Core Charge resets.

**Constrained interrupt (the core counterplay):** the discharge windup can be cancelled
**only** by a designated disruptor — a **thrown grenade** (e.g. flashbang) or a
**specific disruption spell** (neural-stun / sensory-overload / kinetic-shove) landing
on the Guardian during the windup. **Generic melee knockdown / bash does NOT interrupt
it** — this is deliberate: it forces *preparation* (bring grenades, save the spell +
CP) rather than incidental CC. A successful interrupt cancels the discharge and resets
Core Charge.

### 4.4 The loop, together

Adds apply constant pressure (kill-priority + a locked player + a stripped pet wall) →
the drain feeds Core Charge on a telegraphed cadence (interrupt it or let it build) →
at full charge the discharge threatens a wipe (interrupt with the *right* tool or eat
it). The party must simultaneously: manage the right add first, free grappled members,
survive with companions periodically gone, and **budget a limited pool of interrupts
between the drains and the discharge.** That is coordination and counterplay, not a
stat race.

---

## 5. Warden-Prime — the teaching fight

Warden-Prime introduces the vocabulary before the Guardian throws everything at once:

- The **core-discharge mechanic** (telegraphed, constrained interrupt) — but on a
  simpler *fixed round-timer* charge rather than a drain-fed one.
- **One add:** the **Repair Frame** — teaching "kill the healer-add or the fight
  stalls."
- No Grapnel Warden, no Hull Sweeper, no life-drain.

A party that clears Warden-Prime has learned: *watch for the telegraph, keep an
interrupt in reserve, and kill the repair add.* The Guardian then demands all of it
at once, under attrition, with the pet wall periodically gone.

---

## 6. The constrained-interrupt toolset

Interrupts are the spine of the counterplay, so what counts is explicit and tunable:

- **Thrown disruptors:** a configurable allowlist of item-ids (v1: **flashbang**;
  Meirok already knows the flashbang recipe, and it's a natural "bring grenades to the
  robot dungeon" fantasy). Extendable to other thrown disruptors later.
- **Disruption spells:** a configurable allowlist of spell-ids (v1: **neural-stun,
  sensory-overload, kinetic-shove**).
- **Explicitly NOT interrupts:** melee bash/kick/trip/generic knockdown. (They remain
  useful for the *adds* — e.g. peeling the Grapnel Warden — but not for the boss's
  telegraphed casts.)

Both allowlists are config/data-driven so the encounter (and future ones) can tune what
qualifies without code changes.

---

## 7. The suppression tie-in (utility-casting counterplay)

The Crash Site suppresses **spell *damage*** and **mutation *damage*** (×0.35). It does
**not** suppress *utility* effects. The interrupt spells (neural-stun / sensory-overload
/ kinetic-shove) are utility/disruption, not damage — **so they still work at full
effect under suppression.**

This produces the intended dilemma for free: in the Crash Site your *damage* spells are
feeble, but your *disruption* spells are your key counterplay tool. The zone's signature
mechanic pushes casters toward exactly the utility-casting the boss fight demands. No
extra mechanic needed — just author the interrupt allowlist around utility spells and
let the existing suppression create the tension.

*(The earlier-discussed melee-retaliation "melee vs cast" dilemma is dropped in favor of
this cleaner, already-supported version.)*

---

## 8. Engine primitives & gaps (grounded in recon)

None of this needs new architecture. Precedent boss to model structurally:
`_datafiles/world/dogmud/behaviors/marches_spur_road/275-old_edrin.yaml` (multi-phase,
summons named adds, `state_equals`/`set_state` phase gates, `cooldown` decorator,
`send_room_text` narration).

| Mechanic | Status | Reuse / gap (file:line) |
|---|---|---|
| Mid-fight add summoning | **EXISTS** | btree `spawn_mob` (`internal/behaviortree/actions_mob.go:26`) / `summon_companion` (`:51`); Old Edrin precedent. Author adds as new mob-ids. |
| Telegraphed / charging attack | **PARTIAL** | Reuse multi-round **fold-casting** — a mob ability with `BaseFolds ≥ 2` auto-telegraphs (`internal/hooks/NewRound_DoCombat_helpers.go:394` `handleMobFoldCasting`, `combat_shared_helpers.go:448` `processFoldRound`). Gap: a *deterministic* cancel hook (§ interrupt). |
| Deterministic interrupt (cancel a mob cast) | **EXISTS (fn), PARTIAL (wiring)** | `internal/actions/cast_interrupt.go:14` `InterruptTargetCast` is caller-agnostic (works on a mob). Gap = new call sites: a thrown-item check in `internal/usercommands/throw.go` (after hit, if item ∈ disruptor allowlist and `mob.Character.Activity.IsCasting()` → interrupt) and a spell check in `internal/hooks/spell_resolution.go:249` `resolveAgainstMob` (if spellId ∈ disruptor allowlist → interrupt). |
| Mob stun that skips its round | **EXISTS** | Buff 84 "Stunned" + `NoCombat` flag (`internal/hooks/NewRound_DoCombat.go:236`). Note: a telegraphed release must explicitly check interrupt/cancel state at fire time (delayed closures fire on their own timer). |
| Mob-to-mob heal (Repair Frame → boss) | **EXISTS** | Mob `HelpSingle` cast targets another named mob (`internal/actions/cast.go:189`, resolve `spell_resolution.go:1043`). Small gap: pass a `target` param through the btree `cast` action (`internal/behaviortree/actions_combat.go:64`). |
| Mob grapples a player (Grapnel Warden) | **EXISTS** | `internal/combat/grapple.go:40` `AttemptGrapple` is symmetric; mob entry `internal/mobcommands/grapple.go`. Player locked (flee vetoed; position FSM auto-escape only at z ≤ −2.0). Mature, no engine work. |
| Companion data model + gear-safe relocation (Hull Sweeper) | **EXISTS (model), PARTIAL (variant)** | `Character.Companions []CompanionInfo` (`internal/characters/companions.go:53`) — a list; gear snapshotted on the struct (survives moves). Relocation precedent `internal/hooks/companion_follow.go:31` `TransportCompanions` moves companions room→room without destroying them. Gap: an *inverse* variant ("push companions to airlock room X" rather than "follow owner"). |
| Party-wide life-drain + self-heal (recharge) | **EXISTS (single), PARTIAL (AoE)** | `internal/actions/combat_drain.go` `ExecuteDrain` (single-target lifesteal). Gap: an AoE variant looping `room.GetPlayers()` (mirror the `HarmArea` pattern). |
| Per-mob "Core Charge" counter | **EXISTS** | btree `BehaviorState` (`set_state`/`increment_state`/`state_greater_than`, `internal/behaviortree/context.md:547`) — purpose-built for phase/counter tracking, YAML-driven. |
| HP-threshold / round-timer phase gates | **EXISTS** | `mob_health_below`, `round_mod` conditions; `cooldown` decorator (`internal/behaviortree/conditions.go:17`). |

**Net new Go work (all small, localized):**
1. Interrupt call sites in `throw.go` + `spell_resolution.go` (`resolveAgainstMob`) keyed
   off configurable disruptor allowlists (item-ids + spell-ids).
2. An inverse companion-relocation function alongside `TransportCompanions` (push to a
   designated room).
3. An AoE variant of `ExecuteDrain` (drain all players in room, heal the mob).
4. A `target` passthrough param on the btree `cast` action (for the Repair Frame).

Everything else is **content**: two boss behavior-tree files, three add mob YAMLs, one
boss "discharge"/"drain" ability definition (fold-cast spell(s)), the airlock room, and
config for the disruptor allowlists + tuning knobs.

---

## 9. Tuning knobs & calibration

All numeric, adjusted against the 3-Meirok harness party (see §11) to hit the target:
**a close wipe or a down-to-the-wire win for #22.**

- Add spawn cadence (HP thresholds / round timers) and per-add statpool.
- Repair Frame heal-per-round (vs party DPS).
- Grapnel Warden grapple strength / duration.
- Hull Sweeper trigger threshold (companion count) + cooldown.
- Life-drain magnitude (party damage + Guardian heal + charge gained).
- Core Charge threshold for discharge.
- Discharge damage (the wipe threat).
- Interrupt budget implied by drain cadence vs discharge threshold (how many interrupts
  the fight demands vs. how many a party can bring).
- Boss statpool coefficient / buy-in (last-resort fine adjustment).

---

## 10. Open questions / decisions to confirm

1. **Airlock room for the Sweeper:** reuse an existing adjacent Crash Site room, or add
   a dedicated dead-end "airlock" room the companions get shoved into? (Leaning: a
   dedicated non-hostile stub so swept pets don't wander into another fight.)
2. **Grapnel Warden lockdown severity:** full control-lock (player can do nothing but
   struggle) vs. partial (can act at reduced effect)? Full is punchier but riskier if
   the party has no escape tool — confirm the intended harshness.
3. **Interrupt allowlist v1 contents:** confirm flashbang (item) + neural-stun /
   sensory-overload / kinetic-shove (spells). Any others?
4. **Does an interrupted discharge fully reset Core Charge, or only partially?** (Full
   reset = interrupts are decisive; partial = the party is always racing the charge.)

---

## 11. Testing

The existing harness rig is the calibration instrument: three geared Meirok clones
(accounts quester4/5/6 → Vael/Ryn/Doss, `isai`+admin), driven concurrently via
`mudagent` adapters, forming a party and entering #22 at a chosen buy-in. Per encounter
iteration:

- Verify each mechanic fires and is *counterable* (the interrupt cancels; killing the
  Repair Frame stops the heal; the Sweeper relocates pets without gear loss; the grapple
  locks and can be broken).
- Measure the boss fight the way a real party experiences it: **reach the Guardian under
  natural attrition (no teleport), companions periodically stripped**, and read
  rounds-to-kill, downs, lowest party HP, interrupt-tool consumption, and win/wipe.
- Tune §9 knobs to the razor's-edge target.

Process notes from prior runs: wipe stale instance saves between runs; drive party
movement through the leader only (followers double-move otherwise); instance rooms use
synthetic runtime ids (navigate by description); portal entry keyword was `crash-<n>`.

---

## 12. Staging (for the implementation plan)

Rough buildable chunks, each independently testable:

1. **Engine glue** (the 4 small Go items in §8) + config for the disruptor allowlists,
   with unit tests (interrupt-on-throw, interrupt-on-spell, AoE drain, companion-push).
2. **The adds** (3 add mob YAMLs) + the mob-to-mob heal wiring — testable via a stub
   boss that just spawns them.
3. **Warden-Prime** (teaching btree: fixed-timer discharge + Repair Frame) — the first
   full playable mechanic; harness-verify the telegraph + constrained interrupt.
4. **The Core Guardian** (full btree: Core Charge, drain-fed recharge, all three adds,
   discharge) + the airlock room.
5. **Calibration pass** against the 3-Meirok party; tune §9 to target.

Each chunk gets its own plan task; content chunks follow the ID-inventory + boot-smoke
SOP; the engine chunk follows TDD.
