# playtestrun

## Purpose

Session layer over `playtestenv` for **ephemeral local playtests**.

- **Single-agent (`run`):** parse goals `ephemeral:`, start/stop env, wall-clock
  watchdog, session sidecar, run-scoped mudagent bridge.
- **Multi-agent (`scenario`):** one shared env, roster of actors (each with own
  goals + bind), per-actor bridges, file blackboard, scenario wall-clock.

It does **not** drive mudagent or write gameplay reports (Claude `/playtest` /
`/playtest-scenario` own those). It does not target prod or read `targets.yaml`.

## Files

- `binding.go` — `ParseGoalsEphemeral` (KnownFields, profile vs creation-flow)
- `scenario.go` — `ParseScenario` (roster, admin/PvP refuse, MaxAIConnections)
- `creds.go` — `SelectCredsPlayer` (single-agent); `SelectCredsByActorID` (multi)
- `sidecar.go` — atomic `session.json`; actor/blackboard path helpers
- `run.go` — blocking `Run` supervisor (lease = wall_clock + 5m)
- `scenario_run.go` — blocking `RunScenario` supervisor
- `*_test.go` — unit coverage with fake env + manual clock
- `cmd/playtestrun` — `run` / `scenario` / `status` / `stop` CLI

## Public API (selected)

```go
func ParseGoalsEphemeral(goalsPath string) (EphemeralBinding, error)
func ParseScenario(scenarioPath, playtestRoot string, opts ScenarioParseOpts) (ScenarioFile, error)
func SelectCredsPlayer(credsPath, profileID string) (username, password string, err error)
func SelectCredsByActorID(credsPath, actorID string) (username, password string, err error)
func Run(ctx context.Context, p RunParams) error
func RunScenario(ctx context.Context, p ScenarioParams) error
func WriteSidecar(checkout string, sc SessionSidecar) (string, error)
func ReadSidecar(checkout, runID string) (SessionSidecar, error)
func WriteStopSignal(checkout, runID string) error
func MarkScenarioAbort(checkout, runID, stoppedActorID string) error
func SidecarPath(checkout, runID string) string
func BridgeDirPath(checkout, runID string) string
func ActorBridgeDirPath(checkout, runID, actorID string) string
func BlackboardDirPath(checkout, runID string) string
```

## Human invocation

### When to use `run` vs `scenario`

| Need | Use |
|------|-----|
| One agent on a disposable env | `playtestrun run` / `/playtest local …` |
| N agents needing a **shared** world | `playtestrun scenario` / `/playtest-scenario` |
| N agents with **no** shared world | multiple `playtestrun run` (not scenario) |

Do not use either for prod. Do not point local playtests at a long-lived mud
via `targets.yaml` after 0.3c.

### CLI

```text
playtestrun run --checkout PATH --goals PATH --personality NAME [--wall-clock 30m]
playtestrun scenario --checkout PATH --scenario PATH [--wall-clock 45m] [--force]
playtestrun status --checkout PATH --run ID
playtestrun stop --checkout PATH --run ID
```

`--force` only bypasses the MaxAIConnections roster-size check.

### Single-agent example

```powershell
go run ./cmd/playtestrun run `
  --checkout $PWD `
  --goals tools/playtest/goals/2026-08-03-prepush-sweep.yaml `
  --personality bug-finder
```

`run` blocks until wall-clock expiry or stop. On ready it prints **one JSON
line** (`endpoint`, `creds`, `run_id`, `bridge_dir`, …). Keep the process
alive as the watchdog — do not start-and-exit.

### Multi-agent (scenario) example

```powershell
go run ./cmd/playtestrun scenario `
  --checkout $PWD `
  --scenario tools/playtest/scenarios/party-formation.yaml `
  --wall-clock 15m
```

Ready JSON includes `blackboard_dir`, `on_actor_stop`, and `actors[]` with
per-actor `bridge_dir` / `creds` / `username`. Login multi-agent characters
via **`actor_id`**, never profile-only.

Default scenario wall-clock is **45m**. Lease = wall_clock + ≥5m.

### Blackboard (scenario)

Directory: `tools/playtest/.run/<run_id>/blackboard/`.

| Rule | Detail |
|------|--------|
| Signal file | `<signal-name>.json` (`[a-zA-Z0-9_-]+`) |
| Payload | `{"signal","actor_id","ts","data"}` |
| Write | temp + atomic rename |
| Read | poll / parse; ignore malformed |
| No ptorch | file I/O only |

Prefer in-game channels for character coordination; blackboard is for
driver-visible group signals.

### Checkout rules

- `--checkout` is required (absolute path to a git work tree).
- Sidecar and gameplay reports must echo `commit` + `dirty` loudly.
- Dirty trees are allowed; they are not auto-cleaned.

### Profile vs creation-flow (goals `ephemeral:`)

**Profile path** — materialize a synthetic user:

```yaml
ephemeral:
  profile: early          # fresh, early, mid, veteran, specialist-caster
                          # (admin allowed in single-agent only)
  start_room: 444
  overlays:
    grant_spells: { heal: 1 }
  budgets:
    wall_clock: 30m       # ignored by Go in scenario mode (soft hint)
```

**Creation-flow** — empty Profiles contribution; agent drives `new`:

```yaml
ephemeral:
  creation_flow: true
  creation_rationale: >
    Why a fresh character is required for this run.
  budgets:
    wall_clock: 30m
```

Multi-agent: admin profile hard-banned; creation-flow RoleUser-only; names
must pass `playtestprofiles.ForbiddenIdentity`.

### Artifacts

| Artifact | Path / owner |
|----------|----------------|
| Session sidecar | `tools/playtest/.run/<run_id>/session.json` |
| Single-agent bridge | `…/.run/<run_id>/bridge/` |
| Scenario actor bridge | `…/.run/<run_id>/actors/<id>/bridge/` |
| Blackboard | `…/.run/<run_id>/blackboard/` |
| Stop signal | `…/.run/<run_id>/stop` (+ single-agent bridge/stop) |
| Env failure MD | `tools/playtest/reports/*-environment-failed.md` |
| Gameplay MD | Claude `/playtest` or `/playtest-scenario` |

Scenario statuses add `incomplete_abort`. Per-actor statuses: `pending` |
`ready` | `stopped` | `failed` | `incomplete` | `aborted_peer`.
Creds fields are **paths** (or null); never embed passwords in markdown.

### Loud failures

| Failure | What you see |
|---------|----------------|
| Missing checkout / goals / scenario | usage/parse error; no Docker |
| Bad `ephemeral:` / roster | parse error before Start |
| Admin / `requires.pvp` in scenario | refuse before Start |
| Env/materialize fail | sidecar `environment_failed`; non-zero |
| Wall-clock exceeded | `incomplete_wallclock`; Stop; non-zero |
| Soft token stop (Claude) | report incomplete; still `playtestrun stop` |

## Gotchas

- Local drivers must not fall back to `targets.yaml` for host/creds.
- Scenario bridges are under `actors/<id>/bridge/`, not the flat `bridge/`.
- `Run` / `RunScenario` are blocking; keep the watchdog alive after ready JSON.
- Pre-0.3c local-path helpers may still exist; deferred dead-code cleanup.
- Duplicate profile templates are allowed but prefer mixed loadouts.

## Dependencies

`playtestenv`, `playtestprofiles`, `gopkg.in/yaml.v3`.

## Consumers

`cmd/playtestrun`, Claude `/playtest` and `/playtest-scenario`
(`.claude/commands/playtest.md`, `playtest-scenario.md`).
