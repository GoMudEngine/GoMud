# Mob Aliveness 5.2 — Bounty Hunting (Design)

**Date:** 2026-05-30
**Chunk:** 5.2 (Cross-cut) — Bounty hunting
**Status:** Design — pending user review
**Depends on:** 1.4 (knowledge), 1.5 (bounty state), 2.8 (track/scan), 4.1–4.4
(goals + planners), 5.1 (justice: crime/bounty/jail substrate)

---

## Summary

Two complementary halves, one combined chunk:

- **Half A — NPCs hunt wanted players (the headline).** When a player's bounty
  climbs high enough, a tough, well-geared **bounty-hunter NPC** is dispatched,
  travels in, and relentlessly tracks the player down to claim the bounty by
  killing them. Pressure you can't simply hide from. Outs: clear the bounty
  through justice (pay fine / serve time — which also covers the
  *intentionally-get-arrested* escape), or kill the hunter for a temporary
  reprieve (the bounty stands; another comes).

- **Half B — players claim bounties on NPCs.** Authored standing kill-bounties
  on notable hostile NPCs (e.g. the Chrysalis Phantom) that **players** collect
  for the reward, via the existing bounty board / `bounty list` / mob-death
  claim path.

**Not PvP.** An NPC hunting a player is NPC-vs-player combat, which the game
does everywhere. No player-vs-player is introduced.

This is purely a *composition* chunk: it adds a dispatch manager, one goal type
+ planner, one archetype, a gear kit, and authored content — on top of
already-shipped substrate.

---

## Component summary

| Component | What |
|-----------|------|
| Dispatch manager | Detects over-threshold player bounties, spawns/scales/telegraphs one hunter per player, owns the hunt lifecycle (jailed-suspend, call-off, re-dispatch). |
| `hunt_bounty_target` goal + planner | Drives pursuit: pathfind toward target, close in over rounds, engage on contact; **holds while the target is jailed**. |
| `bounty_hunter` archetype | Combat-capable mob tree: standard combat cascade + `try_goal_planner`. |
| Hunter gear kit | ~6 new dedicated items worn by the hunter; ~4% independent per-piece drop. |
| Claim on player death | Extend `PlayerDeath_BountyResolve` so a hunter mob claims + clears the faction's crime record (death pays the debt). |
| Half B content | Standing-bounty seed loader + authored NPC bounties + verify the player claim/board path. |
| Config | Threshold, statpool scaling, re-path cadence, re-dispatch cooldown. |

---

## 1. Architecture

Three cooperating pieces plus Half-B content:

1. **Dispatch manager** (new; justice-adjacent — either `internal/bountyhunter`
   or a section of `internal/justice`). Owns *when a hunter exists*: trigger
   detection, spawn, scaling, telegraph, one-per-player registry, lifecycle.
2. **Strategic layer** — a `hunt_bounty_target` goal type (catalog) + a planner
   (`internal/planners/`). Owns *how the hunter pursues*: locate → close in →
   engage, with a jailed-target hold.
3. **`bounty_hunter` archetype** (`_datafiles/world/dogmud/behaviors/
   archetypes/bounty_hunter.yaml`) — the tree that runs combat + dispatches the
   planner via `try_goal_planner`. Modeled on `scout` / `predator` /
   `guard_captain`.

The dispatch manager is the new "brain"; the pursuit reuses the 4.x goal/planner
seam and 2.8 track/move primitives; combat reuses the existing cascade.

---

## 2. Dispatch manager (Half A "brain")

**Trigger.** A hunter is dispatched when **any single open faction bounty
against a player has `GoldReward ≥ BountyHunterGoldThreshold`** and that player
has **no active hunter**. (Single-bounty, not summed — chosen for a clean
per-faction model.) One active hunter per player, always.

**Spawn.** Instantiate a `bounty_hunter` mob (fresh instance) at the **issuing
faction's seat** — a faction → home-room lookup reusing/extending the justice
faction-room registry (Thornwall barracks 473, Stillwater constabulary 4110).
Stamp the **target userId** and **triggering bounty id** onto the hunter
(MiscData + the goal's params). The hunter's statpool is set from the bounty
(see §6). Equip the gear kit (§5).

**Telegraph.** The targeted player receives a system line when a hunter is
dispatched, e.g. *"Word reaches you that a hunter has taken the contract on your
head…"* (Descriptive, no raw numbers.)

**Lifecycle / termination.** The hunt ends and the hunter despawns when any of:
- **Bounty no longer open** — the player paid the fine / served a sentence / it
  expired or was withdrawn. *(This is the intentional-arrest escape: serving or
  paying withdraws the faction bounty → the manager calls off the hunt.)*
- **Hunter claims the kill** (§4).
- **Player kills the hunter** — reprieve; a new hunter may be dispatched only
  after `BountyHunterRedispatchCooldown` rounds, and only if a bounty is still
  ≥ threshold.

**Suspension.** If the target is **jailed** (see §3/§7) the hunt is suspended
(no pursuit) — and since serving/paying withdraws the bounty, the manager then
calls it off. If the **player is offline**, the hunt is suspended (no pursuit of
an absent player) and resumes on return if still valid.

**Cadence.** Trigger detection runs on bounty-declare plus a periodic sweep
(cheap: iterate open bounties, check threshold + active-hunter registry). The
manager never drives movement itself — that is the planner's job.

---

## 3. `hunt_bounty_target` goal + planner (pursuit)

**Goal** (new catalog type in `internal/goals/catalog/`):
- Params: target subject (player userId) + triggering bounty id (stamped at
  dispatch).
- Predicate (still valid?): the bounty is still open AND the target is online.
- Context-score: high/constant — this is the hunter's reason to exist; it
  dominates the hunter's goal set. Conflicts with `survival` (hunter presses on
  rather than fleeing at moderate HP; still allowed to panic-flee at the
  archetype's hard floor).

**Planner** (`internal/planners/hunt_bounty_target.go`, per-tick, returns a
Command + Status):
1. **Jailed-target hold.** If the target is jailed
   (`HasBuffFlag(buffs.Jailed)` or jail MiscData present): return a *hold* —
   loiter; do **not** path into the cell, do **not** attempt to engage. (The
   manager will call the hunt off once serving/paying clears the bounty.)
2. **Same room as target:** engage — `attack @<uid>` (subject to §7 safety).
3. **Else (pursue):** pathfind toward the target's current room and step ~1
   room/tick (a closing chase, not a teleport), re-pathing every
   `BountyHunterRepathRounds`; cross-zone allowed (reuse the `pathto` /
   `move_toward` plumbing the caravans/scout use). `try_scan` / `try_track`
   add flavor and handle a hidden target on the final approach.

The planner reads the engine's authoritative player-room (omniscient closing
pursuit, per design) — you gain ground by moving but cannot simply hide. The
future disguise-kit system (out of scope) is what will later break the track.

---

## 4. Claim on player death (death pays the debt)

Extend `internal/hooks/PlayerDeath_BountyResolve.go` so a **bounty-hunter mob**
is a valid claimer (today: guard-faction mob, killer player, or highest-damager
player). When the hunter lands the killing blow on its contract target:
- `bounties.TryClaim` the triggering bounty (status → claimed).
- **Clear the player's record with that faction** exactly as
  `justice.ResolveDetention` does on a served sentence: resolve that faction's
  (and allies') unresolved crimes against the player, withdraw the faction's
  open bounties, reset rep to the floor if below it.
- The hunter despawns; the manager ends the hunt.

Net: a hunter-death and a served sentence both clear your record with that
faction. (Player death here is the normal respawn-with-setback, not
permadeath — consistent with the rest of the game.)

---

## 5. Hunter gear kit

The hunter wears a **full dedicated kit** so it is mechanically formidable
(weapon `damage_multiplier`, armor mitigation) — not merely high-statpool.

- **Items:** ~6 new items forming a cohesive "bounty hunter" set — weapon +
  body + head + legs + feet + gloves — at a high `rarity_tier`. Stat budget /
  tier chosen to suit a 400–500-statpool elite (above town-guard gear).
- **Worn:** assigned via the hunter template's `equipment:` block (all hunters
  share the kit; difficulty scaling lives on the statpool, §6). A single kit in
  v1; tiered-by-bounty kits are a future refinement.
- **Drops:** governed by the standard equipped-item drop path
  (`hooks/Death_MobLoot.go`), which rolls **each equipped piece independently**
  against the mob's `itemdropchance`. Set `itemdropchance: 4` → ~4% per piece.
  Across ~6 pieces that is ~0.25 pieces per kill — a rare trophy, not a farm
  (and you only face hunters by being repeatedly wanted, which is costly). The
  hunter does **not** carry the `PermaGear` flag (that would drop nothing).
- `loot_pool` (instance-loot generation) is **not** used — the worn-gear drop
  path covers it.

---

## 6. Config knobs + worked scaling examples

A player's bounty power base = the sum of their six stat `ValueAdj` values
(≈600 for a baseline player, higher when developed). Per 5.1b:
`bounty gold = statpool × BountyGoldDefaultMultiplier(0.5) ×
max(JusticeBountyMurderMult(2.0), repMult≤2.0)`.

| Knob | Default | Meaning |
|---|---|---|
| `BountyHunterGoldThreshold` | **500** | A single open bounty ≥ this dispatches a hunter (≈ a murder-tier bounty). |
| `BountyHunterBaseStatpool` | 250 | Base of the hunter's scaled statpool. |
| `BountyHunterStatpoolPerGold` | 0.25 | Statpool added per gold of the triggering bounty. |
| `BountyHunterMinStatpool` / `MaxStatpool` | 300 / 500 | Clamp. |
| `BountyHunterRepathRounds` | 5 | Re-path cadence (closing chase). |
| `BountyHunterRedispatchCooldown` | 500 | Rounds before a new hunter after one is killed. |
| Hunter `itemdropchance` (data) | 4 | ~4% independent per-equipped-piece drop. |

Hunter statpool = `clamp(250 + gold × 0.25, 300, 500)`. Reference points:
city/gate guard statpool 240, Guard Captain Velk 300.

| Player & crime | Power (stat-sum) | Bounty gold | Hunter? | Hunter statpool | Feel |
|---|---|---|---|---|---|
| Assault, mild | ~600 | ~300 | No (< 500) | — | below threshold |
| **Single murder, baseline player** | ~600 | **~600** | Yes | **400** | elite — above Velk (300) |
| Murder, developed player | ~850 | ~850 | Yes | ~460 | near-apex |
| Heinous / very strong / multi-Hostile | ~900+ | ~1000+ | Yes | **500 (cap)** | apex "oh no" |

So a town guard is 240, a captain 300, and a dispatched hunter runs **400–500**
plus a real gear kit — a deliberate step above the watch, scaling with how
wanted (and how strong) the offender is. All values tunable.

---

## 7. Jailed-target safety (belt-and-suspenders, reusing 5.1c)

A player who flees into town and gets themselves arrested must be safe from the
hunter in the cell. Three layers:

1. **Planner hold (§3.1):** the hunter never even tries to enter the cell or
   attack a jailed target.
2. **`no-aggro-target` safety net:** the 5.1c Jailed buff already carries the
   `no-aggro-target` flag, making a jailed player invisible to *all* mob aggro
   paths (the BUG-04 fix that stopped guards piling on in the cell). So even if
   a hunter is mid-combat when the arrest lands, its aggro drops — killing in
   the cell is impossible at the combat layer regardless of the planner.
3. **Cell topology:** the cell is recall-blocked and reached only through the
   guarded station; the hunter gains no path advantage there.

"Sprint to town and get arrested" is therefore a legitimate, designed escape:
you take a sentence/fine (which clears the bounty and calls off the hunt)
instead of a hunter's blade.

---

## 8. Half B — player-claimable NPC bounties

- **Standing-bounty seed.** A committed data file
  (`_datafiles/world/dogmud/bounties.standing.yaml`) + a boot loader declares
  kill-bounties on a handful of notable hostile NPCs (the Chrysalis Phantom and
  a few bandit leaders), **idempotently** — re-declare only if no open bounty
  for that target already exists (the runtime bounty registry is
  gitignored/persistent, so the loader must not duplicate on every boot).
  Issuer = an appropriate faction.
- **Claim path (already exists).** Players discover via `bounty list` and the
  physical bounty boards (Thornwall 473, Stillwater 4110), and claim by killing
  the target — `hooks/MobDeath_BountyClaim.go` already credits the
  highest-damager player with gold + faction rep. This half is verification +
  light flavor polish; **no new claim plumbing**.

---

## 9. Out of scope / future

- **Disguise-kit evasion** — a future skullduggery skill + tailoring recipes +
  gold sink, the intended counter to relentless pursuit. (Designing the hunter
  as omniscient-pursuit now is deliberate: disguises are what will later break
  the track.)
- **Criminal NPCs that commit witnessable crimes and then get hunted**
  (NPC-vs-NPC bounty hunting) — a noted followup, likely once more zones land.
- **Tiered hunter gear kits** by bounty band (v1 ships one kit).
- **Bounty-board "pick up a contract"** UX for player hunters; NPC-bandit
  auto-bounty declaration; squad hunters.

---

## 10. Testing

**Unit:**
- Dispatch trigger: single open bounty ≥ threshold dispatches; sub-threshold
  does not; one-per-player (no second hunter while one is active);
  re-dispatch only after cooldown.
- Statpool scaling: the §6 table rows (`clamp(250 + gold×0.25, 300, 500)`).
- Planner: jailed-target → hold (no engage, no path-in); same-room → engage;
  else → closing step toward target.
- Claim-on-death: hunter kill resolves the bounty AND clears that faction's
  crimes/rep (mirror the `ResolveDetention` clearing).
- Standing-bounty seed: idempotent (no duplicate open bounty across two loads).

**Boot smoke (instance-wipe per SOP):** `bounty_hunter` archetype +
`hunt_bounty_target` goal/planner register; the ~6 hunter items + the
standing-bounty seed load; server boots clean past data-file load.

**In-game smoke (deferred to user):**
1. Commit a murder → telegraph fires → hunter travels in → closing chase →
   it catches and kills you → record clears with that faction.
2. Trigger a hunter → kill it → reprieve → confirm re-dispatch after cooldown.
3. Trigger a hunter → flee to town → get arrested → confirm the hunter cannot
   reach/kill you in the cell, and serving/paying calls off the hunt.
4. As a player, read a standing NPC bounty on a board, kill the target, receive
   the reward.
