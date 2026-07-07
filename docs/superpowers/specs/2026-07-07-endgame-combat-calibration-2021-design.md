# Endgame Combat Calibration — Cascade Pass (#20) & Eastern Highlands (#21)

**Date:** 2026-07-07
**Status:** Design approved (brainstorm), pending plan.
**Scope:** Re-tune the two Eastern-Arc approach legs (#20, #21) into a graduated
solo→duo→trio difficulty staircase, and bank the tuning method as a standing SOP.

## Context

The Eastern Arc endgame is three legs:

| Leg | Type | Current tuning | Calibrated against |
|-----|------|----------------|--------------------|
| **#20 Cascade Pass Road** | Overworld, fixed statpool | Pass-Apex 550; fauna 275/220 | *Nothing* — left as a "trivial-but-appropriate ramp." Feel-test: the Apex only killed a mid char when co-spawned with a cat. |
| **#21 Eastern Highlands** | Overworld, fixed statpool | Sentinel 1200 (`brute`) + adds 300; fauna 350–560 | A geared **solo** master (2026-07-01). |
| **#22 Crash Site** | Instanced, gold-scaled | Full mechanics + ×5/×7 boss scaling | A 3-Meirok **party**. ✅ Done + live on prod. |

The #22 redesign banked the core lesson: **a geared party out-DPSes any
stat-brute regardless of statpool** — bigger numbers don't create difficulty,
counterplay and durability do. `base_pool` inflation is a trap: it scales a
mob's *melee* alongside its pool, which turned #22's support adds into killers.

The **Meirok** (`prod_meirok.yaml`, user 24) is the canonical "uber endgame
character" baseline: HP ~610, stats 95–115, weapon-combat 69 / unarmed 57 /
spellcasting 51 / rhetoric 55, extra-arms L1 (triple drowned-claw), conviction/
summoner kit (conviction-ward shield, rally/warcry/surge self-buffs, summon
steppe-spirit + undead). **"1 Meirok + companions"** = one such character plus
its summoned pets.

## Design intent — the graduated ramp

The arc is a difficulty staircase:

- **#20 Cascade Pass** — tough-but-winnable for **1 Meirok + companions**.
- **#21 Eastern Highlands** — tough-but-winnable for **2 Meiroks + companions**.
- **#22 Crash Site** — the **3-Meirok** finale (already tuned; out of scope here).

Consequences that are **intended, not bugs**:

- An over-strength party trivializing an earlier leg on a repeat pass is the
  ramp working as designed.
- Only the **signature fights** are calibrated: the **Pass-Apex** (#20) and the
  **Sentinel** (#21). Ambient fauna stay "non-trivial but survivable while
  traversing" and are not the tuning target.
- **Neither boss is a mandatory traversal gate** — both sit off the disc-door
  path (the Sentinel's vault is off the door path; the Apex is a road
  encounter). A solo player heading to the #22 Keeper can still reach it by
  avoiding the signature fight. This preserves the #22 entry for solo/duo
  players who are building toward a trio.

## Standing SOP — nodrop gear is the durability lever

**Adopt as standing practice for all endgame combat tuning from 2026-07-07 on.**

Endgame mobs are tuned on **two independent axes**:

1. **`statpool`** → sets *damage dealt, accuracy, and threat*.
2. **`NeverDrops` mitigation/stat gear** → sets *effective durability* (EHP)
   independently of statpool.

Why this beats raw statpool or `base_pool`:

- Raising `statpool` alone couples damage-dealt and pool-size together; a mob
  tuned to *survive* a party's DPS by statpool also hits far too hard.
- `base_pool` inflation scales the mob's *melee* — the #22-adds trap that turned
  supports into killers.
- `NeverDrops` gear (the `NeverDrops` item flag) adds mitigation/EHP with **no
  loot pollution** (the gear never drops) and **no melee-damage side effect**.

This is the #22-adds technique elevated to a standing SOP. It is *not* applied to
the **test party** — the Meirok clones are already the fixed geared baseline and
need no nodrop gear.

## #20 Pass-Apex — pure stat/gear (solo warm-up)

The Apex (mob 9541, `predator`/`leader`, statpool 550) stays mechanically
plain — it is the solo warm-up. Tune **only** two axes:

- **Bump `statpool`** until it threatens a Meirok (current 550 lets a triple-claw
  Meirok burst it; the exact target is found empirically, see Success Criteria).
- **Add `NeverDrops` mitigation gear** so the Meirok can't one-rotation it —
  the fight has to last long enough to matter.

No new mechanics, no adds beyond its existing pack behavior. Ambient fauna
(9538–9540 @275, 9542 @220) are left as-is unless a run shows a specific pack is
trivially ignorable or accidentally lethal.

## #21 Sentinel — stat/gear + ONE light hook

The Sentinel (mob 9552, `brute`, statpool 1200) is the duo optional-boss and
earns a hook so it isn't the "sponge" the #22 critic flagged.

- **Bump `statpool` substantially** + **heavy `NeverDrops` mitigation gear** —
  two Meiroks out-DPS a plain 1200 pool fast, so both damage and EHP go up.
- **ONE telegraphed mechanic — "Rouse the Wards":** at a HP threshold the
  Sentinel telegraphs, then activates/summons its existing adds (**Roused Ward**
  9550 + **Watcher-Shard** 9551), forcing the duo to split focus. This reuses
  content that already exists in the zone rather than inventing a new system.

  **Alternatives considered (pick exactly one during planning if "Rouse the
  Wards" proves awkward):** (a) a telegraphed heavy **discharge** the party must
  spread from; (b) a periodic **self-ward** the party must burn through,
  punishing pure tunnel-DPS. "Rouse the Wards" is the recommended default.

The hook is deliberately *light* — a single telegraphed beat, not the #22
drain/discharge/interrupt/3-add apparatus. Mechanical depth ramps alongside
difficulty: #20 none, #21 one hook, #22 the full system.

## Success criteria — "tough but winnable" operationalized

So harness runs and math can judge a pass/fail:

- The **intended-size** party (1 Meirok for #20, 2 for #21) **wins**, but:
  - ends the fight **below ~30% HP**, and
  - has spent **consumables and/or cooldowns** (potions, shields, summons) to
    do it.
- A party **one size smaller** than intended (solo vs the #21 Sentinel) **loses
  or is forced to disengage**.
- **Fail conditions:** a faceroll (win at >70% HP with no cooldowns burned) is
  under-tuned; a wipe by the intended-size party is over-tuned.

Because the AI harness plays sub-optimally, a harness *win by a clumsy agent* is
treated as roughly equivalent to a *comfortable human win* — so target the
harness landing slightly on the "hard" side of the criteria, and cross-check
against DPS/EHP math rather than trusting a single run.

## Method

1. **Test rig:** reuse the Meirok-clone harness accounts (quester4/5/6 →
   chars Vael/Ryn/Doss, cloned from prod Meirok user 24). Run **1 char** for
   #20 and **2 chars** for #21 via the multi-agent playtest-scenario path
   (`/playtest-scenario`, party mode).
2. **Pre-run hygiene:** nuke `_datafiles/world/dogmud/mobs.instances/*` before
   each boot — overworld mob stat edits are shadowed by stale instance saves
   otherwise (the recurring instance-save SOP).
3. **Iteration loop, per zone:** analyze current numbers vs the Meirok baseline →
   set a statpool + nodrop-gear starting point → boot → run the intended-size
   party (plus a one-smaller control run to confirm the lower bound fails) →
   read the beacon/report against the success criteria → adjust the two axes →
   repeat until the criteria hold.
4. **Order:** #20 first (simpler, pure stat/gear, solo), then #21 (the hook +
   duo run).
5. **Build the Sentinel hook** via the existing btree/spell plumbing used for
   the #22 bosses (telegraph message + threshold-gated add activation); no new
   engine primitives are expected — verify against the #22 boss btrees during
   planning.

## Out of scope

- #22 re-tuning (done + on prod).
- Ambient-fauna rebalance beyond spot-fixes surfaced by runs.
- New engine primitives — this is data + one light btree hook, reusing #22
  plumbing.
- The prod push of any resulting changes (its own pre-push SOP pass afterward).

## Open items to resolve in the plan

- Exact starting statpool + gear values for the Apex and Sentinel (empirical;
  seed from the baseline analysis).
- Final choice + threshold for the Sentinel hook (default "Rouse the Wards").
- Whether the one-size-smaller control run is run every iteration or only at the
  end to confirm the lower bound.
