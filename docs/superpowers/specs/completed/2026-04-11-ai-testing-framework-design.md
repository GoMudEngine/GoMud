# AI Testing Framework for DOGMud

**Date:** 2026-04-11
**Status:** Draft

## Goal

A Claude Code slash command (`/test-mud`) that connects to the MUD,
plays the game autonomously in a specific role with session goals, and
writes a structured test report. Runs in a dedicated Claude Code window
on the user's subscription — no API key costs.

## Scope

**In scope:**
- `/test-mud` slash command
- Target configuration (local / prod)
- Three roles: bug-finder, feature-tester, feel-tester
- Session goals files
- Structured report output
- Bridge startup/teardown

**Out of scope:**
- Multi-character party testing (future extension)
- Automated character setup (admin flags, quest tokens — done manually)
- Changes to the MUD engine itself
- Replacing the existing `ai_player.py` (kept for standalone API use)

## Design

### Invocation

```
/test-mud <target> <role> [goals-file]
```

- `target`: `local` or `prod` — maps to host/port/credentials
- `role`: `bug-finder`, `feature-tester`, or `feel-tester`
- `goals-file`: optional filename in `tools/testing/goals/`. If omitted,
  the role's default behavior applies (explore, test, or play).

### File Layout

```
tools/testing/
├── targets.yaml              # host/port/credentials per target
├── roles/
│   ├── bug-finder.md         # exploratory bug hunting prompt
│   ├── feature-tester.md     # checklist-style goal testing prompt
│   └── feel-tester.md        # natural play + UX feedback prompt
├── goals/
│   └── (session goal files)  # YAML files with test objectives
└── reports/
    └── (output reports)      # timestamped markdown reports
```

### Target Configuration

`tools/testing/targets.yaml`:
```yaml
local:
  host: localhost
  port: 55555
  username: smoketester
  password: smoke123test
prod:
  host: dogmud.org
  port: 55555
  username: aitester
  password: testpass123
```

### Goals File Format

`tools/testing/goals/<name>.yaml`:
```yaml
description: Test Phase 2 companion summoning and buff ticks
goals:
  - Equip gear from inventory, verify status
  - Cast conjure-earth, verify elemental spawns as companion
  - Cast summon-hive-swarm, verify component consumed
  - Kill a mob, cast raise-skeleton on corpse
  - Cast mend-wounds on self, watch for heal tick messages
```

Goals are natural-language objectives. The AI tester interprets them
and figures out how to achieve them in-game. Goals override role
defaults only where they directly conflict.

### Role Prompts

Each role prompt (`tools/testing/roles/<role>.md`) defines:

- **Playstyle** — how the tester approaches the game
- **What to look for** — what constitutes a finding for this role
- **Finding categories** — BUG, CONCERN, OBSERVATION, PASS
- **Essential commands reference** — movement, combat, dialogue, etc.
- **NPC targeting rules** — use exact name keywords from room text
- **Survival rules** — don't die, heal between fights, flee when low

**bug-finder:** Explores broadly, tries edge cases, pokes at system
boundaries. Attempts unusual command sequences, targets invalid objects,
tests system interactions. Reports bugs and unexpected behavior.

**feature-tester:** Follows goals methodically. Tests each objective,
verifies expected output, marks pass/fail. Reports with evidence (what
was sent, what was received, what was expected).

**feel-tester:** Plays naturally as a new player would. Reports on
pacing, immersion, difficulty curve, confusing moments, text quality.
Doesn't look for bugs — reports on how the game *feels*.

### Gameplay Loop

The slash command instructs Claude Code to:

1. **Read config** — load target from `targets.yaml`, role prompt from
   `roles/`, goals from `goals/` (if specified)
2. **Start bridge** — launch `mud_bridge.py` in background with the
   target's host/port/credentials via environment variables
3. **Wait for login** — poll `mud_output.txt` until a room description
   appears (indicates successful login)
4. **Play** — loop:
   - Read `mud_output.txt` for current game state
   - Decide next command based on role + goals + current state
   - Write command to `mud_cmd.txt`
   - Wait 4 seconds (server round timer)
   - Read response from `mud_output.txt`
   - Note any findings
5. **Exit** when: all goals met, stuck for 10+ commands with no
   progress, or 30 minutes elapsed (configurable)
6. **Write report** to `tools/testing/reports/` with timestamped
   filename
7. **Cleanup** — send `quit`, kill bridge process

### Report Format

`tools/testing/reports/YYYY-MM-DD-<target>-<role>-<goals>.md`:

```markdown
# Test Report: [description]

**Date:** 2026-04-11
**Target:** local
**Role:** feature-tester
**Character:** smoketester
**Goals file:** phase2-summons.yaml
**Duration:** 22 minutes, 47 commands sent

## Session Summary

Brief narrative of what the tester did and the overall arc of the
session.

## Goal Results

- [x] Equip gear — PASS: sword and armor equipped, status confirmed
- [x] Conjure-earth — PASS: elemental spawned as companion
- [ ] Raise-skeleton — BLOCKED: no humanoid corpses available
- [x] Heal tick text — FAIL: no per-tick messages during mend-wounds

## Findings

### BUG: Dismissing companion makes it hostile
Dismissed earth elemental via "dismiss elemental" — mob turned hostile
and attacked. Expected: companion despawns peacefully.

### CONCERN: No feedback during healing ticks
Mend-wounds heals silently. Player has no indication that healing is
occurring between the start and end messages.

### OBSERVATION: Thornwall shops have limited stock
Weapon shop had only 2 items available. May be intentional (living
economy) but felt sparse for a new character gearing up.

### PASS: Conviction surge full lifecycle
Cast → wait → resolve → buff start → buff active → buff end. All text
phases fired correctly with proper color formatting.

## Raw Stats

- Commands sent: 47
- Fights: 3
- Deaths: 0
- Spells cast: 12
- Items used: 2
- Bugs found: 1
- Concerns: 1
- Observations: 1
```

### Prerequisites

Before running `/test-mud`, the target character must:
- Exist on the target server (account created)
- Be AI-flagged (admin runs `ai-flag <username>`)
- Have tutorial completed (`questprogress: 1: end` in player YAML or
  admin grants quest token)
- Have appropriate gear, spells, and skills for the goals

Character setup is done manually or by editing the player YAML
directly. The testing framework does not handle character provisioning.

## Future Extensions

- **Party testing:** Two `/test-mud` instances in separate windows,
  same target, different characters. Goals reference each other
  ("party with smoketester2, test group healing").
- **Regression suite:** A meta-goals file that runs a sequence of
  goals files, producing a combined report.
- **Character provisioning:** A setup script that configures a test
  character's stats/spells/gear from a profile YAML.
- **Diff reports:** Compare reports across sessions to detect
  regressions ("this finding was PASS last session, now FAIL").
