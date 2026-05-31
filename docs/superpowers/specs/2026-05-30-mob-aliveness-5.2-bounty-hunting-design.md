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
| Config | Threshold, statpool scaling, gear divisor, re-dispatch cooldown. |

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

**Spawn.** Instantiate a `bounty_hunter` mob at the **issuing faction's seat** —
a faction → home-room lookup reusing/extending the justice faction-room registry
(Thornwall barracks 473, Stillwater constabulary 4110). Use
`mobs.NewMobByIdFresh(id, homeRoom, forceStatPool)` with the **scaled statpool**
(§6) — `NewMobById`/`NewMobByIdFresh` already accept a `forceStatPool` override,
so no engine change is needed. Then generate + wear the affix-scaled gear kit
(§5). Stamp the **target userId** and **triggering bounty id** onto the hunter
**instance's MiscData** (`bh_target_user_id`, `bh_bounty_id`) — this is the
per-instance target the planner reads (§3). Add the param-less
`hunt_bounty_target` goal to the template so `try_goal_planner` fires.

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

**Target lives on the hunter instance, not the goal.** The goals store is keyed
by mob *template* id, so all live hunters of the hunter template share one goal
record — a goal *parameter* cannot describe a different target per hunter.
Rather than fight that, we separate concerns the way the engine already does for
combat: the **goal models the strategic intent** (template-level: "I hunt my
contract"), and the **specific target is per-instance runtime state** (like an
aggro target), stamped on the hunter instance's MiscData at spawn
(`bh_target_user_id`, `bh_bounty_id`). This keeps concurrent hunters correct
(each pursues its own target) without touching the goals architecture. (See §9
for the deferred "instance-keyed goal state" note.)

**Goal** (new catalog type in `internal/goals/catalog/`):
- **No params** — a pure strategic-intent marker for the hunter template.
- Predicate: **always active** (never self-satisfies). Per-hunter lifecycle is
  owned by the dispatch manager (it despawns each hunter when *its* bounty
  closes), not by goal pruning — because the shared template goal can't track an
  individual hunter's bounty.
- Context-score: high constant — the hunter's reason to exist; dominates its
  goal set. (Still allowed to panic-flee at the archetype's hard HP floor.)

**Planner** (`internal/planners/hunt_bounty_target.go`, per-tick, returns a
Command + Status). It reads the target from the **calling hunter instance's**
MiscData (`bh_target_user_id`), not from goal params:
0. **No target on this instance** (`bh_target_user_id` absent) → hold (Running,
   no command). (Shouldn't happen — the manager stamps it at spawn — but safe.)
1. **Jailed-target hold.** If the target is jailed (`jail_until_round` MiscData
   present on the target user): return a *hold* — loiter; do **not** path into
   the cell, do **not** attempt to engage. (The manager calls the hunt off once
   serving/paying clears the bounty.)
2. **Same room as target:** engage — `attack @<uid>` (subject to §7 safety).
3. **Else (pursue):** `pathto <target's current room>` (a closing chase;
   cross-zone supported via the mapper). `try_scan` / `try_track` add flavor and
   handle a hidden target on the final approach.

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

## 5. Hunter gear kit (reuses the instance affix-scaling path)

The hunter wears a **full kit of affix-scaled gear generated at spawn**, scaled
to its power — so a tougher bounty produces both a higher-statpool *and*
better-geared hunter. This reuses the exact mechanism the planar-oasis instance
uses for gold-scaled loot (`internal/rooms/rooms.go:786-795`):
`items.GenerateAffixedItem(baseItemId, goldPaid, LootBudgetScalar)` →
`Character.Wear(...)`, where the affix budget per item is
`LootBudgetScalar × sqrt(goldPaid)` (`LootBudgetScalar` default 7.0).

- **Base items (`loot_pool`):** ~8 new dedicated "bounty hunter" base items, one
  per slot — weapon, body, head, legs, feet, gloves, neck, back — listed in the
  hunter template's `loot_pool`. Base specs are modest (slot + base stats +
  `rarity_tier`); the affixes carry the scaled power.
- **Spawn-time generation + wear.** The dispatch manager replicates the instance
  loot path (the hunter isn't in a `gold_paid` instance room, so the manager
  drives it directly): for each `loot_pool` base item,
  `GenerateAffixedItem(baseId, gearGold, LootBudgetScalar)` then `Wear` it. The
  scale input is **`gearGold = hunterStatpool / BountyHunterGearGoldDivisor`**
  (default divisor 5). The hunter **uses** this gear — its damage + mitigation
  rise with the bounty, on top of the scaled statpool, making a high-bounty
  hunter genuinely dangerous.
- **Drops.** The standard equipped-item drop path (`hooks/Death_MobLoot.go`)
  rolls **each worn piece independently** against the mob's `itemdropchance`.
  Set `itemdropchance: 3` → ~3% per piece. Across ~8 pieces that is ~0.24
  pieces per kill — a nice rare trophy for a hard fight, but no farm: you only
  face hunters by being repeatedly wanted, which is costly. The dropped item is
  the affix-scaled instance, so the trophy reflects how tough the hunter was.
  The hunter does **not** carry the `PermaGear` flag (that would drop nothing).

Worked budget: at hunter statpool 500, `gearGold = 500/5 = 100`, per-item affix
budget ≈ `7 × sqrt(100) = 70`; at statpool 400, `gearGold = 80`, budget ≈
`7 × sqrt(80) ≈ 62`. (For reference, a 300g oasis run yields ≈ `7 × sqrt(300) ≈
121` per item — the hunter sits well below endgame-instance gear.) The divisor
is the balance knob; tune in playtest.

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
| `BountyHunterRedispatchCooldown` | 500 | Rounds before a new hunter after one is killed. |
| `BountyHunterGearGoldDivisor` | 5 | Gear affix scale: `gearGold = statpool / divisor`, fed to `GenerateAffixedItem` (§5). |
| Hunter `itemdropchance` (data) | 3 | ~3% independent per-worn-piece drop (~8 pieces). |

Hunter statpool = `clamp(250 + gold × 0.25, 300, 500)`. Reference points:
city/gate guard statpool 240, Guard Captain Velk 300.

| Player & crime | Power (stat-sum) | Bounty gold | Hunter? | Hunter statpool | Feel |
|---|---|---|---|---|---|
| Assault, mild | ~600 | ~300 | No (< 500) | — | below threshold |
| **Single murder, baseline player** | ~600 | **~600** | Yes | **400** | elite — above Velk (300) |
| Murder, developed player | ~850 | ~850 | Yes | ~460 | near-apex |
| Heinous / very strong / multi-Hostile | ~900+ | ~1000+ | Yes | **500 (cap)** | apex "oh no" |

So a town guard is 240, a captain 300, and a dispatched hunter runs **400–500**
plus an **affix-scaled gear kit (§5) that scales from the same statpool** — a
deliberate step above the watch, scaling with how wanted (and how strong) the
offender is. All values tunable.

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
- **Instance-keyed goal state.** The goals store (chunk 4.x) is keyed by mob
  *template* id, so goal *parameters* can't diverge per live instance of a
  template. 5.2 sidesteps this correctly (target on instance MiscData, goal as
  template-level intent — §3). If future work accumulates more instance-divergent
  goal-param needs (e.g. many same-template NPCs each pursuing a *different*
  goal target via the planner), invest in instance-keyable goal state then. Not
  needed now — "current target" legitimately belongs on instance state.
- **Bounty-board "pick up a contract"** UX for player hunters; NPC-bandit
  auto-bounty declaration; squad hunters.

---

## 10. Testing

**Unit:**
- Dispatch trigger: single open bounty ≥ threshold dispatches; sub-threshold
  does not; one-per-player (no second hunter while one is active);
  re-dispatch only after cooldown.
- Statpool scaling: the §6 table rows (`clamp(250 + gold×0.25, 300, 500)`).
- Gear generation: the spawn wear-loop produces an affixed piece per `loot_pool`
  base item, scaled by `gearGold = statpool / BountyHunterGearGoldDivisor`
  (assert the budget tracks the formula; higher statpool → larger affix budget).
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
