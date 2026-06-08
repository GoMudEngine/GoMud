---
description: Run an AI playtest session by driving the mudagent adapter
argument-hint: <local|prod> <personality> [goals-file]
---

# /playtest `<local|prod> <personality> [goals-file]`

The DOGMud Claude Code driver for the GoMud-Module-Playtest-Harness. It spawns
`mudagent`, drives it through its line-in / JSON-line-out protocol, and writes a
report. Personalities: `bug-finder`, `feature-tester`, `feel-tester`.

## 1. Load configuration

- Read `tools/playtest/targets.yaml`; look up `<target>` → `host`, `port`, and
  the optional `user`/`password`. These ship with working defaults — `local` is
  `localhost:55555` with **blank credentials**, which means "create a character
  on first run." Fill creds in to auto-log-in to an existing character.
- Read `tools/playtest/personalities/<personality>.md` — your **role** (how to
  play).
- Read `tools/playtest/engine-profile.yaml` — command names, world orientation,
  mechanics. **All engine-specific behavior comes from here** (it ships with
  DOGMud defaults).
- **The goals file is *what to test*** — if `[goals-file]` was given, read
  `tools/playtest/goals/<goals-file>` (the ready-made ones live under
  `tools/playtest/goals/`). It defines the objectives and `verify` conditions
  you drive toward and report against. Without one, play free-form to the
  personality (an exploratory run with no set objectives).

## 2. Resolve the harness binary

Resolve the harness directory:
`HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"` (relative to the
DOGMud repo root). If `$HARNESS/mudagent.exe` exists (Windows) use it; else if
`$HARNESS/mudagent` exists use it; else run `go run ./cmd/mudagent` from inside
`$HARNESS`. If `$HARNESS` does not exist, STOP and tell the user:
"playtest harness not found at $HARNESS — set GOMUD_HARNESS_DIR or clone
GoMudEngine/GoMud-Module-Playtest-Harness next to the DOGMud repo."

## 3. Start the adapter (background, file-bridged)

A slash command can't hold a live pipe across tool calls, so bridge `mudagent`'s
stdio to files with `tail -f`. Run it **from the DOGMud repo root** — `go run`
compiles the adapter on the fly, so there's no separate build step:

```sh
mkdir -p tools/playtest/.run && : > tools/playtest/.run/commands.txt && : > tools/playtest/.run/events.jsonl
# Include --user/--password ONLY if the target has them; blank credentials mean
# the agent creates a character on first run (step 4).
tail -n +1 -f tools/playtest/.run/commands.txt \
  | <mudagent-binary-or-go-run> --target <host>:<port> [--user <user> --password <password>] \
  > tools/playtest/.run/events.jsonl 2>&1 &
```

Where `<mudagent-binary-or-go-run>` is the binary or `go run` invocation
resolved in step 2. (If you prefer a prebuilt binary —
`go build -o mudagent ./cmd/mudagent` inside `$HARNESS` — use `./mudagent` in
place of `go run ./cmd/mudagent`.)

You issue a command by appending one line to `tools/playtest/.run/commands.txt`;
you read results from new lines in `tools/playtest/.run/events.jsonl`. Each
event is one JSON object: `{"type":"output","text":...}`,
`{"type":"gmcp","package":...,"data":...}`,
`{"type":"status","state":"connected|logged_in|disconnected"}`,
`{"type":"error","message":...}`. (The adapter handles connect + GMCP
negotiation. With `--user`/`--password` it also auto-logs-in to an existing
account; without them — or if the account doesn't exist yet — *you* drive login
and character creation via commands, see step 4.)

## 4. Log in, or create a character

Poll `tools/playtest/.run/events.jsonl` until
`{"type":"status","state":"logged_in"}` (`Room.Info`/`Char.Info` confirms you're
in the world). Getting there depends on whether your character exists:

- **It exists** (the target had `user`/`password`): the adapter logs in
  automatically — just wait for `logged_in`.
- **It doesn't exist yet** (blank creds, or you see the `username (or "new")`
  prompt repeat / an "invalid login"): **create a character via the normal
  new-player flow** — this is part of what a tester exercises, and a feel-tester
  should grade it. Append responses one per line to
  `tools/playtest/.run/commands.txt`, following the prompts. On stock GoMud the
  sequence is: `new` → desired username → password → password again → email
  (blank is fine) → screen reader? `n` → confirm `y`. You then enter the world.
- **New characters begin as a pre-tutorial "ghost"** (see the engine profile's
  `onboarding`). Take the tutorial or choose to start playing to become a full
  character before attempting goals that need stats or items.

If `disconnected`/`error` arrives first, abort and report. Then run any
`setup_commands` from the engine profile.

## 5. Play (main loop)

Repeat until an exit condition:
1. Read new lines from `tools/playtest/.run/events.jsonl` — the `output` text,
   `gmcp` state, and `beacon` events are your view of the world.
2. Decide the next command from your **personality** + **goals** + **engine
   profile** + current state.
3. Append the command (one line) to `tools/playtest/.run/commands.txt`.
4. **Pace on the round beacon:** wait for the next
   `{"type":"beacon","event":"Round"}` event (the `playtest` module emits one
   per round). It is a reliable per-round tick and carries
   `{round, hp, hp_max, sp, sp_max, room_id}` — use it for pacing and goal
   scoring. **DOGMud note:** the beacon also carries `cp`/`cp_max` (Conviction
   pool) in addition to `hp`/`sp`, so goals can pace and verify against all
   three resource pools. The beacon only fires when the playtest module is
   enabled (`Modules.playtest.Enabled: true` + `Beacons: true` in the server
   config). *Fallback:* if no beacons arrive (the `playtest`/`gmcp` modules are
   absent), fall back to response quiescence (~1–2s with no new events).
5. Pace yourself within a round too: the server caps AI input at
   `AI.CommandsPerRound` (default 2) per round; a dropped command is reported
   back as an `output` notice.
6. Track findings and goal progress as you go (the beacon snapshot is good
   evidence for `verify` conditions).

## 6. Exit conditions

Stop when any holds: all goals met; ~30 minutes elapsed; stuck for 10+ commands
with no progress; or a fatal `error`/`disconnected` status.

## 7. Write the report

Write the final report to
`tools/playtest/reports/YYYY-MM-DD-<target>-<personality>[-<goals-or-session>].md`.
The report should cover: session summary, findings list (severity + repro steps),
goal outcomes (met/unmet with evidence from beacon snapshots), and
recommendations.

## 8. Clean up

```sh
printf '%s\n' '{"control":"quit"}' >> tools/playtest/.run/commands.txt   # closes the adapter
sleep 1
pkill -f 'tail -n +1 -f tools/playtest/.run/commands.txt' 2>/dev/null || true
pkill -f 'cmd/mudagent' 2>/dev/null || true
```

Report completion (and the report path) to the user.
