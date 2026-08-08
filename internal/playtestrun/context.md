# playtestrun

## Purpose

Thin session layer over `playtestenv` for **single-agent ephemeral local
playtests**. Parses goals `ephemeral:` binding, starts/stops the disposable
Docker env, enforces a wall-clock watchdog, writes a session sidecar, and
scopes the mudagent bridge under the run id. It does **not** drive mudagent
or write gameplay reports (Claude `/playtest` owns those). It does not target
prod or read `targets.yaml`.

## Files

- `binding.go` — `ParseGoalsEphemeral` (KnownFields, profile vs creation-flow)
- `creds.go` — `SelectCredsPlayer` (profile match; never logs passwords)
- `sidecar.go` — atomic `session.json` under `.run/<run_id>/`
- `run.go` — blocking `Run` supervisor (lease = wall_clock + 5m)
- `*_test.go` — unit coverage with fake env + manual clock
- `cmd/playtestrun` — `run` / `status` / `stop` CLI

## Public API (selected)

```go
func ParseGoalsEphemeral(goalsPath string) (EphemeralBinding, error)
func SelectCredsPlayer(credsPath, profileID string) (username, password string, err error)
func Run(ctx context.Context, p RunParams) error
func WriteSidecar(checkout string, sc SessionSidecar) (string, error)
func ReadSidecar(checkout, runID string) (SessionSidecar, error)
func WriteStopSignal(checkout, runID string) error
func SidecarPath(checkout, runID string) string
func BridgeDirPath(checkout, runID string) string
```

## Human invocation

### When to use this

Use `playtestrun` (or `/playtest local …`, which calls it) whenever you want a
**local** AI playtest against a disposable server built from a chosen checkout.
Do not use it for prod. Do not point it at a long-lived local mud process via
`targets.yaml` — that path is intentionally unused for local after 0.3c.

### CLI

```text
playtestrun run --checkout PATH --goals PATH --personality NAME [--wall-clock 30m]
playtestrun status --checkout PATH --run ID
playtestrun stop --checkout PATH --run ID
```

From repo root:

```powershell
go run ./cmd/playtestrun run `
  --checkout $PWD `
  --goals tools/playtest/goals/2026-08-03-prepush-sweep.yaml `
  --personality bug-finder
```

`run` blocks until wall-clock expiry or a stop signal. On ready it prints
**one JSON line** to stdout (`endpoint`, `creds`, `run_id`, `checkout`,
`commit`, `dirty`, `deadline_at`, `sidecar`, `bridge_dir`). Non-zero exit on
`environment_failed` or `incomplete_wallclock`.

`--wall-clock` overrides the goals `ephemeral.budgets.wall_clock` (default 30m
when omitted from goals). Lease granted to playtestenv is
`wall_clock + 5m` so cleanup can finish before lease expiry.

### Checkout rules

- `--checkout` is required (absolute path to a git work tree).
- Sidecar and gameplay reports must echo `commit` + `dirty` loudly.
- Dirty trees are allowed; they are not auto-cleaned.

### Profile vs creation-flow

Goals top-level `ephemeral:` (unknown keys fail closed):

**Profile path** — materialize a synthetic user:

```yaml
ephemeral:
  profile: early          # one of: fresh, early, mid, veteran, specialist-caster, admin
  start_room: 444         # must be > 0
  overlays:               # optional; same shape as playtestenv ProfileOverlays
    grant_spells: { heal: 1 }
  budgets:
    wall_clock: 30m
```

**Creation-flow** — empty Profiles list, no `creds.json`; agent drives `new`:

```yaml
ephemeral:
  creation_flow: true
  creation_rationale: >
    Why a fresh character is required for this run.
  budgets:
    wall_clock: 30m
```

`creation_flow` forbids `profile` / `start_room` / `overlays`. Profile and
creation_flow are mutually exclusive. Legacy `session.max_rounds` is ignored
by Go (soft pacing note for Claude only).

Exemplars: `tools/playtest/goals/newbie-naive.yaml` (creation),
`corpse-looting.yaml` (early@444), `2026-08-03-prepush-sweep.yaml` (bug-finder
SOP creation-flow).

### Reading sidecar and reports

| Artifact | Path / owner |
|----------|----------------|
| Session sidecar | `tools/playtest/.run/<run_id>/session.json` (`playtestrun`) |
| Mudagent bridge | `tools/playtest/.run/<run_id>/bridge/` |
| Stop signal | `…/bridge/stop` (`playtestrun stop` or driver) |
| Env failure MD | `tools/playtest/reports/*-environment-failed.md` (`playtestenv`) |
| Gameplay MD | `tools/playtest/reports/…` (Claude `/playtest`) |

Sidecar statuses: `starting` | `ready` | `incomplete_wallclock` | `stopped` |
`environment_failed`. Nested `budgets.wall_clock`. Creds field is a **path**
(or empty); never embed passwords in markdown.

### Worked examples

1. **Adversarial content SOP**

```powershell
go run ./cmd/playtestrun run --checkout $PWD `
  --goals tools/playtest/goals/2026-08-03-prepush-sweep.yaml `
  --personality bug-finder
# ready JSON → start mudagent on bridge_dir → play → stop
go run ./cmd/playtestrun stop --checkout $PWD --run <run_id>
```

2. **Corpse-loot profile smoke**

```powershell
go run ./cmd/playtestrun run --checkout $PWD `
  --goals tools/playtest/goals/corpse-looting.yaml `
  --personality feature-tester
```

Match `creds.json` player with `profile: early` for login.

3. **Inspect mid-run**

```powershell
go run ./cmd/playtestrun status --checkout $PWD --run <run_id>
```

### Loud failures

| Failure | What you see |
|---------|----------------|
| Missing checkout / goals / personality | usage error; no Docker |
| Bad / missing `ephemeral:` | parse error before Start |
| Env/materialize fail | sidecar `environment_failed`; non-zero exit; env report path |
| Wall-clock exceeded | sidecar `incomplete_wallclock`; Stop; non-zero exit |
| Soft token stop (Claude) | gameplay report incomplete; still `playtestrun stop` |

## Gotchas

- Local `/playtest` must not fall back to `targets.yaml` for host/creds.
- Bridge files are under `.run/<run_id>/bridge/`, not the flat `.run/` root.
- Pre-0.3c local-path helpers may still exist in-tree; do not delete them in
  this chunk (deferred dead-code cleanup).
- `Run` is blocking; drivers should background it and parse the ready JSON
  line, then signal stop when play ends.

## Dependencies

`playtestenv`, `playtestprofiles`, `gopkg.in/yaml.v3`.

## Consumers

`cmd/playtestrun`, Claude `/playtest` local driver
(`.claude/commands/playtest.md`).
