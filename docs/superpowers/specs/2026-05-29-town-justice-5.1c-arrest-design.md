# Town Justice 5.1c — Arrest (Surrender / Resist + Jail + Fine)

**Date:** 2026-05-29 • **Roadmap:** Phase 5.1, slice c of d • **Size:** L
**Branch:** `feature/town-justice-5.1c-arrest` (forked off 5.1b — reuses
`justice.bountyGold`)
**Depends on:** 5.1a (justice package, guard enforcement tick), 5.1b
(`bountyGold`, auto-bounty, `bounties.Withdraw`), crimes (1.3, `Resolve`),
factions (1.2/1.5), buffs (`no-go` flag), rooms (`MoveToRoom`, `allow_recall`
TempData).

---

## Overview

5.1a gave guards a warn→attack ladder; 5.1b made factions auto-post bounties.
5.1c makes the top of the ladder **humane and reversible**: instead of killing
wanted players on sight, guards now attempt an **arrest**. The player's
pre-decided `ArrestPolicy` (default `surrender`) forks the outcome — surrender →
hauled to a holding cell to serve a timed sentence (with an early buy-out via a
decaying fine); resist (or fight back during a grace window) → drops to the old
lethal `SeverityAttack` combat path.

Serving the sentence **or** paying the fine is a complete wanted-status reset
for the issuing faction: the triggering crimes clear, the open faction bounty is
withdrawn, and reputation resets to low-ish neutral. The player must commit
fresh crimes to be arrested again. This pulls the pay-fine and rep-reset slices
of 5.1d forward, leaving 5.1d as optional quest-based redemption (or empty).

---

## Locked decisions

| Decision | Value |
|----------|-------|
| Core mechanic | Surrender-or-resist fork, pre-decided by player `ArrestPolicy` (default `surrender`). NOT combat-subdue (deferred — needs guard stat-bump + sap/net content). |
| Verdict reshape | `Verdict` returns `SeverityArrest` (not `Attack`) for all wanted signals. `SeverityAttack` becomes purely the resist outcome, decided at enforcement time. |
| Resist window | `ArrestResistGraceRounds` (default 3) between declaration and haul-off; player attacking a guard mid-window → combat. `resist` policy skips the window straight to attack. |
| Detention end | Timed sentence with early buy-out. Cell-door shows current fine = `original − roundsServed × decayPerRound`. |
| Fine basis | Reuses `justice.bountyGold` (powerBase × max(crimeMult, repMult)). Sentence length = `fine ÷ JusticeFineDecayPerRound`. |
| Fine payment | Person-gold first, bank as fallback. |
| Release effects | Crime clears (`crimes.Resolve`), faction bounty withdrawn (`bounties.Withdraw`), rep reset to low-ish neutral (`JusticeArrestRepReset`, only if currently below it). Applies on BOTH timer-expiry and fine-paid. |
| Jail location | New Thornwall cellblock, `down` from room 473 (Guard Barracks). Faction→cell lookup; Stillwater 4110 reserved for later. |
| Lockdown | Jailed buff with `no-go` flag (blocks walk/flee) + cell room `allow_recall: false` (blocks recall-class spells). Belt-and-suspenders. |

---

## Components

### 1. Verdict reshape (`internal/justice/justice.go`)

`Verdict(guardFactions, userId)` currently returns `SeverityAttack` for the
wanted signals (open bounty, Hostile rep, unresolved crime) and `SeverityWarn`
for Cold rep. 5.1c changes every `SeverityAttack` return in `Verdict` to
**`SeverityArrest`**. The `SeverityWarn` path is unchanged. `Verdict` no longer
produces `SeverityAttack` at all — Attack is now produced only by the resist
fork in enforcement.

The ordered enum (`None < Warn < Arrest < Attack`) is unchanged; only what
`Verdict` emits shifts down one rung.

### 2. Enforcement fork (`internal/justice/enforce.go`)

`RunGuardEnforcement` grows an `Arrest` branch (the existing `Warn` and the
now-vestigial `Attack` switch arms stay — `Attack` is still reachable via the
resist path's `mob.Command("attack @id")`, and warned-past-grace now escalates
to Arrest instead of Attack).

Arrest branch, per wanted player:

1. Read the player's `ArrestPolicy`.
2. **`resist`** → `mob.Command(fmt.Sprintf("attack @%d", uid))` immediately
   (player pre-decided to fight; no window). Stamp nothing.
3. **`surrender`** → check `justice_arrest_pending_<uid>` in the guard's
   MiscData:
   - **Not pending** → declare: `guardSayFn` an arrest-intent line ("steps
     forward, manacles in hand…"), stamp `justice_arrest_pending_<uid> =
     nowRound`. Return (the haul-off waits for the window).
   - **Pending, within grace** (`nowRound − pending < ArrestResistGraceRounds`)
     → do nothing this tick (window still open; player may yet attack and flip
     to combat — handled by the guard going `IsInCombat`, which early-returns
     `RunGuardEnforcement`).
   - **Pending, past grace** → execute: call `justice.ExecuteArrest(guard,
     player, jailCellId, fine, sentenceRounds)`; clear the pending stamp.

Resist-during-window is handled implicitly: if the player attacks the guard, the
guard enters combat, `RunGuardEnforcement`'s existing `IsInCombat` early-return
fires, and the lethal combat path takes over. The pending stamp is left to be
overwritten/ignored (it's inert once combat starts; prune opportunistically or
leave like 5.1a's warn stamps — followup, same as 5.1a).

### 3. `justice.ExecuteArrest` + fine helpers (`internal/justice/arrest.go`, new)

```go
// computed fine at declaration time (reuses bountyGold from 5.1b):
//   fine := bountyGold(DefaultGoldFor(PlayerSubject(uid)), isMurder, rep, murderMult, repMultMax)
//   sentenceRounds := fine / JusticeFineDecayPerRound  (min 1)
```

- `ExecuteArrest(guard, player, cellId, fine, sentenceRounds)` — applies the
  Jailed buff (duration = sentenceRounds), `rooms.MoveToRoom(uid, cellId)`,
  stamps the jail record, plays drag-to-jail flavor (room-broadcast at the
  origin + arrival flavor in the cell).
- `currentFine(original, roundsServed, decayPerRound) int` — pure;
  `max(0, original − roundsServed×decayPerRound)`.
- `ResolveDetention(player, rec)` — the single release path (timer OR fine):
  `crimes.Resolve(rec.CrimeIds)`; for the issuing faction, withdraw any open
  bounty (`bounties.Withdraw`); reset rep via `factions` to
  `JusticeArrestRepReset` **only if current rep is below it**; remove Jailed
  buff; `MoveToRoom` → 473; release flavor.

Seams (test pattern from 5.1a/b): `aResolveCrimeFn`, `aWithdrawBountyFn`,
`aSetRepFn`, `aGetRepFn`, `aMoveFn`, `aNowFn`, etc., so the helpers unit-test
without the live registries.

### 4. Jail record + Jailed buff

**Jail record** — persisted on the character (MiscData keys, so a relog
mid-sentence stays jailed):
`jail_until_round`, `jail_fine_original`, `jail_decay_per_round`,
`jail_faction`, `jail_crime_ids` (comma-joined), `jail_cell_room`.

**Jailed buff** (next free buff ID via `id_inventory.py`) — carries the `no-go`
flag; duration = sentenceRounds (expires with the timer). A buff-expiry hook (or
the per-round mob/justice tick) calls `ResolveDetention` on natural expiry so
the timer-release path runs (crime clear + bounty withdraw + rep reset + move
home), not just a silent buff drop. The buff's presence is also the
`IsJailed(player)` predicate the jail commands gate on.

### 5. Jail lockdown (egress block)

| Vector | Block |
|--------|-------|
| `go <dir>` / walk | `no-go` flag on the Jailed buff — already enforced at `usercommands/go.go:89` (`HasBuffFlag(buffs.NoMovement)`). Override the rejection text to a cell-flavor line. |
| `flee` (combat) | same `no-go` path; no combat happens in the cell anyway. |
| `fold-recall` / recall-class | cell room sets `allow_recall: false`; already enforced in `validateFoldRecall` (`hooks/spell_foldrecall.go:25`). |
| death/respawn | dying in the cell voids the sentence (respawn teleport allowed). Acceptable — rep/crime persist, so it's not a free pardon; just an escape-by-death edge. |

### 6. Jail commands (`internal/usercommands/`)

- **`fine`** (jailed-only) — shows `currentFine`, rounds remaining, and the
  `pay fine` hint. Also surfaced via the cell-door noun (`look door` / `look
  fine`).
- **`pay fine`** (jailed-only) — deducts `currentFine` from person-gold first,
  bank as fallback; on success → `ResolveDetention` (immediate release).
  Insufficient total → flavor refusal, stays jailed.

Both are thin, `IsJailed`-gated commands reading the jail record. Non-jailed
callers get a "you're not in jail" style no-op/usage.

### 7. Player policy (`ArrestPolicy`)

`surrender` (default) / `resist`, stored on the character. Set via the existing
policy-setting command surface — match whatever `SurrenderPolicy` /
submission-policy uses (e.g. a `set` subcommand). Read by the enforcement fork.

### 8. Content: Thornwall cellblock

- **New room**, `down` from 473 (Guard Barracks). 473's current exits are
  `south→460`, `up→5104`; **`down` is free** (no overlap). The barracks
  description already references "three iron-barred cells at the back," so a
  stairway down to a cellblock is coherent with existing prose.
- Room sets `allow_recall: false`. Single ID via `id_inventory.py`.
- **Faction→cell lookup** — `thornwall_guards → <new cellId>` (a small map in
  `internal/justice`, or a faction field). Stillwater 4110 reserved for when
  `stillwater_guards` lands.

### 9. Config (Balance)

| Knob | Default | Purpose |
|------|---------|---------|
| `ArrestResistGraceRounds` | 3 | decision window between arrest declaration and haul-off |
| `JusticeFineDecayPerRound` | (tune; e.g. 5) | gold/round the fine decays while serving; sets sentence length = `fine ÷ this` |
| `JusticeArrestRepReset` | −10 (low neutral) | rep floor restored on release (only raises, never lowers) |

Fine **base** = `justice.bountyGold` (no new knob).

---

## Data flow

### Wanted surrender-policy player meets a guard
Player enters guarded room → guard tick → `Verdict` = `Arrest` → policy
`surrender` → declaration tick (flavor + pending stamp) → grace window → player
stays, doesn't fight → execution tick → `ExecuteArrest` → Jailed buff + move to
cell + jail record. Player waits out the timer (rep/crime/bounty cleared on
expiry) OR runs `pay fine` (same release effects, immediately).

### Wanted resist-policy player meets a guard
Guard tick → `Verdict` = `Arrest` → policy `resist` → `attack @uid` →
combat (the 5.1a lethal path). Or surrender-policy player who attacks the guard
during the grace window → guard `IsInCombat` → enforcement early-returns →
combat.

---

## Implementation notes (for the plan)

- **GUARDS MUST BE ATTACKABLE FOR RESIST TO WORK.** Guards (106 city_guard, 94
  guard_captain_velk, 92) are `behavior_archetype: noncombat_questgiver`,
  `hostile: false`. They do NOT carry `non_combatant: true` or
  `player_attack_immune` in their YAML (grep at spec time found neither). The
  attack-block at `usercommands/attack.go:166` keys off `IsNonCombatant()` /
  `PlayerAttackImmune`. **Plan must FIRST verify the actual current protection
  mechanism** (it may be archetype-derived, a group rule, or simply that nothing
  blocks them today) and ensure a wanted player CAN attack a guard so the resist
  path is reachable. If guards are currently attack-immune by design, that
  immunity must be lifted or conditioned (e.g. only immune to law-abiding
  players, attackable once arrest is declared / player is wanted). This was a
  deliberate pre-5.1c safety; 5.1c is where it's revisited. **Do not assume —
  trace `IsNonCombatant`/archetype handling and confirm before wiring resist.**
- `bountyGold` and the `b*Fn` config seams are unexported in package `justice`
  (5.1b) — `arrest.go` lives in the same package, so it can call `bountyGold`
  and reuse `bMurderMultFn`/`bRepMultMaxFn`/`bDefaultGoldFn`/`bRepFn` directly.
- Reuse the `guardSayFn` seam (5.1b decouple) for arrest-intent speech — no new
  `actions` import in `justice`.
- The cell room and the new buff load at startup; the pre-push boot smoke (SOP)
  catches filename/ID/exit mistakes.

---

## Testing

- **Verdict**: wanted signals now yield `SeverityArrest` (not `Attack`); Cold →
  `Warn`; Neutral+ → `None`. (Amend the 5.1a verdict tests that asserted
  `Attack`.)
- **Enforcement fork** (seam-tested): `resist` policy → issues `attack @id`;
  `surrender` → declares + stamps pending on first tick, no-op within grace,
  `ExecuteArrest` past grace; player-attack-mid-window → combat (guard
  IsInCombat early-return).
- **Fine math**: `currentFine(original, served, decay)` table — full at served=0,
  decays linearly, floors at 0.
- **`ResolveDetention`** (seams): calls `crimes.Resolve` with the stamped IDs,
  `bounties.Withdraw` for the issuing faction, rep set to reset floor only when
  below it, buff removed, move-home invoked. Verified for BOTH the timer-expiry
  caller and the `pay fine` caller.
- **`pay fine`**: deducts person-gold first then bank; insufficient → no release.
- **Lockdown**: Jailed buff carries `no-go`; cell room has `allow_recall:false`.
- **Boot smoke** (instance wipe per SOP): new cell room (down from 473) loads,
  new Jailed buff loads, the three config knobs load, server boots clean.

---

## Out of scope (later)

- **Combat-subdue arrests** — the richer "guards physically subdue you" path
  needs a guard stat-bump + skill ranks + subdue weapons (sap/net). Deferred per
  design decision; surrender/resist policy is the 5.1c mechanic.
- **Stillwater rollout** — `stillwater_guards` faction + wiring cell 4110; when
  that faction exists (tracked in `project_town_justice_5_1_followups`).
- **Quest-based redemption** — the rump of 5.1d (pay-fine + rep-reset land here).
- NPC-on-NPC arrests; third-party bail; escape attempts / picking the cell lock;
  per-zone justice tuning.
