---
description: Run an AI playtest session by driving the mudagent adapter
argument-hint: <local|prod> <personality> [goals-file]  (local also requires --checkout)
---

# /playtest `<local|prod> <personality> [goals-file]`

The DOGMud Claude Code driver for the GoMud-Module-Playtest-Harness. It spawns
`mudagent`, drives it through its line-in / JSON-line-out protocol, and writes a
report. Personalities: `bug-finder`, `feature-tester`, `feel-tester`.

**Local (0.3c+)** always uses ephemeral `playtestrun` (Docker checkout). It does
**not** read `tools/playtest/targets.yaml` for host/port/user/password.
**Prod** still uses `targets.yaml` (unchanged).

## Local usage (required)

```text
/playtest local --checkout <absolute-path> <personality> <goals-file>
```

- `--checkout` is **required** (no cwd default). Loudly record commit + dirty
  in the gameplay report header (also present on the session sidecar).
- Goals file is **required**. It must contain a valid top-level `ephemeral:`
  block (profile+start_room or creation_flow+rationale). See
  `internal/playtestrun` and exemplars under `tools/playtest/goals/`.
- Adversarial SOP default:
  `/playtest local --checkout <abs> bug-finder 2026-08-03-prepush-sweep.yaml`

## 1. Load configuration

### Local

1. Resolve absolute `--checkout`.
2. Read `tools/playtest/personalities/<personality>.md`.
3. Read `tools/playtest/engine-profile.yaml`.
4. Read the goals file (required). Confirm `ephemeral:` is present.
5. If `ephemeral.profile` is set and is **not** `fresh`, also read
   `tools/playtest/profiles/context.md` (short MUD + rate-limit orientation).
6. **Do not** load `targets.yaml` for endpoint/creds.

### Prod

- Read `tools/playtest/targets.yaml`; look up `prod` → `host`, `port`, and
  optional `user`/`password`. This file is **gitignored** and is not in a fresh
  clone. If it is missing, tell the user to
  `cp tools/playtest/targets.example.yaml tools/playtest/targets.yaml` and fill
  it in. Never commit it and never echo its `password` values into output.
- Read personality + engine-profile as above.
- Goals file optional for free-form prod exploration.

## 2. Resolve the harness binary

Resolve the harness directory:
`HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"` (relative to the
DOGMud repo root). If `$HARNESS/mudagent.exe` exists (Windows) use it; else if
`$HARNESS/mudagent` exists use it; else run `go run ./cmd/mudagent` from inside
`$HARNESS`. If `$HARNESS` does not exist, STOP and tell the user:
"playtest harness not found at $HARNESS — set GOMUD_HARNESS_DIR or clone
GoMudEngine/GoMud-Module-Playtest-Harness next to the DOGMud repo."

## 3. Start the environment (local) or connect (prod)

### Local — `playtestrun run` (blocking watchdog)

Start the ephemeral env **before** mudagent. From the DOGMud repo (or any
shell), run in the background so the wall-clock supervisor stays alive:

```powershell
go run ./cmd/playtestrun run --checkout <abs> --goals <goals-path> --personality <name>
```

- Parse the **one JSON line** on stdout when ready. Required fields:
  `endpoint`, `creds` (path or null), `run_id`, `checkout`, `commit`, `dirty`,
  `deadline_at`, `sidecar`, `bridge_dir`.
- **Ready-gate:** do **not** start mudagent unless status/ready JSON shows
  `ready`. On `environment_failed` (non-zero exit / sidecar status), abort,
  point at the environment-failed report path from the sidecar
  (`environment_report` / playtestenv report), and do not invent gameplay
  success.
- Bridge paths are **run-scoped**:
  `tools/playtest/.run/<run_id>/bridge/commands.txt` and `events.jsonl`
  (create empty files under `bridge_dir` from the ready JSON).
- Session sidecar: `tools/playtest/.run/<run_id>/session.json`.

### Prod — legacy flat bridge

```sh
mkdir -p tools/playtest/.run && : > tools/playtest/.run/commands.txt && : > tools/playtest/.run/events.jsonl
tail -n +1 -f tools/playtest/.run/commands.txt \
  | <mudagent-binary-or-go-run> --target <host>:<port> [--user <user> --password <password>] \
  > tools/playtest/.run/events.jsonl 2>&1 &
```

### Local — start mudagent against ready JSON

```sh
# BRIDGE = bridge_dir from ready JSON
mkdir -p "$BRIDGE" && : > "$BRIDGE/commands.txt" && : > "$BRIDGE/events.jsonl"
tail -n +1 -f "$BRIDGE/commands.txt" \
  | <mudagent-binary-or-go-run> --target <endpoint.host>:<endpoint.port> \
      [--user <user> --password <password>] \
  > "$BRIDGE/events.jsonl" 2>&1 &
```

Creds:

- If `ephemeral.creation_flow: true` → `creds` is null; drive `new` (step 4).
- If `ephemeral.profile` is set → select the player whose `profile` field
  matches in the creds JSON (or use `playtestrun` helper semantics). Never
  paste passwords into reports — paths only.

## 4. Log in, or create a character

Poll `<bridge>/events.jsonl` until
`{"type":"status","state":"logged_in"}`.

- **Profile path:** adapter auto-login with matched username/password.
- **Creation-flow:** drive `new` → username → password → confirm → enter world.
  New characters may be pre-tutorial “ghosts” (see engine profile `onboarding`).

**Then ensure an ASCII charset (DOGMud).** `set charset` is a toggle — converge
to ASCII (at most 2 sends) as before.

If `disconnected`/`error` arrives first, abort, stop the env (local), report.

## 5. Play (main loop)

Same as before: read events, decide from personality + goals + engine profile
(+ `profiles/context.md` when profile ≠ `fresh`), append commands, pace on
`Playtest.Round` beacons, respect `AICommandsPerRound` (shipped **3**/round).
Soft token/API budget exhaustion ⇒ stop play and mark the gameplay report
**incomplete** (still run cleanup).

## 6. Exit conditions

Stop when any holds: all goals met; wall-clock / sidecar
`incomplete_wallclock`; soft token stop; stuck 10+ commands; fatal
error/disconnect.

## 7. Write the report

Gameplay report (Claude-owned) under
`tools/playtest/reports/YYYY-MM-DD-<target>-<personality>[-<goals>].md`.
Prefer templates in `tools/playtest/report-templates/` (`newbie-creation.md`,
`bug-finder-sweep.md`).

**Required header fields (local):** checkout, commit, dirty, run_id, binding
summary, wall-clock elapsed vs budget, sidecar status, outcome. Never passwords.

`incomplete_wallclock` or soft token/API stop ⇒ outcome **incomplete**, not
success. Env failures use playtestenv `*-environment-failed.md`, not a fake
gameplay pass.

## 8. Clean up

### Local

```powershell
go run ./cmd/playtestrun stop --checkout <abs> --run <run_id>
# playtestrun run watchdog also Stops on deadline; always ensure stop
printf '%s\n' '{"control":"quit"}' >> <bridge>/commands.txt
```

### Prod

```sh
printf '%s\n' '{"control":"quit"}' >> tools/playtest/.run/commands.txt
sleep 1
pkill -f 'tail -n +1 -f tools/playtest/.run/commands.txt' 2>/dev/null || true
pkill -f 'cmd/mudagent' 2>/dev/null || true
```

Report completion (and the report path) to the user.
