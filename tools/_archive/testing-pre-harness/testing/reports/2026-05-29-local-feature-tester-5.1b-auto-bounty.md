# Town Justice 5.1a+5.1b Auto-Bounty Smoke Test
**Date:** 2026-05-29  
**Branch:** feature/town-justice-5.1b (local)  
**Tester:** smoketester (AI)  
**Session duration:** ~60 minutes  
**Commands sent:** ~110

---

## Session Summary

Tested the Town Justice 5.1a+5.1b crime → auto-bounty → guard enforcement
pipeline in Thornwall. The session started with smoketester already in combat
with a City Guard in Market Square (prior session state), which provided an
opportunity to immediately verify the auto-bounty trigger. Subsequent testing
covered: bounty system visibility, guard reaction inconsistencies, the warn
mechanism, combat stall edge cases, the death/bounty-resolve hook, and
post-death guard treatment.

---

## Goal Results

### Step 1: Connect; check starting location, stats, and current reputation
**PARTIAL** — Connected cleanly. Location: Market Square West, Thornwall City.
Stats: Strength/Dex/Vit/Wil all "keen", Per/Cha "average". Gold: 1, Bank: 90.
No reputation/faction command exists; `factions` and `reputation` return "not
recognized." `rep` maps to the report command (HP/SP/CP). Baseline reputation
was NOT observable before the first crime.

### Step 2: Travel to Thornwall and locate a guard; note law-abiding behavior
**PARTIAL** — Session started in Thornwall. Could not establish clean baseline
because the character was already in combat. Multiple guard types found:
City Guard (Market Square / Gate Ward), City Gate Guard (Gate Plaza),
Guard Captain Velk (Guard Barracks). Pre-crime behavior could not be observed.

### Step 3: Commit theft; observe guard + reputation
**BLOCKED** — Steal requires skullduggery rank 2; smoketester is rank 1
("apprentice"). Could not test theft pathway. Reputation changes for theft
not observable (no rep command).

### Step 4: Assault a Thornwall citizen; observe warn-then-attack escalation
**PARTIAL** — Market Merchant is `non_combatant: true` (confirmed by "You can't
attack market merchant"). Could not perform an assault test. See BUG-3 below
regarding the guard's failure to warn while the player is in the room.

### Step 5: Serious crime / Hostile rep → guards attack on sight
**PASS (partial)** — After killing a City Guard in Market Square (combat from
prior session), two auto-bounties were immediately posted:
- Bounty #1: `faction:thornwall_guards` → `player:17`, 718g + 7 rep, murder
- Bounty #2: `faction:thornwall_citizens` → `player:17`, 718g + 7 rep, murder

Guard Captain Velk in the Guard Barracks triggered "Guard captain Velk prepares
to fight you!" on room entry and immediately attacked. Similarly, an earlier
City Guard in Main Street West triggered "City guard prepares to fight you!"
on entry and engaged in combat. This confirms attack-on-sight behavior fires
for at least some guard instances when the player is wanted for murder.

### Step 6: Probe for bounty/wanted commands
**PASS** — `bounty` command exists with full help page. Subcommands:
`bounty list`, `bounty show <id>`, `bounty declare`, `bounty withdraw`,
`bounty prune-expired`. The bounty board in the Guard Barracks renders
in-fiction and directs players to `bounty list`. Outside notice boards (Gate
Plaza, Old Citadel Plaza) do NOT show bounty postings. `wanted` command does
not exist (returns "not recognized") — acceptable per goal spec.

### Step 7: Allow guard to kill you while wanted; confirm clean death/respawn
**PASS (via suicide)** — Could not engineer a guard kill (character too powerful
for guards). Used `suicide` command to exercise the death path. Result:
- No crash, hang, disconnect, or error spam. **PASS**
- Death message showed cleanly: "*** smoketester has DIED! ***" followed by
  stat/skill decay notices.
- Bounties resolved: both switched to "claimed" status. **PASS**
- Respawned at Temple Interior as expected. **PASS**
- See BUG-1 below for a significant issue with bounty reward recipient.

### Step 8: After respawn, recheck reputation and guard treatment
**PASS (partial)** — After death/bounty resolution:
- Guard Captain Velk (respawned) STILL triggered "prepares to fight you!" on
  entry. Rep-based hostility persists post-death as expected per design.
- Market Square City Guard continued to be passive (pre-existing inconsistency).
- No new bounties posted after death.
- Gold post-death: 1,437g (was 1g before). Bounty rewards incorrectly paid to
  the dying player — see BUG-1.

---

## Findings

### BUG-1 (CRITICAL): Bounty rewards paid to dying player on self-death
**Severity:** High  
**Observed:** On `suicide`, the output showed "You collect a bounty: 718g." twice
(once per open bounty). Gold rose from 1 to 1,437 — confirming the PlayerDeath_
BountyResolve hook awarded both 718g bounties to the dead player themselves.  
**Expected:** Bounties should be awarded to whoever killed the wanted player. On
`suicide` (no killer), the gold should go uncollected, or to the issuing faction,
not back to the target. This represents an economic exploit: a wanted player can
repeatedly murder guards, suicide, and collect their own bounties for net gold
gain.  
**Exact text:**
```
You collect a bounty: 718g.
.
You collect a bounty: 718g.
.
```
Gold before: 1. Gold after: 1,437.

### BUG-2 (MEDIUM): Inconsistent guard reaction to wanted player
**Severity:** Medium  
**Observed:** Some guard instances trigger "prepares to fight you!" on player
entry (Gate Ward, Guard Barracks, Main Street West), while other guards in the
same zone remain completely passive for 3+ minutes in the same room as the
wanted-for-murder player (Market Square Center City Guard, Gate Ward respawn
guard). The passive guards also did idle flavour actions ("A guard checks a
merchant's cargo manifest") confirming they were alive and ticking — just not
reacting.  
**Hypothesis:** The trigger fires on `player_enter` events, but guards in patrol
state or mid-waypoint do not check the wanted-player condition on that hook.
Guards at fixed spawn positions react immediately; patrolling guards do not.  
**Reproduction:** Enter Market Square Center with a wanted player. Guard present,
no warn, no attack, even after 5+ minutes.

### BUG-3 (MEDIUM): Combat stall when attacking non-reactive guard
**Severity:** Medium  
**Observed:** When I explicitly `attack guard` on the passive Market Square City
Guard, the game output "You prepare to enter into mortal combat with City Guard"
and the combat target indicator appeared (`» city guard[|healthy]`), but no
combat rounds fired — neither player attacks nor guard counter-attacks — for 3+
minutes, until I moved to a different room.  
**Expected:** Combat rounds should tick immediately after engagement.  
**Note:** This may be related to BUG-2; if the guard's btree does not recognize
the combat state initiated by the player, it won't swing back, and the player's
attack queue may also stall waiting for the first round signal.

### BUG-4 (MINOR): `sleep` command fails with internal error message
**Severity:** Low  
**Observed:** Running `sleep` in Gate Ward produced:
```
Something prevented you from sleeping: failed to add buff. target: "smoketester" buffId: 15.
```
No conditions were active. The message leaks an internal buff ID ("buffId: 15")
to the player, which is a debug artifact. The player-facing message should say
something like "Something prevents you from resting here" without the internal
reference.  
**Note:** May be a tick-timing artifact (gate ward is an outdoor room — possible
condition check tied to moon phase which was showing "Swiftmoon blazes full
overhead" at the time).

### BUG-5 (PRE-EXISTING): Death help mentions bleed-out mechanic (removed)
**Severity:** Low (pre-existing, already tracked)  
**Observed:** `help death` reads: "When your hitpoints fall to zero or less,
you'll drop to the ground and begin to bleed out." Bleed-out/downed was removed
in 2026-04-18 but this help file hasn't been updated. Already in the bug tracker.

### CONCERN-1: Bounty target shown as raw player ID in `bounty list`
**Observed:** The bounty list output shows "player:17" as the Target column, where
17 is smoketester's internal player ID. This looks odd to any player inspecting
bounties — it should show the player's name (e.g. "player:smoketester" or just
"smoketester"). Fine for admin use but jarring in a player-visible command.

### CONCERN-2: Bounty rewards show raw gold amounts
**Observed:** Bounty list shows `Gold: 718` and `Rep: 7` as numeric values.
Per MEMORY.md policy, player-facing text should not show raw numbers. The bounty
list being admin-level info may make this acceptable, but worth noting if bounty
list becomes broadly player-accessible.

### CONCERN-3: Bounty count doesn't stack per additional kill
**Observed:** After killing 3 guards/the captain, the bounty list stayed at 2
entries. No additional bounties were auto-posted for the 2nd and 3rd kills.  
**Expected:** Unclear whether this is by design (one bounty per faction per
player) or a gap. The first kill triggered both bounties; subsequent kills did
not increment or add new ones. If design intent, document it. If not, it creates
a "free subsequent murders" situation.

### OBSERVATION-1: Bounty board in Guard Barracks works correctly
The weathered bounty board in the Guard Barracks renders an in-fiction
description and directs players to `bounty list`. Does NOT dump raw data into
the room description. Clean implementation.

### OBSERVATION-2: Warn-then-attack vs. attack-on-sight not clearly distinguished
The "warn first, then attack" escalation described in the goal spec was not
clearly observable. The primary guard reaction seen was "prepares to fight you!"
(system line, not spoken dialogue), followed immediately by combat. There was no
spoken warning like "HALT! You are wanted!" before the attack. Whether "prepares
to fight you!" IS the warn or whether the warn layer is missing/not firing could
not be determined from game output alone.

### OBSERVATION-3: Auto-bounty fires on murder of city guard (PASS)
Killing a city guard immediately triggered 2 bounties:
- One from `faction:thornwall_guards`
- One from `faction:thornwall_citizens`
Both set to 5000-round expiry (from round 1372264 to 1377264). Gold reward 718g
each. This confirms the auto-bounty generation on guard-murder works without
crash or error.

### OBSERVATION-4: Death/respawn clean, no crash or hang
The `suicide` command exercised the death path cleanly. No disconnect, no panic,
no error spam. Respawn at Temple Interior. Stat decay ("The shadow of death saps
your keen senses") and skill decay ("Your skullduggery feels rusty") both fired.
The death hook fired without crashing the server.

### OBSERVATION-5: Rep-based hostility persists after bounty resolution
Guard Captain Velk triggered "prepares to fight you!" on room entry even after
both bounties were "claimed" via death. This is correct per design — rep-based
hostility persists beyond the bounty lifecycle.

### OBSERVATION-6: Loot Goblin mechanic fired correctly
After Guard Captain Velk was killed, a Loot Goblin spawned, collected guard
gear, and exited through a portal. No issues observed.

### OBSERVATION-7: Constable Drunn not found
Searched extensively throughout Thornwall City. Constable Drunn (mentioned as
5.1b item needing stillwater_guards faction) was not found. May not be
implemented in this branch yet, or may only spawn under specific conditions.

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Total commands sent | ~110 |
| Guards killed | 3 (City Guard x2, Gate Guard x1, Captain x1 via first combat, Captain wounded in post-death fight) |
| Bounties auto-generated | 2 (both from first guard kill) |
| Bounties status at end | Both "claimed" |
| Crashes / panics | 0 |
| Disconnects | 0 |
| Death tested | Yes (via `suicide`) |
| Respawn worked | Yes |
| Gold pre-death | 1g |
| Gold post-death | 1,437g (WRONG — bounty payout bug) |
| Stat decay on death | Perception ("saps keen senses") |
| Skill decay on death | Skullduggery diminished |
