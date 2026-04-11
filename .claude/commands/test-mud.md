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
