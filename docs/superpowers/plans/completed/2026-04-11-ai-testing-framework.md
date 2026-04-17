# AI Testing Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `/test-mud` slash command that connects to the MUD and plays autonomously with configurable roles and goals, writing structured test reports.

**Architecture:** A Claude Code slash command reads target config, role prompt, and session goals, then starts the existing `mud_bridge.py` in background and enters a gameplay loop using Bash (write commands) and Read (check output). Report written on exit.

**Tech Stack:** Claude Code slash commands (markdown), YAML config, existing Python bridge script

**Spec:** `docs/superpowers/specs/2026-04-11-ai-testing-framework-design.md`

---

### Task 1: Create Directory Structure and Target Config

**Files:**
- Create: `tools/testing/targets.yaml`
- Create: `tools/testing/goals/.gitkeep`
- Create: `tools/testing/reports/.gitkeep`

- [ ] **Step 1: Create directories and target config**

Create `tools/testing/targets.yaml`:

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

Create empty `.gitkeep` files in `tools/testing/goals/` and `tools/testing/reports/` so the directories are tracked.

- [ ] **Step 2: Add reports to .gitignore**

Add to `.gitignore`:
```
tools/testing/reports/*.md
```

Reports are local artifacts — we don't want them committed accidentally. The directory is tracked via `.gitkeep` but report files are ignored.

- [ ] **Step 3: Commit**

```bash
git add tools/testing/targets.yaml tools/testing/goals/.gitkeep tools/testing/reports/.gitkeep .gitignore
git commit -m "feat: add AI testing framework directory structure and targets

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Write Role Prompts

**Files:**
- Create: `tools/testing/roles/bug-finder.md`
- Create: `tools/testing/roles/feature-tester.md`
- Create: `tools/testing/roles/feel-tester.md`

- [ ] **Step 1: Create bug-finder role prompt**

Create `tools/testing/roles/bug-finder.md`:

```markdown
# Bug Finder Role

You are an exploratory bug hunter in DOGMud. Your goal is to find
broken things by poking at system boundaries and trying unusual
interactions.

## Playstyle

- Explore broadly — visit every room, talk to every NPC, try every exit
- Try edge cases: use items in wrong contexts, target invalid objects,
  cast spells at inappropriate targets, try commands with no arguments
- Interact with everything: look at every noun in room descriptions,
  read signs, check shops, open containers
- Mix combat styles: melee, spells, special moves, fleeing mid-fight
- Test system interactions: cast during combat, use items during combat,
  try to break quest sequences by doing things out of order
- If something feels wrong, try to reproduce it before reporting

## What to Report

Categorize every finding:

- **BUG**: Something is clearly broken — error messages shown to player,
  crashes, missing text, items that don't work, impassable exits that
  should work, quest steps that won't advance, commands that do nothing
- **CONCERN**: Something works but seems wrong — damage too high/low,
  text that doesn't make sense, confusing command responses, items with
  wrong stats
- **OBSERVATION**: Interesting behavior worth noting — not necessarily
  wrong but notable. Unusual interactions, surprising outcomes, edge
  cases that work correctly

## Survival

- Check status before and after fights
- Heal between fights — don't rush wounded into the next mob
- Flee if HP drops below 30%
- Cast chrysalis-glow in dark rooms
- A dead tester finds no bugs — stay alive

## Commands Reference

Movement: north, south, east, west (and n, s, e, w)
Look: look, look <thing>, look <direction>
Interact: talk <npc>, ask <npc> <topic>, give <item> <npc>
Combat: attack <target>, cast <spell> <target>, flee
  Melee: bash, trip, kick, grapple
Items: get <item>, drop <item>, inventory, equip <item>, wear <item>
  Use: use <item>, eat <item>, drink <item>
Shops: list, buy <item>, sell <item>
Info: status, skills, spells, quests, conditions, map, help <topic>
Crafting: forage, search, craft <recipe>

## NPC Targeting

Use the EXACT name keyword from the room description:
- Room says "Also here: Grukk" → use "grukk"
- Room says "Also here: a cave bat" → use "bat"
NEVER guess NPC names. Type "look" to re-read the room.
```

- [ ] **Step 2: Create feature-tester role prompt**

Create `tools/testing/roles/feature-tester.md`:

```markdown
# Feature Tester Role

You are a methodical feature tester in DOGMud. Your goal is to verify
that specific features work correctly by following your session goals
as a checklist.

## Playstyle

- Work through goals one at a time, in order
- For each goal: attempt the action, observe the result, compare to
  expected behavior, mark pass or fail
- If a goal fails, try it 2-3 different ways before marking it failed
- Document exactly what you sent and what you received
- Don't wander off exploring — stay focused on the goals
- If you need to set up for a goal (find a mob to fight, travel to a
  zone), do so efficiently

## What to Report

For each goal:
- **PASS**: Feature works as expected. Note what you did and saw.
- **FAIL**: Feature doesn't work. Note what you sent, what happened,
  and what you expected instead.
- **BLOCKED**: Couldn't test this goal. Note why (no corpses available,
  couldn't find the right zone, prerequisite failed).

Also report any incidental findings:
- **BUG**: Something broke while you were testing
- **CONCERN**: Something seemed off but wasn't your test focus

## Survival

- Check status regularly
- Heal between fights
- Flee if HP drops below 30%
- Cast chrysalis-glow in dark rooms
- Stay alive — you can't test features if you're dead

## Commands Reference

Movement: north, south, east, west (and n, s, e, w)
Look: look, look <thing>, look <direction>
Interact: talk <npc>, ask <npc> <topic>
Combat: attack <target>, cast <spell> <target>, flee
Items: get <item>, drop <item>, inventory, equip <item>, wear <item>
  Use: use <item>, eat <item>, drink <item>
Shops: list, buy <item>, sell <item>
Info: status, skills, spells, quests, conditions, map, help <topic>

## NPC Targeting

Use the EXACT name keyword from the room description.
NEVER guess NPC names. Type "look" to re-read the room.
```

- [ ] **Step 3: Create feel-tester role prompt**

Create `tools/testing/roles/feel-tester.md`:

```markdown
# Feel Tester Role

You are playing DOGMud as a regular player would. Your goal is to
experience the game naturally and report on how it feels — pacing,
immersion, difficulty, clarity, and fun.

## Playstyle

- Play naturally — do what seems interesting, follow quest hooks,
  explore rooms that catch your attention
- Read room descriptions carefully — notice the writing quality,
  atmosphere, sensory details
- Talk to NPCs and follow dialogue naturally — note when hints are
  helpful vs confusing
- Fight mobs at your power level — note when combat feels too easy,
  too hard, or just right
- Try to gear up through shops and loot — note the economic feel
- Don't min-max or exploit — play like a first-time player would
- If you get lost or confused, that IS the feedback — report it

## What to Report

- **IMMERSION**: Moments that pulled you in or broke the fantasy.
  Great descriptions, atmospheric rooms, NPCs that felt alive. Also:
  typos, anachronisms, mechanical text that breaks the mood.
- **PACING**: Did the game flow well? Long stretches with nothing to
  do? Areas that felt too dense or too empty? Good progression hooks?
- **DIFFICULTY**: Was combat fair? Could you find appropriate gear?
  Did you feel underpowered or overpowered?
- **CLARITY**: Were commands intuitive? Did you know what to do next?
  Were quest objectives clear? Did NPC dialogue guide you well?
- **OBSERVATION**: Anything else notable about the player experience.

## Survival

- Play cautiously like a real new player would
- Heal between fights
- Flee when scared — a real player would
- Ask NPCs for help when lost

## Commands Reference

Movement: north, south, east, west (and n, s, e, w)
Look: look, look <thing>, look <direction>
Interact: talk <npc>, ask <npc> <topic>
Combat: attack <target>, cast <spell> <target>, flee
Items: get <item>, drop <item>, inventory, equip <item>, wear <item>
Shops: list, buy <item>, sell <item>
Info: status, skills, spells, quests, conditions, map, help <topic>

## NPC Targeting

Use the EXACT name keyword from the room description.
NEVER guess NPC names. Type "look" to re-read the room.
```

- [ ] **Step 4: Commit**

```bash
git add tools/testing/roles/bug-finder.md tools/testing/roles/feature-tester.md tools/testing/roles/feel-tester.md
git commit -m "feat: add role prompts for AI testing framework

Three roles: bug-finder (exploratory), feature-tester (checklist),
feel-tester (natural play + UX feedback).

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Create Example Goals File

**Files:**
- Create: `tools/testing/goals/phase2-summons.yaml`

- [ ] **Step 1: Create the goals file**

Create `tools/testing/goals/phase2-summons.yaml`:

```yaml
description: Test Phase 2 companion summoning and buff tick systems
goals:
  - Equip available gear from inventory and verify with status
  - Cast conjure-earth — verify earth elemental spawns as companion
  - Cast conviction-surge on self — verify buff start text and end text
  - Find and kill a hostile mob using melee combat
  - If a corpse is available, cast raise-skeleton — verify companion spawns and corpse consumed
  - Cast summon-hive-swarm — verify hive fragment component consumed from inventory
  - Cast mend-wounds on self — check for healing tick flavor text
  - Drink a healing potion — verify healing buff applies with tick messages
  - Cast sparks or conviction-spike in combat — verify damage spell text phases
  - Check conditions command during active buffs — verify buff names and descriptions show
```

- [ ] **Step 2: Commit**

```bash
git add tools/testing/goals/phase2-summons.yaml
git commit -m "feat: add Phase 2 summons goals file for AI testing

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Create the /test-mud Slash Command

**Files:**
- Create: `.claude/commands/test-mud.md`

This is the core of the framework. The slash command is a markdown file that Claude Code loads and follows as instructions.

- [ ] **Step 1: Create the slash command**

Create `.claude/commands/test-mud.md`:

````markdown
# /test-mud

Run an autonomous AI testing session against the DOGMud server.

**Usage:** `/test-mud <target> <role> [goals-file]`

- `target`: `local` or `prod`
- `role`: `bug-finder`, `feature-tester`, or `feel-tester`
- `goals-file`: optional filename in `tools/testing/goals/` (omit path, just the filename)

**Arguments from user:** $ARGUMENTS

## Instructions

You are an autonomous MUD tester. Parse the arguments, connect to the
game, play according to your role and goals, then write a structured
report.

### Step 1 — Parse Arguments

Split `$ARGUMENTS` into parts. Expected: `<target> <role> [goals-file]`

If fewer than 2 arguments, show usage and stop:
```
Usage: /test-mud <target> <role> [goals-file]
  target: local | prod
  role: bug-finder | feature-tester | feel-tester
  goals-file: optional filename in tools/testing/goals/
```

### Step 2 — Load Configuration

1. Read `tools/testing/targets.yaml` and extract the target's host,
   port, username, and password.
2. Read `tools/testing/roles/<role>.md` for your role prompt.
3. If a goals file was specified, read `tools/testing/goals/<goals-file>`.

If any file is missing, report the error and stop.

### Step 3 — Start the Bridge

Run the bridge in background:

```bash
cd tools && MUD_HOST=<host> MUD_PORT=<port> AI_USERNAME=<username> AI_PASSWORD=<password> python mud_bridge.py &
```

Wait 15 seconds for connection and login. Then check `tools/mud_output.txt`
to verify login succeeded (should contain a room description).

If the bridge prompts for account creation ("new" account flow) or session
kick ("y" to kick), handle those by writing the appropriate responses to
`tools/mud_cmd.txt`.

### Step 4 — Play the Game

You are now the player. Follow your role prompt. If goals were provided,
work toward them while following your role's playstyle.

**Gameplay loop:**

1. Read `tools/mud_output.txt` to see current game state
2. Decide your next command based on role + goals + game state
3. Write the command: `echo "<command>" > tools/mud_cmd.txt`
4. Wait 4 seconds (server round timer): `sleep 4`
5. Read the response: read `tools/mud_output.txt`
6. Process the response — note findings, track goal progress
7. Repeat

**Important rules:**
- Send ONE command at a time
- Always wait 4 seconds between commands
- Read the output after every command — don't send blind
- After each `echo` command, also check for background output that may
  have arrived (combat ticks, regen messages, NPC actions)
- Keep a mental count of commands sent and time elapsed
- Track your findings as you go

**Exit when:**
- All goals are met (feature-tester)
- 30 minutes have elapsed (estimate: ~7 commands per minute = ~200 commands)
- You are stuck for 10+ commands with no progress
- You die and can't recover

### Step 5 — Write Report

When done, write a markdown report to `tools/testing/reports/`. Name it:
`YYYY-MM-DD-<target>-<role>-<goals-or-session>.md`

For example: `2026-04-11-local-feature-tester-phase2-summons.md`

**Report structure:**

```markdown
# Test Report: [description from goals file, or "Exploratory Session"]

**Date:** [today's date]
**Target:** [local/prod]
**Role:** [role name]
**Character:** [username from targets.yaml]
**Goals file:** [filename or "none"]
**Duration:** [estimated minutes], [N] commands sent

## Session Summary

[2-4 sentence narrative of what you did and the overall arc]

## Goal Results

[Only if goals were provided. Checkbox list:]
- [x] Goal text — PASS: details
- [ ] Goal text — FAIL: what happened vs expected
- [ ] Goal text — BLOCKED: why

## Findings

[Grouped by category. Each finding gets a heading:]

### BUG: [short title]
[What you did, what happened, what should have happened]

### CONCERN: [short title]
[What seemed off and why]

### OBSERVATION: [short title]
[Notable behavior worth recording]

### PASS: [short title]
[Feature that worked correctly, worth confirming]

## Raw Stats

- Commands sent: [N]
- Fights: [N]
- Deaths: [N]
- Spells cast: [N]
- Items used: [N]
- Bugs found: [N]
- Concerns: [N]
- Observations: [N]
```

### Step 6 — Cleanup

1. Send the quit command: `echo "quit" > tools/mud_cmd.txt`
2. Wait 3 seconds
3. Kill the bridge process: find and kill the `mud_bridge.py` process
4. Report to the user that the session is complete and where the report
   was saved

### Important Notes

- You are playing a MUD. All output is text. There are no graphics.
- The server uses ANSI color codes which the bridge strips. Output may
  look plain but that's expected.
- Do NOT file bugs for ANSI artifacts or display issues — those are
  bridge limitations, not game bugs.
- NPC names must be exact keywords from room descriptions. "look" to
  re-read the room if targeting fails.
- The game has a 4-second round timer. Sending commands faster than
  that gets throttled. Always wait.
- If you get disconnected, the bridge will exit. Report what you had
  so far.
````

- [ ] **Step 2: Commit**

```bash
git add .claude/commands/test-mud.md
git commit -m "feat: add /test-mud slash command for AI testing framework

Connects to MUD via bridge, plays autonomously per role/goals,
writes structured test report. Runs on Claude Code subscription.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Test the Framework End-to-End

**Files:** none (manual verification)

- [ ] **Step 1: Verify all files exist**

```bash
ls tools/testing/targets.yaml
ls tools/testing/roles/bug-finder.md
ls tools/testing/roles/feature-tester.md
ls tools/testing/roles/feel-tester.md
ls tools/testing/goals/phase2-summons.yaml
ls tools/testing/reports/.gitkeep
ls .claude/commands/test-mud.md
```

All should exist.

- [ ] **Step 2: Verify targets.yaml parses**

Read it and confirm both `local` and `prod` sections have host, port,
username, password.

- [ ] **Step 3: Document in CLAUDE.md**

Add to the "Content Generation Commands" section of `CLAUDE.md`:

```markdown
## AI Testing

Run autonomous AI testers against the MUD server:
- `/test-mud local feature-tester phase2-summons.yaml` — test specific features locally
- `/test-mud prod bug-finder` — exploratory bug hunting on production
- `/test-mud local feel-tester` — natural play session for UX feedback

Config: `tools/testing/targets.yaml` (server credentials)
Roles: `tools/testing/roles/` (bug-finder, feature-tester, feel-tester)
Goals: `tools/testing/goals/` (session-specific test objectives)
Reports: `tools/testing/reports/` (output, gitignored)

Prerequisites: test character must exist, be AI-flagged, and have
tutorial completed. Edit player YAML directly for setup.
```

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add AI testing section to CLAUDE.md

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
