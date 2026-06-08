# Bounty Thornwall Smoke Test — 2026-05-09

**Tester:** smoketester (AI feature-tester)
**Server:** local localhost:55555
**Substrate under test:** chunk 1.5 bounty state
**Session date:** 2026-05-09

---

## Smoke Verdict

**BLOCKED — end-to-end path not fire-testable without admin pre-declare.**

The `bounty` command is fully implemented and all player-visible surfaces
work correctly. `bounty list`, `bounty show`, `bounty list <filter>`,
and the help text all respond gracefully with no errors. The
Thornwall Guard Barracks bounty board (room 473) renders its flavor text
and `bounty list` hint correctly. All admin-only subcommands
(`declare`, `withdraw`, `prune-expired`) are properly gated with the
"That subcommand is admin-only." response for non-admin users.

However, **no bounty was declared before the test run.** `bounty list`
returned "No bounties." throughout the entire session. As a result,
Goals 1, 2, 5, and 6 — the core declare→list→kill→payout→gone path —
could not be tested at all. The auto-claim hook (`MobDeathBountyClaim`,
registered in `internal/hooks/hooks.go:96`) and the payout message path
were verified by code inspection only, not live execution.

Goal 4 (Stillwater Constabulary board) was not reached due to
multi-zone travel distance (Thornwall → Watchers Crossing → Dustwalk
Road is 8+ rooms, with the Stillwater gate another ~10 rooms further
north through zones with respawning hostiles). The bounty board noun is
correctly defined in room 4110.yaml, verified by file inspection.

---

## Goal Results Table

| # | Goal | Result | Notes |
|---|------|--------|-------|
| 0 | Login + orientation, note starting gold | PASS | Logged in, `look` confirmed Tavern Kitchen [Thornwall City]. Starting gold: 82 (Bank: 90). |
| 1 | `bounty list` shows pre-declared mob:101 bounty | BLOCKED | No bounties declared. "No bounties." returned. See Admin Commands Needed. |
| 2 | `bounty show <id>` formats correctly | BLOCKED | No bounties to show. `bounty show 1` returned "No bounty with id 1". |
| 3 | Travel to room 473, `look bounty board` | PASS | Flavor text rendered correctly. Hints at `bounty list`. See Findings: PASS. |
| 4 | Travel to Stillwater Constabulary (room 4110), `look bounty board` | BLOCKED | Multi-zone journey too long + dangerous. Route confirmed by file inspection. See Findings: OBSERVATION. |
| 5 | Kill street_performer (mob 101), verify auto-claim payout | BLOCKED | No bounty declared; kill would not trigger payout. Street performer located at Main Street Central. |
| 6 | `bounty list` shows bounty gone after kill | BLOCKED | Depends on Goal 5. |
| 7 | Write smoke verdict | COMPLETE | See above. |
| 8 | Report admin commands needed | COMPLETE | See section below. |

---

## Findings

### PASS

**Bounty command player surface works correctly.**
- `bounty` (no args): prints the user-facing help text including
  `bounty list [filter]`, `bounty show <id>`, filter options, and
  the "Notice boards in guard barracks and constabulary offices"
  tip. Help text pulled from `_datafiles/world/dogmud/templates/help/bounty.template`.
- `bounty list` with no bounties: "No bounties." — clean, no crash.
- `bounty list mob`: "No bounties." — filter handling works when empty.
- `bounty list player`: "No bounties." — filter handling works.
- `bounty list thornwall_guards`: "No bounties." — faction filter works.
- `bounty show 1` with no bounties: "No bounty with id 1" — correct.

**Admin-only gate is enforced.**
- `bounty declare faction:thornwall_guards mob:101`: "That subcommand is admin-only."
- `bounty withdraw 1`: "That subcommand is admin-only."
- `bounty prune-expired`: "That subcommand is admin-only."

**Thornwall Guard Barracks bounty board (Goal 3 — PASS).**
Room 473 reached via: Gate Ward (460) → north. `look bounty board`
returned:
> "A weather-worn corkboard stands beside the wall, papered with wanted
> notices and contract slips, recent ones still clean, older ones
> curling at the edges. Anyone with the will to read them can run
> "bounty list" anywhere in town to see what's open."

Flavor text is present, correctly hints at `bounty list`, and the noun
key `bounty board` (space-separated, per project convention) triggers
correctly.

**Bounties package and hook wired correctly (code inspection).**
- `internal/hooks/hooks.go:96`: `events.RegisterListener(events.MobDeath{}, MobDeathBountyClaim)` is present.
- `internal/hooks/MobDeath_BountyClaim.go`: Hook fires on `MobDeath` events, finds `OpenForTarget` bounties, resolves highest-damager player, calls `bounties.TryClaim()`, awards gold, bumps faction rep for faction-issued bounties, sends payout message.
- `bounties.go`: `Declare`, `TryClaim`, `AllOpen`, `OpenForTarget`, `Withdraw`, `PruneExpired` all implemented with mutex-protected registry + save-on-mutation.

### OBSERVATION

**Stillwater Constabulary bounty board (Goal 4 — not live-tested).**
Room 4110 YAML confirmed to have `bounty board` noun key with flavor
text that mentions `bounty list`. The noun text differs slightly from
the Thornwall board (mentions a "fresher notice... doubling the reward
for the lake-cave bounty") which provides per-location flavor. Route
from Thornwall to Stillwater: west through Thornwall Outskirts, Watchers
Crossing, Dustwalk Road, then north through an uncounted chain of zones
to the Stillwater gate (room 4100). Estimated 20+ room hops through
zones containing respawning hostiles (Thornwall Highwayman confirmed
on City Road). Not feasible in this session without fold-anchor in
Stillwater.

**`bounties.yaml` data file does not exist yet.**
`_datafiles/world/dogmud/bounties.yaml` is absent. This is expected
behavior — persistence code uses `loadOrLazyInit()` which creates the
file on first `bounty declare`. No bounties have ever been declared on
this server instance.

**bounty list column order differs between code and goal file.**
The goal file says columns should be: ID, Issuer, Target, Gold, Rep,
Reason. The source (`admin.bounty.go:136-146`) renders:
ID, Status, Issuer, Target, Gold, Rep, Reason — there is an additional
`Status` column not mentioned in the goal file. This is more
informative, not a bug, but worth noting for goal-file accuracy.

### CONCERN

**Payout message contains a raw gold number.**
`MobDeath_BountyClaim.go:72` sends:
```
"You collect a bounty: %dg.\r\n"
```
Per CLAUDE.md ("No Hard Numbers" convention), player-facing text should
not display raw numeric values. The bounty payout amount is a mechanical
number exposed directly to the player. This should use a descriptive
phrase (e.g., "You collect a substantial bounty." or a
`GetCurrencyDescription(amount)` helper if one exists). Not blocking,
but is a design-convention violation.

### BUG

None confirmed by live testing. No crashes or unexpected error
responses were observed during the session.

---

## Per-Event Ledger

| Event | Gold Before | Bounty Amount | Gold After | Message Seen |
|-------|------------|--------------|------------|-------------|
| Goal 5 kill (street_performer) | — | — | — | N/A — no bounty was active |

Starting gold: **82** (on-hand); Bank: 90. No payout occurred.

---

## Admin Commands Needed

### Pre-Test Setup (must run before Goals 1-6 can be tested)

```
bounty declare faction:thornwall_guards mob:101 --reason "Disturbing the peace"
```

This should:
- Create a bounty targeting mob template 101 (street_performer)
- Auto-compute gold from street_performer's StatPool × BountyGoldDefaultMultiplier
  (floor: BountyGoldFloor)
- Auto-compute rep from StatPool/100
- Return "Declared bounty #N."
- Persist to `_datafiles/world/dogmud/bounties.yaml`

### Verification (run after the kill in Goal 5)

```
bounty show <id>
```
Expected: `Status: claimed`, `Claimed: round X by player:<smoketesterId>`, gold/rep matching what was declared.

```
bounty list --all
```
Expected: Shows the claimed row with status=claimed, so the admin can see the bounty did not silently disappear.

```
faction show smoketester
```
Expected: `thornwall_guards` rep increased by the bounty's `RepReward` value.

### Stillwater board verification (if admin wants to run Goal 4 separately)

Travel to room 4110 (Stillwater Constabulary) and run:
```
look bounty board
```
Expected: flavor text matching the `bounty board:` noun in room 4110.yaml,
with the "Anyone with the will to read them can run `bounty list`" hint
and the added "doubling the reward for the lake-cave bounty" flavor line.

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Commands sent | ~55 |
| Player deaths | 0 |
| Mob kills (non-bounty) | 1 (steppe rat, in self-defense outside east gate) |
| Bounties claimed | 0 |
| Gold delta | +0 (no payout; no looting) |
| Session start gold | 82 |
| Session end gold | 82 |
| Goals fully PASS | 2 (Goal 0 login, Goal 3 Thornwall board) |
| Goals BLOCKED | 5 (Goals 1, 2, 4, 5, 6) |
| Goals COMPLETE | 2 (Goals 7, 8 report) |
| Crashes observed | 0 |
| Unexpected errors | 0 |
