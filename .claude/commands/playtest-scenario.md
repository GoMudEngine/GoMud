---
description: Run a multi-agent ephemeral scenario (shared env, N mudagents)
argument-hint: --checkout <abs-path> <scenario-file>
---

# /playtest-scenario `--checkout <abs>` `<scenario-file>`

Conductor for **multi-agent** local playtests on **one shared ephemeral
server** (chunk 0.3d). Go owns the env (`playtestrun scenario`); you drive N
concurrent mudagents and write one combined gameplay report.

Single-agent local runs still use `/playtest` → `playtestrun run`.
Independent non-interacting work that does **not** need a shared world should
be **multiple** `/playtest` / `playtestrun run` invocations — not a scenario.

> ⚠️ **Cost:** N agents ≈ N× a single `/playtest` in tokens. Start with 2.
> Server AI pool: `Network.MaxAIConnections` (default 20).

## Preconditions

1. **`--checkout` required** — absolute path to a DOGMud git work tree. No
   silent cwd. Do **not** use `targets.yaml` for endpoint/creds.
2. Scenario file under `tools/playtest/scenarios/` (or absolute path).
3. Harness for mudagent only:
   `HARNESS="${GOMUD_HARNESS_DIR:-../gomud-playtest-harness}"`.
   If missing, STOP and tell the user to set `GOMUD_HARNESS_DIR` or clone the
   harness next to DOGMud.
4. **No `ptorch`.** Blackboard is file I/O under the run’s `blackboard/` dir.
5. **`requires.pvp` scenarios are refused** (deferred). Do not run
   `adversarial-contest.yaml` until ephemeral PvP overrides exist.
6. **No admin** actors in multi-agent (hard ban). Creation-flow actors are
   RoleUser-only.

## 1. Start the scenario supervisor (long-lived)

From the checkout root, start `playtestrun scenario` as a **blocking** process
you keep alive for the wall-clock window (same pattern as 0.3c `run`). Do
**not** treat start-and-exit as success.

```powershell
$CHECKOUT = "<absolute checkout>"
$SCENARIO = "$CHECKOUT\tools\playtest\scenarios\<name>.yaml"
# Keep this process running; parse the first stdout JSON line as ready.
go run ./cmd/playtestrun scenario `
  --checkout $CHECKOUT `
  --scenario $SCENARIO `
  --wall-clock 15m   # optional override; default from scenario / 45m
```

Ready JSON (one line) includes: `run_id`, `endpoint`, `checkout`, `commit`,
`dirty`, `deadline_at`, `sidecar`, `blackboard_dir`, `on_actor_stop`,
`actors[]` with `id`, `personality`, `goals_path`, `bridge_dir`, `creds|null`,
`username`, `profile|null`, `creation_flow`, `status`.

**Ready-gate:** do not spawn mudagents until this line parses and actors are
`ready`. Surface `commit` + `dirty` loudly.

## 2. Spawn concurrent mudagents

For each `actors[]` entry:

1. Start mudagent against `endpoint` with that actor’s **`bridge_dir`**
   (under `.run/<run_id>/actors/<id>/bridge/`).
2. **Profile actors:** login with username/password selected by **`actor_id`**
   from `creds` (never profile-only when duplicates exist). In-game targeting
   (e.g. `party invite`) uses **character names** from `who`/`look`, not the
   `pt_*` login username.
3. **Creation-flow:** drive `new` as RoleUser only. Before accepting a name,
   call the shared prod-identity gate mentally / via helper: names must not
   match checked-in stems or close variants (`ForbiddenIdentity`).
4. Soft per-actor token guidance only — do **not** hard-kill on tokens.
5. If a mudagent fails to start after ready: honor `on_actor_stop`:
   - `continue` (default) — mark that actor failed; keep peers + env
   - `abort` — stop all mudagents, `playtestrun stop`, set sidecar via
     driver contract (`incomplete_abort` / peer `aborted_peer`), report
     incomplete

## 3. Play + coordinate

- Before play: read each actor’s personality + `engine-profile.yaml`. For
  profile actors whose `profile` is **not** `fresh`, also read
  `tools/playtest/profiles/context.md` (MUD orientation + AI rate limit).
- Drive each actor from its `personality` + `goals_path` (+ scenario
  `group_goals`). Pace under `AICommandsPerRound` (shipped **3**/round).
- Prefer **in-game** channels (say / party / tell) for character↔character.
- **File blackboard** for driver-visible orchestration:
  - Dir: `blackboard_dir` from ready JSON
  - Signal file: `<signal-name>.json` where `signal-name` ∈ `[a-zA-Z0-9_-]+`
  - Payload:
    `{"signal":"<name>","actor_id":"<id>","ts":"<RFC3339>","data":{...}}`
  - Write: temp file in the same dir, then **atomic rename** into place
  - Read: poll for existence / JSON parse; ignore malformed with a log line
  - **No `ptorch bb`**

## 4. Stop and report

1. Always `playtestrun stop --checkout PATH --run <run_id>` (or let wall-clock
   cut; sidecar becomes `incomplete_wallclock`).
2. Quit stray mudagents.
3. Write **one combined** report per
   `tools/playtest/multi-agent-report-format.md` to
   `tools/playtest/reports/<date>-<scenario>.md`.
4. Checklist before handing off:
   - [ ] Ready-gate observed (ready JSON + sidecar `ready`)
   - [ ] Each actor had its own bridge + goals
   - [ ] Login used `actor_id` (or creation-flow)
   - [ ] Blackboard used file I/O only
   - [ ] `on_actor_stop` honored if an actor failed early
   - [ ] Combined report written; passwords never in markdown
   - [ ] `playtestrun stop` (or wall-clock Stop) completed

## When to use `run` vs `scenario`

| Need | Command |
|------|---------|
| One agent, disposable env | `playtestrun run` / `/playtest local …` |
| N agents, **shared** world (party, concurrent AI, group_goals) | `playtestrun scenario` / `/playtest-scenario` |
| N agents, **no** shared world | N× `playtestrun run` |
