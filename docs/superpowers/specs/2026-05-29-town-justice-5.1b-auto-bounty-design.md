# Town Justice 5.1b — Crime → Auto-Bounty Trigger

**Date:** 2026-05-29 • **Roadmap:** Phase 5.1, slice b of d • **Size:** M
**Depends on:** 5.1a (justice package), crimes (1.3), bounties (1.5), factions (1.2/1.5)

---

## Overview

5.1a gave guards a "wanted" verdict that already treats an open faction bounty
as attack-on-sight. 5.1b makes the world *post* those bounties: when a player
commits an identified murder of a faction member, or their standing with a town
faction crosses into Hostile, that faction auto-declares a kill-bounty on the
player. It also closes the loop — bounties resolve when the target dies
(claimed by a third-party killer, or expired when the issuer's own guards/the
environment kill them) so they never persist forever.

Death clears the *bounty*, not the underlying rep/crime wanted status (those
persist as institutional memory; clearing one's name is 5.1d redemption).

---

## Locked decisions

| Decision | Value |
|----------|-------|
| Trigger | Identified murder **OR** rep crosses Hostile (either path) |
| Fire point | The crime-recording sites, via `justice.MaybeDeclareBounty` |
| Reward | `powerBase × max(crimeMult, repMult)` (whichever is higher) |
| Dedup | One open faction bounty per (faction, player) — skip if already open |
| Expiry | `now + JusticeBountyExpiryRounds` |
| Death-resolution | In 5.1b. Third-party (player) kill → claim + payout; guard/env/self → `MarkExpired`, no payout |

---

## Components

### 1. `justice.MaybeDeclareBounty(faction string, userId int, triggerKind crimes.Kind)`

Called from the crime sites after their `BumpRep`. Logic:
1. **Dedup** — return if `bounties.OpenAgainstPlayer(userId)` already has an
   open bounty issued by this faction.
2. **Trigger gate** — proceed only if `triggerKind == crimes.KindMurder`
   (identified-perp murder) OR `factions.TierFor(faction, userId) == TierHostile`.
   Otherwise return.
3. **Reward** (see below) → `gold`.
4. **Declare** — `bounties.Declare(bounties.FactionIssuer(faction),`
   `knowledge.PlayerSubject(userId), bounties.ConditionKill, now + expiry,`
   `bounties.DeclareOpts{GoldOverride: gold, DeclaredReason: reason})`.
   `reason` = `"Murder of <victim/faction>"` (murder path) or
   `"Crimes against <faction display name>"` (rep path).

Lives in `internal/justice` (already imports crimes/bounties/factions). The
murder path is invoked only with an identified player perp (an unknown-perp
murder can't name a target).

### 2. `bounties.DefaultGoldFor(target knowledge.Subject) int`

Small new exported wrapper over the existing `computeDefaultGold` so `justice`
can read the statpool-derived base without duplicating the formula.

### 3. Reward math

- `powerBase = bounties.DefaultGoldFor(PlayerSubject(userId))`.
- `crimeMult` = `JusticeBountyMurderMult` (default 2.0) when `triggerKind ==
  KindMurder`, else `1.0`.
- `repMult` = hostility depth: `1.0` at the Hostile boundary (rep −50),
  ramping linearly to `JusticeBountyRepMultMax` (default 2.0) at rep −100;
  `1.0` if not Hostile. Formula:
  `1.0 + clamp((|rep|-50)/50, 0, 1) × (JusticeBountyRepMultMax - 1.0)`,
  using `rep = factions.GetRep(faction, userId)`.
- `gold = round(powerBase × max(crimeMult, repMult))`.

### 4. Fire points (crime sites call `MaybeDeclareBounty`)

- `internal/hooks/MobDeath_FactionRep.go` — after each murder rep-bump, when the
  perp is an identified player, call with `crimes.KindMurder`.
- `internal/usercommands/attack.go` (assault recording) — after the assault
  rep-bump, call with `crimes.KindAssault` (rep-path only).
- `internal/usercommands/skill.skullduggery.steal.go` (theft) — after recording,
  call with `crimes.KindTheft` (rep-path only).

(Unlike 5.1a's spatial guard reaction, an auto-bounty is a direct consequence of
the crime, so firing at the crime site is the precise trigger.)

### 5. `PlayerDeath_BountyResolve` hook

On player death (reading the killer `ActorRef`), for each open bounty against the
dead player (`bounties.OpenAgainstPlayer`):
- **killer is a player and killer ≠ target** → `bounties.TryClaim(id,
  PlayerSubject(killerUserId))`; on success, transfer `GoldReward` to the killer
  and, if the issuer is a faction, `factions.BumpRep(issuerFaction, killer,
  RepReward)` (mirrors `MobDeath_BountyClaim`). Status → Claimed ("turned in").
- **otherwise** (guard/other mob, environment, self) → `bounties.MarkExpired(id)`,
  no payout.

NPC bounty-hunter pursuit + their claim payout is 5.2; 5.1b only needs the
player-death close so bounties don't leak.

### 6. Config (Balance)

- `JusticeBountyExpiryRounds` (default 5000)
- `JusticeBountyMurderMult` (default 2.0)
- `JusticeBountyRepMultMax` (default 2.0)

---

## Testing

- `justice.MaybeDeclareBounty` (seam-tested like Verdict): murder→declares;
  rep-Hostile→declares; neither→no-op; dedup (existing open faction bounty→skip);
  reward = powerBase × max(crimeMult, repMult) for murder-vs-rep-dominant cases.
- `bounties.DefaultGoldFor` matches `computeDefaultGold`.
- Reward formula pure helper (crimeMult/repMult/max) — table tests.
- `PlayerDeath_BountyResolve`: player-killer→Claimed+payout; guard/mob/env
  killer→MarkExpired no payout; no open bounty→no-op.
- Boot smoke: config knobs load; no panic.

---

## Out of scope

- NPC bounty-hunter seeking + their claim payout (5.2).
- Redemption / clearing rep+crimes (5.1d) — death does NOT clear the underlying
  wanted status.
- Non-kill bounty conditions; per-zone justice tuning.
